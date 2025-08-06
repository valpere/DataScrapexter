// internal/antidetect/advanced.go - Advanced anti-detection mechanisms
package antidetect

import (
	"crypto/rand"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTP2Fingerprint represents HTTP/2 fingerprinting characteristics
type HTTP2Fingerprint struct {
	WindowUpdate     uint32
	HeaderTableSize  uint32
	EnablePush       bool
	MaxConcurrent    uint32
	InitialWindow    uint32
	MaxFrameSize     uint32
	MaxHeaderListSize uint32
	PriorityWeights  []int
	StreamDependency map[int]int
}

// HTTP2Fingerprinter provides HTTP/2 fingerprinting evasion
type HTTP2Fingerprinter struct {
	profiles []HTTP2Fingerprint
}

// NewHTTP2Fingerprinter creates a new HTTP/2 fingerprinter
func NewHTTP2Fingerprinter() *HTTP2Fingerprinter {
	return &HTTP2Fingerprinter{
		profiles: getHTTP2Profiles(),
	}
}

// GetRandomProfile returns a random HTTP/2 profile
func (h2 *HTTP2Fingerprinter) GetRandomProfile() HTTP2Fingerprint {
	return h2.profiles[rand.Intn(len(h2.profiles))]
}

// TimingFingerprint represents request timing characteristics
type TimingFingerprint struct {
	RequestDelay     time.Duration
	MouseMovements   []MouseEvent
	KeyboardEvents   []KeyEvent
	ScrollPatterns   []ScrollEvent
	IdleTime         time.Duration
	TabSwitchDelay   time.Duration
	WindowResizeDelay time.Duration
}

// MouseEvent represents a mouse movement/click event
type MouseEvent struct {
	X         int
	Y         int
	Timestamp time.Time
	EventType string // move, click, scroll
	Button    int    // 0=left, 1=middle, 2=right
}

// KeyEvent represents a keyboard event
type KeyEvent struct {
	Key       string
	Timestamp time.Time
	EventType string // keydown, keyup
	Modifiers []string // ctrl, shift, alt
}

// ScrollEvent represents a scroll event
type ScrollEvent struct {
	DeltaX    int
	DeltaY    int
	Timestamp time.Time
	Mode      string // wheel, touch, key
}

// BehaviorSimulator simulates human-like browsing behavior
type BehaviorSimulator struct {
	profiles []TimingFingerprint
	current  *TimingFingerprint
}

// NewBehaviorSimulator creates a new behavior simulator
func NewBehaviorSimulator() *BehaviorSimulator {
	return &BehaviorSimulator{
		profiles: generateBehaviorProfiles(),
	}
}

// GenerateMousePath generates realistic mouse movements
func (bs *BehaviorSimulator) GenerateMousePath(startX, startY, endX, endY int) []MouseEvent {
	events := []MouseEvent{}
	startTime := time.Now()
	
	// Calculate distance and generate realistic movement
	distance := calculateDistance(startX, startY, endX, endY)
	steps := int(distance / 10) + rand.Intn(5) + 3 // 3-7 extra steps
	
	for i := 0; i < steps; i++ {
		progress := float64(i) / float64(steps)
		
		// Add some randomness to the path
		noise := rand.Float64()*10 - 5
		x := int(float64(startX) + (float64(endX-startX) * progress) + noise)
		y := int(float64(startY) + (float64(endY-startY) * progress) + noise)
		
		// Human-like timing with micro-pauses
		delay := time.Duration(rand.Intn(20)+10) * time.Millisecond
		
		events = append(events, MouseEvent{
			X:         x,
			Y:         y,
			Timestamp: startTime.Add(delay * time.Duration(i)),
			EventType: "move",
		})
	}
	
	// Final click
	events = append(events, MouseEvent{
		X:         endX,
		Y:         endY,
		Timestamp: startTime.Add(time.Duration(len(events)) * 15 * time.Millisecond),
		EventType: "click",
		Button:    0,
	})
	
	return events
}

// GenerateTypingPattern generates realistic typing patterns
func (bs *BehaviorSimulator) GenerateTypingPattern(text string) []KeyEvent {
	events := []KeyEvent{}
	startTime := time.Now()
	
	for i, char := range text {
		// Variable typing speed (150-350ms between keys)
		delay := time.Duration(rand.Intn(200)+150) * time.Millisecond
		
		// Occasional longer pauses (thinking)
		if rand.Float64() < 0.1 {
			delay += time.Duration(rand.Intn(1000)+500) * time.Millisecond
		}
		
		timestamp := startTime.Add(delay * time.Duration(i))
		
		events = append(events, KeyEvent{
			Key:       string(char),
			Timestamp: timestamp,
			EventType: "keydown",
		})
		
		// Key release slightly after press
		events = append(events, KeyEvent{
			Key:       string(char),
			Timestamp: timestamp.Add(time.Duration(rand.Intn(50)+20) * time.Millisecond),
			EventType: "keyup",
		})
	}
	
	return events
}

// RequestOrderingFingerprint represents the order and timing of resource requests
type RequestOrderingFingerprint struct {
	ResourceTypes    []string
	RequestIntervals []time.Duration
	ParallelRequests int
	CacheHeaders     map[string]string
	CompressionTypes []string
}

// ResourceRequestSimulator simulates realistic resource loading patterns
type ResourceRequestSimulator struct {
	patterns []RequestOrderingFingerprint
}

// NewResourceRequestSimulator creates a new resource request simulator
func NewResourceRequestSimulator() *ResourceRequestSimulator {
	return &ResourceRequestSimulator{
		patterns: getResourceLoadingPatterns(),
	}
}

// GenerateResourceLoadingPattern generates a realistic resource loading sequence
func (rrs *ResourceRequestSimulator) GenerateResourceLoadingPattern() RequestOrderingFingerprint {
	patterns := []RequestOrderingFingerprint{
		{
			ResourceTypes: []string{"document", "stylesheet", "script", "image", "font", "xhr"},
			RequestIntervals: []time.Duration{
				0 * time.Millisecond,    // document
				50 * time.Millisecond,   // CSS
				100 * time.Millisecond,  // JS
				200 * time.Millisecond,  // images
				300 * time.Millisecond,  // fonts
				500 * time.Millisecond,  // XHR
			},
			ParallelRequests: 6,
			CacheHeaders: map[string]string{
				"Cache-Control": "max-age=3600",
				"If-None-Match": generateETag(),
			},
			CompressionTypes: []string{"gzip", "br"},
		},
	}
	
	return patterns[rand.Intn(len(patterns))]
}

// DNSFingerprintEvasion provides DNS fingerprinting evasion
type DNSFingerprintEvasion struct {
	dnsServers []string
	queryTypes []string
	ttlRanges  map[string][2]int
}

// NewDNSFingerprintEvasion creates DNS fingerprinting evasion
func NewDNSFingerprintEvasion() *DNSFingerprintEvasion {
	return &DNSFingerprintEvasion{
		dnsServers: []string{
			"8.8.8.8",     // Google
			"1.1.1.1",     // Cloudflare
			"208.67.222.222", // OpenDNS
			"9.9.9.9",     // Quad9
		},
		queryTypes: []string{"A", "AAAA", "CNAME", "MX"},
		ttlRanges: map[string][2]int{
			"A":     {300, 3600},
			"AAAA":  {300, 3600},
			"CNAME": {3600, 86400},
			"MX":    {3600, 86400},
		},
	}
}

// HeaderOrderingEvasion provides HTTP header ordering evasion
type HeaderOrderingEvasion struct {
	browserProfiles map[string][]string
}

// NewHeaderOrderingEvasion creates header ordering evasion
func NewHeaderOrderingEvasion() *HeaderOrderingEvasion {
	return &HeaderOrderingEvasion{
		browserProfiles: map[string][]string{
			"chrome": {
				"Host", "Connection", "Cache-Control", "sec-ch-ua", "sec-ch-ua-mobile",
				"sec-ch-ua-platform", "Upgrade-Insecure-Requests", "User-Agent", "Accept",
				"Sec-Fetch-Site", "Sec-Fetch-Mode", "Sec-Fetch-User", "Sec-Fetch-Dest",
				"Accept-Encoding", "Accept-Language",
			},
			"firefox": {
				"Host", "User-Agent", "Accept", "Accept-Language", "Accept-Encoding",
				"Connection", "Upgrade-Insecure-Requests", "Sec-Fetch-Dest", "Sec-Fetch-Mode",
				"Sec-Fetch-Site", "Cache-Control",
			},
			"safari": {
				"Host", "Accept", "User-Agent", "Accept-Language", "Accept-Encoding",
				"Connection", "Upgrade-Insecure-Requests",
			},
		},
	}
}

// ApplyHeaderOrdering applies browser-specific header ordering
func (hoe *HeaderOrderingEvasion) ApplyHeaderOrdering(req *http.Request, browser string) {
	if ordering, exists := hoe.browserProfiles[browser]; exists {
		// Create new header with proper ordering
		newHeader := http.Header{}
		
		// Add headers in browser-specific order
		for _, headerName := range ordering {
			if values := req.Header[headerName]; len(values) > 0 {
				newHeader[headerName] = values
			}
		}
		
		// Add any remaining headers not in the profile
		for name, values := range req.Header {
			if newHeader.Get(name) == "" {
				newHeader[name] = values
			}
		}
		
		req.Header = newHeader
	}
}

// ConnectionFingerprintEvasion provides TCP connection fingerprinting evasion
type ConnectionFingerprintEvasion struct {
	windowSizes    []int
	tcpOptions     map[string][]byte
	keepAliveProbe int
}

// NewConnectionFingerprintEvasion creates connection fingerprinting evasion
func NewConnectionFingerprintEvasion() *ConnectionFingerprintEvasion {
	return &ConnectionFingerprintEvasion{
		windowSizes: []int{65535, 32768, 16384, 8192},
		tcpOptions: map[string][]byte{
			"chrome":  {0x02, 0x04, 0x05, 0xb4, 0x04, 0x02, 0x08, 0x0a},
			"firefox": {0x02, 0x04, 0x05, 0xb4, 0x01, 0x03, 0x03, 0x06},
			"safari":  {0x02, 0x04, 0x05, 0xb4, 0x01, 0x01, 0x04, 0x02},
		},
		keepAliveProbe: 7200, // 2 hours
	}
}

// AdvancedAntiDetection combines all advanced anti-detection mechanisms
type AdvancedAntiDetection struct {
	HTTP2Fingerprinter        *HTTP2Fingerprinter
	BehaviorSimulator         *BehaviorSimulator
	ResourceRequestSimulator  *ResourceRequestSimulator
	DNSEvasion               *DNSFingerprintEvasion
	HeaderOrdering           *HeaderOrderingEvasion
	ConnectionEvasion        *ConnectionFingerprintEvasion
	FingerprintingEvader     *FingerprintingEvader
	TLSFingerprinter         *TLSFingerprinter
}

// NewAdvancedAntiDetection creates a comprehensive anti-detection system
func NewAdvancedAntiDetection(enabled bool) *AdvancedAntiDetection {
	return &AdvancedAntiDetection{
		HTTP2Fingerprinter:       NewHTTP2Fingerprinter(),
		BehaviorSimulator:        NewBehaviorSimulator(),
		ResourceRequestSimulator: NewResourceRequestSimulator(),
		DNSEvasion:              NewDNSFingerprintEvasion(),
		HeaderOrdering:          NewHeaderOrderingEvasion(),
		ConnectionEvasion:       NewConnectionFingerprintEvasion(),
		FingerprintingEvader:    NewFingerprintingEvader(enabled),
		TLSFingerprinter:        NewTLSFingerprinter(),
	}
}

// ApplyAllEvasions applies all available evasion techniques to a request
func (aad *AdvancedAntiDetection) ApplyAllEvasions(req *http.Request, browser string) {
	// Apply header ordering
	aad.HeaderOrdering.ApplyHeaderOrdering(req, browser)
	
	// Add browser-specific headers that are often checked
	aad.addBrowserSpecificHeaders(req, browser)
	
	// Add realistic cache headers
	aad.addCacheHeaders(req)
	
	// Add security headers that browsers send
	aad.addSecurityHeaders(req)
}

// Helper functions

func getHTTP2Profiles() []HTTP2Fingerprint {
	return []HTTP2Fingerprint{
		// Chrome-like HTTP/2
		{
			WindowUpdate:      65535,
			HeaderTableSize:   65536,
			EnablePush:        true,
			MaxConcurrent:     1000,
			InitialWindow:     6291456,
			MaxFrameSize:      16384,
			MaxHeaderListSize: 262144,
			PriorityWeights:   []int{256, 220, 183, 147, 110, 74, 37},
		},
		// Firefox-like HTTP/2
		{
			WindowUpdate:      32768,
			HeaderTableSize:   65536,
			EnablePush:        true,
			MaxConcurrent:     1000,
			InitialWindow:     12517376,
			MaxFrameSize:      16384,
			MaxHeaderListSize: 262144,
			PriorityWeights:   []int{201, 101, 1, 1, 1, 1, 1},
		},
	}
}

func generateBehaviorProfiles() []TimingFingerprint {
	return []TimingFingerprint{
		{
			RequestDelay:      time.Duration(rand.Intn(2000)+500) * time.Millisecond,
			IdleTime:         time.Duration(rand.Intn(30000)+5000) * time.Millisecond,
			TabSwitchDelay:   time.Duration(rand.Intn(1000)+200) * time.Millisecond,
			WindowResizeDelay: time.Duration(rand.Intn(500)+100) * time.Millisecond,
		},
	}
}

func getResourceLoadingPatterns() []RequestOrderingFingerprint {
	return []RequestOrderingFingerprint{
		{
			ResourceTypes:    []string{"document", "stylesheet", "script", "image"},
			RequestIntervals: []time.Duration{0, 50 * time.Millisecond, 100 * time.Millisecond, 200 * time.Millisecond},
			ParallelRequests: 6,
			CacheHeaders:     map[string]string{"Cache-Control": "max-age=3600"},
			CompressionTypes: []string{"gzip", "br"},
		},
	}
}

func (aad *AdvancedAntiDetection) addBrowserSpecificHeaders(req *http.Request, browser string) {
	switch browser {
	case "chrome":
		req.Header.Set("sec-ch-ua", `" Not A;Brand";v="99", "Chromium";v="96", "Google Chrome";v="96"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Sec-Fetch-Dest", "document")
	case "firefox":
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-User", "?1")
	case "safari":
		// Safari has fewer sec- headers
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
	}
}

func (aad *AdvancedAntiDetection) addCacheHeaders(req *http.Request) {
	// Add realistic cache control
	if req.Header.Get("Cache-Control") == "" {
		req.Header.Set("Cache-Control", "max-age=0")
	}
	
	// Add If-Modified-Since for repeat requests
	if rand.Float64() < 0.3 {
		pastTime := time.Now().Add(-time.Duration(rand.Intn(86400)) * time.Second)
		req.Header.Set("If-Modified-Since", pastTime.Format(http.TimeFormat))
	}
}

func (aad *AdvancedAntiDetection) addSecurityHeaders(req *http.Request) {
	// Add DNT (Do Not Track) header occasionally
	if rand.Float64() < 0.2 {
		req.Header.Set("DNT", "1")
	}
	
	// Add Upgrade-Insecure-Requests
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

func calculateDistance(x1, y1, x2, y2 int) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return (dx*dx + dy*dy) // Skip sqrt for performance
}

func generateETag() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return `"` + fmt.Sprintf("%x", bytes) + `"`
}

// GetRandomDelay returns a human-like delay between requests
func (bs *BehaviorSimulator) GetRandomDelay() time.Duration {
	// Most requests: 0.5-3 seconds
	if rand.Float64() < 0.8 {
		return time.Duration(rand.Intn(2500)+500) * time.Millisecond
	}
	
	// Occasional longer delays: 3-10 seconds
	if rand.Float64() < 0.9 {
		return time.Duration(rand.Intn(7000)+3000) * time.Millisecond
	}
	
	// Rare very long delays: 10-30 seconds (user distraction)
	return time.Duration(rand.Intn(20000)+10000) * time.Millisecond
}

// SimulatePageInteraction simulates realistic page interaction delays
func (bs *BehaviorSimulator) SimulatePageInteraction() time.Duration {
	// Reading time varies by content length (estimated)
	baseReadingTime := time.Duration(rand.Intn(5000)+2000) * time.Millisecond
	
	// Add interaction time (scrolling, clicking)
	interactionTime := time.Duration(rand.Intn(3000)+1000) * time.Millisecond
	
	return baseReadingTime + interactionTime
}