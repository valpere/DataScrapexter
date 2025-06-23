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
			name: "trim spaces",
			rule: TransformRule{Type: "trim"},
			input: "  hello world  ",
			expected: "hello world",
			expectError: false,
		},
		{
			name: "normalize spaces",
			rule: TransformRule{Type: "normalize_spaces"},
			input: "hello    world\n\ttest",
			expected: "hello world test",
			expectError: false,
		},
		{
			name: "regex replace",
			rule: TransformRule{
				Type: "regex",
				Pattern: `\$([0-9,]+\.?[0-9]*)`,
				Replacement: "$1",
			},
			input: "$1,234.56",
			expected: "1,234.56",
			expectError: false,
		},
		{
			name: "regex invalid pattern",
			rule: TransformRule{
				Type: "regex",
				Pattern: `[invalid`,
			},
			input: "test",
			expectError: true,
		},
		{
			name: "parse float valid",
			rule: TransformRule{Type: "parse_float"},
			input: "$1,234.56",
			expected: "1234.56",
			expectError: false,
		},
		{
			name: "parse float invalid",
			rule: TransformRule{Type: "parse_float"},
			input: "not a number",
			expectError: true,
		},
		{
			name: "parse int valid",
			rule: TransformRule{Type: "parse_int"},
			input: "1,234",
			expected: "1234",
			expectError: false,
		},
		{
			name: "lowercase",
			rule: TransformRule{Type: "lowercase"},
			input: "HELLO World",
			expected: "hello world",
			expectError: false,
		},
		{
			name: "uppercase",
			rule: TransformRule{Type: "uppercase"},
			input: "hello world",
			expected: "HELLO WORLD",
			expectError: false,
		},
		{
			name: "remove html",
			rule: TransformRule{Type: "remove_html"},
			input: "<p>Hello <strong>world</strong></p>",
			expected: "Hello world",
			expectError: false,
		},
		{
			name: "extract number",
			rule: TransformRule{Type: "extract_number"},
			input: "Price: $123.45 each",
			expected: "123.45",
			expectError: false,
		},
		{
			name: "extract number none found",
			rule: TransformRule{Type: "extract_number"},
			input: "No numbers here",
			expectError: true,
		},
		{
			name: "prefix",
			rule: TransformRule{
				Type: "prefix",
				Params: map[string]interface{}{"value": "https://"},
			},
			input: "example.com",
			expected: "https://example.com",
			expectError: false,
		},
		{
			name: "suffix",
			rule: TransformRule{
				Type: "suffix",
				Params: map[string]interface{}{"value": ".html"},
			},
			input: "page",
			expected: "page.html",
			expectError: false,
		},
		{
			name: "replace",
			rule: TransformRule{
				Type: "replace",
				Params: map[string]interface{}{
					"old": "foo",
					"new": "bar",
				},
			},
			input: "foo is foo",
			expected: "bar is bar",
			expectError: false,
		},
		{
			name: "unknown transform type",
			rule: TransformRule{Type: "unknown"},
			input: "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.rule.Transform(ctx, tt.input)
			
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
			name: "single rule",
			rules: TransformList{
				{Type: "trim"},
			},
			input: "  hello  ",
			expected: "hello",
			expectError: false,
		},
		{
			name: "multiple rules",
			rules: TransformList{
				{Type: "trim"},
				{Type: "lowercase"},
				{Type: "normalize_spaces"},
			},
			input: "  HELLO    WORLD  ",
			expected: "hello world",
			expectError: false,
		},
		{
			name: "price extraction chain",
			rules: TransformList{
				{
					Type: "regex",
					Pattern: `\$([0-9,]+\.?[0-9]*)`,
					Replacement: "$1",
				},
				{Type: "parse_float"},
			},
			input: "$1,234.56",
			expected: "1234.56",
			expectError: false,
		},
		{
			name: "error in chain",
			rules: TransformList{
				{Type: "trim"},
				{Type: "unknown_type"},
			},
			input: "test",
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

func TestDataTransformer_TransformData(t *testing.T) {
	ctx := context.Background()

	transformer := &DataTransformer{
		Global: TransformList{
			{Type: "trim"},
		},
		Fields: []TransformField{
			{
				Name: "price",
				Rules: TransformList{
					{
						Type: "regex",
						Pattern: `\$([0-9,]+\.?[0-9]*)`,
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
				Required: false,
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
				"price": "$123.45",
				"title": "  Hello    World  ",
				"description": "  Some description  ",
			},
			expected: map[string]interface{}{
				"price": "123.45",
				"title": "Hello World",
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
					Type: "regex",
					Pattern: `\d+`,
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
			name: "unknown rule type",
			rules: TransformList{
				{Type: "unknown_type"},
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
			name: "valid prefix rule",
			rules: TransformList{
				{
					Type: "prefix",
					Params: map[string]interface{}{"value": "https://"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransformRules(tt.rules)
			
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkTransformRule_Trim(b *testing.B) {
	rule := TransformRule{Type: "trim"}
	ctx := context.Background()
	input := "  hello world  "
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Transform(ctx, input)
	}
}

func BenchmarkTransformRule_Regex(b *testing.B) {
	rule := TransformRule{
		Type: "regex",
		Pattern: `\$([0-9,]+\.?[0-9]*)`,
		Replacement: "$1",
	}
	ctx := context.Background()
	input := "$1,234.56"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Transform(ctx, input)
	}
}

func BenchmarkTransformList_Apply(b *testing.B) {
	rules := TransformList{
		{Type: "trim"},
		{Type: "lowercase"},
		{Type: "normalize_spaces"},
	}
	ctx := context.Background()
	input := "  HELLO    WORLD  "
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rules.Apply(ctx, input)
	}
}
