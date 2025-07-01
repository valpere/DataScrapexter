// internal/output/csv.go
package output

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
)

// CSVWriter writes data in CSV format
type CSVWriter struct {
	filename string
	file     *os.File
	writer   *csv.Writer
}

// NewCSVWriter creates a new CSV writer
func NewCSVWriter(filename string) (*CSVWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	writer := csv.NewWriter(file)

	return &CSVWriter{
		filename: filename,
		file:     file,
		writer:   writer,
	}, nil
}

// Write writes data to CSV file
func (w *CSVWriter) Write(data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// Get all unique field names
	fieldSet := make(map[string]bool)
	for _, row := range data {
		for field := range row {
			fieldSet[field] = true
		}
	}

	// Convert to sorted slice for consistent column order
	var fields []string
	for field := range fieldSet {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	// Write header
	if err := w.writer.Write(fields); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write data rows
	for _, row := range data {
		var record []string
		for _, field := range fields {
			value := ""
			if val, exists := row[field]; exists && val != nil {
				value = fmt.Sprintf("%v", val)
			}
			record = append(record, value)
		}
		if err := w.writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	w.writer.Flush()
	return w.writer.Error()
}

// WriteRecord writes a single record to CSV file
func (w *CSVWriter) WriteRecord(record map[string]interface{}) error {
	return w.Write([]map[string]interface{}{record})
}

// Flush flushes any buffered data to the underlying writer
func (w *CSVWriter) Flush() error {
	if w.writer != nil {
		w.writer.Flush()
		return w.writer.Error()
	}
	return nil
}

// Close closes the CSV writer
func (w *CSVWriter) Close() error {
	if w.writer != nil {
		w.writer.Flush()
		w.writer = nil
	}
	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}
