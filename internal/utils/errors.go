// Package utils provides enhanced error handling utilities
// for better error management and debugging.
package utils

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// String returns string representation of error severity
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ErrorCode represents predefined error codes for categorization
type ErrorCode string

const (
	// Network related errors
	ErrCodeNetworkTimeout     ErrorCode = "NETWORK_TIMEOUT"
	ErrCodeNetworkUnreachable ErrorCode = "NETWORK_UNREACHABLE"
	ErrCodeDNSFailure         ErrorCode = "DNS_FAILURE"
	ErrCodeSSLError           ErrorCode = "SSL_ERROR"

	// Configuration related errors
	ErrCodeInvalidConfig ErrorCode = "INVALID_CONFIG"
	ErrCodeMissingConfig ErrorCode = "MISSING_CONFIG"
	ErrCodeConfigSyntax  ErrorCode = "CONFIG_SYNTAX"

	// Scraping related errors
	ErrCodeSelectorNotFound ErrorCode = "SELECTOR_NOT_FOUND"
	ErrCodeExtractionFailed ErrorCode = "EXTRACTION_FAILED"
	ErrCodeParsingError     ErrorCode = "PARSING_ERROR"
	ErrCodeRateLimited      ErrorCode = "RATE_LIMITED"

	// Output related errors
	ErrCodeOutputFailed    ErrorCode = "OUTPUT_FAILED"
	ErrCodeFilePermission  ErrorCode = "FILE_PERMISSION"
	ErrCodeDiskSpace       ErrorCode = "DISK_SPACE"
	ErrCodeDatabaseError   ErrorCode = "DATABASE_ERROR"

	// Authentication and authorization
	ErrCodeAuthFailed      ErrorCode = "AUTH_FAILED"
	ErrCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
	ErrCodeTokenExpired    ErrorCode = "TOKEN_EXPIRED"

	// Anti-detection related
	ErrCodeCaptchaFailed   ErrorCode = "CAPTCHA_FAILED"
	ErrCodeProxyFailed     ErrorCode = "PROXY_FAILED"
	ErrCodeBrowserFailed   ErrorCode = "BROWSER_FAILED"
	ErrCodeDetectionBlocked ErrorCode = "DETECTION_BLOCKED"

	// System related errors
	ErrCodeResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
	ErrCodeMemoryLimit      ErrorCode = "MEMORY_LIMIT"
	ErrCodeCPULimit         ErrorCode = "CPU_LIMIT"
	ErrCodeContextCanceled  ErrorCode = "CONTEXT_CANCELED"

	// Generic errors
	ErrCodeInternal   ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnknown    ErrorCode = "UNKNOWN_ERROR"
	ErrCodeValidation ErrorCode = "VALIDATION_ERROR"
)

