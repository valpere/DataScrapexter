// internal/scraper/config_integration.go
package scraper

import (
	"fmt"
	"time"
)

// ConfigIntegration provides utilities for integrating different configuration formats
type ConfigIntegration struct{}

// NewConfigIntegration creates a new configuration integration helper
func NewConfigIntegration() *ConfigIntegration {
	return &ConfigIntegration{}
}

// CreateHTTPClientFromConfig creates an HTTP client from scraper configuration
func (ci *ConfigIntegration) CreateHTTPClientFromConfig(scraperConfig *ScraperConfig) (*HTTPClient, error) {
	if scraperConfig == nil {
		return nil, fmt.Errorf("scraper configuration cannot be nil")
	}

	if err := ci.validateScraperConfig(scraperConfig); err != nil {
		return nil, fmt.Errorf("invalid scraper configuration: %w", err)
	}

	httpClient, err := NewHTTPClient(scraperConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return httpClient, nil
}

// ApplyDefaultConfiguration applies default values to scraper configuration
func (ci *ConfigIntegration) ApplyDefaultConfiguration(scraperConfig *ScraperConfig) {
	if scraperConfig == nil {
		return
	}

	if scraperConfig.RequestTimeout <= 0 {
		scraperConfig.RequestTimeout = 30 * time.Second
	}

	if scraperConfig.RetryAttempts <= 0 {
		scraperConfig.RetryAttempts = 3
	}

	if scraperConfig.RetryDelay <= 0 {
		scraperConfig.RetryDelay = 1 * time.Second
	}

	if scraperConfig.RateLimit <= 0 {
		scraperConfig.RateLimit = 1.0
	}

	if scraperConfig.MaxRedirects <= 0 {
		scraperConfig.MaxRedirects = 10
	}

	if scraperConfig.MaxConcurrency <= 0 {
		scraperConfig.MaxConcurrency = 5
	}

	if scraperConfig.Headers == nil {
		scraperConfig.Headers = make(map[string]string)
	}

	if scraperConfig.Cookies == nil {
		scraperConfig.Cookies = make(map[string]string)
	}

	if len(scraperConfig.UserAgents) == 0 {
		scraperConfig.UserAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		}
	}
}

// validateScraperConfig validates the scraper configuration
func (ci *ConfigIntegration) validateScraperConfig(config *ScraperConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if config.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive, got: %v", config.RequestTimeout)
	}

	if config.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative, got: %d", config.RetryAttempts)
	}

	if config.RetryDelay < 0 {
		return fmt.Errorf("retry delay cannot be negative, got: %v", config.RetryDelay)
	}

	if config.RateLimit <= 0 {
		return fmt.Errorf("rate limit must be positive, got: %f", config.RateLimit)
	}

	if config.MaxRedirects < 0 {
		return fmt.Errorf("max redirects cannot be negative, got: %d", config.MaxRedirects)
	}

	if config.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive, got: %d", config.MaxConcurrency)
	}

	return nil
}

// GetConfigurationSummary provides a summary of the current configuration
func (ci *ConfigIntegration) GetConfigurationSummary(config *ScraperConfig) map[string]interface{} {
	if config == nil {
		return map[string]interface{}{
			"status": "no configuration provided",
		}
	}

	return map[string]interface{}{
		"request_timeout":    config.RequestTimeout.String(),
		"retry_attempts":     config.RetryAttempts,
		"retry_delay":        config.RetryDelay.String(),
		"rate_limit":         config.RateLimit,
		"max_redirects":      config.MaxRedirects,
		"max_concurrency":    config.MaxConcurrency,
		"ignore_ssl_errors":  config.IgnoreSSLErrors,
		"proxy_configured":   config.ProxyURL != "",
		"user_agents_count":  len(config.UserAgents),
		"headers_count":      len(config.Headers),
		"cookies_count":      len(config.Cookies),
	}
}
