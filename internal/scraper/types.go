// internal/scraper/types.go
package scraper

import (
	"fmt"
	"time"

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
	TotalFields       int           `json:"total_fields"`
	ExtractedFields   int           `json:"extracted_fields"`
	FailedFields      int           `json:"failed_fields"`
	ProcessingTime    time.Duration `json:"processing_time"`
	RequiredFieldsOK  bool          `json:"required_fields_ok"`
	DocumentSize      int64         `json:"document_size"`
	ErrorCount        int           `json:"error_count"`
	WarningCount      int           `json:"warning_count"`
	Duration          time.Duration `json:"duration"`
	Timestamp         time.Time     `json:"timestamp"`
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
