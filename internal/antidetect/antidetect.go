// internal/antidetect/antidetect.go
package antidetect

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// UserAgentRotator rotates user agents
type UserAgentRotator struct {
	agents []string
	mu     sync.RWMutex
	index  int
}

// NewUserAgentRotator creates a new user agent rotator
func NewUserAgentRotator(agents []string) *UserAgentRotator {
	if len(agents) == 0 {
		agents = getDefaultUserAgents()
	}
	return &UserAgentRotator{
		agents: agents,
	}
}

// GetNext returns the next user agent
func (r *UserAgentRotator) GetNext() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent := r.agents[r.index]
	r.index = (r.index + 1) % len(r.agents)
	return agent
}

// GetRandom returns a random user agent
func (r *UserAgentRotator) GetRandom() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.agents[rand.Intn(len(r.agents))]
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond int, burst int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
	}
}

// Wait waits for permission to proceed
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

// ProxyRotator manages proxy rotation
type ProxyRotator struct {
	proxies []string
	healthy map[string]bool
	mu      sync.RWMutex
	index   int
}

// NewProxyRotator creates a new proxy rotator
func NewProxyRotator(proxies []string) *ProxyRotator {
	healthy := make(map[string]bool)
	for _, proxy := range proxies {
		healthy[proxy] = true
	}

	return &ProxyRotator{
		proxies: proxies,
		healthy: healthy,
	}
}

// GetNext returns the next healthy proxy
func (pr *ProxyRotator) GetNext() string {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	startIndex := pr.index
	for {
		proxy := pr.proxies[pr.index]
		pr.index = (pr.index + 1) % len(pr.proxies)

		if pr.healthy[proxy] {
			return proxy
		}

		// If we've checked all proxies and none are healthy, return the first one
		if pr.index == startIndex {
			return pr.proxies[0]
		}
	}
}

// MarkHealthy marks a proxy as healthy
func (pr *ProxyRotator) MarkHealthy(proxy string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.healthy[proxy] = true
}

// MarkUnhealthy marks a proxy as unhealthy
func (pr *ProxyRotator) MarkUnhealthy(proxy string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.healthy[proxy] = false
}

// HeaderRotator rotates HTTP headers
type HeaderRotator struct {
	userAgentRotator *UserAgentRotator
}

// NewHeaderRotator creates a new header rotator
func NewHeaderRotator() *HeaderRotator {
	return &HeaderRotator{
		userAgentRotator: NewUserAgentRotator(nil),
	}
}

// GetHeaders returns a set of headers
func (hr *HeaderRotator) GetHeaders() http.Header {
	headers := make(http.Header)

	headers.Set("User-Agent", hr.userAgentRotator.GetRandom())
	headers.Set("Accept", getRandomAccept())
	headers.Set("Accept-Language", getRandomAcceptLanguage())
	headers.Set("Accept-Encoding", "gzip, deflate, br")
	headers.Set("DNT", "1")
	headers.Set("Connection", "keep-alive")
	headers.Set("Upgrade-Insecure-Requests", "1")

	return headers
}

// DelayRandomizer provides random delays
type DelayRandomizer struct {
	min time.Duration
	max time.Duration
}

// NewDelayRandomizer creates a new delay randomizer
func NewDelayRandomizer(min, max time.Duration) *DelayRandomizer {
	return &DelayRandomizer{min: min, max: max}
}

// GetDelay returns a random delay within the configured range
func (dr *DelayRandomizer) GetDelay() time.Duration {
	diff := dr.max - dr.min
	return dr.min + time.Duration(rand.Int63n(int64(diff)))
}

// SessionManager manages scraping sessions
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// Session represents a scraping session
type Session struct {
	ID       string
	Cookies  http.CookieJar
	Headers  http.Header
	Created  time.Time
	LastUsed time.Time
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession(id string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	jar, _ := cookiejar.New(nil)
	session := &Session{
		ID:       id,
		Cookies:  jar,
		Headers:  make(http.Header),
		Created:  time.Now(),
		LastUsed: time.Now(),
	}

	sm.sessions[id] = session
	return session
}

// GetSession retrieves a session
func (sm *SessionManager) GetSession(id string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if session, exists := sm.sessions[id]; exists {
		session.LastUsed = time.Now()
		return session
	}
	return nil
}

// CleanupSession removes a session
func (sm *SessionManager) CleanupSession(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, id)
}

// BrowserFingerprinter generates browser fingerprints
type BrowserFingerprinter struct{}

// BrowserFingerprint represents a browser fingerprint
type BrowserFingerprint struct {
	UserAgent string
	Viewport  Viewport
	Languages []string
	Platform  string
	Timezone  string
}

// Viewport represents screen dimensions
type Viewport struct {
	Width  int
	Height int
}

// NewBrowserFingerprinter creates a new browser fingerprinter
func NewBrowserFingerprinter() *BrowserFingerprinter {
	return &BrowserFingerprinter{}
}

// Generate creates a new browser fingerprint
func (bf *BrowserFingerprinter) Generate() *BrowserFingerprint {
	viewports := []Viewport{
		{1920, 1080}, {1366, 768}, {1536, 864}, {1440, 900}, {1280, 720},
	}

	languages := [][]string{
		{"en-US", "en"},
		{"en-GB", "en"},
		{"fr-FR", "fr"},
		{"de-DE", "de"},
		{"es-ES", "es"},
	}

	platforms := []string{
		"Win32", "MacIntel", "Linux x86_64",
	}

	timezones := []string{
		"America/New_York", "Europe/London", "Europe/Paris", "Asia/Tokyo",
	}

	return &BrowserFingerprint{
		UserAgent: NewUserAgentRotator(nil).GetRandom(),
		Viewport:  viewports[rand.Intn(len(viewports))],
		Languages: languages[rand.Intn(len(languages))],
		Platform:  platforms[rand.Intn(len(platforms))],
		Timezone:  timezones[rand.Intn(len(timezones))],
	}
}

