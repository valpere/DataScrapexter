// internal/browser/chromedp.go
package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
)

// ChromeClient implements BrowserClient using chromedp
type ChromeClient struct {
	ctx               context.Context
	cancel            context.CancelFunc
	config            *BrowserConfig
	stats             *BrowserStats
	navigationSuccess bool
	navMu             sync.RWMutex
}

// NewChromeClient creates a new Chrome browser client
func NewChromeClient(config *BrowserConfig) (*ChromeClient, error) {
	if config == nil {
		config = DefaultBrowserConfig()
	}

	// Set up Chrome options
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
		chromedp.NoSandbox, // Required for Docker environments
	}

	// Add headless mode
	if config.Headless {
		opts = append(opts, chromedp.Headless)
	}

	// Add user data directory
	if config.UserDataDir != "" {
		opts = append(opts, chromedp.UserDataDir(config.UserDataDir))
	}

	// Add user agent
	if config.UserAgent != "" {
		opts = append(opts, chromedp.UserAgent(config.UserAgent))
	}

	// Disable images for faster loading
	if config.DisableImages {
		opts = append(opts, chromedp.Flag("blink-settings", "imagesEnabled=false"))
	}

	// Create allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Create context with timeout
	ctx, cancel := chromedp.NewContext(allocCtx)
	if config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
	}

	client := &ChromeClient{
		ctx:    ctx,
		cancel: cancel,
		config: config,
		stats:  &BrowserStats{},
	}
	
	// Initialize navigation state with proper synchronization
	client.navMu.Lock()
	client.navigationSuccess = false
	client.navMu.Unlock()

	// Initialize browser with viewport
	if err := client.initialize(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	return client, nil
}

// initialize sets up the browser with initial configuration
func (c *ChromeClient) initialize() error {
	tasks := []chromedp.Action{
		chromedp.EmulateViewport(int64(c.config.ViewportWidth), int64(c.config.ViewportHeight)),
	}

	// Add mobile emulation if needed (could be configurable)
	if c.config.ViewportWidth < 768 {
		tasks = append(tasks, chromedp.Emulate(device.IPhone8))
	}

	return chromedp.Run(c.ctx, tasks...)
}

// Navigate navigates to a URL and waits for page load
func (c *ChromeClient) Navigate(ctx context.Context, url string) error {
	start := time.Now()
	
	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	}

	// Add custom wait for element if specified
	if c.config.WaitForElement != "" {
		tasks = append(tasks, chromedp.WaitVisible(c.config.WaitForElement))
	}

	// Add wait delay if specified
	if c.config.WaitDelay > 0 {
		tasks = append(tasks, chromedp.Sleep(c.config.WaitDelay))
	}

	err := chromedp.Run(c.ctx, tasks...)
	loadTime := time.Since(start)
	
	if err != nil {
		c.stats.Errors++
		c.navMu.Lock()
		c.navigationSuccess = false
		c.navMu.Unlock()
		return fmt.Errorf("navigation failed: %w", err)
	}

	// Update stats and state only after successful navigation
	c.navMu.Lock()
	c.navigationSuccess = true
	c.navMu.Unlock()
	c.stats.PagesLoaded++
	if c.stats.PagesLoaded == 1 {
		c.stats.AverageLoadTime = loadTime
	} else {
		c.stats.AverageLoadTime = (c.stats.AverageLoadTime + loadTime) / 2
	}

	return nil
}

// GetHTML returns the current page HTML
func (c *ChromeClient) GetHTML(ctx context.Context) (string, error) {
	c.navMu.RLock()
	navSuccess := c.navigationSuccess
	c.navMu.RUnlock()
	
	if !navSuccess {
		return "", fmt.Errorf("cannot extract HTML: navigation has not completed successfully")
	}
	
	var html string
	err := chromedp.Run(c.ctx, chromedp.OuterHTML("html", &html))
	if err != nil {
		c.stats.Errors++
		return "", fmt.Errorf("failed to get HTML: %w", err)
	}
	return html, nil
}

