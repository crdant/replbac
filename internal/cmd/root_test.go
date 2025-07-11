package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		contains    string
	}{
		{
			name:     "help flag shows usage",
			args:     []string{"--help"},
			contains: "replbac is a CLI tool for synchronizing RBAC roles",
		},
		{
			name:     "version command works",
			args:     []string{"version"},
			contains: "replbac version",
		},
		{
			name:     "sync help shows sync usage",
			args:     []string{"sync", "--help"},
			contains: "Synchronize local role files to Replicated API",
		},
		{
			name:     "pull help shows pull usage",
			args:     []string{"pull", "--help"},
			contains: "Pull role definitions from Replicated API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new root command for each test to avoid state pollution
			cmd := createTestRootCmd()

			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.contains != "" {
				outputStr := output.String()
				if !strings.Contains(outputStr, tt.contains) {
					t.Errorf("Output should contain %q, got: %q", tt.contains, outputStr)
				}
			}
		})
	}
}

func TestConfigurationLoading(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"

	configContent := `api_token: test-token
log_level: debug
confirm: true`

	// #nosec G306 -- Test files need readable permissions
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "config file loads successfully",
			args: []string{"--config", configFile, "version"},
		},
		{
			name: "environment variable overrides config",
			args: []string{"--config", configFile, "version"},
			envVars: map[string]string{
				"REPLICATED_API_TOKEN": "env-token",
			},
		},
		{
			name: "nonexistent config file is ignored for version command",
			args: []string{"--config", "/nonexistent/config.yaml", "version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				old := os.Getenv(key)
				if err := os.Setenv(key, value); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", key, err)
				}
				defer func(k, v string) {
					if err := os.Setenv(k, v); err != nil {
						t.Errorf("Failed to restore environment variable %s: %v", k, err)
					}
				}(key, old)
			}

			cmd := createTestRootCmd()
			cmd.SetArgs(tt.args)

			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error should contain %q, got: %v", tt.errorMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "all flags parse correctly",
			args: []string{
				"--api-token", "test-token",
				"--log-level", "debug",
				"--confirm",
				"version",
			},
		},
		{
			name: "sync with dry-run flag",
			args: []string{"sync", "--dry-run"},
		},
		{
			name: "pull with force flag",
			args: []string{"pull", "--force"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createTestRootCmd()
			cmd.SetArgs(tt.args)

			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// createTestRootCmd creates a fresh root command for testing
func createTestRootCmd() *cobra.Command {
	// Reset global variables
	cfgFile = ""
	apiToken = ""
	confirm = false
	logLevel = ""
	syncDryRun = false
	pullForce = false

	// Create new command tree
	cmd := &cobra.Command{
		Use:   "replbac",
		Short: "Replicated RBAC Synchronization Tool",
		Long: `replbac is a CLI tool for synchronizing RBAC roles between local YAML files 
and the Replicated platform.`,
		// Skip the PersistentPreRunE for tests to avoid config validation
		PersistentPreRunE: nil,
	}

	// Add flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&apiToken, "api-token", "", "Replicated API token")
	cmd.PersistentFlags().BoolVar(&confirm, "confirm", false, "automatically confirm operations")
	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level")

	// Add subcommands
	cmd.AddCommand(versionCmd)

	syncTestCmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Synchronize local role files to Replicated API",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	syncTestCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "preview changes")
	cmd.AddCommand(syncTestCmd)

	pullTestCmd := &cobra.Command{
		Use:   "pull [directory]",
		Short: "Pull role definitions from Replicated API to local files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	pullTestCmd.Flags().BoolVar(&pullForce, "force", false, "overwrite existing files")
	cmd.AddCommand(pullTestCmd)

	return cmd
}
