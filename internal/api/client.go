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

// GetRoles retrieves all roles from the API
func (c *Client) GetRoles() ([]models.Role, error) {
	url := c.baseURL + "/v1/team/policies"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
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

	var apiRoles []models.APIRole
	if err := json.Unmarshal(body, &apiRoles); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert API roles to local roles
	roles := make([]models.Role, len(apiRoles))
	for i, apiRole := range apiRoles {
		roles[i] = apiRole.ToRole()
	}

	return roles, nil
}

// GetRole retrieves a specific role by name from the API
func (c *Client) GetRole(roleName string) (models.Role, error) {
	url := c.baseURL + "/v1/team/policies/" + roleName

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return models.Role{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return models.Role{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Role{}, c.handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Role{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiRole models.APIRole
	if err := json.Unmarshal(body, &apiRole); err != nil {
		return models.Role{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return apiRole.ToRole(), nil
}

// CreateRole creates a new role via the API
func (c *Client) CreateRole(role models.Role) error {
	url := c.baseURL + "/v1/team/policies"

	// Convert role to API format
	apiRole := role.ToAPIRole()

	body, err := json.Marshal(apiRole)
	if err != nil {
		return fmt.Errorf("failed to marshal role: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
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
	url := c.baseURL + "/v1/team/policies/" + role.Name

	// Convert role to API format
	apiRole := role.ToAPIRole()

	body, err := json.Marshal(apiRole)
	if err != nil {
		return fmt.Errorf("failed to marshal role: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
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
	url := c.baseURL + "/v1/team/policies/" + roleName

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
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