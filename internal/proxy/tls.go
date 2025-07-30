// internal/proxy/tls.go
package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// BuildTLSConfig creates a tls.Config from TLS configuration
func BuildTLSConfig(config *TLSConfig) (*tls.Config, error) {
	if config == nil {
		// Default secure configuration
		return &tls.Config{
			InsecureSkipVerify: false,
		}, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
		ServerName:         config.ServerName,
	}

	// Warn about insecure configuration
	if config.InsecureSkipVerify {
		fmt.Fprintf(os.Stderr, "WARNING: TLS certificate verification is disabled (insecure_skip_verify: true)\n")
		fmt.Fprintf(os.Stderr, "This makes connections vulnerable to man-in-the-middle attacks!\n")
		fmt.Fprintf(os.Stderr, "Only use this setting for testing or with trusted internal services.\n")
	}

	// Set up custom root CAs if provided
	if len(config.RootCAs) > 0 {
		rootCAs := x509.NewCertPool()
		for _, caFile := range config.RootCAs {
			caCert, err := os.ReadFile(caFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read root CA file %s: %v", caFile, err)
			}
			
			if !rootCAs.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse root CA certificate from %s", caFile)
			}
		}
		tlsConfig.RootCAs = rootCAs
	}

	// Set up client certificate for mutual TLS if provided
	if config.ClientCert != "" && config.ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// ValidateTLSConfig validates TLS configuration
func ValidateTLSConfig(config *TLSConfig) error {
	if config == nil {
		return nil
	}

	// Validate client certificate configuration
	if (config.ClientCert != "" && config.ClientKey == "") || 
	   (config.ClientCert == "" && config.ClientKey != "") {
		return fmt.Errorf("both client_cert and client_key must be provided for mutual TLS")
	}

	// Check that certificate files exist
	if config.ClientCert != "" {
		if _, err := os.Stat(config.ClientCert); os.IsNotExist(err) {
			return fmt.Errorf("client certificate file does not exist: %s", config.ClientCert)
		}
	}

	if config.ClientKey != "" {
		if _, err := os.Stat(config.ClientKey); os.IsNotExist(err) {
			return fmt.Errorf("client key file does not exist: %s", config.ClientKey)
		}
	}

	// Check that root CA files exist
	for _, caFile := range config.RootCAs {
		if _, err := os.Stat(caFile); os.IsNotExist(err) {
			return fmt.Errorf("root CA file does not exist: %s", caFile)
		}
	}

	return nil
}

// GetDefaultTLSConfig returns a secure default TLS configuration
func GetDefaultTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}
}

// GetInsecureTLSConfig returns an insecure TLS configuration for testing
// WARNING: This should only be used for testing purposes
func GetInsecureTLSConfig() *tls.Config {
	fmt.Fprintf(os.Stderr, "WARNING: Using insecure TLS configuration for testing\n")
	return &tls.Config{
		InsecureSkipVerify: true,
	}
}