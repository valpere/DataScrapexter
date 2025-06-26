#!/bin/bash
# scripts/setup.sh - Setup script to create project structure and fix build errors

set -e

echo "Setting up DataScrapexter project structure..."

# Create necessary directories
echo "Creating directories..."
mkdir -p cmd/datascrapexter
mkdir -p internal/scraper
mkdir -p internal/pipeline
mkdir -p internal/antidetect
mkdir -p internal/config
mkdir -p internal/compliance
mkdir -p pkg/api
mkdir -p pkg/client
mkdir -p pkg/types
mkdir -p configs
mkdir -p docs
mkdir -p examples
mkdir -p scripts
mkdir -p test
mkdir -p bin

# Create the main.go file
echo "Creating cmd/datascrapexter/main.go..."
cat > cmd/datascrapexter/main.go << 'EOF'
// cmd/datascrapexter/main.go
package main

import (
	"fmt"
	"os"
)

// Build-time variables (set by ldflags)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("DataScrapexter %s\n", version)
		fmt.Printf("Build time: %s\n", buildTime)
		fmt.Printf("Git commit: %s\n", gitCommit)
		return
	}

	fmt.Printf("DataScrapexter v%s\n", version)
	fmt.Println("Universal web scraper built with Go")
	fmt.Println("Use 'datascrapexter help' for usage information")
}
EOF

