// internal/errors/service.go - Comprehensive error recovery service
package errors

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Service provides comprehensive error recovery capabilities
type Service struct {
	retryConfig      RetryConfig
	failurePolicy    FailurePolicy
	messageHandler   *MessageHandler
	circuitBreakers  map[string]*CircuitBreaker
	fallbackRegistry *FallbackRegistry
	mu               sync.RWMutex
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

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker implements circuit breaker pattern for error recovery
type CircuitBreaker struct {
	name              string
	maxFailures       int
	resetTimeout      time.Duration
	state             CircuitBreakerState
	failures          int
	lastFailureTime   time.Time
	nextAttemptTime   time.Time
	mu                sync.RWMutex
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	MaxFailures  int           `yaml:"max_failures" json:"max_failures"`
	ResetTimeout time.Duration `yaml:"reset_timeout" json:"reset_timeout"`
}

// FallbackStrategy defines different fallback approaches
type FallbackStrategy int

const (
	FallbackNone FallbackStrategy = iota
	FallbackCached
	FallbackDefault
	FallbackAlternative
	FallbackDegrade
)

// FallbackConfig configures fallback behavior
type FallbackConfig struct {
	Strategy     FallbackStrategy          `yaml:"strategy" json:"strategy"`
	CacheTimeout time.Duration             `yaml:"cache_timeout" json:"cache_timeout"`
	DefaultValue interface{}               `yaml:"default_value" json:"default_value"`
	Alternative  string                    `yaml:"alternative" json:"alternative"`
	Degraded     map[string]interface{}    `yaml:"degraded" json:"degraded"`
}

// FallbackRegistry manages fallback strategies for different operations
type FallbackRegistry struct {
	strategies map[string]FallbackConfig
	cache      map[string]CachedResult
	mu         sync.RWMutex
}

// CachedResult stores cached fallback data
type CachedResult struct {
	Data      interface{}
	Timestamp time.Time
}

// RecoveryResult contains the result of error recovery attempt
type RecoveryResult struct {
	Success       bool
	UsedFallback  bool
	FallbackType  string
	AttemptCount  int
	RecoveryTime  time.Duration
	OriginalError error
	Result        interface{}
}

// NewService creates a new comprehensive error recovery service
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
		messageHandler:   &MessageHandler{showTechnical: false},
		circuitBreakers:  make(map[string]*CircuitBreaker),
		fallbackRegistry: NewFallbackRegistry(),
	}
}

