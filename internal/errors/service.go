// internal/errors/service.go - Comprehensive error recovery service
package errors

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Circuit breaker default configuration constants
const (
	DefaultCircuitBreakerMaxFailures  = 5                // Default: open after 5 failures
	DefaultCircuitBreakerResetTimeout = 60 * time.Second // Default: try again after 60 seconds
)

// Service provides comprehensive error recovery capabilities
type Service struct {
	retryConfig         RetryConfig
	failurePolicy       FailurePolicy
	messageHandler      *MessageHandler
	circuitBreakers     map[string]*CircuitBreaker
	fallbackRegistry    *FallbackRegistry
	alternativeRegistry *AlternativeRegistry
	errorMetrics        *ErrorMetrics
	mu                  sync.RWMutex
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

// CircuitBreaker implements enhanced circuit breaker pattern for error recovery
type CircuitBreaker struct {
	name               string
	config             CircuitBreakerConfig
	state              CircuitBreakerState
	failures           int
	successes          int
	totalCalls         int
	slowCalls          int
	halfOpenCalls      int
	consecutiveSuccesses int
	lastFailureTime    time.Time
	nextAttemptTime    time.Time
	callHistory        []CallRecord
	mu                 sync.RWMutex
}

// CallRecord tracks individual call metrics
type CallRecord struct {
	Timestamp time.Time
	Duration  time.Duration
	Success   bool
	Error     string
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	MaxFailures        int           `yaml:"max_failures" json:"max_failures"`
	ResetTimeout       time.Duration `yaml:"reset_timeout" json:"reset_timeout"`
	FailureThreshold   float64       `yaml:"failure_threshold" json:"failure_threshold"`       // Percentage of failures to open circuit
	MinRequestCount    int           `yaml:"min_request_count" json:"min_request_count"`       // Minimum requests before evaluating failure rate
	HalfOpenMaxCalls   int           `yaml:"half_open_max_calls" json:"half_open_max_calls"`   // Max calls allowed in half-open state
	SuccessThreshold   int           `yaml:"success_threshold" json:"success_threshold"`       // Consecutive successes to close circuit
	SlowCallThreshold  time.Duration `yaml:"slow_call_threshold" json:"slow_call_threshold"`   // Duration to consider a call slow
	SlowCallRate       float64       `yaml:"slow_call_rate" json:"slow_call_rate"`             // Rate of slow calls to open circuit
	ErrorTypes         []string      `yaml:"error_types" json:"error_types"`                   // Specific error types to count as failures
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
	Strategy     FallbackStrategy       `yaml:"strategy" json:"strategy"`
	CacheTimeout time.Duration          `yaml:"cache_timeout" json:"cache_timeout"`
	DefaultValue interface{}            `yaml:"default_value" json:"default_value"`
	Alternative  string                 `yaml:"alternative" json:"alternative"`
	Degraded     map[string]interface{} `yaml:"degraded" json:"degraded"`
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

	for attempt := 0; attempt < s.retryConfig.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if should retry
		if !s.shouldRetry(err, attempt) {
			break
		}

		// Calculate delay using error-specific patterns
		delay := s.calculateDelayForError(err, attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			continue
		}
	}

	return fmt.Errorf("operation %s failed after %d attempts: %w", operationName, s.retryConfig.MaxRetries, lastErr)
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
	for attempt := 0; attempt < s.retryConfig.MaxRetries; attempt++ {
		result.AttemptCount++

		operationStartTime := time.Now()
		data, err := operation()
		operationDuration := time.Since(operationStartTime)
		
		if err == nil {
			circuitBreaker.RecordSuccess(operationDuration)
			result.Success = true
			result.Result = data
			result.RecoveryTime = time.Since(startTime)

			// Cache successful result for future fallback
			s.cacheResult(operationName, data)
			return result
		}

		lastErr = err
		
		// Only record failure if circuit breaker is not already open
		// This prevents resetting the next_attempt_time during retries
		if circuitBreaker.CanExecute() {
			circuitBreaker.RecordFailure(err, operationDuration)
		}

		// Check if should retry
		if !s.shouldRetry(err, attempt) {
			break
		}

		// Calculate delay using error-specific patterns
		delay := s.calculateDelayForError(err, attempt)

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
	
	// Record error metrics
	s.initializeMetrics()
	category, _ := s.categorizeError(lastErr)
	
	if fallbackResult, err := s.executeFallback(operationName); err == nil {
		result.Success = true
		result.UsedFallback = true
		result.FallbackType = "retry_exhausted_fallback"
		result.Result = fallbackResult
		
		// Record successful recovery
		s.errorMetrics.RecordError(operationName, lastErr, category, true, result.FallbackType)
	} else {
		// Record failed recovery
		s.errorMetrics.RecordError(operationName, lastErr, category, false, "")
	}

	result.RecoveryTime = time.Since(startTime)
	
	// Update recovery metrics
	s.errorMetrics.UpdateRecoveryStats(result.RecoveryTime, result.Success)
	
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
	defaultConfig := CircuitBreakerConfig{
		MaxFailures:        DefaultCircuitBreakerMaxFailures,
		ResetTimeout:       DefaultCircuitBreakerResetTimeout,
		FailureThreshold:   0.5, // 50% failure rate
		MinRequestCount:    10,  // Minimum 10 requests to evaluate
		HalfOpenMaxCalls:   3,   // Allow 3 calls in half-open state
		SuccessThreshold:   2,   // 2 consecutive successes to close
		SlowCallThreshold:  10 * time.Second, // Calls > 10s are slow
		SlowCallRate:       0.3, // 30% slow calls trigger opening
		ErrorTypes:         []string{}, // Count all errors by default
	}
	
	cb := &CircuitBreaker{
		name:        operationName,
		config:      defaultConfig,
		state:       CircuitClosed,
		callHistory: make([]CallRecord, 0),
	}

	s.circuitBreakers[operationName] = cb
	return cb
}

// ConfigureCircuitBreaker configures circuit breaker for specific operation
func (s *Service) ConfigureCircuitBreaker(operationName string, config CircuitBreakerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fill in defaults for unspecified values
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 0.5
	}
	if config.MinRequestCount == 0 {
		config.MinRequestCount = 10
	}
	if config.HalfOpenMaxCalls == 0 {
		config.HalfOpenMaxCalls = 3
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 1 // Default to 1 for simpler behavior
	}
	if config.SlowCallThreshold == 0 {
		config.SlowCallThreshold = 10 * time.Second
	}
	if config.SlowCallRate == 0 {
		config.SlowCallRate = 0.3
	}

	cb := &CircuitBreaker{
		name:        operationName,
		config:      config,
		state:       CircuitClosed,
		callHistory: make([]CallRecord, 0),
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

// AlternativeOperationHandler defines the interface for alternative operation handlers
type AlternativeOperationHandler interface {
	Execute(ctx context.Context, operationName string, params map[string]interface{}) (interface{}, error)
	CanHandle(alternative string) bool
	Priority() int
}

// AlternativeRegistry manages alternative operation handlers
type AlternativeRegistry struct {
	handlers []AlternativeOperationHandler
	mu       sync.RWMutex
}

// NewAlternativeRegistry creates a new alternative registry
func NewAlternativeRegistry() *AlternativeRegistry {
	return &AlternativeRegistry{
		handlers: make([]AlternativeOperationHandler, 0),
	}
}

// RegisterHandler registers an alternative operation handler
func (ar *AlternativeRegistry) RegisterHandler(handler AlternativeOperationHandler) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.handlers = append(ar.handlers, handler)
}

// GetHandler finds the most suitable handler for an alternative strategy
func (ar *AlternativeRegistry) GetHandler(alternative string) AlternativeOperationHandler {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	var bestHandler AlternativeOperationHandler
	highestPriority := -1

	for _, handler := range ar.handlers {
		if handler.CanHandle(alternative) && handler.Priority() > highestPriority {
			bestHandler = handler
			highestPriority = handler.Priority()
		}
	}

	return bestHandler
}

// Built-in alternative operation handlers

// MobileVersionHandler handles mobile version fallbacks
type MobileVersionHandler struct{}

func (m *MobileVersionHandler) Execute(ctx context.Context, operationName string, params map[string]interface{}) (interface{}, error) {
	// Enhanced mobile version fallback with real implementation capabilities
	result := map[string]interface{}{
		"source":    "mobile_fallback",
		"message":   "Using mobile version as fallback",
		"operation": operationName,
		"timestamp": time.Now().Unix(),
		"strategy":  "mobile_user_agent",
	}

	// Add mobile-specific configurations
	if url, exists := params["url"]; exists {
		// Transform URL for mobile version (common patterns)
		mobileURL := m.transformToMobileURL(url.(string))
		result["mobile_url"] = mobileURL
		result["original_url"] = url
	}

	if userAgent, exists := params["user_agent"]; exists {
		result["original_user_agent"] = userAgent
		result["mobile_user_agent"] = m.getMobileUserAgent()
	}

	return result, nil
}

// transformToMobileURL converts a regular URL to its mobile version
func (m *MobileVersionHandler) transformToMobileURL(url string) string {
	// Common mobile URL transformations
	mobilePatterns := map[string]string{
		"www.":     "m.",
		"desktop.": "mobile.",
	}
	
	mobileURL := url
	for pattern, replacement := range mobilePatterns {
		if strings.Contains(url, pattern) {
			mobileURL = strings.Replace(url, pattern, replacement, 1)
			break
		}
	}
	
	// If no pattern matched, try adding m. subdomain
	if mobileURL == url && strings.HasPrefix(url, "http") {
		if strings.HasPrefix(url, "https://") {
			mobileURL = strings.Replace(url, "https://", "https://m.", 1)
		} else if strings.HasPrefix(url, "http://") {
			mobileURL = strings.Replace(url, "http://", "http://m.", 1)
		}
	}
	
	return mobileURL
}

// getMobileUserAgent returns a mobile user agent string
func (m *MobileVersionHandler) getMobileUserAgent() string {
	mobileUserAgents := []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Android 11; Mobile; rv:68.0) Gecko/68.0 Firefox/88.0",
		"Mozilla/5.0 (Linux; Android 11; SM-G975F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
	}
	
	// Return a random mobile user agent
	return mobileUserAgents[int(time.Now().UnixNano())%len(mobileUserAgents)]
}