// CaptchaDetector detects CAPTCHAs in HTML content
type CaptchaDetector struct{}

// CaptchaType represents the type of CAPTCHA
type CaptchaType int

const (
	NoCaptcha CaptchaType = iota
	RecaptchaV2
	RecaptchaV3
	HCaptcha
	FunCaptcha
	ImageCaptcha // Generic image-based CAPTCHA type
)

// NewCaptchaDetector creates a new CAPTCHA detector
func NewCaptchaDetector() *CaptchaDetector {
	return &CaptchaDetector{}
}

// Detect detects CAPTCHA type in HTML content
func (cd *CaptchaDetector) Detect(html string) (CaptchaType, bool) {
	html = strings.ToLower(html)

	if strings.Contains(html, "g-recaptcha") {
		return RecaptchaV2, true
	}

	if strings.Contains(html, "recaptcha/api.js?render=") {
		return RecaptchaV3, true
	}

	if strings.Contains(html, "h-captcha") {
		return HCaptcha, true
	}

	if strings.Contains(html, "funcaptcha") || strings.Contains(html, "arkoselabs") {
		return FunCaptcha, true
	}

	return NoCaptcha, false
}

// AntiDetectionConfig configuration for anti-detection measures
type AntiDetectionConfig struct {
	UserAgentRotation bool
	HeaderRotation    bool
	DelayRange        DelayRange
	RetryConfig       RetryConfig
	ProxyConfig       ProxyConfig
}

// DelayRange represents min/max delay range
type DelayRange struct {
	Min time.Duration
	Max time.Duration
}

// RetryConfig configuration for retry mechanism
type RetryConfig struct {
	MaxRetries int
	BackoffMin time.Duration
	BackoffMax time.Duration
}

// ProxyConfig configuration for proxy usage
type ProxyConfig struct {
	Enabled bool
	URLs    []string
}

// AntiDetectionClient HTTP client with anti-detection features
type AntiDetectionClient struct {
	client           *http.Client
	config           *AntiDetectionConfig
	userAgentRotator *UserAgentRotator
	headerRotator    *HeaderRotator
	delayRandomizer  *DelayRandomizer
	proxyRotator     *ProxyRotator
}

// NewAntiDetectionClient creates a new anti-detection HTTP client
func NewAntiDetectionClient(config *AntiDetectionConfig) *AntiDetectionClient {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var proxyRotator *ProxyRotator
	if config.ProxyConfig.Enabled && len(config.ProxyConfig.URLs) > 0 {
		proxyRotator = NewProxyRotator(config.ProxyConfig.URLs)
	}

	return &AntiDetectionClient{
		client:           client,
		config:           config,
		userAgentRotator: NewUserAgentRotator(nil),
		headerRotator:    NewHeaderRotator(),
		delayRandomizer:  NewDelayRandomizer(config.DelayRange.Min, config.DelayRange.Max),
		proxyRotator:     proxyRotator,
	}
}

// Do executes an HTTP request with anti-detection measures
func (adc *AntiDetectionClient) Do(req *http.Request) (*http.Response, error) {
	// Apply anti-detection headers
	if adc.config.UserAgentRotation {
		req.Header.Set("User-Agent", adc.userAgentRotator.GetRandom())
	}

	if adc.config.HeaderRotation {
		headers := adc.headerRotator.GetHeaders()
		for key, values := range headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// Add random delay
	if adc.delayRandomizer != nil {
		delay := adc.delayRandomizer.GetDelay()
		time.Sleep(delay)
	}

	// Execute request with retry logic
	return adc.executeWithRetry(req)
}

// executeWithRetry executes a request with retry logic
func (adc *AntiDetectionClient) executeWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= adc.config.RetryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Backoff delay
			backoff := adc.calculateBackoff(attempt)
			time.Sleep(backoff)
		}

		resp, err := adc.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check if we should retry based on status code
		if adc.shouldRetry(resp.StatusCode) {
			resp.Body.Close()
			lastErr = fmt.Errorf("received status code %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %v", adc.config.RetryConfig.MaxRetries+1, lastErr)
}

// shouldRetry determines if a request should be retried based on status code
func (adc *AntiDetectionClient) shouldRetry(statusCode int) bool {
	retryableCodes := []int{429, 502, 503, 504}
	for _, code := range retryableCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

// calculateBackoff calculates exponential backoff delay
func (adc *AntiDetectionClient) calculateBackoff(attempt int) time.Duration {
	backoff := adc.config.RetryConfig.BackoffMin * time.Duration(1<<uint(attempt))
	if backoff > adc.config.RetryConfig.BackoffMax {
		backoff = adc.config.RetryConfig.BackoffMax
	}
	return backoff
}

// Helper functions
func getDefaultUserAgents() []string {
	return []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

func getRandomAccept() string {
	accepts := []string{
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	}
	return accepts[rand.Intn(len(accepts))]
}

func getRandomAcceptLanguage() string {
	languages := []string{
		"en-US,en;q=0.9",
		"en-GB,en;q=0.9",
		"en-US,en;q=0.9,fr;q=0.8",
		"en-US,en;q=0.9,es;q=0.8",
		"en-US,en;q=0.9,de;q=0.8",
	}
	return languages[rand.Intn(len(languages))]
}
