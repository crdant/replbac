package sync

import (
	"errors"
	"testing"

	"replbac/internal/models"
)

// MockAPIClient implements the APIClient interface for testing
type MockAPIClient struct {
	CreateRoleFunc func(role models.Role) error
	UpdateRoleFunc func(role models.Role) error
	DeleteRoleFunc func(roleName string) error
	
	// Track calls for verification
	CreatedRoles []models.Role
	UpdatedRoles []models.Role
	DeletedRoles []string
}

func (m *MockAPIClient) CreateRole(role models.Role) error {
	m.CreatedRoles = append(m.CreatedRoles, role)
	if m.CreateRoleFunc != nil {
		return m.CreateRoleFunc(role)
	}
	return nil
}

func (m *MockAPIClient) UpdateRole(role models.Role) error {
	m.UpdatedRoles = append(m.UpdatedRoles, role)
	if m.UpdateRoleFunc != nil {
		return m.UpdateRoleFunc(role)
	}
	return nil
}

func (m *MockAPIClient) DeleteRole(roleName string) error {
	m.DeletedRoles = append(m.DeletedRoles, roleName)
	if m.DeleteRoleFunc != nil {
		return m.DeleteRoleFunc(roleName)
	}
	return nil
}

func TestExecutor_ExecutePlan(t *testing.T) {
	tests := []struct {
		name           string
		plan           SyncPlan
		mockSetup      func(*MockAPIClient)
		wantCreated    []models.Role
		wantUpdated    []models.Role
		wantDeleted    []string
		wantError      bool
		errorContains  string
	}{
		{
			name: "empty plan executes successfully",
			plan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			wantCreated: []models.Role{},
			wantUpdated: []models.Role{},
			wantDeleted: []string{},
			wantError:   false,
		},
		{
			name: "creates roles successfully",
			plan: SyncPlan{
				Creates: []models.Role{
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
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			wantCreated: []models.Role{
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
			wantUpdated: []models.Role{},
			wantDeleted: []string{},
			wantError:   false,
		},
		{
			name: "updates roles successfully",
			plan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{
					{
						Name: "editor",
						Local: models.Role{
							Name: "editor",
							Resources: models.Resources{
								Allowed: []string{"read", "write"},
								Denied:  []string{"delete"},
							},
						},
						Remote: models.Role{
							Name: "editor",
							Resources: models.Resources{
								Allowed: []string{"read"},
								Denied:  []string{},
							},
						},
					},
				},
				Deletes: []string{},
			},
			wantCreated: []models.Role{},
			wantUpdated: []models.Role{
				{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{"delete"},
					},
				},
			},
			wantDeleted: []string{},
			wantError:   false,
		},
		{
			name: "deletes roles successfully",
			plan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{"obsolete", "deprecated"},
			},
			wantCreated: []models.Role{},
			wantUpdated: []models.Role{},
			wantDeleted: []string{"obsolete", "deprecated"},
			wantError:   false,
		},
		{
			name: "complex plan with creates, updates, and deletes",
			plan: SyncPlan{
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
								Allowed: []string{"read", "write"},
								Denied:  []string{},
							},
						},
						Remote: models.Role{
							Name: "existing-role",
							Resources: models.Resources{
								Allowed: []string{"read"},
								Denied:  []string{},
							},
						},
					},
				},
				Deletes: []string{"old-role"},
			},
			wantCreated: []models.Role{
				{
					Name: "new-role",
					Resources: models.Resources{
						Allowed: []string{"create"},
						Denied:  []string{},
					},
				},
			},
			wantUpdated: []models.Role{
				{
					Name: "existing-role",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{},
					},
				},
			},
			wantDeleted: []string{"old-role"},
			wantError:   false,
		},
		{
			name: "create role fails",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "failing-role",
						Resources: models.Resources{
							Allowed: []string{"*"},
							Denied:  []string{},
						},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			mockSetup: func(m *MockAPIClient) {
				m.CreateRoleFunc = func(role models.Role) error {
					if role.Name == "failing-role" {
						return errors.New("API error: role creation failed")
					}
					return nil
				}
			},
			wantError:     true,
			errorContains: "failed to create role 'failing-role'",
		},
		{
			name: "update role fails",
			plan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{
					{
						Name: "failing-update",
						Local: models.Role{
							Name: "failing-update",
							Resources: models.Resources{
								Allowed: []string{"read", "write"},
								Denied:  []string{},
							},
						},
						Remote: models.Role{
							Name: "failing-update",
							Resources: models.Resources{
								Allowed: []string{"read"},
								Denied:  []string{},
							},
						},
					},
				},
				Deletes: []string{},
			},
			mockSetup: func(m *MockAPIClient) {
				m.UpdateRoleFunc = func(role models.Role) error {
					if role.Name == "failing-update" {
						return errors.New("API error: role update failed")
					}
					return nil
				}
			},
			wantError:     true,
			errorContains: "failed to update role 'failing-update'",
		},
		{
			name: "delete role fails",
			plan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{"failing-delete"},
			},
			mockSetup: func(m *MockAPIClient) {
				m.DeleteRoleFunc = func(roleName string) error {
					if roleName == "failing-delete" {
						return errors.New("API error: role deletion failed")
					}
					return nil
				}
			},
			wantError:     true,
			errorContains: "failed to delete role 'failing-delete'",
		},
		{
			name: "partial failure stops execution",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "success-role",
						Resources: models.Resources{
							Allowed: []string{"read"},
							Denied:  []string{},
						},
					},
					{
						Name: "fail-role",
						Resources: models.Resources{
							Allowed: []string{"write"},
							Denied:  []string{},
						},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			mockSetup: func(m *MockAPIClient) {
				m.CreateRoleFunc = func(role models.Role) error {
					if role.Name == "fail-role" {
						return errors.New("API error: creation failed")
					}
					return nil
				}
			},
			wantCreated: []models.Role{
				{
					Name: "success-role",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			wantError:     true,
			errorContains: "failed to create role 'fail-role'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAPIClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			executor := NewExecutor(mockClient)
			result := executor.ExecutePlan(tt.plan)

			// Check error expectations
			if tt.wantError {
				if result.Error == nil {
					t.Errorf("ExecutePlan() error = nil, wantError %v", tt.wantError)
					return
				}
				if tt.errorContains != "" && !containsString(result.Error.Error(), tt.errorContains) {
					t.Errorf("ExecutePlan() error = %v, want error containing %v", result.Error, tt.errorContains)
				}
			} else if result.Error != nil {
				t.Errorf("ExecutePlan() error = %v, wantError %v", result.Error, tt.wantError)
				return
			}

			// Check created roles
			if len(mockClient.CreatedRoles) != len(tt.wantCreated) {
				t.Errorf("ExecutePlan() created %d roles, want %d", len(mockClient.CreatedRoles), len(tt.wantCreated))
			}
			for i, created := range mockClient.CreatedRoles {
				if i < len(tt.wantCreated) {
					if !RolesEqual(created, tt.wantCreated[i]) {
						t.Errorf("ExecutePlan() created role %d = %+v, want %+v", i, created, tt.wantCreated[i])
					}
				}
			}

			// Check updated roles
			if len(mockClient.UpdatedRoles) != len(tt.wantUpdated) {
				t.Errorf("ExecutePlan() updated %d roles, want %d", len(mockClient.UpdatedRoles), len(tt.wantUpdated))
			}
			for i, updated := range mockClient.UpdatedRoles {
				if i < len(tt.wantUpdated) {
					if !RolesEqual(updated, tt.wantUpdated[i]) {
						t.Errorf("ExecutePlan() updated role %d = %+v, want %+v", i, updated, tt.wantUpdated[i])
					}
				}
			}

			// Check deleted roles
			if len(mockClient.DeletedRoles) != len(tt.wantDeleted) {
				t.Errorf("ExecutePlan() deleted %d roles, want %d", len(mockClient.DeletedRoles), len(tt.wantDeleted))
			}
			for i, deleted := range mockClient.DeletedRoles {
				if i < len(tt.wantDeleted) {
					if deleted != tt.wantDeleted[i] {
						t.Errorf("ExecutePlan() deleted role %d = %v, want %v", i, deleted, tt.wantDeleted[i])
					}
				}
			}

			// Verify result counts
			if !tt.wantError {
				expectedCreated := len(tt.wantCreated)
				expectedUpdated := len(tt.wantUpdated)
				expectedDeleted := len(tt.wantDeleted)

				if result.Created != expectedCreated {
					t.Errorf("ExecutePlan() result.Created = %d, want %d", result.Created, expectedCreated)
				}
				if result.Updated != expectedUpdated {
					t.Errorf("ExecutePlan() result.Updated = %d, want %d", result.Updated, expectedUpdated)
				}
				if result.Deleted != expectedDeleted {
					t.Errorf("ExecutePlan() result.Deleted = %d, want %d", result.Deleted, expectedDeleted)
				}
			}
		})
	}
}

