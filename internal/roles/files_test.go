package roles

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"replbac/internal/models"
)

func TestReadRoleFile(t *testing.T) {
	tests := []struct {
		name         string
		fileContent  string
		fileName     string
		expectedRole models.Role
		expectError  bool
		errorMessage string
	}{
		{
			name:     "valid role file",
			fileName: "admin.yaml",
			fileContent: `name: admin
resources:
  allowed:
    - "**/*"
    - kots/app/*/read
    - kots/app/*/write
  denied:
    - kots/app/*/delete`,
			expectedRole: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"**/*", "kots/app/*/read", "kots/app/*/write"},
					Denied:  []string{"kots/app/*/delete"},
				},
			},
		},
		{
			name:     "role with no denied resources",
			fileName: "viewer.yaml",
			fileContent: `name: viewer
resources:
  allowed:
    - kots/app/*/read
    - team/support-issues/read`,
			expectedRole: models.Role{
				Name: "viewer",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read", "team/support-issues/read"},
					Denied:  nil,
				},
			},
		},
		{
			name:     "role with empty resources",
			fileName: "empty.yaml",
			fileContent: `name: empty
resources:
  allowed: []
  denied: []`,
			expectedRole: models.Role{
				Name: "empty",
				Resources: models.Resources{
					Allowed: []string{},
					Denied:  []string{},
				},
			},
		},
		{
			name:     "role with mixed quoted and unquoted strings",
			fileName: "mixed.yaml",
			fileContent: `name: mixed-quotes
resources:
  allowed:
    - "**/*"
    - kots/app/*/read
    - "kots/app/*/channel/*/promote"
    - team/support-issues/read
  denied:
    - kots/app/*/delete`,
			expectedRole: models.Role{
				Name: "mixed-quotes",
				Resources: models.Resources{
					Allowed: []string{"**/*", "kots/app/*/read", "kots/app/*/channel/*/promote", "team/support-issues/read"},
					Denied:  []string{"kots/app/*/delete"},
				},
			},
		},
		{
			name:         "invalid YAML",
			fileName:     "invalid.yaml",
			fileContent:  "name: admin\n  invalid: yaml: structure",
			expectError:  true,
			errorMessage: "failed to parse YAML",
		},
		{
			name:         "missing name field",
			fileName:     "noname.yaml",
			fileContent:  "resources:\n  allowed: []\n  denied: []",
			expectError:  true,
			errorMessage: "role name is required",
		},
		{
			name:         "empty file",
			fileName:     "empty.yaml",
			fileContent:  "",
			expectError:  true,
			errorMessage: "file is empty",
		},
		{
			name:         "non-YAML file extension",
			fileName:     "role.txt",
			fileContent:  "name: test",
			expectError:  true,
			errorMessage: "not a YAML file",
		},
		{
			name:     "role with members field",
			fileName: "team-lead.yaml",
			fileContent: `name: team-lead
resources:
  allowed:
    - "kots/app/*/license/*/read"
    - "kots/app/*/license/*/write"
  denied:
    - "admin/**/*"
members:
  - "john@example.com"
  - "jane@example.com"`,
			expectedRole: models.Role{
				Name: "team-lead",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/license/*/read", "kots/app/*/license/*/write"},
					Denied:  []string{"admin/**/*"},
				},
				Members: []string{"john@example.com", "jane@example.com"},
			},
		},
		{
			name:     "role with empty members field",
			fileName: "empty-members.yaml",
			fileContent: `name: empty-members
resources:
  allowed:
    - "kots/app/*/read"
  denied: []
members: []`,
			expectedRole: models.Role{
				Name: "empty-members",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read"},
					Denied:  []string{},
				},
				Members: []string{},
			},
		},
		{
			name:     "role without members field",
			fileName: "no-members.yaml",
			fileContent: `name: no-members
resources:
  allowed:
    - "kots/app/*/read"`,
			expectedRole: models.Role{
				Name: "no-members",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read"},
					Denied:  nil,
				},
				Members: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, tt.fileName)

			if err := os.WriteFile(filePath, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			role, err := ReadRoleFile(filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMessage != "" && err.Error() != tt.errorMessage {
					t.Errorf("Error message = %v, want %v", err.Error(), tt.errorMessage)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(role, tt.expectedRole) {
				t.Errorf("Role = %+v, want %+v", role, tt.expectedRole)
			}
		})
	}
}

