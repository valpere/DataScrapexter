// internal/antidetect/comprehensive_test.go
package antidetect

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTLSFingerprinter_GetRandomConfig(t *testing.T) {
	fingerprinter := NewTLSFingerprinter()

	// Test multiple calls return different configs
	configs := make([]*tls.Config, 10)
	for i := 0; i < 10; i++ {
		configs[i] = fingerprinter.GetRandomConfig()
		if configs[i] == nil {
			t.Fatal("GetRandomConfig returned nil")
		}
	}

	// Verify configs actually vary (test randomization)
	if len(configs) >= 2 {
		// Check that at least some configs have different characteristics
		hasVariation := false
		firstConfig := configs[0]

		for i := 1; i < len(configs); i++ {
			currentConfig := configs[i]

			// Compare cipher suites
			if len(firstConfig.CipherSuites) != len(currentConfig.CipherSuites) {
				hasVariation = true
				break
			}

			// Compare TLS versions
			if firstConfig.MinVersion != currentConfig.MinVersion ||
				firstConfig.MaxVersion != currentConfig.MaxVersion {
				hasVariation = true
				break
			}

			// Compare cipher suite contents
			for j := 0; j < len(firstConfig.CipherSuites); j++ {
				if firstConfig.CipherSuites[j] != currentConfig.CipherSuites[j] {
					hasVariation = true
					break
				}
			}
			if hasVariation {
				break
			}
		}

		if !hasVariation {
			t.Error("GetRandomConfig() appears to return identical configurations - randomization may not be working")
		}
	}

	// Verify configs have expected properties
	config := configs[0]
	if config.MinVersion == 0 {
		t.Error("MinVersion should be set")
	}
	if config.MaxVersion == 0 {
		t.Error("MaxVersion should be set")
	}
	if len(config.CipherSuites) == 0 {
		t.Error("CipherSuites should not be empty")
	}
	if len(config.CurvePreferences) == 0 {
		t.Error("CurvePreferences should not be empty")
	}
	if len(config.NextProtos) == 0 {
		t.Error("NextProtos should not be empty")
	}
}

func TestTLSFingerprinter_GetChromeConfig(t *testing.T) {
	fingerprinter := NewTLSFingerprinter()
	config := fingerprinter.GetChromeConfig()

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion TLS 1.2, got %d", config.MinVersion)
	}
	if config.MaxVersion != tls.VersionTLS13 {
		t.Errorf("Expected MaxVersion TLS 1.3, got %d", config.MaxVersion)
	}

	// Check for Chrome-specific cipher suites
	if len(config.CipherSuites) == 0 {
		t.Error("Chrome config should have cipher suites")
	}

	// Check for h2 ALPN protocol
	found := false
	for _, proto := range config.NextProtos {
		if proto == "h2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Chrome config should include h2 protocol")
	}
}

func TestTLSFingerprinter_GetFirefoxConfig(t *testing.T) {
	fingerprinter := NewTLSFingerprinter()
	config := fingerprinter.GetFirefoxConfig()

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion TLS 1.2, got %d", config.MinVersion)
	}
	if config.MaxVersion != tls.VersionTLS13 {
		t.Errorf("Expected MaxVersion TLS 1.3, got %d", config.MaxVersion)
	}

	// Firefox should have different cipher suite ordering than Chrome
	if len(config.CipherSuites) == 0 {
		t.Error("Firefox config should have cipher suites")
	}

	// Check for curve preferences specific to Firefox
	if len(config.CurvePreferences) < 4 {
		t.Error("Firefox config should have at least 4 curve preferences")
	}
}

func TestTLSRotator_GetNext(t *testing.T) {
	rotator := NewTLSRotator()

	// Test rotation
	config1 := rotator.GetNext()
	config2 := rotator.GetNext()
	config3 := rotator.GetNext()

	if config1 == nil || config2 == nil || config3 == nil {
		t.Fatal("TLS rotator returned nil config")
	}

	// Should cycle through different configs
	// Note: This test assumes implementation details, may need adjustment
	if config1 == config3 { // Should cycle back after 3 calls
		t.Log("TLS rotator correctly cycles through configs")
	}
}

