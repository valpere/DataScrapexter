// internal/output/mysql.go - MySQL database connector with advanced features
package output

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/valpere/DataScrapexter/internal/utils"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

var mysqlLogger = utils.NewComponentLogger("mysql-output")

// MySQLWriter implements Writer interface for MySQL output
type MySQLWriter struct {
	db         *sql.DB
	config     MySQLOptions
	table      string
	database   string
	columns    []string
	columnTypes map[string]string
	buffer     []map[string]interface{}
	metadata   *MySQLMetadata
	
	// Prepared statements for performance
	insertStmt    *sql.Stmt
	upsertStmt    *sql.Stmt
	
	// Connection state
	connected     bool
	
	// Statistics
	totalWrites   int64
	totalErrors   int64
	lastWriteTime time.Time
	startTime     time.Time
}

// MySQLOptions defines MySQL-specific configuration options
type MySQLOptions struct {
	ConnectionString    string            `yaml:"connection_string" json:"connection_string"`
	Database            string            `yaml:"database" json:"database"`
	Table               string            `yaml:"table" json:"table"`
	BatchSize           int               `yaml:"batch_size,omitempty" json:"batch_size,omitempty"`
	CreateTable         bool              `yaml:"create_table,omitempty" json:"create_table,omitempty"`
	CreateDatabase      bool              `yaml:"create_database,omitempty" json:"create_database,omitempty"`
	OnConflict          ConflictStrategy  `yaml:"on_conflict,omitempty" json:"on_conflict,omitempty"`
	ColumnTypes         map[string]string `yaml:"column_types,omitempty" json:"column_types,omitempty"`
	Engine              string            `yaml:"engine,omitempty" json:"engine,omitempty"`               // InnoDB, MyISAM
	Charset             string            `yaml:"charset,omitempty" json:"charset,omitempty"`             // utf8mb4, utf8
	Collation           string            `yaml:"collation,omitempty" json:"collation,omitempty"`         // utf8mb4_unicode_ci
	MaxConnections      int               `yaml:"max_connections,omitempty" json:"max_connections,omitempty"`
	MaxIdleConnections  int               `yaml:"max_idle_connections,omitempty" json:"max_idle_connections,omitempty"`
	ConnMaxLifetime     time.Duration     `yaml:"conn_max_lifetime,omitempty" json:"conn_max_lifetime,omitempty"`
	ConnMaxIdleTime     time.Duration     `yaml:"conn_max_idle_time,omitempty" json:"conn_max_idle_time,omitempty"`
	Timeout             time.Duration     `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	ReadTimeout         time.Duration     `yaml:"read_timeout,omitempty" json:"read_timeout,omitempty"`
	WriteTimeout        time.Duration     `yaml:"write_timeout,omitempty" json:"write_timeout,omitempty"`
	TransactionIsolation string           `yaml:"transaction_isolation,omitempty" json:"transaction_isolation,omitempty"`
	Indexes             []MySQLIndexSpec  `yaml:"indexes,omitempty" json:"indexes,omitempty"`
	ForeignKeys         []MySQLForeignKey `yaml:"foreign_keys,omitempty" json:"foreign_keys,omitempty"`
	Constraints         []MySQLConstraint `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	Partitioning        *MySQLPartition   `yaml:"partitioning,omitempty" json:"partitioning,omitempty"`
	TLS                 *MySQLTLSOptions  `yaml:"tls,omitempty" json:"tls,omitempty"`
	BulkInsert          bool              `yaml:"bulk_insert,omitempty" json:"bulk_insert,omitempty"`
	DisableFK           bool              `yaml:"disable_fk,omitempty" json:"disable_fk,omitempty"`      // Disable foreign key checks
	AutoCommit          bool              `yaml:"auto_commit" json:"auto_commit"`
	TransactionSize     int               `yaml:"transaction_size,omitempty" json:"transaction_size,omitempty"`
	OptimizeAfterInsert bool              `yaml:"optimize_after_insert,omitempty" json:"optimize_after_insert,omitempty"`
	CompressionLevel    int               `yaml:"compression_level,omitempty" json:"compression_level,omitempty"`
}

// MySQLIndexSpec defines MySQL index specification
type MySQLIndexSpec struct {
	Name      string   `yaml:"name" json:"name"`
	Type      string   `yaml:"type,omitempty" json:"type,omitempty"`           // BTREE, HASH, FULLTEXT, SPATIAL
	Columns   []string `yaml:"columns" json:"columns"`
	Unique    bool     `yaml:"unique,omitempty" json:"unique,omitempty"`
	Length    map[string]int `yaml:"length,omitempty" json:"length,omitempty"` // Column prefix lengths
	Algorithm string   `yaml:"algorithm,omitempty" json:"algorithm,omitempty"` // DEFAULT, INPLACE, COPY
	Lock      string   `yaml:"lock,omitempty" json:"lock,omitempty"`           // DEFAULT, NONE, SHARED, EXCLUSIVE
	Comment   string   `yaml:"comment,omitempty" json:"comment,omitempty"`
}

