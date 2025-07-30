// internal/browser/navigation_state_test.go
package browser

import (
	"context"
	"strings"
	"testing"
)

func TestChromeClient_NavigationStateTracking(t *testing.T) {
	config := DefaultBrowserConfig()
	client, err := NewChromeClient(config)
	if err != nil {
		t.Skipf("Skipping browser test: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Initially, navigationSuccess should be false
	// Attempt to get HTML before navigation should fail
	_, err = client.GetHTML(ctx)
	if err == nil {
		t.Error("Expected error when trying to get HTML before successful navigation")
		return
	}
	if !strings.Contains(err.Error(), "navigation has not completed successfully") {
		t.Errorf("Expected navigation state error, got: %v", err)
		return
	}

	t.Log("✓ Navigation state tracking works: HTML extraction properly blocked before successful navigation")
}