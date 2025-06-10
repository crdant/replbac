package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"replbac/internal/api"
	"replbac/internal/models"
	"replbac/internal/roles"
	"replbac/internal/sync"
)

var (
	syncDryRun   bool
	syncRolesDir string
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync [directory]",
	Short: "Synchronize local role files to Replicated API",
	Long: `Sync reads role definitions from local YAML files and synchronizes them
with the Replicated platform. By default, it will process all YAML files
in the current directory recursively.

The sync operation will:
• Read all role YAML files from the specified directory
• Compare them with existing roles in the API
• Create, update, or delete roles as needed to match local state
• Provide detailed feedback on all operations performed

Use --dry-run to preview changes without applying them.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSyncCommand(cmd, args, cfg, syncDryRun, syncRolesDir)
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	
	// Sync-specific flags
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "preview changes without applying them")
	syncCmd.Flags().StringVar(&syncRolesDir, "roles-dir", "", "directory containing role YAML files (default: current directory)")
}

// RunSyncCommand implements the main sync logic with comprehensive error handling
func RunSyncCommand(cmd *cobra.Command, args []string, config models.Config, dryRun bool, rolesDir string) error {
	// Pre-flight validation
	if err := ValidateConfiguration(config); err != nil {
		return HandleConfigurationError(cmd, err)
	}

	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}
	if rolesDir != "" {
		targetDir = rolesDir
	}

	// Validate directory access
	if err := ValidateDirectoryAccess(targetDir); err != nil {
		return HandleFileSystemError(cmd, err, targetDir)
	}

	// Create API client
	client, err := api.NewClient(config.APIEndpoint, config.APIToken)
	if err != nil {
		return HandleConfigurationError(cmd, fmt.Errorf("failed to create API client: %w", err))
	}
	
	return RunSyncCommandWithClient(cmd, args, client, dryRun, rolesDir)
}

// RunSyncCommandWithClient implements the main sync logic with dependency injection
func RunSyncCommandWithClient(cmd *cobra.Command, args []string, client api.ClientInterface, dryRun bool, rolesDir string) error {
	// Determine roles directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}
	if rolesDir != "" {
		targetDir = rolesDir
	}
	
	cmd.Printf("Synchronizing roles from directory: %s\n", targetDir)
	
	if dryRun {
		cmd.Println("DRY RUN: No changes will be applied")
	}
	
	// Load local roles from directory with detailed feedback
	loadResult, err := roles.LoadRolesFromDirectoryWithDetails(targetDir)
	if err != nil {
		// Check if this is a permission error and handle it properly
		if strings.Contains(err.Error(), "permission denied") {
			permErr := &PermissionError{
				Path:     targetDir,
				Message:  "permission denied",
				Guidance: "Check directory permissions and ensure read access",
			}
			return HandleFileSystemError(cmd, permErr, targetDir)
		}
		return fmt.Errorf("failed to load local roles: %w", err)
	}
	
	// Display warnings for skipped files
	for _, skipped := range loadResult.SkippedFiles {
		cmd.Printf("Warning: Skipped %s (%s)\n", skipped.Path, skipped.Reason)
	}
	
	// Provide user guidance if files were skipped
	if len(loadResult.SkippedFiles) > 0 {
		cmd.Printf("Help: Check your YAML files for proper formatting and structure\n")
	}
	
	localRoles := loadResult.Roles
	
	// Get remote roles from API
	remoteRoles, err := client.GetRoles()
	if err != nil {
		return HandleSyncError(cmd, fmt.Errorf("failed to get remote roles: %w", err))
	}
	
	// Compare roles and generate sync plan
	plan, err := sync.CompareRoles(localRoles, remoteRoles)
	if err != nil {
		return fmt.Errorf("failed to compare roles: %w", err)
	}
	
	// Display plan summary
	if !plan.HasChanges() {
		cmd.Println("No changes needed")
		return nil
	}
	
	cmd.Printf("Sync plan: %s\n", plan.Summary())
	
	// Display detailed plan
	if len(plan.Creates) > 0 {
		cmd.Printf("Will create %d role(s):\n", len(plan.Creates))
		for _, role := range plan.Creates {
			cmd.Printf("  - %s\n", role.Name)
		}
	}
	
	if len(plan.Updates) > 0 {
		cmd.Printf("Will update %d role(s):\n", len(plan.Updates))
		for _, update := range plan.Updates {
			cmd.Printf("  - %s\n", update.Name)
		}
	}
	
	if len(plan.Deletes) > 0 {
		cmd.Printf("Will delete %d role(s):\n", len(plan.Deletes))
		for _, roleName := range plan.Deletes {
			cmd.Printf("  - %s\n", roleName)
		}
	}
	
	// Execute sync plan
	executor := sync.NewExecutor(client)
	var result sync.ExecutionResult
	
	if dryRun {
		result = executor.ExecutePlanDryRun(plan)
	} else {
		result = executor.ExecutePlan(plan)
	}
	
	// Handle execution result
	if result.Error != nil {
		syncErr := &SyncError{
			Operation: "role synchronization",
			Message:   result.Error.Error(),
			Guidance:  "Check your API credentials and network connection",
			Partial:   true, // Since execution failed, no operations completed
		}
		return HandleSyncError(cmd, syncErr)
	}
	
	// Display execution summary
	cmd.Printf("\nSync completed: %s\n", result.Summary())
	
	return nil
}