// MySQLForeignKey defines MySQL foreign key specification
type MySQLForeignKey struct {
	Name           string `yaml:"name" json:"name"`
	Columns        []string `yaml:"columns" json:"columns"`
	RefTable       string `yaml:"ref_table" json:"ref_table"`
	RefColumns     []string `yaml:"ref_columns" json:"ref_columns"`
	OnDelete       string `yaml:"on_delete,omitempty" json:"on_delete,omitempty"`     // CASCADE, SET NULL, RESTRICT
	OnUpdate       string `yaml:"on_update,omitempty" json:"on_update,omitempty"`     // CASCADE, SET NULL, RESTRICT
}

// MySQLConstraint defines MySQL table constraints
type MySQLConstraint struct {
	Name       string `yaml:"name" json:"name"`
	Type       string `yaml:"type" json:"type"`                               // CHECK, UNIQUE
	Columns    []string `yaml:"columns,omitempty" json:"columns,omitempty"`
	Expression string `yaml:"expression,omitempty" json:"expression,omitempty"`
}

// MySQLPartition defines MySQL table partitioning
type MySQLPartition struct {
	Type       string                 `yaml:"type" json:"type"`                       // RANGE, LIST, HASH, KEY
	Expression string                 `yaml:"expression,omitempty" json:"expression,omitempty"`
	Partitions []MySQLPartitionSpec   `yaml:"partitions" json:"partitions"`
}

// MySQLPartitionSpec defines individual partition specification
type MySQLPartitionSpec struct {
	Name     string      `yaml:"name" json:"name"`
	Values   interface{} `yaml:"values,omitempty" json:"values,omitempty"`       // For RANGE/LIST
	Comment  string      `yaml:"comment,omitempty" json:"comment,omitempty"`
	Engine   string      `yaml:"engine,omitempty" json:"engine,omitempty"`
}

// MySQLTLSOptions defines MySQL TLS/SSL configuration
type MySQLTLSOptions struct {
	Enabled            bool   `yaml:"enabled" json:"enabled"`
	CertificateFile    string `yaml:"certificate_file,omitempty" json:"certificate_file,omitempty"`
	PrivateKeyFile     string `yaml:"private_key_file,omitempty" json:"private_key_file,omitempty"`
	CAFile             string `yaml:"ca_file,omitempty" json:"ca_file,omitempty"`
	ServerName         string `yaml:"server_name,omitempty" json:"server_name,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify,omitempty" json:"insecure_skip_verify,omitempty"`
	MinVersion         string `yaml:"min_version,omitempty" json:"min_version,omitempty"`        // TLS1.0, TLS1.1, TLS1.2, TLS1.3
	CipherSuites       []string `yaml:"cipher_suites,omitempty" json:"cipher_suites,omitempty"`
}

// MySQLMetadata contains metadata about MySQL operations
type MySQLMetadata struct {
	Database        string              `json:"database"`
	Table           string              `json:"table"`
	ServerInfo      *MySQLServerInfo    `json:"server_info,omitempty"`
	TableInfo       *MySQLTableInfo     `json:"table_info,omitempty"`
	IndexInfo       []MySQLIndexInfo    `json:"index_info,omitempty"`
	EngineInfo      *MySQLEngineInfo    `json:"engine_info,omitempty"`
	ConnectionState string              `json:"connection_state"`
	Variables       map[string]string   `json:"variables,omitempty"`
}

// MySQLServerInfo contains MySQL server information
type MySQLServerInfo struct {
	Version         string `json:"version"`
	VersionComment  string `json:"version_comment,omitempty"`
	Protocol        int    `json:"protocol"`
	DataDir         string `json:"data_dir,omitempty"`
	BaseDir         string `json:"base_dir,omitempty"`
	PluginDir       string `json:"plugin_dir,omitempty"`
	CharacterSet    string `json:"character_set,omitempty"`
	Collation       string `json:"collation,omitempty"`
	TimeZone        string `json:"time_zone,omitempty"`
}

// MySQLTableInfo contains table metadata
type MySQLTableInfo struct {
	Engine          string    `json:"engine"`
	Version         int       `json:"version"`
	RowFormat       string    `json:"row_format"`
	Rows            int64     `json:"rows"`
	AvgRowLength    int64     `json:"avg_row_length"`
	DataLength      int64     `json:"data_length"`
	MaxDataLength   int64     `json:"max_data_length"`
	IndexLength     int64     `json:"index_length"`
	DataFree        int64     `json:"data_free"`
	AutoIncrement   int64     `json:"auto_increment,omitempty"`
	CreateTime      time.Time `json:"create_time"`
	UpdateTime      time.Time `json:"update_time,omitempty"`
	CheckTime       time.Time `json:"check_time,omitempty"`
	Collation       string    `json:"collation"`
	Checksum        int64     `json:"checksum,omitempty"`
	Comment         string    `json:"comment,omitempty"`
}

// MySQLIndexInfo contains index metadata
type MySQLIndexInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Unique      bool   `json:"unique"`
	Columns     []string `json:"columns"`
	Cardinality int64  `json:"cardinality"`
	Comment     string `json:"comment,omitempty"`
}

