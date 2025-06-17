package sync

import (
	"fmt"
	"sort"
	"strings"

	"replbac/internal/logging"
	"replbac/internal/models"
)

// APIClient defines the interface for API operations needed by the executor
type APIClient interface {
	CreateRole(role models.Role) error
	UpdateRole(role models.Role) error
	DeleteRole(roleName string) error
	GetRole(roleName string) (models.Role, error)
}

// APIClientWithMembers extends APIClient with member management operations
type APIClientWithMembers interface {
	APIClient
	GetTeamMembers() ([]models.TeamMember, error)
	AssignMemberRole(memberEmail, roleID string) error
	InviteUser(email, policyID string) (*models.InviteUserResponse, error)
	DeleteInvite(email string) error
}

// Executor handles the execution of sync plans
type Executor struct {
	client APIClient
	logger *logging.Logger
}

// ExecutorWithMembers handles the execution of sync plans including member assignments
type ExecutorWithMembers struct {
	client     APIClientWithMembers
	logger     *logging.Logger
	autoInvite bool
}

// ExecutionResult represents the result of executing a sync plan
type ExecutionResult struct {
	Created         int              // Number of roles created
	Updated         int              // Number of roles updated
	Deleted         int              // Number of roles deleted
	Error           error            // Error if execution failed
	DryRun          bool             // Whether this was a dry run
	DetailedInfo    string           // Detailed information about changes (for enhanced dry-run)
	MemberDeletions *MemberDeletions // Members and invites that would be deleted
}

// MemberDeletions represents members and invites that need to be deleted
type MemberDeletions struct {
	OrphanedUsers   []string // Users to be removed from team
	OrphanedInvites []string // Invites to be cancelled
}

// NewExecutor creates a new sync executor with the given API client
func NewExecutor(client APIClient, logger *logging.Logger) *Executor {
	return &Executor{
		client: client,
		logger: logger,
	}
}

// NewExecutorWithMembers creates a new sync executor with member management support
func NewExecutorWithMembers(client APIClientWithMembers, logger *logging.Logger) *ExecutorWithMembers {
	return &ExecutorWithMembers{
		client:     client,
		logger:     logger,
		autoInvite: true, // Default to auto-invite for backward compatibility
	}
}

// NewExecutorWithMembersAndInvite creates a new sync executor with configurable invite behavior
func NewExecutorWithMembersAndInvite(client APIClientWithMembers, logger *logging.Logger, autoInvite bool) *ExecutorWithMembers {
	return &ExecutorWithMembers{
		client:     client,
		logger:     logger,
		autoInvite: autoInvite,
	}
}

// ExecutePlan executes a sync plan by making actual API calls
func (e *Executor) ExecutePlan(plan SyncPlan) ExecutionResult {
	e.logger.Info("executing sync plan: %d creates, %d updates, %d deletes", len(plan.Creates), len(plan.Updates), len(plan.Deletes))
	result := ExecutionResult{
		DryRun: false,
	}

	// Execute creates
	for _, role := range plan.Creates {
		e.logger.Debug("creating role: %s", role.Name)
		if err := e.client.CreateRole(role); err != nil {
			e.logger.Error("failed to create role %s: %v", role.Name, err)
			result.Error = fmt.Errorf("failed to create role '%s': %w", role.Name, err)
			return result
		}
		e.logger.Info("successfully created role: %s", role.Name)
		result.Created++
	}

	// Execute updates
	for _, update := range plan.Updates {
		e.logger.Debug("updating role: %s", update.Name)
		if err := e.client.UpdateRole(update.Local); err != nil {
			e.logger.Error("failed to update role %s: %v", update.Name, err)
			result.Error = fmt.Errorf("failed to update role '%s': %w", update.Name, err)
			return result
		}
		e.logger.Info("successfully updated role: %s", update.Name)
		result.Updated++
	}

	// Execute deletes
	for _, roleName := range plan.Deletes {
		e.logger.Debug("deleting role: %s", roleName)
		if err := e.client.DeleteRole(roleName); err != nil {
			e.logger.Error("failed to delete role %s: %v", roleName, err)
			result.Error = fmt.Errorf("failed to delete role '%s': %w", roleName, err)
			return result
		}
		e.logger.Info("successfully deleted role: %s", roleName)
		result.Deleted++
	}

	e.logger.Info("sync plan execution completed successfully")
	return result
}

