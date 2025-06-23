// test/utils/test_utils.go
package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valpere/DataScrapexter/internal/pipeline"
	"github.com/valpere/DataScrapexter/internal/scraper"
)

// TestServer provides a mock HTTP server for testing
type TestServer struct {
	Server *httptest.Server
	Routes map[string]string
}

// NewTestServer creates a new test server with predefined routes
func NewTestServer(routes map[string]string) *TestServer {
	ts := &TestServer{
		Routes: routes,
	}

	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}

		if content, exists := ts.Routes[path]; exists {
			fmt.Fprint(w, content)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Not Found")
		}
	}))

	return ts
}

// Close shuts down the test server
func (ts *TestServer) Close() {
	ts.Server.Close()
}

// URL returns the base URL of the test server
func (ts *TestServer) URL() string {
	return ts.Server.URL
}

// MockHTMLTemplates provides common HTML templates for testing
var MockHTMLTemplates = map[string]string{
	"ecommerce_product": `
		<div class="product">
			<h1 class="title">{{.Title}}</h1>
			<div class="price">{{.Price}}</div>
			<div class="description">{{.Description}}</div>
			<div class="stock">{{.Stock}}</div>
		</div>
	`,
	"news_article": `
		<article class="article">
			<h1 class="headline">{{.Headline}}</h1>
			<div class="author">{{.Author}}</div>
			<time class="date">{{.Date}}</time>
			<div class="content">{{.Content}}</div>
		</article>
	`,
	"pagination_next": `
		<div class="pagination">
			<a href="{{.NextURL}}" class="next">Next</a>
		</div>
	`,
	"pagination_numbered": `
		<div class="pagination">
			{{range .Pages}}
				<a href="?page={{.}}" class="page">{{.}}</a>
			{{end}}
		</div>
	`,
}

// CreateMockHTML creates HTML content from templates
func CreateMockHTML(template string, data map[string]interface{}) string {
	content := MockHTMLTemplates[template]
	if content == "" {
		return ""
	}

	// Simple template replacement (for testing purposes)
	for key, value := range data {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		content = strings.ReplaceAll(content, placeholder, fmt.Sprintf("%v", value))
	}

	return content
}

// AssertTransformResult checks if a transformation produces expected output
func AssertTransformResult(t *testing.T, rule pipeline.TransformRule, input, expected string) {
	t.Helper()
	
	result, err := rule.Transform(nil, input)
	if err != nil {
		t.Errorf("transformation failed: %v", err)
		return
	}

	if result != expected {
		t.Errorf("transform %s: expected %q, got %q", rule.Type, expected, result)
	}
}

// AssertFieldExtraction checks if field extraction produces expected results
func AssertFieldExtraction(t *testing.T, config scraper.FieldConfig, html string, expected interface{}) {
	t.Helper()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
		return
	}

	// Extract value using selector
	var extractedValue interface{}
	selection := doc.Find(config.Selector)
	if selection.Length() > 0 {
		extractedValue = selection.Text()
	}

	// Apply transformations if any
	if len(config.Transform) > 0 && extractedValue != nil {
		if str, ok := extractedValue.(string); ok {
			transformList := pipeline.TransformList(config.Transform)
			transformed, err := transformList.Apply(nil, str)
			if err != nil {
				t.Errorf("transformation failed: %v", err)
				return
			}
			extractedValue = transformed
		}
	}

	if extractedValue != expected {
		t.Errorf("field %s: expected %v, got %v", config.Name, expected, extractedValue)
	}
}

// CreateTestEngineConfig creates a standard test configuration
func CreateTestEngineConfig() *scraper.EngineConfig {
	return &scraper.EngineConfig{
		Fields: []scraper.FieldConfig{
			{
				Name:     "title",
				Selector: "h1, .title",
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
					{Type: "parse_float"},
				},
			},
			{
				Name:     "description",
				Selector: ".description, .desc",
				Type:     "text",
				Required: false,
				Transform: []pipeline.TransformRule{
					{Type: "remove_html"},
					{Type: "trim"},
				},
			},
		},
	}
}

// CreateTestPipelineConfig creates a standard test pipeline configuration
func CreateTestPipelineConfig() *pipeline.PipelineConfig {
	return &pipeline.PipelineConfig{
		BufferSize:    100,
		WorkerCount:   5,
		Timeout:       10 * time.Second,
		EnableMetrics: true,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	}
}

// TimeoutContext creates a context with timeout for tests
func TimeoutContext(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline, ok := t.Deadline()
	if ok {
		timeoutDeadline := time.Now().Add(timeout)
		if deadline.Before(timeoutDeadline) {
			t.Logf("Test timeout set to %v", timeout)
		}
	}
}

// AssertNoError checks that no error occurred
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError checks that an error occurred
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error but got none", msg)
	}
}

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertContains checks if a string contains a substring
func AssertContains(t *testing.T, haystack, needle, msg string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: %q does not contain %q", msg, haystack, needle)
	}
}

