// internal/scraper/types.go
package scraper

import (
	"fmt"
	"time"

	"github.com/valpere/DataScrapexter/internal/config"
	"github.com/valpere/DataScrapexter/internal/pipeline"
)

// Common errors
var (
	ErrEmptySelector    = fmt.Errorf("selector cannot be empty")
	ErrInvalidSelector  = fmt.Errorf("invalid selector expression")
	ErrRequiredField    = fmt.Errorf("required field not found")
	ErrExtractionFailed = fmt.Errorf("field extraction failed")
	ErrTransformFailed  = fmt.Errorf("transformation failed")
	ErrInvalidConfig    = fmt.Errorf("invalid configuration")
)

// FieldConfig defines extraction configuration for a single field
type FieldConfig struct {
	Name      string                   `yaml:"name" json:"name"`
	Selector  string                   `yaml:"selector" json:"selector"`
	Type      string                   `yaml:"type" json:"type"`
	Required  bool                     `yaml:"required,omitempty" json:"required,omitempty"`
	Transform []pipeline.TransformRule `yaml:"transform,omitempty" json:"transform,omitempty"`
	Default   interface{}              `yaml:"default,omitempty" json:"default,omitempty"`
	Attribute string                   `yaml:"attribute,omitempty" json:"attribute,omitempty"`
}

// ExtractionConfig defines configuration for the extraction engine
type ExtractionConfig struct {
	StrictMode      bool `yaml:"strict_mode" json:"strict_mode"`
	ContinueOnError bool `yaml:"continue_on_error" json:"continue_on_error"`
}

// BatchScrapingConfig encapsulates all parameters for batch scraping operations
type BatchScrapingConfig struct {
	URLs           []string              `json:"urls"`
	Extractors     []FieldConfig         `json:"extractors"`
	ScraperConfig  *config.ScraperConfig `json:"scraper_config"`
	BatchSize      int                   `json:"batch_size"`
}

// NewBatchScrapingConfig creates a new BatchScrapingConfig with validation
func NewBatchScrapingConfig(urls []string, extractors []FieldConfig, scraperConfig *config.ScraperConfig, batchSize int) (*BatchScrapingConfig, error) {
	config := &BatchScrapingConfig{
		URLs:          urls,
		Extractors:    extractors,
		ScraperConfig: scraperConfig,
		BatchSize:     batchSize,
	}
	
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	return config, nil
}

// Validate checks if the BatchScrapingConfig is valid
func (bsc *BatchScrapingConfig) Validate() error {
	if len(bsc.URLs) == 0 {
		return fmt.Errorf("URLs list cannot be empty")
	}
	
	if len(bsc.Extractors) == 0 {
		return fmt.Errorf("Extractors list cannot be empty")
	}
	
	if bsc.ScraperConfig == nil {
		return fmt.Errorf("ScraperConfig cannot be nil")
	}
	
	if bsc.BatchSize <= 0 {
		bsc.BatchSize = 10 // Set default batch size
	}
	
	return nil
}

