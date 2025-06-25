// internal/pipeline/transform_test.go
package pipeline

import (
	"context"
	"testing"
)

func TestTransformRule_Transform(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		rule        TransformRule
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "trim spaces",
			rule:        TransformRule{Type: "trim"},
			input:       "  hello world  ",
			expected:    "hello world",
			expectError: false,
		},
		{
			name:        "normalize spaces",
			rule:        TransformRule{Type: "normalize_spaces"},
			input:       "hello    world\n\ttest",
			expected:    "hello world test",
			expectError: false,
		},
		{
			name:        "lowercase",
			rule:        TransformRule{Type: "lowercase"},
			input:       "HELLO World",
			expected:    "hello world",
			expectError: false,
		},
		{
			name:        "uppercase",
			rule:        TransformRule{Type: "uppercase"},
			input:       "hello world",
			expected:    "HELLO WORLD",
			expectError: false,
		},
		{
			name:        "remove html",
			rule:        TransformRule{Type: "remove_html"},
			input:       "This is <b>bold</b> text",
			expected:    "This is bold text",
			expectError: false,
		},
		{
			name:        "extract number",
			rule:        TransformRule{Type: "extract_number"},
			input:       "Price: $123.45",
			expected:    "123.45",
			expectError: false,
		},
		{
			name:        "parse int",
			rule:        TransformRule{Type: "parse_int"},
			input:       "123",
			expected:    "123",
			expectError: false,
		},
		{
			name:        "parse float",
			rule:        TransformRule{Type: "parse_float"},
			input:       "123.45",
			expected:    "123.45",
			expectError: false,
		},
		{
			name: "regex replace",
			rule: TransformRule{
				Type:        "regex",
				Pattern:     `\$([0-9,]+\.?[0-9]*)`,
				Replacement: "$1",
			},
			input:       "$1,299.99",
			expected:    "1,299.99",
			expectError: false,
		},
		{
			name: "prefix transform",
			rule: TransformRule{
				Type: "prefix",
				Params: map[string]interface{}{
					"value": "https://",
				},
			},
			input:       "example.com",
			expected:    "https://example.com",
			expectError: false,
		},
		{
			name: "suffix transform",
			rule: TransformRule{
				Type: "suffix",
				Params: map[string]interface{}{
					"value": ".html",
				},
			},
			input:       "page",
			expected:    "page.html",
			expectError: false,
		},
		{
			name: "replace transform",
			rule: TransformRule{
				Type: "replace",
				Params: map[string]interface{}{
					"old": "old",
					"new": "new",
				},
			},
			input:       "old text",
			expected:    "new text",
			expectError: false,
		},
		{
			name:        "invalid transform type",
			rule:        TransformRule{Type: "invalid_type"},
			input:       "test",
			expected:    "",
			expectError: true,
		},
		{
			name:        "regex without pattern",
			rule:        TransformRule{Type: "regex"},
			input:       "test",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.rule.Apply(ctx, tt.input)

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

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTransformList_Apply(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		rules       TransformList
		input       string
		expected    string
		expectError bool
	}{
		{
			name: "chain transformations",
			rules: TransformList{
				{Type: "trim"},
				{Type: "lowercase"},
				{Type: "normalize_spaces"},
			},
			input:       "  HELLO    WORLD  ",
			expected:    "hello world",
			expectError: false,
		},
		{
			name: "regex then parse",
			rules: TransformList{
				{
					Type:        "regex",
					Pattern:     `\$([0-9,]+\.?[0-9]*)`,
					Replacement: "$1",
				},
				{Type: "parse_float"},
			},
			input:       "$1,299.99",
			expected:    "1299.99",
			expectError: false,
		},
		{
			name: "error in chain",
			rules: TransformList{
				{Type: "trim"},
				{Type: "invalid_type"},
			},
			input:       "test",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.rules.Apply(ctx, tt.input)

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

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateTransformRules(t *testing.T) {
	tests := []struct {
		name        string
		rules       TransformList
		expectError bool
	}{
		{
			name: "valid rules",
			rules: TransformList{
				{Type: "trim"},
				{Type: "lowercase"},
				{Type: "normalize_spaces"},
			},
			expectError: false,
		},
		{
			name: "valid regex rule",
			rules: TransformList{
				{
					Type:        "regex",
					Pattern:     `\d+`,
					Replacement: "NUMBER",
				},
			},
			expectError: false,
		},
		{
			name: "regex without pattern",
			rules: TransformList{
				{Type: "regex"},
			},
			expectError: true,
		},
		{
			name: "invalid transform type",
			rules: TransformList{
				{Type: "invalid_type"},
			},
			expectError: true,
		},
		{
			name: "prefix without value",
			rules: TransformList{
				{Type: "prefix"},
			},
			expectError: true,
		},
		{
			name: "replace without parameters",
			rules: TransformList{
				{Type: "replace"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransformRules(tt.rules)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDataTransformer_TransformData(t *testing.T) {
	ctx := context.Background()

	transformer := &DataTransformer{
		Global: TransformList{
			{Type: "trim"},
		},
		Fields: []FieldTransform{
			{
				Name: "price",
				Rules: TransformList{
					{
						Type:        "regex",
						Pattern:     `\$([0-9,]+\.?[0-9]*)`,
						Replacement: "$1",
					},
					{Type: "parse_float"},
				},
				Required: true,
			},
			{
				Name: "title",
				Rules: TransformList{
					{Type: "normalize_spaces"},
				},
				Required:   false,
				DefaultVal: "No Title",
			},
		},
	}

	tests := []struct {
		name        string
		input       map[string]interface{}
		expected    map[string]interface{}
		expectError bool
	}{
		{
			name: "successful transformation",
			input: map[string]interface{}{
				"price":       "$123.45",
				"title":       "  Hello    World  ",
				"description": "  Some description  ",
			},
			expected: map[string]interface{}{
				"price":       "123.45",
				"title":       "Hello World",
				"description": "Some description",
			},
			expectError: false,
		},
		{
			name: "missing required field",
			input: map[string]interface{}{
				"title": "Hello World",
			},
			expectError: true,
		},
		{
			name: "missing optional field with default",
			input: map[string]interface{}{
				"price": "$99.99",
			},
			expected: map[string]interface{}{
				"price": "99.99",
				"title": "No Title",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.TransformData(ctx, tt.input)

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
