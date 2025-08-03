// internal/output/yaml.go
package output

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultGeneratorName is the default generator name for metadata
	DefaultGeneratorName = "DataScrapexter"
)

// YAMLWriter implements the Writer interface for YAML output
type YAMLWriter struct {
	file       *os.File
	encoder    *yaml.Encoder
	config     YAMLConfig
	records    []map[string]interface{}
	isFirstDoc bool
}

// YAMLConfig configuration for YAML output
type YAMLConfig struct {
	FilePath      string        `json:"file"`
	Indent        int           `json:"indent"`
	ArrayFormat   string        `json:"array_format"`   // "flow" or "block"
	MapFormat     string        `json:"map_format"`     // "flow" or "block"
	MultiDocument bool          `json:"multi_document"` // Each record as separate YAML document
	BufferSize    int           `json:"buffer_size"`
	FlushInterval time.Duration `json:"flush_interval"`
	CompactArrays bool          `json:"compact_arrays"`
	SortKeys      bool          `json:"sort_keys"`
	IncludeNull   bool          `json:"include_null"`
	// Metadata configuration
	GeneratorName    string `json:"generator_name"`
	GeneratorVersion string `json:"generator_version"`
	IncludeMetadata  bool   `json:"include_metadata"`
	MetadataExplicit bool   `json:"-"` // Internal flag to track if metadata inclusion was explicitly set
}

// NewYAMLWriterWithExplicitMetadata creates a new YAML writer with explicit metadata configuration
func NewYAMLWriterWithExplicitMetadata(config YAMLConfig, includeMetadata bool) (*YAMLWriter, error) {
	config.IncludeMetadata = includeMetadata
	config.MetadataExplicit = true
	return NewYAMLWriter(config)
}

// NewYAMLWriter creates a new YAML writer
func NewYAMLWriter(config YAMLConfig) (*YAMLWriter, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("YAML file path is required")
	}

	// Set defaults
	if config.Indent == 0 {
		config.Indent = 2
	}
	if config.ArrayFormat == "" {
		config.ArrayFormat = "block"
	}
	if config.MapFormat == "" {
		config.MapFormat = "block"
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
	// Set metadata defaults
	if config.GeneratorName == "" {
		config.GeneratorName = DefaultGeneratorName
	}
	if config.GeneratorVersion == "" {
		config.GeneratorVersion = "1.0"
	}
	// Default: include metadata for traceability (set IncludeMetadata = false to disable)
	if !config.MetadataExplicit {
		config.IncludeMetadata = true
	}

	file, err := os.Create(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create YAML file: %w", err)
	}

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(config.Indent)

	writer := &YAMLWriter{
		file:       file,
		encoder:    encoder,
		config:     config,
		records:    make([]map[string]interface{}, 0, config.BufferSize),
		isFirstDoc: true,
	}

	return writer, nil
}

// Write writes data to YAML file
func (w *YAMLWriter) Write(data []map[string]interface{}) error {
	for _, record := range data {
		if err := w.WriteRecord(record); err != nil {
			return err
		}
	}
	return nil
}

// WriteRecord writes a single record to YAML
func (w *YAMLWriter) WriteRecord(record map[string]interface{}) error {
	if len(w.records) >= w.config.BufferSize {
		if err := w.flush(); err != nil {
			return err
		}
	}

	w.records = append(w.records, record)
	return nil
}

