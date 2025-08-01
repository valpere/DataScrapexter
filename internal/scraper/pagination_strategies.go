// internal/scraper/pagination_strategies.go
package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// PaginationStrategy defines the interface for pagination strategies
type PaginationStrategy interface {
	// GetNextURL returns the next URL to scrape, or empty string if done
	GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error)

	// IsComplete returns true if pagination is complete
	IsComplete(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) bool

	// GetName returns the strategy name
	GetName() string
}

// OffsetStrategy implements offset-based pagination (e.g., ?offset=20&limit=10)
type OffsetStrategy struct {
	BaseURL    string `yaml:"base_url" json:"base_url"`
	OffsetParam string `yaml:"offset_param" json:"offset_param"` // Default: "offset"
	LimitParam  string `yaml:"limit_param" json:"limit_param"`   // Default: "limit"
	Limit      int    `yaml:"limit" json:"limit"`               // Items per page
	MaxOffset  int    `yaml:"max_offset" json:"max_offset"`     // Maximum offset to prevent infinite loops
	StartOffset int   `yaml:"start_offset" json:"start_offset"` // Starting offset (default: 0)
}

// GetNextURL generates the next URL using offset pagination
func (os *OffsetStrategy) GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error) {
	if os.OffsetParam == "" {
		os.OffsetParam = "offset"
	}
	if os.LimitParam == "" {
		os.LimitParam = "limit"
	}
	if os.Limit <= 0 {
		os.Limit = 10 // Default limit
	}

	baseURL := os.BaseURL
	if baseURL == "" {
		baseURL = currentURL
	}

	// Parse the base URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Calculate next offset
	nextOffset := os.StartOffset + (pageNum * os.Limit)

	// Check if we've reached the maximum offset
	if os.MaxOffset > 0 && nextOffset >= os.MaxOffset {
		return "", nil
	}

	// Add/update query parameters
	query := u.Query()
	query.Set(os.OffsetParam, strconv.Itoa(nextOffset))
	query.Set(os.LimitParam, strconv.Itoa(os.Limit))
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// IsComplete checks if offset pagination is complete
func (os *OffsetStrategy) IsComplete(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) bool {
	// Check if we've hit the maximum offset
	if os.MaxOffset > 0 {
		currentOffset := os.StartOffset + ((pageNum - 1) * os.Limit)
		return currentOffset >= os.MaxOffset
	}

	// If no max offset specified, check if the page has expected content
	// This is a heuristic - you might want to implement specific logic
	return false
}

// GetName returns the strategy name
func (os *OffsetStrategy) GetName() string {
	return "offset"
}

// CursorStrategy implements cursor-based pagination (e.g., ?cursor=xyz&limit=10)
type CursorStrategy struct {
	BaseURL     string `yaml:"base_url" json:"base_url"`
	CursorParam string `yaml:"cursor_param" json:"cursor_param"` // Default: "cursor"
	LimitParam  string `yaml:"limit_param" json:"limit_param"`   // Default: "limit"
	Limit       int    `yaml:"limit" json:"limit"`               // Items per page
	MaxPages    int    `yaml:"max_pages" json:"max_pages"`       // Maximum pages to prevent infinite loops

	// Cursor extraction configuration
	CursorSelector string `yaml:"cursor_selector" json:"cursor_selector"`     // CSS selector to find next cursor
	CursorAttr     string `yaml:"cursor_attr" json:"cursor_attr"`             // Attribute containing cursor value
	CursorPattern  string `yaml:"cursor_pattern" json:"cursor_pattern"`       // Regex pattern to extract cursor

	lastCursor string // Internal state to track the last cursor
}

// GetNextURL generates the next URL using cursor pagination
func (cs *CursorStrategy) GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error) {
	if cs.CursorParam == "" {
		cs.CursorParam = "cursor"
	}
	if cs.LimitParam == "" {
		cs.LimitParam = "limit"
	}
	if cs.Limit <= 0 {
		cs.Limit = 10 // Default limit
	}

	// Check page limit
	if cs.MaxPages > 0 && pageNum > cs.MaxPages {
		return "", nil
	}

	baseURL := cs.BaseURL
	if baseURL == "" {
		baseURL = currentURL
	}

	// Extract the next cursor from the document
	nextCursor, err := cs.extractCursor(doc)
	if err != nil {
		return "", fmt.Errorf("failed to extract cursor: %w", err)
	}

	// If no cursor found, pagination is complete
	if nextCursor == "" {
		return "", nil
	}

	// Check if cursor is the same as last time (infinite loop protection)
	if nextCursor == cs.lastCursor {
		return "", nil
	}

	cs.lastCursor = nextCursor

	// Parse the base URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Add/update query parameters
	query := u.Query()
	query.Set(cs.CursorParam, nextCursor)
	query.Set(cs.LimitParam, strconv.Itoa(cs.Limit))
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// extractCursor extracts the next cursor value from the document
func (cs *CursorStrategy) extractCursor(doc *goquery.Document) (string, error) {
	if cs.CursorSelector == "" {
		return "", fmt.Errorf("cursor_selector is required for cursor strategy")
	}

	// Find the element containing the cursor
	selection := doc.Find(cs.CursorSelector)
	if selection.Length() == 0 {
		return "", nil // No cursor found, pagination complete
	}

	var cursor string
	if cs.CursorAttr != "" {
		// Extract from attribute
		cursor, _ = selection.Attr(cs.CursorAttr)
	} else {
		// Extract from text content
		cursor = selection.Text()
		cursor = strings.TrimSpace(cursor)
	}

	return cursor, nil
}

// IsComplete checks if cursor pagination is complete
func (cs *CursorStrategy) IsComplete(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) bool {
	// Check page limit
	if cs.MaxPages > 0 && pageNum > cs.MaxPages {
		return true
	}

	// Check if we can extract a cursor
	cursor, _ := cs.extractCursor(doc)
	return cursor == "" || cursor == cs.lastCursor
}

// GetName returns the strategy name
func (cs *CursorStrategy) GetName() string {
	return "cursor"
}

// NextButtonStrategy implements next button-based pagination
type NextButtonStrategy struct {
	Selector     string `yaml:"selector" json:"selector"`         // CSS selector for next button
	MaxPages     int    `yaml:"max_pages" json:"max_pages"`       // Maximum pages
	DisabledAttr string `yaml:"disabled_attr" json:"disabled_attr"` // Attribute that indicates disabled state
	DisabledClass string `yaml:"disabled_class" json:"disabled_class"` // Class that indicates disabled state
}

// GetNextURL finds the next URL using a next button
func (nbs *NextButtonStrategy) GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error) {
	if nbs.MaxPages > 0 && pageNum > nbs.MaxPages {
		return "", nil
	}

	if nbs.Selector == "" {
		return "", fmt.Errorf("selector is required for next_button strategy")
	}

	// Find the next button
	selection := doc.Find(nbs.Selector)
	if selection.Length() == 0 {
		return "", nil // No next button found
	}

	// Check if button is disabled
	if nbs.DisabledAttr != "" {
		if _, exists := selection.Attr(nbs.DisabledAttr); exists {
			return "", nil // Button is disabled
		}
	}

	if nbs.DisabledClass != "" {
		if selection.HasClass(nbs.DisabledClass) {
			return "", nil // Button has disabled class
		}
	}

	// Extract the URL
	href, exists := selection.Attr("href")
	if !exists {
		return "", fmt.Errorf("next button has no href attribute")
	}

	// Convert relative URL to absolute
	currentU, err := url.Parse(currentURL)
	if err != nil {
		return "", fmt.Errorf("invalid current URL: %w", err)
	}

	nextU, err := currentU.Parse(href)
	if err != nil {
		return "", fmt.Errorf("invalid next URL: %w", err)
	}

	return nextU.String(), nil
}

