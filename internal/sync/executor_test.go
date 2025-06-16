package sync

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"replbac/internal/logging"
	"replbac/internal/models"
)

// createTestLogger creates a logger for testing
func createTestLogger() *logging.Logger {
	var buf bytes.Buffer
	return logging.NewLogger(&buf, true) // verbose for testing
}

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
		name          string
		plan          SyncPlan
		mockSetup     func(*MockAPIClient)
		wantCreated   []models.Role
		wantUpdated   []models.Role
		wantDeleted   []string
		wantError     bool
		errorContains string
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
			wantCreated: []models.Role{
				{
					Name: "failing-role",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
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
			wantUpdated: []models.Role{
				{
					Name: "failing-update",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{},
					},
				},
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
			wantDeleted:   []string{"failing-delete"},
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
				{
					Name: "fail-role",
					Resources: models.Resources{
						Allowed: []string{"write"},
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

			executor := NewExecutor(mockClient, createTestLogger())
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
	executor := NewExecutor(mockClient, createTestLogger())

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
	executor := NewExecutor(mockClient, createTestLogger())

	if executor == nil {
		t.Error("NewExecutor() returned nil")
	}

	// Test that executor can be used
	result := executor.ExecutePlanDryRun(SyncPlan{})
	if result.Error != nil {
		t.Errorf("NewExecutor() created executor that fails on empty plan: %v", result.Error)
	}
}

func TestExecutor_ExecutePlanDryRunWithDiffs(t *testing.T) {
	tests := []struct {
		name              string
		plan              SyncPlan
		wantDetailedInfo  bool
		wantCreateDetails []string
		wantUpdateDetails []string
		wantDeleteDetails []string
	}{
		{
			name: "dry run with detailed create information",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "new-admin",
						Resources: models.Resources{
							Allowed: []string{"*", "admin:*"},
							Denied:  []string{"sensitive:delete"},
						},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			wantDetailedInfo: true,
			wantCreateDetails: []string{
				"new-admin",
				"allowed: [* admin:*]",
				"denied: [sensitive:delete]",
			},
		},
		{
			name: "dry run with detailed update information",
			plan: SyncPlan{
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
			wantDetailedInfo: true,
			wantUpdateDetails: []string{
				"editor",
				"+ allowed: create",
				"+ denied: delete",
			},
		},
		{
			name: "dry run with detailed delete information",
			plan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{"deprecated-role", "old-service"},
			},
			wantDetailedInfo: true,
			wantDeleteDetails: []string{
				"deprecated-role",
				"old-service",
			},
		},
		{
			name: "complex dry run with all operation types",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "new-viewer",
						Resources: models.Resources{
							Allowed: []string{"read:*"},
							Denied:  []string{},
						},
					},
				},
				Updates: []RoleUpdate{
					{
						Name: "service-account",
						Local: models.Role{
							Name: "service-account",
							Resources: models.Resources{
								Allowed: []string{"api:read", "api:write"},
								Denied:  []string{"admin:*"},
							},
						},
						Remote: models.Role{
							Name: "service-account",
							Resources: models.Resources{
								Allowed: []string{"api:read"},
								Denied:  []string{},
							},
						},
					},
				},
				Deletes: []string{"unused-role"},
			},
			wantDetailedInfo:  true,
			wantCreateDetails: []string{"new-viewer", "allowed: [read:*]"},
			wantUpdateDetails: []string{"service-account", "+ allowed: api:write", "+ denied: admin:*"},
			wantDeleteDetails: []string{"unused-role"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAPIClient{}
			executor := NewExecutor(mockClient, createTestLogger())

			result := executor.ExecutePlanDryRunWithDiffs(tt.plan)

			// Verify it's a dry run
			if !result.DryRun {
				t.Errorf("ExecutePlanDryRunWithDiffs() result.DryRun = %v, want true", result.DryRun)
			}

			// Verify no actual API calls were made
			if len(mockClient.CreatedRoles) != 0 || len(mockClient.UpdatedRoles) != 0 || len(mockClient.DeletedRoles) != 0 {
				t.Errorf("ExecutePlanDryRunWithDiffs() made API calls in dry run mode")
			}

			// Verify result has detailed information
			if tt.wantDetailedInfo && result.DetailedInfo == "" {
				t.Errorf("ExecutePlanDryRunWithDiffs() result.DetailedInfo is empty, want detailed information")
			}

			// Check create details
			for _, detail := range tt.wantCreateDetails {
				if !containsString(result.DetailedInfo, detail) {
					t.Errorf("ExecutePlanDryRunWithDiffs() result.DetailedInfo missing create detail: %s", detail)
				}
			}

			// Check update details
			for _, detail := range tt.wantUpdateDetails {
				if !containsString(result.DetailedInfo, detail) {
					t.Errorf("ExecutePlanDryRunWithDiffs() result.DetailedInfo missing update detail: %s", detail)
				}
			}

			// Check delete details
			for _, detail := range tt.wantDeleteDetails {
				if !containsString(result.DetailedInfo, detail) {
					t.Errorf("ExecutePlanDryRunWithDiffs() result.DetailedInfo missing delete detail: %s", detail)
				}
			}
		})
	}
}

