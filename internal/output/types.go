// internal/output/types.go
package output

import (
	"time"
)

// OutputFormat represents supported output formats
type OutputFormat string

const (
	FormatJSON OutputFormat = "json"
	FormatCSV  OutputFormat = "csv"
	FormatXML  OutputFormat = "xml"
	FormatYAML OutputFormat = "yaml"
	FormatTSV  OutputFormat = "tsv"
)

// ValidOutputFormats returns all valid output format values
func ValidOutputFormats() []OutputFormat {
	return []OutputFormat{FormatJSON, FormatCSV, FormatXML, FormatYAML, FormatTSV}
}

// IsValid checks if the output format is valid
func (of OutputFormat) IsValid() bool {
	for _, valid := range ValidOutputFormats() {
		if of == valid {
			return true
		}
	}
	return false
}

// GetFileExtension returns the appropriate file extension for the format
func (of OutputFormat) GetFileExtension() string {
	switch of {
	case FormatJSON:
		return ".json"
	case FormatCSV:
		return ".csv"
	case FormatXML:
		return ".xml"
	case FormatYAML:
		return ".yaml"
	case FormatTSV:
		return ".tsv"
	default:
		return ".txt"
	}
}

// GetMimeType returns the MIME type for the format
func (of OutputFormat) GetMimeType() string {
	switch of {
	case FormatJSON:
		return "application/json"
	case FormatCSV:
		return "text/csv"
	case FormatXML:
		return "application/xml"
	case FormatYAML:
		return "application/yaml"
	case FormatTSV:
		return "text/tab-separated-values"
	default:
		return "text/plain"
	}
}

// Config defines output configuration without conflicting with existing types
type Config struct {
	Format   OutputFormat      `yaml:"format" json:"format"`
	File     string            `yaml:"file,omitempty" json:"file,omitempty"`
	Options  map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
	Append   bool              `yaml:"append,omitempty" json:"append,omitempty"`
	Template string            `yaml:"template,omitempty" json:"template,omitempty"`
}

// Writer defines the interface for output writers without conflicting
type Writer interface {
	Write(data []map[string]interface{}) error
	Close() error
}

// Result represents the output operation result
type Result struct {
	Success      bool          `json:"success"`
	RecordsCount int           `json:"records_count"`
	FilePath     string        `json:"file_path,omitempty"`
	Format       string        `json:"format"`
	Duration     time.Duration `json:"duration"`
	Error        string        `json:"error,omitempty"`
	Size         int64         `json:"size,omitempty"`
}

// Statistics contains output statistics
type Statistics struct {
	TotalRecords    int           `json:"total_records"`
	TotalFiles      int           `json:"total_files"`
	TotalSize       int64         `json:"total_size"`
	ProcessingTime  time.Duration `json:"processing_time"`
	AverageFileSize int64         `json:"average_file_size"`
	Formats         map[string]int `json:"formats"`
}

// ValidationError represents output validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Record  int    `json:"record"`
}

// FormatOptions defines format-specific options
type FormatOptions struct {
	JSON JSONOptions `yaml:"json,omitempty" json:"json,omitempty"`
	CSV  CSVOptions  `yaml:"csv,omitempty" json:"csv,omitempty"`
}

// JSONOptions defines JSON-specific options
type JSONOptions struct {
	Indent     string `yaml:"indent,omitempty" json:"indent,omitempty"`
	Compact    bool   `yaml:"compact,omitempty" json:"compact,omitempty"`
	SortKeys   bool   `yaml:"sort_keys,omitempty" json:"sort_keys,omitempty"`
	EscapeHTML bool   `yaml:"escape_html,omitempty" json:"escape_html,omitempty"`
}

// CSVOptions defines CSV-specific options
type CSVOptions struct {
	Delimiter string   `yaml:"delimiter,omitempty" json:"delimiter,omitempty"`
	Quote     string   `yaml:"quote,omitempty" json:"quote,omitempty"`
	Header    bool     `yaml:"header" json:"header"`
	Columns   []string `yaml:"columns,omitempty" json:"columns,omitempty"`
	SkipEmpty bool     `yaml:"skip_empty,omitempty" json:"skip_empty,omitempty"`
}

// SupportedFormats lists all supported output formats
var SupportedFormats = []string{
	"json",
	"csv",
	"xml",
	"yaml",
	"txt",
	"html",
	"jsonl", // JSON Lines
}

// DefaultConfigs provides default configurations for each format
var DefaultConfigs = map[string]Config{
	"json": {
		Format: FormatJSON,
		Options: map[string]string{
			"indent":  "  ",
			"compact": "false",
		},
	},
	"csv": {
		Format: FormatCSV,
		Options: map[string]string{
			"delimiter": ",",
			"header":    "true",
			"quote":     "\"",
		},
	},
	"xml": {
		Format: FormatXML,
		Options: map[string]string{
			"indent": "  ",
			"root":   "data",
		},
	},
	"yaml": {
		Format: FormatYAML,
		Options: map[string]string{
			"indent": "2",
		},
	},
}
