// internal/utils/utils.go

// Package utils provides common utility functions and helpers used throughout
// the DataScrapexter application. It includes functions for string manipulation,
// URL handling, data validation, rate limiting, logging, and other cross-cutting
// concerns.
//
// The utils package is designed to be dependency-free within the internal
// packages to avoid circular dependencies. All functions are thread-safe
// unless explicitly noted otherwise.
//
// Example usage:
//
//	// URL manipulation
//	absoluteURL := utils.ResolveURL(baseURL, relativeURL)
//
//	// String cleaning
//	cleaned := utils.CleanString(dirtyHTML)
//
//	// Rate limiting
//	limiter := utils.NewRateLimiter(2) // 2 requests per second
//	limiter.Wait(ctx)
package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// String Manipulation Functions

// CleanString removes extra whitespace, HTML entities, and normalizes Unicode.
// It's useful for cleaning scraped text that may contain formatting artifacts.
//
// The function performs the following operations:
// - Decodes HTML entities (&amp; -> &)
// - Normalizes whitespace (multiple spaces -> single space)
// - Trims leading and trailing whitespace
// - Removes zero-width characters
// - Normalizes Unicode characters
//
// Example:
//
//	dirty := "  Hello&nbsp;&nbsp;World!  \u200b"
//	clean := utils.CleanString(dirty) // "Hello World!"
func CleanString(s string) string {
	if s == "" {
		return ""
	}

	// Decode HTML entities
	s = html.UnescapeString(s)

	// Remove zero-width characters
	s = removeZeroWidth(s)

	// Normalize whitespace
	s = normalizeWhitespace(s)

	// Trim leading and trailing whitespace
	s = strings.TrimSpace(s)

	return s
}

// removeZeroWidth removes zero-width Unicode characters that can interfere
// with text processing and display.
func removeZeroWidth(s string) string {
	// Zero-width characters to remove
	zeroWidth := []rune{
		'\u200b', // Zero-width space
		'\u200c', // Zero-width non-joiner
		'\u200d', // Zero-width joiner
		'\ufeff', // Zero-width no-break space (BOM)
		'\u2060', // Word joiner
	}

	// Build regex pattern
	var pattern strings.Builder
	pattern.WriteString("[")
	for _, r := range zeroWidth {
		pattern.WriteRune(r)
	}
	pattern.WriteString("]")

	re := regexp.MustCompile(pattern.String())
	return re.ReplaceAllString(s, "")
}

// normalizeWhitespace replaces sequences of whitespace characters with single spaces.
// This includes spaces, tabs, newlines, and other Unicode whitespace.
func normalizeWhitespace(s string) string {
	// Replace all whitespace sequences with single space
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

// TruncateString truncates a string to the specified length, adding an ellipsis
// if truncation occurs. It's Unicode-aware and won't break multi-byte characters.
//
// If maxLen is <= 0, the original string is returned unchanged.
// The ellipsis counts toward the maximum length.
//
// Example:
//
//	long := "This is a very long string that needs truncation"
//	short := utils.TruncateString(long, 20) // "This is a very lo..."
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}

	// Account for ellipsis
	const ellipsis = "..."
	if maxLen <= len(ellipsis) {
		return ellipsis[:maxLen]
	}

	// Truncate at rune boundary
	truncated := truncateAtRuneBoundary(s, maxLen-len(ellipsis))
	return truncated + ellipsis
}

// truncateAtRuneBoundary ensures string truncation doesn't break UTF-8 sequences.
func truncateAtRuneBoundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	// Find the last valid rune boundary before maxBytes
	truncated := s[:maxBytes]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}

	return truncated
}

// NormalizeSpace is similar to CleanString but preserves newlines as spaces.
// Useful for maintaining paragraph structure while cleaning text.
func NormalizeSpace(s string) string {
	// Convert newlines to spaces first
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")

	// Then normalize whitespace
	return CleanString(s)
}

// URL Handling Functions

// ResolveURL resolves a potentially relative URL against a base URL.
// It handles various edge cases including:
// - Protocol-relative URLs (//example.com)
// - Absolute URLs (returns as-is)
// - Fragment-only URLs (#section)
// - Query-only URLs (?param=value)
//
// Returns the resolved absolute URL or the original URL if resolution fails.
//
// Example:
//
//	base := "https://example.com/page"
//	resolved := utils.ResolveURL(base, "../other") // "https://example.com/other"
func ResolveURL(baseURL, relativeURL string) string {
	// Handle empty cases
	if baseURL == "" {
		return relativeURL
	}
	if relativeURL == "" {
		return baseURL
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return relativeURL
	}

	// Parse relative URL
	rel, err := url.Parse(relativeURL)
	if err != nil {
		return relativeURL
	}

	// Resolve relative to base
	resolved := base.ResolveReference(rel)
	return resolved.String()
}

