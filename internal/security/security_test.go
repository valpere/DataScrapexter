package security

import (
	"strings"
	"testing"
)

// TestSecurityValidator_ValidateURL tests URL validation functionality
func TestSecurityValidator_ValidateURL(t *testing.T) {
	validator := NewSecurityValidator(DefaultSecurityConfig())
	
	testCases := []struct {
		name        string
		url         string
		expectValid bool
		expectIssues int
	}{
		{
			name:         "Valid HTTPS URL",
			url:          "https://example.com/path",
			expectValid:  true,
			expectIssues: 0,
		},
		{
			name:         "HTTP URL (warning but valid)",
			url:          "http://example.com/path",
			expectValid:  true,
			expectIssues: 0, // HTTP generates warnings but no issues
		},
		{
			name:         "Invalid scheme",
			url:          "ftp://example.com/path",
			expectValid:  false,
			expectIssues: 1,
		},
		{
			name:         "JavaScript protocol (critical)",
			url:          "javascript:alert('xss')",
			expectValid:  false,
			expectIssues: 3, // disallowed_scheme + javascript_protocol + sql_keywords
		},
		{
			name:         "Too long URL",
			url:          "https://example.com/" + strings.Repeat("a", 3000),
			expectValid:  false,
			expectIssues: 1,
		},
		{
			name:         "Invalid URL format",
			url:          "not a url",
			expectValid:  false,
			expectIssues: 1,
		},
		{
			name:         "Localhost (suspicious pattern)",
			url:          "https://localhost/admin",
			expectValid:  false,
			expectIssues: 2, // localhost + admin path
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.ValidateURL(tc.url)
			
			if result.Valid != tc.expectValid {
				t.Errorf("Expected Valid=%v, got %v", tc.expectValid, result.Valid)
			}
			
			if len(result.Issues) != tc.expectIssues {
				t.Errorf("Expected %d issues, got %d: %v", tc.expectIssues, len(result.Issues), result.Issues)
			}
		})
	}
}

// TestSecurityValidator_ValidateInput tests input validation
func TestSecurityValidator_ValidateInput(t *testing.T) {
	validator := NewSecurityValidator(DefaultSecurityConfig())
	
	testCases := []struct {
		name        string
		input       string
		fieldName   string
		expectValid bool
		expectIssues int
	}{
		{
			name:         "Normal input",
			input:        "hello world",
			fieldName:    "test",
			expectValid:  true,
			expectIssues: 0,
		},
		{
			name:         "SQL injection attempt",
			input:        "'; DROP TABLE users; --",
			fieldName:    "test",
			expectValid:  false,
			expectIssues: 1,
		},
		{
			name:         "XSS attempt",
			input:        "<script>alert('xss')</script>",
			fieldName:    "test",
			expectValid:  false,
			expectIssues: 1,
		},
		{
			name:         "Command injection",
			input:        "; rm -rf /",
			fieldName:    "test",
			expectValid:  false,
			expectIssues: 1,
		},
		{
			name:         "Path traversal",
			input:        "../../../etc/passwd",
			fieldName:    "test",
			expectValid:  false,
			expectIssues: 1,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.ValidateInput(tc.input, tc.fieldName)
			
			if result.Valid != tc.expectValid {
				t.Errorf("Expected Valid=%v, got %v", tc.expectValid, result.Valid)
			}
			
			if len(result.Issues) != tc.expectIssues {
				t.Errorf("Expected %d issues, got %d: %v", tc.expectIssues, len(result.Issues), result.Issues)
			}
		})
	}
}

