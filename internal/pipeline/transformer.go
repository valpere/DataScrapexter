// internal/pipeline/transformer.go

// Package pipeline provides a flexible data transformation pipeline for processing
// scraped content through a series of configurable transformation steps.
//
// The pipeline package is designed to handle various data transformations commonly
// needed in web scraping, including text cleaning, format conversion, validation,
// and data enrichment. Each transformer is composable and can be chained together
// to create complex data processing workflows.
//
// Basic usage:
//
//	pipeline := pipeline.New()
//	pipeline.Add(
//	    pipeline.TrimSpace(),
//	    pipeline.RemoveHTML(),
//	    pipeline.ParseNumber(),
//	)
//	result, err := pipeline.Transform(rawData)
//
// Custom transformers can be created by implementing the Transformer interface:
//
//	type MyTransformer struct{}
//	
//	func (t *MyTransformer) Transform(data interface{}) (interface{}, error) {
//	    // Custom transformation logic
//	    return transformedData, nil
//	}
package pipeline

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/valpere/DataScrapexter/internal/utils"
)

// Transformer defines the interface for all data transformation operations.
// Implementations should be stateless and thread-safe for concurrent use.
type Transformer interface {
	// Transform applies a transformation to the input data and returns the result.
	// The input and output types depend on the specific transformer implementation.
	// Transformers should handle nil inputs gracefully and document their type expectations.
	Transform(data interface{}) (interface{}, error)
	
	// Name returns a human-readable name for the transformer, used in logging and debugging.
	Name() string
}

// Pipeline represents a series of transformation steps that process data sequentially.
// Each transformer in the pipeline receives the output of the previous transformer.
// Pipelines are thread-safe and can be reused across multiple goroutines.
type Pipeline struct {
	// transformers holds the ordered list of transformation steps
	transformers []Transformer
	
	// name identifies this pipeline instance for logging
	name string
	
	// stopOnError determines whether the pipeline should halt on the first error
	stopOnError bool
	
	// logger for debugging and error reporting
	logger utils.Logger
}

// PipelineOption configures Pipeline behavior.
type PipelineOption func(*Pipeline)

// WithName sets a name for the pipeline instance, useful for debugging and logging.
func WithName(name string) PipelineOption {
	return func(p *Pipeline) {
		p.name = name
	}
}

// WithStopOnError configures the pipeline to halt processing when any transformer
// returns an error. By default, pipelines continue processing and collect all errors.
func WithStopOnError(stop bool) PipelineOption {
	return func(p *Pipeline) {
		p.stopOnError = stop
	}
}

// WithLogger sets a custom logger for the pipeline. If not set, a default logger is used.
func WithLogger(logger utils.Logger) PipelineOption {
	return func(p *Pipeline) {
		p.logger = logger
	}
}

// New creates a new Pipeline instance with the specified options.
// The pipeline starts empty and transformers must be added using the Add method.
//
// Example:
//
//	pipeline := pipeline.New(
//	    pipeline.WithName("price-processor"),
//	    pipeline.WithStopOnError(true),
//	)
func New(opts ...PipelineOption) *Pipeline {
	p := &Pipeline{
		transformers: make([]Transformer, 0),
		name:         "default",
		stopOnError:  false,
	}
	
	for _, opt := range opts {
		opt(p)
	}
	
	if p.logger == nil {
		p.logger = utils.NewLogger()
	}
	
	return p
}

// Add appends one or more transformers to the pipeline.
// Transformers are executed in the order they are added.
// Returns the pipeline instance for method chaining.
//
// Example:
//
//	pipeline.Add(
//	    TrimSpace(),
//	    RemoveHTML(),
//	).Add(
//	    ParseNumber(),
//	    ValidateRange(0, 100),
//	)
func (p *Pipeline) Add(transformers ...Transformer) *Pipeline {
	p.transformers = append(p.transformers, transformers...)
	return p
}

