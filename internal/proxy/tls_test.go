// internal/proxy/tls_test.go
package proxy

import (
	"os"
	"strings"
	"testing"
)

func TestBuildTLSConfig_Secure(t *testing.T) {
	// Test secure default configuration
	config := &TLSConfig{
		InsecureSkipVerify: false,
	}

	tlsConfig, err := BuildTLSConfig(config)
	if err != nil {
		t.Fatalf("BuildTLSConfig() returned error: %v", err)
	}

	if tlsConfig.InsecureSkipVerify {
		t.Errorf("Expected InsecureSkipVerify to be false")
	}
}

func TestBuildTLSConfig_Insecure(t *testing.T) {
	// Capture stderr to check for warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	config := &TLSConfig{
		InsecureSkipVerify: true,
	}

	tlsConfig, err := BuildTLSConfig(config)
	if err != nil {
		t.Fatalf("BuildTLSConfig() returned error: %v", err)
	}

	if !tlsConfig.InsecureSkipVerify {
		t.Errorf("Expected InsecureSkipVerify to be true")
	}

	// Close write end and read the warning
	w.Close()
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	os.Stderr = oldStderr

	warning := string(buf[:n])
	if !strings.Contains(warning, "WARNING") {
		t.Errorf("Expected warning message about insecure configuration, got: %s", warning)
	}
	if !strings.Contains(warning, "man-in-the-middle") {
		t.Errorf("Expected warning about man-in-the-middle attacks, got: %s", warning)
	}
}

func TestBuildTLSConfig_Nil(t *testing.T) {
	tlsConfig, err := BuildTLSConfig(nil)
	if err != nil {
		t.Fatalf("BuildTLSConfig() returned error: %v", err)
	}

	if tlsConfig.InsecureSkipVerify {
		t.Errorf("Expected default configuration to be secure (InsecureSkipVerify: false)")
	}
}

func TestBuildTLSConfig_ServerName(t *testing.T) {
	config := &TLSConfig{
		ServerName: "example.com",
	}

	tlsConfig, err := BuildTLSConfig(config)
	if err != nil {
		t.Fatalf("BuildTLSConfig() returned error: %v", err)
	}

	if tlsConfig.ServerName != "example.com" {
		t.Errorf("Expected ServerName to be 'example.com', got: %s", tlsConfig.ServerName)
	}
}

func TestValidateTLSConfig_Valid(t *testing.T) {
	config := &TLSConfig{
		InsecureSkipVerify: false,
		ServerName:         "example.com",
	}

	err := ValidateTLSConfig(config)
	if err != nil {
		t.Errorf("ValidateTLSConfig() returned error for valid config: %v", err)
	}
}

func TestValidateTLSConfig_Nil(t *testing.T) {
	err := ValidateTLSConfig(nil)
	if err != nil {
		t.Errorf("ValidateTLSConfig() should not error for nil config: %v", err)
	}
}

func TestValidateTLSConfig_IncompleteClientAuth(t *testing.T) {
	tests := []struct {
		name       string
		config     *TLSConfig
		shouldFail bool
	}{
		{
			name: "cert without key",
			config: &TLSConfig{
				ClientCert: "/path/to/cert.pem",
			},
			shouldFail: true,
		},
		{
			name: "key without cert",
			config: &TLSConfig{
				ClientKey: "/path/to/key.pem",
			},
			shouldFail: true,
		},
		{
			name: "both cert and key",
			config: &TLSConfig{
				ClientCert: "/nonexistent/cert.pem",
				ClientKey:  "/nonexistent/key.pem",
			},
			shouldFail: true, // Files don't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSConfig(tt.config)
			if tt.shouldFail && err == nil {
				t.Errorf("Expected validation to fail for %s", tt.name)
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Expected validation to pass for %s, got: %v", tt.name, err)
			}
		})
	}
}

func TestGetDefaultTLSConfig(t *testing.T) {
	tlsConfig := GetDefaultTLSConfig()
	
	if tlsConfig.InsecureSkipVerify {
		t.Errorf("Default TLS config should be secure")
	}
	
	if tlsConfig.MinVersion == 0 {
		t.Errorf("Default TLS config should specify minimum TLS version")
	}
}

func TestGetInsecureTLSConfig(t *testing.T) {
	// Capture stderr to check for warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	tlsConfig := GetInsecureTLSConfig()

	// Close write end and read the warning
	w.Close()
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	os.Stderr = oldStderr

	if !tlsConfig.InsecureSkipVerify {
		t.Errorf("Insecure TLS config should have InsecureSkipVerify: true")
	}

	warning := string(buf[:n])
	if !strings.Contains(warning, "WARNING") {
		t.Errorf("Expected warning message about insecure configuration")
	}
}