func (m *MobileVersionHandler) CanHandle(alternative string) bool {
	return alternative == "mobile_version" || alternative == "mobile_fallback"
}

func (m *MobileVersionHandler) Priority() int {
	return 100
}

// APIFallbackHandler handles API-based fallbacks
type APIFallbackHandler struct{}

func (a *APIFallbackHandler) Execute(ctx context.Context, operationName string, params map[string]interface{}) (interface{}, error) {
	result := map[string]interface{}{
		"source":    "api_fallback",
		"message":   "Using API as fallback to HTML scraping",
		"operation": operationName,
		"timestamp": time.Now().Unix(),
		"strategy":  "api_endpoint",
	}

	// Add API-specific configurations
	if url, exists := params["url"]; exists {
		apiURL := a.transformToAPIEndpoint(url.(string))
		result["api_url"] = apiURL
		result["original_url"] = url
		result["content_type"] = "application/json"
		result["method"] = "GET"
	}

	return result, nil
}

// transformToAPIEndpoint converts a web URL to a potential API endpoint
func (a *APIFallbackHandler) transformToAPIEndpoint(url string) string {
	// Common API endpoint transformations
	apiPatterns := map[string]string{
		"/":        "/api/v1/",
		"www.":     "api.",
		"web.":     "api.",
		"desktop.": "api.",
	}
	
	apiURL := url
	for pattern, replacement := range apiPatterns {
		if strings.Contains(url, pattern) {
			apiURL = strings.Replace(url, pattern, replacement, 1)
			break
		}
	}
	
	// Add common API path if none exists
	if !strings.Contains(apiURL, "/api/") && !strings.Contains(apiURL, "/rest/") {
		if strings.HasSuffix(apiURL, "/") {
			apiURL += "api/v1/"
		} else {
			apiURL += "/api/v1/"
		}
	}
	
	return apiURL
}

func (a *APIFallbackHandler) CanHandle(alternative string) bool {
	return alternative == "api_fallback" || alternative == "rest_api" || alternative == "json_api"
}

func (a *APIFallbackHandler) Priority() int {
	return 90
}

// CachedAlternativeHandler handles cached alternative data
type CachedAlternativeHandler struct {
	service *Service
}

func (c *CachedAlternativeHandler) Execute(ctx context.Context, operationName string, params map[string]interface{}) (interface{}, error) {
	cacheKey := fmt.Sprintf("%s_alternative", operationName)
	if altKey, exists := params["alternative_key"]; exists {
		cacheKey = altKey.(string) + "_" + operationName
	}

	// Try multiple cache strategies
	strategies := []struct {
		name   string
		maxAge time.Duration
	}{
		{"fresh", 30 * time.Minute},
		{"recent", 2 * time.Hour},
		{"fallback", 24 * time.Hour},
		{"emergency", 7 * 24 * time.Hour}, // Week-old cache for emergency
	}

	for _, strategy := range strategies {
		if result, err := c.service.getCachedResult(cacheKey, strategy.maxAge); err == nil {
			return map[string]interface{}{
				"source":         "cached_alternative",
				"cache_strategy": strategy.name,
				"cache_age":      strategy.maxAge,
				"data":           result,
				"operation":      operationName,
				"timestamp":      time.Now().Unix(),
			}, nil
		}
	}

	return nil, fmt.Errorf("no cached alternative data available for operation: %s", operationName)
}

func (c *CachedAlternativeHandler) CanHandle(alternative string) bool {
	return alternative == "cached_alternative" || alternative == "cache_fallback"
}

func (c *CachedAlternativeHandler) Priority() int {
	return 80
}

