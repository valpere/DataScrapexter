// Package utils provides common validation utilities and helpers
// for the DataScrapexter platform.
package utils

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Pre-compiled regex patterns for better performance
var (
	// CSS selector validation patterns
	elementSelectorPattern   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)
	classSelectorPattern     = regexp.MustCompile(`^\.[a-zA-Z_-][a-zA-Z0-9_-]*$`)
	idSelectorPattern        = regexp.MustCompile(`^#[a-zA-Z_-][a-zA-Z0-9_-]*$`)
	universalSelectorPattern = regexp.MustCompile(`^\*$`)
	attributeSelectorPattern = regexp.MustCompile(`^\[[a-zA-Z][a-zA-Z0-9-]*(?:[~|^$*]?=["']?[^"'\]]*["']?)?\]$`)
	pseudoClassPattern       = regexp.MustCompile(`^:[a-zA-Z-]+(?:\([^)]*\))?$`)
	pseudoElementPattern     = regexp.MustCompile(`^::[a-zA-Z-]{2,}$`)
	complexSelectorPattern   = regexp.MustCompile(`^[a-zA-Z0-9\s\[\].:_#>+~()"'=-]+$`)
	combinatorPattern        = regexp.MustCompile(`\s*[>+~]\s*`)
	compoundSelectorPattern  = regexp.MustCompile(`^(?:[a-zA-Z][a-zA-Z0-9-]*|\*)?(?:\.[a-zA-Z_-][a-zA-Z0-9_-]*)*(?:#[a-zA-Z_-][a-zA-Z0-9_-]*)?(?:\[[^\]]+\])*(?::[a-zA-Z-]+(?:\([^)]*\))?)*(?:::[a-zA-Z-]+)*$`)
	normalizeSpacePattern    = regexp.MustCompile(`\s+`)

	// Security validation patterns
	javascriptProtocolPattern = regexp.MustCompile(`javascript:`)
	cssExpressionPattern      = regexp.MustCompile(`expression\s*\(`)
	javascriptURLPattern      = regexp.MustCompile(`\burl\s*\(\s*["']?javascript:`)
	importStatementPattern    = regexp.MustCompile(`\bimport\b`)

	// CSS combinator pattern
	cssCombinatorPattern = regexp.MustCompile(`[>+~]\s*[a-zA-Z0-9\[\].:_#-]`)

	// Field name sanitization pattern
	fieldNameSanitizePattern = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

// Validation constants for configurable limits
const (
	// MaxSelectorLength defines the maximum allowed length for CSS selectors
	MaxSelectorLength = 1000

	// MaxNestingDepth defines the maximum allowed nesting depth for CSS selectors
	MaxNestingDepth = 20
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

// ValidationResult represents the result of a validation operation
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
	MinLength     int
	MaxLength     int
	Required      bool
	Pattern       *regexp.Regexp
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
	Required       bool
	AllowedSchemes []string // e.g., ["http", "https"]
	AllowedHosts   []string // e.g., ["example.com", "*.example.com"]
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

// SelectorValidator validates CSS selector strings with comprehensive validation
type SelectorValidator struct {
	Required bool
	Strict   bool // Enable strict validation mode
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

	// Comprehensive CSS selector validation
	// Check for obviously invalid characters first
	if strings.ContainsAny(str, "@{};\\`") {
		return &ValidationError{
			Message: "selector contains invalid characters (@, {, }, ;, \\, `)",
			Code:    "INVALID_CHARACTERS",
		}
	}

	// Check for HTML tags (potential XSS attempt)
	if strings.Contains(str, "<") && !isValidCSSCombinator(str) {
		return &ValidationError{
			Message: "selector contains HTML-like content",
			Code:    "INVALID_HTML_CONTENT",
		}
	}

	// Comprehensive pattern validation
	if !isValidSelectorPattern(str) {
		return &ValidationError{
			Message: "selector does not match valid CSS selector syntax",
			Code:    "INVALID_SYNTAX",
		}
	}

	// Additional strict validation if enabled
	if sv.Strict {
		if err := sv.validateSelectorSafety(str); err != nil {
			return err
		}
	}

	return nil
}

// validateSelectorSafety performs additional safety checks for strict mode
func (sv *SelectorValidator) validateSelectorSafety(selector string) *ValidationError {
	// Check for potentially dangerous patterns using pre-compiled regex
	dangerousPatterns := []struct {
		pattern *regexp.Regexp
		message string
		code    string
	}{
		{
			javascriptProtocolPattern,
			"selector contains javascript: protocol",
			"DANGEROUS_PROTOCOL",
		},
		{
			cssExpressionPattern,
			"selector contains CSS expression",
			"CSS_EXPRESSION",
		},
		{
			javascriptURLPattern,
			"selector contains javascript URL",
			"JAVASCRIPT_URL",
		},
		{
			importStatementPattern,
			"selector contains import statement",
			"IMPORT_STATEMENT",
		},
	}

	for _, dangerous := range dangerousPatterns {
		if dangerous.pattern.MatchString(selector) {
			return &ValidationError{
				Message: dangerous.message,
				Code:    dangerous.code,
			}
		}
	}

	// Check selector length (reasonable limit)
	if len(selector) > MaxSelectorLength {
		return &ValidationError{
			Message: fmt.Sprintf("selector is too long (max %d characters)", MaxSelectorLength),
			Code:    "SELECTOR_TOO_LONG",
		}
	}

	// Check nesting depth (prevent deeply nested selectors)
	nestingDepth := strings.Count(selector, " ") + strings.Count(selector, ">") + strings.Count(selector, "+") + strings.Count(selector, "~")
	if nestingDepth > MaxNestingDepth {
		return &ValidationError{
			Message: fmt.Sprintf("selector has too many nested levels (max %d)", MaxNestingDepth),
			Code:    "EXCESSIVE_NESTING",
		}
	}

	return nil
}

// isValidCSSCombinator checks if the string contains valid CSS combinators
func isValidCSSCombinator(selector string) bool {
	// Check for valid CSS combinators: >, +, ~, space
	return cssCombinatorPattern.MatchString(selector)
}

// isValidSelectorPattern performs comprehensive CSS selector pattern validation
func isValidSelectorPattern(selector string) bool {
	// Trim whitespace and check for empty selector
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return false
	}

	// Split by comma to handle multiple selectors
	selectors := strings.Split(trimmed, ",")
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		if !isValidSingleSelector(sel) {
			return false
		}
	}
	return true
}

