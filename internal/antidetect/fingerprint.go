// internal/antidetect/fingerprint.go
package antidetect

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	mathrand "math/rand"
	"time"
)

// CanvasFingerprint represents canvas fingerprinting data
type CanvasFingerprint struct {
	Data      string
	Hash      string
	Width     int
	Height    int
	Spoofed   bool
	Timestamp time.Time
}

// CanvasSpoofing provides canvas fingerprinting evasion
type CanvasSpoofing struct {
	enabled    bool
	variations []string
}

// NewCanvasSpoofing creates a new canvas spoofing system
func NewCanvasSpoofing(enabled bool) *CanvasSpoofing {
	variations, err := generateCanvasVariations()
	if err != nil {
		// Fall back to static variations if crypto/rand fails
		variations = []string{
			"a4b7c3d1e5f2", "b8e1f4g7h2i5", "c2f5g8h1i4j7", "d6g9h2i5j8k1",
			"e0h3i6j9k2l5", "f4i7j0k3l6m9", "g8j1k4l7m0n3", "h2k5l8m1n4o7",
			"i6l9m2n5o8p1", "j0m3n6o9p2q5",
		}
	}
	return &CanvasSpoofing{
		enabled:    enabled,
		variations: variations,
	}
}

// IsEnabled returns whether canvas spoofing is enabled
func (cs *CanvasSpoofing) IsEnabled() bool {
	return cs.enabled
}

// GetSpoofedData returns spoofed canvas data
func (cs *CanvasSpoofing) GetSpoofedData(original string) string {
	if !cs.enabled {
		return original
	}

	// Add slight variations to canvas data to avoid fingerprinting
	variation := cs.variations[mathrand.Intn(len(cs.variations))]
	return original + variation
}

// GenerateFingerprint creates a randomized canvas fingerprint
func (cs *CanvasSpoofing) GenerateFingerprint() *CanvasFingerprint {
	// Generate random canvas data
	data := generateRandomCanvasData()
	hash := generateHash(data)

	return &CanvasFingerprint{
		Data:      data,
		Hash:      hash,
		Width:     800 + mathrand.Intn(400), // 800-1200
		Height:    600 + mathrand.Intn(200), // 600-800
		Spoofed:   cs.enabled,
		Timestamp: time.Now(),
	}
}

// WebGLFingerprint represents WebGL fingerprinting data
type WebGLFingerprint struct {
	Renderer        string
	Vendor          string
	Version         string
	ShadingLanguage string
	Extensions      []string
	Parameters      map[string]interface{}
	Spoofed         bool
}

// WebGLSpoofing provides WebGL fingerprinting evasion
type WebGLSpoofing struct {
	enabled  bool
	profiles []WebGLFingerprint
}

// NewWebGLSpoofing creates a new WebGL spoofing system
func NewWebGLSpoofing(enabled bool) *WebGLSpoofing {
	return &WebGLSpoofing{
		enabled:  enabled,
		profiles: getWebGLProfiles(),
	}
}

// IsEnabled returns whether WebGL spoofing is enabled
func (ws *WebGLSpoofing) IsEnabled() bool {
	return ws.enabled
}

// GetRandomProfile returns a random WebGL profile
func (ws *WebGLSpoofing) GetRandomProfile() WebGLFingerprint {
	if !ws.enabled {
		return WebGLFingerprint{}
	}

	profile := ws.profiles[mathrand.Intn(len(ws.profiles))]
	profile.Spoofed = true
	return profile
}

// AudioFingerprint represents audio context fingerprinting data
type AudioFingerprint struct {
	SampleRate     float64
	BufferSize     int
	Channels       int
	ContextState   string
	OscillatorHash string
	AnalyserData   []float64
	Spoofed        bool
}

// AudioSpoofing provides audio fingerprinting evasion
type AudioSpoofing struct {
	enabled bool
	noise   float64
}

// NewAudioSpoofing creates a new audio spoofing system
func NewAudioSpoofing(enabled bool, noiseLevel float64) *AudioSpoofing {
	return &AudioSpoofing{
		enabled: enabled,
		noise:   noiseLevel,
	}
}

// IsEnabled returns whether audio spoofing is enabled
func (as *AudioSpoofing) IsEnabled() bool {
	return as.enabled
}

