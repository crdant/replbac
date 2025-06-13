package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
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
	GetRolesWithContext(ctx context.Context) ([]models.Role, error)
	GetRole(roleName string) (models.Role, error)
	GetRoleWithContext(ctx context.Context, roleName string) (models.Role, error)
	CreateRole(role models.Role) error
	CreateRoleWithContext(ctx context.Context, role models.Role) error
	UpdateRole(role models.Role) error
	UpdateRoleWithContext(ctx context.Context, role models.Role) error
	DeleteRole(roleName string) error
	DeleteRoleWithContext(ctx context.Context, roleName string) error
	GetTeamMembers() ([]models.TeamMember, error)
	GetTeamMembersWithContext(ctx context.Context) ([]models.TeamMember, error)
	AssignMemberRole(memberEmail, roleID string) error
	AssignMemberRoleWithContext(ctx context.Context, memberEmail, roleID string) error
}

// Client represents an HTTP client for the Replicated API
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
	logger     *logging.Logger
	maxRetries int
}

// NewClient creates a new API client with the given base URL and API token
func NewClient(baseURL, apiToken string, logger *logging.Logger) (*Client, error) {
	return NewClientWithRetry(baseURL, apiToken, logger, 3)
}

// NewClientWithRetry creates a new API client with configurable retry logic
func NewClientWithRetry(baseURL, apiToken string, logger *logging.Logger, maxRetries int) (*Client, error) {
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
		logger:     logger,
		maxRetries: maxRetries,
	}, nil
}