// Transform executes all transformers in sequence on the input data.
// Each transformer receives the output of the previous transformer.
// If stopOnError is true, processing halts on the first error.
// Otherwise, all transformers are executed and a combined error is returned.
//
// Returns the final transformed data and any errors encountered.
func (p *Pipeline) Transform(data interface{}) (interface{}, error) {
	p.logger.Debugf("Pipeline '%s' starting with %d transformers", p.name, len(p.transformers))
	
	current := data
	var errors []error
	
	for i, transformer := range p.transformers {
		p.logger.Debugf("Executing transformer %d: %s", i+1, transformer.Name())
		
		result, err := transformer.Transform(current)
		if err != nil {
			p.logger.Errorf("Transformer '%s' failed: %v", transformer.Name(), err)
			errors = append(errors, fmt.Errorf("%s: %w", transformer.Name(), err))
			
			if p.stopOnError {
				return current, errors[0]
			}
			continue
		}
		
		current = result
	}
	
	if len(errors) > 0 {
		return current, fmt.Errorf("pipeline '%s' had %d errors: %v", p.name, len(errors), errors)
	}
	
	p.logger.Debugf("Pipeline '%s' completed successfully", p.name)
	return current, nil
}

// Len returns the number of transformers in the pipeline.
func (p *Pipeline) Len() int {
	return len(p.transformers)
}

// Clear removes all transformers from the pipeline.
func (p *Pipeline) Clear() {
	p.transformers = p.transformers[:0]
}

// Clone creates a deep copy of the pipeline with the same transformers and configuration.
// The cloned pipeline can be modified without affecting the original.
func (p *Pipeline) Clone() *Pipeline {
	clone := &Pipeline{
		transformers: make([]Transformer, len(p.transformers)),
		name:         p.name + "-clone",
		stopOnError:  p.stopOnError,
		logger:       p.logger,
	}
	copy(clone.transformers, p.transformers)
	return clone
}

// Built-in Transformers

// trimSpaceTransformer removes leading and trailing whitespace from strings.
type trimSpaceTransformer struct{}

func (t *trimSpaceTransformer) Name() string { return "TrimSpace" }

// Transform removes leading and trailing whitespace from string inputs.
// Non-string inputs are returned unchanged.
func (t *trimSpaceTransformer) Transform(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case string:
		return strings.TrimSpace(v), nil
	case *string:
		if v != nil {
			trimmed := strings.TrimSpace(*v)
			return &trimmed, nil
		}
		return v, nil
	default:
		return data, nil
	}
}

// TrimSpace creates a transformer that removes leading and trailing whitespace.
func TrimSpace() Transformer {
	return &trimSpaceTransformer{}
}

// removeHTMLTransformer strips HTML tags and decodes HTML entities.
type removeHTMLTransformer struct {
	// tagRegex matches HTML tags
	tagRegex *regexp.Regexp
}

func (t *removeHTMLTransformer) Name() string { return "RemoveHTML" }

// Transform removes HTML tags and decodes HTML entities from string inputs.
// The transformer preserves text content while removing all markup.
// Non-string inputs are returned unchanged.
func (t *removeHTMLTransformer) Transform(data interface{}) (interface{}, error) {
	str, ok := data.(string)
	if !ok {
		return data, nil
	}
	
	// Remove HTML tags
	cleaned := t.tagRegex.ReplaceAllString(str, " ")
	
	// Decode HTML entities
	cleaned = html.UnescapeString(cleaned)
	
	// Normalize whitespace
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	
	return strings.TrimSpace(cleaned), nil
}

// RemoveHTML creates a transformer that strips HTML tags and decodes entities.
func RemoveHTML() Transformer {
	return &removeHTMLTransformer{
		tagRegex: regexp.MustCompile(`<[^>]+>`),
	}
}

// parseNumberTransformer converts string representations of numbers to numeric types.
type parseNumberTransformer struct {
	// locale determines number format (e.g., decimal separator)
	locale string
	
	// allowCommas permits thousands separators
	allowCommas bool
	
	// returnFloat forces float64 output even for integers
	returnFloat bool
}

func (t *parseNumberTransformer) Name() string { return "ParseNumber" }

