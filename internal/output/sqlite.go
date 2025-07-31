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

// SQLiteWriter writes data to SQLite database
type SQLiteWriter struct {
	db      *sql.DB
	config  SQLiteOptions
	table   string
	columns []string
	closed  bool
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
		options.OnConflict = "ignore"
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
		connectionParams = "?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on"
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
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = 10000",
		"PRAGMA temp_store = memory",
		"PRAGMA mmap_size = 268435456", // 256MB
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	writer := &SQLiteWriter{
		db:     db,
		config: options,
		table:  options.Table,
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
			columnType = userType
		}
		columnDefs = append(columnDefs, fmt.Sprintf("[%s] %s", column, columnType))
	}

	// Add metadata column
	columnDefs = append(columnDefs, "created_at DATETIME DEFAULT CURRENT_TIMESTAMP")

	query := `
		CREATE TABLE IF NOT EXISTS [` + w.table + `] (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			` + strings.Join(columnDefs, ",\n\t\t\t") + `
		)`

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
			} else if _, err := time.Parse(time.RFC3339, v); err == nil {
				hasTime = true
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

	// Build INSERT statement with placeholders
	columnList := make([]string, len(w.columns))
	for i, column := range w.columns {
		columnList[i] = "[" + column + "]"
	}

	placeholders := strings.Repeat("?,", len(w.columns))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

	var query string
	switch w.config.OnConflict {
	case "ignore":
		query = fmt.Sprintf(`
			INSERT OR IGNORE INTO [%s] (%s) 
			VALUES (%s)`,
			w.table,
			strings.Join(columnList, ", "),
			placeholders,
		)
	case "replace":
		query = fmt.Sprintf(`
			INSERT OR REPLACE INTO [%s] (%s) 
			VALUES (%s)`,
			w.table,
			strings.Join(columnList, ", "),
			placeholders,
		)
	default: // "error" or any other value
		query = fmt.Sprintf(`
			INSERT INTO [%s] (%s) 
			VALUES (%s)`,
			w.table,
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
		args := make([]interface{}, len(w.columns))
		for i, column := range w.columns {
			value := record[column]
			args[i] = w.convertValue(value)
		}

		if _, err := stmt.Exec(args...); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
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
func (w *SQLiteWriter) Close() error {
	if w.db != nil && !w.closed {
		// Only optimize database if explicitly configured to do so
		// VACUUM can be expensive and block other operations
		if w.config.OptimizeOnClose {
			// Run VACUUM asynchronously after closing the database
			go func(db *sql.DB) {
				if _, err := db.Exec("PRAGMA optimize"); err != nil {
					fmt.Printf("Error during PRAGMA optimize: %v\n", err)
				}
				if _, err := db.Exec("VACUUM"); err != nil {
					fmt.Printf("Error during VACUUM: %v\n", err)
				}
			}(w.db)
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
		query := fmt.Sprintf("SELECT COUNT(*) FROM [%s]", w.table)
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