func TestFindRoleFiles(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()

	// Create role files
	roleFiles := map[string]string{
		"admin.yaml": `name: admin
resources:
  allowed: ["users:read"]`,
		"viewer.yml": `name: viewer
resources:
  allowed: ["read:only"]`,
		"subdir/manager.yaml": `name: manager
resources:
  allowed: ["users:write"]`,
		"subdir/deep/analyst.yaml": `name: analyst
resources:
  allowed: ["data:read"]`,
		"not-a-role.txt": "just text",
		"config.json":    `{"not": "role"}`,
	}

	for relPath, content := range roleFiles {
		filePath := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	tests := []struct {
		name          string
		rootPath      string
		expectedFiles []string
		expectError   bool
	}{
		{
			name:     "finds all YAML files recursively",
			rootPath: tmpDir,
			expectedFiles: []string{
				filepath.Join(tmpDir, "admin.yaml"),
				filepath.Join(tmpDir, "viewer.yml"),
				filepath.Join(tmpDir, "subdir", "manager.yaml"),
				filepath.Join(tmpDir, "subdir", "deep", "analyst.yaml"),
			},
		},
		{
			name:        "non-existent directory",
			rootPath:    filepath.Join(tmpDir, "nonexistent"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := FindRoleFiles(tt.rootPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(files) != len(tt.expectedFiles) {
				t.Errorf("Found %d files, expected %d", len(files), len(tt.expectedFiles))
				t.Errorf("Found: %v", files)
				t.Errorf("Expected: %v", tt.expectedFiles)
				return
			}

			// Convert to map for easy comparison (order doesn't matter)
			foundMap := make(map[string]bool)
			for _, file := range files {
				foundMap[file] = true
			}

			for _, expected := range tt.expectedFiles {
				if !foundMap[expected] {
					t.Errorf("Expected file %s not found", expected)
				}
			}
		})
	}
}

func TestLoadRolesFromDirectory(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()

	roleFiles := map[string]string{
		"admin.yaml": `name: admin
resources:
  allowed: 
    - "**/*"
    - kots/app/*/read
  denied: 
    - kots/app/*/delete`,
		"viewer.yml": `name: viewer
resources:
  allowed: 
    - kots/app/*/read`,
		"invalid.yaml": "name: invalid\n  bad: yaml: syntax",
		"subdir/manager.yaml": `name: manager
resources:
  allowed: 
    - kots/app/*/write
    - kots/app/*/channel/*/read`,
	}

	for relPath, content := range roleFiles {
		filePath := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	tests := []struct {
		name          string
		rootPath      string
		expectedRoles []models.Role
		expectError   bool
		errorContains string
	}{
		{
			name:     "loads all valid roles",
			rootPath: tmpDir,
			expectedRoles: []models.Role{
				{
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"**/*", "kots/app/*/read"},
						Denied:  []string{"kots/app/*/delete"},
					},
				},
				{
					Name: "viewer",
					Resources: models.Resources{
						Allowed: []string{"kots/app/*/read"},
						Denied:  nil,
					},
				},
				{
					Name: "manager",
					Resources: models.Resources{
						Allowed: []string{"kots/app/*/write", "kots/app/*/channel/*/read"},
						Denied:  nil,
					},
				},
			},
		},
		{
			name:        "non-existent directory",
			rootPath:    filepath.Join(tmpDir, "nonexistent"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles, err := LoadRolesFromDirectory(tt.rootPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.errorContains != "" && err != nil {
					if !containsString(err.Error(), tt.errorContains) {
						t.Errorf("Error should contain %q, got: %v", tt.errorContains, err)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(roles) != len(tt.expectedRoles) {
				t.Errorf("Found %d roles, expected %d", len(roles), len(tt.expectedRoles))
				return
			}

			// Convert to map for easy comparison (order doesn't matter)
			foundMap := make(map[string]models.Role)
			for _, role := range roles {
				foundMap[role.Name] = role
			}

			for _, expected := range tt.expectedRoles {
				found, exists := foundMap[expected.Name]
				if !exists {
					t.Errorf("Expected role %s not found", expected.Name)
					continue
				}
				if !reflect.DeepEqual(found, expected) {
					t.Errorf("Role %s = %+v, want %+v", expected.Name, found, expected)
				}
			}
		})
	}
}

func TestValidateRoleFile(t *testing.T) {
	tests := []struct {
		name        string
		role        models.Role
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid role",
			role: models.Role{
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"users:read"},
				},
			},
		},
		{
			name: "empty name",
			role: models.Role{
				Name: "",
				Resources: models.Resources{
					Allowed: []string{"users:read"},
				},
			},
			expectError: true,
			errorMsg:    "role name is required",
		},
		{
			name: "no resources allowed",
			role: models.Role{
				Name:      "test",
				Resources: models.Resources{},
			},
		},
		{
			name: "empty allowed resources",
			role: models.Role{
				Name: "test",
				Resources: models.Resources{
					Allowed: []string{},
					Denied:  []string{"something"},
				},
			},
		},
		{
			name: "valid role with members",
			role: models.Role{
				Name: "team-lead",
				Resources: models.Resources{
					Allowed: []string{"users:read", "users:write"},
				},
				Members: []string{"john@example.com", "jane@example.com"},
			},
		},
		{
			name: "valid role with empty members",
			role: models.Role{
				Name: "empty-members",
				Resources: models.Resources{
					Allowed: []string{"users:read"},
				},
				Members: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRole(tt.role)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Error message = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// rolesEqual compares two roles considering nil vs empty slice equivalence
func rolesEqual(a, b models.Role) bool {
	if a.ID != b.ID || a.Name != b.Name {
		return false
	}

	if !slicesEqual(a.Resources.Allowed, b.Resources.Allowed) {
		return false
	}

	if !slicesEqual(a.Resources.Denied, b.Resources.Denied) {
		return false
	}

	if !slicesEqual(a.Members, b.Members) {
		return false
	}

	return true
}

// slicesEqual compares two string slices treating nil and empty as equal
func slicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestWriteRoleFile_WithMembers(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		role     models.Role
		fileName string
	}{
		{
			name: "role with members",
			role: models.Role{
				Name: "team-lead",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read", "kots/app/*/write"},
					Denied:  []string{"admin/**/*"},
				},
				Members: []string{"john@example.com", "jane@example.com"},
			},
			fileName: "team-lead.yaml",
		},
		{
			name: "role with empty members",
			role: models.Role{
				Name: "no-members",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read"},
				},
				Members: []string{},
			},
			fileName: "no-members.yaml",
		},
		{
			name: "role without members field",
			role: models.Role{
				Name: "simple",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read"},
				},
				Members: []string{},
			},
			fileName: "simple.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.fileName)

			// Write the role file
			err := WriteRoleFile(tt.role, filePath)
			if err != nil {
				t.Fatalf("Failed to write role file: %v", err)
			}

			// Read it back and verify
			readRole, err := ReadRoleFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read role file: %v", err)
			}

			if !rolesEqual(readRole, tt.role) {
				t.Errorf("Read role = %+v, want %+v", readRole, tt.role)
			}
		})
	}
}

