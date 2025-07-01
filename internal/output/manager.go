// internal/output/manager.go
package output

import (
	"fmt"

	"github.com/valpere/DataScrapexter/internal/config"
)

// Manager manages different output formats
type Manager struct {
	config *Config
}

// NewManager creates a new output manager
func NewManager(cfg *config.OutputConfig) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("output configuration is required")
	}

	config := &Config{
		Format: OutputFormat(cfg.Format),
		File:   cfg.File,
	}

	return &Manager{
		config: config,
	}, nil
}

// GetWriter returns the appropriate writer for the configured format
func (m *Manager) GetWriter() (Writer, error) {
	switch m.config.Format {
	case FormatJSON:
		return NewJSONWriter(m.config.File)
	case FormatCSV:
		return NewCSVWriter(m.config.File)
	default:
		return nil, fmt.Errorf("unsupported output format: %s", m.config.Format)
	}
}

// Write writes data using the configured format
func (m *Manager) Write(data []map[string]interface{}) error {
	writer, err := m.GetWriter()
	if err != nil {
		return fmt.Errorf("failed to get writer: %w", err)
	}
	defer writer.Close()

	return writer.Write(data)
}

// WriteResults writes scraping results using the configured format
func (m *Manager) WriteResults(results []map[string]interface{}) error {
	return m.Write(results)
}