// GenerateFingerprint creates a randomized audio fingerprint
func (as *AudioSpoofing) GenerateFingerprint() *AudioFingerprint {
	// Standard audio context properties with slight variations
	sampleRate := 44100.0
	if as.enabled {
		sampleRate += (mathrand.Float64() - 0.5) * as.noise
	}

	bufferSize := 256
	if as.enabled {
		bufferSizes := []int{256, 512, 1024, 2048, 4096}
		bufferSize = bufferSizes[mathrand.Intn(len(bufferSizes))]
	}

	return &AudioFingerprint{
		SampleRate:     sampleRate,
		BufferSize:     bufferSize,
		Channels:       2,
		ContextState:   "running",
		OscillatorHash: generateOscillatorHash(as.enabled),
		AnalyserData:   generateAnalyserData(as.enabled),
		Spoofed:        as.enabled,
	}
}

// ScreenFingerprint represents screen/display fingerprinting data
type ScreenFingerprint struct {
	Width            int
	Height           int
	AvailWidth       int
	AvailHeight      int
	ColorDepth       int
	PixelDepth       int
	DevicePixelRatio float64
	Orientation      string
	Spoofed          bool
}

// ScreenSpoofing provides screen fingerprinting evasion
type ScreenSpoofing struct {
	enabled   bool
	presets   []ScreenFingerprint
	variation float64
}

// NewScreenSpoofing creates a new screen spoofing system
func NewScreenSpoofing(enabled bool) *ScreenSpoofing {
	return &ScreenSpoofing{
		enabled:   enabled,
		presets:   getCommonScreenSizes(),
		variation: 0.05, // 5% variation
	}
}

// IsEnabled returns whether screen spoofing is enabled
func (ss *ScreenSpoofing) IsEnabled() bool {
	return ss.enabled
}

// GetRandomFingerprint returns a randomized screen fingerprint
func (ss *ScreenSpoofing) GetRandomFingerprint() ScreenFingerprint {
	if !ss.enabled {
		// Return a common default
		return ScreenFingerprint{
			Width:            1920,
			Height:           1080,
			AvailWidth:       1920,
			AvailHeight:      1040,
			ColorDepth:       24,
			PixelDepth:       24,
			DevicePixelRatio: 1.0,
			Orientation:      "landscape-primary",
			Spoofed:          false,
		}
	}

	preset := ss.presets[mathrand.Intn(len(ss.presets))]

	// Add slight variations
	if ss.variation > 0 {
		variation := int(float64(preset.Width) * ss.variation)
		preset.Width += mathrand.Intn(variation*2) - variation
		preset.AvailWidth = preset.Width

		variation = int(float64(preset.Height) * ss.variation)
		preset.Height += mathrand.Intn(variation*2) - variation
		preset.AvailHeight = preset.Height - 40 // Account for taskbar
	}

	preset.Spoofed = true
	return preset
}

// FontFingerprint represents font fingerprinting data
type FontFingerprint struct {
	AvailableFonts []string
	CanvasWidth    map[string]float64
	Spoofed        bool
}

// FontSpoofing provides font fingerprinting evasion
type FontSpoofing struct {
	enabled    bool
	baseFonts  []string
	extraFonts []string
}

// NewFontSpoofing creates a new font spoofing system
func NewFontSpoofing(enabled bool) *FontSpoofing {
	return &FontSpoofing{
		enabled:    enabled,
		baseFonts:  getBaseFonts(),
		extraFonts: getExtraFonts(),
	}
}

// IsEnabled returns whether font spoofing is enabled
func (fs *FontSpoofing) IsEnabled() bool {
	return fs.enabled
}

// GetRandomFontList returns a randomized font list
func (fs *FontSpoofing) GetRandomFontList() []string {
	if !fs.enabled {
		return fs.baseFonts
	}

	// Start with base fonts
	fonts := make([]string, len(fs.baseFonts))
	copy(fonts, fs.baseFonts)

	// Add random selection of extra fonts
	numExtra := mathrand.Intn(len(fs.extraFonts)/2) + 1
	mathrand.Shuffle(len(fs.extraFonts), func(i, j int) {
		fs.extraFonts[i], fs.extraFonts[j] = fs.extraFonts[j], fs.extraFonts[i]
	})

	fonts = append(fonts, fs.extraFonts[:numExtra]...)

	// Shuffle the final list
	mathrand.Shuffle(len(fonts), func(i, j int) {
		fonts[i], fonts[j] = fonts[j], fonts[i]
	})

	return fonts
}

