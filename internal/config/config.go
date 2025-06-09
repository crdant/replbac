package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
		APIEndpoint: "https://api.replicated.com",
		LogLevel:    "info",
		Confirm:     false,
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

// loadFromFile loads configuration from a YAML or JSON file
func loadFromFile(configPath string) (models.Config, error) {
	var config models.Config

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return config, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return config, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return config, fmt.Errorf("unsupported config file format: %s", ext)
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv() models.Config {
	var config models.Config

	if val := os.Getenv("REPLBAC_API_ENDPOINT"); val != "" {
		config.APIEndpoint = val
	}
	if val := os.Getenv("REPLBAC_API_TOKEN"); val != "" {
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
	if source.APIEndpoint != "" {
		target.APIEndpoint = source.APIEndpoint
	}
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

	// Validate API endpoint is a valid URL
	u, err := url.Parse(config.APIEndpoint)
	if err != nil || u.Scheme == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("invalid API endpoint")
	}

	return nil
}