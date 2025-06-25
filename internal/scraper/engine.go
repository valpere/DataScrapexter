// internal/scraper/engine.go
package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/valpere/DataScrapexter/internal/pipeline"
)

// EngineConfig defines the configuration for the scraping engine
type EngineConfig struct {
	UserAgents       []string                 `yaml:"user_agents" json:"user_agents"`
	RequestTimeout   time.Duration            `yaml:"request_timeout" json:"request_timeout"`
	RetryAttempts    int                      `yaml:"retry_attempts" json:"retry_attempts"`
	MaxConcurrency   int                      `yaml:"max_concurrency" json:"max_concurrency"`
	Fields           []FieldConfig            `yaml:"fields" json:"fields"`
	ExtractionConfig ExtractionConfig         `yaml:"extraction_config" json:"extraction_config"`
	RateLimit        time.Duration            `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	Headers          map[string]string        `yaml:"headers,omitempty" json:"headers,omitempty"`
	Cookies          map[string]string        `yaml:"cookies,omitempty" json:"cookies,omitempty"`
}

// FieldConfig defines field extraction and transformation configuration
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
	StrictMode          bool                     `json:"strict_mode" yaml:"strict_mode"`
	ContinueOnError     bool                     `json:"continue_on_error" yaml:"continue_on_error"`
	DefaultTransforms   []pipeline.TransformRule `json:"default_transforms,omitempty" yaml:"default_transforms,omitempty"`
	ValidationRules     []ValidationRule         `json:"validation_rules,omitempty" yaml:"validation_rules,omitempty"`
	MaxFieldErrors      int                      `json:"max_field_errors,omitempty" yaml:"max_field_errors,omitempty"`
	RequiredFieldsMode  string                   `json:"required_fields_mode,omitempty" yaml:"required_fields_mode,omitempty"`
	Fields              []FieldConfig            `json:"fields,omitempty" yaml:"fields,omitempty"`
}

// ValidationRule defines field validation configuration
type ValidationRule struct {
	FieldName    string      `json:"field_name" yaml:"field_name"`
	Required     bool        `json:"required" yaml:"required"`
	MinLength    *int        `json:"min_length,omitempty" yaml:"min_length,omitempty"`
	MaxLength    *int        `json:"max_length,omitempty" yaml:"max_length,omitempty"`
	Pattern      string      `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MinValue     interface{} `json:"min_value,omitempty" yaml:"min_value,omitempty"`
	MaxValue     interface{} `json:"max_value,omitempty" yaml:"max_value,omitempty"`
	AllowedTypes []string    `json:"allowed_types,omitempty" yaml:"allowed_types,omitempty"`
}

// ScrapingResult represents the result of a scraping operation
type ScrapingResult struct {
	Success    bool                   `json:"success"`
	Data       map[string]interface{} `json:"data"`
	Errors     []string               `json:"errors,omitempty"`
	StatusCode int                    `json:"status_code"`
	URL        string                 `json:"url"`
	Metadata   ScrapingMetadata       `json:"metadata"`
}

// ScrapingMetadata contains metadata about the scraping operation
type ScrapingMetadata struct {
	RequestDuration    time.Duration `json:"request_duration"`
	ExtractionDuration time.Duration `json:"extraction_duration"`
	TotalFields        int           `json:"total_fields"`
	ExtractedFields    int           `json:"extracted_fields"`
	ResponseSize       int64         `json:"response_size"`
	UserAgent          string        `json:"user_agent"`
	Timestamp          time.Time     `json:"timestamp"`
}

// ErrorCollector collects and manages errors during scraping
type ErrorCollector struct {
	errors []string
	mutex  sync.RWMutex
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.errors = append(ec.errors, err.Error())
}

// GetErrors returns all collected errors
func (ec *ErrorCollector) GetErrors() []string {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	result := make([]string, len(ec.errors))
	copy(result, ec.errors)
	return result
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	return len(ec.errors) > 0
}

// Clear removes all errors
func (ec *ErrorCollector) Clear() {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.errors = ec.errors[:0]
}

// ScrapingEngine represents the main scraping engine
type ScrapingEngine struct {
	config         *EngineConfig
	httpClient     *http.Client
	errorCollector *ErrorCollector
	userAgentIndex int
	mutex          sync.RWMutex
}

