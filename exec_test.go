//go:build (linux || darwin) && (amd64 || arm64)

package exec_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	stdexec "os/exec"

	"github.com/alecthomas/exec"
)

func TestCommand(t *testing.T) {
	cmd := exec.Command("echo", "hello", "world")
	if cmd == nil {
		t.Fatal("Command returned nil")
	}

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	expected := "hello world\n"
	if string(output) != expected {
		t.Errorf("Expected %q, got %q", expected, string(output))
	}
}

func TestCommandContext(t *testing.T) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "echo", "test")
	if cmd == nil {
		t.Fatal("CommandContext returned nil")
	}

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("CommandContext failed: %v", err)
	}

	expected := "test\n"
	if string(output) != expected {
		t.Errorf("Expected %q, got %q", expected, string(output))
	}
}

func TestCommandContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Use a command that will run longer than the timeout
	cmd := exec.CommandContext(ctx, "sleep", "1")

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected command to fail due to context cancellation")
	}

	// Should have been canceled within a reasonable time
	if duration > 500*time.Millisecond {
		t.Errorf("Command took too long to cancel: %v", duration)
	}
}

func TestLookPath(t *testing.T) {
	// Test with a command that should exist on all systems
	path, err := exec.LookPath("echo")
	if err != nil {
		t.Fatalf("LookPath failed for echo: %v", err)
	}

	if path == "" {
		t.Error("LookPath returned empty path for echo")
	}

	if !strings.Contains(path, "echo") {
		t.Errorf("Expected path to contain 'echo', got %q", path)
	}
}

func TestLookPathNotFound(t *testing.T) {
	_, err := exec.LookPath("this-command-should-not-exist-anywhere")
	if err == nil {
		t.Error("Expected LookPath to fail for non-existent command")
	}
}

func TestBinaryExtraction(t *testing.T) {
	// Force extraction by calling Command
	cmd := exec.Command("echo", "extraction-test")
	if cmd == nil {
		t.Fatal("Command returned nil")
	}

	// Verify the command works (which means extraction succeeded)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed after extraction: %v", err)
	}

	expected := "extraction-test\n"
	if string(output) != expected {
		t.Errorf("Expected %q, got %q", expected, string(output))
	}
}

func TestMultipleCommands(t *testing.T) {
	// Test that multiple commands work (reusing the extracted binary)
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"Echo", []string{"echo", "first"}, "first\n"},
		{"EchoMultiple", []string{"echo", "hello", "world"}, "hello world\n"},
		{"True", []string{"true"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(tt.args[0], tt.args[1:]...)
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Command %v failed: %v", tt.args, err)
			}

			if string(output) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(output))
			}
		})
	}
}

func TestCommandEnvironment(t *testing.T) {
	cmd := exec.Command("sh", "-c", "echo $TEST_VAR")
	cmd.Env = append(os.Environ(), "TEST_VAR=test_value")

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	expected := "test_value\n"
	if string(output) != expected {
		t.Errorf("Expected %q, got %q", expected, string(output))
	}
}

func TestCommandWorkingDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "exec-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("pwd")
	cmd.Dir = tmpDir

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	result := strings.TrimSpace(string(output))
	if result != tmpDir {
		t.Errorf("Expected working directory %q, got %q", tmpDir, result)
	}
}

func TestCommandFailure(t *testing.T) {
	cmd := exec.Command("false")
	err := cmd.Run()
	if err == nil {
		t.Error("Expected command to fail")
	}

	// Should be an ExitError
	if _, ok := err.(*exec.ExitError); !ok {
		t.Errorf("Expected ExitError, got %T", err)
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that error constants are properly exported
	if exec.ErrDot == nil {
		t.Error("ErrDot should not be nil")
	}
	if exec.ErrNotFound == nil {
		t.Error("ErrNotFound should not be nil")
	}
	if exec.ErrWaitDelay == nil {
		t.Error("ErrWaitDelay should not be nil")
	}
}

func TestConcurrentCommands(t *testing.T) {
	// Test that multiple commands can run concurrently
	const numCommands = 10
	done := make(chan error, numCommands)

	for i := range numCommands {
		go func(i int) {
			cmd := exec.Command("echo", fmt.Sprintf("concurrent-%d", i))
			_, err := cmd.Output()
			done <- err
		}(i)
	}

	for i := range numCommands {
		if err := <-done; err != nil {
			t.Errorf("Concurrent command %d failed: %v", i, err)
		}
	}
}

func TestProcessGroupCleanup(t *testing.T) {
	// Build the test program from testdata
	buildCmd := stdexec.Command("go", "build", "-o", "test-program", "./testdata")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build test program: %v", err)
	}
	t.Cleanup(func() { os.Remove("test-program") })

	// Start the test program using our exec package
	testCmd := stdexec.Command("./test-program")
	if err := testCmd.Start(); err != nil {
		t.Fatalf("Failed to start test program: %v", err)
	}

	// Give it time to spawn the sleep processes
	time.Sleep(1 * time.Second)

	// Get initial process count for sleep processes (use 60 to match updated testdata)
	countCmd := stdexec.Command("sh", "-c", "pgrep -f 'sleep 60' | wc -l")
	countOutput, err := countCmd.Output()
	if err != nil {
		t.Fatalf("Failed to count processes: %v", err)
	}
	initialCount := strings.TrimSpace(string(countOutput))

	// Verify we have some sleep processes running
	if initialCount == "0" {
		t.Fatal("No sleep processes found - test program may have failed to start them")
	}

	t.Logf("Initial sleep process count: %s", initialCount)

	// Kill the test program
	if err := testCmd.Process.Kill(); err != nil {
		t.Fatalf("Failed to kill test program: %v", err)
	}

	// Wait for the process to be cleaned up
	testCmd.Wait()

	// Give the intermediary time to clean up child processes
	time.Sleep(2 * time.Second)

	// Check that all sleep processes have been cleaned up
	finalCountCmd := stdexec.Command("sh", "-c", "pgrep -f 'sleep 60' | grep -v 'watchdog' | wc -l")
	finalCountOutput, err := finalCountCmd.Output()
	if err != nil {
		t.Fatalf("Failed to count final processes: %v", err)
	}
	finalCount := strings.TrimSpace(string(finalCountOutput))

	t.Logf("Final sleep process count: %s", finalCount)

	// All sleep processes should have been cleaned up
	if finalCount != "0" {
		t.Errorf("Expected 0 remaining sleep processes, got %s - process group cleanup may not be working", finalCount)

		// Try to clean up any remaining processes for next tests
		cleanupCmd := stdexec.Command("sh", "-c", "pkill -f 'sleep 60'")
		cleanupCmd.Run()
	}
}
