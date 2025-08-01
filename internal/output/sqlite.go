// internal/output/sqlite.go
package output

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLite connection parameter constants
const (
	// Connection string parameters
	DefaultSQLiteBusyTimeout = 5000    // milliseconds
	DefaultSQLiteJournalMode = "WAL"   // Write-Ahead Logging for better concurrency
	DefaultSQLiteForeignKeys = "on"    // Enable foreign key constraints
	
	// PRAGMA optimization parameters
	DefaultSQLiteSynchronous = "NORMAL" // Balance between safety and performance
	DefaultSQLiteCacheSize   = 10000    // Number of pages to cache
	DefaultSQLiteTempStore   = "memory" // Store temporary tables in memory
	DefaultSQLiteMmapSize    = 268435456 // 256MB memory-mapped I/O
)

// buildDefaultConnectionParams creates the default SQLite connection string
func buildDefaultConnectionParams() string {
	return fmt.Sprintf("?_busy_timeout=%d&_journal_mode=%s&_foreign_keys=%s",
		DefaultSQLiteBusyTimeout,
		DefaultSQLiteJournalMode,
		DefaultSQLiteForeignKeys,
	)
}

// SQLiteWriter writes data to SQLite database
type SQLiteWriter struct {
	db            *sql.DB
	config        SQLiteOptions
	table         string
	columns       []string
	systemColumns []string // Columns with DEFAULT values that shouldn't be inserted
	closed        bool
}

