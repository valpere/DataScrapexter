// internal/scraper/client.go
package scraper

import (
    "fmt"
    "math/rand"
    "net/http"
    "net/url"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

// HTTPClient provides a robust HTTP client for web scraping with anti-detection features
type HTTPClient struct {
    httpClient    *http.Client
    userAgents    []string
    currentUA     int
    uaMutex       sync.RWMutex
    rateLimiter   *rate.Limiter
    retryAttempts int
    retryDelay    time.Duration
    headers       map[string]string
    cookies       map[string]string
}

// ClientConfig defines configuration options for the HTTP client
type ClientConfig struct {
    Timeout       time.Duration
    RetryAttempts int
    RetryDelay    time.Duration
    UserAgents    []string
    Headers       map[string]string
    Cookies       map[string]string
    RateLimit     float64 // requests per second
    RateBurst     int
}

// NewHTTPClient creates a new HTTP client with the specified configuration
func NewHTTPClient(config ClientConfig) *HTTPClient {
    // Set default values if not provided
    if config.Timeout == 0 {
        config.Timeout = 30 * time.Second
    }
    if config.RetryAttempts == 0 {
        config.RetryAttempts = 3
    }
    if config.RetryDelay == 0 {
        config.RetryDelay = time.Second
    }
    if config.RateLimit == 0 {
        config.RateLimit = 1.0 // 1 request per second default
    }
    if config.RateBurst == 0 {
        config.RateBurst = 5
    }
    
    // Default user agents if none provided
    if len(config.UserAgents) == 0 {
        config.UserAgents = getDefaultUserAgents()
    }

    // Create HTTP client with timeout
    httpClient := &http.Client{
        Timeout: config.Timeout,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        },
    }

    // Create rate limiter
    rateLimiter := rate.NewLimiter(rate.Limit(config.RateLimit), config.RateBurst)

    return &HTTPClient{
        httpClient:    httpClient,
        userAgents:    config.UserAgents,
        currentUA:     0,
        rateLimiter:   rateLimiter,
        retryAttempts: config.RetryAttempts,
        retryDelay:    config.RetryDelay,
        headers:       config.Headers,
        cookies:       config.Cookies,
    }
}

// Get performs an HTTP GET request with retry logic and anti-detection measures
func (c *HTTPClient) Get(targetURL string) (*http.Response, error) {
    // Validate URL
    if _, err := url.Parse(targetURL); err != nil {
        return nil, fmt.Errorf("invalid URL: %w", err)
    }

    var lastErr error
    
    // Retry loop
    for attempt := 0; attempt <= c.retryAttempts; attempt++ {
        // Wait for rate limiter
        if err := c.rateLimiter.Wait(nil); err != nil {
            return nil, fmt.Errorf("rate limiter error: %w", err)
        }

        // Create request
        req, err := http.NewRequest("GET", targetURL, nil)
        if err != nil {
            return nil, fmt.Errorf("failed to create request: %w", err)
        }

        // Set headers
        c.setRequestHeaders(req)

        // Perform request
        resp, err := c.httpClient.Do(req)
        if err != nil {
            lastErr = fmt.Errorf("request failed (attempt %d/%d): %w", 
                attempt+1, c.retryAttempts+1, err)
            
            // Don't retry on the last attempt
            if attempt < c.retryAttempts {
                c.waitForRetry(attempt)
                continue
            }
            break
        }

        // Check status code
        if resp.StatusCode >= 200 && resp.StatusCode < 300 {
            return resp, nil
        }

        // Handle error status codes
        resp.Body.Close()
        lastErr = fmt.Errorf("HTTP %d: %s (attempt %d/%d)", 
            resp.StatusCode, resp.Status, attempt+1, c.retryAttempts+1)

        // Determine if we should retry based on status code
        if !c.shouldRetryStatusCode(resp.StatusCode) {
            break
        }

        // Don't retry on the last attempt
        if attempt < c.retryAttempts {
            c.waitForRetry(attempt)
        }
    }

    return nil, lastErr
}

// Post performs an HTTP POST request with retry logic
func (c *HTTPClient) Post(targetURL string, contentType string, body []byte) (*http.Response, error) {
    // Implementation similar to Get but for POST requests
    // This would include the same retry logic and error handling
    return nil, fmt.Errorf("POST method not yet implemented")
}

