// pkg/types/types.go
package types

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"time"
)

// ScraperStatus represents the current state of a scraper
type ScraperStatus string

const (
	StatusIdle      ScraperStatus = "idle"
	StatusRunning   ScraperStatus = "running"
	StatusPaused    ScraperStatus = "paused"
	StatusCompleted ScraperStatus = "completed"
	StatusFailed    ScraperStatus = "failed"
	StatusCancelled ScraperStatus = "cancelled"
)

// ValidStatuses returns all valid scraper status values
func ValidStatuses() []ScraperStatus {
	return []ScraperStatus{
		StatusIdle, StatusRunning, StatusPaused,
		StatusCompleted, StatusFailed, StatusCancelled,
	}
}

// IsValid checks if the status is a valid value
func (s ScraperStatus) IsValid() bool {
	for _, valid := range ValidStatuses() {
		if s == valid {
			return true
		}
	}
	return false
}

// JobPriority represents the execution priority of a scraping job
type JobPriority int

const (
	PriorityLow      JobPriority = 1
	PriorityNormal   JobPriority = 5
	PriorityHigh     JobPriority = 10
	PriorityCritical JobPriority = 20
)

// String returns the string representation of job priority
func (p JobPriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// IsValid checks if the priority is within valid range
func (p JobPriority) IsValid() bool {
	return p >= PriorityLow && p <= PriorityCritical
}

// FieldType represents the type of data to extract from a field
type FieldType string

const (
	FieldTypeText      FieldType = "text"
	FieldTypeAttribute FieldType = "attribute"
	FieldTypeHTML      FieldType = "html"
	FieldTypeNumber    FieldType = "number"
	FieldTypeFloat     FieldType = "float"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeDate      FieldType = "date"
	FieldTypeURL       FieldType = "url"
	FieldTypeEmail     FieldType = "email"
	FieldTypeImage     FieldType = "image"
	FieldTypeList      FieldType = "list"
)

// ValidFieldTypes returns all valid field type values
func ValidFieldTypes() []FieldType {
	return []FieldType{
		FieldTypeText, FieldTypeAttribute, FieldTypeHTML,
		FieldTypeNumber, FieldTypeFloat, FieldTypeBoolean,
		FieldTypeDate, FieldTypeURL, FieldTypeEmail,
		FieldTypeImage, FieldTypeList,
	}
}

// IsValid checks if the field type is valid
func (ft FieldType) IsValid() bool {
	for _, valid := range ValidFieldTypes() {
		if ft == valid {
			return true
		}
	}
	return false
}

// PaginationType represents different pagination strategies
type PaginationType string

const (
	PaginationOffset     PaginationType = "offset"
	PaginationCursor     PaginationType = "cursor"
	PaginationNextButton PaginationType = "next_button"
	PaginationNumbered   PaginationType = "numbered"
	PaginationInfinite   PaginationType = "infinite_scroll"
)

// ValidPaginationTypes returns all valid pagination type values
func ValidPaginationTypes() []PaginationType {
	return []PaginationType{
		PaginationOffset, PaginationCursor, PaginationNextButton,
		PaginationNumbered, PaginationInfinite,
	}
}

// IsValid checks if the pagination type is valid
func (pt PaginationType) IsValid() bool {
	for _, valid := range ValidPaginationTypes() {
		if pt == valid {
			return true
		}
	}
	return false
}

// OutputFormat represents supported output formats
type OutputFormat string

const (
	FormatJSON OutputFormat = "json"
	FormatCSV  OutputFormat = "csv"
	FormatXML  OutputFormat = "xml"
	FormatYAML OutputFormat = "yaml"
)

// ValidOutputFormats returns all valid output format values
func ValidOutputFormats() []OutputFormat {
	return []OutputFormat{FormatJSON, FormatCSV, FormatXML, FormatYAML}
}

// IsValid checks if the output format is valid
func (of OutputFormat) IsValid() bool {
	for _, valid := range ValidOutputFormats() {
		if of == valid {
			return true
		}
	}
	return false
}

// GetFileExtension returns the appropriate file extension for the format
func (of OutputFormat) GetFileExtension() string {
	switch of {
	case FormatJSON:
		return ".json"
	case FormatCSV:
		return ".csv"
	case FormatXML:
		return ".xml"
	case FormatYAML:
		return ".yaml"
	default:
		return ".txt"
	}
}

// TransformType represents different data transformation operations
type TransformType string

const (
	TransformTrim            TransformType = "trim"
	TransformLowercase       TransformType = "lowercase"
	TransformUppercase       TransformType = "uppercase"
	TransformNormalizeSpaces TransformType = "normalize_spaces"
	TransformRemoveHTML      TransformType = "remove_html"
	TransformRegex           TransformType = "regex"
	TransformParseFloat      TransformType = "parse_float"
	TransformParseInt        TransformType = "parse_int"
	TransformParseDate       TransformType = "parse_date"
	TransformExtractNumber   TransformType = "extract_number"
	TransformPrefix          TransformType = "prefix"
	TransformSuffix          TransformType = "suffix"
	TransformReplace         TransformType = "replace"
	TransformSplit           TransformType = "split"
	TransformJoin            TransformType = "join"
)

// ValidTransformTypes returns all valid transform type values
func ValidTransformTypes() []TransformType {
	return []TransformType{
		TransformTrim, TransformLowercase, TransformUppercase,
		TransformNormalizeSpaces, TransformRemoveHTML, TransformRegex,
		TransformParseFloat, TransformParseInt, TransformParseDate,
		TransformExtractNumber, TransformPrefix, TransformSuffix,
		TransformReplace, TransformSplit, TransformJoin,
	}
}

// IsValid checks if the transform type is valid
func (tt TransformType) IsValid() bool {
	for _, valid := range ValidTransformTypes() {
		if tt == valid {
			return true
		}
	}
	return false
}

// RequiresParameters returns true if the transform type requires parameters
func (tt TransformType) RequiresParameters() bool {
	switch tt {
	case TransformRegex, TransformPrefix, TransformSuffix,
		TransformReplace, TransformSplit, TransformJoin:
		return true
	default:
		return false
	}
}

// Duration represents a time duration with JSON marshaling support
type Duration time.Duration

// MarshalJSON implements json.Marshaler interface
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements json.Unmarshaler interface
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %s", s)
	}

	*d = Duration(duration)
	return nil
}