// AssertNotEmpty checks if a string is not empty
func AssertNotEmpty(t *testing.T, value, msg string) {
	t.Helper()
	if strings.TrimSpace(value) == "" {
		t.Errorf("%s: value is empty", msg)
	}
}

// MockData provides test data structures
type MockData struct {
	Products []Product `json:"products"`
	Articles []Article `json:"articles"`
}

type Product struct {
	Title       string  `json:"title"`
	Price       string  `json:"price"`
	Description string  `json:"description"`
	Stock       string  `json:"stock"`
	Rating      float64 `json:"rating"`
}

type Article struct {
	Headline string `json:"headline"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	Content  string `json:"content"`
	Tags     []string `json:"tags"`
}

// GenerateMockProducts creates test product data
func GenerateMockProducts(count int) []Product {
	products := make([]Product, count)
	for i := 0; i < count; i++ {
		products[i] = Product{
			Title:       fmt.Sprintf("Product %d", i+1),
			Price:       fmt.Sprintf("$%d.99", 100+i*10),
			Description: fmt.Sprintf("Description for product %d", i+1),
			Stock:       "In Stock",
			Rating:      4.0 + float64(i%5)*0.2,
		}
	}
	return products
}

// GenerateMockArticles creates test article data
func GenerateMockArticles(count int) []Article {
	articles := make([]Article, count)
	for i := 0; i < count; i++ {
		articles[i] = Article{
			Headline: fmt.Sprintf("Breaking News %d", i+1),
			Author:   fmt.Sprintf("Author %d", i+1),
			Date:     fmt.Sprintf("2025-06-%02d", (i%28)+1),
			Content:  fmt.Sprintf("This is the content of article %d with important information.", i+1),
			Tags:     []string{"news", "breaking", fmt.Sprintf("tag%d", i+1)},
		}
	}
	return articles
}

// CreateProductHTML generates HTML for a product
func CreateProductHTML(product Product) string {
	return fmt.Sprintf(`
		<div class="product">
			<h1 class="title">%s</h1>
			<div class="price">%s</div>
			<div class="description">%s</div>
			<div class="stock">%s</div>
			<div class="rating">%.1f/5</div>
		</div>
	`, product.Title, product.Price, product.Description, product.Stock, product.Rating)
}

// CreateArticleHTML generates HTML for an article
func CreateArticleHTML(article Article) string {
	tags := ""
	for _, tag := range article.Tags {
		tags += fmt.Sprintf(`<span class="tag">%s</span>`, tag)
	}

	return fmt.Sprintf(`
		<article class="article">
			<h1 class="headline">%s</h1>
			<div class="author">%s</div>
			<time class="date">%s</time>
			<div class="content">%s</div>
			<div class="tags">%s</div>
		</article>
	`, article.Headline, article.Author, article.Date, article.Content, tags)
}

// BenchmarkHelper provides utilities for benchmark tests
type BenchmarkHelper struct {
	Engine            *scraper.ScrapingEngine
	PaginationManager *scraper.PaginationManager
	TestData          map[string]interface{}
	TestHTML          string
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper() *BenchmarkHelper {
	config := CreateTestEngineConfig()
	engine := scraper.NewScrapingEngine(config)

	paginationConfig := scraper.PaginationConfig{
		Type:     "next_button",
		Selector: ".next",
		MaxPages: 10,
	}
	paginationManager, _ := scraper.NewPaginationManager(paginationConfig)

	product := GenerateMockProducts(1)[0]
	testHTML := CreateProductHTML(product)

	testData := map[string]interface{}{
		"title":       product.Title,
		"price":       product.Price,
		"description": product.Description,
		"stock":       product.Stock,
	}

	return &BenchmarkHelper{
		Engine:            engine,
		PaginationManager: paginationManager,
		TestData:          testData,
		TestHTML:          testHTML,
	}
}

// SetupTestEnvironment prepares a complete test environment
func SetupTestEnvironment(t *testing.T) (*TestServer, *scraper.ScrapingEngine, *scraper.PaginationManager) {
	t.Helper()

	// Create test data
	products := GenerateMockProducts(3)
	
	routes := map[string]string{
		"/":       CreateProductHTML(products[0]) + `<a href="/page/2" class="next">Next</a>`,
		"/page/2": CreateProductHTML(products[1]) + `<a href="/page/3" class="next">Next</a>`,
		"/page/3": CreateProductHTML(products[2]) + `<span class="next disabled">End</span>`,
	}

	// Create test server
	server := NewTestServer(routes)

	// Create scraping engine
	config := CreateTestEngineConfig()
	engine := scraper.NewScrapingEngine(config)

	// Create pagination manager
	paginationConfig := scraper.PaginationConfig{
		Type:     "next_button",
		Selector: ".next",
		MaxPages: 5,
	}
	paginationManager, err := scraper.NewPaginationManager(paginationConfig)
	AssertNoError(t, err, "failed to create pagination manager")

	return server, engine, paginationManager
}

// CleanupTestEnvironment cleans up test resources
func CleanupTestEnvironment(server *TestServer) {
	if server != nil {
		server.Close()
	}
}
