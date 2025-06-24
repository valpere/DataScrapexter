// internal/scraper/extractor.go
package scraper

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/valpere/DataScrapexter/internal/pipeline"
)

// FieldExtractor handles the extraction of individual fields from HTML documents
type FieldExtractor struct {
	parser *HTMLParser
	config FieldConfig
}

// ExtractionEngine orchestrates field extraction operations across multiple fields
type ExtractionEngine struct {
	parser     *HTMLParser
	extractors []FieldExtractor
	config     ExtractionConfig
}

// ExtractionConfig defines configuration for the extraction engine
type ExtractionConfig struct {
	StrictMode          bool                     `json:"strict_mode" yaml:"strict_mode"`
	ContinueOnError     bool                     `json:"continue_on_error" yaml:"continue_on_error"`
	DefaultTransforms   []pipeline.TransformRule `json:"default_transforms,omitempty" yaml:"default_transforms,omitempty"`
	ValidationRules     []ValidationRule         `json:"validation_rules,omitempty" yaml:"validation_rules,omitempty"`
	TypeCoercion        bool                     `json:"type_coercion" yaml:"type_coercion"`
	FailureHandling     string                   `json:"failure_handling" yaml:"failure_handling"`
	RequiredFieldsOnly  bool                     `json:"required_fields_only" yaml:"required_fields_only"`
}