// WriteContext writes data to YAML file with context
func (w *YAMLWriter) WriteContext(ctx context.Context, data interface{}) error {
	switch v := data.(type) {
	case []map[string]interface{}:
		return w.Write(v)
	case map[string]interface{}:
		return w.WriteRecord(v)
	case []interface{}:
		for _, item := range v {
			if record, ok := item.(map[string]interface{}); ok {
				if err := w.WriteRecord(record); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unsupported data type in slice: %T", item)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported data type: %T", data)
	}
}

// Flush writes buffered records to file
func (w *YAMLWriter) Flush() error {
	return w.flush()
}

// Close closes the YAML writer and finalizes the file
func (w *YAMLWriter) Close() error {
	// Flush any remaining records
	if err := w.flush(); err != nil {
		return err
	}

	// Close encoder and file
	if err := w.encoder.Close(); err != nil {
		return err
	}

	return w.file.Close()
}

// GetType returns the output type
func (w *YAMLWriter) GetType() string {
	return "yaml"
}

// flush writes buffered records to the file
func (w *YAMLWriter) flush() error {
	if len(w.records) == 0 {
		return nil
	}

	if w.config.MultiDocument {
		// Write each record as a separate YAML document
		for _, record := range w.records {
			if err := w.writeDocument(record); err != nil {
				return err
			}
		}
	} else {
		// Write all records as a single YAML document (array)
		if err := w.writeArrayDocument(w.records); err != nil {
			return err
		}
	}

	w.records = w.records[:0] // Clear the slice but keep capacity
	return nil
}

// writeDocument writes a single YAML document
func (w *YAMLWriter) writeDocument(record map[string]interface{}) error {
	// Process the record based on configuration
	processedRecord := w.processRecord(record)

	// Write document separator if not the first document
	if !w.isFirstDoc {
		if _, err := w.file.WriteString("---\n"); err != nil {
			return err
		}
	}
	w.isFirstDoc = false

	return w.encoder.Encode(processedRecord)
}

// writeArrayDocument writes all records as a single YAML array document
func (w *YAMLWriter) writeArrayDocument(records []map[string]interface{}) error {
	processedRecords := make([]map[string]interface{}, len(records))
	for i, record := range records {
		processedRecords[i] = w.processRecord(record)
	}

	// Add configurable metadata if this is the first write
	if w.isFirstDoc && w.config.IncludeMetadata {
		document := map[string]interface{}{
			"metadata": map[string]interface{}{
				"generated_at": time.Now().Format(time.RFC3339),
				"generator":    w.config.GeneratorName,
				"version":      w.config.GeneratorVersion,
				"count":        len(processedRecords),
			},
			"data": processedRecords,
		}
		w.isFirstDoc = false
		return w.encoder.Encode(document)
	} else if w.isFirstDoc {
		// No metadata, just encode the data directly
		w.isFirstDoc = false
	}

	return w.encoder.Encode(processedRecords)
}

// processRecord processes a record according to configuration
func (w *YAMLWriter) processRecord(record map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})

	for key, value := range record {
		// Skip null values if not including them
		if !w.config.IncludeNull && value == nil {
			continue
		}

		// Process the value
		processedValue := w.processValue(value)
		processed[key] = processedValue
	}

	return processed
}

// processValue processes a value according to configuration
func (w *YAMLWriter) processValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case map[string]interface{}:
		processed := make(map[string]interface{})
		for key, val := range v {
			if !w.config.IncludeNull && val == nil {
				continue
			}
			processed[key] = w.processValue(val)
		}
		return processed

	case []interface{}:
		processed := make([]interface{}, len(v))
		for i, val := range v {
			processed[i] = w.processValue(val)
		}
		return processed

	case []map[string]interface{}:
		processed := make([]map[string]interface{}, len(v))
		for i, val := range v {
			if processedVal, ok := w.processValue(val).(map[string]interface{}); ok {
				processed[i] = processedVal
			} else {
				processed[i] = map[string]interface{}{"value": val}
			}
		}
		return processed

	case string:
		// Handle special string formatting
		return w.processString(v)

	case time.Time:
		// Format time consistently
		return v.Format(time.RFC3339)

	default:
		return value
	}
}

// processString processes string values
func (w *YAMLWriter) processString(s string) interface{} {
	// Handle multiline strings
	if strings.Contains(s, "\n") {
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Style: yaml.LiteralStyle, // Use literal style for multiline strings
			Value: s,
		}
	}

	// Handle strings that need quoting
	if w.needsQuoting(s) {
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Style: yaml.DoubleQuotedStyle,
			Value: s,
		}
	}

	return s
}

// needsQuoting determines if a string needs to be quoted in YAML
func (w *YAMLWriter) needsQuoting(s string) bool {
	// YAML special values that need quoting
	yamlSpecialValues := map[string]bool{
		"true": true, "false": true, "yes": true, "no": true,
		"on": true, "off": true, "null": true, "~": true,
		"True": true, "False": true, "Yes": true, "No": true,
		"On": true, "Off": true, "Null": true, "NULL": true,
		"TRUE": true, "FALSE": true, "YES": true, "NO": true,
		"ON": true, "OFF": true,
	}

	if yamlSpecialValues[s] {
		return true
	}

	// Check if it looks like a number
	if w.looksLikeNumber(s) {
		return true
	}

	// Check for special characters at the beginning
	if len(s) > 0 {
		first := s[0]
		if first == '-' || first == '?' || first == ':' || first == '[' ||
			first == ']' || first == '{' || first == '}' || first == '|' ||
			first == '>' || first == '*' || first == '&' || first == '!' ||
			first == '%' || first == '@' || first == '`' {
			return true
		}
	}

	// Check for strings that contain special sequences
	specialSequences := []string{"\\n", "\\t", "\\r", "\n", "\t", "\r"}
	for _, seq := range specialSequences {
		if strings.Contains(s, seq) {
			return true
		}
	}

	return false
}