// ExecutePlanDryRun simulates executing a sync plan without making actual API calls
func (e *Executor) ExecutePlanDryRun(plan SyncPlan) ExecutionResult {
	e.logger.Info("executing sync plan in dry-run mode: %d creates, %d updates, %d deletes", len(plan.Creates), len(plan.Updates), len(plan.Deletes))

	result := ExecutionResult{
		Created: len(plan.Creates),
		Updated: len(plan.Updates),
		Deleted: len(plan.Deletes),
		DryRun:  true,
		Error:   nil,
	}

	e.logger.Debug("dry-run completed - no actual changes made")
	return result
}

// ExecutePlanDryRunWithDiffs simulates executing a sync plan with detailed diff information
func (e *Executor) ExecutePlanDryRunWithDiffs(plan SyncPlan) ExecutionResult {
	result := ExecutionResult{
		Created: len(plan.Creates),
		Updated: len(plan.Updates),
		Deleted: len(plan.Deletes),
		DryRun:  true,
		Error:   nil,
	}

	// Generate detailed diff information
	detailsBuilder := make([]string, 0)

	// Add create details
	for _, role := range plan.Creates {
		detailsBuilder = append(detailsBuilder,
			fmt.Sprintf("CREATE: %s (allowed: %v, denied: %v)",
				role.Name, role.Resources.Allowed, role.Resources.Denied))
	}

	// Add update details with diffs
	for _, update := range plan.Updates {
		detailsBuilder = append(detailsBuilder, fmt.Sprintf("UPDATE: %s", update.Name))

		// Compare allowed resources
		allowedDiff := generateResourceDiff("allowed", update.Remote.Resources.Allowed, update.Local.Resources.Allowed)
		if allowedDiff != "" {
			detailsBuilder = append(detailsBuilder, fmt.Sprintf("  %s", allowedDiff))
		}

		// Compare denied resources
		deniedDiff := generateResourceDiff("denied", update.Remote.Resources.Denied, update.Local.Resources.Denied)
		if deniedDiff != "" {
			detailsBuilder = append(detailsBuilder, fmt.Sprintf("  %s", deniedDiff))
		}
	}

	// Add delete details
	for _, roleName := range plan.Deletes {
		detailsBuilder = append(detailsBuilder, fmt.Sprintf("DELETE: %s", roleName))
	}

	result.DetailedInfo = strings.Join(detailsBuilder, "\n")
	return result
}

// Summary returns a human-readable summary of the execution result
func (r ExecutionResult) Summary() string {
	if r.Error != nil {
		return fmt.Sprintf("Execution failed: %v", r.Error)
	}

	if r.Created == 0 && r.Updated == 0 && r.Deleted == 0 {
		if r.DryRun {
			return "Dry run: No changes would be made"
		}
		return "No changes made"
	}

	summary := ""
	if r.DryRun {
		summary = "Dry run: Would "
	} else {
		summary = ""
	}

	actions := []string{}
	if r.Created > 0 {
		actions = append(actions, fmt.Sprintf("create %d role(s)", r.Created))
	}
	if r.Updated > 0 {
		actions = append(actions, fmt.Sprintf("update %d role(s)", r.Updated))
	}
	if r.Deleted > 0 {
		actions = append(actions, fmt.Sprintf("delete %d role(s)", r.Deleted))
	}

	if len(actions) == 1 {
		summary += actions[0]
	} else if len(actions) == 2 {
		summary += actions[0] + " and " + actions[1]
	} else if len(actions) == 3 {
		summary += actions[0] + ", " + actions[1] + ", and " + actions[2]
	}

	return summary
}

// HasChanges returns true if the execution result indicates any changes were made or would be made
func (r ExecutionResult) HasChanges() bool {
	return r.Created > 0 || r.Updated > 0 || r.Deleted > 0
}

// IsSuccess returns true if the execution completed without error
func (r ExecutionResult) IsSuccess() bool {
	return r.Error == nil
}

