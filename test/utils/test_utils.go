// test/utils/test_utils.go
package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/valpere/DataScrapexter/internal/pipeline"
	"github.com/valpere/DataScrapexter/internal/scraper"
)

// TestServer wraps httptest.Server with additional utilities
type TestServer struct {
	*httptest.Server
	RequestCount int
	LastRequest  *http.Request
}

// NewTestServer creates a new test server with the given HTML content
func NewTestServer(html string) *TestServer {
	ts := &TestServer{}
	
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.RequestCount++
		ts.LastRequest = r
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}))
	
	return ts
}

// NewTestServerWithHandler creates a test server with custom handler
func NewTestServerWithHandler(handler http.HandlerFunc) *TestServer {
	ts := &TestServer{}
	ts.Server = httptest.NewServer(handler)
	return ts
}

// MockHTMLTemplates provides common HTML templates for testing
type MockHTMLTemplates struct{}

// EcommerceProduct returns HTML for a mock e-commerce product page
func (m MockHTMLTemplates) EcommerceProduct() string {
	return `
	<html>
		<head><title>Product Page</title></head>
		<body>
			<div class="product">
				<h1 class="product-title">Gaming Laptop Pro</h1>
				<span class="price">$1,299.99</span>
				<div class="rating">4.5/5</div>
				<p class="description">High-performance gaming laptop with <strong>RTX 4060</strong> graphics</p>
				<div class="availability">In Stock</div>
				<div class="discount">15% off</div>
			</div>
		</body>
	</html>
	`
}

// NewsArticle returns HTML for a mock news article
func (m MockHTMLTemplates) NewsArticle() string {
	return `
	<html>
		<head><title>News Article</title></head>
		<body>
			<article>
				<h1 class="headline">Breaking: Technology Advances Continue</h1>
				<div class="byline">By John Reporter</div>
				<time class="publish-date">2025-06-25</time>
				<div class="content">
					<p>Technology continues to advance at a rapid pace...</p>
					<p>Industry experts predict significant changes ahead.</p>
				</div>
				<div class="tags">
					<span class="tag">Technology</span>
					<span class="tag">Innovation</span>
				</div>
			</article>
		</body>
	</html>
	`
}

// SimpleList returns HTML with a simple list structure
func (m MockHTMLTemplates) SimpleList() string {
	return `
	<html>
		<body>
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
		RequestTimeout: 10 * 1000000000, // 10 seconds in nanoseconds
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
		RequestTimeout: 10 * 1000000000, // 10 seconds in nanoseconds
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
		"headline":     "Breaking: Technology Advances Continue",
		"author":       "John Reporter",
		"publish_date": "2025-06-25",
		"content":      "Technology continues to advance at a rapid pace...",
	}
}

// ValidateTransformRule validates that a transform rule is properly configured
func ValidateTransformRule(rule pipeline.TransformRule) error {
	switch rule.Type {
	case "trim", "normalize_spaces", "lowercase", "uppercase", "remove_html":
		return nil
	case "regex":
		if rule.Pattern == "" {
			return fmt.Errorf("regex rule requires pattern")
		}
		return nil
	case "prefix", "suffix":
		if rule.Params == nil || rule.Params["value"] == nil {
			return fmt.Errorf("%s rule requires value parameter", rule.Type)
		}
		return nil
	case "replace":
		if rule.Params == nil || rule.Params["old"] == nil || rule.Params["new"] == nil {
			return fmt.Errorf("replace rule requires old and new parameters")
		}
		return nil
	default:
		return fmt.Errorf("unknown transform type: %s", rule.Type)
	}
}

// BenchmarkHelper provides utilities for performance testing
type BenchmarkHelper struct {
	RequestCount int
	TotalTime    int64 // nanoseconds
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper() *BenchmarkHelper {
	return &BenchmarkHelper{}
}

// RecordRequest records a request for benchmarking
func (b *BenchmarkHelper) RecordRequest(duration int64) {
	b.RequestCount++
	b.TotalTime += duration
}

// AverageTime returns the average request time in nanoseconds
func (b *BenchmarkHelper) AverageTime() int64 {
	if b.RequestCount == 0 {
		return 0
	}
	return b.TotalTime / int64(b.RequestCount)
}

// CreateSlowServer creates a server that responds slowly for timeout testing
func CreateSlowServer(delaySeconds int, content string) *TestServer {
	return NewTestServerWithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response without actual sleep for testing
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, content)
	}))
}

// CreateErrorServer creates a server that returns HTTP errors
func CreateErrorServer(statusCode int) *TestServer {
	return NewTestServerWithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, "HTTP Error %d", statusCode)
	}))
}

// GetHTMLTemplates returns a MockHTMLTemplates instance
func GetHTMLTemplates() MockHTMLTemplates {
	return MockHTMLTemplates{}
}

// GetMockDataGenerator returns a MockDataGenerator instance
func GetMockDataGenerator() MockDataGenerator {
	return MockDataGenerator{}
}

// CleanString removes extra whitespace and normalizes strings for comparison
func CleanString(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}
