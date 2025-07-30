// internal/scraper/extractor.go
package scraper

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valpere/DataScrapexter/internal/pipeline"
	"github.com/valpere/DataScrapexter/internal/utils"
)

var extractorLogger = utils.NewComponentLogger("field-extractor")

// Pre-compiled regular expressions for performance
var (
	numberCleanRegex = regexp.MustCompile(`[^\d.+-]`)                    // Remove non-numeric characters except digits, decimal points, plus and minus signs
	integerRegex     = regexp.MustCompile(`[+-]?\d+`)                   // Allow optional plus/minus prefix
	emailRegex       = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phoneRegex       = regexp.MustCompile(`[\+]?[0-9][\d\s\-\(\)\.]{7,15}`) // Allow phone numbers starting with any digit (0-9) after an optional plus sign
	phoneCleanRegex  = regexp.MustCompile(`[^\d\+]`)
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
		// Enhanced field types
		"number": true, "float": true, "integer": true, "boolean": true,
		"date": true, "datetime": true, "time": true,
		"url": true, "email": true, "phone": true,
		"json": true, "csv": true, "table": true,
		"count": true, "exists": true,
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

	// Numeric types
	case "number", "float":
		return fe.extractNumber(selection.First())

	case "integer":
		return fe.extractInteger(selection.First())

	// Boolean type
	case "boolean":
		return fe.extractBoolean(selection.First())

	// Date/time types
	case "date":
		return fe.extractDate(selection.First())

	case "datetime":
		return fe.extractDateTime(selection.First())

	case "time":
		return fe.extractTime(selection.First())

	// URL and communication types
	case "url":
		return fe.extractURL(selection.First())

	case "email":
		return fe.extractEmail(selection.First())

	case "phone":
		return fe.extractPhone(selection.First())

	// Structured data types
	case "json":
		return fe.extractJSON(selection.First())

	case "csv":
		return fe.extractCSV(selection.First())

	case "table":
		return fe.extractTable(selection)

	// Utility types
	case "count":
		return selection.Length(), nil

	case "exists":
		return selection.Length() > 0, nil

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
	case "text", "html", "attr", "url", "email", "phone", "date", "datetime", "time":
		return ""
	case "list", "csv":
		return []string{}
	case "number", "float":
		return 0.0
	case "integer", "count":
		return 0
	case "boolean", "exists":
		return false
	case "json", "table":
		return make(map[string]interface{})
	default:
		return ""
	}
}

// extractNumber extracts and parses a floating-point number
func (fe *FieldExtractor) extractNumber(selection *goquery.Selection) (float64, error) {
	text := strings.TrimSpace(selection.Text())
	if text == "" {
		return 0.0, nil
	}

	// Clean common number formatting
	cleaned := numberCleanRegex.ReplaceAllString(text, "")
	if cleaned == "" {
		return 0.0, fmt.Errorf("no numeric value found in: %s", text)
	}

	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0.0, fmt.Errorf("failed to parse number '%s': %w", cleaned, err)
	}

	return value, nil
}

// extractInteger extracts and parses an integer
func (fe *FieldExtractor) extractInteger(selection *goquery.Selection) (int64, error) {
	text := strings.TrimSpace(selection.Text())
	if text == "" {
		return 0, nil
	}

	// Extract first integer from text
	match := integerRegex.FindString(text)
	if match == "" {
		return 0, fmt.Errorf("no integer value found in: %s", text)
	}

	value, err := strconv.ParseInt(match, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer '%s': %w", match, err)
	}

	return value, nil
}

// extractBoolean extracts and parses a boolean value
//
// Boolean extraction logic:
// 1. Explicit true values: "true", "yes", "1", "on", "enabled", "active", "available", "checked", "selected", "valid"
// 2. Explicit false values: "false", "no", "0", "off", "disabled", "inactive", "unavailable", etc.
// 3. Common negative phrases: "out of stock", "sold out", "not available", "coming soon", etc.
// 4. CSS classes: "active", "enabled", "checked" → true; "disabled", "inactive", "unchecked" → false
// 5. HTML attributes: "checked" → true; "disabled" → false
// 6. Unrecognized text: defaults to true with warning log
func (fe *FieldExtractor) extractBoolean(selection *goquery.Selection) (bool, error) {
	text := strings.ToLower(strings.TrimSpace(selection.Text()))

	// Explicit true values
	trueValues := map[string]bool{
		"true": true, "yes": true, "1": true, "on": true,
		"enabled": true, "active": true, "available": true,
		"checked": true, "selected": true, "valid": true,
	}

	// Explicit false values
	falseValues := map[string]bool{
		"false": true, "no": true, "0": true, "off": true,
		"disabled": true, "inactive": true, "unavailable": true,
		"unchecked": true, "unselected": true, "invalid": true,
		"null": true, "none": true, "empty": true,
		// Common negative phrases
		"out of stock": true, "sold out": true, "not available": true,
		"not in stock": true, "temporarily unavailable": true,
		"discontinued": true, "coming soon": true, "pre-order": true,
		"pending": true, "suspended": true, "expired": true,
		"closed": true, "locked": true, "blocked": true,
	}

	// Check explicit boolean text values
	if trueValues[text] {
		return true, nil
	}
	if falseValues[text] {
		return false, nil
	}

	// If no text content, check for boolean-indicating CSS classes or attributes
	if text == "" {
		// Check for positive indicators
		if selection.HasClass("active") || selection.HasClass("enabled") || selection.HasClass("checked") {
			return true, nil
		}
		// Check for negative indicators
		if selection.HasClass("disabled") || selection.HasClass("inactive") || selection.HasClass("unchecked") {
			return false, nil
		}
		// Check for boolean attributes
		if _, exists := selection.Attr("checked"); exists {
			return true, nil
		}
		if _, exists := selection.Attr("disabled"); exists {
			return false, nil
		}
		return false, nil
	}

	// For any other non-empty text, we need to be explicit about the behavior
	// Default: treat non-empty unrecognized text as true (document this behavior)
	extractorLogger.Warn(fmt.Sprintf("Boolean extraction: unrecognized text '%s' treated as true", text))
	return true, nil
}

