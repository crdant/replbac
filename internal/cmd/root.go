package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"replbac/internal/config"
	"replbac/internal/models"
)

var (
	cfgFile  string
	cfg      models.Config
	apiToken string
	confirm  bool
	logLevel string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "replbac",
	Short: "Replicated RBAC Synchronization Tool",
	Long: `replbac is a CLI tool for synchronizing RBAC roles between local YAML files 
and the Replicated platform. It allows you to manage team permissions as code,
providing version control and automated deployment of role definitions.

Key features:
• Sync local YAML role files to Replicated API
• Initialize local files from existing API roles  
• Dry-run mode to preview changes before applying
• Support for multiple configuration sources`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		var err error
		
		// If config file is specified, use it; otherwise use defaults
		if cfgFile != "" {
			cfg, err = config.LoadConfig(cfgFile)
		} else {
			cfg, err = config.LoadConfigWithDefaults(nil)
		}
		
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		
		// Override config with command-line flags if provided
		if apiToken != "" {
			cfg.APIToken = apiToken
		}
		if cmd.Flags().Changed("confirm") {
			cfg.Confirm = confirm
		}
		if logLevel != "" {
			cfg.LogLevel = logLevel
		}
		
		// Only validate configuration for commands that need API access
		// Skip validation for version, help, and completion commands
		if cmd.Name() != "version" && cmd.Name() != "help" && cmd.Name() != "completion" {
			if err := config.ValidateConfig(cfg); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
		}
		
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ExecuteWithContext adds all child commands to the root command and sets flags appropriately.
// This version supports context cancellation for graceful shutdown.
func ExecuteWithContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (env: REPLBAC_CONFIG)")
	rootCmd.PersistentFlags().StringVar(&apiToken, "api-token", "", "Replicated API token (env: REPLICATED_API_TOKEN, REPLBAC_API_TOKEN)")
	rootCmd.PersistentFlags().BoolVar(&confirm, "confirm", false, "automatically confirm destructive operations (env: REPLBAC_CONFIRM)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level: debug, info, warn, error (env: REPLBAC_LOG_LEVEL)")
	
	// Mark sensitive flags
	rootCmd.PersistentFlags().MarkHidden("api-token")
	
	// Override help function to position environment variables before final Use message
	originalHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "replbac" {
			// Capture and modify help output to insert environment variables
			var buf strings.Builder
			originalOut := cmd.OutOrStdout()
			cmd.SetOut(&buf)
			originalHelpFunc(cmd, args)
			cmd.SetOut(originalOut)
			helpText := buf.String()
			useMessage := `Use "replbac [command] --help" for more information about a command.`
			if strings.Contains(helpText, useMessage) {
				envVars := "\nEnvironment Variables:\n" + 
					"  Configuration can be provided via environment variables as an alternative to CLI flags:\n\n" +
					"  REPLICATED_API_TOKEN    Replicated API token (for replicated CLI compatibility)\n" +
					"  REPLBAC_API_TOKEN       Replicated API token (alternative to REPLICATED_API_TOKEN)\n" +
					"  REPLBAC_CONFIG          Path to configuration file\n" +
					"  REPLBAC_CONFIRM         Automatically confirm operations (true/false)\n" +
					"  REPLBAC_LOG_LEVEL       Log level (debug, info, warn, error)\n\n" +
					"  Environment variables have lower precedence than CLI flags but higher than config files.\n" +
					"  REPLICATED_API_TOKEN is checked first for compatibility with the replicated CLI.\n\n"
				helpText = strings.Replace(helpText, useMessage, envVars + useMessage, 1)
			}
			cmd.Print(helpText)
		} else {
			originalHelpFunc(cmd, args)
		}
	})
}