// executeWithRetry performs HTTP requests with exponential backoff retry logic
func (c *Client) executeWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply exponential backoff delay (but not on first attempt)
		if attempt > 0 {
			backoffDuration := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			c.logger.Debug("retrying request after %v delay (attempt %d/%d)", backoffDuration, attempt+1, c.maxRetries+1)

			select {
			case <-time.After(backoffDuration):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Clone request for retry (can't reuse request body)
		reqClone := req.Clone(ctx)

		resp, err := c.httpClient.Do(reqClone)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			c.logger.Warn("request attempt %d failed: %v", attempt+1, err)
			continue
		}

		// Check if we should retry based on status code
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: HTTP %d", resp.StatusCode)
			c.logger.Warn("request attempt %d failed with server error: HTTP %d", attempt+1, resp.StatusCode)
			continue
		}

		// Success or client error (don't retry client errors)
		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// getPolicies is a helper method to fetch raw policy data from the API
func (c *Client) getPolicies() ([]models.Policy, error) {
	return c.getPoliciesWithContext(context.Background())
}

// getPoliciesWithContext is a helper method to fetch raw policy data from the API with context
func (c *Client) getPoliciesWithContext(ctx context.Context) ([]models.Policy, error) {
	url := c.baseURL + "/vendor/v3/policies"
	c.logger.Debug("fetching policies from API endpoint: %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create HTTP request for %s: %v", url, err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.executeWithRetry(ctx, req)
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

	c.logger.Debug("successfully fetched %d policies from API", len(response.Policies))
	return response.Policies, nil
}

// GetRoles retrieves all roles from the API
func (c *Client) GetRoles() ([]models.Role, error) {
	c.logger.Debug("starting GetRoles operation")
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

	c.logger.Debug("fetching team members to correlate with roles")
	// Fetch team members and correlate with roles
	members, err := c.GetTeamMembers()
	if err != nil {
		c.logger.Warn("failed to fetch team members (roles will not include member data): %v", err)
		// Continue without member data rather than failing completely
	} else {
		c.logger.Debug("correlating %d members with roles", len(members))
		// Group members by policy ID
		membersByPolicy := make(map[string][]string)
		for _, member := range members {
			if member.PolicyID != "" {
				// Use the member ID (email) for the members list
				membersByPolicy[member.PolicyID] = append(membersByPolicy[member.PolicyID], member.ID)
			}
		}

		// Populate member data for each role
		for i := range roles {
			// Find members for this role by policy ID
			policyID := policies[i].ID
			if memberEmails, found := membersByPolicy[policyID]; found {
				roles[i].Members = memberEmails
				c.logger.Debug("role %s has %d members", roles[i].Name, len(memberEmails))
			}
		}
	}

	c.logger.Debug("successfully retrieved %d roles from API", len(roles))
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

// Context-aware API methods

// GetRolesWithContext fetches all roles from the API with context support
func (c *Client) GetRolesWithContext(ctx context.Context) ([]models.Role, error) {
	c.logger.Info("fetching all roles from API")
	start := time.Now()
	defer func() {
		c.logger.Debug("GetRoles completed in %v", time.Since(start))
	}()

	policies, err := c.getPoliciesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	roles := make([]models.Role, len(policies))
	for i, policy := range policies {
		roles[i] = models.Role{
			Name: policy.Name,
			ID:   policy.ID,
		}
	}

	c.logger.Info("successfully fetched %d roles from API", len(roles))
	return roles, nil
}

// GetRoleWithContext fetches a specific role by name from the API with context support
func (c *Client) GetRoleWithContext(ctx context.Context, roleName string) (models.Role, error) {
	c.logger.Info("fetching role '%s' from API", roleName)

	roles, err := c.GetRolesWithContext(ctx)
	if err != nil {
		return models.Role{}, err
	}

	for _, role := range roles {
		if role.Name == roleName {
			c.logger.Info("successfully found role '%s'", roleName)
			return role, nil
		}
	}

	return models.Role{}, fmt.Errorf("role '%s' not found", roleName)
}

// CreateRoleWithContext creates a new role via the API with context support
func (c *Client) CreateRoleWithContext(ctx context.Context, role models.Role) error {
	c.logger.Info("creating role '%s' via API", role.Name)
	start := time.Now()
	defer func() {
		c.logger.Debug("CreateRole for '%s' completed in %v", role.Name, time.Since(start))
	}()

	url := c.baseURL + "/vendor/v3/policies"
	c.logger.Debug("creating role at endpoint: %s", url)

	requestData := struct {
		Name string `json:"name"`
	}{
		Name: role.Name,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		c.logger.Error("failed to marshal role data for '%s': %v", role.Name, err)
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		c.logger.Error("failed to create HTTP request for creating role '%s': %v", role.Name, err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.executeWithRetry(ctx, req)
	if err != nil {
		c.logger.Error("HTTP request failed for creating role '%s': %v", role.Name, err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received HTTP response for creating role '%s': status=%d", role.Name, resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		c.logger.Error("failed to create role '%s': API returned status %d", role.Name, resp.StatusCode)
		return c.handleErrorResponse(resp)
	}

	c.logger.Info("successfully created role '%s'", role.Name)
	return nil
}

// UpdateRoleWithContext updates an existing role via the API with context support
func (c *Client) UpdateRoleWithContext(ctx context.Context, role models.Role) error {
	c.logger.Info("updating role '%s' via API", role.Name)
	start := time.Now()
	defer func() {
		c.logger.Debug("UpdateRole for '%s' completed in %v", role.Name, time.Since(start))
	}()

	if role.ID == "" {
		c.logger.Error("cannot update role '%s': ID is required", role.Name)
		return fmt.Errorf("ID is required for updating role '%s'", role.Name)
	}

	url := c.baseURL + "/vendor/v3/policies/" + role.ID
	c.logger.Debug("updating role at endpoint: %s", url)

	requestData := struct {
		Name string `json:"name"`
	}{
		Name: role.Name,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		c.logger.Error("failed to marshal role data for '%s': %v", role.Name, err)
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(jsonData))
	if err != nil {
		c.logger.Error("failed to create HTTP request for updating role '%s': %v", role.Name, err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.executeWithRetry(ctx, req)
	if err != nil {
		c.logger.Error("HTTP request failed for updating role '%s': %v", role.Name, err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received HTTP response for updating role '%s': status=%d", role.Name, resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("failed to update role '%s': API returned status %d", role.Name, resp.StatusCode)
		return c.handleErrorResponse(resp)
	}

	c.logger.Info("successfully updated role '%s'", role.Name)
	return nil
}

// DeleteRoleWithContext deletes a role via the API with context support
func (c *Client) DeleteRoleWithContext(ctx context.Context, roleName string) error {
	c.logger.Info("deleting role '%s' via API", roleName)
	start := time.Now()
	defer func() {
		c.logger.Debug("DeleteRole for '%s' completed in %v", roleName, time.Since(start))
	}()

	role, err := c.GetRoleWithContext(ctx, roleName)
	if err != nil {
		return fmt.Errorf("failed to find role '%s' for deletion: %w", roleName, err)
	}

	if role.ID == "" {
		c.logger.Error("cannot delete role '%s': ID is missing", roleName)
		return fmt.Errorf("ID is required for deleting role '%s'", roleName)
	}

	url := c.baseURL + "/vendor/v3/policies/" + role.ID
	c.logger.Debug("deleting role at endpoint: %s", url)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		c.logger.Error("failed to create HTTP request for deleting role '%s': %v", roleName, err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.executeWithRetry(ctx, req)
	if err != nil {
		c.logger.Error("HTTP request failed for deleting role '%s': %v", roleName, err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received HTTP response for deleting role '%s': status=%d", roleName, resp.StatusCode)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.logger.Error("failed to delete role '%s': API returned status %d", roleName, resp.StatusCode)
		return c.handleErrorResponse(resp)
	}

	c.logger.Info("successfully deleted role '%s'", roleName)
	return nil
}

// GetTeamMembers retrieves all team members from the API
func (c *Client) GetTeamMembers() ([]models.TeamMember, error) {
	return c.GetTeamMembersWithContext(context.Background())
}

// GetTeamMembersWithContext retrieves all team members from the API with context support
func (c *Client) GetTeamMembersWithContext(ctx context.Context) ([]models.TeamMember, error) {
	c.logger.Info("fetching team members from API")
	start := time.Now()
	defer func() {
		c.logger.Debug("GetTeamMembers completed in %v", time.Since(start))
	}()

	url := c.baseURL + "/v1/team/members"
	c.logger.Debug("fetching team members from endpoint: %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create HTTP request for %s: %v", url, err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.executeWithRetry(ctx, req)
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

	// Parse the response - /v1/team/members returns an array directly
	var members []models.TeamMember
	if err := json.Unmarshal(body, &members); err != nil {
		c.logger.Error("failed to parse JSON response from %s: %v", url, err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.Info("successfully fetched %d team members from API", len(members))
	return members, nil
}

// AssignMemberRole assigns a role to a team member via the API
func (c *Client) AssignMemberRole(memberEmail, roleID string) error {
	return c.AssignMemberRoleWithContext(context.Background(), memberEmail, roleID)
}

// AssignMemberRoleWithContext assigns a role to a team member via the API with context support
func (c *Client) AssignMemberRoleWithContext(ctx context.Context, memberEmail, roleID string) error {
	c.logger.Info("assigning role '%s' to member '%s' via API", roleID, memberEmail)
	start := time.Now()
	defer func() {
		c.logger.Debug("AssignMemberRole for '%s' completed in %v", memberEmail, time.Since(start))
	}()

	if strings.TrimSpace(memberEmail) == "" {
		c.logger.Error("member email is required for role assignment")
		return fmt.Errorf("member email is required")
	}
	if strings.TrimSpace(roleID) == "" {
		c.logger.Error("role ID is required for role assignment")
		return fmt.Errorf("role ID is required")
	}

	url := c.baseURL + "/vendor/v3/team/member/" + memberEmail + "/role/" + roleID
	c.logger.Debug("assigning role at endpoint: %s", url)

	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		c.logger.Error("failed to create HTTP request for %s: %v", url, err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.executeWithRetry(ctx, req)
	if err != nil {
		c.logger.Error("HTTP request failed for %s: %v", url, err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received HTTP response for role assignment: status=%d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.logger.Error("failed to assign role '%s' to member '%s': API returned status %d", roleID, memberEmail, resp.StatusCode)
		return c.handleErrorResponse(resp)
	}

	c.logger.Info("successfully assigned role '%s' to member '%s'", roleID, memberEmail)
	return nil
}
