// internal/config/config.go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ScraperConfig represents the complete configuration for a scraping job
type ScraperConfig struct {
	Name       string             `yaml:"name" json:"name"`
	BaseURL    string             `yaml:"base_url" json:"base_url"`
	URLs       []string           `yaml:"urls,omitempty" json:"urls,omitempty"`
	UserAgents []string           `yaml:"user_agents,omitempty" json:"user_agents,omitempty"`
	RateLimit  string             `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	Timeout    string             `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	MaxRetries int                `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	Headers    map[string]string  `yaml:"headers,omitempty" json:"headers,omitempty"`
	Cookies    map[string]string  `yaml:"cookies,omitempty" json:"cookies,omitempty"`
	Proxy      *ProxyConfig       `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Fields     []Field            `yaml:"fields" json:"fields"`
	Pagination *PaginationConfig  `yaml:"pagination,omitempty" json:"pagination,omitempty"`
	Output     OutputConfig       `yaml:"output" json:"output"`
}

// Field represents a single field to extract
type Field struct {
	Name      string         `yaml:"name" json:"name"`
	Selector  string         `yaml:"selector" json:"selector"`
	Type      string         `yaml:"type" json:"type"`
	Required  bool           `yaml:"required,omitempty" json:"required,omitempty"`
	Attribute string         `yaml:"attribute,omitempty" json:"attribute,omitempty"`
	Default   interface{}    `yaml:"default,omitempty" json:"default,omitempty"`
	Transform []TransformRule `yaml:"transform,omitempty" json:"transform,omitempty"`
}

// FieldConfig is an alias for Field to maintain backward compatibility
type FieldConfig = Field

// PaginationConfig represents pagination configuration
type PaginationConfig struct {
	Type       string `yaml:"type" json:"type"`
	Selector   string `yaml:"selector,omitempty" json:"selector,omitempty"`
	MaxPages   int    `yaml:"max_pages,omitempty" json:"max_pages,omitempty"`
	URLPattern string `yaml:"url_pattern,omitempty" json:"url_pattern,omitempty"`
	StartPage  int    `yaml:"start_page,omitempty" json:"start_page,omitempty"`
}

// OutputConfig represents output configuration
type OutputConfig struct {
	Format       string `yaml:"format" json:"format"`
	File         string `yaml:"file" json:"file"`
	EnableMetrics bool   `yaml:"enable_metrics,omitempty" json:"enable_metrics,omitempty"`
}

// ProxyConfig represents proxy configuration
type ProxyConfig struct {
	URL      string `yaml:"url" json:"url"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// TransformRule represents a data transformation rule
type TransformRule struct {
	Type        string                 `yaml:"type" json:"type"`
	Pattern     string                 `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Replacement string                 `yaml:"replacement,omitempty" json:"replacement,omitempty"`
	Format      string                 `yaml:"format,omitempty" json:"format,omitempty"`
	Params      map[string]interface{} `yaml:"params,omitempty" json:"params,omitempty"`
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(filename string) (*ScraperConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ScraperConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// LoadFromBytes loads configuration from YAML bytes
func LoadFromBytes(data []byte) (*ScraperConfig, error) {
	var config ScraperConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *ScraperConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	
	if c.BaseURL == "" && len(c.URLs) == 0 {
		return fmt.Errorf("base_url or urls is required")
	}
	
	if len(c.Fields) == 0 {
		return fmt.Errorf("at least one field is required")
	}
	
	// Validate fields
	for i, field := range c.Fields {
		if field.Name == "" {
			return fmt.Errorf("field %d: name is required", i)
		}
		if field.Selector == "" {
			return fmt.Errorf("field %d: selector is required", i)
		}
		if field.Type == "" {
			return fmt.Errorf("field %d: type is required", i)
		}
		
		// Validate field types
		validTypes := map[string]bool{
			"text": true, "html": true, "attr": true, "list": true,
		}
		if !validTypes[field.Type] {
			return fmt.Errorf("field %d: invalid type %s", i, field.Type)
		}
		
		// Require attribute for attr type
		if field.Type == "attr" && field.Attribute == "" {
			return fmt.Errorf("field %d: attribute is required for type 'attr'", i)
		}
	}
	
	// Validate output
	if c.Output.Format == "" {
		c.Output.Format = "json" // Default format
	}
	
	validFormats := map[string]bool{
		"json": true, "csv": true, "yaml": true,
	}
	if !validFormats[c.Output.Format] {
		return fmt.Errorf("invalid output format: %s", c.Output.Format)
	}
	
	if c.Output.File == "" {
		c.Output.File = "output." + c.Output.Format // Default filename
	}
	
	return nil
}

// GenerateTemplate generates a template configuration
func GenerateTemplate(templateType string) *ScraperConfig {
	switch templateType {
	case "ecommerce":
		return &ScraperConfig{
			Name:    "ecommerce_scraper",
			BaseURL: "https://example-shop.com/products",
			Fields: []Field{
				{
					Name:     "title",
					Selector: ".product-title, h1",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "price",
					Selector: ".price, .product-price",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "description",
					Selector: ".product-description",
					Type:     "text",
					Required: false,
				},
				{
					Name:      "image",
					Selector:  ".product-image img",
					Type:      "attr",
					Attribute: "src",
					Required:  false,
				},
			},
			Output: OutputConfig{
				Format: "json",
				File:   "products.json",
			},
			RateLimit: "2s",
		}
	case "news":
		return &ScraperConfig{
			Name:    "news_scraper",
			BaseURL: "https://example-news.com/articles",
			Fields: []Field{
				{
					Name:     "headline",
					Selector: "h1, .headline",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "author",
					Selector: ".author, .byline",
					Type:     "text",
					Required: false,
				},
				{
					Name:     "content",
					Selector: ".article-content, .story-body",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "date",
					Selector: ".publish-date, time",
					Type:     "text",
					Required: false,
				},
			},
			Output: OutputConfig{
				Format: "json",
				File:   "articles.json",
			},
			RateLimit: "3s",
		}
	default: // basic
		return &ScraperConfig{
			Name:    "basic_scraper",
			BaseURL: "https://example.com",
			Fields: []Field{
				{
					Name:     "title",
					Selector: "h1",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "content",
					Selector: "p",
					Type:     "text",
					Required: false,
				},
			},
			Output: OutputConfig{
				Format: "json",
				File:   "output.json",
			},
			RateLimit: "1s",
		}
	}
}