// MySQLEngineInfo contains storage engine information
type MySQLEngineInfo struct {
	Name        string            `json:"name"`
	Support     string            `json:"support"`
	Comment     string            `json:"comment,omitempty"`
	Transactions string           `json:"transactions,omitempty"`
	XA          string            `json:"xa,omitempty"`
	Savepoints  string            `json:"savepoints,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
}

// NewMySQLWriter creates a new MySQL writer
func NewMySQLWriter(options MySQLOptions) (*MySQLWriter, error) {
	if options.ConnectionString == "" {
		return nil, fmt.Errorf("MySQL connection string is required")
	}
	if options.Database == "" {
		return nil, fmt.Errorf("MySQL database name is required")
	}
	if options.Table == "" {
		return nil, fmt.Errorf("MySQL table name is required")
	}

	// Set defaults
	if options.BatchSize == 0 {
		options.BatchSize = 1000
	}
	if options.Engine == "" {
		options.Engine = "InnoDB"
	}
	if options.Charset == "" {
		options.Charset = "utf8mb4"
	}
	if options.Collation == "" {
		options.Collation = "utf8mb4_unicode_ci"
	}
	if options.MaxConnections == 0 {
		options.MaxConnections = 100
	}
	if options.MaxIdleConnections == 0 {
		options.MaxIdleConnections = 10
	}
	if options.ConnMaxLifetime == 0 {
		options.ConnMaxLifetime = time.Hour
	}
	if options.ConnMaxIdleTime == 0 {
		options.ConnMaxIdleTime = 10 * time.Minute
	}
	if options.OnConflict == "" {
		options.OnConflict = ConflictIgnore
	}
	if options.TransactionSize == 0 {
		options.TransactionSize = options.BatchSize
	}
	options.AutoCommit = true // Default to true

	writer := &MySQLWriter{
		config:      options,
		table:       options.Table,
		database:    options.Database,
		buffer:      make([]map[string]interface{}, 0, options.BatchSize),
		columnTypes: make(map[string]string),
		startTime:   time.Now(),
		metadata: &MySQLMetadata{
			Database:        options.Database,
			Table:           options.Table,
			ConnectionState: "disconnected",
		},
	}

	// Connect to MySQL
	if err := writer.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	mysqlLogger.Info(fmt.Sprintf("Connected to MySQL database %s, table %s", 
		options.Database, options.Table))

	return writer, nil
}

// connect establishes connection to MySQL
func (mw *MySQLWriter) connect() error {
	// Parse connection string and add parameters
	connStr := mw.buildConnectionString()

	// Open database connection
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(mw.config.MaxConnections)
	db.SetMaxIdleConns(mw.config.MaxIdleConnections)
	db.SetConnMaxLifetime(mw.config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(mw.config.ConnMaxIdleTime)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	mw.db = db
	mw.connected = true
	mw.metadata.ConnectionState = "connected"

	// Initialize database and table
	if err := mw.initializeDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Gather metadata
	if err := mw.gatherMetadata(); err != nil {
		mysqlLogger.Warn(fmt.Sprintf("Failed to gather metadata: %v", err))
	}

	// Configure session settings
	if err := mw.configureSession(); err != nil {
		mysqlLogger.Warn(fmt.Sprintf("Failed to configure session: %v", err))
	}

	return nil
}

// buildConnectionString builds the final connection string with parameters
func (mw *MySQLWriter) buildConnectionString() string {
	connStr := mw.config.ConnectionString

	// Add parameters if not present
	params := []string{}

	if !strings.Contains(connStr, "charset=") {
		params = append(params, "charset="+mw.config.Charset)
	}

	if !strings.Contains(connStr, "parseTime=") {
		params = append(params, "parseTime=true")
	}

	if !strings.Contains(connStr, "loc=") {
		params = append(params, "loc=UTC")
	}

	// Add timeout parameters
	if mw.config.Timeout > 0 && !strings.Contains(connStr, "timeout=") {
		params = append(params, fmt.Sprintf("timeout=%v", mw.config.Timeout))
	}

	if mw.config.ReadTimeout > 0 && !strings.Contains(connStr, "readTimeout=") {
		params = append(params, fmt.Sprintf("readTimeout=%v", mw.config.ReadTimeout))
	}

	if mw.config.WriteTimeout > 0 && !strings.Contains(connStr, "writeTimeout=") {
		params = append(params, fmt.Sprintf("writeTimeout=%v", mw.config.WriteTimeout))
	}

	// Add TLS configuration
	if mw.config.TLS != nil && mw.config.TLS.Enabled && !strings.Contains(connStr, "tls=") {
		params = append(params, "tls=true")
		
		if mw.config.TLS.InsecureSkipVerify {
			params = append(params, "tls=skip-verify")
		}
	}

	// Append parameters
	if len(params) > 0 {
		separator := "?"
		if strings.Contains(connStr, "?") {
			separator = "&"
		}
		connStr += separator + strings.Join(params, "&")
	}

	return connStr
}

// initializeDatabase creates database and table if needed
func (mw *MySQLWriter) initializeDatabase() error {
	// Create database if requested
	if mw.config.CreateDatabase {
		createDbSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s",
			mw.config.Database, mw.config.Charset, mw.config.Collation)
		
		if _, err := mw.db.Exec(createDbSQL); err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}

		mysqlLogger.Info(fmt.Sprintf("Created database %s", mw.config.Database))
	}

	// Use database
	if _, err := mw.db.Exec(fmt.Sprintf("USE `%s`", mw.config.Database)); err != nil {
		return fmt.Errorf("failed to use database: %w", err)
	}

	// Create table if requested
	if mw.config.CreateTable {
		if err := mw.createTable(nil); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// createTable creates the table with inferred or specified column types
func (mw *MySQLWriter) createTable(sampleData []map[string]interface{}) error {
	// Check if table exists
	var tableName string
	err := mw.db.QueryRow(
		"SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND table_name = ?",
		mw.config.Database, mw.config.Table,
	).Scan(&tableName)

	if err == nil {
		// Table exists, analyze existing structure
		if err := mw.analyzeTableStructure(); err != nil {
			return fmt.Errorf("failed to analyze table structure: %w", err)
		}
		return nil
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	// Table doesn't exist, create it
	if len(sampleData) == 0 && len(mw.config.ColumnTypes) == 0 {
		// Create basic table with ID and timestamp
		mw.columns = []string{"id", "created_at"}
		mw.columnTypes = map[string]string{
			"id":         "BIGINT AUTO_INCREMENT PRIMARY KEY",
			"created_at": "TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		}
	} else {
		// Infer column types from sample data or use configured types
		if err := mw.inferColumnTypes(sampleData); err != nil {
			return fmt.Errorf("failed to infer column types: %w", err)
		}
	}

	// Build CREATE TABLE statement
	createSQL := mw.buildCreateTableSQL()

	// Execute CREATE TABLE
	if _, err := mw.db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	mysqlLogger.Info(fmt.Sprintf("Created table %s with %d columns", mw.config.Table, len(mw.columns)))

	// Create indexes
	if len(mw.config.Indexes) > 0 {
		if err := mw.createIndexes(); err != nil {
			mysqlLogger.Warn(fmt.Sprintf("Failed to create some indexes: %v", err))
		}
	}

	// Create foreign keys
	if len(mw.config.ForeignKeys) > 0 {
		if err := mw.createForeignKeys(); err != nil {
			mysqlLogger.Warn(fmt.Sprintf("Failed to create some foreign keys: %v", err))
		}
	}

	return nil
}

// inferColumnTypes analyzes sample data to determine appropriate column types
func (mw *MySQLWriter) inferColumnTypes(sampleData []map[string]interface{}) error {
	fieldTypes := make(map[string]map[reflect.Kind]int)
	fieldSizes := make(map[string]int)

	// Analyze sample data
	for _, record := range sampleData {
		for field, value := range record {
			if fieldTypes[field] == nil {
				fieldTypes[field] = make(map[reflect.Kind]int)
			}

			if value != nil {
				kind := reflect.TypeOf(value).Kind()
				fieldTypes[field][kind]++

				// Track string lengths for VARCHAR sizing
				if str, ok := value.(string); ok {
					if len(str) > fieldSizes[field] {
						fieldSizes[field] = len(str)
					}
				}
			} else {
				fieldTypes[field][reflect.Invalid]++
			}
		}
	}

	// Convert to MySQL column types
	mw.columnTypes = make(map[string]string)
	mw.columns = make([]string, 0)

	// Add system columns
	mw.columns = append(mw.columns, "id")
	mw.columnTypes["id"] = "BIGINT AUTO_INCREMENT PRIMARY KEY"

	// Add data columns
	for field, types := range fieldTypes {
		// Skip if user provided custom type
		if customType, exists := mw.config.ColumnTypes[field]; exists {
			mw.columnTypes[field] = customType
			mw.columns = append(mw.columns, field)
			continue
		}

		// Infer type from most common type
		var dominantType reflect.Kind
		maxCount := 0
		for kind, count := range types {
			if count > maxCount {
				maxCount = count
				dominantType = kind
			}
		}

		mysqlType := mw.inferMySQLType(dominantType, fieldSizes[field])
		mw.columnTypes[field] = mysqlType
		mw.columns = append(mw.columns, field)
	}

	// Add timestamp column
	mw.columns = append(mw.columns, "created_at")
	mw.columnTypes["created_at"] = "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"

	return nil
}

// inferMySQLType converts Go type to appropriate MySQL type
func (mw *MySQLWriter) inferMySQLType(kind reflect.Kind, maxSize int) string {
	switch kind {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "INT"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INT UNSIGNED"
	case reflect.Uint64:
		return "BIGINT UNSIGNED"
	case reflect.Float32:
		return "FLOAT"
	case reflect.Float64:
		return "DOUBLE"
	case reflect.String:
		if maxSize == 0 {
			maxSize = 255
		}
		// Determine appropriate string type
		if maxSize <= 255 {
			return fmt.Sprintf("VARCHAR(%d)", maxSize*VarcharLengthMultiplier)
		} else if maxSize <= 65535 {
			return "TEXT"
		} else if maxSize <= 16777215 {
			return "MEDIUMTEXT"
		} else {
			return "LONGTEXT"
		}
	case reflect.Slice:
		return "JSON" // For JSON arrays
	case reflect.Map:
		return "JSON" // For JSON objects
	default:
		return "TEXT" // Default fallback
	}
}

// buildCreateTableSQL builds the CREATE TABLE SQL statement
func (mw *MySQLWriter) buildCreateTableSQL() string {
	var sql strings.Builder

	sql.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", mw.config.Table))

	// Add columns
	columnDefs := make([]string, len(mw.columns))
	for i, column := range mw.columns {
		columnType := mw.columnTypes[column]
		columnDefs[i] = fmt.Sprintf("  `%s` %s", column, columnType)
	}

	sql.WriteString(strings.Join(columnDefs, ",\n"))

	// Add constraints
	if len(mw.config.Constraints) > 0 {
		for _, constraint := range mw.config.Constraints {
			switch constraint.Type {
			case "UNIQUE":
				columns := strings.Join(constraint.Columns, "`, `")
				sql.WriteString(fmt.Sprintf(",\n  CONSTRAINT `%s` UNIQUE (`%s`)", 
					constraint.Name, columns))
			case "CHECK":
				sql.WriteString(fmt.Sprintf(",\n  CONSTRAINT `%s` CHECK (%s)", 
					constraint.Name, constraint.Expression))
			}
		}
	}

	sql.WriteString("\n)")

	// Add table options
	sql.WriteString(fmt.Sprintf(" ENGINE=%s", mw.config.Engine))
	sql.WriteString(fmt.Sprintf(" DEFAULT CHARSET=%s", mw.config.Charset))
	sql.WriteString(fmt.Sprintf(" COLLATE=%s", mw.config.Collation))

	// Add partitioning if specified
	if mw.config.Partitioning != nil {
		partSQL := mw.buildPartitioningSQL()
		sql.WriteString(partSQL)
	}

	return sql.String()
}

// buildPartitioningSQL builds partitioning clause
func (mw *MySQLWriter) buildPartitioningSQL() string {
	p := mw.config.Partitioning
	var sql strings.Builder

	sql.WriteString(fmt.Sprintf("\nPARTITION BY %s", p.Type))

	if p.Expression != "" {
		sql.WriteString(fmt.Sprintf(" (%s)", p.Expression))
	}

	if len(p.Partitions) > 0 {
		sql.WriteString("\n(")
		partDefs := make([]string, len(p.Partitions))
		
		for i, part := range p.Partitions {
			partDef := fmt.Sprintf("  PARTITION %s", part.Name)
			
			if part.Values != nil {
				switch p.Type {
				case "RANGE":
					partDef += fmt.Sprintf(" VALUES LESS THAN (%v)", part.Values)
				case "LIST":
					partDef += fmt.Sprintf(" VALUES IN (%v)", part.Values)
				}
			}
			
			if part.Engine != "" {
				partDef += fmt.Sprintf(" ENGINE = %s", part.Engine)
			}
			
			if part.Comment != "" {
				partDef += fmt.Sprintf(" COMMENT = '%s'", strings.ReplaceAll(part.Comment, "'", "''"))
			}
			
			partDefs[i] = partDef
		}
		
		sql.WriteString(strings.Join(partDefs, ",\n"))
		sql.WriteString("\n)")
	}

	return sql.String()
}

// analyzeTableStructure analyzes existing table structure
func (mw *MySQLWriter) analyzeTableStructure() error {
	// Get column information
	rows, err := mw.db.Query(
		"SELECT column_name, column_type FROM information_schema.columns WHERE table_schema = ? AND table_name = ? ORDER BY ordinal_position",
		mw.config.Database, mw.config.Table,
	)
	if err != nil {
		return fmt.Errorf("failed to query table structure: %w", err)
	}
	defer rows.Close()

	mw.columns = make([]string, 0)
	mw.columnTypes = make(map[string]string)

	for rows.Next() {
		var columnName, columnType string
		if err := rows.Scan(&columnName, &columnType); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}

		mw.columns = append(mw.columns, columnName)
		mw.columnTypes[columnName] = columnType
	}

	return rows.Err()
}

// createIndexes creates the specified indexes
func (mw *MySQLWriter) createIndexes() error {
	for _, indexSpec := range mw.config.Indexes {
		indexSQL := mw.buildIndexSQL(indexSpec)
		
		if _, err := mw.db.Exec(indexSQL); err != nil {
			mysqlLogger.Warn(fmt.Sprintf("Failed to create index %s: %v", indexSpec.Name, err))
			continue
		}
		
		mysqlLogger.Info(fmt.Sprintf("Created index %s", indexSpec.Name))
	}
	
	return nil
}

// buildIndexSQL builds index creation SQL
func (mw *MySQLWriter) buildIndexSQL(spec MySQLIndexSpec) string {
	var sql strings.Builder

	// Index type
	if spec.Unique {
		sql.WriteString("CREATE UNIQUE INDEX")
	} else {
		sql.WriteString("CREATE INDEX")
	}

	// Index name
	sql.WriteString(fmt.Sprintf(" `%s` ON `%s`", spec.Name, mw.config.Table))

	// Columns with optional lengths
	columnParts := make([]string, len(spec.Columns))
	for i, col := range spec.Columns {
		if length, hasLength := spec.Length[col]; hasLength {
			columnParts[i] = fmt.Sprintf("`%s`(%d)", col, length)
		} else {
			columnParts[i] = fmt.Sprintf("`%s`", col)
		}
	}
	sql.WriteString(fmt.Sprintf(" (%s)", strings.Join(columnParts, ", ")))

	// Index type hint
	if spec.Type != "" {
		sql.WriteString(fmt.Sprintf(" USING %s", spec.Type))
	}

	// Algorithm and lock options
	options := make([]string, 0)
	if spec.Algorithm != "" {
		options = append(options, fmt.Sprintf("ALGORITHM = %s", spec.Algorithm))
	}
	if spec.Lock != "" {
		options = append(options, fmt.Sprintf("LOCK = %s", spec.Lock))
	}
	
	if len(options) > 0 {
		sql.WriteString(" " + strings.Join(options, " "))
	}

	// Comment
	if spec.Comment != "" {
		sql.WriteString(fmt.Sprintf(" COMMENT '%s'", strings.ReplaceAll(spec.Comment, "'", "''")))
	}

	return sql.String()
}

// createForeignKeys creates the specified foreign keys
func (mw *MySQLWriter) createForeignKeys() error {
	for _, fkSpec := range mw.config.ForeignKeys {
		fkSQL := mw.buildForeignKeySQL(fkSpec)
		
		if _, err := mw.db.Exec(fkSQL); err != nil {
			mysqlLogger.Warn(fmt.Sprintf("Failed to create foreign key %s: %v", fkSpec.Name, err))
			continue
		}
		
		mysqlLogger.Info(fmt.Sprintf("Created foreign key %s", fkSpec.Name))
	}
	
	return nil
}

// buildForeignKeySQL builds foreign key creation SQL
func (mw *MySQLWriter) buildForeignKeySQL(spec MySQLForeignKey) string {
	columns := strings.Join(spec.Columns, "`, `")
	refColumns := strings.Join(spec.RefColumns, "`, `")

	sql := fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s` (`%s`)",
		mw.config.Table, spec.Name, columns, spec.RefTable, refColumns)

	if spec.OnDelete != "" {
		sql += fmt.Sprintf(" ON DELETE %s", spec.OnDelete)
	}
	
	if spec.OnUpdate != "" {
		sql += fmt.Sprintf(" ON UPDATE %s", spec.OnUpdate)
	}

	return sql
}