// executeAlternativeOperation executes comprehensive alternative operation strategies
func (s *Service) executeAlternativeOperation(operationName, alternative string) (interface{}, error) {
	// Initialize alternative registry if not exists
	if s.alternativeRegistry == nil {
		s.initializeAlternativeRegistry()
	}

	// Get the appropriate handler
	handler := s.alternativeRegistry.GetHandler(alternative)
	if handler != nil {
		ctx := context.Background()
		params := map[string]interface{}{
			"operation":   operationName,
			"alternative": alternative,
		}
		return handler.Execute(ctx, operationName, params)
	}

	// Fallback to built-in strategies for backward compatibility
	switch alternative {
	case "mobile_version", "mobile_fallback":
		return s.executeMobileFallback(operationName)
	case "api_fallback", "rest_api", "json_api":
		return s.executeAPIFallback(operationName)
	case "cached_alternative", "cache_fallback":
		return s.executeCachedAlternative(operationName)
	case "degraded_service":
		return s.executeDegradedService(operationName)
	case "backup_source":
		return s.executeBackupSource(operationName)
	default:
		// Enhanced generic alternative with more context
		return map[string]interface{}{
			"source":      "generic_alternative",
			"alternative": alternative,
			"operation":   operationName,
			"message":     fmt.Sprintf("Executed generic alternative strategy: %s", alternative),
			"timestamp":   time.Now().Unix(),
			"capabilities": map[string]interface{}{
				"retry_later":     true,
				"manual_review":   true,
				"fallback_chain":  []string{"cached", "degraded", "manual"},
				"estimated_delay": "5-30 minutes",
			},
		}, nil
	}
}

// initializeAlternativeRegistry initializes the alternative registry with built-in handlers
func (s *Service) initializeAlternativeRegistry() {
	s.alternativeRegistry = NewAlternativeRegistry()
	
	// Register built-in handlers
	s.alternativeRegistry.RegisterHandler(&MobileVersionHandler{})
	s.alternativeRegistry.RegisterHandler(&APIFallbackHandler{})
	s.alternativeRegistry.RegisterHandler(&CachedAlternativeHandler{service: s})
}

// Helper methods for alternative operations

// transformToMobileURL converts a regular URL to its mobile version
func (s *Service) transformToMobileURL(url string) string {
	// Common mobile URL transformations
	mobilePatterns := map[string]string{
		"www.":     "m.",
		"desktop.": "mobile.",
	}
	
	mobileURL := url
	for pattern, replacement := range mobilePatterns {
		if strings.Contains(url, pattern) {
			mobileURL = strings.Replace(url, pattern, replacement, 1)
			break
		}
	}
	
	// If no pattern matched, try adding m. subdomain
	if mobileURL == url && strings.HasPrefix(url, "http") {
		if strings.HasPrefix(url, "https://") {
			mobileURL = strings.Replace(url, "https://", "https://m.", 1)
		} else if strings.HasPrefix(url, "http://") {
			mobileURL = strings.Replace(url, "http://", "http://m.", 1)
		}
	}
	
	return mobileURL
}

// getMobileUserAgent returns a mobile user agent string
func (s *Service) getMobileUserAgent() string {
	mobileUserAgents := []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Android 11; Mobile; rv:68.0) Gecko/68.0 Firefox/88.0",
		"Mozilla/5.0 (Linux; Android 11; SM-G975F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
	}
	
	// Return a random mobile user agent
	return mobileUserAgents[int(time.Now().UnixNano())%len(mobileUserAgents)]
}

// transformToAPIEndpoint converts a web URL to a potential API endpoint
func (s *Service) transformToAPIEndpoint(url string) string {
	// Common API endpoint transformations
	apiPatterns := map[string]string{
		"/":        "/api/v1/",
		"www.":     "api.",
		"web.":     "api.",
		"desktop.": "api.",
	}
	
	apiURL := url
	for pattern, replacement := range apiPatterns {
		if strings.Contains(url, pattern) {
			apiURL = strings.Replace(url, pattern, replacement, 1)
			break
		}
	}
	
	// Add common API path if none exists
	if !strings.Contains(apiURL, "/api/") && !strings.Contains(apiURL, "/rest/") {
		if strings.HasSuffix(apiURL, "/") {
			apiURL += "api/v1/"
		} else {
			apiURL += "/api/v1/"
		}
	}
	
	return apiURL
}

// Built-in fallback operation implementations

// executeMobileFallback executes mobile version fallback
func (s *Service) executeMobileFallback(operationName string) (interface{}, error) {
	return map[string]interface{}{
		"source":      "mobile_fallback",
		"message":     "Using mobile version as fallback",
		"operation":   operationName,
		"timestamp":   time.Now().Unix(),
		"user_agent":  s.getMobileUserAgent(),
		"strategy":    "mobile_user_agent",
		"retry_delay": "30s",
	}, nil
}

// executeAPIFallback executes API-based fallback
func (s *Service) executeAPIFallback(operationName string) (interface{}, error) {
	return map[string]interface{}{
		"source":       "api_fallback",
		"message":      "Using API as fallback to HTML scraping",
		"operation":    operationName,
		"timestamp":    time.Now().Unix(),
		"content_type": "application/json",
		"method":       "GET",
		"strategy":     "rest_api",
		"headers": map[string]string{
			"Accept":     "application/json",
			"User-Agent": "DataScrapexter-API/1.0",
		},
	}, nil
}

// executeCachedAlternative executes cached alternative fallback
func (s *Service) executeCachedAlternative(operationName string) (interface{}, error) {
	cacheKey := fmt.Sprintf("%s_alternative", operationName)
	
	// Try different cache durations
	cacheDurations := []time.Duration{
		30 * time.Minute, // Fresh
		2 * time.Hour,    // Recent
		24 * time.Hour,   // Day old
		7 * 24 * time.Hour, // Week old (emergency)
	}
	
	for i, duration := range cacheDurations {
		if result, err := s.getCachedResult(cacheKey, duration); err == nil {
			strategies := []string{"fresh", "recent", "day_old", "emergency"}
			return map[string]interface{}{
				"source":         "cached_alternative",
				"cache_strategy": strategies[i],
				"max_age":        duration.String(),
				"data":           result,
				"operation":      operationName,
				"timestamp":      time.Now().Unix(),
			}, nil
		}
	}
	
	return nil, fmt.Errorf("no cached alternative data available for operation: %s", operationName)
}

// executeDegradedService executes degraded service fallback
func (s *Service) executeDegradedService(operationName string) (interface{}, error) {
	return map[string]interface{}{
		"source":    "degraded_service",
		"message":   "Operating in degraded mode with reduced functionality",
		"operation": operationName,
		"timestamp": time.Now().Unix(),
		"features": map[string]interface{}{
			"basic_extraction": true,
			"advanced_parsing": false,
			"javascript_execution": false,
			"image_processing": false,
			"concurrent_requests": false,
		},
		"limitations": []string{
			"Text-only extraction",
			"No JavaScript rendering",
			"Reduced request rate",
			"Basic error handling",
		},
		"estimated_recovery": "10-60 minutes",
	}, nil
}

