// internal/scraper/extractor_test.go
package scraper

import (
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/valpere/DataScrapexter/internal/pipeline"
)

func TestNewFieldExtractor(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: true,
	}

	extractor := NewFieldExtractor(config, doc)

	if extractor == nil {
		t.Fatal("Extractor should not be nil")
	}

	if extractor.config.Name != config.Name {
		t.Fatalf("Expected config name '%s', got '%s'", config.Name, extractor.config.Name)
	}

	if extractor.document != doc {
		t.Fatal("Extractor should reference the provided document")
	}
}

func TestFieldExtractor_Extract_Text(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: true,
	}

	extractor := NewFieldExtractor(config, doc)
	ctx := context.Background()

	result, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	if result != "Test Title" {
		t.Errorf("Expected 'Test Title', got %v", result)
	}
}

func TestFieldExtractor_Extract_HTML(t *testing.T) {
	html := `<html><body><div class="content"><p>Test <strong>content</strong></p></div></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:     "content",
		Selector: ".content",
		Type:     "html",
		Required: true,
	}

	extractor := NewFieldExtractor(config, doc)
	ctx := context.Background()

	result, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("Expected string result, got %T", result)
	}

	expected := "<p>Test <strong>content</strong></p>"
	if resultStr != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resultStr)
	}
}

func TestFieldExtractor_Extract_Attribute(t *testing.T) {
	html := `<html><body><a href="https://example.com" class="link">Link</a></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:      "url",
		Selector:  "a",
		Type:      "attr",
		Attribute: "href",
		Required:  true,
	}

	extractor := NewFieldExtractor(config, doc)
	ctx := context.Background()

	result, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	if result != "https://example.com" {
		t.Errorf("Expected 'https://example.com', got %v", result)
	}
}

func TestFieldExtractor_Extract_List(t *testing.T) {
	html := `<html><body><ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:     "items",
		Selector: "li",
		Type:     "list",
		Required: true,
	}

	extractor := NewFieldExtractor(config, doc)
	ctx := context.Background()

	result, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	items, ok := result.([]string)
	if !ok {
		t.Fatalf("Expected []string, got %T", result)
	}

	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}

	expected := []string{"Item 1", "Item 2", "Item 3"}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s', got '%s'", expected[i], item)
		}
	}
}

func TestFieldExtractor_Extract_WithTransform(t *testing.T) {
	html := `<html><body><span class="price">$19.99</span></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:     "price",
		Selector: ".price",
		Type:     "text",
		Required: true,
		Transform: []pipeline.TransformRule{
			{
				Type:        "regex",
				Pattern:     `\$([0-9,]+\.\d*)`,
				Replacement: "$1",
			},
		},
	}

	extractor := NewFieldExtractor(config, doc)
	ctx := context.Background()

	result, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	if result != "19.99" {
		t.Errorf("Expected '19.99', got %v", result)
	}
}

func TestExtractionEngine_ExtractAll(t *testing.T) {
	html := `<html><body><h1>Test Title</h1><p>Test content</p></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
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

	engine := NewExtractionEngine(fields, config, doc)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	if !result.Success {
		t.Errorf("Expected extraction to succeed, got errors: %v", result.Errors)
	}

	if result.Data["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", result.Data["title"])
	}

	if result.Data["content"] != "Test content" {
		t.Errorf("Expected content 'Test content', got %v", result.Data["content"])
	}
}

func TestFieldExtractor_NumericTransformations(t *testing.T) {
	testCases := []struct {
		name       string
		html       string
		selector   string
		transforms []pipeline.TransformRule
		expected   string
		expectErr  bool
	}{
		{
			name:     "Extract Price String",
			html:     `<span class="price">$19.99</span>`,
			selector: ".price",
			transforms: []pipeline.TransformRule{
				{
					Type:        "regex",
					Pattern:     `\$([0-9,]+\.\d*)`,
					Replacement: "$1",
				},
			},
			expected:  "19.99",
			expectErr: false,
		},
		{
			name:     "Extract Count String",
			html:     `<span class="count">42 items</span>`,
			selector: ".count",
			transforms: []pipeline.TransformRule{
				{
					Type:        "regex",
					Pattern:     `(\d+) items`,
					Replacement: "$1",
				},
			},
			expected:  "42",
			expectErr: false,
		},
		{
			name:     "Clean Number String",
			html:     `<span class="number">  1,234  </span>`,
			selector: ".number",
			transforms: []pipeline.TransformRule{
				{Type: "trim"},
				{
					Type:        "regex",
					Pattern:     `,`,
					Replacement: "",
				},
			},
			expected:  "1234",
			expectErr: false,
		},
		{
			name:     "Extract Any Digits",
			html:     `<span class="digits">Price is 157 dollars</span>`,
			selector: ".digits",
			transforms: []pipeline.TransformRule{
				{
					Type:        "regex",
					Pattern:     `.*?(\d+).*`,
					Replacement: "$1",
				},
			},
			expected:  "157",
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			config := FieldConfig{
				Name:      "test_field",
				Selector:  tc.selector,
				Type:      "text",
				Required:  true,
				Transform: tc.transforms,
			}

			extractor := NewFieldExtractor(config, doc)
			ctx := context.Background()

			result, err := extractor.Extract(ctx)

			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result != tc.expected {
					t.Errorf("Expected %v, got %v", tc.expected, result)
				}
			}
		})
	}
}

func TestFieldExtractor_ComplexTransformChain(t *testing.T) {
	html := `<html><body><span class="price">  Price: $1,299.99 USD  </span></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:     "price",
		Selector: ".price",
		Type:     "text",
		Required: true,
		Transform: []pipeline.TransformRule{
			{Type: "trim"},
			{
				Type:        "regex",
				Pattern:     `Price: \$([0-9,]+\.\d+)\s+USD`,
				Replacement: "$1",
			},
			{
				Type:        "regex",
				Pattern:     `,`,
				Replacement: "",
			},
		},
	}

	extractor := NewFieldExtractor(config, doc)
	ctx := context.Background()

	result, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "1299.99"
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
