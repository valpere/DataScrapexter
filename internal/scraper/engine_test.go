// internal/scraper/engine_test.go
package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewScrapingEngine(t *testing.T) {
	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 30 * time.Second,
		RetryAttempts:  3,
		MaxConcurrency: 5,
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
		ExtractionConfig: ExtractionConfig{
			StrictMode:      false,
			ContinueOnError: true,
		},
	}

	engine, err := NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}

	if engine.config != config {
		t.Error("Engine should reference the provided config")
	}
}

func TestNewScrapingEngine_NilConfig(t *testing.T) {
	_, err := NewScrapingEngine(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}

func TestNewScrapingEngine_DefaultValues(t *testing.T) {
	config := &EngineConfig{
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
	}

	engine, err := NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}

	// Check that defaults are applied
	if engine.config.RequestTimeout == 0 {
		t.Error("Expected default timeout to be applied")
	}
	if engine.config.RetryAttempts == 0 {
		t.Error("Expected default retry attempts to be applied")
	}
	if engine.config.MaxConcurrency == 0 {
		t.Error("Expected default concurrency to be applied")
	}
	if len(engine.config.UserAgents) == 0 {
		t.Error("Expected default user agents to be applied")
	}
}

func TestScrapingEngine_Scrape_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test Title</h1></body></html>"))
	}))
	defer server.Close()

	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
		ExtractionConfig: ExtractionConfig{
			StrictMode:      false,
			ContinueOnError: true,
		},
	}

	engine, err := NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL)

	if err != nil {
		t.Errorf("Scraping failed: %v", err)
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	if !result.Success {
		t.Errorf("Expected successful scraping, got errors: %v", result.Errors)
	}

	if result.Data["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", result.Data["title"])
	}
}

func TestScrapingEngine_Scrape_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<html><body><h1>Not Found</h1></body></html>"))
	}))
	defer server.Close()

	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
	}

	engine, err := NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL)

	if err != nil {
		t.Errorf("Scraping failed: %v", err)
	}

	if result.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", result.StatusCode)
	}

	// Should still extract content even with 404
	if result.Data["title"] != "Not Found" {
		t.Errorf("Expected title 'Not Found', got %v", result.Data["title"])
	}
}

func TestScrapingEngine_Scrape_RequiredFieldMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><p>No title here</p></body></html>"))
	}))
	defer server.Close()

	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1", // This won't be found
				Type:     "text",
				Required: true,
			},
		},
	}

	engine, err := NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL)

	if err != nil {
		t.Errorf("Scraping failed: %v", err)
	}

	if result.Success {
		t.Error("Expected scraping to fail due to missing required field")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors for missing required field")
	}
}

func TestScrapingEngine_Scrape_OptionalFieldMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test Title</h1></body></html>"))
	}))
	defer server.Close()

	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
			{
				Name:     "subtitle",
				Selector: ".subtitle", // This won't be found
				Type:     "text",
				Required: false,
				Default:  "No subtitle",
			},
		},
	}

	engine, err := NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL)

	if err != nil {
		t.Errorf("Scraping failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful scraping, got errors: %v", result.Errors)
	}

	if result.Data["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", result.Data["title"])
	}

	if result.Data["subtitle"] != "No subtitle" {
		t.Errorf("Expected subtitle 'No subtitle', got %v", result.Data["subtitle"])
	}

	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for missing optional field")
	}
}

func TestScrapingEngine_Scrape_MultipleFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
			<body>
				<h1>Test Title</h1>
				<p class="description">Test description</p>
				<span class="price">$19.99</span>
				<a href="https://example.com" class="link">Example</a>
				<ul class="tags">
					<li>tag1</li>
					<li>tag2</li>
				</ul>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
			{
				Name:     "description",
				Selector: ".description",
				Type:     "text",
				Required: false,
			},
			{
				Name:     "price",
				Selector: ".price",
				Type:     "text",
				Required: false,
			},
			{
				Name:      "link",
				Selector:  ".link",
				Type:      "attr",
				Attribute: "href",
				Required:  false,
			},
			{
				Name:     "tags",
				Selector: ".tags li",
				Type:     "list",
				Required: false,
			},
		},
	}

	engine, err := NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL)

	if err != nil {
		t.Errorf("Scraping failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful scraping, got errors: %v", result.Errors)
	}

	// Check all extracted fields
	if result.Data["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", result.Data["title"])
	}

	if result.Data["description"] != "Test description" {
		t.Errorf("Expected description 'Test description', got %v", result.Data["description"])
	}

	if result.Data["price"] != "$19.99" {
		t.Errorf("Expected price '$19.99', got %v", result.Data["price"])
	}

	if result.Data["link"] != "https://example.com" {
		t.Errorf("Expected link 'https://example.com', got %v", result.Data["link"])
	}

	tags, ok := result.Data["tags"].([]string)
	if !ok {
		t.Errorf("Expected tags to be []string, got %T", result.Data["tags"])
	} else if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}

func TestScrapingEngine_InvalidURL(t *testing.T) {
	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  1,
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
	}

	engine, err := NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	_, err = engine.Scrape(ctx, "invalid-url")

	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestScrapingEngine_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than timeout
		w.Write([]byte("<html><body><h1>Delayed</h1></body></html>"))
	}))
	defer server.Close()

	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 1 * time.Second, // Short timeout
		RetryAttempts:  1,
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
	}

	engine, err := NewScrapingEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	_, err = engine.Scrape(ctx, server.URL)

	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestValidateEngineConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *EngineConfig
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "no fields",
			config: &EngineConfig{
				Fields: []FieldConfig{},
			},
			expectError: true,
		},
		{
			name: "field missing name",
			config: &EngineConfig{
				Fields: []FieldConfig{
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
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name: "title",
						Type: "text",
					},
				},
			},
			expectError: true,
		},
		{
			name: "attr field missing attribute",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "link",
						Selector: "a",
						Type:     "attr",
					},
				},
			},
			expectError: true,
		},
		{
			name: "valid config",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
						Required: true,
					},
					{
						Name:      "link",
						Selector:  "a",
						Type:      "attr",
						Attribute: "href",
						Required:  false,
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEngineConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
