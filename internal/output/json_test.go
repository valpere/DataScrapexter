// internal/output/json_test.go
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewJSONWriter(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.json")

	config := &JSONWriterConfig{
		FilePath:   filePath,
		Indent:     "  ",
		BufferSize: 1024,
	}

	writer, err := NewJSONWriter(config)
	if err != nil {
		t.Fatalf("Failed to create JSON writer: %v", err)
	}
	defer writer.Close()

	if writer.config.FilePath != filePath {
		t.Errorf("Expected file path %s, got %s", filePath, writer.config.FilePath)
	}
}

func TestJSONWriter_WriteData(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.json")

	config := &JSONWriterConfig{
		FilePath: filePath,
		Indent:   "  ",
	}

	writer, err := NewJSONWriter(config)
	if err != nil {
		t.Fatalf("Failed to create JSON writer: %v", err)
	}

	data := map[string]interface{}{
		"title": "Test Product",
		"price": 99.99,
		"tags":  []string{"electronics", "gadget"},
	}

	err = writer.WriteData(data)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Read and verify file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(content, &result)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["title"] != "Test Product" {
		t.Errorf("Expected title 'Test Product', got %v", result["title"])
	}

	if result["price"] != 99.99 {
		t.Errorf("Expected price 99.99, got %v", result["price"])
	}
}

func TestJSONWriter_WriteBatch(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "batch.json")

	config := &JSONWriterConfig{
		FilePath: filePath,
		Indent:   "  ",
	}

	writer, err := NewJSONWriter(config)
	if err != nil {
		t.Fatalf("Failed to create JSON writer: %v", err)
	}

	items := []map[string]interface{}{
		{"name": "Product 1", "price": 10.0},
		{"name": "Product 2", "price": 20.0},
		{"name": "Product 3", "price": 30.0},
	}

	err = writer.WriteBatch(items)
	if err != nil {
		t.Fatalf("Failed to write batch: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Read and verify file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var result []map[string]interface{}
	err = json.Unmarshal(content, &result)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	if result[0]["name"] != "Product 1" {
		t.Errorf("Expected first item name 'Product 1', got %v", result[0]["name"])
	}
}

func TestJSONWriter_StreamMode(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "stream.jsonl")

	config := &JSONWriterConfig{
		FilePath:   filePath,
		StreamMode: true,
	}

	writer, err := NewJSONWriter(config)
	if err != nil {
		t.Fatalf("Failed to create JSON writer: %v", err)
	}

	// Write multiple items
	for i := 1; i <= 3; i++ {
		data := map[string]interface{}{
			"id":   i,
			"name": fmt.Sprintf("Item %d", i),
		}
		err = writer.WriteData(data)
		if err != nil {
			t.Fatalf("Failed to write data: %v", err)
		}
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Read and verify file content (JSONL format)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var item map[string]interface{}
		err = json.Unmarshal([]byte(line), &item)
		if err != nil {
			t.Fatalf("Failed to parse line %d as JSON: %v", i+1, err)
		}

		expectedID := float64(i + 1) // JSON numbers are float64
		if item["id"] != expectedID {
			t.Errorf("Line %d: expected id %v, got %v", i+1, expectedID, item["id"])
		}
	}
}

func TestJSONWriter_ArrayMode(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "array.json")

	config := &JSONWriterConfig{
		FilePath: filePath,
		Indent:   "  ",
	}

	writer, err := NewJSONWriter(config)
	if err != nil {
		t.Fatalf("Failed to create JSON writer: %v", err)
	}

	err = writer.WriteArray()
	if err != nil {
		t.Fatalf("Failed to start array: %v", err)
	}

	// Write array items
	for i := 1; i <= 3; i++ {
		data := map[string]interface{}{
			"id":   i,
			"name": fmt.Sprintf("Item %d", i),
		}
		err = writer.WriteArrayItem(data)
		if err != nil {
			t.Fatalf("Failed to write array item: %v", err)
		}
	}

	err = writer.CloseArray()
	if err != nil {
		t.Fatalf("Failed to close array: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Read and verify file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var result []map[string]interface{}
	err = json.Unmarshal(content, &result)
	if err != nil {
		t.Fatalf("Failed to parse JSON array: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}
}

func TestJSONWriter_AppendMode(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "append.jsonl")

	// Write initial data
	config1 := &JSONWriterConfig{
		FilePath:   filePath,
		StreamMode: true,
	}

	writer1, err := NewJSONWriter(config1)
	if err != nil {
		t.Fatalf("Failed to create first writer: %v", err)
	}

	err = writer1.WriteData(map[string]interface{}{"id": 1, "name": "First"})
	if err != nil {
		t.Fatalf("Failed to write first data: %v", err)
	}

	err = writer1.Close()
	if err != nil {
		t.Fatalf("Failed to close first writer: %v", err)
	}

	// Append more data
	config2 := &JSONWriterConfig{
		FilePath:   filePath,
		StreamMode: true,
		AppendMode: true,
	}

	writer2, err := NewJSONWriter(config2)
	if err != nil {
		t.Fatalf("Failed to create second writer: %v", err)
	}

	err = writer2.WriteData(map[string]interface{}{"id": 2, "name": "Second"})
	if err != nil {
		t.Fatalf("Failed to write second data: %v", err)
	}

	err = writer2.Close()
	if err != nil {
		t.Fatalf("Failed to close second writer: %v", err)
	}

	// Verify both entries exist
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestJSONWriter_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "stats.json")

	config := &JSONWriterConfig{
		FilePath:   filePath,
		StreamMode: true,
	}

	writer, err := NewJSONWriter(config)
	if err != nil {
		t.Fatalf("Failed to create JSON writer: %v", err)
	}
	defer writer.Close()

	// Write some data
	for i := 0; i < 5; i++ {
		err = writer.WriteData(map[string]interface{}{"id": i})
		if err != nil {
			t.Fatalf("Failed to write data: %v", err)
		}
	}

	stats := writer.GetStats()
	if stats["items_written"] != int64(5) {
		t.Errorf("Expected items_written 5, got %v", stats["items_written"])
	}

	if stats["file_path"] != filePath {
		t.Errorf("Expected file_path %s, got %v", filePath, stats["file_path"])
	}

	if stats["stream_mode"] != true {
		t.Errorf("Expected stream_mode true, got %v", stats["stream_mode"])
	}
}

func TestValidateFilePath(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		filePath    string
		expectError bool
	}{
		{"valid path", filepath.Join(tempDir, "valid.json"), false},
		{"empty path", "", true},
		{"nested path", filepath.Join(tempDir, "subdir", "file.json"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.filePath)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestEnsureJSONExtension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"file.json", "file.json"},
		{"file.jsonl", "file.jsonl"},
		{"file.txt", "file.txt.json"},
		{"file", "file.json"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := EnsureJSONExtension(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestJSONWriter_Flush(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "flush.json")

	config := &JSONWriterConfig{
		FilePath:   filePath,
		BufferSize: 1024,
	}

	writer, err := NewJSONWriter(config)
	if err != nil {
		t.Fatalf("Failed to create JSON writer: %v", err)
	}
	defer writer.Close()

	data := map[string]interface{}{"test": "data"}
	err = writer.WriteData(data)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	err = writer.Flush()
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	// Verify data is written to file
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Expected file content after flush, but file is empty")
	}
}
