package sync

import (
	"fmt"

	"replbac/internal/models"
)

// APIClient defines the interface for API operations needed by the executor
type APIClient interface {
	CreateRole(role models.Role) error
	UpdateRole(role models.Role) error
	DeleteRole(roleName string) error
}

// Executor handles the execution of sync plans
type Executor struct {
	client APIClient
}

// ExecutionResult represents the result of executing a sync plan
type ExecutionResult struct {
	Created int   // Number of roles created
	Updated int   // Number of roles updated
	Deleted int   // Number of roles deleted
	Error   error // Error if execution failed
	DryRun  bool  // Whether this was a dry run
}

// NewExecutor creates a new sync executor with the given API client
func NewExecutor(client APIClient) *Executor {
	return &Executor{
		client: client,
	}
}

// ExecutePlan executes a sync plan by making actual API calls
func (e *Executor) ExecutePlan(plan SyncPlan) ExecutionResult {
	result := ExecutionResult{
		DryRun: false,
	}

	// Execute creates
	for _, role := range plan.Creates {
		if err := e.client.CreateRole(role); err != nil {
			result.Error = fmt.Errorf("failed to create role '%s': %w", role.Name, err)
			return result
		}
		result.Created++
	}

	// Execute updates
	for _, update := range plan.Updates {
		if err := e.client.UpdateRole(update.Local); err != nil {
			result.Error = fmt.Errorf("failed to update role '%s': %w", update.Name, err)
			return result
		}
		result.Updated++
	}

	// Execute deletes
	for _, roleName := range plan.Deletes {
		if err := e.client.DeleteRole(roleName); err != nil {
			result.Error = fmt.Errorf("failed to delete role '%s': %w", roleName, err)
			return result
		}
		result.Deleted++
	}

	return result
}

// ExecutePlanDryRun simulates executing a sync plan without making actual API calls
func (e *Executor) ExecutePlanDryRun(plan SyncPlan) ExecutionResult {
	result := ExecutionResult{
		Created: len(plan.Creates),
		Updated: len(plan.Updates),
		Deleted: len(plan.Deletes),
		DryRun:  true,
		Error:   nil,
	}

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