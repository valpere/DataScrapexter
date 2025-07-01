// test/integration_test.go
package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valpere/DataScrapexter/internal/scraper"
)

func TestBasicScraping(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
			<head><title>Test Page</title></head>
			<body>
				<h1>Test Title</h1>
				<p class="content">Test content here</p>
				<span class="price">$19.99</span>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	// Create configuration
	config := &scraper.EngineConfig{
		Fields: []scraper.FieldConfig{
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
			},
		},
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
	}

	// Create scraping engine
	engine, err := scraper.NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}
	defer engine.Close()

	// Perform scraping
	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL)

	if err != nil {
		t.Fatalf("Scraping failed: %v", err)
	}

	// Verify results
	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	if !result.Success {
		t.Errorf("Expected successful scraping, got errors: %v", result.Errors)
	}

	if result.Data["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", result.Data["title"])
	}

	if result.Data["content"] != "Test content here" {
		t.Errorf("Expected content 'Test content here', got %v", result.Data["content"])
	}

	if result.Data["price"] != "$19.99" {
		t.Errorf("Expected price '$19.99', got %v", result.Data["price"])
	}
}

func TestEcommerceScraping(t *testing.T) {
	// Create a test HTTP server with ecommerce-like content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
			<head><title>Product Page</title></head>
			<body>
				<div class="product">
					<h1 class="product-title">Amazing Product</h1>
					<div class="price-container">
						<span class="price">$99.99</span>
						<span class="original-price">$149.99</span>
					</div>
					<div class="description">
						<p>This is an amazing product with great features.</p>
					</div>
					<div class="availability">In Stock</div>
					<img src="/product-image.jpg" alt="Product Image" class="product-image">
					<div class="specifications">
						<ul>
							<li>Feature 1</li>
							<li>Feature 2</li>
							<li>Feature 3</li>
						</ul>
					</div>
				</div>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	// Create ecommerce configuration
	config := &scraper.EngineConfig{
		Fields: []scraper.FieldConfig{
			{
				Name:     "product_name",
				Selector: ".product-title",
				Type:     "text",
				Required: true,
			},
			{
				Name:     "price",
				Selector: ".price",
				Type:     "text",
				Required: true,
			},
			{
				Name:     "original_price",
				Selector: ".original-price",
				Type:     "text",
				Required: false,
			},
			{
				Name:     "description",
				Selector: ".description p",
				Type:     "text",
				Required: false,
			},
			{
				Name:     "availability",
				Selector: ".availability",
				Type:     "text",
				Required: false,
			},
			{
				Name:      "image_url",
				Selector:  ".product-image",
				Type:      "attr",
				Attribute: "src",
				Required:  false,
			},
			{
				Name:     "features",
				Selector: ".specifications li",
				Type:     "list",
				Required: false,
			},
		},
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
	}

	// Create scraping engine
	engine, err := scraper.NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}
	defer engine.Close()

	// Perform scraping
	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL)

	if err != nil {
		t.Fatalf("Scraping failed: %v", err)
	}

	// Verify results
	if !result.Success {
		t.Errorf("Expected successful scraping, got errors: %v", result.Errors)
	}

	if result.Data["product_name"] != "Amazing Product" {
		t.Errorf("Expected product name 'Amazing Product', got %v", result.Data["product_name"])
	}

	if result.Data["price"] != "$99.99" {
		t.Errorf("Expected price '$99.99', got %v", result.Data["price"])
	}

	if result.Data["image_url"] != "/product-image.jpg" {
		t.Errorf("Expected image URL '/product-image.jpg', got %v", result.Data["image_url"])
	}

	// Check list type field
	features, ok := result.Data["features"].([]string)
	if !ok {
		t.Errorf("Expected features to be []string, got %T", result.Data["features"])
	} else if len(features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(features))
	}
}

func TestScrapingWithErrors(t *testing.T) {
	// Create a test HTTP server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	// Create configuration
	config := &scraper.EngineConfig{
		Fields: []scraper.FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 5 * time.Second,
		RetryAttempts:  1,
	}

	// Create scraping engine
	engine, err := scraper.NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}
	defer engine.Close()

	// Perform scraping
	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL)

	if err != nil {
		t.Fatalf("Scraping failed: %v", err)
	}

	// Verify error handling
	if result.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", result.StatusCode)
	}

	// The scraper should still attempt to parse the response even for 404
	// and should report errors for missing required fields
	if result.Success {
		t.Error("Expected scraping to fail due to missing required fields")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors for missing required fields")
	}
}

