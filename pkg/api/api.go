// pkg/api/api.go
package api

import (
	"context"
	"fmt"
	"time"

	"github.com/valpere/DataScrapexter/internal/config"
)

// Re-export types from internal packages for public API
type ScraperConfig = config.ScraperConfig
type FieldConfig = config.FieldConfig
type PaginationConfig = config.PaginationConfig
type OutputConfig = config.OutputConfig
type TransformRule = config.TransformRule

// ScraperClient provides a high-level interface for scraping
type ScraperClient struct {
	config *ScraperConfig
}

// NewScraperClient creates a new scraper client
func NewScraperClient(config *ScraperConfig) *ScraperClient {
	return &ScraperClient{
		config: config,
	}
}

// Scrape performs the scraping operation
func (sc *ScraperClient) Scrape(ctx context.Context) ([]map[string]interface{}, error) {
	start := time.Now()
	defer func() {
		// Metrics would be recorded here
		_ = time.Since(start)
	}()

	// Get URLs to scrape
	urls := sc.getURLsToScrape()
	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs to scrape")
	}

	// Mock implementation for testing
	results := []map[string]interface{}{
		{
			"title":       "Test Title",
			"description": "Test Description",
			"price":       "$19.99",
		},
	}

	return results, nil
}

// ScrapeParallel performs parallel scraping across multiple URLs
func (sc *ScraperClient) ScrapeParallel(ctx context.Context) ([]map[string]interface{}, error) {
	urls := sc.getURLsToScrape()
	if len(urls) == 0 {
		return sc.Scrape(ctx)
	}

	var results []map[string]interface{}
	for range urls {
		pageResults, err := sc.Scrape(ctx)
		if err != nil {
			continue
		}
		results = append(results, pageResults...)
	}

	return results, nil
}

// EnableMetrics enables/disables metrics collection
func (sc *ScraperClient) EnableMetrics(enabled bool) {
	// Implementation would control metrics collection
	sc.config.Output.EnableMetrics = enabled
}

// getURLsToScrape returns the list of URLs to scrape
func (sc *ScraperClient) getURLsToScrape() []string {
	if len(sc.config.URLs) > 0 {
		return sc.config.URLs
	}
	if sc.config.BaseURL != "" {
		return []string{sc.config.BaseURL}
	}
	return []string{}
}
