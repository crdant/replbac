package config

import (
	"os"
	"path/filepath"
	"testing"

	"replbac/internal/models"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		configFile     string
		configContent  string
		expectedConfig models.Config
		expectError    bool
	}{
		{
			name: "default config when no sources provided",
			expectedConfig: models.Config{
				APIEndpoint: "https://api.replicated.com",
				LogLevel:    "info",
				Confirm:     false,
			},
		},
		{
			name: "loads from environment variables",
			envVars: map[string]string{
				"REPLBAC_API_ENDPOINT": "https://custom.api.com",
				"REPLBAC_API_TOKEN":    "test-token",
				"REPLBAC_LOG_LEVEL":    "debug",
				"REPLBAC_CONFIRM":      "true",
			},
			expectedConfig: models.Config{
				APIEndpoint: "https://custom.api.com",
				APIToken:    "test-token",
				LogLevel:    "debug",
				Confirm:     true,
			},
		},
		{
			name:       "loads from YAML config file",
			configFile: "config.yaml",
			configContent: `api_endpoint: https://yaml.api.com
api_token: yaml-token
log_level: warn
confirm: true`,
			expectedConfig: models.Config{
				APIEndpoint: "https://yaml.api.com",
				APIToken:    "yaml-token",
				LogLevel:    "warn",
				Confirm:     true,
			},
		},
		{
			name:       "loads from JSON config file",
			configFile: "config.json",
			configContent: `{
  "api_endpoint": "https://json.api.com",
  "api_token": "json-token",
  "log_level": "error",
  "confirm": false
}`,
			expectedConfig: models.Config{
				APIEndpoint: "https://json.api.com",
				APIToken:    "json-token",
				LogLevel:    "error",
				Confirm:     false,
			},
		},
		{
			name: "environment variables override config file",
			envVars: map[string]string{
				"REPLBAC_API_TOKEN": "env-token",
				"REPLBAC_LOG_LEVEL": "debug",
			},
			configFile: "config.yaml",
			configContent: `api_endpoint: https://yaml.api.com
api_token: yaml-token
log_level: info
confirm: true`,
			expectedConfig: models.Config{
				APIEndpoint: "https://yaml.api.com",
				APIToken:    "env-token",
				LogLevel:    "debug",
				Confirm:     true,
			},
		},
		{
			name:       "invalid YAML returns error",
			configFile: "config.yaml",
			configContent: `api_endpoint: https://yaml.api.com
  invalid: yaml: content`,
			expectError: true,
		},
		{
			name:       "invalid JSON returns error",
			configFile: "config.json",
			configContent: `{
  "api_endpoint": "https://json.api.com",
  "invalid": json,
}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			defer cleanupEnv()

			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			var configPath string
			if tt.configFile != "" {
				// Create temporary config file
				tmpDir := t.TempDir()
				configPath = filepath.Join(tmpDir, tt.configFile)
				if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}
			}

			config, err := LoadConfig(configPath)

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

			if config.APIEndpoint != tt.expectedConfig.APIEndpoint {
				t.Errorf("APIEndpoint = %v, want %v", config.APIEndpoint, tt.expectedConfig.APIEndpoint)
			}
			if config.APIToken != tt.expectedConfig.APIToken {
				t.Errorf("APIToken = %v, want %v", config.APIToken, tt.expectedConfig.APIToken)
			}
			if config.LogLevel != tt.expectedConfig.LogLevel {
				t.Errorf("LogLevel = %v, want %v", config.LogLevel, tt.expectedConfig.LogLevel)
			}
			if config.Confirm != tt.expectedConfig.Confirm {
				t.Errorf("Confirm = %v, want %v", config.Confirm, tt.expectedConfig.Confirm)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      models.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: models.Config{
				APIEndpoint: "https://api.replicated.com",
				APIToken:    "valid-token",
				LogLevel:    "info",
			},
		},
		{
			name: "missing API token",
			config: models.Config{
				APIEndpoint: "https://api.replicated.com",
				LogLevel:    "info",
			},
			expectError: true,
			errorMsg:    "API token is required",
		},
		{
			name: "invalid log level",
			config: models.Config{
				APIEndpoint: "https://api.replicated.com",
				APIToken:    "valid-token",
				LogLevel:    "invalid",
			},
			expectError: true,
			errorMsg:    "invalid log level",
		},
		{
			name: "invalid API endpoint",
			config: models.Config{
				APIEndpoint: "not-a-url",
				APIToken:    "valid-token",
				LogLevel:    "info",
			},
			expectError: true,
			errorMsg:    "invalid API endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

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

func cleanupEnv() {
	envVars := []string{
		"REPLBAC_API_ENDPOINT",
		"REPLBAC_API_TOKEN",
		"REPLBAC_LOG_LEVEL",
		"REPLBAC_CONFIRM",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}