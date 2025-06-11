package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"replbac/internal/logging"
	"replbac/internal/models"
)

// TestMockRoundTripDataIntegrity tests complete data integrity through init -> modify -> sync cycle
func TestMockRoundTripDataIntegrity(t *testing.T) {
	tests := []struct {
		name              string
		initialAPIRoles   []models.Role
		localModifications map[string]string // filename -> new content
		expectedFinalAPI  []models.Role
		expectError       bool
	}{
		{
			name: "simple role round-trip preserves data",
			initialAPIRoles: []models.Role{
				{
					ID:   "test-id-123",
					Name: "test-role",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{"delete"},
					},
				},
			},
			localModifications: map[string]string{
				"test-role.yaml": `# WARNING: The 'id' field is managed by the Replicated API and should not be modified manually.
# Changing the ID will cause sync operations to fail.

id: test-id-123
name: test-role
resources:
    allowed:
        - read
        - write
        - create
    denied:
        - delete`,
			},
			expectedFinalAPI: []models.Role{
				{
					ID:   "test-id-123",
					Name: "test-role",
					Resources: models.Resources{
						Allowed: []string{"read", "write", "create"},
						Denied:  []string{"delete"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "complex role with special characters survives round-trip",
			initialAPIRoles: []models.Role{
				{
					ID:   "complex-id-456",
					Name: "complex-role_123",
					Resources: models.Resources{
						Allowed: []string{"resource:with:colons", "resource/with/slashes", "unicode-ñoñó"},
						Denied:  []string{"denied:resource", "denied/resource"},
					},
				},
			},
			localModifications: map[string]string{
				"complex-role_123.yaml": `# WARNING: The 'id' field is managed by the Replicated API and should not be modified manually.
# Changing the ID will cause sync operations to fail.

id: complex-id-456
name: complex-role_123
resources:
    allowed:
        - resource:with:colons
        - resource/with/slashes
        - unicode-ñoñó
        - new:special:resource
    denied:
        - denied:resource
        - denied/resource`,
			},
			expectedFinalAPI: []models.Role{
				{
					ID:   "complex-id-456",
					Name: "complex-role_123",
					Resources: models.Resources{
						Allowed: []string{"resource:with:colons", "resource/with/slashes", "unicode-ñoñó", "new:special:resource"},
						Denied:  []string{"denied:resource", "denied/resource"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "multiple roles round-trip with mixed operations",
			initialAPIRoles: []models.Role{
				{
					ID:   "keep-id-111",
					Name: "keep-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
				{
					ID:   "modify-id-222",
					Name: "modify-role",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{"delete"},
					},
				},
				{
					ID:   "delete-id-333",
					Name: "delete-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			localModifications: map[string]string{
				"keep-role.yaml": `# WARNING: The 'id' field is managed by the Replicated API and should not be modified manually.
# Changing the ID will cause sync operations to fail.

id: keep-id-111
name: keep-role
resources:
    allowed:
        - read
    denied: []`,
				"modify-role.yaml": `# WARNING: The 'id' field is managed by the Replicated API and should not be modified manually.
# Changing the ID will cause sync operations to fail.

id: modify-id-222
name: modify-role
resources:
    allowed:
        - read
        - write
        - create
    denied:
        - delete`,
				"new-role.yaml": `name: new-role
resources:
    allowed:
        - read
        - new
    denied: []`,
				// delete-role.yaml is intentionally omitted to test deletion
			},
			expectedFinalAPI: []models.Role{
				{
					ID:   "keep-id-111",
					Name: "keep-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
				{
					ID:   "modify-id-222",
					Name: "modify-role",
					Resources: models.Resources{
						Allowed: []string{"read", "write", "create"},
						Denied:  []string{"delete"},
					},
				},
				{
					ID:   "", // New role won't have ID until API assigns one
					Name: "new-role",
					Resources: models.Resources{
						Allowed: []string{"read", "new"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "replbac-roundtrip-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

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

			// Setup mock API client with initial state
			mockCalls := &MockAPICalls{}
			mockClient := NewMockClient(mockCalls, tt.initialAPIRoles)

			// Step 1: Run init command to download roles from API
			initCmd := NewInitCommand(mockClient)
			var initOutput bytes.Buffer
			initCmd.SetOut(&initOutput)
			initCmd.SetErr(&initOutput)
			initCmd.SetArgs([]string{})

			err = initCmd.Execute()
			if err != nil {
				t.Fatalf("Init command failed: %v", err)
			}


			// Verify init created expected files
			for _, role := range tt.initialAPIRoles {
				fileName := role.Name + ".yaml"
				if _, err := os.Stat(fileName); os.IsNotExist(err) {
					t.Errorf("Expected init to create file %s", fileName)
				}
			}

			// Step 2: Apply local modifications (simulate user editing files)
			for fileName, content := range tt.localModifications {
				err = os.WriteFile(fileName, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to write modified file %s: %v", fileName, err)
				}
			}

			// Step 2.5: Delete files that were intentionally omitted (simulate user deleting files)
			// Check for roles that were in the initial state but not in localModifications
			for _, initialRole := range tt.initialAPIRoles {
				fileName := initialRole.Name + ".yaml"
				if _, exists := tt.localModifications[fileName]; !exists {
					// This file should be deleted
					if _, err := os.Stat(fileName); err == nil {
						err = os.Remove(fileName)
						if err != nil {
							t.Fatalf("Failed to delete file %s: %v", fileName, err)
						}
					}
				}
			}

			// Step 3: Run sync command to upload changes back to API
			syncCmd := NewRoundTripSyncCommand(mockClient)
			var syncOutput bytes.Buffer
			syncCmd.SetOut(&syncOutput)
			syncCmd.SetErr(&syncOutput)
			syncCmd.SetArgs([]string{})

			err = syncCmd.Execute()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected sync error but got none")
				}
				return
			} else {
				if err != nil {
					t.Errorf("Unexpected sync error: %v", err)
				}
			}

			// Step 4: Verify final API state matches expectations
			finalRoles, err := mockClient.GetRoles()
			if err != nil {
				t.Fatalf("Failed to get final roles: %v", err)
			}

			// Validate final state
			if len(finalRoles) != len(tt.expectedFinalAPI) {
				t.Errorf("Expected %d final roles, got %d", len(tt.expectedFinalAPI), len(finalRoles))
			}

			for _, expectedRole := range tt.expectedFinalAPI {
				found := false
				for _, actualRole := range finalRoles {
					if actualRole.Name == expectedRole.Name {
						found = true
						// Check data integrity
						if expectedRole.ID != "" && actualRole.ID != expectedRole.ID {
							t.Errorf("Role %s: expected ID %s, got %s", expectedRole.Name, expectedRole.ID, actualRole.ID)
						}
						if !stringSlicesEqual(actualRole.Resources.Allowed, expectedRole.Resources.Allowed) {
							t.Errorf("Role %s: allowed resources mismatch. Expected %v, got %v", 
								expectedRole.Name, expectedRole.Resources.Allowed, actualRole.Resources.Allowed)
						}
						if !stringSlicesEqual(actualRole.Resources.Denied, expectedRole.Resources.Denied) {
							t.Errorf("Role %s: denied resources mismatch. Expected %v, got %v", 
								expectedRole.Name, expectedRole.Resources.Denied, actualRole.Resources.Denied)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected role %s not found in final API state", expectedRole.Name)
				}
			}
		})
	}
}

// TestYAMLSerializationFidelity tests that YAML serialization preserves all data correctly
func TestYAMLSerializationFidelity(t *testing.T) {
	tests := []struct {
		name string
		role models.Role
	}{
		{
			name: "role with ID and warning comments",
			role: models.Role{
				ID:   "test-id-with-dashes",
				Name: "test-role",
				Resources: models.Resources{
					Allowed: []string{"read", "write"},
					Denied:  []string{"delete"},
				},
			},
		},
		{
			name: "role with empty denied list",
			role: models.Role{
				ID:   "empty-denied-id",
				Name: "empty-denied",
				Resources: models.Resources{
					Allowed: []string{"*"},
					Denied:  []string{},
				},
			},
		},
		{
			name: "role with special characters",
			role: models.Role{
				ID:   "special-chars-id",
				Name: "special-chars_role-123",
				Resources: models.Resources{
					Allowed: []string{"resource:with:colons", "resource/with/slashes", "unicode-ñoñó"},
					Denied:  []string{"denied:resource", "denied/resource"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "replbac-yaml-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Write role to YAML file
			filePath := filepath.Join(tempDir, tt.role.Name+".yaml")
			err = WriteRoleFile(tt.role, filePath)
			if err != nil {
				t.Fatalf("Failed to write role file: %v", err)
			}

			// Read file content and verify structure
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read role file: %v", err)
			}

			contentStr := string(content)

			// Verify ID is present if set
			if tt.role.ID != "" {
				if !strings.Contains(contentStr, "id: "+tt.role.ID) {
					t.Errorf("Expected file to contain ID %s", tt.role.ID)
				}
				// Verify warning comment is present
				if !strings.Contains(contentStr, "WARNING:") {
					t.Errorf("Expected file to contain warning comment about ID")
				}
			}

			// Verify name is present
			if !strings.Contains(contentStr, "name: "+tt.role.Name) {
				t.Errorf("Expected file to contain name %s", tt.role.Name)
			}

			// Verify all allowed resources are present
			for _, resource := range tt.role.Resources.Allowed {
				// Handle special case for "*" which gets quoted in YAML
				expectedFormats := []string{
					"- " + resource,
					"- '" + resource + "'",
					"- \"" + resource + "\"",
				}
				found := false
				for _, format := range expectedFormats {
					if strings.Contains(contentStr, format) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected file to contain allowed resource %s in one of these formats: %v", resource, expectedFormats)
				}
			}

			// Verify all denied resources are present
			for _, resource := range tt.role.Resources.Denied {
				// Handle special case for "*" which gets quoted in YAML
				expectedFormats := []string{
					"- " + resource,
					"- '" + resource + "'",
					"- \"" + resource + "\"",
				}
				found := false
				for _, format := range expectedFormats {
					if strings.Contains(contentStr, format) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected file to contain denied resource %s in one of these formats: %v", resource, expectedFormats)
				}
			}
		})
	}
}

// Helper functions for round-trip testing

// NewRoundTripSyncCommand creates a sync command for round-trip testing with confirm support
func NewRoundTripSyncCommand(mockClient *MockClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Synchronize local role files to Replicated API",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			rolesDir, _ := cmd.Flags().GetString("roles-dir")
			confirm, _ := cmd.Flags().GetBool("confirm")
			
			// For round-trip tests, automatically confirm destructive operations
			config := models.Config{
				APIEndpoint: "https://api.test.com",
				APIToken:    "test-token",
				Confirm:     confirm || true, // Always confirm in tests
				LogLevel:    "info",
			}
			
			// Use the logging version which supports confirmation
			logger := logging.NewLogger(cmd.OutOrStdout(), false)
			return RunSyncCommandWithLogging(cmd, args, mockClient, dryRun, rolesDir, logger, config)
		},
	}
	
	// Add flags
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().String("roles-dir", "", "directory containing role YAML files")
	cmd.Flags().Bool("confirm", true, "automatically confirm destructive operations")
	
	return cmd
}

// stringSlicesEqual compares two string slices for equality
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}