// FingerprintingEvader coordinates all fingerprinting evasion techniques
type FingerprintingEvader struct {
	Canvas *CanvasSpoofing
	WebGL  *WebGLSpoofing
	Audio  *AudioSpoofing
	Screen *ScreenSpoofing
	Font   *FontSpoofing
}

// NewFingerprintingEvader creates a comprehensive fingerprinting evader
func NewFingerprintingEvader(enabled bool) *FingerprintingEvader {
	return &FingerprintingEvader{
		Canvas: NewCanvasSpoofing(enabled),
		WebGL:  NewWebGLSpoofing(enabled),
		Audio:  NewAudioSpoofing(enabled, 0.01),
		Screen: NewScreenSpoofing(enabled),
		Font:   NewFontSpoofing(enabled),
	}
}

// GenerateCompleteFingerprint generates a complete spoofed fingerprint
func (fe *FingerprintingEvader) GenerateCompleteFingerprint() map[string]interface{} {
	return map[string]interface{}{
		"canvas":    fe.Canvas.GenerateFingerprint(),
		"webgl":     fe.WebGL.GetRandomProfile(),
		"audio":     fe.Audio.GenerateFingerprint(),
		"screen":    fe.Screen.GetRandomFingerprint(),
		"fonts":     fe.Font.GetRandomFontList(),
		"timestamp": time.Now(),
	}
}

// Helper functions

func generateCanvasVariations() ([]string, error) {
	variations := make([]string, 10)
	for i := range variations {
		// Generate small random strings that can be appended to canvas data
		bytes := make([]byte, 2)
		if _, err := rand.Read(bytes); err != nil {
			// SECURITY: Fail fast when cryptographic randomness is unavailable
			logEntropyFailure("canvas_variations", err)
			return nil, fmt.Errorf("crypto/rand failed: %w", err)
		} else {
			variations[i] = hex.EncodeToString(bytes)
		}
	}
	return variations, nil
}

func generateRandomCanvasData() string {
	// Simulate canvas rendering output
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		// SECURITY: Fail fast when cryptographic randomness is unavailable
		// Return deterministic error marker to avoid weak entropy
		return "error_crypto_rand_unavailable_canvas_data"
	}
	return hex.EncodeToString(data)
}

func generateHash(data string) string {
	// Generate cryptographically secure hash
	hash := make([]byte, 16)
	if _, err := rand.Read(hash); err != nil {
		// Fallback to deterministic hash of input data if crypto/rand fails
		// This maintains consistency while avoiding weak randomness
		h := sha256.Sum256([]byte(data + time.Now().String()))
		copy(hash, h[:16])
	}
	return hex.EncodeToString(hash)
}

func generateOscillatorHash(spoofed bool) string {
	if !spoofed {
		return "standard_oscillator_hash"
	}

	hash := make([]byte, 8)
	if _, err := rand.Read(hash); err != nil {
		// Fallback to deterministic time-based hash if crypto/rand fails
		// This maintains consistency while avoiding security degradation
		nano := time.Now().UnixNano()
		for i := range hash {
			hash[i] = byte(nano >> (i * 8))
		}
	}
	return hex.EncodeToString(hash)
}

func generateAnalyserData(spoofed bool) []float64 {
	data := make([]float64, 32)
	for i := range data {
		data[i] = mathrand.Float64()
		if spoofed {
			// Add small noise
			data[i] += (mathrand.Float64() - 0.5) * 0.01
		}
	}
	return data
}

func getWebGLProfiles() []WebGLFingerprint {
	return []WebGLFingerprint{
		{
			Renderer:        "ANGLE (Intel, Intel(R) UHD Graphics 620 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Vendor:          "Google Inc. (Intel)",
			Version:         "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguage: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			Extensions:      []string{"ANGLE_instanced_arrays", "EXT_blend_minmax", "EXT_color_buffer_half_float"},
		},
		{
			Renderer:        "ANGLE (NVIDIA, NVIDIA GeForce GTX 1660 Ti Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Vendor:          "Google Inc. (NVIDIA)",
			Version:         "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguage: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			Extensions:      []string{"ANGLE_instanced_arrays", "EXT_blend_minmax", "EXT_color_buffer_half_float"},
		},
		{
			Renderer:        "AMD Radeon RX 580 Series",
			Vendor:          "ATI Technologies Inc.",
			Version:         "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguage: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			Extensions:      []string{"ANGLE_instanced_arrays", "EXT_blend_minmax", "EXT_color_buffer_half_float"},
		},
	}
}

