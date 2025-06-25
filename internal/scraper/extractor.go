// internal/scraper/extractor.go
package scraper

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valpere/DataScrapexter/internal/pipeline"
)

// ExtractionResult represents the complete result of field extraction
type ExtractionResult struct {
	Data        map[string]interface{} `json:"data"`
	Errors      []FieldError          `json:"errors,omitempty"`
	Warnings    []FieldWarning        `json:"warnings,omitempty"`
	Success     bool                  `json:"success"`
	Metadata    ExtractionMetadata    `json:"metadata"`
	ProcessedAt time.Time             `json:"processed_at"`
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

// FieldExtractor handles extraction and transformation of individual fields
type FieldExtractor struct {
	config   FieldConfig
	document *goquery.Document
	parser   *ElementParser
}

// ExtractionEngine orchestrates field extraction for multiple fields
type ExtractionEngine struct {
	extractors []FieldExtractor
	document   *goquery.Document
	config     ExtractionConfig
	parser     *ElementParser
}

// ElementParser handles type-specific parsing and conversion
type ElementParser struct {
	document *goquery.Document
}

// NewExtractionEngine creates a new field extraction engine
func NewExtractionEngine(config ExtractionConfig, document *goquery.Document) *ExtractionEngine {
	parser := &ElementParser{document: document}
	
	extractors := make([]FieldExtractor, len(config.Fields))
	for i, fieldConfig := range config.Fields {
		extractors[i] = FieldExtractor{
			config:   fieldConfig,
			document: document,
			parser:   parser,
		}
	}

	return &ExtractionEngine{
		extractors: extractors,
		document:   document,
		config:     config,
		parser:     parser,
	}
}

// NewFieldExtractor creates a new field extractor for a specific field
func NewFieldExtractor(config FieldConfig, document *goquery.Document) *FieldExtractor {
	return &FieldExtractor{
		config:   config,
		document: document,
		parser:   &ElementParser{document: document},
	}
}

// Extract performs field extraction with proper transformation integration
func (fe *FieldExtractor) Extract(ctx context.Context) (interface{}, error) {
	if err := fe.validateConfig(); err != nil {
		return nil, fmt.Errorf("field configuration invalid: %w", err)
	}

	value, err := fe.extractRawValue()
	if err != nil {
		return nil, fmt.Errorf("raw extraction failed: %w", err)
	}

	if value == nil {
		if fe.config.Required {
			return nil, fmt.Errorf("required field '%s' not found", fe.config.Name)
		}
		return fe.getDefaultValue(), nil
	}

	// Apply transformations if configured
	if len(fe.config.Transform) > 0 {
		stringValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("transformations can only be applied to string values, got %T", value)
		}

		transformList := pipeline.TransformList(fe.config.Transform)
		transformedValue, err := transformList.Apply(ctx, stringValue)
		if err != nil {
			return nil, fmt.Errorf("transformation failed: %w", err)
		}

		// Convert back to appropriate type if needed
		value = transformedValue
	}

	// Validate the final value
	if err := fe.validateValue(value); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
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

	extractedCount := 0
	failedCount := 0
	requiredFieldsOK := true

	// Process each field
	for _, extractor := range ee.extractors {
		fieldValue, err := extractor.Extract(ctx)
		
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
				fieldError.Severity = "CRITICAL"
				requiredFieldsOK = false
			}
			
			result.Errors = append(result.Errors, fieldError)
			
			// Continue on error if configured
			if !ee.config.ContinueOnError {
				break
			}
		} else {
			result.Data[extractor.config.Name] = fieldValue
			extractedCount++
		}
	}

	// Apply global transformations
	if len(ee.config.DefaultTransforms) > 0 {
		if err := ee.applyGlobalTransformations(ctx, result.Data); err != nil {
			result.Errors = append(result.Errors, FieldError{
				FieldName: "global",
				Message:   err.Error(),
				Code:      "GLOBAL_TRANSFORM_FAILED",
				Severity:  "ERROR",
			})
		}
	}

	// Perform global validation
	validationErrors := ee.performGlobalValidation(result.Data)
	result.Errors = append(result.Errors, validationErrors...)

	// Set success status
	result.Success = len(result.Errors) == 0 || (ee.config.ContinueOnError && requiredFieldsOK)
	
	// Build metadata
	totalFields := len(ee.extractors)
	processingTime := time.Since(startTime)
	result.Metadata = ee.buildMetadata(extractedCount, failedCount, totalFields, processingTime, requiredFieldsOK)

	return result
}

