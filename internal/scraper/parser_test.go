// internal/scraper/parser_test.go
package scraper

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHTMLParser(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Test</h1></body></html>`))
	}))
	defer server.Close()

	// Make a request to the test server
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	// Test parser creation
	parser, err := NewHTMLParser(resp)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	if parser == nil {
		t.Fatal("Parser should not be nil")
	}

	if parser.document == nil {
		t.Fatal("Document should not be nil")
	}

	if parser.baseURL != server.URL {
		t.Fatalf("Expected baseURL %s, got %s", server.URL, parser.baseURL)
	}
}

func TestNewHTMLParserFromString(t *testing.T) {
	html := `<html><body><h1>Test Title</h1><p class="content">Test content</p></body></html>`
	baseURL := "https://example.com"

	parser, err := NewHTMLParserFromString(html, baseURL)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	if parser.baseURL != baseURL {
		t.Fatalf("Expected baseURL %s, got %s", baseURL, parser.baseURL)
	}

	// Test that we can find elements
	selection := parser.Find("h1")
	if selection.Length() != 1 {
		t.Fatalf("Expected 1 h1 element, got %d", selection.Length())
	}

	title := selection.Text()
	if title != "Test Title" {
		t.Fatalf("Expected title 'Test Title', got '%s'", title)
	}
}

func TestExtractField_Text(t *testing.T) {
	html := `<html><body><h1>Test Title</h1><p class="content">Test content</p></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "title",
		Selector: "h1",
		Type:     "text",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != "Test Title" {
		t.Fatalf("Expected 'Test Title', got '%v'", value)
	}
}

func TestExtractField_Attribute(t *testing.T) {
	html := `<html><body><a href="https://example.com" title="Example Link">Link</a></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "link_url",
		Selector: "a",
		Type:     "attribute",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != "https://example.com" {
		t.Fatalf("Expected 'https://example.com', got '%v'", value)
	}
}

func TestExtractField_HTML(t *testing.T) {
	html := `<html><body><div class="content"><p>Test <strong>content</strong></p></div></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "content_html",
		Selector: ".content",
		Type:     "html",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	expected := "<p>Test <strong>content</strong></p>"
	if value != expected {
		t.Fatalf("Expected '%s', got '%v'", expected, value)
	}
}

func TestExtractField_Int(t *testing.T) {
	html := `<html><body><span class="price">$1,234</span></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "price",
		Selector: ".price",
		Type:     "int",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != 1234 {
		t.Fatalf("Expected 1234, got %v", value)
	}
}

func TestExtractField_Number(t *testing.T) {
	html := `<html><body><span class="quantity">42</span></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "quantity",
		Selector: ".quantity",
		Type:     "number",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != 42 {
		t.Fatalf("Expected 42, got %v", value)
	}
}

