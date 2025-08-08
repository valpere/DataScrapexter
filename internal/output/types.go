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
	FormatExcel      OutputFormat = "excel"
	FormatParquet    OutputFormat = "parquet"
	FormatPDF        OutputFormat = "pdf"
	FormatMongoDB    OutputFormat = "mongodb"
	FormatMySQL      OutputFormat = "mysql"
	FormatPostgreSQL OutputFormat = "postgresql"
	FormatSQLite     OutputFormat = "sqlite"
)

// ConflictStrategy defines strategies for handling conflicts during database operations,
// such as inserting or updating rows. Conflict strategies determine how the database
// should behave when a constraint violation (e.g., unique key conflict) occurs.
//
// Supported strategies include:
//   - ConflictIgnore: Ignore the conflicting row (e.g., ON CONFLICT DO NOTHING, INSERT OR IGNORE).
//   - ConflictError:  Fail the operation on conflict (default behavior).
//   - ConflictReplace: Replace the existing row (SQLite only).
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
	return []OutputFormat{FormatJSON, FormatCSV, FormatXML, FormatYAML, FormatTSV, FormatExcel, FormatParquet, FormatPDF, FormatMongoDB, FormatMySQL, FormatPostgreSQL, FormatSQLite}
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

	// Reserved SQL keywords for PostgreSQL (from https://www.postgresql.org/docs/current/sql-keywords-appendix.html)
	postgresReservedWords = map[string]bool{
		"ALL": true, "ANALYSE": true, "ANALYZE": true, "AND": true, "ANY": true, "ARRAY": true, "AS": true, "ASC": true,
		"ASYMMETRIC": true, "AUTHORIZATION": true, "BINARY": true, "BOTH": true, "CASE": true, "CAST": true, "CHECK": true,
		"COLLATE": true, "COLLATION": true, "COLUMN": true, "CONCURRENTLY": true, "CONSTRAINT": true, "CREATE": true,
		"CROSS": true, "CURRENT_CATALOG": true, "CURRENT_DATE": true, "CURRENT_ROLE": true, "CURRENT_SCHEMA": true,
		"CURRENT_TIME": true, "CURRENT_TIMESTAMP": true, "CURRENT_USER": true, "DEFAULT": true, "DEFERRABLE": true,
		"DESC": true, "DISTINCT": true, "DO": true, "ELSE": true, "END": true, "EXCEPT": true, "FALSE": true, "FETCH": true,
		"FOR": true, "FOREIGN": true, "FREEZE": true, "FROM": true, "FULL": true, "GRANT": true, "GROUP": true, "HAVING": true,
		"ILIKE": true, "IN": true, "INITIALLY": true, "INNER": true, "INTERSECT": true, "INTO": true, "IS": true, "ISNULL": true,
		"JOIN": true, "LATERAL": true, "LEADING": true, "LEFT": true, "LIKE": true, "LIMIT": true, "LOCALTIME": true,
		"LOCALTIMESTAMP": true, "NATURAL": true, "NOT": true, "NOTNULL": true, "NULL": true, "OFFSET": true, "ON": true,
		"ONLY": true, "OR": true, "ORDER": true, "OUTER": true, "OVERLAPS": true, "PLACING": true, "PRIMARY": true,
		"REFERENCES": true, "RETURNING": true, "RIGHT": true, "SELECT": true, "SESSION_USER": true, "SIMILAR": true,
		"SOME": true, "SYMMETRIC": true, "TABLE": true, "TABLESAMPLE": true, "THEN": true, "TO": true, "TRAILING": true,
		"TRUE": true, "UNION": true, "UNIQUE": true, "USER": true, "USING": true, "VARIADIC": true, "VERBOSE": true,
		"WHEN": true, "WHERE": true, "WINDOW": true, "WITH": true,
	}

	// Reserved SQL keywords for SQLite (from https://www.sqlite.org/lang_keywords.html)
	sqliteReservedWords = map[string]bool{
		"ABORT": true, "ACTION": true, "ADD": true, "AFTER": true, "ALL": true, "ALTER": true, "ANALYZE": true, "AND": true,
		"AS": true, "ASC": true, "ATTACH": true, "AUTOINCREMENT": true, "BEFORE": true, "BEGIN": true, "BETWEEN": true,
		"BY": true, "CASCADE": true, "CASE": true, "CAST": true, "CHECK": true, "COLLATE": true, "COLUMN": true,
		"COMMIT": true, "CONFLICT": true, "CONSTRAINT": true, "CREATE": true, "CROSS": true, "CURRENT": true,
		"CURRENT_DATE": true, "CURRENT_TIME": true, "CURRENT_TIMESTAMP": true, "DATABASE": true, "DEFAULT": true,
		"DEFERRABLE": true, "DEFERRED": true, "DELETE": true, "DESC": true, "DETACH": true, "DISTINCT": true,
		"DROP": true, "EACH": true, "ELSE": true, "END": true, "ESCAPE": true, "EXCEPT": true, "EXCLUSIVE": true,
		"EXISTS": true, "EXPLAIN": true, "FAIL": true, "FOR": true, "FOREIGN": true, "FROM": true, "FULL": true,
		"GLOB": true, "GROUP": true, "HAVING": true, "IF": true, "IGNORE": true, "IMMEDIATE": true, "IN": true,
		"INDEX": true, "INDEXED": true, "INITIALLY": true, "INNER": true, "INSERT": true, "INSTEAD": true, "INTERSECT": true,
		"INTO": true, "IS": true, "ISNULL": true, "JOIN": true, "KEY": true, "LEFT": true, "LIKE": true, "LIMIT": true,
		"MATCH": true, "NATURAL": true, "NO": true, "NOT": true, "NOTNULL": true, "NULL": true, "OF": true, "OFFSET": true,
		"ON": true, "OR": true, "ORDER": true, "OUTER": true, "PLAN": true, "PRAGMA": true, "PRIMARY": true, "QUERY": true,
		"RAISE": true, "RECURSIVE": true, "REFERENCES": true, "REGEXP": true, "REINDEX": true, "RELEASE": true,
		"RENAME": true, "REPLACE": true, "RESTRICT": true, "RIGHT": true, "ROLLBACK": true, "ROW": true, "SAVEPOINT": true,
		"SELECT": true, "SET": true, "TABLE": true, "TEMP": true, "TEMPORARY": true, "THEN": true, "TO": true, "TRANSACTION": true,
		"TRIGGER": true, "UNION": true, "UNIQUE": true, "UPDATE": true, "USING": true, "VACUUM": true, "VALUES": true,
		"VIEW": true, "VIRTUAL": true, "WHEN": true, "WHERE": true, "WITH": true, "WITHOUT": true,
	}
)

