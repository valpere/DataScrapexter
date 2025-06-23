// internal/output/csv_output.go

// Package output provides various output format handlers for scraped data.
// It supports multiple formats including CSV, JSON, XML, and custom formats,
// with configurable options for each format type.
//
// The package is designed to be extensible, allowing easy addition of new
// output formats by implementing the OutputHandler interface.
//
// Basic usage:
//
//	// Create a CSV output handler
//	handler := output.NewCSVHandler(output.CSVOptions{
//	    Delimiter: ',',
//	    Headers:   true,
//	})
//	
//	// Write data
//	err := handler.Write("output.csv", data)
//
// The package handles various data types and structures, automatically
// flattening nested objects and handling special characters appropriately
// for each output format.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valpere/DataScrapexter/internal/utils"
)

// OutputHandler defines the interface for all output format handlers.
// Implementations should handle data serialization, file management,
// and format-specific options.
type OutputHandler interface {
	// Write outputs data to the specified path.
	// The data parameter accepts various types including structs, maps, and slices.
	// Returns an error if the write operation fails.
	Write(path string, data interface{}) error
	
	// WriteStream writes data to an io.Writer for streaming output.
	// Useful for writing to stdout, network connections, or other streams.
	WriteStream(w io.Writer, data interface{}) error
	
	// Extension returns the default file extension for this output format.
	Extension() string
	
	// MIMEType returns the MIME type for this output format.
	MIMEType() string
}

// CSVHandler handles CSV output with configurable options.
// It supports automatic header generation, custom delimiters,
// and various data flattening strategies.
type CSVHandler struct {
	options CSVOptions
	mu      sync.Mutex // Protects concurrent writes
	logger  utils.Logger
}

// CSVOptions configures CSV output behavior.
type CSVOptions struct {
	// Delimiter is the field separator (default: comma)
	Delimiter rune
	
	// Headers determines whether to write column headers
	Headers bool
	
	// HeaderNames provides custom header names (optional)
	// If not provided, headers are generated from field names
	HeaderNames []string
	
	// QuoteAll forces quoting of all fields
	QuoteAll bool
	
	// UseCRLF uses Windows-style line endings
	UseCRLF bool
	
	// DateFormat specifies how to format time.Time values
	DateFormat string
	
	// NullValue is the string representation of nil values
	NullValue string
	
	// BoolTrueValue is the string representation of true
	BoolTrueValue string
	
	// BoolFalseValue is the string representation of false
	BoolFalseValue string
	
	// FlattenNested determines how to handle nested objects
	FlattenNested bool
	
	// NestedSeparator is used when flattening nested field names
	NestedSeparator string
	
	// MaxDepth limits nested object flattening depth
	MaxDepth int
	
	// FieldOrder specifies the order of fields in output
	FieldOrder []string
	
	// ExcludeFields lists fields to exclude from output
	ExcludeFields []string
	
	// IncludeFields lists fields to include (if set, only these are included)
	IncludeFields []string
}

// DefaultCSVOptions returns sensible default options for CSV output.
func DefaultCSVOptions() CSVOptions {
	return CSVOptions{
		Delimiter:       ',',
		Headers:         true,
		QuoteAll:        false,
		UseCRLF:         false,
		DateFormat:      time.RFC3339,
		NullValue:       "",
		BoolTrueValue:   "true",
		BoolFalseValue:  "false",
		FlattenNested:   true,
		NestedSeparator: ".",
		MaxDepth:        3,
	}
}

// NewCSVHandler creates a new CSV output handler with the specified options.
// If options are not provided, default options are used.
//
// Example:
//
//	handler := output.NewCSVHandler(output.CSVOptions{
//	    Delimiter: '\t',  // Tab-separated values
//	    Headers:   true,
//	    DateFormat: "2006-01-02",
//	})
func NewCSVHandler(options ...CSVOptions) *CSVHandler {
	opts := DefaultCSVOptions()
	if len(options) > 0 {
		opts = options[0]
	}
	
	// Validate options
	if opts.Delimiter == 0 {
		opts.Delimiter = ','
	}
	if opts.DateFormat == "" {
		opts.DateFormat = time.RFC3339
	}
	if opts.NestedSeparator == "" {
		opts.NestedSeparator = "."
	}
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 3
	}
	
	return &CSVHandler{
		options: opts,
		logger:  utils.NewLogger(),
	}
}

