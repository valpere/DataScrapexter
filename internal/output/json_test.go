// internal/output/json_test.go
package output

import (
	"encoding/json"
	"os"
	"testing"
)

func TestJSONWriter(t *testing.T) {
	filename := "test_output.json"
	defer os.Remove(filename)

	writer, err := NewJSONWriter(filename)
	if err != nil {
		t.Fatalf("failed to create JSON writer: %v", err)
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
		t.Fatalf("failed to write JSON data: %v", err)
	}

	writer.Close()

	// Verify the file was created and contains valid JSON
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var result []map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}

	if result[0]["title"] != "Test Title" {
		t.Errorf("expected title 'Test Title', got %v", result[0]["title"])
	}
}

func TestJSONWriter_WriteRecord(t *testing.T) {
	filename := "test_record.json"
	defer os.Remove(filename)

	writer, err := NewJSONWriter(filename)
	if err != nil {
		t.Fatalf("failed to create JSON writer: %v", err)
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

	// Verify the file was created and contains valid JSON
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if result["name"] != "Test Record" {
		t.Errorf("expected name 'Test Record', got %v", result["name"])
	}
}

func TestJSONWriter_WriteEmpty(t *testing.T) {
	filename := "test_empty.json"
	defer os.Remove(filename)

	writer, err := NewJSONWriter(filename)
	if err != nil {
		t.Fatalf("failed to create JSON writer: %v", err)
	}
	defer writer.Close()

	// Write empty data
	err = writer.Write([]map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to write empty JSON data: %v", err)
	}

	writer.Close()

	// Verify the file was created and contains valid JSON
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var result []map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestJSONWriter_Close(t *testing.T) {
	filename := "test_close.json"
	defer os.Remove(filename)

	writer, err := NewJSONWriter(filename)
	if err != nil {
		t.Fatalf("failed to create JSON writer: %v", err)
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

func TestJSONWriter_Flush(t *testing.T) {
	filename := "test_flush.json"
	defer os.Remove(filename)

	writer, err := NewJSONWriter(filename)
	if err != nil {
		t.Fatalf("failed to create JSON writer: %v", err)
	}
	defer writer.Close()

	err = writer.Flush()
	if err != nil {
		t.Errorf("flush should not return error: %v", err)
	}
}
