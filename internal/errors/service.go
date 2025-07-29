// internal/errors/service.go - Error management service for existing components
package errors

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Service provides error management capabilities to existing components
type Service struct {
	retryConfig    RetryConfig
	failurePolicy  FailurePolicy
	messageHandler *MessageHandler
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	BaseDelay     time.Duration `yaml:"base_delay" json:"base_delay"`
	BackoffFactor float64       `yaml:"backoff_factor" json:"backoff_factor"`
	MaxDelay      time.Duration `yaml:"max_delay" json:"max_delay"`
}

// FailurePolicy defines failure handling
type FailurePolicy struct {
	Mode               string  `yaml:"mode" json:"mode"` // "stop", "continue", "partial"
	MaxErrorRate       float64 `yaml:"max_error_rate" json:"max_error_rate"`
	SavePartialResults bool    `yaml:"save_partial_results" json:"save_partial_results"`
}

// MessageHandler converts technical errors to user-friendly messages
type MessageHandler struct {
	showTechnical bool
}

// NewService creates a new error service
func NewService() *Service {
	return &Service{
		retryConfig: RetryConfig{
			MaxRetries:    3,
			BaseDelay:     time.Second * 2,
			BackoffFactor: 2.0,
			MaxDelay:      time.Minute * 5,
		},
		failurePolicy: FailurePolicy{
			Mode:               "partial",
			MaxErrorRate:       0.3,
			SavePartialResults: true,
		},
		messageHandler: &MessageHandler{showTechnical: false},
	}
}

// WithVerbose enables technical error details
func (s *Service) WithVerbose(verbose bool) *Service {
	s.messageHandler.showTechnical = verbose
	return s
}

// ExecuteWithRetry adds retry logic to existing functions
func (s *Service) ExecuteWithRetry(ctx context.Context, operation func() error, operationName string) error {
	var lastErr error
	
	for attempt := 0; attempt <= s.retryConfig.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		// Check if should retry
		if !s.shouldRetry(err, attempt) {
			break
		}
		
		// Calculate delay
		delay := s.calculateDelay(attempt)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			continue
		}
	}
	
	return fmt.Errorf("operation %s failed after %d attempts: %w", operationName, s.retryConfig.MaxRetries+1, lastErr)
}

// shouldRetry determines if error is retryable
func (s *Service) shouldRetry(err error, attempt int) bool {
	if attempt >= s.retryConfig.MaxRetries {
		return false
	}
	
	errStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"timeout", "connection refused", "no such host",
		"500", "502", "503", "504", "429",
	}
	
	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}
	
	return false
}

// calculateDelay computes exponential backoff delay
func (s *Service) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(s.retryConfig.BaseDelay) * pow(s.retryConfig.BackoffFactor, float64(attempt)))
	if delay > s.retryConfig.MaxDelay {
		delay = s.retryConfig.MaxDelay
	}
	return delay
}

// GetUserFriendlyError converts technical errors to user-friendly messages
func (s *Service) GetUserFriendlyError(err error) (title, message string, suggestions []string) {
	if err == nil {
		return "", "", nil
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Network errors
	if strings.Contains(errStr, "timeout") {
		return "Connection Timeout", 
			"The request timed out while trying to connect to the website.",
			[]string{
				"Check your internet connection",
				"Increase timeout value in configuration",
				"The website might be slow or experiencing issues",
			}
	}
	
	if strings.Contains(errStr, "no such host") {
		return "Domain Not Found",
			"Could not find the website domain.",
			[]string{
				"Check if the URL is spelled correctly",
				"Verify the domain exists by opening it in a browser",
				"Check your DNS settings",
			}
	}
	
	if strings.Contains(errStr, "connection refused") {
		return "Connection Refused",
			"The website server refused the connection.",
			[]string{
				"Check if the website is accessible in a browser",
				"The server might be temporarily down",
				"Try using a proxy server",
			}
	}
	
	// Parsing errors
	if strings.Contains(errStr, "selector") {
		return "Element Not Found",
			"Could not find the specified element on the webpage.",
			[]string{
				"Check if the CSS selector is correct",
				"Verify the element exists on the page",
				"The website structure might have changed",
			}
	}
	
	// Configuration errors
	if strings.Contains(errStr, "yaml") {
		return "Configuration Error",
			"The configuration file has invalid YAML syntax.",
			[]string{
				"Check YAML indentation (use spaces, not tabs)",
				"Ensure proper quoting of string values",
				"Use a YAML validator online to check syntax",
			}
	}
	
	// Rate limiting
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		return "Rate Limit Exceeded",
			"You're making requests too quickly.",
			[]string{
				"Reduce the scraping speed/frequency",
				"Add longer delays between requests",
				"Use a different IP address or proxy",
			}
	}
	
	// Default
	return "Unexpected Error",
		"An unexpected error occurred during the operation.",
		[]string{
			"Try running the command again",
			"Check your configuration file",
			"Verify your internet connection",
		}
}

// GetExitCode returns appropriate exit code for error
func (s *Service) GetExitCode(err error) int {
	if err == nil {
		return 0
	}
	
	errStr := strings.ToLower(err.Error())
	
	switch {
	case strings.Contains(errStr, "config") || strings.Contains(errStr, "yaml"):
		return 2 // Configuration error
	case strings.Contains(errStr, "network") || strings.Contains(errStr, "timeout") || 
		 strings.Contains(errStr, "connection") || strings.Contains(errStr, "host"):
		return 3 // Network error
	case strings.Contains(errStr, "parse") || strings.Contains(errStr, "selector"):
		return 4 // Parsing error
	case strings.Contains(errStr, "output") || strings.Contains(errStr, "write"):
		return 5 // Output error
	case strings.Contains(errStr, "validation"):
		return 6 // Validation error
	case strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429"):
		return 7 // Rate limit error
	case strings.Contains(errStr, "auth") || strings.Contains(errStr, "401") || strings.Contains(errStr, "403"):
		return 8 // Authentication error
	default:
		return 1 // General error
	}
}

// FormatErrorForCLI formats error for command-line display
func (s *Service) FormatErrorForCLI(err error) string {
	title, message, suggestions := s.GetUserFriendlyError(err)
	
	output := fmt.Sprintf("❌ %s\n%s\n", title, message)
	
	if s.messageHandler.showTechnical {
		output += fmt.Sprintf("\nTechnical details: %s\n", err.Error())
	}
	
	if len(suggestions) > 0 {
		output += "\n💡 Suggestions:\n"
		for _, suggestion := range suggestions {
			output += fmt.Sprintf("  • %s\n", suggestion)
		}
	}
	
	return output
}

// Helper function for power calculation
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}
