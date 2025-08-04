// internal/config/config.go
package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

// ScraperConfig represents the complete configuration for a scraping job
type ScraperConfig struct {
	Name       string            `yaml:"name" json:"name"`
	BaseURL    string            `yaml:"base_url" json:"base_url"`
	URLs       []string          `yaml:"urls,omitempty" json:"urls,omitempty"`
	UserAgents []string          `yaml:"user_agents,omitempty" json:"user_agents,omitempty"`
	RateLimit  string            `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	Timeout    string            `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	MaxRetries int               `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	Retries    int               `yaml:"retries,omitempty" json:"retries,omitempty"` // Added missing field
	Headers    map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Cookies    map[string]string `yaml:"cookies,omitempty" json:"cookies,omitempty"`
	Proxy      *ProxyConfig      `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Browser    *BrowserConfig    `yaml:"browser,omitempty" json:"browser,omitempty"`
	Fields     []Field           `yaml:"fields" json:"fields"`
	Pagination *PaginationConfig `yaml:"pagination,omitempty" json:"pagination,omitempty"`
	Output     OutputConfig      `yaml:"output" json:"output"`
}

// Field represents a single field to extract
type Field struct {
	Name      string          `yaml:"name" json:"name"`
	Selector  string          `yaml:"selector" json:"selector"`
	Type      string          `yaml:"type" json:"type"`
	Required  bool            `yaml:"required,omitempty" json:"required,omitempty"`
	Attribute string          `yaml:"attribute,omitempty" json:"attribute,omitempty"`
	Default   interface{}     `yaml:"default,omitempty" json:"default,omitempty"`
	Transform []TransformRule `yaml:"transform,omitempty" json:"transform,omitempty"`
}

// FieldConfig is an alias for Field to maintain backward compatibility
type FieldConfig = Field

// PaginationConfig represents pagination configuration
type PaginationConfig struct {
	Type       string `yaml:"type" json:"type"`
	Selector   string `yaml:"selector,omitempty" json:"selector,omitempty"`
	MaxPages   int    `yaml:"max_pages,omitempty" json:"max_pages,omitempty"`
	URLPattern string `yaml:"url_pattern,omitempty" json:"url_pattern,omitempty"`
	StartPage  int    `yaml:"start_page,omitempty" json:"start_page,omitempty"`
}

// OutputConfig represents output configuration
type OutputConfig struct {
	Format        string `yaml:"format" json:"format"`
	File          string `yaml:"file" json:"file"`
	EnableMetrics bool   `yaml:"enable_metrics,omitempty" json:"enable_metrics,omitempty"`
}

