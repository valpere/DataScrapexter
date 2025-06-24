// internal/config/config_test.go
package config

import (
    "os"
    "testing"
)

func TestLoadFromBytes(t *testing.T) {
    configYAML := `
name: "bytes_test"
base_url: "https://test.com"
fields:
  - name: "content"
    selector: ".content"
    type: "text"
output:
  format: "csv"
  file: "output.csv"
`

    config, err := LoadFromBytes([]byte(configYAML))
    if err != nil {
        t.Fatalf("LoadFromBytes failed: %v", err)
    }

    if config.Name != "bytes_test" {
        t.Errorf("expected name 'bytes_test', got %q", config.Name)
    }
}

func TestLoadFromFile(t *testing.T) {
    configYAML := `
name: "test_scraper"
base_url: "https://example.com"
fields:
  - name: "title"
    selector: "h1"
    type: "text"
    required: true
output:
  format: "json"
  file: "output.json"
`

    tmpFile, err := os.CreateTemp("", "test_config_*.yaml")
    if err != nil {
        t.Fatalf("failed to create temp file: %v", err)
    }
    defer os.Remove(tmpFile.Name())

    if _, err := tmpFile.WriteString(configYAML); err != nil {
        t.Fatalf("failed to write temp file: %v", err)
    }
    tmpFile.Close()

    config, err := LoadFromFile(tmpFile.Name())
    if err != nil {
        t.Fatalf("LoadFromFile failed: %v", err)
    }

    if config.Name != "test_scraper" {
        t.Errorf("expected name 'test_scraper', got %q", config.Name)
    }
}

func TestGenerateTemplate(t *testing.T) {
    config := GenerateTemplate("basic")

    if config.Name != "basic_scraper" {
        t.Errorf("expected name 'basic_scraper', got %q", config.Name)
    }

    if len(config.Fields) == 0 {
        t.Error("template should have fields")
    }

    if err := config.Validate(); err != nil {
        t.Errorf("generated template should be valid: %v", err)
    }
}
