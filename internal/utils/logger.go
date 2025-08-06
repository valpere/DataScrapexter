// internal/utils/logger.go
package utils

import (
	"bytes"
	"fmt"
	"log"
	"os"
)

// ComponentLogger represents a component-specific logger
type ComponentLogger struct {
	component string
	logger    *log.Logger
}

// NewComponentLogger creates a new component logger
func NewComponentLogger(component string) *ComponentLogger {
	return &ComponentLogger{
		component: component,
		logger:    log.New(os.Stdout, fmt.Sprintf("[%s] ", component), log.LstdFlags),
	}
}

// WithField adds a field to the log context (simplified implementation)
func (cl *ComponentLogger) WithField(key string, value interface{}) *ComponentLogger {
	return cl
}

// WithFields adds multiple fields to the log context (simplified implementation)
func (cl *ComponentLogger) WithFields(fields map[string]interface{}) *ComponentLogger {
	return cl
}

// LogLevel represents logging levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var globalLogLevel = LevelInfo

// SetGlobalLogLevel sets the global logging level
func SetGlobalLogLevel(level LogLevel) {
	globalLogLevel = level
}

// SetGlobalLogOutput sets the global log output (for testing)
func SetGlobalLogOutput(output *bytes.Buffer) {
	// Simplified implementation for now
}

// Debug logs a debug message
func (cl *ComponentLogger) Debug(msg string) {
	if globalLogLevel <= LevelDebug {
		cl.logger.Printf("DEBUG: %s", msg)
	}
}

// Debugf logs a formatted debug message
func (cl *ComponentLogger) Debugf(format string, args ...interface{}) {
	if globalLogLevel <= LevelDebug {
		cl.logger.Printf("DEBUG: "+format, args...)
	}
}

// Info logs an info message
func (cl *ComponentLogger) Info(msg string) {
	if globalLogLevel <= LevelInfo {
		cl.logger.Printf("INFO: %s", msg)
	}
}

// Infof logs a formatted info message
func (cl *ComponentLogger) Infof(format string, args ...interface{}) {
	if globalLogLevel <= LevelInfo {
		cl.logger.Printf("INFO: "+format, args...)
	}
}

// Warn logs a warning message
func (cl *ComponentLogger) Warn(msg string) {
	if globalLogLevel <= LevelWarn {
		cl.logger.Printf("WARN: %s", msg)
	}
}

// Warnf logs a formatted warning message
func (cl *ComponentLogger) Warnf(format string, args ...interface{}) {
	if globalLogLevel <= LevelWarn {
		cl.logger.Printf("WARN: "+format, args...)
	}
}

// Error logs an error message
func (cl *ComponentLogger) Error(msg string) {
	if globalLogLevel <= LevelError {
		cl.logger.Printf("ERROR: %s", msg)
	}
}

// Errorf logs a formatted error message
func (cl *ComponentLogger) Errorf(format string, args ...interface{}) {
	if globalLogLevel <= LevelError {
		cl.logger.Printf("ERROR: "+format, args...)
	}
}

// Security logs a security-related message (always visible regardless of log level)
func (cl *ComponentLogger) Security(msg string) {
	cl.logger.Printf("SECURITY: %s", msg)
}

// Securityf logs a formatted security-related message (always visible)
func (cl *ComponentLogger) Securityf(format string, args ...interface{}) {
	cl.logger.Printf("SECURITY: "+format, args...)
}

// Panic logs a panic recovery message (always visible)
func (cl *ComponentLogger) Panic(msg string) {
	cl.logger.Printf("PANIC_RECOVERED: %s", msg)
}

// Panicf logs a formatted panic recovery message (always visible)
func (cl *ComponentLogger) Panicf(format string, args ...interface{}) {
	cl.logger.Printf("PANIC_RECOVERED: "+format, args...)
}

// GetLogger returns a component logger for the specified component
// This provides a centralized way to get loggers across the application
func GetLogger(component string) *ComponentLogger {
	return NewComponentLogger(component)
}
