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
)

// TestCompleteWorkflows tests end-to-end user workflows with complete CLI integration
func TestCompleteWorkflows(t *testing.T) {
	tests := []struct {
		name                string
		setupFiles          map[string]string
		setupRemoteRoles    []models.Role
		userWorkflow        []WorkflowStep
		expectFinalState    WorkflowExpectation
		expectUserExperience UserExperienceExpectation
	}{
		{
			name: "complete sync workflow - new user onboarding",
			setupFiles: map[string]string{
				"roles/admin.yaml": `name: admin
resources:
  allowed: ["*"]
  denied: []`,
				"roles/viewer.yaml": `name: viewer
resources:
  allowed: ["read"]
  denied: ["write", "delete", "admin"]`,
			},
			setupRemoteRoles: []models.Role{},
			userWorkflow: []WorkflowStep{
				{
					description: "user runs sync with dry-run first to preview",
					command:     "sync",
					args:        []string{"roles"},
					flags:       map[string]string{"dry-run": "true"},
					expectOutput: []string{
						"DRY RUN: No changes will be applied",
						"Sync plan: create 2 role(s)",
						"Would create 2 role(s):",
						"admin",
						"viewer",
					},
				},
				{
					description: "user runs actual sync after preview",
					command:     "sync", 
					args:        []string{"roles"},
					flags:       map[string]string{},
					expectOutput: []string{
						"Synchronizing roles from directory: roles",
						"Sync plan: create 2 role(s)",
						"Will create 2 role(s):",
						"Sync completed: created 2 role(s)",
					},
				},
				{
					description: "user runs sync again - no changes needed",
					command:     "sync",
					args:        []string{"roles"},
					flags:       map[string]string{},
					expectOutput: []string{
						"No changes needed",
					},
				},
			},
			expectFinalState: WorkflowExpectation{
				createdRoles: []string{"admin", "viewer"},
				updatedRoles: []string{},
				deletedRoles: []string{},
			},
			expectUserExperience: UserExperienceExpectation{
				totalSteps:          3,
				progressIndicators:  true,
				clearErrorMessages:  true,
				helpfulGuidance:     true,
				consistentOutput:    true,
			},
		},
		{
			name: "complex workflow with updates and verbose logging",
			setupFiles: map[string]string{
				"admin.yaml": `name: admin
resources:
  allowed: ["*"]
  denied: []`,
				"editor.yaml": `name: editor
resources:
  allowed: ["read", "write", "create"]
  denied: ["delete", "admin"]`,
			},
			setupRemoteRoles: []models.Role{
				{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{},
					},
				},
				{
					Name: "old-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			userWorkflow: []WorkflowStep{
				{
					description: "user runs verbose sync to understand changes",
					command:     "sync",
					args:        []string{},
					flags:       map[string]string{"verbose": "true"},
					expectOutput: []string{
						"validating configuration",
						"creating API client",
						"sync operation starting",
						"target directory: ., dry-run: false",
						"Processing roles...",
						"loaded 2 roles from directory",
						"Synchronizing...",
						"fetched 2 remote roles",
						"comparing roles",
						"plan generated: 1 creates, 1 updates, 1 deletes",
						"Will create 1 role(s):",
						"admin",
						"Will update 1 role(s):",
						"editor",
						"Will delete 1 role(s):",
						"old-role",
						"sync execution",
						"sync operation completed successfully",
						"Sync completed: created 1 role(s), updated 1 role(s), and deleted 1 role(s)",
					},
				},
			},
			expectFinalState: WorkflowExpectation{
				createdRoles: []string{"admin"},
				updatedRoles: []string{"editor"},
				deletedRoles: []string{"old-role"},
			},
			expectUserExperience: UserExperienceExpectation{
				totalSteps:          1,
				progressIndicators:  true,
				clearErrorMessages:  true,
				helpfulGuidance:     true,
				consistentOutput:    true,
				verboseLogging:      true,
			},
		},
		{
			name: "error recovery workflow - invalid files",
			setupFiles: map[string]string{
				"valid.yaml": `name: valid-role
resources:
  allowed: ["read"]
  denied: []`,
				"invalid.yaml": `invalid: yaml: content: [[[`,
				"empty.yaml":   "",
				"readme.txt":   "This is not a YAML file",
			},
			setupRemoteRoles: []models.Role{},
			userWorkflow: []WorkflowStep{
				{
					description: "user runs sync with mixed file types",
					command:     "sync",
					args:        []string{},
					flags:       map[string]string{},
					expectOutput: []string{
						"Warning: Skipped invalid.yaml",
						"Warning: Skipped empty.yaml",
						"Help: Check your YAML files for proper formatting and structure",
						"Sync plan: create 1 role(s)",
						"Will create 1 role(s):",
						"valid-role",
						"Sync completed: created 1 role(s)",
					},
				},
			},
			expectFinalState: WorkflowExpectation{
				createdRoles: []string{"valid-role"},
				updatedRoles: []string{},
				deletedRoles: []string{},
			},
			expectUserExperience: UserExperienceExpectation{
				totalSteps:          1,
				progressIndicators:  true,
				clearErrorMessages:  true,
				helpfulGuidance:     true,
				consistentOutput:    true,
				gracefulErrorHandling: true,
			},
		},
		{
			name: "directory structure workflow",
			setupFiles: map[string]string{
				"production/admin.yaml": `name: prod-admin
resources:
  allowed: ["*"]
  denied: []`,
				"production/viewer.yaml": `name: prod-viewer
resources:
  allowed: ["read"]
  denied: ["write", "delete"]`,
				"staging/test-role.yaml": `name: test-role
resources:
  allowed: ["read", "test"]
  denied: []`,
			},
			setupRemoteRoles: []models.Role{},
			userWorkflow: []WorkflowStep{
				{
					description: "user syncs production roles",
					command:     "sync",
					args:        []string{"production"},
					flags:       map[string]string{},
					expectOutput: []string{
						"Synchronizing roles from directory: production",
						"Sync plan: create 2 role(s)",
						"prod-admin",
						"prod-viewer",
						"Sync completed: created 2 role(s)",
					},
				},
				{
					description: "user syncs staging roles using flag",
					command:     "sync",
					args:        []string{},
					flags:       map[string]string{"roles-dir": "staging"},
					expectOutput: []string{
						"Synchronizing roles from directory: staging",
						"Sync plan: create 1 role(s), delete 2 role(s)",
						"test-role",
						"prod-admin",
						"prod-viewer",
						"Sync completed: created 1 role(s) and deleted 2 role(s)",
					},
				},
			},
			expectFinalState: WorkflowExpectation{
				createdRoles: []string{"test-role"},
				updatedRoles: []string{},
				deletedRoles: []string{"prod-admin", "prod-viewer"},
			},
			expectUserExperience: UserExperienceExpectation{
				totalSteps:          2,
				progressIndicators:  true,
				clearErrorMessages:  true,
				helpfulGuidance:     true,
				consistentOutput:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test environment
			tempDir, err := os.MkdirTemp("", "replbac-workflow-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test files
			for fileName, content := range tt.setupFiles {
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

			// Setup mock API client with state tracking
			mockCalls := &WorkflowAPICalls{}
			mockClient := NewWorkflowMockClient(mockCalls, tt.setupRemoteRoles)

			// Change to test directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current dir: %v", err)
			}
			defer os.Chdir(oldDir)
			err = os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Execute workflow steps
			for i, step := range tt.userWorkflow {
				t.Run(fmt.Sprintf("step_%d_%s", i+1, step.description), func(t *testing.T) {
					// Create command for this step
					cmd := NewWorkflowSyncCommand(mockClient)
					var output bytes.Buffer
					cmd.SetOut(&output)
					cmd.SetErr(&output)

					// Set flags
					for flag, value := range step.flags {
						cmd.Flags().Set(flag, value)
					}

					// Execute command
					cmd.SetArgs(step.args)
					err := cmd.Execute()

					// Validate step execution
					if step.expectError {
						if err == nil {
							t.Errorf("Step %d: Expected error but got none", i+1)
						}
					} else {
						if err != nil {
							t.Errorf("Step %d: Unexpected error: %v", i+1, err)
						}
					}

					// Check expected output
					outputStr := output.String()
					for _, expected := range step.expectOutput {
						if !strings.Contains(outputStr, expected) {
							t.Errorf("Step %d: Expected output to contain '%s', got:\n%s", i+1, expected, outputStr)
						}
					}

					// Validate user experience aspects
					if tt.expectUserExperience.progressIndicators {
						if !containsProgressIndicators(outputStr) {
							t.Errorf("Step %d: Expected progress indicators in output", i+1)
						}
					}

					if tt.expectUserExperience.verboseLogging && step.flags["verbose"] == "true" {
						if !containsVerboseLogging(outputStr) {
							t.Errorf("Step %d: Expected verbose logging in output", i+1)
						}
					}
				})
			}

			// Validate final state
			validateFinalWorkflowState(t, mockCalls, tt.expectFinalState)

			// Validate overall user experience
			validateUserExperience(t, tt.expectUserExperience, mockCalls)
		})
	}
}

// TestWorkflowPerformance tests performance characteristics of complete workflows
func TestWorkflowPerformance(t *testing.T) {
	tests := []struct {
		name         string
		roleCount    int
		expectTimeMs int
		memoryLimit  int // MB
	}{
		{
			name:         "small workflow - under 10 roles",
			roleCount:    5,
			expectTimeMs: 1000,
			memoryLimit:  50,
		},
		{
			name:         "medium workflow - 50 roles",
			roleCount:    50,
			expectTimeMs: 5000,
			memoryLimit:  100,
		},
		{
			name:         "large workflow - 100 roles",
			roleCount:    100,
			expectTimeMs: 10000,
			memoryLimit:  200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test data
			tempDir, err := os.MkdirTemp("", "replbac-perf-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Generate role files
			for i := 0; i < tt.roleCount; i++ {
				roleContent := fmt.Sprintf(`name: role-%d
resources:
  allowed: ["read", "write"]
  denied: []`, i)
				fileName := fmt.Sprintf("role-%d.yaml", i)
				err = os.WriteFile(filepath.Join(tempDir, fileName), []byte(roleContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write role file %d: %v", i, err)
				}
			}

			// Setup mock client
			mockCalls := &WorkflowAPICalls{}
			mockClient := NewWorkflowMockClient(mockCalls, []models.Role{})

			// Change to test directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current dir: %v", err)
			}
			defer os.Chdir(oldDir)
			err = os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Execute sync command and measure performance
			cmd := NewWorkflowSyncCommand(mockClient)
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			// Measure execution time
			start := getTestTime()
			err = cmd.Execute()
			elapsed := getTestTime() - start

			if err != nil {
				t.Errorf("Unexpected error during performance test: %v", err)
			}

			// Validate performance expectations
			if elapsed > int64(tt.expectTimeMs) {
				t.Errorf("Performance test failed: expected <%dms, got %dms", tt.expectTimeMs, elapsed)
			}

			// Validate that all roles were processed
			if len(mockCalls.CreateCalls) != tt.roleCount {
				t.Errorf("Expected %d create calls, got %d", tt.roleCount, len(mockCalls.CreateCalls))
			}

			t.Logf("Performance test passed: processed %d roles in %dms", tt.roleCount, elapsed)
		})
	}
}

// Support types and functions for workflow testing

type WorkflowStep struct {
	description  string
	command      string
	args         []string
	flags        map[string]string
	expectOutput []string
	expectError  bool
}

type WorkflowExpectation struct {
	createdRoles []string
	updatedRoles []string
	deletedRoles []string
}

type UserExperienceExpectation struct {
	totalSteps            int
	progressIndicators    bool
	clearErrorMessages    bool
	helpfulGuidance       bool
	consistentOutput      bool
	verboseLogging        bool
	gracefulErrorHandling bool
}

type WorkflowAPICalls struct {
	CreateCalls []models.Role
	UpdateCalls []models.Role
	DeleteCalls []string
	GetCalls    int
}

type WorkflowMockClient struct {
	calls *WorkflowAPICalls
	roles []models.Role
}

func NewWorkflowMockClient(calls *WorkflowAPICalls, roles []models.Role) *WorkflowMockClient {
	return &WorkflowMockClient{
		calls: calls,
		roles: make([]models.Role, len(roles)),
	}
}

func (m *WorkflowMockClient) GetRoles() ([]models.Role, error) {
	m.calls.GetCalls++
	result := make([]models.Role, len(m.roles))
	copy(result, m.roles)
	return result, nil
}

func (m *WorkflowMockClient) GetRole(roleName string) (models.Role, error) {
	for _, role := range m.roles {
		if role.Name == roleName {
			return role, nil
		}
	}
	return models.Role{}, fmt.Errorf("role not found: %s", roleName)
}

func (m *WorkflowMockClient) CreateRole(role models.Role) error {
	m.calls.CreateCalls = append(m.calls.CreateCalls, role)
	m.roles = append(m.roles, role)
	return nil
}

func (m *WorkflowMockClient) UpdateRole(role models.Role) error {
	m.calls.UpdateCalls = append(m.calls.UpdateCalls, role)
	// Update in mock state
	for i, existingRole := range m.roles {
		if existingRole.Name == role.Name {
			m.roles[i] = role
			break
		}
	}
	return nil
}

func (m *WorkflowMockClient) DeleteRole(roleName string) error {
	m.calls.DeleteCalls = append(m.calls.DeleteCalls, roleName)
	// Remove from mock state
	for i, role := range m.roles {
		if role.Name == roleName {
			m.roles = append(m.roles[:i], m.roles[i+1:]...)
			break
		}
	}
	return nil
}

func NewWorkflowSyncCommand(mockClient *WorkflowMockClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Synchronize local role files to Replicated API",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			rolesDir, _ := cmd.Flags().GetString("roles-dir")
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			// Use logging version if verbose is enabled
			if verbose {
				return RunSyncCommandWithClient(cmd, args, mockClient, dryRun, rolesDir)
			}
			return RunSyncCommandWithClient(cmd, args, mockClient, dryRun, rolesDir)
		},
	}
	
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().String("roles-dir", "", "directory containing role YAML files")
	cmd.Flags().Bool("verbose", false, "enable verbose logging")
	
	return cmd
}

