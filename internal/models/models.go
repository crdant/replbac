package models

import (
	"encoding/json"
	"fmt"
)

// Constants for hardcoded values
const (
	// ReplicatedAPIEndpoint is the hardcoded Replicated API endpoint
	ReplicatedAPIEndpoint = "https://api.replicated.com"
)

// Resources represents the allowed and denied resources for a role
type Resources struct {
	Allowed []string `yaml:"allowed" json:"allowed"`
	Denied  []string `yaml:"denied" json:"denied"`
}

// Role represents a role as stored in local YAML files
type Role struct {
	ID        string    `yaml:"id,omitempty" json:"id,omitempty"`
	Name      string    `yaml:"name" json:"name"`
	Resources Resources `yaml:"resources" json:"resources"`
	Members   []string  `yaml:"members,omitempty" json:"members,omitempty"`
}

// APIRole represents a role as expected by the Replicated API with v1 wrapper
type APIRole struct {
	V1 Role `json:"v1"`
}

// Policy represents a full policy object from the Replicated API
type Policy struct {
	ID          string  `json:"id"`
	TeamID      string  `json:"teamId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Definition  string  `json:"definition"` // JSON string containing APIRole
	CreatedAt   string  `json:"createdAt"`
	ModifiedAt  *string `json:"modifiedAt"`
	ReadOnly    bool    `json:"readOnly"`
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

// ToRole converts a Policy to a Role by parsing the definition JSON
func (p Policy) ToRole() (Role, error) {
	var apiRole APIRole
	if err := json.Unmarshal([]byte(p.Definition), &apiRole); err != nil {
		return Role{}, fmt.Errorf("failed to parse policy definition: %w", err)
	}
	role := apiRole.ToRole()
	// Use the actual policy name and ID instead of values from the definition
	role.ID = p.ID
	role.Name = p.Name
	return role, nil
}

// TeamMember represents a team member from the Replicated API
type TeamMember struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name,omitempty"`
	Username     string `json:"username,omitempty"`
	PolicyID     string `json:"policyId,omitempty"`
	CreatedAt    string `json:"createdAt,omitempty"`
	LastActiveAt string `json:"lastActiveAt,omitempty"`
	Status       string `json:"status,omitempty"`   // "active", "pending", etc.
	InviteID     string `json:"inviteId,omitempty"` // Present for pending invites
}

// IsPendingInvite returns true if this team member is actually a pending invite
func (tm TeamMember) IsPendingInvite() bool {
	// Check explicit pending status fields
	if tm.Status == "pending" || tm.InviteID != "" {
		return true
	}

	// Heuristic: if the member ID equals the email address, it's likely a pending invite
	// This handles cases where the API doesn't set explicit pending status fields
	if tm.ID == tm.Email {
		return true
	}

	return false
}

// InviteUserRequest represents the request payload for inviting a user
type InviteUserRequest struct {
	Email    string `json:"email"`
	PolicyID string `json:"policy_id"`
}

// InviteUserResponse represents the response from inviting a user
type InviteUserResponse struct {
	ID       string `json:"id,omitempty"`
	Email    string `json:"email"`
	PolicyID string `json:"policy_id,omitempty"`
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
}

// Config represents the application configuration
type Config struct {
	APIToken string `yaml:"api_token" json:"api_token"`
	Confirm  bool   `yaml:"confirm" json:"confirm"`
	LogLevel string `yaml:"log_level" json:"log_level"`
}
