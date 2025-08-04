// Package utils provides common validation utilities and helpers
// for the DataScrapexter platform.
package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a structured validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Error implements the error interface
func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
}

// ValidationResult contains the result of validation operations
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// AddError adds a validation error to the result
func (vr *ValidationResult) AddError(field, value, message, code string) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
		Code:    code,
	})
}

// HasErrors returns true if there are validation errors
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// FirstError returns the first validation error if any
func (vr *ValidationResult) FirstError() *ValidationError {
	if len(vr.Errors) > 0 {
		return &vr.Errors[0]
	}
	return nil
}

// Validator interface for creating custom validators
type Validator interface {
	Validate(value interface{}) *ValidationError
}

// StringValidator validates string fields
type StringValidator struct {
	MinLength    int
	MaxLength    int
	Required     bool
	Pattern      *regexp.Regexp
	AllowedValues []string
}

// Validate implements the Validator interface for strings
func (sv *StringValidator) Validate(value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		return &ValidationError{
			Message: "value must be a string",
			Code:    "INVALID_TYPE",
		}
	}

	// Check if required
	if sv.Required && strings.TrimSpace(str) == "" {
		return &ValidationError{
			Message: "field is required",
			Code:    "REQUIRED",
		}
	}

	// Skip other validations if empty and not required
	if !sv.Required && strings.TrimSpace(str) == "" {
		return nil
	}

	// Check length constraints
	if sv.MinLength > 0 && utf8.RuneCountInString(str) < sv.MinLength {
		return &ValidationError{
			Message: fmt.Sprintf("must be at least %d characters long", sv.MinLength),
			Code:    "MIN_LENGTH",
		}
	}

	if sv.MaxLength > 0 && utf8.RuneCountInString(str) > sv.MaxLength {
		return &ValidationError{
			Message: fmt.Sprintf("must not exceed %d characters", sv.MaxLength),
			Code:    "MAX_LENGTH",
		}
	}

	// Check pattern
	if sv.Pattern != nil && !sv.Pattern.MatchString(str) {
		return &ValidationError{
			Message: "does not match required pattern",
			Code:    "PATTERN_MISMATCH",
		}
	}

	// Check allowed values
	if len(sv.AllowedValues) > 0 {
		for _, allowed := range sv.AllowedValues {
			if str == allowed {
				return nil
			}
		}
		return &ValidationError{
			Message: fmt.Sprintf("must be one of: %s", strings.Join(sv.AllowedValues, ", ")),
			Code:    "INVALID_VALUE",
		}
	}

	return nil
}

// URLValidator validates URL fields
type URLValidator struct {
	Required      bool
	AllowedSchemes []string // e.g., ["http", "https"]
	AllowedHosts  []string // e.g., ["example.com", "*.example.com"]
}

// Validate implements the Validator interface for URLs
func (uv *URLValidator) Validate(value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		return &ValidationError{
			Message: "value must be a string",
			Code:    "INVALID_TYPE",
		}
	}

	// Check if required
	if uv.Required && strings.TrimSpace(str) == "" {
		return &ValidationError{
			Message: "URL is required",
			Code:    "REQUIRED",
		}
	}

	// Skip validation if empty and not required
	if !uv.Required && strings.TrimSpace(str) == "" {
		return nil
	}

	// Parse URL
	parsedURL, err := url.Parse(str)
	if err != nil {
		return &ValidationError{
			Message: fmt.Sprintf("invalid URL format: %v", err),
			Code:    "INVALID_FORMAT",
		}
	}

	// Check scheme
	if len(uv.AllowedSchemes) > 0 {
		schemeAllowed := false
		for _, allowed := range uv.AllowedSchemes {
			if parsedURL.Scheme == allowed {
				schemeAllowed = true
				break
			}
		}
		if !schemeAllowed {
			return &ValidationError{
				Message: fmt.Sprintf("scheme must be one of: %s", strings.Join(uv.AllowedSchemes, ", ")),
				Code:    "INVALID_SCHEME",
			}
		}
	}

	// Check host (basic implementation - could be extended for wildcard matching)
	if len(uv.AllowedHosts) > 0 {
		hostAllowed := false
		for _, allowed := range uv.AllowedHosts {
			if parsedURL.Host == allowed {
				hostAllowed = true
				break
			}
			// Basic wildcard support for subdomains
			if strings.HasPrefix(allowed, "*.") {
				domain := allowed[2:] // Remove "*."
				if strings.HasSuffix(parsedURL.Host, "."+domain) || parsedURL.Host == domain {
					hostAllowed = true
					break
				}
			}
		}
		if !hostAllowed {
			return &ValidationError{
				Message: fmt.Sprintf("host must be one of: %s", strings.Join(uv.AllowedHosts, ", ")),
				Code:    "INVALID_HOST",
			}
		}
	}

	return nil
}

