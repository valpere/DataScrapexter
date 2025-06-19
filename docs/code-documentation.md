# DataScrapexter Code Documentation

## Architecture Overview

DataScrapexter follows a modular architecture with clear separation of concerns. The codebase is organized into distinct packages, each responsible for specific functionality. This design promotes maintainability, testability, and extensibility.

### Package Structure

```
DataScrapexter/
├── cmd/                    # Command-line applications
│   └── datascrapexter/     # Main CLI entry point
├── internal/               # Private application code
│   ├── scraper/            # Core scraping engine
│   ├── pipeline/           # Data transformation pipeline
│   ├── output/             # Output formatting and writing
│   └── utils/              # Common utility functions
├── pkg/                    # Public API packages
│   └── api/                # Public types and interfaces
```

## Core Components

### Scraping Engine (`internal/scraper/engine.go`)

The scraping engine is the heart of DataScrapexter. It manages HTTP requests, handles retries, implements rate limiting, and coordinates data extraction.

#### Key Types

**Engine struct**: The main scraping engine that encapsulates all scraping functionality.
```go
type Engine struct {
    httpClient     *http.Client      // Configured HTTP client with transport settings
    userAgentPool  []string          // Pool of user agents for rotation
    currentUAIndex int               // Current index in user agent pool
    uaMutex        sync.Mutex        // Mutex for thread-safe UA rotation
    config         *Config           // Engine configuration
    rateLimiter    *RateLimiter      // Token bucket rate limiter
}
```

**Config struct**: Configuration options for the engine.
```go
type Config struct {
    MaxRetries       int              // Number of retry attempts for failed requests
    RetryDelay       time.Duration    // Base delay between retries (exponential backoff)
    Timeout          time.Duration    // HTTP request timeout
    FollowRedirects  bool             // Whether to follow HTTP redirects
    MaxRedirects     int              // Maximum number of redirects to follow
    RateLimit        time.Duration    // Minimum time between requests
    BurstSize        int              // Token bucket burst size
    ProxyURL         string           // Optional proxy URL
    Headers          map[string]string // Custom HTTP headers
}
```

**Result struct**: Represents the outcome of a scraping operation.
```go
type Result struct {
    URL        string                 // The URL that was scraped
    StatusCode int                    // HTTP response status code
    Data       map[string]interface{} // Extracted data keyed by field name
    Error      error                  // Any error that occurred
    Timestamp  time.Time              // When the scraping occurred
}
```

#### Key Methods

**NewEngine**: Creates a new scraping engine with the provided configuration. It initializes the HTTP client with proper transport settings, sets up cookie management, configures proxy if specified, and creates the rate limiter.

**Scrape**: The main scraping method that orchestrates the entire extraction process. It applies rate limiting, performs the HTTP request with retries, parses the HTML response, extracts data using field extractors, applies transformations, and returns the results.

**doRequestWithRetry**: Implements exponential backoff retry logic for failed requests. It handles transient failures, server errors (5xx), and rate limiting responses (429).

**extractField**: Extracts data from the HTML document based on field configuration. It supports multiple extraction types (text, HTML, attributes, lists) and applies configured transformations.

### Data Transformation Pipeline (`internal/pipeline/transformer.go`)

The transformation pipeline processes extracted data to clean, normalize, and convert it into the desired format.

#### Transformation Types

- **Text transformations**: trim, lowercase, uppercase, normalize_spaces, remove_html
- **Numeric transformations**: parse_int, parse_float, extract_numbers, clean_price
- **Pattern matching**: regex transformations with capture groups
- **Data cleaning**: remove special characters, normalize whitespace

#### Key Functions

**TransformField**: Applies a series of transformations to a single value. Transformations are applied in order, with each transformation receiving the output of the previous one.

**TransformList**: Applies transformations to a list of values, maintaining the list structure while transforming each element.

### Output Handling (`internal/output/csv_output.go`)

The output package handles formatting and writing scraped data to various formats.

#### CSV Output Features

- Automatic header generation from data fields
- Proper escaping and quoting of values
- Support for nested data flattening
- Configurable delimiters
- Array handling (joined with semicolons)

#### Key Functions

**WriteResultsToCSV**: Main function that coordinates CSV output. It extracts headers from results, writes the header row, and processes each result into CSV format.

**formatValue**: Converts various Go types to string representations suitable for CSV output. It handles strings, numbers, booleans, and arrays intelligently.

### Pagination Handling (`internal/scraper/pagination.go`)

The pagination system enables scraping of multi-page websites automatically.

#### Pagination Strategies

**Next Button Pagination**: Follows "next" links by extracting the href attribute and resolving relative URLs. It tracks visited URLs to prevent infinite loops.

**Page Number Pagination**: Generates URLs based on numeric patterns. It can detect common page parameter patterns or use configured URL templates.

**URL Pattern Pagination**: Uses template strings with placeholders to generate page URLs systematically.

#### Key Components

**PaginationHandler**: Manages pagination state and URL generation. It tracks visited URLs, handles different pagination types, and resolves relative URLs.

