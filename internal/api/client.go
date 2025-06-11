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

	"replbac/internal/logging"
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
	logger     *logging.Logger
}

// NewClient creates a new API client with the given base URL and API token
func NewClient(baseURL, apiToken string, logger *logging.Logger) (*Client, error) {
	// Validate base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fmt.Errorf("invalid base URL: must be a valid HTTP or HTTPS URL")
	}

	// Validate API token
	if strings.TrimSpace(apiToken) == "" {
		return nil, fmt.Errorf("API token is required")
	}

	logger.Debug("creating API client for endpoint: %s", baseURL)
	
	return &Client{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}, nil
}

// getPolicies is a helper method to fetch raw policy data from the API
func (c *Client) getPolicies() ([]models.Policy, error) {
	url := c.baseURL + "/vendor/v3/policies"
	c.logger.Debug("fetching policies from API endpoint: %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create HTTP request for %s: %v", url, err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed for %s: %v", url, err)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received HTTP response: status=%d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("API request failed: GET %s returned status %d", url, resp.StatusCode)
		return nil, c.handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read response body from %s: %v", url, err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.Debug("received response body of %d bytes", len(body))
	// Parse the response which has a "policies" wrapper
	var response struct {
		Policies []models.Policy `json:"policies"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		c.logger.Error("failed to parse JSON response from %s: %v", url, err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.Info("successfully fetched %d policies from API", len(response.Policies))
	return response.Policies, nil
}

// GetRoles retrieves all roles from the API
func (c *Client) GetRoles() ([]models.Role, error) {
	c.logger.Info("starting GetRoles operation")
	policies, err := c.getPolicies()
	if err != nil {
		c.logger.Error("GetRoles failed during policy fetch: %v", err)
		return nil, err
	}

	c.logger.Debug("converting %d policies to roles", len(policies))
	// Convert policies to local roles
	roles := make([]models.Role, 0, len(policies))
	for _, policy := range policies {
		c.logger.Debug("converting policy: %s", policy.Name)
		role, err := policy.ToRole()
		if err != nil {
			c.logger.Error("failed to convert policy %s to role: %v", policy.Name, err)
			return nil, fmt.Errorf("failed to convert policy %s: %w", policy.Name, err)
		}
		roles = append(roles, role)
	}

	c.logger.Info("successfully retrieved %d roles from API", len(roles))
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
	c.logger.Info("creating role: %s", role.Name)
	url := c.baseURL + "/vendor/v3/policy"
	c.logger.Debug("creating role at endpoint: %s", url)

	// Convert role to API format and create the policy structure
	c.logger.Debug("converting role %s to API format", role.Name)
	apiRole := role.ToAPIRole()
	definitionJSON, err := json.Marshal(apiRole)
	if err != nil {
		c.logger.Error("failed to marshal role definition for %s: %v", role.Name, err)
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
		c.logger.Error("HTTP request failed for CreateRole %s: %v", role.Name, err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("CreateRole response for %s: status=%d", role.Name, resp.StatusCode)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		c.logger.Error("CreateRole failed for %s: status=%d", role.Name, resp.StatusCode)
		return c.handleErrorResponse(resp)
	}

	c.logger.Info("successfully created role: %s", role.Name)
	return nil
}

// UpdateRole updates an existing role via the API
func (c *Client) UpdateRole(role models.Role) error {
	c.logger.Info("updating role: %s (ID: %s)", role.Name, role.ID)
	if role.ID == "" {
		c.logger.Error("UpdateRole failed for %s: missing role ID", role.Name)
		return fmt.Errorf("role ID is required for update operation")
	}
	
	url := c.baseURL + "/vendor/v3/policy/" + role.ID
	c.logger.Debug("updating role at endpoint: %s", url)

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
		c.logger.Error("HTTP request failed for UpdateRole %s: %v", role.Name, err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("UpdateRole response for %s: status=%d", role.Name, resp.StatusCode)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.logger.Error("UpdateRole failed for %s: status=%d", role.Name, resp.StatusCode)
		return c.handleErrorResponse(resp)
	}

	c.logger.Info("successfully updated role: %s", role.Name)
	return nil
}

// DeleteRole deletes a role by name via the API
func (c *Client) DeleteRole(roleName string) error {
	c.logger.Info("deleting role: %s", roleName)
	// Find the policy ID by name
	c.logger.Debug("looking up policy ID for role: %s", roleName)
	policies, err := c.getPolicies()
	if err != nil {
		c.logger.Error("DeleteRole failed for %s during policy lookup: %v", roleName, err)
		return fmt.Errorf("failed to fetch policies: %w", err)
	}
	
	var policyID string
	for _, policy := range policies {
		if policy.Name == roleName {
			policyID = policy.ID
			break
		}
	}
	
	if policyID == "" {
		c.logger.Warn("role not found for deletion: %s", roleName)
		return fmt.Errorf("role not found: %s", roleName)
	}
	
	c.logger.Debug("found policy ID %s for role %s", policyID, roleName)
	url := c.baseURL + "/vendor/v3/policy/" + policyID
	c.logger.Debug("deleting role at endpoint: %s", url)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed for DeleteRole %s: %v", roleName, err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("DeleteRole response for %s: status=%d", roleName, resp.StatusCode)
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		c.logger.Error("DeleteRole failed for %s: status=%d", roleName, resp.StatusCode)
		return c.handleErrorResponse(resp)
	}

	c.logger.Info("successfully deleted role: %s", roleName)
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