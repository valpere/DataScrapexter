# DataScrapexter API Reference

## Overview

This reference covers the complete Go programming interface for DataScrapexter, including core types, interfaces, and usage patterns for integrating web scraping capabilities into Go applications.

## Table of Contents

1. [Core Packages](#core-packages)
2. [Configuration Types](#configuration-types)
3. [Scraping Engine](#scraping-engine)
4. [Data Processing](#data-processing)
5. [Output Management](#output-management)
6. [Monitoring Integration](#monitoring-integration)
7. [Anti-Detection Features](#anti-detection-features)
8. [Error Handling](#error-handling)
9. [Performance Optimization](#performance-optimization)
10. [Complete Examples](#complete-examples)

## Core Packages

### github.com/valpere/DataScrapexter/pkg/scraper

The primary public API package containing all configuration structures and interfaces.

```go
import "github.com/valpere/DataScrapexter/pkg/scraper"
```

### github.com/valpere/DataScrapexter/internal/engine

Core scraping engine implementation (internal package).

```go
import "github.com/valpere/DataScrapexter/internal/engine"
```

### github.com/valpere/DataScrapexter/pkg/config

Configuration management and validation.

```go
import "github.com/valpere/DataScrapexter/pkg/config"
```

## Configuration Types

### ScraperConfig

Primary configuration structure for scraping operations.

```go
type ScraperConfig struct {
    // Basic identification
    Name        string             `yaml:"name" json:"name"`
    BaseURL     string             `yaml:"base_url" json:"base_url"`
    
    // Request configuration
    UserAgents  []string           `yaml:"user_agents,omitempty" json:"user_agents,omitempty"`
    RateLimit   string             `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
    Timeout     string             `yaml:"timeout,omitempty" json:"timeout,omitempty"`
    MaxRetries  int                `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
    Headers     map[string]string  `yaml:"headers,omitempty" json:"headers,omitempty"`
    
    // Advanced configuration
    Proxy           *ProxyConfig       `yaml:"proxy,omitempty" json:"proxy,omitempty"`
    AntiDetection   *AntiDetectionConfig `yaml:"anti_detection,omitempty" json:"anti_detection,omitempty"`
    Browser         *BrowserConfig     `yaml:"browser,omitempty" json:"browser,omitempty"`
    Monitoring      *MonitoringConfig  `yaml:"monitoring,omitempty" json:"monitoring,omitempty"`
    
    // Data extraction
    Fields          []Field            `yaml:"fields" json:"fields"`
    Pagination      *PaginationConfig  `yaml:"pagination,omitempty" json:"pagination,omitempty"`
    
    // Output
    Output          OutputConfig       `yaml:"output" json:"output"`
}
```

### Field

Data extraction field definition.

```go
type Field struct {
    Name        string            `yaml:"name" json:"name"`
    Selector    string            `yaml:"selector" json:"selector"`
    Type        FieldType         `yaml:"type" json:"type"`
    Attribute   string            `yaml:"attribute,omitempty" json:"attribute,omitempty"`
    Required    bool              `yaml:"required,omitempty" json:"required,omitempty"`
    Transform   []TransformRule   `yaml:"transform,omitempty" json:"transform,omitempty"`
    
    // Advanced options
    FallbackSelector string         `yaml:"fallback_selector,omitempty" json:"fallback_selector,omitempty"`
    Validation       *FieldValidation `yaml:"validation,omitempty" json:"validation,omitempty"`
    DefaultValue     interface{}     `yaml:"default_value,omitempty" json:"default_value,omitempty"`
}

type FieldType string

const (
    FieldTypeText FieldType = "text"
    FieldTypeHTML FieldType = "html"
    FieldTypeAttr FieldType = "attr"
    FieldTypeList FieldType = "list"
)
```

### TransformRule

Data transformation rule.

```go
type TransformRule struct {
    Type        TransformType `yaml:"type" json:"type"`
    Pattern     string        `yaml:"pattern,omitempty" json:"pattern,omitempty"`
    Replacement string        `yaml:"replacement,omitempty" json:"replacement,omitempty"`
    Options     map[string]interface{} `yaml:"options,omitempty" json:"options,omitempty"`
}

type TransformType string

const (
    TransformTrim           TransformType = "trim"
    TransformLowercase      TransformType = "lowercase"
    TransformUppercase      TransformType = "uppercase"
    TransformNormalizeSpaces TransformType = "normalize_spaces"
    TransformRegex          TransformType = "regex"
    TransformParseFloat     TransformType = "parse_float"
    TransformParseInt       TransformType = "parse_int"
    TransformCleanPrice     TransformType = "clean_price"
    TransformExtractNumbers TransformType = "extract_numbers"
    TransformRemoveHTML     TransformType = "remove_html"
)
```

### OutputConfig

Output configuration for scraped data.

```go
type OutputConfig struct {
    // Single output
    Format   OutputFormat   `yaml:"format,omitempty" json:"format,omitempty"`
    File     string         `yaml:"file,omitempty" json:"file,omitempty"`
    
    // Multiple outputs
    Multiple bool           `yaml:"multiple,omitempty" json:"multiple,omitempty"`
    Outputs  []OutputConfig `yaml:"outputs,omitempty" json:"outputs,omitempty"`
    
    // Format-specific configurations
    JSON         *JSONConfig     `yaml:"json,omitempty" json:"json,omitempty"`
    CSV          *CSVConfig      `yaml:"csv,omitempty" json:"csv,omitempty"`
    Excel        *ExcelConfig    `yaml:"excel,omitempty" json:"excel,omitempty"`
    XML          *XMLConfig      `yaml:"xml,omitempty" json:"xml,omitempty"`
    YAML         *YAMLConfig     `yaml:"yaml,omitempty" json:"yaml,omitempty"`
    Database     *DatabaseConfig `yaml:"database,omitempty" json:"database,omitempty"`
    CloudStorage *CloudStorageConfig `yaml:"cloud_storage,omitempty" json:"cloud_storage,omitempty"`
}

type OutputFormat string

const (
    OutputFormatJSON     OutputFormat = "json"
    OutputFormatCSV      OutputFormat = "csv"
    OutputFormatExcel    OutputFormat = "excel"
    OutputFormatXML      OutputFormat = "xml"
    OutputFormatYAML     OutputFormat = "yaml"
    OutputFormatDatabase OutputFormat = "database"
)
```

## Scraping Engine

### Engine Interface

Core scraping engine interface.

```go
type Engine interface {
    // Primary scraping operations
    Scrape(ctx context.Context, config *ScraperConfig) (*ScrapeResult, error)
    ScrapeURL(ctx context.Context, url string, extractors []FieldExtractor) (*PageResult, error)
    
    // Batch operations
    ScrapeMultiple(ctx context.Context, configs []*ScraperConfig) ([]*ScrapeResult, error)
    ScrapeURLs(ctx context.Context, urls []string, extractors []FieldExtractor) ([]*PageResult, error)
    
    // Configuration and management
    ValidateConfig(config *ScraperConfig) error
    GetStats() *EngineStats
    Shutdown(ctx context.Context) error
}
```

### Engine Creation

```go
// Create engine with default configuration
engine, err := scraper.NewEngine(nil)
if err != nil {
    log.Fatal(err)
}

// Create engine with custom configuration
engineConfig := &scraper.EngineConfig{
    MaxRetries:      5,
    RetryDelay:      3 * time.Second,
    Timeout:         45 * time.Second,
    RateLimit:       2 * time.Second,
    BurstSize:       3,
    MaxConcurrency:  10,
    ProxyURL:        "http://proxy.example.com:8080",
}

engine, err := scraper.NewEngine(engineConfig)
if err != nil {
    log.Fatal(err)
}
```

### ScrapeResult

Result structure from scraping operations.

```go
type ScrapeResult struct {
    // Metadata
    JobID       string            `json:"job_id"`
    ScraperName string            `json:"scraper_name"`
    StartTime   time.Time         `json:"start_time"`
    EndTime     time.Time         `json:"end_time"`
    Duration    time.Duration     `json:"duration"`
    
    // Results
    Pages       []*PageResult     `json:"pages"`
    TotalPages  int               `json:"total_pages"`
    TotalRecords int              `json:"total_records"`
    
    // Statistics
    Stats       *ScrapeStats      `json:"stats"`
    Errors      []error           `json:"errors,omitempty"`
}

type PageResult struct {
    URL         string                 `json:"url"`
    Data        map[string]interface{} `json:"data"`
    StatusCode  int                    `json:"status_code"`
    ResponseTime time.Duration         `json:"response_time"`
    Error       error                  `json:"error,omitempty"`
    Timestamp   time.Time              `json:"timestamp"`
}

type ScrapeStats struct {
    PagesRequested    int           `json:"pages_requested"`
    PagesSuccessful   int           `json:"pages_successful"`
    PagesFailed       int           `json:"pages_failed"`
    RecordsExtracted  int           `json:"records_extracted"`
    AverageResponseTime time.Duration `json:"average_response_time"`
    TotalDataSize     int64         `json:"total_data_size"`
}
```

## Data Processing

### Field Extraction

```go
// Define field extractors
extractors := []scraper.FieldExtractor{
    {
        Name:     "title",
        Selector: "h1",
        Type:     scraper.FieldTypeText,
        Required: true,
        Transform: []scraper.TransformRule{
            {Type: scraper.TransformTrim},
            {Type: scraper.TransformNormalizeSpaces},
        },
    },
    {
        Name:      "price",
        Selector:  ".price",
        Type:      scraper.FieldTypeText,
        Transform: []scraper.TransformRule{
            {Type: scraper.TransformCleanPrice},
            {Type: scraper.TransformParseFloat},
        },
    },
    {
        Name:      "image",
        Selector:  "img.main",
        Type:      scraper.FieldTypeAttr,
        Attribute: "src",
    },
}

// Execute extraction
ctx := context.Background()
result, err := engine.ScrapeURL(ctx, "https://example.com", extractors)
if err != nil {
    log.Fatal(err)
}

// Access extracted data
fmt.Printf("Title: %v\n", result.Data["title"])
fmt.Printf("Price: %v\n", result.Data["price"])
fmt.Printf("Image: %v\n", result.Data["image"])
```

### Custom Transformations

```go
// Register custom transformation
engine.RegisterTransform("custom_clean", func(input string, options map[string]interface{}) (string, error) {
    // Custom cleaning logic
    cleaned := strings.TrimSpace(input)
    cleaned = strings.ReplaceAll(cleaned, "\n", " ")
    return cleaned, nil
})

// Use in field configuration
field := scraper.Field{
    Name:     "description",
    Selector: ".description",
    Type:     scraper.FieldTypeText,
    Transform: []scraper.TransformRule{
        {
            Type: "custom_clean",
            Options: map[string]interface{}{
                "preserve_formatting": false,
            },
        },
    },
}
```

### Data Validation

```go
type FieldValidation struct {
    Required      bool                   `yaml:"required,omitempty" json:"required,omitempty"`
    MinLength     int                    `yaml:"min_length,omitempty" json:"min_length,omitempty"`
    MaxLength     int                    `yaml:"max_length,omitempty" json:"max_length,omitempty"`
    Pattern       string                 `yaml:"pattern,omitempty" json:"pattern,omitempty"`
    AllowedValues []string               `yaml:"allowed_values,omitempty" json:"allowed_values,omitempty"`
    Custom        func(interface{}) error `yaml:"-" json:"-"`
}

// Example with validation
field := scraper.Field{
    Name:     "email",
    Selector: ".contact-email",
    Type:     scraper.FieldTypeText,
    Validation: &scraper.FieldValidation{
        Required: true,
        Pattern:  `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
    },
}
```

## Output Management

### Output Writers

```go
// JSON output writer
jsonConfig := &scraper.JSONConfig{
    PrettyPrint:     true,
    IncludeMetadata: true,
    SortKeys:        true,
}

jsonWriter, err := scraper.NewJSONWriter("output.json", jsonConfig)
if err != nil {
    log.Fatal(err)
}

// Write results
err = jsonWriter.WriteResults(results)
if err != nil {
    log.Fatal(err)
}

// Close writer
err = jsonWriter.Close()
if err != nil {
    log.Fatal(err)
}
```

### Excel Output with Advanced Formatting

```go
excelConfig := &scraper.ExcelConfig{
    FilePath:       "report.xlsx",
    SheetName:      "Scraped Data",
    IncludeHeaders: true,
    AutoFilter:     true,
    FreezePane:     true,
    
    HeaderStyle: &scraper.ExcelCellStyle{
        Font: &scraper.ExcelFont{
            Bold:  true,
            Size:  12,
            Color: "#FFFFFF",
        },
        Fill: &scraper.ExcelFill{
            Type:  "pattern",
            Color: "#4472C4",
        },
        Alignment: &scraper.ExcelAlignment{
            Horizontal: "center",
            Vertical:   "center",
        },
    },
    
    ColumnWidths: map[string]int{
        "title":       30,
        "description": 50,
        "price":       15,
        "url":         40,
    },
    
    NumberFormats: map[string]string{
        "price":      "$#,##0.00",
        "percentage": "0.00%",
    },
}

excelWriter, err := scraper.NewExcelWriter(excelConfig)
if err != nil {
    log.Fatal(err)
}
```

### Database Output

```go
dbConfig := &scraper.DatabaseConfig{
    Driver:     "postgresql",
    Host:       "localhost",
    Port:       5432,
    Database:   "scraping_data",
    Username:   "scraper",
    Password:   "secure_password",
    Table:      "products",
    BatchSize:  1000,
    SSL:        true,
    
    // Automatic table creation
    AutoCreateTable: true,
    TableSchema: map[string]string{
        "id":          "SERIAL PRIMARY KEY",
        "title":       "VARCHAR(255) NOT NULL",
        "description": "TEXT",
        "price":       "DECIMAL(10,2)",
        "url":         "TEXT",
        "scraped_at":  "TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
    },
    
    // Upsert configuration
    OnConflict:      "update",
    ConflictColumns: []string{"url"},
    UpdateColumns:   []string{"title", "price", "scraped_at"},
}

dbWriter, err := scraper.NewDatabaseWriter(dbConfig)
if err != nil {
    log.Fatal(err)
}
```

### Cloud Storage Integration

```go
// AWS S3 configuration
s3Config := &scraper.CloudStorageConfig{
    Provider: "aws_s3",
    Bucket:   "my-scraping-data",
    KeyPrefix: "datascrapexter/",
    
    AWSConfig: &scraper.AWSConfig{
        AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
        SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
        Region:          "us-east-1",
        StorageClass:    "STANDARD",
        Encryption:      "AES256",
    },
}

cloudWriter, err := scraper.NewCloudStorageWriter(s3Config)
if err != nil {
    log.Fatal(err)
}
```

## Monitoring Integration

### Metrics Manager

```go
import "github.com/valpere/DataScrapexter/internal/monitoring"

// Initialize metrics
metricsConfig := monitoring.MetricsConfig{
    Namespace:     "datascrapexter",
    Subsystem:     "scraper",
    ListenAddress: ":9090",
    MetricsPath:   "/metrics",
}

metricsManager := monitoring.NewMetricsManager(metricsConfig)

// Start metrics server
ctx := context.Background()
go metricsManager.StartMetricsServer(ctx, ":9090", "/metrics")

// Record metrics during scraping
metricsManager.RecordRequest("GET", "example.com", "job-123", 200, time.Second)
metricsManager.RecordPageScraped("example.com", "job-123", "success")
metricsManager.RecordExtractionSuccess("title", "job-123")
```

### Health Monitoring

```go
// Initialize health manager
healthConfig := monitoring.HealthConfig{
    CheckInterval:     30 * time.Second,
    HealthEndpoint:    "/health",
    ReadinessEndpoint: "/ready",
    LivenessEndpoint:  "/live",
}

healthManager := monitoring.NewHealthManager(healthConfig)

// Register built-in health checks
healthManager.RegisterCheck(monitoring.MemoryHealthCheck(80.0))
healthManager.RegisterCheck(monitoring.GoroutineHealthCheck(1000))

// Register custom health check
customCheck := &monitoring.HealthCheck{
    Name: "scraper_engine",
    CheckFunc: func(ctx context.Context) monitoring.HealthCheckResult {
        if engine.IsHealthy() {
            return monitoring.HealthCheckResult{
                Status:  monitoring.HealthStatusHealthy,
                Message: "Scraper engine operational",
            }
        }
        return monitoring.HealthCheckResult{
            Status:  monitoring.HealthStatusUnhealthy,
            Message: "Scraper engine not responding",
        }
    },
}

healthManager.RegisterCheck(customCheck)

// Start health monitoring
healthManager.Start(ctx)

// Get health status
health := healthManager.GetHealth()
fmt.Printf("System Status: %s\n", health.Status)
```

## Anti-Detection Features

### Browser Fingerprinting Evasion

```go
import "github.com/valpere/DataScrapexter/internal/antidetect"

// Initialize fingerprinting evader
evader := antidetect.NewFingerprintingEvader(true)

// Generate spoofed fingerprints
canvasFingerprint := evader.Canvas.GenerateFingerprint()
webglProfile := evader.WebGL.GetRandomProfile()
audioFingerprint := evader.Audio.GenerateFingerprint()

// Use with browser automation
browserConfig := &scraper.BrowserConfig{
    Enabled:  true,
    Headless: true,
    
    FingerprintEvasion: &scraper.FingerprintEvasionConfig{
        Canvas: &scraper.CanvasEvasionConfig{
            Enabled:     true,
            NoiseLevel:  0.1,
            Fingerprint: canvasFingerprint,
        },
        WebGL: &scraper.WebGLEvasionConfig{
            Enabled: true,
            Profile: webglProfile,
        },
        Audio: &scraper.AudioEvasionConfig{
            Enabled:     true,
            Fingerprint: audioFingerprint,
        },
    },
}
```

### CAPTCHA Solving

```go
// Configure CAPTCHA solver
captchaConfig := antidetect.CaptchaConfig{
    Service: "2captcha",
    APIKey:  os.Getenv("CAPTCHA_API_KEY"),
    Timeout: 60 * time.Second,
}

solver := antidetect.NewCaptchaSolver(captchaConfig)

// Solve reCAPTCHA
solution, err := solver.SolveRecaptcha(ctx, "site-key", "page-url")
if err != nil {
    log.Printf("CAPTCHA solving failed: %v", err)
    return
}

// Use solution in browser automation
err = page.EvaluateScript(fmt.Sprintf(`
    document.getElementById('g-recaptcha-response').innerHTML = '%s';
    document.getElementById('captcha-form').submit();
`, solution.Token))
```

### Proxy Management

```go
// Advanced proxy manager
proxyConfig := antidetect.ProxyConfig{
    Enabled:     true,
    Rotation:    "weighted",
    HealthCheck: true,
    
    Providers: []antidetect.ProxyProvider{
        {
            URL:           "http://proxy1.example.com:8080",
            Username:      "user1",
            Password:      "pass1",
            Weight:        2,
            MaxConcurrent: 10,
            Type:          "residential",
            Country:       "US",
        },
        {
            URL:           "http://proxy2.example.com:8080",
            Username:      "user2",
            Password:      "pass2",
            Weight:        1,
            MaxConcurrent: 5,
            Type:          "datacenter",
            Country:       "UK",
        },
    },
}

proxyManager := antidetect.NewProxyManager(proxyConfig)

// Get healthy proxy
proxy, err := proxyManager.GetHealthyProxy()
if err != nil {
    log.Printf("No healthy proxies: %v", err)
    return
}

// Use proxy with HTTP client
transport := &http.Transport{
    Proxy: http.ProxyURL(proxy.URL),
}
client := &http.Client{Transport: transport}
```

## Error Handling

### Error Types

```go
// DataScrapexter error types
type ScraperError struct {
    Type    ErrorType
    Message string
    Cause   error
    Context map[string]interface{}
}

type ErrorType int

const (
    ErrorTypeNetwork ErrorType = iota
    ErrorTypeParsing
    ErrorTypeValidation
    ErrorTypeRateLimit
    ErrorTypeCaptcha
    ErrorTypeProxy
    ErrorTypeTransformation
    ErrorTypeOutput
)

// Error handling example
result, err := engine.Scrape(ctx, config)
if err != nil {
    var scraperErr *scraper.ScraperError
    if errors.As(err, &scraperErr) {
        switch scraperErr.Type {
        case scraper.ErrorTypeNetwork:
            log.Printf("Network error: %v", scraperErr.Message)
            // Implement retry logic
        case scraper.ErrorTypeCaptcha:
            log.Printf("CAPTCHA challenge: %v", scraperErr.Message)
            // Handle CAPTCHA
        case scraper.ErrorTypeRateLimit:
            log.Printf("Rate limited: %v", scraperErr.Message)
            // Implement backoff
        default:
            log.Printf("Scraping error: %v", scraperErr.Message)
        }
    }
}
```

### Retry Logic

```go
// Configure retry behavior
retryConfig := &scraper.RetryConfig{
    MaxRetries:    5,
    BackoffType:   scraper.BackoffExponential,
    InitialDelay:  1 * time.Second,
    MaxDelay:      30 * time.Second,
    BackoffFactor: 2.0,
    
    // Retry conditions
    RetryConditions: []scraper.RetryCondition{
        scraper.RetryOnNetworkError,
        scraper.RetryOnServerError,
        scraper.RetryOnRateLimit,
    },
}

engine.SetRetryConfig(retryConfig)
```

## Performance Optimization

### Connection Pooling

```go
// Configure HTTP transport
transportConfig := &scraper.TransportConfig{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
    TLSHandshakeTimeout: 10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
}

engine.SetTransportConfig(transportConfig)
```

### Concurrency Control

```go
// Configure concurrency
concurrencyConfig := &scraper.ConcurrencyConfig{
    MaxConcurrentRequests:   10,
    MaxConcurrentExtractions: 5,
    MaxConcurrentOutputs:     3,
    
    // Rate limiting
    RateLimit: &scraper.RateLimitConfig{
        RequestsPerSecond: 2.0,
        BurstSize:         5,
        Adaptive:          true,
    },
}

engine.SetConcurrencyConfig(concurrencyConfig)
```

### Memory Management

```go
// Configure memory limits
memoryConfig := &scraper.MemoryConfig{
    MaxMemoryUsage:     2 * 1024 * 1024 * 1024, // 2GB
    GCTargetPercentage: 70,
    
    // Streaming for large datasets
    StreamingThreshold: 1000, // Stream if >1000 records
    BufferSize:         100,  // Buffer size for streaming
}

engine.SetMemoryConfig(memoryConfig)
```

## Complete Examples

### Basic Web Scraper

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/valpere/DataScrapexter/pkg/scraper"
)

func main() {
    // Create engine
    engine, err := scraper.NewEngine(nil)
    if err != nil {
        log.Fatal(err)
    }
    defer engine.Shutdown(context.Background())
    
    // Configure scraper
    config := &scraper.ScraperConfig{
        Name:      "basic_scraper",
        BaseURL:   "https://quotes.toscrape.com/",
        RateLimit: "2s",
        
        Fields: []scraper.Field{
            {
                Name:     "quote",
                Selector: ".quote .text",
                Type:     scraper.FieldTypeText,
                Required: true,
            },
            {
                Name:     "author",
                Selector: ".quote .author",
                Type:     scraper.FieldTypeText,
                Required: true,
            },
        },
        
        Output: scraper.OutputConfig{
            Format: scraper.OutputFormatJSON,
            File:   "quotes.json",
        },
    }
    
    // Execute scraping
    ctx := context.Background()
    result, err := engine.Scrape(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Scraped %d pages, extracted %d records", 
               result.TotalPages, result.TotalRecords)
}
```

### Advanced E-commerce Scraper

```go
package main

import (
    "context"
    "log"
    "os"
    "time"
    
    "github.com/valpere/DataScrapexter/pkg/scraper"
    "github.com/valpere/DataScrapexter/internal/monitoring"
    "github.com/valpere/DataScrapexter/internal/antidetect"
)

func main() {
    // Initialize monitoring
    metricsManager := monitoring.NewMetricsManager(monitoring.MetricsConfig{
        Namespace: "ecommerce_scraper",
        Enabled:   true,
    })
    
    // Initialize anti-detection
    fingerprintEvader := antidetect.NewFingerprintingEvader(true)
    
    // Create engine with advanced configuration
    engineConfig := &scraper.EngineConfig{
        MaxRetries:      5,
        Timeout:         45 * time.Second,
        RateLimit:       3 * time.Second,
        MaxConcurrency:  5,
        MetricsManager:  metricsManager,
    }
    
    engine, err := scraper.NewEngine(engineConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer engine.Shutdown(context.Background())
    
    // Configure advanced scraper
    config := &scraper.ScraperConfig{
        Name:    "ecommerce_monitor",
        BaseURL: "https://example-store.com/products",
        
        // Anti-detection configuration
        AntiDetection: &scraper.AntiDetectionConfig{
            Fingerprinting: &scraper.FingerprintingConfig{
                Enabled: true,
                Evader:  fingerprintEvader,
            },
            Captcha: &scraper.CaptchaConfig{
                Enabled: true,
                Service: "2captcha",
                APIKey:  os.Getenv("CAPTCHA_API_KEY"),
            },
        },
        
        // Proxy configuration
        Proxy: &scraper.ProxyConfig{
            Enabled:  true,
            Rotation: "random",
            Providers: []scraper.ProxyProvider{
                {
                    URL:      os.Getenv("PROXY_URL"),
                    Username: os.Getenv("PROXY_USER"),
                    Password: os.Getenv("PROXY_PASS"),
                },
            },
        },
        
        // Field extraction
        Fields: []scraper.Field{
            {
                Name:     "title",
                Selector: "h1.product-title",
                Type:     scraper.FieldTypeText,
                Required: true,
                Transform: []scraper.TransformRule{
                    {Type: scraper.TransformTrim},
                    {Type: scraper.TransformNormalizeSpaces},
                },
            },
            {
                Name:     "price",
                Selector: ".price-current",
                Type:     scraper.FieldTypeText,
                Required: true,
                Transform: []scraper.TransformRule{
                    {Type: scraper.TransformCleanPrice},
                    {Type: scraper.TransformParseFloat},
                },
                Validation: &scraper.FieldValidation{
                    Required: true,
                    Custom: func(value interface{}) error {
                        if price, ok := value.(float64); ok && price <= 0 {
                            return fmt.Errorf("invalid price: %f", price)
                        }
                        return nil
                    },
                },
            },
        },
        
        // Pagination
        Pagination: &scraper.PaginationConfig{
            Type:     "next_button",
            Selector: ".pagination .next",
            MaxPages: 10,
        },
        
        // Multiple outputs
        Output: scraper.OutputConfig{
            Multiple: true,
            Outputs: []scraper.OutputConfig{
                {
                    Format: scraper.OutputFormatExcel,
                    File:   "products.xlsx",
                    Excel: &scraper.ExcelConfig{
                        IncludeHeaders: true,
                        AutoFilter:     true,
                    },
                },
                {
                    Format: scraper.OutputFormatDatabase,
                    Database: &scraper.DatabaseConfig{
                        Driver:   "postgresql",
                        Host:     "localhost",
                        Database: "ecommerce_data",
                        Username: "scraper",
                        Password: os.Getenv("DB_PASSWORD"),
                        Table:    "products",
                    },
                },
            },
        },
    }
    
    // Execute scraping with monitoring
    ctx := context.Background()
    startTime := time.Now()
    
    result, err := engine.Scrape(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Log results
    duration := time.Since(startTime)
    log.Printf("Scraping completed in %v", duration)
    log.Printf("Pages: %d, Records: %d, Errors: %d", 
               result.TotalPages, 
               result.TotalRecords, 
               len(result.Errors))
    
    // Record final metrics
    metricsManager.RecordJobComplete("ecommerce-job", "scheduled", duration)
}
```

This API reference provides comprehensive coverage of DataScrapexter's Go programming interface. Use these APIs to integrate web scraping capabilities into your Go applications with full control over configuration, execution, and monitoring.
