// internal/pipeline/pipeline_types.go
package pipeline

import (
	"context"
	"fmt"
	"net/url"
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
		
	// Advanced transformations
	case "split":
		if tr.Pattern == "" {
			return input, nil
		}
		parts := strings.Split(input, tr.Pattern)
		if tr.Params != nil {
			if index, ok := tr.Params["index"].(int); ok && index < len(parts) {
				return parts[index], nil
			}
		}
		return strings.Join(parts, ","), nil
		
	case "substring":
		if tr.Params == nil {
			return input, nil
		}
		start, hasStart := tr.Params["start"].(int)
		end, hasEnd := tr.Params["end"].(int)
		if hasStart && start >= 0 && start < len(input) {
			if hasEnd && end > start && end <= len(input) {
				return input[start:end], nil
			}
			return input[start:], nil
		}
		return input, nil
		
	case "truncate":
		if tr.Params == nil {
			return input, nil
		}
		if maxLen, ok := tr.Params["length"].(int); ok && maxLen > 0 && len(input) > maxLen {
			suffix := "..."
			if s, ok := tr.Params["suffix"].(string); ok {
				suffix = s
			}
			if maxLen <= len(suffix) {
				return input[:maxLen], nil
			}
			return input[:maxLen-len(suffix)] + suffix, nil
		}
		return input, nil
		
	case "title_case":
		return strings.Title(strings.ToLower(input)), nil
		
	case "reverse":
		runes := []rune(input)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), nil
		
	case "remove_commas":
		return strings.ReplaceAll(input, ",", ""), nil
		
	case "format_currency":
		// Clean the number first
		cleaned := regexp.MustCompile(`[^\d.-]`).ReplaceAllString(input, "")
		if cleaned == "" {
			return input, nil
		}
		if value, err := strconv.ParseFloat(cleaned, 64); err == nil {
			currency := "$"
			if tr.Params != nil && tr.Params["symbol"] != nil {
				currency = fmt.Sprintf("%v", tr.Params["symbol"])
			}
			return fmt.Sprintf("%s%.2f", currency, value), nil
		}
		return input, nil
		
	case "extract_domain":
		if u, err := url.Parse(input); err == nil && u.Host != "" {
			return u.Host, nil
		}
		return input, nil
		
	case "extract_filename":
		if u, err := url.Parse(input); err == nil {
			parts := strings.Split(u.Path, "/")
			if len(parts) > 0 && parts[len(parts)-1] != "" {
				return parts[len(parts)-1], nil
			}
		}
		// Fallback to simple path extraction
		parts := strings.Split(input, "/")
		if len(parts) > 0 && parts[len(parts)-1] != "" {
			return parts[len(parts)-1], nil
		}
		return input, nil
		
	case "capitalize_words":
		words := strings.Fields(input)
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
			}
		}
		return strings.Join(words, " "), nil
		
	case "remove_duplicates":
		// For comma-separated values
		delimiter := ","
		if tr.Params != nil && tr.Params["delimiter"] != nil {
			delimiter = fmt.Sprintf("%v", tr.Params["delimiter"])
		}
		parts := strings.Split(input, delimiter)
		seen := make(map[string]bool)
		var unique []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" && !seen[trimmed] {
				seen[trimmed] = true
				unique = append(unique, trimmed)
			}
		}
		return strings.Join(unique, delimiter), nil
		
	case "pad_left":
		if tr.Params == nil {
			return input, nil
		}
		if length, ok := tr.Params["length"].(int); ok && length > len(input) {
			padChar := " "
			if char, ok := tr.Params["char"].(string); ok && char != "" {
				padChar = char
			}
			padding := strings.Repeat(padChar, length-len(input))
			return padding + input, nil
		}
		return input, nil
		
	case "pad_right":
		if tr.Params == nil {
			return input, nil
		}
		if length, ok := tr.Params["length"].(int); ok && length > len(input) {
			padChar := " "
			if char, ok := tr.Params["char"].(string); ok && char != "" {
				padChar = char
			}
			padding := strings.Repeat(padChar, length-len(input))
			return input + padding, nil
		}
		return input, nil

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
		// Advanced transformations
		"split": true, "substring": true, "truncate": true, "title_case": true,
		"reverse": true, "remove_commas": true, "format_currency": true,
		"extract_domain": true, "extract_filename": true, "capitalize_words": true,
		"remove_duplicates": true, "pad_left": true, "pad_right": true,
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
		case "split":
			if rule.Pattern == "" {
				return fmt.Errorf("rule %d: pattern is required for transform type %s", i, rule.Type)
			}
		case "substring", "truncate", "pad_left", "pad_right":
			if rule.Params == nil {
				return fmt.Errorf("rule %d: parameters are required for transform type %s", i, rule.Type)
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