// SelectorValidator validates CSS selector strings
type SelectorValidator struct {
	Required bool
}

// Validate implements the Validator interface for CSS selectors
func (sv *SelectorValidator) Validate(value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		return &ValidationError{
			Message: "selector must be a string",
			Code:    "INVALID_TYPE",
		}
	}

	// Check if required
	if sv.Required && strings.TrimSpace(str) == "" {
		return &ValidationError{
			Message: "selector is required",
			Code:    "REQUIRED",
		}
	}

	// Skip validation if empty and not required
	if !sv.Required && strings.TrimSpace(str) == "" {
		return nil
	}

	// Basic CSS selector validation
	// This is a simplified check - in production, you might want more sophisticated validation
	if strings.Contains(str, "<") || strings.Contains(str, ">") && !isValidCSSCombinator(str) {
		return &ValidationError{
			Message: "selector contains invalid characters",
			Code:    "INVALID_SELECTOR",
		}
	}

	// Check for common selector patterns
	if !isValidSelectorPattern(str) {
		return &ValidationError{
			Message: "selector does not match valid CSS selector patterns",
			Code:    "INVALID_PATTERN",
		}
	}

	return nil
}

// isValidCSSCombinator checks if the string contains valid CSS combinators
func isValidCSSCombinator(selector string) bool {
	// Check for valid CSS combinators: >, +, ~, space
	combinatorPattern := regexp.MustCompile(`[>+~]\s*[a-zA-Z0-9\[\].:_#-]`)
	return combinatorPattern.MatchString(selector)
}

// isValidSelectorPattern performs basic CSS selector pattern validation
func isValidSelectorPattern(selector string) bool {
	// This is a simplified CSS selector validation
	// Matches: element, .class, #id, [attribute], :pseudo, element.class, etc.
	pattern := regexp.MustCompile(`^[a-zA-Z0-9\s\[\].:_#>+~()"'=-]+$`)
	return pattern.MatchString(selector)
}

// ValidateStruct validates a struct using field tags or custom validators
// This is a basic implementation that can be extended
func ValidateStruct(v interface{}, validators map[string]Validator) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// This would typically use reflection to iterate over struct fields
	// For now, this is a placeholder for custom validation logic
	// In a full implementation, you would:
	// 1. Use reflection to get struct fields
	// 2. Check for validation tags
	// 3. Apply appropriate validators
	// 4. Collect all validation errors

	return result
}

// Common validation functions

// IsValidFieldType checks if a field type is valid
func IsValidFieldType(fieldType string) bool {
	validTypes := map[string]bool{
		"text":  true,
		"attr":  true,
		"html":  true,
		"array": true,
		"list":  true,
	}
	return validTypes[fieldType]
}

// IsValidOutputFormat checks if an output format is valid
func IsValidOutputFormat(format string) bool {
	validFormats := map[string]bool{
		"json":     true,
		"csv":      true,
		"excel":    true,
		"xml":      true,
		"yaml":     true,
		"database": true,
	}
	return validFormats[format]
}

// SanitizeFieldName ensures field names are safe for use in outputs
func SanitizeFieldName(name string) string {
	// Remove or replace problematic characters
	clean := regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(name, "_")
	
	// Ensure it doesn't start with a number
	if len(clean) > 0 && clean[0] >= '0' && clean[0] <= '9' {
		clean = "field_" + clean
	}
	
	// Ensure it's not empty
	if clean == "" {
		clean = "unnamed_field"
	}
	
	return clean
}

// ValidateConfigIntegrity performs cross-field validation
func ValidateConfigIntegrity(config interface{}) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// This would contain logic to validate relationships between fields
	// For example:
	// - If proxy is enabled, ensure proxy providers are configured
	// - If browser automation is enabled, ensure required browser settings
	// - If database output is configured, ensure connection details are valid

	return result
}