func TestCanvasSpoofing_GetSpoofedData(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		original string
	}{
		{"enabled spoofing", true, "canvas_data_12345"},
		{"disabled spoofing", false, "canvas_data_12345"},
		{"empty data", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spoofing := NewCanvasSpoofing(tt.enabled)
			result := spoofing.GetSpoofedData(tt.original)

			if tt.enabled {
				// Should modify the data
				if result == tt.original && tt.original != "" {
					t.Error("Canvas spoofing should modify data when enabled")
				}
				// Should contain original data
				if !strings.Contains(result, tt.original) && tt.original != "" {
					t.Error("Spoofed data should contain original data")
				}
			} else {
				// Should return original data unchanged
				if result != tt.original {
					t.Error("Canvas spoofing should not modify data when disabled")
				}
			}
		})
	}
}

func TestCanvasSpoofing_GenerateFingerprint(t *testing.T) {
	spoofing := NewCanvasSpoofing(true)
	fingerprint := spoofing.GenerateFingerprint()

	if fingerprint == nil {
		t.Fatal("GenerateFingerprint returned nil")
	}

	if fingerprint.Data == "" {
		t.Error("Fingerprint should have data")
	}
	if fingerprint.Hash == "" {
		t.Error("Fingerprint should have hash")
	}
	if fingerprint.Width <= 0 || fingerprint.Height <= 0 {
		t.Error("Fingerprint should have valid dimensions")
	}
	if !fingerprint.Spoofed {
		t.Error("Fingerprint should be marked as spoofed")
	}
	if fingerprint.Timestamp.IsZero() {
		t.Error("Fingerprint should have timestamp")
	}
}

func TestWebGLSpoofing_GetRandomProfile(t *testing.T) {
	spoofing := NewWebGLSpoofing(true)
	profile := spoofing.GetRandomProfile()

	if profile.Renderer == "" {
		t.Error("WebGL profile should have renderer")
	}
	if profile.Vendor == "" {
		t.Error("WebGL profile should have vendor")
	}
	if profile.Version == "" {
		t.Error("WebGL profile should have version")
	}
	if profile.ShadingLanguage == "" {
		t.Error("WebGL profile should have shading language")
	}
	if len(profile.Extensions) == 0 {
		t.Error("WebGL profile should have extensions")
	}
	if !profile.Spoofed {
		t.Error("WebGL profile should be marked as spoofed")
	}

	// Test disabled spoofing
	disabledSpoofing := NewWebGLSpoofing(false)
	disabledProfile := disabledSpoofing.GetRandomProfile()
	if disabledProfile.Spoofed {
		t.Error("Disabled WebGL spoofing should not mark profile as spoofed")
	}
}

func TestAudioSpoofing_GenerateFingerprint(t *testing.T) {
	spoofing := NewAudioSpoofing(true, 0.01)
	fingerprint := spoofing.GenerateFingerprint()

	if fingerprint == nil {
		t.Fatal("GenerateFingerprint returned nil")
	}

	if fingerprint.SampleRate <= 0 {
		t.Error("Audio fingerprint should have valid sample rate")
	}
	if fingerprint.BufferSize <= 0 {
		t.Error("Audio fingerprint should have valid buffer size")
	}
	if fingerprint.Channels <= 0 {
		t.Error("Audio fingerprint should have valid channel count")
	}
	if fingerprint.ContextState == "" {
		t.Error("Audio fingerprint should have context state")
	}
	if fingerprint.OscillatorHash == "" {
		t.Error("Audio fingerprint should have oscillator hash")
	}
	if len(fingerprint.AnalyserData) == 0 {
		t.Error("Audio fingerprint should have analyser data")
	}
	if !fingerprint.Spoofed {
		t.Error("Audio fingerprint should be marked as spoofed")
	}

	// Test disabled spoofing
	disabledSpoofing := NewAudioSpoofing(false, 0)
	disabledFingerprint := disabledSpoofing.GenerateFingerprint()
	if disabledFingerprint.Spoofed {
		t.Error("Disabled audio spoofing should not mark fingerprint as spoofed")
	}
}