// extractDate extracts and parses a date
func (fe *FieldExtractor) extractDate(selection *goquery.Selection) (string, error) {
	var text string

	// First check for datetime attribute
	if datetime, exists := selection.Attr("datetime"); exists {
		text = datetime
	} else {
		// Fall back to text content
		text = strings.TrimSpace(selection.Text())
	}

	if text == "" {
		return "", nil
	}

	// Try to parse various date formats
	dateFormats := []string{
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
	}

	for _, format := range dateFormats {
		if parsed, err := time.Parse(format, text); err == nil {
			return parsed.Format("2006-01-02"), nil
		}
	}

	extractorLogger.Warn(fmt.Sprintf("Could not parse date '%s', returning as-is", text))
	return text, nil
}

// extractDateTime extracts and parses a datetime
func (fe *FieldExtractor) extractDateTime(selection *goquery.Selection) (string, error) {
	var text string

	// First check for datetime attribute
	if datetime, exists := selection.Attr("datetime"); exists {
		text = datetime
	} else {
		// Fall back to text content
		text = strings.TrimSpace(selection.Text())
	}

	if text == "" {
		return "", nil
	}

	// Try to parse various datetime formats
	datetimeFormats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		"January 2, 2006 3:04 PM",
		"Jan 2, 2006 3:04 PM",
		"2006-01-02",
	}

	for _, format := range datetimeFormats {
		if parsed, err := time.Parse(format, text); err == nil {
			return parsed.Format("2006-01-02T15:04:05Z"), nil
		}
	}

	extractorLogger.Warn(fmt.Sprintf("Could not parse datetime '%s', returning as-is", text))
	return text, nil
}

// extractTime extracts and parses a time
func (fe *FieldExtractor) extractTime(selection *goquery.Selection) (string, error) {
	text := strings.TrimSpace(selection.Text())
	if text == "" {
		return "", nil
	}

	// Try to parse various time formats
	timeFormats := []string{
		"15:04:05",
		"15:04",
		"3:04 PM",
		"3:04:05 PM",
	}

	for _, format := range timeFormats {
		if parsed, err := time.Parse(format, text); err == nil {
			return parsed.Format("15:04:05"), nil
		}
	}

	extractorLogger.Warn(fmt.Sprintf("Could not parse time '%s', returning as-is", text))
	return text, nil
}

// extractURL extracts and validates a URL
// 
// URL extraction behavior:
// 1. Attempts href attribute first (for links)
// 2. Falls back to src attribute (for images, scripts)
// 3. Falls back to text content
// 4. Tries to resolve relative URLs using document base URL
// 5. Returns relative URLs as-is if no base URL available (with warning)
func (fe *FieldExtractor) extractURL(selection *goquery.Selection) (string, error) {
	var urlStr string

	// First try to get URL from href attribute
	if href, exists := selection.Attr("href"); exists {
		urlStr = href
	} else if src, exists := selection.Attr("src"); exists {
		// Try src attribute for images, etc.
		urlStr = src
	} else {
		// Fall back to text content
		urlStr = strings.TrimSpace(selection.Text())
	}

	if urlStr == "" {
		return "", nil
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL '%s': %w", urlStr, err)
	}

	// Handle relative URLs by attempting base URL resolution
	if parsedURL.Scheme == "" {
		// Try to find a base URL from the document
		baseURL := fe.findDocumentBaseURL()
		if baseURL != nil {
			// Resolve relative URL against base URL
			absoluteURL := baseURL.ResolveReference(parsedURL)
			extractorLogger.Info(fmt.Sprintf("Resolved relative URL '%s' to '%s'", urlStr, absoluteURL.String()))
			return absoluteURL.String(), nil
		} else {
			// No base URL available, return relative URL with warning
			extractorLogger.Warn(fmt.Sprintf("Relative URL found '%s' - no base URL available, returning as-is", urlStr))
		}
	}

	return parsedURL.String(), nil
}