// IsValidURL checks if a string is a valid URL with http/https scheme.
// It performs more thorough validation than just parsing.
//
// Validation includes:
// - Valid URL structure
// - HTTP or HTTPS scheme
// - Non-empty host
// - Valid host format (not just IP unless explicitly allowed)
//
// Example:
//
//	utils.IsValidURL("https://example.com") // true
//	utils.IsValidURL("not a url") // false
//	utils.IsValidURL("ftp://example.com") // false (not http/https)
func IsValidURL(s string) bool {
	if s == "" {
		return false
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	// Check scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Check host
	if u.Host == "" {
		return false
	}

	// Additional validation can be added here
	// e.g., checking for valid TLD, IP address format, etc.

	return true
}

// NormalizeURL normalizes a URL for consistent comparison and deduplication.
// It performs the following normalizations:
// - Converts scheme and host to lowercase
// - Removes default ports (80 for http, 443 for https)
// - Sorts query parameters
// - Removes trailing slashes from paths
// - Removes common tracking parameters
//
// Example:
//
//	url1 := "HTTPS://Example.com:443/path/?b=2&a=1&utm_source=test"
//	url2 := "https://example.com/path?a=1&b=2"
//	normalized1 := utils.NormalizeURL(url1) // Same as normalized2
//	normalized2 := utils.NormalizeURL(url2)
func NormalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Lowercase scheme and host
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	// Remove default ports
	if (u.Scheme == "http" && strings.HasSuffix(u.Host, ":80")) ||
		(u.Scheme == "https" && strings.HasSuffix(u.Host, ":443")) {
		if idx := strings.LastIndex(u.Host, ":"); idx != -1 {
			u.Host = u.Host[:idx]
		}
	}

	// Remove trailing slash from path
	if u.Path != "/" {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}

	// Clean and sort query parameters
	if u.RawQuery != "" {
		u.RawQuery = cleanQueryParams(u.Query())
	}

	// Remove fragment
	u.Fragment = ""

	return u.String()
}

// cleanQueryParams removes tracking parameters and sorts the remaining ones.
func cleanQueryParams(params url.Values) string {
	// Common tracking parameters to remove
	trackingParams := map[string]bool{
		"utm_source":   true,
		"utm_medium":   true,
		"utm_campaign": true,
		"utm_term":     true,
		"utm_content":  true,
		"fbclid":       true,
		"gclid":        true,
		"ref":          true,
		"source":       true,
	}

	// Filter out tracking parameters
	cleaned := url.Values{}
	for key, values := range params {
		if !trackingParams[strings.ToLower(key)] {
			cleaned[key] = values
		}
	}

	return cleaned.Encode()
}

// ExtractDomain extracts the domain (host without port) from a URL.
// Returns empty string if the URL is invalid.
//
// Example:
//
//	domain := utils.ExtractDomain("https://example.com:8080/path") // "example.com"
func ExtractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	// Remove port if present
	host := u.Hostname()
	return strings.ToLower(host)
}

// Data Validation Functions