func TestExecutionResult_DetailedSummary(t *testing.T) {
	tests := []struct {
		name            string
		result          ExecutionResult
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "dry run with detailed info shows enhanced summary",
			result: ExecutionResult{
				Created:      1,
				Updated:      1,
				Deleted:      1,
				DryRun:       true,
				DetailedInfo: "CREATE: admin (allowed: [*], denied: [])\nUPDATE: editor\n+ allowed: write\nDELETE: obsolete",
			},
			wantContains: []string{
				"Dry run: Would create 1 role(s), update 1 role(s), and delete 1 role(s)",
				"CREATE: admin",
				"UPDATE: editor",
				"DELETE: obsolete",
			},
		},
		{
			name: "regular execution result without detailed info",
			result: ExecutionResult{
				Created:      2,
				Updated:      0,
				Deleted:      1,
				DryRun:       false,
				DetailedInfo: "",
			},
			wantContains: []string{
				"create 2 role(s) and delete 1 role(s)",
			},
			wantNotContains: []string{
				"CREATE:",
				"UPDATE:",
				"DELETE:",
			},
		},
		{
			name: "dry run with no changes",
			result: ExecutionResult{
				Created:      0,
				Updated:      0,
				Deleted:      0,
				DryRun:       true,
				DetailedInfo: "",
			},
			wantContains: []string{
				"Dry run: No changes would be made",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.result.DetailedSummary()

			for _, want := range tt.wantContains {
				if !containsString(summary, want) {
					t.Errorf("DetailedSummary() = %q, want to contain %q", summary, want)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if containsString(summary, notWant) {
					t.Errorf("DetailedSummary() = %q, should not contain %q", summary, notWant)
				}
			}
		})
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

func TestExecutor_ExecutePlanWithMembers(t *testing.T) {
	tests := []struct {
		name            string
		plan            SyncPlan
		wantCreated     int
		wantUpdated     int
		wantDeleted     int
		expectError     bool
		expectedMembers map[string][]string // roleName -> members that should be assigned
	}{
		{
			name: "create role with members",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "admin",
						Resources: models.Resources{
							Allowed: []string{"*"},
							Denied:  []string{},
						},
						Members: []string{"john@example.com", "jane@example.com"},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			wantCreated: 1,
			wantUpdated: 0,
			wantDeleted: 0,
			expectError: false,
			expectedMembers: map[string][]string{
				"admin": {"john@example.com", "jane@example.com"},
			},
		},
		{
			name: "update role with member changes",
			plan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{
					{
						Name: "viewer",
						Local: models.Role{
							Name: "viewer",
							Resources: models.Resources{
								Allowed: []string{"read"},
								Denied:  []string{},
							},
							Members: []string{"alice@example.com", "bob@example.com"},
						},
						Remote: models.Role{
							Name: "viewer",
							Resources: models.Resources{
								Allowed: []string{"read"},
								Denied:  []string{},
							},
							Members: []string{"charlie@example.com"},
						},
					},
				},
				Deletes: []string{},
			},
			wantCreated: 0,
			wantUpdated: 1,
			wantDeleted: 0,
			expectError: false,
			expectedMembers: map[string][]string{
				"viewer": {"alice@example.com", "bob@example.com"},
			},
		},
		{
			name: "member assignment fails",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "admin",
						Resources: models.Resources{
							Allowed: []string{"*"},
							Denied:  []string{},
						},
						Members: []string{"invalid@example.com"},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			wantCreated: 1,
			wantUpdated: 0,
			wantDeleted: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAPIClientWithMembers{
				MockAPIClient: MockAPIClient{},
				AssignMemberRoleFunc: func(memberEmail, roleID string) error {
					if memberEmail == "invalid@example.com" {
						return errors.New("member not found")
					}
					return nil
				},
				GetTeamMembersFunc: func() ([]models.TeamMember, error) {
					return []models.TeamMember{
						{ID: "1", Email: "john@example.com"},
						{ID: "2", Email: "jane@example.com"},
						{ID: "3", Email: "alice@example.com"},
						{ID: "4", Email: "bob@example.com"},
						{ID: "5", Email: "charlie@example.com"},
					}, nil
				},
			}

			executor := NewExecutorWithMembers(mockClient, createTestLogger())
			result := executor.ExecutePlan(tt.plan)

			// Check basic execution results
			if result.Created != tt.wantCreated {
				t.Errorf("ExecutePlan() Created = %v, want %v", result.Created, tt.wantCreated)
			}
			if result.Updated != tt.wantUpdated {
				t.Errorf("ExecutePlan() Updated = %v, want %v", result.Updated, tt.wantUpdated)
			}
			if result.Deleted != tt.wantDeleted {
				t.Errorf("ExecutePlan() Deleted = %v, want %v", result.Deleted, tt.wantDeleted)
			}

			// Check error expectation
			if tt.expectError && result.Error == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
			}

			// Check member assignments if no error expected
			if !tt.expectError && tt.expectedMembers != nil {
				for roleName, expectedMembers := range tt.expectedMembers {
					actualMembers := mockClient.AssignedMembers[roleName]
					if len(actualMembers) != len(expectedMembers) {
						t.Errorf("Role %s: expected %d members, got %d", roleName, len(expectedMembers), len(actualMembers))
						continue
					}
					for _, expectedMember := range expectedMembers {
						found := false
						for _, actualMember := range actualMembers {
							if actualMember == expectedMember {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("Role %s: expected member %s not found in assignments", roleName, expectedMember)
						}
					}
				}
			}
		})
	}
}

