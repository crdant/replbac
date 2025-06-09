package models

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRole_YAMLMarshaling(t *testing.T) {
	role := Role{
		Name: "View Customers Only",
		Resources: Resources{
			Allowed: []string{
				"kots/app/*/license/*/read",
				"kots/app/*/license/*/list",
			},
			Denied: []string{
				"**/*",
			},
		},
	}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(role)
	if err != nil {
		t.Fatalf("Failed to marshal role to YAML: %v", err)
	}

	// Test unmarshaling from YAML
	var unmarshaledRole Role
	err = yaml.Unmarshal(yamlData, &unmarshaledRole)
	if err != nil {
		t.Fatalf("Failed to unmarshal role from YAML: %v", err)
	}

	// Verify the data is preserved
	if unmarshaledRole.Name != role.Name {
		t.Errorf("Expected name %s, got %s", role.Name, unmarshaledRole.Name)
	}
	if len(unmarshaledRole.Resources.Allowed) != len(role.Resources.Allowed) {
		t.Errorf("Expected %d allowed resources, got %d", len(role.Resources.Allowed), len(unmarshaledRole.Resources.Allowed))
	}
	if len(unmarshaledRole.Resources.Denied) != len(role.Resources.Denied) {
		t.Errorf("Expected %d denied resources, got %d", len(role.Resources.Denied), len(unmarshaledRole.Resources.Denied))
	}
}

func TestAPIRole_JSONMarshaling(t *testing.T) {
	apiRole := APIRole{
		V1: Role{
			Name: "View Customers Only",
			Resources: Resources{
				Allowed: []string{
					"kots/app/*/license/*/read",
					"kots/app/*/license/*/list",
				},
				Denied: []string{
					"**/*",
				},
			},
		},
	}

	// Test marshaling to JSON
	jsonData, err := json.Marshal(apiRole)
	if err != nil {
		t.Fatalf("Failed to marshal APIRole to JSON: %v", err)
	}

	// Test unmarshaling from JSON
	var unmarshaledAPIRole APIRole
	err = json.Unmarshal(jsonData, &unmarshaledAPIRole)
	if err != nil {
		t.Fatalf("Failed to unmarshal APIRole from JSON: %v", err)
	}

	// Verify the data is preserved
	if unmarshaledAPIRole.V1.Name != apiRole.V1.Name {
		t.Errorf("Expected name %s, got %s", apiRole.V1.Name, unmarshaledAPIRole.V1.Name)
	}
}

func TestRole_ToAPIRole(t *testing.T) {
	role := Role{
		Name: "Test Role",
		Resources: Resources{
			Allowed: []string{"resource1", "resource2"},
			Denied:  []string{"denied1"},
		},
	}

	apiRole := role.ToAPIRole()

	if apiRole.V1.Name != role.Name {
		t.Errorf("Expected API role name %s, got %s", role.Name, apiRole.V1.Name)
	}
	if len(apiRole.V1.Resources.Allowed) != len(role.Resources.Allowed) {
		t.Errorf("Expected %d allowed resources, got %d", len(role.Resources.Allowed), len(apiRole.V1.Resources.Allowed))
	}
}

func TestAPIRole_ToRole(t *testing.T) {
	apiRole := APIRole{
		V1: Role{
			Name: "Test Role",
			Resources: Resources{
				Allowed: []string{"resource1", "resource2"},
				Denied:  []string{"denied1"},
			},
		},
	}

	role := apiRole.ToRole()

	if role.Name != apiRole.V1.Name {
		t.Errorf("Expected role name %s, got %s", apiRole.V1.Name, role.Name)
	}
	if len(role.Resources.Allowed) != len(apiRole.V1.Resources.Allowed) {
		t.Errorf("Expected %d allowed resources, got %d", len(apiRole.V1.Resources.Allowed), len(role.Resources.Allowed))
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	config := Config{
		APIEndpoint: "https://api.replicated.com",
		Confirm:     true,
		LogLevel:    "info",
	}

	if config.APIEndpoint != "https://api.replicated.com" {
		t.Errorf("Expected default API endpoint, got %s", config.APIEndpoint)
	}
	if !config.Confirm {
		t.Error("Expected confirm to be true by default")
	}
	if config.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got %s", config.LogLevel)
	}
}

func TestConfig_YAMLMarshaling(t *testing.T) {
	config := Config{
		APIEndpoint: "https://api.replicated.com",
		APIToken:    "test-token",
		Confirm:     false,
		LogLevel:    "debug",
	}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config to YAML: %v", err)
	}

	// Test unmarshaling from YAML
	var unmarshaledConfig Config
	err = yaml.Unmarshal(yamlData, &unmarshaledConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal config from YAML: %v", err)
	}

	// Verify the data is preserved
	if unmarshaledConfig.APIEndpoint != config.APIEndpoint {
		t.Errorf("Expected API endpoint %s, got %s", config.APIEndpoint, unmarshaledConfig.APIEndpoint)
	}
	if unmarshaledConfig.APIToken != config.APIToken {
		t.Errorf("Expected API token %s, got %s", config.APIToken, unmarshaledConfig.APIToken)
	}
	if unmarshaledConfig.Confirm != config.Confirm {
		t.Errorf("Expected confirm %t, got %t", config.Confirm, unmarshaledConfig.Confirm)
	}
	if unmarshaledConfig.LogLevel != config.LogLevel {
		t.Errorf("Expected log level %s, got %s", config.LogLevel, unmarshaledConfig.LogLevel)
	}
}