// NewSQLiteWriter creates a new SQLite writer
func NewSQLiteWriter(options SQLiteOptions) (*SQLiteWriter, error) {
	if options.DatabasePath == "" {
		return nil, fmt.Errorf("SQLite database path is required")
	}
	if options.Table == "" {
		return nil, fmt.Errorf("SQLite table name is required")
	}

	// Set defaults
	if options.BatchSize == 0 {
		options.BatchSize = 1000
	}
	if options.OnConflict == "" {
		options.OnConflict = ConflictIgnore
	}
	
	// Validate conflict strategy
	if !IsValidConflictStrategy(options.OnConflict) {
		return nil, fmt.Errorf("invalid conflict strategy: %s", options.OnConflict)
	}
	
	// Validate table name
	if err := ValidateSQLIdentifier(options.Table); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	if options.ColumnTypes == nil {
		options.ColumnTypes = make(map[string]string)
	}

	// Create directory if it doesn't exist
	if dir := filepath.Dir(options.DatabasePath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Connect to database
	connectionParams := options.ConnectionParams
	if connectionParams == "" {
		connectionParams = buildDefaultConnectionParams()
	}
	db, err := sql.Open("sqlite3", options.DatabasePath+connectionParams)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	// Configure connection
	db.SetMaxOpenConns(1) // SQLite works best with single writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // No timeout for SQLite

	// Enable optimizations
	pragmas := []string{
		fmt.Sprintf("PRAGMA synchronous = %s", DefaultSQLiteSynchronous),
		fmt.Sprintf("PRAGMA cache_size = %d", DefaultSQLiteCacheSize),
		fmt.Sprintf("PRAGMA temp_store = %s", DefaultSQLiteTempStore),
		fmt.Sprintf("PRAGMA mmap_size = %d", DefaultSQLiteMmapSize),
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	writer := &SQLiteWriter{
		db:            db,
		config:        options,
		table:         options.Table,
		systemColumns: []string{SystemColumnCreatedAt}, // Initialize system columns
	}

	return writer, nil
}

// Write writes data to SQLite database
func (w *SQLiteWriter) Write(data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// Analyze data structure and create table if needed
	if err := w.analyzeAndCreateTable(data); err != nil {
		return fmt.Errorf("failed to analyze/create table: %w", err)
	}

	// Insert data in batches
	return w.insertBatches(data)
}

// analyzeAndCreateTable analyzes data structure and creates table if needed
func (w *SQLiteWriter) analyzeAndCreateTable(data []map[string]interface{}) error {
	// Extract all unique column names
	columnSet := make(map[string]bool)
	for _, record := range data {
		for column := range record {
			columnSet[column] = true
		}
	}

	// Convert to sorted slice for consistency
	w.columns = make([]string, 0, len(columnSet))
	for column := range columnSet {
		w.columns = append(w.columns, column)
	}

	// Create table if requested
	if w.config.CreateTable {
		return w.createTable(data)
	}

	return nil
}

// createTable creates the table with appropriate column types
func (w *SQLiteWriter) createTable(data []map[string]interface{}) error {
	// Infer column types from data
	columnTypes := w.inferColumnTypes(data)

	// Build CREATE TABLE statement
	var columnDefs []string
	for _, column := range w.columns {
		columnType := columnTypes[column]
		// Override with user-specified types if provided
		if userType, exists := w.config.ColumnTypes[column]; exists {
			// Validate user-specified column type
			if err := ValidateColumnType(userType, "sqlite"); err != nil {
				return fmt.Errorf("invalid column type for column '%s': %w", column, err)
			}
			columnType = userType
		}
		// SQLite uses square brackets for identifier quoting (also supports double quotes)
		// but square brackets are more commonly used in SQLite contexts
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", w.quoteIdentifier(column), columnType))
	}

	// Add system columns (created_at timestamp column)
	columnDefs = append(columnDefs, SystemColumnCreatedAtSQLite)
	// Note: systemColumns are initialized in constructor and handled separately in INSERT operations

	var queryBuilder strings.Builder
	queryBuilder.WriteString("CREATE TABLE IF NOT EXISTS ")
	queryBuilder.WriteString(w.quoteIdentifier(w.table))
	queryBuilder.WriteString(" (\n")
	queryBuilder.WriteString("\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n")
	queryBuilder.WriteString("\t")
	queryBuilder.WriteString(strings.Join(columnDefs, ",\n\t"))
	queryBuilder.WriteString("\n);")

	query := queryBuilder.String()

	if _, err := w.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table '%s': %w", w.table, err)
	}

	return nil
}

// inferColumnTypes infers SQLite column types from data
func (w *SQLiteWriter) inferColumnTypes(data []map[string]interface{}) map[string]string {
	columnTypes := make(map[string]string)

	// Initialize all columns as TEXT
	for _, column := range w.columns {
		columnTypes[column] = "TEXT"
	}

	// Analyze sample of data to infer better types
	sampleSize := len(data)
	if sampleSize > 100 {
		sampleSize = 100 // Analyze first 100 records
	}
	for _, column := range w.columns {
		columnType := w.inferColumnType(data[:sampleSize], column)
		columnTypes[column] = columnType
	}

	return columnTypes
}

// inferColumnType infers SQLite type for a specific column
func (w *SQLiteWriter) inferColumnType(data []map[string]interface{}, column string) string {
	var hasInts, hasFloats, hasTime bool
	maxTextLength := 0

	for _, record := range data {
		value, exists := record[column]
		if !exists || value == nil {
			continue
		}

		switch v := value.(type) {
		case bool:
			hasInts = true // SQLite stores booleans as integers
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			hasInts = true
		case float32, float64:
			hasFloats = true
		case time.Time:
			hasTime = true
		case string:
			if len(v) > maxTextLength {
				maxTextLength = len(v)
			}
			// Try to parse as number
			if _, err := strconv.ParseInt(v, 10, 64); err == nil {
				hasInts = true
			} else if _, err := strconv.ParseFloat(v, 64); err == nil {
				hasFloats = true
			} else if HasTimeFormatPattern(v) { // Quick format check before expensive parsing
				timeFormats := []string{
					time.RFC3339,
					time.RFC3339Nano,
					time.RFC1123,
					time.RFC1123Z,
					"2006-01-02", // ISO date
					"2006-01-02 15:04:05", // Common datetime format
				}
				for _, format := range timeFormats {
					if _, err := time.Parse(format, v); err == nil {
						hasTime = true
						break
					}
				}
			}
		}
	}

	// Determine best type based on analysis
	if hasTime {
		return "DATETIME"
	}
	if hasFloats {
		return "REAL"
	}
	if hasInts && !hasFloats {
		return "INTEGER"
	}

	return "TEXT"
}

// insertBatches inserts data in batches
func (w *SQLiteWriter) insertBatches(data []map[string]interface{}) error {
	// Begin transaction for better performance
	tx, err := w.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	batchSize := w.config.BatchSize
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		batch := data[i:end]

		if err := w.insertBatch(tx, batch); err != nil {
			return fmt.Errorf("failed to insert batch %d-%d: %w", i, end-1, err)
		}
	}

	return tx.Commit()
}

