// internal/output/manager.go
package output

import (
	"fmt"
	"path/filepath"
	"strings"
)

// OutputFormat represents supported output formats
type OutputFormat string

const (
	FormatJSON OutputFormat = "json"
	FormatCSV  OutputFormat = "csv"
	FormatJSONL OutputFormat = "jsonl"
)

// OutputConfig defines output configuration
type OutputConfig struct {
	Format    OutputFormat `yaml:"format" json:"format"`
	FilePath  string       `yaml:"file" json:"file"`
	Append    bool         `yaml:"append" json:"append"`
	
	// JSON-specific
	Indent    string `yaml:"indent,omitempty" json:"indent,omitempty"`
	Stream    bool   `yaml:"stream,omitempty" json:"stream,omitempty"`
	
	// CSV-specific  
	Delimiter string   `yaml:"delimiter,omitempty" json:"delimiter,omitempty"`
	Headers   bool     `yaml:"headers,omitempty" json:"headers,omitempty"`
	Fields    []string `yaml:"fields,omitempty" json:"fields,omitempty"`
}

// Writer interface for output writers
type Writer interface {
	WriteData(data map[string]interface{}) error
	WriteBatch(items []map[string]interface{}) error
	Flush() error
	Close() error
	GetStats() map[string]interface{}
}

// OutputManager manages output writers with factory pattern
type OutputManager struct{}

// NewOutputManager creates a new output manager
func NewOutputManager() *OutputManager {
	return &OutputManager{}
}

// CreateWriter creates appropriate writer based on config
func (om *OutputManager) CreateWriter(config *OutputConfig) (Writer, error) {
	if err := om.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Auto-detect format from file extension if not specified
	if config.Format == "" {
		config.Format = om.DetectFormat(config.FilePath)
	}

	// Ensure proper file extension
	config.FilePath = om.EnsureExtension(config.FilePath, config.Format)

	switch config.Format {
	case FormatJSON:
		return om.createJSONWriter(config)
	case FormatJSONL:
		return om.createJSONLWriter(config)
	case FormatCSV:
		return om.createCSVWriter(config)
	default:
		return nil, fmt.Errorf("unsupported format: %s", config.Format)
	}
}

// createJSONWriter creates JSON writer
func (om *OutputManager) createJSONWriter(config *OutputConfig) (Writer, error) {
	jsonConfig := &JSONWriterConfig{
		FilePath:   config.FilePath,
		Indent:     config.Indent,
		AppendMode: config.Append,
		StreamMode: config.Stream,
	}
	
	return NewJSONWriter(jsonConfig)
}

// createJSONLWriter creates JSONL (streaming) writer
func (om *OutputManager) createJSONLWriter(config *OutputConfig) (Writer, error) {
	jsonConfig := &JSONWriterConfig{
		FilePath:   config.FilePath,
		AppendMode: config.Append,
		StreamMode: true, // Force stream mode for JSONL
	}
	
	return NewJSONWriter(jsonConfig)
}

// createCSVWriter creates CSV writer
func (om *OutputManager) createCSVWriter(config *OutputConfig) (Writer, error) {
	delimiter := ','
	if config.Delimiter != "" {
		if len(config.Delimiter) == 1 {
			delimiter = rune(config.Delimiter[0])
		} else {
			return nil, fmt.Errorf("delimiter must be single character")
		}
	}

	csvConfig := &CSVWriterConfig{
		FilePath:         config.FilePath,
		Delimiter:        delimiter,
		Headers:          config.Headers,
		AppendMode:       config.Append,
		FieldOrder:       config.Fields,
		AutoDetectFields: len(config.Fields) == 0,
	}
	
	return NewCSVWriter(csvConfig)
}

// DetectFormat detects output format from file extension
func (om *OutputManager) DetectFormat(filePath string) OutputFormat {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return FormatJSON
	case ".jsonl":
		return FormatJSONL
	case ".csv":
		return FormatCSV
	default:
		return FormatJSON // Default fallback
	}
}

// EnsureExtension ensures proper file extension for format
func (om *OutputManager) EnsureExtension(filePath string, format OutputFormat) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch format {
	case FormatJSON:
		if ext != ".json" {
			return filePath + ".json"
		}
	case FormatJSONL:
		if ext != ".jsonl" {
			return filePath + ".jsonl"
		}
	case FormatCSV:
		if ext != ".csv" {
			return filePath + ".csv"
		}
	}
	
	return filePath
}

// ValidateConfig validates output configuration
func (om *OutputManager) ValidateConfig(config *OutputConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	if config.FilePath == "" {
		return fmt.Errorf("file path is required")
	}
	
	// Validate format if specified
	if config.Format != "" {
		switch config.Format {
		case FormatJSON, FormatJSONL, FormatCSV:
			// Valid formats
		default:
			return fmt.Errorf("unsupported format: %s", config.Format)
		}
	}
	
	// Validate CSV delimiter
	if config.Delimiter != "" && len(config.Delimiter) != 1 {
		return fmt.Errorf("CSV delimiter must be single character")
	}
	
	return nil
}

// GetSupportedFormats returns list of supported formats
func (om *OutputManager) GetSupportedFormats() []OutputFormat {
	return []OutputFormat{FormatJSON, FormatJSONL, FormatCSV}
}

// CreateMultiWriter creates multiple writers for different formats
func (om *OutputManager) CreateMultiWriter(configs []*OutputConfig) ([]Writer, error) {
	writers := make([]Writer, 0, len(configs))
	
	for i, config := range configs {
		writer, err := om.CreateWriter(config)
		if err != nil {
			// Clean up already created writers
			for j := 0; j < i; j++ {
				writers[j].Close()
			}
			return nil, fmt.Errorf("failed to create writer %d: %w", i, err)
		}
		writers = append(writers, writer)
	}
	
	return writers, nil
}

// MultiWriter writes to multiple outputs simultaneously
type MultiWriter struct {
	writers []Writer
}

// NewMultiWriter creates writer that outputs to multiple destinations
func NewMultiWriter(writers []Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// WriteData writes to all writers
func (mw *MultiWriter) WriteData(data map[string]interface{}) error {
	for i, writer := range mw.writers {
		if err := writer.WriteData(data); err != nil {
			return fmt.Errorf("writer %d failed: %w", i, err)
		}
	}
	return nil
}

// WriteBatch writes batch to all writers
func (mw *MultiWriter) WriteBatch(items []map[string]interface{}) error {
	for i, writer := range mw.writers {
		if err := writer.WriteBatch(items); err != nil {
			return fmt.Errorf("writer %d failed: %w", i, err)
		}
	}
	return nil
}

// Flush flushes all writers
func (mw *MultiWriter) Flush() error {
	for i, writer := range mw.writers {
		if err := writer.Flush(); err != nil {
			return fmt.Errorf("writer %d flush failed: %w", i, err)
		}
	}
	return nil
}

// Close closes all writers
func (mw *MultiWriter) Close() error {
	var lastErr error
	for _, writer := range mw.writers {
		if err := writer.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// GetStats returns combined stats from all writers
func (mw *MultiWriter) GetStats() map[string]interface{} {
	allStats := make(map[string]interface{})
	
	for i, writer := range mw.writers {
		stats := writer.GetStats()
		for key, value := range stats {
			allStats[fmt.Sprintf("writer_%d_%s", i, key)] = value
		}
	}
	
	allStats["writer_count"] = len(mw.writers)
	return allStats
}