// extractRawValue extracts the raw value from the HTML document
func (fe *FieldExtractor) extractRawValue() (interface{}, error) {
	selection := fe.document.Find(fe.config.Selector)
	
	if selection.Length() == 0 {
		return nil, nil
	}

	switch fe.config.Type {
	case "text":
		return strings.TrimSpace(selection.First().Text()), nil
		
	case "html":
		html, err := selection.First().Html()
		if err != nil {
			return nil, err
		}
		return strings.TrimSpace(html), nil
		
	case "attribute":
		if fe.config.Attribute == "" {
			return nil, fmt.Errorf("attribute name required for attribute extraction")
		}
		value, exists := selection.First().Attr(fe.config.Attribute)
		if !exists {
			return nil, nil
		}
		return strings.TrimSpace(value), nil
		
	case "href":
		href, exists := selection.First().Attr("href")
		if !exists {
			return nil, nil
		}
		return strings.TrimSpace(href), nil
		
	case "src":
		src, exists := selection.First().Attr("src")
		if !exists {
			return nil, nil
		}
		return strings.TrimSpace(src), nil
		
	case "int", "number":
		text := strings.TrimSpace(selection.First().Text())
		if text == "" {
			return nil, nil
		}
		value, err := strconv.Atoi(text)
		if err != nil {
			return nil, fmt.Errorf("failed to parse integer: %w", err)
		}
		return value, nil
		
	case "float":
		text := strings.TrimSpace(selection.First().Text())
		if text == "" {
			return nil, nil
		}
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse float: %w", err)
		}
		return value, nil
		
	case "bool", "boolean":
		text := strings.ToLower(strings.TrimSpace(selection.First().Text()))
		switch text {
		case "true", "yes", "1", "on":
			return true, nil
		case "false", "no", "0", "off":
			return false, nil
		default:
			return nil, fmt.Errorf("cannot parse boolean from: %s", text)
		}
		
	case "array":
		var results []string
		selection.Each(func(i int, s *goquery.Selection) {
			results = append(results, strings.TrimSpace(s.Text()))
		})
		// Convert to []interface{} for consistency
		interfaceResults := make([]interface{}, len(results))
		for i, v := range results {
			interfaceResults[i] = v
		}
		return interfaceResults, nil
		
	case "date":
		text := strings.TrimSpace(selection.First().Text())
		if text == "" {
			return nil, nil
		}
		// Try common date formats
		formats := []string{
			"2006-01-02",
			"2006-01-02T15:04:05Z",
			"January 2, 2006",
			"Jan 2, 2006",
			"02/01/2006",
			"01/02/2006",
		}
		
		for _, format := range formats {
			if date, err := time.Parse(format, text); err == nil {
				return date, nil
			}
		}
		return nil, fmt.Errorf("failed to parse date: %s", text)
		
	default:
		return nil, fmt.Errorf("unsupported field type: %s", fe.config.Type)
	}
}

// validateConfig validates the field configuration
func (fe *FieldExtractor) validateConfig() error {
	if fe.config.Name == "" {
		return fmt.Errorf("field name cannot be empty")
	}
	
	if fe.config.Selector == "" {
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

// validateValue validates the extracted value against field constraints
func (fe *FieldExtractor) validateValue(value interface{}) error {
	if value == nil && fe.config.Required {
		return fmt.Errorf("required field cannot have null value")
	}

	if value == nil {
		return nil
	}

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
			continue
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

// buildMetadata constructs extraction metadata from processing results
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
