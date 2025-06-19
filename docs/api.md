# DataScrapexter API Documentation

## Overview

DataScrapexter provides a comprehensive Go API for web scraping with advanced anti-detection capabilities. This document covers the core types, interfaces, and usage patterns for integrating DataScrapexter into your Go applications.

## Core Packages

### github.com/valpere/DataScrapexter/pkg/api

The primary public API package containing all configuration structures and types needed to interact with DataScrapexter.

### github.com/valpere/DataScrapexter/internal/scraper

The core scraping engine implementation. While this is an internal package, understanding its structure helps in advanced usage scenarios.

## Configuration Types

### ScraperConfig

The `ScraperConfig` struct represents the complete configuration for a scraping job. It defines what to scrape, how to extract data, and where to output results.

```go
type ScraperConfig struct {
    Name        string             `yaml:"name" json:"name"`
    BaseURL     string             `yaml:"base_url" json:"base_url"`
    UserAgents  []string           `yaml:"user_agents,omitempty" json:"user_agents,omitempty"`
    RateLimit   string             `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
    Timeout     string             `yaml:"timeout,omitempty" json:"timeout,omitempty"`
    MaxRetries  int                `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
    Headers     map[string]string  `yaml:"headers,omitempty" json:"headers,omitempty"`
    Proxy       *ProxyConfig       `yaml:"proxy,omitempty" json:"proxy,omitempty"`
    Fields      []Field            `yaml:"fields" json:"fields"`
    Pagination  *PaginationConfig  `yaml:"pagination,omitempty" json:"pagination,omitempty"`
    Output      OutputConfig       `yaml:"output" json:"output"`
}
```

#### Field Descriptions

- **Name**: Unique identifier for the scraper configuration
- **BaseURL**: The starting URL for scraping operations
- **UserAgents**: Custom user agent strings for rotation (optional, defaults provided)
- **RateLimit**: Duration string (e.g., "2s", "500ms") controlling request frequency
- **Timeout**: Maximum time to wait for a single request (e.g., "30s")
- **MaxRetries**: Number of retry attempts for failed requests
- **Headers**: Additional HTTP headers to include in requests
- **Proxy**: Proxy configuration for requests
- **Fields**: Data extraction field definitions
- **Pagination**: Configuration for multi-page scraping
- **Output**: Specifies output format and destination

### Field

The `Field` struct defines a single data point to extract from web pages.

```go
type Field struct {
    Name        string            `yaml:"name" json:"name"`
    Selector    string            `yaml:"selector" json:"selector"`
    Type        string            `yaml:"type" json:"type"`
    Attribute   string            `yaml:"attribute,omitempty" json:"attribute,omitempty"`
    Required    bool              `yaml:"required,omitempty" json:"required,omitempty"`
    Transform   []TransformRule   `yaml:"transform,omitempty" json:"transform,omitempty"`
}
```

#### Field Types

- **text**: Extracts text content from matched elements
- **html**: Extracts raw HTML from matched elements
- **attr**: Extracts a specific attribute value (requires `Attribute` field)
- **list**: Extracts text from all matching elements as an array

#### CSS Selector Support

DataScrapexter uses standard CSS selectors for element targeting:
- Class selectors: `.product-name`
- ID selectors: `#main-content`
- Attribute selectors: `[data-price]`
- Pseudo-selectors: `:first-child`, `:nth-of-type(2)`
- Combinators: `div > p`, `h1 + p`

### TransformRule

Transform rules modify extracted data before output.

```go
type TransformRule struct {
    Type        string `yaml:"type" json:"type"`
    Pattern     string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
    Replacement string `yaml:"replacement,omitempty" json:"replacement,omitempty"`
}
```

#### Available Transformations

- **trim**: Remove leading/trailing whitespace
- **lowercase**: Convert to lowercase
- **uppercase**: Convert to uppercase
- **normalize_spaces**: Replace multiple spaces with single space
- **remove_html**: Strip HTML tags
- **regex**: Apply regex pattern matching (requires Pattern and Replacement)
- **parse_float**: Convert string to floating-point number
- **parse_int**: Convert string to integer
- **clean_price**: Extract numeric price from text
- **extract_numbers**: Extract all numeric values

### PaginationConfig

Controls how the scraper navigates through multiple pages.

```go
type PaginationConfig struct {
    Type        string `yaml:"type" json:"type"`
    Selector    string `yaml:"selector,omitempty" json:"selector,omitempty"`
    MaxPages    int    `yaml:"max_pages,omitempty" json:"max_pages,omitempty"`
    URLPattern  string `yaml:"url_pattern,omitempty" json:"url_pattern,omitempty"`
    StartPage   int    `yaml:"start_page,omitempty" json:"start_page,omitempty"`
}
```

#### Pagination Types

- **next_button**: Follow "next" links using the specified selector
- **page_numbers**: Navigate numbered pages
- **url_pattern**: Generate URLs using a pattern (e.g., `/page/{page}`)

### OutputConfig

Defines how scraped data is saved or transmitted.

```go
type OutputConfig struct {
    Format   string         `yaml:"format" json:"format"`
    File     string         `yaml:"file,omitempty" json:"file,omitempty"`
    Database *DatabaseConfig `yaml:"database,omitempty" json:"database,omitempty"`
}
```