// TestObfuscatedString tests the legacy ObfuscatedString functionality
func TestObfuscatedString(t *testing.T) {
	testData := "secret password"
	
	// Test creation
	obfuscated, err := NewObfuscatedStringFromString(testData)
	if err != nil {
		t.Fatalf("Failed to create ObfuscatedString: %v", err)
	}
	
	// Test retrieval
	retrieved := obfuscated.String()
	if retrieved != testData {
		t.Errorf("Expected %q, got %q", testData, retrieved)
	}
	
	// Test hash comparison
	obfuscated2, err := NewObfuscatedStringFromString(testData)
	if err != nil {
		t.Fatalf("Failed to create second ObfuscatedString: %v", err)
	}
	
	if !obfuscated.Equals(obfuscated2) {
		t.Error("Expected equal ObfuscatedStrings to be equal")
	}
	
	// Test clearing
	obfuscated.Clear()
	
	// Test empty data
	empty, err := NewObfuscatedStringFromString("")
	if err != nil {
		t.Fatalf("Failed to create empty ObfuscatedString: %v", err)
	}
	
	if empty.String() != "" {
		t.Error("Expected empty string")
	}
}

// TestSecureString tests the new enhanced SecureString functionality
func TestSecureString(t *testing.T) {
	testData := []byte("super secret password")
	passphrase := "my-passphrase"
	
	// Test Argon2 creation
	secure, err := NewSecureString(testData, passphrase)
	if err != nil {
		t.Fatalf("Failed to create SecureString: %v", err)
	}
	
	// Test decryption
	decrypted, err := secure.Decrypt(passphrase)
	if err != nil {
		t.Fatalf("Failed to decrypt SecureString: %v", err)
	}
	
	if string(decrypted) != string(testData) {
		t.Errorf("Expected %q, got %q", string(testData), string(decrypted))
	}
	
	// Test string decryption
	decryptedStr, err := secure.DecryptToString(passphrase)
	if err != nil {
		t.Fatalf("Failed to decrypt to string: %v", err)
	}
	
	if decryptedStr != string(testData) {
		t.Errorf("Expected %q, got %q", string(testData), decryptedStr)
	}
	
	// Test wrong passphrase
	_, err = secure.Decrypt("wrong-passphrase")
	if err == nil {
		t.Error("Expected error with wrong passphrase")
	}
	
	// Test hash comparison
	secure2, err := NewSecureString(testData, passphrase)
	if err != nil {
		t.Fatalf("Failed to create second SecureString: %v", err)
	}
	
	if !secure.Equals(secure2) {
		t.Error("Expected equal SecureStrings to be equal")
	}
	
	// Test clearing
	secure.Clear()
	
	// Test decryption after clearing
	_, err = secure.Decrypt(passphrase)
	if err == nil {
		t.Error("Expected error when decrypting cleared SecureString")
	}
}

// TestSecureStringPBKDF2 tests PBKDF2 key derivation
func TestSecureStringPBKDF2(t *testing.T) {
	testData := []byte("secret data")
	passphrase := "test-passphrase"
	
	secure, err := NewSecureStringPBKDF2(testData, passphrase)
	if err != nil {
		t.Fatalf("Failed to create PBKDF2 SecureString: %v", err)
	}
	
	decrypted, err := secure.Decrypt(passphrase)
	if err != nil {
		t.Fatalf("Failed to decrypt PBKDF2 SecureString: %v", err)
	}
	
	if string(decrypted) != string(testData) {
		t.Errorf("Expected %q, got %q", string(testData), string(decrypted))
	}
	
	// Test that KDF type is correct
	if secure.kdfType != KeyDerivationPBKDF2 {
		t.Errorf("Expected PBKDF2 key derivation, got %v", secure.kdfType)
	}
}

// TestSecretManager tests the secret manager functionality
func TestSecretManager(t *testing.T) {
	sm := NewSecretManager()
	
	// Test storing and retrieving
	key := "test-key"
	value := "secret-value"
	
	err := sm.Store(key, value)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}
	
	retrieved, exists := sm.Retrieve(key)
	if !exists {
		t.Fatal("Expected secret to exist")
	}
	
	if retrieved != value {
		t.Errorf("Expected %q, got %q", value, retrieved)
	}
	
	// Test non-existent key
	_, exists = sm.Retrieve("non-existent")
	if exists {
		t.Error("Expected non-existent key to not exist")
	}
	
	// Test overwriting
	newValue := "new-secret-value"
	err = sm.Store(key, newValue)
	if err != nil {
		t.Fatalf("Failed to overwrite secret: %v", err)
	}
	
	retrieved, exists = sm.Retrieve(key)
	if !exists || retrieved != newValue {
		t.Errorf("Expected %q, got %q", newValue, retrieved)
	}
	
	// Test clearing
	sm.Clear()
	_, exists = sm.Retrieve(key)
	if exists {
		t.Error("Expected secret to be cleared")
	}
}

