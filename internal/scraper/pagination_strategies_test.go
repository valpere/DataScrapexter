// internal/scraper/pagination_strategies_test.go
package scraper

import (
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestOffsetStrategy_GetNextURL(t *testing.T) {
	ctx := context.Background()
	
	strategy := OffsetStrategy{
		BaseURL:     "https://example.com/search",
		OffsetParam: "offset",
		LimitParam:  "limit",
		Limit:       10,
	}
	
	result, err := strategy.GetNextURL(ctx, "https://example.com/search", nil, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	expected := "https://example.com/search?limit=10&offset=10"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestOffsetStrategy_IsComplete(t *testing.T) {
	ctx := context.Background()
	
	strategy := OffsetStrategy{
		Limit:     10,
		MaxOffset: 50,
	}
	
	// Should not be complete at page 3
	if strategy.IsComplete(ctx, "", nil, 3) {
		t.Errorf("expected pagination not to be complete at page 3")
	}
	
	// Should be complete at page 6 (offset 50)
	if !strategy.IsComplete(ctx, "", nil, 6) {
		t.Errorf("expected pagination to be complete at page 6")
	}
}

func TestCursorStrategy_extractCursor(t *testing.T) {
	strategy := CursorStrategy{
		CursorSelector: ".next-link",
		CursorAttr:     "data-cursor",
	}
	
	html := `<a class="next-link" data-cursor="abc123">Next</a>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}
	
	result, err := strategy.extractCursor(doc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	expected := "abc123"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNextButtonStrategy_GetNextURL(t *testing.T) {
	ctx := context.Background()
	
	strategy := NextButtonStrategy{
		Selector: ".next-btn",
	}
	
	html := `<a class="next-btn" href="/page/2">Next</a>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}
	
	result, err := strategy.GetNextURL(ctx, "https://example.com/page/1", doc, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	expected := "https://example.com/page/2"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCreatePaginationStrategy(t *testing.T) {
	config := PaginationConfig{
		Type: "offset",
	}
	
	strategy, err := CreatePaginationStrategy(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	if strategy == nil {
		t.Errorf("expected strategy but got nil")
	}
	
	if strategy.GetName() != "offset" {
		t.Errorf("expected strategy type %q, got %q", "offset", strategy.GetName())
	}
}

func TestValidatePaginationConfig(t *testing.T) {
	// Valid config
	config := PaginationConfig{
		Type:  "offset",
		Limit: 10,
	}
	
	err := ValidatePaginationConfig(config)
	if err != nil {
		t.Errorf("unexpected error for valid config: %v", err)
	}
	
	// Invalid config - missing type
	invalidConfig := PaginationConfig{
		Limit: 10,
	}
	
	err = ValidatePaginationConfig(invalidConfig)
	if err == nil {
		t.Errorf("expected error for invalid config but got none")
	}
}

func TestSimpleOffsetPagination(t *testing.T) {
	result, err := SimpleOffsetPagination("https://example.com/api", 2, 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	expected := "https://example.com/api?limit=10&offset=20"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func BenchmarkOffsetStrategy_GetNextURL(b *testing.B) {
	strategy := &OffsetStrategy{
		BaseURL:     "https://example.com/api",
		OffsetParam: "offset",
		LimitParam:  "limit",
		Limit:       10,
	}
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strategy.GetNextURL(ctx, "https://example.com/api", nil, i%100)
	}
}