// String returns the string representation of the duration
func (d Duration) String() string {
	return time.Duration(d).String()
}

// ToDuration converts to standard time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// NewDuration creates a Duration from time.Duration
func NewDuration(td time.Duration) Duration {
	return Duration(td)
}

// URL represents a URL with validation and JSON marshaling support
type URL struct {
	*url.URL
}

// MarshalJSON implements json.Marshaler interface
func (u URL) MarshalJSON() ([]byte, error) {
	if u.URL == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(u.URL.String())
}

// UnmarshalJSON implements json.Unmarshaler interface
func (u *URL) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if s == "" {
		u.URL = nil
		return nil
	}

	parsed, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("invalid URL format: %s", s)
	}

	u.URL = parsed
	return nil
}

// String returns the string representation of the URL
func (u URL) String() string {
	if u.URL == nil {
		return ""
	}
	return u.URL.String()
}

// IsValid checks if the URL is valid and has required components
func (u URL) IsValid() bool {
	if u.URL == nil {
		return false
	}
	return u.URL.Scheme != "" && u.URL.Host != ""
}

// NewURL creates a new URL from string
func NewURL(s string) (*URL, error) {
	if s == "" {
		return &URL{}, nil
	}

	parsed, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %s", s)
	}

	return &URL{URL: parsed}, nil
}

// MustNewURL creates a new URL from string, panicking on error
func MustNewURL(s string) *URL {
	u, err := NewURL(s)
	if err != nil {
		panic(err)
	}
	return u
}

// Regex represents a compiled regular expression with JSON support
type Regex struct {
	*regexp.Regexp
	Pattern string `json:"pattern"`
}

// MarshalJSON implements json.Marshaler interface
func (r Regex) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Pattern)
}

// UnmarshalJSON implements json.Unmarshaler interface
func (r *Regex) UnmarshalJSON(data []byte) error {
	var pattern string
	if err := json.Unmarshal(data, &pattern); err != nil {
		return err
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %s", pattern)
	}

	r.Regexp = compiled
	r.Pattern = pattern
	return nil
}

