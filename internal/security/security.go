// Package security provides comprehensive security utilities and safeguards
// for the DataScrapexter web scraping platform.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
	"github.com/valpere/DataScrapexter/internal/utils"
)

// securityWarningOnce ensures security warning is only logged once per application run
var securityWarningOnce sync.Once

// Security warning messages
const (
	obfuscatedStringWarning = "ObfuscatedString uses only XOR obfuscation and is NOT suitable for secrets. " +
		"Use proper secret management for production (AWS Secrets Manager, HashiCorp Vault, etc.). " +
		"This warning is shown only once per application run."
)

// Enhanced cryptographic constants for secure encryption
const (
	// AESKeySize is the key size for AES-256 encryption
	AESKeySize = 32 // 256 bits
	
	// AESNonceSize is the nonce size for AES-GCM
	AESNonceSize = 12 // 96 bits for GCM
	
	// SaltSize is the size of salt for key derivation
	SaltSize = 32 // 256 bits
	
	// Argon2Time is the number of iterations for Argon2
	Argon2Time = 3
	
	// Argon2Memory is the memory usage for Argon2 (in KiB)
	Argon2Memory = 64 * 1024 // 64 MB
	
	// Argon2Threads is the number of threads for Argon2
	Argon2Threads = 4
	
	// PBKDF2Iterations is the number of iterations for PBKDF2
	PBKDF2Iterations = 100000 // NIST recommended minimum
)

// SecurityLevel represents different security validation levels
type SecurityLevel int

const (
	SecurityLevelBasic SecurityLevel = iota
	SecurityLevelStandard
	SecurityLevelStrict
	SecurityLevelPCI // Payment Card Industry level
)

// EncryptionType represents different encryption methods
type EncryptionType int

const (
	// EncryptionXOR provides basic XOR obfuscation (legacy, not secure)
	EncryptionXOR EncryptionType = iota
	
	// EncryptionAES256GCM provides AES-256-GCM authenticated encryption
	EncryptionAES256GCM
)

// KeyDerivationType represents different key derivation methods
type KeyDerivationType int

const (
	// KeyDerivationPBKDF2 uses PBKDF2 key derivation
	KeyDerivationPBKDF2 KeyDerivationType = iota
	
	// KeyDerivationArgon2 uses Argon2id key derivation (recommended)
	KeyDerivationArgon2
)

// SecureString provides enhanced cryptographic protection for sensitive data
type SecureString struct {
	encryptedData []byte
	nonce         []byte
	salt          []byte
	encType       EncryptionType
	kdfType       KeyDerivationType
	hash          string
	keyCleared    bool
	dataCleared   bool
	options       *SecureStringOptions // Configuration options for behavior control
}

// SecureStringOptions provides configuration options for SecureString behavior
type SecureStringOptions struct {
	ForceGCOnClear bool // Whether to force garbage collection when clearing (may impact performance)
}

// DefaultSecureStringOptions returns default options with conservative settings
func DefaultSecureStringOptions() *SecureStringOptions {
	return &SecureStringOptions{
		ForceGCOnClear: false, // Disabled by default for better performance
	}
}

// NewSecureStringWithOptions creates a SecureString with custom options
func NewSecureStringWithOptions(data []byte, passphrase string, options *SecureStringOptions) (*SecureString, error) {
	ss, err := NewSecureString(data, passphrase)
	if err != nil {
		return nil, err
	}
	
	if options != nil {
		ss.options = options
	}
	
	return ss, nil
}

// SecurityValidator provides comprehensive security validation
type SecurityValidator struct {
	level                SecurityLevel
	allowedSchemes      []string
	blockedDomains      []string
	maxURLLength        int
	enableContentFilter bool
	customRules         []ValidationRule
}

// ValidationRule represents a custom security validation rule
type ValidationRule struct {
	Name        string
	Description string
	Validator   func(input string) (bool, string)
	Severity    Severity
}