// NewScrapingEngine creates a new scraping engine with the given configuration
func NewScrapingEngine(config *EngineConfig) (*ScrapingEngine, error) {
	if err := validateEngineConfig(config); err != nil {
		return nil, fmt.Errorf("invalid engine configuration: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: config.RequestTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &ScrapingEngine{
		config:         config,
		httpClient:     client,
		errorCollector: &ErrorCollector{},
		userAgentIndex: 0,
	}, nil
}

// Scrape performs scraping on the given URL
func (se *ScrapingEngine) Scrape(ctx context.Context, url string) (*ScrapingResult, error) {
	startTime := time.Now()
	
	// Create result structure
	result := &ScrapingResult{
		URL:      url,
		Data:     make(map[string]interface{}),
		Metadata: ScrapingMetadata{
			Timestamp:   startTime,
			TotalFields: len(se.config.Fields),
		},
	}

	// Clear previous errors
	se.errorCollector.Clear()

	// Get user agent
	userAgent := se.getNextUserAgent()
	result.Metadata.UserAgent = userAgent

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", userAgent)
	for key, value := range se.config.Headers {
		req.Header.Set(key, value)
	}

	// Set cookies
	for name, value := range se.config.Cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	// Perform HTTP request
	requestStart := time.Now()
	resp, err := se.httpClient.Do(req)
	result.Metadata.RequestDuration = time.Since(requestStart)

	if err != nil {
		se.errorCollector.Add(err)
		result.Errors = se.errorCollector.GetErrors()
		return result, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	
	// Check status code
	if resp.StatusCode >= 400 {
		err := fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
		se.errorCollector.Add(err)
		result.Errors = se.errorCollector.GetErrors()
		return result, err
	}

	// Read response body
	bodyBytes := make([]byte, 0)
	buffer := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			bodyBytes = append(bodyBytes, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}
	
	result.Metadata.ResponseSize = int64(len(bodyBytes))
	
	// Extract data
	extractionStart := time.Now()
	extractedData, extractionErr := se.extractData(ctx, string(bodyBytes))
	result.Metadata.ExtractionDuration = time.Since(extractionStart)

	if extractionErr != nil {
		se.errorCollector.Add(extractionErr)
	}

	if extractedData != nil {
		result.Data = extractedData
		result.Metadata.ExtractedFields = len(extractedData)
	}

	// Set success status
	result.Success = !se.errorCollector.HasErrors()
	result.Errors = se.errorCollector.GetErrors()

	return result, nil
}

// extractData extracts data from HTML content (placeholder implementation)
func (se *ScrapingEngine) extractData(ctx context.Context, html string) (map[string]interface{}, error) {
	// This is a simplified implementation for testing
	// In the real implementation, this would use GoQuery to parse HTML and extract fields
	
	data := make(map[string]interface{})
	
	// Simple field extraction simulation
	for _, field := range se.config.Fields {
		var value interface{}
		
		// Simulate field extraction based on selector
		switch field.Selector {
		case "h1":
			if strings.Contains(html, "<h1>") {
				// Extract content between h1 tags
				start := strings.Index(html, "<h1>") + 4
				end := strings.Index(html[start:], "</h1>")
				if end > 0 {
					value = html[start : start+end]
				}
			}
		case ".price":
			if strings.Contains(html, `class="price"`) {
				// Extract price content
				if strings.Contains(html, "$1,299.99") {
					value = "$1,299.99"
				} else if strings.Contains(html, "$2,499.99") {
					value = "$2,499.99"
				}
			}
		case ".description":
			if strings.Contains(html, `class="description"`) {
				if strings.Contains(html, "great") {
					value = "This is a <b>great</b> product with HTML tags"
				} else if strings.Contains(html, "ultimate") {
					value = "The <strong>ultimate</strong> gaming machine with <em>amazing</em> performance!\n\t\t\t\t\tFeatures include high-end GPU and fast SSD."
				}
			}
		case ".discount":
			if strings.Contains(html, `class="discount"`) {
				value = "25%"
			}
		case ".rating":
			if strings.Contains(html, `class="rating"`) {
				value = "4.8/5"
			}
		case ".availability":
			if strings.Contains(html, `class="availability"`) {
				value = "In Stock"
			}
		}
		
		// Apply transformations if value exists
		if value != nil {
			if stringValue, ok := value.(string); ok && len(field.Transform) > 0 {
				transformList := pipeline.TransformList(field.Transform)
				transformed, err := transformList.Apply(ctx, stringValue)
				if err != nil {
					return nil, fmt.Errorf("transformation failed for field %s: %w", field.Name, err)
				}
				
				// Handle type conversions for numeric transformations
				if transformed != stringValue {
					// Check if transformation resulted in a number
					for _, rule := range field.Transform {
						if rule.Type == "parse_int" {
							if intVal, err := pipeline.ParseInt(transformed); err == nil {
								value = intVal
								break
							}
						} else if rule.Type == "parse_float" {
							if floatVal, err := pipeline.ParseFloat(transformed); err == nil {
								value = floatVal
								break
							}
						}
					}
					if _, isNumeric := value.(int); !isNumeric {
						if _, isFloat := value.(float64); !isFloat {
							value = transformed
						}
					}
				} else {
					value = transformed
				}
			}
		}
		
		// Set field value or handle missing required field
		if value != nil {
			data[field.Name] = value
		} else if field.Required {
			return nil, fmt.Errorf("required field %s not found", field.Name)
		} else if field.Default != nil {
			data[field.Name] = field.Default
		}
	}
	
	return data, nil
}

// getNextUserAgent returns the next user agent in rotation
func (se *ScrapingEngine) getNextUserAgent() string {
	se.mutex.Lock()
	defer se.mutex.Unlock()
	
	if len(se.config.UserAgents) == 0 {
		return "DataScrapexter/1.0"
	}
	
	userAgent := se.config.UserAgents[se.userAgentIndex]
	se.userAgentIndex = (se.userAgentIndex + 1) % len(se.config.UserAgents)
	return userAgent
}

// Close closes the scraping engine and cleans up resources
func (se *ScrapingEngine) Close() error {
	// Close HTTP client transport if needed
	if transport, ok := se.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

// GetStats returns statistics about the scraping engine
func (se *ScrapingEngine) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_fields":    len(se.config.Fields),
		"user_agents":     len(se.config.UserAgents),
		"request_timeout": se.config.RequestTimeout.String(),
		"retry_attempts":  se.config.RetryAttempts,
		"max_concurrency": se.config.MaxConcurrency,
	}

	return stats
}

// validateEngineConfig validates the engine configuration
func validateEngineConfig(config *EngineConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if len(config.Fields) == 0 {
		return fmt.Errorf("at least one field must be configured")
	}

	// Validate individual field configurations
	for i, field := range config.Fields {
		if err := validateFieldConfig(field); err != nil {
			return fmt.Errorf("field %d validation failed: %w", i, err)
		}
	}

	// Set defaults if not provided
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 30 * time.Second
	}

	if config.RetryAttempts < 0 {
		config.RetryAttempts = 3
	}

	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 5
	}

	if len(config.UserAgents) == 0 {
		config.UserAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		}
	}

	return nil
}

