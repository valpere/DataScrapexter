// internal/config/validation.go - Enhanced validation with detailed error messages
package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a detailed validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ValidationResult holds validation results
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []string          `json:"warnings"`
}

// Enhanced Validate method for ScraperConfig (existing signature preserved)
func (sc *ScraperConfig) Validate() error {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]string, 0),
	}
	
	// Validate basic fields
	sc.validateBasicFields(result)
	
	// Validate URL
	sc.validateURL(result)
	
	// Validate fields configuration
	sc.validateFields(result)
	
	// Validate output configuration
	sc.validateOutput(result)
	
	// Validate engine settings
	sc.validateEngineSettings(result)
	
	if len(result.Errors) > 0 {
		return sc.formatValidationError(result)
	}
	
	return nil
}

// validateBasicFields checks required basic fields
func (sc *ScraperConfig) validateBasicFields(result *ValidationResult) {
	if sc.Name == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Value:   "",
			Message: "Scraper name is required",
		})
	}
	
	if sc.BaseURL == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "base_url",
			Value:   "",
			Message: "Base URL is required",
		})
	}
	
	if len(sc.Fields) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "fields",
			Value:   "[]",
			Message: "At least one field must be configured",
		})
	}
}

// validateURL checks URL format and accessibility
func (sc *ScraperConfig) validateURL(result *ValidationResult) {
	if sc.BaseURL == "" {
		return
	}
	
	parsedURL, err := url.Parse(sc.BaseURL)
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "base_url",
			Value:   sc.BaseURL,
			Message: fmt.Sprintf("Invalid URL format: %s", err.Error()),
		})
		return
	}
	
	if parsedURL.Scheme == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "base_url",
			Value:   sc.BaseURL,
			Message: "URL must include protocol (http:// or https://)",
		})
	}
	
	if parsedURL.Host == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "base_url",
			Value:   sc.BaseURL,
			Message: "URL must include hostname",
		})
	}
	
	// Warn about HTTP vs HTTPS
	if parsedURL.Scheme == "http" {
		result.Warnings = append(result.Warnings, 
			"Using HTTP instead of HTTPS may cause security issues")
	}
}

// validateFields checks field configurations
func (sc *ScraperConfig) validateFields(result *ValidationResult) {
	fieldNames := make(map[string]bool)
	
	for i, field := range sc.Fields {
		fieldPrefix := fmt.Sprintf("fields[%d]", i)
		
		// Check required field properties
		if field.Name == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("%s.name", fieldPrefix),
				Value:   "",
				Message: "Field name is required",
			})
		}
		
		// Check for duplicate field names
		if fieldNames[field.Name] {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("%s.name", fieldPrefix),
				Value:   field.Name,
				Message: fmt.Sprintf("Duplicate field name: %s", field.Name),
			})
		}
		fieldNames[field.Name] = true
		
		// Validate selector
		if field.Selector == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("%s.selector", fieldPrefix),
				Value:   "",
				Message: "CSS selector is required",
			})
		} else {
			// Basic CSS selector validation
			if err := validateCSSSelector(field.Selector); err != nil {
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("%s.selector", fieldPrefix),
					Value:   field.Selector,
					Message: fmt.Sprintf("Invalid CSS selector: %s", err.Error()),
				})
			}
		}
		
		// Validate field type
		validTypes := []string{"text", "attr", "html", "array", "list", "int", "float", "bool"}
		if !contains(validTypes, field.Type) {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("%s.type", fieldPrefix),
				Value:   field.Type,
				Message: fmt.Sprintf("Invalid field type. Valid types: %s", strings.Join(validTypes, ", ")),
			})
		}
		
		// Validate attribute for attr type
		if field.Type == "attr" && field.Attribute == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("%s.attribute", fieldPrefix),
				Value:   "",
				Message: "Attribute name is required for 'attr' type fields",
			})
		}
		
		// Validate transforms if present
		sc.validateFieldTransforms(field, fieldPrefix, result)
	}
}

// validateFieldTransforms checks field transformation rules
func (sc *ScraperConfig) validateFieldTransforms(field FieldConfig, fieldPrefix string, result *ValidationResult) {
	for i, transform := range field.Transform {
		transformPrefix := fmt.Sprintf("%s.transform[%d]", fieldPrefix, i)
		
		if transform.Type == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("%s.type", transformPrefix),
				Value:   "",
				Message: "Transform type is required",
			})
			continue
		}
		
		// Validate regex transforms
		if transform.Type == "regex" {
			if transform.Pattern == "" {
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("%s.pattern", transformPrefix),
					Value:   "",
					Message: "Pattern is required for regex transforms",
				})
			} else {
				// Test regex pattern
				if _, err := regexp.Compile(transform.Pattern); err != nil {
					result.Errors = append(result.Errors, ValidationError{
						Field:   fmt.Sprintf("%s.pattern", transformPrefix),
						Value:   transform.Pattern,
						Message: fmt.Sprintf("Invalid regex pattern: %s", err.Error()),
					})
				}
			}
		}
	}
}

// validateOutput checks output configuration
func (sc *ScraperConfig) validateOutput(result *ValidationResult) {
	if sc.Output.Format == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "output.format",
			Value:   "",
			Message: "Output format is required",
		})
		return
	}
	
	validFormats := []string{"json", "csv", "yaml"}
	if !contains(validFormats, sc.Output.Format) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "output.format",
			Value:   sc.Output.Format,
			Message: fmt.Sprintf("Invalid output format. Valid formats: %s", strings.Join(validFormats, ", ")),
		})
	}
	
	if sc.Output.File == "" {
		result.Warnings = append(result.Warnings, 
			"No output file specified, results will be written to stdout")
	}
}

