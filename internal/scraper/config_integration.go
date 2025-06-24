// internal/scraper/config_integration.go
package scraper

import (
    "fmt"
    "time"

    "github.com/valpere/DataScrapexter/internal/config"
)

// ClientFactory creates HTTP clients configured from ScraperConfig
type ClientFactory struct{}

// NewClientFactory creates a new client factory instance
func NewClientFactory() *ClientFactory {
    return &ClientFactory{}
}

// CreateClient builds an HTTPClient from ScraperConfig with all configuration applied
func (f *ClientFactory) CreateClient(scraperConfig *config.ScraperConfig) (*HTTPClient, error) {
    if scraperConfig == nil {
        return nil, fmt.Errorf("scraper configuration cannot be nil")
    }

    clientConfig := f.buildClientConfig(scraperConfig)
    
    if err := f.validateClientConfig(clientConfig); err != nil {
        return nil, fmt.Errorf("invalid client configuration: %w", err)
    }

    return NewHTTPClient(clientConfig), nil
}

// buildClientConfig transforms ScraperConfig into ClientConfig
func (f *ClientFactory) buildClientConfig(scraperConfig *config.ScraperConfig) ClientConfig {
    clientConfig := ClientConfig{
        Timeout:       scraperConfig.RequestTimeout,
        RetryAttempts: scraperConfig.RetryAttempts,
        RetryDelay:    scraperConfig.RetryDelay,
        Headers:       make(map[string]string),
        Cookies:       make(map[string]string),
    }

    // Apply rate limiting configuration
    f.configureRateLimiting(&clientConfig, scraperConfig)

    // Apply user agents from configuration
    f.configureUserAgents(&clientConfig, scraperConfig)

    // Apply custom headers
    f.configureHeaders(&clientConfig, scraperConfig)

    // Apply cookies
    f.configureCookies(&clientConfig, scraperConfig)

    // Apply anti-detection settings if configured
    f.configureAntiDetection(&clientConfig, scraperConfig)

    return clientConfig
}

// configureRateLimiting applies rate limiting settings from ScraperConfig
func (f *ClientFactory) configureRateLimiting(clientConfig *ClientConfig, scraperConfig *config.ScraperConfig) {
    // Default rate limiting values
    clientConfig.RateLimit = 1.0  // 1 request per second
    clientConfig.RateBurst = 5    // burst of 5 requests

    // Parse rate_limit string if provided (e.g., "2s", "500ms")
    if scraperConfig.RateLimit != "" {
        if duration, err := time.ParseDuration(scraperConfig.RateLimit); err == nil {
            // Convert duration to requests per second
            if duration > 0 {
                clientConfig.RateLimit = 1.0 / duration.Seconds()
            }
        }
    }

    // Apply anti-detection rate limiting if configured
    if scraperConfig.AntiDetection != nil && scraperConfig.AntiDetection.RateLimiting.RequestsPerSecond > 0 {
        clientConfig.RateLimit = scraperConfig.AntiDetection.RateLimiting.RequestsPerSecond
        if scraperConfig.AntiDetection.RateLimiting.Burst > 0 {
            clientConfig.RateBurst = scraperConfig.AntiDetection.RateLimiting.Burst
        }
    }
}

// configureUserAgents applies user agent settings from ScraperConfig
func (f *ClientFactory) configureUserAgents(clientConfig *ClientConfig, scraperConfig *config.ScraperConfig) {
    // Start with user agents from main config
    if len(scraperConfig.UserAgents) > 0 {
        clientConfig.UserAgents = make([]string, len(scraperConfig.UserAgents))
        copy(clientConfig.UserAgents, scraperConfig.UserAgents)
    }

    // Override with anti-detection user agents if configured
    if scraperConfig.AntiDetection != nil && len(scraperConfig.AntiDetection.UserAgents) > 0 {
        clientConfig.UserAgents = make([]string, len(scraperConfig.AntiDetection.UserAgents))
        copy(clientConfig.UserAgents, scraperConfig.AntiDetection.UserAgents)
    }

    // If no user agents specified, the HTTPClient will use defaults
}

