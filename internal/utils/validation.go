// Package utils provides common validation utilities and helpers
// for the DataScrapexter platform.
package utils

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"
)

// Regex patterns for validation - initialized once for thread safety and performance
var (
	// CSS selector validation patterns
	elementSelectorPattern   *regexp.Regexp
	classSelectorPattern     *regexp.Regexp
	idSelectorPattern        *regexp.Regexp
	universalSelectorPattern *regexp.Regexp
	attributeSelectorPattern *regexp.Regexp
	pseudoClassPattern       *regexp.Regexp
	pseudoElementPattern     *regexp.Regexp
	complexSelectorPattern   *regexp.Regexp
	combinatorPattern        *regexp.Regexp
	compoundSelectorPattern  *regexp.Regexp
	normalizeSpacePattern    *regexp.Regexp

	// Security validation patterns
	javascriptProtocolPattern *regexp.Regexp
	cssExpressionPattern      *regexp.Regexp
	javascriptURLPattern      *regexp.Regexp
	importStatementPattern    *regexp.Regexp

	// CSS combinator pattern
	cssCombinatorPattern *regexp.Regexp

	// Field name sanitization pattern
	fieldNameSanitizePattern *regexp.Regexp

	// Sync.Once for thread-safe initialization
	regexInitOnce sync.Once
)

// initRegexPatterns initializes all regex patterns in a thread-safe manner
func initRegexPatterns() {
	regexInitOnce.Do(func() {
		// CSS selector validation patterns
		elementSelectorPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)
		classSelectorPattern = regexp.MustCompile(`^\.[a-zA-Z_-][a-zA-Z0-9_-]*$`)
		idSelectorPattern = regexp.MustCompile(`^#[a-zA-Z_-][a-zA-Z0-9_-]*$`)
		universalSelectorPattern = regexp.MustCompile(`^\*$`)
		attributeSelectorPattern = regexp.MustCompile(`^\[[a-zA-Z][a-zA-Z0-9-]*(?:[~|^$*]?=["']?[^"'\]]*["']?)?\]$`)
		pseudoClassPattern = regexp.MustCompile(`^:[a-zA-Z-]+(?:\([^)]*\))?$`)
		pseudoElementPattern = regexp.MustCompile(`^::[a-zA-Z-]{2,}$`)
		complexSelectorPattern = regexp.MustCompile(`^[a-zA-Z0-9\s\[\].:_#>+~()"'=-]+$`)
		combinatorPattern = regexp.MustCompile(`\s*[>+~]\s*`)
		normalizeSpacePattern = regexp.MustCompile(`\s+`)

		// Compound selector pattern components
		elementSelectorComponent := `(?:[a-zA-Z][a-zA-Z0-9-]*|\*)?`
		classSelectorComponent := `(?:\.[a-zA-Z_-][a-zA-Z0-9_-]*)*`
		idSelectorComponent := `(?:#[a-zA-Z_-][a-zA-Z0-9_-]*)?`
		attributeSelectorComponent := `(?:\[[^\]]+\])*`
		pseudoClassComponent := `(?:\:[a-zA-Z-]+(?:\([^)]*\))?)*`
		pseudoElementComponent := `(?:\:\:[a-zA-Z-]+)*`
		
		compoundSelectorPattern = regexp.MustCompile(
			`^` +
				elementSelectorComponent +
				classSelectorComponent +
				idSelectorComponent +
				attributeSelectorComponent +
				pseudoClassComponent +
				pseudoElementComponent +
			`$`)

		// Security validation patterns
		javascriptProtocolPattern = regexp.MustCompile(`javascript:`)
		cssExpressionPattern = regexp.MustCompile(`expression\s*\(`)
		javascriptURLPattern = regexp.MustCompile(`\burl\s*\(\s*["']?javascript:`)
		importStatementPattern = regexp.MustCompile(`\bimport\b`)

		// CSS combinator pattern
		cssCombinatorPattern = regexp.MustCompile(`[>+~]\s*[a-zA-Z0-9\[\].:_#-]`)

		// Field name sanitization pattern
		fieldNameSanitizePattern = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	})
}

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

	// Check length constraints using optimized character counting
	charCount := countCharsOptimized(str)
	if sv.MinLength > 0 && charCount < sv.MinLength {
		return &ValidationError{
			Message: fmt.Sprintf("must be at least %d characters long", sv.MinLength),
			Code:    "MIN_LENGTH",
		}
	}

	if sv.MaxLength > 0 && charCount > sv.MaxLength {
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
	// Ensure regex patterns are initialized
	initRegexPatterns()
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
	// Ensure regex patterns are initialized
	initRegexPatterns()
	// Check for valid CSS combinators: >, +, ~, space
	return cssCombinatorPattern.MatchString(selector)
}