// StructuredError provides rich error information for better debugging and handling
type StructuredError struct {
	Code        ErrorCode              `json:"code"`
	Message     string                 `json:"message"`
	Severity    ErrorSeverity          `json:"severity"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Cause       error                  `json:"-"` // Original error
	Timestamp   time.Time              `json:"timestamp"`
	StackTrace  []string               `json:"stack_trace,omitempty"`
	Retryable   bool                   `json:"retryable"`
	UserMessage string                 `json:"user_message,omitempty"`
}

// Error implements the error interface
func (e *StructuredError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for error unwrapping
func (e *StructuredError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches a target error code
func (e *StructuredError) Is(target error) bool {
	if se, ok := target.(*StructuredError); ok {
		return e.Code == se.Code
	}
	return false
}

// WithContext adds contextual information to the error
func (e *StructuredError) WithContext(key string, value interface{}) *StructuredError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithUserMessage sets a user-friendly error message
func (e *StructuredError) WithUserMessage(message string) *StructuredError {
	e.UserMessage = message
	return e
}

// ErrorBuilder provides a fluent interface for creating structured errors
type ErrorBuilder struct {
	error           *StructuredError
	stackTraceDepth int // Configurable stack trace depth
}

// ErrorConfig holds configuration for error handling
type ErrorConfig struct {
	StackTraceDepth     int  // Number of stack frames to capture (default: 15)
	EnableStackTrace    bool // Whether to capture stack traces (default: true)
	IncludeGoroutineID  bool // Whether to include goroutine ID in stack traces (default: false)
}

// DefaultErrorConfig returns the default error configuration
func DefaultErrorConfig() *ErrorConfig {
	return &ErrorConfig{
		StackTraceDepth:    15,
		EnableStackTrace:   true,
		IncludeGoroutineID: false,
	}
}

// Global error configuration
var globalErrorConfig = DefaultErrorConfig()

// SetGlobalErrorConfig sets the global error configuration
func SetGlobalErrorConfig(config *ErrorConfig) {
	if config != nil {
		globalErrorConfig = config
	}
}

// GetGlobalErrorConfig returns the current global error configuration
func GetGlobalErrorConfig() *ErrorConfig {
	return globalErrorConfig
}

// NewError creates a new error builder with default configuration
func NewError(code ErrorCode, message string) *ErrorBuilder {
	return NewErrorWithConfig(code, message, globalErrorConfig)
}

// NewErrorWithConfig creates a new error builder with custom configuration
func NewErrorWithConfig(code ErrorCode, message string, config *ErrorConfig) *ErrorBuilder {
	if config == nil {
		config = DefaultErrorConfig()
	}

	builder := &ErrorBuilder{
		error: &StructuredError{
			Code:      code,
			Message:   message,
			Severity:  SeverityError,
			Timestamp: time.Now(),
			Retryable: false,
		},
		stackTraceDepth: config.StackTraceDepth,
	}

	// Capture stack trace if enabled
	if config.EnableStackTrace {
		builder.error.StackTrace = captureStackTraceWithDepth(config.StackTraceDepth, config.IncludeGoroutineID)
	}

	return builder
}

// WithSeverity sets the error severity
func (eb *ErrorBuilder) WithSeverity(severity ErrorSeverity) *ErrorBuilder {
	eb.error.Severity = severity
	return eb
}

// WithCause sets the underlying cause
func (eb *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	eb.error.Cause = cause
	return eb
}

// WithContext adds contextual information
func (eb *ErrorBuilder) WithContext(key string, value interface{}) *ErrorBuilder {
	if eb.error.Context == nil {
		eb.error.Context = make(map[string]interface{})
	}
	eb.error.Context[key] = value
	return eb
}

// WithRetryable marks the error as retryable
func (eb *ErrorBuilder) WithRetryable(retryable bool) *ErrorBuilder {
	eb.error.Retryable = retryable
	return eb
}

// WithUserMessage sets a user-friendly message
func (eb *ErrorBuilder) WithUserMessage(message string) *ErrorBuilder {
	eb.error.UserMessage = message
	return eb
}

// WithStackTraceDepth sets the stack trace depth for this error
func (eb *ErrorBuilder) WithStackTraceDepth(depth int) *ErrorBuilder {
	if depth > 0 {
		eb.stackTraceDepth = depth
		// Recapture stack trace with new depth
		eb.error.StackTrace = captureStackTraceWithDepth(depth, globalErrorConfig.IncludeGoroutineID)
	}
	return eb
}

// WithoutStackTrace disables stack trace capture for this error
func (eb *ErrorBuilder) WithoutStackTrace() *ErrorBuilder {
	eb.error.StackTrace = nil
	return eb
}

// Build returns the constructed error
func (eb *ErrorBuilder) Build() *StructuredError {
	return eb.error
}

// ErrorCollector collects multiple errors for batch processing
type ErrorCollector struct {
	errors   []*StructuredError
	maxErrors int
}

// NewErrorCollector creates a new error collector
func NewErrorCollector(maxErrors int) *ErrorCollector {
	if maxErrors <= 0 {
		maxErrors = 100 // Default limit
	}
	return &ErrorCollector{
		errors:   make([]*StructuredError, 0),
		maxErrors: maxErrors,
	}
}

// Add adds an error to the collection
func (ec *ErrorCollector) Add(err *StructuredError) {
	if len(ec.errors) < ec.maxErrors {
		ec.errors = append(ec.errors, err)
	}
}

// AddSimple adds a simple error with basic information
func (ec *ErrorCollector) AddSimple(code ErrorCode, message string) {
	err := NewError(code, message).Build()
	ec.Add(err)
}

// HasErrors returns true if there are collected errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []*StructuredError {
	return ec.errors
}

// Count returns the number of collected errors
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// FirstError returns the first collected error
func (ec *ErrorCollector) FirstError() *StructuredError {
	if len(ec.errors) > 0 {
		return ec.errors[0]
	}
	return nil
}

// Clear removes all collected errors
func (ec *ErrorCollector) Clear() {
	ec.errors = ec.errors[:0]
}

// ToMultiError converts the collection to a multi-error
func (ec *ErrorCollector) ToMultiError() error {
	if !ec.HasErrors() {
		return nil
	}
	return &MultiError{errors: ec.errors}
}

// MultiError represents multiple errors as a single error
type MultiError struct {
	errors []*StructuredError
}

// Error implements the error interface
func (me *MultiError) Error() string {
	if len(me.errors) == 0 {
		return "no errors"
	}
	if len(me.errors) == 1 {
		return me.errors[0].Error()
	}
	
	messages := make([]string, len(me.errors))
	for i, err := range me.errors {
		messages[i] = err.Error()
	}
	return fmt.Sprintf("multiple errors occurred: [%s]", strings.Join(messages, "; "))
}

// Errors returns all contained errors
func (me *MultiError) Errors() []*StructuredError {
	return me.errors
}

// ErrorHandler provides centralized error handling with recovery strategies
type ErrorHandler struct {
	logger func(error)        // Optional logger function
	retryStrategies map[ErrorCode]RetryStrategy
}

// RetryStrategy defines how to handle retry logic for specific error types
type RetryStrategy struct {
	MaxAttempts int
	Delay       time.Duration
	BackoffType BackoffType
	ShouldRetry func(error) bool
}

// BackoffType defines different backoff strategies
type BackoffType int

const (
	BackoffFixed BackoffType = iota
	BackoffLinear
	BackoffExponential
	BackoffJittered
)

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		retryStrategies: make(map[ErrorCode]RetryStrategy),
	}
}

// WithLogger sets a logger function for the error handler
func (eh *ErrorHandler) WithLogger(logger func(error)) *ErrorHandler {
	eh.logger = logger
	return eh
}

// AddRetryStrategy adds a retry strategy for a specific error code
func (eh *ErrorHandler) AddRetryStrategy(code ErrorCode, strategy RetryStrategy) {
	eh.retryStrategies[code] = strategy
}

// Handle processes an error with appropriate handling strategy
func (eh *ErrorHandler) Handle(ctx context.Context, err error) error {
	if eh.logger != nil {
		eh.logger(err)
	}

	// Check if it's a structured error with retry strategy
	if structErr, ok := err.(*StructuredError); ok {
		if strategy, exists := eh.retryStrategies[structErr.Code]; exists && structErr.Retryable {
			return eh.executeWithRetry(ctx, strategy, func() error {
				return err // This would typically be the operation that failed
			})
		}
	}

	return err
}

// executeWithRetry executes a function with retry logic
func (eh *ErrorHandler) executeWithRetry(ctx context.Context, strategy RetryStrategy, fn func() error) error {
	var lastErr error
	delay := strategy.Delay

	for attempt := 0; attempt < strategy.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry this error
		if strategy.ShouldRetry != nil && !strategy.ShouldRetry(err) {
			break
		}

		// Don't sleep on last attempt
		if attempt < strategy.MaxAttempts-1 {
			time.Sleep(delay)
			delay = eh.calculateNextDelay(delay, strategy.BackoffType, attempt)
		}
	}

	return lastErr
}

// calculateNextDelay calculates the next delay based on backoff type
func (eh *ErrorHandler) calculateNextDelay(currentDelay time.Duration, backoffType BackoffType, attempt int) time.Duration {
	switch backoffType {
	case BackoffFixed:
		return currentDelay
	case BackoffLinear:
		return currentDelay * time.Duration(attempt+2)
	case BackoffExponential:
		return currentDelay * 2
	case BackoffJittered:
		// Add some randomness to prevent thundering herd
		jitter := time.Duration(attempt) * time.Millisecond * 100
		return currentDelay*2 + jitter
	default:
		return currentDelay
	}
}

// Helper functions

// captureStackTrace captures the current stack trace with default depth
func captureStackTrace() []string {
	return captureStackTraceWithDepth(globalErrorConfig.StackTraceDepth, globalErrorConfig.IncludeGoroutineID)
}

// captureStackTraceWithDepth captures the current stack trace with specified depth
func captureStackTraceWithDepth(depth int, includeGoroutineID bool) []string {
	if depth <= 0 {
		return nil
	}

	var stack []string
	
	// Add goroutine ID if requested
	if includeGoroutineID {
		stack = append(stack, fmt.Sprintf("[goroutine %d]", getGoroutineID()))
	}

	// Skip first 2 frames (this function and caller) by default
	// For functions called from NewError*, we need to skip more frames
	skipFrames := 2
	if depth > 50 { // Assume this is for deep debugging
		skipFrames = 1
	}

	frameCount := 0
	for i := skipFrames; frameCount < depth; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		// Get function name
		funcName := "unknown"
		if fn := runtime.FuncForPC(pc); fn != nil {
			funcName = fn.Name()
		}

		// Format stack frame with more detail
		stackFrame := fmt.Sprintf("%s:%d (%s)", shortenFilePath(file), line, shortenFuncName(funcName))
		stack = append(stack, stackFrame)
		frameCount++
	}

	return stack
}

// shortenFilePath shortens file paths for better readability
func shortenFilePath(filePath string) string {
	// Keep only the last two path components for readability
	parts := strings.Split(filePath, "/")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return filePath
}

// shortenFuncName shortens function names for better readability
func shortenFuncName(funcName string) string {
	// Remove package path, keep only the last component
	parts := strings.Split(funcName, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Further shorten if it contains dots
		if dotIndex := strings.LastIndex(lastPart, "."); dotIndex != -1 && dotIndex < len(lastPart)-1 {
			return lastPart[dotIndex+1:]
		}
		return lastPart
	}
	return funcName
}

// getGoroutineID returns the current goroutine ID.
// WARNING: This is an expensive operation (uses runtime.Stack and string parsing).
// Only use for debugging purposes.
func getGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	if id, err := strconv.ParseUint(idField, 10, 64); err == nil {
		return id
	}
	return 0
}

// IsRetryableError checks if an error should be retried
func IsRetryableError(err error) bool {
	if structErr, ok := err.(*StructuredError); ok {
		return structErr.Retryable
	}

	// Check for common retryable error patterns
	errorStr := err.Error()
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"temporary failure",
		"503 service unavailable",
		"502 bad gateway",
		"504 gateway timeout",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errorStr), pattern) {
			return true
		}
	}

	return false
}

// WrapError wraps an existing error in a structured error
func WrapError(err error, code ErrorCode, message string) *StructuredError {
	return NewError(code, message).WithCause(err).Build()
}

// IsTemporaryError checks if an error is temporary
func IsTemporaryError(err error) bool {
	if temp, ok := err.(interface{ Temporary() bool }); ok {
		return temp.Temporary()
	}
	return IsRetryableError(err)
}

// GetUserFriendlyMessage extracts a user-friendly message from an error
func GetUserFriendlyMessage(err error) string {
	if structErr, ok := err.(*StructuredError); ok && structErr.UserMessage != "" {
		return structErr.UserMessage
	}

	// Provide default user-friendly messages for common error codes
	if structErr, ok := err.(*StructuredError); ok {
		switch structErr.Code {
		case ErrCodeNetworkTimeout:
			return "The request timed out. Please check your internet connection and try again."
		case ErrCodeRateLimited:
			return "Too many requests. Please wait a moment before trying again."
		case ErrCodeCaptchaFailed:
			return "CAPTCHA verification failed. Please try again."
		case ErrCodeSelectorNotFound:
			return "Unable to find the requested data on the page. The website structure may have changed."
		case ErrCodeOutputFailed:
			return "Failed to save the results. Please check file permissions and available disk space."
		default:
			return "An unexpected error occurred. Please try again or contact support if the problem persists."
		}
	}

	return "An error occurred. Please try again."
}
