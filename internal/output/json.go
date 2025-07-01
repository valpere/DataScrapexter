// internal/output/json.go
package output

import (
	"encoding/json"
	"os"
)

// JSONWriter writes data in JSON format
type JSONWriter struct {
	filename string
	file     *os.File
}

// NewJSONWriter creates a new JSON writer
func NewJSONWriter(filename string) (*JSONWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	return &JSONWriter{
		filename: filename,
		file:     file,
	}, nil
}

// Write writes data to JSON file
func (w *JSONWriter) Write(data []map[string]interface{}) error {
	encoder := json.NewEncoder(w.file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// WriteRecord writes a single record to JSON file
func (w *JSONWriter) WriteRecord(record map[string]interface{}) error {
	encoder := json.NewEncoder(w.file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(record)
}

// Flush flushes any buffered data to the underlying writer
func (w *JSONWriter) Flush() error {
	if w.file != nil {
		return w.file.Sync()
	}
	return nil
}

// Close closes the JSON writer
func (w *JSONWriter) Close() error {
	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}
