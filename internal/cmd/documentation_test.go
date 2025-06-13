package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestMemberDocumentationExists validates that comprehensive member field documentation exists
func TestMemberDocumentationExists(t *testing.T) {
	tests := []struct {
		name         string
		filePath     string
		requiredText []string
	}{
		{
			name:     "README contains member field documentation",
			filePath: "../../README.md",
			requiredText: []string{
				"members",
				"member assignment",
				"team member",
				"@example.com",
			},
		},
		{
			name:     "Example YAML file with members exists",
			filePath: "../../examples/admin-with-members.yaml",
			requiredText: []string{
				"members:",
				"@",
			},
		},
		{
			name:     "Example YAML file with member validation exists",
			filePath: "../../examples/viewer-with-members.yaml",
			requiredText: []string{
				"members:",
				"@",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(tt.filePath)
			if err != nil {
				t.Errorf("Documentation file %s does not exist: %v", tt.filePath, err)
				return
			}

			contentStr := string(content)
			for _, required := range tt.requiredText {
				if !strings.Contains(contentStr, required) {
					t.Errorf("File %s missing required documentation text: %s", tt.filePath, required)
				}
			}
		})
	}
}

// TestREADMEMemberExamples validates that README contains comprehensive member examples
func TestREADMEMemberExamples(t *testing.T) {
	content, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("README.md does not exist: %v", err)
	}

	readme := string(content)
	
	requiredSections := []string{
		"## Member Management",
		"### Member Assignment",
		"### Member Validation Rules",
		"replbac sync",
		"members:",
	}

	for _, section := range requiredSections {
		if !strings.Contains(readme, section) {
			t.Errorf("README.md missing required section or content: %s", section)
		}
	}
}

// TestExampleYAMLFiles validates that example files demonstrate member usage properly
func TestExampleYAMLFiles(t *testing.T) {
	exampleFiles := []string{
		"../../examples/admin-with-members.yaml",
		"../../examples/viewer-with-members.yaml",
	}

	for _, filePath := range exampleFiles {
		t.Run(filePath, func(t *testing.T) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("Example file %s does not exist: %v", filePath, err)
				return
			}

			contentStr := string(content)
			
			// Check for required YAML structure
			if !strings.Contains(contentStr, "name:") {
				t.Errorf("Example file %s missing 'name:' field", filePath)
			}
			if !strings.Contains(contentStr, "resources:") {
				t.Errorf("Example file %s missing 'resources:' field", filePath)
			}
			if !strings.Contains(contentStr, "members:") {
				t.Errorf("Example file %s missing 'members:' field", filePath)
			}
			if !strings.Contains(contentStr, "@") {
				t.Errorf("Example file %s missing email addresses in members", filePath)
			}
		})
	}
}