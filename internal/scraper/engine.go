// Package scraper provides the core web scraping functionality for DataScrapexter.
// It includes an HTTP client with advanced features such as retry logic, rate limiting,
// user agent rotation, and session management.
package scraper

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/publicsuffix"

	"github.com/valpere/DataScrapexter/internal/pipeline"
)

// Engine represents the core scraping engine
type Engine struct {
	httpClient     *http.Client
	userAgentPool  []string
	currentUAIndex int
	uaMutex        sync.Mutex
	config         *Config
	rateLimiter    *RateLimiter
}

// Config holds the engine configuration
type Config struct {
	MaxRetries      int
	RetryDelay      time.Duration
	Timeout         time.Duration
	FollowRedirects bool
	MaxRedirects    int
	RateLimit       time.Duration
	BurstSize       int
	ProxyURL        string
	Headers         map[string]string
}

// Result represents a scraping result
type Result struct {
	URL        string
	StatusCode int
	Data       map[string]interface{}
	Error      error
	Timestamp  time.Time
}

// NewEngine creates a new scraping engine instance
func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create cookie jar for session management
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Configure HTTP transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}

	// Set proxy if configured
	if config.ProxyURL != "" {
		proxyURL, err := url.Parse(config.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	// Create HTTP client
	client := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !config.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	engine := &Engine{
		httpClient:    client,
		userAgentPool: defaultUserAgents(),
		config:        config,
		rateLimiter:   NewRateLimiter(config.RateLimit, config.BurstSize),
	}

	return engine, nil
}

// DefaultConfig returns the default engine configuration
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:      3,
		RetryDelay:      time.Second * 2,
		Timeout:         time.Second * 30,
		FollowRedirects: true,
		MaxRedirects:    10,
		RateLimit:       time.Second,
		BurstSize:       5,
		Headers:         make(map[string]string),
	}
}

// Scrape performs a scraping operation on the given URL
func (e *Engine) Scrape(ctx context.Context, targetURL string, extractors []FieldExtractor) (*Result, error) {
	// Apply rate limiting
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	result := &Result{
		URL:       targetURL,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	// Perform HTTP request with retries
	resp, err := e.doRequestWithRetry(ctx, targetURL)
	if err != nil {
		result.Error = err
		return result, err
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Check for successful response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("HTTP error: %d", resp.StatusCode)
		result.Error = err
		return result, err
	}

	// Parse HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse HTML: %w", err)
		return result, result.Error
	}

	// Extract data using field extractors
	for _, extractor := range extractors {
		value, err := e.extractField(doc, extractor)
		if err != nil && extractor.Required {
			result.Error = fmt.Errorf("failed to extract required field %s: %w", extractor.Name, err)
			return result, result.Error
		}
		if value != nil {
			result.Data[extractor.Name] = value
		}
	}

	return result, nil
}

