// internal/pipeline/components_test.go
package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestDataExtractor_Extract(t *testing.T) {
	ctx := context.Background()
	extractor := &DataExtractor{
		SelectorEngines:   make(map[string]SelectorEngine),
		ContentProcessors: []ContentProcessor{},
		StructuredData:    &StructuredDataExtractor{},
		MediaExtractor:    &MediaContentExtractor{},
	}

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "simple data extraction",
			input: map[string]interface{}{
				"title": "Test Title",
				"price": "$19.99",
			},
			expected: map[string]interface{}{
				"title": "Test Title",
				"price": "$19.99",
			},
		},
		{
			name: "complex nested data",
			input: map[string]interface{}{
				"product": map[string]interface{}{
					"name": "Test Product",
					"specs": []string{"spec1", "spec2"},
				},
				"metadata": map[string]interface{}{
					"timestamp": "2024-01-01T00:00:00Z",
					"version":   1,
				},
			},
			expected: map[string]interface{}{
				"product": map[string]interface{}{
					"name": "Test Product",
					"specs": []string{"spec1", "spec2"},
				},
				"metadata": map[string]interface{}{
					"timestamp": "2024-01-01T00:00:00Z",
					"version":   1,
				},
			},
		},
		{
			name:     "empty data",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name:     "nil data",
			input:    nil,
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.Extract(ctx, tt.input)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d", len(tt.expected), len(result))
				return
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("expected key %s not found", key)
				} else if !deepEqual(actualValue, expectedValue) {
					t.Errorf("for key %s: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestDataValidator_Validate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		validator   *DataValidator
		input       map[string]interface{}
		expected    map[string]interface{}
		expectError bool
	}{
		{
			name: "valid string field",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "name", Type: "string", Required: true, MinLen: 2, MaxLen: 50},
				},
				StrictMode: true,
			},
			input: map[string]interface{}{
				"name": "John Doe",
			},
			expected: map[string]interface{}{
				"name": "John Doe",
			},
			expectError: false,
		},
		{
			name: "missing required field - strict mode",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "name", Type: "string", Required: true},
				},
				StrictMode: true,
			},
			input:       map[string]interface{}{},
			expected:    nil,
			expectError: true,
		},
		{
			name: "missing required field - non-strict mode with default",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "name", Type: "string", Required: true, Default: "Anonymous"},
				},
				StrictMode: false,
			},
			input: map[string]interface{}{},
			expected: map[string]interface{}{
				"name": "Anonymous",
			},
			expectError: false,
		},
		{
			name: "string too short - strict mode",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "name", Type: "string", Required: true, MinLen: 5},
				},
				StrictMode: true,
			},
			input: map[string]interface{}{
				"name": "Joe",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "string too long - non-strict mode",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "name", Type: "string", Required: true, MaxLen: 5, Default: "Short"},
				},
				StrictMode: false,
			},
			input: map[string]interface{}{
				"name": "Very Long Name",
			},
			expected: map[string]interface{}{
				"name": "Short",
			},
			expectError: false,
		},
		{
			name: "valid number field",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "age", Type: "number", Required: true},
				},
				StrictMode: true,
			},
			input: map[string]interface{}{
				"age": 25,
			},
			expected: map[string]interface{}{
				"age": 25,
			},
			expectError: false,
		},
		{
			name: "invalid number type",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "age", Type: "number", Required: true},
				},
				StrictMode: true,
			},
			input: map[string]interface{}{
				"age": "twenty-five",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "valid boolean field",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "active", Type: "boolean", Required: true},
				},
				StrictMode: true,
			},
			input: map[string]interface{}{
				"active": true,
			},
			expected: map[string]interface{}{
				"active": true,
			},
			expectError: false,
		},
		{
			name: "string with valid options",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "status", Type: "string", Required: true, Options: []string{"active", "inactive", "pending"}},
				},
				StrictMode: true,
			},
			input: map[string]interface{}{
				"status": "active",
			},
			expected: map[string]interface{}{
				"status": "active",
			},
			expectError: false,
		},
		{
			name: "string with invalid options",
			validator: &DataValidator{
				Rules: []ValidationRule{
					{Field: "status", Type: "string", Required: true, Options: []string{"active", "inactive", "pending"}},
				},
				StrictMode: true,
			},
			input: map[string]interface{}{
				"status": "unknown",
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.validator.Validate(ctx, tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !deepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRecordDeduplicator_Deduplicate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		deduplicator *RecordDeduplicator
		input        map[string]interface{}
		expected     map[string]interface{}
	}{
		{
			name: "hash method - pass through",
			deduplicator: &RecordDeduplicator{
				Method:    "hash",
				CacheSize: 1000,
			},
			input: map[string]interface{}{
				"title": "Test Title",
				"url":   "https://example.com",
			},
			expected: map[string]interface{}{
				"title": "Test Title",
				"url":   "https://example.com",
			},
		},
		{
			name: "field method - pass through",
			deduplicator: &RecordDeduplicator{
				Method:    "field",
				Fields:    []string{"url", "title"},
				CacheSize: 1000,
			},
			input: map[string]interface{}{
				"title": "Test Title",
				"url":   "https://example.com",
			},
			expected: map[string]interface{}{
				"title": "Test Title",
				"url":   "https://example.com",
			},
		},
		{
			name: "similarity method - pass through",
			deduplicator: &RecordDeduplicator{
				Method:    "similarity",
				Threshold: 0.8,
				CacheSize: 1000,
			},
			input: map[string]interface{}{
				"title": "Test Title",
				"content": "Some content here",
			},
			expected: map[string]interface{}{
				"title": "Test Title",
				"content": "Some content here",
			},
		},
		{
			name: "unknown method - pass through",
			deduplicator: &RecordDeduplicator{
				Method:    "unknown",
				CacheSize: 1000,
			},
			input: map[string]interface{}{
				"title": "Test Title",
			},
			expected: map[string]interface{}{
				"title": "Test Title",
			},
		},
		{
			name: "empty method - pass through",
			deduplicator: &RecordDeduplicator{
				CacheSize: 1000,
			},
			input: map[string]interface{}{
				"title": "Test Title",
			},
			expected: map[string]interface{}{
				"title": "Test Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.deduplicator.Deduplicate(ctx, tt.input)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !deepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDataEnricher_Enrich(t *testing.T) {
	ctx := context.Background()

	// Mock enricher for testing
	mockEnricher := &MockEnricher{
		name: "test_enricher",
		enrichFunc: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
			enriched := make(map[string]interface{})
			for k, v := range data {
				enriched[k] = v
			}
			enriched["enriched"] = true
			return enriched, nil
		},
	}

	tests := []struct {
		name     string
		enricher *DataEnricher
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "single enricher - sequential",
			enricher: &DataEnricher{
				Enrichers: []Enricher{mockEnricher},
				Timeout:   30 * time.Second,
				Parallel:  false,
			},
			input: map[string]interface{}{
				"title": "Test Title",
			},
			expected: map[string]interface{}{
				"title":    "Test Title",
				"enriched": true,
			},
		},
		{
			name: "single enricher - parallel (falls back to sequential)",
			enricher: &DataEnricher{
				Enrichers: []Enricher{mockEnricher},
				Timeout:   30 * time.Second,
				Parallel:  true,
			},
			input: map[string]interface{}{
				"title": "Test Title",
			},
			expected: map[string]interface{}{
				"title":    "Test Title",
				"enriched": true,
			},
		},
		{
			name: "no enrichers",
			enricher: &DataEnricher{
				Enrichers: []Enricher{},
				Timeout:   30 * time.Second,
				Parallel:  false,
			},
			input: map[string]interface{}{
				"title": "Test Title",
			},
			expected: map[string]interface{}{
				"title": "Test Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.enricher.Enrich(ctx, tt.input)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !deepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestOutputManager_Write(t *testing.T) {
	ctx := context.Background()

	// Mock output handler for testing
	mockOutput := &MockOutputHandler{
		writeFunc: func(ctx context.Context, data interface{}) error {
			return nil
		},
		outputType: "mock",
	}

	tests := []struct {
		name          string
		manager       *OutputManager
		input         interface{}
		expectError   bool
	}{
		{
			name: "single output handler",
			manager: &OutputManager{
				Outputs: []OutputHandler{mockOutput},
			},
			input:       map[string]interface{}{"test": "data"},
			expectError: false,
		},
		{
			name: "no output handlers",
			manager: &OutputManager{
				Outputs: []OutputHandler{},
			},
			input:       map[string]interface{}{"test": "data"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manager.Write(ctx, tt.input)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOutputManager_Close(t *testing.T) {
	tests := []struct {
		name        string
		manager     *OutputManager
		expectError bool
	}{
		{
			name: "successful close",
			manager: &OutputManager{
				Outputs: []OutputHandler{
					&MockOutputHandler{
						closeFunc: func() error { return nil },
						outputType: "mock1",
					},
				},
			},
			expectError: false,
		},
		{
			name: "no outputs",
			manager: &OutputManager{
				Outputs: []OutputHandler{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manager.Close()

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Helper functions and mock types for testing

func deepEqual(a, b interface{}) bool {
	// Simple deep equality check for testing purposes
	// In production, you might want to use reflect.DeepEqual or a more sophisticated comparison
	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for k, v := range aVal {
			if bv, exists := bVal[k]; !exists || !deepEqual(v, bv) {
				return false
			}
		}
		return true
	case []string:
		bVal, ok := b.([]string)
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for i, v := range aVal {
			if v != bVal[i] {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// Mock enricher for testing
type MockEnricher struct {
	name       string
	enrichFunc func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)
}

func (m *MockEnricher) Enrich(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	if m.enrichFunc != nil {
		return m.enrichFunc(ctx, data)
	}
	return data, nil
}

func (m *MockEnricher) GetName() string {
	return m.name
}

// Mock output handler for testing
type MockOutputHandler struct {
	writeFunc  func(ctx context.Context, data interface{}) error
	closeFunc  func() error
	outputType string
}

func (m *MockOutputHandler) Write(ctx context.Context, data interface{}) error {
	if m.writeFunc != nil {
		return m.writeFunc(ctx, data)
	}
	return nil
}

func (m *MockOutputHandler) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *MockOutputHandler) GetType() string {
	return m.outputType
}