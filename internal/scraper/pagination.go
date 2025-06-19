package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valpere/DataScrapexter/pkg/api"
)

// PaginationHandler manages pagination logic for scraping multiple pages
type PaginationHandler struct {
	config      *api.PaginationConfig
	engine      *Engine
	baseURL     *url.URL
	currentPage int
	visitedURLs map[string]bool
}

// NewPaginationHandler creates a new pagination handler
func NewPaginationHandler(config *api.PaginationConfig, engine *Engine, baseURL string) (*PaginationHandler, error) {
	if config == nil {
		return nil, fmt.Errorf("pagination config is required")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &PaginationHandler{
		config:      config,
		engine:      engine,
		baseURL:     parsedURL,
		currentPage: 1,
		visitedURLs: make(map[string]bool),
	}, nil
}

// GetNextPages retrieves all pages based on pagination configuration
func (ph *PaginationHandler) GetNextPages(ctx context.Context, firstResult *Result) ([]string, error) {
	if firstResult == nil {
		return nil, fmt.Errorf("first result is required for pagination")
	}

	// Mark first page as visited
	ph.visitedURLs[firstResult.URL] = true

	switch ph.config.Type {
	case "next_button":
		return ph.handleNextButtonPagination(ctx, firstResult.URL)
	case "page_numbers":
		return ph.handlePageNumberPagination(ctx)
	case "url_pattern":
		return ph.handleURLPatternPagination(ctx)
	case "infinite_scroll":
		return nil, fmt.Errorf("infinite scroll pagination requires browser automation (coming in v0.5)")
	default:
		return nil, fmt.Errorf("unsupported pagination type: %s", ph.config.Type)
	}
}

// handleNextButtonPagination follows "next" button links
func (ph *PaginationHandler) handleNextButtonPagination(ctx context.Context, startURL string) ([]string, error) {
	var urls []string
	currentURL := startURL
	pagesProcessed := 1 // First page already processed

	for pagesProcessed < ph.config.MaxPages {
		select {
		case <-ctx.Done():
			return urls, ctx.Err()
		default:
		}

		// Fetch the current page to find next button
		nextURL, err := ph.findNextButtonURL(ctx, currentURL)
		if err != nil {
			return urls, fmt.Errorf("failed to find next button: %w", err)
		}

		if nextURL == "" {
			// No more pages
			break
		}

		// Check if we've already visited this URL (avoid loops)
		if ph.visitedURLs[nextURL] {
			break
		}

		ph.visitedURLs[nextURL] = true
		urls = append(urls, nextURL)
		currentURL = nextURL
		pagesProcessed++
	}

	return urls, nil
}

// findNextButtonURL finds the URL of the next page button
func (ph *PaginationHandler) findNextButtonURL(ctx context.Context, pageURL string) (string, error) {
	// Apply rate limiting
	if err := ph.engine.rateLimiter.Wait(ctx); err != nil {
		return "", err
	}

	// Fetch the page
	resp, err := ph.engine.doRequestWithRetry(ctx, pageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// Find next button using selector
	selection := doc.Find(ph.config.Selector).First()
	if selection.Length() == 0 {
		return "", nil // No next button found
	}

	// Extract href attribute
	href, exists := selection.Attr("href")
	if !exists {
		// Check if it's an anchor tag inside the selected element
		anchor := selection.Find("a").First()
		if anchor.Length() > 0 {
			href, exists = anchor.Attr("href")
		}

		// If still no href, check if the selector itself is an anchor
		if !exists && selection.Is("a") {
			href, exists = selection.Attr("href")
		}

		if !exists {
			return "", nil
		}
	}

	// Resolve relative URLs
	nextURL, err := ph.resolveURL(href)
	if err != nil {
		return "", fmt.Errorf("failed to resolve URL: %w", err)
	}

	return nextURL, nil
}

// handlePageNumberPagination generates URLs for numbered pages
func (ph *PaginationHandler) handlePageNumberPagination(ctx context.Context) ([]string, error) {
	var urls []string

	startPage := ph.config.StartPage
	if startPage <= 0 {
		startPage = 1
	}

	// If URL pattern is provided, use it
	if ph.config.URLPattern != "" {
		for page := startPage + 1; page <= ph.config.MaxPages; page++ {
			select {
			case <-ctx.Done():
				return urls, ctx.Err()
			default:
			}

			pageURL := strings.ReplaceAll(ph.config.URLPattern, "{page}", strconv.Itoa(page))
			pageURL = strings.ReplaceAll(pageURL, "{PAGE}", strconv.Itoa(page))

			resolvedURL, err := ph.resolveURL(pageURL)
			if err != nil {
				return urls, fmt.Errorf("failed to resolve page URL: %w", err)
			}

			urls = append(urls, resolvedURL)
		}
	} else {
		// Try to detect page parameter from base URL
		urls = ph.generatePageURLs(startPage+1, ph.config.MaxPages)
	}

	return urls, nil
}

// handleURLPatternPagination generates URLs based on a pattern
func (ph *PaginationHandler) handleURLPatternPagination(ctx context.Context) ([]string, error) {
	if ph.config.URLPattern == "" {
		return nil, fmt.Errorf("URL pattern is required for pattern-based pagination")
	}

	var urls []string
	startPage := ph.config.StartPage
	if startPage <= 0 {
		startPage = 1
	}

	for page := startPage + 1; page <= ph.config.MaxPages; page++ {
		select {
		case <-ctx.Done():
			return urls, ctx.Err()
		default:
		}

		// Replace placeholders in pattern
		pageURL := ph.config.URLPattern
		pageURL = strings.ReplaceAll(pageURL, "{page}", strconv.Itoa(page))
		pageURL = strings.ReplaceAll(pageURL, "{PAGE}", strconv.Itoa(page))
		pageURL = strings.ReplaceAll(pageURL, "{offset}", strconv.Itoa((page-1)*10)) // Assuming 10 items per page
		pageURL = strings.ReplaceAll(pageURL, "{OFFSET}", strconv.Itoa((page-1)*10))

		resolvedURL, err := ph.resolveURL(pageURL)
		if err != nil {
			return urls, fmt.Errorf("failed to resolve pattern URL: %w", err)
		}

		urls = append(urls, resolvedURL)
	}

	return urls, nil
}

// generatePageURLs generates numbered page URLs by detecting common patterns
func (ph *PaginationHandler) generatePageURLs(startPage, maxPages int) []string {
	var urls []string
	baseURL := ph.baseURL.String()

	// Common page parameter patterns
	pageParams := []string{"page", "p", "pg", "offset", "start"}

	// Check if base URL already has query parameters
	if ph.baseURL.RawQuery != "" {
		// Try to detect existing page parameter
		values := ph.baseURL.Query()

		for _, param := range pageParams {
			if values.Has(param) {
				// Found page parameter, generate URLs with it
				for page := startPage; page <= maxPages; page++ {
					newURL := *ph.baseURL
					q := newURL.Query()
					q.Set(param, strconv.Itoa(page))
					newURL.RawQuery = q.Encode()
					urls = append(urls, newURL.String())
				}
				return urls
			}
		}
	}

	// No page parameter found, try common patterns
	for page := startPage; page <= maxPages; page++ {
		// Try path-based pagination first (e.g., /page/2/)
		if strings.Contains(baseURL, "/page/") {
			// Replace existing page number
			newURL := strings.ReplaceAll(baseURL, "/page/1/", fmt.Sprintf("/page/%d/", page))
			urls = append(urls, newURL)
		} else if strings.HasSuffix(baseURL, "/") {
			// Add page to path
			urls = append(urls, fmt.Sprintf("%spage/%d/", baseURL, page))
		} else {
			// Use query parameter
			separator := "?"
			if strings.Contains(baseURL, "?") {
				separator = "&"
			}
			urls = append(urls, fmt.Sprintf("%s%spage=%d", baseURL, separator, page))
		}
	}

	return urls
}

// resolveURL resolves a potentially relative URL to an absolute URL
func (ph *PaginationHandler) resolveURL(href string) (string, error) {
	if href == "" {
		return "", nil
	}

	// Parse the href
	parsedHref, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	// Resolve against base URL
	resolvedURL := ph.baseURL.ResolveReference(parsedHref)
	return resolvedURL.String(), nil
}

// PaginatedScraper handles scraping with pagination
type PaginatedScraper struct {
	engine            *Engine
	config            *api.ScraperConfig
	extractors        []FieldExtractor
	paginationHandler *PaginationHandler
}

// NewPaginatedScraper creates a new paginated scraper
func NewPaginatedScraper(engine *Engine, config *api.ScraperConfig, extractors []FieldExtractor) (*PaginatedScraper, error) {
	if config.Pagination == nil {
		return nil, fmt.Errorf("pagination config is required")
	}

	paginationHandler, err := NewPaginationHandler(config.Pagination, engine, config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pagination handler: %w", err)
	}

	return &PaginatedScraper{
		engine:            engine,
		config:            config,
		extractors:        extractors,
		paginationHandler: paginationHandler,
	}, nil
}

// ScrapeAll scrapes all pages according to pagination configuration
func (ps *PaginatedScraper) ScrapeAll(ctx context.Context) ([]*Result, error) {
	var allResults []*Result

	// Scrape first page
	firstResult, err := ps.engine.Scrape(ctx, ps.config.BaseURL, ps.extractors)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape first page: %w", err)
	}
	allResults = append(allResults, firstResult)

	// Get next pages
	nextPages, err := ps.paginationHandler.GetNextPages(ctx, firstResult)
	if err != nil {
		// Log error but continue with results we have
		fmt.Printf("Warning: pagination error: %v\n", err)
		return allResults, nil
	}

	// Scrape remaining pages
	for i, pageURL := range nextPages {
		select {
		case <-ctx.Done():
			return allResults, ctx.Err()
		default:
		}

		// Log progress
		fmt.Printf("Scraping page %d of %d: %s\n", i+2, len(nextPages)+1, pageURL)

		// Scrape the page
		result, err := ps.engine.Scrape(ctx, pageURL, ps.extractors)
		if err != nil {
			// Log error but continue
			fmt.Printf("Error scraping page %s: %v\n", pageURL, err)
			continue
		}

		allResults = append(allResults, result)

		// Small delay between pages (in addition to rate limiting)
		time.Sleep(100 * time.Millisecond)
	}

	return allResults, nil
}

// PaginationInfo provides information about pagination progress
type PaginationInfo struct {
	CurrentPage  int
	TotalPages   int
	ItemsPerPage int
	TotalItems   int
	HasNextPage  bool
	NextPageURL  string
}

// ExtractPaginationInfo extracts pagination information from a page
func ExtractPaginationInfo(doc *goquery.Document, config *api.PaginationConfig) (*PaginationInfo, error) {
	info := &PaginationInfo{
		CurrentPage: 1,
	}

	// Try to extract pagination info based on common patterns
	// This is a simplified version - can be extended based on specific needs

	// Look for page numbers
	pageNumbers := doc.Find(".pagination .page-number, .pager .page, nav[aria-label='pagination'] a")
	if pageNumbers.Length() > 0 {
		lastPage := 1
		pageNumbers.Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if pageNum, err := strconv.Atoi(text); err == nil && pageNum > lastPage {
				lastPage = pageNum
			}
		})
		info.TotalPages = lastPage
	}

	// Check for next button
	if config != nil && config.Selector != "" {
		nextButton := doc.Find(config.Selector)
		info.HasNextPage = nextButton.Length() > 0

		if href, exists := nextButton.Attr("href"); exists {
			info.NextPageURL = href
		}
	}

	// Try to extract current page from URL or page content
	currentPageText := doc.Find(".current-page, .active.page, [aria-current='page']").First().Text()
	if current, err := strconv.Atoi(strings.TrimSpace(currentPageText)); err == nil {
		info.CurrentPage = current
	}

	return info, nil
}