// Transform converts string inputs to numeric values.
// Handles various formats including:
// - Integer and decimal numbers
// - Numbers with thousands separators (1,234.56)
// - Negative numbers
// - Scientific notation (1.23e4)
// - Currency symbols (removes them)
//
// Returns int64 for integers or float64 for decimals unless returnFloat is true.
// Non-string inputs are checked if they're already numeric and returned as-is.
func (t *parseNumberTransformer) Transform(data interface{}) (interface{}, error) {
	// Check if already a number
	switch v := data.(type) {
	case int, int8, int16, int32, int64:
		if t.returnFloat {
			return toFloat64(v)
		}
		return v, nil
	case uint, uint8, uint16, uint32, uint64:
		if t.returnFloat {
			return toFloat64(v)
		}
		return v, nil
	case float32, float64:
		return v, nil
	}
	
	// Convert to string for parsing
	str, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("expected string or number, got %T", data)
	}
	
	// Clean the string
	cleaned := strings.TrimSpace(str)
	
	// Remove currency symbols and other non-numeric characters
	cleaned = regexp.MustCompile(`[¤$€£¥₹¢]`).ReplaceAllString(cleaned, "")
	
	// Handle thousands separators based on locale
	if t.allowCommas {
		if t.locale == "de" || t.locale == "fr" {
			// European format: 1.234,56
			cleaned = strings.ReplaceAll(cleaned, ".", "")
			cleaned = strings.ReplaceAll(cleaned, ",", ".")
		} else {
			// US/UK format: 1,234.56
			cleaned = strings.ReplaceAll(cleaned, ",", "")
		}
	}
	
	// Remove any remaining whitespace
	cleaned = strings.TrimSpace(cleaned)
	
	// Try parsing as integer first
	if !t.returnFloat && !strings.Contains(cleaned, ".") && !strings.Contains(cleaned, "e") && !strings.Contains(cleaned, "E") {
		if i, err := strconv.ParseInt(cleaned, 10, 64); err == nil {
			return i, nil
		}
	}
	
	// Parse as float
	f, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse number from '%s': %w", str, err)
	}
	
	return f, nil
}

// ParseNumber creates a transformer that converts strings to numbers.
// Uses default settings (US locale, allows commas, returns appropriate type).
func ParseNumber() Transformer {
	return &parseNumberTransformer{
		locale:      "en",
		allowCommas: true,
		returnFloat: false,
	}
}

// ParseNumberOptions configures the number parser behavior.
type ParseNumberOptions struct {
	Locale      string // Locale for number format (en, de, fr, etc.)
	AllowCommas bool   // Whether to handle thousands separators
	ForceFloat  bool   // Always return float64 even for integers
}

// ParseNumberWithOptions creates a customized number parser.
func ParseNumberWithOptions(opts ParseNumberOptions) Transformer {
	return &parseNumberTransformer{
		locale:      opts.Locale,
		allowCommas: opts.AllowCommas,
		returnFloat: opts.ForceFloat,
	}
}

// parseDateTransformer converts string representations to time.Time values.
type parseDateTransformer struct {
	// formats to try in order
	formats []string
	
	// timezone for parsing
	location *time.Location
	
	// whether to return UTC time
	toUTC bool
}

func (t *parseDateTransformer) Name() string { return "ParseDate" }

// Transform converts string inputs to time.Time values.
// Tries multiple date formats in sequence until one succeeds.
// Common formats are tried first for performance.
// Non-string inputs that are already time.Time are returned as-is.
func (t *parseDateTransformer) Transform(data interface{}) (interface{}, error) {
	// Check if already a time
	if timeVal, ok := data.(time.Time); ok {
		if t.toUTC {
			return timeVal.UTC(), nil
		}
		return timeVal, nil
	}
	
	str, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("expected string or time.Time, got %T", data)
	}
	
	str = strings.TrimSpace(str)
	if str == "" {
		return nil, fmt.Errorf("empty date string")
	}
	
	// Try each format
	for _, format := range t.formats {
		if parsed, err := time.ParseInLocation(format, str, t.location); err == nil {
			if t.toUTC {
				return parsed.UTC(), nil
			}
			return parsed, nil
		}
	}
	
	// Try relative date parsing (e.g., "2 days ago", "yesterday")
	if relative, err := parseRelativeDate(str); err == nil {
		if t.toUTC {
			return relative.UTC(), nil
		}
		return relative, nil
	}
	
	return nil, fmt.Errorf("failed to parse date from '%s'", str)
}