// configureHeaders applies custom headers from ScraperConfig
func (f *ClientFactory) configureHeaders(clientConfig *ClientConfig, scraperConfig *config.ScraperConfig) {
    // Copy headers from main configuration
    if scraperConfig.Headers != nil {
        for key, value := range scraperConfig.Headers {
            clientConfig.Headers[key] = value
        }
    }

    // Apply browser configuration headers if present
    if scraperConfig.Browser != nil && scraperConfig.Browser.Enabled {
        f.applyBrowserHeaders(clientConfig, scraperConfig.Browser)
    }
}

// applyBrowserHeaders configures headers based on browser configuration
func (f *ClientFactory) applyBrowserHeaders(clientConfig *ClientConfig, browserConfig *config.BrowserConfig) {
    // Set user agent from browser config if specified
    if browserConfig.UserAgent != "" {
        // When browser user agent is specified, use it exclusively
        clientConfig.UserAgents = []string{browserConfig.UserAgent}
    }

    // Add browser-specific headers
    if browserConfig.DisableJavaScript {
        clientConfig.Headers["X-Requested-With"] = "XMLHttpRequest"
    }

    // Configure viewport-related headers if viewport is specified
    if browserConfig.ViewportWidth > 0 && browserConfig.ViewportHeight > 0 {
        clientConfig.Headers["Viewport-Width"] = fmt.Sprintf("%d", browserConfig.ViewportWidth)
    }
}

// configureCookies applies cookie settings from ScraperConfig
func (f *ClientFactory) configureCookies(clientConfig *ClientConfig, scraperConfig *config.ScraperConfig) {
    // Copy cookies from main configuration
    if scraperConfig.Cookies != nil {
        for name, value := range scraperConfig.Cookies {
            clientConfig.Cookies[name] = value
        }
    }

    // Apply browser cookies if configured
    if scraperConfig.Browser != nil && scraperConfig.Browser.Cookies != nil {
        for name, value := range scraperConfig.Browser.Cookies {
            clientConfig.Cookies[name] = value
        }
    }
}

// configureAntiDetection applies anti-detection specific settings
func (f *ClientFactory) configureAntiDetection(clientConfig *ClientConfig, scraperConfig *config.ScraperConfig) {
    if scraperConfig.AntiDetection == nil {
        return
    }

    antiDetect := scraperConfig.AntiDetection

    // Apply delay settings to retry configuration
    if antiDetect.DelayMin > 0 {
        clientConfig.RetryDelay = antiDetect.DelayMin
    }

    // Configure randomized headers if enabled
    if antiDetect.RandomizeHeaders {
        f.addRandomizedHeaders(clientConfig)
    }

    // Configure proxy settings (basic implementation)
    if antiDetect.Proxy.Enabled && len(antiDetect.Proxy.URLs) > 0 {
        // Note: Full proxy implementation would require additional HTTP transport configuration
        clientConfig.Headers["X-Proxy-Enabled"] = "true"
    }
}

// addRandomizedHeaders adds headers that help with anti-detection
func (f *ClientFactory) addRandomizedHeaders(clientConfig *ClientConfig) {
    // Add various browser-like headers with slight randomization
    randomHeaders := map[string][]string{
        "Accept-Charset":  {"utf-8", "iso-8859-1;q=0.5"},
        "Cache-Control":   {"no-cache", "max-age=0"},
        "Pragma":          {"no-cache"},
        "Sec-Fetch-Dest":  {"document"},
        "Sec-Fetch-Mode":  {"navigate"},
        "Sec-Fetch-Site":  {"none"},
        "Sec-Fetch-User":  {"?1"},
    }

    // Apply one random header from each category
    for header, options := range randomHeaders {
        if len(options) > 0 {
            // For simplicity, use first option. In production, this could be randomized
            clientConfig.Headers[header] = options[0]
        }
    }
}

