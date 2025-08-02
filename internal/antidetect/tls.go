// internal/antidetect/tls.go
package antidetect

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"time"
)

// TLSFingerprintConfig configures TLS fingerprinting
type TLSFingerprintConfig struct {
	MinVersion         uint16
	MaxVersion         uint16
	CipherSuites       []uint16
	CurvePreferences   []tls.CurveID
	SignatureSchemes   []tls.SignatureScheme
	ALPNProtocols      []string
	InsecureSkipVerify bool
	ServerName         string
}

// TLSFingerprinter provides TLS fingerprinting evasion
type TLSFingerprinter struct {
	profiles []TLSFingerprintConfig
}

// NewTLSFingerprinter creates a new TLS fingerprinter
func NewTLSFingerprinter() *TLSFingerprinter {
	return &TLSFingerprinter{
		profiles: getDefaultTLSProfiles(),
	}
}

// GetRandomConfig returns a random TLS configuration
func (tf *TLSFingerprinter) GetRandomConfig() *tls.Config {
	profile := tf.profiles[rand.Intn(len(tf.profiles))]
	
	return &tls.Config{
		MinVersion:               profile.MinVersion,
		MaxVersion:               profile.MaxVersion,
		CipherSuites:             profile.CipherSuites,
		CurvePreferences:         profile.CurvePreferences,
		NextProtos:               profile.ALPNProtocols,
		InsecureSkipVerify:       profile.InsecureSkipVerify,
		ServerName:               profile.ServerName,
		ClientSessionCache:       tls.NewLRUClientSessionCache(64),
		Renegotiation:           tls.RenegotiateOnceAsClient,
		PreferServerCipherSuites: false,
	}
}

// GetChromeConfig returns a Chrome-like TLS configuration
func (tf *TLSFingerprinter) GetChromeConfig() *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: getChromeCipherSuites(),
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
		NextProtos: []string{"h2", "http/1.1"},
		ClientSessionCache: tls.NewLRUClientSessionCache(64),
		Renegotiation: tls.RenegotiateOnceAsClient,
	}
}

// GetFirefoxConfig returns a Firefox-like TLS configuration
func (tf *TLSFingerprinter) GetFirefoxConfig() *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: getFirefoxCipherSuites(),
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
		},
		NextProtos: []string{"h2", "http/1.1"},
		ClientSessionCache: tls.NewLRUClientSessionCache(32),
		Renegotiation: tls.RenegotiateOnceAsClient,
	}
}

// JA3Fingerprint represents a JA3 fingerprint
type JA3Fingerprint struct {
	TLSVersion       uint16
	CipherSuites     []uint16
	Extensions       []uint16
	EllipticCurves   []tls.CurveID
	ECPointFormats   []uint8
	SignatureSchemes []tls.SignatureScheme
}

// JA3Calculator calculates JA3 fingerprints
type JA3Calculator struct{}

// NewJA3Calculator creates a new JA3 calculator
func NewJA3Calculator() *JA3Calculator {
	return &JA3Calculator{}
}

// Calculate calculates JA3 fingerprint for a TLS config
func (calc *JA3Calculator) Calculate(config *tls.Config) string {
	// Simplified JA3 calculation - in practice, this would be more complex
	// and would need to match the actual JA3 algorithm
	version := config.MaxVersion
	if version == 0 {
		version = tls.VersionTLS13
	}
	
	cipherSuites := config.CipherSuites
	if len(cipherSuites) == 0 {
		cipherSuites = getDefaultCipherSuites()
	}
	
	// Format: TLSVersion,CipherSuites,Extensions,EllipticCurves,ECPointFormats
	return fmt.Sprintf("%d,%v,773-35-16-5-10-51-43-13-45-28,23-24-25,0",
		version, cipherSuites)
}

// Randomize randomizes JA3 fingerprint elements
func (calc *JA3Calculator) Randomize(base JA3Fingerprint) JA3Fingerprint {
	randomized := base
	
	// Shuffle cipher suites order
	rand.Shuffle(len(randomized.CipherSuites), func(i, j int) {
		randomized.CipherSuites[i], randomized.CipherSuites[j] = 
			randomized.CipherSuites[j], randomized.CipherSuites[i]
	})
	
	// Randomize some extensions order
	rand.Shuffle(len(randomized.Extensions), func(i, j int) {
		randomized.Extensions[i], randomized.Extensions[j] = 
			randomized.Extensions[j], randomized.Extensions[i]
	})
	
	return randomized
}

// TLSRotator rotates TLS configurations
type TLSRotator struct {
	fingerprinter *TLSFingerprinter
	configs       []*tls.Config
	index         int
}

// NewTLSRotator creates a new TLS rotator
func NewTLSRotator() *TLSRotator {
	fp := NewTLSFingerprinter()
	
	configs := []*tls.Config{
		fp.GetChromeConfig(),
		fp.GetFirefoxConfig(),
		fp.GetRandomConfig(),
	}
	
	return &TLSRotator{
		fingerprinter: fp,
		configs:       configs,
	}
}

// GetNext returns the next TLS configuration
func (tr *TLSRotator) GetNext() *tls.Config {
	config := tr.configs[tr.index]
	tr.index = (tr.index + 1) % len(tr.configs)
	return config
}

// GetRandom returns a random TLS configuration
func (tr *TLSRotator) GetRandom() *tls.Config {
	return tr.configs[rand.Intn(len(tr.configs))]
}

// CustomDialer creates a dialer with TLS fingerprinting
func (tr *TLSRotator) CustomDialer() func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		d := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		
		if network == "tcp" && isHTTPSAddr(addr) {
			// Use custom TLS config for HTTPS
			conn, err := d.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			
			tlsConfig := tr.GetRandom()
			tlsConn := tls.Client(conn, tlsConfig)
			
			return tlsConn, nil
		}
		
		return d.Dial(network, addr)
	}
}

// Helper functions

func getDefaultTLSProfiles() []TLSFingerprintConfig {
	return []TLSFingerprintConfig{
		// Chrome-like profile
		{
			MinVersion:       tls.VersionTLS12,
			MaxVersion:       tls.VersionTLS13,
			CipherSuites:     getChromeCipherSuites(),
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
			ALPNProtocols:    []string{"h2", "http/1.1"},
		},
		// Firefox-like profile
		{
			MinVersion:       tls.VersionTLS12,
			MaxVersion:       tls.VersionTLS13,
			CipherSuites:     getFirefoxCipherSuites(),
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384, tls.CurveP521},
			ALPNProtocols:    []string{"h2", "http/1.1"},
		},
		// Safari-like profile
		{
			MinVersion:       tls.VersionTLS12,
			MaxVersion:       tls.VersionTLS13,
			CipherSuites:     getSafariCipherSuites(),
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
			ALPNProtocols:    []string{"h2", "http/1.1"},
		},
	}
}

func getChromeCipherSuites() []uint16 {
	return []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	}
}

func getFirefoxCipherSuites() []uint16 {
	return []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	}
}

func getSafariCipherSuites() []uint16 {
	return []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	}
}

func getDefaultCipherSuites() []uint16 {
	return []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	}
}

func isHTTPSAddr(addr string) bool {
	// Simple check for HTTPS ports
	return addr[len(addr)-4:] == ":443" || addr[len(addr)-5:] == ":8443"
}