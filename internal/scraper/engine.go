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
	rateLimiter    *AdaptiveRateLimiter
	
	// Enhanced features: error handling, browser automation, and proxy management
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
			return nil, fmt.Errorf("failed to create browser manager (enabled=%t, headless=%t, timeout=%v): %w", 
				config.Browser.Enabled, config.Browser.Headless, config.Browser.Timeout, err)
		}
		engine.browserManager = bm
	}

	// Setup proxy manager if configured
	if config.Proxy != nil {
		// Convert scraper ProxyConfig to proxy package ProxyConfig
		// Parse rotation strategy
		rotation, err := ParseRotationStrategy(config.Proxy.Rotation)
		if err != nil {
			return nil, fmt.Errorf("invalid rotation strategy: %w", err)
		}
		
		proxyConfig := &proxy.ProxyConfig{
			Enabled:          config.Proxy.Enabled,
			Rotation:         rotation,
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

	// Enhanced rate limiter setup
	if config.RateLimit >= 0 || config.RateLimiter != nil {
		// Validate rate limit duration
		if config.RateLimit < 0 {
			return nil, fmt.Errorf("invalid rate limit duration: %v (must be >= 0)", config.RateLimit)
		}
		var rlConfig *RateLimiterConfig
		if config.RateLimiter != nil {
			rlConfig = config.RateLimiter
		} else {
			// Convert legacy config to new format with production defaults
			rlConfig = &RateLimiterConfig{
				BaseInterval:         config.RateLimit,
				BurstSize:           config.BurstSize,
				Strategy:            StrategyFixed,
				MaxInterval:         config.RateLimit * 10,
				AdaptationRate:      DefaultAdaptationRate,
				BurstRefillRate:     DefaultBurstRefillRate,
				HealthWindow:        DefaultHealthWindow,
				AdaptationThreshold: DefaultAdaptationThreshold,
				ErrorRateThreshold:  DefaultErrorRateThreshold,
				ConsecutiveErrLimit: DefaultConsecutiveErrLimit,
				MinChangeThreshold:  DefaultMinChangeThreshold,
			}
		}
		engine.rateLimiter = NewAdaptiveRateLimiter(rlConfig)
	}

	// Configure error recovery if specified
	if config.ErrorRecovery != nil && config.ErrorRecovery.Enabled {
		// Configure circuit breakers
		for operationName, cbSpec := range config.ErrorRecovery.CircuitBreakers {
			circuitConfig := errors.CircuitBreakerConfig{
				MaxFailures:  cbSpec.MaxFailures,
				ResetTimeout: cbSpec.ResetTimeout,
			}
			engine.errorService.ConfigureCircuitBreaker(operationName, circuitConfig)
		}
		
		// Configure fallbacks
		for operationName, fbSpec := range config.ErrorRecovery.Fallbacks {
			var strategy errors.FallbackStrategy
			switch fbSpec.Strategy {
			case "cached":
				strategy = errors.FallbackCached
			case "default":
				strategy = errors.FallbackDefault
			case "alternative":
				strategy = errors.FallbackAlternative
			case "degrade":
				strategy = errors.FallbackDegrade
			default:
				strategy = errors.FallbackNone
			}
			
			fallbackConfig := errors.FallbackConfig{
				Strategy:     strategy,
				CacheTimeout: fbSpec.CacheTimeout,
				DefaultValue: fbSpec.DefaultValue,
				Alternative:  fbSpec.Alternative,
				Degraded:     fbSpec.Degraded,
			}
			engine.errorService.ConfigureFallback(operationName, fallbackConfig)
		}
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

	// Execute with comprehensive error recovery
	recoveryResult := e.errorService.ExecuteWithRecovery(ctx, "fetch_document", func() (interface{}, error) {
		doc, err := e.fetchDocument(ctx, url)
		return doc, err
	})

	if !recoveryResult.Success {
		result.Error = recoveryResult.OriginalError
		result.Errors = append(result.Errors, recoveryResult.OriginalError.Error())
		if recoveryResult.UsedFallback {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Used fallback strategy: %s", recoveryResult.FallbackType))
		}
		return result, fmt.Errorf("failed to fetch document after %d attempts: %w", recoveryResult.AttemptCount, recoveryResult.OriginalError)
	}

	var doc *goquery.Document
	var ok bool
	if doc, ok = recoveryResult.Result.(*goquery.Document); !ok {
		err := fmt.Errorf("unexpected result type from document fetch")
		result.Error = err
		result.Errors = append(result.Errors, err.Error())
		return result, err
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
	// Enhanced rate limiting with context support
	if e.rateLimiter != nil {
		if err := e.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiting failed: %w", err)
		}
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
		// Report rate limiter failure for adaptive behavior
		if e.rateLimiter != nil {
			e.rateLimiter.ReportError()
		}
		// Report proxy failure if proxy was used
		if proxyInstance != nil {
			e.proxyManager.ReportFailure(proxyInstance, err)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Existing status code handling preserved
	if resp.StatusCode >= 400 {
		// Report rate limiter failure for adaptive behavior
		if e.rateLimiter != nil {
			e.rateLimiter.ReportError()
		}
		// Report proxy failure for client errors when using proxy
		if proxyInstance != nil {
			httpErr := fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
			e.proxyManager.ReportFailure(proxyInstance, httpErr)
		}
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	// Report success for adaptive rate limiting
	if e.rateLimiter != nil {
		e.rateLimiter.ReportSuccess()
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

// GetRateLimiterStats returns current rate limiter statistics
func (e *Engine) GetRateLimiterStats() *RateLimiterStats {
	if e.rateLimiter == nil {
		return nil
	}
	return e.rateLimiter.GetStats()
}

// SetRateLimitStrategy changes the rate limiting strategy
func (e *Engine) SetRateLimitStrategy(strategy RateLimitStrategy) {
	if e.rateLimiter != nil {
		e.rateLimiter.SetStrategy(strategy)
	}
}

// ResetRateLimiter resets rate limiter statistics
func (e *Engine) ResetRateLimiter() {
	if e.rateLimiter != nil {
		e.rateLimiter.Reset()
	}
}

// ConfigureErrorRecovery configures error recovery mechanisms
func (e *Engine) ConfigureErrorRecovery(operationName string, circuitConfig *errors.CircuitBreakerConfig, fallbackConfig *errors.FallbackConfig) {
	if e.errorService == nil {
		return
	}
	
	if circuitConfig != nil {
		e.errorService.ConfigureCircuitBreaker(operationName, *circuitConfig)
	}
	
	if fallbackConfig != nil {
		e.errorService.ConfigureFallback(operationName, *fallbackConfig)
	}
}

// GetErrorRecoveryStats returns error recovery statistics
func (e *Engine) GetErrorRecoveryStats() map[string]interface{} {
	if e.errorService == nil {
		return nil
	}
	
	return map[string]interface{}{
		"circuit_breakers": e.errorService.GetCircuitBreakerStats(),
		"cache":           e.errorService.GetCacheStats(),
	}
}

// ResetErrorRecovery resets all error recovery mechanisms
func (e *Engine) ResetErrorRecovery() {
	if e.errorService != nil {
		e.errorService.ClearCache()
		// Reset circuit breakers by getting their names and resetting each
		stats := e.errorService.GetCircuitBreakerStats()
		for name := range stats {
			e.errorService.ResetCircuitBreaker(name)
		}
	}
}

// ScrapeWithPagination scrapes multiple pages based on pagination configuration
func (e *Engine) ScrapeWithPagination(ctx context.Context, baseURL string, extractors []FieldConfig) (*PaginationResult, error) {
	if e.config.Pagination == nil || !e.config.Pagination.Enabled {
		// If pagination is disabled, just scrape the single page
		result, err := e.Scrape(ctx, baseURL, extractors)
		if err != nil {
			return nil, fmt.Errorf("failed to scrape single page: %w", err)
		}
		
		return &PaginationResult{
			Pages: []ScrapingResult{{
				URL:        baseURL,
				StatusCode: 200,
				Data:       result.Data,
				Success:    result.Success,
				Errors:     result.Errors,
			}},
			TotalPages:     1,
			ProcessedPages: 1,
			Success:        result.Success,
			Duration:       0,
			StartTime:      time.Now(),
			EndTime:        time.Now(),
		}, nil
	}

	// Create pagination manager
	paginationManager, err := NewPaginationManager(*e.config.Pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to create pagination manager: %w", err)
	}

	startTime := time.Now()
	results := make([]ScrapingResult, 0)
	errors := make([]string, 0)
	
	currentURL := baseURL
	pageNum := 0  // Start from 0 for offset-based pagination
	maxPages := e.config.Pagination.MaxPages
	if maxPages <= 0 {
		maxPages = 10 // Default safety limit
	}

	for pageNum < maxPages {
		// Handle offset-based pagination separately
		if e.config.Pagination.Type == PaginationTypeOffset {
			// Calculate the next URL directly using the offset
			offset := pageNum * e.config.Pagination.PageSize
			offsetParam := e.config.Pagination.OffsetParam
			limitParam := e.config.Pagination.LimitParam
			if offsetParam == "" {
				offsetParam = "offset"
			}
			if limitParam == "" {
				limitParam = "limit"
			}
			currentURL = fmt.Sprintf("%s?%s=%d&%s=%d", baseURL, offsetParam, offset, limitParam, e.config.Pagination.PageSize)
		} else if pageNum > 0 {
			// For other pagination types, fetch the document to determine the next URL
			doc, err := e.fetchDocument(ctx, currentURL)
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to fetch document for pagination on page %d: %v", pageNum+1, err)
				errors = append(errors, errorMsg)
				break
			}

			// Check if pagination is complete
			if paginationManager.IsComplete(ctx, currentURL, doc, pageNum) {
				break
			}

			// Get next URL
			nextURL, err := paginationManager.GetNextURL(ctx, currentURL, doc, pageNum)
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to get next URL on page %d: %v", pageNum+1, err)
				errors = append(errors, errorMsg)
				break
			}
			
			if nextURL == "" {
				break // No more pages
			}

			currentURL = nextURL
		}

		// Scrape current page
		result, err := e.Scrape(ctx, currentURL, extractors)
		if err != nil {
			errorMsg := fmt.Sprintf("Page %d failed: %v", pageNum+1, err)
			errors = append(errors, errorMsg)
			
			if !e.config.Pagination.ContinueOnError {
				break
			}
			pageNum++
			continue
		}

		// Convert to ScrapingResult format
		scrapingResult := ScrapingResult{
			URL:        currentURL,
			StatusCode: 200,
			Data:       result.Data,
			Success:    result.Success,
			Errors:     result.Errors,
		}
		results = append(results, scrapingResult)

		pageNum++

		// Add delay between pages if configured
		if e.config.Pagination.DelayBetweenPages > 0 {
			time.Sleep(e.config.Pagination.DelayBetweenPages)
		}
	}

	return &PaginationResult{
		Pages:          results,
		TotalPages:     len(results),
		ProcessedPages: len(results),
		Success:        len(results) > 0,
		Errors:         errors,
		Duration:       time.Since(startTime),
		StartTime:      startTime,
		EndTime:        time.Now(),
	}, nil
}