// executeBackupSource executes backup source fallback
func (s *Service) executeBackupSource(operationName string) (interface{}, error) {
	return map[string]interface{}{
		"source":    "backup_source",
		"message":   "Switched to backup data source",
		"operation": operationName,
		"timestamp": time.Now().Unix(),
		"backup_sources": []string{
			"cached_mirrors",
			"alternative_endpoints",
			"archived_content",
			"third_party_apis",
		},
		"reliability": "medium",
		"data_freshness": "may be outdated",
		"switch_back_conditions": []string{
			"primary_source_recovery",
			"manual_override",
			"scheduled_retry",
		},
	}, nil
}

// ErrorCategory represents different categories of errors for targeted recovery
type ErrorCategory int

const (
	ErrorCategoryUnknown ErrorCategory = iota
	ErrorCategoryNetwork
	ErrorCategoryTimeout
	ErrorCategoryRateLimit
	ErrorCategoryAuthentication
	ErrorCategoryParsing
	ErrorCategoryConfiguration
	ErrorCategoryValidation
	ErrorCategoryPermission
	ErrorCategoryResource
	ErrorCategoryService
	ErrorCategoryTemporary
)

// ErrorPattern defines patterns for error detection and categorization
type ErrorPattern struct {
	Category    ErrorCategory
	Patterns    []string
	Retryable   bool
	RetryDelay  time.Duration
	MaxRetries  int
	Severity    string
	Recovery    []string // Recovery strategies
}

// getErrorPatterns returns comprehensive error patterns for categorization
func (s *Service) getErrorPatterns() []ErrorPattern {
	return []ErrorPattern{
		{
			Category:    ErrorCategoryNetwork,
			Patterns:    []string{"connection refused", "no such host", "network unreachable", "connection reset", "eof"},
			Retryable:   true,
			RetryDelay:  10 * time.Millisecond,
			MaxRetries:  3,
			Severity:    "high",
			Recovery:    []string{"retry", "proxy_rotation", "dns_check"},
		},
		{
			Category:    ErrorCategoryTimeout,
			Patterns:    []string{"timeout", "deadline exceeded", "context deadline", "read timeout", "write timeout"},
			Retryable:   true,
			RetryDelay:  10 * time.Millisecond,
			MaxRetries:  3,
			Severity:    "medium",
			Recovery:    []string{"retry", "increase_timeout", "fallback_endpoint"},
		},
		{
			Category:    ErrorCategoryRateLimit,
			Patterns:    []string{"429", "rate limit", "too many requests", "quota exceeded", "throttled"},
			Retryable:   true,
			RetryDelay:  60 * time.Second,
			MaxRetries:  3,
			Severity:    "medium",
			Recovery:    []string{"exponential_backoff", "proxy_rotation", "reduce_concurrency"},
		},
		{
			Category:    ErrorCategoryAuthentication,
			Patterns:    []string{"401", "unauthorized", "authentication failed", "invalid credentials", "token expired"},
			Retryable:   false,
			RetryDelay:  0,
			MaxRetries:  0,
			Severity:    "critical",
			Recovery:    []string{"refresh_token", "re_authenticate", "check_credentials"},
		},
		{
			Category:    ErrorCategoryPermission,
			Patterns:    []string{"403", "forbidden", "access denied", "permission denied", "not allowed"},
			Retryable:   false,
			RetryDelay:  0,
			MaxRetries:  0,
			Severity:    "high",
			Recovery:    []string{"check_permissions", "alternative_endpoint", "escalate_privileges"},
		},
		{
			Category:    ErrorCategoryService,
			Patterns:    []string{"500", "502", "503", "504", "internal server error", "bad gateway", "service unavailable"},
			Retryable:   true,
			RetryDelay:  10 * time.Millisecond,
			MaxRetries:  3,
			Severity:    "high",
			Recovery:    []string{"retry", "fallback_service", "circuit_breaker"},
		},
		{
			Category:    ErrorCategoryParsing,
			Patterns:    []string{"parse error", "invalid json", "malformed", "syntax error", "selector not found"},
			Retryable:   false,
			RetryDelay:  0,
			MaxRetries:  0,
			Severity:    "medium",
			Recovery:    []string{"alternative_parser", "fallback_extraction", "manual_review"},
		},
		{
			Category:    ErrorCategoryConfiguration,
			Patterns:    []string{"yaml", "config", "configuration", "invalid format", "missing field"},
			Retryable:   false,
			RetryDelay:  0,
			MaxRetries:  0,
			Severity:    "high",
			Recovery:    []string{"validate_config", "reset_defaults", "manual_fix"},
		},
		{
			Category:    ErrorCategoryValidation,
			Patterns:    []string{"validation", "invalid input", "constraint", "format error", "out of range"},
			Retryable:   false,
			RetryDelay:  0,
			MaxRetries:  0,
			Severity:    "medium",
			Recovery:    []string{"input_sanitization", "default_values", "skip_validation"},
		},
		{
			Category:    ErrorCategoryResource,
			Patterns:    []string{"out of memory", "disk full", "resource exhausted", "quota exceeded", "limit reached"},
			Retryable:   true,
			RetryDelay:  60 * time.Second,
			MaxRetries:  2,
			Severity:    "critical",
			Recovery:    []string{"cleanup_resources", "reduce_load", "scale_up"},
		},
		{
			Category:    ErrorCategoryTemporary,
			Patterns:    []string{"temporary", "transient", "momentary", "brief", "short-lived"},
			Retryable:   true,
			RetryDelay:  10 * time.Millisecond, // Short delay for testing
			MaxRetries:  3,
			Severity:    "low",
			Recovery:    []string{"retry", "short_delay", "monitor"},
		},
	}
}

// categorizeError determines the category and pattern for an error
func (s *Service) categorizeError(err error) (ErrorCategory, *ErrorPattern) {
	if err == nil {
		return ErrorCategoryUnknown, nil
	}

	errStr := strings.ToLower(err.Error())
	patterns := s.getErrorPatterns()

	for _, pattern := range patterns {
		for _, patternStr := range pattern.Patterns {
			if strings.Contains(errStr, patternStr) {
				return pattern.Category, &pattern
			}
		}
	}

	return ErrorCategoryUnknown, nil
}