// GetReservedWords returns the reserved word set for the given SQL dialect.
// Supported dialects: "postgresql", "sqlite". Defaults to PostgreSQL if unknown.
func GetReservedWords(dialect string) map[string]bool {
	switch strings.ToLower(dialect) {
	case "sqlite":
		return sqliteReservedWords
	case "postgresql":
		fallthrough
	default:
		return postgresReservedWords
	}
}

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
	SystemColumnCreatedAt           = "created_at"
	SystemColumnCreatedAtType       = "TIMESTAMP DEFAULT CURRENT_TIMESTAMP" // PostgreSQL format
	SystemColumnCreatedAtSQLiteName = "created_at"                          // SQLite column name
	SystemColumnCreatedAtSQLiteType = "DATETIME DEFAULT CURRENT_TIMESTAMP"  // SQLite column type

	// Database-specific limits
	MaxPostgreSQLIdentifierLength = 63  // PostgreSQL maximum identifier length
	MaxSQLiteIdentifierLength     = 999 // SQLite maximum identifier length (much higher than PostgreSQL)
	
	// Database column type inference constants - shared across database implementations
	VarcharLengthMultiplier = 2 // Multiplier for VARCHAR length to provide extra space
)

// Time format patterns for quick validation before parsing
// Compiled once at package initialization for better performance
var (
	compiledTimeFormatPatterns []struct {
		minLen, maxLen int
		pattern        *regexp.Regexp
	}
)

