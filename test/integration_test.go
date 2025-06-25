// test/integration_test.go
package test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valpere/DataScrapexter/internal/pipeline"
	"github.com/valpere/DataScrapexter/internal/scraper"
)

// TestTransformationIntegrationFix validates that field transformations are properly applied
func TestTransformationIntegrationFix(t *testing.T) {
	ctx := context.Background()

	// Test HTML with data that needs transformation
	testHTML := `
	<html>
		<body>
			<h1>  Product Title With Extra Spaces  </h1>
			<span class="price">$1,299.99</span>
			<div class="description">This is a <b>great</b> product with HTML tags</div>
			<span class="discount">25%</span>
		</body>
	</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testHTML)
	}))
	defer server.Close()

	// Create engine config with field-level transformations
	config := &scraper.EngineConfig{
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		UserAgents:     []string{"TestAgent/1.0"},
		Fields: []scraper.FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
				Transform: []pipeline.TransformRule{
					{Type: "trim"},
					{Type: "normalize_spaces"},
					{Type: "lowercase"},
				},
			},
			{
				Name:     "price",
				Selector: ".price",
				Type:     "text",
				Required: true,
				Transform: []pipeline.TransformRule{
					{
						Type:        "regex",
						Pattern:     `\$([0-9,]+\.?[0-9]*)`,
						Replacement: "$1",
					},
				},
			},
			{
				Name:     "description",
				Selector: ".description",
				Type:     "text",
				Required: true,
				Transform: []pipeline.TransformRule{
					{Type: "remove_html"},
					{Type: "trim"},
					{Type: "normalize_spaces"},
				},
			},
			{
				Name:     "discount",
				Selector: ".discount",
				Type:     "text",
				Required: true,
				Transform: []pipeline.TransformRule{
					{
						Type:        "regex",
						Pattern:     `([0-9]+)%`,
						Replacement: "$1",
					},
					{Type: "parse_int"},
				},
			},
		},
		ExtractionConfig: scraper.ExtractionConfig{
			StrictMode:      false,
			ContinueOnError: false,
		},
	}

	// Create scraping engine
	engine, err := scraper.NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}
	defer engine.Close()

	// Perform scraping
	result, err := engine.Scrape(ctx, server.URL)
	if err != nil {
		t.Fatalf("Scraping failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful extraction, got errors: %v", result.Errors)
	}

	// Validate all transformed results with strict comparison
	expectedTitle := "product title with extra spaces"
	if result.Data["title"] != expectedTitle {
		t.Errorf("Title transformation failed. Expected '%s', got '%v'", expectedTitle, result.Data["title"])
		return
	}

	expectedPrice := "1,299.99"
	if result.Data["price"] != expectedPrice {
		t.Errorf("Price transformation failed. Expected '%s', got '%v'", expectedPrice, result.Data["price"])
		return
	}

	expectedDescription := "This is a great product with HTML tags"
	if result.Data["description"] != expectedDescription {
		t.Errorf("Description transformation failed. Expected '%s', got '%v'", expectedDescription, result.Data["description"])
		return
	}

	// Fixed type assertion for discount comparison
	expectedDiscount := 25
	if discount, ok := result.Data["discount"].(int); !ok || discount != expectedDiscount {
		t.Errorf("Discount transformation failed. Expected %d, got %v (type: %T)", expectedDiscount, result.Data["discount"], result.Data["discount"])
		return
	}

	t.Logf("✅ Transformation integration test passed successfully")
	t.Logf("   Title: %v", result.Data["title"])
	t.Logf("   Price: %v", result.Data["price"])
	t.Logf("   Description: %v", result.Data["description"])
	t.Logf("   Discount: %v", result.Data["discount"])
}

// TestCompleteWorkflowIntegration validates the entire workflow with transformations and timeouts
func TestCompleteWorkflowIntegration(t *testing.T) {
	ctx := context.Background()

	// Test HTML with complex data requiring transformations
	testHTML := `
	<html>
		<head><title>E-commerce Site</title></head>
		<body>
			<div class="product">
				<h1>   Super Gaming Laptop Pro Max   </h1>
				<span class="price">$2,499.99</span>
				<div class="rating">4.8/5</div>
				<p class="description">
					The <strong>ultimate</strong> gaming machine with <em>amazing</em> performance!
					Features include high-end GPU and fast SSD.
				</p>
				<div class="availability">In Stock</div>
			</div>
		</body>
	</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testHTML)
	}))
	defer server.Close()

	config := &scraper.EngineConfig{
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		UserAgents:     []string{"TestAgent/1.0"},
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
				Required: true,
				Transform: []pipeline.TransformRule{
					{
						Type:        "regex",
						Pattern:     `\$([0-9,]+\.?[0-9]*)`,
						Replacement: "$1",
					},
				},
			},
			{
				Name:     "rating",
				Selector: ".rating",
				Type:     "text",
				Required: true,
				Transform: []pipeline.TransformRule{
					{
						Type:        "regex",
						Pattern:     `^(\d+\.\d+)/\d+$`,
						Replacement: "$1",
					},
					{Type: "parse_float"},
				},
			},
			{
				Name:     "description",
				Selector: ".description",
				Type:     "text",
				Required: true,
				Transform: []pipeline.TransformRule{
					{Type: "remove_html"},
					{Type: "trim"},
					{Type: "normalize_spaces"},
				},
			},
			{
				Name:     "availability",
				Selector: ".availability",
				Type:     "text",
				Required: true,
				Transform: []pipeline.TransformRule{
					{Type: "lowercase"},
					{Type: "replace", Params: map[string]interface{}{"old": " ", "new": " "}},
				},
			},
		},
		ExtractionConfig: scraper.ExtractionConfig{
			StrictMode:      false,
			ContinueOnError: false,
		},
	}

	// Create scraping engine and measure time
	startTime := time.Now()
	engine, err := scraper.NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}
	defer engine.Close()

	// Perform scraping
	result, err := engine.Scrape(ctx, server.URL)
	totalDuration := time.Since(startTime)
	
	if err != nil {
		t.Fatalf("Scraping failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful extraction, got errors: %v", result.Errors)
	}

	// Validate all extracted data with type-safe comparisons
	expectedTitle := "Super Gaming Laptop Pro Max"
	if result.Data["title"] != expectedTitle {
		t.Errorf("Title: expected '%s', got '%v'", expectedTitle, result.Data["title"])
		return
	}

	expectedPrice := "2,499.99"
	if result.Data["price"] != expectedPrice {
		t.Errorf("Price: expected '%s', got '%v'", expectedPrice, result.Data["price"])
		return
	}

	expectedRating := 4.8
	if rating, ok := result.Data["rating"].(float64); !ok || rating != expectedRating {
		t.Errorf("Rating: expected %.1f, got %v (type: %T)", expectedRating, result.Data["rating"], result.Data["rating"])
		return
	}

	expectedDescription := "The ultimate gaming machine with amazing performance! Features include high-end GPU and fast SSD."
	if result.Data["description"] != expectedDescription {
		t.Errorf("Description: expected '%s', got '%v'", expectedDescription, result.Data["description"])
		return
	}

	expectedAvailability := "in stock"
	if result.Data["availability"] != expectedAvailability {
		t.Errorf("Availability: expected '%s', got '%v'", expectedAvailability, result.Data["availability"])
		return
	}

	t.Logf("✅ Complete workflow integration test passed successfully")
	t.Logf("   Total Duration: %v", totalDuration)
	t.Logf("   Request Duration: %v", result.Metadata.RequestDuration)
	t.Logf("   Extraction Duration: %v", result.Metadata.ExtractionDuration)
	t.Logf("   Response Size: %d bytes", result.Metadata.ResponseSize)
	t.Logf("   Fields Extracted: %d/%d", result.Metadata.ExtractedFields, result.Metadata.TotalFields)
	t.Logf("   Data: %+v", result.Data)
}

