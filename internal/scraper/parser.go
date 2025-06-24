// internal/scraper/parser.go
package scraper

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// HTMLParser provides robust HTML parsing capabilities using GoQuery for CSS selector-based extraction
type HTMLParser struct {
	document *goquery.Document
	baseURL  string
}

// NewHTMLParser creates a new HTML parser instance from an HTTP response
func NewHTMLParser(response *http.Response) (*HTMLParser, error) {
	if response == nil {
		return nil, fmt.Errorf("HTTP response cannot be nil")
	}

	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML document: %w", err)
	}

	return &HTMLParser{
		document: document,
		baseURL:  response.Request.URL.String(),
	}, nil
}

// NewHTMLParserFromReader creates a parser from an io.Reader containing HTML content
func NewHTMLParserFromReader(reader io.Reader, baseURL string) (*HTMLParser, error) {
	if reader == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}

	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML document: %w", err)
	}

	return &HTMLParser{
		document: document,
		baseURL:  baseURL,
	}, nil
}

// NewHTMLParserFromString creates a parser from HTML string content
func NewHTMLParserFromString(html, baseURL string) (*HTMLParser, error) {
	return NewHTMLParserFromReader(strings.NewReader(html), baseURL)
}

// ExtractField extracts data from the document using a field configuration
func (p *HTMLParser) ExtractField(field FieldConfig) (interface{}, error) {
	if p.document == nil {
		return nil, fmt.Errorf("HTML document not initialized")
	}

	selection := p.document.Find(field.Selector)
	if selection.Length() == 0 {
		if field.Required {
			return nil, fmt.Errorf("required field '%s' not found with selector '%s'", field.Name, field.Selector)
		}
		return nil, nil
	}

	return p.extractValue(selection, field)
}

// extractValue extracts the appropriate value based on field type
func (p *HTMLParser) extractValue(selection *goquery.Selection, field FieldConfig) (interface{}, error) {
	switch field.Type {
	case "text":
		return p.extractText(selection), nil
	case "html":
		return p.extractHTML(selection), nil
	case "attribute":
		// For attribute extraction, we'll use a simple approach since the field doesn't have an Attribute field
		// This will extract the first common attribute if none specified
		return p.extractFirstAttribute(selection), nil
	case "href":
		return p.extractHref(selection), nil
	case "src":
		return p.extractSrc(selection), nil
	case "int", "number":
		text := p.extractText(selection)
		return p.parseInt(text)
	case "float":
		text := p.extractText(selection)
		return p.parseFloat(text)
	case "bool", "boolean":
		text := p.extractText(selection)
		return p.parseBool(text), nil
	case "date":
		text := p.extractText(selection)
		return p.parseDate(text, "2006-01-02") // Use default format
	case "array":
		return p.extractArray(selection), nil
	default:
		return p.extractText(selection), nil
	}
}

// extractText extracts and cleans text content
func (p *HTMLParser) extractText(selection *goquery.Selection) string {
	text := selection.Text()
	// Clean up whitespace
	text = strings.TrimSpace(text)
	// Normalize multiple whitespace to single space
	text = strings.Join(strings.Fields(text), " ")
	return text
}

// extractHTML extracts HTML content
func (p *HTMLParser) extractHTML(selection *goquery.Selection) string {
	html, _ := selection.Html()
	return strings.TrimSpace(html)
}

