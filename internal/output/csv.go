// internal/output/csv.go
package output

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// CSVWriterConfig configures CSV output behavior
type CSVWriterConfig struct {
	FilePath      string `yaml:"file_path" json:"file_path"`
	Delimiter     rune   `yaml:"delimiter" json:"delimiter"`
	Headers       bool   `yaml:"headers" json:"headers"`
	AppendMode    bool   `yaml:"append_mode" json:"append_mode"`
	BufferSize    int    `yaml:"buffer_size" json:"buffer_size"`
	FieldOrder    []string `yaml:"field_order" json:"field_order"`
	AutoDetectFields bool `yaml:"auto_detect_fields" json:"auto_detect_fields"`
}

// CSVWriter handles CSV output with dynamic field handling
type CSVWriter struct {
	config     *CSVWriterConfig
	file       *os.File
	bufferedWriter *bufio.Writer
	csvWriter  *csv.Writer
	mutex      sync.Mutex
	headers    []string
	headersWritten bool
	rowCount   int64
}

// NewCSVWriter creates a new CSV output writer
func NewCSVWriter(config *CSVWriterConfig) (*CSVWriter, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Set defaults
	if config.Delimiter == 0 {
		config.Delimiter = ','
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 8192
	}

	// Create directory if needed
	dir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file
	var file *os.File
	var err error
	
	if config.AppendMode {
		file, err = os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		file, err = os.Create(config.FilePath)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Create buffered writer and CSV writer
	bufferedWriter := bufio.NewWriterSize(file, config.BufferSize)
	csvWriter := csv.NewWriter(bufferedWriter)
	csvWriter.Comma = config.Delimiter

	cw := &CSVWriter{
		config:         config,
		file:           file,
		bufferedWriter: bufferedWriter,
		csvWriter:      csvWriter,
		headers:        config.FieldOrder,
	}

	// If appending to existing file, assume headers already written
	if config.AppendMode {
		info, err := file.Stat()
		if err == nil && info.Size() > 0 {
			cw.headersWritten = true
		}
	}

	return cw, nil
}

// WriteData writes a single data record
func (cw *CSVWriter) WriteData(data map[string]interface{}) error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	// Auto-detect fields if needed
	if cw.config.AutoDetectFields && len(cw.headers) == 0 {
		cw.detectFields(data)
	}

	// Write headers if not written yet
	if cw.config.Headers && !cw.headersWritten {
		if err := cw.writeHeaders(); err != nil {
			return err
		}
	}

	// Convert data to row
	row := cw.dataToRow(data)
	
	if err := cw.csvWriter.Write(row); err != nil {
		return fmt.Errorf("failed to write CSV row: %w", err)
	}

	cw.rowCount++
	return nil
}

// WriteBatch writes multiple data records
func (cw *CSVWriter) WriteBatch(items []map[string]interface{}) error {
	for _, item := range items {
		if err := cw.WriteData(item); err != nil {
			return err
		}
	}
	return nil
}

// WriteHeaders explicitly writes header row
func (cw *CSVWriter) WriteHeaders(headers []string) error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	if cw.headersWritten {
		return fmt.Errorf("headers already written")
	}

	cw.headers = headers
	return cw.writeHeaders()
}