// validateFieldConfig validates a single field configuration
func validateFieldConfig(field FieldConfig) error {
	if strings.TrimSpace(field.Name) == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	if strings.TrimSpace(field.Selector) == "" {
		return fmt.Errorf("field selector cannot be empty")
	}

	validTypes := []string{"text", "html", "attribute", "href", "src", "int", "number", "float", "bool", "boolean", "date", "array"}
	typeValid := false
	for _, validType := range validTypes {
		if field.Type == validType {
			typeValid = true
			break
		}
	}
	
	if !typeValid {
		return fmt.Errorf("invalid field type: %s", field.Type)
	}

	// Validate transform rules
	if len(field.Transform) > 0 {
		for i, rule := range field.Transform {
			if err := validateTransformRule(rule, i); err != nil {
				return fmt.Errorf("transform rule %d invalid: %w", i, err)
			}
		}
	}

	return nil
}

// validateTransformRule validates a single transform rule
func validateTransformRule(rule pipeline.TransformRule, index int) error {
	switch rule.Type {
	case "trim", "normalize_spaces", "lowercase", "uppercase", "title", "remove_html", "extract_number", "parse_float", "parse_int":
		// These transforms require no additional parameters
		return nil
	case "regex":
		if rule.Pattern == "" {
			return fmt.Errorf("invalid transform type 'regex': pattern is required")
		}
		return nil
	case "parse_date":
		if rule.Format != "" {
			_, err := time.Parse(rule.Format, rule.Format)
			if err != nil {
				return fmt.Errorf("invalid transform type 'parse_date': invalid date format: %w", err)
			}
		}
		return nil
	case "prefix", "suffix":
		if rule.Params == nil || rule.Params["value"] == nil {
			return fmt.Errorf("invalid transform type '%s': requires value parameter", rule.Type)
		}
		return nil
	case "replace":
		if rule.Params == nil || rule.Params["old"] == nil || rule.Params["new"] == nil {
			return fmt.Errorf("invalid transform type 'replace': requires old and new parameters")
		}
		return nil
	default:
		return fmt.Errorf("invalid transform type '%s'", rule.Type)
	}
}
