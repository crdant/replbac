package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"replbac/internal/api"
	"replbac/internal/logging"
	"replbac/internal/models"
	"replbac/internal/roles"
	"replbac/internal/sync"
)

var (
	syncDryRun   bool
	syncDiff     bool
	syncDelete   bool
	syncForce    bool
	syncNoInvite bool
	verbose      bool
	debug        bool
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
• Show clean results on stdout, with errors and progress on stderr

Logging levels (to stderr):
• Default: ERROR level only (quiet operation)
• --verbose: INFO level (progress and results)  
• --debug: DEBUG level (detailed operation info)

Use --dry-run to preview changes without applying them, or --diff 
for enhanced reporting with detailed diffs showing exactly what will change.

Environment Variables:
  This command supports all global environment variables. 
  See 'replbac --help' for full environment variable documentation.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If diff is enabled, enable dry-run too
		effectiveDryRun := syncDryRun || syncDiff
		// Auto-invite is enabled by default, disabled by --no-invite flag
		effectiveAutoInvite := !syncNoInvite
		return RunSyncCommand(cmd, args, cfg, effectiveDryRun, syncDiff, syncDelete, syncForce, effectiveAutoInvite)
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Sync-specific flags
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "preview changes without applying them")
	syncCmd.Flags().BoolVar(&syncDiff, "diff", false, "preview changes with detailed diffs (implies --dry-run)")
	syncCmd.Flags().BoolVar(&syncDelete, "delete", false, "delete remote roles not present in local files (default: false)")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "skip confirmation prompts (requires --delete)")
	syncCmd.Flags().BoolVar(&syncNoInvite, "no-invite", false, "disable automatic invitation of missing members (default: auto-invite enabled)")
	syncCmd.Flags().BoolVar(&verbose, "verbose", false, "enable info-level logging to stderr (progress and results)")
	syncCmd.Flags().BoolVar(&debug, "debug", false, "enable debug-level logging to stderr (detailed operation info)")
}

// RunSyncCommand implements the main sync logic with comprehensive error handling
func RunSyncCommand(cmd *cobra.Command, args []string, config models.Config, dryRun bool, diff bool, delete bool, force bool, autoInvite bool) error {
	// Ensure command output goes to stdout and logs go to stderr (unless already set for testing)
	if cmd.OutOrStdout() == os.Stderr {
		cmd.SetOut(os.Stdout)
	}
	if cmd.ErrOrStderr() == os.Stdout {
		cmd.SetErr(os.Stderr)
	}

	// Create logger that outputs to stderr
	verbose := false
	debug := false
	if cmd.Flags().Lookup("verbose") != nil {
		verbose, _ = cmd.Flags().GetBool("verbose")
	}
	if cmd.Flags().Lookup("debug") != nil {
		debug, _ = cmd.Flags().GetBool("debug")
	}

	var logger *logging.Logger
	if debug {
		logger = logging.NewDebugLogger(cmd.ErrOrStderr())
	} else {
		logger = logging.NewLogger(cmd.ErrOrStderr(), verbose)
	}

	// Pre-flight validation with logging
	logger.Debug("validating configuration")
	if err := ValidateConfiguration(config); err != nil {
		logger.Error("configuration validation failed: %v", err)
		return HandleConfigurationError(cmd, err)
	}

	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	logger.Debug("validating directory access: %s", targetDir)
	// Validate directory access
	if err := ValidateDirectoryAccess(targetDir); err != nil {
		logger.Error("directory access validation failed: %v", err)
		return HandleFileSystemError(cmd, err, targetDir)
	}

	// Create API client
	logger.Debug("creating API client")
	client, err := api.NewClient(models.ReplicatedAPIEndpoint, config.APIToken, logger)
	if err != nil {
		logger.Error("failed to create API client: %v", err)
		return HandleConfigurationError(cmd, fmt.Errorf("failed to create API client: %w", err))
	}

	// Use the enhanced logging version
	return RunSyncCommandWithLogging(cmd, args, client, dryRun, diff, delete, force, autoInvite, logger, config)
}