func TestExtractField_Float(t *testing.T) {
	html := `<html><body><span class="price">$12.34</span></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "price",
		Selector: ".price",
		Type:     "float",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != 12.34 {
		t.Fatalf("Expected 12.34, got %v", value)
	}
}

func TestExtractField_Bool(t *testing.T) {
	html := `<html><body><span class="available">Yes</span><span class="disabled">No</span></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Test true case
	field := FieldConfig{
		Name:     "available",
		Selector: ".available",
		Type:     "bool",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != true {
		t.Fatalf("Expected true, got %v", value)
	}

	// Test false case
	field.Name = "disabled"
	field.Selector = ".disabled"

	value, err = parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != false {
		t.Fatalf("Expected false, got %v", value)
	}
}

func TestExtractField_Boolean(t *testing.T) {
	html := `<html><body><span class="active">true</span></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "active",
		Selector: ".active",
		Type:     "boolean",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != true {
		t.Fatalf("Expected true, got %v", value)
	}
}

func TestExtractField_Date(t *testing.T) {
	html := `<html><body><time class="date">2023-12-25</time></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "date",
		Selector: ".date",
		Type:     "date",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	expectedDate := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	if value != expectedDate {
		t.Fatalf("Expected %v, got %v", expectedDate, value)
	}
}

func TestExtractField_Array(t *testing.T) {
	html := `<html><body>
		<ul class="items">
			<li>Item 1</li>
			<li>Item 2</li>
			<li>Item 3</li>
		</ul>
	</body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "items",
		Selector: ".items li",
		Type:     "array",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	items, ok := value.([]interface{})
	if !ok {
		t.Fatalf("Expected array, got %T", value)
	}

	if len(items) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(items))
	}

	expected := []string{"Item 1", "Item 2", "Item 3"}
	for i, item := range items {
		if item != expected[i] {
			t.Fatalf("Expected item %d to be '%s', got '%v'", i, expected[i], item)
		}
	}
}

func TestExtractField_Href(t *testing.T) {
	html := `<html><body><a href="/test-link">Test Link</a></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "link",
		Selector: "a",
		Type:     "href",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != "/test-link" {
		t.Fatalf("Expected '/test-link', got '%v'", value)
	}
}

func TestExtractField_Src(t *testing.T) {
	html := `<html><body><img src="/test-image.jpg" alt="Test"></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "image",
		Selector: "img",
		Type:     "src",
		Required: true,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Failed to extract field: %v", err)
	}

	if value != "/test-image.jpg" {
		t.Fatalf("Expected '/test-image.jpg', got '%v'", value)
	}
}

func TestExtractField_NotFound_Required(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "missing",
		Selector: ".missing",
		Type:     "text",
		Required: true,
	}

	_, err = parser.ExtractField(field)
	if err == nil {
		t.Fatal("Expected error for missing required field")
	}

	if !strings.Contains(err.Error(), "required field") {
		t.Fatalf("Expected error about required field, got: %v", err)
	}
}

func TestExtractField_NotFound_Optional(t *testing.T) {
	html := `<html><body><h1>Test Title</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	field := FieldConfig{
		Name:     "missing",
		Selector: ".missing",
		Type:     "text",
		Required: false,
	}

	value, err := parser.ExtractField(field)
	if err != nil {
		t.Fatalf("Unexpected error for optional field: %v", err)
	}

	if value != nil {
		t.Fatalf("Expected nil for missing optional field, got %v", value)
	}
}

func TestExtractMultiple(t *testing.T) {
	html := `<html><body>
		<h1>Test Title</h1>
		<p class="content">Test content</p>
		<span class="price">$123.45</span>
		<a href="https://example.com">Link</a>
	</body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	fields := []FieldConfig{
		{
			Name:     "title",
			Selector: "h1",
			Type:     "text",
			Required: true,
		},
		{
			Name:     "content",
			Selector: ".content",
			Type:     "text",
			Required: false,
		},
		{
			Name:     "price",
			Selector: ".price",
			Type:     "float",
			Required: true,
		},
		{
			Name:     "link",
			Selector: "a",
			Type:     "href",
			Required: false,
		},
	}

	results, err := parser.ExtractMultiple(fields)
	if err != nil {
		t.Fatalf("Failed to extract fields: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("Expected 4 results, got %d", len(results))
	}

	if results["title"] != "Test Title" {
		t.Fatalf("Expected title 'Test Title', got '%v'", results["title"])
	}

	if results["content"] != "Test content" {
		t.Fatalf("Expected content 'Test content', got '%v'", results["content"])
	}

	if results["price"] != 123.45 {
		t.Fatalf("Expected price 123.45, got %v", results["price"])
	}

	if results["link"] != "https://example.com" {
		t.Fatalf("Expected link 'https://example.com', got '%v'", results["link"])
	}
}

func TestValidateSelector(t *testing.T) {
	html := `<html><body><h1>Test</h1></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Test valid selector
	if !parser.ValidateSelector("h1") {
		t.Fatal("Expected h1 selector to be valid")
	}

	// Test valid complex selector
	if !parser.ValidateSelector("body h1") {
		t.Fatal("Expected complex selector to be valid")
	}

	// Test that the function doesn't panic with empty selector
	_ = parser.ValidateSelector("")
}

func TestExtractText_Whitespace(t *testing.T) {
	html := `<html><body><p>   Test   with   multiple   spaces   </p></body></html>`
	parser, err := NewHTMLParserFromString(html, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	selection := parser.Find("p")
	text := parser.extractText(selection)

	expected := "Test with multiple spaces"
	if text != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, text)
	}
}

func TestNewHTMLParser_NilResponse(t *testing.T) {
	_, err := NewHTMLParser(nil)
	if err == nil {
		t.Fatal("Expected error for nil response")
	}

	if !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("Expected error about nil response, got: %v", err)
	}
}

func TestNewHTMLParserFromReader_NilReader(t *testing.T) {
	_, err := NewHTMLParserFromReader(nil, "https://example.com")
	if err == nil {
		t.Fatal("Expected error for nil reader")
	}

	if !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("Expected error about nil reader, got: %v", err)
	}
}
