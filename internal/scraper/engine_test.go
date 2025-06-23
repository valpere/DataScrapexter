// internal/scraper/engine_test.go
package scraper

import (
	"context"
	"testing"

	"github.com/valpere/DataScrapexter/internal/pipeline"
)

func TestNewScrapingEngine(t *testing.T) {
	config := &EngineConfig{
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
			},
		},
	}

	engine := NewScrapingEngine(config)

	if engine == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	if engine.Config != config {
		t.Errorf("expected config to be set correctly")
	}
}

func TestScrapingEngine_ProcessFields(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *EngineConfig
		input       map[string]interface{}
		expected    map[string]interface{}
		expectError bool
	}{
		{
			name: "basic field processing",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
						Required: false,
					},
					{
						Name:     "price",
						Selector: ".price",
						Type:     "text",
						Required: false,
					},
				},
			},
			input: map[string]interface{}{
				"title": "Product Title",
				"price": "$99.99",
			},
			expected: map[string]interface{}{
				"title": "Product Title",
				"price": "$99.99",
			},
			expectError: false,
		},
		{
			name: "field with transformations",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "price",
						Selector: ".price",
						Type:     "text",
						Required: false,
						Transform: []pipeline.TransformRule{
							{
								Type:        "regex",
								Pattern:     `\$([0-9,]+\.?[0-9]*)`,
								Replacement: "$1",
							},
							{
								Type: "parse_float",
							},
						},
					},
				},
			},
			input: map[string]interface{}{
				"price": "$1,234.56",
			},
			expected: map[string]interface{}{
				"price": "1234.56",
			},
			expectError: false,
		},
		{
			name: "required field missing",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
						Required: true,
					},
				},
			},
			input: map[string]interface{}{
				"description": "Some description",
			},
			expectError: true,
		},
		{
			name: "transformation error",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "price",
						Selector: ".price",
						Type:     "text",
						Required: false,
						Transform: []pipeline.TransformRule{
							{
								Type: "parse_float",
							},
						},
					},
				},
			},
			input: map[string]interface{}{
				"price": "not a number",
			},
			expectError: true,
		},
		{
			name: "non-string value with transformations",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "count",
						Selector: ".count",
						Type:     "number",
						Required: false,
						Transform: []pipeline.TransformRule{
							{
								Type: "trim",
							},
						},
					},
				},
			},
			input: map[string]interface{}{
				"count": 42,
			},
			expected: map[string]interface{}{
				"count": 42,
			},
			expectError: false,
		},
		{
			name: "empty field configuration",
			config: &EngineConfig{
				Fields: []FieldConfig{},
			},
			input: map[string]interface{}{
				"title": "Test",
			},
			expected: map[string]interface{}{
				"title": "Test", // Now includes unconfigured fields
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewScrapingEngine(tt.config)
			result, err := engine.ProcessFields(ctx, tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d", len(tt.expected), len(result))
			}

			for key, expectedVal := range tt.expected {
				if actualVal, exists := result[key]; !exists {
					t.Errorf("expected key %q not found in result", key)
				} else if actualVal != expectedVal {
					t.Errorf("for key %q: expected %v, got %v", key, expectedVal, actualVal)
				}
			}
		})
	}
}