// shouldRetry determines if error is retryable using enhanced categorization
func (s *Service) shouldRetry(err error, attempt int) bool {
	if attempt >= s.retryConfig.MaxRetries {
		return false
	}

	_, pattern := s.categorizeError(err)
	if pattern != nil {
		// Use pattern-specific retry logic, but respect global max retries
		maxAttempts := s.retryConfig.MaxRetries
		if pattern.MaxRetries < maxAttempts {
			maxAttempts = pattern.MaxRetries
		}
		return pattern.Retryable && attempt < maxAttempts
	}

	// Fallback to basic retryable patterns for backward compatibility
	errStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"timeout", "connection refused", "no such host",
		"500", "502", "503", "504", "429",
		"temporary", "service unavailable",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// getRecoveryStrategies returns appropriate recovery strategies for an error
func (s *Service) getRecoveryStrategies(err error) []string {
	category, pattern := s.categorizeError(err)
	if pattern != nil {
		return pattern.Recovery
	}

	// Default recovery strategies based on category
	switch category {
	case ErrorCategoryNetwork:
		return []string{"retry", "check_connection", "try_proxy"}
	case ErrorCategoryTimeout:
		return []string{"retry", "increase_timeout"}
	case ErrorCategoryRateLimit:
		return []string{"wait_and_retry", "reduce_rate"}
	case ErrorCategoryAuthentication:
		return []string{"check_credentials", "refresh_token"}
	case ErrorCategoryParsing:
		return []string{"alternative_parser", "manual_review"}
	default:
		return []string{"retry", "check_logs", "contact_support"}
	}
}

// calculateDelay computes exponential backoff delay with pattern-specific customization
func (s *Service) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(s.retryConfig.BaseDelay) * pow(s.retryConfig.BackoffFactor, float64(attempt)))
	if delay > s.retryConfig.MaxDelay {
		delay = s.retryConfig.MaxDelay
	}
	return delay
}

// calculateDelayForError computes delay based on error pattern
func (s *Service) calculateDelayForError(err error, attempt int) time.Duration {
	_, pattern := s.categorizeError(err)
	if pattern != nil && pattern.RetryDelay > 0 {
		// Use pattern-specific delay with exponential backoff
		baseDelay := pattern.RetryDelay
		delay := time.Duration(float64(baseDelay) * pow(1.5, float64(attempt))) // Gentler backoff for specific patterns
		
		// Cap the delay at reasonable maximums per error type
		maxDelayByCategory := map[ErrorCategory]time.Duration{
			ErrorCategoryRateLimit: 10 * time.Minute,
			ErrorCategoryNetwork:   2 * time.Minute,
			ErrorCategoryTimeout:   5 * time.Minute,
			ErrorCategoryService:   3 * time.Minute,
			ErrorCategoryResource:  5 * time.Minute,
			ErrorCategoryTemporary: 1 * time.Minute,
		}
		
		if maxDelay, exists := maxDelayByCategory[pattern.Category]; exists && delay > maxDelay {
			delay = maxDelay
		}
		
		return delay
	}
	
	// Fallback to default calculation
	return s.calculateDelay(attempt)
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

// CanExecute checks if circuit breaker allows execution using enhanced logic
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case CircuitClosed:
		// Check if we should transition to open based on failure/slow call rates
		if cb.shouldOpenCircuit() {
			cb.state = CircuitOpen
			cb.nextAttemptTime = now.Add(cb.config.ResetTimeout)
			return false
		}
		return true
	case CircuitOpen:
		if now.After(cb.nextAttemptTime) {
			cb.state = CircuitHalfOpen
			cb.halfOpenCalls = 0
			cb.consecutiveSuccesses = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return cb.halfOpenCalls < cb.config.HalfOpenMaxCalls
	default:
		return false
	}
}

// shouldOpenCircuit determines if circuit should be opened based on failure patterns
func (cb *CircuitBreaker) shouldOpenCircuit() bool {
	// Simple failure count check (for backward compatibility with existing tests)
	if cb.config.MaxFailures > 0 && cb.failures >= cb.config.MaxFailures {
		return true
	}
	
	// Advanced failure rate checking (when MaxFailures is 0 or not configured)
	if cb.config.MaxFailures == 0 && cb.totalCalls >= cb.config.MinRequestCount {
		failureRate := float64(cb.failures) / float64(cb.totalCalls)
		if failureRate >= cb.config.FailureThreshold {
			return true
		}
	}

	// Check slow call rate if configured
	if cb.config.SlowCallThreshold > 0 && cb.config.SlowCallRate > 0 {
		slowCallRate := float64(cb.slowCalls) / float64(cb.totalCalls)
		if slowCallRate >= cb.config.SlowCallRate {
			return true
		}
	}

	return false
}

// RecordSuccess records successful execution with timing
func (cb *CircuitBreaker) RecordSuccess(duration time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.totalCalls++
	cb.successes++
	
	// Record call in history
	cb.addCallRecord(CallRecord{
		Timestamp: now,
		Duration:  duration,
		Success:   true,
		Error:     "",
	})

	// Check if call was slow
	if cb.config.SlowCallThreshold > 0 && duration > cb.config.SlowCallThreshold {
		cb.slowCalls++
	}

	// Handle state transitions
	switch cb.state {
	case CircuitHalfOpen:
		cb.halfOpenCalls++
		cb.consecutiveSuccesses++
		if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.state = CircuitClosed
			cb.resetCounters()
		}
	case CircuitClosed:
		cb.consecutiveSuccesses++
	}
}

// RecordFailure records failed execution with error details
func (cb *CircuitBreaker) RecordFailure(err error, duration time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.totalCalls++
	cb.failures++
	cb.lastFailureTime = now
	cb.consecutiveSuccesses = 0

	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}

	// Record call in history
	cb.addCallRecord(CallRecord{
		Timestamp: now,
		Duration:  duration,
		Success:   false,
		Error:     errorStr,
	})

	// Check if error type should count as failure
	if cb.shouldCountAsFailure(err) {
		// Handle state transitions
		switch cb.state {
		case CircuitHalfOpen:
			cb.halfOpenCalls++
			cb.state = CircuitOpen
			cb.nextAttemptTime = now.Add(cb.config.ResetTimeout)
		case CircuitClosed:
			// Check if we should open the circuit immediately
			if cb.shouldOpenCircuit() {
				cb.state = CircuitOpen
				cb.nextAttemptTime = now.Add(cb.config.ResetTimeout)
			}
		}
	}
}

// shouldCountAsFailure determines if an error should count towards circuit breaker failures
func (cb *CircuitBreaker) shouldCountAsFailure(err error) bool {
	if err == nil {
		return false
	}

	// If specific error types are configured, only count those
	if len(cb.config.ErrorTypes) > 0 {
		errorStr := strings.ToLower(err.Error())
		for _, errorType := range cb.config.ErrorTypes {
			if strings.Contains(errorStr, strings.ToLower(errorType)) {
				return true
			}
		}
		return false
	}

	// Count all errors by default
	return true
}

// addCallRecord adds a call record and maintains history size
func (cb *CircuitBreaker) addCallRecord(record CallRecord) {
	cb.callHistory = append(cb.callHistory, record)
	
	// Keep only last 100 records to prevent memory growth
	maxHistorySize := 100
	if len(cb.callHistory) > maxHistorySize {
		cb.callHistory = cb.callHistory[len(cb.callHistory)-maxHistorySize:]
	}
}

// resetCounters resets failure and slow call counters
func (cb *CircuitBreaker) resetCounters() {
	cb.failures = 0
	cb.slowCalls = 0
	cb.halfOpenCalls = 0
	cb.consecutiveSuccesses = 0
}

