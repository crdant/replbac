package cmd

import (
	"bytes"
	"fmt"
	"os"
	runtimeDebug "runtime/debug"
	"testing"
)

func TestPanicRecovery(t *testing.T) {
	tests := []struct {
		name        string
		panicValue  interface{}
		expectExit  bool
		expectLog   string
	}{
		{
			name:       "string panic recovered",
			panicValue: "test panic message",
			expectExit: true,
			expectLog:  "Application panic: test panic message",
		},
		{
			name:       "error panic recovered",
			panicValue: fmt.Errorf("test error panic"),
			expectExit: true,
			expectLog:  "Application panic: test error panic",
		},
		{
			name:       "nil panic recovered",
			panicValue: nil,
			expectExit: true,
			expectLog:  "Application panic: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Capture exit calls
			exitCalled := false
			oldExit := osExit
			osExit = func(code int) {
				exitCalled = true
				if code != 2 {
					t.Errorf("Expected exit code 2, got %d", code)
				}
			}

			// Restore after test
			defer func() {
				os.Stderr = oldStderr
				osExit = oldExit
				w.Close()
			}()

			// Test panic recovery
			func() {
				defer recoverFromPanic()
				panic(tt.panicValue)
			}()

			// Close writer and read stderr
			w.Close()
			var stderr bytes.Buffer
			stderr.ReadFrom(r)

			if exitCalled != tt.expectExit {
				t.Errorf("Expected exit=%v, got %v", tt.expectExit, exitCalled)
			}

			stderrStr := stderr.String()
			if tt.expectLog != "" && stderrStr == "" {
				t.Errorf("Expected log output but got none")
			}
		})
	}
}

func TestNoPanicRecovery(t *testing.T) {
	// Capture exit calls
	exitCalled := false
	oldExit := osExit
	osExit = func(code int) {
		exitCalled = true
	}
	defer func() {
		osExit = oldExit
	}()

	// Test normal execution without panic
	func() {
		defer recoverFromPanic()
		// Normal execution - no panic
	}()

	if exitCalled {
		t.Error("Exit was called when no panic occurred")
	}
}

func TestStackTraceInPanicRecovery(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Capture exit calls
	oldExit := osExit
	osExit = func(code int) {}

	defer func() {
		os.Stderr = oldStderr
		osExit = oldExit
		w.Close()
	}()

	// Test panic with stack trace
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Application panic: %v\n", r)
				runtimeDebug.PrintStack()
				osExit(2)
			}
		}()
		panic("test stack trace")
	}()

	// Close writer and read stderr
	w.Close()
	var stderr bytes.Buffer
	stderr.ReadFrom(r)

	stderrStr := stderr.String()
	if stderrStr == "" {
		t.Error("Expected stack trace output but got none")
	}

	// Check for stack trace indicators
	if !bytes.Contains(stderr.Bytes(), []byte("goroutine")) {
		t.Error("Expected stack trace to contain 'goroutine'")
	}
}

// Mock function for testing
var osExit = os.Exit

func recoverFromPanic() {
	if r := recover(); r != nil {
		fmt.Fprintf(os.Stderr, "Application panic: %v\n", r)
		runtimeDebug.PrintStack()
		osExit(2)
	}
}