// writeHeaders writes the header row (internal)
func (cw *CSVWriter) writeHeaders() error {
	if len(cw.headers) == 0 {
		return fmt.Errorf("no headers to write")
	}

	if err := cw.csvWriter.Write(cw.headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	cw.headersWritten = true
	return nil
}

// detectFields auto-detects field names from data
func (cw *CSVWriter) detectFields(data map[string]interface{}) {
	if len(cw.headers) > 0 {
		return
	}

	// Get all keys and sort for consistent order
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	
	cw.headers = keys
}

// dataToRow converts map data to CSV row
func (cw *CSVWriter) dataToRow(data map[string]interface{}) []string {
	row := make([]string, len(cw.headers))
	
	for i, header := range cw.headers {
		if value, exists := data[header]; exists {
			row[i] = cw.formatValue(value)
		}
		// Empty string for missing values
	}
	
	return row
}

// formatValue converts interface{} to string for CSV
func (cw *CSVWriter) formatValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []interface{}:
		// Convert array to comma-separated string
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = cw.formatValue(item)
		}
		return strings.Join(parts, "|")
	default:
		// Handle complex types
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			parts := make([]string, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				parts[i] = cw.formatValue(rv.Index(i).Interface())
			}
			return strings.Join(parts, "|")
		case reflect.Map:
			// Convert map to key=value pairs
			parts := make([]string, 0, rv.Len())
			for _, key := range rv.MapKeys() {
				keyStr := cw.formatValue(key.Interface())
				valStr := cw.formatValue(rv.MapIndex(key).Interface())
				parts = append(parts, fmt.Sprintf("%s=%s", keyStr, valStr))
			}
			sort.Strings(parts) // Consistent order
			return strings.Join(parts, "|")
		default:
			return fmt.Sprintf("%v", value)
		}
	}
}

// AddField dynamically adds a new field to the schema
func (cw *CSVWriter) AddField(fieldName string) error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	if cw.headersWritten {
		return fmt.Errorf("cannot add field after headers are written")
	}

	// Check if field already exists
	for _, header := range cw.headers {
		if header == fieldName {
			return nil // Already exists
		}
	}

	cw.headers = append(cw.headers, fieldName)
	return nil
}

// GetHeaders returns current headers
func (cw *CSVWriter) GetHeaders() []string {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	
	result := make([]string, len(cw.headers))
	copy(result, cw.headers)
	return result
}

// Flush forces write of buffered data
func (cw *CSVWriter) Flush() error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	cw.csvWriter.Flush()
	if err := cw.csvWriter.Error(); err != nil {
		return fmt.Errorf("CSV writer error: %w", err)
	}

	if err := cw.bufferedWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}

	return cw.file.Sync()
}

// Close closes the writer and file
func (cw *CSVWriter) Close() error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	cw.csvWriter.Flush()
	if err := cw.csvWriter.Error(); err != nil {
		return fmt.Errorf("CSV writer error on close: %w", err)
	}

	if err := cw.bufferedWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush on close: %w", err)
	}

	if err := cw.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}

// GetStats returns writer statistics
func (cw *CSVWriter) GetStats() map[string]interface{} {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	return map[string]interface{}{
		"rows_written":    cw.rowCount,
		"file_path":       cw.config.FilePath,
		"headers":         cw.headers,
		"headers_written": cw.headersWritten,
		"field_count":     len(cw.headers),
		"delimiter":       string(cw.config.Delimiter),
	}
}

// ValidateCSVConfig validates CSV writer configuration
func ValidateCSVConfig(config *CSVWriterConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	// Validate delimiter
	if config.Delimiter != 0 {
		switch config.Delimiter {
		case ',', ';', '\t', '|':
			// Valid delimiters
		default:
			return fmt.Errorf("invalid delimiter: %c", config.Delimiter)
		}
	}

	return nil
}

// EnsureCSVExtension ensures the file has .csv extension
func EnsureCSVExtension(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext != ".csv" {
		return filePath + ".csv"
	}
	return filePath
}

// ReadCSV reads CSV file and returns data as slice of maps
func ReadCSV(filePath string, delimiter rune) ([]map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = delimiter

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return []map[string]interface{}{}, nil
	}

	// First row as headers
	headers := records[0]
	data := make([]map[string]interface{}, 0, len(records)-1)

	for i := 1; i < len(records); i++ {
		row := make(map[string]interface{})
		for j, value := range records[i] {
			if j < len(headers) {
				// Try to parse as number or keep as string
				if num, err := strconv.ParseFloat(value, 64); err == nil {
					if num == float64(int64(num)) {
						row[headers[j]] = int64(num)
					} else {
						row[headers[j]] = num
					}
				} else if value == "true" {
					row[headers[j]] = true
				} else if value == "false" {
					row[headers[j]] = false
				} else {
					row[headers[j]] = value
				}
			}
		}
		data = append(data, row)
	}

	return data, nil
}
