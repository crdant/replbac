package sync

import (
	"reflect"
	"sort"
	"testing"

	"replbac/internal/models"
)

func TestCompareRoles(t *testing.T) {
	tests := []struct {
		name      string
		local     []models.Role
		remote    []models.Role
		wantPlan  SyncPlan
		wantError bool
	}{
		{
			name:   "empty roles",
			local:  []models.Role{},
			remote: []models.Role{},
			wantPlan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			wantError: false,
		},
		{
			name: "local role needs to be created on remote",
			local: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
			},
			remote: []models.Role{},
			wantPlan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "admin",
						Resources: models.Resources{
							Allowed: []string{"*"},
							Denied:  []string{},
						},
					},
				},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			wantError: false,
		},
		{
			name:  "remote role needs to be deleted",
			local: []models.Role{},
			remote: []models.Role{
				{
					Name: "obsolete",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			wantPlan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{"obsolete"},
			},
			wantError: false,
		},
		{
			name: "role needs to be updated",
			local: []models.Role{
				{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{"delete"},
					},
				},
			},
			remote: []models.Role{
				{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			wantPlan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{
					{
						Name: "editor",
						Local: models.Role{
							Name: "editor",
							Resources: models.Resources{
								Allowed: []string{"read", "write"},
								Denied:  []string{"delete"},
							},
						},
						Remote: models.Role{
							Name: "editor",
							Resources: models.Resources{
								Allowed: []string{"read"},
								Denied:  []string{},
							},
						},
					},
				},
				Deletes: []string{},
			},
			wantError: false,
		},
		{
			name: "complex scenario with creates, updates, and deletes",
			local: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
				{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read", "write"},
						Denied:  []string{"delete"},
					},
				},
				{
					Name: "viewer",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			remote: []models.Role{
				{
					Name: "editor",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
				{
					Name: "obsolete",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
				{
					Name: "viewer",
					Resources: models.Resources{
						Allowed: []string{"read"},
						Denied:  []string{},
					},
				},
			},
			wantPlan: SyncPlan{
				Creates: []models.Role{
					{
						Name: "admin",
						Resources: models.Resources{
							Allowed: []string{"*"},
							Denied:  []string{},
						},
					},
				},
				Updates: []RoleUpdate{
					{
						Name: "editor",
						Local: models.Role{
							Name: "editor",
							Resources: models.Resources{
								Allowed: []string{"read", "write"},
								Denied:  []string{"delete"},
							},
						},
						Remote: models.Role{
							Name: "editor",
							Resources: models.Resources{
								Allowed: []string{"read"},
								Denied:  []string{},
							},
						},
					},
				},
				Deletes: []string{"obsolete"},
			},
			wantError: false,
		},
		{
			name: "roles are identical - no changes needed",
			local: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
			},
			remote: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"*"},
						Denied:  []string{},
					},
				},
			},
			wantPlan: SyncPlan{
				Creates: []models.Role{},
				Updates: []RoleUpdate{},
				Deletes: []string{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPlan, err := CompareRoles(tt.local, tt.remote)

			if (err != nil) != tt.wantError {
				t.Errorf("CompareRoles() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// Sort slices for consistent comparison
			sort.Slice(gotPlan.Creates, func(i, j int) bool {
				return gotPlan.Creates[i].Name < gotPlan.Creates[j].Name
			})
			sort.Slice(gotPlan.Updates, func(i, j int) bool {
				return gotPlan.Updates[i].Name < gotPlan.Updates[j].Name
			})
			sort.Strings(gotPlan.Deletes)

			sort.Slice(tt.wantPlan.Creates, func(i, j int) bool {
				return tt.wantPlan.Creates[i].Name < tt.wantPlan.Creates[j].Name
			})
			sort.Slice(tt.wantPlan.Updates, func(i, j int) bool {
				return tt.wantPlan.Updates[i].Name < tt.wantPlan.Updates[j].Name
			})
			sort.Strings(tt.wantPlan.Deletes)

			if !reflect.DeepEqual(gotPlan, tt.wantPlan) {
				t.Errorf("CompareRoles() = %+v, want %+v", gotPlan, tt.wantPlan)
			}
		})
	}
}

func TestRolesEqual(t *testing.T) {
	tests := []struct {
		name string
		r1   models.Role
		r2   models.Role
		want bool
	}{
		{
			name: "identical roles",
			r1: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"read", "write"},
					Denied:  []string{"delete"},
				},
			},
			r2: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"read", "write"},
					Denied:  []string{"delete"},
				},
			},
			want: true,
		},
		{
			name: "different names",
			r1: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"read"},
					Denied:  []string{},
				},
			},
			r2: models.Role{
				Name: "user",
				Resources: models.Resources{
					Allowed: []string{"read"},
					Denied:  []string{},
				},
			},
			want: false,
		},
		{
			name: "different allowed resources",
			r1: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"read", "write"},
					Denied:  []string{},
				},
			},
			r2: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"read"},
					Denied:  []string{},
				},
			},
			want: false,
		},
		{
			name: "different denied resources",
			r1: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"read"},
					Denied:  []string{"delete"},
				},
			},
			r2: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"read"},
					Denied:  []string{},
				},
			},
			want: false,
		},
		{
			name: "same resources different order",
			r1: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"write", "read"},
					Denied:  []string{"admin", "delete"},
				},
			},
			r2: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"read", "write"},
					Denied:  []string{"delete", "admin"},
				},
			},
			want: true,
		},
		{
			name: "empty resources",
			r1: models.Role{
				Name: "empty",
				Resources: models.Resources{
					Allowed: []string{},
					Denied:  []string{},
				},
			},
			r2: models.Role{
				Name: "empty",
				Resources: models.Resources{
					Allowed: []string{},
					Denied:  []string{},
				},
			},
			want: true,
		},
		{
			name: "nil vs empty slices",
			r1: models.Role{
				Name: "test",
				Resources: models.Resources{
					Allowed: nil,
					Denied:  nil,
				},
			},
			r2: models.Role{
				Name: "test",
				Resources: models.Resources{
					Allowed: []string{},
					Denied:  []string{},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RolesEqual(tt.r1, tt.r2); got != tt.want {
				t.Errorf("RolesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}