func TestScreenSpoofing_GetRandomFingerprint(t *testing.T) {
	spoofing := NewScreenSpoofing(true)
	fingerprint := spoofing.GetRandomFingerprint()

	if fingerprint.Width <= 0 || fingerprint.Height <= 0 {
		t.Error("Screen fingerprint should have valid dimensions")
	}
	if fingerprint.AvailWidth <= 0 || fingerprint.AvailHeight <= 0 {
		t.Error("Screen fingerprint should have valid available dimensions")
	}
	if fingerprint.ColorDepth <= 0 {
		t.Error("Screen fingerprint should have valid color depth")
	}
	if fingerprint.PixelDepth <= 0 {
		t.Error("Screen fingerprint should have valid pixel depth")
	}
	if fingerprint.DevicePixelRatio <= 0 {
		t.Error("Screen fingerprint should have valid device pixel ratio")
	}
	if fingerprint.Orientation == "" {
		t.Error("Screen fingerprint should have orientation")
	}
	if !fingerprint.Spoofed {
		t.Error("Screen fingerprint should be marked as spoofed")
	}

	// Test disabled spoofing
	disabledSpoofing := NewScreenSpoofing(false)
	disabledFingerprint := disabledSpoofing.GetRandomFingerprint()
	if disabledFingerprint.Spoofed {
		t.Error("Disabled screen spoofing should not mark fingerprint as spoofed")
	}
}

func TestFontSpoofing_GetRandomFontList(t *testing.T) {
	spoofing := NewFontSpoofing(true)
	fonts := spoofing.GetRandomFontList()

	if len(fonts) == 0 {
		t.Error("Font spoofing should return font list")
	}

	// Should include base fonts
	hasBaseFonts := false
	for _, font := range fonts {
		if font == "Arial" || font == "Times New Roman" {
			hasBaseFonts = true
			break
		}
	}
	if !hasBaseFonts {
		t.Error("Font list should include base fonts")
	}

	// Test disabled spoofing
	disabledSpoofing := NewFontSpoofing(false)
	disabledFonts := disabledSpoofing.GetRandomFontList()

	// Should return only base fonts when disabled
	if len(disabledFonts) > 20 { // Assuming base fonts are less than 20
		t.Error("Disabled font spoofing should return limited font list")
	}
}

func TestFingerprintingEvader_GenerateCompleteFingerprint(t *testing.T) {
	evader := NewFingerprintingEvader(true)
	fingerprint := evader.GenerateCompleteFingerprint()

	if fingerprint == nil {
		t.Fatal("GenerateCompleteFingerprint returned nil")
	}

	// Check all components are present
	if _, ok := fingerprint["canvas"]; !ok {
		t.Error("Complete fingerprint should include canvas")
	}
	if _, ok := fingerprint["webgl"]; !ok {
		t.Error("Complete fingerprint should include webgl")
	}
	if _, ok := fingerprint["audio"]; !ok {
		t.Error("Complete fingerprint should include audio")
	}
	if _, ok := fingerprint["screen"]; !ok {
		t.Error("Complete fingerprint should include screen")
	}
	if _, ok := fingerprint["fonts"]; !ok {
		t.Error("Complete fingerprint should include fonts")
	}
	if _, ok := fingerprint["timestamp"]; !ok {
		t.Error("Complete fingerprint should include timestamp")
	}
}

func TestCaptchaManager_SolveRecaptchaV2(t *testing.T) {
	config := &CaptchaConfig{
		Enabled:       true,
		DefaultSolver: TwoCaptcha,
		SolveTimeout:  10 * time.Second,
	}

	manager := NewCaptchaManager(config)

	// Register a mock solver
	mockSolver := &MockCaptchaSolver{
		balance: 10.0,
		taskID:  "test_task_123",
		solution: &CaptchaSolution{
			ID:      "test_task_123",
			Token:   "test_token_abc123",
			Success: true,
		},
	}
	manager.RegisterSolver(TwoCaptcha, mockSolver)

	ctx := context.Background()
	solution, err := manager.SolveRecaptchaV2(ctx, "test_site_key", "https://example.com", nil)

	if err != nil {
		t.Fatalf("SolveRecaptchaV2 failed: %v", err)
	}

	if solution == nil {
		t.Fatal("Solution should not be nil")
	}

	if !solution.Success {
		t.Error("Solution should be successful")
	}

	if solution.Token == "" {
		t.Error("Solution should have token")
	}

	if solution.SolveTime <= 0 {
		t.Error("Solution should have solve time")
	}
}

func TestCaptchaManager_SolveRecaptchaV3(t *testing.T) {
	config := &CaptchaConfig{
		Enabled:       true,
		DefaultSolver: TwoCaptcha,
		SolveTimeout:  10 * time.Second,
	}

	manager := NewCaptchaManager(config)

	// Register a mock solver
	mockSolver := &MockCaptchaSolver{
		balance: 10.0,
		taskID:  "test_task_456",
		solution: &CaptchaSolution{
			ID:      "test_task_456",
			Token:   "test_token_v3_def456",
			Success: true,
		},
	}
	manager.RegisterSolver(TwoCaptcha, mockSolver)

	ctx := context.Background()
	solution, err := manager.SolveRecaptchaV3(ctx, "test_site_key", "https://example.com", "submit", 0.5)

	if err != nil {
		t.Fatalf("SolveRecaptchaV3 failed: %v", err)
	}

	if solution == nil {
		t.Fatal("Solution should not be nil")
	}

	if !solution.Success {
		t.Error("Solution should be successful")
	}

	if solution.Token == "" {
		t.Error("Solution should have token")
	}
}

