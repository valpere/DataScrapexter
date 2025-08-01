// internal/output/types.go
package output

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// OutputFormat represents supported output formats
type OutputFormat string

const (
	FormatJSON       OutputFormat = "json"
	FormatCSV        OutputFormat = "csv"
	FormatXML        OutputFormat = "xml"
	FormatYAML       OutputFormat = "yaml"
	FormatTSV        OutputFormat = "tsv"
	FormatPostgreSQL OutputFormat = "postgresql"
	FormatSQLite     OutputFormat = "sqlite"
)

// ConflictStrategy represents database conflict resolution strategies
type ConflictStrategy string

// Common conflict strategies (supported by both PostgreSQL and SQLite)
const (
	ConflictIgnore ConflictStrategy = "ignore" // Ignore conflicts (ON CONFLICT DO NOTHING / INSERT OR IGNORE)
	ConflictError  ConflictStrategy = "error"  // Fail on conflicts (default INSERT behavior)
)

// SQLite-specific conflict strategies
const (
	ConflictReplace ConflictStrategy = "replace" // SQLite only: REPLACE existing row
)

// ValidOutputFormats returns all valid output format values
func ValidOutputFormats() []OutputFormat {
	return []OutputFormat{FormatJSON, FormatCSV, FormatXML, FormatYAML, FormatTSV, FormatPostgreSQL, FormatSQLite}
}

// ValidConflictStrategies returns all valid conflict strategy values
func ValidConflictStrategies() []ConflictStrategy {
	return []ConflictStrategy{ConflictIgnore, ConflictError, ConflictReplace}
}

// IsValidConflictStrategy checks if a conflict strategy is valid
func IsValidConflictStrategy(strategy ConflictStrategy) bool {
	for _, valid := range ValidConflictStrategies() {
		if strategy == valid {
			return true
		}
	}
	return false
}

// SQL identifier validation
var (
	// SQL identifier regex: starts with letter or underscore, contains letters, digits, underscores
	sqlIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	
	// Reserved SQL keywords that should not be used as identifiers
	sqlReservedWords = map[string]bool{
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true, "CREATE": true, "DROP": true,
		"ALTER": true, "TABLE": true, "INDEX": true, "VIEW": true, "FROM": true, "WHERE": true,
		"ORDER": true, "GROUP": true, "HAVING": true, "JOIN": true, "INNER": true, "LEFT": true,
		"RIGHT": true, "FULL": true, "UNION": true, "INTERSECT": true, "EXCEPT": true, "AS": true,
		"ON": true, "AND": true, "OR": true, "NOT": true, "NULL": true, "TRUE": true, "FALSE": true,
		"PRIMARY": true, "KEY": true, "FOREIGN": true, "REFERENCES": true, "UNIQUE": true,
		"CONSTRAINT": true, "DEFAULT": true, "CHECK": true, "GRANT": true, "REVOKE": true,
	}
)

// SQL column type validation
var (
	// Valid PostgreSQL column types
	postgreSQLColumnTypes = map[string]bool{
		"SMALLINT": true, "INTEGER": true, "BIGINT": true, "DECIMAL": true, "NUMERIC": true,
		"REAL": true, "DOUBLE PRECISION": true, "SMALLSERIAL": true, "SERIAL": true, "BIGSERIAL": true,
		"MONEY": true, "CHARACTER VARYING": true, "VARCHAR": true, "CHARACTER": true, "CHAR": true,
		"TEXT": true, "BYTEA": true, "TIMESTAMP": true, "TIMESTAMPTZ": true, "DATE": true,
		"TIME": true, "TIMETZ": true, "INTERVAL": true, "BOOLEAN": true, "POINT": true,
		"LINE": true, "LSEG": true, "BOX": true, "PATH": true, "POLYGON": true, "CIRCLE": true,
		"CIDR": true, "INET": true, "MACADDR": true, "BIT": true, "BIT VARYING": true,
		"TSVECTOR": true, "TSQUERY": true, "UUID": true, "XML": true, "JSON": true, "JSONB": true,
	}
	
	// Valid SQLite column types
	sqliteColumnTypes = map[string]bool{
		"NULL": true, "INTEGER": true, "REAL": true, "TEXT": true, "BLOB": true,
		"NUMERIC": true, "BOOLEAN": true, "DATE": true, "DATETIME": true,
	}
	
	// Common column type patterns (for VARCHAR(n), DECIMAL(p,s), DOUBLE PRECISION, etc.)
	// Allows multi-word types like 'DOUBLE PRECISION' and 'CHARACTER VARYING'
	columnTypePatternRegex = regexp.MustCompile(`^[A-Z]+(?: [A-Z]+)*(?:\([0-9]+(,[0-9]+)*\))?$`)
)

// System column definitions - consistent across database implementations
const (
	SystemColumnCreatedAt     = "created_at"
	SystemColumnCreatedAtType = "TIMESTAMP DEFAULT CURRENT_TIMESTAMP" // PostgreSQL format
	SystemColumnCreatedAtSQLite = "created_at DATETIME DEFAULT CURRENT_TIMESTAMP" // SQLite format
)

// Time format patterns for quick validation before parsing
// Compiled once at package initialization for better performance
var (
	timeFormatPatterns []struct{
		minLen, maxLen int
		pattern *regexp.Regexp
	}
)

// Initialize time format patterns once at package load
func init() {
	timeFormatPatterns = []struct{
		minLen, maxLen int
		pattern *regexp.Regexp
	}{
		// RFC3339 format: "2006-01-02T15:04:05Z07:00"
		{19, 35, regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)},
		// ISO date format: "2006-01-02"
		{10, 10, regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)},
		// DateTime format: "2006-01-02 15:04:05"
		{19, 19, regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)},
	}
}