func TestScrapingEngine_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *EngineConfig
		expectError bool
	}{
		{
			name: "valid configuration",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
					},
					{
						Name:     "price",
						Selector: ".price",
						Type:     "text",
						Transform: []pipeline.TransformRule{
							{Type: "trim"},
							{Type: "parse_float"},
						},
					},
				},
				Transform: []pipeline.TransformRule{
					{Type: "normalize_spaces"},
				},
			},
			expectError: false,
		},
		{
			name:        "nil configuration",
			config:      nil,
			expectError: true,
		},
		{
			name: "field without name",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Selector: "h1",
						Type:     "text",
					},
				},
			},
			expectError: true,
		},
		{
			name: "field without selector",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name: "title",
						Type: "text",
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid field transformation",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
						Transform: []pipeline.TransformRule{
							{Type: "regex"}, // Missing pattern
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid global transformation",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
					},
				},
				Transform: []pipeline.TransformRule{
					{Type: "unknown_type"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewScrapingEngine(tt.config)
			err := engine.ValidateConfig()

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFieldConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		field  FieldConfig
		valid  bool
	}{
		{
			name: "valid field",
			field: FieldConfig{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
			},
			valid: true,
		},
		{
			name: "missing name",
			field: FieldConfig{
				Selector: "h1",
				Type:     "text",
			},
			valid: false,
		},
		{
			name: "missing selector",
			field: FieldConfig{
				Name: "title",
				Type: "text",
			},
			valid: false,
		},
		{
			name: "empty strings",
			field: FieldConfig{
				Name:     "",
				Selector: "",
				Type:     "text",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.field.Name != "" && tt.field.Selector != ""
			
			if isValid != tt.valid {
				t.Errorf("expected valid=%t, got valid=%t", tt.valid, isValid)
			}
		})
	}
}

// Integration test with real HTML-like data
func TestScrapingEngine_Integration(t *testing.T) {
	ctx := context.Background()

	config := &EngineConfig{
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
				Transform: []pipeline.TransformRule{
					{Type: "trim"},
					{Type: "normalize_spaces"},
				},
			},
			{
				Name:     "price",
				Selector: ".price",
				Type:     "text",
				Required: true,
				Transform: []pipeline.TransformRule{
					{
						Type:        "regex",
						Pattern:     `\$([0-9,]+\.?[0-9]*)`,
						Replacement: "$1",
					},
					{Type: "parse_float"},
				},
			},
			{
				Name:     "description",
				Selector: ".description",
				Type:     "text",
				Required: false,
				Transform: []pipeline.TransformRule{
					{Type: "trim"},
					{Type: "remove_html"},
				},
			},
		},
		Transform: []pipeline.TransformRule{
			{Type: "normalize_spaces"},
		},
	}

	engine := NewScrapingEngine(config)

	// Simulate extracted data
	input := map[string]interface{}{
		"title":       "  Amazing    Product  ",
		"price":       "$1,234.56",
		"description": "<p>Great <strong>product</strong> for everyone!</p>",
		"extra_field": "This should be processed by global transforms",
	}

	result, err := engine.ProcessFields(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify results
	expectedTitle := "Amazing Product"
	if result["title"] != expectedTitle {
		t.Errorf("expected title %q, got %q", expectedTitle, result["title"])
	}

	expectedPrice := "1234.56"
	if result["price"] != expectedPrice {
		t.Errorf("expected price %q, got %q", expectedPrice, result["price"])
	}

	expectedDescription := "Great product for everyone!"
	if result["description"] != expectedDescription {
		t.Errorf("expected description %q, got %q", expectedDescription, result["description"])
	}

	// extra_field should be processed by global transforms
	expectedExtra := "This should be processed by global transforms"
	if result["extra_field"] != expectedExtra {
		t.Errorf("expected extra_field %q, got %q", expectedExtra, result["extra_field"])
	}
}

// Benchmark tests
func BenchmarkScrapingEngine_ProcessFields(b *testing.B) {
	ctx := context.Background()
	
	config := &EngineConfig{
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Transform: []pipeline.TransformRule{
					{Type: "trim"},
					{Type: "normalize_spaces"},
				},
			},
			{
				Name:     "price",
				Selector: ".price",
				Type:     "text",
				Transform: []pipeline.TransformRule{
					{
						Type:        "regex",
						Pattern:     `\$([0-9,]+\.?[0-9]*)`,
						Replacement: "$1",
					},
					{Type: "parse_float"},
				},
			},
		},
	}

	engine := NewScrapingEngine(config)
	input := map[string]interface{}{
		"title": "  Product   Title  ",
		"price": "$99.99",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.ProcessFields(ctx, input)
	}
}

func BenchmarkScrapingEngine_ValidateConfig(b *testing.B) {
	config := &EngineConfig{
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Transform: []pipeline.TransformRule{
					{Type: "trim"},
				},
			},
		},
	}

	engine := NewScrapingEngine(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.ValidateConfig()
	}
}