// configureSession configures session-level MySQL settings
func (mw *MySQLWriter) configureSession() error {
	settings := make(map[string]string)

	// Transaction isolation level
	if mw.config.TransactionIsolation != "" {
		settings["transaction_isolation"] = fmt.Sprintf("'%s'", mw.config.TransactionIsolation)
	}

	// Foreign key checks
	if mw.config.DisableFK {
		settings["foreign_key_checks"] = "0"
	}

	// Auto commit
	if !mw.config.AutoCommit {
		settings["autocommit"] = "0"
	}

	// Apply settings
	for variable, value := range settings {
		query := fmt.Sprintf("SET SESSION %s = %s", variable, value)
		if _, err := mw.db.Exec(query); err != nil {
			mysqlLogger.Warn(fmt.Sprintf("Failed to set %s: %v", variable, err))
		}
	}

	return nil
}

// gatherMetadata collects server and table metadata
func (mw *MySQLWriter) gatherMetadata() error {
	// Get server information
	var version, versionComment string
	if err := mw.db.QueryRow("SELECT VERSION(), @@version_comment").Scan(&version, &versionComment); err == nil {
		mw.metadata.ServerInfo = &MySQLServerInfo{
			Version:        version,
			VersionComment: versionComment,
		}
	}

	// Get table information if table exists
	var engine, rowFormat string
	var rows, dataLength, indexLength int64
	var createTime time.Time

	err := mw.db.QueryRow(`
		SELECT engine, row_format, table_rows, data_length, index_length, create_time 
		FROM information_schema.tables 
		WHERE table_schema = ? AND table_name = ?`,
		mw.config.Database, mw.config.Table,
	).Scan(&engine, &rowFormat, &rows, &dataLength, &indexLength, &createTime)

	if err == nil {
		mw.metadata.TableInfo = &MySQLTableInfo{
			Engine:      engine,
			RowFormat:   rowFormat,
			Rows:        rows,
			DataLength:  dataLength,
			IndexLength: indexLength,
			CreateTime:  createTime,
		}
	}

	return nil
}

