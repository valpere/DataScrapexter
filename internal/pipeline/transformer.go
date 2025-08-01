// internal/pipeline/transformer.go
package pipeline

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DataTransformer handles field-specific data transformations
type DataTransformer struct {
	Fields []TransformField `yaml:"fields" json:"fields"`
}

// NewDataTransformer creates a new data transformer
func NewDataTransformer(fields []TransformField) *DataTransformer {
	return &DataTransformer{Fields: fields}
}

// TransformData applies transformations to a data map
func (dt *DataTransformer) TransformData(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Copy original data first
	for k, v := range data {
		result[k] = v
	}

	// Apply field transformations
	for _, field := range dt.Fields {
		if value, exists := data[field.Name]; exists {
			transformedValue, err := ApplyFieldTransforms(ctx, field, value)
			if err != nil {
				if field.Required {
					return nil, fmt.Errorf("required field %s transformation failed: %w", field.Name, err)
				}

				// Use default value if available
				if field.DefaultVal != nil {
					result[field.Name] = field.DefaultVal
				}
				continue
			}
			result[field.Name] = transformedValue
		} else if field.Required {
			return nil, fmt.Errorf("required field %s not found in input data", field.Name)
		} else if field.DefaultVal != nil {
			result[field.Name] = field.DefaultVal
		}
	}

	return result, nil
}

// ValidateTransformFields validates transformation field configurations
func (dt *DataTransformer) ValidateTransformFields() error {
	for i, field := range dt.Fields {
		if field.Name == "" {
			return fmt.Errorf("field %d: name is required", i)
		}

		if err := ValidateTransformRules(field.Rules); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}
	return nil
}

// ApplyFieldTransforms applies transformations to a field value
func ApplyFieldTransforms(ctx context.Context, field TransformField, input interface{}) (interface{}, error) {
	inputStr := fmt.Sprintf("%v", input)

	if len(field.Rules) == 0 {
		return input, nil
	}

	result, err := field.Rules.Apply(ctx, inputStr)
	if err != nil {
		if field.Required {
			return nil, fmt.Errorf("required field %s transformation failed: %w", field.Name, err)
		}
		if field.DefaultVal != nil {
			return field.DefaultVal, nil
		}
		return input, nil
	}

	return result, nil
}

// Basic transformation functions

func transformTrim(value string) string {
	return strings.TrimSpace(value)
}

func transformLowercase(value string) string {
	return strings.ToLower(value)
}

func transformUppercase(value string) string {
	return strings.ToUpper(value)
}

func transformNormalizeSpaces(value string) string {
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(value, " "))
}

func transformRemoveHTML(value string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(value, ""))
}

func transformRegex(value, pattern, replacement string) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("regex pattern cannot be empty")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	return re.ReplaceAllString(value, replacement), nil
}

func transformPrefix(value string, params map[string]interface{}) (string, error) {
	prefixVal, ok := params["value"]
	if !ok {
		return "", fmt.Errorf("prefix requires 'value' parameter")
	}

	prefix, ok := prefixVal.(string)
	if !ok {
		return "", fmt.Errorf("prefix value must be string")
	}

	return prefix + value, nil
}

func transformSuffix(value string, params map[string]interface{}) (string, error) {
	suffixVal, ok := params["value"]
	if !ok {
		return "", fmt.Errorf("suffix requires 'value' parameter")
	}

	suffix, ok := suffixVal.(string)
	if !ok {
		return "", fmt.Errorf("suffix value must be string")
	}

	return value + suffix, nil
}

func transformReplace(value string, params map[string]interface{}) (string, error) {
	oldVal, ok := params["old"]
	if !ok {
		return "", fmt.Errorf("replace requires 'old' parameter")
	}

	newVal, ok := params["new"]
	if !ok {
		return "", fmt.Errorf("replace requires 'new' parameter")
	}

	old, ok := oldVal.(string)
	if !ok {
		return "", fmt.Errorf("replace 'old' value must be string")
	}

	new, ok := newVal.(string)
	if !ok {
		return "", fmt.Errorf("replace 'new' value must be string")
	}

	return strings.ReplaceAll(value, old, new), nil
}

func transformExtractNumber(value string) string {
	re := regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?`)
	match := re.FindString(value)
	if match == "" {
		return "0"
	}
	return match
}

func transformParseInt(value string) (string, error) {
	cleaned := strings.TrimSpace(value)
	re := regexp.MustCompile(`[^0-9-]`)
	cleaned = re.ReplaceAllString(cleaned, "")

	if cleaned == "" {
		return "0", nil
	}

	_, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil {
		return "", fmt.Errorf("cannot parse as integer: %w", err)
	}

	return cleaned, nil
}

func transformParseFloat(value string) (string, error) {
	cleaned := strings.TrimSpace(value)
	re := regexp.MustCompile(`[^0-9.-]`)
	cleaned = re.ReplaceAllString(cleaned, "")

	if cleaned == "" {
		return "0", nil
	}

	_, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return "", fmt.Errorf("cannot parse as float: %w", err)
	}

	return cleaned, nil
}
