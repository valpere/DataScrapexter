package api

import (
	"time"
)

// ScraperConfig represents the complete configuration for a scraping job
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

// Field represents a data field to extract from the page
type Field struct {
	Name        string            `yaml:"name" json:"name"`
	Selector    string            `yaml:"selector" json:"selector"`
	Type        string            `yaml:"type" json:"type"`                               // text, html, attr, list
	Attribute   string            `yaml:"attribute,omitempty" json:"attribute,omitempty"`   // For attr type
	Required    bool              `yaml:"required,omitempty" json:"required,omitempty"`
	Transform   []TransformRule   `yaml:"transform,omitempty" json:"transform,omitempty"`
}

// TransformRule defines a transformation to apply to extracted data
type TransformRule struct {
	Type        string `yaml:"type" json:"type"`                               // regex, trim, lowercase, uppercase, parse_float, parse_int
	Pattern     string `yaml:"pattern,omitempty" json:"pattern,omitempty"`     // For regex
	Replacement string `yaml:"replacement,omitempty" json:"replacement,omitempty"` // For regex
}

// ProxyConfig represents proxy configuration
type ProxyConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	URL      string   `yaml:"url,omitempty" json:"url,omitempty"`
	Rotation string   `yaml:"rotation,omitempty" json:"rotation,omitempty"` // random, round-robin
	List     []string `yaml:"list,omitempty" json:"list,omitempty"`
}

// PaginationConfig defines how to handle pagination
type PaginationConfig struct {
	Type        string `yaml:"type" json:"type"`                               // next_button, page_numbers, infinite_scroll
	Selector    string `yaml:"selector,omitempty" json:"selector,omitempty"`   // CSS selector for pagination element
	MaxPages    int    `yaml:"max_pages,omitempty" json:"max_pages,omitempty"`
	URLPattern  string `yaml:"url_pattern,omitempty" json:"url_pattern,omitempty"` // For page number patterns
	StartPage   int    `yaml:"start_page,omitempty" json:"start_page,omitempty"`
}

// OutputConfig defines how to output the scraped data
type OutputConfig struct {
	Format   string         `yaml:"format" json:"format"`     // json, csv, excel
	File     string         `yaml:"file,omitempty" json:"file,omitempty"`
	Database *DatabaseConfig `yaml:"database,omitempty" json:"database,omitempty"`
}

// DatabaseConfig represents database output configuration
type DatabaseConfig struct {
	Type     string `yaml:"type" json:"type"`         // postgresql, mysql, mongodb
	URL      string `yaml:"url" json:"url"`
	Table    string `yaml:"table" json:"table"`
}

// ScrapeJob represents a scraping job
type ScrapeJob struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Config      ScraperConfig  `json:"config"`
	Status      JobStatus      `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Error       string         `json:"error,omitempty"`
	Progress    JobProgress    `json:"progress"`
}

// JobStatus represents the status of a scraping job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// JobProgress tracks the progress of a scraping job
type JobProgress struct {
	TotalPages      int `json:"total_pages"`
	ProcessedPages  int `json:"processed_pages"`
	SuccessfulPages int `json:"successful_pages"`
	FailedPages     int `json:"failed_pages"`
	TotalItems      int `json:"total_items"`
}

// ScrapeResult represents the result of a single page scrape
type ScrapeResult struct {
	URL        string                 `json:"url"`
	StatusCode int                    `json:"status_code"`
	Data       map[string]interface{} `json:"data"`
	Error      string                 `json:"error,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// ScrapeResponse represents the complete response for a scraping job
type ScrapeResponse struct {
	JobID    string         `json:"job_id"`
	Status   JobStatus      `json:"status"`
	Results  []ScrapeResult `json:"results"`
	Progress JobProgress    `json:"progress"`
	Error    string         `json:"error,omitempty"`
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ConfigValidationResult represents the result of configuration validation
type ConfigValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// HealthStatus represents the health status of the scraper service
type HealthStatus struct {
	Status    string            `json:"status"` // healthy, degraded, unhealthy
	Version   string            `json:"version"`
	Uptime    time.Duration     `json:"uptime"`
	Checks    map[string]bool   `json:"checks"`
	Timestamp time.Time         `json:"timestamp"`
}

// MetricsSnapshot represents a snapshot of scraper metrics
type MetricsSnapshot struct {
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	ActiveJobs          int           `json:"active_jobs"`
	QueuedJobs          int           `json:"queued_jobs"`
	Timestamp           time.Time     `json:"timestamp"`
}
