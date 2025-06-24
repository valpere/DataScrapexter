// internal/config/config.go
package config

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"

    "gopkg.in/yaml.v3"
)

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(filename string) (*ScraperConfig, error) {
    if filename == "" {
        return nil, fmt.Errorf("configuration filename cannot be empty")
    }

    // Check if file exists
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return nil, fmt.Errorf("configuration file not found: %s", filename)
    }

    // Read file content
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read configuration file: %v", err)
    }

    return LoadFromBytes(data)
}

// LoadFromBytes loads configuration from YAML bytes
func LoadFromBytes(data []byte) (*ScraperConfig, error) {
    if len(data) == 0 {
        return nil, fmt.Errorf("configuration data cannot be empty")
    }

    // Substitute environment variables
    expandedData := expandEnvironmentVariables(string(data))

    var config ScraperConfig
    if err := yaml.Unmarshal([]byte(expandedData), &config); err != nil {
        return nil, fmt.Errorf("failed to parse YAML configuration: %v", err)
    }

    // Apply defaults
    applyDefaults(&config)

    // Validate configuration
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %v", err)
    }

    return &config, nil
}

// LoadFromReader loads configuration from an io.Reader
func LoadFromReader(reader io.Reader) (*ScraperConfig, error) {
    if reader == nil {
        return nil, fmt.Errorf("reader cannot be nil")
    }

    data, err := io.ReadAll(reader)
    if err != nil {
        return nil, fmt.Errorf("failed to read from reader: %v", err)
    }

    return LoadFromBytes(data)
}

// SaveToFile saves configuration to a YAML file
func SaveToFile(config *ScraperConfig, filename string) error {
    if config == nil {
        return fmt.Errorf("configuration cannot be nil")
    }

    if filename == "" {
        return fmt.Errorf("filename cannot be empty")
    }

    // Validate configuration before saving
    if err := config.Validate(); err != nil {
        return fmt.Errorf("invalid configuration: %v", err)
    }

    // Marshal to YAML
    data, err := yaml.Marshal(config)
    if err != nil {
        return fmt.Errorf("failed to marshal configuration to YAML: %v", err)
    }

    // Ensure directory exists
    dir := filepath.Dir(filename)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory %s: %v", dir, err)
    }

    // Write to file
    if err := os.WriteFile(filename, data, 0644); err != nil {
        return fmt.Errorf("failed to write configuration file: %v", err)
    }

    return nil
}

// SaveToWriter saves configuration to an io.Writer
func SaveToWriter(config *ScraperConfig, writer io.Writer) error {
    if config == nil {
        return fmt.Errorf("configuration cannot be nil")
    }

    if writer == nil {
        return fmt.Errorf("writer cannot be nil")
    }

    // Validate configuration before saving
    if err := config.Validate(); err != nil {
        return fmt.Errorf("invalid configuration: %v", err)
    }

    // Marshal to YAML
    data, err := yaml.Marshal(config)
    if err != nil {
        return fmt.Errorf("failed to marshal configuration to YAML: %v", err)
    }

    // Write to writer
    if _, err := writer.Write(data); err != nil {
        return fmt.Errorf("failed to write configuration: %v", err)
    }

    return nil
}

// MergeConfigs merges multiple configurations, with later configs overriding earlier ones
func MergeConfigs(configs ...*ScraperConfig) (*ScraperConfig, error) {
    if len(configs) == 0 {
        return nil, fmt.Errorf("at least one configuration is required")
    }

    // Start with the first config
    merged := *configs[0]

    // Merge each subsequent config
    for i := 1; i < len(configs); i++ {
        if configs[i] == nil {
            continue
        }

        mergeConfig(&merged, configs[i])
    }

    // Apply defaults to merged config
    applyDefaults(&merged)

    // Validate merged configuration
    if err := merged.Validate(); err != nil {
        return nil, fmt.Errorf("merged configuration is invalid: %v", err)
    }

    return &merged, nil
}

// GenerateTemplate generates a template configuration for the specified type
func GenerateTemplate(templateType string) ScraperConfig {
    switch strings.ToLower(templateType) {
    case "ecommerce":
        return generateEcommerceTemplate()
    case "news":
        return generateNewsTemplate()
    case "api":
        return generateAPITemplate()
    case "basic":
        return generateBasicTemplate()
    default:
        return generateBasicTemplate()
    }
}