// TestSecurityAuditor tests security auditing functionality
func TestSecurityAuditor(t *testing.T) {
	validator := NewSecurityValidator(DefaultSecurityConfig())
	auditor := NewSecurityAuditor(validator)
	
	// Test configuration with issues
	config := map[string]interface{}{
		"base_url":  "http://example.com",
		"password":  "hardcoded-secret",
		"debug":     true,
		"tls": map[string]interface{}{
			"insecure_skip_verify": true,
		},
	}
	
	result := auditor.AuditConfiguration(config)
	
	if result.Valid {
		t.Error("Expected configuration to be invalid")
	}
	
	if len(result.Issues) == 0 {
		t.Error("Expected security issues to be found")
	}
	
	if result.RiskScore == 0 {
		t.Error("Expected non-zero risk score")
	}
	
	// Test secure configuration
	secureConfig := map[string]interface{}{
		"base_url": "https://secure.example.com/api",
		"debug":    false,
	}
	
	secureResult := auditor.AuditConfiguration(secureConfig)
	
	// Check if any issues are critical/high severity that would make it invalid
	hasCriticalIssues := false
	for _, issue := range secureResult.Issues {
		if issue.Severity >= 3 { // Medium severity and above
			hasCriticalIssues = true
			break
		}
	}
	
	if hasCriticalIssues {
		t.Errorf("Expected secure configuration to not have critical issues, got: %v", secureResult.Issues)
	}
}

// TestGenerateSecureToken tests secure token generation
func TestGenerateSecureToken(t *testing.T) {
	// Test normal token generation
	token, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	
	if len(token) == 0 {
		t.Error("Expected non-empty token")
	}
	
	// Test multiple tokens are different
	token2, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("Failed to generate second token: %v", err)
	}
	
	if token == token2 {
		t.Error("Expected different tokens")
	}
	
	// Test invalid length
	_, err = GenerateSecureToken(0)
	if err == nil {
		t.Error("Expected error for zero length")
	}
	
	_, err = GenerateSecureToken(-1)
	if err == nil {
		t.Error("Expected error for negative length")
	}
}

// TestSanitizeInput tests input sanitization
func TestSanitizeInput(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal input",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "Remove null bytes",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "Remove control characters",
			input:    "hello\x01\x02world",
			expected: "helloworld",
		},
		{
			name:     "Keep whitespace",
			input:    "hello\t\n\rworld",
			expected: "hello\t\n\rworld",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeInput(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestIsSecureContext tests secure context checking
func TestIsSecureContext(t *testing.T) {
	testCases := []struct {
		scheme   string
		host     string
		expected bool
	}{
		{"https", "example.com", true},
		{"http", "example.com", false},
		{"http", "localhost", true},
		{"http", "127.0.0.1", true},
		{"http", "::1", true},
		{"ftp", "localhost", false},
	}
	
	for _, tc := range testCases {
		result := IsSecureContext(tc.scheme, tc.host)
		if result != tc.expected {
			t.Errorf("IsSecureContext(%q, %q) = %v, expected %v", tc.scheme, tc.host, result, tc.expected)
		}
	}
}

// TestSecureZero tests secure memory clearing
func TestSecureZero(t *testing.T) {
	data := []byte("sensitive data")
	original := make([]byte, len(data))
	copy(original, data)
	
	SecureZero(data)
	
	// Verify all bytes are zero
	for i, b := range data {
		if b != 0 {
			t.Errorf("Expected zero at index %d, got %d", i, b)
		}
	}
	
	// Test empty slice
	SecureZero(nil)
	SecureZero([]byte{})
}

// TestTimingResistantCompare tests timing-resistant comparison
func TestTimingResistantCompare(t *testing.T) {
	// Test equal strings
	if !TimingResistantCompare("secret", "secret") {
		t.Error("Expected equal strings to be equal")
	}
	
	// Test different strings
	if TimingResistantCompare("secret", "different") {
		t.Error("Expected different strings to be different")
	}
	
	// Test empty strings
	if !TimingResistantCompare("", "") {
		t.Error("Expected empty strings to be equal")
	}
}

// TestSecureRandom tests secure random byte generation
func TestSecureRandom(t *testing.T) {
	// Test normal generation
	bytes, err := SecureRandom(32)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}
	
	if len(bytes) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(bytes))
	}
	
	// Test multiple generations are different
	bytes2, err := SecureRandom(32)
	if err != nil {
		t.Fatalf("Failed to generate second random bytes: %v", err)
	}
	
	// Very unlikely to be equal
	equal := true
	for i := range bytes {
		if bytes[i] != bytes2[i] {
			equal = false
			break
		}
	}
	
	if equal {
		t.Error("Expected different random byte sequences")
	}
	
	// Test invalid size
	_, err = SecureRandom(0)
	if err == nil {
		t.Error("Expected error for zero size")
	}
	
	_, err = SecureRandom(-1)
	if err == nil {
		t.Error("Expected error for negative size")
	}
}

