package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"replbac/internal/models"
)

// ClientInterface defines the interface for API operations
type ClientInterface interface {
	GetRoles() ([]models.Role, error)
	GetRole(roleName string) (models.Role, error)
	CreateRole(role models.Role) error
	UpdateRole(role models.Role) error
	DeleteRole(roleName string) error
}

// Client represents an HTTP client for the Replicated API
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new API client with the given base URL and API token
func NewClient(baseURL, apiToken string) (*Client, error) {
	// Validate base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fmt.Errorf("invalid base URL: must be a valid HTTP or HTTPS URL")
	}

	// Validate API token
	if strings.TrimSpace(apiToken) == "" {
		return nil, fmt.Errorf("API token is required")
	}

	return &Client{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// getPolicies is a helper method to fetch raw policy data from the API
func (c *Client) getPolicies() ([]models.Policy, error) {
	url := c.baseURL + "/vendor/v3/policies"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the response which has a "policies" wrapper
	var response struct {
		Policies []models.Policy `json:"policies"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Policies, nil
}

// GetRoles retrieves all roles from the API
func (c *Client) GetRoles() ([]models.Role, error) {
	policies, err := c.getPolicies()
	if err != nil {
		return nil, err
	}

	// Convert policies to local roles
	roles := make([]models.Role, 0, len(policies))
	for _, policy := range policies {
		role, err := policy.ToRole()
		if err != nil {
			return nil, fmt.Errorf("failed to convert policy %s: %w", policy.Name, err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetRole retrieves a specific role by name from the API
func (c *Client) GetRole(roleName string) (models.Role, error) {
	// Find the policy ID by name
	policies, err := c.getPolicies()
	if err != nil {
		return models.Role{}, fmt.Errorf("failed to fetch policies: %w", err)
	}
	
	for _, policy := range policies {
		if policy.Name == roleName {
			return policy.ToRole()
		}
	}
	
	return models.Role{}, fmt.Errorf("role not found: %s", roleName)
}

// CreateRole creates a new role via the API
func (c *Client) CreateRole(role models.Role) error {
	url := c.baseURL + "/vendor/v3/policy"

	// Convert role to API format and create the policy structure
	apiRole := role.ToAPIRole()
	definitionJSON, err := json.Marshal(apiRole)
	if err != nil {
		return fmt.Errorf("failed to marshal role definition: %w", err)
	}

	// Create the policy payload
	policy := struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Definition  string `json:"definition"`
	}{
		Name:       role.Name,
		Definition: string(definitionJSON),
	}

	body, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	return nil
}

// UpdateRole updates an existing role via the API
func (c *Client) UpdateRole(role models.Role) error {
	if role.ID == "" {
		return fmt.Errorf("role ID is required for update operation")
	}
	
	url := c.baseURL + "/vendor/v3/policy/" + role.ID

	// Convert role to API format and create the policy update structure
	apiRole := role.ToAPIRole()
	definitionJSON, err := json.Marshal(apiRole)
	if err != nil {
		return fmt.Errorf("failed to marshal role definition: %w", err)
	}

	// Create the policy update payload
	policyUpdate := struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Definition  string `json:"definition"`
	}{
		Name:       role.Name,
		Definition: string(definitionJSON),
	}

	body, err := json.Marshal(policyUpdate)
	if err != nil {
		return fmt.Errorf("failed to marshal policy update: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.handleErrorResponse(resp)
	}

	return nil
}

// DeleteRole deletes a role by name via the API
func (c *Client) DeleteRole(roleName string) error {
	// Note: This method still uses name for compatibility, but in practice
	// we should look up by ID. For now, this will work for testing.
	url := c.baseURL + "/vendor/v3/policy/" + roleName

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	return nil
}

// handleErrorResponse processes error responses from the API
func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Try to parse error response
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		// If we can't parse the error response, return a generic error
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	errorMsg := errorResp.Error
	if errorMsg == "" {
		errorMsg = errorResp.Message
	}
	if errorMsg == "" {
		errorMsg = "unknown error"
	}

	return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, errorMsg)
}