// CouldBeTimeFormat performs a quick check to see if a string might be a time format
// This avoids expensive time.Parse calls on obviously non-time strings
func CouldBeTimeFormat(s string) bool {
	if len(s) < 8 || len(s) > 35 { // Reasonable time format length bounds
		return false
	}
	
	for _, pattern := range timeFormatPatterns {
		if len(s) >= pattern.minLen && len(s) <= pattern.maxLen {
			if pattern.pattern.MatchString(s) {
				return true
			}
		}
	}
	
	return false
}

// ValidateSQLIdentifier validates that a string is a safe SQL identifier
func ValidateSQLIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	
	if len(identifier) > MaxPostgreSQLIdentifierLength { // PostgreSQL limit
		return fmt.Errorf("identifier too long (max %d characters): %s", MaxPostgreSQLIdentifierLength, identifier)
	}
	
	if !sqlIdentifierRegex.MatchString(identifier) {
		return fmt.Errorf("invalid identifier format: %s", identifier)
	}
	
	upperIdent := strings.ToUpper(identifier)
	if sqlReservedWords[upperIdent] {
		return fmt.Errorf("identifier is a reserved SQL keyword: %s", identifier)
	}
	
	return nil
}

// ValidateColumnType validates that a column type is safe for the specified database
func ValidateColumnType(columnType string, dbType string) error {
	if columnType == "" {
		return fmt.Errorf("column type cannot be empty")
	}
	
	// Normalize to uppercase for comparison
	upperType := strings.ToUpper(strings.TrimSpace(columnType))
	
	// Check if it matches a pattern (like VARCHAR(255))
	if !columnTypePatternRegex.MatchString(upperType) {
		return fmt.Errorf("invalid column type format: %s", columnType)
	}
	
	// Extract base type (remove parentheses and parameters)
	baseType := strings.Split(upperType, "(")[0]
	
	// Validate against database-specific types
	switch dbType {
	case "postgresql":
		if !postgreSQLColumnTypes[baseType] {
			return fmt.Errorf("unsupported PostgreSQL column type: %s", baseType)
		}
	case "sqlite":
		if !sqliteColumnTypes[baseType] {
			return fmt.Errorf("unsupported SQLite column type: %s", baseType)
		}
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	
	return nil
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
	JSON       JSONOptions       `yaml:"json,omitempty" json:"json,omitempty"`
	CSV        CSVOptions        `yaml:"csv,omitempty" json:"csv,omitempty"`
	PostgreSQL PostgreSQLOptions `yaml:"postgresql,omitempty" json:"postgresql,omitempty"`
	SQLite     SQLiteOptions     `yaml:"sqlite,omitempty" json:"sqlite,omitempty"`
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

// PostgreSQLOptions defines PostgreSQL-specific options
type PostgreSQLOptions struct {
	ConnectionString string            `yaml:"connection_string" json:"connection_string"`
	Table            string            `yaml:"table" json:"table"`
	Schema           string            `yaml:"schema,omitempty" json:"schema,omitempty"`
	BatchSize        int               `yaml:"batch_size,omitempty" json:"batch_size,omitempty"`
	CreateTable      bool              `yaml:"create_table,omitempty" json:"create_table,omitempty"`
	OnConflict       ConflictStrategy  `yaml:"on_conflict,omitempty" json:"on_conflict,omitempty"` // PostgreSQL: ConflictIgnore, ConflictError
	ColumnTypes      map[string]string `yaml:"column_types,omitempty" json:"column_types,omitempty"`
}

// SQLiteOptions defines SQLite-specific options  
type SQLiteOptions struct {
	DatabasePath     string            `yaml:"database_path" json:"database_path"`
	Table            string            `yaml:"table" json:"table"`
	BatchSize        int               `yaml:"batch_size,omitempty" json:"batch_size,omitempty"`
	CreateTable      bool              `yaml:"create_table,omitempty" json:"create_table,omitempty"`
	OnConflict       ConflictStrategy  `yaml:"on_conflict,omitempty" json:"on_conflict,omitempty"` // SQLite: ConflictIgnore, ConflictReplace, ConflictError
	ColumnTypes      map[string]string `yaml:"column_types,omitempty" json:"column_types,omitempty"`
	OptimizeOnClose  bool              `yaml:"optimize_on_close,omitempty" json:"optimize_on_close,omitempty"` // Run VACUUM and PRAGMA optimize on close
	ConnectionParams string            `yaml:"connection_params,omitempty" json:"connection_params,omitempty"` // SQLite connection parameters
}

// SupportedFormats lists all supported output formats
var SupportedFormats = []string{
	"json",
	"csv",
	"xml",
	"yaml",
	"txt",
	"html",
	"jsonl",     // JSON Lines
	"postgresql", // PostgreSQL database
	"sqlite",    // SQLite database
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
	"postgresql": {
		Format: FormatPostgreSQL,
		Options: map[string]string{
			"table":        "scraped_data",
			"schema":       "public",
			"batch_size":   "1000",
			"create_table": "true",
			"on_conflict":  string(ConflictIgnore),
		},
	},
	"sqlite": {
		Format: FormatSQLite,
		Options: map[string]string{
			"table":        "scraped_data",
			"batch_size":   "1000",
			"create_table": "true",
			"on_conflict":  string(ConflictIgnore),
		},
	},
}