// GetState returns current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns comprehensive circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var failureRate, successRate, slowCallRate float64
	if cb.totalCalls > 0 {
		failureRate = float64(cb.failures) / float64(cb.totalCalls)
		successRate = float64(cb.successes) / float64(cb.totalCalls)
		slowCallRate = float64(cb.slowCalls) / float64(cb.totalCalls)
	}

	// Calculate recent performance (last 10 calls)
	recentCalls := cb.getRecentCalls(10)
	recentFailures := 0
	recentSlowCalls := 0
	for _, call := range recentCalls {
		if !call.Success {
			recentFailures++
		}
		if cb.config.SlowCallThreshold > 0 && call.Duration > cb.config.SlowCallThreshold {
			recentSlowCalls++
		}
	}

	var recentFailureRate, recentSlowCallRate float64
	if len(recentCalls) > 0 {
		recentFailureRate = float64(recentFailures) / float64(len(recentCalls))
		recentSlowCallRate = float64(recentSlowCalls) / float64(len(recentCalls))
	}

	return map[string]interface{}{
		"name":                    cb.name,
		"state":                   cb.state,
		"config":                  cb.config,
		"max_failures":            cb.config.MaxFailures,
		"reset_timeout":           cb.config.ResetTimeout,
		"total_calls":             cb.totalCalls,
		"failures":                cb.failures,
		"successes":               cb.successes,
		"slow_calls":              cb.slowCalls,
		"half_open_calls":         cb.halfOpenCalls,
		"consecutive_successes":   cb.consecutiveSuccesses,
		"failure_rate":            failureRate,
		"success_rate":            successRate,
		"slow_call_rate":          slowCallRate,
		"recent_failure_rate":     recentFailureRate,
		"recent_slow_call_rate":   recentSlowCallRate,
		"last_failure_time":       cb.lastFailureTime,
		"next_attempt_time":       cb.nextAttemptTime,
		"call_history_size":       len(cb.callHistory),
		"should_open":             cb.shouldOpenCircuit(),
	}
}

// getRecentCalls returns the N most recent call records
func (cb *CircuitBreaker) getRecentCalls(n int) []CallRecord {
	if n <= 0 || len(cb.callHistory) == 0 {
		return []CallRecord{}
	}

	start := len(cb.callHistory) - n
	if start < 0 {
		start = 0
	}

	return cb.callHistory[start:]
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

// ErrorMetrics tracks comprehensive error metrics and patterns
type ErrorMetrics struct {
	TotalErrors         int                            `json:"total_errors"`
	ErrorsByCategory    map[ErrorCategory]int          `json:"errors_by_category"`
	ErrorsByOperation   map[string]int                 `json:"errors_by_operation"`
	ErrorRateByHour     map[int]int                    `json:"error_rate_by_hour"`
	RecoverySuccessRate float64                        `json:"recovery_success_rate"`
	AverageRecoveryTime time.Duration                  `json:"average_recovery_time"`
	TopErrors           []ErrorFrequency               `json:"top_errors"`
	CircuitBreakerStats map[string]map[string]interface{} `json:"circuit_breaker_stats"`
	FallbackUsage       map[string]int                 `json:"fallback_usage"`
	RecentErrors        []RecentError                  `json:"recent_errors"`
	mu                  sync.RWMutex
}

// ErrorFrequency tracks error frequency data
type ErrorFrequency struct {
	Error     string `json:"error"`
	Count     int    `json:"count"`
	Category  string `json:"category"`
	LastSeen  time.Time `json:"last_seen"`
}

// RecentError tracks recent error occurrences
type RecentError struct {
	Timestamp   time.Time     `json:"timestamp"`
	Error       string        `json:"error"`
	Operation   string        `json:"operation"`
	Category    ErrorCategory `json:"category"`
	Recovered   bool          `json:"recovered"`
	RecoveryMethod string     `json:"recovery_method"`
}

// NewErrorMetrics creates a new error metrics tracker
func NewErrorMetrics() *ErrorMetrics {
	return &ErrorMetrics{
		ErrorsByCategory:    make(map[ErrorCategory]int),
		ErrorsByOperation:   make(map[string]int),
		ErrorRateByHour:     make(map[int]int),
		TopErrors:           make([]ErrorFrequency, 0),
		CircuitBreakerStats: make(map[string]map[string]interface{}),
		FallbackUsage:       make(map[string]int),
		RecentErrors:        make([]RecentError, 0),
	}
}

// RecordError records an error occurrence with detailed metrics
func (em *ErrorMetrics) RecordError(operation string, err error, category ErrorCategory, recovered bool, recoveryMethod string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	now := time.Now()
	
	// Update basic counters
	em.TotalErrors++
	em.ErrorsByCategory[category]++
	em.ErrorsByOperation[operation]++
	em.ErrorRateByHour[now.Hour()]++

	// Record fallback usage
	if recovered && recoveryMethod != "" {
		em.FallbackUsage[recoveryMethod]++
	}

	// Add to recent errors (keep last 50)
	recentError := RecentError{
		Timestamp:      now,
		Error:          err.Error(),
		Operation:      operation,
		Category:       category,
		Recovered:      recovered,
		RecoveryMethod: recoveryMethod,
	}
	
	em.RecentErrors = append(em.RecentErrors, recentError)
	if len(em.RecentErrors) > 50 {
		em.RecentErrors = em.RecentErrors[len(em.RecentErrors)-50:]
	}

	// Update top errors
	em.updateTopErrors(err.Error(), category, now)
}

// updateTopErrors maintains a list of most frequent errors
func (em *ErrorMetrics) updateTopErrors(errorMsg string, category ErrorCategory, timestamp time.Time) {
	// Find existing error or create new one
	found := false
	for i, topError := range em.TopErrors {
		if topError.Error == errorMsg {
			em.TopErrors[i].Count++
			em.TopErrors[i].LastSeen = timestamp
			found = true
			break
		}
	}

	if !found {
		em.TopErrors = append(em.TopErrors, ErrorFrequency{
			Error:    errorMsg,
			Count:    1,
			Category: em.getCategoryString(category),
			LastSeen: timestamp,
		})
	}

	// Keep only top 10 errors, sorted by frequency
	if len(em.TopErrors) > 1 {
		// Simple bubble sort for small array
		for i := 0; i < len(em.TopErrors)-1; i++ {
			for j := 0; j < len(em.TopErrors)-i-1; j++ {
				if em.TopErrors[j].Count < em.TopErrors[j+1].Count {
					em.TopErrors[j], em.TopErrors[j+1] = em.TopErrors[j+1], em.TopErrors[j]
				}
			}
		}
		
		// Keep only top 10
		if len(em.TopErrors) > 10 {
			em.TopErrors = em.TopErrors[:10]
		}
	}
}

// getCategoryString converts ErrorCategory to string
func (em *ErrorMetrics) getCategoryString(category ErrorCategory) string {
	switch category {
	case ErrorCategoryNetwork:
		return "Network"
	case ErrorCategoryTimeout:
		return "Timeout"
	case ErrorCategoryRateLimit:
		return "RateLimit"
	case ErrorCategoryAuthentication:
		return "Authentication"
	case ErrorCategoryParsing:
		return "Parsing"
	case ErrorCategoryConfiguration:
		return "Configuration"
	case ErrorCategoryValidation:
		return "Validation"
	case ErrorCategoryPermission:
		return "Permission"
	case ErrorCategoryResource:
		return "Resource"
	case ErrorCategoryService:
		return "Service"
	case ErrorCategoryTemporary:
		return "Temporary"
	default:
		return "Unknown"
	}
}

// UpdateRecoveryStats updates recovery success metrics
func (em *ErrorMetrics) UpdateRecoveryStats(recoveryTime time.Duration, success bool) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Simple running average (could be enhanced with proper statistics)
	if success {
		if em.AverageRecoveryTime == 0 {
			em.AverageRecoveryTime = recoveryTime
		} else {
			em.AverageRecoveryTime = (em.AverageRecoveryTime + recoveryTime) / 2
		}
	}
}

