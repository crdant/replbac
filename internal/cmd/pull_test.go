package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"replbac/internal/models"
)

// TestPullCommand tests the complete pull command workflow with new flags
func TestPullCommand(t *testing.T) {
	tests := []struct {
		name                 string
		args                 []string
		flags                map[string]string
		mockAPIRoles         []models.Role
		existingFiles        map[string]string
		expectError          bool
		expectOutput         []string
		expectFiles          map[string]string
		expectNoFiles        []string
		validateAPICallsFunc func(t *testing.T, calls *MockAPICalls)
	}{
		{
			name: "pull empty directory - creates all role files",
			args: []string{},
			mockAPIRoles: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
				{
					Name: "viewer",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{"write", "delete"},
					},
				},
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 2 role(s) from API",
				"Created admin.yaml",
				"Created viewer.yaml",
				"Pull completed successfully",
			},
			expectFiles: map[string]string{
				"admin.yaml":  "name: admin\nresources:\n    allowed:\n        - '*'\n    denied: []\n",
				"viewer.yaml": "name: viewer\nresources:\n    allowed:\n        - read\n    denied:\n        - write\n        - delete\n",
			},
			validateAPICallsFunc: func(t *testing.T, calls *MockAPICalls) {
				if calls.GetCalls != 1 {
					t.Errorf("Expected 1 GetRoles call, got %d", calls.GetCalls)
				}
			},
		},
		{
			name: "pull with custom directory argument",
			args: []string{"custom-dir"},
			mockAPIRoles: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 1 role(s) from API",
				"Created custom-dir/admin.yaml",
				"Pull completed successfully",
			},
			expectFiles: map[string]string{
				"custom-dir/admin.yaml": "name: admin\nresources:\n    allowed:\n        - '*'\n    denied: []\n",
			},
		},
		{
			name: "pull with existing files - preserves without force",
			args: []string{},
			mockAPIRoles: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
				{
					Name: "viewer",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			existingFiles: map[string]string{
				"admin.yaml": "# existing admin file\nname: admin\nresources:\n  allowed: [\"old\"]\n",
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 2 role(s) from API",
				"Skipped admin.yaml (file already exists)",
				"Created viewer.yaml",
				"Pull completed: 1 created, 1 skipped",
			},
			expectFiles: map[string]string{
				"admin.yaml":  "# existing admin file\nname: admin\nresources:\n  allowed: [\"old\"]\n",
				"viewer.yaml": "name: viewer\nresources:\n    allowed:\n        - read\n    denied: []\n",
			},
		},
		{
			name:  "pull with existing files - overwrites with force",
			args:  []string{},
			flags: map[string]string{"force": "true"},
			mockAPIRoles: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
			},
			existingFiles: map[string]string{
				"admin.yaml": "# existing admin file\nname: admin\nresources:\n  allowed: [\"old\"]\n",
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 1 role(s) from API",
				"Overwrote admin.yaml",
				"Pull completed successfully",
			},
			expectFiles: map[string]string{
				"admin.yaml": "name: admin\nresources:\n    allowed:\n        - '*'\n    denied: []\n",
			},
		},
		{
			name:  "pull with dry-run flag - shows what would be done",
			args:  []string{},
			flags: map[string]string{"dry-run": "true"},
			mockAPIRoles: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 1 role(s) from API",
				"Would create admin.yaml",
				"Pull completed (dry-run): 1 would be created",
			},
			expectNoFiles: []string{"admin.yaml"},
		},
		{
			name:  "pull with diff flag - shows detailed differences",
			args:  []string{},
			flags: map[string]string{"diff": "true"},
			mockAPIRoles: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
			},
			existingFiles: map[string]string{
				"admin.yaml": "name: admin\nresources:\n  allowed: [\"read\"]\n  denied: []\n",
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 1 role(s) from API",
				"Would update admin.yaml",
				"Pull completed (dry-run): 1 would be updated",
			},
			expectFiles: map[string]string{
				"admin.yaml": "name: admin\nresources:\n  allowed: [\"read\"]\n  denied: []\n",
			},
		},
		{
			name:         "pull empty API - no files created",
			args:         []string{},
			flags:        map[string]string{},
			mockAPIRoles: []models.Role{},
			expectError:  false,
			expectOutput: []string{
				"No roles found in API",
				"Pull completed: no files created",
			},
		},
		{
			name:         "pull with API error",
			args:         []string{},
			mockAPIRoles: nil, // This will trigger an error in the mock
			expectError:  true,
			expectOutput: []string{
				"Failed to fetch roles from API",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "replbac-pull-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tempDir) }()

			// Change to temp directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(oldDir); err != nil {
					t.Fatalf("Failed to restore working directory: %v", err)
				}
			}()

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Create existing files if specified
			for fileName, content := range tt.existingFiles {
				filePath := fileName
				if strings.Contains(fileName, "/") {
					// Create directory structure if needed
					// #nosec G301 -- Test directories need readable permissions
					if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
						t.Fatalf("Failed to create directory structure: %v", err)
					}
				}
				// #nosec G306 -- Test files need readable permissions
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create existing file %s: %v", fileName, err)
				}
			}

			// Create mock client
			mockCalls := &MockAPICalls{}
			var mockClient *MockClient
			if tt.mockAPIRoles == nil {
				// Create a mock that returns an error
				mockClient = &MockClient{
					calls:       mockCalls,
					roles:       []models.Role{},
					shouldError: true,
				}
			} else {
				mockClient = NewMockClient(mockCalls, tt.mockAPIRoles)
			}

			// Create pull command
			cmd := NewPullCommand(mockClient)

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

			// Check expected files
			for fileName, expectedContent := range tt.expectFiles {
				// #nosec G304 -- Reading test file path is expected behavior in tests
				actualContent, err := os.ReadFile(fileName)
				if err != nil {
					t.Errorf("Expected file %s to exist but couldn't read it: %v", fileName, err)
					continue
				}
				if string(actualContent) != expectedContent {
					t.Errorf("File %s content mismatch.\nExpected:\n%s\nActual:\n%s", fileName, expectedContent, string(actualContent))
				}
			}

			// Check files that should not exist
			for _, fileName := range tt.expectNoFiles {
				if _, err := os.Stat(fileName); err == nil {
					t.Errorf("Expected file %s to not exist, but it does", fileName)
				}
			}

			// Validate API calls if function provided
			if tt.validateAPICallsFunc != nil {
				tt.validateAPICallsFunc(t, mockCalls)
			}
		})
	}
}

// NewPullCommand creates a pull command for testing
func NewPullCommand(mockClient *MockClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull [directory]",
		Short: "Pull role definitions from Replicated API to local files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			diff, _ := cmd.Flags().GetBool("diff")
			force, _ := cmd.Flags().GetBool("force")

			// Determine target directory
			targetDir := "."
			if len(args) > 0 {
				targetDir = args[0]
			}

			// If diff is enabled, enable dry-run too
			effectiveDryRun := dryRun || diff

			return RunPullCommandWithClient(cmd, targetDir, effectiveDryRun, diff, force, mockClient)
		},
	}

	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().Bool("diff", false, "preview changes with detailed diffs (implies --dry-run)")
	cmd.Flags().Bool("force", false, "overwrite existing files")
	cmd.Flags().Bool("verbose", false, "enable verbose logging")

	return cmd
}

// This function is now implemented in pull.go and uses api.ClientInterface
