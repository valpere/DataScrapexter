// internal/scraper/engine.go - Enhanced with error management (existing code preserved)
package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valpere/DataScrapexter/internal/browser"
	"github.com/valpere/DataScrapexter/internal/errors"
	"github.com/valpere/DataScrapexter/internal/proxy"
)

// Enhanced Engine struct (existing fields preserved, error service added)
type Engine struct {
	// Existing fields preserved
	httpClient     *http.Client
	userAgentPool  []string
	currentUAIndex int
	config         *Config
	rateLimiter    *RateLimiter
	
	// Enhanced features
	errorService   *errors.Service
	browserManager *browser.BrowserManager
	proxyManager   proxy.Manager
}

// Enhanced Result struct (existing fields preserved, error info added)
type Result struct {
	// Existing fields preserved
	Data      map[string]interface{} `json:"data"`
	Success   bool                   `json:"success"`
	Error     error                  `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	
	// Enhanced error information
	Errors    []string `json:"errors,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
	ErrorRate float64  `json:"error_rate,omitempty"`
}

// Enhanced NewEngine function (existing signature preserved)
func NewEngine(config *Config) (*Engine, error) {
	// Existing validation logic preserved
	if config == nil {
		config = &Config{
			MaxRetries:      3,
			RetryDelay:      2 * time.Second,
			Timeout:         30 * time.Second,
			FollowRedirects: true,
			MaxRedirects:    10,
			RateLimit:       1 * time.Second,
			BurstSize:       5,
		}
	}

	// Existing HTTP client setup preserved
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Enhanced with error service and browser manager
	engine := &Engine{
		httpClient:   client,
		config:       config,
		errorService: errors.NewService(),
	}

	// Setup browser automation if configured
	if config.Browser != nil {
		// Convert scraper BrowserConfig to browser package BrowserConfig
		browserConfig := &browser.BrowserConfig{
			Enabled:        config.Browser.Enabled,
			Headless:       config.Browser.Headless,
			UserDataDir:    config.Browser.UserDataDir,
			Timeout:        config.Browser.Timeout,
			ViewportWidth:  config.Browser.ViewportWidth,
			ViewportHeight: config.Browser.ViewportHeight,
			WaitForElement: config.Browser.WaitForElement,
			WaitDelay:      config.Browser.WaitDelay,
			UserAgent:      config.Browser.UserAgent,
			DisableImages:  config.Browser.DisableImages,
			DisableCSS:     config.Browser.DisableCSS,
			DisableJS:      config.Browser.DisableJS,
		}
		
		bm, err := browser.NewBrowserManager(browserConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create browser manager: %w", err)
		}
		engine.browserManager = bm
	}

	// Setup proxy manager if configured
	if config.Proxy != nil {
		// Convert scraper ProxyConfig to proxy package ProxyConfig
		proxyConfig := &proxy.ProxyConfig{
			Enabled:          config.Proxy.Enabled,
			Rotation: func() proxy.RotationStrategy {
				rotation, err := ParseRotationStrategy(config.Proxy.Rotation)
				if err != nil {
					panic(fmt.Sprintf("invalid rotation strategy: %s", err)) // Replace panic with proper error handling if needed
				}
				return rotation
			}(),
			HealthCheck:      config.Proxy.HealthCheck,
			HealthCheckURL:   config.Proxy.HealthCheckURL,
			HealthCheckRate:  config.Proxy.HealthCheckRate,
			Timeout:          config.Proxy.Timeout,
			MaxRetries:       config.Proxy.MaxRetries,
			RetryDelay:       config.Proxy.RetryDelay,
			FailureThreshold: config.Proxy.FailureThreshold,
			RecoveryTime:     config.Proxy.RecoveryTime,
			Providers:        make([]proxy.ProxyProvider, len(config.Proxy.Providers)),
		}
		
		// Convert providers
		for i, provider := range config.Proxy.Providers {
			proxyConfig.Providers[i] = proxy.ProxyProvider{
				Name:     provider.Name,
				Type:     proxy.ProxyType(provider.Type),
				Host:     provider.Host,
				Port:     provider.Port,
				Username: provider.Username,
				Password: provider.Password,
				Weight:   provider.Weight,
				Enabled:  provider.Enabled,
			}
		}
		
		// Convert TLS configuration if present
		if config.Proxy.TLS != nil {
			proxyConfig.TLS = &proxy.TLSConfig{
				InsecureSkipVerify: config.Proxy.TLS.InsecureSkipVerify,
				ServerName:         config.Proxy.TLS.ServerName,
				RootCAs:            config.Proxy.TLS.RootCAs,
				ClientCert:         config.Proxy.TLS.ClientCert,
				ClientKey:          config.Proxy.TLS.ClientKey,
			}
		}
		
		pm := proxy.NewProxyManager(proxyConfig)
		if err := pm.Start(); err != nil {
			return nil, fmt.Errorf("failed to start proxy manager: %w", err)
		}
		engine.proxyManager = pm
	}

	// Existing rate limiter setup preserved
	if config.RateLimit > 0 {
		engine.rateLimiter = NewRateLimiter(config.RateLimit, config.BurstSize)
	}

	return engine, nil
}

