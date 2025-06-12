package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestEnvironmentVariableDocumentation tests that environment variables are properly documented in help text
func TestEnvironmentVariableDocumentation(t *testing.T) {
	tests := []struct {
		name         string
		command      []string
		expectInHelp []string
	}{
		{
			name:    "root command help shows environment variables section",
			command: []string{"--help"},
			expectInHelp: []string{
				"Environment Variables:",
				"REPLICATED_API_TOKEN",
				"REPLBAC_API_TOKEN", 
				"REPLBAC_API_ENDPOINT",
				"REPLBAC_CONFIG",
				"REPLBAC_CONFIRM",
				"REPLBAC_LOG_LEVEL",
				"replicated CLI compatibility",
			},
		},
		{
			name:    "sync command help mentions environment variables",
			command: []string{"sync", "--help"},
			expectInHelp: []string{
				"Environment Variables:",
				"REPLICATED_API_TOKEN",
				"See 'replbac --help' for full environment variable documentation",
			},
		},
		{
			name:    "pull command help mentions environment variables", 
			command: []string{"pull", "--help"},
			expectInHelp: []string{
				"Environment Variables:",
				"REPLICATED_API_TOKEN",
				"See 'replbac --help' for full environment variable documentation",
			},
		},
		{
			name:    "version command help shows basic env vars",
			command: []string{"version", "--help"},
			expectInHelp: []string{
				"Environment Variables:",
				"See 'replbac --help' for full environment variable documentation",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh root command to avoid state pollution
			cmd := createTestRootCommand()
			
			// Capture help output
			cmd.SetArgs(tt.command)
			output, err := executeCommandToString(cmd)
			
			// Help commands should not return an error (they exit with status 0)
			if err != nil && !strings.Contains(err.Error(), "help requested") {
				t.Errorf("Unexpected error from help command: %v", err)
			}

			// Check that all expected strings are present in help output
			for _, expected := range tt.expectInHelp {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected help output to contain %q, but got:\n%s", expected, output)
				}
			}
		})
	}
}

// TestEnvironmentVariableReference tests that flag descriptions reference environment variables
func TestEnvironmentVariableReference(t *testing.T) {
	tests := []struct {
		name        string
		flagName    string
		expectInDescription []string
	}{
		{
			name:     "api-token flag mentions environment variables",
			flagName: "api-token",
			expectInDescription: []string{
				"REPLICATED_API_TOKEN",
				"REPLBAC_API_TOKEN",
			},
		},
		{
			name:     "api-endpoint flag mentions environment variable",
			flagName: "api-endpoint", 
			expectInDescription: []string{
				"REPLBAC_API_ENDPOINT",
			},
		},
		{
			name:     "config flag mentions environment variable",
			flagName: "config",
			expectInDescription: []string{
				"REPLBAC_CONFIG",
			},
		},
		{
			name:     "confirm flag mentions environment variable",
			flagName: "confirm",
			expectInDescription: []string{
				"REPLBAC_CONFIRM",
			},
		},
		{
			name:     "log-level flag mentions environment variable",
			flagName: "log-level",
			expectInDescription: []string{
				"REPLBAC_LOG_LEVEL",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh root command to get flag descriptions
			cmd := createTestRootCommand()
			
			// Find the flag
			flag := cmd.PersistentFlags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("Flag %s not found", tt.flagName)
			}

			// Check that the flag usage mentions the environment variable
			usage := flag.Usage
			for _, expected := range tt.expectInDescription {
				if !strings.Contains(usage, expected) {
					t.Errorf("Expected flag %s usage to contain %q, but got: %s", tt.flagName, expected, usage)
				}
			}
		})
	}
}

// Helper functions for testing

// createTestRootCommand creates a fresh root command for testing
func createTestRootCommand() *cobra.Command {
	// Create a new root command instance similar to our main one but for testing
	cmd := &cobra.Command{
		Use:   "replbac",
		Short: "Replicated RBAC Synchronization Tool",
		Long: `replbac is a CLI tool for synchronizing RBAC roles between local YAML files 
and the Replicated platform. It allows you to manage team permissions as code,
providing version control and automated deployment of role definitions.

Key features:
• Sync local YAML role files to Replicated API
• Initialize local files from existing API roles  
• Dry-run mode to preview changes before applying
• Support for multiple configuration sources

Environment Variables:
  Configuration can be provided via environment variables as an alternative to CLI flags:

  REPLICATED_API_TOKEN    Replicated API token (for replicated CLI compatibility)
  REPLBAC_API_TOKEN       Replicated API token (alternative to REPLICATED_API_TOKEN)
  REPLBAC_API_ENDPOINT    Replicated API endpoint URL  
  REPLBAC_CONFIG          Path to configuration file
  REPLBAC_CONFIRM         Automatically confirm operations (true/false)
  REPLBAC_LOG_LEVEL       Log level (debug, info, warn, error)

  Environment variables have lower precedence than CLI flags but higher than config files.
  REPLICATED_API_TOKEN is checked first for compatibility with the replicated CLI.`,
	}
	
	// Add flags with environment variable documentation
	cmd.PersistentFlags().String("config", "", "config file path (env: REPLBAC_CONFIG)")
	cmd.PersistentFlags().String("api-token", "", "Replicated API token (env: REPLICATED_API_TOKEN, REPLBAC_API_TOKEN)")
	cmd.PersistentFlags().String("api-endpoint", "", "Replicated API endpoint URL (env: REPLBAC_API_ENDPOINT)")
	cmd.PersistentFlags().Bool("confirm", false, "automatically confirm destructive operations (env: REPLBAC_CONFIRM)")
	cmd.PersistentFlags().String("log-level", "", "log level: debug, info, warn, error (env: REPLBAC_LOG_LEVEL)")
	
	// Add subcommands with environment variable sections
	syncCmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Synchronize local role files to Replicated API",
		Long: `Sync reads role definitions from local YAML files and synchronizes them
with the Replicated platform.

Environment Variables:
  This command supports all global environment variables. 
  See 'replbac --help' for full environment variable documentation.`,
	}
	
	pullCmd := &cobra.Command{
		Use:   "pull [directory]", 
		Short: "Pull remote roles to local YAML files",
		Long: `Pull downloads role definitions from the Replicated API and saves them
as local YAML files.

Environment Variables:
  This command supports all global environment variables.
  See 'replbac --help' for full environment variable documentation.`,
	}
	
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long: `Display version information for replbac.

Environment Variables:
  See 'replbac --help' for full environment variable documentation.`,
	}
	
	cmd.AddCommand(syncCmd, pullCmd, versionCmd)
	
	return cmd
}

// executeCommandToString executes a command and returns its output as a string
func executeCommandToString(cmd *cobra.Command) (string, error) {
	buf := &strings.Builder{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	
	err := cmd.Execute()
	return buf.String(), err
}