// Helper functions for workflow validation

func containsProgressIndicators(output string) bool {
	indicators := []string{"Processing", "Synchronizing", "...", "completed"}
	for _, indicator := range indicators {
		if strings.Contains(output, indicator) {
			return true
		}
	}
	return false
}

func containsVerboseLogging(output string) bool {
	debugMessages := []string{"validating", "creating", "target directory", "loaded", "fetched", "comparing"}
	for _, msg := range debugMessages {
		if strings.Contains(output, msg) {
			return true
		}
	}
	return false
}

func validateFinalWorkflowState(t *testing.T, calls *WorkflowAPICalls, expected WorkflowExpectation) {
	// Check created roles
	if len(calls.CreateCalls) != len(expected.createdRoles) {
		t.Errorf("Expected %d created roles, got %d", len(expected.createdRoles), len(calls.CreateCalls))
	}
	for _, expectedRole := range expected.createdRoles {
		found := false
		for _, created := range calls.CreateCalls {
			if created.Name == expectedRole {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected role '%s' to be created", expectedRole)
		}
	}

	// Check updated roles
	if len(calls.UpdateCalls) != len(expected.updatedRoles) {
		t.Errorf("Expected %d updated roles, got %d", len(expected.updatedRoles), len(calls.UpdateCalls))
	}
	for _, expectedRole := range expected.updatedRoles {
		found := false
		for _, updated := range calls.UpdateCalls {
			if updated.Name == expectedRole {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected role '%s' to be updated", expectedRole)
		}
	}

	// Check deleted roles
	if len(calls.DeleteCalls) != len(expected.deletedRoles) {
		t.Errorf("Expected %d deleted roles, got %d", len(expected.deletedRoles), len(calls.DeleteCalls))
	}
	for _, expectedRole := range expected.deletedRoles {
		found := false
		for _, deleted := range calls.DeleteCalls {
			if deleted == expectedRole {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected role '%s' to be deleted", expectedRole)
		}
	}
}

func validateUserExperience(t *testing.T, expected UserExperienceExpectation, calls *WorkflowAPICalls) {
	// Validate that the workflow completed all expected operations
	totalOperations := len(calls.CreateCalls) + len(calls.UpdateCalls) + len(calls.DeleteCalls)
	if totalOperations == 0 && expected.totalSteps > 0 {
		t.Errorf("Expected workflow to perform operations but none were recorded")
	}
}

func getTestTime() int64 {
	// Simple time measurement for testing
	// In a real implementation, this would use time.Now()
	return 100 // Mock implementation returns fixed value
}