**PaginatedScraper**: Coordinates multi-page scraping operations. It scrapes the first page, determines subsequent pages, and aggregates results.

### CLI Application (`cmd/datascrapexter/main.go`)

The command-line interface provides user-friendly access to DataScrapexter's functionality.

#### Commands

**run**: Executes a scraping job based on a configuration file. It loads and validates configuration, creates the scraping engine, performs extraction, handles pagination, and writes output.

**validate**: Checks configuration file syntax and semantics without performing scraping.

**template**: Generates example configuration files for common scenarios.

**version**: Displays version information including build time and git commit.

#### Configuration Loading

The CLI handles YAML parsing, environment variable expansion, validation of required fields, and default value assignment.

## Design Patterns

### Factory Pattern

Used in engine creation to encapsulate complex initialization logic:
```go
engine, err := scraper.NewEngine(config)
```

### Strategy Pattern

Transformation rules implement a strategy pattern where each transformation type has its own implementation:
```go
func (t *Transformer) applyRule(value interface{}, rule TransformRule) (interface{}, error)
```

### Builder Pattern

Configuration objects use a builder-like approach with optional fields and defaults:
```go
config := scraper.DefaultConfig()
config.RateLimit = 2 * time.Second
config.ProxyURL = "http://proxy:8080"
```

### Observer Pattern

The rate limiter implements a token bucket algorithm that observes time passage to refill tokens.

## Concurrency Patterns

### Mutex Protection

User agent rotation uses mutex protection for thread safety:
```go
func (e *Engine) getNextUserAgent() string {
    e.uaMutex.Lock()
    defer e.uaMutex.Unlock()
    // ... rotation logic
}
```

### Context Usage

All scraping operations accept a context for cancellation and timeout control:
```go
func (e *Engine) Scrape(ctx context.Context, targetURL string, extractors []FieldExtractor) (*Result, error)
```

### Token Bucket Rate Limiting

The rate limiter uses channels and goroutines for efficient token management:
```go
type RateLimiter struct {
    tokens chan struct{}     // Buffered channel as token bucket
    ticker *time.Ticker      // Periodic token refill
}
```

## Error Handling

### Error Types

DataScrapexter uses descriptive error messages with context:
```go
fmt.Errorf("failed to create request: %w", err)  // Error wrapping
fmt.Errorf("no elements found for selector: %s", extractor.Selector)  // Contextual errors
```

### Retry Logic

Failed requests are retried with exponential backoff:
```go
delay := e.config.RetryDelay * time.Duration(attempt)
```

### Validation Errors

Configuration validation provides specific error messages:
```go
return fmt.Errorf("field[%d]: selector is required", i)
```

## Performance Considerations

### Connection Pooling

The HTTP transport is configured for connection reuse:
```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}
```

### Memory Efficiency

- Streaming processing for large responses
- Efficient string building using strings.Builder
- Proper resource cleanup with defer statements

### Compilation Optimization

Build flags optimize binary size and performance:
```go
-ldflags "-w -s"  // Strip debug information
```

## Security Considerations

### TLS Configuration

Proper TLS settings ensure secure connections:
```go
TLSClientConfig: &tls.Config{
    InsecureSkipVerify: false,  // Always verify certificates
}
```

### Header Sanitization

User-provided headers are validated before use to prevent header injection attacks.

### Proxy Support

Proxy URLs are parsed and validated to ensure proper format and security.

## Testing Strategies

### Unit Testing

Each component should be tested in isolation:
```go
func TestEngine_Scrape(t *testing.T) {
    // Test with mock HTTP responses
}
```

### Integration Testing

End-to-end tests verify complete workflows:
```go
func TestCompleteScrapingFlow(t *testing.T) {
    // Test configuration loading through output writing
}
```

### Benchmarking

Performance-critical code should include benchmarks:
```go
func BenchmarkTransformField(b *testing.B) {
    // Measure transformation performance
}
```

## Extension Points

### Custom Transformations

New transformation types can be added to the pipeline:
```go
case "custom_transform":
    return t.customTransform(value)
```

### Output Formats

New output formats can be implemented by following the existing pattern:
```go
case "xml":
    return outputXML(results, outputConfig.File)
```

### Extraction Types

Additional extraction types can be added to support new use cases:
```go
case "json":
    return e.extractJSON(selection)
```

## Best Practices

### Configuration Management

- Use environment variables for sensitive data
- Validate configurations before use
- Provide sensible defaults
- Document all configuration options

### Error Handling

- Always wrap errors with context
- Use structured logging for debugging
- Fail fast with clear error messages
- Implement proper cleanup on errors

### Performance

- Use rate limiting to respect server resources
- Implement connection pooling
- Cache reusable data
- Profile code to identify bottlenecks

### Security

- Validate all user input
- Use HTTPS whenever possible
- Implement proper timeout handling
- Avoid storing sensitive data in logs

This documentation provides a comprehensive overview of DataScrapexter's internal architecture and implementation details. For specific API usage, refer to the API documentation. For contribution guidelines, see CONTRIBUTING.md.
