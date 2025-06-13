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
	"gopkg.in/yaml.v3"

	"replbac/internal/models"
)

// TestSyncCommandWithMembers tests that sync command properly handles roles with members
func TestSyncCommandWithMembers(t *testing.T) {
	tests := []struct {
		name          string
		localRoles    []models.Role
		remoteRoles   []models.Role
		expectMembers bool
		expectedCalls map[string][]string // roleName -> member emails that should be assigned
	}{
		{
			name: "sync role with members uses ExecutorWithMembers",
			localRoles: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
					Members: []string{"john@example.com", "jane@example.com"},
				},
			},
			remoteRoles:   []models.Role{},
			expectMembers: true,
			expectedCalls: map[string][]string{
				"admin": {"john@example.com", "jane@example.com"},
			},
		},
		{
			name: "sync role without members uses regular Executor",
			localRoles: []models.Role{
				{
					Name: "viewer",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
					Members: nil,
				},
			},
			remoteRoles:   []models.Role{},
			expectMembers: false,
			expectedCalls: map[string][]string{},
		},
		{
			name: "sync mixed roles with and without members",
			localRoles: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
					Members: []string{"admin@example.com"},
				},
				{
					Name: "viewer",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
					Members: nil,
				},
			},
			remoteRoles:   []models.Role{},
			expectMembers: true,
			expectedCalls: map[string][]string{
				"admin": {"admin@example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client that tracks member assignments
			mockClient := &MockAPIClientWithMemberTracking{
				roles:             tt.remoteRoles,
				memberAssignments: make(map[string][]string),
			}

			// Create temporary directory with test roles
			tempDir := t.TempDir()
			for _, role := range tt.localRoles {
				err := createTestRoleFile(tempDir, role)
				if err != nil {
					t.Fatalf("Failed to create test role file: %v", err)
				}
			}

			// Create command and capture output
			cmd := &cobra.Command{}
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Add flags to command for testing
			cmd.Flags().Bool("verbose", false, "verbose logging")
			cmd.Flags().Bool("debug", false, "debug logging")

			// Run sync command (not dry-run so we can verify member assignments)
			err := RunSyncCommandWithClient(cmd, []string{tempDir}, mockClient, false, false, false)
			if err != nil {
				t.Fatalf("RunSyncCommandWithClient failed: %v", err)
			}

			// Verify member assignment calls were made as expected
			if tt.expectMembers {
				for roleName, expectedMembers := range tt.expectedCalls {
					actualMembers, found := mockClient.memberAssignments[roleName]
					if !found {
						t.Errorf("Expected member assignments for role %s, but none were made", roleName)
						continue
					}

					if len(actualMembers) != len(expectedMembers) {
						t.Errorf("Role %s: expected %d member assignments, got %d", roleName, len(expectedMembers), len(actualMembers))
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
			} else {
				// Should have no member assignments
				if len(mockClient.memberAssignments) > 0 {
					t.Errorf("Expected no member assignments, but got %d", len(mockClient.memberAssignments))
				}
			}
		})
	}
}

// TestPullCommandWithMembers tests that pull command preserves member fields
func TestPullCommandWithMembers(t *testing.T) {
	// Test data with members
	rolesWithMembers := []models.Role{
		{
			Name: "admin",
			Resources: models.Resources{
				Allowed: []string{"*"},
				Denied:  []string{},
			},
			Members: []string{"admin@example.com", "manager@example.com"},
		},
		{
			Name: "viewer",
			Resources: models.Resources{
				Allowed: []string{"read"},
				Denied:  []string{},
			},
			Members: []string{"user1@example.com", "user2@example.com"},
		},
	}

	mockClient := &MockAPIClientWithMemberTracking{
		roles: rolesWithMembers,
	}

	// Create command and capture output
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	tempDir := t.TempDir()

	// Run pull command
	err := RunPullCommandWithClient(cmd, tempDir, true, false, false, mockClient) // dry-run
	if err != nil {
		t.Fatalf("RunPullCommandWithClient failed: %v", err)
	}

	// Verify output mentions member fields
	output := stdout.String()
	if !strings.Contains(output, "admin") || !strings.Contains(output, "viewer") {
		t.Errorf("Expected role names in output, got: %s", output)
	}

	// The actual YAML generation is tested in roles package,
	// here we just verify the command runs successfully with member data
}