// Write writes data to MySQL table
func (mw *MySQLWriter) Write(data []map[string]interface{}) error {
	if !mw.connected {
		return fmt.Errorf("not connected to MySQL")
	}

	// If this is the first write and table needs to be created, use sample data
	if mw.config.CreateTable && len(mw.columns) == 0 {
		if err := mw.createTable(data); err != nil {
			return fmt.Errorf("failed to create table from sample data: %w", err)
		}
	}

	// Buffer data
	for _, record := range data {
		mw.buffer = append(mw.buffer, record)

		// Flush if buffer is full
		if len(mw.buffer) >= mw.config.BatchSize {
			if err := mw.flush(); err != nil {
				return err
			}
		}
	}

	mysqlLogger.Debug(fmt.Sprintf("Buffered %d records (buffer size: %d)", len(data), len(mw.buffer)))
	return nil
}

// flush writes buffered data to MySQL
func (mw *MySQLWriter) flush() error {
	if len(mw.buffer) == 0 {
		return nil
	}

	startTime := time.Now()

	var err error
	if mw.config.BulkInsert {
		err = mw.bulkInsert()
	} else {
		// Handle conflicts based on strategy
		switch mw.config.OnConflict {
		case ConflictIgnore:
			err = mw.insertIgnore()
		case ConflictReplace:
			err = mw.insertReplace()
		default: // ConflictError
			err = mw.insertNormal()
		}
	}

	if err != nil {
		mw.totalErrors++
		return err
	}

	duration := time.Since(startTime)
	mw.totalWrites += int64(len(mw.buffer))
	mw.lastWriteTime = time.Now()

	mysqlLogger.Debug(fmt.Sprintf("Flushed %d records in %v", len(mw.buffer), duration))

	// Clear buffer
	mw.buffer = mw.buffer[:0]
	return nil
}