// ValidateConfig validates a configuration and returns detailed error information
func ValidateConfig(config *ScraperConfig) []ValidationError {
    var errors []ValidationError

    if config == nil {
        errors = append(errors, ValidationError{
            Path:    "config",
            Message: "configuration cannot be nil",
        })
        return errors
    }

    if err := config.Validate(); err != nil {
        errors = append(errors, ValidationError{
            Path:    "config",
            Message: err.Error(),
        })
    }

    return errors
}

// ValidationError represents a configuration validation error
type ValidationError struct {
    Path    string `json:"path"`
    Message string `json:"message"`
}

// Error implements the error interface
func (ve ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", ve.Path, ve.Message)
}

// Helper functions

// expandEnvironmentVariables substitutes environment variables in the configuration
func expandEnvironmentVariables(content string) string {
    return os.ExpandEnv(content)
}

// applyDefaults applies default values to the configuration
func applyDefaults(config *ScraperConfig) {
    if config.RequestTimeout == 0 {
        config.RequestTimeout = 30 * time.Second
    }

    if config.RetryAttempts == 0 {
        config.RetryAttempts = 3
    }

    if config.RetryDelay == 0 {
        config.RetryDelay = 1 * time.Second
    }

    if config.Concurrency == 0 {
        config.Concurrency = 1
    }

    if config.LogLevel == "" {
        config.LogLevel = "info"
    }

    // Apply defaults to output configuration
    if config.Output.Format == "" {
        config.Output.Format = "json"
    }

    if config.Output.BufferSize == 0 {
        config.Output.BufferSize = 1000
    }

    // Apply defaults to pagination if present
    if config.Pagination != nil {
        if config.Pagination.StartPage == 0 {
            config.Pagination.StartPage = 1
        }
        if config.Pagination.PageSize == 0 {
            config.Pagination.PageSize = 20
        }
    }

    // Apply defaults to browser configuration if present
    if config.Browser != nil {
        if config.Browser.Timeout == 0 {
            config.Browser.Timeout = 30 * time.Second
        }
        if config.Browser.ViewportWidth == 0 {
            config.Browser.ViewportWidth = 1920
        }
        if config.Browser.ViewportHeight == 0 {
            config.Browser.ViewportHeight = 1080
        }
    }

    // Apply defaults to anti-detection configuration if present
    if config.AntiDetection != nil {
        if config.AntiDetection.DelayMin == 0 {
            config.AntiDetection.DelayMin = 100 * time.Millisecond
        }
        if config.AntiDetection.DelayMax == 0 {
            config.AntiDetection.DelayMax = 2 * time.Second
        }

        // Apply proxy defaults
        if config.AntiDetection.Proxy.Enabled {
            if config.AntiDetection.Proxy.RotationStrategy == "" {
                config.AntiDetection.Proxy.RotationStrategy = "round_robin"
            }
            if config.AntiDetection.Proxy.Timeout == 0 {
                config.AntiDetection.Proxy.Timeout = 10 * time.Second
            }
            if config.AntiDetection.Proxy.MaxRetries == 0 {
                config.AntiDetection.Proxy.MaxRetries = 3
            }
        }

        // Apply CAPTCHA defaults
        if config.AntiDetection.Captcha.Enabled {
            if config.AntiDetection.Captcha.Timeout == 0 {
                config.AntiDetection.Captcha.Timeout = 60 * time.Second
            }
            if config.AntiDetection.Captcha.MaxRetries == 0 {
                config.AntiDetection.Captcha.MaxRetries = 3
            }
        }

        // Apply rate limiting defaults
        if config.AntiDetection.RateLimiting.RequestsPerSecond == 0 {
            config.AntiDetection.RateLimiting.RequestsPerSecond = 1.0
        }
        if config.AntiDetection.RateLimiting.Burst == 0 {
            config.AntiDetection.RateLimiting.Burst = 5
        }

        // Apply session rotation defaults
        if config.AntiDetection.SessionRotation.Enabled {
            if config.AntiDetection.SessionRotation.RotationInterval == 0 {
                config.AntiDetection.SessionRotation.RotationInterval = 10 * time.Minute
            }
        }
    }

    // Apply defaults to webhook configuration if present
    if config.Output.Webhook != nil {
        if config.Output.Webhook.Method == "" {
            config.Output.Webhook.Method = "POST"
        }
        if config.Output.Webhook.Timeout == 0 {
            config.Output.Webhook.Timeout = 30 * time.Second
        }
        if config.Output.Webhook.RetryCount == 0 {
            config.Output.Webhook.RetryCount = 3
        }
        if config.Output.Webhook.RetryDelay == 0 {
            config.Output.Webhook.RetryDelay = 1 * time.Second
        }
    }

    // Apply defaults to database configuration if present
    if config.Output.Database != nil {
        if config.Output.Database.BatchSize == 0 {
            config.Output.Database.BatchSize = 100
        }
    }
}