// String returns the pattern string
func (r Regex) String() string {
	return r.Pattern
}

// IsValid checks if the regex is compiled and valid
func (r Regex) IsValid() bool {
	return r.Regexp != nil && r.Pattern != ""
}

// NewRegex creates a new Regex from pattern string
func NewRegex(pattern string) (*Regex, error) {
	if pattern == "" {
		return nil, fmt.Errorf("regex pattern cannot be empty")
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %s", pattern)
	}

	return &Regex{
		Regexp:  compiled,
		Pattern: pattern,
	}, nil
}

// MustNewRegex creates a new Regex from pattern string, panicking on error
func MustNewRegex(pattern string) *Regex {
	r, err := NewRegex(pattern)
	if err != nil {
		panic(err)
	}
	return r
}

// HTTPMethod represents HTTP request methods
type HTTPMethod string

const (
	MethodGET     HTTPMethod = "GET"
	MethodPOST    HTTPMethod = "POST"
	MethodPUT     HTTPMethod = "PUT"
	MethodDELETE  HTTPMethod = "DELETE"
	MethodHEAD    HTTPMethod = "HEAD"
	MethodOPTIONS HTTPMethod = "OPTIONS"
	MethodPATCH   HTTPMethod = "PATCH"
)

// ValidHTTPMethods returns all valid HTTP method values
func ValidHTTPMethods() []HTTPMethod {
	return []HTTPMethod{
		MethodGET, MethodPOST, MethodPUT, MethodDELETE,
		MethodHEAD, MethodOPTIONS, MethodPATCH,
	}
}

