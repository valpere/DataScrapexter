// internal/pipeline/transform.go
package pipeline

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TransformRule defines a single transformation rule
type TransformRule struct {
	Type        string                 `yaml:"type" json:"type"`
	Pattern     string                 `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Replacement string                 `yaml:"replacement,omitempty" json:"replacement,omitempty"`
	Format      string                 `yaml:"format,omitempty" json:"format,omitempty"`
	Params      map[string]interface{} `yaml:"params,omitempty" json:"params,omitempty"`
}

// TransformList represents a list of transformation rules
type TransformList []TransformRule

// TransformField defines field-specific transformations
type TransformField struct {
	Name       string        `yaml:"name" json:"name"`
	Rules      TransformList `yaml:"rules,omitempty" json:"rules,omitempty"`
	Required   bool          `yaml:"required,omitempty" json:"required,omitempty"`
	DefaultVal interface{}   `yaml:"default,omitempty" json:"default,omitempty"`
}

// DataTransformer handles data transformations
type DataTransformer struct {
	Fields []TransformField `yaml:"fields" json:"fields"`
	Global TransformList    `yaml:"global,omitempty" json:"global,omitempty"`
}

// Transform applies transformation rules to input data
func (tr *TransformRule) Transform(ctx context.Context, input string) (string, error) {
	switch tr.Type {
	case "trim":
		return strings.TrimSpace(input), nil
	
	case "normalize_spaces":
		re := regexp.MustCompile(`\s+`)
		return re.ReplaceAllString(strings.TrimSpace(input), " "), nil
	
	case "regex":
		if tr.Pattern == "" {
			return "", fmt.Errorf("regex pattern is required")
		}
		re, err := regexp.Compile(tr.Pattern)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern: %w", err)
		}
		return re.ReplaceAllString(input, tr.Replacement), nil
	
	case "parse_float":
		// Remove common currency symbols and commas
		cleaned := strings.ReplaceAll(input, ",", "")
		cleaned = strings.ReplaceAll(cleaned, "$", "")
		cleaned = strings.TrimSpace(cleaned)
		
		if _, err := strconv.ParseFloat(cleaned, 64); err != nil {
			return "", fmt.Errorf("failed to parse float: %w", err)
		}
		return cleaned, nil
	
	case "parse_int":
		// Remove common separators
		cleaned := strings.ReplaceAll(input, ",", "")
		cleaned = strings.TrimSpace(cleaned)
		
		if _, err := strconv.ParseInt(cleaned, 10, 64); err != nil {
			return "", fmt.Errorf("failed to parse int: %w", err)
		}
		return cleaned, nil
	
	case "parse_date":
		format := tr.Format
		if format == "" {
			format = "2006-01-02" // Default format
		}
		
		t, err := time.Parse(format, strings.TrimSpace(input))
		if err != nil {
			return "", fmt.Errorf("failed to parse date: %w", err)
		}
		return t.Format(time.RFC3339), nil
	
	case "lowercase":
		return strings.ToLower(input), nil
	
	case "uppercase":
		return strings.ToUpper(input), nil
	
	case "title":
		return strings.Title(input), nil
	
	case "remove_html":
		// Basic HTML tag removal
		re := regexp.MustCompile(`<[^>]*>`)
		return re.ReplaceAllString(input, ""), nil
	
	case "extract_number":
		re := regexp.MustCompile(`\d+(?:\.\d+)?`)
		matches := re.FindString(input)
		if matches == "" {
			return "", fmt.Errorf("no number found in input")
		}
		return matches, nil
	
	case "prefix":
		prefix, ok := tr.Params["value"].(string)
		if !ok {
			return "", fmt.Errorf("prefix value not specified")
		}
		return prefix + input, nil
	
	case "suffix":
		suffix, ok := tr.Params["value"].(string)
		if !ok {
			return "", fmt.Errorf("suffix value not specified")
		}
		return input + suffix, nil
	
	case "replace":
		old, ok := tr.Params["old"].(string)
		if !ok {
			return "", fmt.Errorf("old value not specified for replace")
		}
		new, ok := tr.Params["new"].(string)
		if !ok {
			return "", fmt.Errorf("new value not specified for replace")
		}
		return strings.ReplaceAll(input, old, new), nil
	
	default:
		return "", fmt.Errorf("unknown transform type: %s", tr.Type)
	}
}

// Apply applies all transformation rules in sequence
func (tl TransformList) Apply(ctx context.Context, input string) (string, error) {
	result := input
	for _, rule := range tl {
		var err error
		result, err = rule.Transform(ctx, result)
		if err != nil {
			return "", fmt.Errorf("transform failed at rule %s: %w", rule.Type, err)
		}
	}
	return result, nil
}

// TransformData applies transformations to a map of data
func (dt *DataTransformer) TransformData(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	// Apply global transformations first
	for key, value := range data {
		if str, ok := value.(string); ok {
			transformed, err := dt.Global.Apply(ctx, str)
			if err != nil {
				return nil, fmt.Errorf("global transform failed for field %s: %w", key, err)
			}
			result[key] = transformed
		} else {
			result[key] = value
		}
	}
	
	// Apply field-specific transformations
	for _, field := range dt.Fields {
		value, exists := result[field.Name]
		
		if !exists {
			if field.Required {
				return nil, fmt.Errorf("required field %s is missing", field.Name)
			}
			if field.DefaultVal != nil {
				result[field.Name] = field.DefaultVal
			}
			continue
		}
		
		if str, ok := value.(string); ok {
			transformed, err := field.Rules.Apply(ctx, str)
			if err != nil {
				return nil, fmt.Errorf("field transform failed for %s: %w", field.Name, err)
			}
			result[field.Name] = transformed
		}
	}
	
	return result, nil
}

// ValidateTransformRules validates transformation rule configuration
func ValidateTransformRules(rules TransformList) error {
	for i, rule := range rules {
		switch rule.Type {
		case "trim", "normalize_spaces", "lowercase", "uppercase", "title", "remove_html", "extract_number", "parse_float", "parse_int":
			// These transforms require no additional parameters
		case "regex":
			if rule.Pattern == "" {
				return fmt.Errorf("rule %d: regex pattern is required", i)
			}
		case "parse_date":
			if rule.Format != "" {
				_, err := time.Parse(rule.Format, rule.Format)
				if err != nil {
					return fmt.Errorf("rule %d: invalid date format: %w", i, err)
				}
			}
		case "prefix", "suffix":
			if rule.Params == nil || rule.Params["value"] == nil {
				return fmt.Errorf("rule %d: %s requires value parameter", i, rule.Type)
			}
		case "replace":
			if rule.Params == nil || rule.Params["old"] == nil || rule.Params["new"] == nil {
				return fmt.Errorf("rule %d: replace requires old and new parameters", i)
			}
		default:
			return fmt.Errorf("rule %d: unknown transform type: %s", i, rule.Type)
		}
	}
	return nil
}