// mergeConfig merges source configuration into target
func mergeConfig(target, source *ScraperConfig) {
    if source.Name != "" {
        target.Name = source.Name
    }
    if source.BaseURL != "" {
        target.BaseURL = source.BaseURL
    }
    if len(source.URLs) > 0 {
        target.URLs = source.URLs
    }
    if len(source.Fields) > 0 {
        target.Fields = source.Fields
    }
    if source.Pagination != nil {
        target.Pagination = source.Pagination
    }
    if source.Output.Format != "" {
        target.Output = source.Output
    }
    if source.Browser != nil {
        target.Browser = source.Browser
    }
    if source.AntiDetection != nil {
        target.AntiDetection = source.AntiDetection
    }
    if source.RateLimit != "" {
        target.RateLimit = source.RateLimit
    }
    if source.MaxPages > 0 {
        target.MaxPages = source.MaxPages
    }
    if source.Concurrency > 0 {
        target.Concurrency = source.Concurrency
    }
    if source.RequestTimeout > 0 {
        target.RequestTimeout = source.RequestTimeout
    }
    if source.RetryAttempts > 0 {
        target.RetryAttempts = source.RetryAttempts
    }
    if source.RetryDelay > 0 {
        target.RetryDelay = source.RetryDelay
    }
    if len(source.UserAgents) > 0 {
        target.UserAgents = source.UserAgents
    }
    if len(source.Headers) > 0 {
        target.Headers = source.Headers
    }
    if len(source.Cookies) > 0 {
        target.Cookies = source.Cookies
    }
    if source.LogLevel != "" {
        target.LogLevel = source.LogLevel
    }
}

// Template generation functions

func generateBasicTemplate() ScraperConfig {
    return ScraperConfig{
        Name:    "basic_scraper",
        BaseURL: "https://example.com",
        Fields: []FieldConfig{
            {
                Name:     "title",
                Selector: "h1",
                Type:     "text",
                Required: true,
                Transform: []TransformRule{
                    {Type: "trim"},
                    {Type: "normalize_spaces"},
                },
            },
            {
                Name:     "description",
                Selector: ".description",
                Type:     "text",
                Transform: []TransformRule{
                    {Type: "trim"},
                    {Type: "remove_html"},
                },
            },
        },
        Output: OutputConfig{
            Format: "json",
            File:   "output.json",
        },
        RequestTimeout: 30 * time.Second,
        RetryAttempts:  3,
        Concurrency:    1,
    }
}