// Severity levels for security issues
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// SecurityConfig configures the security validator
type SecurityConfig struct {
	Level                SecurityLevel `json:"level"`
	AllowedSchemes      []string      `json:"allowed_schemes"`
	BlockedDomains      []string      `json:"blocked_domains"`
	MaxURLLength        int           `json:"max_url_length"`
	EnableContentFilter bool          `json:"enable_content_filter"`
	EnableCSPValidation bool          `json:"enable_csp_validation"`
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Level:               SecurityLevelStandard,
		AllowedSchemes:     []string{"https", "http"},
		BlockedDomains:     []string{},
		MaxURLLength:       2048,
		EnableContentFilter: true,
		EnableCSPValidation: true,
	}
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(config *SecurityConfig) *SecurityValidator {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &SecurityValidator{
		level:                config.Level,
		allowedSchemes:      config.AllowedSchemes,
		blockedDomains:      config.BlockedDomains,
		maxURLLength:        config.MaxURLLength,
		enableContentFilter: config.EnableContentFilter,
		customRules:         make([]ValidationRule, 0),
	}
}

// ValidationResult represents the result of security validation
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Issues      []SecurityIssue   `json:"issues"`
	Warnings    []string          `json:"warnings"`
	Suggestions []string          `json:"suggestions"`
	RiskScore   int               `json:"risk_score"` // 0-100, higher is more risky
}