func TestScrapingTimeout(t *testing.T) {
	// Create a test HTTP server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than timeout
		w.Write([]byte("<html><body><h1>Delayed Response</h1></body></html>"))
	}))
	defer server.Close()

	// Create configuration with short timeout
	config := &scraper.EngineConfig{
		Fields: []scraper.FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 1 * time.Second, // Short timeout
		RetryAttempts:  1,
	}

	// Create scraping engine
	engine, err := scraper.NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}
	defer engine.Close()

	// Perform scraping
	ctx := context.Background()
	_, err = engine.Scrape(ctx, server.URL)

	// Should timeout
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestMultipleFieldTypes(t *testing.T) {
	// Create a test HTTP server with various field types
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
			<head><title>Test Page</title></head>
			<body>
				<h1>Main Title</h1>
				<div class="content">
					<p>Paragraph 1</p>
					<p>Paragraph 2</p>
				</div>
				<a href="https://example.com" class="main-link">Example Link</a>
				<img src="/image.jpg" alt="Test Image" class="main-image">
				<div class="metadata" data-category="test" data-priority="high">
					<span>Metadata content</span>
				</div>
				<ul class="tags">
					<li>tag1</li>
					<li>tag2</li>
					<li>tag3</li>
				</ul>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	// Create configuration with various field types
	config := &scraper.EngineConfig{
		Fields: []scraper.FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
			{
				Name:     "content_html",
				Selector: ".content",
				Type:     "html",
				Required: false,
			},
			{
				Name:      "link_url",
				Selector:  ".main-link",
				Type:      "attr",
				Attribute: "href",
				Required:  false,
			},
			{
				Name:      "image_alt",
				Selector:  ".main-image",
				Type:      "attr",
				Attribute: "alt",
				Required:  false,
			},
			{
				Name:      "category",
				Selector:  ".metadata",
				Type:      "attr",
				Attribute: "data-category",
				Required:  false,
			},
			{
				Name:     "tags",
				Selector: ".tags li",
				Type:     "list",
				Required: false,
			},
		},
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
	}

	// Create scraping engine
	engine, err := scraper.NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}
	defer engine.Close()

	// Perform scraping
	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL)

	if err != nil {
		t.Fatalf("Scraping failed: %v", err)
	}

	// Verify results
	if !result.Success {
		t.Errorf("Expected successful scraping, got errors: %v", result.Errors)
	}

	// Check text extraction
	if result.Data["title"] != "Main Title" {
		t.Errorf("Expected title 'Main Title', got %v", result.Data["title"])
	}

	// Check HTML extraction
	contentHTML, ok := result.Data["content_html"].(string)
	if !ok {
		t.Errorf("Expected content_html to be string, got %T", result.Data["content_html"])
	} else if !strings.Contains(contentHTML, "<p>Paragraph 1</p>") {
		t.Errorf("Expected HTML content to contain paragraphs, got %v", contentHTML)
	}

	// Check attribute extraction
	if result.Data["link_url"] != "https://example.com" {
		t.Errorf("Expected link URL 'https://example.com', got %v", result.Data["link_url"])
	}

	if result.Data["image_alt"] != "Test Image" {
		t.Errorf("Expected image alt 'Test Image', got %v", result.Data["image_alt"])
	}

	if result.Data["category"] != "test" {
		t.Errorf("Expected category 'test', got %v", result.Data["category"])
	}

	// Check list extraction
	tags, ok := result.Data["tags"].([]string)
	if !ok {
		t.Errorf("Expected tags to be []string, got %T", result.Data["tags"])
	} else if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	} else {
		expectedTags := []string{"tag1", "tag2", "tag3"}
		for i, expected := range expectedTags {
			if i < len(tags) && tags[i] != expected {
				t.Errorf("Expected tag[%d] to be '%s', got '%s'", i, expected, tags[i])
			}
		}
	}
}
