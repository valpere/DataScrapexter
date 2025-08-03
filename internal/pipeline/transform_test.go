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
			name:        "extract numbers",
			rule:        TransformRule{Type: "extract_numbers"},
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
			name:        "regex replace",
			rule:        TransformRule{Type: "regex", Pattern: `\$([0-9,]+\.\d*)`, Replacement: "$1"},
			input:       "$1,299.99",
			expected:    "1,299.99",
			expectError: false,
		},
		{
			name:        "prefix transform",
			rule:        TransformRule{Type: "prefix", Params: map[string]interface{}{"value": "https://"}},
			input:       "example.com",
			expected:    "https://example.com",
			expectError: false,
		},
		{
			name:        "suffix transform",
			rule:        TransformRule{Type: "suffix", Params: map[string]interface{}{"value": ".html"}},
			input:       "page",
			expected:    "page.html",
			expectError: false,
		},
		{
			name:        "replace transform",
			rule:        TransformRule{Type: "replace", Pattern: "old", Replacement: "new"},
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
			name: "chain transforms",
			rules: TransformList{
				{Type: "trim"},
				{Type: "lowercase"},
			},
			input:       "  HELLO WORLD  ",
			expected:    "hello world",
			expectError: false,
		},
		{
			name: "complex chain",
			rules: TransformList{
				{Type: "trim"},
				{Type: "regex", Pattern: `Price: \$([0-9,]+\.\d+)`, Replacement: "$1"},
				{Type: "regex", Pattern: `,`, Replacement: ""},
			},
			input:       "  Price: $1,299.99  ",
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
				{Type: "regex", Pattern: `\d+`, Replacement: "X"},
			},
			expectError: false,
		},
		{
			name: "invalid type",
			rules: TransformList{
				{Type: "invalid_type"},
			},
			expectError: true,
		},
		{
			name: "regex without pattern",
			rules: TransformList{
				{Type: "regex"},
			},
			expectError: true,
		},
		{
			name: "invalid regex pattern",
			rules: TransformList{
				{Type: "regex", Pattern: "["},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransformRules(tt.rules)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
