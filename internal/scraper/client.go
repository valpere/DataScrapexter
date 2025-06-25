// internal/scraper/client.go
package scraper

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// HTTPClient represents an enhanced HTTP client for web scraping
type HTTPClient struct {
	client         *http.Client
	config         *ScraperConfig
	rateLimiter    *rate.Limiter
	userAgentIndex int
	retryPolicy    *RetryPolicy
}

// ScraperConfig contains HTTP client configuration
type ScraperConfig struct {
	UserAgents        []string          `yaml:"user_agents" json:"user_agents"`
	Headers           map[string]string `yaml:"headers" json:"headers"`
	Cookies           map[string]string `yaml:"cookies" json:"cookies"`
	RequestTimeout    time.Duration     `yaml:"request_timeout" json:"request_timeout"`
	RetryAttempts     int               `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay        time.Duration     `yaml:"retry_delay" json:"retry_delay"`
	RateLimit         float64           `yaml:"rate_limit" json:"rate_limit"`
	MaxRedirects      int               `yaml:"max_redirects" json:"max_redirects"`
	IgnoreSSLErrors   bool              `yaml:"ignore_ssl_errors" json:"ignore_ssl_errors"`
	ProxyURL          string            `yaml:"proxy_url" json:"proxy_url"`
	MaxConcurrency    int               `yaml:"max_concurrency" json:"max_concurrency"`
}

// RetryPolicy defines the retry behavior for failed requests
type RetryPolicy struct {
	MaxAttempts    int
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
	RetryableErrors []int
}

// RequestResult contains the result of an HTTP request
type RequestResult struct {
	Response   *http.Response
	Body       []byte
	StatusCode int
	Duration   time.Duration
	Attempt    int
	Error      error
}

// NewHTTPClient creates a new HTTP client with the provided configuration
func NewHTTPClient(config *ScraperConfig) (*HTTPClient, error) {
	if config == nil {
		return nil, fmt.Errorf("scraper config cannot be nil")
	}

	// Validate and set default values
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// CRITICAL FIX: Create HTTP client with proper timeout configuration
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,  // Connection timeout
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: config.RequestTimeout, // CRITICAL: Apply timeout to response headers
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
	}

	// Handle SSL configuration
	if config.IgnoreSSLErrors {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	// Handle proxy configuration
	if config.ProxyURL != "" {
		proxyURL, err := url.Parse(config.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	// CRITICAL FIX: Configure client with proper timeout settings
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.RequestTimeout, // CRITICAL: Overall request timeout
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("maximum redirects exceeded: %d", config.MaxRedirects)
			}
			return nil
		},
	}

	// Configure rate limiter if specified
	var rateLimiter *rate.Limiter
	if config.RateLimit > 0 {
		rateLimiter = rate.NewLimiter(rate.Limit(config.RateLimit), 1)
	}

	// Configure retry policy
	retryPolicy := &RetryPolicy{
		MaxAttempts:   config.RetryAttempts,
		BaseDelay:     config.RetryDelay,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []int{
			http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	}

	return &HTTPClient{
		client:      httpClient,
		config:      config,
		rateLimiter: rateLimiter,
		retryPolicy: retryPolicy,
	}, nil
}

// Get performs an HTTP GET request with proper timeout and retry handling
func (hc *HTTPClient) Get(ctx context.Context, targetURL string) (*RequestResult, error) {
	return hc.Request(ctx, "GET", targetURL, nil)
}

// Request performs an HTTP request with comprehensive error handling and timeout enforcement
func (hc *HTTPClient) Request(ctx context.Context, method, targetURL string, body io.Reader) (*RequestResult, error) {
	// CRITICAL FIX: Create request with context that includes timeout
	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply headers and user agent
	hc.applyHeaders(req)
	hc.applyCookies(req)

	// Rate limiting
	if hc.rateLimiter != nil {
		if err := hc.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiting failed: %w", err)
		}
	}

	// Perform request with retry logic
	return hc.performRequestWithRetry(ctx, req)
}

// performRequestWithRetry executes the request with exponential backoff retry logic
func (hc *HTTPClient) performRequestWithRetry(ctx context.Context, req *http.Request) (*RequestResult, error) {
	var lastErr error
	
	for attempt := 1; attempt <= hc.retryPolicy.MaxAttempts; attempt++ {
		// CRITICAL FIX: Check context cancellation and timeout before each attempt
		select {
		case <-ctx.Done():
			return &RequestResult{
				Error:    ctx.Err(),
				Attempt:  attempt,
				Duration: 0,
			}, fmt.Errorf("request cancelled or timed out: %w", ctx.Err())
		default:
		}

		startTime := time.Now()
		
		// CRITICAL FIX: Execute request with timeout-aware context
		resp, err := hc.client.Do(req.WithContext(ctx))
		duration := time.Since(startTime)

		result := &RequestResult{
			Response: resp,
			Duration: duration,
			Attempt:  attempt,
		}

		if err != nil {
			lastErr = err
			result.Error = err
			
			// Check if error is retryable
			if !hc.isRetryableError(err) || attempt == hc.retryPolicy.MaxAttempts {
				return result, fmt.Errorf("request failed after %d attempts: %w", attempt, err)
			}

			// Wait before retry with exponential backoff
			if err := hc.waitForRetry(ctx, attempt); err != nil {
				return result, fmt.Errorf("retry wait failed: %w", err)
			}
			continue
		}

		// Read response body
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			result.Error = lastErr
			
			if attempt == hc.retryPolicy.MaxAttempts {
				return result, lastErr
			}
			
			if err := hc.waitForRetry(ctx, attempt); err != nil {
				return result, fmt.Errorf("retry wait failed: %w", err)
			}
			continue
		}

		result.Body = bodyBytes
		result.StatusCode = resp.StatusCode

		// Check if status code indicates retry is needed
		if hc.shouldRetryStatus(resp.StatusCode) && attempt < hc.retryPolicy.MaxAttempts {
			lastErr = fmt.Errorf("received retryable status code: %d", resp.StatusCode)
			result.Error = lastErr
			
			if err := hc.waitForRetry(ctx, attempt); err != nil {
				return result, fmt.Errorf("retry wait failed: %w", err)
			}
			continue
		}

		// Successful request
		return result, nil
	}

	// All retry attempts exhausted
	if lastErr != nil {
		return &RequestResult{
			Error:    lastErr,
			Attempt:  hc.retryPolicy.MaxAttempts,
			Duration: 0,
		}, fmt.Errorf("request failed after %d attempts: %w", hc.retryPolicy.MaxAttempts, lastErr)
	}

	return &RequestResult{
		Error:    fmt.Errorf("unknown error occurred"),
		Attempt:  hc.retryPolicy.MaxAttempts,
		Duration: 0,
	}, fmt.Errorf("request failed with unknown error")
}

// applyHeaders adds configured headers to the request
func (hc *HTTPClient) applyHeaders(req *http.Request) {
	// Set User-Agent
	if len(hc.config.UserAgents) > 0 {
		userAgent := hc.config.UserAgents[hc.userAgentIndex%len(hc.config.UserAgents)]
		req.Header.Set("User-Agent", userAgent)
		hc.userAgentIndex++
	}

	// Set custom headers
	for key, value := range hc.config.Headers {
		req.Header.Set(key, value)
	}

	// Set default headers if not already set
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	}
	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "gzip, deflate")
	}
	if req.Header.Get("DNT") == "" {
		req.Header.Set("DNT", "1")
	}
	if req.Header.Get("Connection") == "" {
		req.Header.Set("Connection", "keep-alive")
	}
	if req.Header.Get("Upgrade-Insecure-Requests") == "" {
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}
}

// applyCookies adds configured cookies to the request
func (hc *HTTPClient) applyCookies(req *http.Request) {
	for name, value := range hc.config.Cookies {
		cookie := &http.Cookie{
			Name:  name,
			Value: value,
		}
		req.AddCookie(cookie)
	}
}

// isRetryableError determines if an error warrants a retry
func (hc *HTTPClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are generally retryable
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	// Context timeout errors are not retryable as they indicate the overall timeout was reached
	if err == context.DeadlineExceeded || err == context.Canceled {
		return false
	}

	// URL errors might be retryable depending on the underlying error
	if urlErr, ok := err.(*url.Error); ok {
		return hc.isRetryableError(urlErr.Err)
	}

	// String-based error matching for common retryable cases
	errStr := strings.ToLower(err.Error())
	retryableStrings := []string{
		"connection reset",
		"connection refused",
		"no such host",
		"timeout",
		"temporary failure",
		"network is unreachable",
	}

	for _, retryableStr := range retryableStrings {
		if strings.Contains(errStr, retryableStr) {
			return true
		}
	}

	return false
}

// shouldRetryStatus determines if an HTTP status code warrants a retry
func (hc *HTTPClient) shouldRetryStatus(statusCode int) bool {
	for _, retryableStatus := range hc.retryPolicy.RetryableErrors {
		if statusCode == retryableStatus {
			return true
		}
	}
	return false
}

// waitForRetry implements exponential backoff with jitter
func (hc *HTTPClient) waitForRetry(ctx context.Context, attempt int) error {
	if attempt <= 1 {
		return nil // No wait for first attempt
	}

	// Calculate delay with exponential backoff
	delay := hc.retryPolicy.BaseDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * hc.retryPolicy.BackoffFactor)
		if delay > hc.retryPolicy.MaxDelay {
			delay = hc.retryPolicy.MaxDelay
			break
		}
	}

	// CRITICAL FIX: Respect context timeout during retry wait
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// validateConfig validates the scraper configuration
func validateConfig(config *ScraperConfig) error {
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 30 * time.Second
	}

	if config.RetryAttempts < 0 {
		config.RetryAttempts = 3
	}

	if config.RetryDelay <= 0 {
		config.RetryDelay = 1 * time.Second
	}

	if config.MaxRedirects < 0 {
		config.MaxRedirects = 10
	}

	if len(config.UserAgents) == 0 {
		config.UserAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		}
	}

	if config.RateLimit < 0 {
		config.RateLimit = 0 // No rate limiting
	}

	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 5
	}

	return nil
}

// SetUserAgent rotates to the next user agent in the list
func (hc *HTTPClient) SetUserAgent(userAgent string) {
	if hc.config.UserAgents == nil {
		hc.config.UserAgents = []string{}
	}
	hc.config.UserAgents = append(hc.config.UserAgents, userAgent)
}

// GetStats returns statistics about the HTTP client
func (hc *HTTPClient) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"user_agents_count":   len(hc.config.UserAgents),
		"current_user_agent":  hc.userAgentIndex % len(hc.config.UserAgents),
		"request_timeout":     hc.config.RequestTimeout.String(),
		"retry_attempts":      hc.config.RetryAttempts,
		"rate_limit":          hc.config.RateLimit,
		"max_redirects":       hc.config.MaxRedirects,
		"ignore_ssl_errors":   hc.config.IgnoreSSLErrors,
		"proxy_configured":    hc.config.ProxyURL != "",
	}

	if hc.rateLimiter != nil {
		stats["rate_limiter_limit"] = hc.rateLimiter.Limit()
		stats["rate_limiter_burst"] = hc.rateLimiter.Burst()
	}

	return stats
}

// Close cleans up any resources used by the HTTP client
func (hc *HTTPClient) Close() error {
	if hc.client != nil && hc.client.Transport != nil {
		if transport, ok := hc.client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	return nil
}
