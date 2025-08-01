// internal/output/manager_test.go
package output

import (
	"testing"

	"github.com/valpere/DataScrapexter/internal/config"
)

func TestNewManager(t *testing.T) {
	cfg := &config.OutputConfig{
		Format: "json",
		File:   "test.json",
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("manager should not be nil")
	}

	if manager.config.Format != "json" {
		t.Errorf("expected format 'json', got %s", manager.config.Format)
	}
}

func TestNewManagerWithNilConfig(t *testing.T) {
	_, err := NewManager(nil)
	if err == nil {
		t.Fatal("expected error with nil config")
	}
}

func TestGetWriter(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		expectError    bool
		expectedType   string
	}{
		{
			name:         "JSON writer",
			format:       "json",
			expectError:  false,
			expectedType: "*output.JSONWriter",
		},
		{
			name:         "CSV writer",
			format:       "csv",
			expectError:  false,
			expectedType: "*output.CSVWriter",
		},
		{
			name:        "unsupported format",
			format:      "xml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.OutputConfig{
				Format: tt.format,
				File:   "test." + tt.format,
			}

			manager, err := NewManager(cfg)
			if err != nil {
				t.Fatalf("failed to create manager: %v", err)
			}

			writer, err := manager.GetWriter()
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if writer == nil {
				t.Error("writer should not be nil")
			}
		})
	}
}

func TestManagerWrite(t *testing.T) {
	cfg := &config.OutputConfig{
		Format: "json",
		File:   "test_output.json",
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	testData := []map[string]interface{}{
		{
			"title": "Test Title",
			"price": "$19.99",
		},
	}

	err = manager.Write(testData)
	if err != nil {
		t.Errorf("failed to write data: %v", err)
	}
}
