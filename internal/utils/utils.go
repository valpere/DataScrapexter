package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// NormalizeURL normalizes a URL for consistent comparison
func NormalizeURL(rawURL string) (string, error) {
	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Convert to lowercase host
	u.Host = strings.ToLower(u.Host)

	// Remove default ports
	if (u.Scheme == "http" && strings.HasSuffix(u.Host, ":80")) ||
		(u.Scheme == "https" && strings.HasSuffix(u.Host, ":443")) {
		u.Host = strings.TrimSuffix(u.Host, ":80")
		u.Host = strings.TrimSuffix(u.Host, ":443")
	}

	// Sort query parameters for consistency
	if u.RawQuery != "" {
		values := u.Query()
		u.RawQuery = values.Encode()
	}

	// Remove trailing slash from path
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}

	// Remove fragment
	u.Fragment = ""

	return u.String(), nil
}

// HashURL creates a hash of a URL for deduplication
func HashURL(url string) string {
	h := sha256.New()
	h.Write([]byte(url))
	return hex.EncodeToString(h.Sum(nil))
}

// ExtractDomain extracts the domain from a URL
func ExtractDomain(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}

// IsValidURL checks if a string is a valid URL
func IsValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// CleanFileName removes invalid characters from a filename
func CleanFileName(name string) string {
	// Remove or replace invalid filename characters
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	cleaned := re.ReplaceAllString(name, "_")

	// Trim spaces and dots
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, ".")

	// Limit length
	if len(cleaned) > 200 {
		cleaned = cleaned[:200]
	}

	// Default if empty
	if cleaned == "" {
		cleaned = "output"
	}

	return cleaned
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(fn func() error, maxRetries int, baseDelay time.Duration) error {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			delay := baseDelay * time.Duration(1<<uint(i-1))
			time.Sleep(delay)
		}

		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// ExtractNumbers extracts all numbers from a string
func ExtractNumbers(s string) []string {
	re := regexp.MustCompile(`\d+\.?\d*`)
	return re.FindAllString(s, -1)
}

// StripHTMLTags removes HTML tags from a string
func StripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(s, "")
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// ParseContentType extracts the content type from a Content-Type header
func ParseContentType(contentType string) string {
	parts := strings.Split(contentType, ";")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return contentType
}

// IsTextContent checks if a content type represents text content
func IsTextContent(contentType string) bool {
	textTypes := []string{
		"text/html",
		"text/plain",
		"text/xml",
		"application/xml",
		"application/xhtml+xml",
		"application/json",
	}

	ct := ParseContentType(contentType)
	for _, textType := range textTypes {
		if ct == textType {
			return true
		}
	}

	return strings.HasPrefix(ct, "text/")
}

// GenerateOutputFileName generates a filename based on URL and timestamp
func GenerateOutputFileName(url string, format string) string {
	// Extract domain
	domain, err := ExtractDomain(url)
	if err != nil {
		domain = "output"
	}

	// Clean domain name
	domain = CleanFileName(domain)

	// Add timestamp
	timestamp := time.Now().Format("20060102_150405")

	// Generate filename
	return fmt.Sprintf("%s_%s.%s", domain, timestamp, format)
}

// MergeStringMaps merges multiple string maps
func MergeStringMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)

	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}

	return result
}

// SanitizeSelector cleans a CSS selector
func SanitizeSelector(selector string) string {
	// Remove excessive whitespace
	selector = strings.TrimSpace(selector)
	selector = regexp.MustCompile(`\s+`).ReplaceAllString(selector, " ")

	return selector
}

// DetectEncoding attempts to detect the encoding of HTML content
func DetectEncoding(contentType string, body []byte) string {
	// Check Content-Type header first
	if contentType != "" {
		parts := strings.Split(contentType, "charset=")
		if len(parts) > 1 {
			charset := strings.TrimSpace(parts[1])
			charset = strings.Split(charset, ";")[0]
			return charset
		}
	}

	// Check for BOM
	if len(body) >= 3 {
		if body[0] == 0xEF && body[1] == 0xBB && body[2] == 0xBF {
			return "utf-8"
		}
	}

	// Default to UTF-8
	return "utf-8"
}
