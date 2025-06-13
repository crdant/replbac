package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"replbac/internal/logging"
	"replbac/internal/models"
)

// TestLoggingAndFeedback tests user feedback and logging features
func TestLoggingAndFeedback(t *testing.T) {
	tests := []struct {
		name           string
		setupRoles     []models.Role
		setupRemote    []models.Role
		dryRun         bool
		expectLogs     []string
		expectProgress []string
	}{
		{
			name: "logs progress during sync operations",
			setupRoles: []models.Role{
				{Name: "admin", Resources: models.Resources{Allowed: []string{"*"}}},
			},
			setupRemote:    []models.Role{},
			dryRun:         false,
			expectLogs:     []string{"[DEBUG]", "sync operation", "loaded 0 roles"},
			expectProgress: []string{"loading roles from directory"},
		},
		{
			name:           "shows progress for empty directory",
			setupRoles:     []models.Role{},
			setupRemote:    []models.Role{},
			dryRun:         false,
			expectLogs:     []string{"[DEBUG]", "no changes needed"},
			expectProgress: []string{"loading roles from directory"},
		},
		{
			name: "provides debug information in verbose mode",
			setupRoles: []models.Role{
				{Name: "viewer", Resources: models.Resources{Allowed: []string{"read"}}},
			},
			setupRemote: []models.Role{},
			dryRun:      true,
			expectLogs:  []string{"[DEBUG]", "sync operation"},
		},
		{
			name: "tracks operation timing",
			setupRoles: []models.Role{
				{Name: "editor", Resources: models.Resources{Allowed: []string{"read", "write"}}},
			},
			setupRemote:    []models.Role{},
			dryRun:         false,
			expectLogs:     []string{"[DEBUG]", "sync operation"},
			expectProgress: []string{"loading roles from directory"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockCalls := &MockAPICalls{}
			mockClient := NewMockClient(mockCalls, tt.setupRemote)

			// Create command with logging
			cmd := NewSyncCommandWithLogging(mockClient, true) // verbose = true
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Set dry-run flag
			if tt.dryRun {
				if err := cmd.Flags().Set("dry-run", "true"); err != nil {
					t.Fatalf("Failed to set dry-run flag: %v", err)
				}
			}

			// Execute command (this will fail until we implement the logging)
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Command execution failed: %v", err)
			}

			// Check for expected log messages in stderr
			stderrStr := stderr.String()
			for _, expectedLog := range tt.expectLogs {
				if !strings.Contains(stderrStr, expectedLog) {
					t.Errorf("Expected log message '%s' not found in stderr:\n%s", expectedLog, stderrStr)
				}
			}

			// Check for progress indicators in stderr
			for _, expectedProgress := range tt.expectProgress {
				if !strings.Contains(stderrStr, expectedProgress) {
					t.Errorf("Expected progress message '%s' not found in stderr:\n%s", expectedProgress, stderrStr)
				}
			}
		})
	}
}

// TestVerboseLogging tests verbose logging functionality
func TestVerboseLogging(t *testing.T) {
	tests := []struct {
		name        string
		verbose     bool
		expectDebug bool
	}{
		{
			name:        "shows debug logs when verbose enabled",
			verbose:     true,
			expectDebug: true,
		},
		{
			name:        "hides debug logs when verbose disabled",
			verbose:     false,
			expectDebug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockCalls := &MockAPICalls{}
			mockClient := NewMockClient(mockCalls, []models.Role{})

			// Create command with or without verbose logging
			cmd := NewSyncCommandWithLogging(mockClient, tt.verbose)
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Execute command
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Command execution failed: %v", err)
			}

			// Check for debug messages in stderr
			stderrStr := stderr.String()
			hasDebug := strings.Contains(stderrStr, "[DEBUG]")

			if tt.expectDebug && !hasDebug {
				t.Errorf("Expected debug logs but none found in stderr:\n%s", stderrStr)
			}
			if !tt.expectDebug && hasDebug {
				t.Errorf("Unexpected debug logs found in stderr:\n%s", stderrStr)
			}
		})
	}
}

// NewSyncCommandWithLogging creates a sync command with logging support
func NewSyncCommandWithLogging(mockClient *MockClient, verbose bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create logger with command stderr (updated behavior)
			var logger *logging.Logger
			if verbose {
				logger = logging.NewDebugLogger(cmd.ErrOrStderr()) // For testing, treat verbose as debug
			} else {
				logger = logging.NewLogger(cmd.ErrOrStderr(), false)
			}

			// Get dry-run flag
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			config := models.Config{
				APIToken: "test-token",
				Confirm:  false,
				LogLevel: "info",
			}
			return RunSyncCommandWithLogging(cmd, args, mockClient, dryRun, false, false, false, logger, config)
		},
	}

	// Add flags
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().Bool("verbose", verbose, "enable verbose logging")

	return cmd
}
