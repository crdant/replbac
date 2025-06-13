package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateManPage(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
	}{
		{
			name: "man page contains required sections",
			expected: []string{
				".TH REPLBAC 1",
				".SH NAME",
				".SH SYNOPSIS",
				".SH DESCRIPTION",
				".SH COMMANDS",
				".SH OPTIONS",
				".SH ENVIRONMENT",
				".SH FILES",
				".SH EXAMPLES",
				".SH SEE ALSO",
				".SH AUTHOR",
			},
		},
		{
			name: "man page contains command descriptions",
			expected: []string{
				"sync",
				"pull",
				"version",
				"Synchronize local role files",
				"Pull role definitions from",
			},
		},
		{
			name: "man page contains flag descriptions",
			expected: []string{
				"--api-token",
				"--config",
				"--dry-run",
				"--delete",
				"--force",
				"--verbose",
				"--debug",
			},
		},
		{
			name: "man page contains environment variables",
			expected: []string{
				"REPLICATED_API_TOKEN",
				"REPLBAC_API_TOKEN",
				"REPLBAC_CONFIG",
				"REPLBAC_CONFIRM",
				"REPLBAC_LOG_LEVEL",
			},
		},
	}

	// Generate the man page content
	content, err := GenerateManPage()
	if err != nil {
		t.Fatalf("GenerateManPage() failed: %v", err)
	}

	// Validate content is not empty
	if strings.TrimSpace(content) == "" {
		t.Fatal("Generated man page content is empty")
	}

	// Run all test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, expected := range tc.expected {
				if !strings.Contains(content, expected) {
					t.Errorf("Man page content missing expected string: %q", expected)
				}
			}
		})
	}
}

func TestWriteManPageToFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	manFile := filepath.Join(tempDir, "replbac.1")

	// Write man page to file
	err := WriteManPageToFile(manFile)
	if err != nil {
		t.Fatalf("WriteManPageToFile() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(manFile); os.IsNotExist(err) {
		t.Fatal("Man page file was not created")
	}

	// Read and verify content
	// #nosec G304 -- Reading test file path is expected behavior in tests
	content, err := os.ReadFile(manFile)
	if err != nil {
		t.Fatalf("Failed to read man page file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("Man page file is empty")
	}

	// Check it starts with proper man page header
	contentStr := string(content)
	if !strings.HasPrefix(contentStr, ".TH REPLBAC 1") {
		t.Error("Man page does not start with proper header")
	}
}

func TestManPageGenerator(t *testing.T) {
	// Test that the man page generation works standalone
	content, err := GenerateManPage()
	if err != nil {
		t.Fatalf("GenerateManPage() failed: %v", err)
	}

	if strings.TrimSpace(content) == "" {
		t.Fatal("Generated man page content is empty")
	}

	// Check it starts with proper man page header
	if !strings.HasPrefix(content, ".TH REPLBAC 1") {
		t.Error("Man page does not start with proper header")
	}
}