func generateEcommerceTemplate() ScraperConfig {
    return ScraperConfig{
        Name:    "ecommerce_scraper",
        BaseURL: "https://shop.example.com",
        Fields: []FieldConfig{
            {
                Name:     "product_name",
                Selector: "h1.product-title",
                Type:     "text",
                Required: true,
                Transform: []TransformRule{
                    {Type: "trim"},
                    {Type: "normalize_spaces"},
                },
            },
            {
                Name:     "price",
                Selector: ".price",
                Type:     "text",
                Required: true,
                Transform: []TransformRule{
                    {Type: "trim"},
                    {
                        Type:        "regex",
                        Pattern:     `\$([0-9,]+\.?[0-9]*)`,
                        Replacement: "$1",
                    },
                    {Type: "parse_float"},
                },
            },
            {
                Name:     "description",
                Selector: ".product-description",
                Type:     "text",
                Transform: []TransformRule{
                    {Type: "remove_html"},
                    {Type: "trim"},
                    {Type: "normalize_spaces"},
                },
            },
            {
                Name:      "image_url",
                Selector:  ".product-image img",
                Type:      "attribute",
                Attribute: "src",
            },
            {
                Name:     "availability",
                Selector: ".stock-status",
                Type:     "text",
                Transform: []TransformRule{
                    {Type: "trim"},
                    {Type: "lowercase"},
                },
            },
        },
        Pagination: &PaginationConfig{
            Type:     "next_button",
            Selector: ".pagination .next",
            MaxPages: 50,
        },
        Output: OutputConfig{
            Format: "json",
            File:   "ecommerce_products.json",
        },
        AntiDetection: &AntiDetectionConfig{
            UserAgents: []string{
                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
            },
            DelayMin:         500 * time.Millisecond,
            DelayMax:         2 * time.Second,
            RandomizeHeaders: true,
        },
        RequestTimeout: 30 * time.Second,
        RetryAttempts:  3,
        Concurrency:    2,
    }
}

func generateNewsTemplate() ScraperConfig {
    return ScraperConfig{
        Name:    "news_scraper",
        BaseURL: "https://news.example.com",
        Fields: []FieldConfig{
            {
                Name:     "headline",
                Selector: "h1, .headline",
                Type:     "text",
                Required: true,
                Transform: []TransformRule{
                    {Type: "trim"},
                    {Type: "normalize_spaces"},
                },
            },
            {
                Name:     "content",
                Selector: ".article-content",
                Type:     "text",
                Required: true,
                Transform: []TransformRule{
                    {Type: "remove_html"},
                    {Type: "trim"},
                    {Type: "normalize_spaces"},
                },
            },
            {
                Name:     "author",
                Selector: ".author, .byline",
                Type:     "text",
                Transform: []TransformRule{
                    {Type: "trim"},
                },
            },
            {
                Name:     "published_date",
                Selector: ".publish-date, time",
                Type:     "date",
                Transform: []TransformRule{
                    {Type: "trim"},
                    {Type: "parse_date"},
                },
            },
            {
                Name:     "category",
                Selector: ".category, .section",
                Type:     "text",
                Transform: []TransformRule{
                    {Type: "trim"},
                    {Type: "lowercase"},
                },
            },
        },
        Pagination: &PaginationConfig{
            Type:     "numbered",
            MaxPages: 20,
            URLPattern: "{base_url}/page/{page}",
        },
        Output: OutputConfig{
            Format: "json",
            File:   "news_articles.json",
        },
        AntiDetection: &AntiDetectionConfig{
            DelayMin:         1 * time.Second,
            DelayMax:         3 * time.Second,
            RandomizeHeaders: true,
        },
        RequestTimeout: 45 * time.Second,
        RetryAttempts:  3,
        Concurrency:    1,
    }
}

func generateAPITemplate() ScraperConfig {
    return ScraperConfig{
        Name:    "api_scraper",
        BaseURL: "https://api.example.com/data",
        Fields: []FieldConfig{
            {
                Name:     "id",
                Selector: "id",
                Type:     "number",
                Required: true,
            },
            {
                Name:     "name",
                Selector: "name",
                Type:     "text",
                Required: true,
                Transform: []TransformRule{
                    {Type: "trim"},
                },
            },
            {
                Name:     "value",
                Selector: "value",
                Type:     "float",
            },
            {
                Name:     "active",
                Selector: "active",
                Type:     "boolean",
            },
        },
        Pagination: &PaginationConfig{
            Type:        "offset",
            MaxPages:    100,
            URLPattern:  "{base_url}?offset={offset}&limit={limit}",
            OffsetParam: "offset",
            LimitParam:  "limit",
            PageSize:    50,
        },
        Output: OutputConfig{
            Format: "json",
            File:   "api_data.json",
        },
        Headers: map[string]string{
            "Accept":       "application/json",
            "Content-Type": "application/json",
        },
        RequestTimeout: 30 * time.Second,
        RetryAttempts:  5,
        Concurrency:    3,
    }
}
