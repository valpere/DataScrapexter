// internal/antidetect/antidetect_test.go
package antidetect

import (
	"testing"
)

func TestUserAgentRotatorIntegration(t *testing.T) {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
	}

	rotator := NewUserAgentRotator(userAgents)

	if rotator == nil {
		t.Error("rotator should not be nil")
	}
}

func TestAntiDetectionClientCreation(t *testing.T) {
	config := &AntiDetectionConfig{}

	client := NewAntiDetectionClient(config)

	if client == nil {
		t.Error("anti-detection client should not be nil")
	}
}