// IsComplete checks if next button pagination is complete
func (nbs *NextButtonStrategy) IsComplete(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) bool {
	if nbs.MaxPages > 0 && pageNum > nbs.MaxPages {
		return true
	}

	selection := doc.Find(nbs.Selector)
	if selection.Length() == 0 {
		return true // No next button
	}

	// Check if disabled by attribute
	if nbs.DisabledAttr != "" {
		if _, exists := selection.Attr(nbs.DisabledAttr); exists {
			return true
		}
	}

	// Check if disabled by class
	if nbs.DisabledClass != "" {
		if selection.HasClass(nbs.DisabledClass) {
			return true
		}
	}

	// Check for common disabled patterns
	if selection.HasClass("disabled") {
		return true
	}

	// Check if it's a span instead of a link (common pattern for disabled buttons)
	if selection.Is("span") {
		return true
	}

	// Check if href is missing or empty
	href, exists := selection.Attr("href")
	if !exists || href == "" || href == "#" {
		return true
	}

	return false
}

// GetName returns the strategy name
func (nbs *NextButtonStrategy) GetName() string {
	return "next_button"
}

// NumberedPagesStrategy implements numbered page pagination (1, 2, 3, ...)
type NumberedPagesStrategy struct {
	BaseURL   string `yaml:"base_url" json:"base_url"`
	PageParam string `yaml:"page_param" json:"page_param"` // Default: "page"
	StartPage int    `yaml:"start_page" json:"start_page"` // Default: 1
	MaxPages  int    `yaml:"max_pages" json:"max_pages"`   // Maximum pages
}

