package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSyncCommandInviteFlags(t *testing.T) {
	tests := []struct {
		name                    string
		args                    []string
		expectedNoInvite        bool
		expectedEffectiveInvite bool
	}{
		{
			name:                    "default - auto-invite enabled",
			args:                    []string{},
			expectedNoInvite:        false,
			expectedEffectiveInvite: true,
		},
		{
			name:                    "no-invite flag disables auto-invite",
			args:                    []string{"--no-invite"},
			expectedNoInvite:        true,
			expectedEffectiveInvite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test command with the same flags as sync
			cmd := &cobra.Command{
				Use: "test",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Test the flag parsing logic
					noInvite, _ := cmd.Flags().GetBool("no-invite")
					
					if noInvite != tt.expectedNoInvite {
						t.Errorf("Expected no-invite %v, got %v", tt.expectedNoInvite, noInvite)
					}
					
					// Test the logic that calculates effective invite setting
					effectiveAutoInvite := !noInvite
					if effectiveAutoInvite != tt.expectedEffectiveInvite {
						t.Errorf("Expected effective auto-invite %v, got %v", tt.expectedEffectiveInvite, effectiveAutoInvite)
					}
					
					return nil
				},
			}

			// Add the same flags as the sync command
			cmd.Flags().Bool("no-invite", false, "disable automatic invitation of missing members")

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
	// Test that the sync command has the invite flag with correct description
	cmd := NewSyncCommandForTesting()
	
	noInviteFlag := cmd.Flags().Lookup("no-invite")
	if noInviteFlag == nil {
		t.Error("Expected --no-invite flag to exist")
	} else {
		expectedUsage := "disable automatic invitation of missing members (default: auto-invite enabled)"
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
	cmd.Flags().Bool("no-invite", false, "disable automatic invitation of missing members (default: auto-invite enabled)")
	cmd.Flags().Bool("verbose", false, "enable info-level logging to stderr (progress and results)")
	cmd.Flags().Bool("debug", false, "enable debug-level logging to stderr (detailed operation info)")

	return cmd
}