// ValidationRule defines field validation criteria
type ValidationRule struct {
	FieldName    string      `json:"field_name" yaml:"field_name"`
	MinLength    *int        `json:"min_length,omitempty" yaml:"min_length,omitempty"`
	MaxLength    *int        `json:"max_length,omitempty" yaml:"max_length,omitempty"`
	Pattern      string      `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	AllowedTypes []string    `json:"allowed_types,omitempty" yaml:"allowed_types,omitempty"`
	CustomRule   string      `json:"custom_rule,omitempty" yaml:"custom_rule,omitempty"`
	Required     bool        `json:"required" yaml:"required"`
}

// ExtractionResult represents the outcome of field extraction operations
type ExtractionResult struct {
	Data        map[string]interface{} `json:"data"`
	Errors      []FieldError           `json:"errors,omitempty"`
	Warnings    []FieldWarning         `json:"warnings,omitempty"`
	Metadata    ExtractionMetadata     `json:"metadata"`
	Success     bool                   `json:"success"`
	ProcessedAt time.Time              `json:"processed_at"`
}

// FieldError represents an error that occurred during field extraction
type FieldError struct {
	FieldName string `json:"field_name"`
	Selector  string `json:"selector"`
	Message   string `json:"message"`
	Code      string `json:"code"`
	Severity  string `json:"severity"`
}

// FieldWarning represents a warning generated during field extraction
type FieldWarning struct {
	FieldName string `json:"field_name"`
	Message   string `json:"message"`
	Code      string `json:"code"`
}

// ExtractionMetadata provides information about the extraction process
type ExtractionMetadata struct {
	TotalFields      int           `json:"total_fields"`
	ExtractedFields  int           `json:"extracted_fields"`
	FailedFields     int           `json:"failed_fields"`
	ProcessingTime   time.Duration `json:"processing_time"`
	RequiredFieldsOK bool          `json:"required_fields_ok"`
	DocumentSize     int           `json:"document_size"`
}

// NewFieldExtractor creates a new field extractor for a specific field configuration
func NewFieldExtractor(parser *HTMLParser, config FieldConfig) *FieldExtractor {
	return &FieldExtractor{
		parser: parser,
		config: config,
	}
}

// NewExtractionEngine creates a new extraction engine with the provided configuration
func NewExtractionEngine(parser *HTMLParser, fields []FieldConfig, config ExtractionConfig) *ExtractionEngine {
	extractors := make([]FieldExtractor, len(fields))
	for i, field := range fields {
		extractors[i] = *NewFieldExtractor(parser, field)
	}

	return &ExtractionEngine{
		parser:     parser,
		extractors: extractors,
		config:     config,
	}
}

// Extract performs field extraction for a single field configuration
func (fe *FieldExtractor) Extract(ctx context.Context) (interface{}, error) {
	if fe.parser == nil {
		return nil, fmt.Errorf("HTML parser not initialized for field '%s'", fe.config.Name)
	}

	// Validate field configuration before extraction
	if err := fe.validateConfig(); err != nil {
		return nil, fmt.Errorf("field configuration validation failed for '%s': %w", fe.config.Name, err)
	}

	// Perform the actual extraction using the parser
	value, err := fe.parser.ExtractField(fe.config)
	if err != nil {
		if fe.config.Required {
			return nil, fmt.Errorf("extraction failed for required field '%s': %w", fe.config.Name, err)
		}
		// For optional fields, return default value if extraction fails
		return fe.getDefaultValue(), nil
	}

	// If extraction succeeded but returned nil (element not found), handle based on field requirement
	if value == nil {
		if fe.config.Required {
			return nil, fmt.Errorf("required field '%s' not found with selector '%s'", fe.config.Name, fe.config.Selector)
		}
		// For optional fields, return default value when element not found
		return fe.getDefaultValue(), nil
	}

	// Apply transformations if configured
	if len(fe.config.Transform) > 0 {
		transformedValue, err := fe.applyTransformations(ctx, value)
		if err != nil {
			return nil, fmt.Errorf("transformation failed for field '%s': %w", fe.config.Name, err)
		}
		value = transformedValue
	}

	// Validate extracted value
	if err := fe.validateValue(value); err != nil {
		return nil, fmt.Errorf("value validation failed for field '%s': %w", fe.config.Name, err)
	}

	return value, nil
}

// ExtractAll performs extraction for all configured fields
func (ee *ExtractionEngine) ExtractAll(ctx context.Context) *ExtractionResult {
	startTime := time.Now()
	result := &ExtractionResult{
		Data:        make(map[string]interface{}),
		Errors:      []FieldError{},
		Warnings:    []FieldWarning{},
		ProcessedAt: startTime,
	}

	var requiredFieldsSuccess = true
	var extractedCount, failedCount int

	for _, extractor := range ee.extractors {
		value, err := extractor.Extract(ctx)
		
		if err != nil {
			failedCount++
			fieldError := FieldError{
				FieldName: extractor.config.Name,
				Selector:  extractor.config.Selector,
				Message:   err.Error(),
				Code:      "EXTRACTION_FAILED",
				Severity:  "ERROR",
			}

			if extractor.config.Required {
				requiredFieldsSuccess = false
				fieldError.Severity = "CRITICAL"
			}

			result.Errors = append(result.Errors, fieldError)

			// Handle extraction failure based on configuration
			if !ee.config.ContinueOnError && extractor.config.Required {
				result.Success = false
				result.Metadata = ee.buildMetadata(extractedCount, failedCount, len(ee.extractors), 
					time.Since(startTime), requiredFieldsSuccess)
				return result
			}
			continue
		}

		// Only add successfully extracted values to result data
		// Skip default values from optional fields that weren't actually found
		if value != nil && !ee.isDefaultValue(value, extractor.config) {
			result.Data[extractor.config.Name] = value
			extractedCount++
		} else if value != nil && extractor.config.Required {
			// Required field extracted successfully
			result.Data[extractor.config.Name] = value
			extractedCount++
		} else if value == nil && extractor.config.Required {
			// Required field extracted as nil - this is a problem
			requiredFieldsSuccess = false
			result.Errors = append(result.Errors, FieldError{
				FieldName: extractor.config.Name,
				Selector:  extractor.config.Selector,
				Message:   "Required field extracted as null value",
				Code:      "NULL_REQUIRED_FIELD",
				Severity:  "CRITICAL",
			})
			failedCount++
		}
	}

	// Apply global transformations if configured
	if len(ee.config.DefaultTransforms) > 0 {
		err := ee.applyGlobalTransformations(ctx, result.Data)
		if err != nil {
			result.Warnings = append(result.Warnings, FieldWarning{
				FieldName: "global",
				Message:   fmt.Sprintf("Global transformation warning: %v", err),
				Code:      "GLOBAL_TRANSFORM_WARNING",
			})
		}
	}

	// Perform global validation
	validationErrors := ee.performGlobalValidation(result.Data)
	result.Errors = append(result.Errors, validationErrors...)

	// Determine overall success
	result.Success = requiredFieldsSuccess && (ee.config.StrictMode == false || len(result.Errors) == 0)
	result.Metadata = ee.buildMetadata(extractedCount, failedCount, len(ee.extractors), 
		time.Since(startTime), requiredFieldsSuccess)

	return result
}

// validateConfig validates the field configuration before extraction
func (fe *FieldExtractor) validateConfig() error {
	if strings.TrimSpace(fe.config.Name) == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	if strings.TrimSpace(fe.config.Selector) == "" {
		return fmt.Errorf("field selector cannot be empty")
	}

	validTypes := []string{"text", "html", "attribute", "href", "src", "int", "number", "float", "bool", "boolean", "date", "array"}
	typeValid := false
	for _, validType := range validTypes {
		if fe.config.Type == validType {
			typeValid = true
			break
		}
	}

	if !typeValid {
		return fmt.Errorf("invalid field type '%s', must be one of: %s", fe.config.Type, strings.Join(validTypes, ", "))
	}

	return nil
}

// getDefaultValue returns the appropriate default value for the field type
func (fe *FieldExtractor) getDefaultValue() interface{} {
	if fe.config.Default != nil {
		return fe.config.Default
	}

	switch fe.config.Type {
	case "int", "number":
		return 0
	case "float":
		return 0.0
	case "bool", "boolean":
		return false
	case "array":
		return []interface{}{}
	case "date":
		return time.Time{}
	default:
		return ""
	}
}

// applyTransformations applies configured transformations to the extracted value
func (fe *FieldExtractor) applyTransformations(ctx context.Context, value interface{}) (interface{}, error) {
	if len(fe.config.Transform) == 0 {
		return value, nil
	}

	// Convert value to string for transformation if it's not already
	stringValue, ok := value.(string)
	if !ok {
		stringValue = fmt.Sprintf("%v", value)
	}

	transformList := pipeline.TransformList(fe.config.Transform)
	transformedValue, err := transformList.Apply(ctx, stringValue)
	if err != nil {
		return nil, err
	}

	// Try to convert back to the original type if possible
	return fe.coerceType(transformedValue)
}

// coerceType attempts to convert the transformed string back to the expected field type
func (fe *FieldExtractor) coerceType(value string) (interface{}, error) {
	switch fe.config.Type {
	case "int", "number":
		if parser := fe.parser; parser != nil {
			return parser.parseInt(value)
		}
		return value, nil
	case "float":
		if parser := fe.parser; parser != nil {
			return parser.parseFloat(value)
		}
		return value, nil
	case "bool", "boolean":
		if parser := fe.parser; parser != nil {
			return parser.parseBool(value), nil
		}
		return value, nil
	case "date":
		if parser := fe.parser; parser != nil {
			return parser.parseDate(value, "2006-01-02")
		}
		return value, nil
	default:
		return value, nil
	}
}

// validateValue validates the extracted value against field constraints
func (fe *FieldExtractor) validateValue(value interface{}) error {
	if value == nil && fe.config.Required {
		return fmt.Errorf("required field cannot have null value")
	}

	if value == nil {
		return nil // Optional field with null value is acceptable
	}

	// Type-specific validation
	switch fe.config.Type {
	case "text", "html", "attribute", "href", "src":
		return fe.validateStringValue(value)
	case "int", "number":
		return fe.validateNumericValue(value)
	case "float":
		return fe.validateFloatValue(value)
	case "array":
		return fe.validateArrayValue(value)
	default:
		return nil
	}
}

// validateStringValue validates string-type field values
func (fe *FieldExtractor) validateStringValue(value interface{}) error {
	stringValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string value but got %T", value)
	}

	if fe.config.Required && strings.TrimSpace(stringValue) == "" {
		return fmt.Errorf("required string field cannot be empty")
	}

	return nil
}

// validateNumericValue validates numeric field values
func (fe *FieldExtractor) validateNumericValue(value interface{}) error {
	switch value.(type) {
	case int, int32, int64, float32, float64:
		return nil
	default:
		return fmt.Errorf("expected numeric value but got %T", value)
	}
}

// validateFloatValue validates float field values
func (fe *FieldExtractor) validateFloatValue(value interface{}) error {
	switch value.(type) {
	case float32, float64, int, int32, int64:
		return nil
	default:
		return fmt.Errorf("expected float value but got %T", value)
	}
}

// validateArrayValue validates array field values
func (fe *FieldExtractor) validateArrayValue(value interface{}) error {
	arrayValue := reflect.ValueOf(value)
	if arrayValue.Kind() != reflect.Slice && arrayValue.Kind() != reflect.Array {
		return fmt.Errorf("expected array value but got %T", value)
	}

	if fe.config.Required && arrayValue.Len() == 0 {
		return fmt.Errorf("required array field cannot be empty")
	}

	return nil
}

// applyGlobalTransformations applies global transformations to all extracted string fields
func (ee *ExtractionEngine) applyGlobalTransformations(ctx context.Context, data map[string]interface{}) error {
	transformList := pipeline.TransformList(ee.config.DefaultTransforms)
	
	for key, value := range data {
		if stringValue, ok := value.(string); ok {
			transformedValue, err := transformList.Apply(ctx, stringValue)
			if err != nil {
				return fmt.Errorf("global transformation failed for field '%s': %w", key, err)
			}
			data[key] = transformedValue
		}
	}

	return nil
}

// performGlobalValidation performs validation rules across all extracted data
func (ee *ExtractionEngine) performGlobalValidation(data map[string]interface{}) []FieldError {
	var errors []FieldError

	for _, rule := range ee.config.ValidationRules {
		value, exists := data[rule.FieldName]
		
		if !exists && rule.Required {
			errors = append(errors, FieldError{
				FieldName: rule.FieldName,
				Message:   "Required field missing from extraction result",
				Code:      "MISSING_REQUIRED_FIELD",
				Severity:  "CRITICAL",
			})
			continue
		}

		if !exists {
			continue // Optional field not present - acceptable
		}

		if err := ee.validateFieldAgainstRule(value, rule); err != nil {
			errors = append(errors, FieldError{
				FieldName: rule.FieldName,
				Message:   err.Error(),
				Code:      "VALIDATION_FAILED",
				Severity:  "ERROR",
			})
		}
	}

	return errors
}

// validateFieldAgainstRule validates a field value against a specific validation rule
func (ee *ExtractionEngine) validateFieldAgainstRule(value interface{}, rule ValidationRule) error {
	if stringValue, ok := value.(string); ok {
		if rule.MinLength != nil && len(stringValue) < *rule.MinLength {
			return fmt.Errorf("field length %d is below minimum %d", len(stringValue), *rule.MinLength)
		}
		
		if rule.MaxLength != nil && len(stringValue) > *rule.MaxLength {
			return fmt.Errorf("field length %d exceeds maximum %d", len(stringValue), *rule.MaxLength)
		}
	}

	if len(rule.AllowedTypes) > 0 {
		valueType := fmt.Sprintf("%T", value)
		typeAllowed := false
		for _, allowedType := range rule.AllowedTypes {
			if strings.Contains(valueType, allowedType) {
				typeAllowed = true
				break
			}
		}
		if !typeAllowed {
			return fmt.Errorf("field type %s not in allowed types %v", valueType, rule.AllowedTypes)
		}
	}

	return nil
}

// isDefaultValue checks if the extracted value matches the default value for the field
func (ee *ExtractionEngine) isDefaultValue(value interface{}, config FieldConfig) bool {
	// If a custom default is configured, check against it
	if config.Default != nil {
		return value == config.Default
	}

	// Check against type-specific default values
	switch config.Type {
	case "int", "number":
		return value == 0
	case "float":
		return value == 0.0
	case "bool", "boolean":
		return value == false
	case "array":
		if arr, ok := value.([]interface{}); ok {
			return len(arr) == 0
		}
		return false
	case "date":
		if t, ok := value.(time.Time); ok {
			return t.IsZero()
		}
		return false
	default: // text, html, attribute, href, src
		return value == ""
	}
}
func (ee *ExtractionEngine) buildMetadata(extracted, failed, total int, duration time.Duration, requiredOK bool) ExtractionMetadata {
	documentSize := 0
	if ee.parser != nil && ee.parser.document != nil {
		if html, err := ee.parser.document.Html(); err == nil {
			documentSize = len(html)
		}
	}

	return ExtractionMetadata{
		TotalFields:      total,
		ExtractedFields:  extracted,
		FailedFields:     failed,
		ProcessingTime:   duration,
		RequiredFieldsOK: requiredOK,
		DocumentSize:     documentSize,
	}
}
