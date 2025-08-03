// internal/scraper/client.go
package scraper

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// HTTPClientConfig configures the HTTP client behavior
type HTTPClientConfig struct {
	Timeout             time.Duration     `yaml:"timeout" json:"timeout"`
	RetryAttempts       int               `yaml:"retry_attempts" json:"retry_attempts"`
	RetryBackoffBase    time.Duration     `yaml:"retry_backoff_base" json:"retry_backoff_base"`
	RetryBackoffMax     time.Duration     `yaml:"retry_backoff_max" json:"retry_backoff_max"`
	UserAgents          []string          `yaml:"user_agents" json:"user_agents"`
	Headers             map[string]string `yaml:"headers" json:"headers"`
	Cookies             map[string]string `yaml:"cookies" json:"cookies"`
	FollowRedirects     bool              `yaml:"follow_redirects" json:"follow_redirects"`
	MaxRedirects        int               `yaml:"max_redirects" json:"max_redirects"`
	RateLimit           time.Duration     `yaml:"rate_limit" json:"rate_limit"`
	MaxIdleConns        int               `yaml:"max_idle_conns" json:"max_idle_conns"`
	MaxIdleConnsPerHost int               `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"`
	IdleConnTimeout     time.Duration     `yaml:"idle_conn_timeout" json:"idle_conn_timeout"`
	DisableCompression  bool              `yaml:"disable_compression" json:"disable_compression"`
	DisableKeepAlives   bool              `yaml:"disable_keep_alives" json:"disable_keep_alives"`
}

// HTTPClient wraps http.Client with additional functionality
type HTTPClient struct {
	client       *http.Client
	config       *HTTPClientConfig
	userAgentIdx int
	userAgentMux sync.Mutex
	rateLimiter  *rate.Limiter
	stats        *HTTPStats
	statsMux     sync.RWMutex
}

// HTTPStats tracks HTTP client statistics
type HTTPStats struct {
	TotalRequests   int64         `json:"total_requests"`
	SuccessfulReqs  int64         `json:"successful_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	RetryCount      int64         `json:"retry_count"`
	TotalBytes      int64         `json:"total_bytes"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastRequestTime time.Time     `json:"last_request_time"`
	ErrorsByCode    map[int]int64 `json:"errors_by_code"`
}

// HTTPResponse represents an HTTP response with additional metadata
type HTTPResponse struct {
	*http.Response
	Duration  time.Duration `json:"duration"`
	Attempts  int           `json:"attempts"`
	BodyBytes []byte        `json:"-"`
	BodySize  int64         `json:"body_size"`
	FromCache bool          `json:"from_cache"`
}

// NewHTTPClient creates a new HTTP client with the given configuration
func NewHTTPClient(config *HTTPClientConfig) *HTTPClient {
	if config == nil {
		config = &HTTPClientConfig{
			Timeout:             30 * time.Second,
			RetryAttempts:       3,
			RetryBackoffBase:    1 * time.Second,
			RetryBackoffMax:     30 * time.Second,
			UserAgents:          []string{"DataScrapexter/1.0"},
			FollowRedirects:     true,
			MaxRedirects:        10,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		}
	}

	// Create HTTP transport
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		DisableCompression:  config.DisableCompression,
		DisableKeepAlives:   config.DisableKeepAlives,
	}

	// Create HTTP client
	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	// Configure redirect policy
	if !config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if config.MaxRedirects > 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", config.MaxRedirects)
			}
			return nil
		}
	}

	// Create rate limiter if configured
	var rateLimiter *rate.Limiter
	if config.RateLimit > 0 {
		rateLimiter = rate.NewLimiter(rate.Every(config.RateLimit), 1)
	}

	return &HTTPClient{
		client:      client,
		config:      config,
		rateLimiter: rateLimiter,
		stats: &HTTPStats{
			ErrorsByCode: make(map[int]int64),
		},
	}
}

// Get performs an HTTP GET request with retries and rate limiting
func (c *HTTPClient) Get(ctx context.Context, url string) (*HTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.Do(req)
}

