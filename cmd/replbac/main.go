package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"replbac/internal/cmd"
)

func main() {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Application panic: %v\n", r)
			debug.PrintStack()
			os.Exit(2)
		}
	}()

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Fprintf(os.Stderr, "\nReceived signal %v, shutting down gracefully...\n", sig)
		cancel()
	}()

	// Execute command with context
	if err := cmd.ExecuteWithContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