# Create the pipeline transform.go file
echo "Creating internal/pipeline/transform.go..."
cat > internal/pipeline/transform.go << 'EOF'
// internal/pipeline/transform.go
package pipeline

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TransformRule defines a single transformation rule
type TransformRule struct {
	Type        string                 `yaml:"type" json:"type"`
	Pattern     string                 `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Replacement string                 `yaml:"replacement,omitempty" json:"replacement,omitempty"`
	Format      string                 `yaml:"format,omitempty" json:"format,omitempty"`
	Params      map[string]interface{} `yaml:"params,omitempty" json:"params,omitempty"`
}

// TransformList represents a list of transformation rules
type TransformList []TransformRule

// TransformField defines field-specific transformations
type TransformField struct {
	Name       string        `yaml:"name" json:"name"`
	Rules      TransformList `yaml:"rules,omitempty" json:"rules,omitempty"`
	Required   bool          `yaml:"required,omitempty" json:"required,omitempty"`
	DefaultVal interface{}   `yaml:"default,omitempty" json:"default,omitempty"`
}

// Transform applies transformation rules to input data
func (tr *TransformRule) Transform(ctx context.Context, input string) (string, error) {
	switch tr.Type {
	case "trim":
		return strings.TrimSpace(input), nil
	case "normalize_spaces":
		re := regexp.MustCompile(`\s+`)
		return re.ReplaceAllString(strings.TrimSpace(input), " "), nil
	case "regex":
		if tr.Pattern == "" {
			return "", fmt.Errorf("regex pattern is required")
		}
		re, err := regexp.Compile(tr.Pattern)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern: %w", err)
		}
		return re.ReplaceAllString(input, tr.Replacement), nil
	case "parse_float":
		cleaned := strings.ReplaceAll(input, ",", "")
		cleaned = strings.ReplaceAll(cleaned, "$", "")
		cleaned = strings.TrimSpace(cleaned)
		if _, err := strconv.ParseFloat(cleaned, 64); err != nil {
			return "", fmt.Errorf("failed to parse float: %w", err)
		}
		return cleaned, nil
	default:
		return "", fmt.Errorf("unknown transform type: %s", tr.Type)
	}
}

// Apply applies all transformation rules in sequence
func (tl TransformList) Apply(ctx context.Context, input string) (string, error) {
	result := input
	for _, rule := range tl {
		var err error
		result, err = rule.Transform(ctx, result)
		if err != nil {
			return "", fmt.Errorf("transform failed at rule %s: %w", rule.Type, err)
		}
	}
	return result, nil
}

// ValidateTransformRules validates transformation rule configuration
func ValidateTransformRules(rules TransformList) error {
	for i, rule := range rules {
		switch rule.Type {
		case "trim", "normalize_spaces", "parse_float":
			// These transforms require no additional parameters
		case "regex":
			if rule.Pattern == "" {
				return fmt.Errorf("rule %d: regex pattern is required", i)
			}
		default:
			return fmt.Errorf("rule %d: unknown transform type: %s", i, rule.Type)
		}
	}
	return nil
}
EOF

# Create the pagination strategies file
echo "Creating internal/scraper/pagination_strategies.go..."
cat > internal/scraper/pagination_strategies.go << 'EOF'
// internal/scraper/pagination_strategies.go
package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// PaginationStrategy defines the interface for pagination strategies
type PaginationStrategy interface {
	GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error)
	IsComplete(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) bool
	GetName() string
}

// OffsetStrategy implements offset-based pagination
type OffsetStrategy struct {
	BaseURL     string `yaml:"base_url" json:"base_url"`
	OffsetParam string `yaml:"offset_param" json:"offset_param"`
	LimitParam  string `yaml:"limit_param" json:"limit_param"`
	Limit       int    `yaml:"limit" json:"limit"`
	MaxOffset   int    `yaml:"max_offset" json:"max_offset"`
	StartOffset int    `yaml:"start_offset" json:"start_offset"`
}

// GetNextURL generates the next URL using offset pagination
func (os *OffsetStrategy) GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error) {
	if os.OffsetParam == "" {
		os.OffsetParam = "offset"
	}
	if os.LimitParam == "" {
		os.LimitParam = "limit"
	}
	if os.Limit <= 0 {
		os.Limit = 10
	}

	baseURL := os.BaseURL
	if baseURL == "" {
		baseURL = currentURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	nextOffset := os.StartOffset + (pageNum * os.Limit)
	if os.MaxOffset > 0 && nextOffset >= os.MaxOffset {
		return "", nil
	}

	query := u.Query()
	query.Set(os.OffsetParam, strconv.Itoa(nextOffset))
	query.Set(os.LimitParam, strconv.Itoa(os.Limit))
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// IsComplete checks if offset pagination is complete
func (os *OffsetStrategy) IsComplete(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) bool {
	if os.MaxOffset > 0 {
		currentOffset := os.StartOffset + ((pageNum - 1) * os.Limit)
		return currentOffset >= os.MaxOffset
	}
	return false
}

// GetName returns the strategy name
func (os *OffsetStrategy) GetName() string {
	return "offset"
}

// CursorStrategy implements cursor-based pagination
type CursorStrategy struct {
	BaseURL        string `yaml:"base_url" json:"base_url"`
	CursorParam    string `yaml:"cursor_param" json:"cursor_param"`
	LimitParam     string `yaml:"limit_param" json:"limit_param"`
	Limit          int    `yaml:"limit" json:"limit"`
	MaxPages       int    `yaml:"max_pages" json:"max_pages"`
	CursorSelector string `yaml:"cursor_selector" json:"cursor_selector"`
	CursorAttr     string `yaml:"cursor_attr" json:"cursor_attr"`
	
	lastCursor string
}

// GetNextURL generates the next URL using cursor pagination
func (cs *CursorStrategy) GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error) {
	if cs.CursorParam == "" {
		cs.CursorParam = "cursor"
	}
	if cs.LimitParam == "" {
		cs.LimitParam = "limit"
	}
	if cs.Limit <= 0 {
		cs.Limit = 10
	}

	if cs.MaxPages > 0 && pageNum > cs.MaxPages {
		return "", nil
	}

	nextCursor, err := cs.extractCursor(doc)
	if err != nil {
		return "", fmt.Errorf("failed to extract cursor: %w", err)
	}

	if nextCursor == "" || nextCursor == cs.lastCursor {
		return "", nil
	}
	
	cs.lastCursor = nextCursor

	baseURL := cs.BaseURL
	if baseURL == "" {
		baseURL = currentURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	query := u.Query()
	query.Set(cs.CursorParam, nextCursor)
	query.Set(cs.LimitParam, strconv.Itoa(cs.Limit))
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// extractCursor extracts the next cursor value from the document
func (cs *CursorStrategy) extractCursor(doc *goquery.Document) (string, error) {
	if cs.CursorSelector == "" {
		return "", fmt.Errorf("cursor_selector is required for cursor strategy")
	}

	selection := doc.Find(cs.CursorSelector)
	if selection.Length() == 0 {
		return "", nil
	}

	var cursor string
	if cs.CursorAttr != "" {
		cursor, _ = selection.Attr(cs.CursorAttr)
	} else {
		cursor = strings.TrimSpace(selection.Text())
	}

	return cursor, nil
}

// IsComplete checks if cursor pagination is complete
func (cs *CursorStrategy) IsComplete(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) bool {
	if cs.MaxPages > 0 && pageNum > cs.MaxPages {
		return true
	}

	cursor, _ := cs.extractCursor(doc)
	return cursor == "" || cursor == cs.lastCursor
}

// GetName returns the strategy name
func (cs *CursorStrategy) GetName() string {
	return "cursor"
}
EOF

# Create the basic engine.go file
echo "Creating internal/scraper/engine.go..."
cat > internal/scraper/engine.go << 'EOF'
// internal/scraper/engine.go
package scraper

import (
	"context"
	"fmt"

	"github.com/valpere/DataScrapexter/internal/pipeline"
)

// ScrapingEngine is the main scraping engine
type ScrapingEngine struct {
	Config *EngineConfig
}

// EngineConfig holds engine configuration
type EngineConfig struct {
	Fields    []FieldConfig                `yaml:"fields" json:"fields"`
	Transform []pipeline.TransformRule     `yaml:"transform,omitempty" json:"transform,omitempty"`
}

// FieldConfig defines field extraction configuration
type FieldConfig struct {
	Name      string                   `yaml:"name" json:"name"`
	Selector  string                   `yaml:"selector" json:"selector"`
	Type      string                   `yaml:"type" json:"type"`
	Required  bool                     `yaml:"required,omitempty" json:"required,omitempty"`
	Transform []pipeline.TransformRule `yaml:"transform,omitempty" json:"transform,omitempty"`
}

// NewScrapingEngine creates a new scraping engine
func NewScrapingEngine(config *EngineConfig) *ScrapingEngine {
	return &ScrapingEngine{
		Config: config,
	}
}

// ProcessFields processes field extraction with transformations
func (se *ScrapingEngine) ProcessFields(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for _, field := range se.Config.Fields {
		value, exists := data[field.Name]
		if !exists {
			if field.Required {
				return nil, fmt.Errorf("required field %s not found", field.Name)
			}
			continue
		}
		
		if len(field.Transform) > 0 {
			transformList := pipeline.TransformList(field.Transform)
			if str, ok := value.(string); ok {
				transformed, err := transformList.Apply(ctx, str)
				if err != nil {
					return nil, fmt.Errorf("field transformation failed for %s: %w", field.Name, err)
				}
				result[field.Name] = transformed
			} else {
				result[field.Name] = value
			}
		} else {
			result[field.Name] = value
		}
	}
	
	return result, nil
}
EOF

# Create the pagination.go file
echo "Creating internal/scraper/pagination.go..."
cat > internal/scraper/pagination.go << 'EOF'
// internal/scraper/pagination.go
package scraper

import (
	"context"
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

// PaginationConfig defines pagination configuration
type PaginationConfig struct {
	Type     string `yaml:"type" json:"type"`
	Selector string `yaml:"selector,omitempty" json:"selector,omitempty"`
	MaxPages int    `yaml:"max_pages,omitempty" json:"max_pages,omitempty"`
}

// PaginationManager manages pagination across different strategies
type PaginationManager struct {
	config   PaginationConfig
	strategy PaginationStrategy
}

// NewPaginationManager creates a new pagination manager
func NewPaginationManager(config PaginationConfig) (*PaginationManager, error) {
	pm := &PaginationManager{
		config: config,
	}
	
	strategy, err := pm.createStrategy()
	if err != nil {
		return nil, fmt.Errorf("failed to create pagination strategy: %w", err)
	}
	
	pm.strategy = strategy
	return pm, nil
}

// createStrategy creates the appropriate pagination strategy
func (pm *PaginationManager) createStrategy() (PaginationStrategy, error) {
	switch pm.config.Type {
	case "offset":
		return &OffsetStrategy{}, nil
	case "cursor":
		return &CursorStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown pagination type: %s", pm.config.Type)
	}
}

// GetNextURL gets the next URL to scrape
func (pm *PaginationManager) GetNextURL(ctx context.Context, currentURL string, doc *goquery.Document, pageNum int) (string, error) {
	if pm.strategy == nil {
		return "", fmt.Errorf("pagination strategy not initialized")
	}
	
	return pm.strategy.GetNextURL(ctx, currentURL, doc, pageNum)
}
EOF

# Create go.mod if it doesn't exist
if [ ! -f "go.mod" ]; then
	echo "Creating go.mod..."
	cat > go.mod << 'EOF'
module github.com/valpere/DataScrapexter

go 1.24

require (
	github.com/PuerkitoBio/goquery v1.8.1
	github.com/gocolly/colly/v2 v2.1.0
)

require (
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/antchfx/htmlquery v1.3.0 // indirect
	github.com/antchfx/xmlquery v1.3.18 // indirect
	github.com/antchfx/xpath v1.2.4 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/kennygrant/sanitize v1.2.4 // indirect
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d // indirect
	github.com/temoto/robotstxt v1.1.2 // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
EOF
fi

# Create example config
echo "Creating example config..."
cat > configs/example.yaml << 'EOF'
name: "example_scraper"
base_url: "https://example.com"

fields:
  - name: "title"
    selector: "h1"
    type: "text"
    transform:
      - type: "trim"
      - type: "normalize_spaces"

  - name: "price"
    selector: ".price"
    type: "text"
    transform:
      - type: "regex"
        pattern: "\\$([0-9,]+\\.?[0-9]*)"
        replacement: "$1"
      - type: "parse_float"

pagination:
  type: "offset"
  max_pages: 10

output:
  format: "json"
  file: "output.json"
EOF

# Run go mod tidy to resolve dependencies
echo "Running go mod tidy..."
go mod tidy || echo "Warning: go mod tidy failed - you may need to run it manually"

echo "Project setup complete!"
echo ""
echo "To build the project:"
echo "  make build"
echo ""
echo "To run the scraper:"
echo "  ./bin/datascrapexter run configs/example.yaml"
echo ""
echo "To generate a template:"
echo "  ./bin/datascrapexter template > my_config.yaml"