// setRequestHeaders configures request headers including user agent rotation
func (c *HTTPClient) setRequestHeaders(req *http.Request) {
    // Set User-Agent with rotation
    userAgent := c.getNextUserAgent()
    req.Header.Set("User-Agent", userAgent)

    // Set default headers that make requests look more browser-like
    req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
    req.Header.Set("Accept-Language", "en-US,en;q=0.5")
    req.Header.Set("Accept-Encoding", "gzip, deflate")
    req.Header.Set("DNT", "1")
    req.Header.Set("Connection", "keep-alive")
    req.Header.Set("Upgrade-Insecure-Requests", "1")

    // Set custom headers from configuration
    for key, value := range c.headers {
        req.Header.Set(key, value)
    }

    // Set cookies from configuration
    for name, value := range c.cookies {
        cookie := &http.Cookie{
            Name:  name,
            Value: value,
        }
        req.AddCookie(cookie)
    }
}

// getNextUserAgent returns the next user agent in rotation
func (c *HTTPClient) getNextUserAgent() string {
    c.uaMutex.Lock()
    defer c.uaMutex.Unlock()
    
    if len(c.userAgents) == 0 {
        return "DataScrapexter/1.0"
    }
    
    userAgent := c.userAgents[c.currentUA]
    c.currentUA = (c.currentUA + 1) % len(c.userAgents)
    
    return userAgent
}

// waitForRetry implements exponential backoff with jitter
func (c *HTTPClient) waitForRetry(attempt int) {
    // Exponential backoff: base_delay * 2^attempt
    backoffDelay := c.retryDelay * time.Duration(1<<uint(attempt))
    
    // Add jitter to prevent thundering herd
    jitter := time.Duration(rand.Int63n(int64(backoffDelay / 2)))
    totalDelay := backoffDelay + jitter
    
    // Cap maximum delay at 30 seconds
    if totalDelay > 30*time.Second {
        totalDelay = 30*time.Second + jitter/4
    }
    
    time.Sleep(totalDelay)
}

// shouldRetryStatusCode determines if a status code warrants a retry
func (c *HTTPClient) shouldRetryStatusCode(statusCode int) bool {
    // Retry on server errors and some client errors
    retryableStatusCodes := map[int]bool{
        429: true, // Too Many Requests
        500: true, // Internal Server Error
        502: true, // Bad Gateway
        503: true, // Service Unavailable
        504: true, // Gateway Timeout
        520: true, // CloudFlare errors
        521: true,
        522: true,
        523: true,
        524: true,
    }
    
    return retryableStatusCodes[statusCode]
}

// SetRateLimit updates the rate limiting configuration
func (c *HTTPClient) SetRateLimit(requestsPerSecond float64, burst int) {
    c.rateLimiter = rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
}

// AddUserAgent adds a new user agent to the rotation pool
func (c *HTTPClient) AddUserAgent(userAgent string) {
    c.uaMutex.Lock()
    defer c.uaMutex.Unlock()
    c.userAgents = append(c.userAgents, userAgent)
}

// SetHeaders updates the custom headers configuration
func (c *HTTPClient) SetHeaders(headers map[string]string) {
    c.headers = headers
}

// SetCookies updates the cookies configuration
func (c *HTTPClient) SetCookies(cookies map[string]string) {
    c.cookies = cookies
}

// GetStats returns basic statistics about the client usage
func (c *HTTPClient) GetStats() ClientStats {
    return ClientStats{
        UserAgentsCount: len(c.userAgents),
        CurrentUA:       c.currentUA,
        RetryAttempts:   c.retryAttempts,
        Timeout:         c.httpClient.Timeout,
    }
}

// ClientStats provides information about client configuration and usage
type ClientStats struct {
    UserAgentsCount int
    CurrentUA       int
    RetryAttempts   int
    Timeout         time.Duration
}

// getDefaultUserAgents returns a set of realistic user agent strings
func getDefaultUserAgents() []string {
    return []string{
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/119.0",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/119.0",
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
        "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/119.0",
    }
}

// HTTPError represents an HTTP-related error with additional context
type HTTPError struct {
    StatusCode int
    Status     string
    URL        string
    Attempt    int
}

func (e *HTTPError) Error() string {
    return fmt.Sprintf("HTTP %d: %s (URL: %s, Attempt: %d)", 
        e.StatusCode, e.Status, e.URL, e.Attempt)
}

// IsRetryableError checks if an error indicates the request should be retried
func IsRetryableError(err error) bool {
    if httpErr, ok := err.(*HTTPError); ok {
        return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
    }
    return false
}