// Do performs an HTTP request with retries and rate limiting
func (c *HTTPClient) Do(req *http.Request) (*HTTPResponse, error) {
	// Apply rate limiting
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(req.Context()); err != nil {
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	var lastErr error
	var response *HTTPResponse

	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		// Clone request for retry
		reqClone := req.Clone(req.Context())

		// Set headers
		c.setRequestHeaders(reqClone)

		// Perform request
		startTime := time.Now()
		resp, err := c.client.Do(reqClone)
		duration := time.Since(startTime)

		// Update stats
		c.updateStats(resp, err, duration, attempt > 0)

		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed: %w", attempt+1, err)

			if attempt < c.config.RetryAttempts {
				backoff := c.calculateBackoff(attempt)
				select {
				case <-time.After(backoff):
					continue
				case <-req.Context().Done():
					return nil, req.Context().Err()
				}
			}
			continue
		}

		// Read response body
		bodyBytes, bodySize, readErr := c.readResponseBody(resp)
		if readErr != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("failed to read response body: %w", readErr)

			if attempt < c.config.RetryAttempts {
				backoff := c.calculateBackoff(attempt)
				select {
				case <-time.After(backoff):
					continue
				case <-req.Context().Done():
					return nil, req.Context().Err()
				}
			}
			continue
		}

		response = &HTTPResponse{
			Response:  resp,
			Duration:  duration,
			Attempts:  attempt + 1,
			BodyBytes: bodyBytes,
			BodySize:  bodySize,
		}

		// Check if we should retry based on status code
		if c.shouldRetry(resp.StatusCode) && attempt < c.config.RetryAttempts {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)

			backoff := c.calculateBackoff(attempt)
			select {
			case <-time.After(backoff):
				continue
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}

		// Don't close response body here - caller will close it
		return response, nil
	}

	return response, lastErr
}

// setRequestHeaders sets standard headers and user agent
func (c *HTTPClient) setRequestHeaders(req *http.Request) {
	// Set user agent
	if len(c.config.UserAgents) > 0 {
		c.userAgentMux.Lock()
		userAgent := c.config.UserAgents[c.userAgentIdx%len(c.config.UserAgents)]
		c.userAgentIdx++
		c.userAgentMux.Unlock()

		req.Header.Set("User-Agent", userAgent)
	}

	// Set custom headers
	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	// Set cookies
	for name, value := range c.config.Cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	// Set default headers if not already set
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	}
	if req.Header.Get("Accept-Encoding") == "" && !c.config.DisableCompression {
		req.Header.Set("Accept-Encoding", "gzip, deflate")
	}
	if req.Header.Get("DNT") == "" {
		req.Header.Set("DNT", "1")
	}
	if req.Header.Get("Connection") == "" && !c.config.DisableKeepAlives {
		req.Header.Set("Connection", "keep-alive")
	}
	if req.Header.Get("Upgrade-Insecure-Requests") == "" {
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}
}

// readResponseBody reads and returns the response body
func (c *HTTPClient) readResponseBody(resp *http.Response) ([]byte, int64, error) {
	if resp.Body == nil {
		return nil, 0, nil
	}

	bodyBytes := make([]byte, 0)
	buffer := make([]byte, 4096)
	totalSize := int64(0)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			bodyBytes = append(bodyBytes, buffer[:n]...)
			totalSize += int64(n)
		}
		if err != nil {
			break
		}
	}

	return bodyBytes, totalSize, nil
}

// shouldRetry determines if a request should be retried based on status code
func (c *HTTPClient) shouldRetry(statusCode int) bool {
	// Retry on server errors and certain client errors
	switch statusCode {
	case 408, 429: // Request Timeout, Too Many Requests
		return true
	case 500, 502, 503, 504: // Server errors
		return true
	case 520, 521, 522, 523, 524: // Cloudflare errors
		return true
	default:
		return false
	}
}

