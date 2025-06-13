package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"replbac/internal/models"
)

var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

// LoadConfig loads configuration from multiple sources with proper precedence:
// 1. Environment variables (highest priority)
// 2. Configuration file
// 3. Default values (lowest priority)
func LoadConfig(configPath string) (models.Config, error) {
	// Start with default configuration
	config := models.Config{
		LogLevel: "info",
		Confirm:  false,
	}

	// Load from config file if provided
	if configPath != "" {
		fileConfig, err := loadFromFile(configPath)
		if err != nil {
			return models.Config{}, fmt.Errorf("failed to load config file: %w", err)
		}
		mergeConfigs(&config, &fileConfig)
	}

	// Load from environment variables (highest priority)
	envConfig := loadFromEnv()
	mergeConfigs(&config, &envConfig)

	return config, nil
}

// GetDefaultConfigPaths returns platform-specific default configuration file paths
func GetDefaultConfigPaths() []string {
	var paths []string

	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Preferences/com.replicated.replbac/
		if home, err := os.UserHomeDir(); err == nil {
			paths = append(paths, filepath.Join(home, "Library", "Preferences", "com.replicated.replbac", "config.yaml"))

			// Also check .config as fallback
			paths = append(paths, filepath.Join(home, ".config", "replbac", "config.yaml"))
		}

	case "linux":
		// Linux: XDG_CONFIG_HOME or $HOME/.config
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			if home, err := os.UserHomeDir(); err == nil {
				configDir = filepath.Join(home, ".config")
			}
		}

		if configDir != "" {
			paths = append(paths, filepath.Join(configDir, "replbac", "config.yaml"))
		}

	default:
		// Windows and other platforms: use home directory
		if home, err := os.UserHomeDir(); err == nil {
			paths = append(paths, filepath.Join(home, ".replbac", "config.yaml"))
		}
	}

	return paths
}

// LoadConfigWithDefaults loads configuration from multiple sources, checking default paths if no explicit path provided
func LoadConfigWithDefaults(defaultPaths []string) (models.Config, error) {
	// Start with default configuration
	config := models.Config{
		LogLevel: "info",
		Confirm:  false,
	}

	// Check if config path is specified via environment variable
	if configPath := os.Getenv("REPLBAC_CONFIG"); configPath != "" {
		fileConfig, err := loadFromFile(configPath)
		if err != nil {
			return models.Config{}, fmt.Errorf("failed to load config from REPLBAC_CONFIG path: %w", err)
		}
		mergeConfigs(&config, &fileConfig)
	} else {
		// Try to load from default paths
		if len(defaultPaths) == 0 {
			defaultPaths = GetDefaultConfigPaths()
		}

		for _, configPath := range defaultPaths {
			if _, err := os.Stat(configPath); err == nil {
				// File exists, try to load it
				fileConfig, err := loadFromFile(configPath)
				if err != nil {
					// Log error but continue to next path
					continue
				}
				mergeConfigs(&config, &fileConfig)
				break // Use first found config file
			}
		}
	}

	// Load from environment variables (highest priority)
	envConfig := loadFromEnv()
	mergeConfigs(&config, &envConfig)

	return config, nil
}

// loadFromFile loads configuration from a YAML or JSON file
func loadFromFile(configPath string) (models.Config, error) {
	var config models.Config

	data, err := os.ReadFile(configPath) // #nosec G304 -- Reading user-provided config file is expected behavior
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return config, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return config, fmt.Errorf("unsupported config file format: %s (only YAML is supported)", ext)
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv() models.Config {
	var config models.Config

	// Check REPLICATED_API_TOKEN first (for compatibility with replicated CLI)
	if val := os.Getenv("REPLICATED_API_TOKEN"); val != "" {
		config.APIToken = val
	} else if val := os.Getenv("REPLBAC_API_TOKEN"); val != "" {
		config.APIToken = val
	}
	if val := os.Getenv("REPLBAC_LOG_LEVEL"); val != "" {
		config.LogLevel = val
	}
	if val := os.Getenv("REPLBAC_CONFIRM"); val != "" {
		if confirm, err := strconv.ParseBool(val); err == nil {
			config.Confirm = confirm
		}
	}

	return config
}

// mergeConfigs merges source config into target, only overriding non-zero values
func mergeConfigs(target, source *models.Config) {
	if source.APIToken != "" {
		target.APIToken = source.APIToken
	}
	if source.LogLevel != "" {
		target.LogLevel = source.LogLevel
	}
	// For boolean fields, we can't distinguish between false and zero value,
	// so we'll use a simple assignment for now
	if source.Confirm {
		target.Confirm = source.Confirm
	}
}

// ValidateConfig validates the configuration and returns an error if invalid
func ValidateConfig(config models.Config) error {
	// API token is required
	if config.APIToken == "" {
		return errors.New("API token is required")
	}

	// Validate log level
	if !validLogLevels[config.LogLevel] {
		return errors.New("invalid log level")
	}

	// Note: API endpoint is now hardcoded to models.ReplicatedAPIEndpoint

	return nil
}
