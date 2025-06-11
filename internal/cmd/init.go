package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"replbac/internal/api"
	"replbac/internal/logging"
	"replbac/internal/models"
	"replbac/internal/roles"
)

var (
	initOutputDir string
	initForce     bool
	initVerbose   bool
	initDebug     bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize local role files from Replicated API",
	Long: `Init downloads existing role definitions from the Replicated platform
and creates local YAML files. This is useful for getting started with
role management or for migrating existing roles to code.

The init operation will:
• Fetch all existing roles from the Replicated API
• Generate YAML files for each role in the specified directory
• Preserve existing files unless --force is used
• Create directory structure as needed

Use --force to overwrite existing files.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInitCommand(cmd, args, cfg, initForce, initOutputDir)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	
	// Init-specific flags
	initCmd.Flags().StringVar(&initOutputDir, "output-dir", "", "directory to create role files (default: current directory)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing files")
	initCmd.Flags().BoolVar(&initVerbose, "verbose", false, "enable info-level logging to stderr (progress and results)")
	initCmd.Flags().BoolVar(&initDebug, "debug", false, "enable debug-level logging to stderr (detailed operation info)")
}

// InitResult contains the results of init operation
type InitResult struct {
	Created     int
	Skipped     int
	Overwritten int
	Total       int
}

// RunInitCommand implements the main init logic with comprehensive error handling
func RunInitCommand(cmd *cobra.Command, args []string, config models.Config, force bool, outputDir string) error {
	// Create logger that outputs to stderr
	var logger *logging.Logger
	if initDebug {
		logger = logging.NewDebugLogger(cmd.ErrOrStderr())
	} else {
		logger = logging.NewLogger(cmd.ErrOrStderr(), initVerbose)
	}
	
	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}
	if outputDir != "" {
		targetDir = outputDir
	}

	cmd.Printf("Initializing role files in directory: %s\n", targetDir)
	logger.Debug("starting init operation in directory: %s", targetDir)
	
	if force {
		cmd.Println("FORCE: Existing files will be overwritten")
		logger.Debug("force mode enabled - existing files will be overwritten")
	}

	// Create API client
	logger.Debug("creating API client")
	client, err := api.NewClient(config.APIEndpoint, config.APIToken, logger)
	if err != nil {
		logger.Error("failed to create API client: %v", err)
		return fmt.Errorf("failed to create API client: %w", err)
	}
	
	return RunInitCommandWithClient(cmd, targetDir, force, client)
}

// RunInitCommandWithClient implements init with dependency injection for testing
func RunInitCommandWithClient(cmd *cobra.Command, outputDir string, force bool, client api.ClientInterface) error {
	// Fetch roles from API
	apiRoles, err := client.GetRoles()
	if err != nil {
		cmd.Printf("Failed to fetch roles from API: %v\n", err)
		return fmt.Errorf("failed to fetch roles from API: %w", err)
	}

	if len(apiRoles) == 0 {
		cmd.Println("No roles found in API")
		cmd.Println("Initialization completed: no files created")
		return nil
	}

	cmd.Printf("Downloaded %d role(s) from API\n", len(apiRoles))

	// Initialize result tracking
	result := InitResult{Total: len(apiRoles)}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write role files
	for _, role := range apiRoles {
		fileName := fmt.Sprintf("%s.yaml", role.Name)
		filePath := filepath.Join(outputDir, fileName)

		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			// File exists
			if force {
				// Overwrite with force
				if err := roles.WriteRoleFile(role, filePath); err != nil {
					return fmt.Errorf("failed to write role file %s: %w", fileName, err)
				}
				cmd.Printf("Overwrote %s\n", filePath)
				result.Overwritten++
			} else {
				// Skip existing file
				cmd.Printf("Skipped %s (file already exists)\n", fileName)
				result.Skipped++
			}
		} else if os.IsNotExist(err) {
			// File doesn't exist, create it
			if err := roles.WriteRoleFile(role, filePath); err != nil {
				return fmt.Errorf("failed to write role file %s: %w", fileName, err)
			}
			cmd.Printf("Created %s\n", filePath)
			result.Created++
		} else {
			// Other error checking file
			return fmt.Errorf("failed to check file %s: %w", fileName, err)
		}
	}

	// Display completion message
	if result.Created > 0 && result.Skipped > 0 {
		cmd.Printf("Initialization completed: %d created, %d skipped\n", result.Created, result.Skipped)
	} else if result.Total > 0 {
		cmd.Println("Initialization completed successfully")
	}

	return nil
}