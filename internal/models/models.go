package models

// Resources represents the allowed and denied resources for a role
type Resources struct {
	Allowed []string `yaml:"allowed" json:"allowed"`
	Denied  []string `yaml:"denied" json:"denied"`
}

// Role represents a role as stored in local YAML files
type Role struct {
	Name      string    `yaml:"name" json:"name"`
	Resources Resources `yaml:"resources" json:"resources"`
}

// APIRole represents a role as expected by the Replicated API with v1 wrapper
type APIRole struct {
	V1 Role `json:"v1"`
}

// ToAPIRole converts a Role to an APIRole for API communication
func (r Role) ToAPIRole() APIRole {
	return APIRole{
		V1: r,
	}
}

// ToRole converts an APIRole to a Role for local processing
func (ar APIRole) ToRole() Role {
	return ar.V1
}

// Config represents the application configuration
type Config struct {
	APIEndpoint string `yaml:"api_endpoint" json:"api_endpoint"`
	APIToken    string `yaml:"api_token" json:"api_token"`
	Confirm     bool   `yaml:"confirm" json:"confirm"`
	LogLevel    string `yaml:"log_level" json:"log_level"`
}