// findDocumentBaseURL attempts to find the base URL from the document
func (fe *FieldExtractor) findDocumentBaseURL() *url.URL {
	// Check for HTML <base> tag first
	baseSelection := fe.document.Find("base[href]").First()
	if baseSelection.Length() > 0 {
		if href, exists := baseSelection.Attr("href"); exists {
			if baseURL, err := url.Parse(href); err == nil && baseURL.Scheme != "" {
				return baseURL
			}
		}
	}

	// Try to extract base URL from the document URL if available in meta tags
	canonicalSelection := fe.document.Find("link[rel='canonical'][href]").First()
	if canonicalSelection.Length() > 0 {
		if href, exists := canonicalSelection.Attr("href"); exists {
			if canonicalURL, err := url.Parse(href); err == nil && canonicalURL.Scheme != "" {
				// Use the canonical URL's base (scheme + host)
				baseURL := &url.URL{
					Scheme: canonicalURL.Scheme,
					Host:   canonicalURL.Host,
				}
				return baseURL
			}
		}
	}

	// Could not determine base URL
	return nil
}

// extractEmail extracts and validates an email address
func (fe *FieldExtractor) extractEmail(selection *goquery.Selection) (string, error) {
	text := strings.TrimSpace(selection.Text())

	// Also check href attribute for mailto links
	if href, exists := selection.Attr("href"); exists && strings.HasPrefix(href, "mailto:") {
		text = strings.TrimPrefix(href, "mailto:")
	}

	if text == "" {
		return "", nil
	}

	// Extract email pattern from text
	match := emailRegex.FindString(text)
	if match == "" {
		return "", fmt.Errorf("no valid email found in: %s", text)
	}

	// Validate email
	if _, err := mail.ParseAddress(match); err != nil {
		return "", fmt.Errorf("invalid email '%s': %w", match, err)
	}

	return match, nil
}

// extractPhone extracts and formats a phone number
func (fe *FieldExtractor) extractPhone(selection *goquery.Selection) (string, error) {
	text := strings.TrimSpace(selection.Text())

	// Also check href attribute for tel links
	if href, exists := selection.Attr("href"); exists && strings.HasPrefix(href, "tel:") {
		text = strings.TrimPrefix(href, "tel:")
	}

	if text == "" {
		return "", nil
	}

	// Extract phone number pattern (basic international format)
	match := phoneRegex.FindString(text)
	if match == "" {
		return "", fmt.Errorf("no valid phone number found in: %s", text)
	}

	// Clean up the phone number
	cleaned := phoneCleanRegex.ReplaceAllString(match, "")
	return cleaned, nil
}

// extractJSON extracts and parses JSON data
func (fe *FieldExtractor) extractJSON(selection *goquery.Selection) (interface{}, error) {
	text := strings.TrimSpace(selection.Text())
	if text == "" {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

// extractCSV extracts and parses CSV data
func (fe *FieldExtractor) extractCSV(selection *goquery.Selection) ([][]string, error) {
	text := strings.TrimSpace(selection.Text())
	if text == "" {
		return nil, nil
	}

	reader := csv.NewReader(strings.NewReader(text))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	return records, nil
}

// extractTable extracts table data into a structured format
func (fe *FieldExtractor) extractTable(selection *goquery.Selection) (interface{}, error) {
	// Find the table element
	table := selection.Filter("table").First()
	if table.Length() == 0 {
		table = selection.Find("table").First()
	}

	if table.Length() == 0 {
		return nil, fmt.Errorf("no table found")
	}

	var headers []string
	var rows []map[string]interface{}

	// Extract headers
	table.Find("thead tr th, tbody tr:first-child th, tr:first-child th").Each(func(i int, s *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(s.Text()))
	})

	// If no headers found, create generic ones
	if len(headers) == 0 {
		// Count columns from first row
		firstRow := table.Find("tbody tr, tr").First()
		if firstRow.Length() > 0 {
			firstRow.Find("td, th").Each(func(i int, s *goquery.Selection) {
				headers = append(headers, fmt.Sprintf("column_%d", i+1))
			})
		}
	}

	// Extract data rows
	if table.Find("tbody").Length() > 0 {
		// If there's a tbody, only extract from tbody
		table.Find("tbody tr").Each(func(i int, row *goquery.Selection) {
			rowData := make(map[string]interface{})
			row.Find("td").Each(func(j int, cell *goquery.Selection) {
				if j < len(headers) {
					cellText := strings.TrimSpace(cell.Text())
					rowData[headers[j]] = cellText
				}
			})

			if len(rowData) > 0 {
				rows = append(rows, rowData)
			}
		})
	} else {
		// If no tbody, extract from all tr but skip header row
		table.Find("tr").Each(func(i int, row *goquery.Selection) {
			// Skip first row if it contains th elements (header row)
			if i == 0 && row.Find("th").Length() > 0 {
				return
			}

			rowData := make(map[string]interface{})
			row.Find("td").Each(func(j int, cell *goquery.Selection) {
				if j < len(headers) {
					cellText := strings.TrimSpace(cell.Text())
					rowData[headers[j]] = cellText
				}
			})

			if len(rowData) > 0 {
				rows = append(rows, rowData)
			}
		})
	}

	return map[string]interface{}{
		"headers": headers,
		"rows":    rows,
		"count":   len(rows),
	}, nil
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
