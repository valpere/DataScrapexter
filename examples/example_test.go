package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/valpere/DataScrapexter/internal/scraper"
)

func main() {
	// Example 1: Basic scraping with the engine
	basicExample()

	// Example 2: Scraping with custom configuration
	customConfigExample()

	// Example 3: Scraping multiple pages
	multiPageExample()
}

func basicExample() {
	fmt.Println("=== Basic Scraping Example ===")

	// Create engine with default configuration
	engine, err := scraper.NewEngine(nil)
	if err != nil {
		log.Fatal("Failed to create engine:", err)
	}

	// Define what to extract
	extractors := []scraper.FieldExtractor{
		{
			Name:     "title",
			Selector: "h1",
			Type:     "text",
			Required: true,
		},
		{
			Name:      "description",
			Selector:  "meta[name='description']",
			Type:      "attr",
			Attribute: "content",
		},
	}

	// Scrape a page
	ctx := context.Background()
	result, err := engine.Scrape(ctx, "https://example.com", extractors)
	if err != nil {
		log.Printf("Scraping error: %v", err)
		return
	}

	// Print results
	fmt.Printf("URL: %s\n", result.URL)
	fmt.Printf("Status Code: %d\n", result.StatusCode)
	fmt.Printf("Data: %+v\n", result.Data)
	fmt.Println()
}

func customConfigExample() {
	fmt.Println("=== Custom Configuration Example ===")

	// Create custom configuration
	config := &scraper.Config{
		MaxRetries:      5,
		RetryDelay:      3 * time.Second,
		Timeout:         45 * time.Second,
		FollowRedirects: true,
		MaxRedirects:    5,
		RateLimit:       2 * time.Second,
		BurstSize:       3,
		Headers: map[string]string{
			"Accept-Language": "en-US,en;q=0.9",
			"Cache-Control":   "no-cache",
		},
	}

	// Create engine with custom config
	engine, err := scraper.NewEngine(config)
	if err != nil {
		log.Fatal("Failed to create engine:", err)
	}

	// Extract quotes from a test site
	extractors := []scraper.FieldExtractor{
		{
			Name:     "quotes",
			Selector: ".quote .text",
			Type:     "list",
			Required: true,
		},
		{
			Name:     "authors",
			Selector: ".quote .author",
			Type:     "list",
		},
	}

	// Scrape quotes
	ctx := context.Background()
	result, err := engine.Scrape(ctx, "http://quotes.toscrape.com/", extractors)
	if err != nil {
		log.Printf("Scraping error: %v", err)
		return
	}

	// Print extracted quotes
	if quotes, ok := result.Data["quotes"].([]string); ok {
		fmt.Printf("Found %d quotes:\n", len(quotes))
		for i, quote := range quotes {
			if i < 3 { // Print first 3 quotes
				fmt.Printf("%d. %s\n", i+1, quote)
			}
		}
	}
	fmt.Println()
}

func multiPageExample() {
	fmt.Println("=== Multi-Page Scraping Example ===")

	// Create engine
	engine, err := scraper.NewEngine(nil)
	if err != nil {
		log.Fatal("Failed to create engine:", err)
	}

	// URLs to scrape
	urls := []string{
		"http://quotes.toscrape.com/page/1/",
		"http://quotes.toscrape.com/page/2/",
		"http://quotes.toscrape.com/page/3/",
	}

	// Define extractors
	extractors := []scraper.FieldExtractor{
		{
			Name:     "quote_count",
			Selector: ".quote",
			Type:     "list",
		},
	}

	// Results collection
	var allResults []*scraper.Result

	// Scrape multiple pages
	ctx := context.Background()
	for _, url := range urls {
		fmt.Printf("Scraping: %s\n", url)

		result, err := engine.Scrape(ctx, url, extractors)
		if err != nil {
			log.Printf("Error scraping %s: %v", url, err)
			continue
		}

		allResults = append(allResults, result)

		// Get quote count
		if quotes, ok := result.Data["quote_count"].([]string); ok {
			fmt.Printf("  Found %d quotes on this page\n", len(quotes))
		}
	}

	// Save all results to JSON
	output := struct {
		TotalPages int               `json:"total_pages"`
		Results    []*scraper.Result `json:"results"`
		Timestamp  time.Time         `json:"timestamp"`
	}{
		TotalPages: len(allResults),
		Results:    allResults,
		Timestamp:  time.Now(),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal results: %v", err)
		return
	}

	fmt.Printf("\nScraping completed. Total pages: %d\n", len(allResults))
	fmt.Println("Results summary (JSON):")
	fmt.Println(string(data)[:200] + "...") // Print first 200 chars
}

// Example of custom field extractor for complex scenarios
func customExtractorExample() {
	fmt.Println("=== Custom Extractor Example ===")

	config := scraper.DefaultConfig()
	engine, err := scraper.NewEngine(config)
	if err != nil {
		log.Fatal("Failed to create engine:", err)
	}

	// Extract product information
	extractors := []scraper.FieldExtractor{
		{
			Name:     "product_name",
			Selector: "h1.product-title",
			Type:     "text",
			Required: true,
		},
		{
			Name:     "price",
			Selector: ".price-now",
			Type:     "text",
		},
		{
			Name:      "image_url",
			Selector:  "img.product-photo",
			Type:      "attr",
			Attribute: "src",
		},
		{
			Name:     "availability",
			Selector: ".availability-msg",
			Type:     "text",
		},
		{
			Name:     "features",
			Selector: ".product-features li",
			Type:     "list",
		},
	}

	// Example product page (would need a real URL)
	ctx := context.Background()
	result, err := engine.Scrape(ctx, "https://example-shop.com/product/123", extractors)
	if err != nil {
		log.Printf("Scraping error: %v", err)
		return
	}

	// Process and display results
	fmt.Printf("Product Information:\n")
	for key, value := range result.Data {
		fmt.Printf("  %s: %v\n", key, value)
	}
}
