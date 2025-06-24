// internal/config/watcher.go
package config

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// ConfigWatcher watches configuration files for changes
type ConfigWatcher struct {
	watcher    *fsnotify.Watcher
	configPath string
	callbacks  []func(*ScraperConfig)
	mu         sync.RWMutex
	stopped    bool
}

// NewConfigWatcher creates a new configuration file watcher
func NewConfigWatcher(configPath string) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	cw := &ConfigWatcher{
		watcher:    watcher,
		configPath: configPath,
		callbacks:  make([]func(*ScraperConfig), 0),
	}

	// Watch the config file
	if err := watcher.Add(configPath); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch config file: %w", err)
	}

	// Watch the directory as well (for editors that create temp files)
	dir := filepath.Dir(configPath)
	if err := watcher.Add(dir); err != nil {
		log.Printf("Warning: failed to watch config directory: %v", err)
	}

	// Start watching in a goroutine
	go cw.watch()

	return cw, nil
}

// OnChange registers a callback to be called when the config changes
func (cw *ConfigWatcher) OnChange(callback func(*ScraperConfig)) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

// watch handles file system events
func (cw *ConfigWatcher) watch() {
	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			// Check if it's our config file and if it was modified
			if event.Name == cw.configPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				cw.handleConfigChange()
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Config watcher error: %v", err)
		}
	}
}

// handleConfigChange processes configuration file changes
func (cw *ConfigWatcher) handleConfigChange() {
	cw.mu.RLock()
	if cw.stopped {
		cw.mu.RUnlock()
		return
	}
	callbacks := make([]func(*ScraperConfig), len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.mu.RUnlock()

	// Load the updated configuration
	config, err := LoadFromFile(cw.configPath)
	if err != nil {
		log.Printf("Failed to reload config: %v", err)
		return
	}

	// Call all registered callbacks
	for _, callback := range callbacks {
		callback(config)
	}
}

// Close stops the watcher and releases resources
func (cw *ConfigWatcher) Close() error {
	cw.mu.Lock()
	cw.stopped = true
	cw.mu.Unlock()

	return cw.watcher.Close()
}
