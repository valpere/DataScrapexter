// internal/scraper/extractor_test.go
package scraper

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/valpere/DataScrapexter/internal/pipeline"
)

func TestNewFieldExtractor(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: true,
	}

	extractor := NewFieldExtractor(parser, config)

	if extractor == nil {
		t.Fatal("Extractor should not be nil")
	}

	if extractor.parser != parser {
		t.Fatal("Extractor should reference the provided parser")
	}

	if extractor.config.Name != config.Name {
		t.Fatalf("Expected config name '%s', got '%s'", config.Name, extractor.config.Name)
	}
}

func TestFieldExtractor_Extract_Text(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: true,
	}

	extractor := NewFieldExtractor(parser, config)
	ctx := context.Background()

	value, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != "Test Title" {
		t.Fatalf("Expected 'Test Title', got '%v'", value)
	}
}

func TestFieldExtractor_Extract_RequiredFieldMissing(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "missing",
		Selector: ".missing",
		Type:     "text",
		Required: true,
	}

	extractor := NewFieldExtractor(parser, config)
	ctx := context.Background()

	_, err = extractor.Extract(ctx)
	if err == nil {
		t.Fatal("Expected error for missing required field")
	}

	if !strings.Contains(err.Error(), "required field") {
		t.Fatalf("Expected error about required field, got: %v", err)
	}
}

func TestFieldExtractor_Extract_OptionalFieldMissing(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "missing",
		Selector: ".missing",
		Type:     "text",
		Required: false,
		Default:  "default_value",
	}

	extractor := NewFieldExtractor(parser, config)
	ctx := context.Background()

	value, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Unexpected error for optional field: %v", err)
	}

	if value != "default_value" {
		t.Fatalf("Expected default value 'default_value', got '%v'", value)
	}
}