// insertNormal performs regular INSERT (fails on duplicate keys)
func (mw *MySQLWriter) insertNormal() error {
	return mw.executeBatchInsert("INSERT INTO")
}

// insertIgnore performs INSERT IGNORE (ignores duplicate keys)
func (mw *MySQLWriter) insertIgnore() error {
	return mw.executeBatchInsert("INSERT IGNORE INTO")
}

// insertReplace performs REPLACE INTO (replaces duplicate keys)
func (mw *MySQLWriter) insertReplace() error {
	return mw.executeBatchInsert("REPLACE INTO")
}

// executeBatchInsert executes batch insert with specified statement type
func (mw *MySQLWriter) executeBatchInsert(insertType string) error {
	if len(mw.buffer) == 0 {
		return nil
	}

	// Get data columns (exclude auto-generated columns)
	dataColumns := make([]string, 0)
	for _, col := range mw.columns {
		// Skip auto-increment and timestamp columns with defaults
		if col == "id" || (col == "created_at" && strings.Contains(mw.columnTypes[col], "DEFAULT")) {
			continue
		}
		dataColumns = append(dataColumns, col)
	}

	if len(dataColumns) == 0 {
		return fmt.Errorf("no data columns to insert")
	}

	// Build SQL
	placeholders := strings.Repeat("?,", len(dataColumns))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

	values := make([]string, len(mw.buffer))
	for i := range values {
		values[i] = "(" + placeholders + ")"
	}

	sql := fmt.Sprintf("%s `%s` (`%s`) VALUES %s",
		insertType,
		mw.config.Table,
		strings.Join(dataColumns, "`, `"),
		strings.Join(values, ", "),
	)

	// Collect all values
	args := make([]interface{}, 0, len(mw.buffer)*len(dataColumns))
	for _, record := range mw.buffer {
		for _, col := range dataColumns {
			value := record[col]
			// Handle nil values and type conversions
			args = append(args, mw.formatValue(value))
		}
	}

	// Execute query
	_, err := mw.db.Exec(sql, args...)
	return err
}