// SecurityIssue represents a security concern
type SecurityIssue struct {
	Type        string    `json:"type"`
	Severity    Severity  `json:"severity"`
	Message     string    `json:"message"`
	Field       string    `json:"field,omitempty"`
	Remediation string    `json:"remediation,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ValidateURL performs comprehensive URL security validation
func (sv *SecurityValidator) ValidateURL(inputURL string) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Issues:      make([]SecurityIssue, 0),
		Warnings:    make([]string, 0),
		Suggestions: make([]string, 0),
		RiskScore:   0,
	}

	// Check URL length
	if len(inputURL) > sv.maxURLLength {
		result.addIssue(SecurityIssue{
			Type:        "url_length_exceeded",
			Severity:    SeverityMedium,
			Message:     fmt.Sprintf("URL length %d exceeds maximum allowed %d", len(inputURL), sv.maxURLLength),
			Field:       "url",
			Remediation: "Use shorter URLs or increase max_url_length setting",
			Timestamp:   time.Now(),
		})
	}

	// Parse URL
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		result.addIssue(SecurityIssue{
			Type:        "invalid_url_format",
			Severity:    SeverityHigh,
			Message:     fmt.Sprintf("Invalid URL format: %v", err),
			Field:       "url",
			Remediation: "Ensure URL follows proper format (scheme://host/path)",
			Timestamp:   time.Now(),
		})
		return result
	}

	// Validate scheme
	if !sv.isSchemeAllowed(parsedURL.Scheme) {
		result.addIssue(SecurityIssue{
			Type:        "disallowed_scheme",
			Severity:    SeverityHigh,
			Message:     fmt.Sprintf("Scheme '%s' not in allowed list", parsedURL.Scheme),
			Field:       "url",
			Remediation: fmt.Sprintf("Use one of the allowed schemes: %s", strings.Join(sv.allowedSchemes, ", ")),
			Timestamp:   time.Now(),
		})
	}

	// Check for blocked domains
	if sv.isDomainBlocked(parsedURL.Host) {
		result.addIssue(SecurityIssue{
			Type:        "blocked_domain",
			Severity:    SeverityCritical,
			Message:     fmt.Sprintf("Domain '%s' is in blocked list", parsedURL.Host),
			Field:       "url",
			Remediation: "Remove domain from blocked list or use a different domain",
			Timestamp:   time.Now(),
		})
	}

	// Check for suspicious URL patterns
	sv.validateSuspiciousPatterns(inputURL, result)

	// Security recommendations
	if parsedURL.Scheme == "http" {
		result.Warnings = append(result.Warnings, "Using HTTP instead of HTTPS reduces security")
		result.Suggestions = append(result.Suggestions, "Consider using HTTPS when available")
		result.RiskScore += 10
	}

	// Check for common attack patterns
	sv.validateForAttackPatterns(inputURL, result)

	return result
}

// ValidateInput performs general input validation
func (sv *SecurityValidator) ValidateInput(input, fieldName string) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Issues:      make([]SecurityIssue, 0),
		Warnings:    make([]string, 0),
		Suggestions: make([]string, 0),
		RiskScore:   0,
	}

	// Check for SQL injection patterns
	if sv.containsSQLInjection(input) {
		result.addIssue(SecurityIssue{
			Type:        "sql_injection_risk",
			Severity:    SeverityHigh,
			Message:     "Input contains potential SQL injection patterns",
			Field:       fieldName,
			Remediation: "Sanitize input or use parameterized queries",
			Timestamp:   time.Now(),
		})
	}

	// Check for XSS patterns
	if sv.containsXSS(input) {
		result.addIssue(SecurityIssue{
			Type:        "xss_risk",
			Severity:    SeverityHigh,
			Message:     "Input contains potential XSS patterns",
			Field:       fieldName,
			Remediation: "Sanitize HTML content and encode output",
			Timestamp:   time.Now(),
		})
	}

	// Check for command injection
	if sv.containsCommandInjection(input) {
		result.addIssue(SecurityIssue{
			Type:        "command_injection_risk",
			Severity:    SeverityCritical,
			Message:     "Input contains potential command injection patterns",
			Field:       fieldName,
			Remediation: "Validate and sanitize input before any system operations",
			Timestamp:   time.Now(),
		})
	}

	// Check for path traversal
	if sv.containsPathTraversal(input) {
		result.addIssue(SecurityIssue{
			Type:        "path_traversal_risk",
			Severity:    SeverityHigh,
			Message:     "Input contains potential path traversal patterns",
			Field:       fieldName,
			Remediation: "Validate file paths and restrict access to safe directories",
			Timestamp:   time.Now(),
		})
	}

	// Run custom validation rules
	for _, rule := range sv.customRules {
		if valid, message := rule.Validator(input); !valid {
			result.addIssue(SecurityIssue{
				Type:        "custom_rule_violation",
				Severity:    rule.Severity,
				Message:     fmt.Sprintf("Custom rule '%s': %s", rule.Name, message),
				Field:       fieldName,
				Remediation: rule.Description,
				Timestamp:   time.Now(),
			})
		}
	}

	return result
}

// Helper methods

func (vr *ValidationResult) addIssue(issue SecurityIssue) {
	vr.Issues = append(vr.Issues, issue)
	vr.Valid = false
	
	// Update risk score based on severity
	switch issue.Severity {
	case SeverityInfo:
		vr.RiskScore += 1
	case SeverityLow:
		vr.RiskScore += 5
	case SeverityMedium:
		vr.RiskScore += 15
	case SeverityHigh:
		vr.RiskScore += 30
	case SeverityCritical:
		vr.RiskScore += 50
	}
}

func (sv *SecurityValidator) isSchemeAllowed(scheme string) bool {
	for _, allowed := range sv.allowedSchemes {
		if scheme == allowed {
			return true
		}
	}
	return false
}

func (sv *SecurityValidator) isDomainBlocked(domain string) bool {
	for _, blocked := range sv.blockedDomains {
		if domain == blocked || strings.HasSuffix(domain, "."+blocked) {
			return true
		}
	}
	return false
}

func (sv *SecurityValidator) validateSuspiciousPatterns(input string, result *ValidationResult) {
	suspiciousPatterns := []struct {
		pattern     *regexp.Regexp
		name        string
		severity    Severity
		remediation string
	}{
		{
			regexp.MustCompile(`(?i)(localhost|127\.0\.0\.1|0\.0\.0\.0|::1)`),
			"localhost_access",
			SeverityMedium,
			"Avoid localhost URLs in production configurations",
		},
		{
			regexp.MustCompile(`(?i)\.onion$`),
			"tor_hidden_service",
			SeverityHigh,
			"Review if Tor hidden services are intentionally required",
		},
		{
			regexp.MustCompile(`(?i)(admin|login|auth|secure|private|internal|management|config)`),
			"sensitive_path",
			SeverityMedium,
			"Be cautious when accessing administrative or sensitive paths",
		},
	}

	for _, pattern := range suspiciousPatterns {
		if pattern.pattern.MatchString(input) {
			result.addIssue(SecurityIssue{
				Type:        pattern.name,
				Severity:    pattern.severity,
				Message:     fmt.Sprintf("Detected suspicious pattern: %s", pattern.name),
				Field:       "url",
				Remediation: pattern.remediation,
				Timestamp:   time.Now(),
			})
		}
	}
}

func (sv *SecurityValidator) validateForAttackPatterns(input string, result *ValidationResult) {
	attackPatterns := []struct {
		pattern     *regexp.Regexp
		name        string
		severity    Severity
		remediation string
	}{
		{
			regexp.MustCompile(`(?i)javascript:`),
			"javascript_protocol",
			SeverityCritical,
			"JavaScript protocol can be used for XSS attacks",
		},
		{
			regexp.MustCompile(`(?i)data:`),
			"data_protocol",
			SeverityMedium,
			"Data URLs can contain malicious content",
		},
		{
			regexp.MustCompile(`(?i)(union|select|insert|delete|update|drop|exec|script)`),
			"sql_keywords",
			SeverityHigh,
			"Input contains SQL keywords that could indicate injection attempts",
		},
	}

	for _, pattern := range attackPatterns {
		if pattern.pattern.MatchString(input) {
			result.addIssue(SecurityIssue{
				Type:        pattern.name,
				Severity:    pattern.severity,
				Message:     fmt.Sprintf("Detected potential attack pattern: %s", pattern.name),
				Field:       "url",
				Remediation: pattern.remediation,
				Timestamp:   time.Now(),
			})
		}
	}
}

func (sv *SecurityValidator) containsSQLInjection(input string) bool {
	sqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)'.*(\sor\s|\sand\s).*'`),
		regexp.MustCompile(`(?i)union\s+select`),
		regexp.MustCompile(`(?i)(exec|execute)\s*\(`),
		regexp.MustCompile(`(?i)drop\s+table`),
		regexp.MustCompile(`(?i)1\s*=\s*1`),
		regexp.MustCompile(`(?i)'\s*or\s*'.*'`),
	}

	for _, pattern := range sqlPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