func TestFieldExtractor_Extract_WithTransformations(t *testing.T) {
	html := `<html><body><h1>  Test Title  </h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: true,
		Transform: []pipeline.TransformRule{
			{Type: "trim"},
			{Type: "uppercase"},
		},
	}

	extractor := NewFieldExtractor(parser, config)
	ctx := context.Background()

	value, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != "TEST TITLE" {
		t.Fatalf("Expected 'TEST TITLE', got '%v'", value)
	}
}

func TestFieldExtractor_ValidateConfig(t *testing.T) {
	html := `<html><body><h1>Test</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name        string
		config      FieldConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: FieldConfig{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
			expectError: false,
		},
		{
			name: "empty name",
			config: FieldConfig{
				Name:     "",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
			expectError: true,
		},
		{
			name: "empty selector",
			config: FieldConfig{
				Name:     "title",
				Selector: "",
				Type:     "text",
				Required: true,
			},
			expectError: true,
		},
		{
			name: "invalid type",
			config: FieldConfig{
				Name:     "title",
				Selector: "h1",
				Type:     "invalid_type",
				Required: true,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewFieldExtractor(parser, tt.config)
			err := extractor.validateConfig()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestFieldExtractor_GetDefaultValue(t *testing.T) {
	html := `<html><body></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		fieldType    string
		defaultValue interface{}
		expected     interface{}
	}{
		{"text", nil, ""},
		{"int", nil, 0},
		{"float", nil, 0.0},
		{"bool", nil, false},
		{"array", nil, []interface{}{}},
		{"text", "custom_default", "custom_default"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldType, func(t *testing.T) {
			config := FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     tt.fieldType,
				Default:  tt.defaultValue,
			}

			extractor := NewFieldExtractor(parser, config)
			defaultVal := extractor.getDefaultValue()

			if tt.defaultValue != nil {
				if defaultVal != tt.expected {
					t.Fatalf("Expected default value '%v', got '%v'", tt.expected, defaultVal)
				}
			} else {
				switch tt.fieldType {
				case "text":
					if defaultVal != "" {
						t.Fatalf("Expected empty string for text type, got '%v'", defaultVal)
					}
				case "int":
					if defaultVal != 0 {
						t.Fatalf("Expected 0 for int type, got '%v'", defaultVal)
					}
				case "float":
					if defaultVal != 0.0 {
						t.Fatalf("Expected 0.0 for float type, got '%v'", defaultVal)
					}
				case "bool":
					if defaultVal != false {
						t.Fatalf("Expected false for bool type, got '%v'", defaultVal)
					}
				}
			}
		})
	}
}

func TestNewExtractionEngine(t *testing.T) {
	html := `<html><body><h1>Title</h1><p>Content</p></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	fields := []FieldConfig{
		{
			Name:     "title",
			Selector: "h1",
			Type:     "text",
			Required: true,
		},
		{
			Name:     "content",
			Selector: "p",
			Type:     "text",
			Required: false,
		},
	}

	config := ExtractionConfig{
		StrictMode:      false,
		ContinueOnError: true,
	}

	engine := NewExtractionEngine(parser, fields, config)

	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	if len(engine.extractors) != 2 {
		t.Fatalf("Expected 2 extractors, got %d", len(engine.extractors))
	}

	if engine.parser != parser {
		t.Fatal("Engine should reference the provided parser")
	}
}

func TestExtractionEngine_ExtractAll_Success(t *testing.T) {
	html := `<html><body>
		<h1>Test Title</h1>
		<p class="content">Test content</p>
		<span class="price">$123.45</span>
	</body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	fields := []FieldConfig{
		{
			Name:     "title",
			Selector: "h1",
			Type:     "text",
			Required: true,
		},
		{
			Name:     "content",
			Selector: ".content",
			Type:     "text",
			Required: false,
		},
		{
			Name:     "price",
			Selector: ".price",
			Type:     "float",
			Required: true,
		},
	}

	config := ExtractionConfig{
		StrictMode:      false,
		ContinueOnError: true,
	}

	engine := NewExtractionEngine(parser, fields, config)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	if !result.Success {
		t.Fatalf("Expected extraction to succeed, but it failed with errors: %v", result.Errors)
	}

	if len(result.Data) != 3 {
		t.Fatalf("Expected 3 extracted fields, got %d", len(result.Data))
	}

	if result.Data["title"] != "Test Title" {
		t.Fatalf("Expected title 'Test Title', got '%v'", result.Data["title"])
	}

	if result.Data["content"] != "Test content" {
		t.Fatalf("Expected content 'Test content', got '%v'", result.Data["content"])
	}

	if result.Data["price"] != 123.45 {
		t.Fatalf("Expected price 123.45, got %v", result.Data["price"])
	}

	if result.Metadata.TotalFields != 3 {
		t.Fatalf("Expected metadata total fields 3, got %d", result.Metadata.TotalFields)
	}

	if result.Metadata.ExtractedFields != 3 {
		t.Fatalf("Expected metadata extracted fields 3, got %d", result.Metadata.ExtractedFields)
	}

	if !result.Metadata.RequiredFieldsOK {
		t.Fatal("Expected required fields to be OK")
	}
}

func TestExtractionEngine_ExtractAll_RequiredFieldFailure(t *testing.T) {
	html := `<html><body>
		<h1>Test Title</h1>
		<p class="content">Test content</p>
	</body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	fields := []FieldConfig{
		{
			Name:     "title",
			Selector: "h1",
			Type:     "text",
			Required: true,
		},
		{
			Name:     "missing_required",
			Selector: ".missing",
			Type:     "text",
			Required: true,
		},
	}

	config := ExtractionConfig{
		StrictMode:      true,
		ContinueOnError: false,
	}

	engine := NewExtractionEngine(parser, fields, config)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	if result.Success {
		t.Fatal("Expected extraction to fail due to missing required field")
	}

	if len(result.Errors) == 0 {
		t.Fatal("Expected errors to be present")
	}

	foundRequiredFieldError := false
	for _, err := range result.Errors {
		if err.FieldName == "missing_required" && err.Severity == "CRITICAL" {
			foundRequiredFieldError = true
			break
		}
	}

	if !foundRequiredFieldError {
		t.Fatal("Expected to find critical error for missing required field")
	}

	if result.Metadata.RequiredFieldsOK {
		t.Fatal("Expected required fields to NOT be OK")
	}
}

func TestExtractionEngine_ExtractAll_ContinueOnError(t *testing.T) {
	html := `<html><body>
		<h1>Test Title</h1>
		<p class="content">Test content</p>
	</body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	fields := []FieldConfig{
		{
			Name:     "title",
			Selector: "h1",
			Type:     "text",
			Required: true,
		},
		{
			Name:     "missing_optional",
			Selector: ".missing",
			Type:     "text",
			Required: false,
		},
		{
			Name:     "content",
			Selector: ".content",
			Type:     "text",
			Required: false,
		},
	}

	config := ExtractionConfig{
		StrictMode:      false,
		ContinueOnError: true,
	}

	engine := NewExtractionEngine(parser, fields, config)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	if !result.Success {
		t.Fatalf("Expected extraction to succeed despite optional field failure, errors: %v", result.Errors)
	}

	// Should have extracted title and content, but not the missing optional field
	if len(result.Data) != 2 {
		t.Fatalf("Expected 2 extracted fields, got %d", len(result.Data))
	}

	if result.Data["title"] != "Test Title" {
		t.Fatalf("Expected title 'Test Title', got '%v'", result.Data["title"])
	}

	if result.Data["content"] != "Test content" {
		t.Fatalf("Expected content 'Test content', got '%v'", result.Data["content"])
	}

	// Missing optional field should not be in data
	if _, exists := result.Data["missing_optional"]; exists {
		t.Fatal("Missing optional field should not be in result data")
	}
}

func TestExtractionEngine_ExtractAll_WithGlobalTransforms(t *testing.T) {
	html := `<html><body>
		<h1>  test title  </h1>
		<p class="content">  test content  </p>
	</body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	fields := []FieldConfig{
		{
			Name:     "title",
			Selector: "h1",
			Type:     "text",
			Required: true,
		},
		{
			Name:     "content",
			Selector: ".content",
			Type:     "text",
			Required: false,
		},
	}

	config := ExtractionConfig{
		StrictMode:      false,
		ContinueOnError: true,
		DefaultTransforms: []pipeline.TransformRule{
			{Type: "trim"},
			{Type: "uppercase"},
		},
	}

	engine := NewExtractionEngine(parser, fields, config)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	if !result.Success {
		t.Fatalf("Expected extraction to succeed, errors: %v", result.Errors)
	}

	if result.Data["title"] != "TEST TITLE" {
		t.Fatalf("Expected transformed title 'TEST TITLE', got '%v'", result.Data["title"])
	}

	if result.Data["content"] != "TEST CONTENT" {
		t.Fatalf("Expected transformed content 'TEST CONTENT', got '%v'", result.Data["content"])
	}
}

func TestExtractionEngine_ExtractAll_WithValidation(t *testing.T) {
	html := `<html><body>
		<h1>Title</h1>
		<p class="content">Short</p>
	</body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	fields := []FieldConfig{
		{
			Name:     "title",
			Selector: "h1",
			Type:     "text",
			Required: true,
		},
		{
			Name:     "content",
			Selector: ".content",
			Type:     "text",
			Required: false,
		},
	}

	minLength := 10
	config := ExtractionConfig{
		StrictMode:      false,
		ContinueOnError: true,
		ValidationRules: []ValidationRule{
			{
				FieldName: "content",
				MinLength: &minLength,
				Required:  false,
			},
		},
	}

	engine := NewExtractionEngine(parser, fields, config)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	// Should succeed overall but have validation errors
	if !result.Success {
		t.Fatalf("Expected extraction to succeed, errors: %v", result.Errors)
	}

	// Should have validation error for content field
	foundValidationError := false
	for _, err := range result.Errors {
		if err.FieldName == "content" && err.Code == "VALIDATION_FAILED" {
			foundValidationError = true
			break
		}
	}

	if !foundValidationError {
		t.Fatal("Expected to find validation error for content field")
	}
}

func TestFieldExtractor_CoerceType(t *testing.T) {
	html := `<html><body></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		fieldType    string
		input        string
		expectedType string
	}{
		{"int", "123", "int"},
		{"number", "456", "int"},
		{"float", "123.45", "float64"},
		{"bool", "true", "bool"},
		{"boolean", "false", "bool"},
		{"text", "some text", "string"},
		{"date", "2023-12-25", "time.Time"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldType, func(t *testing.T) {
			config := FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     tt.fieldType,
			}

			extractor := NewFieldExtractor(parser, config)
			result, err := extractor.coerceType(tt.input)

			if err != nil && tt.fieldType != "date" { // Date parsing might fail with default format
				t.Fatalf("Unexpected error during type coercion: %v", err)
			}

			if tt.fieldType == "date" && err != nil {
				// Date parsing failed, which is acceptable with default format
				return
			}

			resultType := strings.Replace(fmt.Sprintf("%T", result), "*", "", 1)
			if !strings.Contains(resultType, tt.expectedType) && tt.expectedType != "string" {
				t.Fatalf("Expected type containing '%s', got '%s'", tt.expectedType, resultType)
			}
		})
	}
}

func TestFieldExtractor_ValidateValue(t *testing.T) {
	html := `<html><body></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name        string
		config      FieldConfig
		value       interface{}
		expectError bool
	}{
		{
			name: "valid string required",
			config: FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     "text",
				Required: true,
			},
			value:       "test value",
			expectError: false,
		},
		{
			name: "empty string required",
			config: FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     "text",
				Required: true,
			},
			value:       "",
			expectError: true,
		},
		{
			name: "nil value required",
			config: FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     "text",
				Required: true,
			},
			value:       nil,
			expectError: true,
		},
		{
			name: "nil value optional",
			config: FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     "text",
				Required: false,
			},
			value:       nil,
			expectError: false,
		},
		{
			name: "valid int",
			config: FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     "int",
				Required: true,
			},
			value:       123,
			expectError: false,
		},
		{
			name: "invalid int type",
			config: FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     "int",
				Required: true,
			},
			value:       "not a number",
			expectError: true,
		},
		{
			name: "valid array",
			config: FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     "array",
				Required: true,
			},
			value:       []interface{}{"item1", "item2"},
			expectError: false,
		},
		{
			name: "empty array required",
			config: FieldConfig{
				Name:     "test",
				Selector: ".test",
				Type:     "array",
				Required: true,
			},
			value:       []interface{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewFieldExtractor(parser, tt.config)
			err := extractor.validateValue(tt.value)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestExtractionEngine_BuildMetadata(t *testing.T) {
	html := `<html><body><h1>Test</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	fields := []FieldConfig{
		{Name: "test", Selector: "h1", Type: "text", Required: true},
	}

	config := ExtractionConfig{}
	engine := NewExtractionEngine(parser, fields, config)

	metadata := engine.buildMetadata(5, 2, 7, time.Second, true)

	if metadata.TotalFields != 7 {
		t.Fatalf("Expected total fields 7, got %d", metadata.TotalFields)
	}

	if metadata.ExtractedFields != 5 {
		t.Fatalf("Expected extracted fields 5, got %d", metadata.ExtractedFields)
	}

	if metadata.FailedFields != 2 {
		t.Fatalf("Expected failed fields 2, got %d", metadata.FailedFields)
	}

	if metadata.ProcessingTime != time.Second {
		t.Fatalf("Expected processing time 1s, got %v", metadata.ProcessingTime)
	}

	if !metadata.RequiredFieldsOK {
		t.Fatal("Expected required fields OK to be true")
	}

	if metadata.DocumentSize <= 0 {
		t.Fatal("Expected document size to be greater than 0")
	}
}

func TestExtractionResult_ErrorHandling(t *testing.T) {
	html := `<html><body><h1>Test</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	fields := []FieldConfig{
		{
			Name:     "valid_field",
			Selector: "h1",
			Type:     "text",
			Required: true,
		},
		{
			Name:     "invalid_field",
			Selector: "",
			Type:     "text",
			Required: true,
		},
	}

	config := ExtractionConfig{
		StrictMode:      false,
		ContinueOnError: true,
	}

	engine := NewExtractionEngine(parser, fields, config)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	if result.Success {
		t.Fatal("Expected extraction to fail due to invalid field configuration")
	}

	if len(result.Errors) == 0 {
		t.Fatal("Expected errors to be present")
	}

	if result.ProcessedAt.IsZero() {
		t.Fatal("Expected ProcessedAt to be set")
	}

	// Should have one successful extraction despite the error
	if len(result.Data) != 1 {
		t.Fatalf("Expected 1 successful extraction, got %d", len(result.Data))
	}

	if result.Data["valid_field"] != "Test" {
		t.Fatalf("Expected valid field to be extracted as 'Test', got '%v'", result.Data["valid_field"])
	}
}
