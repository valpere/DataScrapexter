// internal/output/postgresql.go
package output

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgreSQLWriter writes data to PostgreSQL database
type PostgreSQLWriter struct {
	db      *sql.DB
	config  PostgreSQLOptions
	table   string
	schema  string
	columns []string
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
		options.OnConflict = "ignore"
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
		db:     db,
		config: options,
		table:  options.Table,
		schema: options.Schema,
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

	// Build CREATE TABLE statement
	var columnDefs []string
	for _, column := range w.columns {
		columnType := columnTypes[column]
		// Override with user-specified types if provided
		if userType, exists := w.config.ColumnTypes[column]; exists {
			columnType = userType
		}
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", w.quoteIdentifier(column), columnType))
	}

	// Add created_at timestamp column
	columnDefs = append(columnDefs, "created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP")

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
			} else if _, err := time.Parse(time.RFC3339, v); err == nil {
				hasTime = true
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
		return "VARCHAR(" + strconv.Itoa(maxTextLength*2) + ")" // Give some extra space
	}

	return "TEXT"
}

// insertBatches inserts data in batches
func (w *PostgreSQLWriter) insertBatches(data []map[string]interface{}) error {
	batchSize := w.config.BatchSize
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		batch := data[i:end]

		if err := w.insertBatch(batch); err != nil {
			return fmt.Errorf("failed to insert batch %d-%d: %w", i, end-1, err)
		}
	}
	return nil
}

// insertBatch inserts a single batch of data
func (w *PostgreSQLWriter) insertBatch(batch []map[string]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	// Build INSERT statement
	placeholders := make([]string, len(batch))
	args := make([]interface{}, 0, len(batch)*len(w.columns))
	argIndex := 1

	for i, record := range batch {
		rowPlaceholders := make([]string, len(w.columns))
		for j, column := range w.columns {
			rowPlaceholders[j] = "$" + strconv.Itoa(argIndex)
			argIndex++

			// Get value, use nil if missing
			value := record[column]
			args = append(args, w.convertValue(value))
		}
		placeholders[i] = "(" + strings.Join(rowPlaceholders, ", ") + ")"
	}

	// Build column list
	quotedColumns := make([]string, len(w.columns))
	for i, column := range w.columns {
		quotedColumns[i] = w.quoteIdentifier(column)
	}

	var query string
	switch w.config.OnConflict {
	case "ignore":
		query = fmt.Sprintf(`
			INSERT INTO %s.%s (%s) 
			VALUES %s 
			ON CONFLICT DO NOTHING`,
			w.quoteIdentifier(w.schema),
			w.quoteIdentifier(w.table),
			strings.Join(quotedColumns, ", "),
			strings.Join(placeholders, ", "),
		)
	case "update":
		// Build update clause for ON CONFLICT UPDATE
		// Use all columns for conflict detection to avoid assuming specific column names
		updateClauses := make([]string, len(w.columns))
		for i, column := range w.columns {
			quotedCol := w.quoteIdentifier(column)
			updateClauses[i] = fmt.Sprintf("%s = EXCLUDED.%s", quotedCol, quotedCol)
		}
		query = fmt.Sprintf(`
			INSERT INTO %s.%s (%s) 
			VALUES %s 
			ON CONFLICT (id) DO UPDATE SET %s`,
			w.quoteIdentifier(w.schema),
			w.quoteIdentifier(w.table),
			strings.Join(quotedColumns, ", "),
			strings.Join(placeholders, ", "),
			"id", // Conflict target: specific unique column
			strings.Join(updateClauses, ", "),
		)
	default: // "error" or any other value
		query = fmt.Sprintf(`
			INSERT INTO %s.%s (%s) 
			VALUES %s`,
			w.quoteIdentifier(w.schema),
			w.quoteIdentifier(w.table),
			strings.Join(quotedColumns, ", "),
			strings.Join(placeholders, ", "),
		)
	}

	_, err := w.db.Exec(query, args...)
	return err
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

