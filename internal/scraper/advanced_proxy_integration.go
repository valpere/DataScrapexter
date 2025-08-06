// internal/scraper/advanced_proxy_integration.go - Integration of advanced proxy rotation with scraper engine
package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/valpere/DataScrapexter/internal/proxy"
	"github.com/valpere/DataScrapexter/internal/utils"
)

var advancedProxyLogger = utils.NewComponentLogger("advanced-proxy-integration")

// AdvancedProxyEngine extends the basic Engine with advanced proxy capabilities
type AdvancedProxyEngine struct {
	*Engine                                    // Embed basic engine
	advancedProxyManager *proxy.AdvancedProxyManager
	proxyMonitor        *proxy.ProxyMonitor
	config              *AdvancedProxyConfig
}

// AdvancedProxyConfig extends scraper config with advanced proxy settings
type AdvancedProxyConfig struct {
	Enabled             bool                              `yaml:"enabled" json:"enabled"`
	Strategy            proxy.AdvancedRotationStrategy    `yaml:"strategy" json:"strategy"`
	GeographicPref      []string                          `yaml:"geographic_preference,omitempty" json:"geographic_preference,omitempty"`
	PerformanceThresholds *proxy.PerformanceThresholds    `yaml:"performance_thresholds,omitempty" json:"performance_thresholds,omitempty"`
	Groups              []proxy.ProxyGroup                `yaml:"groups,omitempty" json:"groups,omitempty"`
	LoadBalancing       *proxy.LoadBalancingConfig        `yaml:"load_balancing,omitempty" json:"load_balancing,omitempty"`
	MLConfig            *proxy.MLPredictionConfig         `yaml:"ml_config,omitempty" json:"ml_config,omitempty"`
	CostOptimization    *proxy.CostOptimizationConfig     `yaml:"cost_optimization,omitempty" json:"cost_optimization,omitempty"`
	Monitoring          *proxy.MonitoringConfig           `yaml:"monitoring,omitempty" json:"monitoring,omitempty"`
	Providers           []proxy.AdvancedProxyProvider     `yaml:"providers,omitempty" json:"providers,omitempty"`
	AutoFailover        bool                              `yaml:"auto_failover" json:"auto_failover"`
	HealthCheckInterval time.Duration                     `yaml:"health_check_interval" json:"health_check_interval"`
	RetryStrategy       *AdvancedRetryStrategy            `yaml:"retry_strategy,omitempty" json:"retry_strategy,omitempty"`
}