// Common date formats to try
var defaultDateFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"02/01/2006",
	"01/02/2006",
	"02-01-2006",
	"01-02-2006",
	"Jan 2, 2006",
	"January 2, 2006",
	"2 Jan 2006",
	"2 January 2006",
	"Mon, 02 Jan 2006 15:04:05 MST",
	"Mon, 02 Jan 2006 15:04:05 -0700",
}

// ParseDate creates a transformer that converts strings to time.Time values.
// Uses a comprehensive list of common date formats.
func ParseDate() Transformer {
	return &parseDateTransformer{
		formats:  defaultDateFormats,
		location: time.Local,
		toUTC:    false,
	}
}

// ParseDateOptions configures the date parser behavior.
type ParseDateOptions struct {
	Formats  []string      // Custom format strings to try
	Location *time.Location // Timezone for parsing
	ToUTC    bool          // Convert result to UTC
}

// ParseDateWithOptions creates a customized date parser.
func ParseDateWithOptions(opts ParseDateOptions) Transformer {
	formats := opts.Formats
	if len(formats) == 0 {
		formats = defaultDateFormats
	}
	
	loc := opts.Location
	if loc == nil {
		loc = time.Local
	}
	
	return &parseDateTransformer{
		formats:  formats,
		location: loc,
		toUTC:    opts.ToUTC,
	}
}

// regexExtractTransformer extracts data using regular expressions.
type regexExtractTransformer struct {
	pattern *regexp.Regexp
	group   int // Which capture group to return (0 for whole match)
}

func (t *regexExtractTransformer) Name() string { 
	return fmt.Sprintf("RegexExtract(%s)", t.pattern.String())
}

// Transform extracts text matching a regular expression from string inputs.
// If the regex has capture groups, returns the specified group (default: 1).
// Returns an error if no match is found.
func (t *regexExtractTransformer) Transform(data interface{}) (interface{}, error) {
	str, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", data)
	}
	
	matches := t.pattern.FindStringSubmatch(str)
	if matches == nil {
		return nil, fmt.Errorf("no match found for pattern %s", t.pattern)
	}
	
	if t.group >= len(matches) {
		return nil, fmt.Errorf("capture group %d not found (only %d groups)", t.group, len(matches)-1)
	}
	
	return matches[t.group], nil
}

// RegexExtract creates a transformer that extracts text using a regular expression.
// By default, returns the first capture group. Use group=0 to return the whole match.
func RegexExtract(pattern string, group ...int) (Transformer, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	
	g := 1 // Default to first capture group
	if len(group) > 0 {
		g = group[0]
	}
	
	return &regexExtractTransformer{
		pattern: re,
		group:   g,
	}, nil
}

// Helper functions

// toFloat64 converts numeric types to float64.
func toFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int8:
		return float64(n), nil
	case int16:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case uint:
		return float64(n), nil
	case uint8:
		return float64(n), nil
	case uint16:
		return float64(n), nil
	case uint32:
		return float64(n), nil
	case uint64:
		return float64(n), nil
	case float32:
		return float64(n), nil
	case float64:
		return n, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// parseRelativeDate handles relative date strings like "2 days ago", "yesterday", etc.
func parseRelativeDate(str string) (time.Time, error) {
	now := time.Now()
	lower := strings.ToLower(str)
	
	switch lower {
	case "today":
		return now, nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	case "tomorrow":
		return now.AddDate(0, 0, 1), nil
	}
	
	// Pattern: "X days/weeks/months/years ago"
	agoPattern := regexp.MustCompile(`(\d+)\s+(second|minute|hour|day|week|month|year)s?\s+ago`)
	if matches := agoPattern.FindStringSubmatch(lower); matches != nil {
		n, _ := strconv.Atoi(matches[1])
		unit := matches[2]
		
		switch unit {
		case "second":
			return now.Add(-time.Duration(n) * time.Second), nil
		case "minute":
			return now.Add(-time.Duration(n) * time.Minute), nil
		case "hour":
			return now.Add(-time.Duration(n) * time.Hour), nil
		case "day":
			return now.AddDate(0, 0, -n), nil
		case "week":
			return now.AddDate(0, 0, -n*7), nil
		case "month":
			return now.AddDate(0, -n, 0), nil
		case "year":
			return now.AddDate(-n, 0, 0), nil
		}
	}
	
	return time.Time{}, fmt.Errorf("cannot parse relative date: %s", str)
}
