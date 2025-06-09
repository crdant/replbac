package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
			name: "REPLICATED_API_TOKEN takes precedence over REPLBAC_API_TOKEN",
			envVars: map[string]string{
				"REPLICATED_API_TOKEN": "replicated-token",
				"REPLBAC_API_TOKEN":    "replbac-token",
				"REPLBAC_API_ENDPOINT": "https://test.api.com",
			},
			expectedConfig: models.Config{
				APIEndpoint: "https://test.api.com",
				APIToken:    "replicated-token",
				LogLevel:    "info",
				Confirm:     false,
			},
		},
		{
			name: "REPLBAC_API_TOKEN used when REPLICATED_API_TOKEN not set",
			envVars: map[string]string{
				"REPLBAC_API_TOKEN":    "replbac-token",
				"REPLBAC_API_ENDPOINT": "https://test.api.com",
			},
			expectedConfig: models.Config{
				APIEndpoint: "https://test.api.com",
				APIToken:    "replbac-token",
				LogLevel:    "info",
				Confirm:     false,
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
			name:       "unsupported file format returns error",
			configFile: "config.json",
			configContent: `{
  "api_endpoint": "https://json.api.com"
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

func TestGetDefaultConfigPaths(t *testing.T) {
	paths := GetDefaultConfigPaths()
	
	if len(paths) == 0 {
		t.Error("Expected at least one default config path")
	}
	
	// Check platform-specific behavior
	switch runtime.GOOS {
	case "darwin":
		found := false
		for _, path := range paths {
			if strings.Contains(path, "Library/Preferences/com.replicated.replbac") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected macOS default path not found")
		}
	case "linux":
		// Check for XDG or HOME fallback
		found := false
		for _, path := range paths {
			if strings.Contains(path, ".config/replbac") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected Linux default path not found")
		}
	}
	
	// All paths should be absolute
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			t.Errorf("Path should be absolute: %s", path)
		}
	}
}

func TestLoadConfigWithDefaultPaths(t *testing.T) {
	// Clean up environment
	defer cleanupEnv()
	
	// Create a temporary config file in a known location
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `api_endpoint: https://test.api.com
api_token: test-token
log_level: debug`
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Test LoadConfigWithDefaults with our test path
	config, err := LoadConfigWithDefaults([]string{configPath})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	
	if config.APIEndpoint != "https://test.api.com" {
		t.Errorf("APIEndpoint = %v, want %v", config.APIEndpoint, "https://test.api.com")
	}
	if config.APIToken != "test-token" {
		t.Errorf("APIToken = %v, want %v", config.APIToken, "test-token")
	}
	if config.LogLevel != "debug" {
		t.Errorf("LogLevel = %v, want %v", config.LogLevel, "debug")
	}
}

func TestLoadConfigWithEnvironmentConfigPath(t *testing.T) {
	// Clean up environment
	defer cleanupEnv()
	
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom-config.yaml")
	configContent := `api_endpoint: https://custom.api.com
api_token: custom-token
log_level: warn`
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Set REPLBAC_CONFIG environment variable
	os.Setenv("REPLBAC_CONFIG", configPath)
	
	// Test LoadConfigWithDefaults - it should use REPLBAC_CONFIG path
	config, err := LoadConfigWithDefaults(nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	
	if config.APIEndpoint != "https://custom.api.com" {
		t.Errorf("APIEndpoint = %v, want %v", config.APIEndpoint, "https://custom.api.com")
	}
	if config.APIToken != "custom-token" {
		t.Errorf("APIToken = %v, want %v", config.APIToken, "custom-token")
	}
	if config.LogLevel != "warn" {
		t.Errorf("LogLevel = %v, want %v", config.LogLevel, "warn")
	}
}

func TestLoadConfigEnvironmentOverridesConfigFile(t *testing.T) {
	// Clean up environment
	defer cleanupEnv()
	
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `api_endpoint: https://file.api.com
api_token: file-token
log_level: info`
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Set both REPLBAC_CONFIG and override env vars
	os.Setenv("REPLBAC_CONFIG", configPath)
	os.Setenv("REPLBAC_API_TOKEN", "env-token")
	os.Setenv("REPLBAC_LOG_LEVEL", "debug")
	
	config, err := LoadConfigWithDefaults(nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	
	// Environment variables should override config file values
	if config.APIEndpoint != "https://file.api.com" {
		t.Errorf("APIEndpoint = %v, want %v", config.APIEndpoint, "https://file.api.com")
	}
	if config.APIToken != "env-token" {
		t.Errorf("APIToken = %v, want %v (env should override)", config.APIToken, "env-token")
	}
	if config.LogLevel != "debug" {
		t.Errorf("LogLevel = %v, want %v (env should override)", config.LogLevel, "debug")
	}
}

func cleanupEnv() {
	envVars := []string{
		"REPLBAC_API_ENDPOINT",
		"REPLBAC_API_TOKEN",
		"REPLICATED_API_TOKEN",
		"REPLBAC_LOG_LEVEL",
		"REPLBAC_CONFIRM",
		"REPLBAC_CONFIG",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}