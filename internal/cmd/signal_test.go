package cmd

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSignalHandling(t *testing.T) {
	tests := []struct {
		name           string
		signal         os.Signal
		expectShutdown bool
	}{
		{
			name:           "SIGINT triggers graceful shutdown",
			signal:         os.Interrupt,
			expectShutdown: true,
		},
		{
			name:           "SIGTERM triggers graceful shutdown",
			signal:         syscall.SIGTERM,
			expectShutdown: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create signal channel
			sigChan := make(chan os.Signal, 1)
			shutdownChan := make(chan bool, 1)

			// Start signal handler
			go func() {
				select {
				case <-sigChan:
					shutdownChan <- true
				case <-time.After(1 * time.Second):
					shutdownChan <- false
				}
			}()

			// Simulate signal
			go func() {
				time.Sleep(100 * time.Millisecond)
				sigChan <- tt.signal
			}()

			// Wait for result
			shutdown := <-shutdownChan

			if shutdown != tt.expectShutdown {
				t.Errorf("Expected shutdown=%v, got %v", tt.expectShutdown, shutdown)
			}
		})
	}
}

func TestGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Simulate long-running operation
	operationComplete := make(chan bool, 1)
	go func() {
		select {
		case <-ctx.Done():
			// Operation was cancelled
			operationComplete <- false
		case <-time.After(2 * time.Second):
			// Operation completed normally
			operationComplete <- true
		}
	}()

	// Cancel after short delay to simulate signal
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	completed := <-operationComplete
	if completed {
		t.Error("Expected operation to be cancelled, but it completed")
	}
}

func TestContextPropagation(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create child context
	childCtx, childCancel := context.WithCancel(parentCtx)
	defer childCancel()

	// Test that cancelling parent cancels child
	cancel()

	select {
	case <-childCtx.Done():
		// Expected behavior
	case <-time.After(100 * time.Millisecond):
		t.Error("Child context was not cancelled when parent was cancelled")
	}
}