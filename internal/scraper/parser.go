// internal/scraper/parser.go
package scraper

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// HTMLParser handles HTML parsing with GoQuery
type HTMLParser struct {
	document *goquery.Document
}

// NewHTMLParser creates a new HTML parser from HTML content
func NewHTMLParser(html string) (*HTMLParser, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return &HTMLParser{
		document: doc,
	}, nil
}

// NewHTMLParserFromDocument creates a parser from existing goquery document
func NewHTMLParserFromDocument(doc *goquery.Document) *HTMLParser {
	return &HTMLParser{
		document: doc,
	}
}

// ExtractField extracts a field value using CSS selector
func (p *HTMLParser) ExtractField(config FieldConfig) (interface{}, error) {
	selection := p.document.Find(config.Selector)
	
	if selection.Length() == 0 {
		if config.Required {
			return nil, fmt.Errorf("required field '%s' not found with selector '%s'", config.Name, config.Selector)
		}
		return p.getDefaultValue(config), nil
	}

	switch config.Type {
	case "text":
		return p.extractText(selection)
	case "html":
		return p.extractHTML(selection)
	case "attribute":
		return p.extractAttribute(selection, config.Attribute)
	case "href":
		return p.extractAttribute(selection, "href")
	case "src":
		return p.extractAttribute(selection, "src")
	case "int", "number":
		return p.extractInt(selection)
	case "float":
		return p.extractFloat(selection)
	case "bool", "boolean":
		return p.extractBool(selection)
	case "array":
		return p.extractArray(selection)
	case "date":
		return p.extractDate(selection)
	default:
		return nil, fmt.Errorf("unsupported field type: %s", config.Type)
	}
}

// extractText extracts text content from selection
func (p *HTMLParser) extractText(selection *goquery.Selection) (string, error) {
	text := strings.TrimSpace(selection.First().Text())
	return text, nil
}

// extractHTML extracts HTML content from selection
func (p *HTMLParser) extractHTML(selection *goquery.Selection) (string, error) {
	html, err := selection.First().Html()
	if err != nil {
		return "", fmt.Errorf("failed to extract HTML: %w", err)
	}
	return strings.TrimSpace(html), nil
}

// extractAttribute extracts attribute value from selection
func (p *HTMLParser) extractAttribute(selection *goquery.Selection, attribute string) (string, error) {
	if attribute == "" {
		return "", fmt.Errorf("attribute name is required")
	}
	
	value, exists := selection.First().Attr(attribute)
	if !exists {
		return "", fmt.Errorf("attribute '%s' not found", attribute)
	}
	return strings.TrimSpace(value), nil
}

// extractInt extracts integer value from selection
func (p *HTMLParser) extractInt(selection *goquery.Selection) (int, error) {
	text := strings.TrimSpace(selection.First().Text())
	if text == "" {
		return 0, fmt.Errorf("empty text for integer extraction")
	}
	
	// Clean the text (remove common non-numeric characters)
	cleaned := strings.ReplaceAll(text, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	
	value, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer from '%s': %w", text, err)
	}
	return value, nil
}

// extractFloat extracts float value from selection
func (p *HTMLParser) extractFloat(selection *goquery.Selection) (float64, error) {
	text := strings.TrimSpace(selection.First().Text())
	if text == "" {
		return 0.0, fmt.Errorf("empty text for float extraction")
	}
	
	// Clean the text (remove common non-numeric characters except decimal point)
	cleaned := strings.ReplaceAll(text, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0.0, fmt.Errorf("failed to parse float from '%s': %w", text, err)
	}
	return value, nil
}

// extractBool extracts boolean value from selection
func (p *HTMLParser) extractBool(selection *goquery.Selection) (bool, error) {
	text := strings.ToLower(strings.TrimSpace(selection.First().Text()))
	
	switch text {
	case "true", "yes", "1", "on", "enabled", "active":
		return true, nil
	case "false", "no", "0", "off", "disabled", "inactive":
		return false, nil
	default:
		return false, fmt.Errorf("cannot parse boolean from: %s", text)
	}
}

// extractArray extracts array of values from selection
func (p *HTMLParser) extractArray(selection *goquery.Selection) ([]interface{}, error) {
	var results []interface{}
	
	selection.Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			results = append(results, text)
		}
	})
	
	return results, nil
}