// NewFallbackRegistry creates a new fallback registry
func NewFallbackRegistry() *FallbackRegistry {
	return &FallbackRegistry{
		strategies: make(map[string]FallbackConfig),
		cache:      make(map[string]CachedResult),
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

// ExecuteWithRecovery executes an operation with comprehensive error recovery
func (s *Service) ExecuteWithRecovery(ctx context.Context, operationName string, operation func() (interface{}, error)) *RecoveryResult {
	startTime := time.Now()
	result := &RecoveryResult{
		Success:      false,
		UsedFallback: false,
		AttemptCount: 0,
	}

	// Check circuit breaker first
	circuitBreaker := s.getOrCreateCircuitBreaker(operationName)
	if !circuitBreaker.CanExecute() {
		result.OriginalError = fmt.Errorf("circuit breaker is open for operation: %s", operationName)
		result.RecoveryTime = time.Since(startTime)
		
		// Try fallback
		if fallbackResult, err := s.executeFallback(operationName); err == nil {
			result.Success = true
			result.UsedFallback = true
			result.FallbackType = "circuit_breaker_fallback"
			result.Result = fallbackResult
		}
		return result
	}

	// Execute with retry logic
	var lastErr error
	for attempt := 0; attempt <= s.retryConfig.MaxRetries; attempt++ {
		result.AttemptCount++
		
		data, err := operation()
		if err == nil {
			circuitBreaker.RecordSuccess()
			result.Success = true
			result.Result = data
			result.RecoveryTime = time.Since(startTime)
			
			// Cache successful result for future fallback
			s.cacheResult(operationName, data)
			return result
		}

		lastErr = err
		circuitBreaker.RecordFailure()

		// Check if should retry
		if !s.shouldRetry(err, attempt) {
			break
		}

		// Calculate delay
		delay := s.calculateDelay(attempt)

		select {
		case <-ctx.Done():
			result.OriginalError = ctx.Err()
			result.RecoveryTime = time.Since(startTime)
			return result
		case <-time.After(delay):
			continue
		}
	}

	// All retries failed, try fallback
	result.OriginalError = lastErr
	if fallbackResult, err := s.executeFallback(operationName); err == nil {
		result.Success = true
		result.UsedFallback = true
		result.FallbackType = "retry_exhausted_fallback"
		result.Result = fallbackResult
	}

	result.RecoveryTime = time.Since(startTime)
	return result
}

// getOrCreateCircuitBreaker gets or creates a circuit breaker for an operation
func (s *Service) getOrCreateCircuitBreaker(operationName string) *CircuitBreaker {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cb, exists := s.circuitBreakers[operationName]; exists {
		return cb
	}

	// Create new circuit breaker with default config
	cb := &CircuitBreaker{
		name:         operationName,
		maxFailures:  5,                   // Default: open after 5 failures
		resetTimeout: 60 * time.Second,    // Default: try again after 60 seconds
		state:        CircuitClosed,
	}

	s.circuitBreakers[operationName] = cb
	return cb
}

// ConfigureCircuitBreaker configures circuit breaker for specific operation
func (s *Service) ConfigureCircuitBreaker(operationName string, config CircuitBreakerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cb := &CircuitBreaker{
		name:         operationName,
		maxFailures:  config.MaxFailures,
		resetTimeout: config.ResetTimeout,
		state:        CircuitClosed,
	}

	s.circuitBreakers[operationName] = cb
}

// ConfigureFallback configures fallback strategy for specific operation
func (s *Service) ConfigureFallback(operationName string, config FallbackConfig) {
	s.fallbackRegistry.mu.Lock()
	defer s.fallbackRegistry.mu.Unlock()
	s.fallbackRegistry.strategies[operationName] = config
}

// executeFallback attempts to execute fallback strategy
func (s *Service) executeFallback(operationName string) (interface{}, error) {
	s.fallbackRegistry.mu.RLock()
	config, exists := s.fallbackRegistry.strategies[operationName]
	s.fallbackRegistry.mu.RUnlock()

	if !exists {
		config = FallbackConfig{Strategy: FallbackNone}
	}

	switch config.Strategy {
	case FallbackCached:
		return s.getCachedResult(operationName, config.CacheTimeout)
	case FallbackDefault:
		if config.DefaultValue != nil {
			return config.DefaultValue, nil
		}
		return nil, fmt.Errorf("no default value configured for operation: %s", operationName)
	case FallbackAlternative:
		if config.Alternative != "" {
			// Execute alternative operation - this is a placeholder that should be
			// replaced with actual alternative logic in production implementations
			return s.executeAlternativeOperation(operationName, config.Alternative)
		}
		return nil, fmt.Errorf("no alternative configured for operation: %s", operationName)
	case FallbackDegrade:
		if config.Degraded != nil {
			return config.Degraded, nil
		}
		return map[string]interface{}{"degraded": true, "operation": operationName}, nil
	default:
		return nil, fmt.Errorf("no fallback strategy configured for operation: %s", operationName)
	}
}

// cacheResult caches successful result for fallback
func (s *Service) cacheResult(operationName string, result interface{}) {
	s.fallbackRegistry.mu.Lock()
	defer s.fallbackRegistry.mu.Unlock()
	
	s.fallbackRegistry.cache[operationName] = CachedResult{
		Data:      result,
		Timestamp: time.Now(),
	}
}

// getCachedResult retrieves cached result if still valid
func (s *Service) getCachedResult(operationName string, maxAge time.Duration) (interface{}, error) {
	s.fallbackRegistry.mu.RLock()
	defer s.fallbackRegistry.mu.RUnlock()

	cached, exists := s.fallbackRegistry.cache[operationName]
	if !exists {
		return nil, fmt.Errorf("no cached result for operation: %s", operationName)
	}

	if maxAge > 0 && time.Since(cached.Timestamp) > maxAge {
		return nil, fmt.Errorf("cached result expired for operation: %s", operationName)
	}

	return cached.Data, nil
}

// executeAlternativeOperation executes an alternative operation strategy
// This is a framework method that should be extended for specific use cases
func (s *Service) executeAlternativeOperation(operationName, alternative string) (interface{}, error) {
	// This method provides a framework for alternative operation execution
	// In production implementations, this should be extended to:
	// 1. Route to alternative endpoints/services
	// 2. Use different scraping methods (e.g., mobile version, API instead of HTML)
	// 3. Switch to backup data sources
	// 4. Apply different extraction strategies
	
	switch alternative {
	case "mobile_version":
		return map[string]interface{}{
			"source": "mobile_fallback",
			"message": "Using mobile version as fallback",
			"operation": operationName,
		}, nil
	case "api_fallback":
		return map[string]interface{}{
			"source": "api_fallback", 
			"message": "Using API as fallback to HTML scraping",
			"operation": operationName,
		}, nil
	case "cached_alternative":
		// Try to get alternative cached data
		return s.getCachedResult(alternative+"_"+operationName, time.Hour)
	default:
		return map[string]interface{}{
			"source": "generic_alternative",
			"alternative": alternative,
			"operation": operationName,
			"message": "Alternative strategy executed",
		}, nil
	}
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
		"temporary", "temporary error", "service unavailable",
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

// CircuitBreaker methods

// CanExecute checks if circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if now.After(cb.nextAttemptTime) {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records successful execution
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = CircuitClosed
}

// RecordFailure records failed execution
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = CircuitOpen
		cb.nextAttemptTime = time.Now().Add(cb.resetTimeout)
	}
}