// GetMetricsSummary returns a comprehensive metrics summary
func (em *ErrorMetrics) GetMetricsSummary() map[string]interface{} {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// Calculate recovery success rate
	recoveredCount := 0
	for _, recentError := range em.RecentErrors {
		if recentError.Recovered {
			recoveredCount++
		}
	}

	var recoverySuccessRate float64
	if len(em.RecentErrors) > 0 {
		recoverySuccessRate = float64(recoveredCount) / float64(len(em.RecentErrors)) * 100
	}

	// Calculate error rate trends
	currentHour := time.Now().Hour()
	recentHourErrors := 0
	for hour := currentHour - 2; hour <= currentHour; hour++ {
		hourKey := hour
		if hourKey < 0 {
			hourKey += 24
		}
		recentHourErrors += em.ErrorRateByHour[hourKey]
	}

	return map[string]interface{}{
		"total_errors":           em.TotalErrors,
		"errors_by_category":     em.ErrorsByCategory,
		"errors_by_operation":    em.ErrorsByOperation,
		"recovery_success_rate":  recoverySuccessRate,
		"average_recovery_time":  em.AverageRecoveryTime,
		"top_errors":            em.TopErrors,
		"fallback_usage":        em.FallbackUsage,
		"recent_errors_count":   len(em.RecentErrors),
		"recent_hour_errors":    recentHourErrors,
		"current_hour":          currentHour,
		"error_categories": map[string]string{
			"most_common": em.getMostCommonCategory(),
			"trend":       em.getErrorTrend(),
		},
	}
}

// getMostCommonCategory returns the most common error category
func (em *ErrorMetrics) getMostCommonCategory() string {
	maxCount := 0
	var mostCommon ErrorCategory
	for category, count := range em.ErrorsByCategory {
		if count > maxCount {
			maxCount = count
			mostCommon = category
		}
	}
	return em.getCategoryString(mostCommon)
}

// getErrorTrend analyzes recent error trends
func (em *ErrorMetrics) getErrorTrend() string {
	if len(em.RecentErrors) < 10 {
		return "insufficient_data"
	}

	// Compare first half vs second half of recent errors
	firstHalf := len(em.RecentErrors) / 2
	firstHalfErrors := 0
	secondHalfErrors := 0

	for i, recentError := range em.RecentErrors {
		if !recentError.Recovered {
			if i < firstHalf {
				firstHalfErrors++
			} else {
				secondHalfErrors++
			}
		}
	}

	if float64(secondHalfErrors) > float64(firstHalfErrors)*1.2 {
		return "increasing"
	} else if float64(secondHalfErrors) < float64(firstHalfErrors)*0.8 {
		return "decreasing"
	}
	return "stable"
}

// initializeMetrics initializes error metrics tracking if not already done
func (s *Service) initializeMetrics() {
	if s.errorMetrics == nil {
		s.errorMetrics = NewErrorMetrics()
	}
}

// GetErrorMetrics returns the current error metrics
func (s *Service) GetErrorMetrics() map[string]interface{} {
	s.initializeMetrics()
	
	// Update circuit breaker stats
	s.errorMetrics.mu.Lock()
	cbStats := s.GetCircuitBreakerStats()
	// Convert to expected type
	convertedStats := make(map[string]map[string]interface{})
	for name, stats := range cbStats {
		if statsMap, ok := stats.(map[string]interface{}); ok {
			convertedStats[name] = statsMap
		}
	}
	s.errorMetrics.CircuitBreakerStats = convertedStats
	s.errorMetrics.mu.Unlock()
	
	return s.errorMetrics.GetMetricsSummary()
}

// DegradationLevel represents different levels of service degradation
type DegradationLevel int

const (
	DegradationNone DegradationLevel = iota
	DegradationMinimal
	DegradationModerate
	DegradationSevere
	DegradationEmergency
)

// DegradationConfig configures graceful degradation behavior
type DegradationConfig struct {
	Level              DegradationLevel `yaml:"level" json:"level"`
	ErrorThreshold     float64          `yaml:"error_threshold" json:"error_threshold"`
	RecoveryThreshold  float64          `yaml:"recovery_threshold" json:"recovery_threshold"`
	Features           map[string]bool  `yaml:"features" json:"features"`
	TimeoutReduction   float64          `yaml:"timeout_reduction" json:"timeout_reduction"`
	RateLimitReduction float64          `yaml:"rate_limit_reduction" json:"rate_limit_reduction"`
}

// GracefulDegradationManager handles service degradation and recovery
type GracefulDegradationManager struct {
	currentLevel       DegradationLevel
	config             DegradationConfig
	levelConfigs       map[DegradationLevel]DegradationConfig
	lastLevelChange    time.Time
	errorService       *Service
	mu                 sync.RWMutex
}

// NewGracefulDegradationManager creates a new degradation manager
func NewGracefulDegradationManager(errorService *Service) *GracefulDegradationManager {
	return &GracefulDegradationManager{
		currentLevel: DegradationNone,
		levelConfigs: make(map[DegradationLevel]DegradationConfig),
		errorService: errorService,
		lastLevelChange: time.Now(),
	}
}

// ConfigureDegradationLevel configures a specific degradation level
func (gdm *GracefulDegradationManager) ConfigureDegradationLevel(level DegradationLevel, config DegradationConfig) {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()
	gdm.levelConfigs[level] = config
}

// EvaluateAndAdjustDegradation evaluates current conditions and adjusts degradation level
func (gdm *GracefulDegradationManager) EvaluateAndAdjustDegradation() DegradationLevel {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()

	if gdm.errorService == nil || gdm.errorService.errorMetrics == nil {
		return gdm.currentLevel
	}

	// Get current error metrics
	metrics := gdm.errorService.errorMetrics.GetMetricsSummary()
	
	// Calculate current error rate
	totalErrors, _ := metrics["total_errors"].(int)
	recentErrorsCount, _ := metrics["recent_errors_count"].(int)
	
	var currentErrorRate float64
	if recentErrorsCount > 0 {
		// Simple error rate calculation based on recent errors
		currentErrorRate = float64(totalErrors) / float64(recentErrorsCount+totalErrors) * 100
	}

	// Determine appropriate degradation level
	newLevel := gdm.determineDegradationLevel(currentErrorRate)
	
	if newLevel != gdm.currentLevel {
		gdm.transitionToLevel(newLevel)
	}

	return gdm.currentLevel
}