// insertBatch inserts a single batch of data within a transaction
func (w *SQLiteWriter) insertBatch(tx *sql.Tx, batch []map[string]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	// Filter out system columns that have DEFAULT values
	insertColumns := make([]string, 0, len(w.columns))
	systemColumnMap := make(map[string]bool)
	for _, col := range w.systemColumns {
		systemColumnMap[col] = true
	}
	
	for _, column := range w.columns {
		if !systemColumnMap[column] {
			insertColumns = append(insertColumns, column)
		}
	}

	// Build INSERT statement with placeholders
	columnList := make([]string, len(insertColumns))
	for i, column := range insertColumns {
		columnList[i] = w.quoteIdentifier(column)
	}

	placeholders := strings.Repeat("?,", len(insertColumns))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

	var query string
	switch w.config.OnConflict {
	case ConflictIgnore:
		query = fmt.Sprintf(`
			INSERT OR IGNORE INTO %s (%s) 
			VALUES (%s)`,
			w.quoteIdentifier(w.table),
			strings.Join(columnList, ", "),
			placeholders,
		)
	case ConflictReplace:
		query = fmt.Sprintf(`
			INSERT OR REPLACE INTO %s (%s) 
			VALUES (%s)`,
			w.quoteIdentifier(w.table),
			strings.Join(columnList, ", "),
			placeholders,
		)
	default: // ConflictError or any other value
		query = fmt.Sprintf(`
			INSERT INTO %s (%s) 
			VALUES (%s)`,
			w.quoteIdentifier(w.table),
			strings.Join(columnList, ", "),
			placeholders,
		)
	}

	// Prepare statement
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute for each record
	for _, record := range batch {
		args := make([]interface{}, len(insertColumns))
		for i, column := range insertColumns {
			value := record[column]
			args[i] = w.convertValue(value)
		}

		if _, err := stmt.Exec(args...); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
}

// quoteIdentifier quotes SQLite identifiers using double quotes
func (w *SQLiteWriter) quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// convertValue converts Go values to SQLite-compatible values
func (w *SQLiteWriter) convertValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		return v.Format(time.RFC3339)
	case []interface{}:
		// Convert slice to proper JSON string
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return "[]" // fallback for invalid JSON
	case map[string]interface{}:
		// Convert map to proper JSON string
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return "{}" // fallback for invalid JSON
	case string:
		return v
	case bool:
		if v {
			return 1
		}
		return 0
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return v
	case float32, float64:
		return v
	default:
		// Convert anything else to string representation
		return fmt.Sprintf("%v", v)
	}
}

// Close closes the SQLite connection
// 
// IMPORTANT PERFORMANCE NOTE: If OptimizeOnClose is enabled, this method
// will run PRAGMA optimize and incremental VACUUM operations before closing.
// Full VACUUM can be EXTREMELY slow for large databases (minutes to hours) and will
// BLOCK all other database operations during execution.
//
// For better performance in production:
// 1. Set OptimizeOnClose to false for large databases
// 2. Run VACUUM manually during scheduled maintenance windows
// 3. Use PRAGMA incremental_vacuum for regular cleanup (default behavior)
// 4. Monitor database size and plan VACUUM operations accordingly
func (w *SQLiteWriter) Close() error {
	if w.db != nil && !w.closed {
		// Only optimize database if explicitly configured to do so
		if w.config.OptimizeOnClose {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel() // Ensure the context is canceled when Close exits

			var wg sync.WaitGroup
			wg.Add(1)
			go w.performDatabaseOptimization(ctx, &wg)

			// Optionally, wait for the optimization to complete if needed
			// wg.Wait()
		}

		err := w.db.Close()
		w.db = nil
		w.closed = true
		return err
	}
	return nil
}

// GetStats returns SQLite writer statistics
func (w *SQLiteWriter) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"driver":      "sqlite3",
		"database":    w.config.DatabasePath,
		"table":       w.table,
		"columns":     len(w.columns),
		"batch_size":  w.config.BatchSize,
		"on_conflict": w.config.OnConflict,
	}

	if w.db != nil {
		// Get database size
		if info, err := os.Stat(w.config.DatabasePath); err == nil {
			stats["database_size"] = info.Size()
		}

		// Get table row count
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", w.quoteIdentifier(w.table))
		if err := w.db.QueryRow(query).Scan(&count); err == nil {
			stats["row_count"] = count
		}

		dbStats := w.db.Stats()
		stats["open_connections"] = dbStats.OpenConnections
		stats["in_use"] = dbStats.InUse
		stats["idle"] = dbStats.Idle
	}

	return stats
}

// performDatabaseOptimization runs database optimization operations
// Uses incremental_vacuum for better performance on large databases
func (w *SQLiteWriter) performDatabaseOptimization() error {
	// Run PRAGMA optimize first (fast operation)
	if _, err := w.db.Exec("PRAGMA optimize"); err != nil {
		return fmt.Errorf("PRAGMA optimize failed: %w", err)
	}
	
	// Use incremental_vacuum instead of full VACUUM for better performance
	// This is non-blocking and more suitable for production environments
	if _, err := w.db.Exec("PRAGMA incremental_vacuum"); err != nil {
		// If incremental_vacuum fails, log but don't fail the close operation
		fmt.Printf("Warning: PRAGMA incremental_vacuum failed: %v\n", err)
		
		// Fallback to a limited VACUUM with timeout protection would be ideal,
		// but SQLite doesn't support VACUUM timeouts. In production, consider
		// implementing this as a background goroutine with context cancellation.
	}
	
	return nil
}