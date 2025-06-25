// internal/output/csv_test.go
package output

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestNewCSVWriter(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.csv")

	config := &CSVWriterConfig{
		FilePath:   filePath,
		Headers:    true,
		Delimiter:  ',',
		FieldOrder: []string{"name", "price", "category"},
	}

	writer, err := NewCSVWriter(config)
	if err != nil {
		t.Fatalf("Failed to create CSV writer: %v", err)
	}
	defer writer.Close()

	if writer.config.FilePath != filePath {
		t.Errorf("Expected file path %s, got %s", filePath, writer.config.FilePath)
	}
}

func TestCSVWriter_WriteData(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "products.csv")

	config := &CSVWriterConfig{
		FilePath:   filePath,
		Headers:    true,
		FieldOrder: []string{"name", "price", "category"},
	}

	writer, err := NewCSVWriter(config)
	if err != nil {
		t.Fatalf("Failed to create CSV writer: %v", err)
	}

	data := map[string]interface{}{
		"name":     "Laptop",
		"price":    999.99,
		"category": "Electronics",
	}

	err = writer.WriteData(data)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Read and verify
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 2 { // header + 1 data row
		t.Errorf("Expected 2 records, got %d", len(records))
	}

	// Check header
	if records[0][0] != "name" || records[0][1] != "price" || records[0][2] != "category" {
		t.Errorf("Unexpected header row: %v", records[0])
	}

	// Check data
	if records[1][0] != "Laptop" || records[1][1] != "999.99" || records[1][2] != "Electronics" {
		t.Errorf("Unexpected data row: %v", records[1])
	}
}

func TestCSVWriter_AutoDetectFields(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "auto.csv")

	config := &CSVWriterConfig{
		FilePath:         filePath,
		Headers:          true,
		AutoDetectFields: true,
	}

	writer, err := NewCSVWriter(config)
	if err != nil {
		t.Fatalf("Failed to create CSV writer: %v", err)
	}

	data := map[string]interface{}{
		"product": "Phone",
		"brand":   "TechCorp",
		"price":   799,
	}

	err = writer.WriteData(data)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Verify headers were auto-detected (sorted alphabetically)
	headers := []string{"brand", "price", "product"}
	
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	for i, expected := range headers {
		if records[0][i] != expected {
			t.Errorf("Expected header %s, got %s", expected, records[0][i])
		}
	}
}

func TestCSVWriter_WriteBatch(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "batch.csv")

	config := &CSVWriterConfig{
		FilePath:   filePath,
		Headers:    true,
		FieldOrder: []string{"id", "name", "value"},
	}

	writer, err := NewCSVWriter(config)
	if err != nil {
		t.Fatalf("Failed to create CSV writer: %v", err)
	}

	items := []map[string]interface{}{
		{"id": 1, "name": "Item1", "value": 10.5},
		{"id": 2, "name": "Item2", "value": 20.0},
		{"id": 3, "name": "Item3", "value": 30.25},
	}

	err = writer.WriteBatch(items)
	if err != nil {
		t.Fatalf("Failed to write batch: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Verify
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 4 { // header + 3 data rows
		t.Errorf("Expected 4 records, got %d", len(records))
	}

	// Check first data row
	if records[1][0] != "1" || records[1][1] != "Item1" || records[1][2] != "10.5" {
		t.Errorf("Unexpected first data row: %v", records[1])
	}
}

func TestCSVWriter_FormatValue(t *testing.T) {
	writer := &CSVWriter{}

	tests := []struct {
		input    interface{}
		expected string
	}{
		{"hello", "hello"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
		{nil, ""},
		{[]interface{}{"a", "b", "c"}, "a|b|c"},
		{map[string]interface{}{"key": "value"}, "key=value"},
	}

	for _, test := range tests {
		result := writer.formatValue(test.input)
		if result != test.expected {
			t.Errorf("formatValue(%v): expected %s, got %s", test.input, test.expected, result)
		}
	}
}

func TestCSVWriter_AppendMode(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "append.csv")

	// Write initial data
	config1 := &CSVWriterConfig{
		FilePath:   filePath,
		Headers:    true,
		FieldOrder: []string{"name", "value"},
	}

	writer1, err := NewCSVWriter(config1)
	if err != nil {
		t.Fatalf("Failed to create first writer: %v", err)
	}

	err = writer1.WriteData(map[string]interface{}{"name": "First", "value": 1})
	if err != nil {
		t.Fatalf("Failed to write first data: %v", err)
	}

	err = writer1.Close()
	if err != nil {
		t.Fatalf("Failed to close first writer: %v", err)
	}

	// Append more data
	config2 := &CSVWriterConfig{
		FilePath:   filePath,
		AppendMode: true,
		FieldOrder: []string{"name", "value"},
	}

	writer2, err := NewCSVWriter(config2)
	if err != nil {
		t.Fatalf("Failed to create second writer: %v", err)
	}

	err = writer2.WriteData(map[string]interface{}{"name": "Second", "value": 2})
	if err != nil {
		t.Fatalf("Failed to write second data: %v", err)
	}

	err = writer2.Close()
	if err != nil {
		t.Fatalf("Failed to close second writer: %v", err)
	}

	// Verify both entries exist
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 3 { // header + 2 data rows
		t.Errorf("Expected 3 records, got %d", len(records))
	}
}

func TestCSVWriter_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "stats.csv")

	config := &CSVWriterConfig{
		FilePath:   filePath,
		FieldOrder: []string{"id", "value"},
	}

	writer, err := NewCSVWriter(config)
	if err != nil {
		t.Fatalf("Failed to create CSV writer: %v", err)
	}
	defer writer.Close()

	// Write some data
	for i := 0; i < 5; i++ {
		err = writer.WriteData(map[string]interface{}{"id": i, "value": i * 10})
		if err != nil {
			t.Fatalf("Failed to write data: %v", err)
		}
	}

	stats := writer.GetStats()
	if stats["rows_written"] != int64(5) {
		t.Errorf("Expected rows_written 5, got %v", stats["rows_written"])
	}

	if stats["field_count"] != 2 {
		t.Errorf("Expected field_count 2, got %v", stats["field_count"])
	}
}

func TestEnsureCSVExtension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"file.csv", "file.csv"},
		{"file.txt", "file.txt.csv"},
		{"file", "file.csv"},
	}

	for _, test := range tests {
		result := EnsureCSVExtension(test.input)
		if result != test.expected {
			t.Errorf("EnsureCSVExtension(%s): expected %s, got %s", test.input, test.expected, result)
		}
	}
}

func TestReadCSV(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "read_test.csv")

	// Create test CSV file
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	writer := csv.NewWriter(file)
	writer.Write([]string{"name", "price", "active"})
	writer.Write([]string{"Product1", "99.99", "true"})
	writer.Write([]string{"Product2", "149", "false"})
	writer.Flush()
	file.Close()

	// Read and verify
	data, err := ReadCSV(filePath, ',')
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(data) != 2 {
		t.Errorf("Expected 2 records, got %d", len(data))
	}

	if data[0]["name"] != "Product1" {
		t.Errorf("Expected name 'Product1', got %v", data[0]["name"])
	}

	if data[0]["price"] != 99.99 {
		t.Errorf("Expected price 99.99, got %v", data[0]["price"])
	}

	if data[0]["active"] != true {
		t.Errorf("Expected active true, got %v", data[0]["active"])
	}
}