func TestExecutePlanDryRunWithDiffs_Members(t *testing.T) {
	plan := SyncPlan{
		Creates: []models.Role{},
		Updates: []RoleUpdate{
			{
				Name: "admin",
				Local: models.Role{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
					Members: []string{"john@example.com", "jane@example.com"},
				},
				Remote: models.Role{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
					Members: []string{"bob@example.com"},
				},
			},
		},
		Deletes: []string{},
	}

	mockClient := &MockAPIClientWithMembers{
		MockAPIClient: MockAPIClient{},
	}
	executor := NewExecutorWithMembers(mockClient, createTestLogger())
	result := executor.ExecutePlanDryRunWithDiffs(plan)

	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}

	// Check that detailed info includes member changes
	if result.DetailedInfo == "" {
		t.Error("Expected detailed info but got empty string")
	}

	// Should contain member diff information
	expectedStrings := []string{
		"UPDATE: admin",
		"+ members: john@example.com",
		"+ members: jane@example.com",
		"- members: bob@example.com",
	}

	for _, expected := range expectedStrings {
		if !findInString(result.DetailedInfo, expected) {
			t.Errorf("Expected to find %q in detailed info: %s", expected, result.DetailedInfo)
		}
	}
}

// MockAPIClientWithMembers extends MockAPIClient with member operations
type MockAPIClientWithMembers struct {
	MockAPIClient
	GetTeamMembersFunc   func() ([]models.TeamMember, error)
	AssignMemberRoleFunc func(memberEmail, roleID string) error
	InviteUserFunc       func(email, policyID string) (*models.InviteUserResponse, error)

	// Track member assignments and invites for verification
	AssignedMembers map[string][]string                     // roleName -> list of member emails
	InvitedMembers  map[string]*models.InviteUserResponse   // email -> invite response
}

