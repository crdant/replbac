package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"replbac/internal/models"
)

// MockInviteAPICalls tracks invite-related API calls
type MockInviteAPICalls struct {
	MockAPICalls
	InvitedUsers []InviteCall
}

type InviteCall struct {
	Email    string
	PolicyID string
}

// MockInviteClient extends MockClient with invite functionality
type MockInviteClient struct {
	*MockClient
	inviteAPICalls *MockInviteAPICalls
	teamMembers    []models.TeamMember
	inviteResponse *models.InviteUserResponse
	inviteError    error
}

func (m *MockInviteClient) GetTeamMembers() ([]models.TeamMember, error) {
	return m.teamMembers, nil
}

func (m *MockInviteClient) GetTeamMembersWithContext(ctx context.Context) ([]models.TeamMember, error) {
	return m.GetTeamMembers()
}

func (m *MockInviteClient) AssignMemberRole(memberEmail, roleID string) error {
	// Track member assignments
	return nil
}

func (m *MockInviteClient) AssignMemberRoleWithContext(ctx context.Context, memberEmail, roleID string) error {
	return m.AssignMemberRole(memberEmail, roleID)
}

func (m *MockInviteClient) InviteUser(email, policyID string) (*models.InviteUserResponse, error) {
	m.inviteAPICalls.InvitedUsers = append(m.inviteAPICalls.InvitedUsers, InviteCall{
		Email:    email,
		PolicyID: policyID,
	})
	
	if m.inviteError != nil {
		return nil, m.inviteError
	}
	
	if m.inviteResponse != nil {
		return m.inviteResponse, nil
	}
	
	return &models.InviteUserResponse{
		Email:    email,
		PolicyID: policyID,
		Status:   "pending",
	}, nil
}

func (m *MockInviteClient) InviteUserWithContext(ctx context.Context, email, policyID string) (*models.InviteUserResponse, error) {
	return m.InviteUser(email, policyID)
}

