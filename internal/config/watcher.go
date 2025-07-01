// internal/config/watcher.go
package config

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/valpere/DataScrapexter/internal/utils"
)

var logger = utils.NewComponentLogger("config-watcher")

// ConfigChangeCallback is called when configuration changes
type ConfigChangeCallback func(*ScraperConfig)

// ConfigWatcher watches configuration files for changes
type ConfigWatcher struct {
	watcher   *fsnotify.Watcher
	callbacks []ConfigChangeCallback
	mu        sync.RWMutex
	stopped   bool
}

// NewConfigWatcher creates a new configuration file watcher
func NewConfigWatcher() (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.WithField("error", err).Error("Failed to create file watcher")
		return nil, err
	}

	cw := &ConfigWatcher{
		watcher:   watcher,
		callbacks: make([]ConfigChangeCallback, 0),
		stopped:   false,
	}

	logger.Info("Config watcher created successfully")
	return cw, nil
}

// Watch starts watching a configuration file for changes
func (cw *ConfigWatcher) Watch(configPath string) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.stopped {
		return nil
	}

	// Watch the directory containing the config file
	dir := filepath.Dir(configPath)
	if err := cw.watcher.Add(dir); err != nil {
		logger.WithFields(map[string]interface{}{
			"config_path": configPath,
			"directory":   dir,
			"error":       err,
		}).Error("Failed to add directory to watcher")
		return err
	}

	logger.WithFields(map[string]interface{}{
		"config_path": configPath,
		"directory":   dir,
	}).Info("Started watching configuration file")

	// Start the event processing goroutine
	go cw.processEvents(configPath)

	return nil
}

// AddCallback adds a callback function to be called when configuration changes
func (cw *ConfigWatcher) AddCallback(callback ConfigChangeCallback) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	cw.callbacks = append(cw.callbacks, callback)
	logger.WithField("callback_count", len(cw.callbacks)).Debug("Added config change callback")
}

// processEvents processes file system events
func (cw *ConfigWatcher) processEvents(configPath string) {
	configFileName := filepath.Base(configPath)

	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				logger.Debug("Watcher events channel closed")
				return
			}

			// Check if the event is for our config file
			if filepath.Base(event.Name) == configFileName && event.Op&fsnotify.Write == fsnotify.Write {
				logger.WithFields(map[string]interface{}{
					"file":  event.Name,
					"event": event.Op.String(),
				}).Info("Configuration file changed, reloading")

				cw.reloadConfig(configPath)
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				logger.Debug("Watcher errors channel closed")
				return
			}
			logger.WithField("error", err).Error("Config watcher error")
		}
	}
}

// reloadConfig reloads the configuration and notifies callbacks
func (cw *ConfigWatcher) reloadConfig(configPath string) {
	// Add a small delay to ensure file write is complete
	time.Sleep(100 * time.Millisecond)

	cw.mu.RLock()
	callbacks := make([]ConfigChangeCallback, len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.mu.RUnlock()

	// Load the new configuration
	newConfig, err := LoadFromFile(configPath)
	if err != nil {
		logger.WithField("error", err).Error("Failed to reload configuration")
		return
	}

	// Validate the new configuration
	if err := newConfig.Validate(); err != nil {
		logger.WithField("error", err).Error("Reloaded configuration is invalid")
		return
	}

	logger.WithFields(map[string]interface{}{
		"config_name":    newConfig.Name,
		"callback_count": len(callbacks),
	}).Info("Configuration updated successfully, notifying callbacks")

	// Notify all callbacks
	for i, callback := range callbacks {
		logger.WithField("callback_index", i).Debug("Executing config change callback")
		callback(newConfig)
	}
}

// Stop stops the configuration watcher
func (cw *ConfigWatcher) Stop() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.stopped {
		logger.Debug("Config watcher already stopped")
		return nil
	}

	cw.stopped = true
	logger.Info("Stopping config watcher")

	if err := cw.watcher.Close(); err != nil {
		logger.WithField("error", err).Error("Error closing file watcher")
		return err
	}

	logger.Info("Config watcher stopped successfully")
	return nil
}
