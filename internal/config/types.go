// internal/config/types.go

// Package config provides configuration types and structures for DataScrapexter.
// It defines the various configuration options available for scraping operations,
// including target URLs, extraction rules, pagination settings, and output formats.
package config

import (
	"time"
)

// Config represents the main configuration structure for a scraping job.
// It contains all the settings needed to scrape a website and extract data.
type Config struct {
	// Name identifies this configuration
	Name string `yaml:"name" json:"name"`
	
	// Version of the configuration format
	Version string `yaml:"version" json:"version"`
	
	// Description provides human-readable information about this config
	Description string `yaml:"description" json:"description"`
	
	// Target defines the website to scrape
	Target TargetConfig `yaml:"target" json:"target"`
	
	// Extraction rules for data extraction
	Extraction []ExtractionRule `yaml:"extraction" json:"extraction"`
	
	// Pagination settings for multi-page scraping
	Pagination *PaginationConfig `yaml:"pagination,omitempty" json:"pagination,omitempty"`
	
	// Output configuration
	Output OutputConfig `yaml:"output" json:"output"`
	
	// Request configuration
	Request RequestConfig `yaml:"request" json:"request"`
	
	// AntiDetect settings to avoid being blocked
	AntiDetect AntiDetectConfig `yaml:"antidetect" json:"antidetect"`
}

// TargetConfig defines the target website configuration.
type TargetConfig struct {
	// URL is the starting URL for scraping
	URL string `yaml:"url" json:"url"`
	
	// Method is the HTTP method (GET, POST, etc.)
	Method string `yaml:"method" json:"method"`
	
	// Headers to send with requests
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	
	// Body for POST requests
	Body string `yaml:"body,omitempty" json:"body,omitempty"`
}

// ExtractionRule defines how to extract data from pages.
type ExtractionRule struct {
	// Name of the extraction rule
	Name string `yaml:"name" json:"name"`
	
	// Type of extraction (listing, detail, etc.)
	Type string `yaml:"type" json:"type"`
	
	// Container selector for the items
	Container ContainerConfig `yaml:"container,omitempty" json:"container,omitempty"`
	
	// Fields to extract
	Fields []FieldConfig `yaml:"fields" json:"fields"`
}

// ContainerConfig defines a container element that holds multiple items.
type ContainerConfig struct {
	// Selector is the CSS selector for the container
	Selector string `yaml:"selector" json:"selector"`
	
	// Multiple indicates if multiple containers are expected
	Multiple bool `yaml:"multiple" json:"multiple"`
}

// FieldConfig defines how to extract a single field.
type FieldConfig struct {
	// Name of the field
	Name string `yaml:"name" json:"name"`
	
	// Selector is the CSS selector for the field
	Selector string `yaml:"selector" json:"selector"`
	
	// Type of the field (text, attribute, etc.)
	Type string `yaml:"type" json:"type"`
	
	// Attribute to extract (if type is attribute)
	Attribute string `yaml:"attribute,omitempty" json:"attribute,omitempty"`
	
	// Transform rules to apply
	Transform []TransformConfig `yaml:"transform,omitempty" json:"transform,omitempty"`
	
	// Required indicates if this field must be present
	Required bool `yaml:"required" json:"required"`
	
	// Default value if field is missing
	Default string `yaml:"default,omitempty" json:"default,omitempty"`
}

// TransformConfig defines a transformation to apply to extracted data.
type TransformConfig struct {
	// Type of transformation
	Type string `yaml:"type" json:"type"`
	
	// Parameters for the transformation
	Params map[string]interface{} `yaml:"params,omitempty" json:"params,omitempty"`
}

