// internal/scraper/enhanced_extractor_test.go
package scraper

import (
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/valpere/DataScrapexter/internal/pipeline"
)

func TestFieldExtractor_EnhancedTypes(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		config   FieldConfig
		expected interface{}
		wantErr  bool
	}{
		// Number extraction
		{
			name: "Extract Number",
			html: `<div class="price">$19.99</div>`,
			config: FieldConfig{
				Name:     "price",
				Selector: ".price",
				Type:     "number",
			},
			expected: 19.99,
			wantErr:  false,
		},
		{
			name: "Extract Positive Number",
			html: `<div class="balance">+1500.75</div>`,
			config: FieldConfig{
				Name:     "balance",
				Selector: ".balance",
				Type:     "number",
			},
			expected: 1500.75,
			wantErr:  false,
		},
		{
			name: "Extract Negative Number",
			html: `<div class="deficit">-250.50</div>`,
			config: FieldConfig{
				Name:     "deficit",
				Selector: ".deficit",
				Type:     "number",
			},
			expected: -250.50,
			wantErr:  false,
		},
		{
			name: "Extract Integer",
			html: `<div class="count">42 items</div>`,
			config: FieldConfig{
				Name:     "count",
				Selector: ".count",
				Type:     "integer",
			},
			expected: int64(42),
			wantErr:  false,
		},

		// Boolean extraction
		{
			name: "Extract Boolean True",
			html: `<div class="status">Available</div>`,
			config: FieldConfig{
				Name:     "available",
				Selector: ".status",
				Type:     "boolean",
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "Extract Boolean False from Text",
			html: `<div class="status">false</div>`,
			config: FieldConfig{
				Name:     "available",
				Selector: ".status",
				Type:     "boolean",
			},
			expected: false, // Explicit false value
			wantErr:  false,
		},
		{
			name: "Extract Boolean Out of Stock",
			html: `<div class="status">Out of Stock</div>`,
			config: FieldConfig{
				Name:     "available",
				Selector: ".status",
				Type:     "boolean",
			},
			expected: false, // Common negative phrase recognized as false
			wantErr:  false,
		},
		{
			name: "Extract Boolean from Class",
			html: `<div class="status active"></div>`,
			config: FieldConfig{
				Name:     "active",
				Selector: ".status",
				Type:     "boolean",
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "Extract Boolean Sold Out",
			html: `<div class="availability">Sold Out</div>`,
			config: FieldConfig{
				Name:     "in_stock",
				Selector: ".availability",
				Type:     "boolean",
			},
			expected: false, // Common negative phrase
			wantErr:  false,
		},
		{
			name: "Extract Boolean Coming Soon",
			html: `<div class="status">Coming Soon</div>`,
			config: FieldConfig{
				Name:     "available_now",
				Selector: ".status",
				Type:     "boolean",
			},
			expected: false, // Future availability is false for current availability
			wantErr:  false,
		},
		{
			name: "Extract Boolean Unrecognized Text with Warning",
			html: `<div class="custom">Custom Status Message</div>`,
			config: FieldConfig{
				Name:     "status",
				Selector: ".custom",
				Type:     "boolean",
			},
			expected: true, // Truly unrecognized text defaults to true (with warning)
			wantErr:  false,
		},
		{
			name: "Extract Boolean from Disabled Attribute",
			html: `<input type="checkbox" disabled>`,
			config: FieldConfig{
				Name:     "enabled",
				Selector: "input",
				Type:     "boolean",
			},
			expected: false, // Disabled attribute indicates false
			wantErr:  false,
		},
		{
			name: "Extract Boolean from Checked Attribute",
			html: `<input type="checkbox" checked>`,
			config: FieldConfig{
				Name:     "selected",
				Selector: "input",
				Type:     "boolean",
			},
			expected: true, // Checked attribute indicates true
			wantErr:  false,
		},

		// Date/Time extraction
		{
			name: "Extract Date",
			html: `<time datetime="2023-12-25">December 25, 2023</time>`,
			config: FieldConfig{
				Name:     "date",
				Selector: "time",
				Type:     "date",
			},
			expected: "2023-12-25",
			wantErr:  false,
		},
		{
			name: "Extract DateTime",
			html: `<time datetime="2023-12-25T15:30:00Z">Christmas Day</time>`,
			config: FieldConfig{
				Name:     "datetime",
				Selector: "time",
				Type:     "datetime",
			},
			expected: "2023-12-25T15:30:00Z",
			wantErr:  false,
		},

		// URL extraction
		{
			name: "Extract URL from href",
			html: `<a href="https://example.com/page">Link</a>`,
			config: FieldConfig{
				Name:     "url",
				Selector: "a",
				Type:     "url",
			},
			expected: "https://example.com/page",
			wantErr:  false,
		},
		{
			name: "Extract URL from src",
			html: `<img src="https://example.com/image.jpg" alt="Image">`,
			config: FieldConfig{
				Name:     "image_url",
				Selector: "img",
				Type:     "url",
			},
			expected: "https://example.com/image.jpg",
			wantErr:  false,
		},
		{
			name: "Extract Relative URL with Base Tag",
			html: `<base href="https://example.com/"><a href="/page">Link</a>`,
			config: FieldConfig{
				Name:     "page_url",
				Selector: "a",
				Type:     "url",
			},
			expected: "https://example.com/page",
			wantErr:  false,
		},
		{
			name: "Extract Relative URL with Canonical",
			html: `<link rel="canonical" href="https://example.com/current"><a href="relative/page">Link</a>`,
			config: FieldConfig{
				Name:     "page_url",
				Selector: "a",
				Type:     "url",
			},
			expected: "https://example.com/relative/page",
			wantErr:  false,
		},

		// Email extraction
		{
			name: "Extract Email from text",
			html: `<div class="contact">Contact us at info@example.com</div>`,
			config: FieldConfig{
				Name:     "email",
				Selector: ".contact",
				Type:     "email",
			},
			expected: "info@example.com",
			wantErr:  false,
		},
		{
			name: "Extract Email from mailto",
			html: `<a href="mailto:support@example.com">Email Us</a>`,
			config: FieldConfig{
				Name:     "email",
				Selector: "a",
				Type:     "email",
			},
			expected: "support@example.com",
			wantErr:  false,
		},

		// Phone extraction
		{
			name: "Extract Phone",
			html: `<div class="phone">Call us: +1 (555) 123-4567</div>`,
			config: FieldConfig{
				Name:     "phone",
				Selector: ".phone",
				Type:     "phone",
			},
			expected: "+15551234567",
			wantErr:  false,
		},
		{
			name: "Extract Phone Starting with 0",
			html: `<div class="phone">0123 456 789</div>`,
			config: FieldConfig{
				Name:     "phone",
				Selector: ".phone",
				Type:     "phone",
			},
			expected: "0123456789",
			wantErr:  false,
		},

		// JSON extraction
		{
			name: "Extract JSON",
			html: `<script type="application/json">{"name": "John", "age": 30}</script>`,
			config: FieldConfig{
				Name:     "data",
				Selector: "script",
				Type:     "json",
			},
			expected: map[string]interface{}{"name": "John", "age": float64(30)},
			wantErr:  false,
		},

		// Table extraction
		{
			name: "Extract Table",
			html: `
				<table>
					<thead>
						<tr><th>Name</th><th>Age</th></tr>
					</thead>
					<tbody>
						<tr><td>John</td><td>30</td></tr>
						<tr><td>Jane</td><td>25</td></tr>
					</tbody>
				</table>
			`,
			config: FieldConfig{
				Name:     "table_data",
				Selector: "table",
				Type:     "table",
			},
			expected: map[string]interface{}{
				"headers": []string{"Name", "Age"},
				"rows": []map[string]interface{}{
					{"Name": "John", "Age": "30"},
					{"Name": "Jane", "Age": "25"},
				},
				"count": 2,
			},
			wantErr: false,
		},

		// Utility types
		{
			name: "Count Elements",
			html: `<ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul>`,
			config: FieldConfig{
				Name:     "item_count",
				Selector: "li",
				Type:     "count",
			},
			expected: 3,
			wantErr:  false,
		},
		{
			name: "Check Exists",
			html: `<div class="warning">Warning message</div>`,
			config: FieldConfig{
				Name:     "has_warning",
				Selector: ".warning",
				Type:     "exists",
			},
			expected: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			extractor := NewFieldExtractor(tt.config, doc)
			result, err := extractor.Extract(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Type-specific comparisons
			switch tt.config.Type {
			case "table":
				// Safe type assertions for table comparison
				expectedMap, ok := tt.expected.(map[string]interface{})
				if !ok {
					t.Errorf("Expected result is not a map[string]interface{}")
					return
				}

				actualMap, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Actual result is not a map[string]interface{}")
					return
				}

				// Safe assertion for headers
				expectedHeaders, ok := expectedMap["headers"].([]string)
				if !ok {
					t.Errorf("Expected headers is not []string")
					return
				}

				actualHeaders, ok := actualMap["headers"].([]string)
				if !ok {
					t.Errorf("Actual headers is not []string")
					return
				}

				if len(actualHeaders) != len(expectedHeaders) {
					t.Errorf("Headers length mismatch: expected %d, got %d",
						len(expectedHeaders), len(actualHeaders))
				}

				if actualMap["count"] != expectedMap["count"] {
					t.Errorf("Row count mismatch: expected %v, got %v", expectedMap["count"], actualMap["count"])
				}

			case "json":
				// Safe type assertions for JSON comparison
				expectedMap, ok := tt.expected.(map[string]interface{})
				if !ok {
					t.Errorf("Expected result is not a map[string]interface{}")
					return
				}

				actualMap, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Actual result is not a map[string]interface{}")
					return
				}

				if expectedMap["name"] != actualMap["name"] {
					t.Errorf("JSON name mismatch: expected %v, got %v", expectedMap["name"], actualMap["name"])
				}
				if expectedMap["age"] != actualMap["age"] {
					t.Errorf("JSON age mismatch: expected %v, got %v", expectedMap["age"], actualMap["age"])
				}

			default:
				if result != tt.expected {
					t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
				}
			}
		})
	}
}