func TestExecutor_ExecutePlanDryRun(t *testing.T) {
	plan := SyncPlan{
		Creates: []models.Role{
			{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"*"},
					Denied:  []string{},
				},
			},
		},
		Updates: []RoleUpdate{
			{
				Name: "editor",
				Local: models.Role{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{},
					},
				},
				Remote: models.Role{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
		},
		Deletes: []string{"obsolete"},
	}

	mockClient := &MockAPIClient{}
	executor := NewExecutor(mockClient)
	
	result := executor.ExecutePlanDryRun(plan)

	// In dry run mode, no actual API calls should be made
	if len(mockClient.CreatedRoles) != 0 {
		t.Errorf("ExecutePlanDryRun() made %d create calls, want 0", len(mockClient.CreatedRoles))
	}
	if len(mockClient.UpdatedRoles) != 0 {
		t.Errorf("ExecutePlanDryRun() made %d update calls, want 0", len(mockClient.UpdatedRoles))
	}
	if len(mockClient.DeletedRoles) != 0 {
		t.Errorf("ExecutePlanDryRun() made %d delete calls, want 0", len(mockClient.DeletedRoles))
	}

	// But the result should reflect what would be done
	if result.Created != 1 {
		t.Errorf("ExecutePlanDryRun() result.Created = %d, want 1", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("ExecutePlanDryRun() result.Updated = %d, want 1", result.Updated)
	}
	if result.Deleted != 1 {
		t.Errorf("ExecutePlanDryRun() result.Deleted = %d, want 1", result.Deleted)
	}
	if result.Error != nil {
		t.Errorf("ExecutePlanDryRun() error = %v, want nil", result.Error)
	}
	if !result.DryRun {
		t.Errorf("ExecutePlanDryRun() result.DryRun = %v, want true", result.DryRun)
	}
}

func TestNewExecutor(t *testing.T) {
	mockClient := &MockAPIClient{}
	executor := NewExecutor(mockClient)
	
	if executor == nil {
		t.Error("NewExecutor() returned nil")
	}
	
	// Test that executor can be used
	result := executor.ExecutePlanDryRun(SyncPlan{})
	if result.Error != nil {
		t.Errorf("NewExecutor() created executor that fails on empty plan: %v", result.Error)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    s[:len(substr)] == substr || 
		    s[len(s)-len(substr):] == substr ||
		    findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}