#### Supported Formats

- **json**: JSON format with pretty printing
- **csv**: Comma-separated values with automatic header generation
- **excel**: Excel spreadsheet format (coming in v0.5)

## Scraping Engine

### Engine Initialization

```go
import (
    "github.com/valpere/DataScrapexter/internal/scraper"
)

// Create engine with default configuration
engine, err := scraper.NewEngine(nil)

// Create engine with custom configuration
config := &scraper.Config{
    MaxRetries:      5,
    RetryDelay:      3 * time.Second,
    Timeout:         45 * time.Second,
    RateLimit:       2 * time.Second,
    BurstSize:       3,
    ProxyURL:        "http://proxy.example.com:8080",
}
engine, err := scraper.NewEngine(config)
```

### Performing Scraping Operations

```go
// Define extraction rules
extractors := []scraper.FieldExtractor{
    {
        Name:     "title",
        Selector: "h1",
        Type:     "text",
        Required: true,
    },
    {
        Name:      "image",
        Selector:  "img.main",
        Type:      "attr",
        Attribute: "src",
    },
}

// Execute scraping
ctx := context.Background()
result, err := engine.Scrape(ctx, "https://example.com", extractors)
if err != nil {
    log.Fatal(err)
}

// Access extracted data
fmt.Printf("Title: %v\n", result.Data["title"])
fmt.Printf("Image: %v\n", result.Data["image"])
```

## Advanced Usage

### Custom HTTP Headers

```go
config := &scraper.Config{
    Headers: map[string]string{
        "Accept-Language": "en-US,en;q=0.9",
        "Referer":         "https://google.com",
        "X-Custom-Header": "custom-value",
    },
}
```

### Proxy Configuration

```go
// Single proxy
config := &api.ScraperConfig{
    Proxy: &api.ProxyConfig{
        Enabled: true,
        URL:     "http://user:pass@proxy.example.com:8080",
    },
}

// Multiple proxies with rotation
config := &api.ScraperConfig{
    Proxy: &api.ProxyConfig{
        Enabled:  true,
        Rotation: "random",
        List: []string{
            "http://proxy1.example.com:8080",
            "http://proxy2.example.com:8080",
            "http://proxy3.example.com:8080",
        },
    },
}
```

### Handling Pagination

```go
// Create paginated scraper
paginatedScraper, err := scraper.NewPaginatedScraper(engine, config, extractors)
if err != nil {
    log.Fatal(err)
}

// Scrape all pages
results, err := paginatedScraper.ScrapeAll(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Scraped %d pages\n", len(results))
```

### Data Transformation Pipeline

```go
// Define field with transformations
field := api.Field{
    Name:     "price",
    Selector: ".price-tag",
    Type:     "text",
    Transform: []api.TransformRule{
        {Type: "clean_price"},
        {Type: "parse_float"},
    },
}

// Custom regex transformation
field := api.Field{
    Name:     "product_id",
    Selector: ".sku",
    Type:     "text",
    Transform: []api.TransformRule{
        {
            Type:        "regex",
            Pattern:     `SKU:\s*(\d+)`,
            Replacement: "$1",
        },
    },
}
```

## Error Handling

DataScrapexter provides detailed error information for debugging:

```go
result, err := engine.Scrape(ctx, url, extractors)
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        log.Println("Request timed out")
    case errors.Is(err, context.Canceled):
        log.Println("Request was cancelled")
    default:
        log.Printf("Scraping error: %v", err)
    }
}

// Check individual field errors
if result.Error != nil {
    log.Printf("Page error: %v", result.Error)
}
```

## Performance Optimization

### Connection Pooling

The engine automatically manages connection pooling for optimal performance:

```go
// Transport configuration in engine
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}
```

### Rate Limiting

Built-in rate limiting prevents overwhelming target servers:

```go
// Configure rate limiting
config := &scraper.Config{
    RateLimit: 500 * time.Millisecond,  // 2 requests per second
    BurstSize: 5,                       // Allow bursts up to 5 requests
}
```

### Context Management

Use context for proper cancellation and timeout control:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
// Cancel on signal
go func() {
    <-signalChan
    cancel()
}()
```

## Best Practices

### Configuration Management

Store configurations in YAML files for easy maintenance and version control. Use environment variables for sensitive data like API keys and proxy credentials.

### Error Recovery

Implement proper error handling and recovery mechanisms. Use the built-in retry functionality for transient failures, and log errors appropriately for debugging.

### Resource Management

Always use context for cancellation support. The engine handles connection pooling automatically, but be mindful of concurrent operations to avoid overwhelming system resources.

### Data Quality

Use field validation and transformation rules to ensure data quality. Mark critical fields as required to catch extraction failures early.

### Compliance

Respect robots.txt files and implement appropriate rate limiting. Always check website terms of service before scraping, and handle personal data in compliance with privacy regulations.

## Type Reference

For complete type definitions and additional documentation, refer to the source code in:
- `pkg/api/types.go` - Public API types
- `internal/scraper/engine.go` - Scraping engine implementation
- `internal/pipeline/transformer.go` - Data transformation pipeline
- `internal/output/csv_output.go` - Output formatting

## Version Compatibility

This documentation covers DataScrapexter v0.1.0. Future versions will maintain backward compatibility for the core API while adding new features and improvements.
