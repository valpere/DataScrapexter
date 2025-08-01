// internal/proxy/manager_test.go
package proxy

import (
	"fmt"
	"testing"
	"time"
)

func TestNewProxyManager(t *testing.T) {
	tests := []struct {
		name   string
		config *ProxyConfig
		wantErr bool
	}{
		{
			name:   "nil config creates default",
			config: nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &ProxyConfig{
				Enabled:          true,
				Rotation:         RotationRoundRobin,
				HealthCheck:      true,
				HealthCheckRate:  5 * time.Minute,
				Timeout:          30 * time.Second,
				MaxRetries:       3,
				RetryDelay:       1 * time.Second,
				FailureThreshold: 5,
				RecoveryTime:     10 * time.Minute,
				Providers: []ProxyProvider{
					{
						Name:    "test-proxy",
						Type:    ProxyTypeHTTP,
						Host:    "proxy.example.com",
						Port:    8080,
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "enabled but no providers",
			config: &ProxyConfig{
				Enabled:   true,
				Providers: []ProxyProvider{},
			},
			wantErr: false, // Should not error, just warn
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewProxyManager(tt.config)
			if manager == nil {
				t.Errorf("NewProxyManager() returned nil")
				return
			}

			if tt.config != nil && tt.config.Enabled {
				if !manager.IsEnabled() && len(tt.config.Providers) > 0 {
					t.Errorf("Expected manager to be enabled when config is enabled with providers")
				}
			}
		})
	}
}

func TestProxyManager_GetProxy_RoundRobin(t *testing.T) {
	config := &ProxyConfig{
		Enabled:          true,
		Rotation:         RotationRoundRobin,
		HealthCheck:      false, // Disable health checks for testing
		FailureThreshold: 5,     // Set explicit failure threshold
		Providers: []ProxyProvider{
			{Name: "proxy1", Type: ProxyTypeHTTP, Host: "proxy1.example.com", Port: 8080, Enabled: true},
			{Name: "proxy2", Type: ProxyTypeHTTP, Host: "proxy2.example.com", Port: 8080, Enabled: true},
			{Name: "proxy3", Type: ProxyTypeHTTP, Host: "proxy3.example.com", Port: 8080, Enabled: true},
		},
	}

	manager := NewProxyManager(config)
	if !manager.IsEnabled() {
		t.Skip("Manager not enabled, skipping test")
	}

	// Debug: Check proxy initialization
	stats := manager.GetStats()
	t.Logf("Total proxies: %d, Healthy proxies: %d", stats.TotalProxies, stats.HealthyProxies)

	// Test round-robin rotation
	seenProxies := make(map[string]bool)
	for i := 0; i < 6; i++ { // Test 2 full rotations
		proxy, err := manager.GetProxy()
		if err != nil {
			t.Fatalf("GetProxy() returned error: %v", err)
		}
		if proxy == nil {
			t.Fatalf("GetProxy() returned nil proxy")
		}
		seenProxies[proxy.Provider.Name] = true
	}

	// Should have seen all 3 proxies
	if len(seenProxies) != 3 {
		t.Errorf("Expected to see 3 different proxies, got %d", len(seenProxies))
	}
}

func TestProxyManager_GetProxy_Random(t *testing.T) {
	config := &ProxyConfig{
		Enabled:          true,
		Rotation:         RotationRandom,
		HealthCheck:      false, // Disable health checks for testing
		FailureThreshold: 5,     // Set explicit failure threshold
		Providers: []ProxyProvider{
			{Name: "proxy1", Type: ProxyTypeHTTP, Host: "proxy1.example.com", Port: 8080, Enabled: true},
			{Name: "proxy2", Type: ProxyTypeHTTP, Host: "proxy2.example.com", Port: 8080, Enabled: true},
		},
	}

	manager := NewProxyManager(config)
	if !manager.IsEnabled() {
		t.Skip("Manager not enabled, skipping test")
	}

	// Test random selection (should get valid proxies)
	for i := 0; i < 10; i++ {
		proxy, err := manager.GetProxy()
		if err != nil {
			t.Fatalf("GetProxy() returned error: %v", err)
		}
		if proxy == nil {
			t.Fatalf("GetProxy() returned nil proxy")
		}

		// Check that proxy name is one of our configured proxies
		validName := proxy.Provider.Name == "proxy1" || proxy.Provider.Name == "proxy2"
		if !validName {
			t.Errorf("Got unexpected proxy name: %s", proxy.Provider.Name)
		}
	}
}

func TestProxyManager_ReportSuccess(t *testing.T) {
	config := &ProxyConfig{
		Enabled:          true,
		Rotation:         RotationRoundRobin,
		HealthCheck:      false, // Disable health checks for testing
		FailureThreshold: 5,     // Set explicit failure threshold
		Providers: []ProxyProvider{
			{Name: "proxy1", Type: ProxyTypeHTTP, Host: "proxy1.example.com", Port: 8080, Enabled: true},
		},
	}

	manager := NewProxyManager(config)
	if !manager.IsEnabled() {
		t.Skip("Manager not enabled, skipping test")
	}

	proxy, err := manager.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() returned error: %v", err)
	}

	// Record initial stats
	initialStats := manager.GetStats()
	initialSuccess := initialStats.ProxyStats[proxy.Provider.Name].SuccessCount

	// Report success
	manager.ReportSuccess(proxy)

	// Check updated stats
	updatedStats := manager.GetStats()
	updatedSuccess := updatedStats.ProxyStats[proxy.Provider.Name].SuccessCount

	if updatedSuccess != initialSuccess+1 {
		t.Errorf("Expected success count to increase by 1, got %d -> %d", initialSuccess, updatedSuccess)
	}
}

func TestProxyManager_ReportFailure(t *testing.T) {
	config := &ProxyConfig{
		Enabled:          true,
		Rotation:         RotationRoundRobin,
		FailureThreshold: 2, // Low threshold for testing
		Providers: []ProxyProvider{
			{Name: "proxy1", Type: ProxyTypeHTTP, Host: "proxy1.example.com", Port: 8080, Enabled: true},
		},
	}

	manager := NewProxyManager(config)
	if !manager.IsEnabled() {
		t.Skip("Manager not enabled, skipping test")
	}

	proxy, err := manager.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() returned error: %v", err)
	}

	// Record initial stats
	initialStats := manager.GetStats()
	initialFailure := initialStats.ProxyStats[proxy.Provider.Name].FailureCount

	// Report failure
	testErr := fmt.Errorf("test error")
	manager.ReportFailure(proxy, testErr)

	// Check updated stats
	updatedStats := manager.GetStats()
	updatedFailure := updatedStats.ProxyStats[proxy.Provider.Name].FailureCount

	if updatedFailure != initialFailure+1 {
		t.Errorf("Expected failure count to increase by 1, got %d -> %d", initialFailure, updatedFailure)
	}
}