// Initialize time format patterns once at package load
func init() {
	compiledTimeFormatPatterns = []struct {
		minLen, maxLen int
		pattern        *regexp.Regexp
	}{
		// RFC3339 format: "2006-01-02T15:04:05Z07:00"
		{19, 35, regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)},
		// ISO date format: "2006-01-02"
		{10, 10, regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)},
		// DateTime format: "2006-01-02 15:04:05"
		{19, 19, regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)},
	}
}

// HasTimeFormatPattern performs a quick check to see if a string might be a time format
// This avoids expensive time.Parse calls on obviously non-time strings
func HasTimeFormatPattern(s string) bool {
	if len(s) < 8 || len(s) > 35 { // Reasonable time format length bounds
		return false
	}

	for _, pattern := range compiledTimeFormatPatterns {
		if len(s) >= pattern.minLen && len(s) <= pattern.maxLen {
			if pattern.pattern.MatchString(s) {
				return true
			}
		}
	}

	return false
}

// ValidateSQLIdentifier validates that a string is a safe SQL identifier using PostgreSQL limits
// For database-specific validation, use ValidatePostgreSQLIdentifier or ValidateSQLiteIdentifier
func ValidateSQLIdentifier(identifier string) error {
	return ValidatePostgreSQLIdentifier(identifier) // Default to more restrictive PostgreSQL limits
}

// ValidatePostgreSQLIdentifier validates PostgreSQL-specific identifier constraints
func ValidatePostgreSQLIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}

	if len(identifier) > MaxPostgreSQLIdentifierLength {
		return fmt.Errorf("identifier too long (max %d characters): %s", MaxPostgreSQLIdentifierLength, identifier)
	}

	if !sqlIdentifierRegex.MatchString(identifier) {
		return fmt.Errorf("invalid identifier format: %s", identifier)
	}

	upperIdent := strings.ToUpper(identifier)
	if postgresReservedWords[upperIdent] {
		return fmt.Errorf("identifier is a reserved SQL keyword: %s", identifier)
	}

	return nil
}

