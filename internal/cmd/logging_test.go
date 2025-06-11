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
			expectLogs:     []string{"[INFO]", "sync operation", "loaded 0 roles"},
			expectProgress: []string{"Processing roles..."},
		},
		{
			name:           "shows progress for empty directory",
			setupRoles:     []models.Role{},
			setupRemote:    []models.Role{},
			dryRun:         false,
			expectLogs:     []string{"[INFO]", "no roles found"},
			expectProgress: []string{"Processing roles..."},
		},
		{
			name: "provides debug information in verbose mode",
			setupRoles: []models.Role{
				{Name: "viewer", Resources: models.Resources{Allowed: []string{"read"}}},
			},
			setupRemote: []models.Role{},
			dryRun:      true,
			expectLogs:  []string{"[DEBUG]", "comparing roles", "plan generated"},
		},
		{
			name: "tracks operation timing",
			setupRoles: []models.Role{
				{Name: "editor", Resources: models.Resources{Allowed: []string{"read", "write"}}},
			},
			setupRemote:    []models.Role{},
			dryRun:         false,
			expectLogs:     []string{"[INFO]", "sync operation"},
			expectProgress: []string{"Processing roles..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockCalls := &MockAPICalls{}
			mockClient := NewMockClient(mockCalls, tt.setupRemote)

			// Create command with logging
			cmd := NewSyncCommandWithLogging(mockClient, true) // verbose = true
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			// Set dry-run flag
			if tt.dryRun {
				cmd.Flags().Set("dry-run", "true")
			}

			// Execute command (this will fail until we implement the logging)
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Command execution failed: %v", err)
			}

			// Check for expected log messages
			outputStr := output.String()
			for _, expectedLog := range tt.expectLogs {
				if !strings.Contains(outputStr, expectedLog) {
					t.Errorf("Expected log message '%s' not found in output:\n%s", expectedLog, outputStr)
				}
			}

			// Check for progress indicators
			for _, expectedProgress := range tt.expectProgress {
				if !strings.Contains(outputStr, expectedProgress) {
					t.Errorf("Expected progress message '%s' not found in output:\n%s", expectedProgress, outputStr)
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
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			// Execute command
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Command execution failed: %v", err)
			}

			// Check for debug messages
			outputStr := output.String()
			hasDebug := strings.Contains(outputStr, "[DEBUG]")

			if tt.expectDebug && !hasDebug {
				t.Errorf("Expected debug logs but none found in output:\n%s", outputStr)
			}
			if !tt.expectDebug && hasDebug {
				t.Errorf("Unexpected debug logs found in output:\n%s", outputStr)
			}
		})
	}
}

// NewSyncCommandWithLogging creates a sync command with logging support
func NewSyncCommandWithLogging(mockClient *MockClient, verbose bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create logger with command output
			logger := logging.NewLogger(cmd.OutOrStdout(), verbose)
			
			// Get dry-run flag
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			
			config := models.Config{
				APIEndpoint: "https://api.test.com",
				APIToken:    "test-token",
				Confirm:     false,
				LogLevel:    "info",
			}
			return RunSyncCommandWithLogging(cmd, args, mockClient, dryRun, "", logger, config)
		},
	}
	
	// Add flags
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().Bool("verbose", verbose, "enable verbose logging")
	
	return cmd
}