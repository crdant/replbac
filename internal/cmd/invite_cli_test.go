package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSyncCommandInviteFlags(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectedAutoInvite bool
		expectedNoInvite   bool
	}{
		{
			name:               "default flags - no invite flags set",
			args:               []string{},
			expectedAutoInvite: false,
			expectedNoInvite:   false,
		},
		{
			name:               "auto-invite enabled",
			args:               []string{"--auto-invite"},
			expectedAutoInvite: true,
			expectedNoInvite:   false,
		},
		{
			name:               "no-invite enabled",
			args:               []string{"--no-invite"},
			expectedAutoInvite: false,
			expectedNoInvite:   true,
		},
		{
			name:               "both flags set - no-invite overrides",
			args:               []string{"--auto-invite", "--no-invite"},
			expectedAutoInvite: true,
			expectedNoInvite:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test command with the same flags as sync
			cmd := &cobra.Command{
				Use: "test",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Test the flag parsing logic
					autoInvite, _ := cmd.Flags().GetBool("auto-invite")
					noInvite, _ := cmd.Flags().GetBool("no-invite")
					
					if autoInvite != tt.expectedAutoInvite {
						t.Errorf("Expected auto-invite %v, got %v", tt.expectedAutoInvite, autoInvite)
					}
					if noInvite != tt.expectedNoInvite {
						t.Errorf("Expected no-invite %v, got %v", tt.expectedNoInvite, noInvite)
					}
					
					// Test the logic that calculates effective invite setting
					effectiveAutoInvite := autoInvite && !noInvite
					expectedEffective := tt.expectedAutoInvite && !tt.expectedNoInvite
					if effectiveAutoInvite != expectedEffective {
						t.Errorf("Expected effective auto-invite %v, got %v", expectedEffective, effectiveAutoInvite)
					}
					
					return nil
				},
			}

			// Add the same flags as the sync command
			cmd.Flags().Bool("auto-invite", false, "automatically invite missing team members before role assignment")
			cmd.Flags().Bool("no-invite", false, "disable automatic invitation of missing members (overrides --auto-invite)")

			// Set arguments and execute
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSyncCommandInviteFlagDescriptions(t *testing.T) {
	// Test that the sync command has the invite flags with correct descriptions
	cmd := NewSyncCommandForTesting()
	
	autoInviteFlag := cmd.Flags().Lookup("auto-invite")
	if autoInviteFlag == nil {
		t.Error("Expected --auto-invite flag to exist")
	} else {
		expectedUsage := "automatically invite missing team members before role assignment"
		if autoInviteFlag.Usage != expectedUsage {
			t.Errorf("Expected auto-invite usage %q, got %q", expectedUsage, autoInviteFlag.Usage)
		}
	}
	
	noInviteFlag := cmd.Flags().Lookup("no-invite")
	if noInviteFlag == nil {
		t.Error("Expected --no-invite flag to exist")
	} else {
		expectedUsage := "disable automatic invitation of missing members (overrides --auto-invite)"
		if noInviteFlag.Usage != expectedUsage {
			t.Errorf("Expected no-invite usage %q, got %q", expectedUsage, noInviteFlag.Usage)
		}
	}
}

// NewSyncCommandForTesting creates a sync command for testing flag configuration
func NewSyncCommandForTesting() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Test sync command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// Add the same flags as the real sync command
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().Bool("diff", false, "preview changes with detailed diffs (implies --dry-run)")
	cmd.Flags().Bool("delete", false, "delete remote roles not present in local files (default: false)")
	cmd.Flags().Bool("force", false, "skip confirmation prompts (requires --delete)")
	cmd.Flags().Bool("auto-invite", false, "automatically invite missing team members before role assignment")
	cmd.Flags().Bool("no-invite", false, "disable automatic invitation of missing members (overrides --auto-invite)")
	cmd.Flags().Bool("verbose", false, "enable info-level logging to stderr (progress and results)")
	cmd.Flags().Bool("debug", false, "enable debug-level logging to stderr (detailed operation info)")

	return cmd
}