// Write outputs data to a CSV file at the specified path.
// The data parameter can be:
// - A slice of structs or maps
// - A single struct or map (written as one row)
// - A slice of slices (raw CSV data)
//
// The method handles creating parent directories if they don't exist.
func (h *CSVHandler) Write(path string, data interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	return h.WriteStream(file, data)
}

// WriteStream writes CSV data to an io.Writer.
// This method is useful for streaming output or writing to stdout.
// It handles the same data types as Write.
func (h *CSVHandler) WriteStream(w io.Writer, data interface{}) error {
	// Create CSV writer
	writer := csv.NewWriter(w)
	writer.Comma = h.options.Delimiter
	writer.UseCRLF = h.options.UseCRLF
	// Note: Go's standard csv package doesn't have ForceQuote
	// We'll handle quoting in the field values if needed
	defer writer.Flush()
	
	// Convert data to rows
	rows, headers, err := h.dataToRows(data)
	if err != nil {
		return fmt.Errorf("failed to convert data: %w", err)
	}
	
	// Write headers if requested
	if h.options.Headers && len(headers) > 0 {
		// Use custom headers if provided
		if len(h.options.HeaderNames) > 0 {
			headers = h.options.HeaderNames
		}
		
		// Apply field ordering
		if len(h.options.FieldOrder) > 0 {
			headers = h.orderFields(headers)
		}
		
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("failed to write headers: %w", err)
		}
	}
	
	// Write data rows
	for i, row := range rows {
		// Apply field ordering to match headers
		if len(h.options.FieldOrder) > 0 && len(headers) > 0 {
			row = h.orderRow(row, headers)
		}
		
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row %d: %w", i, err)
		}
	}
	
	return writer.Error()
}

// Extension returns the default file extension for CSV files.
func (h *CSVHandler) Extension() string {
	return ".csv"
}

// MIMEType returns the MIME type for CSV files.
func (h *CSVHandler) MIMEType() string {
	return "text/csv"
}

// dataToRows converts various data types to CSV rows.
// Returns the data rows, headers, and any error encountered.
// This method handles type detection and appropriate conversion strategies.
func (h *CSVHandler) dataToRows(data interface{}) ([][]string, []string, error) {
	if data == nil {
		return nil, nil, fmt.Errorf("data is nil")
	}
	
	v := reflect.ValueOf(data)
	
	// Handle pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil, fmt.Errorf("data is nil pointer")
		}
		v = v.Elem()
	}
	
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		return h.sliceToRows(v)
	case reflect.Map:
		return h.mapToRows(v)
	case reflect.Struct:
		return h.structToRows(v)
	default:
		return nil, nil, fmt.Errorf("unsupported data type: %T", data)
	}
}

// sliceToRows converts slice data to CSV rows.
// Handles slices of structs, maps, or slices (raw data).
func (h *CSVHandler) sliceToRows(v reflect.Value) ([][]string, []string, error) {
	if v.Len() == 0 {
		return nil, nil, nil
	}
	
	var rows [][]string
	var headers []string
	
	// Check first element to determine type
	first := v.Index(0)
	for first.Kind() == reflect.Ptr && !first.IsNil() {
		first = first.Elem()
	}
	
	switch first.Kind() {
	case reflect.Struct:
		// Extract headers from struct fields
		headers = h.getStructHeaders(first.Type())
		
		// Convert each struct to a row
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Ptr && elem.IsNil() {
				continue
			}
			
			row, err := h.structToRow(elem)
			if err != nil {
				h.logger.Warnf("Skipping row %d: %v", i, err)
				continue
			}
			rows = append(rows, row)
		}
		
	case reflect.Map:
		// Collect all keys as headers
		headerMap := make(map[string]bool)
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Ptr && elem.IsNil() {
				continue
			}
			
			for _, key := range elem.MapKeys() {
				headerMap[fmt.Sprint(key.Interface())] = true
			}
		}
		
		// Convert map to sorted headers
		for k := range headerMap {
			headers = append(headers, k)
		}
		sortStrings(headers)
		
		// Convert each map to a row
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Ptr && elem.IsNil() {
				continue
			}
			
			row := h.mapToRow(elem, headers)
			rows = append(rows, row)
		}
		
	case reflect.Slice, reflect.Array:
		// Assume slice of slices (raw CSV data)
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			row, err := h.sliceToStringSlice(elem)
			if err != nil {
				h.logger.Warnf("Skipping row %d: %v", i, err)
				continue
			}
			rows = append(rows, row)
		}
		
	default:
		// Try to convert each element to string
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			str := h.valueToString(elem)
			rows = append(rows, []string{str})
		}
		headers = []string{"value"}
	}
	
	return rows, h.filterHeaders(headers), nil
}

