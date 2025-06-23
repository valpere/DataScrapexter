// internal/scraper/pagination.go
package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// PaginationConfig defines pagination configuration
type PaginationConfig struct {
	Type        string `yaml:"type" json:"type"`
	Selector    string `yaml:"selector,omitempty" json:"selector,omitempty"`
	MaxPages    int    `yaml:"max_pages,omitempty" json:"max_pages,omitempty"`
	
	// For offset pagination
	OffsetParam string `yaml:"offset_param,omitempty" json:"offset_param,omitempty"`
	LimitParam  string `yaml:"limit_param,omitempty" json:"limit_param,omitempty"`
	Limit       int    `yaml:"limit,omitempty" json:"limit,omitempty"`
	
	// For cursor pagination  
	CursorParam    string `yaml:"cursor_param,omitempty" json:"cursor_param,omitempty"`
	CursorSelector string `yaml:"cursor_selector,omitempty" json:"cursor_selector,omitempty"`
}

// PaginationManager manages pagination across different strategies
type PaginationManager struct {
	config   PaginationConfig
	strategy PaginationStrategy
}

// PaginationResult holds the result of a pagination operation
type PaginationResult struct {
	NextURL     string `json:"next_url"`
	CurrentPage int    `json:"current_page"`
	IsComplete  bool   `json:"is_complete"`
	Strategy    string `json:"strategy"`
	Error       string `json:"error,omitempty"`
}

// NewPaginationManager creates a new pagination manager
func NewPaginationManager(config PaginationConfig) (*PaginationManager, error) {
	pm := &PaginationManager{
		config: config,
	}
	
	// Create the appropriate strategy based on type
	strategy, err := pm.createStrategy()
	if err != nil {
		return nil, fmt.Errorf("failed to create pagination strategy: %w", err)
	}
	
	pm.strategy = strategy
	return pm, nil
}

// createStrategy creates the appropriate pagination strategy
func (pm *PaginationManager) createStrategy() (PaginationStrategy, error) {
	switch pm.config.Type {
	case "offset":
		return &OffsetStrategy{
			BaseURL:     "",
			OffsetParam: pm.config.OffsetParam,
			LimitParam:  pm.config.LimitParam,
			Limit:       pm.config.Limit,
			MaxOffset:   pm.config.MaxPages * pm.config.Limit,
		}, nil
		
	case "cursor":
		return &CursorStrategy{
			BaseURL:        "",
			CursorParam:    pm.config.CursorParam,
			LimitParam:     pm.config.LimitParam,
			Limit:          pm.config.Limit,
			MaxPages:       pm.config.MaxPages,
			CursorSelector: pm.config.CursorSelector,
		}, nil
		
	case "next_button":
		return &NextButtonStrategy{
			Selector: pm.config.Selector,
			MaxPages: pm.config.MaxPages,
		}, nil
		
	case "numbered":
		return &NumberedPagesStrategy{
			BaseURL:  "",
			MaxPages: pm.config.MaxPages,
		}, nil
		
	default:
		return nil, fmt.Errorf("unknown pagination type: %s", pm.config.Type)
	}
}

// GetNextURL gets the next URL to scrape
func (pm *PaginationManager) GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error) {
	if pm.strategy == nil {
		return "", fmt.Errorf("pagination strategy not initialized")
	}
	
	return pm.strategy.GetNextURL(ctx, currentURL, doc, pageNum)
}

// IsComplete checks if pagination is complete
func (pm *PaginationManager) IsComplete(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) bool {
	if pm.strategy == nil {
		return true
	}
	
	return pm.strategy.IsComplete(ctx, currentURL, doc, pageNum)
}

// GetStrategyName returns the name of the current strategy
func (pm *PaginationManager) GetStrategyName() string {
	if pm.strategy == nil {
		return "none"
	}
	
	return pm.strategy.GetName()
}

// ValidatePaginationConfig validates pagination configuration
func ValidatePaginationConfig(config PaginationConfig) error {
	if config.Type == "" {
		return fmt.Errorf("pagination type is required")
	}
	
	switch config.Type {
	case "offset":
		if config.Limit <= 0 {
			return fmt.Errorf("limit must be greater than 0 for offset pagination")
		}
		
	case "cursor":
		if config.CursorSelector == "" {
			return fmt.Errorf("cursor_selector is required for cursor pagination")
		}
		
	case "next_button":
		if config.Selector == "" {
			return fmt.Errorf("selector is required for next_button pagination")
		}
		
	case "numbered":
		// No additional validation needed for numbered pagination
		
	default:
		return fmt.Errorf("unsupported pagination type: %s", config.Type)
	}
	
	if config.MaxPages < 0 {
		return fmt.Errorf("max_pages cannot be negative")
	}
	
	return nil
}

// SimpleOffsetPagination provides a simple offset-based pagination implementation
func SimpleOffsetPagination(baseURL string, pageNum int, limit int) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	
	offset := pageNum * limit
	query := u.Query()
	query.Set("offset", strconv.Itoa(offset))
	query.Set("limit", strconv.Itoa(limit))
	u.RawQuery = query.Encode()
	
	return u.String(), nil
}

// SimplePagedPagination provides simple page-based pagination  
func SimplePagedPagination(baseURL string, pageNum int) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	
	query := u.Query()
	query.Set("page", strconv.Itoa(pageNum))
	u.RawQuery = query.Encode()
	
	return u.String(), nil
}