// validateClientConfig ensures the client configuration is valid and complete
func (f *ClientFactory) validateClientConfig(clientConfig ClientConfig) error {
    if clientConfig.Timeout <= 0 {
        return fmt.Errorf("timeout must be positive, got: %v", clientConfig.Timeout)
    }

    if clientConfig.RetryAttempts < 0 {
        return fmt.Errorf("retry attempts cannot be negative, got: %d", clientConfig.RetryAttempts)
    }

    if clientConfig.RetryDelay < 0 {
        return fmt.Errorf("retry delay cannot be negative, got: %v", clientConfig.RetryDelay)
    }

    if clientConfig.RateLimit <= 0 {
        return fmt.Errorf("rate limit must be positive, got: %f", clientConfig.RateLimit)
    }

    if clientConfig.RateBurst <= 0 {
        return fmt.Errorf("rate burst must be positive, got: %d", clientConfig.RateBurst)
    }

    return nil
}

// ConfigurationSummary provides a summary of applied client configuration
type ConfigurationSummary struct {
    Timeout       time.Duration
    RetryAttempts int
    RetryDelay    time.Duration
    RateLimit     float64
    RateBurst     int
    UserAgents    int
    Headers       int
    Cookies       int
    AntiDetection bool
}

// GetConfigurationSummary returns a summary of the configuration applied to the client
func (f *ClientFactory) GetConfigurationSummary(scraperConfig *config.ScraperConfig) ConfigurationSummary {
    clientConfig := f.buildClientConfig(scraperConfig)
    
    return ConfigurationSummary{
        Timeout:       clientConfig.Timeout,
        RetryAttempts: clientConfig.RetryAttempts,
        RetryDelay:    clientConfig.RetryDelay,
        RateLimit:     clientConfig.RateLimit,
        RateBurst:     clientConfig.RateBurst,
        UserAgents:    len(clientConfig.UserAgents),
        Headers:       len(clientConfig.Headers),
        Cookies:       len(clientConfig.Cookies),
        AntiDetection: scraperConfig.AntiDetection != nil,
    }
}

// ClientBuilder provides a fluent interface for building configured HTTP clients
type ClientBuilder struct {
    config *config.ScraperConfig
    factory *ClientFactory
}

// NewClientBuilder creates a new client builder with the specified configuration
func NewClientBuilder(scraperConfig *config.ScraperConfig) *ClientBuilder {
    return &ClientBuilder{
        config: scraperConfig,
        factory: NewClientFactory(),
    }
}

// WithTimeout overrides the timeout configuration
func (b *ClientBuilder) WithTimeout(timeout time.Duration) *ClientBuilder {
    if b.config != nil {
        b.config.RequestTimeout = timeout
    }
    return b
}

// WithRetries overrides the retry configuration
func (b *ClientBuilder) WithRetries(attempts int, delay time.Duration) *ClientBuilder {
    if b.config != nil {
        b.config.RetryAttempts = attempts
        b.config.RetryDelay = delay
    }
    return b
}

// WithRateLimit overrides the rate limiting configuration
func (b *ClientBuilder) WithRateLimit(requestsPerSecond float64) *ClientBuilder {
    if b.config != nil {
        if b.config.AntiDetection == nil {
            b.config.AntiDetection = &config.AntiDetectionConfig{}
        }
        b.config.AntiDetection.RateLimiting.RequestsPerSecond = requestsPerSecond
    }
    return b
}

// Build creates the configured HTTP client
func (b *ClientBuilder) Build() (*HTTPClient, error) {
    if b.config == nil {
        return nil, fmt.Errorf("scraper configuration is required")
    }
    
    return b.factory.CreateClient(b.config)
}

// DefaultClientConfig creates a ScraperConfig with sensible defaults for HTTP client usage
func DefaultClientConfig() *config.ScraperConfig {
    return &config.ScraperConfig{
        RequestTimeout: 30 * time.Second,
        RetryAttempts:  3,
        RetryDelay:     time.Second,
        RateLimit:      "1s",
        Headers: map[string]string{
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        },
        AntiDetection: &config.AntiDetectionConfig{
            RateLimiting: config.RateLimitConfig{
                RequestsPerSecond: 1.0,
                Burst:            5,
            },
            RandomizeHeaders: true,
        },
    }
}
