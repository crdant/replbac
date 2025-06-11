package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"replbac/internal/models"
	"replbac/internal/roles"
)

// TestInitCommand tests the complete init command workflow
func TestInitCommand(t *testing.T) {
	tests := []struct {
		name                string
		args                []string
		flags               map[string]string
		mockAPIRoles        []models.Role
		existingFiles       map[string]string
		expectError         bool
		expectOutput        []string
		expectFiles         map[string]string
		expectNoFiles       []string
		validateAPICallsFunc func(t *testing.T, calls *MockAPICalls)
	}{
		{
			name: "init empty directory - creates all role files",
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
				"Initialization completed successfully",
			},
			expectFiles: map[string]string{
				"admin.yaml": `name: admin
resources:
    allowed:
        - '*'
    denied: []`,
				"viewer.yaml": `name: viewer
resources:
    allowed:
        - read
    denied:
        - write
        - delete`,
			},
			validateAPICallsFunc: func(t *testing.T, calls *MockAPICalls) {
				if calls.GetCalls != 1 {
					t.Errorf("Expected 1 GetRoles call, got %d", calls.GetCalls)
				}
			},
		},
		{
			name: "init with custom directory argument",
			args: []string{"roles"},
			mockAPIRoles: []models.Role{
				{
					Name: "custom",
					Resources: models.Resources{
						Allowed: []string{"custom"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 1 role(s) from API",
				"Created roles/custom.yaml",
				"Initialization completed successfully",
			},
			expectFiles: map[string]string{
				"roles/custom.yaml": `name: custom
resources:
    allowed:
        - custom
    denied: []`,
			},
		},
		{
			name:  "init with output-dir flag",
			args:  []string{},
			flags: map[string]string{"output-dir": "output"},
			mockAPIRoles: []models.Role{
				{
					Name: "flagged",
					Resources: models.Resources{
						Allowed: []string{"flag"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 1 role(s) from API",
				"Created output/flagged.yaml",
			},
			expectFiles: map[string]string{
				"output/flagged.yaml": `name: flagged
resources:
    allowed:
        - flag
    denied: []`,
			},
		},
		{
			name: "init with existing files - preserves without force",
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
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{},
					},
				},
			},
			existingFiles: map[string]string{
				"admin.yaml": "# existing admin file",
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 2 role(s) from API",
				"Skipped admin.yaml (file already exists)",
				"Created editor.yaml",
				"Initialization completed: 1 created, 1 skipped",
			},
			expectFiles: map[string]string{
				"admin.yaml": "# existing admin file", // Should preserve existing
				"editor.yaml": `name: editor
resources:
    allowed:
        - read
        - write
    denied: []`,
			},
		},
		{
			name:  "init with existing files - overwrites with force",
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
				"admin.yaml": "# existing admin file",
			},
			expectError: false,
			expectOutput: []string{
				"Downloaded 1 role(s) from API",
				"Overwrote admin.yaml",
				"Initialization completed successfully",
			},
			expectFiles: map[string]string{
				"admin.yaml": `name: admin
resources:
    allowed:
        - '*'
    denied: []`,
			},
		},
		{
			name:         "init empty API - no files created",
			args:         []string{},
			mockAPIRoles: []models.Role{},
			expectError:  false,
			expectOutput: []string{
				"No roles found in API",
				"Initialization completed: no files created",
			},
			expectFiles: map[string]string{},
		},
		{
			name: "init with API error",
			args: []string{},
			// mockAPIRoles left nil to trigger API error
			expectError: true,
			expectOutput: []string{
				"Failed to fetch roles from API",
			},
		},
		{
			name: "init with role containing special characters",
			args: []string{},
			mockAPIRoles: []models.Role{
				{
					Name: "special-chars_123",
					Resources: models.Resources{
						Allowed: []string{"resource:with:colons", "resource/with/slashes"},
						Denied:  []string{"denied:resource"},
					},
				},
			},
			expectError: false,
			expectFiles: map[string]string{
				"special-chars_123.yaml": `name: special-chars_123
resources:
    allowed:
        - resource:with:colons
        - resource/with/slashes
    denied:
        - denied:resource`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "replbac-init-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create existing files if specified
			for fileName, content := range tt.existingFiles {
				filePath := filepath.Join(tempDir, fileName)
				fileDir := filepath.Dir(filePath)
				err := os.MkdirAll(fileDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create file dir: %v", err)
				}
				err = os.WriteFile(filePath, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to write existing file: %v", err)
				}
			}

			// Setup mock API
			mockCalls := &MockAPICalls{}
			var mockClient *MockClient
			if tt.mockAPIRoles != nil {
				mockClient = NewMockClient(mockCalls, tt.mockAPIRoles)
			} else {
				// Create client that will return error
				mockClient = NewMockClient(mockCalls, []models.Role{})
				mockClient.shouldError = true
			}

			// Change to temp directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current dir: %v", err)
			}
			defer os.Chdir(oldDir)
			err = os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Setup command with captured output
			cmd := NewInitCommand(mockClient)
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			// Set flags
			for flag, value := range tt.flags {
				cmd.Flags().Set(flag, value)
			}

			// Execute command
			cmd.SetArgs(tt.args)
			err = cmd.Execute()

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check output
			outputStr := output.String()
			for _, expected := range tt.expectOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain '%s', got:\n%s", expected, outputStr)
				}
			}

			// Check expected files were created with correct content
			for expectedFile, expectedContent := range tt.expectFiles {
				filePath := filepath.Join(tempDir, expectedFile)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected file %s to exist", expectedFile)
					continue
				}

				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("Failed to read file %s: %v", expectedFile, err)
					continue
				}

				if strings.TrimSpace(string(content)) != strings.TrimSpace(expectedContent) {
					t.Errorf("File %s content mismatch.\nExpected:\n%s\nGot:\n%s", 
						expectedFile, expectedContent, string(content))
				}
			}

			// Check that files we don't expect are not created
			for _, unexpectedFile := range tt.expectNoFiles {
				filePath := filepath.Join(tempDir, unexpectedFile)
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					t.Errorf("Expected file %s to not exist", unexpectedFile)
				}
			}

			// Validate API calls
			if tt.validateAPICallsFunc != nil {
				tt.validateAPICallsFunc(t, mockCalls)
			}
		})
	}
}

