// internal/scraper/parser_test.go
package scraper

import (
	"strings"
	"testing"
	"time"
)

func TestNewHTMLParser(t *testing.T) {
	html := `<html><body><h1>Test</h1></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	if parser == nil {
		t.Fatal("Parser should not be nil")
	}

	if parser.document == nil {
		t.Fatal("Document should not be nil")
	}
}

func TestExtractField_Text(t *testing.T) {
	html := `<html><body><h1>Test Title</h1><p class="content">Test content</p></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: true,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if result != "Test Title" {
		t.Errorf("Expected 'Test Title', got %v", result)
	}
}

func TestExtractField_HTML(t *testing.T) {
	html := `<html><body><div class="content"><p>Test <strong>content</strong></p></div></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "content",
		Selector: ".content",
		Type:     "html",
		Required: true,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	expected := "<p>Test <strong>content</strong></p>"
	if result != expected {
		t.Errorf("Expected '%s', got %v", expected, result)
	}
}

func TestExtractField_Attribute(t *testing.T) {
	html := `<html><body><a href="https://example.com" class="link">Link</a></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:      "url",
		Selector:  "a",
		Type:      "attribute",
		Attribute: "href",
		Required:  true,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if result != "https://example.com" {
		t.Errorf("Expected 'https://example.com', got %v", result)
	}
}

func TestExtractField_Int(t *testing.T) {
	html := `<html><body><span class="price">1,299</span></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "price",
		Selector: ".price",
		Type:     "int",
		Required: true,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if result != 1299 {
		t.Errorf("Expected 1299, got %v", result)
	}
}

func TestExtractField_Float(t *testing.T) {
	html := `<html><body><span class="rating">4.8</span></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "rating",
		Selector: ".rating",
		Type:     "float",
		Required: true,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if result != 4.8 {
		t.Errorf("Expected 4.8, got %v", result)
	}
}

func TestExtractField_Bool(t *testing.T) {
	html := `<html><body><span class="available">true</span><span class="enabled">yes</span></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		selector string
		expected bool
	}{
		{".available", true},
		{".enabled", true},
	}

	for _, test := range tests {
		config := FieldConfig{
			Name:     "status",
			Selector: test.selector,
			Type:     "bool",
			Required: true,
		}

		result, err := parser.ExtractField(config)
		if err != nil {
			t.Fatalf("Failed to extract field: %v", err)
		}

		if result != test.expected {
			t.Errorf("Expected %v, got %v", test.expected, result)
		}
	}
}

func TestExtractField_Array(t *testing.T) {
	html := `<html><body><ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "items",
		Selector: "li",
		Type:     "array",
		Required: true,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	items, ok := result.([]interface{})
	if !ok {
		t.Fatalf("Expected array, got %T", result)
	}

	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}

	expected := []string{"Item 1", "Item 2", "Item 3"}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected '%s', got '%v'", expected[i], item)
		}
	}
}

func TestExtractField_Date(t *testing.T) {
	html := `<html><body><time class="published">2025-06-25</time></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "published",
		Selector: ".published",
		Type:     "date",
		Required: true,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	date, ok := result.(time.Time)
	if !ok {
		t.Fatalf("Expected time.Time, got %T", result)
	}

	expected := time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC)
	if !date.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, date)
	}
}

func TestExtractField_NotFound_Required(t *testing.T) {
	html := `<html><body><p>No title here</p></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: true,
	}

	_, err = parser.ExtractField(config)
	if err == nil {
		t.Error("Expected error for missing required field")
	}
}

func TestExtractField_NotFound_Optional(t *testing.T) {
	html := `<html><body><p>No title here</p></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: false,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract optional field: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty string for missing optional field, got %v", result)
	}
}

func TestExtractField_WithDefault(t *testing.T) {
	html := `<html><body><p>No title here</p></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: false,
		Default:  "Default Title",
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if result != "Default Title" {
		t.Errorf("Expected 'Default Title', got %v", result)
	}
}

func TestGetTitle(t *testing.T) {
	html := `<html><head><title>Page Title</title></head><body></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	title := parser.GetTitle()
	if title != "Page Title" {
		t.Errorf("Expected 'Page Title', got '%s'", title)
	}
}

func TestGetMetaContent(t *testing.T) {
	html := `<html><head><meta name="description" content="Page description"></head><body></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	description := parser.GetMetaContent("description")
	if description != "Page description" {
		t.Errorf("Expected 'Page description', got '%s'", description)
	}
}

func TestGetLinks(t *testing.T) {
	html := `<html><body><a href="/page1">Link 1</a><a href="/page2">Link 2</a></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	links := parser.GetLinks()
	if len(links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(links))
	}

	expected := []string{"/page1", "/page2"}
	for i, link := range links {
		if link != expected[i] {
			t.Errorf("Expected '%s', got '%s'", expected[i], link)
		}
	}
}

func TestExtractTable(t *testing.T) {
	html := `
	<html>
		<body>
			<table>
				<thead>
					<tr><th>Name</th><th>Price</th><th>Rating</th></tr>
				</thead>
				<tbody>
					<tr><td>Product 1</td><td>$99.99</td><td>4.5</td></tr>
					<tr><td>Product 2</td><td>$149.99</td><td>4.8</td></tr>
				</tbody>
			</table>
		</body>
	</html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	rows, err := parser.ExtractTable("table")
	if err != nil {
		t.Fatalf("Failed to extract table: %v", err)
	}

	// Should extract data rows only, not header
	if len(rows) != 2 {
		t.Errorf("Expected 2 data rows, got %d", len(rows))
		for i, row := range rows {
			t.Logf("Row %d: %+v", i, row)
		}
		return
	}

	if rows[0]["Name"] != "Product 1" {
		t.Errorf("Expected 'Product 1', got '%v'", rows[0]["Name"])
	}

	if rows[1]["Price"] != "$149.99" {
		t.Errorf("Expected '$149.99', got '%v'", rows[1]["Price"])
	}
}

func TestValidateSelector(t *testing.T) {
	html := `<html><body><h1>Title</h1><p class="content">Content</p></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Valid selector
	err = parser.ValidateSelector("h1")
	if err != nil {
		t.Errorf("Expected valid selector, got error: %v", err)
	}

	// Invalid selector
	err = parser.ValidateSelector(".nonexistent")
	if err == nil {
		t.Error("Expected error for invalid selector")
	}
}

func TestParserWithMalformedHTML(t *testing.T) {
	html := `<html><body><h1>Title<p>Unclosed tags<div>More content</body>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser with malformed HTML: %v", err)
	}

	// Should still be able to extract content
	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: true,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract from malformed HTML: %v", err)
	}

	if !strings.Contains(result.(string), "Title") {
		t.Errorf("Expected title to contain 'Title', got %v", result)
	}
}
