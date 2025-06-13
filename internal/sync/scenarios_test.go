package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"replbac/internal/models"
	"replbac/internal/roles"
)

// TestSyncScenarios tests various end-to-end sync scenarios
func TestSyncScenarios(t *testing.T) {
	tests := []struct {
		name           string
		localRoles     []models.Role
		remoteRoles    []models.Role
		expectedPlan   SyncPlan
		expectedResult ExecutionResult
		shouldError    bool
	}{
		{
			name:        "empty local and remote - no sync needed",
			localRoles:  []models.Role{},
			remoteRoles: []models.Role{},
			expectedPlan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			expectedResult: ExecutionResult{
				Created: 0,
				Updated: 0,
				Deleted: 0,
				Error:   nil,
				DryRun:  false,
			},
			shouldError: false,
		},
		{
			name: "new local role - should create",
			localRoles: []models.Role{
				{
					Name: "new-admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
			},
			remoteRoles: []models.Role{},
			expectedPlan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "new-admin",
						Resources: models.Resources{
							Allowed: []string{"*"},
							Denied:  []string{},
						},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			expectedResult: ExecutionResult{
				Created: 1,
				Updated: 0,
				Deleted: 0,
				Error:   nil,
				DryRun:  false,
			},
			shouldError: false,
		},
		{
			name:       "remote role not in local - should delete",
			localRoles: []models.Role{},
			remoteRoles: []models.Role{
				{
					Name: "obsolete-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			expectedPlan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{"obsolete-role"},
			},
			expectedResult: ExecutionResult{
				Created: 0,
				Updated: 0,
				Deleted: 1,
				Error:   nil,
				DryRun:  false,
			},
			shouldError: false,
		},
		{
			name: "role differs between local and remote - should update",
			localRoles: []models.Role{
				{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read", "write", "create"},
						Denied:  []string{"delete"},
					},
				},
			},
			remoteRoles: []models.Role{
				{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{},
					},
				},
			},
			expectedPlan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{
					{
						Name: "editor",
						Local: models.Role{
							Name: "editor",
							Resources: models.Resources{
								Allowed: []string{"read", "write", "create"},
								Denied:  []string{"delete"},
							},
						},
						Remote: models.Role{
							Name: "editor",
							Resources: models.Resources{
								Allowed: []string{"read", "write"},
								Denied:  []string{},
							},
						},
					},
				},
				Deletes: []string{},
			},
			expectedResult: ExecutionResult{
				Created: 0,
				Updated: 1,
				Deleted: 0,
				Error:   nil,
				DryRun:  false,
			},
			shouldError: false,
		},
		{
			name: "complex scenario - create, update, delete",
			localRoles: []models.Role{
				{
					Name: "new-role",
					Resources: models.Resources{
						Allowed: []string{"create"},
						Denied:  []string{},
					},
				},
				{
					Name: "existing-role",
					Resources: models.Resources{
						Allowed: []string{"read", "write", "update"},
						Denied:  []string{},
					},
				},
				{
					Name: "unchanged-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			remoteRoles: []models.Role{
				{
					Name: "existing-role",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{},
					},
				},
				{
					Name: "unchanged-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
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
			expectedPlan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "new-role",
						Resources: models.Resources{
							Allowed: []string{"create"},
							Denied:  []string{},
						},
					},
				},
				Updates: []RoleUpdate{
					{
						Name: "existing-role",
						Local: models.Role{
							Name: "existing-role",
							Resources: models.Resources{
								Allowed: []string{"read", "write", "update"},
								Denied:  []string{},
							},
						},
						Remote: models.Role{
							Name: "existing-role",
							Resources: models.Resources{
								Allowed: []string{"read", "write"},
								Denied:  []string{},
							},
						},
					},
				},
				Deletes: []string{"old-role"},
			},
			expectedResult: ExecutionResult{
				Created: 1,
				Updated: 1,
				Deleted: 1,
				Error:   nil,
				DryRun:  false,
			},
			shouldError: false,
		},
		{
			name: "identical roles - no changes needed",
			localRoles: []models.Role{
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
			remoteRoles: []models.Role{
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
			expectedPlan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			expectedResult: ExecutionResult{
				Created: 0,
				Updated: 0,
				Deleted: 0,
				Error:   nil,
				DryRun:  false,
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test sync plan generation
			plan, err := CompareRoles(tt.localRoles, tt.remoteRoles)
			if (err != nil) != tt.shouldError {
				t.Errorf("CompareRoles() error = %v, shouldError %v", err, tt.shouldError)
				return
			}

			if err == nil {
				// Verify plan matches expectations
				if !syncPlansEqual(plan, tt.expectedPlan) {
					t.Errorf("CompareRoles() plan = %+v, want %+v", plan, tt.expectedPlan)
				}

				// Test execution
				mockClient := &MockAPIClient{}
				executor := NewExecutor(mockClient, createTestLogger())
				result := executor.ExecutePlan(plan)

				// Verify execution result
				if result.Created != tt.expectedResult.Created {
					t.Errorf("ExecutePlan() created = %d, want %d", result.Created, tt.expectedResult.Created)
				}
				if result.Updated != tt.expectedResult.Updated {
					t.Errorf("ExecutePlan() updated = %d, want %d", result.Updated, tt.expectedResult.Updated)
				}
				if result.Deleted != tt.expectedResult.Deleted {
					t.Errorf("ExecutePlan() deleted = %d, want %d", result.Deleted, tt.expectedResult.Deleted)
				}
				if result.Error != nil && tt.expectedResult.Error == nil {
					t.Errorf("ExecutePlan() error = %v, want nil", result.Error)
				}
				if result.DryRun != tt.expectedResult.DryRun {
					t.Errorf("ExecutePlan() dryRun = %v, want %v", result.DryRun, tt.expectedResult.DryRun)
				}

				// Test dry run
				dryResult := executor.ExecutePlanDryRun(plan)
				if dryResult.Created != tt.expectedResult.Created {
					t.Errorf("ExecutePlanDryRun() created = %d, want %d", dryResult.Created, tt.expectedResult.Created)
				}
				if dryResult.Updated != tt.expectedResult.Updated {
					t.Errorf("ExecutePlanDryRun() updated = %d, want %d", dryResult.Updated, tt.expectedResult.Updated)
				}
				if dryResult.Deleted != tt.expectedResult.Deleted {
					t.Errorf("ExecutePlanDryRun() deleted = %d, want %d", dryResult.Deleted, tt.expectedResult.Deleted)
				}
				if !dryResult.DryRun {
					t.Errorf("ExecutePlanDryRun() dryRun = %v, want true", dryResult.DryRun)
				}
				if dryResult.Error != nil {
					t.Errorf("ExecutePlanDryRun() error = %v, want nil", dryResult.Error)
				}

				// Verify no actual API calls were made in dry run
				if len(mockClient.CreatedRoles) != len(tt.expectedPlan.Creates) {
					t.Errorf("ExecutePlan() actual API calls = %d, expected = %d", len(mockClient.CreatedRoles), len(tt.expectedPlan.Creates))
				}
			}
		})
	}
}