func (m *MockAPIClientWithMembers) GetTeamMembers() ([]models.TeamMember, error) {
	if m.GetTeamMembersFunc != nil {
		return m.GetTeamMembersFunc()
	}
	return []models.TeamMember{}, nil
}

func (m *MockAPIClientWithMembers) AssignMemberRole(memberEmail, roleID string) error {
	if m.AssignedMembers == nil {
		m.AssignedMembers = make(map[string][]string)
	}
	m.AssignedMembers[roleID] = append(m.AssignedMembers[roleID], memberEmail)

	if m.AssignMemberRoleFunc != nil {
		return m.AssignMemberRoleFunc(memberEmail, roleID)
	}
	return nil
}

func (m *MockAPIClientWithMembers) InviteUser(email, policyID string) (*models.InviteUserResponse, error) {
	if m.InvitedMembers == nil {
		m.InvitedMembers = make(map[string]*models.InviteUserResponse)
	}

	// Default response
	response := &models.InviteUserResponse{
		Email:    email,
		PolicyID: policyID,
		Status:   "pending",
	}
	m.InvitedMembers[email] = response

	if m.InviteUserFunc != nil {
		return m.InviteUserFunc(email, policyID)
	}
	return response, nil
}

func TestExecutorWithMembers_InviteMissingMembers(t *testing.T) {
	tests := []struct {
		name            string
		existingMembers []models.TeamMember
		roleMembers     []string
		expectInvites   []string
		expectAssigns   []string
		expectError     bool
	}{
		{
			name: "invite missing member and assign existing member",
			existingMembers: []models.TeamMember{
				{ID: "1", Email: "existing@example.com"},
			},
			roleMembers:   []string{"existing@example.com", "new@example.com"},
			expectInvites: []string{"new@example.com"},
			expectAssigns: []string{"existing@example.com", "new@example.com"},
			expectError:   false,
		},
		{
			name: "all members exist - no invites needed",
			existingMembers: []models.TeamMember{
				{ID: "1", Email: "alice@example.com"},
				{ID: "2", Email: "bob@example.com"},
			},
			roleMembers:   []string{"alice@example.com", "bob@example.com"},
			expectInvites: []string{},
			expectAssigns: []string{"alice@example.com", "bob@example.com"},
			expectError:   false,
		},
		{
			name:            "all members need invites",
			existingMembers: []models.TeamMember{},
			roleMembers:     []string{"new1@example.com", "new2@example.com"},
			expectInvites:   []string{"new1@example.com", "new2@example.com"},
			expectAssigns:   []string{"new1@example.com", "new2@example.com"},
			expectError:     false,
		},
		{
			name:            "no members to assign",
			existingMembers: []models.TeamMember{},
			roleMembers:     []string{},
			expectInvites:   []string{},
			expectAssigns:   []string{},
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAPIClientWithMembers{
				MockAPIClient: MockAPIClient{},
				GetTeamMembersFunc: func() ([]models.TeamMember, error) {
					return tt.existingMembers, nil
				},
			}

			executor := NewExecutorWithMembers(mockClient, createTestLogger())

			role := models.Role{
				ID:      "test-role-id",
				Name:    "test-role",
				Members: tt.roleMembers,
			}

			// Test the member assignment logic directly
			err := executor.assignMembersToRole(role.Name, role.Members)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify invites
			if len(mockClient.InvitedMembers) != len(tt.expectInvites) {
				t.Errorf("Expected %d invites, got %d", len(tt.expectInvites), len(mockClient.InvitedMembers))
			}
			for _, expectedEmail := range tt.expectInvites {
				if _, found := mockClient.InvitedMembers[expectedEmail]; !found {
					t.Errorf("Expected invite for %s but not found", expectedEmail)
				}
			}

			// Verify assignments
			assignedMembers := mockClient.AssignedMembers[role.Name]
			if len(assignedMembers) != len(tt.expectAssigns) {
				t.Errorf("Expected %d assignments, got %d", len(tt.expectAssigns), len(assignedMembers))
			}
			for _, expectedEmail := range tt.expectAssigns {
				found := false
				for _, assignedEmail := range assignedMembers {
					if assignedEmail == expectedEmail {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected assignment for %s but not found", expectedEmail)
				}
			}
		})
	}
}