// bulkInsert performs optimized bulk insert using LOAD DATA or batch transactions
func (mw *MySQLWriter) bulkInsert() error {
	// For simplicity, this implementation uses batch insert with transactions
	// In production, you might want to use LOAD DATA INFILE for better performance
	
	tx, err := mw.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert in smaller batches within transaction
	batchSize := mw.config.TransactionSize
	for i := 0; i < len(mw.buffer); i += batchSize {
		end := i + batchSize
		if end > len(mw.buffer) {
			end = len(mw.buffer)
		}

		batch := mw.buffer[i:end]
		if err := mw.executeBatchInTransaction(tx, batch); err != nil {
			return fmt.Errorf("failed to execute batch: %w", err)
		}
	}

	return tx.Commit()
}

// executeBatchInTransaction executes a batch within a transaction
func (mw *MySQLWriter) executeBatchInTransaction(tx *sql.Tx, batch []map[string]interface{}) error {
	// Similar to executeBatchInsert but uses transaction
	dataColumns := make([]string, 0)
	for _, col := range mw.columns {
		if col == "id" || (col == "created_at" && strings.Contains(mw.columnTypes[col], "DEFAULT")) {
			continue
		}
		dataColumns = append(dataColumns, col)
	}

	if len(dataColumns) == 0 {
		return fmt.Errorf("no data columns to insert")
	}

	placeholders := strings.Repeat("?,", len(dataColumns))
	placeholders = placeholders[:len(placeholders)-1]

	values := make([]string, len(batch))
	for i := range values {
		values[i] = "(" + placeholders + ")"
	}

	sql := fmt.Sprintf("INSERT INTO `%s` (`%s`) VALUES %s",
		mw.config.Table,
		strings.Join(dataColumns, "`, `"),
		strings.Join(values, ", "),
	)

	args := make([]interface{}, 0, len(batch)*len(dataColumns))
	for _, record := range batch {
		for _, col := range dataColumns {
			args = append(args, mw.formatValue(record[col]))
		}
	}

	_, err := tx.Exec(sql, args...)
	return err
}