// Enhanced Scrape method (existing signature preserved, error handling improved)
func (e *Engine) Scrape(ctx context.Context, url string, extractors []FieldConfig) (*Result, error) {
	result := &Result{
		Data:      make(map[string]interface{}),
		Success:   false,
		Timestamp: time.Now(),
		Errors:    make([]string, 0),
		Warnings:  make([]string, 0),
	}

	// Execute with retry logic
	var doc *goquery.Document
	err := e.errorService.ExecuteWithRetry(ctx, func() error {
		var fetchErr error
		doc, fetchErr = e.fetchDocument(ctx, url)
		return fetchErr
	}, "fetch_document")

	if err != nil {
		result.Error = err
		result.Errors = append(result.Errors, err.Error())
		return result, fmt.Errorf("failed to fetch document: %w", err)
	}

	// Extract fields with error tracking
	successCount := 0
	totalFields := len(extractors)

	for _, extractor := range extractors {
		value, err := e.extractField(doc, extractor)
		if err != nil {
			errorMsg := fmt.Sprintf("Field '%s': %s", extractor.Name, err.Error())
			result.Errors = append(result.Errors, errorMsg)
			
			// Use default value if available and not required
			if !extractor.Required && extractor.Default != nil {
				result.Data[extractor.Name] = extractor.Default
				result.Warnings = append(result.Warnings, 
					fmt.Sprintf("Used default value for field '%s'", extractor.Name))
				successCount++
			}
		} else {
			result.Data[extractor.Name] = value
			successCount++
		}
	}

	// Calculate success metrics
	if totalFields > 0 {
		result.ErrorRate = float64(totalFields-successCount) / float64(totalFields)
		result.Success = successCount > 0 // Partial success if any field extracted
	}

	return result, nil
}

// Enhanced fetchDocument method (existing logic preserved, browser automation added)
func (e *Engine) fetchDocument(ctx context.Context, url string) (*goquery.Document, error) {
	// Existing rate limiting preserved
	if e.rateLimiter != nil {
		e.rateLimiter.Wait()
	}

	// Use browser automation if enabled
	if e.browserManager != nil && e.browserManager.IsEnabled() {
		return e.fetchDocumentWithBrowser(ctx, url)
	}

	// Fallback to existing HTTP client logic
	return e.fetchDocumentWithHTTP(ctx, url)
}

// fetchDocumentWithBrowser uses browser automation to fetch the document
func (e *Engine) fetchDocumentWithBrowser(ctx context.Context, url string) (*goquery.Document, error) {
	html, err := e.browserManager.FetchHTML(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("browser fetch failed: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML from browser: %w", err)
	}

	return doc, nil
}