// DetailedSummary returns a human-readable summary with detailed information when available
func (r ExecutionResult) DetailedSummary() string {
	// Start with basic summary
	summary := r.Summary()

	// If we have detailed information, append it
	if r.DetailedInfo != "" {
		summary += "\n\nDetails:\n" + r.DetailedInfo
	}

	return summary
}

// generateResourceDiff generates a diff string showing changes between old and new resource lists
func generateResourceDiff(resourceType string, oldResources, newResources []string) string {
	// Normalize slices (handle nil as empty)
	if oldResources == nil {
		oldResources = []string{}
	}
	if newResources == nil {
		newResources = []string{}
	}

	// Create maps for efficient lookup
	oldMap := make(map[string]bool)
	newMap := make(map[string]bool)

	for _, resource := range oldResources {
		oldMap[resource] = true
	}
	for _, resource := range newResources {
		newMap[resource] = true
	}

	// Find additions and removals
	var additions, removals []string

	// Check for additions (in new but not in old)
	for resource := range newMap {
		if !oldMap[resource] {
			additions = append(additions, resource)
		}
	}

	// Check for removals (in old but not in new)
	for resource := range oldMap {
		if !newMap[resource] {
			removals = append(removals, resource)
		}
	}

	// Sort for consistent output
	sort.Strings(additions)
	sort.Strings(removals)

	// Build diff string
	var diffParts []string

	if len(additions) > 0 {
		for _, addition := range additions {
			diffParts = append(diffParts, fmt.Sprintf("+ %s: %s", resourceType, addition))
		}
	}

	if len(removals) > 0 {
		for _, removal := range removals {
			diffParts = append(diffParts, fmt.Sprintf("- %s: %s", resourceType, removal))
		}
	}

	return strings.Join(diffParts, "\n")
}

// ExecutePlan executes a sync plan by making actual API calls including member assignments
func (e *ExecutorWithMembers) ExecutePlan(plan SyncPlan) ExecutionResult {
	e.logger.Info("executing sync plan with member support: %d creates, %d updates, %d deletes", len(plan.Creates), len(plan.Updates), len(plan.Deletes))
	result := ExecutionResult{
		DryRun: false,
	}

	// Execute creates
	for _, role := range plan.Creates {
		e.logger.Debug("creating role: %s", role.Name)
		if err := e.client.CreateRole(role); err != nil {
			e.logger.Error("failed to create role %s: %v", role.Name, err)
			result.Error = fmt.Errorf("failed to create role '%s': %w", role.Name, err)
			return result
		}
		e.logger.Info("successfully created role: %s", role.Name)
		result.Created++
	}

	// Execute updates
	for _, update := range plan.Updates {
		e.logger.Debug("updating role: %s", update.Name)
		if err := e.client.UpdateRole(update.Local); err != nil {
			e.logger.Error("failed to update role %s: %v", update.Name, err)
			result.Error = fmt.Errorf("failed to update role '%s': %w", update.Name, err)
			return result
		}
		e.logger.Info("successfully updated role: %s", update.Name)
		result.Updated++
	}

	// Execute deletes
	for _, roleName := range plan.Deletes {
		e.logger.Debug("deleting role: %s", roleName)
		if err := e.client.DeleteRole(roleName); err != nil {
			e.logger.Error("failed to delete role %s: %v", roleName, err)
			result.Error = fmt.Errorf("failed to delete role '%s': %w", roleName, err)
			return result
		}
		e.logger.Info("successfully deleted role: %s", roleName)
		result.Deleted++
	}

	// After all role operations are complete, sync members
	// Note: This method only syncs members for creates/updates, not all local roles
	// Use ExecutePlanWithLocalRoles for complete member sync
	memberDeletions, err := e.syncAllMembersFromPlan(plan)
	if err != nil {
		e.logger.Error("failed to sync members: %v", err)
		result.Error = fmt.Errorf("failed to sync members: %w", err)
		return result
	}
	result.MemberDeletions = memberDeletions

	e.logger.Info("sync plan execution completed successfully")
	return result
}