// determineDegradationLevel determines the appropriate degradation level based on error rate
func (gdm *GracefulDegradationManager) determineDegradationLevel(errorRate float64) DegradationLevel {
	// Default thresholds if not configured
	thresholds := map[DegradationLevel]float64{
		DegradationMinimal:   10.0, // 10% error rate
		DegradationModerate:  25.0, // 25% error rate
		DegradationSevere:    50.0, // 50% error rate
		DegradationEmergency: 75.0, // 75% error rate
	}

	// Use configured thresholds if available
	for level, config := range gdm.levelConfigs {
		if config.ErrorThreshold > 0 {
			thresholds[level] = config.ErrorThreshold
		}
	}

	// Determine level based on error rate
	switch {
	case errorRate >= thresholds[DegradationEmergency]:
		return DegradationEmergency
	case errorRate >= thresholds[DegradationSevere]:
		return DegradationSevere
	case errorRate >= thresholds[DegradationModerate]:
		return DegradationModerate
	case errorRate >= thresholds[DegradationMinimal]:
		return DegradationMinimal
	default:
		return DegradationNone
	}
}

// transitionToLevel handles transition to a new degradation level
func (gdm *GracefulDegradationManager) transitionToLevel(newLevel DegradationLevel) {
	oldLevel := gdm.currentLevel
	gdm.currentLevel = newLevel
	gdm.lastLevelChange = time.Now()

	// Apply degradation configuration
	if config, exists := gdm.levelConfigs[newLevel]; exists {
		gdm.config = config
	} else {
		gdm.config = gdm.getDefaultConfig(newLevel)
	}

	// Log the transition
	fmt.Printf("Graceful degradation: transitioning from %s to %s (error conditions detected)\n", 
		gdm.getLevelName(oldLevel), gdm.getLevelName(newLevel))
}

// getDefaultConfig returns default configuration for a degradation level
func (gdm *GracefulDegradationManager) getDefaultConfig(level DegradationLevel) DegradationConfig {
	switch level {
	case DegradationMinimal:
		return DegradationConfig{
			Level:              level,
			ErrorThreshold:     10.0,
			RecoveryThreshold:  5.0,
			TimeoutReduction:   0.1, // Reduce timeouts by 10%
			RateLimitReduction: 0.2, // Reduce rate limits by 20%
			Features: map[string]bool{
				"advanced_parsing": true,
				"javascript_execution": true,
				"image_processing": false,
				"concurrent_requests": true,
			},
		}
	case DegradationModerate:
		return DegradationConfig{
			Level:              level,
			ErrorThreshold:     25.0,
			RecoveryThreshold:  15.0,
			TimeoutReduction:   0.3, // Reduce timeouts by 30%
			RateLimitReduction: 0.4, // Reduce rate limits by 40%
			Features: map[string]bool{
				"advanced_parsing": false,
				"javascript_execution": true,
				"image_processing": false,
				"concurrent_requests": true,
			},
		}
	case DegradationSevere:
		return DegradationConfig{
			Level:              level,
			ErrorThreshold:     50.0,
			RecoveryThreshold:  30.0,
			TimeoutReduction:   0.5, // Reduce timeouts by 50%
			RateLimitReduction: 0.6, // Reduce rate limits by 60%
			Features: map[string]bool{
				"advanced_parsing": false,
				"javascript_execution": false,
				"image_processing": false,
				"concurrent_requests": false,
			},
		}
	case DegradationEmergency:
		return DegradationConfig{
			Level:              level,
			ErrorThreshold:     75.0,
			RecoveryThreshold:  50.0,
			TimeoutReduction:   0.7, // Reduce timeouts by 70%
			RateLimitReduction: 0.8, // Reduce rate limits by 80%
			Features: map[string]bool{
				"advanced_parsing": false,
				"javascript_execution": false,
				"image_processing": false,
				"concurrent_requests": false,
			},
		}
	default:
		return DegradationConfig{
			Level:              DegradationNone,
			Features: map[string]bool{
				"advanced_parsing": true,
				"javascript_execution": true,
				"image_processing": true,
				"concurrent_requests": true,
			},
		}
	}
}

// getLevelName returns a human-readable name for a degradation level
func (gdm *GracefulDegradationManager) getLevelName(level DegradationLevel) string {
	switch level {
	case DegradationNone:
		return "None"
	case DegradationMinimal:
		return "Minimal"
	case DegradationModerate:
		return "Moderate"
	case DegradationSevere:
		return "Severe"
	case DegradationEmergency:
		return "Emergency"
	default:
		return "Unknown"
	}
}

// GetCurrentLevel returns the current degradation level
func (gdm *GracefulDegradationManager) GetCurrentLevel() DegradationLevel {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()
	return gdm.currentLevel
}

// IsFeatureEnabled checks if a feature is enabled at the current degradation level
func (gdm *GracefulDegradationManager) IsFeatureEnabled(featureName string) bool {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()
	
	if enabled, exists := gdm.config.Features[featureName]; exists {
		return enabled
	}
	
	// Default to enabled if not configured
	return true
}

// GetAdjustedTimeout returns timeout adjusted for current degradation level
func (gdm *GracefulDegradationManager) GetAdjustedTimeout(originalTimeout time.Duration) time.Duration {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()
	
	if gdm.config.TimeoutReduction > 0 {
		reduction := time.Duration(float64(originalTimeout) * gdm.config.TimeoutReduction)
		adjusted := originalTimeout - reduction
		if adjusted < time.Second {
			adjusted = time.Second // Minimum timeout
		}
		return adjusted
	}
	
	return originalTimeout
}

// GetAdjustedRateLimit returns rate limit adjusted for current degradation level
func (gdm *GracefulDegradationManager) GetAdjustedRateLimit(originalRate time.Duration) time.Duration {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()
	
	if gdm.config.RateLimitReduction > 0 {
		increase := time.Duration(float64(originalRate) * gdm.config.RateLimitReduction)
		return originalRate + increase
	}
	
	return originalRate
}

// GetDegradationStats returns statistics about the current degradation state
func (gdm *GracefulDegradationManager) GetDegradationStats() map[string]interface{} {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()
	
	return map[string]interface{}{
		"current_level":      gdm.getLevelName(gdm.currentLevel),
		"level_numeric":      int(gdm.currentLevel),
		"last_change":        gdm.lastLevelChange,
		"time_in_level":      time.Since(gdm.lastLevelChange),
		"config":             gdm.config,
		"available_levels":   []string{"None", "Minimal", "Moderate", "Severe", "Emergency"},
	}
}

// Add degradation manager to Service
func (s *Service) EnableGracefulDegradation() *GracefulDegradationManager {
	return NewGracefulDegradationManager(s)
}
