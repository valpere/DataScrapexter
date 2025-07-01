// internal/scraper/extractor.go
package scraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valpere/DataScrapexter/internal/pipeline"
)

// FieldExtractor handles extraction and transformation of individual fields
type FieldExtractor struct {
	config   FieldConfig
	document *goquery.Document
}

// ExtractionEngine orchestrates field extraction for multiple fields
type ExtractionEngine struct {
	fields   []FieldConfig
	document *goquery.Document
	config   ExtractionConfig
}

// NewExtractionEngine creates a new field extraction engine
func NewExtractionEngine(fields []FieldConfig, config ExtractionConfig, document *goquery.Document) *ExtractionEngine {
	return &ExtractionEngine{
		fields:   fields,
		document: document,
		config:   config,
	}
}

// NewFieldExtractor creates a new field extractor for a specific field
func NewFieldExtractor(config FieldConfig, document *goquery.Document) *FieldExtractor {
	return &FieldExtractor{
		config:   config,
		document: document,
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
		stringValue := fmt.Sprintf("%v", value)
		transformList := pipeline.TransformList(fe.config.Transform)
		transformedValue, err := transformList.Apply(ctx, stringValue)
		if err != nil {
			return nil, fmt.Errorf("transformation failed: %w", err)
		}
		value = transformedValue
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

	// Process each field - use ee.fields instead of config.Fields
	for _, fieldConfig := range ee.fields {
		extractor := NewFieldExtractor(fieldConfig, ee.document)
		fieldValue, err := extractor.Extract(ctx)

		if err != nil {
			failedCount++

			fieldError := FieldError{
				FieldName: fieldConfig.Name,
				Selector:  fieldConfig.Selector,
				Message:   err.Error(),
				Code:      "EXTRACTION_FAILED",
				Severity:  "ERROR",
			}

			if fieldConfig.Required {
				fieldError.Severity = "CRITICAL"
				requiredFieldsOK = false
			}

			result.Errors = append(result.Errors, fieldError)

			// Continue on error if configured
			if !ee.config.ContinueOnError {
				break
			}
		} else {
			result.Data[fieldConfig.Name] = fieldValue
			extractedCount++
		}
	}

	duration := time.Since(startTime)
	result.Success = requiredFieldsOK && (ee.config.ContinueOnError || failedCount == 0)
	result.Metadata = ee.buildMetadata(extractedCount, failedCount, len(ee.fields), duration, requiredFieldsOK)

	return result
}

// validateConfig validates the field configuration
func (fe *FieldExtractor) validateConfig() error {
	if fe.config.Name == "" {
		return fmt.Errorf("field name is required")
	}
	if fe.config.Selector == "" {
		return fmt.Errorf("field selector is required")
	}
	if fe.config.Type == "" {
		return fmt.Errorf("field type is required")
	}

	validTypes := map[string]bool{
		"text": true, "html": true, "attr": true, "list": true,
	}
	if !validTypes[fe.config.Type] {
		return fmt.Errorf("invalid field type: %s", fe.config.Type)
	}

	if fe.config.Type == "attr" && fe.config.Attribute == "" {
		return fmt.Errorf("attribute name required for attr type")
	}

	return nil
}

// extractRawValue extracts the raw value based on field type
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
		return html, err
	case "attr":
		attr, exists := selection.First().Attr(fe.config.Attribute)
		if !exists {
			return nil, nil
		}
		return attr, nil
	case "list":
		var items []string
		selection.Each(func(i int, s *goquery.Selection) {
			items = append(items, strings.TrimSpace(s.Text()))
		})
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported field type: %s", fe.config.Type)
	}
}

// getDefaultValue returns the default value for the field
func (fe *FieldExtractor) getDefaultValue() interface{} {
	if fe.config.Default != nil {
		return fe.config.Default
	}

	switch fe.config.Type {
	case "text", "html", "attr":
		return ""
	case "list":
		return []string{}
	default:
		return ""
	}
}

// buildMetadata constructs extraction metadata from processing results
func (ee *ExtractionEngine) buildMetadata(extracted, failed, total int, duration time.Duration, requiredOK bool) ExtractionMetadata {
	documentSize := int64(0)
	if ee.document != nil {
		if html, err := ee.document.Html(); err == nil {
			documentSize = int64(len(html))
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