// doRequestWithRetry performs an HTTP request with retry logic
func (e *Engine) doRequestWithRetry(ctx context.Context, targetURL string) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry with exponential backoff
			delay := e.config.RetryDelay * time.Duration(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		e.setRequestHeaders(req)

		// Perform request
		resp, err := e.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check if we should retry based on status code
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// setRequestHeaders sets the request headers including User-Agent rotation
func (e *Engine) setRequestHeaders(req *http.Request) {
	// Set User-Agent with rotation
	req.Header.Set("User-Agent", e.getNextUserAgent())

	// Set default headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// Set custom headers from config
	for key, value := range e.config.Headers {
		req.Header.Set(key, value)
	}
}

// getNextUserAgent returns the next user agent from the pool
func (e *Engine) getNextUserAgent() string {
	e.uaMutex.Lock()
	defer e.uaMutex.Unlock()

	ua := e.userAgentPool[e.currentUAIndex]
	e.currentUAIndex = (e.currentUAIndex + 1) % len(e.userAgentPool)
	return ua
}

// extractField extracts a field from the document using the provided extractor
func (e *Engine) extractField(doc *goquery.Document, extractor FieldExtractor) (interface{}, error) {
	selection := doc.Find(extractor.Selector)

	if selection.Length() == 0 {
		if extractor.Required {
			return nil, fmt.Errorf("no elements found for selector: %s", extractor.Selector)
		}
		return nil, nil
	}

	var value interface{}

	switch extractor.Type {
	case "text":
		text := strings.TrimSpace(selection.First().Text())
		value = text

	case "html":
		html, err := selection.First().Html()
		if err != nil {
			return nil, err
		}
		value = html

	case "attr":
		if extractor.Attribute == "" {
			return nil, fmt.Errorf("attribute name required for attr type")
		}
		attr, exists := selection.First().Attr(extractor.Attribute)
		if !exists && extractor.Required {
			return nil, fmt.Errorf("attribute %s not found", extractor.Attribute)
		}
		value = attr

	case "list":
		var items []string
		selection.Each(func(i int, s *goquery.Selection) {
			items = append(items, strings.TrimSpace(s.Text()))
		})
		value = items

	default:
		return nil, fmt.Errorf("unsupported extractor type: %s", extractor.Type)
	}

	// Apply transformations if configured
	if len(extractor.Transform) > 0 {
		// Convert TransformRule to pipeline.TransformRule
		pipelineRules := make([]pipeline.TransformRule, len(extractor.Transform))
		for i, rule := range extractor.Transform {
			pipelineRules[i] = pipeline.TransformRule{
				Type:        rule.Type,
				Pattern:     rule.Pattern,
				Replacement: rule.Replacement,
			}
		}

		if extractor.Type == "list" {
			// Transform list values
			if items, ok := value.([]string); ok {
				transformed, err := pipeline.TransformList(items, pipelineRules)
				if err != nil {
					return nil, fmt.Errorf("transformation failed: %w", err)
				}
				value = transformed
			}
		} else {
			// Transform single value
			transformed, err := pipeline.TransformField(value, pipelineRules)
			if err != nil {
				return nil, fmt.Errorf("transformation failed: %w", err)
			}
			value = transformed
		}
	}

	return value, nil
}

// FieldExtractor defines how to extract a field from HTML
type FieldExtractor struct {
	Name      string          `yaml:"name" json:"name"`
	Selector  string          `yaml:"selector" json:"selector"`
	Type      string          `yaml:"type" json:"type"` // text, html, attr, list
	Attribute string          `yaml:"attribute,omitempty" json:"attribute,omitempty"`
	Required  bool            `yaml:"required,omitempty" json:"required,omitempty"`
	Transform []TransformRule `yaml:"transform,omitempty" json:"transform,omitempty"`
}

// TransformRule defines a transformation to apply to extracted data
type TransformRule struct {
	Type        string `yaml:"type" json:"type"`
	Pattern     string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Replacement string `yaml:"replacement,omitempty" json:"replacement,omitempty"`
}

// defaultUserAgents returns a list of common user agents for rotation
func defaultUserAgents() []string {
	return []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	}
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	rate   time.Duration
	burst  int
	tokens chan struct{}
	ticker *time.Ticker
	stopCh chan struct{}
	once   sync.Once
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate time.Duration, burst int) *RateLimiter {
	if burst <= 0 {
		burst = 1
	}
	if rate <= 0 {
		rate = time.Second
	}

	rl := &RateLimiter{
		rate:   rate,
		burst:  burst,
		tokens: make(chan struct{}, burst),
		stopCh: make(chan struct{}),
	}

	// Fill initial tokens
	for i := 0; i < burst; i++ {
		rl.tokens <- struct{}{}
	}

	// Start token refill goroutine
	rl.ticker = time.NewTicker(rate)
	go rl.refillTokens()

	return rl
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// refillTokens periodically adds tokens to the bucket
func (rl *RateLimiter) refillTokens() {
	for {
		select {
		case <-rl.ticker.C:
			select {
			case rl.tokens <- struct{}{}:
				// Token added successfully
			default:
				// Bucket is full, skip
			}
		case <-rl.stopCh:
			rl.ticker.Stop()
			return
		}
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	rl.once.Do(func() {
		close(rl.stopCh)
	})
}