// RunSyncCommandWithLogging implements sync with enhanced logging and user feedback
func RunSyncCommandWithLogging(cmd *cobra.Command, args []string, client api.ClientInterface, dryRun bool, diff bool, delete bool, force bool, autoInvite bool, logger *logging.Logger, config models.Config) error {
	// Determine roles directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	cmd.Printf("Synchronizing roles from directory: %s\n", targetDir)
	logger.Debug("sync operation starting: target directory: %s, dry-run: %v", targetDir, dryRun)

	if dryRun {
		cmd.Println("DRY RUN: No changes will be applied")
		logger.Debug("running in dry-run mode")
	}

	// Load local roles
	logger.Debug("loading roles from directory: %s", targetDir)

	loadResult, err := roles.LoadRolesFromDirectoryWithDetails(targetDir)
	if err != nil {
		logger.Error("failed to load roles from directory: %v", err)
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

	logger.Debug("loaded %d roles from directory", len(loadResult.Roles))
	if len(loadResult.SkippedFiles) > 0 {
		logger.Warn("skipped %d invalid files", len(loadResult.SkippedFiles))
	}

	// Display warnings for skipped files
	for _, skipped := range loadResult.SkippedFiles {
		cmd.Printf("Warning: Skipped %s (%s)\n", skipped.Path, skipped.Reason)
		logger.Debug("skipped file: %s (reason: %s)", skipped.Path, skipped.Reason)
	}

	if len(loadResult.SkippedFiles) > 0 {
		cmd.Printf("Help: Check your YAML files for proper formatting and structure\n")
	}

	localRoles := loadResult.Roles

	// Get remote roles with progress feedback
	if len(localRoles) > 0 {
		logger.Debug("synchronizing with remote API")
	}
	logger.Debug("fetching remote roles from API")

	remoteRoles, err := client.GetRoles()
	if err != nil {
		logger.Error("failed to fetch remote roles: %v", err)
		return HandleSyncError(cmd, fmt.Errorf("failed to get remote roles: %w", err))
	}

	logger.Debug("fetched %d remote roles", len(remoteRoles))
	logger.Debug("comparing roles")

	// Compare roles and generate sync plan
	plan, err := sync.CompareRoles(localRoles, remoteRoles)
	if err != nil {
		logger.Error("failed to compare roles: %v", err)
		return fmt.Errorf("failed to compare roles: %w", err)
	}

	// Remove deletions from plan if delete flag is not set
	if !delete && len(plan.Deletes) > 0 {
		logger.Debug("removing %d deletions from plan because --delete flag not set", len(plan.Deletes))
		plan.Deletes = []string{} // Clear deletions
	}

	logger.Debug("plan generated: %d creates, %d updates, %d deletes", len(plan.Creates), len(plan.Updates), len(plan.Deletes))

	// Display plan summary
	if !plan.HasChanges() {
		cmd.Println("No changes needed")
		logger.Debug("no changes needed - plan has no changes")
		return nil
	}

	cmd.Printf("Sync plan: %s\n", plan.Summary())
	logger.Debug("sync plan: %s", plan.Summary())

	// Display detailed plan
	if len(plan.Creates) > 0 {
		cmd.Printf("Will create %d role(s):\n", len(plan.Creates))
		for _, role := range plan.Creates {
			cmd.Printf("  - %s\n", role.Name)
			logger.Debug("will create role: %s", role.Name)
		}
	}

	if len(plan.Updates) > 0 {
		cmd.Printf("Will update %d role(s):\n", len(plan.Updates))
		for _, update := range plan.Updates {
			cmd.Printf("  - %s\n", update.Name)
			logger.Debug("will update role: %s", update.Name)
		}
	}

	if len(plan.Deletes) > 0 {
		cmd.Printf("Will delete %d role(s):\n", len(plan.Deletes))
		for _, roleName := range plan.Deletes {
			cmd.Printf("  - %s\n", roleName)
			logger.Debug("will delete role: %s", roleName)
		}
	}

	// Ask for confirmation if deletions are planned and not in dry-run mode and not forced
	if len(plan.Deletes) > 0 && !dryRun && !config.Confirm && !force {
		cmd.Printf("\nThis operation will permanently delete %d role(s) from the API.\n", len(plan.Deletes))
		cmd.Print("Do you want to continue? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			cmd.Println("Operation cancelled by user")
			logger.Debug("sync operation cancelled by user")
			return nil
		}
		logger.Debug("user confirmed deletion operation")
	}

	// Execute sync plan with timing
	var result sync.ExecutionResult

	err = logger.TimedOperation("sync execution", func() error {
		// Check if any roles have members to determine which executor to use
		hasMembers := rolesHaveMembers(localRoles)

		if hasMembers {
			logger.Debug("roles contain members - using ExecutorWithMembers (auto-invite: %v)", autoInvite)
			executor := sync.NewExecutorWithMembersAndInvite(client.(sync.APIClientWithMembers), logger, autoInvite)
			if dryRun {
				if diff {
					result = executor.ExecutePlanDryRunWithDiffs(plan)
				} else {
					result = executor.ExecutePlanDryRun(plan)
				}
			} else {
				result = executor.ExecutePlanWithLocalRoles(plan, localRoles)
			}
		} else {
			logger.Debug("roles contain no members - using standard Executor")
			executor := sync.NewExecutor(client, logger)
			if dryRun {
				if diff {
					result = executor.ExecutePlanDryRunWithDiffs(plan)
				} else {
					result = executor.ExecutePlanDryRun(plan)
				}
			} else {
				result = executor.ExecutePlan(plan)
			}
		}
		return result.Error
	})

	if err != nil {
		syncErr := &SyncError{
			Operation: "role synchronization",
			Message:   err.Error(),
			Guidance:  "Check your API credentials and network connection",
			Partial:   true,
		}
		return HandleSyncError(cmd, syncErr)
	}

	// Handle member deletions if needed
	if !dryRun && result.MemberDeletions != nil && (len(result.MemberDeletions.OrphanedUsers) > 0 || len(result.MemberDeletions.OrphanedInvites) > 0) {
		if err := confirmAndDeleteMembers(cmd, client, result.MemberDeletions, force, logger); err != nil {
			return fmt.Errorf("failed to handle member deletions: %w", err)
		}
	}

	// Display execution summary
	if diff && result.DetailedInfo != "" {
		cmd.Printf("\nSync completed: %s\n", result.DetailedSummary())
	} else {
		cmd.Printf("\nSync completed: %s\n", result.Summary())
	}
	logger.Debug("sync operation completed successfully")

	return nil
}

// RunSyncCommandWithClient implements the main sync logic with dependency injection
func RunSyncCommandWithClient(cmd *cobra.Command, args []string, client api.ClientInterface, dryRun bool, delete bool, force bool) error {
	// For backward compatibility, default autoInvite to true
	return RunSyncCommandWithClientAndInvite(cmd, args, client, dryRun, delete, force, true)
}

// RunSyncCommandWithClientAndInvite implements the main sync logic with dependency injection and invite control
func RunSyncCommandWithClientAndInvite(cmd *cobra.Command, args []string, client api.ClientInterface, dryRun bool, delete bool, force bool, autoInvite bool) error {
	// Ensure command output goes to stdout and logs go to stderr (unless already set for testing)
	if cmd.OutOrStdout() == os.Stderr {
		cmd.SetOut(os.Stdout)
	}
	if cmd.ErrOrStderr() == os.Stdout {
		cmd.SetErr(os.Stderr)
	}

	// Create a logger that outputs to stderr
	verbose := false
	debug := false
	if cmd.Flags().Lookup("verbose") != nil {
		verbose, _ = cmd.Flags().GetBool("verbose")
	}
	if cmd.Flags().Lookup("debug") != nil {
		debug, _ = cmd.Flags().GetBool("debug")
	}

	var logger *logging.Logger
	if debug {
		logger = logging.NewDebugLogger(cmd.ErrOrStderr())
	} else {
		logger = logging.NewLogger(cmd.ErrOrStderr(), verbose)
	}
	// Determine roles directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
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

	// Remove deletions from plan if delete flag is not set
	if !delete && len(plan.Deletes) > 0 {
		plan.Deletes = []string{} // Clear deletions
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
	var result sync.ExecutionResult

	// Check if any roles have members to determine which executor to use
	hasMembers := rolesHaveMembers(localRoles)

	if hasMembers {
		logger.Debug("roles contain members - using ExecutorWithMembers (auto-invite: %v)", autoInvite)
		executor := sync.NewExecutorWithMembersAndInvite(client.(sync.APIClientWithMembers), logger, autoInvite)
		if dryRun {
			result = executor.ExecutePlanDryRun(plan)
		} else {
			result = executor.ExecutePlanWithLocalRoles(plan, localRoles)
		}
	} else {
		logger.Debug("roles contain no members - using standard Executor")
		executor := sync.NewExecutor(client, logger)
		if dryRun {
			result = executor.ExecutePlanDryRun(plan)
		} else {
			result = executor.ExecutePlan(plan)
		}
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

	// Handle member deletions if needed
	if !dryRun && result.MemberDeletions != nil && (len(result.MemberDeletions.OrphanedUsers) > 0 || len(result.MemberDeletions.OrphanedInvites) > 0) {
		if err := confirmAndDeleteMembers(cmd, client, result.MemberDeletions, force, logger); err != nil {
			return fmt.Errorf("failed to handle member deletions: %w", err)
		}
	}

	// Display execution summary
	cmd.Printf("\nSync completed: %s\n", result.Summary())

	return nil
}

// rolesHaveMembers checks if any of the provided roles have member assignments
func rolesHaveMembers(roles []models.Role) bool {
	for _, role := range roles {
		if len(role.Members) > 0 {
			return true
		}
	}
	return false
}

// confirmAndDeleteMembers prompts for confirmation and deletes orphaned members/invites
func confirmAndDeleteMembers(cmd *cobra.Command, client api.ClientInterface, deletions *sync.MemberDeletions, force bool, logger *logging.Logger) error {
	totalDeletions := len(deletions.OrphanedUsers) + len(deletions.OrphanedInvites)

	// Show what will be deleted
	if len(deletions.OrphanedUsers) > 0 {
		cmd.Printf("\nThis operation will permanently remove %d team member(s) from the API:\n", len(deletions.OrphanedUsers))
		for _, email := range deletions.OrphanedUsers {
			cmd.Printf("  - %s\n", email)
		}
	}

	if len(deletions.OrphanedInvites) > 0 {
		cmd.Printf("\nThis operation will cancel %d pending invitation(s):\n", len(deletions.OrphanedInvites))
		for _, email := range deletions.OrphanedInvites {
			cmd.Printf("  - %s\n", email)
		}
	}

	// Ask for confirmation if not forced
	if !force {
		cmd.Printf("\nDo you want to continue with these %d deletion(s)? (y/N): ", totalDeletions)

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			cmd.Println("Member deletion cancelled by user")
			logger.Debug("member deletion operation cancelled by user")
			return nil
		}
		logger.Debug("user confirmed member deletion operation")
	}

	// Perform the deletions
	memberClient, ok := client.(sync.APIClientWithMembers)
	if !ok {
		return fmt.Errorf("client does not support member operations")
	}

	executor := sync.NewExecutorWithMembersAndInvite(memberClient, logger, true)
	if err := executor.DeleteMembersAndInvites(deletions); err != nil {
		return fmt.Errorf("failed to delete members and invites: %w", err)
	}

	cmd.Printf("Successfully removed %d member(s) and cancelled %d invitation(s)\n",
		len(deletions.OrphanedUsers), len(deletions.OrphanedInvites))

	return nil
}