// extractFirstAttribute extracts the first available common attribute
func (p *HTMLParser) extractFirstAttribute(selection *goquery.Selection) string {
	// Common attributes to check in order
	commonAttrs := []string{"href", "src", "alt", "title", "class", "id", "data-value", "value"}
	
	for _, attr := range commonAttrs {
		if value, exists := selection.Attr(attr); exists && value != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// extractHref extracts href attribute (convenience method)
func (p *HTMLParser) extractHref(selection *goquery.Selection) string {
	attr, _ := selection.Attr("href")
	return strings.TrimSpace(attr)
}

// extractSrc extracts src attribute (convenience method)
func (p *HTMLParser) extractSrc(selection *goquery.Selection) string {
	attr, _ := selection.Attr("src")
	return strings.TrimSpace(attr)
}

// parseInt parses text to integer
func (p *HTMLParser) parseInt(text string) (int, error) {
	if text == "" {
		return 0, fmt.Errorf("empty text cannot be parsed as integer")
	}
	
	// Clean text for parsing (remove commas, currency symbols, etc.)
	cleaned := strings.ReplaceAll(text, ",", "")
	cleaned = strings.ReplaceAll(cleaned, "$", "")
	cleaned = strings.ReplaceAll(cleaned, "€", "")
	cleaned = strings.ReplaceAll(cleaned, "£", "")
	cleaned = strings.TrimSpace(cleaned)
	
	return strconv.Atoi(cleaned)
}

// parseFloat parses text to float64
func (p *HTMLParser) parseFloat(text string) (float64, error) {
	if text == "" {
		return 0.0, fmt.Errorf("empty text cannot be parsed as float")
	}
	
	// Clean text for parsing
	cleaned := strings.ReplaceAll(text, ",", "")
	cleaned = strings.ReplaceAll(cleaned, "$", "")
	cleaned = strings.ReplaceAll(cleaned, "€", "")
	cleaned = strings.ReplaceAll(cleaned, "£", "")
	cleaned = strings.TrimSpace(cleaned)
	
	return strconv.ParseFloat(cleaned, 64)
}

// parseBool parses text to boolean
func (p *HTMLParser) parseBool(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	switch text {
	case "true", "yes", "1", "on", "enabled", "active", "available", "in stock":
		return true
	default:
		return false
	}
}

// parseDate parses text to time.Time using default format
func (p *HTMLParser) parseDate(text, format string) (time.Time, error) {
	if text == "" {
		return time.Time{}, fmt.Errorf("empty text cannot be parsed as date")
	}
	
	if format == "" {
		format = "2006-01-02" // Default format
	}
	
	return time.Parse(format, strings.TrimSpace(text))
}

// extractArray extracts multiple values as an array of strings
func (p *HTMLParser) extractArray(selection *goquery.Selection) []interface{} {
	var results []interface{}
	
	selection.Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			results = append(results, text)
		}
	})
	
	return results
}

// GetDocument returns the underlying goquery document
func (p *HTMLParser) GetDocument() *goquery.Document {
	return p.document
}

// GetBaseURL returns the base URL for resolving relative URLs
func (p *HTMLParser) GetBaseURL() string {
	return p.baseURL
}

// Find returns a selection matching the CSS selector
func (p *HTMLParser) Find(selector string) *goquery.Selection {
	if p.document == nil {
		return &goquery.Selection{}
	}
	return p.document.Find(selector)
}

// ValidateSelector checks if a CSS selector is valid by attempting to use it
func (p *HTMLParser) ValidateSelector(selector string) bool {
	if p.document == nil {
		return false
	}
	
	// Try to use the selector - if it doesn't panic, it's valid
	defer func() {
		if recover() != nil {
			// Selector caused a panic, so it's invalid
		}
	}()
	
	_ = p.document.Find(selector)
	return true
}

// ExtractMultiple extracts data for multiple field configurations
func (p *HTMLParser) ExtractMultiple(fields []FieldConfig) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	var errors []string
	
	for _, field := range fields {
		value, err := p.ExtractField(field)
		if err != nil {
			if field.Required {
				return nil, fmt.Errorf("required field '%s' extraction failed: %w", field.Name, err)
			}
			errors = append(errors, fmt.Sprintf("field '%s': %v", field.Name, err))
			continue
		}
		
		if value != nil {
			results[field.Name] = value
		}
	}
	
	// Log non-fatal errors but don't fail the extraction
	if len(errors) > 0 {
		// In a real implementation, you would use a proper logger here
		// For now, we'll just continue without the failed fields
	}
	
	return results, nil
}
