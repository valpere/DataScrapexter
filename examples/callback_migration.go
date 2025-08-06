// Package examples demonstrates how to migrate from legacy callbacks to context-aware callbacks
package examples

import (
	"context"
	"fmt"
	"time"

	"github.com/valpere/DataScrapexter/internal/config"
)

// MigrationExample shows how to migrate from legacy callbacks to context-aware callbacks
func MigrationExample() {
	watcher := config.NewConfigWatcher("config.yaml", 5*time.Second)

	// OLD WAY (DEPRECATED) - Legacy callback without context support
	// This approach can lead to goroutine leaks if callbacks block indefinitely
	watcher.OnChange(func(config *config.ScraperConfig, err error) {
		if err != nil {
			fmt.Printf("Legacy callback - Configuration error: %v\n", err)
			return
		}
		
		// This callback has no way to be cancelled if it blocks indefinitely
		// It can potentially leak goroutines
		fmt.Printf("Legacy callback - Configuration updated: %+v\n", config)
	})

	// NEW WAY (RECOMMENDED) - Context-aware callback that prevents goroutine leaks
	// This approach provides proper cancellation support and bounded concurrency
	watcher.OnChangeWithContext(func(ctx context.Context, config *config.ScraperConfig, err error) {
		if err != nil {
			fmt.Printf("Context-aware callback - Configuration error: %v\n", err)
			return
		}

		// Simulate some processing work that respects context cancellation
		select {
		case <-ctx.Done():
			// Context was cancelled (timeout or shutdown), exit gracefully
			fmt.Printf("Context-aware callback - Cancelled due to: %v\n", ctx.Err())
			return
		case <-time.After(1 * time.Second):
			// Normal processing completed
			fmt.Printf("Context-aware callback - Configuration updated: %+v\n", config)
		}

		// Always check context before doing more work
		if ctx.Err() != nil {
			fmt.Printf("Context-aware callback - Context cancelled, stopping work\n")
			return
		}

		// More processing...
		fmt.Printf("Context-aware callback - Processing complete\n")
	})

	// Start watching (this would typically be done in a real application)
	if err := watcher.Start(); err != nil {
		fmt.Printf("Failed to start watcher: %v\n", err)
		return
	}

	// In a real application, you would keep the watcher running
	// Here we just demonstrate the stats
	time.Sleep(2 * time.Second)

	// Check goroutine statistics
	fmt.Printf("Goroutine stats: %+v\n", watcher.GetGoroutineStats())
	fmt.Printf("Callback registry stats: %+v\n", watcher.GetCallbackRegistryStats())

	// Clean shutdown
	watcher.Stop()
}

// ContextAwareCallbackExample shows best practices for writing context-aware callbacks
func ContextAwareCallbackExample() {
	watcher := config.NewConfigWatcher("config.yaml", 5*time.Second)

	// Example of a well-behaved context-aware callback
	watcher.OnChangeWithContext(func(ctx context.Context, config *config.ScraperConfig, err error) {
		// Always check for errors first
		if err != nil {
			fmt.Printf("Configuration error: %v\n", err)
			return
		}

		// Example: Perform database update with context
		if err := updateDatabaseConfig(ctx, config); err != nil {
			if ctx.Err() != nil {
				fmt.Printf("Database update cancelled: %v\n", ctx.Err())
			} else {
				fmt.Printf("Database update failed: %v\n", err)
			}
			return
		}

		// Example: Send notification with context
		if err := sendConfigChangeNotification(ctx, config); err != nil {
			if ctx.Err() != nil {
				fmt.Printf("Notification cancelled: %v\n", ctx.Err())
			} else {
				fmt.Printf("Notification failed: %v\n", err)
			}
			return
		}

		fmt.Printf("Configuration change processed successfully\n")
	})
}

// Mock functions to demonstrate context usage
func updateDatabaseConfig(ctx context.Context, config *config.ScraperConfig) error {
	// Simulate database operation with context support
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(500 * time.Millisecond):
		// Database update completed
		return nil
	}
}

func sendConfigChangeNotification(ctx context.Context, config *config.ScraperConfig) error {
	// Simulate notification with context support
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(200 * time.Millisecond):
		// Notification sent
		return nil
	}
}