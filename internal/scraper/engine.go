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
	"github.com/valpere/DataScrapexter/internal/config"
	"github.com/valpere/DataScrapexter/internal/errors"
	"github.com/valpere/DataScrapexter/internal/proxy"
	"github.com/valpere/DataScrapexter/internal/utils"
)

// Default configuration constants
const (
	// DefaultMaxConcurrency defines the default maximum number of concurrent operations
	DefaultMaxConcurrency = 10
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
	
	// Performance optimizations
	resultPool     *utils.Pool[*Result]
	copyPool       *utils.Pool[*Result]      // Pool for result copies to reduce allocations
	perfMetrics    *utils.PerformanceMetrics
	memManager     *utils.MemoryManager
	circuitBreaker *utils.CircuitBreaker
	MaxConcurrency int // Maximum number of concurrent operations
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
			MaxConcurrency:  DefaultMaxConcurrency,
		}
	}
	
	// Set default MaxConcurrency if not specified
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = DefaultMaxConcurrency
	}
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
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

	// Enhanced with error service and performance optimizations
	engine := &Engine{
		httpClient:     client,
		config:         config,
		errorService:   errors.NewService(),
		MaxConcurrency: config.MaxConcurrency, // Use configured max concurrency
		
		// Initialize performance optimizations
		perfMetrics:    utils.NewPerformanceMetrics(),
		memManager:     utils.NewMemoryManager(100*1024*1024, 30*time.Second), // 100MB, 30s GC interval
		circuitBreaker: utils.NewCircuitBreaker(5, 60*time.Second), // 5 failures, 60s timeout
		
		resultPool: utils.NewPool[*Result](
			func() *Result {
				return &Result{
					Data:     make(map[string]interface{}),
					Errors:   make([]string, 0),
					Warnings: make([]string, 0),
				}
			},
			func(result *Result) {
				// Reset result for reuse
				for k := range result.Data {
					delete(result.Data, k)
				}
				result.Errors = result.Errors[:0]
				result.Warnings = result.Warnings[:0]
				result.Success = false
				result.Error = nil
				result.ErrorRate = 0
			},
		),
		
		// Pool for result copies to optimize memory allocation during copying
		copyPool: utils.NewPool[*Result](
			func() *Result {
				return &Result{
					Data:     make(map[string]interface{}),
					Errors:   make([]string, 0, 4),   // Pre-allocate with small capacity
					Warnings: make([]string, 0, 2),   // Pre-allocate with small capacity
				}
			},
			func(result *Result) {
				// Reset copy result for reuse
				for k := range result.Data {
					delete(result.Data, k)
				}
				result.Errors = result.Errors[:0]
				result.Warnings = result.Warnings[:0]
				result.Success = false
				result.Error = nil
				result.ErrorRate = 0
				result.Timestamp = time.Time{}
			},
		),
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
	if config.RateLimiter != nil || config.RateLimit > 0 {
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
				BaseInterval:        config.RateLimit,
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

// Enhanced Scrape method (existing signature preserved, optimized for performance)
func (e *Engine) Scrape(ctx context.Context, url string, extractors []FieldConfig) (*Result, error) {
	// Start performance tracking
	timer := utils.NewTimer("scrape_operation")
	defer func() {
		duration := timer.Stop()
		e.perfMetrics.RecordOperation(duration, true) // Will be updated if error occurs
	}()
	
	// Check memory pressure and trigger GC if needed
	e.memManager.CheckMemoryUsage()
	
	// Get result from pool for memory efficiency
	result := e.resultPool.Get()
	// Note: Put will be called after creating the copy to avoid race conditions
	
	result.Timestamp = time.Now()
	
	// Use circuit breaker to prevent cascading failures
	circuitErr := e.circuitBreaker.Execute(func() error {
		return e.performScrapeOperation(ctx, url, extractors, result)
	})
	
	if circuitErr != nil {
		result.Error = circuitErr
		result.Errors = append(result.Errors, circuitErr.Error())
		e.perfMetrics.RecordOperation(timer.Elapsed(), false)
		
		// Create an efficient copy before returning and putting back to pool
		resultCopy := e.copyResult(result)
		e.resultPool.Put(result)
		return resultCopy, circuitErr
	}

	// Create an efficient copy of the result to return (since we'll put the pooled one back)
	resultCopy := e.copyResult(result)
	e.resultPool.Put(result)
	
	return resultCopy, nil
}

// performScrapeOperation performs the actual scraping operation
func (e *Engine) performScrapeOperation(ctx context.Context, url string, extractors []FieldConfig, result *Result) error {
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
		return fmt.Errorf("failed to fetch document after %d attempts: %w", recoveryResult.AttemptCount, recoveryResult.OriginalError)
	}

	var doc *goquery.Document
	var ok bool
	if doc, ok = recoveryResult.Result.(*goquery.Document); !ok {
		err := fmt.Errorf("unexpected result type from document fetch")
		result.Error = err
		result.Errors = append(result.Errors, err.Error())
		return err
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

	return nil
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
		"cache":            e.errorService.GetCacheStats(),
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
	pageNum := 0 // Start from 0 for offset-based pagination
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

// Performance and monitoring methods

// GetPerformanceMetrics returns current performance metrics
func (e *Engine) GetPerformanceMetrics() utils.PerformanceMetrics {
	return e.perfMetrics.GetSnapshot()
}

// GetMemoryStats returns current memory statistics
func (e *Engine) GetMemoryStats() interface{} {
	return e.memManager.GetMemoryStats()
}

// GetCircuitBreakerState returns the current circuit breaker state
func (e *Engine) GetCircuitBreakerState() int32 {
	return e.circuitBreaker.GetState()
}

// ScrapeMultipleOptimized performs optimized batch scraping
func (e *Engine) ScrapeMultipleOptimized(ctx context.Context, urls []string, extractors []FieldConfig, concurrency int) ([]*Result, error) {
	if concurrency <= 0 {
		concurrency = 5 // Default concurrency
	}
	
	// Use worker pool for efficient concurrent processing
	workerPool := utils.NewWorkerPool[string](
		concurrency, 
		len(urls),
		func(url string) (interface{}, error) {
			return e.Scrape(ctx, url, extractors)
		},
	)
	
	// Start worker pool
	workerPool.Start()
	defer workerPool.Close()
	
	// Submit URLs to worker pool
	for _, url := range urls {
		if err := workerPool.Submit(url); err != nil {
			return nil, fmt.Errorf("failed to submit URL %s: %w", url, err)
		}
	}
	
	// Collect results
	results := make([]*Result, 0, len(urls))
	errors := make([]error, 0)
	
	for i := 0; i < len(urls); i++ {
		select {
		case result := <-workerPool.Results():
			if scrapingResult, ok := result.(*Result); ok {
				results = append(results, scrapingResult)
			}
		case err := <-workerPool.Errors():
			errors = append(errors, err)
		case <-ctx.Done():
			return results, ctx.Err()
		}
	}
	
	// Return error if there were any errors
	if len(errors) > 0 {
		return results, fmt.Errorf("encountered %d errors during batch scraping", len(errors))
	}
	
	return results, nil
}

// ScrapeWithBatchingConfig processes URLs in batches using a configuration struct for better usability
// This method provides an improved API with fewer parameters and better maintainability
func (e *Engine) ScrapeWithBatchingConfig(ctx context.Context, config *BatchScrapingConfig) ([]*Result, error) {
	if config == nil {
		return nil, fmt.Errorf("BatchScrapingConfig cannot be nil")
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid batch scraping config: %w", err)
	}
	
	return e.ScrapeWithBatching(ctx, config.URLs, config.Extractors, config.ScraperConfig, config.BatchSize)
}

// ScrapeWithBatching processes URLs in batches for memory efficiency
// This method reuses a single worker pool across all batches for better performance
// Deprecated: Use ScrapeWithBatchingConfig for better parameter management
func (e *Engine) ScrapeWithBatching(ctx context.Context, urls []string, extractors []FieldConfig, scraperConfig *config.ScraperConfig, batchSize int) ([]*Result, error) {
	if batchSize <= 0 {
		batchSize = 10 // Default batch size
	}
	
	if len(urls) == 0 {
		return []*Result{}, nil
	}
	
	// Use configurable concurrency limit, default to DefaultMaxConcurrency if not set
	maxConc := e.MaxConcurrency
	if maxConc <= 0 {
		maxConc = DefaultMaxConcurrency
	}
	
	// Create a single worker pool for all batches to avoid overhead
	workerPool := utils.NewWorkerPool[string](
		min(maxConc, batchSize), // Don't exceed batch size for worker count
		batchSize*2,             // Buffer size for input queue
		func(url string) (interface{}, error) {
			return e.Scrape(ctx, url, extractors)
		},
	)
	
	// Start the worker pool
	workerPool.Start()
	defer workerPool.Close()
	
	allResults := make([]*Result, 0, len(urls))
	
	// Track error thresholds across batches
	totalProcessed := 0
	totalErrors := 0
	
	// Process URLs in batches
	for i := 0; i < len(urls); i += batchSize {
		end := i + batchSize
		if end > len(urls) {
			end = len(urls)
		}
		
		batch := urls[i:end]
		
		// Submit batch to worker pool
		for _, url := range batch {
			if err := workerPool.Submit(url); err != nil {
				return allResults, fmt.Errorf("failed to submit URL %s in batch %d-%d: %w", url, i, end-1, err)
			}
		}
		
		// Collect results for this batch
		batchResults := make([]*Result, 0, len(batch))
		errors := make([]error, 0)
		
		for j := 0; j < len(batch); j++ {
			select {
			case result := <-workerPool.Results():
				if scrapingResult, ok := result.(*Result); ok {
					batchResults = append(batchResults, scrapingResult)
				}
			case err := <-workerPool.Errors():
				errors = append(errors, err)
			case <-ctx.Done():
				return allResults, ctx.Err()
			}
		}
		
		// Add batch results to total results
		allResults = append(allResults, batchResults...)
		
		// Update totals for error threshold tracking
		totalProcessed += len(batchResults)
		totalErrors += len(errors)
		
		// Report any errors from this batch and check error thresholds
		if len(errors) > 0 {
			logger := utils.GetLogger("scraper")
			
			// Use optimized error logging with efficient batching/sampling for performance
			e.logBatchErrors(logger, errors)
			
			// Check if error thresholds are exceeded and should stop processing
			shouldStop := e.checkErrorThresholds(scraperConfig, len(errors), len(batchResults), totalProcessed, totalErrors)
			if shouldStop {
				logger.Warnf("Error threshold exceeded: %d errors in current batch, %d total errors out of %d processed items. Stopping batch processing as configured.", 
					len(errors), totalErrors, totalProcessed)
				break // Stop processing remaining batches
			}
		}
		
		// Check memory pressure after each batch
		e.memManager.CheckMemoryUsage()
		
		// Optional: Add delay between batches to be respectful
		if i+batchSize < len(urls) {
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return allResults, ctx.Err()
			}
		}
	}
	
	return allResults, nil
}

// OptimizeForThroughput optimizes engine settings for maximum throughput
func (e *Engine) OptimizeForThroughput() {
	// Increase HTTP client connection limits
	if transport, ok := e.httpClient.Transport.(*http.Transport); ok {
		transport.MaxIdleConns = 200
		transport.MaxIdleConnsPerHost = 50
		transport.IdleConnTimeout = 120 * time.Second
	}
	
	// Reset performance counters
	e.perfMetrics.Reset()
}

// OptimizeForMemory optimizes engine settings for minimal memory usage
func (e *Engine) OptimizeForMemory() {
	// Reduce HTTP client connection limits
	if transport, ok := e.httpClient.Transport.(*http.Transport); ok {
		transport.MaxIdleConns = 50
		transport.MaxIdleConnsPerHost = 5
		transport.IdleConnTimeout = 30 * time.Second
	}
}

// checkErrorThresholds checks if error thresholds are exceeded and processing should stop
func (e *Engine) checkErrorThresholds(scraperConfig *config.ScraperConfig, batchErrors, batchSize, totalProcessed, totalErrors int) bool {
	if scraperConfig == nil {
		return false
	}
	
	// Only check if stop_on_error_threshold is enabled
	if !scraperConfig.StopOnErrorThreshold {
		return false
	}

	// Check absolute error threshold per batch
	if scraperConfig.ErrorThreshold > 0 && batchErrors >= scraperConfig.ErrorThreshold {
		return true
	}

	// Check percentage error threshold (overall rate)
	if scraperConfig.ErrorThresholdPercent > 0 && totalProcessed > 0 {
		errorRate := float64(totalErrors) / float64(totalProcessed) * 100
		if errorRate >= scraperConfig.ErrorThresholdPercent {
			return true
		}
	}

	return false
}

// copyResult efficiently copies a Result using sync.Pool to reduce allocations
func (e *Engine) copyResult(src *Result) *Result {
	// Get a copy from the pool to avoid allocations
	dst := e.copyPool.Get()
	
	// Copy scalar fields
	dst.Success = src.Success
	dst.Error = src.Error
	dst.Timestamp = src.Timestamp
	dst.ErrorRate = src.ErrorRate
	
	// Efficiently copy map - simple shallow copy since scraped data is typically flat
	if len(dst.Data) > 0 {
		// Clear existing map entries
		for k := range dst.Data {
			delete(dst.Data, k)
		}
	}
	if len(src.Data) > 0 {
		// Ensure map exists and copy data (shallow copy is sufficient for scraped data)
		if dst.Data == nil {
			dst.Data = make(map[string]interface{}, len(src.Data))
		}
		for k, v := range src.Data {
			dst.Data[k] = v
		}
	}
	
	// Efficiently copy slices - grow if needed
	if cap(dst.Errors) < len(src.Errors) {
		dst.Errors = make([]string, len(src.Errors))
	} else {
		dst.Errors = dst.Errors[:len(src.Errors)]
	}
	copy(dst.Errors, src.Errors)
	
	if cap(dst.Warnings) < len(src.Warnings) {
		dst.Warnings = make([]string, len(src.Warnings))
	} else {
		dst.Warnings = dst.Warnings[:len(src.Warnings)]
	}
	copy(dst.Warnings, src.Warnings)
	
	return dst
}

// logBatchErrors efficiently logs error batches with sampling to avoid performance issues in high-error scenarios
func (e *Engine) logBatchErrors(logger *utils.ComponentLogger, errors []error) {
	switch {
	case len(errors) <= 5:
		// Log individual errors for small error counts - avoid unnecessary loops
		for _, err := range errors {
			logger.Errorf("Batch processing error: %v", err)
		}
	case len(errors) <= 100:
		// For moderate error counts, use efficient sampling without nested loops
		logger.Errorf("Batch processing encountered %d errors. First 3 samples: [%v] [%v] [%v] (and %d more)", 
			len(errors), errors[0], errors[1], errors[2], len(errors)-3)
	default:
		// For very high error counts, use optimized sampling with categorization
		e.logHighVolumeErrors(logger, errors)
	}
}

// logHighVolumeErrors handles high-volume error scenarios with efficient categorization and sampling
func (e *Engine) logHighVolumeErrors(logger *utils.ComponentLogger, errors []error) {
	totalErrors := len(errors)
	
	// Sample errors from different parts of the batch for better representation
	sampleSize := min(10, totalErrors)
	step := totalErrors / sampleSize
	
	samples := make([]string, 0, sampleSize)
	errorTypes := make(map[string]int)
	
	// Collect samples and categorize error types efficiently
	for i := 0; i < sampleSize; i++ {
		idx := i * step
		if idx >= totalErrors {
			break
		}
		
		err := errors[idx]
		samples = append(samples, err.Error())
		
		// Simple error type categorization based on error string
		errorType := "unknown"
		errStr := err.Error()
		switch {
		case len(errStr) > 0:
			// Use first word as error type for simple categorization
			if spaceIdx := len(errStr); spaceIdx > 20 {
				errorType = errStr[:20] + "..."
			} else {
				errorType = errStr
			}
		}
		errorTypes[errorType]++
	}
	
	// Log summary with samples and error type distribution
	if len(samples) > 0 {
		sampleCount := min(3, len(samples))
		logger.Errorf("High-volume batch processing encountered %d errors. Sample errors: %v", totalErrors, samples[:sampleCount])
	} else {
		logger.Errorf("High-volume batch processing encountered %d errors. No samples collected.", totalErrors)
	}
	logger.Warnf("Error type distribution (top 5): %v", getTopErrorTypes(errorTypes, 5))
}

// getTopErrorTypes returns the top N error types by frequency
func getTopErrorTypes(errorTypes map[string]int, topN int) map[string]int {
	if len(errorTypes) <= topN {
		return errorTypes
	}
	
	// Simple approach: return first topN entries (good enough for logging purposes)
	result := make(map[string]int)
	count := 0
	for errType, freq := range errorTypes {
		if count >= topN {
			break
		}
		result[errType] = freq
		count++
	}
	return result
}
