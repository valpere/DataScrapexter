// test/utils/test_utils.go
package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/valpere/DataScrapexter/internal/pipeline"
	"github.com/valpere/DataScrapexter/internal/scraper"
)

// CreateTestServer creates a test HTTP server with predefined HTML content
func CreateTestServer(htmlContent string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlContent)
	}))
}

// CreateSlowTestServer creates a test server that responds slowly
func CreateSlowTestServer(delay time.Duration, htmlContent string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlContent)
	}))
}

// CreateErrorTestServer creates a test server that returns an HTTP error
func CreateErrorTestServer(statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, "HTTP %d Error", statusCode)
	}))
}

// CreateUserAgentTestServer creates a server that captures user agents
func CreateUserAgentTestServer(capturedUserAgents *[]string, htmlContent string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*capturedUserAgents = append(*capturedUserAgents, r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlContent)
	}))
}

// CreateBasicHTML returns basic HTML for testing
func CreateBasicHTML() string {
	return `
	<html>
		<head><title>Test Page</title></head>
		<body>
			<h1>Test Title</h1>
			<p class="description">Test description</p>
			<span class="price">$99.99</span>
			<div class="rating">4.5/5</div>
			<ul class="items">
				<li class="item">Item 1</li>
				<li class="item">Item 2</li>
				<li class="item">Item 3</li>
			</ul>
		</body>
	</html>
	`
}

// CreateBasicEngineConfig creates a basic engine configuration for testing
func CreateBasicEngineConfig() *scraper.EngineConfig {
	return &scraper.EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		MaxConcurrency: 1,
		Fields: []scraper.FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
		ExtractionConfig: scraper.ExtractionConfig{
			StrictMode:      false,
			ContinueOnError: true,
		},
	}
}

// CreateTransformEngineConfig creates an engine config with transformations
func CreateTransformEngineConfig() *scraper.EngineConfig {
	return &scraper.EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		MaxConcurrency: 1,
		Fields: []scraper.FieldConfig{
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
				Required: false,
				Transform: []pipeline.TransformRule{
					{
						Type:        "regex",
						Pattern:     `\$([0-9,]+\.?[0-9]*)`,
						Replacement: "$1",
					},
				},
			},
		},
		ExtractionConfig: scraper.ExtractionConfig{
			StrictMode:      false,
			ContinueOnError: true,
		},
	}
}

// AssertFieldValue checks if a field has the expected value
func AssertFieldValue(data map[string]interface{}, fieldName string, expected interface{}) error {
	actual, exists := data[fieldName]
	if !exists {
		return fmt.Errorf("field %s not found in data", fieldName)
	}

	if actual != expected {
		return fmt.Errorf("field %s: expected %v, got %v", fieldName, expected, actual)
	}

	return nil
}

// AssertFieldExists checks if a field exists in the data
func AssertFieldExists(data map[string]interface{}, fieldName string) error {
	if _, exists := data[fieldName]; !exists {
		return fmt.Errorf("field %s not found in data", fieldName)
	}
	return nil
}

// AssertNoErrors checks that there are no errors in the result
func AssertNoErrors(errors []string) error {
	if len(errors) > 0 {
		return fmt.Errorf("unexpected errors: %v", errors)
	}
	return nil
}

// MockDataGenerator generates test data for various scenarios
type MockDataGenerator struct{}

// GenerateProductData creates mock product data
func (m MockDataGenerator) GenerateProductData() map[string]interface{} {
	return map[string]interface{}{
		"title":        "Gaming Laptop Pro",
		"price":        "1299.99",
		"rating":       "4.5",
		"description":  "High-performance gaming laptop",
		"availability": "in stock",
	}
}

// GenerateNewsData creates mock news article data
func (m MockDataGenerator) GenerateNewsData() map[string]interface{} {
	return map[string]interface{}{
		"headline":    "Breaking News Story",
		"author":      "John Doe",
		"date":        "2025-01-15",
		"content":     "This is the article content...",
		"category":    "technology",
	}
}