// EngineConfig defines scraping engine configuration
type EngineConfig struct {
	Fields           []FieldConfig    `yaml:"fields" json:"fields"`
	UserAgents       []string         `yaml:"user_agents,omitempty" json:"user_agents,omitempty"`
	RequestTimeout   time.Duration    `yaml:"request_timeout,omitempty" json:"request_timeout,omitempty"`
	RetryAttempts    int              `yaml:"retry_attempts,omitempty" json:"retry_attempts,omitempty"`
	MaxConcurrency   int              `yaml:"max_concurrency,omitempty" json:"max_concurrency,omitempty"`
	RateLimit        time.Duration    `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	ExtractionConfig ExtractionConfig `yaml:"extraction_config,omitempty" json:"extraction_config,omitempty"`
}

// ScrapingMetadata contains metadata about the scraping operation
type ScrapingMetadata struct {
	RequestDuration    string `json:"request_duration"`
	ExtractionDuration string `json:"extraction_duration"`
	URL                string `json:"url"`
	UserAgent          string `json:"user_agent,omitempty"`
	StatusCode         int    `json:"status_code"`
	ContentLength      int64  `json:"content_length,omitempty"`
	Timestamp          string `json:"timestamp"`
}

// ScrapingResult represents the result of a scraping operation
type ScrapingResult struct {
	URL        string                 `json:"url"`
	StatusCode int                    `json:"status_code"`
	Data       map[string]interface{} `json:"data"`
	Metadata   ScrapingMetadata       `json:"metadata"`
	Success    bool                   `json:"success"`
	Errors     []string               `json:"errors,omitempty"`
	Warnings   []string               `json:"warnings,omitempty"`
}

// FieldError represents an error during field extraction
type FieldError struct {
	FieldName string `json:"field_name"`
	Message   string `json:"message"`
	Selector  string `json:"selector,omitempty"`
	Code      string `json:"code,omitempty"`
	Severity  string `json:"severity,omitempty"`
}

// FieldWarning represents a warning during field extraction
type FieldWarning struct {
	FieldName string `json:"field_name"`
	Message   string `json:"message"`
	Selector  string `json:"selector,omitempty"`
}

// ExtractionResult represents the result of field extraction
type ExtractionResult struct {
	Data        map[string]interface{} `json:"data"`
	Errors      []FieldError           `json:"errors,omitempty"`
	Warnings    []FieldWarning         `json:"warnings,omitempty"`
	ProcessedAt time.Time              `json:"processed_at"`
	Duration    time.Duration          `json:"duration"`
	Success     bool                   `json:"success"`
	Metadata    ExtractionMetadata     `json:"metadata"`
}

// ExtractionMetadata contains metadata about the extraction operation
type ExtractionMetadata struct {
	TotalFields      int           `json:"total_fields"`
	ExtractedFields  int           `json:"extracted_fields"`
	FailedFields     int           `json:"failed_fields"`
	ProcessingTime   time.Duration `json:"processing_time"`
	RequiredFieldsOK bool          `json:"required_fields_ok"`
	DocumentSize     int64         `json:"document_size"`
	ErrorCount       int           `json:"error_count"`
	WarningCount     int           `json:"warning_count"`
	Duration         time.Duration `json:"duration"`
	Timestamp        time.Time     `json:"timestamp"`
}

// Selector represents a CSS selector with validation
type Selector struct {
	Expression string `yaml:"expression" json:"expression"`
	Validated  bool   `yaml:"-" json:"-"`
}

// ValidateSelector validates a CSS selector expression
func (s *Selector) ValidateSelector(expression string) error {
	// Basic validation - in a real implementation this would use goquery
	if expression == "" {
		return ErrEmptySelector
	}
	s.Expression = expression
	s.Validated = true
	return nil
}

// Config represents the scraper engine configuration
type Config struct {
	MaxRetries      int                  `yaml:"max_retries" json:"max_retries"`
	RetryDelay      time.Duration        `yaml:"retry_delay" json:"retry_delay"`
	Timeout         time.Duration        `yaml:"timeout" json:"timeout"`
	FollowRedirects bool                 `yaml:"follow_redirects" json:"follow_redirects"`
	MaxRedirects    int                  `yaml:"max_redirects" json:"max_redirects"`
	RateLimit       time.Duration        `yaml:"rate_limit" json:"rate_limit"`
	BurstSize       int                  `yaml:"burst_size" json:"burst_size"`
	Headers         map[string]string    `yaml:"headers" json:"headers"`
	UserAgents      []string             `yaml:"user_agents" json:"user_agents"`
	Browser         *BrowserConfig       `yaml:"browser" json:"browser"`
	Proxy           *ProxyConfig         `yaml:"proxy" json:"proxy"`
	Pagination      *PaginationConfig    `yaml:"pagination" json:"pagination"`
	RateLimiter     *RateLimiterConfig   `yaml:"rate_limiter" json:"rate_limiter"`
	ErrorRecovery   *ErrorRecoveryConfig `yaml:"error_recovery" json:"error_recovery"`
	MaxConcurrency  int                  `yaml:"max_concurrency" json:"max_concurrency"` // Maximum concurrent operations
}

// Validate validates the scraper configuration
func (c *Config) Validate() error {
	// Validate MaxConcurrency
	if c.MaxConcurrency < 0 {
		return fmt.Errorf("max_concurrency must be non-negative, got %d", c.MaxConcurrency)
	}
	if c.MaxConcurrency > 1000 {
		return fmt.Errorf("max_concurrency exceeds reasonable limit of 1000, got %d", c.MaxConcurrency)
	}
	
	// Validate other fields
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative, got %d", c.MaxRetries)
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative, got %v", c.Timeout)
	}
	if c.RateLimit < 0 {
		return fmt.Errorf("rate_limit must be non-negative, got %v", c.RateLimit)
	}
	if c.BurstSize < 0 {
		return fmt.Errorf("burst_size must be non-negative, got %d", c.BurstSize)
	}
	
	return nil
}

// ProxyConfig represents proxy configuration for the scraper
type ProxyConfig struct {
	Enabled          bool            `yaml:"enabled" json:"enabled"`
	Rotation         string          `yaml:"rotation" json:"rotation"`
	HealthCheck      bool            `yaml:"health_check" json:"health_check"`
	HealthCheckURL   string          `yaml:"health_check_url,omitempty" json:"health_check_url,omitempty"`
	HealthCheckRate  time.Duration   `yaml:"health_check_rate" json:"health_check_rate"`
	Timeout          time.Duration   `yaml:"timeout" json:"timeout"`
	MaxRetries       int             `yaml:"max_retries" json:"max_retries"`
	RetryDelay       time.Duration   `yaml:"retry_delay" json:"retry_delay"`
	Providers        []ProxyProvider `yaml:"providers" json:"providers"`
	FailureThreshold int             `yaml:"failure_threshold" json:"failure_threshold"`
	RecoveryTime     time.Duration   `yaml:"recovery_time" json:"recovery_time"`
	TLS              *ProxyTLSConfig `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// ProxyProvider represents a proxy provider configuration
type ProxyProvider struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type" json:"type"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
	Weight   int    `yaml:"weight,omitempty" json:"weight,omitempty"`
	Enabled  bool   `yaml:"enabled" json:"enabled"`
}

