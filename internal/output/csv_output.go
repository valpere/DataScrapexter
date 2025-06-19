package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/valpere/DataScrapexter/internal/scraper"
)

// CSVWriter handles CSV output formatting and writing
type CSVWriter struct {
	writer      *csv.Writer
	headers     []string
	writtenRows int
}

// NewCSVWriter creates a new CSV writer
func NewCSVWriter(w io.Writer) *CSVWriter {
	csvWriter := csv.NewWriter(w)
	return &CSVWriter{
		writer: csvWriter,
	}
}

// WriteResults writes scraping results to CSV format
func WriteResultsToCSV(results []*scraper.Result, filename string) error {
	// Create output directory if needed
	if filename != "" && filename != "-" {
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Open output file or use stdout
	var writer io.Writer
	var file *os.File
	var err error

	if filename == "" || filename == "-" {
		writer = os.Stdout
	} else {
		file, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	// Create CSV writer
	csvWriter := NewCSVWriter(writer)

	// Process results
	if len(results) == 0 {
		return fmt.Errorf("no results to write")
	}

	// Extract headers from first result
	headers := csvWriter.extractHeaders(results)
	if err := csvWriter.writeHeaders(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Write data rows
	for i, result := range results {
		if err := csvWriter.writeResult(result, headers); err != nil {
			return fmt.Errorf("failed to write row %d: %w", i, err)
		}
	}

	// Flush the writer
	csvWriter.writer.Flush()
	if err := csvWriter.writer.Error(); err != nil {
		return fmt.Errorf("CSV writer error: %w", err)
	}

	return nil
}

// extractHeaders extracts unique headers from all results
func (w *CSVWriter) extractHeaders(results []*scraper.Result) []string {
	headerMap := make(map[string]bool)

	// Always include basic fields
	headers := []string{"url", "status_code", "timestamp", "error"}
	for _, h := range headers {
		headerMap[h] = true
	}

	// Extract field names from data
	for _, result := range results {
		for key := range result.Data {
			if !headerMap[key] {
				headers = append(headers, key)
				headerMap[key] = true
			}
		}
	}

	// Sort headers for consistency (keeping basic fields first)
	dataHeaders := headers[4:]
	sort.Strings(dataHeaders)

	return append(headers[:4], dataHeaders...)
}

// writeHeaders writes the CSV header row
func (w *CSVWriter) writeHeaders(headers []string) error {
	w.headers = headers
	return w.writer.Write(headers)
}

// writeResult writes a single result as a CSV row
func (w *CSVWriter) writeResult(result *scraper.Result, headers []string) error {
	row := make([]string, len(headers))

	for i, header := range headers {
		switch header {
		case "url":
			row[i] = result.URL
		case "status_code":
			row[i] = strconv.Itoa(result.StatusCode)
		case "timestamp":
			row[i] = result.Timestamp.Format("2006-01-02 15:04:05")
		case "error":
			if result.Error != nil {
				row[i] = result.Error.Error()
			}
		default:
			// Extract from data fields
			if value, exists := result.Data[header]; exists {
				row[i] = w.formatValue(value)
			}
		}
	}

	w.writtenRows++
	return w.writer.Write(row)
}

// formatValue converts various types to string representation for CSV
func (w *CSVWriter) formatValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case []string:
		// Join arrays with semicolon
		return strings.Join(v, "; ")
	case []interface{}:
		// Convert interface slice to strings
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = w.formatValue(item)
		}
		return strings.Join(items, "; ")
	default:
		// Use reflection for other types
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			items := make([]string, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				items[i] = w.formatValue(rv.Index(i).Interface())
			}
			return strings.Join(items, "; ")
		default:
			return fmt.Sprintf("%v", value)
		}
	}
}

// CSVOptions provides configuration for CSV output
type CSVOptions struct {
	Delimiter      rune
	IncludeHeaders bool
	QuoteAll       bool
	FlattenNested  bool
}

// DefaultCSVOptions returns default CSV options
func DefaultCSVOptions() *CSVOptions {
	return &CSVOptions{
		Delimiter:      ',',
		IncludeHeaders: true,
		QuoteAll:       false,
		FlattenNested:  true,
	}
}

