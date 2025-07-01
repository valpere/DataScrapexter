// internal/output/csv_test.go
package output

import (
	"os"
	"strings"
	"testing"
)

func TestNewCSVWriter(t *testing.T) {
	filename := "test_output.csv"
	defer os.Remove(filename)

	writer, err := NewCSVWriter(filename)
	if err != nil {
		t.Fatalf("failed to create CSV writer: %v", err)
	}
	defer writer.Close()

	if writer == nil {
		t.Fatal("writer should not be nil")
	}

	if writer.filename != filename {
		t.Errorf("expected filename %s, got %s", filename, writer.filename)
	}
}

func TestCSVWriter_Write(t *testing.T) {
	filename := "test_output.csv"
	defer os.Remove(filename)

	writer, err := NewCSVWriter(filename)
	if err != nil {
		t.Fatalf("failed to create CSV writer: %v", err)
	}
	defer writer.Close()

	testData := []map[string]interface{}{
		{
			"title": "Test Title",
			"price": "$19.99",
			"count": 42,
		},
		{
			"title": "Another Item", 
			"price": "$25.50",
			"count": 15,
		},
	}

	err = writer.Write(testData)
	if err != nil {
		t.Fatalf("failed to write CSV data: %v", err)
	}

	writer.Close()

	// Verify the file was created and contains expected content
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)
	lines := strings.Split(strings.TrimSpace(content), "\n")

	// Should have header + 2 data rows
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (header + 2 data), got %d", len(lines))
	}

	// Check header contains expected fields
	header := lines[0]
	expectedFields := []string{"count", "price", "title"} // Sorted alphabetically
	for _, field := range expectedFields {
		if !strings.Contains(header, field) {
			t.Errorf("header missing field %s: %s", field, header)
		}
	}

	// Check data rows contain expected values
	if !strings.Contains(content, "Test Title") {
		t.Error("CSV missing 'Test Title'")
	}
	if !strings.Contains(content, "$19.99") {
		t.Error("CSV missing '$19.99'")
	}
}

func TestCSVWriter_WriteEmpty(t *testing.T) {
	filename := "test_empty.csv"
	defer os.Remove(filename)

	writer, err := NewCSVWriter(filename)
	if err != nil {
		t.Fatalf("failed to create CSV writer: %v", err)
	}
	defer writer.Close()

	// Write empty data
	err = writer.Write([]map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to write empty CSV data: %v", err)
	}

	writer.Close()

	// Verify file exists but is empty or minimal
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if len(data) > 10 { // Allow for minimal content
		t.Errorf("expected minimal content for empty data, got %d bytes", len(data))
	}
}

func TestCSVWriter_WriteRecord(t *testing.T) {
	filename := "test_record.csv"
	defer os.Remove(filename)

	writer, err := NewCSVWriter(filename)
	if err != nil {
		t.Fatalf("failed to create CSV writer: %v", err)
	}
	defer writer.Close()

	record := map[string]interface{}{
		"name":  "Test Record",
		"value": 123,
	}

	err = writer.WriteRecord(record)
	if err != nil {
		t.Fatalf("failed to write record: %v", err)
	}

	writer.Close()

	// Verify the file was created
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Test Record") {
		t.Error("CSV missing 'Test Record'")
	}
}

func TestCSVWriter_Close(t *testing.T) {
	filename := "test_close.csv"
	defer os.Remove(filename)

	writer, err := NewCSVWriter(filename)
	if err != nil {
		t.Fatalf("failed to create CSV writer: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Errorf("close should not return error: %v", err)
	}

	// Should be safe to close multiple times
	err = writer.Close()
	if err != nil {
		t.Errorf("multiple close should not return error: %v", err)
	}
}