// GetNextURL generates the next numbered page URL
func (nps *NumberedPagesStrategy) GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error) {
	if nps.PageParam == "" {
		nps.PageParam = "page"
	}
	if nps.StartPage <= 0 {
		nps.StartPage = 1
	}

	nextPageNum := nps.StartPage + pageNum

	if nps.MaxPages > 0 && nextPageNum > nps.MaxPages {
		return "", nil
	}

	baseURL := nps.BaseURL
	if baseURL == "" {
		baseURL = currentURL
	}

	// Check if baseURL contains template patterns like {page} or {PAGE}
	if strings.Contains(baseURL, "{page}") || strings.Contains(baseURL, "{PAGE}") {
		// Handle URL template pattern
		pageURL := strings.ReplaceAll(baseURL, "{page}", strconv.Itoa(nextPageNum))
		pageURL = strings.ReplaceAll(pageURL, "{PAGE}", strconv.Itoa(nextPageNum))
		return pageURL, nil
	}

	// Parse the base URL for query parameter approach
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Add/update page parameter
	query := u.Query()
	query.Set(nps.PageParam, strconv.Itoa(nextPageNum))
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// IsComplete checks if numbered pagination is complete
func (nps *NumberedPagesStrategy) IsComplete(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) bool {
	if nps.MaxPages > 0 {
		nextPageNum := nps.StartPage + pageNum
		return nextPageNum > nps.MaxPages
	}
	return false
}

// GetName returns the strategy name
func (nps *NumberedPagesStrategy) GetName() string {
	return "numbered"
}

// CreatePaginationStrategy creates a pagination strategy from config
func CreatePaginationStrategy(config PaginationConfig) (PaginationStrategy, error) {
	switch config.Type {
	case PaginationTypeOffset:
		return &OffsetStrategy{
			BaseURL:     "",
			OffsetParam: config.OffsetParam,
			LimitParam:  config.LimitParam,
			Limit:       config.PageSize,
			MaxOffset:   config.MaxPages * config.PageSize,
		}, nil

	case PaginationTypePages, "numbered":
		return &NumberedPagesStrategy{
			BaseURL:   "",
			PageParam: config.PageParam,
			StartPage: config.StartPage,
			MaxPages:  config.MaxPages,
		}, nil

	case PaginationTypeNextButton:
		return &NextButtonStrategy{
			Selector: config.NextSelector,
			MaxPages: config.MaxPages,
		}, nil

	case PaginationTypeURLPattern:
		return &NumberedPagesStrategy{
			BaseURL:   config.URLTemplate,
			PageParam: "page",
			StartPage: config.StartPage,
			MaxPages:  config.MaxPages,
		}, nil

	case "cursor":
		return &CursorStrategy{
			BaseURL:        "",
			CursorParam:    config.PageParam,
			LimitParam:     config.LimitParam,
			Limit:          config.PageSize,
			MaxPages:       config.MaxPages,
			CursorSelector: config.ScrollSelector,
		}, nil

	default:
		return nil, fmt.Errorf("unknown pagination strategy: %s", config.Type)
	}
}