// TestErrorHandlingAndRecovery validates error handling in both HTTP and extraction stages
func TestErrorHandlingAndRecovery(t *testing.T) {
	// Test configuration with invalid transform type - this should fail during engine creation
	invalidConfig := &scraper.EngineConfig{
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		UserAgents:     []string{"TestAgent/1.0"},
		Fields: []scraper.FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
			{
				Name:     "price",
				Selector: ".price",
				Type:     "text",
				Required: false,
			},
			{
				Name:     "description",
				Selector: ".description",
				Type:     "text",
				Required: false,
				Transform: []pipeline.TransformRule{
					{Type: "invalid_type"}, // This should cause validation to fail
				},
			},
		},
		ExtractionConfig: scraper.ExtractionConfig{
			StrictMode:      false,
			ContinueOnError: true,
		},
	}

	// This should fail due to invalid transform type
	_, err := scraper.NewScrapingEngine(invalidConfig)
	if err == nil {
		t.Fatalf("Expected engine creation to fail with invalid transform type, but it succeeded")
	}

	// Validate the specific error message
	expectedErrorSubstring := "invalid transform type 'invalid_type'"
	if !strings.Contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedErrorSubstring, err)
		return
	}

	t.Logf("✅ Error handling and recovery test passed successfully")
	t.Logf("   Failed to create scraping engine: %v", err)
}

