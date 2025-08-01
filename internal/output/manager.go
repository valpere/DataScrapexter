// internal/output/manager.go
package output

import (
	"fmt"

	"github.com/valpere/DataScrapexter/internal/config"
)

// Manager manages different output formats
type Manager struct {
	config        *Config
	formatOptions *FormatOptions
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
		config:        config,
		formatOptions: &FormatOptions{},
	}, nil
}

// NewManagerWithOptions creates a new output manager with format-specific options
func NewManagerWithOptions(cfg *Config, options *FormatOptions) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("output configuration is required")
	}

	if options == nil {
		options = &FormatOptions{}
	}

	return &Manager{
		config:        cfg,
		formatOptions: options,
	}, nil
}

// GetWriter returns the appropriate writer for the configured format
func (m *Manager) GetWriter() (Writer, error) {
	switch m.config.Format {
	case FormatJSON:
		return NewJSONWriter(m.config.File)
	case FormatCSV:
		return NewCSVWriter(m.config.File)
	case FormatPostgreSQL:
		return m.createPostgreSQLWriter()
	case FormatSQLite:
		return m.createSQLiteWriter()
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

// createPostgreSQLWriter creates a PostgreSQL writer from configuration
func (m *Manager) createPostgreSQLWriter() (Writer, error) {
	// Use format options if available, otherwise use defaults
	options := m.formatOptions.PostgreSQL
	
	// Set defaults if not specified
	if options.Table == "" {
		options.Table = "scraped_data"
	}
	if options.Schema == "" {
		options.Schema = "public"
	}
	if options.BatchSize == 0 {
		options.BatchSize = 1000
	}
	if options.OnConflict == "" {
		options.OnConflict = ConflictIgnore
	}
	if options.ColumnTypes == nil {
		options.ColumnTypes = make(map[string]string)
	}

	// Connection string is required
	if options.ConnectionString == "" {
		return nil, fmt.Errorf("PostgreSQL connection_string is required")
	}

	return NewPostgreSQLWriter(options)
}

// createSQLiteWriter creates a SQLite writer from configuration
func (m *Manager) createSQLiteWriter() (Writer, error) {
	// Use format options if available, otherwise use defaults
	options := m.formatOptions.SQLite
	
	// Set defaults if not specified
	if options.Table == "" {
		options.Table = "scraped_data"
	}
	if options.BatchSize == 0 {
		options.BatchSize = 1000
	}
	if options.OnConflict == "" {
		options.OnConflict = ConflictIgnore
	}
	if options.ColumnTypes == nil {
		options.ColumnTypes = make(map[string]string)
	}

	// Database path is required
	if options.DatabasePath == "" {
		// If no database_path specified, use the File field or default
		if m.config.File != "" {
			options.DatabasePath = m.config.File
		} else {
			options.DatabasePath = "output.db"
		}
	}

	return NewSQLiteWriter(options)
}