func TestSyncInviteFlags(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		flags             map[string]string
		localFiles        map[string]string
		teamMembers       []models.TeamMember
		expectError       bool
		expectInvites     []InviteCall
		expectOutput      []string
		expectNotInOutput []string
	}{
		{
			name: "auto-invite enabled for missing members",
			args: []string{},
			flags: map[string]string{
				"auto-invite": "true",
				"dry-run":     "true",
			},
			localFiles: map[string]string{
				"role-with-members.yaml": `name: test-role
members:
  - existing@example.com
  - new@example.com
resources:
  allowed: ["read"]
  denied: []`,
			},
			teamMembers: []models.TeamMember{
				{ID: "1", Email: "existing@example.com"},
			},
			expectError:   false,
			expectInvites: []InviteCall{{Email: "new@example.com", PolicyID: "test-role"}},
			expectOutput:  []string{"Will invite 1 new member(s)", "new@example.com"},
		},
		{
			name: "auto-invite disabled - no invites sent",
			args: []string{},
			flags: map[string]string{
				"auto-invite": "false",
				"dry-run":     "true",
			},
			localFiles: map[string]string{
				"role-with-members.yaml": `name: test-role
members:
  - existing@example.com
  - new@example.com
resources:
  allowed: ["read"]
  denied: []`,
			},
			teamMembers: []models.TeamMember{
				{ID: "1", Email: "existing@example.com"},
			},
			expectError:       false,
			expectInvites:     []InviteCall{},
			expectOutput:      []string{"Will assign members"},
			expectNotInOutput: []string{"Will invite"},
		},
		{
			name: "no-invite flag overrides auto-invite",
			args: []string{},
			flags: map[string]string{
				"auto-invite": "true",
				"no-invite":   "true",
				"dry-run":     "true",
			},
			localFiles: map[string]string{
				"role-with-members.yaml": `name: test-role
members:
  - new@example.com
resources:
  allowed: ["read"]
  denied: []`,
			},
			teamMembers:       []models.TeamMember{},
			expectError:       false,
			expectInvites:     []InviteCall{},
			expectOutput:      []string{"Will assign members"},
			expectNotInOutput: []string{"Will invite"},
		},
		{
			name: "all members exist - no invites needed",
			args: []string{},
			flags: map[string]string{
				"auto-invite": "true",
				"dry-run":     "true",
			},
			localFiles: map[string]string{
				"role-with-members.yaml": `name: test-role
members:
  - existing1@example.com
  - existing2@example.com
resources:
  allowed: ["read"]
  denied: []`,
			},
			teamMembers: []models.TeamMember{
				{ID: "1", Email: "existing1@example.com"},
				{ID: "2", Email: "existing2@example.com"},
			},
			expectError:       false,
			expectInvites:     []InviteCall{},
			expectOutput:      []string{"Will assign members"},
			expectNotInOutput: []string{"Will invite"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test files
			tempDir := t.TempDir()

			// Create local YAML files
			for filename, content := range tt.localFiles {
				filePath := filepath.Join(tempDir, filename)
				// #nosec G306 -- Test files need readable permissions
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", filename, err)
				}
			}

			// Create mock API calls tracking
			mockCalls := &MockInviteAPICalls{
				MockAPICalls:  MockAPICalls{},
				InvitedUsers: []InviteCall{},
			}

			// Create mock client
			baseClient := NewMockClient(&mockCalls.MockAPICalls, []models.Role{})
			mockClient := &MockInviteClient{
				MockClient:     baseClient,
				inviteAPICalls: mockCalls,
				teamMembers:    tt.teamMembers,
			}

			// Create command with flags
			cmd := NewSyncCommandWithInviteSupport(mockClient)
			
			// Set flags
			for flagName, flagValue := range tt.flags {
				if err := cmd.Flags().Set(flagName, flagValue); err != nil {
					t.Fatalf("Failed to set flag %s=%s: %v", flagName, flagValue, err)
				}
			}

			// Capture output
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			// Run command
			cmd.SetArgs(append(tt.args, tempDir))
			err := cmd.Execute()

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check invites
			if len(mockCalls.InvitedUsers) != len(tt.expectInvites) {
				t.Errorf("Expected %d invites, got %d", len(tt.expectInvites), len(mockCalls.InvitedUsers))
			}
			for i, expectedInvite := range tt.expectInvites {
				if i < len(mockCalls.InvitedUsers) {
					actualInvite := mockCalls.InvitedUsers[i]
					if actualInvite.Email != expectedInvite.Email || actualInvite.PolicyID != expectedInvite.PolicyID {
						t.Errorf("Expected invite %+v, got %+v", expectedInvite, actualInvite)
					}
				}
			}

			// Check output
			outputStr := output.String()
			for _, expected := range tt.expectOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				}
			}
			for _, notExpected := range tt.expectNotInOutput {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain %q, but got:\n%s", notExpected, outputStr)
				}
			}
		})
	}
}

// NewSyncCommandWithInviteSupport creates a sync command for testing invite functionality
func NewSyncCommandWithInviteSupport(mockClient *MockInviteClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Synchronize local role files to Replicated API",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			autoInvite, _ := cmd.Flags().GetBool("auto-invite")
			noInvite, _ := cmd.Flags().GetBool("no-invite")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			
			return RunSyncCommandWithInviteClient(cmd, args, mockClient, dryRun, autoInvite, !noInvite)
		},
	}

	// Add flags
	cmd.Flags().Bool("auto-invite", false, "automatically invite missing team members")
	cmd.Flags().Bool("no-invite", false, "disable automatic invitation of missing members")
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")

	return cmd
}

// RunSyncCommandWithInviteClient runs sync with invite support for testing
func RunSyncCommandWithInviteClient(cmd *cobra.Command, args []string, client *MockInviteClient, dryRun, autoInvite, enableInvite bool) error {
	// This is a simplified version for testing
	// The real implementation will be added when we implement the actual functionality
	
	// For now, just simulate the behavior in dry-run mode
	if dryRun {
		// This logic will fail the tests until we implement the real functionality
		cmd.Print("Will assign members\n")
	}
	return nil
}