func TestExecutorWithMembers_InviteFailure(t *testing.T) {
	mockClient := &MockAPIClientWithMembers{
		MockAPIClient: MockAPIClient{},
		GetTeamMembersFunc: func() ([]models.TeamMember, error) {
			return []models.TeamMember{}, nil // No existing members
		},
		InviteUserFunc: func(email, policyID string) (*models.InviteUserResponse, error) {
			if email == "fail@example.com" {
				return nil, errors.New("invite failed")
			}
			return &models.InviteUserResponse{
				Email:    email,
				PolicyID: policyID,
				Status:   "pending",
			}, nil
		},
	}

	executor := NewExecutorWithMembers(mockClient, createTestLogger())

	// Test invite failure
	err := executor.assignMembersToRole("test-role", []string{"fail@example.com"})
	if err == nil {
		t.Errorf("Expected error from invite failure but got none")
	}
	if !strings.Contains(err.Error(), "failed to invite member") {
		t.Errorf("Expected invite failure error, got: %v", err)
	}
}

func TestExecutorWithMembers_GetTeamMembersFailure(t *testing.T) {
	mockClient := &MockAPIClientWithMembers{
		MockAPIClient: MockAPIClient{},
		GetTeamMembersFunc: func() ([]models.TeamMember, error) {
			return nil, errors.New("failed to get team members")
		},
	}

	executor := NewExecutorWithMembers(mockClient, createTestLogger())

	// Test GetTeamMembers failure
	err := executor.assignMembersToRole("test-role", []string{"test@example.com"})
	if err == nil {
		t.Errorf("Expected error from GetTeamMembers failure but got none")
	}
	if !strings.Contains(err.Error(), "failed to get team members") {
		t.Errorf("Expected GetTeamMembers failure error, got: %v", err)
	}
}

func TestExecutorWithMembers_AutoInviteDisabled(t *testing.T) {
	mockClient := &MockAPIClientWithMembers{
		MockAPIClient: MockAPIClient{},
		GetTeamMembersFunc: func() ([]models.TeamMember, error) {
			return []models.TeamMember{
				{ID: "1", Email: "existing@example.com"},
			}, nil
		},
	}

	// Create executor with auto-invite disabled
	executor := NewExecutorWithMembersAndInvite(mockClient, createTestLogger(), false)

	// Test with missing members
	err := executor.assignMembersToRole("test-role", []string{"existing@example.com", "new@example.com"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify no invites were sent
	if len(mockClient.InvitedMembers) != 0 {
		t.Errorf("Expected no invites when auto-invite is disabled, got %d", len(mockClient.InvitedMembers))
	}

	// Verify assignments still happened
	assignedMembers := mockClient.AssignedMembers["test-role"]
	if len(assignedMembers) != 2 {
		t.Errorf("Expected 2 assignments, got %d", len(assignedMembers))
	}
}