func getCommonScreenSizes() []ScreenFingerprint {
	return []ScreenFingerprint{
		{Width: 1920, Height: 1080, AvailWidth: 1920, AvailHeight: 1040, ColorDepth: 24, PixelDepth: 24, DevicePixelRatio: 1.0, Orientation: "landscape-primary"},
		{Width: 1366, Height: 768, AvailWidth: 1366, AvailHeight: 728, ColorDepth: 24, PixelDepth: 24, DevicePixelRatio: 1.0, Orientation: "landscape-primary"},
		{Width: 1536, Height: 864, AvailWidth: 1536, AvailHeight: 824, ColorDepth: 24, PixelDepth: 24, DevicePixelRatio: 1.25, Orientation: "landscape-primary"},
		{Width: 1440, Height: 900, AvailWidth: 1440, AvailHeight: 860, ColorDepth: 24, PixelDepth: 24, DevicePixelRatio: 1.0, Orientation: "landscape-primary"},
		{Width: 1280, Height: 720, AvailWidth: 1280, AvailHeight: 680, ColorDepth: 24, PixelDepth: 24, DevicePixelRatio: 1.0, Orientation: "landscape-primary"},
		{Width: 2560, Height: 1440, AvailWidth: 2560, AvailHeight: 1400, ColorDepth: 24, PixelDepth: 24, DevicePixelRatio: 1.0, Orientation: "landscape-primary"},
		{Width: 1600, Height: 900, AvailWidth: 1600, AvailHeight: 860, ColorDepth: 24, PixelDepth: 24, DevicePixelRatio: 1.0, Orientation: "landscape-primary"},
	}
}

func getBaseFonts() []string {
	return []string{
		"Arial", "Arial Black", "Comic Sans MS", "Courier New", "Georgia",
		"Impact", "Lucida Console", "Lucida Sans Unicode", "Palatino Linotype",
		"Tahoma", "Times New Roman", "Trebuchet MS", "Verdana",
	}
}

func getExtraFonts() []string {
	return []string{
		"Calibri", "Cambria", "Candara", "Consolas", "Constantia", "Corbel",
		"Franklin Gothic Medium", "Gabriola", "Garamond", "Helvetica",
		"Monaco", "MS Sans Serif", "MS Serif", "Segoe UI", "Symbol",
		"Webdings", "Wingdings", "Century Gothic", "Book Antiqua",
	}
}

// logEntropyFailure logs entropy generation failures for security monitoring
// This is critical for detecting entropy issues that could compromise anti-detection
func logEntropyFailure(context string, err error) {
	// Structure the security event for monitoring systems
	_ = map[string]interface{}{
		"event_type":   "entropy_failure",
		"component":    "fingerprint",
		"context":      context,
		"error":        err.Error(),
		"severity":     "critical",
		"timestamp":    time.Now().UTC(),
		"impact":       "fallback_to_static_values",
		"action_taken": "continued_operation_with_static_fallback",
	}

	// In production, this should trigger immediate alerts
	// Entropy failures are critical security events that need investigation
	// Implement proper logging infrastructure in production:
	// 1. Send to SIEM/security monitoring system
	// 2. Trigger alerts for security teams
	// 3. Consider automatic remediation if patterns are detected

	// Use structured logging for critical security events
	// In production, configure slog to send to security monitoring systems
	slog.Error("CRITICAL SECURITY EVENT: Entropy failure",
		slog.String("event_type", "entropy_failure"),
		slog.String("component", "fingerprint"),
		slog.String("context", context),
		slog.String("error", err.Error()),
		slog.String("severity", "critical"),
		slog.Time("timestamp", time.Now().UTC()),
		slog.String("impact", "fallback_to_static_values"),
		slog.String("action_taken", "continued_operation_with_static_fallback"),
	)
}
