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

	if !strings.Contains(resultStr, "<strong>content</strong>") {
		t.Errorf("Expected HTML content with tags, got %v", resultStr)
	}
}

func TestFieldExtractor_Extract_Attribute(t *testing.T) {
	html := `<html><body><a href="https://example.com" class="link">Test Link</a></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:      "link_url",
		Selector:  "a.link",
		Type:      "attribute",
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

func TestFieldExtractor_Extract_WithTransformation(t *testing.T) {
	html := `<html><body><span class="price">  $1,299.99  </span></body></html>`
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
				Pattern:     `\$([0-9,]+\.?\d*)`,
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

	if result != "1,299.99" {
		t.Errorf("Expected '1,299.99', got %v", result)
	}
}

func TestFieldExtractor_Extract_Array(t *testing.T) {
	html := `
	<html>
		<body>
			<ul class="features">
				<li>Feature 1</li>
				<li>Feature 2</li>
				<li>Feature 3</li>
			</ul>
		</body>
	</html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:     "features",
		Selector: ".features li",
		Type:     "array",
		Required: true,
	}

	extractor := NewFieldExtractor(config, doc)
	ctx := context.Background()

	result, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	features, ok := result.([]interface{})
	if !ok {
		t.Fatalf("Expected array result, got %T", result)
	}

	if len(features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(features))
	}

	expectedFeatures := []string{"Feature 1", "Feature 2", "Feature 3"}
	for i, expected := range expectedFeatures {
		if features[i] != expected {
			t.Errorf("Feature %d: expected '%s', got %v", i, expected, features[i])
		}
	}
}

func TestFieldExtractor_Extract_RequiredFieldMissing(t *testing.T) {
	html := `<html><body><p>Some content</p></body></html>`
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

	_, err = extractor.Extract(ctx)
	if err == nil {
		t.Error("Expected error for missing required field")
	}

	if err != nil && !strings.Contains(err.Error(), "required field") {
		t.Errorf("Expected error message to mention required field, got: %v", err)
	}
}

func TestFieldExtractor_Extract_OptionalFieldMissing(t *testing.T) {
	html := `<html><body><p>Some content</p></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := FieldConfig{
		Name:     "description",
		Selector: ".description",
		Type:     "text",
		Required: false,
		Default:  "No description available",
	}

	extractor := NewFieldExtractor(config, doc)
	ctx := context.Background()

	result, err := extractor.Extract(ctx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	if result != "No description available" {
		t.Errorf("Expected default value 'No description available', got %v", result)
	}
}

func TestFieldExtractor_Extract_InvalidTransformation(t *testing.T) {
	html := `<html><body><span class="price">not a number</span></body></html>`
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
			{Type: "parse_float"},
		},
	}

	extractor := NewFieldExtractor(config, doc)
	ctx := context.Background()

	_, err = extractor.Extract(ctx)
	if err == nil {
		t.Error("Expected error for invalid transformation")
	}

	if err != nil && !strings.Contains(err.Error(), "transformation failed") {
		t.Errorf("Expected transformation error, got: %v", err)
	}
}

func TestNewExtractionEngine(t *testing.T) {
	html := `<html><body><h1>Test</h1></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := ExtractionConfig{
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
		StrictMode:      false,
		ContinueOnError: true,
	}

	engine := NewExtractionEngine(config, doc)

	if engine == nil {
		t.Fatal("Extraction engine should not be nil")
	}

	if len(engine.extractors) != 1 {
		t.Errorf("Expected 1 extractor, got %d", len(engine.extractors))
	}

	if engine.document != doc {
		t.Fatal("Engine should reference the provided document")
	}
}

func TestExtractionEngine_ExtractAll(t *testing.T) {
	html := `
	<html>
		<body>
			<h1>Test Title</h1>
			<p class="content">Test content</p>
			<span class="price">$99.99</span>
			<div class="rating">4.5</div>
		</body>
	</html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := ExtractionConfig{
		Fields: []FieldConfig{
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
				Type:     "text",
				Required: false,
				Transform: []pipeline.TransformRule{
					{
						Type:        "regex",
						Pattern:     `\$([0-9,]+\.?\d*)`,
						Replacement: "$1",
					},
				},
			},
		},
		StrictMode:      false,
		ContinueOnError: true,
	}

	engine := NewExtractionEngine(config, doc)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	if !result.Success {
		t.Fatalf("Expected successful extraction, got errors: %v", result.Errors)
	}

	if result.Data["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", result.Data["title"])
	}

	if result.Data["content"] != "Test content" {
		t.Errorf("Expected content 'Test content', got %v", result.Data["content"])
	}

	if result.Data["price"] != "99.99" {
		t.Errorf("Expected price '99.99', got %v", result.Data["price"])
	}

	if result.Metadata.ExtractedFields != 3 {
		t.Errorf("Expected 3 extracted fields, got %d", result.Metadata.ExtractedFields)
	}

	if result.Metadata.TotalFields != 3 {
		t.Errorf("Expected 3 total fields, got %d", result.Metadata.TotalFields)
	}
}

func TestExtractionEngine_ExtractAll_WithErrors(t *testing.T) {
	html := `<html><body><p>Some content without title</p></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := ExtractionConfig{
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
			{
				Name:     "description",
				Selector: "p",
				Type:     "text",
				Required: false,
			},
		},
		StrictMode:      false,
		ContinueOnError: true,
	}

	engine := NewExtractionEngine(config, doc)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	if result.Success {
		t.Error("Expected extraction to fail due to missing required field")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected at least one error for missing required field")
	}

	if result.Data["description"] != "Some content without title" {
		t.Errorf("Expected description to be extracted, got %v", result.Data["description"])
	}

	if result.Metadata.FailedFields != 1 {
		t.Errorf("Expected 1 failed field, got %d", result.Metadata.FailedFields)
	}

	if result.Metadata.ExtractedFields != 1 {
		t.Errorf("Expected 1 extracted field, got %d", result.Metadata.ExtractedFields)
	}
}

func TestExtractionEngine_StrictMode(t *testing.T) {
	html := `<html><body><p>Content without title</p></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	config := ExtractionConfig{
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
		StrictMode:      true,
		ContinueOnError: false,
	}

	engine := NewExtractionEngine(config, doc)
	ctx := context.Background()

	result := engine.ExtractAll(ctx)

	if result.Success {
		t.Error("Expected strict mode to fail on any error")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors in strict mode")
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
					Pattern:     `\$([0-9,]+\.?\d*)`,
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
				Pattern:     `Price: \$([0-9,]+\.?\d*) USD`,
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
		t.Fatalf("Extraction failed: %v", err)
	}

	if result != "1299.99" {
		t.Errorf("Expected '1299.99', got %v", result)
	}
}