// TestTimeoutImplementationFix validates that HTTP client timeouts are properly enforced
func TestTimeoutImplementationFix(t *testing.T) {
	// Create a slow server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Delay longer than client timeout
		fmt.Fprint(w, "<html><body><h1>Slow Response</h1></body></html>")
	}))
	defer server.Close()

	// Create engine config with short timeout
	config := &scraper.EngineConfig{
		RequestTimeout: 1 * time.Second, // Short timeout to trigger timeout error
		RetryAttempts:  0,                // No retries to avoid retry delays
		UserAgents:     []string{"TestAgent/1.0"},
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
			ContinueOnError: false,
		},
	}

	// Create scraping engine
	engine, err := scraper.NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}
	defer engine.Close()

	// Test timeout enforcement
	startTime := time.Now()
	result, err := engine.Scrape(context.Background(), server.URL)
	duration := time.Since(startTime)

	// Expect the request to fail due to timeout
	if err == nil {
		t.Errorf("Expected timeout error, but request succeeded")
		return
	}

	if result != nil && result.Success {
		t.Errorf("Expected scraping to fail due to timeout, but it succeeded")
		return
	}

	// Verify that the timeout was actually enforced (should be around 1 second, not 5)
	if duration > 2*time.Second {
		t.Errorf("Timeout not properly enforced. Expected ~1s, took %v", duration)
		return
	}

	// Check that the error indicates a timeout
	if err != nil {
		errorStr := strings.ToLower(err.Error())
		if !strings.Contains(errorStr, "timeout") && !strings.Contains(errorStr, "deadline") && !strings.Contains(errorStr, "context") {
			t.Errorf("Error should indicate timeout, but got: %v", err)
			return
		}
	}

	t.Logf("✅ Timeout implementation test passed successfully")
	t.Logf("   Duration: %v", duration)
	t.Logf("   Error: %v", err)
}

// TestContextTimeoutRespected validates that context timeouts are properly handled
func TestContextTimeoutRespected(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		fmt.Fprint(w, "<html><body><h1>Response</h1></body></html>")
	}))
	defer server.Close()

	// Create engine with longer timeout than context
	config := &scraper.EngineConfig{
		RequestTimeout: 5 * time.Second, // Longer than context timeout
		RetryAttempts:  1,
		UserAgents:     []string{"TestAgent/1.0"},
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
			ContinueOnError: false,
		},
	}

	engine, err := scraper.NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}
	defer engine.Close()

	// Create context with shorter timeout than server delay
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	startTime := time.Now()
	result, err := engine.Scrape(ctx, server.URL)
	duration := time.Since(startTime)

	// Should fail due to context timeout
	if err == nil {
		t.Errorf("Expected context timeout error, but request succeeded")
		return
	}

	if result != nil && result.Success {
		t.Errorf("Expected scraping to fail due to context timeout")
		return
	}

	// Should respect context timeout (~1s), not server delay (2s) or client timeout (5s)
	if duration > 1500*time.Millisecond {
		t.Errorf("Context timeout not respected. Expected ~1s, took %v", duration)
		return
	}

	t.Logf("✅ Context timeout test passed successfully")
	t.Logf("   Duration: %v", duration)
	t.Logf("   Error: %v", err)
}
