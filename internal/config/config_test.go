// internal/config/config_test.go
package config

import (
	"testing"
)

func TestScraperConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      ScraperConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: ScraperConfig{
				Name:    "test_scraper",
				BaseURL: "https://example.com",
				Fields: []Field{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
						Required: true,
					},
				},
				Output: OutputConfig{
					Format: "json",
					File:   "output.json",
				},
			},
			expectError: false,
		},
		{
			name: "missing name",
			config: ScraperConfig{
				BaseURL: "https://example.com",
				Fields: []Field{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
					},
				},
			},
			expectError: true,
		},
		{
			name: "missing base_url and urls",
			config: ScraperConfig{
				Name: "test_scraper",
				Fields: []Field{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
					},
				},
			},
			expectError: true,
		},
		{
			name: "no fields",
			config: ScraperConfig{
				Name:    "test_scraper",
				BaseURL: "https://example.com",
				Fields:  []Field{},
			},
			expectError: true,
		},
		{
			name: "field missing name",
			config: ScraperConfig{
				Name:    "test_scraper",
				BaseURL: "https://example.com",
				Fields: []Field{
					{
						Selector: "h1",
						Type:     "text",
					},
				},
			},
			expectError: true,
		},
		{
			name: "field missing selector",
			config: ScraperConfig{
				Name:    "test_scraper",
				BaseURL: "https://example.com",
				Fields: []Field{
					{
						Name: "title",
						Type: "text",
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid field type",
			config: ScraperConfig{
				Name:    "test_scraper",
				BaseURL: "https://example.com",
				Fields: []Field{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "invalid",
					},
				},
			},
			expectError: true,
		},
		{
			name: "attr type missing attribute",
			config: ScraperConfig{
				Name:    "test_scraper",
				BaseURL: "https://example.com",
				Fields: []Field{
					{
						Name:     "image",
						Selector: "img",
						Type:     "attr",
					},
				},
			},
			expectError: true,
		},
		{
			name: "attr type with attribute",
			config: ScraperConfig{
				Name:    "test_scraper",
				BaseURL: "https://example.com",
				Fields: []Field{
					{
						Name:      "image",
						Selector:  "img",
						Type:      "attr",
						Attribute: "src",
					},
				},
				Output: OutputConfig{
					Format: "json",
					File:   "output.json",
				},
			},
			expectError: false,
		},
		{
			name: "invalid output format",
			config: ScraperConfig{
				Name:    "test_scraper",
				BaseURL: "https://example.com",
				Fields: []Field{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
					},
				},
				Output: OutputConfig{
					Format: "invalid",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadFromBytes(t *testing.T) {
	yamlData := []byte(`
name: test_scraper
base_url: https://example.com
fields:
  - name: title
    selector: h1
    type: text
    required: true
output:
  format: json
  file: output.json
`)

	config, err := LoadFromBytes(yamlData)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if config.Name != "test_scraper" {
		t.Errorf("expected name 'test_scraper', got %s", config.Name)
	}

	if len(config.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(config.Fields))
	}

	if config.Fields[0].Name != "title" {
		t.Errorf("expected field name 'title', got %s", config.Fields[0].Name)
	}
}

func TestGenerateTemplate(t *testing.T) {
	tests := []struct {
		templateType string
		expectedName string
		expectedURL  string
	}{
		{"basic", "basic_scraper", "https://example.com"},
		{"ecommerce", "ecommerce_scraper", "https://example-shop.com/products"},
		{"news", "news_scraper", "https://example-news.com/articles"},
		{"unknown", "basic_scraper", "https://example.com"}, // Should default to basic
	}

	for _, tt := range tests {
		t.Run(tt.templateType, func(t *testing.T) {
			config := GenerateTemplate(tt.templateType)
			
			if config.Name != tt.expectedName {
				t.Errorf("expected name %s, got %s", tt.expectedName, config.Name)
			}
			
			if config.BaseURL != tt.expectedURL {
				t.Errorf("expected URL %s, got %s", tt.expectedURL, config.BaseURL)
			}
			
			if len(config.Fields) == 0 {
				t.Error("expected at least one field")
			}
			
			// Validate the generated config
			if err := config.Validate(); err != nil {
				t.Errorf("generated config is invalid: %v", err)
			}
		})
	}
}