// TestInitCommandDirectCalls tests the core functions directly
func TestInitCommandDirectCalls(t *testing.T) {
	t.Run("WriteRoleFile creates valid YAML", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "replbac-write-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Test role
		role := models.Role{
			Name: "test-role",
			Resources: models.Resources{
				Allowed: []string{"read", "write"},
				Denied:  []string{"delete"},
			},
		}

		// Write role file
		filePath := filepath.Join(tempDir, "test-role.yaml")
		err = WriteRoleFile(role, filePath)
		if err != nil {
			t.Fatalf("Unexpected error writing role file: %v", err)
		}

		// Verify file exists and has correct content
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read written file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "name: test-role") {
			t.Error("Expected content to contain role name")
		}
		if !strings.Contains(contentStr, "- read") {
			t.Error("Expected content to contain read permission")
		}
		if !strings.Contains(contentStr, "- delete") {
			t.Error("Expected content to contain denied permission")
		}
	})
}

// TestWriteRoleFile tests YAML file generation
func TestWriteRoleFile(t *testing.T) {
	tests := []struct {
		name         string
		role         models.Role
		expectError  bool
		validateContent func(t *testing.T, content []byte)
	}{
		{
			name: "simple role with allowed resources",
			role: models.Role{
				Name: "test-role",
				Resources: models.Resources{
					Allowed: []string{"read", "write"},
					Denied:  []string{},
				},
			},
			expectError: false,
			validateContent: func(t *testing.T, content []byte) {
				contentStr := string(content)
				if !strings.Contains(contentStr, "name: test-role") {
					t.Error("Expected content to contain role name")
				}
				if !strings.Contains(contentStr, "- read") {
					t.Error("Expected content to contain read permission")
				}
				if !strings.Contains(contentStr, "- write") {
					t.Error("Expected content to contain write permission")
				}
			},
		},
		{
			name: "role with denied resources",
			role: models.Role{
				Name: "restricted",
				Resources: models.Resources{
					Allowed: []string{"read"},
					Denied:  []string{"write", "delete"},
				},
			},
			expectError: false,
			validateContent: func(t *testing.T, content []byte) {
				contentStr := string(content)
				if !strings.Contains(contentStr, "name: restricted") {
					t.Error("Expected content to contain role name")
				}
				if !strings.Contains(contentStr, "- write") || !strings.Contains(contentStr, "- delete") {
					t.Error("Expected content to contain denied resources")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "replbac-write-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Write role file
			filePath := filepath.Join(tempDir, fmt.Sprintf("%s.yaml", tt.role.Name))
			err = WriteRoleFile(tt.role, filePath)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Read and validate file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read written file: %v", err)
			}

			if tt.validateContent != nil {
				tt.validateContent(t, content)
			}
		})
	}
}

// Support types and functions for init testing

func NewInitCommand(mockClient *MockClient) *cobra.Command {
	// This will be a test version of the init command
	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize local role files from Replicated API",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flag values
			force, _ := cmd.Flags().GetBool("force")
			outputDir, _ := cmd.Flags().GetString("output-dir")
			
			// Determine output directory
			targetDir := "."
			if len(args) > 0 {
				targetDir = args[0]
			}
			if outputDir != "" {
				targetDir = outputDir
			}
			
			return RunInitCommandWithClient(cmd, targetDir, force, mockClient)
		},
	}
	
	// Add flags
	cmd.Flags().Bool("force", false, "overwrite existing files")
	cmd.Flags().String("output-dir", "", "directory to create role files")
	
	return cmd
}

// WriteRoleFile is an alias to the roles package function for testing
func WriteRoleFile(role models.Role, filePath string) error {
	return roles.WriteRoleFile(role, filePath)
}