func TestAdvancedTransformations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		rules    []pipeline.TransformRule
		expected string
		wantErr  bool
	}{
		// Split transformation
		{
			name:  "Split String",
			input: "apple,banana,orange",
			rules: []pipeline.TransformRule{
				{
					Type:    "split",
					Pattern: ",",
					Params:  map[string]interface{}{"index": 1},
				},
			},
			expected: "banana",
			wantErr:  false,
		},

		// Substring transformation
		{
			name:  "Substring",
			input: "Hello World",
			rules: []pipeline.TransformRule{
				{
					Type:   "substring",
					Params: map[string]interface{}{"start": 6, "end": 11},
				},
			},
			expected: "World",
			wantErr:  false,
		},

		// Truncate transformation
		{
			name:  "Truncate with ellipsis",
			input: "This is a very long text that needs to be truncated",
			rules: []pipeline.TransformRule{
				{
					Type:   "truncate",
					Params: map[string]interface{}{"length": 20, "suffix": "..."},
				},
			},
			expected: "This is a very lo...",
			wantErr:  false,
		},

		// Title case transformation
		{
			name:  "Title Case",
			input: "hello world from go",
			rules: []pipeline.TransformRule{
				{Type: "title_case"},
			},
			expected: "Hello World From Go",
			wantErr:  false,
		},

		// Currency formatting
		{
			name:  "Format Currency USD",
			input: "1234.56",
			rules: []pipeline.TransformRule{
				{
					Type:   "format_currency",
					Params: map[string]interface{}{"symbol": "$"},
				},
			},
			expected: "$1234.56",
			wantErr:  false,
		},
		{
			name:  "Format Currency Euro",
			input: "€1,234.56 EUR",
			rules: []pipeline.TransformRule{
				{
					Type:   "format_currency",
					Params: map[string]interface{}{"symbol": "€"},
				},
			},
			expected: "€1234.56",
			wantErr:  false,
		},
		{
			name:  "Format Currency with Spaces",
			input: "1 234.56 USD",
			rules: []pipeline.TransformRule{
				{
					Type:   "format_currency",
					Params: map[string]interface{}{"symbol": "$"},
				},
			},
			expected: "$1234.56",
			wantErr:  false,
		},
		{
			name:  "Format Complex Currency",
			input: "€ 1 500,75 EUR",
			rules: []pipeline.TransformRule{
				{
					Type:        "regex",
					Pattern:     ",",
					Replacement: ".",
				},
				{
					Type:   "format_currency",
					Params: map[string]interface{}{"symbol": "€"},
				},
			},
			expected: "€1500.75",
			wantErr:  false,
		},

		// Domain extraction
		{
			name:  "Extract Domain",
			input: "https://www.example.com/path/to/page?query=1",
			rules: []pipeline.TransformRule{
				{Type: "extract_domain"},
			},
			expected: "www.example.com",
			wantErr:  false,
		},

		// Filename extraction
		{
			name:  "Extract Filename",
			input: "https://example.com/images/photo.jpg",
			rules: []pipeline.TransformRule{
				{Type: "extract_filename"},
			},
			expected: "photo.jpg",
			wantErr:  false,
		},

		// Capitalize words
		{
			name:  "Capitalize Words",
			input: "hello WORLD from GO",
			rules: []pipeline.TransformRule{
				{Type: "capitalize_words"},
			},
			expected: "Hello World From Go",
			wantErr:  false,
		},

		// Remove duplicates
		{
			name:  "Remove Duplicates",
			input: "apple,banana,apple,orange,banana",
			rules: []pipeline.TransformRule{
				{
					Type:   "remove_duplicates",
					Params: map[string]interface{}{"delimiter": ","},
				},
			},
			expected: "apple,banana,orange",
			wantErr:  false,
		},

		// Padding
		{
			name:  "Pad Left",
			input: "123",
			rules: []pipeline.TransformRule{
				{
					Type:   "pad_left",
					Params: map[string]interface{}{"length": 6, "char": "0"},
				},
			},
			expected: "000123",
			wantErr:  false,
		},
		{
			name:  "Pad Right",
			input: "test",
			rules: []pipeline.TransformRule{
				{
					Type:   "pad_right",
					Params: map[string]interface{}{"length": 8, "char": "."},
				},
			},
			expected: "test....",
			wantErr:  false,
		},

		// Complex transformation chain
		{
			name:  "Complex Chain: Clean, Extract, Format",
			input: "Price: $1,234.99 USD (including tax)",
			rules: []pipeline.TransformRule{
				{
					Type:        "regex",
					Pattern:     `\$([0-9,]+\.?[0-9]*)`,
					Replacement: "$1",
				},
				{Type: "remove_commas"},
				{
					Type:   "format_currency",
					Params: map[string]interface{}{"symbol": "$"},
				},
			},
			expected: "$1234.99",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformList := pipeline.TransformList(tt.rules)
			result, err := transformList.Apply(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFieldExtractor_WithAdvancedTransformations(t *testing.T) {
	html := `
		<div class="product">
			<h1 class="title">AWESOME PRODUCT NAME</h1>
			<div class="price">Price: $1,299.99</div>
			<div class="description">This is a very long product description that might need to be truncated for display purposes in various UI components</div>
			<a href="https://example.com/products/awesome-product-123" class="link">View Details</a>
		</div>
	`

	tests := []struct {
		name     string
		config   FieldConfig
		expected string
	}{
		{
			name: "Clean and Format Title",
			config: FieldConfig{
				Name:     "title",
				Selector: ".title",
				Type:     "text",
				Transform: []pipeline.TransformRule{
					{Type: "lowercase"},
					{Type: "capitalize_words"},
				},
			},
			expected: "Awesome Product Name",
		},
		{
			name: "Extract and Format Price",
			config: FieldConfig{
				Name:     "price",
				Selector: ".price",
				Type:     "text",
				Transform: []pipeline.TransformRule{
					{
						Type:        "regex",
						Pattern:     `\$([0-9,]+\.?[0-9]*)`,
						Replacement: "$1",
					},
					{Type: "remove_commas"},
					{
						Type:   "format_currency",
						Params: map[string]interface{}{"symbol": "$"},
					},
				},
			},
			expected: "$1299.99",
		},
		{
			name: "Truncate Description",
			config: FieldConfig{
				Name:     "description",
				Selector: ".description",
				Type:     "text",
				Transform: []pipeline.TransformRule{
					{
						Type:   "truncate",
						Params: map[string]interface{}{"length": 50, "suffix": "..."},
					},
				},
			},
			expected: "This is a very long product description that mi...",
		},
		{
			name: "Extract Product ID from URL",
			config: FieldConfig{
				Name:     "product_id",
				Selector: ".link",
				Type:     "url",
				Transform: []pipeline.TransformRule{
					{Type: "extract_filename"},
					{
						Type:    "split",
						Pattern: "-",
						Params:  map[string]interface{}{"index": 2},
					},
				},
			},
			expected: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			extractor := NewFieldExtractor(tt.config, doc)
			result, err := extractor.Extract(context.Background())

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
