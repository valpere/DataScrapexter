// internal/output/postgresql.go
package output

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgreSQL column type inference constants moved to types.go for shared use

// PostgreSQLWriter provides methods for writing data to a PostgreSQL database.
// It manages the connection, table creation, and batch insertion of records.
// Use NewPostgreSQLWriter to create an instance, then call Write to insert data.
type PostgreSQLWriter struct {
	db            *sql.DB
	config        PostgreSQLOptions
	table         string
	schema        string
	columns       []string
	systemColumns []string // Columns with DEFAULT values that shouldn't be inserted
}

// NewPostgreSQLWriter creates a new PostgreSQL writer
func NewPostgreSQLWriter(options PostgreSQLOptions) (*PostgreSQLWriter, error) {
	if options.ConnectionString == "" {
		return nil, fmt.Errorf("PostgreSQL connection string is required")
	}
	if options.Table == "" {
		return nil, fmt.Errorf("PostgreSQL table name is required")
	}

	// Set defaults
	if options.BatchSize == 0 {
		options.BatchSize = 1000
	}
	if options.Schema == "" {
		options.Schema = "public"
	}
	if options.OnConflict == "" {
		options.OnConflict = ConflictIgnore
	}

	// Validate conflict strategy
	if !IsValidConflictStrategy(options.OnConflict) {
		return nil, fmt.Errorf("invalid conflict strategy: %s", options.OnConflict)
	}

	// Validate table and schema names using PostgreSQL-specific validation
	if err := ValidatePostgreSQLIdentifier(options.Table); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	if err := ValidatePostgreSQLIdentifier(options.Schema); err != nil {
		return nil, fmt.Errorf("invalid schema name: %w", err)
	}
	if options.ColumnTypes == nil {
		options.ColumnTypes = make(map[string]string)
	}

	// Connect to database
	db, err := sql.Open("postgres", options.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	writer := &PostgreSQLWriter{
		db:            db,
		config:        options,
		table:         options.Table,
		schema:        options.Schema,
		systemColumns: []string{SystemColumnCreatedAt}, // Initialize system columns
	}

	return writer, nil
}

// Write writes data to PostgreSQL database
func (w *PostgreSQLWriter) Write(data []map[string]interface{}) error {
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

// WriteContext writes data to PostgreSQL database with context support
func (w *PostgreSQLWriter) WriteContext(ctx context.Context, data interface{}) error {
	switch v := data.(type) {
	case []map[string]interface{}:
		return w.Write(v)
	case map[string]interface{}:
		return w.Write([]map[string]interface{}{v})
	case []interface{}:
		records := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if record, ok := item.(map[string]interface{}); ok {
				records = append(records, record)
			} else {
				return fmt.Errorf("unsupported data type in slice: %T", item)
			}
		}
		return w.Write(records)
	default:
		return fmt.Errorf("unsupported data type: %T", data)
	}
}

// GetType returns the output type
func (w *PostgreSQLWriter) GetType() string {
	return "postgresql"
}

// Flush ensures all pending writes are committed to the database
// For PostgreSQL, this forces any pending transactions to be committed
func (w *PostgreSQLWriter) Flush() error {
	if w.db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	// PostgreSQL handles auto-commit by default for individual statements
	// This method provides consistency with other writers and can be extended
	// for future buffering or transaction management features
	
	// Ping the connection to ensure it's still alive
	if err := w.db.Ping(); err != nil {
		return fmt.Errorf("database connection lost: %w", err)
	}
	
	return nil
}

// analyzeAndCreateTable analyzes data structure and creates table if needed
func (w *PostgreSQLWriter) analyzeAndCreateTable(data []map[string]interface{}) error {
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
func (w *PostgreSQLWriter) createTable(data []map[string]interface{}) error {
	// Infer column types from data
	columnTypes := w.inferColumnTypes(data)

	// Build CREATE TABLE statement using shared utility
	builder := &ColumnDefinitionBuilder{
		DBType:      "postgresql",
		Columns:     w.columns,
		ColumnTypes: columnTypes,
		UserTypes:   w.config.ColumnTypes,
		QuoteFunc:   w.quoteIdentifier,
	}

	columnDefs, err := builder.BuildColumnDefinitions()
	if err != nil {
		return err
	}

	// Add system columns (created_at timestamp column)
	columnDefs = append(columnDefs, SystemColumnCreatedAtType)
	// Note: systemColumns are initialized in constructor and handled separately in INSERT operations

	var queryBuilder strings.Builder
	queryBuilder.WriteString("CREATE TABLE IF NOT EXISTS ")
	queryBuilder.WriteString(w.quoteIdentifier(w.schema))
	queryBuilder.WriteString(".")
	queryBuilder.WriteString(w.quoteIdentifier(w.table))
	queryBuilder.WriteString(" (\n")
	queryBuilder.WriteString("\tid SERIAL PRIMARY KEY,\n")
	queryBuilder.WriteString("\t")
	queryBuilder.WriteString(strings.Join(columnDefs, ",\n\t"))
	queryBuilder.WriteString("\n);")

	query := queryBuilder.String()
	if _, err := w.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table '%s.%s': %w", w.schema, w.table, err)
	}

	return nil
}

// inferColumnTypes infers PostgreSQL column types from data
func (w *PostgreSQLWriter) inferColumnTypes(data []map[string]interface{}) map[string]string {
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

// inferColumnType infers PostgreSQL type for a specific column
func (w *PostgreSQLWriter) inferColumnType(data []map[string]interface{}, column string) string {
	var hasInts, hasFloats, hasBools, hasTime, hasLargeText bool
	maxTextLength := 0

	for _, record := range data {
		value, exists := record[column]
		if !exists || value == nil {
			continue
		}

		switch v := value.(type) {
		case bool:
			hasBools = true
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
			if len(v) > 1000 {
				hasLargeText = true
			}
			// Try to parse as number
			if _, err := strconv.ParseInt(v, 10, 64); err == nil {
				hasInts = true
			} else if _, err := strconv.ParseFloat(v, 64); err == nil {
				hasFloats = true
			} else if HasTimeFormatPattern(v) { // Quick format check before expensive parsing
				// Attempt to parse using multiple time formats
				timeFormats := []string{
					time.RFC3339,
					"2006-01-02 15:04:05", // Common SQL datetime format
					"2006-01-02",          // Date only
					"15:04:05",            // Time only
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
		return "TIMESTAMP"
	}
	if hasBools && !hasInts && !hasFloats {
		return "BOOLEAN"
	}
	if hasFloats {
		return "DOUBLE PRECISION"
	}
	if hasInts && !hasFloats {
		return "BIGINT"
	}
	if hasLargeText {
		return "TEXT"
	}
	if maxTextLength > 255 {
		return "TEXT"
	}
	if maxTextLength > 0 {
		return fmt.Sprintf("VARCHAR(%d)", maxTextLength*VarcharLengthMultiplier) // Give some extra space
	}

	return "TEXT"
}

// insertBatches inserts data in batches with transaction support for improved performance
func (w *PostgreSQLWriter) insertBatches(data []map[string]interface{}) error {
	batchSize := w.config.BatchSize
	
	// Use transactions for multiple batches to improve performance and reliability
	if len(data) > batchSize {
		return w.insertBatchesWithTransaction(data)
	}
	
	// Single batch can be inserted without explicit transaction
	return w.insertBatch(data)
}

// insertBatchesWithTransaction inserts multiple batches within a single transaction
func (w *PostgreSQLWriter) insertBatchesWithTransaction(data []map[string]interface{}) error {
	tx, err := w.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	batchSize := w.config.BatchSize
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		batch := data[i:end]

		if err := w.insertBatchWithTx(tx, batch); err != nil {
			return fmt.Errorf("failed to insert batch %d-%d: %w", i, end-1, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// insertBatch inserts a single batch of data
func (w *PostgreSQLWriter) insertBatch(batch []map[string]interface{}) error {
	return w.insertBatchWithTx(nil, batch)
}

// insertBatchWithTx inserts a single batch of data with optional transaction support
func (w *PostgreSQLWriter) insertBatchWithTx(tx *sql.Tx, batch []map[string]interface{}) error {
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

	// Build INSERT statement
	placeholders := make([]string, len(batch))
	args := make([]interface{}, 0, len(batch)*len(insertColumns))
	argIndex := 1

	for i, record := range batch {
		rowPlaceholders := make([]string, len(insertColumns))
		for j, column := range insertColumns {
			rowPlaceholders[j] = "$" + strconv.Itoa(argIndex)
			argIndex++

			// Get value, use nil if missing
			value := record[column]
			args = append(args, w.convertValue(value))
		}
		placeholders[i] = "(" + strings.Join(rowPlaceholders, ", ") + ")"
	}

	// Build column list (quoted)
	quotedColumns := make([]string, len(insertColumns))
	for i, column := range insertColumns {
		quotedColumns[i] = w.quoteIdentifier(column)
	}

	var query string
	switch w.config.OnConflict {
	case ConflictIgnore:
		query = fmt.Sprintf(`
			INSERT INTO %s.%s (%s)
			VALUES %s
			ON CONFLICT DO NOTHING`,
			w.quoteIdentifier(w.schema),
			w.quoteIdentifier(w.table),
			strings.Join(quotedColumns, ", "),
			strings.Join(placeholders, ", "),
		)
	default: // ConflictError or any other value
		query = fmt.Sprintf(`
			INSERT INTO %s.%s (%s)
			VALUES %s`,
			w.quoteIdentifier(w.schema),
			w.quoteIdentifier(w.table),
			strings.Join(quotedColumns, ", "),
			strings.Join(placeholders, ", "),
		)
	}

	// Execute with transaction if provided, otherwise use direct connection
	if tx != nil {
		_, err := tx.Exec(query, args...)
		return err
	} else {
		_, err := w.db.Exec(query, args...)
		return err
	}
}

// convertValue converts Go values to PostgreSQL-compatible values
func (w *PostgreSQLWriter) convertValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		return v
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
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return v
	case float32, float64:
		return v
	default:
		// Convert anything else to string representation
		return fmt.Sprintf("%v", v)
	}
}

// quoteIdentifier quotes PostgreSQL identifiers
func (w *PostgreSQLWriter) quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// Close closes the PostgreSQL connection
func (w *PostgreSQLWriter) Close() error {
	if w.db != nil {
		err := w.db.Close()
		w.db = nil
		return err
	}
	return nil
}

// GetStats returns PostgreSQL writer statistics
func (w *PostgreSQLWriter) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"driver":     "postgresql",
		"table":      w.table,
		"schema":     w.schema,
		"columns":    len(w.columns),
		"batch_size": w.config.BatchSize,
	}

	if w.db != nil {
		dbStats := w.db.Stats()
		stats["open_connections"] = dbStats.OpenConnections
		stats["in_use"] = dbStats.InUse
		stats["idle"] = dbStats.Idle
	}

	return stats
}
