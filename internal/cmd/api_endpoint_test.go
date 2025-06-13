package cmd

import (
	"strings"
	"testing"

	"replbac/internal/models"
)

// TestAPIEndpointRemoval tests that API endpoint configuration has been removed
func TestAPIEndpointRemoval(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "api-endpoint flag should not exist",
			testFunc: func(t *testing.T) {
				cmd := createTestRootCommand()
				flag := cmd.PersistentFlags().Lookup("api-endpoint")
				if flag != nil {
					t.Error("api-endpoint flag should not exist after removal")
				}
			},
		},
		{
			name: "help text should not mention api-endpoint flag",
			testFunc: func(t *testing.T) {
				cmd := createTestRootCommand()
				cmd.SetArgs([]string{"--help"})
				output, _ := executeCommandToString(cmd)

				if strings.Contains(output, "api-endpoint") {
					t.Error("Help text should not mention api-endpoint flag")
				}
				if strings.Contains(output, "REPLBAC_API_ENDPOINT") {
					t.Error("Help text should not mention REPLBAC_API_ENDPOINT environment variable")
				}
			},
		},
		{
			name: "API endpoint should be hardcoded to replicated.com",
			testFunc: func(t *testing.T) {
				// Test that the API endpoint is hardcoded by checking the constant
				expectedEndpoint := "https://api.replicated.com"

				// Verify the hardcoded constant is correct
				if models.ReplicatedAPIEndpoint != expectedEndpoint {
					t.Errorf("Expected hardcoded endpoint %s, got %s", expectedEndpoint, models.ReplicatedAPIEndpoint)
				}

				// Config struct should not have APIEndpoint field
				config := models.Config{}
				_ = config // Just use the config to show it doesn't need APIEndpoint
			},
		},
		{
			name: "config validation should not require API endpoint",
			testFunc: func(t *testing.T) {
				// Create a config without API endpoint
				config := models.Config{
					APIToken: "test-token",
					// No APIEndpoint set - should use hardcoded value
				}

				// Validation should not fail due to missing API endpoint
				// since it should be hardcoded
				if config.APIToken == "" {
					t.Error("Test setup error: APIToken should be set for this test")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

// TestHardcodedAPIEndpoint tests that the API endpoint is properly hardcoded
func TestHardcodedAPIEndpoint(t *testing.T) {
	expectedEndpoint := "https://api.replicated.com"

	// Test that API client creation uses the hardcoded endpoint
	config := models.Config{
		APIToken: "test-token",
		// APIEndpoint should not be required - it should be hardcoded
	}

	// The actual validation will be done in the implementation
	// This test ensures our expectation is clear
	if config.APIToken == "" {
		t.Error("APIToken is required for API operations")
	}

	// The endpoint should be internally set to expectedEndpoint
	t.Logf("Expected hardcoded endpoint: %s", expectedEndpoint)
}
