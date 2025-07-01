// internal/pipeline/pipeline_types.go
package pipeline

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

// Transform applies a single transformation rule
func (tr *TransformRule) Transform(ctx context.Context, input string) (string, error) {
	switch tr.Type {
	case "trim":
		return strings.TrimSpace(input), nil
	case "lowercase":
		return strings.ToLower(input), nil
	case "uppercase":
		return strings.ToUpper(input), nil
	case "normalize_spaces":
		re := regexp.MustCompile(`\s+`)
		return re.ReplaceAllString(strings.TrimSpace(input), " "), nil
	case "remove_html":
		re := regexp.MustCompile(`<[^>]*>`)
		return strings.TrimSpace(re.ReplaceAllString(input, "")), nil
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
		cleaned := strings.ReplaceAll(input, ",", "")
		cleaned = strings.ReplaceAll(cleaned, "$", "")
		cleaned = strings.TrimSpace(cleaned)
		if _, err := strconv.ParseFloat(cleaned, 64); err != nil {
			return "", fmt.Errorf("failed to parse float: %w", err)
		}
		return cleaned, nil
	case "parse_int":
		re := regexp.MustCompile(`[^0-9-]`)
		cleaned := re.ReplaceAllString(input, "")
		if cleaned == "" {
			return "0", nil
		}
		if _, err := strconv.ParseInt(cleaned, 10, 64); err != nil {
			return "", fmt.Errorf("failed to parse int: %w", err)
		}
		return cleaned, nil
	case "extract_numbers":
		re := regexp.MustCompile(`\d+(?:\.\d+)?`)
		match := re.FindString(input)
		if match == "" {
			return "0", nil
		}
		return match, nil
	case "prefix":
		if tr.Params != nil && tr.Params["value"] != nil {
			prefix := fmt.Sprintf("%v", tr.Params["value"])
			return prefix + input, nil
		}
		return input, nil
	case "suffix":
		if tr.Params != nil && tr.Params["value"] != nil {
			suffix := fmt.Sprintf("%v", tr.Params["value"])
			return input + suffix, nil
		}
		return input, nil
	case "replace":
		old := tr.Pattern
		new := tr.Replacement
		if old == "" {
			return input, nil
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

// ValidateTransformRules validates transformation rule configuration
func ValidateTransformRules(rules TransformList) error {
	validTypes := map[string]bool{
		"trim": true, "lowercase": true, "uppercase": true,
		"normalize_spaces": true, "remove_html": true, "regex": true,
		"parse_float": true, "parse_int": true, "extract_numbers": true,
		"prefix": true, "suffix": true, "replace": true,
	}

	for i, rule := range rules {
		if !validTypes[rule.Type] {
			return fmt.Errorf("rule %d: unknown transform type: %s", i, rule.Type)
		}

		switch rule.Type {
		case "regex", "replace":
			if rule.Pattern == "" {
				return fmt.Errorf("rule %d: pattern is required for transform type %s", i, rule.Type)
			}
		case "prefix", "suffix":
			if rule.Params == nil || rule.Params["value"] == nil {
				return fmt.Errorf("rule %d: 'value' parameter is required for transform type %s", i, rule.Type)
			}
		}

		if rule.Type == "regex" {
			if _, err := regexp.Compile(rule.Pattern); err != nil {
				return fmt.Errorf("rule %d: invalid regex pattern: %w", i, err)
			}
		}
	}
	return nil
}