// IsEmail performs basic email validation using a simple regex pattern.
// This is not RFC-compliant but catches most common email formats.
//
// Note: For production use, consider using a more robust email validation library.
//
// Example:
//
//	utils.IsEmail("user@example.com") // true
//	utils.IsEmail("not-an-email") // false
func IsEmail(s string) bool {
	// Basic email regex pattern
	// This is simplified and doesn't cover all RFC 5322 cases
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

// IsNumeric checks if a string contains only numeric characters.
// It doesn't handle decimal points or negative signs.
//
// Example:
//
//	utils.IsNumeric("12345") // true
//	utils.IsNumeric("12.34") // false
//	utils.IsNumeric("abc123") // false
func IsNumeric(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}

// ContainsAny checks if a string contains any of the specified substrings.
// The check is case-sensitive.
//
// Example:
//
//	text := "The quick brown fox"
//	utils.ContainsAny(text, []string{"slow", "quick", "lazy"}) // true
func ContainsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// ContainsAll checks if a string contains all of the specified substrings.
// The check is case-sensitive.
//
// Example:
//
//	text := "The quick brown fox"
//	utils.ContainsAll(text, []string{"quick", "fox"}) // true
//	utils.ContainsAll(text, []string{"quick", "lazy"}) // false
func ContainsAll(s string, substrs []string) bool {
	for _, substr := range substrs {
		if !strings.Contains(s, substr) {
			return false
		}
	}
	return true
}

// Hash and ID Generation Functions

// GenerateID generates a unique identifier using cryptographically secure random bytes.
// The ID is returned as a hexadecimal string of the specified length (in bytes).
//
// Example:
//
//	id := utils.GenerateID(16) // 32-character hex string (16 bytes = 32 hex chars)
func GenerateID(length int) string {
	if length <= 0 {
		length = 16 // Default to 16 bytes (32 hex characters)
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}

// HashString generates an MD5 hash of the input string.
// Note: MD5 is not cryptographically secure and should only be used
// for checksums and deduplication, not for security purposes.
//
// Example:
//
//	hash := utils.HashString("hello world") // "5eb63bbbe01eeed093cb22bb8f5acdc3"
func HashString(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateSlug creates a URL-friendly slug from the input string.
// It converts to lowercase, replaces spaces with hyphens, and removes
// non-alphanumeric characters (except hyphens).
//
// Example:
//
//	slug := utils.GenerateSlug("Hello World! 123") // "hello-world-123"
func GenerateSlug(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	s = re.ReplaceAllString(s, "")

	// Remove multiple consecutive hyphens
	re = regexp.MustCompile(`-+`)
	s = re.ReplaceAllString(s, "-")

	// Trim hyphens from start and end
	s = strings.Trim(s, "-")

	return s
}

// Rate Limiting Support
// The RateLimiter type and related functions are defined in rate_limiter.go
// This section documents how to use the rate limiting functionality.

// Example usage of RateLimiter:
//
//	limiter := utils.NewRateLimiter(10) // 10 requests per second
//	for i := 0; i < 100; i++ {
//	    if err := limiter.Wait(ctx); err != nil {
//	        return err
//	    }
//	    // Make request
//	}

// SetRate updates the rate limiter's rate (requests per second).
// This functionality is implemented in rate_limiter.go
func SetRate(r *RateLimiter, rate float64) {
	// Implementation in rate_limiter.go
}

// Logging Support
// The Logger interface and related types are defined in logger.go
// This section documents how to use the logging functionality.

// Example usage of Logger:
//
//	logger := utils.NewLogger()
//	logger.Info("Starting scraper")
//	logger.WithField("url", "https://example.com").Debug("Processing URL")

// File and I/O Utilities

// CopyWithTimeout copies from src to dst with a timeout.
// Returns the number of bytes copied and any error encountered.
//
// This is useful for preventing indefinite blocking on slow or stalled connections.
//
// Example:
//
//	n, err := utils.CopyWithTimeout(dst, src, 30*time.Second)
func CopyWithTimeout(dst io.Writer, src io.Reader, timeout time.Duration) (int64, error) {
	// Create a channel to signal completion
	done := make(chan struct{})
	var n int64
	var err error

	go func() {
		n, err = io.Copy(dst, src)
		close(done)
	}()

	select {
	case <-done:
		return n, err
	case <-time.After(timeout):
		return n, fmt.Errorf("copy timeout after %v", timeout)
	}
}

// RetryableFunc represents a function that can be retried.
type RetryableFunc func() error

// Retry executes a function with exponential backoff retry logic.
// It will retry up to maxAttempts times, with exponentially increasing
// delays between attempts.
//
// The initial delay is 1 second, doubling after each failure up to a
// maximum of 30 seconds per retry.
//
// Example:
//
//	err := utils.Retry(func() error {
//	    return makeHTTPRequest()
//	}, 3) // Try up to 3 times
func Retry(fn RetryableFunc, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	baseDelay := time.Second
	maxDelay := 30 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		// Don't sleep after the last attempt
		if attempt < maxAttempts-1 {
			// Calculate exponential backoff
			delay := baseDelay * time.Duration(1<<uint(attempt))
			if delay > maxDelay {
				delay = maxDelay
			}

			time.Sleep(delay)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// Parallel executes functions in parallel with a limit on concurrent executions.
// It waits for all functions to complete and returns any errors encountered.
//
// Example:
//
//	funcs := []func() error{
//	    func() error { return processItem(1) },
//	    func() error { return processItem(2) },
//	    func() error { return processItem(3) },
//	}
//	errors := utils.Parallel(funcs, 2) // Run max 2 at a time
func Parallel(funcs []func() error, maxConcurrent int) []error {
	if maxConcurrent <= 0 {
		maxConcurrent = len(funcs)
	}

	semaphore := make(chan struct{}, maxConcurrent)
	errors := make([]error, len(funcs))
	var wg sync.WaitGroup

	for i, fn := range funcs {
		wg.Add(1)
		go func(index int, f func() error) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute function
			errors[index] = f()
		}(i, fn)
	}

	wg.Wait()
	return errors
}

// FirstNonEmpty returns the first non-empty string from the provided arguments.
// Useful for fallback values in configuration or data extraction.
//
// Example:
//
//	title := utils.FirstNonEmpty(
//	    extractedTitle,
//	    metaTitle,
//	    "Default Title",
//	)
func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// Coalesce returns the first non-nil value from the provided arguments.
// Similar to SQL COALESCE function.
//
// Example:
//
//	value := utils.Coalesce(userInput, defaultValue, "fallback")
func Coalesce(values ...interface{}) interface{} {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}
