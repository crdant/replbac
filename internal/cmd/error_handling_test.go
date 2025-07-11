package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"replbac/internal/models"
)

// TestEnhancedErrorHandling tests comprehensive error handling scenarios
func TestEnhancedErrorHandling(t *testing.T) {
	tests := []struct {
		name               string
		setup              func(t *testing.T) (string, func())
		config             models.Config
		args               []string
		flags              map[string]string
		expectError        bool
		expectExitCode     int
		expectOutput       []string
		expectErrorMessage string
		expectUserGuidance bool
	}{
		{
			name: "invalid directory path - user guidance",
			setup: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
			config: models.Config{
				APIToken: "test-token",
			},
			args:               []string{"/nonexistent/path"},
			expectError:        true,
			expectExitCode:     1,
			expectOutput:       []string{"Error:", "directory does not exist"},
			expectErrorMessage: "failed to load local roles",
			expectUserGuidance: true,
		},
		{
			name: "invalid YAML files - recoverable error with guidance",
			setup: func(t *testing.T) (string, func()) {
				tempDir, err := os.MkdirTemp("", "replbac-error-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create invalid YAML file
				invalidYAML := `name: test
resources:
  - invalid: structure
  - that: [will, cause, parsing, errors`
				// #nosec G306 -- Test files need readable permissions
				err = os.WriteFile(filepath.Join(tempDir, "invalid.yaml"), []byte(invalidYAML), 0644)
				if err != nil {
					t.Fatalf("Failed to write invalid YAML: %v", err)
				}

				return tempDir, func() { _ = os.RemoveAll(tempDir) }
			},
			config: models.Config{
				APIToken: "test-token",
			},
			expectError:        false, // Should continue with valid files
			expectOutput:       []string{"Warning:", "Skipped", "invalid.yaml"},
			expectUserGuidance: true,
		},
		{
			name: "API connection failure - clear error message",
			setup: func(t *testing.T) (string, func()) {
				tempDir, err := os.MkdirTemp("", "replbac-error-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create valid role file
				validYAML := `name: test
resources:
  allowed: ["read"]
  denied: []`
				// #nosec G306 -- Test files need readable permissions
				err = os.WriteFile(filepath.Join(tempDir, "valid.yaml"), []byte(validYAML), 0644)
				if err != nil {
					t.Fatalf("Failed to write valid YAML: %v", err)
				}

				return tempDir, func() { _ = os.RemoveAll(tempDir) }
			},
			config: models.Config{
				APIToken: "test-token",
			},
			expectError:        true,
			expectExitCode:     1,
			expectOutput:       []string{"Error:", "failed to get remote roles"},
			expectErrorMessage: "failed to get remote roles",
			expectUserGuidance: true,
		},
		{
			name: "missing API token - configuration error with guidance",
			setup: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
			config: models.Config{
				APIToken: "", // Missing token
			},
			expectError:        true,
			expectExitCode:     1,
			expectOutput:       []string{"Configuration Error:", "API token is required"},
			expectErrorMessage: "invalid configuration",
			expectUserGuidance: true,
		},
		{
			name: "partial sync failure - rollback with clear status",
			setup: func(t *testing.T) (string, func()) {
				tempDir, err := os.MkdirTemp("", "replbac-error-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create role that will trigger API error during sync
				problematicYAML := `name: problematic-role
resources:
  allowed: ["*"]
  denied: []`
				// #nosec G306 -- Test files need readable permissions
				err = os.WriteFile(filepath.Join(tempDir, "problematic.yaml"), []byte(problematicYAML), 0644)
				if err != nil {
					t.Fatalf("Failed to write problematic YAML: %v", err)
				}

				return tempDir, func() { _ = os.RemoveAll(tempDir) }
			},
			config: models.Config{
				APIToken: "test-token",
			},
			expectError:        true,
			expectExitCode:     1,
			expectOutput:       []string{"Sync failed:", "0 operations completed", "Rollback"},
			expectErrorMessage: "sync operation failed",
			expectUserGuidance: true,
		},
		{
			name: "permission denied during file operations",
			setup: func(t *testing.T) (string, func()) {
				tempDir, err := os.MkdirTemp("", "replbac-error-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create directory with restricted permissions
				restrictedDir := filepath.Join(tempDir, "restricted")
				err = os.Mkdir(restrictedDir, 0000) // No permissions
				if err != nil {
					t.Fatalf("Failed to create restricted dir: %v", err)
				}

				return restrictedDir, func() {
					_ = os.Chmod(restrictedDir, 0755) // #nosec G302 -- Restore permissions for test cleanup
					_ = os.RemoveAll(tempDir)
				}
			},
			config: models.Config{
				APIToken: "test-token",
			},
			expectError:        true,
			expectExitCode:     1,
			expectOutput:       []string{"Permission Error:", "Check directory permissions"},
			expectErrorMessage: "permission denied",
			expectUserGuidance: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			testDir, cleanup := tt.setup(t)
			defer cleanup()

			// Create enhanced sync command with or without mock client based on test type
			var cmd *cobra.Command
			if tt.name == "API connection failure - clear error message" || tt.name == "missing API token - configuration error with guidance" {
				// These tests need real error handling, not mocks
				cmd = CreateEnhancedSyncCommand(tt.config)
			} else {
				// Other tests use mock client to avoid real API calls
				mockCalls := &MockAPICalls{}
				mockClient := NewMockClient(mockCalls, []models.Role{})
				cmd = CreateEnhancedSyncCommandWithClient(mockClient)
			}
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Set flags
			for flag, value := range tt.flags {
				if err := cmd.Flags().Set(flag, value); err != nil {
					t.Fatalf("Failed to set flag %s: %v", flag, err)
				}
			}

			// Change to test directory if provided
			if testDir != "" {
				oldDir, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get current dir: %v", err)
				}
				defer func() {
					if err := os.Chdir(oldDir); err != nil {
						t.Errorf("Failed to restore directory: %v", err)
					}
				}()

				// Use testDir as argument instead of changing directory
				tt.args = []string{testDir}
			}

			// Execute command
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else {
					if tt.expectErrorMessage != "" && !strings.Contains(err.Error(), tt.expectErrorMessage) {
						t.Errorf("Expected error to contain '%s', got: %v", tt.expectErrorMessage, err)
					}
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output expectations (check both stdout and stderr)
			stdoutStr := stdout.String()
			stderrStr := stderr.String()
			combinedOutput := stdoutStr + stderrStr
			for _, expected := range tt.expectOutput {
				if !strings.Contains(combinedOutput, expected) {
					t.Errorf("Expected output to contain '%s', got:\nSTDOUT:\n%s\nSTDERR:\n%s", expected, stdoutStr, stderrStr)
				}
			}

			// Check for user guidance when expected
			if tt.expectUserGuidance {
				guidanceKeywords := []string{"help", "try", "check", "ensure", "verify", "documentation"}
				foundGuidance := false
				for _, keyword := range guidanceKeywords {
					if strings.Contains(strings.ToLower(combinedOutput), keyword) {
						foundGuidance = true
						break
					}
				}
				if !foundGuidance {
					t.Errorf("Expected user guidance in error message, got:\nSTDOUT:\n%s\nSTDERR:\n%s", stdoutStr, stderrStr)
				}
			}
		})
	}
}

// TestErrorRecovery tests error recovery and retry mechanisms
func TestErrorRecovery(t *testing.T) {
	tests := []struct {
		name           string
		scenario       string
		retryable      bool
		expectRetry    bool
		expectRecovery bool
	}{
		{
			name:           "network timeout - retryable",
			scenario:       "network_timeout",
			retryable:      true,
			expectRetry:    true,
			expectRecovery: true,
		},
		{
			name:           "API rate limit - retryable with backoff",
			scenario:       "rate_limit",
			retryable:      true,
			expectRetry:    true,
			expectRecovery: true,
		},
		{
			name:           "authentication failure - not retryable",
			scenario:       "auth_failure",
			retryable:      false,
			expectRetry:    false,
			expectRecovery: true,
		},
		{
			name:           "invalid role data - recoverable",
			scenario:       "invalid_data",
			retryable:      false,
			expectRetry:    false,
			expectRecovery: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test error categorization
			err := CreateScenarioError(tt.scenario)

			isRetryable := IsRetryableError(err)
			if isRetryable != tt.retryable {
				t.Errorf("Expected retryable=%v, got %v for scenario %s", tt.retryable, isRetryable, tt.scenario)
			}

			// Test recovery suggestions
			recovery := GetErrorRecovery(err)
			hasRecovery := recovery != ""
			if hasRecovery != tt.expectRecovery {
				t.Errorf("Expected recovery=%v, got %v for scenario %s", tt.expectRecovery, hasRecovery, tt.scenario)
			}
		})
	}
}

// TestUserFriendlyErrorMessages tests error message clarity and helpfulness
func TestUserFriendlyErrorMessages(t *testing.T) {
	tests := []struct {
		name             string
		error            error
		expectClear      bool
		expectActionable bool
		expectContext    bool
	}{
		{
			name:             "clear network error",
			error:            errors.New("failed to get remote roles: connection refused"),
			expectClear:      true,
			expectActionable: true,
			expectContext:    true,
		},
		{
			name:             "clear configuration error",
			error:            errors.New("invalid configuration: API token is required"),
			expectClear:      true,
			expectActionable: true,
			expectContext:    true,
		},
		{
			name:             "clear file permission error",
			error:            errors.New("failed to load local roles: permission denied"),
			expectClear:      true,
			expectActionable: true,
			expectContext:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enhanced := EnhanceErrorMessage(tt.error)

			// Check if error message is clear
			if tt.expectClear {
				if !strings.Contains(enhanced, "Error:") && !strings.Contains(enhanced, "Failed:") {
					t.Errorf("Expected clear error indication in: %s", enhanced)
				}
			}

			// Check if error message is actionable
			if tt.expectActionable {
				actionWords := []string{"check", "verify", "ensure", "try", "set", "configure"}
				foundAction := false
				for _, word := range actionWords {
					if strings.Contains(strings.ToLower(enhanced), word) {
						foundAction = true
						break
					}
				}
				if !foundAction {
					t.Errorf("Expected actionable guidance in: %s", enhanced)
				}
			}

			// Check if error provides context
			if tt.expectContext {
				if len(enhanced) <= len(tt.error.Error())+10 {
					t.Errorf("Expected enhanced context, got minimal enhancement: %s", enhanced)
				}
			}
		})
	}
}

// Helper functions are now implemented in errors.go
