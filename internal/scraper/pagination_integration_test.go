// internal/scraper/pagination_integration_test.go
package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestPaginationIntegration tests end-to-end pagination functionality
func TestPaginationIntegration(t *testing.T) {
	// Create a test server that simulates paginated content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}
		
		var content string
		switch page {
		case "1":
			content = `
				<div class="content">
					<div class="item">Item 1</div>
					<div class="item">Item 2</div>
					<div class="item">Item 3</div>
				</div>
				<a href="?page=2" class="next-btn">Next</a>
			`
		case "2":
			content = `
				<div class="content">
					<div class="item">Item 4</div>
					<div class="item">Item 5</div>
					<div class="item">Item 6</div>
				</div>
				<a href="?page=3" class="next-btn">Next</a>
			`
		case "3":
			content = `
				<div class="content">
					<div class="item">Item 7</div>
					<div class="item">Item 8</div>
				</div>
				<span class="next-btn disabled">Next</span>
			`
		default:
			http.NotFound(w, r)
			return
		}
		
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head><title>Test Page %s</title></head>
			<body>%s</body>
			</html>
		`, page, content)
		
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	// Test URL pattern pagination
	t.Run("URL Pattern Pagination", func(t *testing.T) {
		config := &Config{
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			Pagination: &PaginationConfig{
				Enabled:           true,
				Type:              PaginationTypeURLPattern,
				URLTemplate:       server.URL + "?page={page}",
				MaxPages:          3,
				StartPage:         1,
				DelayBetweenPages: 100 * time.Millisecond,
				ContinueOnError:   true,
			},
		}

		engine, err := NewEngine(config)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}
		defer engine.Close()

		extractors := []FieldConfig{
			{
				Name:     "items",
				Selector: ".item",
				Type:     "array",
				Required: true,
			},
		}

		ctx := context.Background()
		result, err := engine.ScrapeWithPagination(ctx, server.URL, extractors)
		if err != nil {
			t.Fatalf("Pagination scraping failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Expected successful pagination result")
		}

		if result.TotalPages != 3 {
			t.Errorf("Expected 3 pages, got %d", result.TotalPages)
		}

		if result.ProcessedPages != 3 {
			t.Errorf("Expected 3 processed pages, got %d", result.ProcessedPages)
		}

		// Check that we got items from all pages
		totalItems := 0
		for i, page := range result.Pages {
			if page.Data["items"] != nil {
				items, ok := page.Data["items"].([]string)
				if !ok {
					t.Errorf("Page %d: expected items to be []string", i+1)
					continue
				}
				totalItems += len(items)
			}
		}

		expectedTotalItems := 8 // Items 1-8 across 3 pages
		if totalItems != expectedTotalItems {
			t.Errorf("Expected %d total items, got %d", expectedTotalItems, totalItems)
		}
	})

	// Test offset pagination  
	t.Run("Offset Pagination", func(t *testing.T) {
		// Create a server for offset pagination
		offsetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			offset := r.URL.Query().Get("offset")
			limit := r.URL.Query().Get("limit")
			
			if offset == "" {
				offset = "0"
			}
			if limit == "" {
				limit = "10"
			}

			// Simple content for offset testing
			content := fmt.Sprintf(`
				<div class="content">
					<div class="item">Item at offset %s</div>
					<div class="item">Another item at offset %s</div>
				</div>
			`, offset, offset)

			html := fmt.Sprintf(`
				<!DOCTYPE html>
				<html>
				<head><title>Offset Page</title></head>
				<body>%s</body>
				</html>
			`, content)

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
		}))
		defer offsetServer.Close()

		config := &Config{
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			Pagination: &PaginationConfig{
				Enabled:           true,
				Type:              PaginationTypeOffset,
				OffsetParam:       "offset",
				LimitParam:        "limit",
				PageSize:          2,
				MaxPages:          3,
				DelayBetweenPages: 100 * time.Millisecond,
				ContinueOnError:   true,
			},
		}

		engine, err := NewEngine(config)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}
		defer engine.Close()

		extractors := []FieldConfig{
			{
				Name:     "items",
				Selector: ".item",
				Type:     "array",
				Required: true,
			},
		}

		ctx := context.Background()
		result, err := engine.ScrapeWithPagination(ctx, offsetServer.URL, extractors)
		if err != nil {
			t.Fatalf("Offset pagination scraping failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Expected successful pagination result")
		}

		if result.TotalPages != 3 {
			t.Errorf("Expected 3 pages, got %d", result.TotalPages)
		}

		// Verify URLs contain offset parameters
		for i, page := range result.Pages {
			expectedOffset := i * 2
			if !strings.Contains(page.URL, fmt.Sprintf("offset=%d", expectedOffset)) {
				t.Errorf("Page %d URL should contain offset=%d, got: %s", i+1, expectedOffset, page.URL)
			}
			if !strings.Contains(page.URL, "limit=2") {
				t.Errorf("Page %d URL should contain limit=2, got: %s", i+1, page.URL)
			}
		}
	})

	// Test disabled pagination (single page)
	t.Run("Disabled Pagination", func(t *testing.T) {
		config := &Config{
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			Pagination: &PaginationConfig{
				Enabled: false,
			},
		}

		engine, err := NewEngine(config)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}
		defer engine.Close()

		extractors := []FieldConfig{
			{
				Name:     "items",
				Selector: ".item",
				Type:     "array",
				Required: true,
			},
		}

		ctx := context.Background()
		result, err := engine.ScrapeWithPagination(ctx, server.URL, extractors)
		if err != nil {
			t.Fatalf("Single page scraping failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Expected successful single page result")
		}

		if result.TotalPages != 1 {
			t.Errorf("Expected 1 page for disabled pagination, got %d", result.TotalPages)
		}
	})
}

// TestPaginationValidation tests pagination configuration validation
func TestPaginationValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      PaginationConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid URL Pattern Config",
			config: PaginationConfig{
				Enabled:     true,
				Type:        PaginationTypeURLPattern,
				URLTemplate: "https://example.com/page/{page}",
				MaxPages:    10,
			},
			expectError: false,
		},
		{
			name: "Valid Offset Config",
			config: PaginationConfig{
				Enabled:     true,
				Type:        PaginationTypeOffset,
				OffsetParam: "skip",
				LimitParam:  "take",
				PageSize:    20,
				MaxPages:    5,
			},
			expectError: false,
		},
		{
			name: "Invalid URL Pattern - Missing Template",
			config: PaginationConfig{
				Enabled:  true,
				Type:     PaginationTypeURLPattern,
				MaxPages: 10,
			},
			expectError: true,
			errorMsg:    "url_template is required",
		},
		{
			name: "Invalid Offset - Zero Page Size",
			config: PaginationConfig{
				Enabled:  true,
				Type:     PaginationTypeOffset,
				PageSize: 0,
				MaxPages: 5,
			},
			expectError: true,
			errorMsg:    "page_size must be greater than 0",
		},
		{
			name: "Invalid Next Button - Missing Selector",
			config: PaginationConfig{
				Enabled:  true,
				Type:     PaginationTypeNextButton,
				MaxPages: 10,
			},
			expectError: true,
			errorMsg:    "next_selector is required",
		},
		{
			name: "Invalid Type",
			config: PaginationConfig{
				Enabled: true,
				Type:    "invalid_type",
			},
			expectError: true,
			errorMsg:    "unsupported pagination type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePaginationConfig(tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}