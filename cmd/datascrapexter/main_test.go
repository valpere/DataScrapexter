// cmd/datascrapexter/main_test.go
package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestCLIVersion(t *testing.T) {
	// Set test values
	version = "test-version"
	buildTime = "2025-06-23"
	gitCommit = "abc123"

	// Capture output
	output := captureOutput(func() {
		printVersion()
	})

	if !strings.Contains(output, "test-version") {
		t.Errorf("version output should contain version, got: %s", output)
	}
	if !strings.Contains(output, "2025-06-23") {
		t.Errorf("version output should contain build time, got: %s", output)
	}
	if !strings.Contains(output, "abc123") {
		t.Errorf("version output should contain git commit, got: %s", output)
	}
}

func TestCLIHelp(t *testing.T) {
	output := captureOutput(func() {
		printUsage()
	})

	commands := []string{"run", "validate", "template", "version", "help"}
	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("help output should contain command %q, got: %s", cmd, output)
		}
	}
}

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	f()
	w.Close()
	os.Stdout = old
	out := <-outC

	return out
}
