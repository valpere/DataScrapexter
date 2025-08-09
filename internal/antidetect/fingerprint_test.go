package antidetect

import (
	"errors"
	"testing"
	"time"
)

func TestEntropyMetrics(t *testing.T) {
	// Create a new metrics instance for testing
	em := &EntropyMetrics{
		failures:        make(map[string]*int64),
		recentFailures:  make([]EntropyFailureEvent, 0, 10),
		retentionPeriod: 1 * time.Hour,
		alertThreshold:  5,
	}
	
	// Test recording failures
	testError := errors.New("test entropy failure")
	em.RecordFailure("test_context", testError)
	em.RecordFailure("test_context", testError)
	em.RecordFailure("other_context", testError)
	
	// Get metrics and verify
	metrics := em.GetMetrics()
	
	if metrics["total_failures"] != int64(3) {
		t.Errorf("Expected total_failures=3, got %v", metrics["total_failures"])
	}
	
	if metrics["consecutive_failures"] != int64(3) {
		t.Errorf("Expected consecutive_failures=3, got %v", metrics["consecutive_failures"])
	}
	
	contextBreakdown := metrics["context_breakdown"].(map[string]int64)
	if contextBreakdown["test_context"] != int64(2) {
		t.Errorf("Expected test_context=2, got %v", contextBreakdown["test_context"])
	}
	
	if contextBreakdown["other_context"] != int64(1) {
		t.Errorf("Expected other_context=1, got %v", contextBreakdown["other_context"])
	}
	
	// Test reset consecutive failures
	em.ResetConsecutiveFailures()
	metrics = em.GetMetrics()
	
	if metrics["consecutive_failures"] != int64(0) {
		t.Errorf("Expected consecutive_failures=0 after reset, got %v", metrics["consecutive_failures"])
	}
	
	// Total failures should remain the same
	if metrics["total_failures"] != int64(3) {
		t.Errorf("Expected total_failures=3 after reset, got %v", metrics["total_failures"])
	}
}

func TestCanvasSpoofing(t *testing.T) {
	// Test enabled canvas spoofing
	cs := NewCanvasSpoofing(true)
	if !cs.IsEnabled() {
		t.Error("Expected canvas spoofing to be enabled")
	}
	
	original := "test_canvas_data"
	spoofed := cs.GetSpoofedData(original)
	
	if spoofed == original {
		t.Error("Expected spoofed data to be different from original")
	}
	
	if len(spoofed) <= len(original) {
		t.Error("Expected spoofed data to be longer than original")
	}
	
	// Test disabled canvas spoofing
	csDisabled := NewCanvasSpoofing(false)
	if csDisabled.IsEnabled() {
		t.Error("Expected canvas spoofing to be disabled")
	}
	
	spoofedDisabled := csDisabled.GetSpoofedData(original)
	if spoofedDisabled != original {
		t.Error("Expected disabled spoofing to return original data")
	}
}

func TestWebGLSpoofing(t *testing.T) {
	ws := NewWebGLSpoofing(true)
	if !ws.IsEnabled() {
		t.Error("Expected WebGL spoofing to be enabled")
	}
	
	profile := ws.GetRandomProfile()
	if !profile.Spoofed {
		t.Error("Expected profile to be marked as spoofed")
	}
	
	if profile.Renderer == "" {
		t.Error("Expected renderer to be set")
	}
	
	if profile.Vendor == "" {
		t.Error("Expected vendor to be set")
	}
}

func TestAudioSpoofing(t *testing.T) {
	as := NewAudioSpoofing(true, 0.01)
	if !as.IsEnabled() {
		t.Error("Expected audio spoofing to be enabled")
	}
	
	fingerprint := as.GenerateFingerprint()
	if !fingerprint.Spoofed {
		t.Error("Expected fingerprint to be marked as spoofed")
	}
	
	if fingerprint.SampleRate == 44100.0 {
		t.Error("Expected sample rate to be modified when spoofing enabled")
	}
	
	if fingerprint.Channels != 2 {
		t.Error("Expected 2 channels")
	}
}

func TestScreenSpoofing(t *testing.T) {
	ss := NewScreenSpoofing(true)
	if !ss.IsEnabled() {
		t.Error("Expected screen spoofing to be enabled")
	}
	
	fingerprint := ss.GetRandomFingerprint()
	if !fingerprint.Spoofed {
		t.Error("Expected fingerprint to be marked as spoofed")
	}
	
	if fingerprint.Width == 0 || fingerprint.Height == 0 {
		t.Error("Expected valid screen dimensions")
	}
	
	if fingerprint.ColorDepth != 24 {
		t.Error("Expected color depth of 24")
	}
}

func TestFontSpoofing(t *testing.T) {
	fs := NewFontSpoofing(true)
	if !fs.IsEnabled() {
		t.Error("Expected font spoofing to be enabled")
	}
	
	fonts := fs.GetRandomFontList()
	if len(fonts) == 0 {
		t.Error("Expected non-empty font list")
	}
	
	// Check that we have at least some base fonts
	hasArial := false
	for _, font := range fonts {
		if font == "Arial" {
			hasArial = true
			break
		}
	}
	
	if !hasArial {
		t.Error("Expected Arial to be in font list")
	}
}

func TestFingerprintingEvader(t *testing.T) {
	evader := NewFingerprintingEvader(true)
	
	if evader.Canvas == nil {
		t.Error("Expected canvas spoofing to be initialized")
	}
	
	if evader.WebGL == nil {
		t.Error("Expected WebGL spoofing to be initialized")
	}
	
	if evader.Audio == nil {
		t.Error("Expected audio spoofing to be initialized")
	}
	
	if evader.Screen == nil {
		t.Error("Expected screen spoofing to be initialized")
	}
	
	if evader.Font == nil {
		t.Error("Expected font spoofing to be initialized")
	}
	
	// Test complete fingerprint generation
	complete := evader.GenerateCompleteFingerprint()
	if len(complete) == 0 {
		t.Error("Expected non-empty complete fingerprint")
	}
	
	expectedKeys := []string{"canvas", "webgl", "audio", "screen", "fonts", "timestamp"}
	for _, key := range expectedKeys {
		if _, exists := complete[key]; !exists {
			t.Errorf("Expected key %s in complete fingerprint", key)
		}
	}
}

func TestSanitizeErrorForLogging(t *testing.T) {
	testCases := []struct {
		input    error
		expected string
	}{
		{errors.New("crypto/rand: read failed"), "entropy_source_unavailable"},
		{errors.New("random generation failed"), "randomization_failure"},
		{errors.New("read operation failed"), "read_operation_failed"},
		{errors.New("no such file"), "resource_unavailable"},
		{errors.New("permission denied"), "permission_denied"},
		{errors.New("operation timeout"), "operation_timeout"},
		{errors.New("context deadline exceeded"), "context_error"},
		{errors.New("unknown error"), "unclassified_error"},
		{nil, "unknown_error"},
	}
	
	for _, tc := range testCases {
		result := sanitizeErrorForLogging(tc.input)
		if result != tc.expected {
			t.Errorf("Expected %s for error %v, got %s", tc.expected, tc.input, result)
		}
	}
}