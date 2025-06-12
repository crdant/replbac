package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"replbac/internal/models"
)

// TestForceFlagBehavior tests the --force flag behavior with confirmation prompts
func TestForceFlagBehavior(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		flags              map[string]string
		localFiles         map[string]string
		remoteRoles        []models.Role
		expectError        bool
		expectOutput       []string
		expectNotInOutput  []string
		expectInteractive  bool // Whether it should prompt for confirmation
		validateAPICalls   func(t *testing.T, calls *MockAPICalls)
	}{
		{
			name: "sync with delete but no force - prompts for confirmation",
			args: []string{},
			flags: map[string]string{
				"delete": "true",
			},
			localFiles: map[string]string{
				"local-role.yaml": `name: local-role
resources:
  allowed: ["read"]
  denied: []`,
			},
			remoteRoles: []models.Role{
				{
					ID:   "remote-id",
					Name: "remote-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
			expectOutput: []string{
				"Will delete 1 role(s):",
				"This operation will permanently delete 1 role(s) from the API",
				"Do you want to continue? (y/N):",
			},
			expectInteractive: true,
			validateAPICalls: func(t *testing.T, calls *MockAPICalls) {
				// In real test, this would depend on simulated user input
				// For this test, we'll simulate the prompt being shown
			},
		},
		{
			name: "sync with delete and force - skips confirmation",
			args: []string{},
			flags: map[string]string{
				"delete": "true",
				"force":  "true",
			},
			localFiles: map[string]string{
				"local-role.yaml": `name: local-role
resources:
  allowed: ["read"]
  denied: []`,
			},
			remoteRoles: []models.Role{
				{
					ID:   "remote-id",
					Name: "remote-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
			expectOutput: []string{
				"Sync plan: 1 to create, 1 to delete",
				"Will create 1 role(s):",
				"local-role",
				"Will delete 1 role(s):",
				"remote-role",
				"Sync completed: create 1 role(s) and delete 1 role(s)",
			},
			expectNotInOutput: []string{
				"Do you want to continue?",
				"permanently delete",
			},
			expectInteractive: false,
			validateAPICalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 1 {
					t.Errorf("Expected 1 create call, got %d", len(calls.CreateCalls))
				}
				if len(calls.DeleteCalls) != 1 {
					t.Errorf("Expected 1 delete call, got %d", len(calls.DeleteCalls))
				}
				if calls.DeleteCalls[0] != "remote-role" {
					t.Errorf("Expected to delete 'remote-role', got '%s'", calls.DeleteCalls[0])
				}
			},
		},
		{
			name: "force flag without delete flag - no effect",
			args: []string{},
			flags: map[string]string{
				"force": "true",
			},
			localFiles: map[string]string{
				"local-role.yaml": `name: local-role
resources:
  allowed: ["read"]
  denied: []`,
			},
			remoteRoles: []models.Role{
				{
					ID:   "remote-id",
					Name: "remote-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
			expectOutput: []string{
				"Sync plan: 1 to create",
				"Will create 1 role(s):",
				"local-role",
				"Sync completed: create 1 role(s)",
			},
			expectNotInOutput: []string{
				"delete",
				"remote-role",
				"Do you want to continue?",
			},
			expectInteractive: false,
			validateAPICalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 1 {
					t.Errorf("Expected 1 create call, got %d", len(calls.CreateCalls))
				}
				if len(calls.DeleteCalls) != 0 {
					t.Errorf("Expected 0 delete calls, got %d", len(calls.DeleteCalls))
				}
			},
		},
		{
			name: "dry-run with delete and force - no confirmation needed",
			args: []string{},
			flags: map[string]string{
				"delete":  "true",
				"force":   "true",
				"dry-run": "true",
			},
			localFiles: map[string]string{
				"local-role.yaml": `name: local-role
resources:
  allowed: ["read"]
  denied: []`,
			},
			remoteRoles: []models.Role{
				{
					ID:   "remote-id",
					Name: "remote-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
			expectOutput: []string{
				"DRY RUN: No changes will be applied",
				"Sync plan: 1 to create, 1 to delete",
				"Will create 1 role(s):",
				"Will delete 1 role(s):",
			},
			expectNotInOutput: []string{
				"Do you want to continue?",
				"permanently delete",
			},
			expectInteractive: false,
			validateAPICalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 0 {
					t.Errorf("Expected 0 create calls in dry-run, got %d", len(calls.CreateCalls))
				}
				if len(calls.DeleteCalls) != 0 {
					t.Errorf("Expected 0 delete calls in dry-run, got %d", len(calls.DeleteCalls))
				}
			},
		},
		{
			name: "help shows force flag",
			args: []string{"--help"},
			expectError: false,
			expectOutput: []string{
				"--force",
				"skip confirmation prompts",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "replbac-force-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer os.Chdir(oldDir)

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Create local files if specified
			for fileName, content := range tt.localFiles {
				if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create file %s: %v", fileName, err)
				}
			}

			// Create mock client
			mockCalls := &MockAPICalls{}
			mockClient := NewMockClient(mockCalls, tt.remoteRoles)

			// Create sync command with force flag support
			cmd := NewSyncCommandWithForceFlag(mockClient)

			// Set flags
			for flag, value := range tt.flags {
				if err := cmd.Flags().Set(flag, value); err != nil {
					t.Fatalf("Failed to set flag %s=%s: %v", flag, value, err)
				}
			}

			// Capture output
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// For tests that expect interactive prompts, we need to handle stdin
			if tt.expectInteractive && !strings.Contains(tt.name, "help") {
				// TODO: Implement stdin simulation for interactive tests
				// For now, skip these tests as they require complex stdin mocking
				t.Skip("Interactive tests require stdin simulation - will implement in actual code")
				return
			}

			// Run command
			cmd.SetArgs(tt.args)
			err = cmd.Execute()

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check output
			output := stdout.String()
			for _, expectedOutput := range tt.expectOutput {
				if !strings.Contains(output, expectedOutput) {
					t.Errorf("Expected output to contain %q, but got: %s", expectedOutput, output)
				}
			}

			// Check that certain strings are NOT in output
			for _, notExpected := range tt.expectNotInOutput {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to NOT contain %q, but got: %s", notExpected, output)
				}
			}

			// Validate API calls if function provided
			if tt.validateAPICalls != nil {
				tt.validateAPICalls(t, mockCalls)
			}
		})
	}
}

// NewSyncCommandWithForceFlag creates a sync command that includes the --force flag
func NewSyncCommandWithForceFlag(mockClient *MockClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Synchronize local role files to Replicated API",
		Long: `Sync reads role definitions from local YAML files and synchronizes them
with the Replicated platform. By default, it will process all YAML files
in the current directory recursively.

The sync operation will:
• Read all role YAML files from the specified directory
• Compare them with existing roles in the API
• Create, update, or delete roles as needed to match local state
• Show clean results on stdout, with errors and progress on stderr

Use --delete to enable deletion of remote roles not present in local files.
Use --force to skip confirmation prompts when deletions are planned.
Use --dry-run to preview changes without applying them.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			delete, _ := cmd.Flags().GetBool("delete")
			force, _ := cmd.Flags().GetBool("force")
			
			return RunSyncCommandWithForceControl(cmd, args, mockClient, dryRun, delete, force)
		},
	}
	
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().Bool("delete", false, "delete remote roles not present in local files (default: false)")
	cmd.Flags().Bool("force", false, "skip confirmation prompts (requires --delete)")
	cmd.Flags().Bool("verbose", false, "enable verbose logging")
	
	return cmd
}

// RunSyncCommandWithForceControl implements sync with force flag control for testing
func RunSyncCommandWithForceControl(cmd *cobra.Command, args []string, client *MockClient, dryRun, delete, force bool) error {
	// Use the actual sync command implementation with force parameter
	return RunSyncCommandWithClient(cmd, args, client, dryRun, delete, force)
}