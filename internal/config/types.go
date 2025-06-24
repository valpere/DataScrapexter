// internal/config/types.go
package config

import (
    "fmt"
    "net/url"
    "regexp"
    "strings"
    "time"
)

// ScraperConfig represents the complete configuration for a scraping job
type ScraperConfig struct {
    Name            string             `json:"name" yaml:"name"`
    BaseURL         string             `json:"base_url" yaml:"base_url"`
    URLs            []string           `json:"urls,omitempty" yaml:"urls,omitempty"`
    Fields          []FieldConfig      `json:"fields" yaml:"fields"`
    Pagination      *PaginationConfig  `json:"pagination,omitempty" yaml:"pagination,omitempty"`
    Output          OutputConfig       `json:"output" yaml:"output"`
    Browser         *BrowserConfig     `json:"browser,omitempty" yaml:"browser,omitempty"`
    AntiDetection   *AntiDetectionConfig `json:"anti_detection,omitempty" yaml:"anti_detection,omitempty"`
    RateLimit       string             `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
    MaxPages        int                `json:"max_pages,omitempty" yaml:"max_pages,omitempty"`
    Concurrency     int                `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
    RequestTimeout  time.Duration      `json:"request_timeout,omitempty" yaml:"request_timeout,omitempty"`
    RetryAttempts   int                `json:"retry_attempts,omitempty" yaml:"retry_attempts,omitempty"`
    RetryDelay      time.Duration      `json:"retry_delay,omitempty" yaml:"retry_delay,omitempty"`
    UserAgents      []string           `json:"user_agents,omitempty" yaml:"user_agents,omitempty"`
    Headers         map[string]string  `json:"headers,omitempty" yaml:"headers,omitempty"`
    Cookies         map[string]string  `json:"cookies,omitempty" yaml:"cookies,omitempty"`
    EnableMetrics   bool               `json:"enable_metrics,omitempty" yaml:"enable_metrics,omitempty"`
    LogLevel        string             `json:"log_level,omitempty" yaml:"log_level,omitempty"`
}

// FieldConfig defines how to extract a specific field from the page
type FieldConfig struct {
    Name       string          `json:"name" yaml:"name"`
    Selector   string          `json:"selector" yaml:"selector"`
    Type       string          `json:"type" yaml:"type"`
    Attribute  string          `json:"attribute,omitempty" yaml:"attribute,omitempty"`
    Required   bool            `json:"required,omitempty" yaml:"required,omitempty"`
    Multiple   bool            `json:"multiple,omitempty" yaml:"multiple,omitempty"`
    Transform  []TransformRule `json:"transform,omitempty" yaml:"transform,omitempty"`
    Default    interface{}     `json:"default,omitempty" yaml:"default,omitempty"`
    Validation *ValidationRule `json:"validation,omitempty" yaml:"validation,omitempty"`
}

// TransformRule defines a data transformation operation
type TransformRule struct {
    Type        string                 `json:"type" yaml:"type"`
    Pattern     string                 `json:"pattern,omitempty" yaml:"pattern,omitempty"`
    Replacement string                 `json:"replacement,omitempty" yaml:"replacement,omitempty"`
    Params      map[string]interface{} `json:"params,omitempty" yaml:"params,omitempty"`
}

// ValidationRule defines validation criteria for extracted data
type ValidationRule struct {
    MinLength *int    `json:"min_length,omitempty" yaml:"min_length,omitempty"`
    MaxLength *int    `json:"max_length,omitempty" yaml:"max_length,omitempty"`
    Pattern   string  `json:"pattern,omitempty" yaml:"pattern,omitempty"`
    Required  bool    `json:"required,omitempty" yaml:"required,omitempty"`
}