// ValidateSQLiteIdentifier validates SQLite-specific identifier constraints
func ValidateSQLiteIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}

	if len(identifier) > MaxSQLiteIdentifierLength {
		return fmt.Errorf("identifier too long (max %d characters): %s", MaxSQLiteIdentifierLength, identifier)
	}

	if !sqlIdentifierRegex.MatchString(identifier) {
		return fmt.Errorf("invalid identifier format: %s", identifier)
	}

	upperIdent := strings.ToUpper(identifier)
	if sqliteReservedWords[upperIdent] {
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

// ColumnDefinitionBuilder assists in constructing column definitions for SQL CREATE TABLE statements.
//
// It is designed to support multiple database backends (such as PostgreSQL and SQLite) by managing
// column names, types, user-defined types, and identifier quoting. The struct's fields allow you to
// specify the target database type (DBType), the list of column names (Columns), a mapping of column
// names to their SQL types (ColumnTypes), any user-defined types (UserTypes), and a function for
// quoting identifiers (QuoteFunc) according to the database's requirements.
//
// Use the BuildColumnDefinitions method to generate a slice of column definition strings suitable
// for inclusion in a CREATE TABLE statement, with appropriate validation and quoting.
type ColumnDefinitionBuilder struct {
	DBType      string
	Columns     []string
	ColumnTypes map[string]string
	UserTypes   map[string]string
	QuoteFunc   func(string) string
}

// BuildColumnDefinitions builds column definitions for CREATE TABLE statements
// Returns the column definitions slice and any validation errors
func (cdb *ColumnDefinitionBuilder) BuildColumnDefinitions() ([]string, error) {
	var columnDefs []string

	for _, column := range cdb.Columns {
		// Validate column name for SQL safety based on database type
		var err error
		switch cdb.DBType {
		case "postgresql":
			err = ValidatePostgreSQLIdentifier(column)
		case "sqlite":
			err = ValidateSQLiteIdentifier(column)
		default:
			err = ValidateSQLIdentifier(column) // Default validation
		}

		if err != nil {
			return nil, fmt.Errorf("invalid column name '%s': %w", column, err)
		}

		columnType := cdb.ColumnTypes[column]
		// Override with user-specified types if provided
		if userType, exists := cdb.UserTypes[column]; exists {
			// Validate user-specified column type
			if err := ValidateColumnType(userType, cdb.DBType); err != nil {
				return nil, fmt.Errorf("invalid column type for column '%s': %w", column, err)
			}
			columnType = userType
		}

		// Use the provided quote function to quote identifiers
		quotedColumn := column
		if cdb.QuoteFunc != nil {
			quotedColumn = cdb.QuoteFunc(column)
		}

		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", quotedColumn, columnType))
	}

	return columnDefs, nil
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
	case FormatExcel:
		return ".xlsx"
	case FormatParquet:
		return ".parquet"
	case FormatPDF:
		return ".pdf"
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
	case FormatExcel:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case FormatParquet:
		return "application/octet-stream"
	case FormatPDF:
		return "application/pdf"
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
	TotalRecords    int            `json:"total_records"`
	TotalFiles      int            `json:"total_files"`
	TotalSize       int64          `json:"total_size"`
	ProcessingTime  time.Duration  `json:"processing_time"`
	AverageFileSize int64          `json:"average_file_size"`
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
	PDF        PDFOptions        `yaml:"pdf,omitempty" json:"pdf,omitempty"`
	MongoDB    MongoDBOptions    `yaml:"mongodb,omitempty" json:"mongodb,omitempty"`
	MySQL      MySQLOptions      `yaml:"mysql,omitempty" json:"mysql,omitempty"`
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

// PDFOptions defines PDF-specific options
type PDFOptions struct {
	Title          string            `yaml:"title,omitempty" json:"title,omitempty"`
	Author         string            `yaml:"author,omitempty" json:"author,omitempty"`
	Subject        string            `yaml:"subject,omitempty" json:"subject,omitempty"`
	Keywords       []string          `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	PageSize       string            `yaml:"page_size,omitempty" json:"page_size,omitempty"`       // A4, Letter, Legal, etc.
	Orientation    string            `yaml:"orientation,omitempty" json:"orientation,omitempty"`   // Portrait, Landscape
	Margins        *PDFMargins       `yaml:"margins,omitempty" json:"margins,omitempty"`
	Font           *PDFFont          `yaml:"font,omitempty" json:"font,omitempty"`
	HeaderFooter   *PDFHeaderFooter  `yaml:"header_footer,omitempty" json:"header_footer,omitempty"`
	Watermark      *PDFWatermark     `yaml:"watermark,omitempty" json:"watermark,omitempty"`
	Template       string            `yaml:"template,omitempty" json:"template,omitempty"`         // report, table, detailed, compact
	IncludeCharts  bool              `yaml:"include_charts,omitempty" json:"include_charts,omitempty"`
	IncludeImages  bool              `yaml:"include_images,omitempty" json:"include_images,omitempty"`
	CompressImages bool              `yaml:"compress_images,omitempty" json:"compress_images,omitempty"`
	TableOptions   *PDFTableOptions  `yaml:"table_options,omitempty" json:"table_options,omitempty"`
	Colors         *PDFColorScheme   `yaml:"colors,omitempty" json:"colors,omitempty"`
	Security       *PDFSecurity      `yaml:"security,omitempty" json:"security,omitempty"`
	CustomFields   map[string]string `yaml:"custom_fields,omitempty" json:"custom_fields,omitempty"`
}

// PDFMargins defines PDF page margins
type PDFMargins struct {
	Top    float64 `yaml:"top" json:"top"`       // Points (1 point = 1/72 inch)
	Bottom float64 `yaml:"bottom" json:"bottom"`
	Left   float64 `yaml:"left" json:"left"`
	Right  float64 `yaml:"right" json:"right"`
}

// PDFFont defines PDF font settings
type PDFFont struct {
	Family   string  `yaml:"family,omitempty" json:"family,omitempty"`     // Helvetica, Times, Courier
	Size     float64 `yaml:"size,omitempty" json:"size,omitempty"`         // Font size in points
	Style    string  `yaml:"style,omitempty" json:"style,omitempty"`       // Regular, Bold, Italic, BoldItalic
	Color    string  `yaml:"color,omitempty" json:"color,omitempty"`       // Hex color #000000
	LineHeight float64 `yaml:"line_height,omitempty" json:"line_height,omitempty"` // Line height multiplier
}

// PDFHeaderFooter defines PDF header and footer settings
type PDFHeaderFooter struct {
	Header *PDFHeaderFooterContent `yaml:"header,omitempty" json:"header,omitempty"`
	Footer *PDFHeaderFooterContent `yaml:"footer,omitempty" json:"footer,omitempty"`
}

// PDFHeaderFooterContent defines header/footer content
type PDFHeaderFooterContent struct {
	Text         string            `yaml:"text,omitempty" json:"text,omitempty"`
	Font         *PDFFont          `yaml:"font,omitempty" json:"font,omitempty"`
	Alignment    string            `yaml:"alignment,omitempty" json:"alignment,omitempty"`     // left, center, right
	ShowPageNum  bool              `yaml:"show_page_num,omitempty" json:"show_page_num,omitempty"`
	ShowDate     bool              `yaml:"show_date,omitempty" json:"show_date,omitempty"`
	DateFormat   string            `yaml:"date_format,omitempty" json:"date_format,omitempty"` // 2006-01-02 15:04:05
	Variables    map[string]string `yaml:"variables,omitempty" json:"variables,omitempty"`     // Custom variables like {{title}}, {{url}}
}

// PDFWatermark defines PDF watermark settings
type PDFWatermark struct {
	Text      string  `yaml:"text,omitempty" json:"text,omitempty"`
	Opacity   float64 `yaml:"opacity,omitempty" json:"opacity,omitempty"`       // 0.0 to 1.0
	Rotation  float64 `yaml:"rotation,omitempty" json:"rotation,omitempty"`     // Degrees
	Font      *PDFFont `yaml:"font,omitempty" json:"font,omitempty"`
	Position  string  `yaml:"position,omitempty" json:"position,omitempty"`     // center, diagonal, corners
}

// PDFTableOptions defines PDF table formatting options
type PDFTableOptions struct {
	HeaderStyle    *PDFTableStyle `yaml:"header_style,omitempty" json:"header_style,omitempty"`
	RowStyle       *PDFTableStyle `yaml:"row_style,omitempty" json:"row_style,omitempty"`
	AlternateStyle *PDFTableStyle `yaml:"alternate_style,omitempty" json:"alternate_style,omitempty"`
	BorderWidth    float64        `yaml:"border_width,omitempty" json:"border_width,omitempty"`
	BorderColor    string         `yaml:"border_color,omitempty" json:"border_color,omitempty"`
	CellPadding    float64        `yaml:"cell_padding,omitempty" json:"cell_padding,omitempty"`
	AutoResize     bool           `yaml:"auto_resize,omitempty" json:"auto_resize,omitempty"`
	BreakPages     bool           `yaml:"break_pages,omitempty" json:"break_pages,omitempty"`
	MaxCellHeight  float64        `yaml:"max_cell_height,omitempty" json:"max_cell_height,omitempty"`
}

// PDFTableStyle defines styling for table elements
type PDFTableStyle struct {
	BackgroundColor string   `yaml:"background_color,omitempty" json:"background_color,omitempty"`
	TextColor       string   `yaml:"text_color,omitempty" json:"text_color,omitempty"`
	Font            *PDFFont `yaml:"font,omitempty" json:"font,omitempty"`
	Alignment       string   `yaml:"alignment,omitempty" json:"alignment,omitempty"` // left, center, right
}

// PDFColorScheme defines color scheme for PDF documents
type PDFColorScheme struct {
	Primary     string `yaml:"primary,omitempty" json:"primary,omitempty"`         // Main theme color
	Secondary   string `yaml:"secondary,omitempty" json:"secondary,omitempty"`     // Secondary theme color
	Accent      string `yaml:"accent,omitempty" json:"accent,omitempty"`           // Accent color
	Text        string `yaml:"text,omitempty" json:"text,omitempty"`               // Main text color
	Background  string `yaml:"background,omitempty" json:"background,omitempty"`   // Background color
	Border      string `yaml:"border,omitempty" json:"border,omitempty"`           // Border color
	Success     string `yaml:"success,omitempty" json:"success,omitempty"`         // Success indicator color
	Warning     string `yaml:"warning,omitempty" json:"warning,omitempty"`         // Warning indicator color
	Error       string `yaml:"error,omitempty" json:"error,omitempty"`             // Error indicator color
}

// PDFSecurity defines PDF security and protection settings
type PDFSecurity struct {
	UserPassword  string   `yaml:"user_password,omitempty" json:"user_password,omitempty"`     // Password to open PDF
	OwnerPassword string   `yaml:"owner_password,omitempty" json:"owner_password,omitempty"`   // Password for editing
	Permissions   []string `yaml:"permissions,omitempty" json:"permissions,omitempty"`         // print, copy, modify, annotate
	Encryption    string   `yaml:"encryption,omitempty" json:"encryption,omitempty"`           // 40bit, 128bit, 256bit
}

// PostgreSQLOptions defines PostgreSQL-specific options
type PostgreSQLOptions struct {
	ConnectionString     string            `yaml:"connection_string" json:"connection_string"`
	Table                string            `yaml:"table" json:"table"`
	Schema               string            `yaml:"schema,omitempty" json:"schema,omitempty"`
	BatchSize            int               `yaml:"batch_size,omitempty" json:"batch_size,omitempty"`
	TransactionThreshold int               `yaml:"transaction_threshold,omitempty" json:"transaction_threshold,omitempty"`
	CreateTable          bool              `yaml:"create_table,omitempty" json:"create_table,omitempty"`
	OnConflict           ConflictStrategy  `yaml:"on_conflict,omitempty" json:"on_conflict,omitempty"` // PostgreSQL: ConflictIgnore, ConflictError
	ColumnTypes          map[string]string `yaml:"column_types,omitempty" json:"column_types,omitempty"`
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
	"pdf",        // PDF documents
	"jsonl",      // JSON Lines
	"mongodb",    // MongoDB database
	"mysql",      // MySQL database
	"postgresql", // PostgreSQL database
	"sqlite",     // SQLite database
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
	"pdf": {
		Format: FormatPDF,
		Options: map[string]string{
			"template":     "report",
			"page_size":    "A4",
			"orientation":  "Portrait",
			"title":        "DataScrapexter Report",
			"author":       "DataScrapexter",
		},
	},
	"mongodb": {
		Format: FormatMongoDB,
		Options: map[string]string{
			"database":     "scraped_data",
			"collection":   "records",
			"batch_size":   "1000",
			"on_conflict":  string(ConflictIgnore),
		},
	},
	"mysql": {
		Format: FormatMySQL,
		Options: map[string]string{
			"database":     "scraped_data",
			"table":        "records",
			"batch_size":   "1000",
			"engine":       "InnoDB",
			"charset":      "utf8mb4",
			"create_table": "true",
			"on_conflict":  string(ConflictIgnore),
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
