// internal/types.go
package internal

import (
	"time"
)

// Common types used across the internal package hierarchy

// BaseConfig represents common configuration fields
type BaseConfig struct {
	Timeout    time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	RetryDelay time.Duration `yaml:"retry_delay,omitempty" json:"retry_delay,omitempty"`
	MaxRetries int           `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	UserAgent  string        `yaml:"user_agent,omitempty" json:"user_agent,omitempty"`
	RateLimit  time.Duration `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	BurstSize  int           `yaml:"burst_size,omitempty" json:"burst_size,omitempty"`
}

// RequestMetadata contains request-specific metadata
type RequestMetadata struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers,omitempty"`
	StatusCode    int               `json:"status_code"`
	ResponseTime  time.Duration     `json:"response_time"`
	Timestamp     time.Time         `json:"timestamp"`
	UserAgent     string            `json:"user_agent,omitempty"`
	ContentLength int64             `json:"content_length,omitempty"`
}

// ErrorContext provides context for errors
type ErrorContext struct {
	Operation string                 `json:"operation"`
	URL       string                 `json:"url,omitempty"`
	Field     string                 `json:"field,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ValidationError represents validation-specific errors
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// Progress represents operation progress
type Progress struct {
	Current   int    `json:"current"`
	Total     int    `json:"total"`
	Operation string `json:"operation"`
	Message   string `json:"message,omitempty"`
}