// ProxyConfig represents proxy configuration
type ProxyConfig struct {
	Enabled          bool            `yaml:"enabled" json:"enabled"`
	Rotation         string          `yaml:"rotation,omitempty" json:"rotation,omitempty"`
	HealthCheck      bool            `yaml:"health_check,omitempty" json:"health_check,omitempty"`
	HealthCheckURL   string          `yaml:"health_check_url,omitempty" json:"health_check_url,omitempty"`
	HealthCheckRate  string          `yaml:"health_check_rate,omitempty" json:"health_check_rate,omitempty"`
	Timeout          string          `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	MaxRetries       int             `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	RetryDelay       string          `yaml:"retry_delay,omitempty" json:"retry_delay,omitempty"`
	Providers        []ProxyProvider `yaml:"providers,omitempty" json:"providers,omitempty"`
	FailureThreshold int             `yaml:"failure_threshold,omitempty" json:"failure_threshold,omitempty"`
	RecoveryTime     string          `yaml:"recovery_time,omitempty" json:"recovery_time,omitempty"`
	TLS              *TLSConfig      `yaml:"tls,omitempty" json:"tls,omitempty"`

	// Legacy support for single proxy URL
	URL      string `yaml:"url,omitempty" json:"url,omitempty"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// TLSConfig defines TLS/SSL configuration
type TLSConfig struct {
	// InsecureSkipVerify controls whether certificate verification is skipped.
	// WARNING: Setting this to true is dangerous and makes connections vulnerable to attacks.
	// Only use this for testing or with trusted internal services.
	InsecureSkipVerify bool `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`

	// ServerName is used to verify the hostname on returned certificates.
	ServerName string `yaml:"server_name,omitempty" json:"server_name,omitempty"`

	// RootCAs defines the set of root certificate authorities.
	RootCAs []string `yaml:"root_cas,omitempty" json:"root_cas,omitempty"`

	// ClientCert and ClientKey define the client certificate and key for mutual TLS.
	ClientCert string `yaml:"client_cert,omitempty" json:"client_cert,omitempty"`
	ClientKey  string `yaml:"client_key,omitempty" json:"client_key,omitempty"`

	// SuppressWarnings controls whether security warnings are logged when insecure settings are used.
	SuppressWarnings bool `yaml:"suppress_warnings,omitempty" json:"suppress_warnings,omitempty"`
}

// ProxyProvider represents a proxy provider configuration
type ProxyProvider struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type" json:"type"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
	Weight   int    `yaml:"weight,omitempty" json:"weight,omitempty"`
	Enabled  bool   `yaml:"enabled" json:"enabled"`
}

// TransformRule represents a data transformation rule
type TransformRule struct {
	Type        string                 `yaml:"type" json:"type"`
	Pattern     string                 `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Replacement string                 `yaml:"replacement,omitempty" json:"replacement,omitempty"`
	Format      string                 `yaml:"format,omitempty" json:"format,omitempty"`
	Params      map[string]interface{} `yaml:"params,omitempty" json:"params,omitempty"`
}