// ExecutePlanWithLocalRoles executes a sync plan and performs comprehensive member sync using all local roles
func (e *ExecutorWithMembers) ExecutePlanWithLocalRoles(plan SyncPlan, allLocalRoles []models.Role) ExecutionResult {
	e.logger.Info("executing sync plan with comprehensive member sync: %d creates, %d updates, %d deletes", len(plan.Creates), len(plan.Updates), len(plan.Deletes))
	result := ExecutionResult{
		DryRun: false,
	}

	// Execute creates
	for _, role := range plan.Creates {
		e.logger.Debug("creating role: %s", role.Name)
		if err := e.client.CreateRole(role); err != nil {
			e.logger.Error("failed to create role %s: %v", role.Name, err)
			result.Error = fmt.Errorf("failed to create role '%s': %w", role.Name, err)
			return result
		}
		e.logger.Info("successfully created role: %s", role.Name)
		result.Created++
	}

	// Execute updates
	for _, update := range plan.Updates {
		e.logger.Debug("updating role: %s", update.Name)
		if err := e.client.UpdateRole(update.Local); err != nil {
			e.logger.Error("failed to update role %s: %v", update.Name, err)
			result.Error = fmt.Errorf("failed to update role '%s': %w", update.Name, err)
			return result
		}
		e.logger.Info("successfully updated role: %s", update.Name)
		result.Updated++
	}

	// Execute deletes
	for _, roleName := range plan.Deletes {
		e.logger.Debug("deleting role: %s", roleName)
		if err := e.client.DeleteRole(roleName); err != nil {
			e.logger.Error("failed to delete role %s: %v", roleName, err)
			result.Error = fmt.Errorf("failed to delete role '%s': %w", roleName, err)
			return result
		}
		e.logger.Info("successfully deleted role: %s", roleName)
		result.Deleted++
	}

	// After all role operations are complete, sync members using ALL local roles
	memberDeletions, err := e.syncAllMembers(allLocalRoles)
	if err != nil {
		e.logger.Error("failed to sync members: %v", err)
		result.Error = fmt.Errorf("failed to sync members: %w", err)
		return result
	}
	result.MemberDeletions = memberDeletions

	e.logger.Info("sync plan execution completed successfully")
	return result
}