func TestGenerateRoleYAML_WithMembers(t *testing.T) {
	tests := []struct {
		name            string
		role            models.Role
		expectMembers   bool
		expectedContent []string
	}{
		{
			name: "role with members",
			role: models.Role{
				Name: "team-lead",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read"},
				},
				Members: []string{"john@example.com", "jane@example.com"},
			},
			expectMembers: true,
			expectedContent: []string{
				"name: team-lead",
				"members:",
				"- john@example.com",
				"- jane@example.com",
			},
		},
		{
			name: "role with empty members",
			role: models.Role{
				Name: "empty-members",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read"},
				},
				Members: []string{},
			},
			expectMembers: false,
			expectedContent: []string{
				"name: empty-members",
				"allowed:",
				"- kots/app/*/read",
			},
		},
		{
			name: "role without members field",
			role: models.Role{
				Name: "no-members",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read"},
				},
			},
			expectMembers: false,
			expectedContent: []string{
				"name: no-members",
				"allowed:",
				"- kots/app/*/read",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml, err := GenerateRoleYAML(tt.role)
			if err != nil {
				t.Fatalf("Failed to generate YAML: %v", err)
			}

			// Check for expected content
			for _, content := range tt.expectedContent {
				if !containsSubstring(yaml, content) {
					t.Errorf("Expected YAML to contain %q, got:\n%s", content, yaml)
				}
			}

			// Check if members field is present when expected
			hasMembersField := containsSubstring(yaml, "members:")
			if tt.expectMembers && !hasMembersField {
				t.Errorf("Expected YAML to contain members field, got:\n%s", yaml)
			}
			if !tt.expectMembers && hasMembersField {
				t.Errorf("Expected YAML to not contain members field, got:\n%s", yaml)
			}
		})
	}
}