func (sv *SecurityValidator) containsXSS(input string) bool {
	xssPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*</script>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=\s*['"'][^'"]*['"']`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*</iframe>`),
		regexp.MustCompile(`(?i)alert\s*\(`),
		regexp.MustCompile(`(?i)document\.cookie`),
	}

	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

func (sv *SecurityValidator) containsCommandInjection(input string) bool {
	cmdPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i);.*\s*(rm|del|format|fdisk)`),
		regexp.MustCompile(`(?i)\|\s*(nc|netcat|wget|curl)`),
		regexp.MustCompile(`(?i)&&\s*(cat|type|more|less)`),
		regexp.MustCompile(`(?i)\$\([^)]+\)`),
		regexp.MustCompile("(?i)`[^`]+`"),
	}

	for _, pattern := range cmdPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

func (sv *SecurityValidator) containsPathTraversal(input string) bool {
	pathPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\.\.[\\/]`),
		regexp.MustCompile(`[\\/]\.\.[\\/]`),
		regexp.MustCompile(`%2e%2e`),
		regexp.MustCompile(`%2f%2e%2e%2f`),
		regexp.MustCompile(`(?i)(etc[\\/]passwd|windows[\\/]system32)`),
	}

	for _, pattern := range pathPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// AddCustomRule adds a custom validation rule
func (sv *SecurityValidator) AddCustomRule(rule ValidationRule) {
	sv.customRules = append(sv.customRules, rule)
}

// ObfuscatedString provides basic XOR obfuscation for in-memory data with memory clearing capabilities.
//
// ⚠️  SECURITY WARNING: This implementation uses BASIC XOR obfuscation and is NOT cryptographically secure!
//
// LIMITATIONS:
// - Does NOT protect against memory dumps or OS-level memory inspection
// - Does NOT prevent memory swapping to disk
// - Does NOT use OS-level memory protection (mlock)
// - Data may be visible in core dumps or swap files
// - Uses simple XOR obfuscation which can be easily reversed
// - The name "ObfuscatedString" reflects its limited security guarantees
//
// PRODUCTION USAGE: This type is NOT suitable for protecting secrets in production.
// For production, use dedicated secret management solutions (e.g., HashiCorp Vault, AWS Secrets Manager).
//
// TODO: Future Security Improvements
// - Replace XOR obfuscation with proper encryption (AES-256-GCM)
// - Implement key derivation functions (PBKDF2, Argon2, or scrypt)
// - Add secure random salt generation
// - Implement memory-locked pages where supported (mlock/VirtualLock)
// - Add integration with hardware security modules (HSM)
// - Implement automatic key rotation mechanisms
// - Add support for external key management systems
// - Consider using libsodium or similar cryptographic libraries
// - Add timing attack resistance for sensitive operations
// - Implement secure comparison functions using constant-time operations
type ObfuscatedString struct {
	data []byte
	key  []byte
	hash string
}

// NewObfuscatedString creates a new ObfuscatedString with basic XOR obfuscation.
// Returns an error if secure key generation fails.
//
// SECURITY WARNING: This provides basic XOR obfuscation only, not cryptographic security.
// Always call Clear() when done to zero out memory.
// 
// MIGRATION PATH: For production use, migrate to a proper secret management solution such as:
//   - HashiCorp Vault: https://www.vaultproject.io/
//   - AWS Secrets Manager: https://aws.amazon.com/secrets-manager/
//   - GCP Secret Manager: https://cloud.google.com/secret-manager
//   - Azure Key Vault: https://azure.microsoft.com/en-us/products/key-vault/
// Replace usage of ObfuscatedString with integration to one of these services for secure secret storage and retrieval.
func NewObfuscatedString(dataBytes []byte) (*ObfuscatedString, error) {
	// TODO: FUTURE IMPROVEMENTS - Replace XOR obfuscation with secure alternatives
	// Priority improvements for enhanced security:
	// 1. Implement AES-256-GCM encryption with proper key derivation (PBKDF2/Argon2)
	// 2. Add secure random salt generation and storage
	// 3. Implement memory protection (mlock) where available
	// 4. Add integration with hardware security modules (HSM)
	// 5. Implement secure key rotation mechanisms
	// 6. Add constant-time operations to prevent timing attacks
	// 7. Consider using established libraries like libsodium or Go's crypto/nacl
	// 8. Add secure memory wiping techniques beyond simple zeroing
	
	// Log security warning only once per application run to prevent log spam
	securityWarningOnce.Do(func() {
		logger := utils.GetLogger("security")
		logger.Security(obfuscatedStringWarning)
	})
	// Create a copy to avoid modifying the original slice
	dataCopy := make([]byte, len(dataBytes))
	copy(dataCopy, dataBytes)
	
	hash := sha256.Sum256(dataCopy)
	
	// Handle empty data case
	if len(dataCopy) == 0 {
		return &ObfuscatedString{
			data: dataCopy,
			key:  nil,
			hash: hex.EncodeToString(hash[:]),
		}, nil
	}
	
	// Generate a random key for XOR obfuscation
	key := make([]byte, len(dataCopy))
	if _, err := rand.Read(key); err != nil {
		// SECURITY: Fail securely rather than silently degrading security
		return nil, fmt.Errorf("failed to generate secure random key for obfuscation: %w", err)
	}
	
	// Apply XOR obfuscation
	obfuscated := make([]byte, len(dataCopy))
	for i := range dataCopy {
		obfuscated[i] = dataCopy[i] ^ key[i]
	}
	
	return &ObfuscatedString{
		data: obfuscated,
		key:  key,
		hash: hex.EncodeToString(hash[:]),
	}, nil
}

// NewObfuscatedStringFromString creates a new ObfuscatedString from a string.
// This is a convenience wrapper that converts string to []byte.
// For better security, prefer using NewObfuscatedString with []byte directly.
func NewObfuscatedStringFromString(data string) (*ObfuscatedString, error) {
	return NewObfuscatedString([]byte(data))
}

// String returns the deobfuscated string data
// SECURITY WARNING: This exposes the sensitive data. Use with caution.
func (os *ObfuscatedString) String() string {
	if os.key == nil || len(os.data) == 0 {
		return string(os.data)
	}
	
	// Deobfuscate by applying XOR with the key
	deobfuscated := make([]byte, len(os.data))
	for i := range os.data {
		deobfuscated[i] = os.data[i] ^ os.key[i]
	}
	
	return string(deobfuscated)
}

// Equals performs constant-time string comparison using SHA256 hash
func (os *ObfuscatedString) Equals(other *ObfuscatedString) bool {
	// Compare the SHA256 hashes in constant time to avoid exposing sensitive data
	return subtle.ConstantTimeCompare([]byte(os.hash), []byte(other.hash)) == 1
}

// Hash returns the SHA256 hash of the string
func (os *ObfuscatedString) Hash() string {
	return os.hash
}

// Clear securely clears the string data and key
func (os *ObfuscatedString) Clear() {
	// Clear the obfuscated data
	for i := range os.data {
		os.data[i] = 0
	}
	
	// Clear the XOR key
	if os.key != nil {
		for i := range os.key {
			os.key[i] = 0
		}
	}
}

// SecretManager handles sensitive configuration data
type SecretManager struct {
	secrets map[string]*ObfuscatedString
}

// NewSecretManager creates a new secret manager
func NewSecretManager() *SecretManager {
	return &SecretManager{
		secrets: make(map[string]*ObfuscatedString),
	}
}

// Store stores a secret securely
func (sm *SecretManager) Store(key, value string) error {
	if existing, exists := sm.secrets[key]; exists {
		existing.Clear() // Clear existing secret
	}
	
	obfuscatedString, err := NewObfuscatedStringFromString(value)
	if err != nil {
		return fmt.Errorf("failed to store secret '%s': %w", key, err)
	}
	
	sm.secrets[key] = obfuscatedString
	return nil
}

// Retrieve retrieves a secret
func (sm *SecretManager) Retrieve(key string) (string, bool) {
	if secret, exists := sm.secrets[key]; exists {
		return secret.String(), true
	}
	return "", false
}

// Clear clears all secrets
func (sm *SecretManager) Clear() {
	for _, secret := range sm.secrets {
		secret.Clear()
	}
	sm.secrets = make(map[string]*ObfuscatedString)
}

// SecurityAuditor performs security audits
type SecurityAuditor struct {
	validator *SecurityValidator
	logger    func(level string, message string)
}

// NewSecurityAuditor creates a new security auditor
func NewSecurityAuditor(validator *SecurityValidator) *SecurityAuditor {
	return &SecurityAuditor{
		validator: validator,
	}
}

// SetLogger sets a logger function
func (sa *SecurityAuditor) SetLogger(logger func(level string, message string)) {
	sa.logger = logger
}

// AuditConfiguration performs a security audit of configuration
func (sa *SecurityAuditor) AuditConfiguration(config map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Issues:      make([]SecurityIssue, 0),
		Warnings:    make([]string, 0),
		Suggestions: make([]string, 0),
		RiskScore:   0,
	}

	// Audit common configuration fields
	if baseURL, ok := config["base_url"].(string); ok {
		urlResult := sa.validator.ValidateURL(baseURL)
		result.mergeResult(urlResult)
	}

	// Check for hardcoded credentials
	sa.checkForHardcodedCredentials(config, result)

	// Check for insecure settings
	sa.checkForInsecureSettings(config, result)

	if sa.logger != nil {
		sa.logger("INFO", fmt.Sprintf("Security audit completed with risk score: %d", result.RiskScore))
	}

	return result
}

func (vr *ValidationResult) mergeResult(other *ValidationResult) {
	vr.Issues = append(vr.Issues, other.Issues...)
	vr.Warnings = append(vr.Warnings, other.Warnings...)
	vr.Suggestions = append(vr.Suggestions, other.Suggestions...)
	vr.RiskScore += other.RiskScore
	if !other.Valid {
		vr.Valid = false
	}
}

func (sa *SecurityAuditor) checkForHardcodedCredentials(config map[string]interface{}, result *ValidationResult) {
	credentialFields := []string{"password", "api_key", "secret", "token", "key"}
	
	for field, value := range config {
		fieldLower := strings.ToLower(field)
		for _, credField := range credentialFields {
			if strings.Contains(fieldLower, credField) {
				if strValue, ok := value.(string); ok && strValue != "" {
					result.addIssue(SecurityIssue{
						Type:        "hardcoded_credentials",
						Severity:    SeverityCritical,
						Message:     fmt.Sprintf("Potential hardcoded credential in field: %s", field),
						Field:       field,
						Remediation: "Use environment variables or secure secret management",
						Timestamp:   time.Now(),
					})
				}
			}
		}
	}
}

func (sa *SecurityAuditor) checkForInsecureSettings(config map[string]interface{}, result *ValidationResult) {
	// Check for insecure TLS settings
	if tlsConfig, ok := config["tls"].(map[string]interface{}); ok {
		if insecure, ok := tlsConfig["insecure_skip_verify"].(bool); ok && insecure {
			result.addIssue(SecurityIssue{
				Type:        "insecure_tls",
				Severity:    SeverityHigh,
				Message:     "TLS certificate verification is disabled",
				Field:       "tls.insecure_skip_verify",
				Remediation: "Enable TLS certificate verification for production use",
				Timestamp:   time.Now(),
			})
		}
	}

	// Check for debug mode in production
	if debug, ok := config["debug"].(bool); ok && debug {
		result.Warnings = append(result.Warnings, "Debug mode is enabled - ensure this is disabled in production")
		result.RiskScore += 5
	}
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("token length must be positive")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// SanitizeInput sanitizes input strings to prevent various attacks
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Remove control characters except common whitespace
	var sanitized strings.Builder
	for _, r := range input {
		if r >= 32 || r == '\t' || r == '\n' || r == '\r' {
			sanitized.WriteRune(r)
		}
	}
	
	return sanitized.String()
}

// IsSecureContext checks if the current context is secure for sensitive operations
func IsSecureContext(scheme string, host string) bool {
	// HTTPS is always secure
	if scheme == "https" {
		return true
	}
	
	// HTTP is only secure for localhost in development
	if scheme == "http" && (host == "localhost" || host == "127.0.0.1" || host == "::1") {
		return true
	}
	
	return false
}

// NewSecureString creates a new SecureString with AES-256-GCM encryption and Argon2 key derivation
// This addresses the TODOs for enhanced security by implementing:
// - AES-256-GCM authenticated encryption instead of XOR obfuscation
// - Argon2id key derivation function for secure key generation
// - Secure random salt generation
// - Constant-time operations for security-sensitive comparisons
func NewSecureString(data []byte, passphrase string) (*SecureString, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}
	
	// Generate cryptographically secure salt
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	
	// Generate cryptographically secure nonce
	nonce := make([]byte, AESNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Derive key using Argon2id (recommended for password hashing)
	key := argon2.IDKey([]byte(passphrase), salt, Argon2Time, Argon2Memory, Argon2Threads, AESKeySize)
	
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	
	// Create GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}
	
	// Encrypt the data with authentication
	encryptedData := gcm.Seal(nil, nonce, data, nil)
	
	// Create hash for comparison
	hash := sha256.Sum256(data)
	
	// Clear the derived key from memory
	for i := range key {
		key[i] = 0
	}
	
	return &SecureString{
		encryptedData: encryptedData,
		nonce:         nonce,
		salt:          salt,
		encType:       EncryptionAES256GCM,
		kdfType:       KeyDerivationArgon2,
		hash:          hex.EncodeToString(hash[:]),
		keyCleared:    true,
		dataCleared:   false,
		options:       DefaultSecureStringOptions(), // Use default options
	}, nil
}

// NewSecureStringPBKDF2 creates a SecureString using PBKDF2 key derivation (alternative implementation)
func NewSecureStringPBKDF2(data []byte, passphrase string) (*SecureString, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}
	
	// Generate cryptographically secure salt
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	
	// Generate cryptographically secure nonce
	nonce := make([]byte, AESNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(passphrase), salt, PBKDF2Iterations, AESKeySize, sha256.New)
	
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	
	// Create GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}
	
	// Encrypt the data with authentication
	encryptedData := gcm.Seal(nil, nonce, data, nil)
	
	// Create hash for comparison
	hash := sha256.Sum256(data)
	
	// Clear the derived key from memory
	for i := range key {
		key[i] = 0
	}
	
	return &SecureString{
		encryptedData: encryptedData,
		nonce:         nonce,
		salt:          salt,
		encType:       EncryptionAES256GCM,
		kdfType:       KeyDerivationPBKDF2,
		hash:          hex.EncodeToString(hash[:]),
		keyCleared:    true,
		dataCleared:   false,
		options:       DefaultSecureStringOptions(), // Use default options
	}, nil
}

// Decrypt decrypts and returns the original data
func (ss *SecureString) Decrypt(passphrase string) ([]byte, error) {
	if ss.dataCleared {
		return nil, fmt.Errorf("secure string has been cleared")
	}
	
	var key []byte
	
	// Derive key using the same method used for encryption
	switch ss.kdfType {
	case KeyDerivationArgon2:
		key = argon2.IDKey([]byte(passphrase), ss.salt, Argon2Time, Argon2Memory, Argon2Threads, AESKeySize)
	case KeyDerivationPBKDF2:
		key = pbkdf2.Key([]byte(passphrase), ss.salt, PBKDF2Iterations, AESKeySize, sha256.New)
	default:
		return nil, fmt.Errorf("unsupported key derivation type")
	}
	
	// Ensure key is cleared after use
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()
	
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}
	
	// Decrypt the data
	decryptedData, err := gcm.Open(nil, ss.nonce, ss.encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data (wrong passphrase or corrupted data): %w", err)
	}
	
	return decryptedData, nil
}

// DecryptToString decrypts and returns the data as a string
func (ss *SecureString) DecryptToString(passphrase string) (string, error) {
	data, err := ss.Decrypt(passphrase)
	if err != nil {
		return "", err
	}
	
	// Clear the decrypted data from memory after conversion
	defer func() {
		for i := range data {
			data[i] = 0
		}
	}()
	
	return string(data), nil
}

// Equals performs constant-time comparison using cryptographic hash
func (ss *SecureString) Equals(other *SecureString) bool {
	return subtle.ConstantTimeCompare([]byte(ss.hash), []byte(other.hash)) == 1
}

// Hash returns the SHA256 hash of the original data
func (ss *SecureString) Hash() string {
	return ss.hash
}

// Clear securely clears all sensitive data from memory
func (ss *SecureString) Clear() {
	// Clear encrypted data
	for i := range ss.encryptedData {
		ss.encryptedData[i] = 0
	}
	
	// Clear nonce
	for i := range ss.nonce {
		ss.nonce[i] = 0
	}
	
	// Clear salt
	for i := range ss.salt {
		ss.salt[i] = 0
	}
	
	ss.dataCleared = true
	
	// Force garbage collection only if configured to do so
	// Note: This may impact performance in high-throughput scenarios
	if ss.options != nil && ss.options.ForceGCOnClear {
		runtime.GC()
	}
}

// MemoryProtection provides memory protection utilities
type MemoryProtection struct {
	protectedPages []uintptr
	mutex          sync.RWMutex
}

// NewMemoryProtection creates a new memory protection manager
func NewMemoryProtection() *MemoryProtection {
	return &MemoryProtection{
		protectedPages: make([]uintptr, 0),
	}
}

// SecureZero securely zeros memory using multiple passes for enhanced security
func SecureZero(data []byte) {
	if len(data) == 0 {
		return
	}
	
	// Multiple pass secure erase
	patterns := []byte{0x00, 0xFF, 0xAA, 0x55, 0x00}
	
	for _, pattern := range patterns {
		for i := range data {
			data[i] = pattern
		}
		runtime.KeepAlive(data) // Prevent compiler optimizations
	}
	
	// Final zero pass
	for i := range data {
		data[i] = 0
	}
}

// TimingResistantCompare performs timing-resistant string comparison
func TimingResistantCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// SecureRandom generates cryptographically secure random bytes
func SecureRandom(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("size must be positive")
	}
	
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate secure random bytes: %w", err)
	}
	
	return bytes, nil
}