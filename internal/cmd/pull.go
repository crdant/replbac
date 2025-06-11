package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"replbac/internal/api"
	"replbac/internal/logging"
	"replbac/internal/models"
	"replbac/internal/roles"
)

var (
	pullForce    bool
	pullDryRun   bool
	pullDiff     bool
	pullVerbose  bool
	pullDebug    bool
)

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull [directory]",
	Short: "Pull role definitions from Replicated API to local files",
	Long: `Pull downloads existing role definitions from the Replicated platform
and creates local YAML files. This is useful for getting started with
role management or for migrating existing roles to code.

The pull operation will:
• Fetch all existing roles from the Replicated API
• Generate YAML files for each role in the specified directory
• Preserve existing files unless --force is used
• Create directory structure as needed

Use --dry-run to preview what files would be created or updated.
Use --diff to see detailed differences when files would be changed.
Use --force to overwrite existing files.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If diff is enabled, enable dry-run too
		effectiveDryRun := pullDryRun || pullDiff
		return RunPullCommand(cmd, args, cfg, effectiveDryRun, pullDiff, pullForce)
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
	
	// Pull-specific flags
	pullCmd.Flags().BoolVar(&pullForce, "force", false, "overwrite existing files")
	pullCmd.Flags().BoolVar(&pullDryRun, "dry-run", false, "preview changes without applying them")
	pullCmd.Flags().BoolVar(&pullDiff, "diff", false, "preview changes with detailed diffs (implies --dry-run)")
	pullCmd.Flags().BoolVar(&pullVerbose, "verbose", false, "enable info-level logging to stderr (progress and results)")
	pullCmd.Flags().BoolVar(&pullDebug, "debug", false, "enable debug-level logging to stderr (detailed operation info)")
}

// PullResult contains the results of pull operation
type PullResult struct {
	Created     int
	Skipped     int
	Overwritten int
	WouldCreate int
	WouldUpdate int
	Total       int
	DryRun      bool
}

// RunPullCommand implements the main pull logic with comprehensive error handling
func RunPullCommand(cmd *cobra.Command, args []string, config models.Config, dryRun, diff bool, force bool) error {
	// Ensure command output goes to stdout and logs go to stderr (unless already set for testing)
	if cmd.OutOrStdout() == os.Stderr {
		cmd.SetOut(os.Stdout)
	}
	if cmd.ErrOrStderr() == os.Stdout {
		cmd.SetErr(os.Stderr)
	}
	
	// Create logger that outputs to stderr
	var logger *logging.Logger
	if pullDebug {
		logger = logging.NewDebugLogger(cmd.ErrOrStderr())
	} else {
		logger = logging.NewLogger(cmd.ErrOrStderr(), pullVerbose)
	}
	
	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	if dryRun {
		if diff {
			cmd.Printf("DRY-RUN: Showing what would be done with detailed diffs\n")
		} else {
			cmd.Printf("DRY-RUN: Showing what would be done\n")
		}
	}

	cmd.Printf("Pulling role files in directory: %s\n", targetDir)
	logger.Debug("starting pull operation in directory: %s", targetDir)
	
	if force && !dryRun {
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
	
	return RunPullCommandWithClient(cmd, targetDir, dryRun, diff, force, client)
}

// RunPullCommandWithClient implements pull with dependency injection for testing
func RunPullCommandWithClient(cmd *cobra.Command, outputDir string, dryRun, diff, force bool, client api.ClientInterface) error {
	// Fetch roles from API
	apiRoles, err := client.GetRoles()
	if err != nil {
		cmd.Printf("Failed to fetch roles from API: %v\n", err)
		return fmt.Errorf("failed to fetch roles from API: %w", err)
	}

	if len(apiRoles) == 0 {
		cmd.Println("No roles found in API")
		cmd.Println("Pull completed: no files created")
		return nil
	}

	cmd.Printf("Downloaded %d role(s) from API\n", len(apiRoles))
	
	// Initialize result tracking
	result := PullResult{Total: len(apiRoles), DryRun: dryRun}

	// Create output directory if it doesn't exist (unless dry-run)
	if !dryRun {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Process role files
	for _, role := range apiRoles {
		fileName := fmt.Sprintf("%s.yaml", role.Name)
		filePath := filepath.Join(outputDir, fileName)

		// Check if file exists
		existingContent := ""
		if _, err := os.Stat(filePath); err == nil {
			// File exists
			if existingBytes, readErr := os.ReadFile(filePath); readErr == nil {
				existingContent = string(existingBytes)
			}
			
			if force || dryRun {
				// Generate new content
				newContent, err := roles.GenerateRoleYAML(role)
				if err != nil {
					return fmt.Errorf("failed to generate YAML for role %s: %w", role.Name, err)
				}
				
				if dryRun {
					if existingContent != newContent {
						if diff {
							cmd.Printf("Would update %s\n", filePath)
							showDiff(cmd, existingContent, newContent)
						} else {
							cmd.Printf("Would update %s\n", filePath)
						}
						result.WouldUpdate++
					} else {
						cmd.Printf("Would skip %s (no changes)\n", fileName)
						result.Skipped++
					}
				} else {
					// Actually update the file
					if err := roles.WriteRoleFile(role, filePath); err != nil {
						return fmt.Errorf("failed to write role file %s: %w", fileName, err)
					}
					cmd.Printf("Overwrote %s\n", filePath)
					result.Overwritten++
				}
			} else {
				// Skip existing file
				cmd.Printf("Skipped %s (file already exists)\n", fileName)
				result.Skipped++
			}
		} else if os.IsNotExist(err) {
			// File doesn't exist
			if dryRun {
				cmd.Printf("Would create %s\n", filePath)
				result.WouldCreate++
			} else {
				// Create it
				if err := roles.WriteRoleFile(role, filePath); err != nil {
					return fmt.Errorf("failed to write role file %s: %w", fileName, err)
				}
				cmd.Printf("Created %s\n", filePath)
				result.Created++
			}
		} else {
			// Other error checking file
			return fmt.Errorf("failed to check file %s: %w", fileName, err)
		}
	}

	// Display completion message
	if dryRun {
		if result.WouldCreate > 0 && result.WouldUpdate > 0 {
			cmd.Printf("Pull completed (dry-run): %d would be created, %d would be updated\n", result.WouldCreate, result.WouldUpdate)
		} else if result.WouldCreate > 0 {
			cmd.Printf("Pull completed (dry-run): %d would be created\n", result.WouldCreate)
		} else if result.WouldUpdate > 0 {
			cmd.Printf("Pull completed (dry-run): %d would be updated\n", result.WouldUpdate)
		} else {
			cmd.Printf("Pull completed (dry-run): no changes needed\n")
		}
	} else {
		if result.Created > 0 && result.Skipped > 0 {
			cmd.Printf("Pull completed: %d created, %d skipped\n", result.Created, result.Skipped)
		} else if result.Total > 0 {
			cmd.Println("Pull completed successfully")
		}
	}

	return nil
}

// showDiff displays a simple diff between old and new content
func showDiff(cmd *cobra.Command, oldContent, newContent string) {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")
	
	// Simple line-by-line diff
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}
	
	for i := 0; i < maxLines; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		
		if oldLine != newLine {
			if oldLine != "" {
				cmd.Printf("- %s\n", oldLine)
			}
			if newLine != "" {
				cmd.Printf("+ %s\n", newLine)
			}
		}
	}
}