// isValidSelectorPattern performs comprehensive CSS selector pattern validation
func isValidSelectorPattern(selector string) bool {
	// Ensure regex patterns are initialized
	initRegexPatterns()
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
	// Ensure regex patterns are initialized
	initRegexPatterns()
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
	// Ensure regex patterns are initialized
	initRegexPatterns()
	if selector == "" || selector == "*" {
		return true
	}

	// Use pre-compiled pattern for compound selectors
	return compoundSelectorPattern.MatchString(selector)
}

// ValidateStruct validates a struct using field tags and optional custom validators
// Supports comprehensive struct tag validation with rules like:
//   `validate:"required,min=3,max=50,email"`
//   `validate:"required,url"`
//   `validate:"numeric,min=0,max=100"`
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

	// Validate each field
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Name
		fieldValue := val.Field(i)
		
		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// First, check custom validators
		if validator, ok := validators[fieldName]; ok {
			fieldError := validator.Validate(fieldValue.Interface())
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

		// Then, check struct tags
		validateTag := field.Tag.Get("validate")
		if validateTag != "" {
			tagErrors := validateFieldByTags(fieldName, fieldValue, validateTag)
			if len(tagErrors) > 0 {
				result.Valid = false
				result.Errors = append(result.Errors, tagErrors...)
			}
		}

		// Recursively validate nested structs
		if fieldValue.Kind() == reflect.Struct {
			nestedResult := ValidateStruct(fieldValue.Interface(), nil)
			if !nestedResult.Valid {
				result.Valid = false
				// Prefix nested field errors with parent field name
				for _, nestedError := range nestedResult.Errors {
					nestedError.Field = fieldName + "." + nestedError.Field
					result.Errors = append(result.Errors, nestedError)
				}
			}
		} else if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() && fieldValue.Elem().Kind() == reflect.Struct {
			// Handle pointer to struct
			nestedResult := ValidateStruct(fieldValue.Interface(), nil)
			if !nestedResult.Valid {
				result.Valid = false
				for _, nestedError := range nestedResult.Errors {
					nestedError.Field = fieldName + "." + nestedError.Field
					result.Errors = append(result.Errors, nestedError)
				}
			}
		}
	}
	return result
}

// validateFieldByTags validates a field using struct tag rules
func validateFieldByTags(fieldName string, fieldValue reflect.Value, tag string) []ValidationError {
	var errors []ValidationError
	
	// Parse validation rules from tag
	rules := parseValidationTag(tag)
	
	for _, rule := range rules {
		err := applyValidationRule(fieldName, fieldValue, rule)
		if err != nil {
			errors = append(errors, *err)
		}
	}
	
	return errors
}

// ValidationRule represents a parsed validation rule from struct tag
type ValidationRule struct {
	Name      string
	Parameter string
}

// parseValidationTag parses a validation tag into individual rules
// Example: "required,min=3,max=50,email" -> [{"required", ""}, {"min", "3"}, {"max", "50"}, {"email", ""}]
func parseValidationTag(tag string) []ValidationRule {
	var rules []ValidationRule
	
	if tag == "" {
		return rules
	}
	
	// Split by comma and trim spaces
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Check if rule has parameter (e.g., "min=3")
		if equalPos := strings.Index(part, "="); equalPos != -1 {
			rules = append(rules, ValidationRule{
				Name:      strings.TrimSpace(part[:equalPos]),
				Parameter: strings.TrimSpace(part[equalPos+1:]),
			})
		} else {
			rules = append(rules, ValidationRule{
				Name:      part,
				Parameter: "",
			})
		}
	}
	
	return rules
}