// GetState returns current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"name":              cb.name,
		"state":             cb.state,
		"failures":          cb.failures,
		"max_failures":      cb.maxFailures,
		"last_failure_time": cb.lastFailureTime,
		"next_attempt_time": cb.nextAttemptTime,
		"reset_timeout":     cb.resetTimeout,
	}
}

// Additional Service methods for comprehensive error recovery

// GetCircuitBreakerStats returns statistics for all circuit breakers
func (s *Service) GetCircuitBreakerStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, cb := range s.circuitBreakers {
		stats[name] = cb.GetStats()
	}
	return stats
}

// ResetCircuitBreaker manually resets a circuit breaker
func (s *Service) ResetCircuitBreaker(operationName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cb, exists := s.circuitBreakers[operationName]
	if !exists {
		return fmt.Errorf("circuit breaker not found for operation: %s", operationName)
	}

	cb.mu.Lock()
	cb.failures = 0
	cb.state = CircuitClosed
	cb.mu.Unlock()

	return nil
}

// ClearCache clears all cached fallback results
func (s *Service) ClearCache() {
	s.fallbackRegistry.mu.Lock()
	defer s.fallbackRegistry.mu.Unlock()
	s.fallbackRegistry.cache = make(map[string]CachedResult)
}

// GetCacheStats returns cache statistics
func (s *Service) GetCacheStats() map[string]interface{} {
	s.fallbackRegistry.mu.RLock()
	defer s.fallbackRegistry.mu.RUnlock()

	stats := map[string]interface{}{
		"total_entries": len(s.fallbackRegistry.cache),
		"entries":       make(map[string]interface{}),
	}

	entries := stats["entries"].(map[string]interface{})
	for key, cached := range s.fallbackRegistry.cache {
		entries[key] = map[string]interface{}{
			"timestamp": cached.Timestamp,
			"age":       time.Since(cached.Timestamp),
		}
	}

	return stats
}
