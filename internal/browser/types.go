// internal/browser/types.go
package browser

import (
	"context"
	"time"
)

// BrowserConfig defines browser automation configuration
type BrowserConfig struct {
	Enabled      bool          `yaml:"enabled" json:"enabled"`
	Headless     bool          `yaml:"headless" json:"headless"`
	UserDataDir  string        `yaml:"user_data_dir,omitempty" json:"user_data_dir,omitempty"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`
	ViewportWidth int          `yaml:"viewport_width" json:"viewport_width"`
	ViewportHeight int         `yaml:"viewport_height" json:"viewport_height"`
	WaitForElement string      `yaml:"wait_for_element,omitempty" json:"wait_for_element,omitempty"`
	WaitDelay    time.Duration `yaml:"wait_delay,omitempty" json:"wait_delay,omitempty"`
	UserAgent    string        `yaml:"user_agent,omitempty" json:"user_agent,omitempty"`
	DisableImages bool         `yaml:"disable_images" json:"disable_images"`
	DisableCSS   bool         `yaml:"disable_css" json:"disable_css"`
	DisableJS    bool         `yaml:"disable_js" json:"disable_js"`
}

// DefaultBrowserConfig returns default browser configuration
func DefaultBrowserConfig() *BrowserConfig {
	return &BrowserConfig{
		Enabled:        false,
		Headless:       true,
		Timeout:        30 * time.Second,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		WaitDelay:      2 * time.Second,
		DisableImages:  true, // Faster loading
		DisableCSS:     false,
		DisableJS:      false,
	}
}

// BrowserClient interface defines browser automation operations
type BrowserClient interface {
	// Navigate to a URL and wait for page load
	Navigate(ctx context.Context, url string) error

	// GetHTML returns the current page HTML
	GetHTML(ctx context.Context) (string, error)

	// WaitForElement waits for an element to appear
	WaitForElement(ctx context.Context, selector string, timeout time.Duration) error

	// ExecuteScript runs JavaScript code
	ExecuteScript(ctx context.Context, script string) (*interface{}, error)

	// Screenshot takes a screenshot of the page
	Screenshot(ctx context.Context) ([]byte, error)

	// SetViewport sets the browser viewport size
	SetViewport(ctx context.Context, width, height int) error

	// Close closes the browser
	Close() error
}

// Pool manages a pool of browser instances
type Pool interface {
	// Get retrieves a browser from the pool
	Get(ctx context.Context) (BrowserClient, error)

	// Put returns a browser to the pool
	Put(browser BrowserClient) error

	// Close closes all browsers in the pool
	Close() error

	// Size returns the current pool size
	Size() int
}

// BrowserStats contains browser automation statistics
type BrowserStats struct {
	PagesLoaded    int           `json:"pages_loaded"`
	AverageLoadTime time.Duration `json:"average_load_time"`
	Errors         int           `json:"errors"`
	JavaScriptErrors int         `json:"javascript_errors"`
	TimeoutsOccurred int         `json:"timeouts_occurred"`
}