// PaginationConfig defines pagination strategy and parameters
type PaginationConfig struct {
    Type         string            `json:"type" yaml:"type"`
    Selector     string            `json:"selector,omitempty" yaml:"selector,omitempty"`
    URLPattern   string            `json:"url_pattern,omitempty" yaml:"url_pattern,omitempty"`
    MaxPages     int               `json:"max_pages" yaml:"max_pages"`
    StartPage    int               `json:"start_page,omitempty" yaml:"start_page,omitempty"`
    PageParam    string            `json:"page_param,omitempty" yaml:"page_param,omitempty"`
    OffsetParam  string            `json:"offset_param,omitempty" yaml:"offset_param,omitempty"`
    LimitParam   string            `json:"limit_param,omitempty" yaml:"limit_param,omitempty"`
    PageSize     int               `json:"page_size,omitempty" yaml:"page_size,omitempty"`
    CursorField  string            `json:"cursor_field,omitempty" yaml:"cursor_field,omitempty"`
    StopCondition *StopCondition   `json:"stop_condition,omitempty" yaml:"stop_condition,omitempty"`
    Headers      map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
    Delay        time.Duration     `json:"delay,omitempty" yaml:"delay,omitempty"`
}

// StopCondition defines when to stop pagination
type StopCondition struct {
    NoNewItems     bool   `json:"no_new_items,omitempty" yaml:"no_new_items,omitempty"`
    EmptyPage      bool   `json:"empty_page,omitempty" yaml:"empty_page,omitempty"`
    ErrorThreshold int    `json:"error_threshold,omitempty" yaml:"error_threshold,omitempty"`
    MaxErrors      int    `json:"max_errors,omitempty" yaml:"max_errors,omitempty"`
    Pattern        string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

// OutputConfig defines output format and destination
type OutputConfig struct {
    Format       string            `json:"format" yaml:"format"`
    File         string            `json:"file,omitempty" yaml:"file,omitempty"`
    Append       bool              `json:"append,omitempty" yaml:"append,omitempty"`
    Compression  string            `json:"compression,omitempty" yaml:"compression,omitempty"`
    Streaming    bool              `json:"streaming,omitempty" yaml:"streaming,omitempty"`
    BufferSize   int               `json:"buffer_size,omitempty" yaml:"buffer_size,omitempty"`
    ThreadSafe   bool              `json:"thread_safe,omitempty" yaml:"thread_safe,omitempty"`
    Template     string            `json:"template,omitempty" yaml:"template,omitempty"`
    Headers      map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
    Webhook      *WebhookConfig    `json:"webhook,omitempty" yaml:"webhook,omitempty"`
    Database     *DatabaseConfig   `json:"database,omitempty" yaml:"database,omitempty"`
}

// WebhookConfig defines webhook delivery settings
type WebhookConfig struct {
    URL         string            `json:"url" yaml:"url"`
    Method      string            `json:"method,omitempty" yaml:"method,omitempty"`
    Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
    Timeout     time.Duration     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
    RetryCount  int               `json:"retry_count,omitempty" yaml:"retry_count,omitempty"`
    RetryDelay  time.Duration     `json:"retry_delay,omitempty" yaml:"retry_delay,omitempty"`
}

// DatabaseConfig defines database output settings
type DatabaseConfig struct {
    Driver       string `json:"driver" yaml:"driver"`
    DSN          string `json:"dsn" yaml:"dsn"`
    Table        string `json:"table" yaml:"table"`
    BatchSize    int    `json:"batch_size,omitempty" yaml:"batch_size,omitempty"`
    UpsertMode   bool   `json:"upsert_mode,omitempty" yaml:"upsert_mode,omitempty"`
    KeyFields    []string `json:"key_fields,omitempty" yaml:"key_fields,omitempty"`
}

// BrowserConfig defines browser automation settings
type BrowserConfig struct {
    Enabled          bool              `json:"enabled" yaml:"enabled"`
    Headless         bool              `json:"headless,omitempty" yaml:"headless,omitempty"`
    Timeout          time.Duration     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
    UserAgent        string            `json:"user_agent,omitempty" yaml:"user_agent,omitempty"`
    ViewportWidth    int               `json:"viewport_width,omitempty" yaml:"viewport_width,omitempty"`
    ViewportHeight   int               `json:"viewport_height,omitempty" yaml:"viewport_height,omitempty"`
    DisableImages    bool              `json:"disable_images,omitempty" yaml:"disable_images,omitempty"`
    DisableJavaScript bool             `json:"disable_javascript,omitempty" yaml:"disable_javascript,omitempty"`
    Extensions       []string          `json:"extensions,omitempty" yaml:"extensions,omitempty"`
    Args             []string          `json:"args,omitempty" yaml:"args,omitempty"`
    Cookies          map[string]string `json:"cookies,omitempty" yaml:"cookies,omitempty"`
    LocalStorage     map[string]string `json:"local_storage,omitempty" yaml:"local_storage,omitempty"`
    WaitConditions   []WaitCondition   `json:"wait_conditions,omitempty" yaml:"wait_conditions,omitempty"`
}

// WaitCondition defines browser wait conditions
type WaitCondition struct {
    Type      string        `json:"type" yaml:"type"`
    Selector  string        `json:"selector,omitempty" yaml:"selector,omitempty"`
    Text      string        `json:"text,omitempty" yaml:"text,omitempty"`
    Timeout   time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
    Visible   bool          `json:"visible,omitempty" yaml:"visible,omitempty"`
}

// AntiDetectionConfig defines anti-detection mechanisms
type AntiDetectionConfig struct {
    UserAgents       []string      `json:"user_agents,omitempty" yaml:"user_agents,omitempty"`
    DelayMin         time.Duration `json:"delay_min,omitempty" yaml:"delay_min,omitempty"`
    DelayMax         time.Duration `json:"delay_max,omitempty" yaml:"delay_max,omitempty"`
    RandomizeHeaders bool          `json:"randomize_headers,omitempty" yaml:"randomize_headers,omitempty"`
    Proxy            ProxyConfig   `json:"proxy,omitempty" yaml:"proxy,omitempty"`
    Captcha          CaptchaConfig `json:"captcha,omitempty" yaml:"captcha,omitempty"`
    RateLimiting     RateLimitConfig `json:"rate_limiting,omitempty" yaml:"rate_limiting,omitempty"`
    SessionRotation  SessionConfig `json:"session_rotation,omitempty" yaml:"session_rotation,omitempty"`
}

// ProxyConfig defines proxy settings
type ProxyConfig struct {
    Enabled          bool     `json:"enabled" yaml:"enabled"`
    URLs             []string `json:"urls,omitempty" yaml:"urls,omitempty"`
    RotationStrategy string   `json:"rotation_strategy,omitempty" yaml:"rotation_strategy,omitempty"`
    HealthCheck      bool     `json:"health_check,omitempty" yaml:"health_check,omitempty"`
    Timeout          time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
    MaxRetries       int      `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
}

// CaptchaConfig defines CAPTCHA solving settings
type CaptchaConfig struct {
    Enabled    bool          `json:"enabled" yaml:"enabled"`
    Service    string        `json:"service,omitempty" yaml:"service,omitempty"`
    APIKey     string        `json:"api_key,omitempty" yaml:"api_key,omitempty"`
    Timeout    time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
    MaxRetries int           `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
}

// RateLimitConfig defines rate limiting settings
type RateLimitConfig struct {
    RequestsPerSecond float64       `json:"requests_per_second,omitempty" yaml:"requests_per_second,omitempty"`
    Burst             int           `json:"burst,omitempty" yaml:"burst,omitempty"`
    Delay             time.Duration `json:"delay,omitempty" yaml:"delay,omitempty"`
}

// SessionConfig defines session management settings
type SessionConfig struct {
    Enabled         bool          `json:"enabled" yaml:"enabled"`
    RotationInterval time.Duration `json:"rotation_interval,omitempty" yaml:"rotation_interval,omitempty"`
    CookieJar       bool          `json:"cookie_jar,omitempty" yaml:"cookie_jar,omitempty"`
    PersistCookies  bool          `json:"persist_cookies,omitempty" yaml:"persist_cookies,omitempty"`
}

// Validation methods

// Validate validates the scraper configuration
func (sc *ScraperConfig) Validate() error {
    if strings.TrimSpace(sc.Name) == "" {
        return fmt.Errorf("scraper name is required")
    }

    if strings.TrimSpace(sc.BaseURL) == "" && len(sc.URLs) == 0 {
        return fmt.Errorf("base_url or urls must be specified")
    }

    if sc.BaseURL != "" {
        if _, err := url.Parse(sc.BaseURL); err != nil {
            return fmt.Errorf("invalid base_url: %v", err)
        }
    }

    for i, u := range sc.URLs {
        if _, err := url.Parse(u); err != nil {
            return fmt.Errorf("invalid url[%d]: %v", i, err)
        }
    }

    if len(sc.Fields) == 0 {
        return fmt.Errorf("at least one field must be defined")
    }

    for i, field := range sc.Fields {
        if err := field.Validate(); err != nil {
            return fmt.Errorf("field[%d]: %v", i, err)
        }
    }

    if sc.Pagination != nil {
        if err := sc.Pagination.Validate(); err != nil {
            return fmt.Errorf("pagination: %v", err)
        }
    }

    if err := sc.Output.Validate(); err != nil {
        return fmt.Errorf("output: %v", err)
    }

    if sc.Browser != nil {
        if err := sc.Browser.Validate(); err != nil {
            return fmt.Errorf("browser: %v", err)
        }
    }

    if sc.AntiDetection != nil {
        if err := sc.AntiDetection.Validate(); err != nil {
            return fmt.Errorf("anti_detection: %v", err)
        }
    }

    return nil
}

// Validate validates the field configuration
func (fc *FieldConfig) Validate() error {
    if strings.TrimSpace(fc.Name) == "" {
        return fmt.Errorf("field name is required")
    }

    if strings.TrimSpace(fc.Selector) == "" {
        return fmt.Errorf("field selector is required")
    }

    validTypes := []string{"text", "attribute", "html", "number", "float", "boolean", "date", "url", "email", "image", "list"}
    validType := false
    for _, vt := range validTypes {
        if fc.Type == vt {
            validType = true
            break
        }
    }
    if !validType {
        return fmt.Errorf("invalid field type: %s", fc.Type)
    }

    if fc.Type == "attribute" && strings.TrimSpace(fc.Attribute) == "" {
        return fmt.Errorf("attribute name is required for attribute type fields")
    }

    for i, transform := range fc.Transform {
        if err := transform.Validate(); err != nil {
            return fmt.Errorf("transform[%d]: %v", i, err)
        }
    }

    if fc.Validation != nil {
        if err := fc.Validation.Validate(); err != nil {
            return fmt.Errorf("validation: %v", err)
        }
    }

    return nil
}

// Validate validates the transform rule
func (tr *TransformRule) Validate() error {
    if strings.TrimSpace(tr.Type) == "" {
        return fmt.Errorf("transform type is required")
    }

    validTypes := []string{
        "trim", "lowercase", "uppercase", "normalize_spaces", "remove_html",
        "regex", "parse_float", "parse_int", "parse_date", "extract_number",
        "prefix", "suffix", "replace", "split", "join",
    }

    validType := false
    for _, vt := range validTypes {
        if tr.Type == vt {
            validType = true
            break
        }
    }
    if !validType {
        return fmt.Errorf("invalid transform type: %s", tr.Type)
    }

    // Validate parameters for transforms that require them
    switch tr.Type {
    case "regex":
        if tr.Pattern == "" {
            return fmt.Errorf("regex transform requires pattern parameter")
        }
        if _, err := regexp.Compile(tr.Pattern); err != nil {
            return fmt.Errorf("invalid regex pattern: %v", err)
        }
    case "prefix", "suffix":
        if value, ok := tr.Params["value"]; !ok || value == "" {
            return fmt.Errorf("%s transform requires value parameter", tr.Type)
        }
    case "replace":
        if old, ok := tr.Params["old"]; !ok || old == "" {
            return fmt.Errorf("replace transform requires old parameter")
        }
        if _, ok := tr.Params["new"]; !ok {
            return fmt.Errorf("replace transform requires new parameter")
        }
    }

    return nil
}

// Validate validates the validation rule
func (vr *ValidationRule) Validate() error {
    if vr.MinLength != nil && *vr.MinLength < 0 {
        return fmt.Errorf("min_length cannot be negative")
    }

    if vr.MaxLength != nil && *vr.MaxLength < 0 {
        return fmt.Errorf("max_length cannot be negative")
    }

    if vr.MinLength != nil && vr.MaxLength != nil && *vr.MinLength > *vr.MaxLength {
        return fmt.Errorf("min_length cannot be greater than max_length")
    }

    if vr.Pattern != "" {
        if _, err := regexp.Compile(vr.Pattern); err != nil {
            return fmt.Errorf("invalid validation pattern: %v", err)
        }
    }

    return nil
}

// Validate validates the pagination configuration
func (pc *PaginationConfig) Validate() error {
    if strings.TrimSpace(pc.Type) == "" {
        return fmt.Errorf("pagination type is required")
    }

    validTypes := []string{"offset", "cursor", "next_button", "numbered", "infinite_scroll"}
    validType := false
    for _, vt := range validTypes {
        if pc.Type == vt {
            validType = true
            break
        }
    }
    if !validType {
        return fmt.Errorf("invalid pagination type: %s", pc.Type)
    }

    if pc.MaxPages <= 0 {
        return fmt.Errorf("max_pages must be greater than 0")
    }

    switch pc.Type {
    case "next_button":
        if strings.TrimSpace(pc.Selector) == "" {
            return fmt.Errorf("selector is required for next_button pagination")
        }
    case "offset":
        if strings.TrimSpace(pc.URLPattern) == "" {
            return fmt.Errorf("url_pattern is required for offset pagination")
        }
    case "cursor":
        if strings.TrimSpace(pc.CursorField) == "" {
            return fmt.Errorf("cursor_field is required for cursor pagination")
        }
    }

    return nil
}

// Validate validates the output configuration
func (oc *OutputConfig) Validate() error {
    if strings.TrimSpace(oc.Format) == "" {
        return fmt.Errorf("output format is required")
    }

    validFormats := []string{"json", "csv", "xml", "yaml"}
    validFormat := false
    for _, vf := range validFormats {
        if oc.Format == vf {
            validFormat = true
            break
        }
    }
    if !validFormat {
        return fmt.Errorf("invalid output format: %s", oc.Format)
    }

    if !oc.Streaming && strings.TrimSpace(oc.File) == "" && oc.Webhook == nil && oc.Database == nil {
        return fmt.Errorf("file, webhook, or database must be specified for non-streaming output")
    }

    if oc.Compression != "" {
        validCompressions := []string{"none", "gzip", "zlib"}
        validCompression := false
        for _, vc := range validCompressions {
            if oc.Compression == vc {
                validCompression = true
                break
            }
        }
        if !validCompression {
            return fmt.Errorf("invalid compression: %s", oc.Compression)
        }
    }

    if oc.Webhook != nil {
        if err := oc.Webhook.Validate(); err != nil {
            return fmt.Errorf("webhook: %v", err)
        }
    }

    if oc.Database != nil {
        if err := oc.Database.Validate(); err != nil {
            return fmt.Errorf("database: %v", err)
        }
    }

    return nil
}

// Validate validates the webhook configuration
func (wc *WebhookConfig) Validate() error {
    if strings.TrimSpace(wc.URL) == "" {
        return fmt.Errorf("webhook URL is required")
    }

    if _, err := url.Parse(wc.URL); err != nil {
        return fmt.Errorf("invalid webhook URL: %v", err)
    }

    if wc.Method != "" {
        validMethods := []string{"GET", "POST", "PUT", "PATCH"}
        validMethod := false
        for _, vm := range validMethods {
            if wc.Method == vm {
                validMethod = true
                break
            }
        }
        if !validMethod {
            return fmt.Errorf("invalid webhook method: %s", wc.Method)
        }
    }

    return nil
}

// Validate validates the database configuration
func (dc *DatabaseConfig) Validate() error {
    if strings.TrimSpace(dc.Driver) == "" {
        return fmt.Errorf("database driver is required")
    }

    if strings.TrimSpace(dc.DSN) == "" {
        return fmt.Errorf("database DSN is required")
    }

    if strings.TrimSpace(dc.Table) == "" {
        return fmt.Errorf("database table is required")
    }

    validDrivers := []string{"postgres", "mysql", "sqlite", "sqlserver"}
    validDriver := false
    for _, vd := range validDrivers {
        if dc.Driver == vd {
            validDriver = true
            break
        }
    }
    if !validDriver {
        return fmt.Errorf("invalid database driver: %s", dc.Driver)
    }

    return nil
}

// Validate validates the browser configuration
func (bc *BrowserConfig) Validate() error {
    if bc.ViewportWidth < 0 || bc.ViewportHeight < 0 {
        return fmt.Errorf("viewport dimensions cannot be negative")
    }

    for i, condition := range bc.WaitConditions {
        if err := condition.Validate(); err != nil {
            return fmt.Errorf("wait_condition[%d]: %v", i, err)
        }
    }

    return nil
}

// Validate validates the wait condition
func (wc *WaitCondition) Validate() error {
    if strings.TrimSpace(wc.Type) == "" {
        return fmt.Errorf("wait condition type is required")
    }

    validTypes := []string{"element", "text", "url", "network", "timeout"}
    validType := false
    for _, vt := range validTypes {
        if wc.Type == vt {
            validType = true
            break
        }
    }
    if !validType {
        return fmt.Errorf("invalid wait condition type: %s", wc.Type)
    }

    switch wc.Type {
    case "element":
        if strings.TrimSpace(wc.Selector) == "" {
            return fmt.Errorf("selector is required for element wait condition")
        }
    case "text":
        if strings.TrimSpace(wc.Text) == "" {
            return fmt.Errorf("text is required for text wait condition")
        }
    }

    return nil
}

// Validate validates the anti-detection configuration
func (adc *AntiDetectionConfig) Validate() error {
    if adc.DelayMin < 0 || adc.DelayMax < 0 {
        return fmt.Errorf("delays cannot be negative")
    }

    if adc.DelayMin > adc.DelayMax {
        return fmt.Errorf("delay_min cannot be greater than delay_max")
    }

    if err := adc.Proxy.Validate(); err != nil {
        return fmt.Errorf("proxy: %v", err)
    }

    if err := adc.Captcha.Validate(); err != nil {
        return fmt.Errorf("captcha: %v", err)
    }

    if err := adc.RateLimiting.Validate(); err != nil {
        return fmt.Errorf("rate_limiting: %v", err)
    }

    if err := adc.SessionRotation.Validate(); err != nil {
        return fmt.Errorf("session_rotation: %v", err)
    }

    return nil
}

// Validate validates the proxy configuration
func (pc *ProxyConfig) Validate() error {
    if pc.Enabled && len(pc.URLs) == 0 {
        return fmt.Errorf("proxy URLs are required when proxy is enabled")
    }

    for i, proxyURL := range pc.URLs {
        if _, err := url.Parse(proxyURL); err != nil {
            return fmt.Errorf("invalid proxy URL[%d]: %v", i, err)
        }
    }

    if pc.RotationStrategy != "" {
        validStrategies := []string{"round_robin", "random", "least_used"}
        validStrategy := false
        for _, vs := range validStrategies {
            if pc.RotationStrategy == vs {
                validStrategy = true
                break
            }
        }
        if !validStrategy {
            return fmt.Errorf("invalid rotation strategy: %s", pc.RotationStrategy)
        }
    }

    return nil
}

// Validate validates the CAPTCHA configuration
func (cc *CaptchaConfig) Validate() error {
    if cc.Enabled && strings.TrimSpace(cc.Service) == "" {
        return fmt.Errorf("captcha service is required when captcha is enabled")
    }

    if cc.Enabled && strings.TrimSpace(cc.APIKey) == "" {
        return fmt.Errorf("captcha API key is required when captcha is enabled")
    }

    return nil
}

// Validate validates the rate limit configuration
func (rlc *RateLimitConfig) Validate() error {
    if rlc.RequestsPerSecond < 0 {
        return fmt.Errorf("requests_per_second cannot be negative")
    }

    if rlc.Burst < 0 {
        return fmt.Errorf("burst cannot be negative")
    }

    if rlc.Delay < 0 {
        return fmt.Errorf("delay cannot be negative")
    }

    return nil
}

// Validate validates the session configuration
func (sc *SessionConfig) Validate() error {
    if sc.Enabled && sc.RotationInterval < 0 {
        return fmt.Errorf("rotation_interval cannot be negative")
    }

    return nil
}
