// internal/proxy/health.go
package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DefaultHealthChecker implements basic HTTP health checking
type DefaultHealthChecker struct {
	client         *http.Client
	healthCheckURL string
	timeout        time.Duration
}

// NewDefaultHealthChecker creates a new default health checker
func NewDefaultHealthChecker(timeout time.Duration, healthCheckURL string) *DefaultHealthChecker {
	if healthCheckURL == "" {
		healthCheckURL = DefaultHealthCheckURL
	}
	// Use secure TLS configuration by default
	transport := &http.Transport{
		TLSClientConfig: GetDefaultTLSConfig(),
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &DefaultHealthChecker{
		client:         client,
		healthCheckURL: healthCheckURL,
		timeout:        timeout,
	}
}

// Check performs a health check on the given proxy
func (hc *DefaultHealthChecker) Check(proxy *ProxyInstance) error {
	if proxy == nil || proxy.URL == nil {
		return fmt.Errorf("invalid proxy instance")
	}

	// Create transport with proxy using secure TLS by default
	transport := &http.Transport{
		Proxy:           http.ProxyURL(proxy.URL),
		TLSClientConfig: GetDefaultTLSConfig(),
	}

	// Create client with proxy transport
	client := &http.Client{
		Transport: transport,
		Timeout:   hc.timeout,
	}

	// Make request through proxy
	start := time.Now()
	resp, err := client.Get(hc.healthCheckURL)
	duration := time.Since(start)

	if err != nil {
		proxy.mu.Lock()
		proxy.Status.Available = false
		proxy.Status.FailureCount++
		proxy.Status.LastFailure = time.Now()
		proxy.Status.ResponseTime = duration
		proxy.mu.Unlock()
		return fmt.Errorf("proxy health check failed: %v", err)
	}

	defer resp.Body.Close()

	// Update proxy status
	proxy.mu.Lock()
	proxy.Status.Available = resp.StatusCode == http.StatusOK
	proxy.Status.ResponseTime = duration
	proxy.Status.LastChecked = time.Now()

	if resp.StatusCode == http.StatusOK {
		proxy.Status.LastSuccess = time.Now()
		proxy.Status.FailureCount = 0
	} else {
		proxy.Status.FailureCount++
		proxy.Status.LastFailure = time.Now()
	}
	proxy.mu.Unlock()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy health check returned status %d", resp.StatusCode)
	}

	return nil
}

// GetHealthCheckURL returns the current health check URL
func (hc *DefaultHealthChecker) GetHealthCheckURL() string {
	return hc.healthCheckURL
}

// SetHealthCheckURL sets the health check URL
func (hc *DefaultHealthChecker) SetHealthCheckURL(url string) {
	hc.healthCheckURL = url
}

// CustomHealthChecker allows for custom health check implementations
type CustomHealthChecker struct {
	checkFunc      func(*ProxyInstance) error
	healthCheckURL string
}

// NewCustomHealthChecker creates a new custom health checker
func NewCustomHealthChecker(checkFunc func(*ProxyInstance) error, healthCheckURL string) *CustomHealthChecker {
	return &CustomHealthChecker{
		checkFunc:      checkFunc,
		healthCheckURL: healthCheckURL,
	}
}

// Check performs the custom health check
func (chc *CustomHealthChecker) Check(proxy *ProxyInstance) error {
	if chc.checkFunc == nil {
		return fmt.Errorf("no check function defined")
	}
	return chc.checkFunc(proxy)
}

// GetHealthCheckURL returns the health check URL
func (chc *CustomHealthChecker) GetHealthCheckURL() string {
	return chc.healthCheckURL
}

// SetHealthCheckURL sets the health check URL
func (chc *CustomHealthChecker) SetHealthCheckURL(url string) {
	chc.healthCheckURL = url
}

// ProxyValidator provides validation for proxy configurations
type ProxyValidator struct{}

// NewProxyValidator creates a new proxy validator
func NewProxyValidator() *ProxyValidator {
	return &ProxyValidator{}
}

// ValidateProvider validates a proxy provider configuration
func (pv *ProxyValidator) ValidateProvider(provider *ProxyProvider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	if provider.Name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if provider.Host == "" {
		return fmt.Errorf("provider host cannot be empty")
	}

	if provider.Port <= 0 || provider.Port > 65535 {
		return fmt.Errorf("provider port must be between 1 and 65535")
	}

	switch provider.Type {
	case ProxyTypeHTTP, ProxyTypeHTTPS, ProxyTypeSOCKS5:
		// Valid types
	default:
		return fmt.Errorf("unsupported proxy type: %s", provider.Type)
	}

	return nil
}

// ValidateConfig validates a proxy configuration
func (pv *ProxyValidator) ValidateConfig(config *ProxyConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Enabled && len(config.Providers) == 0 {
		return fmt.Errorf("proxy enabled but no providers configured")
	}

	if config.HealthCheckRate < 0 {
		return fmt.Errorf("health check rate cannot be negative")
	}

	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}

	if config.RecoveryTime <= 0 {
		config.RecoveryTime = 10 * time.Minute
	}

	// Validate each provider
	for i, provider := range config.Providers {
		if err := pv.ValidateProvider(&provider); err != nil {
			return fmt.Errorf("provider %d validation failed: %v", i, err)
		}
	}

	return nil
}

// TestProxy tests a single proxy configuration
func TestProxy(provider *ProxyProvider, testURL string, timeout time.Duration) error {
	if testURL == "" {
		testURL = "http://httpbin.org/ip"
	}

	// Build proxy URL
	proxyURLStr := fmt.Sprintf("%s://%s:%d", provider.Type, provider.Host, provider.Port)
	if provider.Username != "" && provider.Password != "" {
		proxyURLStr = fmt.Sprintf("%s://%s:%s@%s:%d",
			provider.Type, provider.Username, provider.Password, provider.Host, provider.Port)
	}

	// Parse proxy URL
	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return fmt.Errorf("invalid proxy URL: %v", err)
	}

	// Create HTTP client with proxy using secure TLS by default
	transport := &http.Transport{
		Proxy:           http.ProxyURL(proxyURL),
		TLSClientConfig: GetDefaultTLSConfig(),
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	// Test connection
	resp, err := client.Get(testURL)
	if err != nil {
		return fmt.Errorf("proxy test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy test returned status %d", resp.StatusCode)
	}

	return nil
}