// validateEngineSettings checks engine configuration
func (sc *ScraperConfig) validateEngineSettings(result *ValidationResult) {
	// Validate RateLimit if provided
	if sc.RateLimit != "" {
		if duration, err := time.ParseDuration(sc.RateLimit); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "rate_limit",
				Value:   sc.RateLimit,
				Message: fmt.Sprintf("Invalid rate limit format: %s", err.Error()),
			})
		} else if duration < 0 {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "rate_limit",
				Value:   sc.RateLimit,
				Message: "Rate limit cannot be negative",
			})
		} else if duration < 500*time.Millisecond {
			result.Warnings = append(result.Warnings, 
				"Rate limit below 500ms may overwhelm target servers")
		}
	}
	
	// Validate Timeout if provided
	if sc.Timeout != "" {
		if duration, err := time.ParseDuration(sc.Timeout); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "timeout",
				Value:   sc.Timeout,
				Message: fmt.Sprintf("Invalid timeout format: %s", err.Error()),
			})
		} else if duration < 0 {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "timeout",
				Value:   sc.Timeout,
				Message: "Timeout cannot be negative",
			})
		} else if duration > 60*time.Second {
			result.Warnings = append(result.Warnings, 
				"Timeout above 60 seconds may cause unnecessary delays")
		}
	}
	
	// Validate Retries
	if sc.Retries < 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "retries",
			Value:   fmt.Sprintf("%d", sc.Retries),
			Message: "Retries cannot be negative",
		})
	}
	
	// Validate MaxRetries
	if sc.MaxRetries < 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "max_retries",
			Value:   fmt.Sprintf("%d", sc.MaxRetries),
			Message: "Max retries cannot be negative",
		})
	}
}

// validateCSSSelector performs basic CSS selector validation
func validateCSSSelector(selector string) error {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return fmt.Errorf("empty selector")
	}
	
	// Check for obviously invalid patterns
	invalidPatterns := []string{
		"[", "]", "(", ")", "{", "}", 
		"<<", ">>", "|||", "&&&",
	}
	
	for _, pattern := range invalidPatterns {
		if strings.Contains(selector, pattern) {
			return fmt.Errorf("invalid character sequence: %s", pattern)
		}
	}
	
	// Check for unclosed quotes
	singleQuotes := strings.Count(selector, "'")
	doubleQuotes := strings.Count(selector, "\"")
	
	if singleQuotes%2 != 0 {
		return fmt.Errorf("unclosed single quote")
	}
	
	if doubleQuotes%2 != 0 {
		return fmt.Errorf("unclosed double quote")
	}
	
	return nil
}

// formatValidationError creates a comprehensive error message
func (sc *ScraperConfig) formatValidationError(result *ValidationResult) error {
	var errorMsg strings.Builder
	
	errorMsg.WriteString("Configuration validation failed:\n")
	
	for i, err := range result.Errors {
		errorMsg.WriteString(fmt.Sprintf("  %d. %s", i+1, err.Message))
		if err.Field != "" {
			errorMsg.WriteString(fmt.Sprintf(" (field: %s)", err.Field))
		}
		if err.Value != "" {
			errorMsg.WriteString(fmt.Sprintf(" (value: %s)", err.Value))
		}
		errorMsg.WriteString("\n")
	}
	
	if len(result.Warnings) > 0 {
		errorMsg.WriteString("\nWarnings:\n")
		for i, warning := range result.Warnings {
			errorMsg.WriteString(fmt.Sprintf("  %d. %s\n", i+1, warning))
		}
	}
	
	return fmt.Errorf("%s", errorMsg.String())
}

// ValidateWithDetails provides detailed validation results
func (sc *ScraperConfig) ValidateWithDetails() *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]string, 0),
	}
	
	sc.validateBasicFields(result)
	sc.validateURL(result)
	sc.validateFields(result)
	sc.validateOutput(result)
	sc.validateEngineSettings(result)
	
	result.Valid = len(result.Errors) == 0
	return result
}

// GetValidationSuggestions provides actionable suggestions for fixing validation errors
func (sc *ScraperConfig) GetValidationSuggestions(result *ValidationResult) []string {
	suggestions := make([]string, 0)
	
	hasURLError := false
	hasSelectorError := false
	hasFieldError := false
	
	for _, err := range result.Errors {
		if strings.Contains(err.Field, "url") {
			hasURLError = true
		}
		if strings.Contains(err.Field, "selector") {
			hasSelectorError = true
		}
		if strings.Contains(err.Field, "fields") {
			hasFieldError = true
		}
	}
	
	if hasURLError {
		suggestions = append(suggestions,
			"Ensure URLs include protocol (http:// or https://)",
			"Verify domain names are correct",
			"Test URLs in a browser first")
	}
	
	if hasSelectorError {
		suggestions = append(suggestions,
			"Test CSS selectors using browser developer tools",
			"Use the browser's element inspector to generate selectors",
			"Start with simple selectors and make them more specific as needed")
	}
	
	if hasFieldError {
		suggestions = append(suggestions,
			"Ensure all field names are unique",
			"Check that required field properties are set",
			"Verify field types match expected data")
	}
	
	if len(suggestions) == 0 {
		suggestions = append(suggestions,
			"Review the configuration file for syntax errors",
			"Check YAML indentation and formatting",
			"Ensure all required fields are present")
	}
	
	return suggestions
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