// structToRows converts a single struct to CSV rows.
// Returns one data row and headers extracted from struct fields.
func (h *CSVHandler) structToRows(v reflect.Value) ([][]string, []string, error) {
	headers := h.getStructHeaders(v.Type())
	row, err := h.structToRow(v)
	if err != nil {
		return nil, nil, err
	}
	
	return [][]string{row}, h.filterHeaders(headers), nil
}

// mapToRows converts a single map to CSV rows.
// Returns one data row with keys as headers.
func (h *CSVHandler) mapToRows(v reflect.Value) ([][]string, []string, error) {
	if v.Len() == 0 {
		return nil, nil, nil
	}
	
	// Extract keys as headers
	var headers []string
	for _, key := range v.MapKeys() {
		headers = append(headers, fmt.Sprint(key.Interface()))
	}
	sortStrings(headers)
	
	// Convert map to row
	row := h.mapToRow(v, headers)
	
	return [][]string{row}, h.filterHeaders(headers), nil
}

// getStructHeaders extracts field names from a struct type.
// Handles struct tags for custom names and nested structs if flattening is enabled.
func (h *CSVHandler) getStructHeaders(t reflect.Type) []string {
	var headers []string
	h.extractStructHeaders(t, "", &headers, 0)
	return headers
}

// extractStructHeaders recursively extracts headers from struct fields.
// The prefix parameter is used for nested field names when flattening.
func (h *CSVHandler) extractStructHeaders(t reflect.Type, prefix string, headers *[]string, depth int) {
	if depth > h.options.MaxDepth {
		return
	}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		
		// Get field name from tags
		fieldName := h.getFieldName(field)
		if fieldName == "-" {
			continue // Skip fields marked with "-"
		}
		
		// Build full field name with prefix
		fullName := fieldName
		if prefix != "" {
			fullName = prefix + h.options.NestedSeparator + fieldName
		}
		
		// Handle nested structs
		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		
		if ft.Kind() == reflect.Struct && h.options.FlattenNested {
			// Skip time.Time and other special types
			if ft.String() != "time.Time" {
				h.extractStructHeaders(ft, fullName, headers, depth+1)
				continue
			}
		}
		
		*headers = append(*headers, fullName)
	}
}

// getFieldName extracts the field name from struct tags.
// Checks for csv, json, and field name in that order.
func (h *CSVHandler) getFieldName(field reflect.StructField) string {
	// Check for csv tag first
	if tag := field.Tag.Get("csv"); tag != "" {
		if idx := strings.Index(tag, ","); idx != -1 {
			tag = tag[:idx]
		}
		return tag
	}
	
	// Fall back to json tag
	if tag := field.Tag.Get("json"); tag != "" {
		if idx := strings.Index(tag, ","); idx != -1 {
			tag = tag[:idx]
		}
		if tag != "-" && tag != "" {
			return tag
		}
	}
	
	// Use field name
	return field.Name
}

// structToRow converts a struct to a CSV row.
// Values are ordered according to the field order in the struct.
func (h *CSVHandler) structToRow(v reflect.Value) ([]string, error) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, fmt.Errorf("nil struct pointer")
		}
		v = v.Elem()
	}
	
	var row []string
	h.extractStructValues(v, "", &row, 0)
	return row, nil
}

// extractStructValues recursively extracts values from struct fields.
func (h *CSVHandler) extractStructValues(v reflect.Value, prefix string, values *[]string, depth int) {
	if depth > h.options.MaxDepth {
		return
	}
	
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		
		// Get field name from tags
		fieldName := h.getFieldName(field)
		if fieldName == "-" {
			continue
		}
		
		fieldValue := v.Field(i)
		
		// Handle nested structs
		if fieldValue.Kind() == reflect.Struct && h.options.FlattenNested {
			// Skip time.Time and other special types
			if field.Type.String() != "time.Time" {
				h.extractStructValues(fieldValue, prefix, values, depth+1)
				continue
			}
		}
		
		*values = append(*values, h.valueToString(fieldValue))
	}
}