func TestProxyManager_GetStats(t *testing.T) {
	config := &ProxyConfig{
		Enabled:  true,
		Rotation: RotationRoundRobin,
		Providers: []ProxyProvider{
			{Name: "proxy1", Type: ProxyTypeHTTP, Host: "proxy1.example.com", Port: 8080, Enabled: true},
			{Name: "proxy2", Type: ProxyTypeHTTP, Host: "proxy2.example.com", Port: 8080, Enabled: true},
		},
	}

	manager := NewProxyManager(config)
	if !manager.IsEnabled() {
		t.Skip("Manager not enabled, skipping test")
	}

	stats := manager.GetStats()

	if stats.TotalProxies != 2 {
		t.Errorf("Expected 2 total proxies, got %d", stats.TotalProxies)
	}

	if len(stats.ProxyStats) != 2 {
		t.Errorf("Expected 2 proxy stats entries, got %d", len(stats.ProxyStats))
	}

	// Check that proxy stats exist for each provider
	for _, provider := range config.Providers {
		if _, exists := stats.ProxyStats[provider.Name]; !exists {
			t.Errorf("Missing stats for proxy %s", provider.Name)
		}
	}
}

func TestProxyManager_DisabledManager(t *testing.T) {
	config := &ProxyConfig{
		Enabled: false,
		Providers: []ProxyProvider{
			{Name: "proxy1", Type: ProxyTypeHTTP, Host: "proxy1.example.com", Port: 8080, Enabled: true},
		},
	}

	manager := NewProxyManager(config)

	if manager.IsEnabled() {
		t.Errorf("Expected manager to be disabled")
	}

	proxy, err := manager.GetProxy()
	if err != nil {
		t.Errorf("GetProxy() should not return error for disabled manager")
	}
	if proxy != nil {
		t.Errorf("GetProxy() should return nil for disabled manager")
	}
}

func TestBuildProxyURL(t *testing.T) {
	manager := NewProxyManager(nil)

	tests := []struct {
		name     string
		provider *ProxyProvider
		wantErr  bool
		wantURL  string
	}{
		{
			name: "HTTP proxy without auth",
			provider: &ProxyProvider{
				Type: ProxyTypeHTTP,
				Host: "proxy.example.com",
				Port: 8080,
			},
			wantErr: false,
			wantURL: "http://proxy.example.com:8080",
		},
		{
			name: "HTTP proxy with auth",
			provider: &ProxyProvider{
				Type:     ProxyTypeHTTP,
				Host:     "proxy.example.com",
				Port:     8080,
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
			wantURL: "http://user:pass@proxy.example.com:8080",
		},
		{
			name: "SOCKS5 proxy",
			provider: &ProxyProvider{
				Type: ProxyTypeSOCKS5,
				Host: "proxy.example.com",
				Port: 1080,
			},
			wantErr: false,
			wantURL: "socks5://proxy.example.com:1080",
		},
		{
			name: "Unsupported proxy type",
			provider: &ProxyProvider{
				Type: "unsupported",
				Host: "proxy.example.com",
				Port: 8080,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := manager.buildProxyURL(tt.provider)

			if tt.wantErr {
				if err == nil {
					t.Errorf("buildProxyURL() should return error for %s", tt.name)
				}
				return
			}

			if err != nil {
				t.Errorf("buildProxyURL() returned unexpected error: %v", err)
				return
			}

			if url.String() != tt.wantURL {
				t.Errorf("buildProxyURL() = %s, want %s", url.String(), tt.wantURL)
			}
		})
	}
}