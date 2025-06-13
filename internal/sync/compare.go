package sync

import (
	"fmt"
	"sort"

	"replbac/internal/models"
)

// SyncPlan represents the plan for synchronizing local roles with remote roles
type SyncPlan struct {
	Creates []models.Role // Roles that need to be created on remote
	Updates []RoleUpdate  // Roles that need to be updated on remote
	Deletes []string      // Role names that need to be deleted from remote
}

// RoleUpdate represents a role that needs to be updated
type RoleUpdate struct {
	Name   string      // Role name
	Local  models.Role // Local version of the role
	Remote models.Role // Remote version of the role
}

// CompareRoles compares local roles with remote roles and returns a sync plan
func CompareRoles(local, remote []models.Role) (SyncPlan, error) {
	plan := SyncPlan{
		Creates: []models.Role{},
		Updates: []RoleUpdate{},
		Deletes: []string{},
	}

	// Create maps for efficient lookups
	localMap := make(map[string]models.Role)
	remoteMap := make(map[string]models.Role)

	// Build local role map
	for _, role := range local {
		localMap[role.Name] = role
	}

	// Build remote role map
	for _, role := range remote {
		remoteMap[role.Name] = role
	}

	// Find roles that need to be created or updated
	for _, localRole := range local {
		remoteRole, exists := remoteMap[localRole.Name]
		if !exists {
			// Role doesn't exist on remote, needs to be created
			plan.Creates = append(plan.Creates, localRole)
		} else if !RolesEqual(localRole, remoteRole) {
			// Role exists but is different, needs to be updated
			plan.Updates = append(plan.Updates, RoleUpdate{
				Name:   localRole.Name,
				Local:  localRole,
				Remote: remoteRole,
			})
		}
		// If roles are equal, no action needed
	}

	// Find roles that need to be deleted
	for _, remoteRole := range remote {
		if _, exists := localMap[remoteRole.Name]; !exists {
			// Role exists on remote but not local, needs to be deleted
			plan.Deletes = append(plan.Deletes, remoteRole.Name)
		}
	}

	return plan, nil
}

// RolesEqual compares two roles for equality, ignoring order of resources and members
func RolesEqual(r1, r2 models.Role) bool {
	// Compare names
	if r1.Name != r2.Name {
		return false
	}

	// Compare resources
	if !ResourcesEqual(r1.Resources, r2.Resources) {
		return false
	}

	// Compare members
	return StringSlicesEqual(r1.Members, r2.Members)
}

// ResourcesEqual compares two resource structures for equality, ignoring order
func ResourcesEqual(r1, r2 models.Resources) bool {
	// Compare allowed resources
	if !StringSlicesEqual(r1.Allowed, r2.Allowed) {
		return false
	}

	// Compare denied resources
	if !StringSlicesEqual(r1.Denied, r2.Denied) {
		return false
	}

	return true
}

// StringSlicesEqual compares two string slices for equality, ignoring order
// and treating nil slices as equivalent to empty slices
func StringSlicesEqual(s1, s2 []string) bool {
	// Handle nil slices
	if s1 == nil {
		s1 = []string{}
	}
	if s2 == nil {
		s2 = []string{}
	}

	// Check lengths
	if len(s1) != len(s2) {
		return false
	}

	// If both are empty, they're equal
	if len(s1) == 0 {
		return true
	}

	// Sort copies to compare regardless of order
	sorted1 := make([]string, len(s1))
	sorted2 := make([]string, len(s2))
	copy(sorted1, s1)
	copy(sorted2, s2)
	sort.Strings(sorted1)
	sort.Strings(sorted2)

	// Compare sorted slices
	for i := range sorted1 {
		if sorted1[i] != sorted2[i] {
			return false
		}
	}

	return true
}

// HasChanges returns true if the sync plan contains any changes
func (p SyncPlan) HasChanges() bool {
	return len(p.Creates) > 0 || len(p.Updates) > 0 || len(p.Deletes) > 0
}

// Summary returns a human-readable summary of the sync plan
func (p SyncPlan) Summary() string {
	if !p.HasChanges() {
		return "No changes needed"
	}

	summary := ""
	if len(p.Creates) > 0 {
		if summary != "" {
			summary += ", "
		}
		summary += fmt.Sprintf("%d to create", len(p.Creates))
	}
	if len(p.Updates) > 0 {
		if summary != "" {
			summary += ", "
		}
		summary += fmt.Sprintf("%d to update", len(p.Updates))
	}
	if len(p.Deletes) > 0 {
		if summary != "" {
			summary += ", "
		}
		summary += fmt.Sprintf("%d to delete", len(p.Deletes))
	}

	return summary
}
