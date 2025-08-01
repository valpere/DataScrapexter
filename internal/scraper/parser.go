// internal/scraper/parser.go
package scraper

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// HTMLParser provides HTML parsing and extraction capabilities
type HTMLParser struct {
	document *goquery.Document
	content  string
}

// NewHTMLParser creates a new HTML parser from HTML content
func NewHTMLParser(html string) (*HTMLParser, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return &HTMLParser{
		document: doc,
		content:  html,
	}, nil
}

// ExtractField extracts a field value based on the field configuration
func (hp *HTMLParser) ExtractField(config FieldConfig) (interface{}, error) {
	if config.Selector == "" {
		return nil, fmt.Errorf("selector cannot be empty")
	}

	selection := hp.document.Find(config.Selector)
	if selection.Length() == 0 {
		if config.Required {
			return nil, fmt.Errorf("required field '%s' not found with selector '%s'", config.Name, config.Selector)
		}
		return hp.getDefaultValue(config), nil
	}

	switch config.Type {
	case "text":
		return strings.TrimSpace(selection.First().Text()), nil
	case "html":
		html, err := selection.First().Html()
		if err != nil {
			return nil, fmt.Errorf("failed to extract HTML: %w", err)
		}
		return html, nil
	case "attr":
		if config.Attribute == "" {
			return nil, fmt.Errorf("attribute name required for attr type")
		}
		value, exists := selection.First().Attr(config.Attribute)
		if !exists {
			return nil, fmt.Errorf("attribute '%s' not found", config.Attribute)
		}
		return value, nil
	case "list":
		var items []string
		selection.Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				items = append(items, text)
			}
		})
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported field type: %s", config.Type)
	}
}

// ValidateSelector validates that a CSS selector is valid and returns results
func (hp *HTMLParser) ValidateSelector(selector string) error {
	if selector == "" {
		return fmt.Errorf("selector cannot be empty")
	}

	selection := hp.document.Find(selector)
	if selection.Length() == 0 {
		return fmt.Errorf("selector '%s' returned no results", selector)
	}

	return nil
}

// ExtractTable extracts table data from the document
func (hp *HTMLParser) ExtractTable(selector string) ([]map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("table selector cannot be empty")
	}

	table := hp.document.Find(selector).First()
	if table.Length() == 0 {
		return nil, fmt.Errorf("table not found with selector '%s'", selector)
	}

	var headers []string
	var rows []map[string]string

	// Extract headers
	table.Find("thead tr th, tr:first-child th, tr:first-child td").Each(func(i int, s *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(s.Text()))
	})

	// If no headers found in thead, use first row
	if len(headers) == 0 {
		table.Find("tr:first-child td").Each(func(i int, s *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(s.Text()))
		})
	}

	// Extract data rows (skip header row if it was in tbody)
	var dataRows *goquery.Selection
	if table.Find("tbody").Length() > 0 {
		dataRows = table.Find("tbody tr")
	} else {
		dataRows = table.Find("tr").Slice(1, goquery.ToEnd) // Skip first row (header)
	}

	dataRows.Each(func(i int, row *goquery.Selection) {
		rowData := make(map[string]string)
		row.Find("td").Each(func(j int, cell *goquery.Selection) {
			if j < len(headers) {
				rowData[headers[j]] = strings.TrimSpace(cell.Text())
			}
		})
		if len(rowData) > 0 {
			rows = append(rows, rowData)
		}
	})

	return rows, nil
}

// GetLinks extracts all links from the document
func (hp *HTMLParser) GetLinks() []map[string]string {
	var links []map[string]string

	hp.document.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		text := strings.TrimSpace(s.Text())

		links = append(links, map[string]string{
			"href": href,
			"text": text,
		})
	})

	return links
}

// GetImages extracts all images from the document
func (hp *HTMLParser) GetImages() []map[string]string {
	var images []map[string]string

	hp.document.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		alt, _ := s.Attr("alt")

		images = append(images, map[string]string{
			"src": src,
			"alt": alt,
		})
	})

	return images
}

// getDefaultValue returns the default value for a field
func (hp *HTMLParser) getDefaultValue(config FieldConfig) interface{} {
	if config.Default != nil {
		return config.Default
	}

	switch config.Type {
	case "text", "html", "attr":
		return ""
	case "list":
		return []string{}
	default:
		return ""
	}
}
