package sync

import (
	"testing"

	"replbac/internal/models"
)

func TestExecutorWithMembers_MemberDeletions(t *testing.T) {
	tests := []struct {
		name                  string
		plan                  SyncPlan
		existingMembers       []models.TeamMember
		expectedOrphanedUsers []string
		expectedOrphanedInvites []string
		expectError           bool
		errorContains         string
	}{
		{
			name: "orphaned users and invites are identified",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name:    "admin",
						Members: []string{"keep@example.com"},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			existingMembers: []models.TeamMember{
				{ID: "1", Email: "keep@example.com", Status: "active"},
				{ID: "2", Email: "delete-user@example.com", Status: "active"},
				{ID: "3", Email: "delete-invite@example.com", Status: "pending"},
			},
			expectedOrphanedUsers:   []string{"delete-user@example.com"},
			expectedOrphanedInvites: []string{"delete-invite@example.com"},
			expectError:             false,
		},
		{
			name: "no orphans when all members are in roles",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name:    "admin",
						Members: []string{"user1@example.com", "user2@example.com"},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			existingMembers: []models.TeamMember{
				{ID: "1", Email: "user1@example.com", Status: "active"},
				{ID: "2", Email: "user2@example.com", Status: "active"},
			},
			expectedOrphanedUsers:   []string{},
			expectedOrphanedInvites: []string{},
			expectError:             false,
		},
		{
			name: "member appears in multiple roles - error",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name:    "admin",
						Members: []string{"duplicate@example.com"},
					},
					{
						Name:    "viewer",
						Members: []string{"duplicate@example.com"},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			existingMembers: []models.TeamMember{
				{ID: "1", Email: "duplicate@example.com", Status: "active"},
			},
			expectError:   true,
			errorContains: "member duplicate@example.com appears in multiple roles",
		},
		{
			name: "member in update role conflicts with create role - error",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name:    "admin",
						Members: []string{"conflict@example.com"},
					},
				},
				Updates: []RoleUpdate{
					{
						Name: "viewer",
						Local: models.Role{
							Name:    "viewer", 
							Members: []string{"conflict@example.com"},
						},
						Remote: models.Role{
							Name:    "viewer",
							Members: []string{},
						},
					},
				},
				Deletes: []string{},
			},
			existingMembers: []models.TeamMember{
				{ID: "1", Email: "conflict@example.com", Status: "active"},
			},
			expectError:   true,
			errorContains: "member conflict@example.com appears in multiple roles",
		},
		{
			name: "orphaned users only",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name:    "admin",
						Members: []string{"keep@example.com"},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			existingMembers: []models.TeamMember{
				{ID: "1", Email: "keep@example.com", Status: "active"},
				{ID: "2", Email: "delete1@example.com", Status: "active"},
				{ID: "3", Email: "delete2@example.com", Status: "active"},
			},
			expectedOrphanedUsers:   []string{"delete1@example.com", "delete2@example.com"},
			expectedOrphanedInvites: []string{},
			expectError:             false,
		},
		{
			name: "orphaned invites only",
			plan: SyncPlan{
				Creates: []models.Role{
					{
						Name:    "admin",
						Members: []string{"keep@example.com"},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			existingMembers: []models.TeamMember{
				{ID: "1", Email: "keep@example.com", Status: "active"},
				{ID: "2", Email: "delete-invite1@example.com", Status: "pending"},
				{ID: "3", Email: "delete-invite2@example.com", Status: "pending"},
			},
			expectedOrphanedUsers:   []string{},
			expectedOrphanedInvites: []string{"delete-invite1@example.com", "delete-invite2@example.com"},
			expectError:             false,
		},
	}

	for _, tt := range tests {
		testName := tt.name
		if testName == "" {
			testName = "unnamed_test"
		}
		t.Run(testName, func(t *testing.T) {
			mockClient := &MockAPIClientWithMembers{
				MockAPIClient: MockAPIClient{
					GetRoleFunc: func(roleName string) (models.Role, error) {
						return models.Role{
							ID:   "mock-id-" + roleName,
							Name: roleName,
						}, nil
					},
				},
				GetTeamMembersFunc: func() ([]models.TeamMember, error) {
					return tt.existingMembers, nil
				},
			}

			executor := NewExecutorWithMembersAndInvite(mockClient, createTestLogger(), true)
			result := executor.ExecutePlan(tt.plan)

			// Check error expectation
			if tt.expectError && result.Error == nil {
				t.Errorf("Expected error containing '%s' but got none", tt.errorContains)
				return
			}
			if !tt.expectError && result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
				return
			}
			if tt.expectError && result.Error != nil {
				if tt.errorContains != "" && !contains(result.Error.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, result.Error)
				}
				return
			}

			// Check member deletions
			if result.MemberDeletions == nil {
				if len(tt.expectedOrphanedUsers) > 0 || len(tt.expectedOrphanedInvites) > 0 {
					t.Errorf("Expected MemberDeletions but got nil")
				}
				return
			}

			// Check orphaned users
			if !slicesEqual(result.MemberDeletions.OrphanedUsers, tt.expectedOrphanedUsers) {
				t.Errorf("OrphanedUsers = %v, want %v", result.MemberDeletions.OrphanedUsers, tt.expectedOrphanedUsers)
			}

			// Check orphaned invites
			if !slicesEqual(result.MemberDeletions.OrphanedInvites, tt.expectedOrphanedInvites) {
				t.Errorf("OrphanedInvites = %v, want %v", result.MemberDeletions.OrphanedInvites, tt.expectedOrphanedInvites)
			}
		})
	}
}