// ProxyTLSConfig represents TLS configuration for proxy connections
type ProxyTLSConfig struct {
	// InsecureSkipVerify controls whether certificate verification is skipped.
	// WARNING: Setting this to true is dangerous and makes connections vulnerable to attacks.
	InsecureSkipVerify bool     `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	ServerName         string   `yaml:"server_name,omitempty" json:"server_name,omitempty"`
	RootCAs            []string `yaml:"root_cas,omitempty" json:"root_cas,omitempty"`
	ClientCert         string   `yaml:"client_cert,omitempty" json:"client_cert,omitempty"`
	ClientKey          string   `yaml:"client_key,omitempty" json:"client_key,omitempty"`
	SuppressWarnings   bool     `yaml:"suppress_warnings,omitempty" json:"suppress_warnings,omitempty"`
}

// BrowserConfig represents browser-specific configuration for the scraper
type BrowserConfig struct {
	Enabled        bool          `yaml:"enabled" json:"enabled"`
	Headless       bool          `yaml:"headless" json:"headless"`
	UserDataDir    string        `yaml:"user_data_dir,omitempty" json:"user_data_dir,omitempty"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	ViewportWidth  int           `yaml:"viewport_width" json:"viewport_width"`
	ViewportHeight int           `yaml:"viewport_height" json:"viewport_height"`
	WaitForElement string        `yaml:"wait_for_element,omitempty" json:"wait_for_element,omitempty"`
	WaitDelay      time.Duration `yaml:"wait_delay,omitempty" json:"wait_delay,omitempty"`
	UserAgent      string        `yaml:"user_agent,omitempty" json:"user_agent,omitempty"`
	DisableImages  bool          `yaml:"disable_images" json:"disable_images"`
	DisableCSS     bool          `yaml:"disable_css" json:"disable_css"`
	DisableJS      bool          `yaml:"disable_js" json:"disable_js"`
}

// PaginationType represents different pagination strategies
type PaginationType string

const (
	PaginationTypeNextButton PaginationType = "next_button" // Click next button
	PaginationTypePages      PaginationType = "pages"       // Navigate through numbered pages
	PaginationTypeURLPattern PaginationType = "url_pattern" // URL pattern with page number
	PaginationTypeScrolling  PaginationType = "scrolling"   // Infinite scroll or load more
	PaginationTypeOffset     PaginationType = "offset"      // URL offset/limit parameters
)

