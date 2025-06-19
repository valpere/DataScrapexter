package pipeline

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// TransformRule defines a transformation to apply to extracted data
type TransformRule struct {
	Type        string `yaml:"type" json:"type"`                                   // regex, trim, lowercase, uppercase, parse_float, parse_int
	Pattern     string `yaml:"pattern,omitempty" json:"pattern,omitempty"`         // For regex
	Replacement string `yaml:"replacement,omitempty" json:"replacement,omitempty"` // For regex
}

// Transformer handles data transformation operations
type Transformer struct {
	rules []TransformRule
}

// NewTransformer creates a new transformer instance
func NewTransformer(rules []TransformRule) *Transformer {
	return &Transformer{
		rules: rules,
	}
}

// Transform applies all transformation rules to the input value
func (t *Transformer) Transform(value interface{}) (interface{}, error) {
	if t == nil || len(t.rules) == 0 {
		return value, nil
	}

	var result interface{} = value
	var err error

	for _, rule := range t.rules {
		result, err = t.applyRule(result, rule)
		if err != nil {
			return nil, fmt.Errorf("transformation rule '%s' failed: %w", rule.Type, err)
		}
	}

	return result, nil
}

// applyRule applies a single transformation rule
func (t *Transformer) applyRule(value interface{}, rule TransformRule) (interface{}, error) {
	switch rule.Type {
	case "trim":
		return t.trimTransform(value)
	case "lowercase":
		return t.lowercaseTransform(value)
	case "uppercase":
		return t.uppercaseTransform(value)
	case "regex":
		return t.regexTransform(value, rule.Pattern, rule.Replacement)
	case "parse_float":
		return t.parseFloatTransform(value)
	case "parse_int":
		return t.parseIntTransform(value)
	case "remove_spaces":
		return t.removeSpacesTransform(value)
	case "normalize_spaces":
		return t.normalizeSpacesTransform(value)
	case "remove_html":
		return t.removeHTMLTransform(value)
	case "extract_numbers":
		return t.extractNumbersTransform(value)
	case "clean_price":
		return t.cleanPriceTransform(value)
	default:
		return value, fmt.Errorf("unknown transformation type: %s", rule.Type)
	}
}

// trimTransform removes leading and trailing whitespace
func (t *Transformer) trimTransform(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}
	return strings.TrimSpace(str), nil
}

// lowercaseTransform converts string to lowercase
func (t *Transformer) lowercaseTransform(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}
	return strings.ToLower(str), nil
}

// uppercaseTransform converts string to uppercase
func (t *Transformer) uppercaseTransform(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}
	return strings.ToUpper(str), nil
}

// regexTransform applies regex pattern matching and replacement
func (t *Transformer) regexTransform(value interface{}, pattern, replacement string) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	if pattern == "" {
		return value, fmt.Errorf("regex pattern is required")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return value, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return re.ReplaceAllString(str, replacement), nil
}

// parseFloatTransform converts string to float64
func (t *Transformer) parseFloatTransform(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Remove common formatting characters
		cleaned := strings.ReplaceAll(v, ",", "")
		cleaned = strings.ReplaceAll(cleaned, "$", "")
		cleaned = strings.ReplaceAll(cleaned, "€", "")
		cleaned = strings.ReplaceAll(cleaned, "£", "")
		cleaned = strings.TrimSpace(cleaned)

		if cleaned == "" {
			return 0.0, nil
		}

		return strconv.ParseFloat(cleaned, 64)
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	default:
		return value, fmt.Errorf("cannot parse float from type %T", value)
	}
}

// parseIntTransform converts string to int
func (t *Transformer) parseIntTransform(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Remove common formatting characters
		cleaned := strings.ReplaceAll(v, ",", "")
		cleaned = strings.ReplaceAll(cleaned, ".", "")
		cleaned = strings.TrimSpace(cleaned)

		if cleaned == "" {
			return 0, nil
		}

		return strconv.Atoi(cleaned)
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return value, fmt.Errorf("cannot parse int from type %T", value)
	}
}

// removeSpacesTransform removes all spaces from string
func (t *Transformer) removeSpacesTransform(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}
	return strings.ReplaceAll(str, " ", ""), nil
}

// normalizeSpacesTransform replaces multiple spaces with single space
func (t *Transformer) normalizeSpacesTransform(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Replace multiple spaces with single space
	re := regexp.MustCompile(`\s+`)
	normalized := re.ReplaceAllString(str, " ")

	return strings.TrimSpace(normalized), nil
}

// removeHTMLTransform removes HTML tags from string
func (t *Transformer) removeHTMLTransform(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]+>`)
	cleaned := re.ReplaceAllString(str, " ")

	// Normalize spaces after removing tags
	return t.normalizeSpacesTransform(cleaned)
}

// extractNumbersTransform extracts all numbers from string
func (t *Transformer) extractNumbersTransform(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Extract all numeric sequences
	re := regexp.MustCompile(`\d+\.?\d*`)
	matches := re.FindAllString(str, -1)

	if len(matches) == 0 {
		return "", nil
	}

	return strings.Join(matches, ""), nil
}

// cleanPriceTransform extracts and cleans price values
func (t *Transformer) cleanPriceTransform(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Extract price pattern (handles various currency formats)
	re := regexp.MustCompile(`[£$€¥]?\s*(\d{1,3}(?:[,.\s]\d{3})*(?:[.,]\d{1,2})?)\s*[£$€¥]?`)
	match := re.FindStringSubmatch(str)

	if len(match) < 2 {
		// Try to find any number pattern
		return t.extractNumbersTransform(value)
	}

	// Clean the extracted price
	price := match[1]
	price = strings.ReplaceAll(price, ",", "")
	price = strings.ReplaceAll(price, " ", "")

	return price, nil
}

// TransformField applies transformations to a field value
func TransformField(value interface{}, rules []TransformRule) (interface{}, error) {
	if len(rules) == 0 {
		return value, nil
	}

	transformer := NewTransformer(rules)
	return transformer.Transform(value)
}

// TransformList applies transformations to a list of values
func TransformList(values []string, rules []TransformRule) ([]interface{}, error) {
	if len(rules) == 0 {
		// Convert to interface slice without transformation
		result := make([]interface{}, len(values))
		for i, v := range values {
			result[i] = v
		}
		return result, nil
	}

	transformer := NewTransformer(rules)
	result := make([]interface{}, len(values))

	for i, value := range values {
		transformed, err := transformer.Transform(value)
		if err != nil {
			return nil, fmt.Errorf("failed to transform item %d: %w", i, err)
		}
		result[i] = transformed
	}

	return result, nil
}

// CleanText performs basic text cleaning operations
func CleanText(text string) string {
	// Trim whitespace
	text = strings.TrimSpace(text)

	// Replace non-breaking spaces with regular spaces
	text = strings.ReplaceAll(text, "\u00A0", " ")

	// Remove zero-width spaces
	text = strings.ReplaceAll(text, "\u200B", "")
	text = strings.ReplaceAll(text, "\u200C", "")
	text = strings.ReplaceAll(text, "\u200D", "")
	text = strings.ReplaceAll(text, "\uFEFF", "")

	// Normalize whitespace
	text = strings.TrimFunc(text, unicode.IsSpace)

	return text
}