// BrowserConfig represents browser automation configuration
type BrowserConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Headless       bool   `yaml:"headless" json:"headless"`
	UserDataDir    string `yaml:"user_data_dir,omitempty" json:"user_data_dir,omitempty"`
	Timeout        string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	ViewportWidth  int    `yaml:"viewport_width,omitempty" json:"viewport_width,omitempty"`
	ViewportHeight int    `yaml:"viewport_height,omitempty" json:"viewport_height,omitempty"`
	WaitForElement string `yaml:"wait_for_element,omitempty" json:"wait_for_element,omitempty"`
	WaitDelay      string `yaml:"wait_delay,omitempty" json:"wait_delay,omitempty"`
	UserAgent      string `yaml:"user_agent,omitempty" json:"user_agent,omitempty"`
	DisableImages  bool   `yaml:"disable_images" json:"disable_images"`
	DisableCSS     bool   `yaml:"disable_css" json:"disable_css"`
	DisableJS      bool   `yaml:"disable_js" json:"disable_js"`
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(filename string) (*ScraperConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ScraperConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// LoadFromBytes loads configuration from YAML bytes
func LoadFromBytes(data []byte) (*ScraperConfig, error) {
	var config ScraperConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// SimpleValidate provides basic validation (kept for backward compatibility)
func (c *ScraperConfig) SimpleValidate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if c.BaseURL == "" && len(c.URLs) == 0 {
		return fmt.Errorf("base_url or urls is required")
	}

	if len(c.Fields) == 0 {
		return fmt.Errorf("at least one field is required")
	}

	// Validate fields
	for i, field := range c.Fields {
		if field.Name == "" {
			return fmt.Errorf("field %d: name is required", i)
		}
		if field.Selector == "" {
			return fmt.Errorf("field %d: selector is required", i)
		}
		if field.Type == "" {
			return fmt.Errorf("field %d: type is required", i)
		}

		// Validate field types
		validTypes := map[string]bool{
			"text": true, "html": true, "attr": true, "list": true,
		}
		if !validTypes[field.Type] {
			return fmt.Errorf("field %d: invalid type %s", i, field.Type)
		}

		// Require attribute for attr type
		if field.Type == "attr" && field.Attribute == "" {
			return fmt.Errorf("field %d: attribute is required for type 'attr'", i)
		}
	}

	// Validate output
	if c.Output.Format == "" {
		c.Output.Format = "json" // Default format
	}

	validFormats := map[string]bool{
		"json": true, "csv": true, "yaml": true,
	}
	if !validFormats[c.Output.Format] {
		return fmt.Errorf("invalid output format: %s", c.Output.Format)
	}

	if c.Output.File == "" {
		c.Output.File = "output." + c.Output.Format // Default filename
	}

	return nil
}

// ConfigCache provides thread-safe configuration caching with efficient LRU eviction
type ConfigCache struct {
	cache         map[string]*CachedConfig
	lruList       *lruNode // Doubly-linked list for O(1) LRU operations
	lruTail       *lruNode // Tail of the LRU list
	mutex         sync.RWMutex
	maxSize       int
	timeout       time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// lruNode represents a node in the doubly-linked LRU list
type lruNode struct {
	key  string
	prev *lruNode
	next *lruNode
}

// CachedConfig holds a configuration with metadata
type CachedConfig struct {
	Config      *ScraperConfig
	Hash        string
	LoadTime    time.Time
	AccessTime  time.Time
	FileName    string
	FileSize    int64
	AccessCount int64
	lruNode     *lruNode // Reference to LRU list node for O(1) operations
}

// ConfigManager provides advanced configuration management
type ConfigManager struct {
	cache      *ConfigCache
	validator  *ConfigValidator
	metrics    *ConfigMetrics
}

// ConfigValidator provides comprehensive validation
type ConfigValidator struct {
	strict        bool
	customRules   []ValidationRule
	schemaVersion string
}

// ValidationRule represents a custom validation rule
type ValidationRule struct {
	Name      string
	Validator func(*ScraperConfig) error
	Severity  ValidationSeverity
}

// ValidationSeverity levels
type ValidationSeverity int

const (
	SeverityError ValidationSeverity = iota
	SeverityWarning
	SeverityInfo
)

// ConfigMetrics tracks configuration usage statistics
type ConfigMetrics struct {
	loadsTotal     int64
	cacheHits      int64
	cacheMisses    int64
	validationTime time.Duration
	loadTime       time.Duration
	mutex          sync.RWMutex
}

// Global instances
var (
	defaultConfigManager *ConfigManager
	managerOnce         sync.Once
)

// GetConfigManager returns the singleton configuration manager
func GetConfigManager() *ConfigManager {
	managerOnce.Do(func() {
		defaultConfigManager = NewConfigManager(ConfigManagerOptions{
			CacheSize:    100,
			CacheTimeout: 30 * time.Minute,
			StrictMode:   false,
		})
	})
	return defaultConfigManager
}

// ConfigManagerOptions configures the configuration manager
type ConfigManagerOptions struct {
	CacheSize    int
	CacheTimeout time.Duration
	StrictMode   bool
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(opts ConfigManagerOptions) *ConfigManager {
	if opts.CacheSize <= 0 {
		opts.CacheSize = 50
	}
	if opts.CacheTimeout <= 0 {
		opts.CacheTimeout = 15 * time.Minute
	}

	cache := &ConfigCache{
		cache:       make(map[string]*CachedConfig),
		maxSize:     opts.CacheSize,
		timeout:     opts.CacheTimeout,
		stopCleanup: make(chan bool),
	}
	
	// Initialize LRU list with sentinel nodes to simplify operations
	cache.lruList = &lruNode{}
	cache.lruTail = &lruNode{}
	cache.lruList.next = cache.lruTail
	cache.lruTail.prev = cache.lruList

	// Start cleanup goroutine
	cache.cleanupTicker = time.NewTicker(opts.CacheTimeout / 4)
	go cache.cleanupExpired()

	validator := &ConfigValidator{
		strict:        opts.StrictMode,
		customRules:   make([]ValidationRule, 0),
		schemaVersion: "1.0",
	}

	metrics := &ConfigMetrics{}

	return &ConfigManager{
		cache:     cache,
		validator: validator,
		metrics:   metrics,
	}
}

// LoadFromFileWithCache loads configuration with caching support
func (cm *ConfigManager) LoadFromFileWithCache(filename string) (*ScraperConfig, error) {
	start := time.Now()
	defer func() {
		cm.metrics.mutex.Lock()
		cm.metrics.loadsTotal++
		cm.metrics.loadTime += time.Since(start)
		cm.metrics.mutex.Unlock()
	}()

	// Get file info for cache validation
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	// Check cache first
	if cached, hit := cm.cache.get(filename, fileInfo.Size(), fileInfo.ModTime()); hit {
		cm.metrics.mutex.Lock()
		cm.metrics.cacheHits++
		cm.metrics.mutex.Unlock()
		return cached.Config, nil
	}

	cm.metrics.mutex.Lock()
	cm.metrics.cacheMisses++
	cm.metrics.mutex.Unlock()

	// Load from file
	config, err := LoadFromFile(filename)
	if err != nil {
		return nil, err
	}

	// Validate configuration
	validationStart := time.Now()
	if err := cm.validator.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	cm.metrics.mutex.Lock()
	cm.metrics.validationTime += time.Since(validationStart)
	cm.metrics.mutex.Unlock()

	// Cache the configuration
	cm.cache.put(filename, config, fileInfo.Size())

	return config, nil
}

// Cache methods
func (cc *ConfigCache) get(filename string, fileSize int64, modTime time.Time) (*CachedConfig, bool) {
	cc.mutex.Lock() // Use write lock since we modify LRU order
	defer cc.mutex.Unlock()

	cached, exists := cc.cache[filename]
	if !exists {
		return nil, false
	}

	// Check if file has been modified
	if cached.FileSize != fileSize {
		// Remove invalid entry
		cc.removeFromLRU(cached.lruNode)
		delete(cc.cache, filename)
		return nil, false
	}

	// Check if cache entry is expired
	if time.Since(cached.LoadTime) > cc.timeout {
		// Remove expired entry
		cc.removeFromLRU(cached.lruNode)
		delete(cc.cache, filename)
		return nil, false
	}

	// Update access time and count
	cached.AccessTime = time.Now()
	cached.AccessCount++
	
	// Move to front of LRU list (most recently used)
	cc.moveToFront(cached.lruNode)

	return cached, true
}

func (cc *ConfigCache) put(filename string, config *ScraperConfig, fileSize int64) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	// Check if entry already exists
	if existing, exists := cc.cache[filename]; exists {
		// Update existing entry and move to front
		existing.Config = config
		existing.Hash = cc.calculateHash(config)
		existing.LoadTime = time.Now()
		existing.AccessTime = time.Now()
		existing.FileSize = fileSize
		existing.AccessCount++
		cc.moveToFront(existing.lruNode)
		return
	}

	// Check cache size and evict if necessary
	if len(cc.cache) >= cc.maxSize {
		cc.evictLRU()
	}

	// Create new LRU node
	node := &lruNode{key: filename}
	
	// Calculate hash for integrity checking
	hash := cc.calculateHash(config)

	// Create cached config with LRU node reference
	cached := &CachedConfig{
		Config:      config,
		Hash:        hash,
		LoadTime:    time.Now(),
		AccessTime:  time.Now(),
		FileName:    filename,
		FileSize:    fileSize,
		AccessCount: 1,
		lruNode:     node,
	}
	
	// Add to cache and LRU list
	cc.cache[filename] = cached
	cc.addToFront(node)
}

// evictLRU removes the least recently used item in O(1) time
func (cc *ConfigCache) evictLRU() {
	// Get the least recently used node (tail's previous)
	lru := cc.lruTail.prev
	if lru == cc.lruList {
		// List is empty, nothing to evict
		return
	}
	
	// Remove from cache and LRU list
	delete(cc.cache, lru.key)
	cc.removeFromLRU(lru)
}

// addToFront adds a node to the front of the LRU list (most recently used)
func (cc *ConfigCache) addToFront(node *lruNode) {
	node.prev = cc.lruList
	node.next = cc.lruList.next
	cc.lruList.next.prev = node
	cc.lruList.next = node
}

// removeFromLRU removes a node from the LRU list
func (cc *ConfigCache) removeFromLRU(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// moveToFront moves an existing node to the front of the LRU list
func (cc *ConfigCache) moveToFront(node *lruNode) {
	cc.removeFromLRU(node)
	cc.addToFront(node)
}

func (cc *ConfigCache) calculateHash(config *ScraperConfig) string {
	data, _ := yaml.Marshal(config)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (cc *ConfigCache) cleanupExpired() {
	for {
		select {
		case <-cc.cleanupTicker.C:
			cc.mutex.Lock()
			now := time.Now()
			for key, cached := range cc.cache {
				if now.Sub(cached.LoadTime) > cc.timeout {
					delete(cc.cache, key)
				}
			}
			cc.mutex.Unlock()
		case <-cc.stopCleanup:
			return
		}
	}
}

// Stop cleanup goroutine
func (cc *ConfigCache) Stop() {
	if cc.cleanupTicker != nil {
		cc.cleanupTicker.Stop()
	}
	close(cc.stopCleanup)
}

// Clear removes all cached configurations
func (cc *ConfigCache) Clear() {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	cc.cache = make(map[string]*CachedConfig)
	
	// Reset LRU list
	cc.lruList.next = cc.lruTail
	cc.lruTail.prev = cc.lruList
}

// GetStats returns cache statistics
func (cc *ConfigCache) GetStats() map[string]interface{} {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	return map[string]interface{}{
		"size":      len(cc.cache),
		"max_size":  cc.maxSize,
		"timeout":   cc.timeout.String(),
		"entries":   len(cc.cache),
	}
}

// ValidateConfig performs comprehensive validation
func (cv *ConfigValidator) ValidateConfig(config *ScraperConfig) error {
	// Run standard validation
	if err := config.Validate(); err != nil {
		return err
	}

	// Run custom validation rules
	for _, rule := range cv.customRules {
		if err := rule.Validator(config); err != nil {
			if rule.Severity == SeverityError {
				return fmt.Errorf("custom validation rule '%s' failed: %w", rule.Name, err)
			}
			// For warnings and info, log but don't fail
		}
	}

	return nil
}

// AddValidationRule adds a custom validation rule
func (cv *ConfigValidator) AddValidationRule(rule ValidationRule) {
	cv.customRules = append(cv.customRules, rule)
}

// SetStrictMode enables or disables strict validation
func (cv *ConfigValidator) SetStrictMode(strict bool) {
	cv.strict = strict
}

// GetMetrics returns configuration manager metrics
func (cm *ConfigManager) GetMetrics() map[string]interface{} {
	cm.metrics.mutex.RLock()
	defer cm.metrics.mutex.RUnlock()

	return map[string]interface{}{
		"loads_total":     cm.metrics.loadsTotal,
		"cache_hits":      cm.metrics.cacheHits,
		"cache_misses":    cm.metrics.cacheMisses,
		"hit_ratio":       func() float64 {
			denom := cm.metrics.cacheHits + cm.metrics.cacheMisses
			if denom == 0 {
				return 0.0
			}
			return float64(cm.metrics.cacheHits) / float64(denom)
		}(),
		"avg_load_time":   func() time.Duration {
			if cm.metrics.loadsTotal == 0 {
				return 0
			}
			return cm.metrics.loadTime / time.Duration(cm.metrics.loadsTotal)
		}(),
		"avg_validation_time": func() time.Duration {
			if cm.metrics.loadsTotal == 0 {
				return 0
			}
			return cm.metrics.validationTime / time.Duration(cm.metrics.loadsTotal)
		}(),
	}
}

// LoadFromFileOptimized provides the most optimized loading experience
func LoadFromFileOptimized(filename string) (*ScraperConfig, error) {
	return GetConfigManager().LoadFromFileWithCache(filename)
}

// GenerateTemplate generates a template configuration
func GenerateTemplate(templateType string) *ScraperConfig {
	switch templateType {
	case "ecommerce":
		return &ScraperConfig{
			Name:    "ecommerce_scraper",
			BaseURL: "https://example-shop.com/products",
			Fields: []Field{
				{
					Name:     "title",
					Selector: ".product-title, h1",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "price",
					Selector: ".price, .product-price",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "description",
					Selector: ".product-description",
					Type:     "text",
					Required: false,
				},
				{
					Name:      "image",
					Selector:  ".product-image img",
					Type:      "attr",
					Attribute: "src",
					Required:  false,
				},
			},
			Output: OutputConfig{
				Format: "json",
				File:   "products.json",
			},
			RateLimit: "2s",
		}
	case "news":
		return &ScraperConfig{
			Name:    "news_scraper",
			BaseURL: "https://example-news.com/articles",
			Fields: []Field{
				{
					Name:     "headline",
					Selector: "h1, .headline",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "author",
					Selector: ".author, .byline",
					Type:     "text",
					Required: false,
				},
				{
					Name:     "content",
					Selector: ".article-content, .story-body",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "date",
					Selector: ".publish-date, time",
					Type:     "text",
					Required: false,
				},
			},
			Output: OutputConfig{
				Format: "json",
				File:   "articles.json",
			},
			RateLimit: "3s",
		}
	default: // basic
		return &ScraperConfig{
			Name:    "basic_scraper",
			BaseURL: "https://example.com",
			Fields: []Field{
				{
					Name:     "title",
					Selector: "h1",
					Type:     "text",
					Required: true,
				},
				{
					Name:     "content",
					Selector: "p",
					Type:     "text",
					Required: false,
				},
			},
			Output: OutputConfig{
				Format: "json",
				File:   "output.json",
			},
			RateLimit: "1s",
		}
	}
}

// ConfigWatcher provides file watching capabilities for configuration hot-reloading
type ConfigWatcher struct {
	filename        string
	lastModTime     time.Time
	lastSize        int64
	pollInterval    time.Duration
	callbacks       []func(*ScraperConfig, error)
	stopWatching    chan bool
	running         bool
	mutex           sync.RWMutex
	callbackWorkers chan struct{} // Semaphore to limit concurrent callback executions
	maxWorkers      int           // Maximum number of concurrent callback workers
	ctx             context.Context
	cancel          context.CancelFunc
	
	// Goroutine monitoring
	activeGoroutines int64 // Atomic counter for active callback goroutines
	totalCallbacks   int64 // Total callbacks executed (for metrics)
}

// NewConfigWatcher creates a new configuration file watcher
func NewConfigWatcher(filename string, pollInterval time.Duration) *ConfigWatcher {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}

	// Limit concurrent callback executions to prevent resource exhaustion
	maxWorkers := 10
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ConfigWatcher{
		filename:        filename,
		pollInterval:    pollInterval,
		callbacks:       make([]func(*ScraperConfig, error), 0),
		stopWatching:    make(chan bool),
		callbackWorkers: make(chan struct{}, maxWorkers),
		maxWorkers:      maxWorkers,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// OnChange adds a callback for configuration changes
func (cw *ConfigWatcher) OnChange(callback func(*ScraperConfig, error)) {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

// Start begins watching the configuration file
func (cw *ConfigWatcher) Start() error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	if cw.running {
		return fmt.Errorf("watcher is already running")
	}

	// Get initial file info
	fileInfo, err := os.Stat(cw.filename)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	cw.lastModTime = fileInfo.ModTime()
	cw.lastSize = fileInfo.Size()
	cw.running = true

	go cw.watchLoop()
	return nil
}

// Stop stops watching the configuration file
func (cw *ConfigWatcher) Stop() {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	if !cw.running {
		return
	}

	cw.running = false
	cw.cancel() // Cancel the context to coordinate with goroutines
	close(cw.stopWatching)
}

func (cw *ConfigWatcher) watchLoop() {
	ticker := time.NewTicker(cw.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cw.checkForChanges()
		case <-cw.stopWatching:
			return
		}
	}
}

func (cw *ConfigWatcher) checkForChanges() {
	fileInfo, err := os.Stat(cw.filename)
	if err != nil {
		cw.notifyCallbacks(nil, fmt.Errorf("failed to stat config file: %w", err))
		return
	}

	cw.mutex.RLock()
	lastModTime := cw.lastModTime
	lastSize := cw.lastSize
	cw.mutex.RUnlock()

	// Check if file has changed
	if fileInfo.ModTime().After(lastModTime) || fileInfo.Size() != lastSize {
		// File has changed, reload configuration
		config, err := LoadFromFileOptimized(cw.filename)
		
		cw.mutex.Lock()
		cw.lastModTime = fileInfo.ModTime()
		cw.lastSize = fileInfo.Size()
		cw.mutex.Unlock()
		
		cw.notifyCallbacks(config, err)
	}
}

func (cw *ConfigWatcher) notifyCallbacks(config *ScraperConfig, err error) {
	cw.mutex.RLock()
	callbacks := make([]func(*ScraperConfig, error), len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.mutex.RUnlock()

	// Execute callbacks with limited concurrency and goroutine monitoring
	for _, callback := range callbacks {
		cw.wg.Add(1)
		go func(cb func(*ScraperConfig, error)) {
			defer cw.wg.Done()
			// Increment active goroutine counter for monitoring
			atomic.AddInt64(&cw.activeGoroutines, 1)
			defer func() {
				atomic.AddInt64(&cw.activeGoroutines, -1)
				atomic.AddInt64(&cw.totalCallbacks, 1)
			}()
			
			// Try to acquire worker semaphore with context coordination
			select {
			case cw.callbackWorkers <- struct{}{}:
				// Worker slot acquired, execute callback
				defer func() { <-cw.callbackWorkers }() // Release worker slot
				
				// Use the watcher's context with additional timeout
				ctx, cancel := context.WithTimeout(cw.ctx, 30*time.Second)
				defer cancel() // Ensure resources are cleaned up
				
				// Execute callback with proper cancellation support
				cw.executeCallbackWithContext(ctx, cb, config, err)
			case <-cw.ctx.Done():
				// Watcher context is cancelled, don't execute callback
				return
			default:
				// No worker slots available, check if we should still try to execute
				select {
				case <-cw.ctx.Done():
					// Watcher is stopping, don't execute callback
					return
				default:
					// Skip this callback to prevent blocking and potential goroutine leak
					// This prevents resource exhaustion when too many callbacks are queued
					return
				}
			}
		}(callback)
	}
}

// executeCallbackWithContext executes a callback with context cancellation support
func (cw *ConfigWatcher) executeCallbackWithContext(ctx context.Context, callback func(*ScraperConfig, error), config *ScraperConfig, err error) {
	// Channel to signal callback completion
	done := make(chan struct{})
	
	go func() {
		defer func() {
			// Recover from any panic in the callback to prevent crashing
			if r := recover(); r != nil {
				// Log panic if possible, but don't crash the watcher
			}
			close(done)
		}()
		
		// Execute the callback
		callback(config, err)
	}()
	
	// Wait for either completion or context cancellation
	select {
	case <-done:
		// Callback completed successfully
		return
	case <-ctx.Done():
		// Context cancelled (timeout or explicit cancellation)
		// The callback goroutine will continue but we don't wait for it
		// The defer in the goroutine will still close the done channel
		return
	}
}

// GetGoroutineStats returns statistics about callback goroutine usage
func (cw *ConfigWatcher) GetGoroutineStats() map[string]interface{} {
	return map[string]interface{}{
		"active_goroutines": atomic.LoadInt64(&cw.activeGoroutines),
		"total_callbacks":   atomic.LoadInt64(&cw.totalCallbacks),
		"max_workers":       cw.maxWorkers,
		"available_slots":   cw.maxWorkers - len(cw.callbackWorkers),
	}
}

// ConfigBuilder provides a fluent interface for building configurations
type ConfigBuilder struct {
	config *ScraperConfig
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &ScraperConfig{
			Fields:  make([]Field, 0),
			Headers: make(map[string]string),
			Cookies: make(map[string]string),
		},
	}
}

// WithName sets the scraper name
func (cb *ConfigBuilder) WithName(name string) *ConfigBuilder {
	cb.config.Name = name
	return cb
}

// WithBaseURL sets the base URL
func (cb *ConfigBuilder) WithBaseURL(url string) *ConfigBuilder {
	cb.config.BaseURL = url
	return cb
}

// WithField adds a field configuration
func (cb *ConfigBuilder) WithField(name, selector, fieldType string) *ConfigBuilder {
	cb.config.Fields = append(cb.config.Fields, Field{
		Name:     name,
		Selector: selector,
		Type:     fieldType,
	})
	return cb
}

// WithRequiredField adds a required field
func (cb *ConfigBuilder) WithRequiredField(name, selector, fieldType string) *ConfigBuilder {
	cb.config.Fields = append(cb.config.Fields, Field{
		Name:     name,
		Selector: selector,
		Type:     fieldType,
		Required: true,
	})
	return cb
}

// WithRateLimit sets the rate limit
func (cb *ConfigBuilder) WithRateLimit(rateLimit string) *ConfigBuilder {
	cb.config.RateLimit = rateLimit
	return cb
}

// WithTimeout sets the timeout
func (cb *ConfigBuilder) WithTimeout(timeout string) *ConfigBuilder {
	cb.config.Timeout = timeout
	return cb
}

// WithMaxRetries sets the maximum retries
func (cb *ConfigBuilder) WithMaxRetries(retries int) *ConfigBuilder {
	cb.config.MaxRetries = retries
	return cb
}

// WithHeader adds a header
func (cb *ConfigBuilder) WithHeader(key, value string) *ConfigBuilder {
	cb.config.Headers[key] = value
	return cb
}

// WithUserAgent sets the user agent
func (cb *ConfigBuilder) WithUserAgent(userAgent string) *ConfigBuilder {
	cb.config.UserAgents = []string{userAgent}
	return cb
}

// WithMultipleUserAgents sets multiple user agents
func (cb *ConfigBuilder) WithMultipleUserAgents(userAgents []string) *ConfigBuilder {
	cb.config.UserAgents = userAgents
	return cb
}

// WithOutput sets the output configuration
func (cb *ConfigBuilder) WithOutput(format, file string) *ConfigBuilder {
	cb.config.Output = OutputConfig{
		Format: format,
		File:   file,
	}
	return cb
}

// WithProxy enables proxy configuration
func (cb *ConfigBuilder) WithProxy(enabled bool) *ConfigBuilder {
	if cb.config.Proxy == nil {
		cb.config.Proxy = &ProxyConfig{}
	}
	cb.config.Proxy.Enabled = enabled
	return cb
}

// WithBrowser enables browser automation
func (cb *ConfigBuilder) WithBrowser(enabled, headless bool) *ConfigBuilder {
	if cb.config.Browser == nil {
		cb.config.Browser = &BrowserConfig{}
	}
	cb.config.Browser.Enabled = enabled
	cb.config.Browser.Headless = headless
	return cb
}

// Build returns the built configuration
func (cb *ConfigBuilder) Build() *ScraperConfig {
	return cb.config
}

// Validate validates the built configuration
func (cb *ConfigBuilder) Validate() error {
	return cb.config.Validate()
}

// BuildAndValidate builds and validates the configuration
func (cb *ConfigBuilder) BuildAndValidate() (*ScraperConfig, error) {
	config := cb.Build()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}