// applyValidationRule applies a single validation rule to a field
func applyValidationRule(fieldName string, fieldValue reflect.Value, rule ValidationRule) *ValidationError {
	// Get the actual value to validate
	value := fieldValue.Interface()
	
	switch rule.Name {
	case "required":
		if isEmpty(fieldValue) {
			return &ValidationError{
				Field:   fieldName,
				Message: "field is required",
				Code:    "REQUIRED",
				Value:   fmt.Sprintf("%v", value),
			}
		}
		
	case "email":
		if str, ok := value.(string); ok && str != "" {
			if !IsValidEmail(str) {
				return &ValidationError{
					Field:   fieldName,
					Message: "invalid email format",
					Code:    "INVALID_EMAIL",
					Value:   str,
				}
			}
		}
		
	case "url":
		if str, ok := value.(string); ok && str != "" {
			if !IsValidURL(str) {
				return &ValidationError{
					Field:   fieldName,
					Message: "invalid URL format",
					Code:    "INVALID_URL",
					Value:   str,
				}
			}
		}
		
	case "numeric":
		if !isNumeric(fieldValue) {
			return &ValidationError{
				Field:   fieldName,
				Message: "value must be numeric",
				Code:    "INVALID_NUMERIC",
				Value:   fmt.Sprintf("%v", value),
			}
		}
		
	case "min":
		if rule.Parameter != "" {
			err := validateMinValue(fieldName, fieldValue, rule.Parameter)
			if err != nil {
				return err
			}
		}
		
	case "max":
		if rule.Parameter != "" {
			err := validateMaxValue(fieldName, fieldValue, rule.Parameter)
			if err != nil {
				return err
			}
		}
		
	case "len":
		if rule.Parameter != "" {
			err := validateExactLength(fieldName, fieldValue, rule.Parameter)
			if err != nil {
				return err
			}
		}
		
	case "alpha":
		if str, ok := value.(string); ok && str != "" {
			if !IsAlpha(str) {
				return &ValidationError{
					Field:   fieldName,
					Message: "value must contain only alphabetic characters",
					Code:    "INVALID_ALPHA",
					Value:   str,
				}
			}
		}
		
	case "alphanumeric":
		if str, ok := value.(string); ok && str != "" {
			if !IsAlphaNumeric(str) {
				return &ValidationError{
					Field:   fieldName,
					Message: "value must contain only alphanumeric characters",
					Code:    "INVALID_ALPHANUMERIC",
					Value:   str,
				}
			}
		}
	}
	
	return nil
}

// Helper functions for validation rules

// isEmpty checks if a value is considered empty
func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0 || strings.TrimSpace(v.String()) == ""
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Invalid:
		return true
	default:
		// For other types, use zero value check
		zero := reflect.Zero(v.Type())
		return reflect.DeepEqual(v.Interface(), zero.Interface())
	}
}

// isNumeric checks if a value is numeric
func isNumeric(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	case reflect.String:
		str := v.String()
		if str == "" {
			return true // Empty string is considered valid for optional numeric fields
		}
		// Try parsing as float64 to cover both int and float
		_, err := regexp.MatchString(`^-?\d*\.?\d+$`, str)
		return err == nil
	default:
		return false
	}
}

// validateMinValue validates minimum value/length constraints
func validateMinValue(fieldName string, fieldValue reflect.Value, param string) *ValidationError {
	// Implementation would depend on field type (string length, numeric value, etc.)
	// This is a simplified version
	if str, ok := fieldValue.Interface().(string); ok {
		var minLen int
		if _, err := fmt.Sscanf(param, "%d", &minLen); err == nil {
			if utf8.RuneCountInString(str) < minLen {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("minimum length is %d characters", minLen),
					Code:    "MIN_LENGTH",
					Value:   str,
				}
			}
		}
	}
	return nil
}

// validateMaxValue validates maximum value/length constraints
func validateMaxValue(fieldName string, fieldValue reflect.Value, param string) *ValidationError {
	if str, ok := fieldValue.Interface().(string); ok {
		var maxLen int
		if _, err := fmt.Sscanf(param, "%d", &maxLen); err == nil {
			if utf8.RuneCountInString(str) > maxLen {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("maximum length is %d characters", maxLen),
					Code:    "MAX_LENGTH",
					Value:   str,
				}
			}
		}
	}
	return nil
}

// validateExactLength validates exact length constraints
func validateExactLength(fieldName string, fieldValue reflect.Value, param string) *ValidationError {
	if str, ok := fieldValue.Interface().(string); ok {
		var exactLen int
		if _, err := fmt.Sscanf(param, "%d", &exactLen); err == nil {
			if utf8.RuneCountInString(str) != exactLen {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("length must be exactly %d characters", exactLen),
					Code:    "EXACT_LENGTH",
					Value:   str,
				}
			}
		}
	}
	return nil
}

// Common validation functions