// TestMemoryProtection tests memory protection utilities
func TestMemoryProtection(t *testing.T) {
	mp := NewMemoryProtection()
	if mp == nil {
		t.Fatal("Expected non-nil MemoryProtection")
	}
	
	if len(mp.protectedPages) != 0 {
		t.Error("Expected empty protected pages initially")
	}
}

// TestCustomValidationRules tests custom validation rule functionality
func TestCustomValidationRules(t *testing.T) {
	validator := NewSecurityValidator(DefaultSecurityConfig())
	
	// Add custom rule
	customRule := ValidationRule{
		Name:        "no_test_words",
		Description: "Input should not contain test words",
		Validator: func(input string) (bool, string) {
			if strings.Contains(strings.ToLower(input), "test") {
				return false, "contains forbidden word 'test'"
			}
			return true, ""
		},
		Severity: SeverityMedium,
	}
	
	validator.AddCustomRule(customRule)
	
	// Test input that violates custom rule
	result := validator.ValidateInput("this is a test", "field")
	if result.Valid {
		t.Error("Expected validation to fail due to custom rule")
	}
	
	found := false
	for _, issue := range result.Issues {
		if issue.Type == "custom_rule_violation" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected custom rule violation in issues")
	}
	
	// Test input that passes custom rule
	result2 := validator.ValidateInput("this is valid input", "field")
	if !result2.Valid {
		// Check if failure is due to custom rule
		customRuleViolation := false
		for _, issue := range result2.Issues {
			if issue.Type == "custom_rule_violation" {
				customRuleViolation = true
				break
			}
		}
		
		if customRuleViolation {
			t.Error("Expected custom rule to pass for valid input")
		}
	}
}

// BenchmarkSecureString benchmarks SecureString operations
func BenchmarkSecureString(b *testing.B) {
	testData := []byte("benchmark test data")
	passphrase := "benchmark-passphrase"
	
	b.Run("Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			secure, err := NewSecureString(testData, passphrase)
			if err != nil {
				b.Fatal(err)
			}
			secure.Clear()
		}
	})
	
	b.Run("Decrypt", func(b *testing.B) {
		secure, err := NewSecureString(testData, passphrase)
		if err != nil {
			b.Fatal(err)
		}
		defer secure.Clear()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := secure.Decrypt(passphrase)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkObfuscatedString benchmarks ObfuscatedString operations
func BenchmarkObfuscatedString(b *testing.B) {
	testData := "benchmark test data"
	
	b.Run("Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			obfuscated, err := NewObfuscatedStringFromString(testData)
			if err != nil {
				b.Fatal(err)
			}
			obfuscated.Clear()
		}
	})
	
	b.Run("Retrieve", func(b *testing.B) {
		obfuscated, err := NewObfuscatedStringFromString(testData)
		if err != nil {
			b.Fatal(err)
		}
		defer obfuscated.Clear()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = obfuscated.String()
		}
	})
}