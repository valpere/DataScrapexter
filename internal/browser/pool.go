// internal/browser/pool.go
package browser

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BrowserPool implements the Pool interface
type BrowserPool struct {
	config     *BrowserConfig
	browsers   chan BrowserClient
	maxSize    int
	currentSize int
	mu         sync.RWMutex
	closed     bool
}

// NewBrowserPool creates a new browser pool
func NewBrowserPool(config *BrowserConfig, maxSize int) (*BrowserPool, error) {
	if config == nil {
		config = DefaultBrowserConfig()
	}

	if maxSize <= 0 {
		maxSize = 5 // Default pool size
	}

	pool := &BrowserPool{
		config:   config,
		browsers: make(chan BrowserClient, maxSize),
		maxSize:  maxSize,
	}

	return pool, nil
}

// Get retrieves a browser from the pool or creates a new one
func (p *BrowserPool) Get(ctx context.Context) (BrowserClient, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	select {
	case browser := <-p.browsers:
		return browser, nil
	default:
		// No available browser in pool, create new one if under limit
		p.mu.Lock()
		defer p.mu.Unlock()

		if p.currentSize < p.maxSize {
			browser, err := NewChromeClient(p.config)
			if err != nil {
				return nil, fmt.Errorf("failed to create browser: %w", err)
			}
			p.currentSize++
			return browser, nil
		}

		// Wait for available browser (with timeout)
		select {
		case browser := <-p.browsers:
			return browser, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(30 * time.Second):
			return nil, fmt.Errorf("timeout waiting for available browser")
		}
	}
}

// Put returns a browser to the pool
func (p *BrowserPool) Put(browser BrowserClient) error {
	if browser == nil {
		return fmt.Errorf("cannot put nil browser in pool")
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		browser.Close()
		return fmt.Errorf("pool is closed")
	}

	select {
	case p.browsers <- browser:
		return nil
	default:
		// Pool is full, close the browser
		browser.Close()
		p.mu.Lock()
		defer p.mu.Unlock()
		p.currentSize--
		return nil
	}
}

// Size returns the current number of browsers in the pool
func (p *BrowserPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.browsers)
}

// TotalSize returns the total number of browsers created
func (p *BrowserPool) TotalSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentSize
}

// Close closes all browsers in the pool
func (p *BrowserPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Close all browsers in the pool
	close(p.browsers)
	for browser := range p.browsers {
		browser.Close()
	}

	p.currentSize = 0
	return nil
}

// PooledBrowserManager manages browser operations using a pool
type PooledBrowserManager struct {
	pool   *BrowserPool
	config *BrowserConfig
}

// NewPooledBrowserManager creates a browser manager with pooling
func NewPooledBrowserManager(config *BrowserConfig, poolSize int) (*PooledBrowserManager, error) {
	if config == nil {
		config = DefaultBrowserConfig()
	}

	pool, err := NewBrowserPool(config, poolSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser pool: %w", err)
	}

	return &PooledBrowserManager{
		pool:   pool,
		config: config,
	}, nil
}

// FetchHTML fetches HTML using a pooled browser
func (pbm *PooledBrowserManager) FetchHTML(ctx context.Context, url string) (string, error) {
	if !pbm.config.Enabled {
		return "", fmt.Errorf("browser automation is not enabled")
	}

	browser, err := pbm.pool.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get browser from pool: %w", err)
	}
	defer pbm.pool.Put(browser)

	err = browser.Navigate(ctx, url)
	if err != nil {
		return "", fmt.Errorf("navigation failed: %w", err)
	}

	html, err := browser.GetHTML(ctx)
	if err != nil {
		return "", fmt.Errorf("HTML extraction failed: %w", err)
	}

	return html, nil
}

// ExecuteJavaScript executes JavaScript using a pooled browser
func (pbm *PooledBrowserManager) ExecuteJavaScript(ctx context.Context, url, script string) (*interface{}, error) {
	if !pbm.config.Enabled {
		return nil, fmt.Errorf("browser automation is not enabled")
	}

	browser, err := pbm.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser from pool: %w", err)
	}
	defer pbm.pool.Put(browser)

	err = browser.Navigate(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	return browser.ExecuteScript(ctx, script)
}

// IsEnabled returns whether browser automation is enabled
func (pbm *PooledBrowserManager) IsEnabled() bool {
	return pbm.config.Enabled
}

// GetPoolStats returns pool statistics
func (pbm *PooledBrowserManager) GetPoolStats() map[string]interface{} {
	return map[string]interface{}{
		"available_browsers": pbm.pool.Size(),
		"total_browsers":     pbm.pool.TotalSize(),
		"max_pool_size":      pbm.pool.maxSize,
		"pool_closed":        pbm.pool.closed,
	}
}

// Close closes the pooled browser manager
func (pbm *PooledBrowserManager) Close() error {
	return pbm.pool.Close()
}