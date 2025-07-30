// internal/browser/browser_test.go
package browser

import (
	"context"
	"testing"
	"time"
)

func TestDefaultBrowserConfig(t *testing.T) {
	config := DefaultBrowserConfig()
	
	if config == nil {
		t.Fatal("Expected non-nil config")
	}
	
	if config.Enabled {
		t.Error("Expected browser to be disabled by default")
	}
	
	if !config.Headless {
		t.Error("Expected headless mode by default")
	}
	
	if config.ViewportWidth != 1920 {
		t.Errorf("Expected viewport width 1920, got %d", config.ViewportWidth)
	}
	
	if config.ViewportHeight != 1080 {
		t.Errorf("Expected viewport height 1080, got %d", config.ViewportHeight)
	}
}

func TestBrowserManager_Disabled(t *testing.T) {
	config := &BrowserConfig{
		Enabled: false,
	}
	
	manager, err := NewBrowserManager(config)
	if err != nil {
		t.Fatalf("Failed to create browser manager: %v", err)
	}
	defer manager.Close()
	
	if manager.IsEnabled() {
		t.Error("Expected browser manager to be disabled")
	}
	
	ctx := context.Background()
	_, err = manager.FetchHTML(ctx, "https://example.com")
	if err == nil {
		t.Error("Expected error when browser is disabled")
	}
}

func TestBrowserManager_Enabled(t *testing.T) {
	// Skip this test if we don't have Chrome installed
	config := &BrowserConfig{
		Enabled:   true,
		Headless:  true,
		Timeout:   10 * time.Second,
		WaitDelay: 1 * time.Second,
	}
	
	manager, err := NewBrowserManager(config)
	if err != nil {
		t.Skipf("Skipping browser test - Chrome may not be available: %v", err)
	}
	defer manager.Close()
	
	if !manager.IsEnabled() {
		t.Error("Expected browser manager to be enabled")
	}
	
	// Test with a simple page that doesn't require JavaScript
	ctx := context.Background()
	html, err := manager.FetchHTML(ctx, "data:text/html,<html><body><h1>Test</h1></body></html>")
	if err != nil {
		t.Fatalf("Failed to fetch HTML: %v", err)
	}
	
	if html == "" {
		t.Error("Expected non-empty HTML")
	}
	
	if !contains(html, "<h1>Test</h1>") {
		t.Error("Expected HTML to contain test content")
	}
}

func TestBrowserPool(t *testing.T) {
	config := DefaultBrowserConfig()
	config.Enabled = true
	config.Headless = true
	
	pool, err := NewBrowserPool(config, 2)
	if err != nil {
		t.Fatalf("Failed to create browser pool: %v", err)
	}
	defer pool.Close()
	
	if pool.Size() != 0 {
		t.Errorf("Expected empty pool, got size %d", pool.Size())
	}
	
	if pool.TotalSize() != 0 {
		t.Errorf("Expected total size 0, got %d", pool.TotalSize())
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}