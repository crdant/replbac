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
}

// APIClientWithMembers extends APIClient with member management operations
type APIClientWithMembers interface {
	APIClient
	GetTeamMembers() ([]models.TeamMember, error)
	AssignMemberRole(memberEmail, roleID string) error
	InviteUser(email, policyID string) (*models.InviteUserResponse, error)
}

// Executor handles the execution of sync plans
type Executor struct {
	client APIClient
	logger *logging.Logger
}

// ExecutorWithMembers handles the execution of sync plans including member assignments
type ExecutorWithMembers struct {
	client APIClientWithMembers
	logger *logging.Logger
}

// ExecutionResult represents the result of executing a sync plan
type ExecutionResult struct {
	Created      int    // Number of roles created
	Updated      int    // Number of roles updated
	Deleted      int    // Number of roles deleted
	Error        error  // Error if execution failed
	DryRun       bool   // Whether this was a dry run
	DetailedInfo string // Detailed information about changes (for enhanced dry-run)
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
		client: client,
		logger: logger,
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

	// Execute creates with member assignments
	for _, role := range plan.Creates {
		e.logger.Debug("creating role: %s", role.Name)
		if err := e.client.CreateRole(role); err != nil {
			e.logger.Error("failed to create role %s: %v", role.Name, err)
			result.Error = fmt.Errorf("failed to create role '%s': %w", role.Name, err)
			return result
		}
		e.logger.Info("successfully created role: %s", role.Name)
		result.Created++

		// Assign members to the newly created role
		if err := e.assignMembersToRole(role.Name, role.Members); err != nil {
			e.logger.Error("failed to assign members to role %s: %v", role.Name, err)
			result.Error = fmt.Errorf("failed to assign members to role '%s': %w", role.Name, err)
			return result
		}
	}

	// Execute updates with member assignments
	for _, update := range plan.Updates {
		e.logger.Debug("updating role: %s", update.Name)
		if err := e.client.UpdateRole(update.Local); err != nil {
			e.logger.Error("failed to update role %s: %v", update.Name, err)
			result.Error = fmt.Errorf("failed to update role '%s': %w", update.Name, err)
			return result
		}
		e.logger.Info("successfully updated role: %s", update.Name)
		result.Updated++

		// Update member assignments for the role
		if err := e.assignMembersToRole(update.Name, update.Local.Members); err != nil {
			e.logger.Error("failed to assign members to role %s: %v", update.Name, err)
			result.Error = fmt.Errorf("failed to assign members to role '%s': %w", update.Name, err)
			return result
		}
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

// assignMembersToRole assigns all specified members to a role, inviting missing members first
func (e *ExecutorWithMembers) assignMembersToRole(roleName string, members []string) error {
	if len(members) == 0 {
		e.logger.Debug("no members to assign to role: %s", roleName)
		return nil
	}

	e.logger.Debug("assigning %d members to role: %s", len(members), roleName)

	// Get current team members to check who needs to be invited
	teamMembers, err := e.client.GetTeamMembers()
	if err != nil {
		e.logger.Error("failed to get team members: %v", err)
		return fmt.Errorf("failed to get team members: %w", err)
	}

	// Create a map of existing member emails for fast lookup
	existingMembers := make(map[string]bool)
	for _, member := range teamMembers {
		existingMembers[member.Email] = true
	}

	// Invite missing members first
	var inviteCount int
	for _, memberEmail := range members {
		if !existingMembers[memberEmail] {
			e.logger.Debug("member %s not found in team, sending invite", memberEmail)
			response, err := e.client.InviteUser(memberEmail, roleName)
			if err != nil {
				e.logger.Error("failed to invite member %s: %v", memberEmail, err)
				return fmt.Errorf("failed to invite member '%s': %w", memberEmail, err)
			}
			e.logger.Info("successfully invited member %s (status: %s)", memberEmail, response.Status)
			inviteCount++
		}
	}

	if inviteCount > 0 {
		e.logger.Info("invited %d new members for role: %s", inviteCount, roleName)
	}

	// Assign all members to the role (both existing and newly invited)
	for _, memberEmail := range members {
		e.logger.Debug("assigning member %s to role %s", memberEmail, roleName)
		if err := e.client.AssignMemberRole(memberEmail, roleName); err != nil {
			e.logger.Error("failed to assign member %s to role %s: %v", memberEmail, roleName, err)
			return fmt.Errorf("failed to assign member '%s' to role '%s': %w", memberEmail, roleName, err)
		}
	}
	e.logger.Info("successfully assigned %d members to role: %s", len(members), roleName)
	return nil
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