// isValidSingleSelector validates a single CSS selector
func isValidSingleSelector(selector string) bool {
	if selector == "" {
		return false
	}

	// Check for invalid characters that shouldn't appear in CSS selectors
	if strings.ContainsAny(selector, "@{};\\`") {
		return false
	}

	// Use pre-compiled patterns for better performance
	patterns := []*regexp.Regexp{
		elementSelectorPattern,   // Element selectors: div, span, etc.
		classSelectorPattern,     // Class selectors: .class-name
		idSelectorPattern,        // ID selectors: #id-name
		universalSelectorPattern, // Universal selector: *
		attributeSelectorPattern, // Attribute selectors: [attr], [attr="value"], etc.
		pseudoClassPattern,       // Pseudo-class selectors: :hover, :nth-child(n), etc.
		pseudoElementPattern,     // Pseudo-element selectors: ::before, ::after
		complexSelectorPattern,   // Complex selectors with combinators and multiple parts
	}

	// Check if selector matches any valid pattern
	for _, pattern := range patterns {
		if pattern.MatchString(selector) {
			return isValidComplexSelector(selector)
		}
	}

	return false
}

// isValidComplexSelector validates complex selectors with combinators
func isValidComplexSelector(selector string) bool {
	// Remove extra spaces and normalize using pre-compiled pattern
	normalized := normalizeSpacePattern.ReplaceAllString(strings.TrimSpace(selector), " ")

	// Check for valid combinator patterns using pre-compiled pattern
	parts := combinatorPattern.Split(normalized, -1)

	// Validate each part of the complex selector
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false
		}

		// Each part should be a valid simple selector or compound selector
		if !isValidCompoundSelector(part) {
			return false
		}
	}

	return true
}

// isValidCompoundSelector validates compound selectors (element.class#id:pseudo)
func isValidCompoundSelector(selector string) bool {
	if selector == "" || selector == "*" {
		return true
	}

	// Use pre-compiled pattern for compound selectors
	return compoundSelectorPattern.MatchString(selector)
}

// ValidateStruct validates a struct using field tags or custom validators
// TODO: This is a placeholder for future struct validation implementation
// Use individual field validators for now (StringValidator, URLValidator, etc.)
func ValidateStruct(v interface{}, validators map[string]Validator) *ValidationResult {
	result := &ValidationResult{Valid: true}

	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	// If pointer, get the element
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	// Only validate structs
	if val.Kind() != reflect.Struct {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "input",
			Message: "ValidateStruct: input is not a struct",
			Code:    "invalid_type",
		})
		return result
	}

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Name
		fieldValue := val.Field(i).Interface()

		validator, ok := validators[fieldName]
		if ok {
			fieldError := validator.Validate(fieldValue)
			if fieldError != nil {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fieldName,
					Message: fieldError.Message,
					Code:    fieldError.Code,
					Value:   fieldError.Value,
				})
			}
		}
	}
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
	// Remove or replace problematic characters using pre-compiled pattern
	clean := fieldNameSanitizePattern.ReplaceAllString(name, "_")

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

// TODO: This is a placeholder for future cross-field validation implementation
// Use the configuration's own Validate() methods for now
func ValidateConfigIntegrity(config interface{}) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Note: This is a minimal implementation that always returns valid
	// For production use, implement specific cross-field validation logic:
	// - Check if proxy is enabled but no providers are configured
	// - Validate browser automation settings consistency
	// - Ensure database connection details are complete
	// - Verify output format compatibility with selected fields

	return result
}