// calculateBackoff calculates exponential backoff with jitter
func (c *HTTPClient) calculateBackoff(attempt int) time.Duration {
	backoff := time.Duration(math.Pow(2, float64(attempt))) * c.config.RetryBackoffBase

	if backoff > c.config.RetryBackoffMax {
		backoff = c.config.RetryBackoffMax
	}

	// Add jitter (up to 25% of backoff time)
	jitter := time.Duration(float64(backoff) * 0.25 * (0.5 + (float64(time.Now().UnixNano()%1000) / 1000.0)))

	return backoff + jitter
}

// updateStats updates HTTP client statistics
func (c *HTTPClient) updateStats(resp *http.Response, err error, duration time.Duration, isRetry bool) {
	c.statsMux.Lock()
	defer c.statsMux.Unlock()

	c.stats.TotalRequests++
	c.stats.LastRequestTime = time.Now()

	if isRetry {
		c.stats.RetryCount++
	}

	if err != nil {
		c.stats.FailedRequests++
		return
	}

	if resp != nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			c.stats.SuccessfulReqs++
		} else {
			c.stats.FailedRequests++
			c.stats.ErrorsByCode[resp.StatusCode]++
		}

		// Update average latency
		if c.stats.SuccessfulReqs == 1 {
			c.stats.AverageLatency = duration
		} else {
			c.stats.AverageLatency = time.Duration(
				(int64(c.stats.AverageLatency)*c.stats.SuccessfulReqs + int64(duration)) / (c.stats.SuccessfulReqs + 1),
			)
		}
	}
}

// GetStats returns current HTTP client statistics
func (c *HTTPClient) GetStats() *HTTPStats {
	c.statsMux.RLock()
	defer c.statsMux.RUnlock()

	// Create a copy to avoid race conditions
	statsCopy := &HTTPStats{
		TotalRequests:   c.stats.TotalRequests,
		SuccessfulReqs:  c.stats.SuccessfulReqs,
		FailedRequests:  c.stats.FailedRequests,
		RetryCount:      c.stats.RetryCount,
		TotalBytes:      c.stats.TotalBytes,
		AverageLatency:  c.stats.AverageLatency,
		LastRequestTime: c.stats.LastRequestTime,
		ErrorsByCode:    make(map[int]int64),
	}

	for code, count := range c.stats.ErrorsByCode {
		statsCopy.ErrorsByCode[code] = count
	}

	return statsCopy
}

// Close closes the HTTP client and cleans up resources
func (c *HTTPClient) Close() error {
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

// SetUserAgent sets a single user agent (overwrites user agent list)
func (c *HTTPClient) SetUserAgent(userAgent string) {
	c.userAgentMux.Lock()
	defer c.userAgentMux.Unlock()
	c.config.UserAgents = []string{userAgent}
	c.userAgentIdx = 0
}

// AddUserAgent adds a user agent to the rotation list
func (c *HTTPClient) AddUserAgent(userAgent string) {
	c.userAgentMux.Lock()
	defer c.userAgentMux.Unlock()
	c.config.UserAgents = append(c.config.UserAgents, userAgent)
}

// SetHeader sets a custom header
func (c *HTTPClient) SetHeader(key, value string) {
	if c.config.Headers == nil {
		c.config.Headers = make(map[string]string)
	}
	c.config.Headers[key] = value
}

// SetCookie sets a cookie to be sent with requests
func (c *HTTPClient) SetCookie(name, value string) {
	if c.config.Cookies == nil {
		c.config.Cookies = make(map[string]string)
	}
	c.config.Cookies[name] = value
}

// GetCurrentUserAgent returns the current user agent that will be used
func (c *HTTPClient) GetCurrentUserAgent() string {
	c.userAgentMux.Lock()
	defer c.userAgentMux.Unlock()

	if len(c.config.UserAgents) == 0 {
		return "DataScrapexter/1.0"
	}

	return c.config.UserAgents[c.userAgentIdx%len(c.config.UserAgents)]
}
