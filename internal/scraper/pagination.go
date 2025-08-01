// internal/scraper/pagination.go
package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// Note: PaginationConfig is now defined in types.go to avoid conflicts

// PaginationManager manages pagination across different strategies
type PaginationManager struct {
	config   PaginationConfig
	strategy PaginationStrategy
}

// Note: PaginationResult is now defined in types.go to avoid conflicts

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
	case PaginationTypeOffset:
		return &OffsetStrategy{
			BaseURL:     "",
			OffsetParam: pm.config.OffsetParam,
			LimitParam:  pm.config.LimitParam,
			Limit:       pm.config.PageSize,
			MaxOffset:   pm.config.MaxPages * pm.config.PageSize,
		}, nil

	case PaginationTypePages, "numbered":
		return &NumberedPagesStrategy{
			BaseURL:   "",
			PageParam: pm.config.PageParam,
			StartPage: pm.config.StartPage,
			MaxPages:  pm.config.MaxPages,
		}, nil

	case PaginationTypeNextButton:
		return &NextButtonStrategy{
			Selector: pm.config.NextSelector,
			MaxPages: pm.config.MaxPages,
		}, nil

	case PaginationTypeURLPattern:
		// For URL pattern, we'll use NumberedPagesStrategy but with URL template handling
		return &NumberedPagesStrategy{
			BaseURL:   pm.config.URLTemplate,
			PageParam: "page", // Will be replaced in URL template
			StartPage: pm.config.StartPage,
			MaxPages:  pm.config.MaxPages,
		}, nil

	case "cursor":
		return &CursorStrategy{
			BaseURL:        "",
			CursorParam:    pm.config.PageParam, // Reuse page param for cursor
			LimitParam:     pm.config.LimitParam,
			Limit:          pm.config.PageSize,
			MaxPages:       pm.config.MaxPages,
			CursorSelector: pm.config.ScrollSelector, // Reuse scroll selector for cursor
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
func ValidatePaginationConfig(config *PaginationConfig) error {
	if config.Type == "" {
		return fmt.Errorf("pagination type is required")
	}

	switch config.Type {
	case PaginationTypeOffset:
		if config.PageSize <= 0 {
			return fmt.Errorf("page_size must be greater than 0 for offset pagination")
		}
		if config.OffsetParam == "" {
			config.OffsetParam = "offset"
		}
		if config.LimitParam == "" {
			config.LimitParam = "limit"
		}

	case PaginationTypeNextButton:
		if config.NextSelector == "" {
			return fmt.Errorf("next_selector is required for next_button pagination")
		}

	case PaginationTypeURLPattern:
		if config.URLTemplate == "" {
			return fmt.Errorf("url_template is required for url_pattern pagination")
		}

	case PaginationTypePages:
		if config.PageParam == "" {
			config.PageParam = "page"
		}

	case PaginationTypeScrolling:
		if config.ScrollSelector == "" && config.LoadMoreSelector == "" {
			return fmt.Errorf("either scroll_selector or load_more_selector is required for scrolling pagination")
		}

	default:
		return fmt.Errorf("unsupported pagination type: %s", config.Type)
	}

	if config.MaxPages < 0 {
		return fmt.Errorf("max_pages cannot be negative")
	}

	if config.StartPage <= 0 {
		config.StartPage = 1
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