// AdvancedRetryStrategy defines advanced retry behavior with proxy rotation
type AdvancedRetryStrategy struct {
	MaxRetries           int                               `yaml:"max_retries" json:"max_retries"`
	InitialDelay         time.Duration                     `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay             time.Duration                     `yaml:"max_delay" json:"max_delay"`
	BackoffMultiplier    float64                           `yaml:"backoff_multiplier" json:"backoff_multiplier"`
	RotateOnFailure      bool                              `yaml:"rotate_on_failure" json:"rotate_on_failure"`
	RotateOnStatusCodes  []int                             `yaml:"rotate_on_status_codes,omitempty" json:"rotate_on_status_codes,omitempty"`
	FallbackStrategy     proxy.AdvancedRotationStrategy    `yaml:"fallback_strategy" json:"fallback_strategy"`
	// CircuitBreakerConfig would be added when circuit breaker is implemented
}

// ScrapingContext contains context for a scraping operation with advanced proxy info
type ScrapingContext struct {
	Context            context.Context
	TargetURL          string
	RequestHeaders     map[string]string
	UserAgent          string
	ProxyInstance      *proxy.AdvancedProxyInstance
	AttemptCount       int
	TotalAttempts      int
	StartTime          time.Time
	LastError          error
	LastStatusCode     int
	DataQualityScore   float64
	GeographicHint     *proxy.GeographicLocation
	PerformanceHint    *proxy.PerformanceMetrics
	RetryDelay         time.Duration
}

// RequestMetrics contains detailed metrics for a scraping request
type RequestMetrics struct {
	ProxyName        string                      `json:"proxy_name"`
	TargetURL        string                      `json:"target_url"`
	StartTime        time.Time                   `json:"start_time"`
	EndTime          time.Time                   `json:"end_time"`
	Duration         time.Duration               `json:"duration"`
	Success          bool                        `json:"success"`
	StatusCode       int                         `json:"status_code"`
	ResponseSize     int64                       `json:"response_size"`
	RequestSize      int64                       `json:"request_size"`
	DataQuality      float64                     `json:"data_quality"`
	ErrorType        string                      `json:"error_type,omitempty"`
	RetryCount       int                         `json:"retry_count"`
	ProxyLatency     time.Duration               `json:"proxy_latency"`
	DNSLookupTime    time.Duration               `json:"dns_lookup_time"`
	ConnectionTime   time.Duration               `json:"connection_time"`
	TLSHandshakeTime time.Duration               `json:"tls_handshake_time"`
	Geographic       *proxy.GeographicLocation   `json:"geographic,omitempty"`
	UserAgent        string                      `json:"user_agent"`
	Headers          map[string]string           `json:"headers,omitempty"`
}

// NewAdvancedProxyEngine creates an enhanced scraper engine with advanced proxy capabilities
func NewAdvancedProxyEngine(config *Config, advancedConfig *AdvancedProxyConfig) (*AdvancedProxyEngine, error) {
	// Create base engine
	baseEngine, err := NewEngine(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base engine: %w", err)
	}

	if advancedConfig == nil || !advancedConfig.Enabled {
		// Return basic engine wrapped as advanced engine
		return &AdvancedProxyEngine{
			Engine: baseEngine,
			config: &AdvancedProxyConfig{Enabled: false},
		}, nil
	}

	// Create advanced proxy configuration
	proxyConfig := &proxy.AdvancedProxyConfig{
		ProxyConfig: proxy.ProxyConfig{
			Enabled:          true,
			HealthCheck:      advancedConfig.HealthCheckInterval > 0,
			HealthCheckRate:  advancedConfig.HealthCheckInterval,
			Timeout:          config.Timeout,
			MaxRetries:       3,
			RetryDelay:       time.Second,
			FailureThreshold: 5,
			RecoveryTime:     10 * time.Minute,
		},
		AdvancedStrategy:      advancedConfig.Strategy,
		GeographicPreference:  advancedConfig.GeographicPref,
		PerformanceThresholds: advancedConfig.PerformanceThresholds,
		Groups:                advancedConfig.Groups,
		LoadBalancing:         advancedConfig.LoadBalancing,
		MLConfig:              advancedConfig.MLConfig,
		CostOptimization:      advancedConfig.CostOptimization,
		AdvancedProviders:     advancedConfig.Providers,
	}

	// Copy basic proxy providers to advanced providers if needed
	if config.Proxy != nil {
		for _, basicProvider := range config.Proxy.Providers {
			advancedProvider := proxy.AdvancedProxyProvider{
				ProxyProvider: proxy.ProxyProvider{
					Name:     basicProvider.Name,
					Type:     proxy.ProxyType(basicProvider.Type),
					Host:     basicProvider.Host,
					Port:     basicProvider.Port,
					Username: basicProvider.Username,
					Password: basicProvider.Password,
					Weight:   basicProvider.Weight,
					Enabled:  basicProvider.Enabled,
				},
				Performance: &proxy.PerformanceMetrics{
					LastMeasured: time.Now(),
				},
			}
			proxyConfig.AdvancedProviders = append(proxyConfig.AdvancedProviders, advancedProvider)
		}
	}

	// Create advanced proxy manager
	advancedManager := proxy.NewAdvancedProxyManager(proxyConfig)

	// Create proxy monitor
	var monitor *proxy.ProxyMonitor
	if advancedConfig.Monitoring != nil && advancedConfig.Monitoring.Enabled {
		monitor = proxy.NewProxyMonitor(advancedConfig.Monitoring, advancedManager)
	}

	return &AdvancedProxyEngine{
		Engine:               baseEngine,
		advancedProxyManager: advancedManager,
		proxyMonitor:        monitor,
		config:              advancedConfig,
	}, nil
}

// Start starts the advanced proxy engine
func (ape *AdvancedProxyEngine) Start(ctx context.Context) error {
	if !ape.config.Enabled {
		return nil
	}

	advancedProxyLogger.Info("Starting advanced proxy engine")

	// Start proxy monitor if enabled
	if ape.proxyMonitor != nil {
		if err := ape.proxyMonitor.Start(ctx); err != nil {
			return fmt.Errorf("failed to start proxy monitor: %w", err)
		}
	}

	return nil
}

// Stop stops the advanced proxy engine
func (ape *AdvancedProxyEngine) Stop() error {
	if !ape.config.Enabled {
		return nil
	}

	advancedProxyLogger.Info("Stopping advanced proxy engine")

	// Stop proxy monitor
	if ape.proxyMonitor != nil {
		if err := ape.proxyMonitor.Stop(); err != nil {
			advancedProxyLogger.Warn(fmt.Sprintf("Error stopping proxy monitor: %v", err))
		}
	}

	return nil
}

// ScrapeWithAdvancedProxy performs scraping with advanced proxy selection and monitoring
func (ape *AdvancedProxyEngine) ScrapeWithAdvancedProxy(ctx context.Context, targetURL string, options *ScrapeOptions) (*Result, error) {
	if !ape.config.Enabled {
		// Fall back to basic scraping
		return ape.Engine.Scrape(ctx, targetURL, []FieldConfig{})
	}

	// Create scraping context
	scrapeCtx := &ScrapingContext{
		Context:      ctx,
		TargetURL:    targetURL,
		StartTime:    time.Now(),
		AttemptCount: 1,
	}

	// Apply options if provided
	if options != nil {
		scrapeCtx.RequestHeaders = options.Headers
		scrapeCtx.UserAgent = options.UserAgent
		scrapeCtx.GeographicHint = options.GeographicHint
		scrapeCtx.PerformanceHint = options.PerformanceHint
		if options.MaxRetries > 0 {
			scrapeCtx.TotalAttempts = options.MaxRetries
		} else {
			scrapeCtx.TotalAttempts = 3 // Default
		}
	}

	// Execute scraping with retry logic
	return ape.executeWithRetry(scrapeCtx)
}

// ScrapeOptions provides options for advanced scraping
type ScrapeOptions struct {
	Headers         map[string]string           `json:"headers,omitempty"`
	UserAgent       string                      `json:"user_agent,omitempty"`
	MaxRetries      int                         `json:"max_retries,omitempty"`
	GeographicHint  *proxy.GeographicLocation   `json:"geographic_hint,omitempty"`
	PerformanceHint *proxy.PerformanceMetrics   `json:"performance_hint,omitempty"`
	Strategy        *proxy.AdvancedRotationStrategy `json:"strategy,omitempty"`
	RequiredTags    []string                    `json:"required_tags,omitempty"`
	ExcludedTags    []string                    `json:"excluded_tags,omitempty"`
	CostBudget      float64                     `json:"cost_budget,omitempty"`
}

// executeWithRetry executes scraping with advanced retry logic and proxy rotation
func (ape *AdvancedProxyEngine) executeWithRetry(scrapeCtx *ScrapingContext) (*Result, error) {
	var lastErr error
	var lastResult *Result

	for attempt := 1; attempt <= scrapeCtx.TotalAttempts; attempt++ {
		scrapeCtx.AttemptCount = attempt

		// Select proxy for this attempt
		proxyInstance, err := ape.advancedProxyManager.GetAdvancedProxy(scrapeCtx.TargetURL)
		if err != nil {
			advancedProxyLogger.Warn(fmt.Sprintf("Failed to get proxy for attempt %d: %v", attempt, err))
			if attempt == scrapeCtx.TotalAttempts {
				return nil, fmt.Errorf("exhausted all proxies: %w", err)
			}
			time.Sleep(ape.calculateRetryDelay(attempt))
			continue
		}

		scrapeCtx.ProxyInstance = proxyInstance
		advancedProxyLogger.Debug(fmt.Sprintf("Attempt %d using proxy %s for %s", 
			attempt, proxyInstance.Provider.Name, scrapeCtx.TargetURL))

		// Execute the scraping request
		result, metrics, err := ape.executeSingleRequest(scrapeCtx)

		// Record metrics regardless of success/failure
		if ape.proxyMonitor != nil && metrics != nil {
			dataPoint := proxy.MetricDataPoint{
				Timestamp:    metrics.StartTime,
				Latency:      metrics.Duration,
				Success:      metrics.Success,
				Cost:         0, // Would be calculated based on proxy cost
				DataQuality:  metrics.DataQuality,
				ErrorType:    metrics.ErrorType,
				RequestSize:  metrics.RequestSize,
				ResponseSize: metrics.ResponseSize,
				TargetURL:    metrics.TargetURL,
				UserAgent:    metrics.UserAgent,
			}
			ape.proxyMonitor.RecordRequest(proxyInstance.Provider.Name, dataPoint)
		}

		if err == nil && result != nil && result.Success {
			// Success - report to proxy manager
			ape.advancedProxyManager.ReportAdvancedSuccess(proxyInstance, metrics.Duration, metrics.DataQuality)
			advancedProxyLogger.Debug(fmt.Sprintf("Successfully scraped %s with proxy %s in %v", 
				scrapeCtx.TargetURL, proxyInstance.Provider.Name, metrics.Duration))
			return result, nil
		}

		// Handle failure
		lastErr = err
		lastResult = result
		scrapeCtx.LastError = err
		if metrics != nil {
			scrapeCtx.LastStatusCode = metrics.StatusCode
		}

		// Determine error type for reporting
		errorType := ape.categorizeError(err, scrapeCtx.LastStatusCode)
		ape.advancedProxyManager.ReportAdvancedFailure(proxyInstance, err, errorType)

		// Check if we should retry or fail fast
		if !ape.shouldRetry(err, scrapeCtx.LastStatusCode, attempt, scrapeCtx.TotalAttempts) {
			advancedProxyLogger.Debug(fmt.Sprintf("Not retrying after attempt %d for %s: %s", 
				attempt, scrapeCtx.TargetURL, errorType))
			break
		}

		// Calculate and apply retry delay
		delay := ape.calculateRetryDelay(attempt)
		scrapeCtx.RetryDelay = delay
		advancedProxyLogger.Debug(fmt.Sprintf("Retrying in %v (attempt %d/%d) for %s", 
			delay, attempt+1, scrapeCtx.TotalAttempts, scrapeCtx.TargetURL))
		
		select {
		case <-scrapeCtx.Context.Done():
			return nil, scrapeCtx.Context.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All attempts exhausted
	if lastResult != nil {
		lastResult.Error = lastErr
		return lastResult, lastErr
	}
	
	return nil, fmt.Errorf("all %d attempts failed, last error: %w", scrapeCtx.TotalAttempts, lastErr)
}

// executeSingleRequest executes a single scraping request with detailed metrics collection
func (ape *AdvancedProxyEngine) executeSingleRequest(scrapeCtx *ScrapingContext) (*Result, *RequestMetrics, error) {
	startTime := time.Now()
	
	metrics := &RequestMetrics{
		ProxyName:  scrapeCtx.ProxyInstance.Provider.Name,
		TargetURL:  scrapeCtx.TargetURL,
		StartTime:  startTime,
		RetryCount: scrapeCtx.AttemptCount - 1,
		UserAgent:  scrapeCtx.UserAgent,
		Headers:    scrapeCtx.RequestHeaders,
	}

	// Create HTTP client with proxy
	client := ape.createProxyClient(scrapeCtx.ProxyInstance)
	
	// Create request
	req, err := http.NewRequestWithContext(scrapeCtx.Context, "GET", scrapeCtx.TargetURL, nil)
	if err != nil {
		metrics.EndTime = time.Now()
		metrics.Duration = metrics.EndTime.Sub(startTime)
		metrics.ErrorType = "request_creation"
		return nil, metrics, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if scrapeCtx.UserAgent != "" {
		req.Header.Set("User-Agent", scrapeCtx.UserAgent)
	} else {
		// Use a rotating user agent from the pool
		req.Header.Set("User-Agent", ape.Engine.getNextUserAgent())
	}

	for key, value := range scrapeCtx.RequestHeaders {
		req.Header.Set(key, value)
	}

	// Measure request size
	metrics.RequestSize = int64(len(req.URL.String()))
	for key, values := range req.Header {
		for _, value := range values {
			metrics.RequestSize += int64(len(key) + len(value) + 4) // +4 for ": " and "\r\n"
		}
	}

	// Execute request with timing
	// DNS timing would be tracked here
	resp, err := client.Do(req)
	if err != nil {
		metrics.EndTime = time.Now()
		metrics.Duration = metrics.EndTime.Sub(startTime)
		metrics.ErrorType = ape.categorizeHTTPError(err)
		return nil, metrics, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Record response metrics
	metrics.StatusCode = resp.StatusCode
	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(startTime)
	metrics.ResponseSize = resp.ContentLength
	if metrics.ResponseSize == -1 {
		metrics.ResponseSize = 0 // Unknown size
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		metrics.ErrorType = fmt.Sprintf("http_%d", resp.StatusCode)
		return nil, metrics, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	// Parse response and extract data
	result, dataQuality, err := ape.parseResponseWithQuality(resp, scrapeCtx.TargetURL)
	metrics.DataQuality = dataQuality
	
	if err != nil {
		metrics.ErrorType = "parsing_error"
		return result, metrics, err
	}

	metrics.Success = true
	return result, metrics, nil
}

// createProxyClient creates an HTTP client configured for the specified proxy
func (ape *AdvancedProxyEngine) createProxyClient(proxyInstance *proxy.AdvancedProxyInstance) *http.Client {
	if proxyInstance == nil {
		return ape.Engine.httpClient
	}

	// Create proxy URL
	proxyURL := proxyInstance.URL
	
	// Create transport with proxy
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	// Configure TLS if needed
	if ape.config != nil && ape.Engine.config.Proxy != nil && ape.Engine.config.Proxy.TLS != nil {
		tlsConfig, err := proxy.BuildTLSConfig(&proxy.TLSConfig{
			InsecureSkipVerify: ape.Engine.config.Proxy.TLS.InsecureSkipVerify,
			ServerName:         ape.Engine.config.Proxy.TLS.ServerName,
			RootCAs:            ape.Engine.config.Proxy.TLS.RootCAs,
			ClientCert:         ape.Engine.config.Proxy.TLS.ClientCert,
			ClientKey:          ape.Engine.config.Proxy.TLS.ClientKey,
		})
		if err == nil {
			transport.TLSClientConfig = tlsConfig
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   ape.Engine.config.Timeout,
	}
}

// parseResponseWithQuality parses HTTP response and calculates data quality score
func (ape *AdvancedProxyEngine) parseResponseWithQuality(resp *http.Response, targetURL string) (*Result, float64, error) {
	// Use the base engine's parsing logic
	doc, err := ape.Engine.parseDocument(resp)
	if err != nil {
		return nil, 0, err
	}

	// Extract data using base engine
	result, err := ape.Engine.extractData(doc, targetURL)
	if err != nil {
		return result, 0, err
	}

	// Calculate data quality score
	dataQuality := ape.calculateDataQuality(result, doc)
	
	return result, dataQuality, nil
}

// calculateDataQuality calculates a quality score for extracted data
func (ape *AdvancedProxyEngine) calculateDataQuality(result *Result, doc interface{}) float64 {
	if result == nil {
		return 0
	}

	score := 0.0
	maxScore := 0.0

	// Base score for successful extraction
	if result.Success {
		score += 30
	}
	maxScore += 30

	// Score based on data completeness
	if len(result.Data) > 0 {
		score += 25
		maxScore += 25

		// Additional points for meaningful data
		nonEmptyFields := 0
		for _, value := range result.Data {
			if value != nil && value != "" {
				nonEmptyFields++
			}
		}
		
		completenessRatio := float64(nonEmptyFields) / float64(len(result.Data))
		score += completenessRatio * 25
		maxScore += 25
	} else {
		maxScore += 50
	}

	// Score based on error count (fewer errors = higher quality)
	errorPenalty := float64(len(result.Errors)) * 5
	score = score - errorPenalty
	maxScore += 20 // Max potential score from having no errors

	// Normalize score to 0-100 range
	if maxScore > 0 {
		normalizedScore := (score / maxScore) * 100
		if normalizedScore < 0 {
			return 0
		}
		if normalizedScore > 100 {
			return 100
		}
		return normalizedScore
	}

	return 50 // Default mid-range score if we can't calculate properly
}

// categorizeError categorizes errors for reporting and retry decisions
func (ape *AdvancedProxyEngine) categorizeError(err error, statusCode int) string {
	if err == nil {
		if statusCode >= 400 {
			return fmt.Sprintf("http_%d", statusCode)
		}
		return "unknown"
	}

	errStr := strings.ToLower(err.Error())
	
	switch {
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "no such host"):
		return "dns_error"
	case strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate"):
		return "tls_error"
	case strings.Contains(errStr, "proxy"):
		return "proxy_error"
	case strings.Contains(errStr, "context canceled"):
		return "canceled"
	case strings.Contains(errStr, "context deadline"):
		return "deadline_exceeded"
	case statusCode == 429:
		return "rate_limited"
	case statusCode >= 500:
		return "server_error"
	case statusCode >= 400:
		return "client_error"
	default:
		return "unknown"
	}
}

// categorizeHTTPError categorizes HTTP client errors
func (ape *AdvancedProxyEngine) categorizeHTTPError(err error) string {
	if err == nil {
		return "unknown"
	}

	errStr := strings.ToLower(err.Error())
	
	switch {
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "no such host"):
		return "dns_error"
	case strings.Contains(errStr, "tls"):
		return "tls_error"
	case strings.Contains(errStr, "proxy"):
		return "proxy_error"
	default:
		return "network_error"
	}
}

// shouldRetry determines if a request should be retried based on error type and context
func (ape *AdvancedProxyEngine) shouldRetry(err error, statusCode, attempt, maxAttempts int) bool {
	if attempt >= maxAttempts {
		return false
	}

	// Don't retry on context cancellation
	if err != nil && strings.Contains(err.Error(), "context canceled") {
		return false
	}

	// Retry strategy based on error type
	errorType := ape.categorizeError(err, statusCode)
	
	switch errorType {
	case "timeout", "connection_refused", "proxy_error", "network_error":
		return true // Always retry network issues
	case "dns_error":
		return attempt <= 2 // Retry DNS errors up to 2 times
	case "tls_error":
		return attempt <= 1 // Retry TLS errors once
	case "rate_limited", "server_error":
		return true // Retry server-side issues
	case "client_error":
		return false // Don't retry 4xx errors (except 429)
	default:
		return attempt <= 2 // Conservative retry for unknown errors
	}
}

// calculateRetryDelay calculates exponential backoff delay for retries
func (ape *AdvancedProxyEngine) calculateRetryDelay(attempt int) time.Duration {
	if ape.config.RetryStrategy != nil {
		baseDelay := ape.config.RetryStrategy.InitialDelay
		maxDelay := ape.config.RetryStrategy.MaxDelay
		multiplier := ape.config.RetryStrategy.BackoffMultiplier

		if baseDelay == 0 {
			baseDelay = time.Second
		}
		if maxDelay == 0 {
			maxDelay = 30 * time.Second
		}
		if multiplier == 0 {
			multiplier = 2.0
		}

		delay := time.Duration(float64(baseDelay) * (multiplier * float64(attempt-1)))
		if delay > maxDelay {
			delay = maxDelay
		}
		
		return delay
	}

	// Default exponential backoff: 1s, 2s, 4s, 8s...
	delay := time.Duration(1<<uint(attempt-1)) * time.Second
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	
	return delay
}

// GetProxyStats returns current proxy statistics
func (ape *AdvancedProxyEngine) GetProxyStats() interface{} {
	if !ape.config.Enabled || ape.advancedProxyManager == nil {
		return map[string]interface{}{
			"enabled": false,
			"message": "Advanced proxy features not enabled",
		}
	}

	stats := ape.advancedProxyManager.GetStats()
	
	response := map[string]interface{}{
		"enabled":         true,
		"strategy":        ape.config.Strategy,
		"total_proxies":   stats.TotalProxies,
		"healthy_proxies": stats.HealthyProxies,
		"failed_proxies":  stats.FailedProxies,
		"success_rate":    stats.SuccessRate,
		"total_requests":  stats.TotalRequests,
		"average_response": stats.AverageResponse,
		"last_health_check": stats.LastHealthCheck,
	}

	// Add monitoring stats if available
	if ape.proxyMonitor != nil {
		if currentMetrics := ape.proxyMonitor.GetCurrentMetrics(); currentMetrics != nil {
			response["monitoring"] = map[string]interface{}{
				"current_metrics":    currentMetrics,
				"active_alerts":      ape.proxyMonitor.GetActiveAlerts(),
				"monitoring_enabled": true,
			}
		}
	}

	return response
}

// GetPerformanceReport returns a detailed performance report
func (ape *AdvancedProxyEngine) GetPerformanceReport(period time.Duration) interface{} {
	if !ape.config.Enabled || ape.proxyMonitor == nil {
		return map[string]interface{}{
			"enabled": false,
			"message": "Advanced proxy monitoring not enabled",
		}
	}

	return ape.proxyMonitor.GetPerformanceReport(period)
}

// GetCostReport returns a cost analysis report
func (ape *AdvancedProxyEngine) GetCostReport(period time.Duration) interface{} {
	if !ape.config.Enabled || ape.proxyMonitor == nil {
		return map[string]interface{}{
			"enabled": false,
			"message": "Advanced proxy monitoring not enabled",
		}
	}

	return ape.proxyMonitor.GetCostReport(period)
}

// Helper method to get next user agent from the base engine
func (e *Engine) getNextUserAgent() string {
	if len(e.userAgentPool) == 0 {
		return "DataScrapexter/1.0"
	}
	
	userAgent := e.userAgentPool[e.currentUAIndex]
	e.currentUAIndex = (e.currentUAIndex + 1) % len(e.userAgentPool)
	return userAgent
}

// Helper methods that need to access private engine methods
func (e *Engine) parseDocument(resp *http.Response) (interface{}, error) {
	// TODO: This is a placeholder implementation that needs to be replaced with proper HTML parsing.
	// The real implementation should:
	// 1. Use goquery to parse the HTTP response body into a queryable document
	// 2. Handle encoding detection and conversion
	// 3. Implement proper error handling for malformed HTML
	// 4. Support both HTML and XML document types based on content-type
	// 5. Cache parsed documents for performance optimization
	// 
	// Example implementation:
	// doc, err := goquery.NewDocumentFromReader(resp.Body)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to parse HTML document: %w", err)
	// }
	// return doc, nil
	return nil, fmt.Errorf("document parsing not implemented in integration layer - TODO: implement proper HTML parsing with goquery")
}

func (e *Engine) extractData(doc interface{}, url string) (*Result, error) {
	// TODO: This is a placeholder implementation that needs to be replaced with proper data extraction.
	// The real implementation should:
	// 1. Use the configured field selectors to extract data from the parsed document
	// 2. Apply transformation rules to the extracted data
	// 3. Validate extracted data against field requirements
	// 4. Handle extraction errors gracefully with fallback strategies
	// 5. Support complex extraction patterns (nested selectors, conditional extraction)
	// 6. Track extraction success metrics for quality assessment
	//
	// Example implementation:
	// result := &Result{
	//     Data:      make(map[string]interface{}),
	//     Success:   true,
	//     Timestamp: time.Now(),
	//     Errors:    make([]string, 0),
	// }
	// 
	// for _, field := range e.config.Fields {
	//     value, err := extractFieldValue(doc, field)
	//     if err != nil {
	//         result.Errors = append(result.Errors, err.Error())
	//         continue
	//     }
	//     result.Data[field.Name] = value
	// }
	// 
	// return result, nil
	return &Result{
		Data:      make(map[string]interface{}),
		Success:   true,
		Timestamp: time.Now(),
		Errors:    []string{"TODO: implement proper data extraction based on field configurations"},
	}, nil
}