// MockAPIClientWithMemberTracking implements api.ClientInterface and tracks member assignments
type MockAPIClientWithMemberTracking struct {
	roles             []models.Role
	memberAssignments map[string][]string // roleName -> assigned member emails
	teamMembers       []models.TeamMember
}

func (m *MockAPIClientWithMemberTracking) GetRoles() ([]models.Role, error) {
	return m.roles, nil
}

func (m *MockAPIClientWithMemberTracking) GetRolesWithContext(ctx context.Context) ([]models.Role, error) {
	return m.GetRoles()
}

func (m *MockAPIClientWithMemberTracking) GetRole(roleName string) (models.Role, error) {
	for _, role := range m.roles {
		if role.Name == roleName {
			return role, nil
		}
	}
	return models.Role{}, fmt.Errorf("role not found: %s", roleName)
}

func (m *MockAPIClientWithMemberTracking) GetRoleWithContext(ctx context.Context, roleName string) (models.Role, error) {
	return m.GetRole(roleName)
}

func (m *MockAPIClientWithMemberTracking) CreateRole(role models.Role) error {
	m.roles = append(m.roles, role)
	return nil
}

func (m *MockAPIClientWithMemberTracking) CreateRoleWithContext(ctx context.Context, role models.Role) error {
	return m.CreateRole(role)
}

func (m *MockAPIClientWithMemberTracking) UpdateRole(role models.Role) error {
	for i, existingRole := range m.roles {
		if existingRole.Name == role.Name {
			m.roles[i] = role
			return nil
		}
	}
	return fmt.Errorf("role not found for update: %s", role.Name)
}

func (m *MockAPIClientWithMemberTracking) UpdateRoleWithContext(ctx context.Context, role models.Role) error {
	return m.UpdateRole(role)
}

func (m *MockAPIClientWithMemberTracking) DeleteRole(roleName string) error {
	for i, role := range m.roles {
		if role.Name == roleName {
			m.roles = append(m.roles[:i], m.roles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("role not found for deletion: %s", roleName)
}

func (m *MockAPIClientWithMemberTracking) DeleteRoleWithContext(ctx context.Context, roleName string) error {
	return m.DeleteRole(roleName)
}

func (m *MockAPIClientWithMemberTracking) GetTeamMembers() ([]models.TeamMember, error) {
	if m.teamMembers == nil {
		// Return default test members
		return []models.TeamMember{
			{ID: "1", Email: "john@example.com"},
			{ID: "2", Email: "jane@example.com"},
			{ID: "3", Email: "admin@example.com"},
			{ID: "4", Email: "manager@example.com"},
			{ID: "5", Email: "user1@example.com"},
			{ID: "6", Email: "user2@example.com"},
		}, nil
	}
	return m.teamMembers, nil
}

func (m *MockAPIClientWithMemberTracking) GetTeamMembersWithContext(ctx context.Context) ([]models.TeamMember, error) {
	return m.GetTeamMembers()
}

func (m *MockAPIClientWithMemberTracking) AssignMemberRole(memberEmail, roleID string) error {
	if m.memberAssignments == nil {
		m.memberAssignments = make(map[string][]string)
	}
	m.memberAssignments[roleID] = append(m.memberAssignments[roleID], memberEmail)
	return nil
}

func (m *MockAPIClientWithMemberTracking) AssignMemberRoleWithContext(ctx context.Context, memberEmail, roleID string) error {
	return m.AssignMemberRole(memberEmail, roleID)
}

// createTestRoleFile creates a YAML file for a test role
func createTestRoleFile(dir string, role models.Role) error {
	filename := filepath.Join(dir, role.Name+".yaml")

	// Create YAML content
	yamlData, err := yaml.Marshal(role)
	if err != nil {
		return fmt.Errorf("failed to marshal role to YAML: %w", err)
	}

	// #nosec G306 -- Test files need readable permissions
	return os.WriteFile(filename, yamlData, 0644)
}
