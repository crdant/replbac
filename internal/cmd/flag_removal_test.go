package cmd

import (
	"bytes"
	"testing"
)

// TestRolesDirFlagRemoval tests that --roles-dir flag has been removed from commands
func TestRolesDirFlagRemoval(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
		errorContains string
	}{
		{
			name:        "sync rejects --roles-dir flag",
			command:     "sync",
			args:        []string{"--roles-dir", "test"},
			expectError: true,
			errorContains: "unknown flag: --roles-dir",
		},
		{
			name:        "pull rejects --roles-dir flag", 
			command:     "pull",
			args:        []string{"--roles-dir", "test"},
			expectError: true,
			errorContains: "unknown flag: --roles-dir",
		},
		{
			name:        "sync accepts positional directory argument",
			command:     "sync",
			args:        []string{"test-dir", "--help"},
			expectError: false,
		},
		{
			name:        "pull accepts positional directory argument",
			command:     "pull", 
			args:        []string{"test-dir", "--help"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh root command for each test
			cmd := createTestRootCmd()
			
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			
			// Set the command and args
			cmdArgs := append([]string{tt.command}, tt.args...)
			cmd.SetArgs(cmdArgs)

			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestPositionalArgumentFunctionality tests that positional directory arguments work correctly
func TestPositionalArgumentFunctionality(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		args     []string
		contains string
	}{
		{
			name:     "sync help shows positional directory argument",
			command:  "sync",
			args:     []string{"--help"},
			contains: "sync [directory]",
		},
		{
			name:     "pull help shows positional directory argument",
			command:  "pull",
			args:     []string{"--help"},
			contains: "pull [directory]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createTestRootCmd()
			
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			
			cmdArgs := append([]string{tt.command}, tt.args...)
			cmd.SetArgs(cmdArgs)

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			outputStr := output.String()
			if !containsString(outputStr, tt.contains) {
				t.Errorf("Expected output to contain %q, got: %q", tt.contains, outputStr)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		   (len(s) > len(substr) && func() bool {
			   for i := 0; i <= len(s)-len(substr); i++ {
				   if s[i:i+len(substr)] == substr {
					   return true
				   }
			   }
			   return false
		   }()))
}