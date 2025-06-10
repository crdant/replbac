package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	initOutputDir string
	initForce     bool
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
		// Determine output directory
		outputDir := "."
		if len(args) > 0 {
			outputDir = args[0]
		}
		if initOutputDir != "" {
			outputDir = initOutputDir
		}
		
		fmt.Printf("Initializing role files in directory: %s\n", outputDir)
		
		if initForce {
			fmt.Println("FORCE: Existing files will be overwritten")
		}
		
		// TODO: Implement actual init logic
		fmt.Println("Init functionality will be implemented in a future step")
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	
	// Init-specific flags
	initCmd.Flags().StringVar(&initOutputDir, "output-dir", "", "directory to create role files (default: current directory)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing files")
}