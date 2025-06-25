// internal/output/json.go
package output

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JSONWriterConfig configures JSON output behavior
type JSONWriterConfig struct {
	FilePath     string `yaml:"file_path" json:"file_path"`
	Indent       string `yaml:"indent" json:"indent"`
	AppendMode   bool   `yaml:"append_mode" json:"append_mode"`
	StreamMode   bool   `yaml:"stream_mode" json:"stream_mode"`
	BufferSize   int    `yaml:"buffer_size" json:"buffer_size"`
	SyncInterval int    `yaml:"sync_interval" json:"sync_interval"` // seconds
}

// JSONWriter handles JSON output with streaming support
type JSONWriter struct {
	config     *JSONWriterConfig
	file       *os.File
	writer     *bufio.Writer
	encoder    *json.Encoder
	mutex      sync.Mutex
	itemCount  int64
	isArray    bool
	syncTicker *time.Ticker
	done       chan bool
}

// NewJSONWriter creates a new JSON output writer
func NewJSONWriter(config *JSONWriterConfig) (*JSONWriter, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Set defaults
	if config.BufferSize <= 0 {
		config.BufferSize = 8192
	}
	if config.SyncInterval <= 0 {
		config.SyncInterval = 30
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

	// Create buffered writer
	writer := bufio.NewWriterSize(file, config.BufferSize)
	
	// Create encoder
	encoder := json.NewEncoder(writer)
	if config.Indent != "" {
		encoder.SetIndent("", config.Indent)
	}

	jw := &JSONWriter{
		config:  config,
		file:    file,
		writer:  writer,
		encoder: encoder,
		done:    make(chan bool),
	}

	// Start sync ticker if configured
	if config.SyncInterval > 0 {
		jw.syncTicker = time.NewTicker(time.Duration(config.SyncInterval) * time.Second)
		go jw.syncRoutine()
	}

	return jw, nil
}

// WriteData writes a single data item
func (jw *JSONWriter) WriteData(data map[string]interface{}) error {
	jw.mutex.Lock()
	defer jw.mutex.Unlock()

	if jw.config.StreamMode {
		return jw.writeStreamItem(data)
	}

	return jw.encoder.Encode(data)
}

// WriteBatch writes multiple data items
func (jw *JSONWriter) WriteBatch(items []map[string]interface{}) error {
	jw.mutex.Lock()
	defer jw.mutex.Unlock()

	if jw.config.StreamMode {
		return jw.writeStreamBatch(items)
	}

	return jw.encoder.Encode(items)
}

// WriteArray starts array mode for streaming individual items
func (jw *JSONWriter) WriteArray() error {
	jw.mutex.Lock()
	defer jw.mutex.Unlock()

	if jw.isArray {
		return fmt.Errorf("array mode already started")
	}

	if _, err := jw.writer.WriteString("[\n"); err != nil {
		return fmt.Errorf("failed to write array start: %w", err)
	}

	jw.isArray = true
	return nil
}

// WriteArrayItem writes an item to the array
func (jw *JSONWriter) WriteArrayItem(data map[string]interface{}) error {
	jw.mutex.Lock()
	defer jw.mutex.Unlock()

	if !jw.isArray {
		return fmt.Errorf("array mode not started")
	}

	// Add comma separator for subsequent items
	if jw.itemCount > 0 {
		if _, err := jw.writer.WriteString(",\n"); err != nil {
			return fmt.Errorf("failed to write separator: %w", err)
		}
	}

	// Write indented item
	if jw.config.Indent != "" {
		if _, err := jw.writer.WriteString(jw.config.Indent); err != nil {
			return err
		}
	}

	if err := jw.encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode item: %w", err)
	}

	jw.itemCount++
	return nil
}

// CloseArray closes array mode
func (jw *JSONWriter) CloseArray() error {
	jw.mutex.Lock()
	defer jw.mutex.Unlock()

	if !jw.isArray {
		return fmt.Errorf("array mode not started")
	}

	if _, err := jw.writer.WriteString("\n]"); err != nil {
		return fmt.Errorf("failed to write array end: %w", err)
	}

	jw.isArray = false
	return jw.writer.Flush()
}

// writeStreamItem writes a single item in stream mode (JSONL)
func (jw *JSONWriter) writeStreamItem(data map[string]interface{}) error {
	if err := jw.encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode stream item: %w", err)
	}
	jw.itemCount++
	return nil
}

// writeStreamBatch writes multiple items in stream mode
func (jw *JSONWriter) writeStreamBatch(items []map[string]interface{}) error {
	for _, item := range items {
		if err := jw.writeStreamItem(item); err != nil {
			return err
		}
	}
	return nil
}

// Flush forces write of buffered data
func (jw *JSONWriter) Flush() error {
	jw.mutex.Lock()
	defer jw.mutex.Unlock()

	if err := jw.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}

	return jw.file.Sync()
}

// Close closes the writer and file
func (jw *JSONWriter) Close() error {
	jw.mutex.Lock()
	defer jw.mutex.Unlock()

	// Stop sync routine
	if jw.syncTicker != nil {
		jw.syncTicker.Stop()
		close(jw.done)
	}

	// Close array if open
	if jw.isArray {
		if _, err := jw.writer.WriteString("\n]"); err != nil {
			return fmt.Errorf("failed to close array: %w", err)
		}
	}

	// Flush and close
	if err := jw.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush on close: %w", err)
	}

	if err := jw.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}

// GetStats returns writer statistics
func (jw *JSONWriter) GetStats() map[string]interface{} {
	jw.mutex.Lock()
	defer jw.mutex.Unlock()

	return map[string]interface{}{
		"items_written": jw.itemCount,
		"file_path":     jw.config.FilePath,
		"stream_mode":   jw.config.StreamMode,
		"array_mode":    jw.isArray,
		"buffer_size":   jw.config.BufferSize,
	}
}

// syncRoutine periodically syncs data to disk
func (jw *JSONWriter) syncRoutine() {
	for {
		select {
		case <-jw.syncTicker.C:
			jw.Flush()
		case <-jw.done:
			return
		}
	}
}

// ValidateFilePath validates the output file path
func ValidateFilePath(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	// Check if we can write to the directory
	tempFile := filepath.Join(dir, ".write_test")
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("cannot write to directory %s: %w", dir, err)
	}
	f.Close()
	os.Remove(tempFile)

	return nil
}

// EnsureJSONExtension ensures the file has .json extension
func EnsureJSONExtension(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext != ".json" && ext != ".jsonl" {
		return filePath + ".json"
	}
	return filePath
}

// CompactJSON compacts JSON by removing unnecessary whitespace
func CompactJSON(src io.Reader, dst io.Writer) error {
	decoder := json.NewDecoder(src)
	encoder := json.NewEncoder(dst)

	for {
		var data interface{}
		if err := decoder.Decode(&data); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode JSON: %w", err)
		}

		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	}

	return nil
}

// PrettyJSON formats JSON with indentation
func PrettyJSON(src io.Reader, dst io.Writer, indent string) error {
	decoder := json.NewDecoder(src)
	encoder := json.NewEncoder(dst)
	encoder.SetIndent("", indent)

	for {
		var data interface{}
		if err := decoder.Decode(&data); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode JSON: %w", err)
		}

		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	}

	return nil
}