// fetchDocumentWithHTTP uses HTTP client to fetch the document (existing logic preserved)
func (e *Engine) fetchDocumentWithHTTP(ctx context.Context, url string) (*goquery.Document, error) {
	// Get proxy if proxy manager is enabled
	var proxyInstance *proxy.ProxyInstance
	if e.proxyManager != nil && e.proxyManager.IsEnabled() {
		var err error
		proxyInstance, err = e.proxyManager.GetProxy()
		if err != nil {
			return nil, fmt.Errorf("failed to get proxy: %w", err)
		}
	}

	// Create HTTP client with proxy if available
	client := e.httpClient
	if proxyInstance != nil {
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyInstance.URL),
		}
		client = &http.Client{
			Transport: transport,
			Timeout:   e.config.Timeout,
		}
	}

	// Existing request creation preserved
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Existing header setting preserved
	req.Header.Set("User-Agent", e.getUserAgent())
	for key, value := range e.config.Headers {
		req.Header.Set(key, value)
	}

	// Execute request with proxy-aware client
	resp, err := client.Do(req)
	if err != nil {
		// Report proxy failure if proxy was used
		if proxyInstance != nil {
			e.proxyManager.ReportFailure(proxyInstance, err)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Existing status code handling preserved
	if resp.StatusCode >= 400 {
		// Report proxy failure for client errors when using proxy
		if proxyInstance != nil {
			httpErr := fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
			e.proxyManager.ReportFailure(proxyInstance, httpErr)
		}
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	// Report proxy success if proxy was used
	if proxyInstance != nil {
		e.proxyManager.ReportSuccess(proxyInstance)
	}

	// Existing document parsing preserved
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc, nil
}

// Enhanced extractField method (existing logic preserved, error handling improved)
func (e *Engine) extractField(doc *goquery.Document, extractor FieldConfig) (interface{}, error) {
	selection := doc.Find(extractor.Selector)
	if selection.Length() == 0 {
		return nil, fmt.Errorf("no elements found for selector: %s", extractor.Selector)
	}

	// Existing extraction logic preserved
	switch extractor.Type {
	case "text":
		text := strings.TrimSpace(selection.First().Text())
		if text == "" && extractor.Required {
			return nil, fmt.Errorf("required field is empty")
		}
		return text, nil
		
	case "attr":
		if extractor.Attribute == "" {
			return nil, fmt.Errorf("attribute name required for attr type")
		}
		attr, exists := selection.First().Attr(extractor.Attribute)
		if !exists && extractor.Required {
			return nil, fmt.Errorf("required attribute '%s' not found", extractor.Attribute)
		}
		return attr, nil
		
	case "html":
		html, err := selection.First().Html()
		if err != nil {
			return nil, fmt.Errorf("failed to extract HTML: %w", err)
		}
		return html, nil
		
	case "array", "list":
		var items []string
		selection.Each(func(i int, s *goquery.Selection) {
			items = append(items, strings.TrimSpace(s.Text()))
		})
		return items, nil
		
	default:
		return nil, fmt.Errorf("unsupported extraction type: %s", extractor.Type)
	}
}

// Enhanced getUserAgent method (existing logic preserved)
func (e *Engine) getUserAgent() string {
	// Existing user agent rotation logic preserved
	if len(e.userAgentPool) == 0 {
		return "DataScrapexter/1.0"
	}
	
	ua := e.userAgentPool[e.currentUAIndex]
	e.currentUAIndex = (e.currentUAIndex + 1) % len(e.userAgentPool)
	return ua
}

// GetErrorSummary provides detailed error information
func (e *Engine) GetErrorSummary(result *Result) string {
	if result == nil || len(result.Errors) == 0 {
		return "No errors"
	}
	
	summary := fmt.Sprintf("Encountered %d error(s):\n", len(result.Errors))
	for i, err := range result.Errors {
		summary += fmt.Sprintf("  %d. %s\n", i+1, err)
	}
	
	if len(result.Warnings) > 0 {
		summary += fmt.Sprintf("\nWarnings (%d):\n", len(result.Warnings))
		for i, warning := range result.Warnings {
			summary += fmt.Sprintf("  %d. %s\n", i+1, warning)
		}
	}
	
	return summary
}

// GetUserFriendlyError converts engine errors to user-friendly format
func (e *Engine) GetUserFriendlyError(err error) (title, message string, suggestions []string) {
	return e.errorService.GetUserFriendlyError(err)
}

// Close closes the scraper engine and releases resources
func (e *Engine) Close() error {
	if e.browserManager != nil {
		return e.browserManager.Close()
	}
	return nil
}

// IsBrowserEnabled returns whether browser automation is enabled
func (e *Engine) IsBrowserEnabled() bool {
	return e.browserManager != nil && e.browserManager.IsEnabled()
}