func TestValidateRoleMembers(t *testing.T) {
	tests := []struct {
		name        string
		roles       []models.Role
		expectError bool
		errorMsg    string
	}{
		{
			name: "no members - valid",
			roles: []models.Role{
				{Name: "admin", Members: []string{}},
				{Name: "viewer", Members: []string{}},
			},
		},
		{
			name: "unique members across roles - valid",
			roles: []models.Role{
				{
					Name:    "admin",
					Members: []string{"john@example.com", "jane@example.com"},
				},
				{
					Name:    "viewer",
					Members: []string{"bob@example.com", "alice@example.com"},
				},
			},
		},
		{
			name: "single role with members - valid",
			roles: []models.Role{
				{
					Name:    "admin",
					Members: []string{"john@example.com", "jane@example.com"},
				},
			},
		},
		{
			name: "duplicate member across roles - invalid",
			roles: []models.Role{
				{
					Name:    "admin",
					Members: []string{"john@example.com", "jane@example.com"},
				},
				{
					Name:    "viewer",
					Members: []string{"bob@example.com", "john@example.com"},
				},
			},
			expectError: true,
			errorMsg:    "member john@example.com appears in multiple roles: admin, viewer",
		},
		{
			name: "duplicate member within same role - invalid",
			roles: []models.Role{
				{
					Name:    "admin",
					Members: []string{"john@example.com", "jane@example.com", "john@example.com"},
				},
			},
			expectError: true,
			errorMsg:    "member john@example.com appears multiple times in role admin",
		},
		{
			name: "multiple duplicates - reports first found",
			roles: []models.Role{
				{
					Name:    "admin",
					Members: []string{"john@example.com", "jane@example.com"},
				},
				{
					Name:    "viewer",
					Members: []string{"john@example.com", "alice@example.com"},
				},
				{
					Name:    "editor",
					Members: []string{"bob@example.com", "jane@example.com"},
				},
			},
			expectError: true,
			errorMsg:    "member john@example.com appears in multiple roles: admin, viewer",
		},
		{
			name: "empty member email - invalid",
			roles: []models.Role{
				{
					Name:    "admin",
					Members: []string{"john@example.com", "", "jane@example.com"},
				},
			},
			expectError: true,
			errorMsg:    "empty member email found in role admin",
		},
		{
			name: "whitespace-only member email - invalid",
			roles: []models.Role{
				{
					Name:    "admin",
					Members: []string{"john@example.com", "   ", "jane@example.com"},
				},
			},
			expectError: true,
			errorMsg:    "empty member email found in role admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoleMembers(tt.roles)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Error message = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