// syncAllMembersFromPlan performs member synchronization based only on plan operations (creates/updates)
func (e *ExecutorWithMembers) syncAllMembersFromPlan(plan SyncPlan) (*MemberDeletions, error) {
	e.logger.Info("synchronizing team members from plan operations only")

	// Collect members from created and updated roles only
	localMembers := make(map[string]string) // email -> roleName

	// Add members from created roles
	for _, role := range plan.Creates {
		for _, memberEmail := range role.Members {
			if existingRole, exists := localMembers[memberEmail]; exists {
				return nil, fmt.Errorf("member %s appears in multiple roles: %s and %s", memberEmail, existingRole, role.Name)
			}
			localMembers[memberEmail] = role.Name
		}
	}

	// Add members from updated roles (use local version)
	for _, update := range plan.Updates {
		for _, memberEmail := range update.Local.Members {
			if existingRole, exists := localMembers[memberEmail]; exists {
				return nil, fmt.Errorf("member %s appears in multiple roles: %s and %s", memberEmail, existingRole, update.Name)
			}
			localMembers[memberEmail] = update.Name
		}
	}

	// Get current team members
	teamMembers, err := e.client.GetTeamMembers()
	if err != nil {
		e.logger.Error("failed to get team members: %v", err)
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	// Create maps for existing team members
	existingMembers := make(map[string]models.TeamMember)
	for _, member := range teamMembers {
		existingMembers[member.Email] = member
	}

	// Process member assignments
	if err := e.processMemberAssignments(localMembers, existingMembers); err != nil {
		return nil, fmt.Errorf("failed to process member assignments: %w", err)
	}

	// Identify orphaned members and invites (but don't delete them yet)
	memberDeletions := e.identifyOrphanedMembers(localMembers, existingMembers)

	return memberDeletions, nil
}

// syncAllMembers performs comprehensive member synchronization across all roles
func (e *ExecutorWithMembers) syncAllMembers(allLocalRoles []models.Role) (*MemberDeletions, error) {
	e.logger.Info("synchronizing team members across all local roles")

	// Collect all members from ALL local role definitions
	localMembers := make(map[string]string) // email -> roleName

	// Add members from ALL local roles
	for _, role := range allLocalRoles {
		for _, memberEmail := range role.Members {
			if existingRole, exists := localMembers[memberEmail]; exists {
				return nil, fmt.Errorf("member %s appears in multiple roles: %s and %s", memberEmail, existingRole, role.Name)
			}
			localMembers[memberEmail] = role.Name
		}
	}

	// Get current team members
	teamMembers, err := e.client.GetTeamMembers()
	if err != nil {
		e.logger.Error("failed to get team members: %v", err)
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	// Create maps for existing team members
	existingMembers := make(map[string]models.TeamMember)
	for _, member := range teamMembers {
		existingMembers[member.Email] = member
	}

	// Process member assignments
	if err := e.processMemberAssignments(localMembers, existingMembers); err != nil {
		return nil, fmt.Errorf("failed to process member assignments: %w", err)
	}

	// Identify orphaned members and invites (but don't delete them yet)
	memberDeletions := e.identifyOrphanedMembers(localMembers, existingMembers)

	return memberDeletions, nil
}

// processMemberAssignments handles assigning members to roles (invite if needed, assign if exists)
func (e *ExecutorWithMembers) processMemberAssignments(localMembers map[string]string, existingMembers map[string]models.TeamMember) error {
	for memberEmail, roleName := range localMembers {
		e.logger.Debug("processing member assignment: %s -> %s", memberEmail, roleName)

		// Get role ID
		role, err := e.client.GetRole(roleName)
		if err != nil {
			return fmt.Errorf("failed to get role %s for member %s: %w", roleName, memberEmail, err)
		}
		roleID := role.ID

		existingMember, memberExists := existingMembers[memberEmail]

		if memberExists {
			// Member exists - check if they're already assigned to the correct role
			if existingMember.PolicyID == roleID {
				e.logger.Debug("member %s already assigned to role %s (ID: %s), skipping", memberEmail, roleName, roleID)
			} else {
				// Member exists but assigned to different role - reassign them
				e.logger.Debug("reassigning member %s from policy %s to role %s (ID: %s)", memberEmail, existingMember.PolicyID, roleName, roleID)
				if err := e.client.AssignMemberRole(memberEmail, roleID); err != nil {
					return fmt.Errorf("failed to assign member %s to role %s: %w", memberEmail, roleName, err)
				}
				e.logger.Info("successfully assigned member %s to role %s", memberEmail, roleName)
			}
		} else if e.autoInvite {
			// Member doesn't exist - invite them
			e.logger.Debug("member %s not found in team, sending invite for role %s", memberEmail, roleName)
			response, err := e.client.InviteUser(memberEmail, roleID)
			if err != nil {
				return fmt.Errorf("failed to invite member %s to role %s: %w", memberEmail, roleName, err)
			}
			e.logger.Info("successfully invited member %s to role %s (status: %s)", memberEmail, roleName, response.Status)
		} else {
			// Auto-invite disabled - log warning
			e.logger.Warn("member %s not found in team for role %s (auto-invite disabled)", memberEmail, roleName)
		}
	}

	return nil
}

// identifyOrphanedMembers identifies members and invites that should be deleted
func (e *ExecutorWithMembers) identifyOrphanedMembers(localMembers map[string]string, existingMembers map[string]models.TeamMember) *MemberDeletions {
	var orphanedUsers []string
	var orphanedInvites []string

	// Find members who exist in team but not in any role files
	for memberEmail, member := range existingMembers {
		if _, inLocalRoles := localMembers[memberEmail]; !inLocalRoles {
			// Only consider them orphaned if they're NOT assigned to any local roles
			if member.IsPendingInvite() {
				orphanedInvites = append(orphanedInvites, memberEmail)
			} else {
				orphanedUsers = append(orphanedUsers, memberEmail)
			}
		}
	}

	if len(orphanedUsers) > 0 {
		e.logger.Info("identified %d orphaned users for deletion: %v", len(orphanedUsers), orphanedUsers)
	}
	if len(orphanedInvites) > 0 {
		e.logger.Info("identified %d orphaned invites for deletion: %v", len(orphanedInvites), orphanedInvites)
	}

	return &MemberDeletions{
		OrphanedUsers:   orphanedUsers,
		OrphanedInvites: orphanedInvites,
	}
}

// deleteMembersAndInvites performs the actual deletion of orphaned users and invites
func (e *ExecutorWithMembers) deleteMembersAndInvites(deletions *MemberDeletions) error {
	if deletions == nil {
		return nil
	}

	// Delete orphaned invites
	for _, email := range deletions.OrphanedInvites {
		e.logger.Debug("deleting orphaned invitation for %s", email)
		if err := e.client.DeleteInvite(email); err != nil {
			return fmt.Errorf("failed to delete invitation for %s: %w", email, err)
		}
		e.logger.Info("successfully deleted orphaned invitation for %s", email)
	}

	// Remove orphaned users from team
	for _, email := range deletions.OrphanedUsers {
		e.logger.Debug("removing orphaned user %s from team", email)
		// Try to assign empty role to remove them
		if err := e.client.AssignMemberRole(email, ""); err != nil {
			return fmt.Errorf("failed to remove orphaned user %s: %w", email, err)
		}
		e.logger.Info("successfully removed orphaned user %s from team", email)
	}

	if len(deletions.OrphanedUsers) > 0 {
		e.logger.Info("removed %d orphaned users from team", len(deletions.OrphanedUsers))
	}
	if len(deletions.OrphanedInvites) > 0 {
		e.logger.Info("deleted %d orphaned invitations", len(deletions.OrphanedInvites))
	}

	return nil
}

// DeleteMembersAndInvites performs the actual deletion of orphaned users and invites
func (e *ExecutorWithMembers) DeleteMembersAndInvites(deletions *MemberDeletions) error {
	return e.deleteMembersAndInvites(deletions)
}

// ExecutePlanDryRun simulates executing a sync plan without making actual API calls
func (e *ExecutorWithMembers) ExecutePlanDryRun(plan SyncPlan) ExecutionResult {
	e.logger.Info("executing sync plan in dry-run mode with member support: %d creates, %d updates, %d deletes", len(plan.Creates), len(plan.Updates), len(plan.Deletes))

	result := ExecutionResult{
		Created: len(plan.Creates),
		Updated: len(plan.Updates),
		Deleted: len(plan.Deletes),
		DryRun:  true,
		Error:   nil,
	}

	e.logger.Debug("dry-run completed - no actual changes made")
	return result
}

// ExecutePlanDryRunWithDiffs simulates executing a sync plan with detailed diff information including members
func (e *ExecutorWithMembers) ExecutePlanDryRunWithDiffs(plan SyncPlan) ExecutionResult {
	result := ExecutionResult{
		Created: len(plan.Creates),
		Updated: len(plan.Updates),
		Deleted: len(plan.Deletes),
		DryRun:  true,
		Error:   nil,
	}

	// Generate detailed diff information
	detailsBuilder := make([]string, 0)

	// Add create details
	for _, role := range plan.Creates {
		detailsBuilder = append(detailsBuilder,
			fmt.Sprintf("CREATE: %s (allowed: %v, denied: %v, members: %v)",
				role.Name, role.Resources.Allowed, role.Resources.Denied, role.Members))
	}

	// Add update details with diffs
	for _, update := range plan.Updates {
		detailsBuilder = append(detailsBuilder, fmt.Sprintf("UPDATE: %s", update.Name))

		// Compare allowed resources
		allowedDiff := generateResourceDiff("allowed", update.Remote.Resources.Allowed, update.Local.Resources.Allowed)
		if allowedDiff != "" {
			detailsBuilder = append(detailsBuilder, fmt.Sprintf("  %s", allowedDiff))
		}

		// Compare denied resources
		deniedDiff := generateResourceDiff("denied", update.Remote.Resources.Denied, update.Local.Resources.Denied)
		if deniedDiff != "" {
			detailsBuilder = append(detailsBuilder, fmt.Sprintf("  %s", deniedDiff))
		}

		// Compare members
		membersDiff := generateResourceDiff("members", update.Remote.Members, update.Local.Members)
		if membersDiff != "" {
			detailsBuilder = append(detailsBuilder, fmt.Sprintf("  %s", membersDiff))
		}
	}

	// Add delete details
	for _, roleName := range plan.Deletes {
		detailsBuilder = append(detailsBuilder, fmt.Sprintf("DELETE: %s", roleName))
	}

	result.DetailedInfo = strings.Join(detailsBuilder, "\n")
	return result
}