// looksLikeNumber checks if a string looks like a number
func (w *YAMLWriter) looksLikeNumber(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Simple check for numeric patterns
	for i, r := range s {
		if i == 0 && (r == '-' || r == '+') {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '.' || r == 'e' || r == 'E' {
			continue
		}
		return false
	}

	return true
}

// YAMLDocument represents a structured YAML document
type YAMLDocument struct {
	Metadata YAMLMetadata             `yaml:"metadata"`
	Data     []map[string]interface{} `yaml:"data"`
}

// YAMLMetadata represents metadata for YAML documents
type YAMLMetadata struct {
	GeneratedAt time.Time `yaml:"generated_at"`
	Generator   string    `yaml:"generator"`
	Version     string    `yaml:"version"`
	Count       int       `yaml:"count"`
	Source      string    `yaml:"source,omitempty"`
	Tags        []string  `yaml:"tags,omitempty"`
}

// YAMLRecord represents a single record for YAML output
type YAMLRecord struct {
	ID        string                 `yaml:"id,omitempty"`
	Timestamp time.Time              `yaml:"timestamp,omitempty"`
	Data      map[string]interface{} `yaml:"data"`
	Metadata  map[string]interface{} `yaml:"metadata,omitempty"`
}

// StreamingYAMLWriter writes YAML documents as a stream
type StreamingYAMLWriter struct {
	file    *os.File
	encoder *yaml.Encoder
	config  YAMLConfig
	count   int
}

// NewStreamingYAMLWriter creates a new streaming YAML writer
func NewStreamingYAMLWriter(config YAMLConfig) (*StreamingYAMLWriter, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("YAML file path is required")
	}

	file, err := os.Create(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create YAML file: %w", err)
	}

	encoder := yaml.NewEncoder(file)
	if config.Indent > 0 {
		encoder.SetIndent(config.Indent)
	}

	return &StreamingYAMLWriter{
		file:    file,
		encoder: encoder,
		config:  config,
	}, nil
}

// WriteRecord writes a single record immediately
func (sw *StreamingYAMLWriter) WriteRecord(record map[string]interface{}) error {
	// Add document separator if not the first document
	if sw.count > 0 {
		if _, err := sw.file.WriteString("---\n"); err != nil {
			return err
		}
	}

	sw.count++
	return sw.encoder.Encode(record)
}

// WriteContext writes data with context
func (sw *StreamingYAMLWriter) WriteContext(ctx context.Context, data interface{}) error {
	switch v := data.(type) {
	case map[string]interface{}:
		return sw.WriteRecord(v)
	case []map[string]interface{}:
		for _, record := range v {
			if err := sw.WriteRecord(record); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported data type: %T", data)
	}
}

// Close closes the streaming writer
func (sw *StreamingYAMLWriter) Close() error {
	if err := sw.encoder.Close(); err != nil {
		return err
	}
	return sw.file.Close()
}

// GetType returns the output type
func (sw *StreamingYAMLWriter) GetType() string {
	return "yaml-stream"
}

// ValidateYAMLConfig validates YAML configuration
func ValidateYAMLConfig(config YAMLConfig) error {
	if config.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	if config.Indent < 0 || config.Indent > 10 {
		return fmt.Errorf("indent must be between 0 and 10")
	}

	if config.ArrayFormat != "" && config.ArrayFormat != "flow" && config.ArrayFormat != "block" {
		return fmt.Errorf("array_format must be 'flow' or 'block'")
	}

	if config.MapFormat != "" && config.MapFormat != "flow" && config.MapFormat != "block" {
		return fmt.Errorf("map_format must be 'flow' or 'block'")
	}

	if config.BufferSize < 0 {
		return fmt.Errorf("buffer size must be non-negative")
	}

	return nil
}
