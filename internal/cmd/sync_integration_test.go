package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"replbac/internal/models"
)

// TestSyncCommandIntegration tests the complete sync command workflow
func TestSyncCommandIntegration(t *testing.T) {
	tests := []struct {
		name           string
		files          map[string]string
		mockAPIRoles   []models.Role
		args           []string
		flags          map[string]string
		expectError    bool
		expectOutput   []string
		expectNoOutput []string
		validateCalls  func(t *testing.T, calls *MockAPICalls)
	}{
		{
			name: "sync single role file - dry run",
			files: map[string]string{
				"admin.yaml": `name: admin
resources:
  allowed: ["*"]
  denied: []`,
			},
			mockAPIRoles: []models.Role{},
			args:         []string{},
			flags: map[string]string{
				"dry-run": "true",
			},
			expectError: false,
			expectOutput: []string{
				"DRY RUN",
				"Would create 1 role(s)",
				"admin",
			},
			validateCalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 0 {
					t.Errorf("Expected no create calls in dry run, got %d", len(calls.CreateCalls))
				}
			},
		},
		{
			name: "sync single role file - real execution",
			files: map[string]string{
				"viewer.yaml": `name: viewer
resources:
  allowed: ["read"]
  denied: ["write", "delete"]`,
			},
			mockAPIRoles: []models.Role{},
			args:         []string{},
			flags:        map[string]string{},
			expectError:  false,
			expectOutput: []string{
				"create 1 role(s)",
				"viewer",
			},
			validateCalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 1 {
					t.Errorf("Expected 1 create call, got %d", len(calls.CreateCalls))
				}
				if len(calls.CreateCalls) > 0 && calls.CreateCalls[0].Name != "viewer" {
					t.Errorf("Expected to create 'viewer', got '%s'", calls.CreateCalls[0].Name)
				}
			},
		},
		{
			name:  "sync with updates and deletes",
			files: map[string]string{
				"admin.yaml": `name: admin
resources:
  allowed: ["*"]
  denied: []`,
				"editor.yaml": `name: editor
resources:
  allowed: ["read", "write", "create"]
  denied: ["delete"]`,
			},
			mockAPIRoles: []models.Role{
				{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{},
					},
				},
				{
					Name: "obsolete",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			args:        []string{},
			flags:       map[string]string{},
			expectError: false,
			expectOutput: []string{
				"create 1 role(s), update 1 role(s), and delete 1 role(s)",
				"admin",
				"editor",
				"obsolete",
			},
			validateCalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 1 {
					t.Errorf("Expected 1 create call, got %d", len(calls.CreateCalls))
				}
				if len(calls.UpdateCalls) != 1 {
					t.Errorf("Expected 1 update call, got %d", len(calls.UpdateCalls))
				}
				if len(calls.DeleteCalls) != 1 {
					t.Errorf("Expected 1 delete call, got %d", len(calls.DeleteCalls))
				}
			},
		},
		{
			name: "sync with custom directory",
			files: map[string]string{
				"roles/custom.yaml": `name: custom
resources:
  allowed: ["custom"]
  denied: []`,
			},
			mockAPIRoles: []models.Role{},
			args:         []string{"roles"},
			flags:        map[string]string{},
			expectError:  false,
			expectOutput: []string{
				"Synchronizing roles from directory: roles",
				"create 1 role(s)",
				"custom",
			},
			validateCalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 1 {
					t.Errorf("Expected 1 create call, got %d", len(calls.CreateCalls))
				}
			},
		},
		{
			name: "sync with positional directory argument",
			files: map[string]string{
				"special/flagged.yaml": `name: flagged
resources:
  allowed: ["flag"]
  denied: []`,
			},
			mockAPIRoles: []models.Role{},
			args:         []string{"special"},
			expectError: false,
			expectOutput: []string{
				"Synchronizing roles from directory: special",
				"create 1 role(s)",
				"flagged",
			},
			validateCalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 1 {
					t.Errorf("Expected 1 create call, got %d", len(calls.CreateCalls))
				}
			},
		},
		{
			name:         "sync empty directory - no changes",
			files:        map[string]string{},
			mockAPIRoles: []models.Role{},
			args:         []string{},
			flags:        map[string]string{},
			expectError:  false,
			expectOutput: []string{
				"No changes needed",
			},
			validateCalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 0 || len(calls.UpdateCalls) != 0 || len(calls.DeleteCalls) != 0 {
					t.Errorf("Expected no API calls for empty directory")
				}
			},
		},
		{
			name: "sync with API error",
			files: map[string]string{
				"failing.yaml": `name: failing
resources:
  allowed: ["*"]
  denied: []`,
			},
			mockAPIRoles: []models.Role{},
			args:         []string{},
			flags:        map[string]string{},
			expectError:  true,
			expectOutput: []string{
				"Sync failed:",
				"failed to create role",
			},
			validateCalls: func(t *testing.T, calls *MockAPICalls) {
				if len(calls.CreateCalls) != 1 {
					t.Errorf("Expected 1 create call attempt, got %d", len(calls.CreateCalls))
				}
			},
		},
		{
			name:         "sync nonexistent directory",
			files:        map[string]string{},
			mockAPIRoles: []models.Role{},
			args:         []string{"nonexistent"},
			flags:        map[string]string{},
			expectError:  true,
			expectOutput: []string{
				"directory does not exist",
			},
			validateCalls: func(t *testing.T, calls *MockAPICalls) {
				// Should not make any API calls
				if len(calls.CreateCalls) != 0 || len(calls.UpdateCalls) != 0 || len(calls.DeleteCalls) != 0 {
					t.Errorf("Expected no API calls for nonexistent directory")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "replbac-sync-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create test files
			for fileName, content := range tt.files {
				filePath := filepath.Join(tempDir, fileName)
				fileDir := filepath.Dir(filePath)
				err := os.MkdirAll(fileDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create file dir: %v", err)
				}
				err = os.WriteFile(filePath, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			// Setup mock API
			mockCalls := &MockAPICalls{}
			mockClient := NewMockClient(mockCalls, tt.mockAPIRoles)

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
			cmd := NewSyncCommand(mockClient)
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

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

			// Check output (check both stdout and stderr)
			stdoutStr := stdout.String()
			stderrStr := stderr.String()
			combinedOutput := stdoutStr + stderrStr
			for _, expected := range tt.expectOutput {
				if !strings.Contains(combinedOutput, expected) {
					t.Errorf("Expected output to contain '%s', got:\nSTDOUT:\n%s\nSTDERR:\n%s", expected, stdoutStr, stderrStr)
				}
			}
			for _, notExpected := range tt.expectNoOutput {
				if strings.Contains(combinedOutput, notExpected) {
					t.Errorf("Expected output to NOT contain '%s', got:\nSTDOUT:\n%s\nSTDERR:\n%s", notExpected, stdoutStr, stderrStr)
				}
			}

			// Validate API calls
			if tt.validateCalls != nil {
				tt.validateCalls(t, mockCalls)
			}
		})
	}
}

// TestSyncCommandConfiguration tests configuration handling
func TestSyncCommandConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]string
		envVars     map[string]string
		flags       map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: map[string]string{
				"api_endpoint": "https://api.replicated.com",
				"api_token":   "test-token",
			},
			expectError: false,
		},
		{
			name:        "missing API token",
			config:      map[string]string{},
			expectError: true,
			errorMsg:    "API token is required",
		},
		{
			name: "environment variable override",
			config: map[string]string{
				"api_token": "config-token",
			},
			envVars: map[string]string{
				"REPLICATED_API_TOKEN": "env-token",
			},
			expectError: false,
		},
		{
			name: "command line flag override",
			config: map[string]string{
				"api_token": "config-token",
			},
			flags: map[string]string{
				"api-token": "flag-token",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "replbac-config-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create config file if needed
			if len(tt.config) > 0 {
				configPath := filepath.Join(tempDir, "config.yaml")
				var configContent strings.Builder
				for key, value := range tt.config {
					configContent.WriteString(fmt.Sprintf("%s: %s\n", key, value))
				}
				err = os.WriteFile(configPath, []byte(configContent.String()), 0644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
			}

			// Set environment variables
			for key, value := range tt.envVars {
				oldValue := os.Getenv(key)
				os.Setenv(key, value)
				defer os.Setenv(key, oldValue)
			}

			// Setup command based on test type
			var cmd *cobra.Command
			if tt.name == "missing API token" {
				// For API token validation, use real command (no mock) to test validation
				cmd = &cobra.Command{
					Use:   "sync [directory]",
					Short: "Synchronize local role files to Replicated API",
					Args:  cobra.MaximumNArgs(1),
					RunE: func(cmd *cobra.Command, args []string) error {
						config := models.Config{
							APIEndpoint: "https://api.replicated.com",
							APIToken:    "", // Empty token for this test
						}
						return RunSyncCommand(cmd, args, config, false, false)
					},
				}
			} else {
				// Other tests use mock client to avoid real API calls
				mockCalls := &MockAPICalls{}
				mockClient := NewMockClient(mockCalls, []models.Role{})
				cmd = NewSyncCommand(mockClient)
			}
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Set config file flag if config exists
			if len(tt.config) > 0 {
				configPath := filepath.Join(tempDir, "config.yaml")
				cmd.Flags().Set("config", configPath)
			}

			// Set other flags
			for flag, value := range tt.flags {
				cmd.Flags().Set(flag, value)
			}

			// Execute command
			err = cmd.Execute()

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// MockAPICalls tracks API calls for testing
type MockAPICalls struct {
	CreateCalls []models.Role
	UpdateCalls []models.Role
	DeleteCalls []string
	GetCalls    int
}

// MockClient implements ClientInterface for testing
type MockClient struct {
	calls       *MockAPICalls
	roles       []models.Role
	shouldError bool
}

// NewMockClient creates a new mock client
func NewMockClient(calls *MockAPICalls, roles []models.Role) *MockClient {
	return &MockClient{
		calls: calls,
		roles: roles,
	}
}

// GetRoles returns the configured roles
func (m *MockClient) GetRoles() ([]models.Role, error) {
	m.calls.GetCalls++
	if m.shouldError {
		return nil, fmt.Errorf("API connection failed")
	}
	return m.roles, nil
}

// GetRole returns a specific role by name
func (m *MockClient) GetRole(roleName string) (models.Role, error) {
	for _, role := range m.roles {
		if role.Name == roleName {
			return role, nil
		}
	}
	return models.Role{}, fmt.Errorf("role not found: %s", roleName)
}

// CreateRole tracks create calls and adds to mock state
func (m *MockClient) CreateRole(role models.Role) error {
	m.calls.CreateCalls = append(m.calls.CreateCalls, role)
	if role.Name == "failing" {
		return fmt.Errorf("failed to create role 'failing': API error")
	}
	if role.Name == "problematic-role" {
		return fmt.Errorf("failed to create role 'problematic-role': API error")
	}
	// Add role to mock state (simulate API assigning ID if missing)
	if role.ID == "" {
		role.ID = "mock-generated-id-" + role.Name
	}
	m.roles = append(m.roles, role)
	return nil
}

// UpdateRole tracks update calls and updates the mock state
func (m *MockClient) UpdateRole(role models.Role) error {
	m.calls.UpdateCalls = append(m.calls.UpdateCalls, role)
	// Update the role in mock state
	for i, existingRole := range m.roles {
		if existingRole.ID == role.ID || (existingRole.ID == "" && existingRole.Name == role.Name) {
			m.roles[i] = role
			return nil
		}
	}
	// If role not found, this is an error
	return fmt.Errorf("role not found for update: %s", role.Name)
}

// DeleteRole tracks delete calls and removes from mock state
func (m *MockClient) DeleteRole(roleName string) error {
	m.calls.DeleteCalls = append(m.calls.DeleteCalls, roleName)
	// Remove from mock state
	for i, role := range m.roles {
		if role.Name == roleName {
			m.roles = append(m.roles[:i], m.roles[i+1:]...)
			return nil
		}
	}
	// If role not found, this might be expected in some tests
	return nil
}

// Context-aware methods (delegate to non-context versions for mock simplicity)

// GetRolesWithContext returns the configured roles with context support
func (m *MockClient) GetRolesWithContext(ctx context.Context) ([]models.Role, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return m.GetRoles()
	}
}

// GetRoleWithContext returns a specific role by name with context support
func (m *MockClient) GetRoleWithContext(ctx context.Context, roleName string) (models.Role, error) {
	select {
	case <-ctx.Done():
		return models.Role{}, ctx.Err()
	default:
		return m.GetRole(roleName)
	}
}

// CreateRoleWithContext tracks create calls with context support
func (m *MockClient) CreateRoleWithContext(ctx context.Context, role models.Role) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return m.CreateRole(role)
	}
}

// UpdateRoleWithContext tracks update calls with context support
func (m *MockClient) UpdateRoleWithContext(ctx context.Context, role models.Role) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return m.UpdateRole(role)
	}
}

// DeleteRoleWithContext tracks delete calls with context support
func (m *MockClient) DeleteRoleWithContext(ctx context.Context, roleName string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return m.DeleteRole(roleName)
	}
}

// Helper functions for testing
func NewSyncCommand(mockClient *MockClient) *cobra.Command {
	// Create a test version of the sync command
	cmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Synchronize local role files to Replicated API",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flag values
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			
			return RunSyncCommandWithClient(cmd, args, mockClient, dryRun)
		},
	}
	
	// Add flags
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().String("roles-dir", "", "directory containing role YAML files")
	
	return cmd
}

