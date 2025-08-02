// internal/output/database_test.go
package output

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidatePostgreSQLIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		identifier  string
		expectError bool
	}{
		{"valid identifier", "user_name", false},
		{"valid with numbers", "user123", false},
		{"starts with underscore", "_private", false},
		{"mixed case", "UserName", false},
		{"empty string", "", true},
		{"starts with number", "123user", true},
		{"contains space", "user name", true},
		{"contains hyphen", "user-name", true},
		{"reserved word", "select", true},
		{"reserved word case", "SELECT", true},
		{"too long", "a" + strings.Repeat("b", 63), true}, // 64 chars total (> MaxPostgreSQLIdentifierLength)
		{"max length", "a" + strings.Repeat("b", 61), false}, // 62 chars total (< MaxPostgreSQLIdentifierLength)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePostgreSQLIdentifier(tt.identifier)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSQLiteIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		identifier  string
		expectError bool
	}{
		{"valid identifier", "user_name", false},
		{"valid with numbers", "user123", false},
		{"starts with underscore", "_private", false},
		{"mixed case", "UserName", false},
		{"empty string", "", true},
		{"starts with number", "123user", true},
		{"contains space", "user name", true},
		{"contains hyphen", "user-name", true},
		{"reserved word", "select", true},
		{"reserved word case", "SELECT", true},
		{"too long", "a" + strings.Repeat("b", 999), true}, // 1000 chars total (> MaxSQLiteIdentifierLength)
		{"max length", "a" + strings.Repeat("b", 997), false}, // 998 chars total (< MaxSQLiteIdentifierLength)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSQLiteIdentifier(tt.identifier)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestColumnDefinitionBuilder_BuildColumnDefinitions(t *testing.T) {
	tests := []struct {
		name        string
		builder     *ColumnDefinitionBuilder
		expected    []string
		expectError bool
	}{
		{
			name: "PostgreSQL basic columns",
			builder: &ColumnDefinitionBuilder{
				DBType:      "postgresql",
				Columns:     []string{"id", "name", "age"},
				ColumnTypes: map[string]string{"id": "SERIAL", "name": "VARCHAR(255)", "age": "INTEGER"},
				UserTypes:   map[string]string{},
				QuoteFunc:   func(s string) string { return fmt.Sprintf(`"%s"`, s) },
			},
			expected: []string{
				`"id" SERIAL`,
				`"name" VARCHAR(255)`,
				`"age" INTEGER`,
			},
			expectError: false,
		},
		{
			name: "SQLite basic columns",
			builder: &ColumnDefinitionBuilder{
				DBType:      "sqlite",
				Columns:     []string{"id", "title", "content"},
				ColumnTypes: map[string]string{"id": "INTEGER PRIMARY KEY", "title": "TEXT", "content": "TEXT"},
				UserTypes:   map[string]string{},
				QuoteFunc:   func(s string) string { return fmt.Sprintf("`%s`", s) },
			},
			expected: []string{
				"`id` INTEGER PRIMARY KEY",
				"`title` TEXT",
				"`content` TEXT",
			},
			expectError: false,
		},
		{
			name: "with user-defined types",
			builder: &ColumnDefinitionBuilder{
				DBType:      "postgresql",
				Columns:     []string{"id", "custom_field"},
				ColumnTypes: map[string]string{"id": "SERIAL", "custom_field": "TEXT"},
				UserTypes:   map[string]string{"custom_field": "JSONB"},
				QuoteFunc:   func(s string) string { return fmt.Sprintf(`"%s"`, s) },
			},
			expected: []string{
				`"id" SERIAL`,
				`"custom_field" JSONB`,
			},
			expectError: false,
		},
		{
			name: "invalid column name",
			builder: &ColumnDefinitionBuilder{
				DBType:      "postgresql",
				Columns:     []string{"123invalid"},
				ColumnTypes: map[string]string{"123invalid": "TEXT"},
				UserTypes:   map[string]string{},
				QuoteFunc:   func(s string) string { return fmt.Sprintf(`"%s"`, s) },
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.builder.BuildColumnDefinitions()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d columns, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("column %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestTableNameValidation(t *testing.T) {
	tests := []struct {
		name        string
		tableName   string
		dbType      string
		expectError bool
	}{
		{"valid PostgreSQL table", "users", "postgresql", false},
		{"valid SQLite table", "products", "sqlite", false},
		{"empty table name", "", "postgresql", true},
		{"invalid PostgreSQL table", "select", "postgresql", true},
		{"invalid SQLite table", "order", "sqlite", true},
		{"table with underscore", "user_profiles", "postgresql", false},
		{"table with numbers", "table123", "sqlite", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.dbType {
			case "postgresql":
				err := ValidatePostgreSQLIdentifier(tt.tableName)
				if tt.expectError && err == nil {
					t.Error("expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			case "sqlite":
				err := ValidateSQLiteIdentifier(tt.tableName)
				if tt.expectError && err == nil {
					t.Error("expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDatabaseWriterConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		options     interface{}
		expectError bool
		errorType   string
	}{
		{
			name: "invalid PostgreSQL - empty connection string",
			options: PostgreSQLOptions{
				ConnectionString: "",
				Table:           "test_table",
			},
			expectError: true,
			errorType:   "connection string",
		},
		{
			name: "invalid PostgreSQL - empty table name",
			options: PostgreSQLOptions{
				ConnectionString: "postgres://localhost/test",
				Table:           "",
			},
			expectError: true,
			errorType:   "table name",
		},
		{
			name: "valid PostgreSQL options",
			options: PostgreSQLOptions{
				ConnectionString: "postgres://user:pass@localhost/testdb",
				Table:           "valid_table",
				BatchSize:       100,
			},
			expectError: true, // Will fail on database connection, not config validation
			errorType:   "connect", // Should be a connection error, not config error
		},
		{
			name: "invalid SQLite - empty connection string",
			options: SQLiteOptions{
				DatabasePath: "",
				Table:       "test_table",
			},
			expectError: true,
			errorType:   "database path",
		},
		{
			name: "valid SQLite options",
			options: SQLiteOptions{
				DatabasePath: ":memory:",
				Table:       "valid_table",
				BatchSize:   50,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			
			switch opts := tt.options.(type) {
			case PostgreSQLOptions:
				_, err = NewPostgreSQLWriter(opts)
			case SQLiteOptions:
				_, err = NewSQLiteWriter(opts)
			}

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorType != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorType)) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorType, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConflictStrategyValidation(t *testing.T) {
	validStrategies := []ConflictStrategy{ConflictIgnore, ConflictError, ConflictReplace}
	
	for _, strategy := range validStrategies {
		t.Run(string(strategy), func(t *testing.T) {
			if strategy == "" {
				t.Error("strategy should not be empty")
			}
			
			// Test that the strategy is one of the defined constants
			switch strategy {
			case ConflictIgnore, ConflictError, ConflictReplace:
				// Valid
			default:
				t.Errorf("unexpected conflict strategy: %s", strategy)
			}
		})
	}
}

func TestOutputFormatValidation(t *testing.T) {
	validFormats := ValidOutputFormats()
	
	if len(validFormats) == 0 {
		t.Error("should have at least one valid output format")
	}
	
	expectedFormats := []OutputFormat{
		FormatJSON, FormatCSV, FormatXML, FormatYAML, FormatTSV, FormatPostgreSQL, FormatSQLite,
	}
	
	for _, expected := range expectedFormats {
		found := false
		for _, valid := range validFormats {
			if valid == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected format %s not found in valid formats", expected)
		}
	}
}