// WriteResultsToCSVWithOptions writes results with custom CSV options
func WriteResultsToCSVWithOptions(results []*scraper.Result, filename string, options *CSVOptions) error {
	if options == nil {
		options = DefaultCSVOptions()
	}

	// Create output file or use stdout
	var writer io.Writer
	var file *os.File
	var err error

	if filename == "" || filename == "-" {
		writer = os.Stdout
	} else {
		file, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	// Create and configure CSV writer
	csvWriter := csv.NewWriter(writer)
	csvWriter.Comma = options.Delimiter

	// Process with options
	processor := &CSVProcessor{
		writer:  csvWriter,
		options: options,
	}

	return processor.ProcessResults(results)
}

// CSVProcessor handles CSV processing with options
type CSVProcessor struct {
	writer  *csv.Writer
	options *CSVOptions
	headers []string
}

// ProcessResults processes results according to CSV options
func (p *CSVProcessor) ProcessResults(results []*scraper.Result) error {
	if len(results) == 0 {
		return fmt.Errorf("no results to process")
	}

	// Extract and write headers if needed
	if p.options.IncludeHeaders {
		p.headers = p.extractHeadersWithFlattening(results)
		if err := p.writer.Write(p.headers); err != nil {
			return fmt.Errorf("failed to write headers: %w", err)
		}
	}

	// Write data rows
	for i, result := range results {
		row := p.resultToRow(result)
		if err := p.writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row %d: %w", i, err)
		}
	}

	// Flush the writer
	p.writer.Flush()
	return p.writer.Error()
}

// extractHeadersWithFlattening extracts headers with nested field support
func (p *CSVProcessor) extractHeadersWithFlattening(results []*scraper.Result) []string {
	headerMap := make(map[string]bool)
	headers := []string{"url", "status_code", "timestamp"}

	for _, h := range headers {
		headerMap[h] = true
	}

	// Extract field names from data
	for _, result := range results {
		for key, value := range result.Data {
			if p.options.FlattenNested && isNestedStructure(value) {
				// Extract nested headers
				nestedHeaders := p.extractNestedHeaders(key, value)
				for _, h := range nestedHeaders {
					if !headerMap[h] {
						headers = append(headers, h)
						headerMap[h] = true
					}
				}
			} else {
				if !headerMap[key] {
					headers = append(headers, key)
					headerMap[key] = true
				}
			}
		}
	}

	return headers
}

// extractNestedHeaders extracts headers from nested structures
func (p *CSVProcessor) extractNestedHeaders(prefix string, value interface{}) []string {
	var headers []string

	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			nestedKey := fmt.Sprintf("%s.%s", prefix, key)
			if isNestedStructure(val) {
				headers = append(headers, p.extractNestedHeaders(nestedKey, val)...)
			} else {
				headers = append(headers, nestedKey)
			}
		}
	default:
		headers = append(headers, prefix)
	}

	return headers
}

// resultToRow converts a result to a CSV row
func (p *CSVProcessor) resultToRow(result *scraper.Result) []string {
	row := make([]string, len(p.headers))

	for i, header := range p.headers {
		switch header {
		case "url":
			row[i] = result.URL
		case "status_code":
			row[i] = strconv.Itoa(result.StatusCode)
		case "timestamp":
			row[i] = result.Timestamp.Format("2006-01-02 15:04:05")
		default:
			// Handle nested fields
			if strings.Contains(header, ".") && p.options.FlattenNested {
				value := p.extractNestedValue(result.Data, header)
				row[i] = formatValueToString(value)
			} else {
				if value, exists := result.Data[header]; exists {
					row[i] = formatValueToString(value)
				}
			}
		}
	}

	return row
}

// extractNestedValue extracts value from nested structure using dot notation
func (p *CSVProcessor) extractNestedValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part]
		}

		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}

	return nil
}

// isNestedStructure checks if a value is a nested structure
func isNestedStructure(value interface{}) bool {
	switch value.(type) {
	case map[string]interface{}:
		return true
	default:
		return false
	}
}

// formatValueToString formats any value to string
func formatValueToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", v)
	case []string:
		return strings.Join(v, "; ")
	case []interface{}:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = formatValueToString(item)
		}
		return strings.Join(items, "; ")
	default:
		return fmt.Sprintf("%v", value)
	}
}