// PaginationConfig represents pagination configuration
type PaginationConfig struct {
	Enabled   bool           `yaml:"enabled" json:"enabled"`
	Type      PaginationType `yaml:"type" json:"type"`
	MaxPages  int            `yaml:"max_pages,omitempty" json:"max_pages,omitempty"`
	StartPage int            `yaml:"start_page,omitempty" json:"start_page,omitempty"`

	// Next button pagination
	NextSelector   string        `yaml:"next_selector,omitempty" json:"next_selector,omitempty"`
	NextButtonText string        `yaml:"next_button_text,omitempty" json:"next_button_text,omitempty"`
	WaitAfterClick time.Duration `yaml:"wait_after_click,omitempty" json:"wait_after_click,omitempty"`

	// Page numbers pagination
	PageSelector   string `yaml:"page_selector,omitempty" json:"page_selector,omitempty"`
	PageURLPattern string `yaml:"page_url_pattern,omitempty" json:"page_url_pattern,omitempty"`

	// URL pattern pagination
	URLTemplate string `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	PageParam   string `yaml:"page_param,omitempty" json:"page_param,omitempty"`

	// Scrolling pagination
	ScrollSelector   string        `yaml:"scroll_selector,omitempty" json:"scroll_selector,omitempty"`
	LoadMoreSelector string        `yaml:"load_more_selector,omitempty" json:"load_more_selector,omitempty"`
	ScrollPause      time.Duration `yaml:"scroll_pause,omitempty" json:"scroll_pause,omitempty"`

	// Offset pagination
	OffsetParam string `yaml:"offset_param,omitempty" json:"offset_param,omitempty"`
	LimitParam  string `yaml:"limit_param,omitempty" json:"limit_param,omitempty"`
	PageSize    int    `yaml:"page_size,omitempty" json:"page_size,omitempty"`

	// General settings
	StopCondition     string        `yaml:"stop_condition,omitempty" json:"stop_condition,omitempty"`
	DelayBetweenPages time.Duration `yaml:"delay_between_pages,omitempty" json:"delay_between_pages,omitempty"`
	ContinueOnError   bool          `yaml:"continue_on_error" json:"continue_on_error"`
}

// PaginationResult represents the result of a paginated scraping operation
type PaginationResult struct {
	Pages          []ScrapingResult `json:"pages"`
	TotalPages     int              `json:"total_pages"`
	ProcessedPages int              `json:"processed_pages"`
	Success        bool             `json:"success"`
	Errors         []string         `json:"errors,omitempty"`
	Duration       time.Duration    `json:"duration"`
	StartTime      time.Time        `json:"start_time"`
	EndTime        time.Time        `json:"end_time"`
}

// ErrorRecoveryConfig configures comprehensive error recovery mechanisms
type ErrorRecoveryConfig struct {
	Enabled         bool                          `yaml:"enabled" json:"enabled"`
	CircuitBreakers map[string]CircuitBreakerSpec `yaml:"circuit_breakers,omitempty" json:"circuit_breakers,omitempty"`
	Fallbacks       map[string]FallbackSpec       `yaml:"fallbacks,omitempty" json:"fallbacks,omitempty"`
}

// CircuitBreakerSpec defines circuit breaker configuration for specific operations
type CircuitBreakerSpec struct {
	MaxFailures  int           `yaml:"max_failures" json:"max_failures"`
	ResetTimeout time.Duration `yaml:"reset_timeout" json:"reset_timeout"`
}

// FallbackSpec defines fallback strategy configuration for specific operations
type FallbackSpec struct {
	Strategy     string                 `yaml:"strategy" json:"strategy"` // "cached", "default", "alternative", "degrade"
	CacheTimeout time.Duration          `yaml:"cache_timeout,omitempty" json:"cache_timeout,omitempty"`
	DefaultValue interface{}            `yaml:"default_value,omitempty" json:"default_value,omitempty"`
	Alternative  string                 `yaml:"alternative,omitempty" json:"alternative,omitempty"`
	Degraded     map[string]interface{} `yaml:"degraded,omitempty" json:"degraded,omitempty"`
}

// Note: FieldExtractor is defined in extractor.go as a struct that processes fields
// For engine compatibility, we use FieldConfig as the configuration type
