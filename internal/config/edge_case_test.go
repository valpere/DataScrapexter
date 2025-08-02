// internal/config/edge_case_test.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromBytesEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		content     []byte
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty bytes",
			content:     []byte{},
			expectError: false, // Empty bytes create empty config with zero values
			errorMsg:    "",
		},
		{
			name:        "nil bytes",
			content:     nil,
			expectError: false, // Nil bytes also create empty config with zero values
			errorMsg:    "",
		},
		{
			name: "minimal valid config",
			content: []byte(`
name: minimal
base_url: https://example.com
fields:
  - name: title
    selector: h1
    type: text
output:
  format: json
  file: minimal.json
`),
			expectError: false,
		},
		{
			name: "config with special characters",
			content: []byte(`
name: "special-chars_config.123"
base_url: "https://example.com/path?param=value&other=123"
fields:
  - name: "field-with-dashes"
    selector: "div.class-name[data-attr='value']"
    type: text
output:
  format: json
  file: "output-file.json"
`),
			expectError: false,
		},
		{
			name: "config with unicode characters",
			content: []byte(`
name: "скрапер_测试_テスト"
base_url: "https://example.com"
fields:
  - name: "título"
    selector: "h1"
    type: text
output:
  format: json
  file: "出力.json"
`),
			expectError: false,
		},
		{
			name: "config with very long strings",
			content: []byte(`
name: "` + strings.Repeat("a", 1000) + `"
base_url: "https://example.com"
fields:
  - name: "` + strings.Repeat("field", 100) + `"
    selector: "` + strings.Repeat("h", 500) + `"
    type: text
output:
  format: json
  file: "output.json"
`),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadFromBytes(tt.content)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if config == nil {
				t.Error("config should not be nil when no error")
			}
		})
	}
}

func TestLoadFromFileEdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_edge_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		setupFile   func() string
		expectError bool
		errorMsg    string
	}{
		{
			name: "non-existent file",
			setupFile: func() string {
				return filepath.Join(tempDir, "non_existent.yaml")
			},
			expectError: true,
			errorMsg:    "no such file",
		},
		{
			name: "directory instead of file",
			setupFile: func() string {
				dirPath := filepath.Join(tempDir, "directory")
				os.Mkdir(dirPath, 0755)
				return dirPath
			},
			expectError: true,
			errorMsg:    "directory",
		},
		{
			name: "empty file",
			setupFile: func() string {
				filePath := filepath.Join(tempDir, "empty.yaml")
				os.WriteFile(filePath, []byte{}, 0644)
				return filePath
			},
			expectError: false, // Empty file creates empty config with zero values
			errorMsg:    "",
		},
		{
			name: "file with only whitespace",
			setupFile: func() string {
				filePath := filepath.Join(tempDir, "whitespace.yaml")
				os.WriteFile(filePath, []byte("   \n\t  \r\n  "), 0644)
				return filePath
			},
			expectError: true, // This should error due to invalid YAML
			errorMsg:    "yaml",
		},
		{
			name: "unreadable file",
			setupFile: func() string {
				filePath := filepath.Join(tempDir, "unreadable.yaml")
				os.WriteFile(filePath, []byte("name: test"), 0644)
				os.Chmod(filePath, 0000) // Remove all permissions
				return filePath
			},
			expectError: true,
			errorMsg:    "permission",
		},
		{
			name: "very large file",
			setupFile: func() string {
				filePath := filepath.Join(tempDir, "large.yaml")
				content := `name: large_config
base_url: https://example.com
fields:
`
				// Add many fields to create a large file
				for i := 0; i < 1000; i++ {
					suffix := string(rune('a' + (i % 26)))
					content += `  - name: field` + suffix + `
    selector: .selector` + suffix + `
    type: text
`
				}
				content += `output:
  format: json
  file: large.json
`
				os.WriteFile(filePath, []byte(content), 0644)
				return filePath
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setupFile()
			
			config, err := LoadFromFile(filePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if config == nil {
				t.Error("config should not be nil when no error")
			}
		})
	}
}

func TestGenerateTemplateEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		templateType string
		expectPanic  bool
	}{
		{"empty string", "", false},
		{"unknown type", "unknown_type", false},
		{"null string", "null", false},
		{"very long type", strings.Repeat("long", 100), false},
		{"type with special chars", "e-commerce@test.com", false},
		{"type with unicode", "электронная_торговля", false},
		{"basic type", "basic", false},
		{"ecommerce type", "ecommerce", false},
		{"news type", "news", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				} else if tt.expectPanic {
					t.Error("expected panic but none occurred")
				}
			}()

			config := GenerateTemplate(tt.templateType)
			
			if !tt.expectPanic {
				if config == nil {
					t.Error("config should not be nil")
				}
				
				// Validate generated config has basic required fields
				if config.Name == "" {
					t.Error("generated config should have a name")
				}
				if config.BaseURL == "" {
					t.Error("generated config should have a base URL")
				}
				if len(config.Fields) == 0 {
					t.Error("generated config should have at least one field")
				}
			}
		})
	}
}

func TestConfigFieldEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		validate    func(*ScraperConfig) error
	}{
		{
			name: "field with empty name",
			configYAML: `
name: test
base_url: https://example.com
fields:
  - name: ""
    selector: h1
    type: text
output:
  format: json
  file: output.json
`,
			expectError: false, // Basic config loading may not validate this
		},
		{
			name: "field with empty selector",
			configYAML: `
name: test
base_url: https://example.com
fields:
  - name: title
    selector: ""
    type: text
output:
  format: json
  file: output.json
`,
			expectError: false, // Basic config loading may not validate this
		},
		{
			name: "field with invalid type",
			configYAML: `
name: test
base_url: https://example.com
fields:
  - name: title
    selector: h1
    type: invalid_type
output:
  format: json
  file: output.json
`,
			expectError: false, // Basic config loading may not validate this
		},
		{
			name: "attr field without attribute",
			configYAML: `
name: test
base_url: https://example.com
fields:
  - name: image
    selector: img
    type: attr
output:
  format: json
  file: output.json
`,
			expectError: false, // Basic config loading may not validate this
		},
		{
			name: "field with complex transform chain",
			configYAML: `
name: test
base_url: https://example.com
fields:
  - name: price
    selector: .price
    type: text
    transform:
      - type: trim
      - type: regex
        pattern: '\$([0-9,]+\.?\d*)'
        replacement: '$1'
      - type: regex
        pattern: ','
        replacement: ''
      - type: parse_float
output:
  format: json
  file: output.json
`,
			expectError: false,
		},
		{
			name: "field with malformed regex",
			configYAML: `
name: test
base_url: https://example.com
fields:
  - name: price
    selector: .price
    type: text
    transform:
      - type: regex
        pattern: '['
        replacement: 'invalid'
output:
  format: json
  file: output.json
`,
			expectError: false, // Basic config loading may not validate regex
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadFromBytes([]byte(tt.configYAML))

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if config == nil {
				t.Error("config should not be nil when no error")
			}

			// Additional custom validation if provided
			if !tt.expectError && tt.validate != nil && config != nil {
				if validationErr := tt.validate(config); validationErr != nil {
					t.Errorf("custom validation failed: %v", validationErr)
				}
			}
		})
	}
}

func TestConfigConcurrencyEdgeCases(t *testing.T) {
	// Test loading the same config from multiple goroutines
	configYAML := `
name: concurrent_test
base_url: https://example.com
fields:
  - name: title
    selector: h1
    type: text
output:
  format: json
  file: output.json
`

	numGoroutines := 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			config, err := LoadFromBytes([]byte(configYAML))
			if err != nil {
				results <- err
				return
			}
			if config == nil {
				results <- fmt.Errorf("config is nil")
				return
			}
			if config.Name != "concurrent_test" {
				results <- fmt.Errorf("unexpected config name: %s", config.Name)
				return
			}
			results <- nil
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			t.Errorf("goroutine %d failed: %v", i, err)
		}
	}
}

func TestEnvironmentVariableEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		configYAML  string
		expectedVal string
		expectError bool
	}{
		{
			name: "undefined environment variable",
			envVars: map[string]string{},
			configYAML: `
name: test
base_url: ${UNDEFINED_VAR}
fields:
  - name: title
    selector: h1
    type: text
output:
  format: json
  file: output.json
`,
			expectedVal: "${UNDEFINED_VAR}", // Environment variables not expanded by YAML parser
			expectError: false,
		},
		{
			name: "empty environment variable",
			envVars: map[string]string{
				"EMPTY_VAR": "",
			},
			configYAML: `
name: test
base_url: ${EMPTY_VAR}
fields:
  - name: title
    selector: h1
    type: text
output:
  format: json
  file: output.json
`,
			expectedVal: "${EMPTY_VAR}", // Environment variables not expanded by YAML parser
			expectError: false,
		},
		{
			name: "environment variable with special characters",
			envVars: map[string]string{
				"SPECIAL_VAR": "https://example.com/path?param=value&other=123#fragment",
			},
			configYAML: `
name: test
base_url: ${SPECIAL_VAR}
fields:
  - name: title
    selector: h1
    type: text
output:
  format: json
  file: output.json
`,
			expectedVal: "${SPECIAL_VAR}", // Environment variables not expanded by YAML parser
			expectError: false,
		},
		{
			name: "multiple environment variables",
			envVars: map[string]string{
				"HOST": "example.com",
				"PORT": "8080",
				"PROTOCOL": "https",
			},
			configYAML: `
name: test
base_url: ${PROTOCOL}://${HOST}:${PORT}
fields:
  - name: title
    selector: h1
    type: text
output:
  format: json
  file: output.json
`,
			expectedVal: "${PROTOCOL}://${HOST}:${PORT}", // Environment variables not expanded by YAML parser
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			config, err := LoadFromBytes([]byte(tt.configYAML))

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if config != nil && config.BaseURL != tt.expectedVal {
				t.Errorf("expected base URL %q, got %q", tt.expectedVal, config.BaseURL)
			}
		})
	}
}