func TestExecutorWithMembers_DeleteMembersAndInvites(t *testing.T) {
	tests := []struct {
		name         string
		deletions    *MemberDeletions
		expectError  bool
		expectInviteDeletes []string
		expectUserRemovals  []string
	}{
		{
			name: "delete orphaned users and invites",
			deletions: &MemberDeletions{
				OrphanedUsers:   []string{"user1@example.com", "user2@example.com"},
				OrphanedInvites: []string{"invite1@example.com", "invite2@example.com"},
			},
			expectError:         false,
			expectInviteDeletes: []string{"invite1@example.com", "invite2@example.com"},
			expectUserRemovals:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "delete only orphaned users",
			deletions: &MemberDeletions{
				OrphanedUsers:   []string{"user@example.com"},
				OrphanedInvites: []string{},
			},
			expectError:         false,
			expectInviteDeletes: []string{},
			expectUserRemovals:  []string{"user@example.com"},
		},
		{
			name: "delete only orphaned invites",
			deletions: &MemberDeletions{
				OrphanedUsers:   []string{},
				OrphanedInvites: []string{"invite@example.com"},
			},
			expectError:         false,
			expectInviteDeletes: []string{"invite@example.com"},
			expectUserRemovals:  []string{},
		},
		{
			name:                "nil deletions",
			deletions:           nil,
			expectError:         false,
			expectInviteDeletes: []string{},
			expectUserRemovals:  []string{},
		},
	}

	for _, tt := range tests {
		testName := tt.name
		if testName == "" {
			testName = "unnamed_test"
		}
		t.Run(testName, func(t *testing.T) {
			mockClient := &MockAPIClientWithMembers{
				MockAPIClient: MockAPIClient{},
			}
			mockClient.DeletedInvites = make([]string, 0)
			mockClient.RemovedUsers = make([]string, 0)

			executor := NewExecutorWithMembersAndInvite(mockClient, createTestLogger(), true)
			err := executor.DeleteMembersAndInvites(tt.deletions)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check invite deletions
			if !slicesEqual(mockClient.DeletedInvites, tt.expectInviteDeletes) {
				t.Errorf("DeletedInvites = %v, want %v", mockClient.DeletedInvites, tt.expectInviteDeletes)
			}

			// Check user removals
			if !slicesEqual(mockClient.RemovedUsers, tt.expectUserRemovals) {
				t.Errorf("RemovedUsers = %v, want %v", mockClient.RemovedUsers, tt.expectUserRemovals)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 containsAny(s, substr))))
}

func containsAny(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to compare string slices (order independent)
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	
	// Create maps for comparison
	mapA := make(map[string]int)
	mapB := make(map[string]int)
	
	for _, item := range a {
		mapA[item]++
	}
	for _, item := range b {
		mapB[item]++
	}
	
	// Compare maps
	for k, v := range mapA {
		if mapB[k] != v {
			return false
		}
	}
	
	return true
}