// IsValidEmail validates email addresses using regex
func IsValidEmail(email string) bool {
	// RFC 5322 compliant email regex (simplified version)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsAlpha checks if string contains only alphabetic characters
func IsAlpha(str string) bool {
	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	return alphaRegex.MatchString(str)
}

// IsAlphaNumeric checks if string contains only alphanumeric characters
func IsAlphaNumeric(str string) bool {
	alphaNumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return alphaNumericRegex.MatchString(str)
}


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
	// Ensure regex patterns are initialized
	initRegexPatterns()
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

// ValidateConfigIntegrity performs cross-field validation on configuration objects.
// This function validates the internal consistency and completeness of configuration data.
func ValidateConfigIntegrity(config interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]ValidationError, 0),
	}

	if config == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "config",
			Message: "Configuration cannot be nil",
		})
		return result
	}

	// Use reflection to perform generic validation based on the config type
	v := reflect.ValueOf(config)
	
	// Handle pointer types
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "config",
				Message: "Configuration pointer cannot be nil",
			})
			return result
		}
		v = v.Elem()
	}

	// If the config has its own Validate method, use it
	if validateMethod := v.MethodByName("Validate"); validateMethod.IsValid() {
		// Call the Validate() method if it exists
		if validateMethod.Type().NumIn() == 0 && validateMethod.Type().NumOut() == 1 {
			returnValues := validateMethod.Call([]reflect.Value{})
			if len(returnValues) > 0 {
				if err, ok := returnValues[0].Interface().(error); ok && err != nil {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Field:   "config",
						Message: fmt.Sprintf("Configuration validation failed: %v", err),
					})
				}
			}
		}
	}

	// Perform additional cross-field validation based on struct tags and types
	if v.Kind() == reflect.Struct {
		result = validateStructIntegrity(v, result)
	}

	return result
}

// validateStructIntegrity performs detailed struct validation
func validateStructIntegrity(v reflect.Value, result *ValidationResult) *ValidationResult {
	structType := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := structType.Field(i)
		
		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}
		
		// Check for required fields that are empty
		if tag := fieldType.Tag.Get("validate"); tag != "" {
			if strings.Contains(tag, "required") {
				if isEmptyValue(field) {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Field:   fieldType.Name,
						Message: "Required field is empty",
					})
				}
			}
		}
		
		// Recursively validate nested structs
		if field.Kind() == reflect.Struct {
			result = validateStructIntegrity(field, result)
		} else if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if !field.IsNil() {
				result = validateStructIntegrity(field.Elem(), result)
			}
		}
	}
	
	return result
}

// isEmptyValue checks if a reflect.Value represents an empty value
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return v.Len() == 0
	}
	return false
}

// countCharsOptimized counts characters in a string with fast path for ASCII-only strings.
// 
// TODO: PERFORMANCE BENCHMARKING NEEDED
// The current manual byte-by-byte scan may be slower than utf8.RuneCountInString 
// for many real-world strings. Consider benchmarking this optimization against:
// 1. utf8.RuneCountInString(s) directly (standard library is highly optimized)
// 2. strings.ContainsAny(s, "\u0080-\uffff") for non-ASCII detection
// 3. utf8.ValidString(s) + range over runes for mixed approach
//
// Benchmark scenarios should include:
// - Pure ASCII strings (current fast path should win)
// - Mixed ASCII/Unicode strings (may be slower due to double processing)  
// - Pure Unicode strings (should be similar to utf8.RuneCountInString)
// - Very long strings (cache effects matter)
// - Very short strings (overhead of optimization may not be worth it)
func countCharsOptimized(s string) int {
	// Fast path: check if string is valid UTF-8 and all characters are ASCII
	// For ASCII-only strings, byte length equals character count
	if utf8.ValidString(s) {
		// Check if all bytes are ASCII (< 128)
		for i := 0; i < len(s); i++ {
			if s[i] >= 128 {
				// Non-ASCII character found, use accurate UTF-8 rune counting
				return utf8.RuneCountInString(s)
			}
		}
		// All characters are ASCII: byte length equals character count
		return len(s)
	}
	
	// Invalid UTF-8: fall back to accurate UTF-8 rune counting for best effort
	return utf8.RuneCountInString(s)
}

// countCharsStandard provides a direct implementation using the standard library
// This can be used for benchmarking comparison against countCharsOptimized
func countCharsStandard(s string) int {
	return utf8.RuneCountInString(s)
}
