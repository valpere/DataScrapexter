// internal/scraper/engine.go
package scraper

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ScrapingEngine is the main scraping engine
type ScrapingEngine struct {
	config     *EngineConfig
	httpClient *http.Client
}

// NewScrapingEngine creates a new scraping engine
func NewScrapingEngine(config *EngineConfig) (*ScrapingEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("engine configuration is required")
	}

	// Apply default configuration values
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 5
	}
	if len(config.UserAgents) == 0 {
		config.UserAgents = []string{"DataScrapexter/1.0"}
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: config.RequestTimeout,
	}

	return &ScrapingEngine{
		config:     config,
		httpClient: client,
	}, nil
}

// Scrape performs the scraping operation
func (se *ScrapingEngine) Scrape(ctx context.Context, url string) (*ScrapingResult, error) {
	start := time.Now()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	if len(se.config.UserAgents) > 0 {
		req.Header.Set("User-Agent", se.config.UserAgents[0])
	}

	// Perform HTTP request
	resp, err := se.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract fields
	data := make(map[string]interface{})
	var errors []string
	var warnings []string

	for _, field := range se.config.Fields {
		value, err := se.extractField(doc, field)
		if err != nil {
			if field.Required {
				errors = append(errors, fmt.Sprintf("required field '%s': %v", field.Name, err))
			} else {
				warnings = append(warnings, fmt.Sprintf("optional field '%s': %v", field.Name, err))
				data[field.Name] = se.getDefaultValue(field)
			}
		} else {
			data[field.Name] = value
		}
	}

	// Create result
	result := &ScrapingResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		Data:       data,
		Success:    len(errors) == 0,
		Errors:     errors,
		Warnings:   warnings,
		Metadata: ScrapingMetadata{
			RequestDuration: time.Since(start).String(),
			URL:             url,
			StatusCode:      resp.StatusCode,
			Timestamp:       time.Now().Format(time.RFC3339),
		},
	}

	return result, nil
}

// extractField extracts a single field from the document
func (se *ScrapingEngine) extractField(doc *goquery.Document, field FieldConfig) (interface{}, error) {
	selection := doc.Find(field.Selector)
	if selection.Length() == 0 {
		return nil, fmt.Errorf("selector '%s' not found", field.Selector)
	}

	switch field.Type {
	case "text":
		return selection.First().Text(), nil
	case "html":
		html, err := selection.First().Html()
		if err != nil {
			return nil, fmt.Errorf("failed to extract HTML: %w", err)
		}
		return html, nil
	case "attr":
		if field.Attribute == "" {
			return nil, fmt.Errorf("attribute name required for attr type")
		}
		value, exists := selection.First().Attr(field.Attribute)
		if !exists {
			return nil, fmt.Errorf("attribute '%s' not found", field.Attribute)
		}
		return value, nil
	case "list":
		var items []string
		selection.Each(func(i int, s *goquery.Selection) {
			items = append(items, s.Text())
		})
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported field type: %s", field.Type)
	}
}

// getDefaultValue returns the default value for a field
func (se *ScrapingEngine) getDefaultValue(field FieldConfig) interface{} {
	if field.Default != nil {
		return field.Default
	}

	switch field.Type {
	case "text", "html", "attr":
		return ""
	case "list":
		return []string{}
	default:
		return ""
	}
}

// Close closes the scraping engine and cleans up resources
func (se *ScrapingEngine) Close() error {
	// Close HTTP client if needed
	if se.httpClient != nil {
		// HTTP client doesn't need explicit closing in Go
		se.httpClient = nil
	}
	return nil
}

// validateEngineConfig validates the engine configuration
func validateEngineConfig(config *EngineConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if len(config.Fields) == 0 {
		return fmt.Errorf("at least one field must be configured")
	}

	for _, field := range config.Fields {
		if field.Name == "" {
			return fmt.Errorf("field name cannot be empty")
		}
		if field.Selector == "" {
			return fmt.Errorf("field selector cannot be empty for field '%s'", field.Name)
		}
		if field.Type == "" {
			field.Type = "text" // default type
		}
		if field.Type == "attr" && field.Attribute == "" {
			return fmt.Errorf("attribute name required for attr type field '%s'", field.Name)
		}
	}

	return nil
}