// PaginationConfig defines pagination settings for multi-page scraping.
type PaginationConfig struct {
	// Strategy for pagination (numbered, next_button, load_more, etc.)
	Strategy string `yaml:"strategy" json:"strategy"`
	
	// MaxPages limits the number of pages to scrape
	MaxPages int `yaml:"max_pages" json:"max_pages"`
	
	// MaxItems limits the total number of items to scrape
	MaxItems int `yaml:"max_items" json:"max_items"`
	
	// MaxDuration limits the total time for pagination
	MaxDuration time.Duration `yaml:"max_duration" json:"max_duration"`
	
	// StopOnEmpty stops pagination when an empty page is found
	StopOnEmpty bool `yaml:"stop_on_empty" json:"stop_on_empty"`
	
	// StopOnError stops pagination on first error
	StopOnError bool `yaml:"stop_on_error" json:"stop_on_error"`
	
	// RequestsPerSecond for rate limiting
	RequestsPerSecond float64 `yaml:"requests_per_second" json:"requests_per_second"`
	
	// Selectors for pagination elements
	NextPageSelector   string `yaml:"next_page_selector" json:"next_page_selector"`
	PageNumberSelector string `yaml:"page_number_selector" json:"page_number_selector"`
	LoadMoreSelector   string `yaml:"load_more_selector" json:"load_more_selector"`
	
	// ItemSelector for extracting items on each page
	ItemSelector string `yaml:"item_selector" json:"item_selector"`
	
	// ItemContainer for infinite scroll scenarios
	ItemContainer string `yaml:"item_container" json:"item_container"`
	
	// ItemFields defines what to extract from each item
	ItemFields map[string]string `yaml:"item_fields" json:"item_fields"`
	
	// PageParameter for URL-based pagination
	PageParameter string `yaml:"page_parameter" json:"page_parameter"`
	
	// PageSize for offset-based pagination
	PageSize int `yaml:"page_size" json:"page_size"`
}

// OutputConfig defines output settings.
type OutputConfig struct {
	// Format of the output (json, csv, etc.)
	Format string `yaml:"format" json:"format"`
	
	// Path where to save the output
	Path string `yaml:"path" json:"path"`
	
	// Compression settings
	Compression string `yaml:"compression,omitempty" json:"compression,omitempty"`
	
	// Encoding for text files
	Encoding string `yaml:"encoding" json:"encoding"`
}

// RequestConfig defines HTTP request settings.
type RequestConfig struct {
	// Timeout for requests
	Timeout int `yaml:"timeout" json:"timeout"`
	
	// Retry configuration
	Retry RetryConfig `yaml:"retry" json:"retry"`
	
	// RateLimit settings
	RateLimit RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
}

// RetryConfig defines retry behavior.
type RetryConfig struct {
	// Attempts is the maximum number of retry attempts
	Attempts int `yaml:"attempts" json:"attempts"`
	
	// Delay between retries in seconds
	Delay int `yaml:"delay" json:"delay"`
	
	// Backoff strategy (linear, exponential)
	Backoff string `yaml:"backoff" json:"backoff"`
}

// RateLimitConfig defines rate limiting settings.
type RateLimitConfig struct {
	// RequestsPerSecond limits the request rate
	RequestsPerSecond float64 `yaml:"requests_per_second" json:"requests_per_second"`
	
	// Burst allows temporary exceeding of the rate
	Burst int `yaml:"burst" json:"burst"`
}

// AntiDetectConfig defines anti-detection settings.
type AntiDetectConfig struct {
	// Enabled turns on anti-detection measures
	Enabled bool `yaml:"enabled" json:"enabled"`
	
	// Strategies to use
	Strategies []string `yaml:"strategies" json:"strategies"`
	
	// BrowserOptions for browser-based scraping
	BrowserOptions BrowserConfig `yaml:"browser_options,omitempty" json:"browser_options,omitempty"`
}

// BrowserConfig defines browser automation settings.
type BrowserConfig struct {
	// Headless mode
	Headless bool `yaml:"headless" json:"headless"`
	
	// WindowSize for the browser window
	WindowSize string `yaml:"window_size" json:"window_size"`
	
	// UserAgent to use
	UserAgent string `yaml:"user_agent" json:"user_agent"`
	
	// DisableImages to speed up loading
	DisableImages bool `yaml:"disable_images" json:"disable_images"`
	
	// BlockAds to block advertisements
	BlockAds bool `yaml:"block_ads" json:"block_ads"`
}