// extractDate extracts date value from selection
func (p *HTMLParser) extractDate(selection *goquery.Selection) (time.Time, error) {
	text := strings.TrimSpace(selection.First().Text())
	if text == "" {
		return time.Time{}, fmt.Errorf("empty text for date extraction")
	}
	
	// Try common date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"January 2, 2006",
		"Jan 2, 2006",
		"02/01/2006",
		"01/02/2006",
		"2/1/2006",
		"1/2/2006",
		"2006/01/02",
		"02-01-2006",
		"01-02-2006",
	}
	
	for _, format := range formats {
		if date, err := time.Parse(format, text); err == nil {
			return date, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("failed to parse date from '%s'", text)
}

// getDefaultValue returns default value for field type
func (p *HTMLParser) getDefaultValue(config FieldConfig) interface{} {
	if config.Default != nil {
		return config.Default
	}
	
	switch config.Type {
	case "int", "number":
		return 0
	case "float":
		return 0.0
	case "bool", "boolean":
		return false
	case "array":
		return []interface{}{}
	case "date":
		return time.Time{}
	default:
		return ""
	}
}

// FindElements finds elements matching CSS selector
func (p *HTMLParser) FindElements(selector string) *goquery.Selection {
	return p.document.Find(selector)
}

// GetDocument returns the underlying goquery document
func (p *HTMLParser) GetDocument() *goquery.Document {
	return p.document
}

// GetTitle extracts page title
func (p *HTMLParser) GetTitle() string {
	return strings.TrimSpace(p.document.Find("title").First().Text())
}

// GetMetaContent extracts meta tag content
func (p *HTMLParser) GetMetaContent(name string) string {
	selector := fmt.Sprintf(`meta[name="%s"]`, name)
	content, _ := p.document.Find(selector).Attr("content")
	return strings.TrimSpace(content)
}

// GetLinks extracts all links from the page
func (p *HTMLParser) GetLinks() []string {
	var links []string
	p.document.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && href != "" {
			links = append(links, strings.TrimSpace(href))
		}
	})
	return links
}

// GetImages extracts all image sources
func (p *HTMLParser) GetImages() []string {
	var images []string
	p.document.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			images = append(images, strings.TrimSpace(src))
		}
	})
	return images
}

// ValidateSelector checks if a CSS selector is valid by testing it
func (p *HTMLParser) ValidateSelector(selector string) error {
	selection := p.document.Find(selector)
	if selection.Length() == 0 {
		return fmt.Errorf("selector '%s' matches no elements", selector)
	}
	return nil
}

// GetElementCount returns the number of elements matching selector
func (p *HTMLParser) GetElementCount(selector string) int {
	return p.document.Find(selector).Length()
}

// ExtractTable extracts table data as structured map
func (p *HTMLParser) ExtractTable(tableSelector string) ([]map[string]interface{}, error) {
	table := p.document.Find(tableSelector).First()
	if table.Length() == 0 {
		return nil, fmt.Errorf("table not found with selector: %s", tableSelector)
	}

	var headers []string
	var rows []map[string]interface{}

	// Extract headers from thead or first row
	headerRow := table.Find("thead tr").First()
	if headerRow.Length() == 0 {
		headerRow = table.Find("tr").First()
	}
	
	headerRow.Find("th, td").Each(func(i int, s *goquery.Selection) {
		header := strings.TrimSpace(s.Text())
		if header != "" {
			headers = append(headers, header)
		}
	})

	// If no headers found, use generic column names
	if len(headers) == 0 {
		firstRow := table.Find("tr").First()
		firstRow.Find("td").Each(func(i int, s *goquery.Selection) {
			headers = append(headers, fmt.Sprintf("column_%d", i+1))
		})
	}

	// Extract data rows (skip header row)
	dataRows := table.Find("tbody tr")
	if dataRows.Length() == 0 {
		// No tbody, use all rows except first (header)
		dataRows = table.Find("tr").Slice(1, goquery.ToEnd)
	}

	dataRows.Each(func(i int, row *goquery.Selection) {
		rowData := make(map[string]interface{})
		row.Find("td, th").Each(func(j int, cell *goquery.Selection) {
			if j < len(headers) {
				cellText := strings.TrimSpace(cell.Text())
				rowData[headers[j]] = cellText
			}
		})

		if len(rowData) > 0 {
			rows = append(rows, rowData)
		}
	})

	return rows, nil
}
