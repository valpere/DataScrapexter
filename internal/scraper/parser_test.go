// internal/scraper/parser_test.go
package scraper

import (
	"strings"
	"testing"
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
		Selector:  ".link",
		Type:      "attr",
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

func TestExtractField_List(t *testing.T) {
	html := `<html><body><ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "items",
		Selector: "li",
		Type:     "list",
		Required: true,
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	items, ok := result.([]string)
	if !ok {
		t.Fatalf("Expected []string, got %T", result)
	}

	expected := []string{"Item 1", "Item 2", "Item 3"}
	if len(items) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(items))
	}

	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected item[%d] to be '%s', got '%s'", i, expected[i], item)
		}
	}
}

func TestExtractField_RequiredFieldMissing(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "missing",
		Selector: ".missing",
		Type:     "text",
		Required: true,
	}

	_, err = parser.ExtractField(config)
	if err == nil {
		t.Error("Expected error for missing required field")
	}
}

func TestExtractField_OptionalFieldMissing(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "missing",
		Selector: ".missing",
		Type:     "text",
		Required: false,
		Default:  "default_value",
	}

	result, err := parser.ExtractField(config)
	if err != nil {
		t.Fatalf("Failed to extract optional field: %v", err)
	}

	if result != "default_value" {
		t.Errorf("Expected 'default_value', got %v", result)
	}
}

func TestExtractField_AttributeTypeMissingAttribute(t *testing.T) {
	html := `<html><body><a href="https://example.com">Link</a></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "url",
		Selector: "a",
		Type:     "attr",
		// Missing Attribute field
		Required: true,
	}

	_, err = parser.ExtractField(config)
	if err == nil {
		t.Error("Expected error for attr type without attribute name")
	}
}

func TestExtractField_AttributeNotFound(t *testing.T) {
	html := `<html><body><a href="https://example.com">Link</a></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:      "target",
		Selector:  "a",
		Type:      "attr",
		Attribute: "target", // This attribute doesn't exist
		Required:  true,
	}

	_, err = parser.ExtractField(config)
	if err == nil {
		t.Error("Expected error for missing attribute")
	}
}

func TestExtractField_UnsupportedType(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "unsupported",
		Required: true,
	}

	_, err = parser.ExtractField(config)
	if err == nil {
		t.Error("Expected error for unsupported field type")
	}
}

func TestExtractTable(t *testing.T) {
	html := `
		<html>
		<body>
			<table>
				<thead>
					<tr>
						<th>Name</th>
						<th>Price</th>
						<th>Stock</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td>Product 1</td>
						<td>$99.99</td>
						<td>10</td>
					</tr>
					<tr>
						<td>Product 2</td>
						<td>$149.99</td>
						<td>5</td>
					</tr>
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

func TestGetLinks(t *testing.T) {
	html := `
		<html>
		<body>
			<a href="https://example.com">Example</a>
			<a href="/internal-link">Internal</a>
			<a href="mailto:test@example.com">Email</a>
		</body>
		</html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	links := parser.GetLinks()

	if len(links) != 3 {
		t.Errorf("Expected 3 links, got %d", len(links))
	}

	// Check first link
	if links[0]["href"] != "https://example.com" {
		t.Errorf("Expected first link href to be 'https://example.com', got '%s'", links[0]["href"])
	}

	if links[0]["text"] != "Example" {
		t.Errorf("Expected first link text to be 'Example', got '%s'", links[0]["text"])
	}
}

func TestGetImages(t *testing.T) {
	html := `
		<html>
		<body>
			<img src="/image1.jpg" alt="Image 1">
			<img src="/image2.png" alt="Image 2">
			<img src="/image3.gif">
		</body>
		</html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	images := parser.GetImages()

	if len(images) != 3 {
		t.Errorf("Expected 3 images, got %d", len(images))
	}

	// Check first image
	if images[0]["src"] != "/image1.jpg" {
		t.Errorf("Expected first image src to be '/image1.jpg', got '%s'", images[0]["src"])
	}

	if images[0]["alt"] != "Image 1" {
		t.Errorf("Expected first image alt to be 'Image 1', got '%s'", images[0]["alt"])
	}
}

func TestExtractField_EmptySelector(t *testing.T) {
	html := `<html><body><h1>Test</h1></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := FieldConfig{
		Name:     "test",
		Selector: "", // Empty selector
		Type:     "text",
		Required: true,
	}

	_, err = parser.ExtractField(config)
	if err == nil {
		t.Error("Expected error for empty selector")
	}
}

func TestExtractTable_NoTable(t *testing.T) {
	html := `<html><body><p>No table here</p></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	_, err = parser.ExtractTable("table")
	if err == nil {
		t.Error("Expected error when no table found")
	}
}

func TestExtractTable_EmptySelector(t *testing.T) {
	html := `<html><body><table><tr><td>Test</td></tr></table></body></html>`

	parser, err := NewHTMLParser(html)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	_, err = parser.ExtractTable("")
	if err == nil {
		t.Error("Expected error for empty table selector")
	}
}
