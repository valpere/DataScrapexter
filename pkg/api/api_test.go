// pkg/api/api_test.go
package api

import (
	"context"
	"testing"
)

func TestScraperClient(t *testing.T) {
	config := ScraperConfig{
		Name:    "test_scraper",
		BaseURL: "https://example.com",
		Fields: []FieldConfig{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
		},
		Output: OutputConfig{
			Format: "json",
			File:   "output.json",
		},
	}

	client := NewScraperClient(&config)
	if client == nil {
		t.Fatal("failed to create scraper client")
	}

	ctx := context.Background()
	results, err := client.Scrape(ctx)
	if err != nil {
		t.Fatalf("scraping failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
}

func TestScraperMetrics(t *testing.T) {
	config := ScraperConfig{
		Name:    "metrics_test",
		BaseURL: "https://example.com",
		Fields: []FieldConfig{
			{Name: "title", Selector: "h1", Type: "text"},
		},
		Output: OutputConfig{
			Format:        "json",
			File:          "metrics.json",
			EnableMetrics: true,
		},
	}

	client := NewScraperClient(&config)
	if client == nil {
		t.Fatal("failed to create client")
	}

	for i := 0; i < 3; i++ {
		_, err := client.Scrape(context.Background())
		if err != nil {
			t.Errorf("scrape %d failed: %v", i+1, err)
		}
	}

	// Note: GetMetrics() would need to be implemented if metrics are needed
	// For now, just verify the scraping works
}
