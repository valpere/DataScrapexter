// internal/utils/logger.go

package utils

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Logger defines the interface for logging throughout the application.
type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// SimpleLogger provides a basic logger implementation for development.
type SimpleLogger struct {
	level  LogLevel
	fields map[string]interface{}
	mu     sync.RWMutex
}

// NewLogger creates a new simple logger instance.
func NewLogger() Logger {
	return &SimpleLogger{
		level:  InfoLevel,
		fields: make(map[string]interface{}),
	}
}

// NewLoggerWithLevel creates a logger with the specified log level.
func NewLoggerWithLevel(level LogLevel) Logger {
	return &SimpleLogger{
		level:  level,
		fields: make(map[string]interface{}),
	}
}

// Implementation of Logger interface for SimpleLogger

func (l *SimpleLogger) Debug(msg string) {
	l.log(DebugLevel, msg)
}

func (l *SimpleLogger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

func (l *SimpleLogger) Info(msg string) {
	l.log(InfoLevel, msg)
}

func (l *SimpleLogger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

func (l *SimpleLogger) Warn(msg string) {
	l.log(WarnLevel, msg)
}

func (l *SimpleLogger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

func (l *SimpleLogger) Error(msg string) {
	l.log(ErrorLevel, msg)
}

func (l *SimpleLogger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

func (l *SimpleLogger) WithField(key string, value interface{}) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	newFields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	
	return &SimpleLogger{
		level:  l.level,
		fields: newFields,
	}
}

func (l *SimpleLogger) WithFields(fields map[string]interface{}) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	
	return &SimpleLogger{
		level:  l.level,
		fields: newFields,
	}
}

// log formats and outputs a log message if it meets the minimum level.
func (l *SimpleLogger) log(level LogLevel, msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if level < l.level {
		return
	}
	
	// Format: [TIME] [LEVEL] message fields={...}
	levelStr := [...]string{"DEBUG", "INFO", "WARN", "ERROR"}[level]
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	output := fmt.Sprintf("[%s] [%s] %s", timestamp, levelStr, msg)
	
	if len(l.fields) > 0 {
		output += " fields=" + formatFields(l.fields)
	}
	
	fmt.Println(output)
}

// formatFields converts fields map to a string representation.
func formatFields(fields map[string]interface{}) string {
	parts := make([]string, 0, len(fields))
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