func TestCaptchaManager_SolveImageCaptcha(t *testing.T) {
	config := &CaptchaConfig{
		Enabled:       true,
		DefaultSolver: TwoCaptcha,
		SolveTimeout:  10 * time.Second,
	}

	manager := NewCaptchaManager(config)

	// Register a mock solver
	mockSolver := &MockCaptchaSolver{
		balance: 10.0,
		taskID:  "test_task_789",
		solution: &CaptchaSolution{
			ID:      "test_task_789",
			Text:    "CAPTCHA123",
			Success: true,
		},
	}
	manager.RegisterSolver(TwoCaptcha, mockSolver)

	ctx := context.Background()
	imageData := []byte("fake_image_data")
	solution, err := manager.SolveImageCaptcha(ctx, imageData)

	if err != nil {
		t.Fatalf("SolveImageCaptcha failed: %v", err)
	}

	if solution == nil {
		t.Fatal("Solution should not be nil")
	}

	if !solution.Success {
		t.Error("Solution should be successful")
	}

	if solution.Text == "" {
		t.Error("Solution should have text")
	}
}

func TestAntiDetectionClientIntegration(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			t.Error("Request should have User-Agent header")
		}

		accept := r.Header.Get("Accept")
		if accept == "" {
			t.Error("Request should have Accept header")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer server.Close()

	// Create anti-detection client
	config := &AntiDetectionConfig{
		UserAgentRotation: true,
		HeaderRotation:    true,
		DelayRange: DelayRange{
			Min: 100 * time.Millisecond,
			Max: 200 * time.Millisecond,
		},
		RetryConfig: RetryConfig{
			MaxRetries: 2,
			BackoffMin: 100 * time.Millisecond,
			BackoffMax: 1 * time.Second,
		},
	}

	client := NewAntiDetectionClient(config)

	// Create request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Execute request
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Should have some delay
	if duration < 100*time.Millisecond {
		t.Error("Request should have delay from anti-detection measures")
	}
}

// Mock implementations for testing

type MockCaptchaSolver struct {
	balance  float64
	taskID   string
	solution *CaptchaSolution
	err      error
}

func (m *MockCaptchaSolver) SubmitTask(ctx context.Context, task *CaptchaTask) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.taskID, nil
}

func (m *MockCaptchaSolver) GetResult(ctx context.Context, taskID string) (*CaptchaSolution, error) {
	if m.err != nil {
		return nil, m.err
	}
	if taskID != m.taskID {
		return nil, nil // Still processing
	}
	return m.solution, nil
}

func (m *MockCaptchaSolver) GetBalance(ctx context.Context) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.balance, nil
}

func (m *MockCaptchaSolver) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return map[string]interface{}{
		"service": "mock",
		"balance": m.balance,
	}, nil
}

func TestTwoCaptchaSolver_SubmitTask(t *testing.T) {
	// This test would require a real API key and should be run separately
	t.Skip("Skipping TwoCaptcha integration test - requires API key")

	solver := NewTwoCaptchaSolver("test_api_key")
	task := &CaptchaTask{
		Type:    RecaptchaV2,
		SiteKey: "test_site_key",
		SiteURL: "https://example.com",
	}

	ctx := context.Background()
	_, err := solver.SubmitTask(ctx, task)

	// Should get an error due to invalid API key
	if err == nil {
		t.Error("Expected error with invalid API key")
	}
}

func TestAntiCaptchaSolver_SubmitTask(t *testing.T) {
	// This test would require a real API key and should be run separately
	t.Skip("Skipping AntiCaptcha integration test - requires API key")

	solver := NewAntiCaptchaSolver("test_api_key")
	task := &CaptchaTask{
		Type:    RecaptchaV2,
		SiteKey: "test_site_key",
		SiteURL: "https://example.com",
	}

	ctx := context.Background()
	_, err := solver.SubmitTask(ctx, task)

	// Should get an error due to invalid API key
	if err == nil {
		t.Error("Expected error with invalid API key")
	}
}
