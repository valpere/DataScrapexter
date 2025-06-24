// cmd/datascrapexter/main_test.go
package main

import (
    "bytes"
    "os"
    "os/exec"
    "strings"
    "testing"
)

func TestCLIVersion(t *testing.T) {
    version = "test-version"
    buildTime = "2025-06-23"
    gitCommit = "abc123"

    oldArgs := os.Args
    defer func() { os.Args = oldArgs }()

    os.Args = []string{"datascrapexter", "version"}

    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    main()

    w.Close()
    os.Stdout = oldStdout

    var output bytes.Buffer
    output.ReadFrom(r)
    result := output.String()

    if !strings.Contains(result, "test-version") {
        t.Errorf("version output should contain version, got: %s", result)
    }
}

func TestCLIHelp(t *testing.T) {
    oldArgs := os.Args
    defer func() { os.Args = oldArgs }()

    os.Args = []string{"datascrapexter", "help"}

    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    main()

    w.Close()
    os.Stdout = oldStdout

    var output bytes.Buffer
    output.ReadFrom(r)
    result := output.String()

    expectedCommands := []string{"run", "validate", "template", "version", "help"}
    for _, cmd := range expectedCommands {
        if !strings.Contains(result, cmd) {
            t.Errorf("help output should contain command %q", cmd)
        }
    }
}

func TestCLIIntegration(t *testing.T) {
    binaryPath := "./datascrapexter"
    if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
        t.Skip("Binary not found, skipping integration test")
    }

    tests := []struct {
        name     string
        args     []string
        wantExit int
    }{
        {"version command", []string{"version"}, 0},
        {"help command", []string{"help"}, 0},
        {"invalid command", []string{"invalid"}, 1},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := exec.Command(binaryPath, tt.args...)
            output, err := cmd.CombinedOutput()

            var exitCode int
            if err != nil {
                if exitError, ok := err.(*exec.ExitError); ok {
                    exitCode = exitError.ExitCode()
                } else {
                    t.Fatalf("failed to run command: %v", err)
                }
            }

            if exitCode != tt.wantExit {
                t.Errorf("expected exit code %d, got %d. Output: %s", 
                    tt.wantExit, exitCode, output)
            }
        })
    }
}
