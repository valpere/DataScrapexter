// internal/output/manager_test.go
package output

import (
	"path/filepath"
	"testing"
)

func TestOutputManager_CreateWriter_JSON(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewOutputManager()

	config := &OutputConfig{
		Format:   FormatJSON,
		FilePath: filepath.Join(tempDir, "test.json"),
		Indent:   "  ",
	}

	writer, err := manager.CreateWriter(config)
	if err != nil {
		t.Fatalf("Failed to create JSON writer: %v", err)
	}
	defer writer.Close()

	data := map[string]interface{}{"test": "value"}
	err = writer.WriteData(data)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}
}

func TestOutputManager_CreateWriter_CSV(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewOutputManager()

	config := &OutputConfig{
		Format:    FormatCSV,
		FilePath:  filepath.Join(tempDir, "test.csv"),
		Headers:   true,
		Delimiter: ",",
		Fields:    []string{"name", "value"},
	}

	writer, err := manager.CreateWriter(config)
	if err != nil {
		t.Fatalf("Failed to create CSV writer: %v", err)
	}
	defer writer.Close()

	data := map[string]interface{}{"name": "test", "value": 123}
	err = writer.WriteData(data)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}
}

func TestOutputManager_DetectFormat(t *testing.T) {
	manager := NewOutputManager()

	tests := []struct {
		filePath string
		expected OutputFormat
	}{
		{"file.json", FormatJSON},
		{"file.csv", FormatCSV},
		{"file.jsonl", FormatJSONL},
		{"file.txt", FormatJSON}, // Default fallback
		{"file", FormatJSON},     // No extension
	}

	for _, test := range tests {
		result := manager.DetectFormat(test.filePath)
		if result != test.expected {
			t.Errorf("DetectFormat(%s): expected %s, got %s", test.filePath, test.expected, result)
		}
	}
}

func TestOutputManager_EnsureExtension(t *testing.T) {
	manager := NewOutputManager()

	tests := []struct {
		filePath string
		format   OutputFormat
		expected string
	}{
		{"file.json", FormatJSON, "file.json"},
		{"file", FormatJSON, "file.json"},
		{"file.txt", FormatCSV, "file.txt.csv"},
		{"data", FormatJSONL, "data.jsonl"},
	}

	for _, test := range tests {
		result := manager.EnsureExtension(test.filePath, test.format)
		if result != test.expected {
			t.Errorf("EnsureExtension(%s, %s): expected %s, got %s", 
				test.filePath, test.format, test.expected, result)
		}
	}
}

func TestOutputManager_ValidateConfig(t *testing.T) {
	manager := NewOutputManager()

	tests := []struct {
		name        string
		config      *OutputConfig
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "empty file path",
			config: &OutputConfig{
				Format: FormatJSON,
			},
			expectError: true,
		},
		{
			name: "invalid format",
			config: &OutputConfig{
				Format:   "invalid",
				FilePath: "test.txt",
			},
			expectError: true,
		},
		{
			name: "invalid delimiter",
			config: &OutputConfig{
				Format:    FormatCSV,
				FilePath:  "test.csv",
				Delimiter: "ab", // Too long
			},
			expectError: true,
		},
		{
			name: "valid config",
			config: &OutputConfig{
				Format:   FormatJSON,
				FilePath: "test.json",
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := manager.ValidateConfig(test.config)
			if test.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !test.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestOutputManager_CreateMultiWriter(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewOutputManager()

	configs := []*OutputConfig{
		{
			Format:   FormatJSON,
			FilePath: filepath.Join(tempDir, "output.json"),
		},
		{
			Format:   FormatCSV,
			FilePath: filepath.Join(tempDir, "output.csv"),
			Headers:  true,
		},
	}

	writers, err := manager.CreateMultiWriter(configs)
	if err != nil {
		t.Fatalf("Failed to create multi-writer: %v", err)
	}

	if len(writers) != 2 {
		t.Errorf("Expected 2 writers, got %d", len(writers))
	}

	// Clean up
	for _, writer := range writers {
		writer.Close()
	}
}

func TestMultiWriter(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewOutputManager()

	// Create multiple writers
	jsonConfig := &OutputConfig{
		Format:   FormatJSON,
		FilePath: filepath.Join(tempDir, "multi.json"),
	}
	csvConfig := &OutputConfig{
		Format:  FormatCSV,
		FilePath: filepath.Join(tempDir, "multi.csv"),
		Headers: true,
		Fields:  []string{"name", "value"},
	}

	jsonWriter, err := manager.CreateWriter(jsonConfig)
	if err != nil {
		t.Fatalf("Failed to create JSON writer: %v", err)
	}

	csvWriter, err := manager.CreateWriter(csvConfig)
	if err != nil {
		t.Fatalf("Failed to create CSV writer: %v", err)
	}

	multiWriter := NewMultiWriter([]Writer{jsonWriter, csvWriter})

	// Write data to both
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}

	err = multiWriter.WriteData(data)
	if err != nil {
		t.Fatalf("Failed to write to multi-writer: %v", err)
	}

	err = multiWriter.Flush()
	if err != nil {
		t.Fatalf("Failed to flush multi-writer: %v", err)
	}

	// Check stats before closing
	multiStats := multiWriter.GetStats()
	if multiStats["writer_count"] != 2 {
		t.Errorf("Expected 2 writers in multi-writer, got %v", multiStats["writer_count"])
	}

	err = multiWriter.Close()
	if err != nil {
		t.Fatalf("Failed to close multi-writer: %v", err)
	}
}

func TestOutputManager_GetSupportedFormats(t *testing.T) {
	manager := NewOutputManager()
	formats := manager.GetSupportedFormats()

	expected := []OutputFormat{FormatJSON, FormatJSONL, FormatCSV}
	if len(formats) != len(expected) {
		t.Errorf("Expected %d formats, got %d", len(expected), len(formats))
	}

	for _, expectedFormat := range expected {
		found := false
		for _, format := range formats {
			if format == expectedFormat {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected format %s not found", expectedFormat)
		}
	}
}

func TestOutputManager_AutoDetectFormat(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewOutputManager()

	// Test auto-detection
	config := &OutputConfig{
		// No format specified - should auto-detect from extension
		FilePath: filepath.Join(tempDir, "auto.csv"),
		Headers:  true,
	}

	writer, err := manager.CreateWriter(config)
	if err != nil {
		t.Fatalf("Failed to create writer with auto-detection: %v", err)
	}
	defer writer.Close()

	// Should have created CSV writer
	stats := writer.GetStats()
	if _, exists := stats["delimiter"]; !exists {
		t.Error("Expected CSV-specific stats (delimiter) but not found")
	}
}
