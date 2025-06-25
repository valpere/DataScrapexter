// internal/scraper/engine_test.go
package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/valpere/DataScrapexter/internal/pipeline"
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
		t.Fatal("Engine should not be nil")
	}

	if engine.config != config {
		t.Fatal("Engine should reference the provided config")
	}

	if engine.httpClient == nil {
		t.Fatal("HTTP client should be initialized")
	}

	if engine.errorCollector == nil {
		t.Fatal("Error collector should be initialized")
	}
}

func TestNewScrapingEngine_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *EngineConfig
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name: "no fields",
			config: &EngineConfig{
				UserAgents:     []string{"TestAgent/1.0"},
				RequestTimeout: 30 * time.Second,
				Fields:         []FieldConfig{},
			},
		},
		{
			name: "empty field name",
			config: &EngineConfig{
				UserAgents:     []string{"TestAgent/1.0"},
				RequestTimeout: 30 * time.Second,
				Fields: []FieldConfig{
					{
						Name:     "",
						Selector: "h1",
						Type:     "text",
					},
				},
			},
		},
		{
			name: "empty field selector",
			config: &EngineConfig{
				UserAgents:     []string{"TestAgent/1.0"},
				RequestTimeout: 30 * time.Second,
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "",
						Type:     "text",
					},
				},
			},
		},
		{
			name: "invalid field type",
			config: &EngineConfig{
				UserAgents:     []string{"TestAgent/1.0"},
				RequestTimeout: 30 * time.Second,
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "invalid_type",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewScrapingEngine(tt.config)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestScrapingEngine_Scrape_Success(t *testing.T) {
	testHTML := `
	<html>
		<body>
			<h1>Test Title</h1>
			<p class="content">Test content</p>
		</body>
	</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testHTML))
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
		t.Fatalf("Scraping failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
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

func TestScrapingEngine_Scrape_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
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

	if err == nil {
		t.Error("Expected HTTP error but got none")
	}

	if result.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", result.StatusCode)
	}
}

func TestScrapingEngine_Scrape_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("<html><body><h1>Title</h1></body></html>"))
	}))
	defer server.Close()

	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 500 * time.Millisecond,
		RetryAttempts:  0, // No retries to ensure quick timeout
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
	start := time.Now()
	_, err = engine.Scrape(ctx, server.URL)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error but got none")
	}

	if duration > 1*time.Second {
		t.Errorf("Request took too long: %v (expected ~500ms)", duration)
	}
}

func TestScrapingEngine_UserAgentRotation(t *testing.T) {
	var receivedUserAgents []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgents = append(receivedUserAgents, r.Header.Get("User-Agent"))
		w.Write([]byte("<html><body><h1>Title</h1></body></html>"))
	}))
	defer server.Close()

	config := &EngineConfig{
		UserAgents:     []string{"Agent1/1.0", "Agent2/1.0", "Agent3/1.0"},
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

	// Make multiple requests
	for i := 0; i < 4; i++ {
		_, err := engine.Scrape(ctx, server.URL)
		if err != nil {
			t.Fatalf("Scraping failed on iteration %d: %v", i, err)
		}
	}

	// Check user agent rotation
	if len(receivedUserAgents) != 4 {
		t.Fatalf("Expected 4 requests, got %d", len(receivedUserAgents))
	}

	// First three should be different, fourth should be same as first
	expectedAgents := []string{"Agent1/1.0", "Agent2/1.0", "Agent3/1.0", "Agent1/1.0"}
	for i, expected := range expectedAgents {
		if receivedUserAgents[i] != expected {
			t.Errorf("Request %d: expected user agent %s, got %s", i, expected, receivedUserAgents[i])
		}
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
			name: "valid config",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid transform type",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
						Transform: []pipeline.TransformRule{
							{Type: "invalid_type"},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "config with defaults applied",
			config: &EngineConfig{
				Fields: []FieldConfig{
					{
						Name:     "title",
						Selector: "h1",
						Type:     "text",
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
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && err == nil {
				if tt.config.RequestTimeout <= 0 {
					t.Error("Expected default timeout to be applied")
				}
				if tt.config.RetryAttempts < 0 {
					t.Error("Expected default retry attempts to be applied")
				}
				if tt.config.MaxConcurrency <= 0 {
					t.Error("Expected default concurrency to be applied")
				}
			}
		})
	}
}

func TestScrapingEngine_GetStats(t *testing.T) {
	config := &EngineConfig{
		UserAgents:     []string{"Agent1/1.0", "Agent2/1.0"},
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
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	stats := engine.GetStats()

	if stats["total_fields"] != 1 {
		t.Errorf("Expected total_fields 1, got %v", stats["total_fields"])
	}

	if stats["user_agents"] != 2 {
		t.Errorf("Expected user_agents 2, got %v", stats["user_agents"])
	}

	if stats["retry_attempts"] != 3 {
		t.Errorf("Expected retry_attempts 3, got %v", stats["retry_attempts"])
	}

	if stats["max_concurrency"] != 5 {
		t.Errorf("Expected max_concurrency 5, got %v", stats["max_concurrency"])
	}
}

func TestScrapingEngine_Close(t *testing.T) {
	config := &EngineConfig{
		UserAgents:     []string{"TestAgent/1.0"},
		RequestTimeout: 30 * time.Second,
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

	err = engine.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
}
