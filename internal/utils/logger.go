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

// Info logs an info message
func (cl *ComponentLogger) Info(msg string) {
	if globalLogLevel <= LevelInfo {
		cl.logger.Printf("INFO: %s", msg)
	}
}

// Warn logs a warning message
func (cl *ComponentLogger) Warn(msg string) {
	if globalLogLevel <= LevelWarn {
		cl.logger.Printf("WARN: %s", msg)
	}
}

// Error logs an error message
func (cl *ComponentLogger) Error(msg string) {
	if globalLogLevel <= LevelError {
		cl.logger.Printf("ERROR: %s", msg)
	}
}
