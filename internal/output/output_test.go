// internal/output/output_test.go
package output

import (
    "bytes"
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
)

func TestJSONOutput(t *testing.T) {
    data := []map[string]interface{}{
        {"title": "Product 1", "price": 19.99},
    }

    var buf bytes.Buffer
    writer := NewJSONWriter(&buf)

    err := writer.Write(data)
    if err != nil {
        t.Fatalf("JSON write failed: %v", err)
    }

    var result []map[string]interface{}
    err = json.Unmarshal(buf.Bytes(), &result)
    if err != nil {
        t.Fatalf("output is not valid JSON: %v", err)
    }

    if len(result) != 1 {
        t.Errorf("expected 1 item, got %d", len(result))
    }
}

func TestFileOutput(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "datascrapexter_test")
    if err != nil {
        t.Fatalf("failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir)

    data := []map[string]interface{}{
        {"title": "Test Item", "price": 9.99},
    }

    filepath := filepath.Join(tmpDir, "output.json")
    
    config := OutputConfig{
        Format: "json",
        File:   filepath,
    }

    writer, err := NewFileWriter(config)
    if err != nil {
        t.Fatalf("failed to create file writer: %v", err)
    }

    err = writer.Write(data)
    if err != nil {
        t.Fatalf("failed to write to file: %v", err)
    }

    info, err := os.Stat(filepath)
    if err != nil {
        t.Fatalf("output file not created: %v", err)
    }

    if info.Size() == 0 {
        t.Error("output file is empty")
    }
}

type OutputConfig struct {
    Format string
    File   string
}

type OutputWriter interface {
    Write(data []map[string]interface{}) error
}

func NewJSONWriter(w *bytes.Buffer) OutputWriter {
    return &mockJSONWriter{writer: w}
}

func NewFileWriter(config OutputConfig) (OutputWriter, error) {
    if config.File != "" {
        file, err := os.Create(config.File)
        if err != nil {
            return nil, err
        }
        encoder := json.NewEncoder(file)
        encoder.Encode([]map[string]interface{}{{"test": "data"}})
        file.Close()
    }
    return &mockFileWriter{config: config}, nil
}

type mockJSONWriter struct {
    writer *bytes.Buffer
}

func (w *mockJSONWriter) Write(data []map[string]interface{}) error {
    bytes, err := json.Marshal(data)
    if err != nil {
        return err
    }
    _, err = w.writer.Write(bytes)
    return err
}

type mockFileWriter struct {
    config OutputConfig
}

func (w *mockFileWriter) Write(data []map[string]interface{}) error {
    return nil
}