// WaitForElement waits for an element to appear
func (c *ChromeClient) WaitForElement(ctx context.Context, selector string, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	err := chromedp.Run(timeoutCtx, chromedp.WaitVisible(selector))
	if err != nil {
		c.stats.TimeoutsOccurred++
		return fmt.Errorf("element wait timeout: %w", err)
	}
	return nil
}

// ExecuteScript runs JavaScript code
func (c *ChromeClient) ExecuteScript(ctx context.Context, script string) (*interface{}, error) {
	var result interface{}
	err := chromedp.Run(c.ctx, chromedp.Evaluate(script, &result))
	if err != nil {
		c.stats.JavaScriptErrors++
		return nil, fmt.Errorf("script execution failed: %w", err)
	}
	return &result, nil
}

// Screenshot takes a screenshot of the page
func (c *ChromeClient) Screenshot(ctx context.Context) ([]byte, error) {
	var buf []byte
	err := chromedp.Run(c.ctx, chromedp.FullScreenshot(&buf, 90))
	if err != nil {
		c.stats.Errors++
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}
	return buf, nil
}

// SetViewport sets the browser viewport size
func (c *ChromeClient) SetViewport(ctx context.Context, width, height int) error {
	err := chromedp.Run(c.ctx, chromedp.EmulateViewport(int64(width), int64(height)))
	if err != nil {
		return fmt.Errorf("viewport change failed: %w", err)
	}
	
	c.config.ViewportWidth = width
	c.config.ViewportHeight = height
	return nil
}

// GetStats returns browser statistics
func (c *ChromeClient) GetStats() *BrowserStats {
	return c.stats
}

// Close closes the browser
func (c *ChromeClient) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

// BrowserManager manages browser instances and provides high-level operations
type BrowserManager struct {
	config *BrowserConfig
	client BrowserClient
}

// NewBrowserManager creates a new browser manager
func NewBrowserManager(config *BrowserConfig) (*BrowserManager, error) {
	if config == nil {
		config = DefaultBrowserConfig()
	}

	// Only create browser client if browser is enabled
	var client BrowserClient
	var err error
	
	if config.Enabled {
		client, err = NewChromeClient(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create browser client: %w", err)
		}
	}

	return &BrowserManager{
		config: config,
		client: client,
	}, nil
}

// IsEnabled returns whether browser automation is enabled
func (bm *BrowserManager) IsEnabled() bool {
	return bm.config.Enabled && bm.client != nil
}

// FetchHTML fetches HTML using browser automation
func (bm *BrowserManager) FetchHTML(ctx context.Context, url string) (string, error) {
	if !bm.IsEnabled() {
		return "", fmt.Errorf("browser automation is not enabled")
	}

	err := bm.client.Navigate(ctx, url)
	if err != nil {
		return "", fmt.Errorf("navigation failed: %w", err)
	}

	html, err := bm.client.GetHTML(ctx)
	if err != nil {
		return "", fmt.Errorf("HTML extraction failed: %w", err)
	}

	return html, nil
}

// ExecuteJavaScript executes JavaScript in the browser
func (bm *BrowserManager) ExecuteJavaScript(ctx context.Context, script string) (*interface{}, error) {
	if !bm.IsEnabled() {
		return nil, fmt.Errorf("browser automation is not enabled")
	}

	return bm.client.ExecuteScript(ctx, script)
}

// WaitForElement waits for an element to appear
func (bm *BrowserManager) WaitForElement(ctx context.Context, selector string) error {
	if !bm.IsEnabled() {
		return fmt.Errorf("browser automation is not enabled")
	}

	timeout := bm.config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return bm.client.WaitForElement(ctx, selector, timeout)
}

// TakeScreenshot takes a screenshot
func (bm *BrowserManager) TakeScreenshot(ctx context.Context) ([]byte, error) {
	if !bm.IsEnabled() {
		return nil, fmt.Errorf("browser automation is not enabled")
	}

	return bm.client.Screenshot(ctx)
}

// Close closes the browser manager
func (bm *BrowserManager) Close() error {
	if bm.client != nil {
		return bm.client.Close()
	}
	return nil
}