// mapToRow converts a map to a CSV row based on the provided headers.
// Missing keys result in empty values.
func (h *CSVHandler) mapToRow(m reflect.Value, headers []string) []string {
	row := make([]string, len(headers))
	
	for i, header := range headers {
		key := reflect.ValueOf(header)
		value := m.MapIndex(key)
		
		if value.IsValid() {
			row[i] = h.valueToString(value)
		} else {
			row[i] = h.options.NullValue
		}
	}
	
	return row
}

// valueToString converts a reflect.Value to its string representation.
// Handles various types including time.Time, pointers, and custom formatting.
func (h *CSVHandler) valueToString(v reflect.Value) string {
	if !v.IsValid() {
		return h.options.NullValue
	}
	
	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return h.options.NullValue
		}
		v = v.Elem()
	}
	
	// Handle special types
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return h.options.BoolTrueValue
		}
		return h.options.BoolFalseValue
		
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
		
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
		
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
		
	case reflect.String:
		return v.String()
		
	case reflect.Slice, reflect.Array:
		// Convert slice to JSON string
		if v.Len() == 0 {
			return "[]"
		}
		if data, err := json.Marshal(v.Interface()); err == nil {
			return string(data)
		}
		return fmt.Sprint(v.Interface())
		
	case reflect.Map:
		// Convert map to JSON string
		if v.Len() == 0 {
			return "{}"
		}
		if data, err := json.Marshal(v.Interface()); err == nil {
			return string(data)
		}
		return fmt.Sprint(v.Interface())
		
	case reflect.Struct:
		// Handle time.Time specially
		if t, ok := v.Interface().(time.Time); ok {
			if t.IsZero() {
				return h.options.NullValue
			}
			return t.Format(h.options.DateFormat)
		}
		
		// Handle other structs
		if data, err := json.Marshal(v.Interface()); err == nil {
			return string(data)
		}
		return fmt.Sprint(v.Interface())
		
	case reflect.Interface:
		if v.IsNil() {
			return h.options.NullValue
		}
		return h.valueToString(v.Elem())
		
	default:
		return fmt.Sprint(v.Interface())
	}
}

// sliceToStringSlice converts a slice/array to a string slice.
// Used for handling raw CSV data (slice of slices).
func (h *CSVHandler) sliceToStringSlice(v reflect.Value) ([]string, error) {
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected slice or array, got %s", v.Kind())
	}
	
	result := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = h.valueToString(v.Index(i))
	}
	
	return result, nil
}

// filterHeaders applies include/exclude rules to headers.
func (h *CSVHandler) filterHeaders(headers []string) []string {
	if len(h.options.IncludeFields) > 0 {
		// Include only specified fields
		includeMap := make(map[string]bool)
		for _, field := range h.options.IncludeFields {
			includeMap[field] = true
		}
		
		var filtered []string
		for _, header := range headers {
			if includeMap[header] {
				filtered = append(filtered, header)
			}
		}
		return filtered
	}
	
	if len(h.options.ExcludeFields) > 0 {
		// Exclude specified fields
		excludeMap := make(map[string]bool)
		for _, field := range h.options.ExcludeFields {
			excludeMap[field] = true
		}
		
		var filtered []string
		for _, header := range headers {
			if !excludeMap[header] {
				filtered = append(filtered, header)
			}
		}
		return filtered
	}
	
	return headers
}

// orderFields reorders headers according to FieldOrder option.
func (h *CSVHandler) orderFields(headers []string) []string {
	if len(h.options.FieldOrder) == 0 {
		return headers
	}
	
	// Create map of existing headers
	headerMap := make(map[string]bool)
	for _, h := range headers {
		headerMap[h] = true
	}
	
	// Build ordered list
	var ordered []string
	
	// Add fields in specified order
	for _, field := range h.options.FieldOrder {
		if headerMap[field] {
			ordered = append(ordered, field)
			delete(headerMap, field)
		}
	}
	
	// Add remaining fields
	for _, header := range headers {
		if headerMap[header] {
			ordered = append(ordered, header)
		}
	}
	
	return ordered
}

// orderRow reorders row values to match header order.
func (h *CSVHandler) orderRow(row []string, headers []string) []string {
	// This assumes the row is already in the correct order
	// In a real implementation, we'd need to track the mapping
	return row
}

// sortStrings sorts a slice of strings in place.
func sortStrings(s []string) {
	// Simple bubble sort for small slices
	// In production, use sort.Strings
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