// TestSyncWithFileOperations tests sync scenarios involving actual file operations
func TestSyncWithFileOperations(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "replbac-sync-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name        string
		files       map[string]string
		remoteRoles []models.Role
		expectError bool
		expectPlan  SyncPlan
	}{
		{
			name: "single role file - should create",
			files: map[string]string{
				"admin.yaml": `name: admin
resources:
  allowed: ["*"]
  denied: []`,
			},
			remoteRoles: []models.Role{},
			expectError: false,
			expectPlan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "admin",
						Resources: models.Resources{
							Allowed: []string{"*"},
							Denied:  []string{},
						},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
		},
		{
			name: "multiple role files in subdirectories",
			files: map[string]string{
				"roles/admin.yaml": `name: admin
resources:
  allowed: ["*"]
  denied: []`,
				"roles/users/viewer.yaml": `name: viewer
resources:
  allowed: ["read"]
  denied: ["write", "delete"]`,
				"roles/users/editor.yml": `name: editor
resources:
  allowed: ["read", "write"]
  denied: ["delete"]`,
			},
			remoteRoles: []models.Role{
				{
					Name: "old-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			expectError: false,
			expectPlan: SyncPlan{
				Creates: []models.Role{
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
							Denied:  []string{"delete"},
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
				Updates: []RoleUpdate{},
				Deletes: []string{"old-role"},
			},
		},
		{
			name: "mixed valid and invalid files - should load valid ones",
			files: map[string]string{
				"valid.yaml": `name: valid-role
resources:
  allowed: ["read"]
  denied: []`,
				"invalid.yaml": `invalid yaml content: [[[`,
				"readme.txt":   "This is not a yaml file",
				"empty.yaml":   "",
			},
			remoteRoles: []models.Role{},
			expectError: false,
			expectPlan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "valid-role",
						Resources: models.Resources{
							Allowed: []string{"read"},
							Denied:  []string{},
						},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test files
			testDir := filepath.Join(tempDir, tt.name)
			// #nosec G301 -- Test directories need readable permissions
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			for fileName, content := range tt.files {
				filePath := filepath.Join(testDir, fileName)
				fileDir := filepath.Dir(filePath)
				// #nosec G301 -- Test directories need readable permissions
				err := os.MkdirAll(fileDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create file dir: %v", err)
				}
				// #nosec G306 -- Test files need readable permissions
				err = os.WriteFile(filePath, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			// Load roles from directory
			localRoles, err := roles.LoadRolesFromDirectory(testDir)
			if (err != nil) != tt.expectError {
				t.Errorf("LoadRolesFromDirectory() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if err == nil {
				// Generate sync plan
				plan, err := CompareRoles(localRoles, tt.remoteRoles)
				if err != nil {
					t.Errorf("CompareRoles() error = %v", err)
					return
				}

				// Verify plan matches expectations
				if !syncPlansEqual(plan, tt.expectPlan) {
					t.Errorf("Sync plan mismatch.\nGot: %+v\nWant: %+v", plan, tt.expectPlan)
				}
			}
		})
	}
}

// TestSyncErrorScenarios tests various error conditions in sync scenarios
func TestSyncErrorScenarios(t *testing.T) {
	tests := []struct {
		name           string
		localRoles     []models.Role
		remoteRoles    []models.Role
		mockSetup      func(*MockAPIClient)
		expectError    bool
		errorContains  string
		partialSuccess bool
	}{
		{
			name: "API error during create",
			localRoles: []models.Role{
				{
					Name: "failing-role",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
			},
			remoteRoles: []models.Role{},
			mockSetup: func(m *MockAPIClient) {
				m.CreateRoleFunc = func(role models.Role) error {
					return fmt.Errorf("API connection failed")
				}
			},
			expectError:   true,
			errorContains: "failed to create role",
		},
		{
			name: "API error during update",
			localRoles: []models.Role{
				{
					Name: "update-role",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{},
					},
				},
			},
			remoteRoles: []models.Role{
				{
					Name: "update-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			mockSetup: func(m *MockAPIClient) {
				m.UpdateRoleFunc = func(role models.Role) error {
					return fmt.Errorf("API permission denied")
				}
			},
			expectError:   true,
			errorContains: "failed to update role",
		},
		{
			name:       "API error during delete",
			localRoles: []models.Role{},
			remoteRoles: []models.Role{
				{
					Name: "delete-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			mockSetup: func(m *MockAPIClient) {
				m.DeleteRoleFunc = func(roleName string) error {
					return fmt.Errorf("API role not found")
				}
			},
			expectError:   true,
			errorContains: "failed to delete role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate sync plan
			plan, err := CompareRoles(tt.localRoles, tt.remoteRoles)
			if err != nil {
				t.Errorf("CompareRoles() error = %v", err)
				return
			}

			// Setup mock client
			mockClient := &MockAPIClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			// Execute sync
			executor := NewExecutor(mockClient, createTestLogger())
			result := executor.ExecutePlan(plan)

			// Verify error expectations
			if tt.expectError {
				if result.Error == nil {
					t.Errorf("ExecutePlan() expected error but got none")
				} else if tt.errorContains != "" && !containsString(result.Error.Error(), tt.errorContains) {
					t.Errorf("ExecutePlan() error = %v, want error containing %v", result.Error, tt.errorContains)
				}
			} else if result.Error != nil {
				t.Errorf("ExecutePlan() unexpected error = %v", result.Error)
			}
		})
	}
}

// Helper function to compare sync plans
func syncPlansEqual(a, b SyncPlan) bool {
	// Compare creates
	if len(a.Creates) != len(b.Creates) {
		return false
	}
	for _, roleA := range a.Creates {
		found := false
		for _, roleB := range b.Creates {
			if RolesEqual(roleA, roleB) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Compare updates
	if len(a.Updates) != len(b.Updates) {
		return false
	}
	for _, updateA := range a.Updates {
		found := false
		for _, updateB := range b.Updates {
			if updateA.Name == updateB.Name &&
				RolesEqual(updateA.Local, updateB.Local) &&
				RolesEqual(updateA.Remote, updateB.Remote) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Compare deletes
	if len(a.Deletes) != len(b.Deletes) {
		return false
	}
	for _, deleteA := range a.Deletes {
		found := false
		for _, deleteB := range b.Deletes {
			if deleteA == deleteB {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