// formatValue formats a value for MySQL insertion
func (mw *MySQLWriter) formatValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		// Check if it's a time string and parse it
		if HasTimeFormatPattern(v) {
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				return parsed
			}
			if parsed, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
				return parsed
			}
			if parsed, err := time.Parse("2006-01-02", v); err == nil {
				return parsed
			}
		}
		return v
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case []interface{}, map[string]interface{}:
		// Convert to JSON for MySQL JSON columns
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	default:
		return v
	}
}

// Close closes the MySQL connection and flushes remaining data
func (mw *MySQLWriter) Close() error {
	if !mw.connected {
		return nil
	}

	// Flush remaining data
	if len(mw.buffer) > 0 {
		if err := mw.flush(); err != nil {
			mysqlLogger.Error(fmt.Sprintf("Failed to flush remaining data: %v", err))
		}
	}

	// Optimize table if requested
	if mw.config.OptimizeAfterInsert {
		if _, err := mw.db.Exec(fmt.Sprintf("OPTIMIZE TABLE `%s`", mw.config.Table)); err != nil {
			mysqlLogger.Warn(fmt.Sprintf("Failed to optimize table: %v", err))
		} else {
			mysqlLogger.Info("Table optimized successfully")
		}
	}

	// Close prepared statements
	if mw.insertStmt != nil {
		mw.insertStmt.Close()
	}
	if mw.upsertStmt != nil {
		mw.upsertStmt.Close()
	}

	// Close database connection
	if mw.db != nil {
		if err := mw.db.Close(); err != nil {
			mysqlLogger.Error(fmt.Sprintf("Failed to close MySQL connection: %v", err))
		}
	}

	mw.connected = false
	mw.metadata.ConnectionState = "disconnected"

	duration := time.Since(mw.startTime)
	mysqlLogger.Info(fmt.Sprintf("Closed MySQL connection. Total writes: %d, errors: %d, duration: %v", 
		mw.totalWrites, mw.totalErrors, duration))

	return nil
}

// GetMetadata returns MySQL operation metadata
func (mw *MySQLWriter) GetMetadata() *MySQLMetadata {
	return mw.metadata
}

// GetStatistics returns detailed statistics about MySQL operations
func (mw *MySQLWriter) GetStatistics() map[string]interface{} {
	duration := time.Since(mw.startTime)
	
	return map[string]interface{}{
		"total_writes":     mw.totalWrites,
		"total_errors":     mw.totalErrors,
		"duration":         duration,
		"writes_per_second": float64(mw.totalWrites) / duration.Seconds(),
		"last_write_time":  mw.lastWriteTime,
		"buffer_size":      len(mw.buffer),
		"connected":        mw.connected,
		"database":         mw.config.Database,
		"table":            mw.config.Table,
		"batch_size":       mw.config.BatchSize,
		"engine":           mw.config.Engine,
		"columns":          len(mw.columns),
	}
}

// Utility methods

// ExecuteSQL executes a raw SQL statement
func (mw *MySQLWriter) ExecuteSQL(query string, args ...interface{}) (sql.Result, error) {
	return mw.db.Exec(query, args...)
}

// QuerySQL executes a query and returns rows
func (mw *MySQLWriter) QuerySQL(query string, args ...interface{}) (*sql.Rows, error) {
	return mw.db.Query(query, args...)
}

// GetTableInfo returns current table information
func (mw *MySQLWriter) GetTableInfo() (*MySQLTableInfo, error) {
	if err := mw.gatherMetadata(); err != nil {
		return nil, err
	}
	return mw.metadata.TableInfo, nil
}

// Default MySQL configuration
func GetDefaultMySQLOptions() MySQLOptions {
	return MySQLOptions{
		BatchSize:           1000,
		Engine:              "InnoDB",
		Charset:             "utf8mb4",
		Collation:           "utf8mb4_unicode_ci",
		MaxConnections:      100,
		MaxIdleConnections:  10,
		ConnMaxLifetime:     time.Hour,
		ConnMaxIdleTime:     10 * time.Minute,
		OnConflict:          ConflictIgnore,
		AutoCommit:          true,
		TransactionSize:     1000,
		BulkInsert:          false,
		OptimizeAfterInsert: false,
		CreateTable:         true,
		CreateDatabase:      false,
		DisableFK:           false,
	}
}