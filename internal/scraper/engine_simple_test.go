// internal/scraper/engine_simple_test.go
package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewEngineSimple(t *testing.T) {
	config := &Config{
		MaxRetries:      3,
		Timeout:         30 * time.Second,
		FollowRedirects: true,
		MaxRedirects:    10,
		RateLimit:       1 * time.Second,
		BurstSize:       5,
	}

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create scraping engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}
}

func TestScrapeBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test Title</h1><p>Test content</p></body></html>"))
	}))
	defer server.Close()

	config := &Config{
		MaxRetries:      1,
		Timeout:         10 * time.Second,
		FollowRedirects: true,
		MaxRedirects:    10,
		RateLimit:       100 * time.Millisecond,
		BurstSize:       1,
	}

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
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

	ctx := context.Background()
	result, err := engine.Scrape(ctx, server.URL, fields)

	if err != nil {
		t.Errorf("Scraping failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful scraping, got errors: %v", result.Errors)
	}

	if result.Data["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", result.Data["title"])
	}

	if result.Data["content"] != "Test content" {
		t.Errorf("Expected content 'Test content', got %v", result.Data["content"])
	}
}
