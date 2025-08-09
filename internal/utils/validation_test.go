package utils

import (
	"reflect"
	"testing"
)

// TestIsValidEmail tests email validation function
func TestIsValidEmail(t *testing.T) {
	testCases := []struct {
		email    string
		expected bool
	}{
		// Valid emails
		{"test@example.com", true},
		{"user.name@domain.co.uk", true},
		{"user+tag@example.org", true},
		{"123@example.com", true},
		{"user@localhost", true},
		
		// Invalid emails
		{"", false},
		{"invalid-email", false},
		{"@example.com", false},
		{"user@", false},
		{"user@@example.com", false},
		{"user name@example.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			result := IsValidEmail(tc.email)
			if result != tc.expected {
				t.Errorf("IsValidEmail(%q) = %v, want %v", tc.email, result, tc.expected)
			}
		})
	}
}

// TestIsAlpha tests alphabetic character validation
func TestIsAlpha(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		// Valid alphabetic strings
		{"Hello", true},
		{"WORLD", true},
		{"aBcDeF", true},
		
		// Invalid strings
		{"", false},
		{"Hello123", false},
		{"Hello World", false},
		{"Hello!", false},
		{"αβγ", false}, // Unicode letters not supported by current regex
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := IsAlpha(tc.input)
			if result != tc.expected {
				t.Errorf("IsAlpha(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestIsAlphaNumeric tests alphanumeric character validation
func TestIsAlphaNumeric(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		// Valid alphanumeric strings
		{"Hello123", true},
		{"ABC", true},
		{"123", true},
		{"a1b2c3", true},
		
		// Invalid strings
		{"", false},
		{"Hello World", false},
		{"Hello!", false},
		{"Hello-123", false},
		{"123.45", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := IsAlphaNumeric(tc.input)
			if result != tc.expected {
				t.Errorf("IsAlphaNumeric(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestIsValidFieldType tests field type validation
func TestIsValidFieldType(t *testing.T) {
	testCases := []struct {
		fieldType string
		expected  bool
	}{
		// Valid field types
		{"text", true},
		{"attr", true},
		{"html", true},
		{"array", true},
		{"list", true},
		
		// Invalid field types
		{"", false},
		{"invalid", false},
		{"TEXT", false}, // Case sensitive
		{"number", false},
	}

	for _, tc := range testCases {
		t.Run(tc.fieldType, func(t *testing.T) {
			result := IsValidFieldType(tc.fieldType)
			if result != tc.expected {
				t.Errorf("IsValidFieldType(%q) = %v, want %v", tc.fieldType, result, tc.expected)
			}
		})
	}
}

// TestIsValidOutputFormat tests output format validation
func TestIsValidOutputFormat(t *testing.T) {
	testCases := []struct {
		format   string
		expected bool
	}{
		// Valid formats
		{"json", true},
		{"csv", true},
		{"excel", true},
		{"xml", true},
		{"yaml", true},
		{"pdf", true},
		{"tsv", true},
		{"parquet", true},
		{"mongodb", true},
		{"mysql", true},
		{"postgresql", true},
		{"sqlite", true},
		{"database", true},
		
		// Invalid formats
		{"", false},
		{"txt", false},
		{"JSON", false}, // Case sensitive
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			result := IsValidOutputFormat(tc.format)
			if result != tc.expected {
				t.Errorf("IsValidOutputFormat(%q) = %v, want %v", tc.format, result, tc.expected)
			}
		})
	}
}

// TestSanitizeFieldName tests field name sanitization
func TestSanitizeFieldName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		// Normal cases
		{"valid_name", "valid_name"},
		{"ValidName", "ValidName"},
		
		// Cases requiring sanitization
		{"field-name", "field_name"},
		{"field name", "field_name"},
		{"field!@#name", "field___name"},
		{"123field", "field_123field"},
		{"", "unnamed_field"},
		{"___", "___"},
		{"field.name", "field_name"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := SanitizeFieldName(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizeFieldName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestStringValidator tests the StringValidator struct
func TestStringValidator(t *testing.T) {
	testCases := []struct {
		name      string
		validator StringValidator
		input     interface{}
		shouldErr bool
	}{
		{
			name:      "required field with valid string",
			validator: StringValidator{Required: true, MinLength: 3, MaxLength: 10},
			input:     "hello",
			shouldErr: false,
		},
		{
			name:      "required field with empty string",
			validator: StringValidator{Required: true},
			input:     "",
			shouldErr: true,
		},
		{
			name:      "non-required field with empty string",
			validator: StringValidator{Required: false},
			input:     "",
			shouldErr: false,
		},
		{
			name:      "string too short",
			validator: StringValidator{MinLength: 5},
			input:     "hi",
			shouldErr: true,
		},
		{
			name:      "string too long",
			validator: StringValidator{MaxLength: 3},
			input:     "hello",
			shouldErr: true,
		},
		{
			name:      "non-string input",
			validator: StringValidator{},
			input:     123,
			shouldErr: true,
		},
		{
			name:      "allowed values - valid",
			validator: StringValidator{AllowedValues: []string{"red", "green", "blue"}},
			input:     "red",
			shouldErr: false,
		},
		{
			name:      "allowed values - invalid",
			validator: StringValidator{AllowedValues: []string{"red", "green", "blue"}},
			input:     "yellow",
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.validator.Validate(tc.input)
			if tc.shouldErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestURLValidator tests the URLValidator struct
func TestURLValidator(t *testing.T) {
	testCases := []struct {
		name      string
		validator URLValidator
		input     interface{}
		shouldErr bool
	}{
		{
			name:      "valid HTTP URL",
			validator: URLValidator{Required: true, AllowedSchemes: []string{"http", "https"}},
			input:     "https://example.com",
			shouldErr: false,
		},
		{
			name:      "required URL is empty",
			validator: URLValidator{Required: true},
			input:     "",
			shouldErr: true,
		},
		{
			name:      "non-required URL is empty",
			validator: URLValidator{Required: false},
			input:     "",
			shouldErr: false,
		},
		{
			name:      "invalid URL format",
			validator: URLValidator{},
			input:     "not-a-url",
			shouldErr: true,
		},
		{
			name:      "disallowed scheme",
			validator: URLValidator{AllowedSchemes: []string{"https"}},
			input:     "http://example.com",
			shouldErr: true,
		},
		{
			name:      "allowed host",
			validator: URLValidator{AllowedHosts: []string{"example.com"}},
			input:     "https://example.com/path",
			shouldErr: false,
		},
		{
			name:      "disallowed host",
			validator: URLValidator{AllowedHosts: []string{"example.com"}},
			input:     "https://other.com/path",
			shouldErr: true,
		},
		{
			name:      "wildcard subdomain allowed",
			validator: URLValidator{AllowedHosts: []string{"*.example.com"}},
			input:     "https://sub.example.com/path",
			shouldErr: false,
		},
		{
			name:      "non-string input",
			validator: URLValidator{},
			input:     123,
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.validator.Validate(tc.input)
			if tc.shouldErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestSelectorValidator tests CSS selector validation
func TestSelectorValidator(t *testing.T) {
	testCases := []struct {
		name      string
		validator SelectorValidator
		input     interface{}
		shouldErr bool
	}{
		// Basic valid selectors
		{
			name:      "element selector",
			validator: SelectorValidator{Required: true},
			input:     "div",
			shouldErr: false,
		},
		{
			name:      "class selector",
			validator: SelectorValidator{},
			input:     ".class-name",
			shouldErr: false,
		},
		{
			name:      "id selector",
			validator: SelectorValidator{},
			input:     "#id-name",
			shouldErr: false,
		},
		{
			name:      "attribute selector",
			validator: SelectorValidator{},
			input:     "input[type='text']",
			shouldErr: false,
		},
		{
			name:      "pseudo-class selector",
			validator: SelectorValidator{},
			input:     ":hover",
			shouldErr: false,
		},
		{
			name:      "pseudo-element selector",
			validator: SelectorValidator{},
			input:     "::before",
			shouldErr: false,
		},
		{
			name:      "complex selector",
			validator: SelectorValidator{},
			input:     "div > p.content",
			shouldErr: false,
		},
		{
			name:      "multiple selectors",
			validator: SelectorValidator{},
			input:     "div, span, p",
			shouldErr: false,
		},
		
		// Invalid cases
		{
			name:      "required but empty",
			validator: SelectorValidator{Required: true},
			input:     "",
			shouldErr: true,
		},
		{
			name:      "non-string input",
			validator: SelectorValidator{},
			input:     123,
			shouldErr: true,
		},
		{
			name:      "invalid characters",
			validator: SelectorValidator{},
			input:     "div{color:red;}",
			shouldErr: true,
		},
		{
			name:      "HTML-like content",
			validator: SelectorValidator{},
			input:     "<script>alert('xss')</script>",
			shouldErr: true,
		},
		
		// Strict mode tests
		{
			name:      "strict mode - javascript protocol",
			validator: SelectorValidator{Strict: true},
			input:     "javascript:alert('xss')",
			shouldErr: true,
		},
		{
			name:      "strict mode - CSS expression",
			validator: SelectorValidator{Strict: true},
			input:     "expression(alert('xss'))",
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.validator.Validate(tc.input)
			if tc.shouldErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestValidateStruct tests struct validation
func TestValidateStruct(t *testing.T) {
	type TestStruct struct {
		Name     string `validate:"required,min=3,max=20"`
		Email    string `validate:"required,email"`
		Age      int    `validate:"numeric,min=0,max=120"`
		Website  string `validate:"url"`
		Optional string `validate:"max=50"`
	}
	
	testCases := []struct {
		name         string
		input        TestStruct
		expectValid  bool
		expectErrors int
	}{
		{
			name: "valid struct",
			input: TestStruct{
				Name:     "John Doe",
				Email:    "john@example.com",
				Age:      30,
				Website:  "https://example.com",
				Optional: "Some text",
			},
			expectValid:  true,
			expectErrors: 0,
		},
		{
			name: "multiple validation errors",
			input: TestStruct{
				Name:     "Jo", // Too short
				Email:    "invalid-email",
				Age:      -5, // Negative
				Website:  "not-a-url",
				Optional: "This is a very long string that exceeds the maximum allowed length for this field",
			},
			expectValid:  false,
			expectErrors: 4,
		},
		{
			name: "required field empty",
			input: TestStruct{
				Name:  "", // Required but empty
				Email: "john@example.com",
				Age:   30,
			},
			expectValid:  false,
			expectErrors: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateStruct(tc.input, nil)
			
			if result.Valid != tc.expectValid {
				t.Errorf("Expected Valid=%v, got %v", tc.expectValid, result.Valid)
			}
			
			if len(result.Errors) != tc.expectErrors {
				t.Errorf("Expected %d errors, got %d: %v", tc.expectErrors, len(result.Errors), result.Errors)
			}
		})
	}
}

// TestValidationResult tests ValidationResult helper methods
func TestValidationResult(t *testing.T) {
	result := &ValidationResult{Valid: true}
	
	// Test initial state
	if result.HasErrors() {
		t.Error("Expected no errors initially")
	}
	
	if result.FirstError() != nil {
		t.Error("Expected no first error initially")
	}
	
	// Add error
	result.AddError("test", "value", "test message", "TEST_CODE")
	
	if result.Valid {
		t.Error("Expected Valid to be false after adding error")
	}
	
	if !result.HasErrors() {
		t.Error("Expected HasErrors to return true")
	}
	
	firstError := result.FirstError()
	if firstError == nil {
		t.Fatal("Expected first error to be non-nil")
	}
	
	if firstError.Field != "test" {
		t.Errorf("Expected field 'test', got %q", firstError.Field)
	}
	
	if firstError.Code != "TEST_CODE" {
		t.Errorf("Expected code 'TEST_CODE', got %q", firstError.Code)
	}
}

// TestCountCharsOptimized tests character counting function
func TestCountCharsOptimized(t *testing.T) {
	testCases := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"A", 1},
		{"Hello", 5},
		{"Hello World", 11},
		{"世界", 2},
		{"Hello 世界", 8},
		{"🌍🧪🚀", 3},
		{"Mix: ABC 世界 🌍 123", 17},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := countCharsOptimized(tc.input)
			if result != tc.expected {
				t.Errorf("countCharsOptimized(%q) = %d, want %d", tc.input, result, tc.expected)
			}
			
			// Verify it matches standard library
			stdResult := countCharsStandard(tc.input)
			if result != stdResult {
				t.Errorf("Mismatch with standard: optimized=%d, standard=%d", result, stdResult)
			}
		})
	}
}

// ConfigWithValidate is a test config type with validation method
type ConfigWithValidate struct {
	Name string
}

// Validate implements validation for ConfigWithValidate
func (c ConfigWithValidate) Validate() error {
	if c.Name == "" {
		return ValidationError{Message: "Name is required"}
	}
	return nil
}

// TestValidateConfigIntegrity tests configuration integrity validation
func TestValidateConfigIntegrity(t *testing.T) {
	
	testCases := []struct {
		name        string
		config      interface{}
		expectValid bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectValid: false,
		},
		{
			name:        "valid struct",
			config:      struct{ Name string }{Name: "test"},
			expectValid: true,
		},
		{
			name:        "struct with validate method - valid",
			config:      ConfigWithValidate{Name: "test"},
			expectValid: true,
		},
		{
			name:        "struct with validate method - invalid",
			config:      ConfigWithValidate{Name: ""},
			expectValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateConfigIntegrity(tc.config)
			if result.Valid != tc.expectValid {
				t.Errorf("Expected Valid=%v, got %v. Errors: %v", tc.expectValid, result.Valid, result.Errors)
			}
		})
	}
}

// TestIsEmpty tests the isEmpty helper function
func TestIsEmpty(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"empty string", "", true},
		{"whitespace string", "   ", true},
		{"non-empty string", "hello", false},
		{"nil pointer", (*string)(nil), true},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"empty slice", []string{}, true},
		{"non-empty slice", []string{"item"}, false},
		{"empty map", map[string]string{}, true},
		{"non-empty map", map[string]string{"key": "value"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := reflect.ValueOf(tc.input)
			result := isEmpty(v)
			if result != tc.expected {
				t.Errorf("isEmpty(%v) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}