// IsValid checks if the HTTP method is valid
func (m HTTPMethod) IsValid() bool {
	for _, valid := range ValidHTTPMethods() {
		if m == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of the HTTP method
func (m HTTPMethod) String() string {
	return string(m)
}

// ProxyType represents different types of proxy protocols
type ProxyType string

const (
	ProxyHTTP   ProxyType = "http"
	ProxyHTTPS  ProxyType = "https"
	ProxySOCKS4 ProxyType = "socks4"
	ProxySOCKS5 ProxyType = "socks5"
)

// ValidProxyTypes returns all valid proxy type values
func ValidProxyTypes() []ProxyType {
	return []ProxyType{ProxyHTTP, ProxyHTTPS, ProxySOCKS4, ProxySOCKS5}
}

// IsValid checks if the proxy type is valid
func (pt ProxyType) IsValid() bool {
	for _, valid := range ValidProxyTypes() {
		if pt == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of the proxy type
func (pt ProxyType) String() string {
	return string(pt)
}

// LogLevel represents different logging levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// ValidLogLevels returns all valid log level values
func ValidLogLevels() []LogLevel {
	return []LogLevel{
		LogLevelDebug, LogLevelInfo, LogLevelWarn,
		LogLevelError, LogLevelFatal,
	}
}

// IsValid checks if the log level is valid
func (ll LogLevel) IsValid() bool {
	for _, valid := range ValidLogLevels() {
		if ll == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of the log level
func (ll LogLevel) String() string {
	return string(ll)
}

// GetNumericLevel returns the numeric level for comparison
func (ll LogLevel) GetNumericLevel() int {
	switch ll {
	case LogLevelDebug:
		return 0
	case LogLevelInfo:
		return 1
	case LogLevelWarn:
		return 2
	case LogLevelError:
		return 3
	case LogLevelFatal:
		return 4
	default:
		return -1
	}
}

// CompressionType represents different compression algorithms
type CompressionType string

const (
	CompressionNone CompressionType = "none"
	CompressionGzip CompressionType = "gzip"
	CompressionZlib CompressionType = "zlib"
)

// ValidCompressionTypes returns all valid compression type values
func ValidCompressionTypes() []CompressionType {
	return []CompressionType{CompressionNone, CompressionGzip, CompressionZlib}
}

// IsValid checks if the compression type is valid
func (ct CompressionType) IsValid() bool {
	for _, valid := range ValidCompressionTypes() {
		if ct == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of the compression type
func (ct CompressionType) String() string {
	return string(ct)
}

// UserAgentType represents different categories of user agents
type UserAgentType string

const (
	UserAgentChrome  UserAgentType = "chrome"
	UserAgentFirefox UserAgentType = "firefox"
	UserAgentSafari  UserAgentType = "safari"
	UserAgentEdge    UserAgentType = "edge"
	UserAgentMobile  UserAgentType = "mobile"
	UserAgentBot     UserAgentType = "bot"
)

// ValidUserAgentTypes returns all valid user agent type values
func ValidUserAgentTypes() []UserAgentType {
	return []UserAgentType{
		UserAgentChrome, UserAgentFirefox, UserAgentSafari,
		UserAgentEdge, UserAgentMobile, UserAgentBot,
	}
}

// IsValid checks if the user agent type is valid
func (uat UserAgentType) IsValid() bool {
	for _, valid := range ValidUserAgentTypes() {
		if uat == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of the user agent type
func (uat UserAgentType) String() string {
	return string(uat)
}

// ErrorCode represents different types of scraping errors
type ErrorCode string

const (
	ErrorCodeNetworkTimeout  ErrorCode = "network_timeout"
	ErrorCodeHTTPError       ErrorCode = "http_error"
	ErrorCodeParseError      ErrorCode = "parse_error"
	ErrorCodeValidationError ErrorCode = "validation_error"
	ErrorCodeRateLimitError  ErrorCode = "rate_limit_error"
	ErrorCodeAuthError       ErrorCode = "auth_error"
	ErrorCodeRobotsError     ErrorCode = "robots_error"
	ErrorCodeCaptchaError    ErrorCode = "captcha_error"
	ErrorCodeProxyError      ErrorCode = "proxy_error"
	ErrorCodeTransformError  ErrorCode = "transform_error"
	ErrorCodeOutputError     ErrorCode = "output_error"
	ErrorCodeConfigError     ErrorCode = "config_error"
	ErrorCodeInternalError   ErrorCode = "internal_error"
)

// ValidErrorCodes returns all valid error code values
func ValidErrorCodes() []ErrorCode {
	return []ErrorCode{
		ErrorCodeNetworkTimeout, ErrorCodeHTTPError, ErrorCodeParseError,
		ErrorCodeValidationError, ErrorCodeRateLimitError, ErrorCodeAuthError,
		ErrorCodeRobotsError, ErrorCodeCaptchaError, ErrorCodeProxyError,
		ErrorCodeTransformError, ErrorCodeOutputError, ErrorCodeConfigError,
		ErrorCodeInternalError,
	}
}

// IsValid checks if the error code is valid
func (ec ErrorCode) IsValid() bool {
	for _, valid := range ValidErrorCodes() {
		if ec == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of the error code
func (ec ErrorCode) String() string {
	return string(ec)
}

// IsRetryable returns true if the error type typically supports retry
func (ec ErrorCode) IsRetryable() bool {
	switch ec {
	case ErrorCodeNetworkTimeout, ErrorCodeHTTPError, ErrorCodeRateLimitError,
		ErrorCodeProxyError, ErrorCodeCaptchaError:
		return true
	default:
		return false
	}
}

// GetDescription returns a human-readable description of the error code
func (ec ErrorCode) GetDescription() string {
	switch ec {
	case ErrorCodeNetworkTimeout:
		return "Network request timed out"
	case ErrorCodeHTTPError:
		return "HTTP request failed with error status"
	case ErrorCodeParseError:
		return "Failed to parse HTML or extract data"
	case ErrorCodeValidationError:
		return "Data validation failed"
	case ErrorCodeRateLimitError:
		return "Rate limit exceeded"
	case ErrorCodeAuthError:
		return "Authentication failed"
	case ErrorCodeRobotsError:
		return "Robots.txt disallows access"
	case ErrorCodeCaptchaError:
		return "CAPTCHA challenge encountered"
	case ErrorCodeProxyError:
		return "Proxy connection failed"
	case ErrorCodeTransformError:
		return "Data transformation failed"
	case ErrorCodeOutputError:
		return "Output generation failed"
	case ErrorCodeConfigError:
		return "Configuration validation failed"
	case ErrorCodeInternalError:
		return "Internal system error"
	default:
		return "Unknown error"
	}
}
