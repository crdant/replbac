package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"replbac/internal/logging"
	"replbac/internal/models"
)

// createTestLogger creates a logger for testing
func createTestLogger() *logging.Logger {
	var buf bytes.Buffer
	return logging.NewLogger(&buf, true) // verbose for testing
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		apiToken    string
		expectError bool
	}{
		{
			name:     "valid client creation",
			baseURL:  "https://api.replicated.com",
			apiToken: "test-token",
		},
		{
			name:        "invalid base URL",
			baseURL:     "not-a-url",
			apiToken:    "test-token",
			expectError: true,
		},
		{
			name:        "empty API token",
			baseURL:     "https://api.replicated.com",
			apiToken:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.baseURL, tt.apiToken, createTestLogger())

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

			if client == nil {
				t.Error("Expected client but got nil")
			}
		})
	}
}

func TestGetRoles(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		expectedRoles  []models.Role
		expectError    bool
	}{
		{
			name:           "successful get roles",
			mockStatusCode: http.StatusOK,
			mockResponse: `{
				"policies": [
					{
						"id": "test-admin-id",
						"teamId": "test-team",
						"name": "admin",
						"description": "Admin policy",
						"definition": "{\"v1\":{\"name\":\"admin\",\"resources\":{\"allowed\":[\"**/*\"],\"denied\":[\"kots/app/*/delete\"]}}}",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": null,
						"readOnly": false
					},
					{
						"id": "test-viewer-id", 
						"teamId": "test-team",
						"name": "viewer",
						"description": "Viewer policy",
						"definition": "{\"v1\":{\"name\":\"viewer\",\"resources\":{\"allowed\":[\"kots/app/*/read\"],\"denied\":[]}}}",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": null,
						"readOnly": false
					}
				]
			}`,
			expectedRoles: []models.Role{
				{
					ID:   "test-admin-id",
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"**/*"},
						Denied:  []string{"kots/app/*/delete"},
					},
				},
				{
					ID:   "test-viewer-id",
					Name: "viewer",
					Resources: models.Resources{
						Allowed: []string{"kots/app/*/read"},
						Denied:  []string{},
					},
				},
			},
		},
		{
			name:           "empty roles list",
			mockStatusCode: http.StatusOK,
			mockResponse:   `{"policies": []}`,
			expectedRoles:  []models.Role{},
		},
		{
			name:           "API error response",
			mockStatusCode: http.StatusUnauthorized,
			mockResponse:   `{"error": "unauthorized"}`,
			expectError:    true,
		},
		{
			name:           "invalid JSON response",
			mockStatusCode: http.StatusOK,
			mockResponse:   `invalid json`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}

				// Handle both endpoints now that GetRoles calls both
				switch r.URL.Path {
				case "/vendor/v3/policies":
					// Verify authorization header
					authHeader := r.Header.Get("Authorization")
					if authHeader != "test-token" {
						t.Errorf("Expected Authorization header 'test-token', got '%s'", authHeader)
					}
					w.WriteHeader(tt.mockStatusCode)
					if _, err := w.Write([]byte(tt.mockResponse)); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				case "/v1/team/members":
					// Return empty members list for simplicity
					w.WriteHeader(http.StatusOK)
					if _, err := w.Write([]byte("[]")); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				default:
					t.Errorf("Unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			roles, err := client.GetRoles()

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

			if !reflect.DeepEqual(roles, tt.expectedRoles) {
				t.Errorf("Roles = %+v, want %+v", roles, tt.expectedRoles)
			}
		})
	}
}

func TestCreateRole(t *testing.T) {
	tests := []struct {
		name           string
		role           models.Role
		mockStatusCode int
		mockResponse   string
		expectError    bool
	}{
		{
			name: "successful role creation",
			role: models.Role{
				Name: "test-role",
				Resources: models.Resources{
					Allowed: []string{"kots/app/*/read"},
					Denied:  []string{},
				},
			},
			mockStatusCode: http.StatusCreated,
			mockResponse: `{
				"v1": {
					"name": "test-role",
					"resources": {
						"allowed": ["kots/app/*/read"],
						"denied": []
					}
				}
			}`,
		},
		{
			name: "API error on creation",
			role: models.Role{
				Name: "invalid-role",
				Resources: models.Resources{
					Allowed: []string{"invalid"},
					Denied:  []string{},
				},
			},
			mockStatusCode: http.StatusBadRequest,
			mockResponse:   `{"error": "invalid role"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/vendor/v3/policy" {
					t.Errorf("Expected path /vendor/v3/policy, got %s", r.URL.Path)
				}

				// Verify content type
				contentType := r.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
				}

				// Verify request body contains the policy structure
				var requestBody struct {
					Name        string `json:"name"`
					Description string `json:"description,omitempty"`
					Definition  string `json:"definition"`
				}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}

				// Verify the policy structure
				if requestBody.Name != tt.role.Name {
					t.Errorf("Request body name = %s, want %s", requestBody.Name, tt.role.Name)
				}

				// Verify the definition contains the correct APIRole
				var definitionContent models.APIRole
				if err := json.Unmarshal([]byte(requestBody.Definition), &definitionContent); err != nil {
					t.Errorf("Failed to decode definition: %v", err)
				}

				expectedAPIRole := tt.role.ToAPIRole()
				if !reflect.DeepEqual(definitionContent, expectedAPIRole) {
					t.Errorf("Definition content = %+v, want %+v", definitionContent, expectedAPIRole)
				}

				w.WriteHeader(tt.mockStatusCode)
				if _, err := w.Write([]byte(tt.mockResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.CreateRole(tt.role)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateRole(t *testing.T) {
	role := models.Role{
		ID:   "test-role-id",
		Name: "test-role",
		Resources: models.Resources{
			Allowed: []string{"kots/app/*/read", "kots/app/*/write"},
			Denied:  []string{},
		},
	}

	tests := []struct {
		name           string
		mockStatusCode int
		mockResponse   string
		expectError    bool
	}{
		{
			name:           "successful role update",
			mockStatusCode: http.StatusOK,
			mockResponse: `{
				"v1": {
					"name": "test-role",
					"resources": {
						"allowed": ["kots/app/*/read", "kots/app/*/write"],
						"denied": []
					}
				}
			}`,
		},
		{
			name:           "role not found",
			mockStatusCode: http.StatusNotFound,
			mockResponse:   `{"error": "role not found"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodPut {
					t.Errorf("Expected PUT request, got %s", r.Method)
				}
				expectedPath := "/vendor/v3/policy/" + role.ID
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.mockStatusCode)
				if _, err := w.Write([]byte(tt.mockResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.UpdateRole(role)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteRole(t *testing.T) {
	tests := []struct {
		name           string
		roleName       string
		mockStatusCode int
		mockResponse   string
		expectError    bool
	}{
		{
			name:           "successful role deletion",
			roleName:       "test-role",
			mockStatusCode: http.StatusNoContent,
			mockResponse:   "",
		},
		{
			name:           "role not found",
			roleName:       "nonexistent-role",
			mockStatusCode: http.StatusNotFound,
			mockResponse:   `{"error": "role not found"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				if requestCount == 1 {
					// First request: GET /vendor/v3/policies to look up ID
					if r.Method != http.MethodGet {
						t.Errorf("Expected GET request for policy lookup, got %s", r.Method)
					}
					if r.URL.Path != "/vendor/v3/policies" {
						t.Errorf("Expected path /vendor/v3/policies, got %s", r.URL.Path)
					}

					// Return policy list with the role
					if tt.roleName == "test-role" {
						w.WriteHeader(http.StatusOK)
						if _, err := w.Write([]byte(`{"policies": [{"id": "test-role-id", "name": "test-role", "definition": "{}"}]}`)); err != nil {
							t.Errorf("Failed to write response: %v", err)
						}
					} else {
						// Role not found in lookup
						w.WriteHeader(http.StatusOK)
						if _, err := w.Write([]byte(`{"policies": []}`)); err != nil {
							t.Errorf("Failed to write response: %v", err)
						}
					}
				} else {
					// Second request: DELETE /vendor/v3/policy/{id}
					if r.Method != http.MethodDelete {
						t.Errorf("Expected DELETE request, got %s", r.Method)
					}
					expectedPath := "/vendor/v3/policy/test-role-id"
					if r.URL.Path != expectedPath {
						t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
					}

					w.WriteHeader(tt.mockStatusCode)
					if _, err := w.Write([]byte(tt.mockResponse)); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.DeleteRole(tt.roleName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetRole(t *testing.T) {
	tests := []struct {
		name           string
		roleName       string
		mockStatusCode int
		mockResponse   string
		expectedRole   models.Role
		expectError    bool
	}{
		{
			name:           "successful get role",
			roleName:       "admin",
			mockStatusCode: http.StatusOK,
			mockResponse: `{
				"v1": {
					"name": "admin",
					"resources": {
						"allowed": ["**/*"],
						"denied": ["kots/app/*/delete"]
					}
				}
			}`,
			expectedRole: models.Role{
				ID:   "admin-id",
				Name: "admin",
				Resources: models.Resources{
					Allowed: []string{"**/*"},
					Denied:  []string{"kots/app/*/delete"},
				},
			},
		},
		{
			name:           "role not found",
			roleName:       "nonexistent",
			mockStatusCode: http.StatusNotFound,
			mockResponse:   `{"error": "role not found"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/vendor/v3/policies" {
					t.Errorf("Expected path /vendor/v3/policies, got %s", r.URL.Path)
				}

				// Return policy list
				if tt.roleName == "admin" {
					w.WriteHeader(http.StatusOK)
					if _, err := w.Write([]byte(`{"policies": [{"id": "admin-id", "name": "admin", "definition": "{\"v1\":{\"name\":\"admin\",\"resources\":{\"allowed\":[\"**/*\"],\"denied\":[\"kots/app/*/delete\"]}}}"}]}`)); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				} else {
					// Role not found
					w.WriteHeader(http.StatusOK)
					if _, err := w.Write([]byte(`{"policies": []}`)); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			role, err := client.GetRole(tt.roleName)

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

			if !reflect.DeepEqual(role, tt.expectedRole) {
				t.Errorf("Role = %+v, want %+v", role, tt.expectedRole)
			}
		})
	}
}

func TestGetTeamMembers(t *testing.T) {
	tests := []struct {
		name            string
		mockStatusCode  int
		mockResponse    string
		expectedMembers []models.TeamMember
		expectError     bool
	}{
		{
			name:           "successful get team members",
			mockStatusCode: http.StatusOK,
			mockResponse: `[
				{
					"id": "member1",
					"email": "john@example.com",
					"name": "John Doe",
					"username": "john"
				},
				{
					"id": "member2", 
					"email": "jane@example.com",
					"name": "Jane Smith",
					"username": "jane"
				}
			]`,
			expectedMembers: []models.TeamMember{
				{
					ID:       "member1",
					Email:    "john@example.com",
					Name:     "John Doe",
					Username: "john",
				},
				{
					ID:       "member2",
					Email:    "jane@example.com",
					Name:     "Jane Smith",
					Username: "jane",
				},
			},
		},
		{
			name:            "empty team members list",
			mockStatusCode:  http.StatusOK,
			mockResponse:    `[]`,
			expectedMembers: []models.TeamMember{},
		},
		{
			name:           "API error",
			mockStatusCode: http.StatusInternalServerError,
			mockResponse:   `{"error": "internal server error"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/v1/team/members" {
					t.Errorf("Expected path /v1/team/members, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.mockStatusCode)
				if _, err := w.Write([]byte(tt.mockResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			members, err := client.GetTeamMembers()

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

			if !reflect.DeepEqual(members, tt.expectedMembers) {
				t.Errorf("Expected members %+v, got %+v", tt.expectedMembers, members)
			}
		})
	}
}

func TestAssignMemberRole(t *testing.T) {
	tests := []struct {
		name           string
		memberEmail    string
		roleID         string
		mockStatusCode int
		mockResponse   string
		expectError    bool
	}{
		{
			name:           "successful role assignment",
			memberEmail:    "john@example.com",
			roleID:         "role123",
			mockStatusCode: http.StatusOK,
			mockResponse:   `{"success": true}`,
		},
		{
			name:           "member not found",
			memberEmail:    "nonexistent@example.com",
			roleID:         "role123",
			mockStatusCode: http.StatusNotFound,
			mockResponse:   `{"error": "member not found"}`,
			expectError:    true,
		},
		{
			name:           "role not found",
			memberEmail:    "john@example.com",
			roleID:         "nonexistent-role",
			mockStatusCode: http.StatusNotFound,
			mockResponse:   `{"error": "role not found"}`,
			expectError:    true,
		},
		{
			name:           "invalid request",
			memberEmail:    "",
			roleID:         "role123",
			mockStatusCode: http.StatusBadRequest,
			mockResponse:   `{"error": "invalid request"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("Expected PUT request, got %s", r.Method)
				}
				expectedPath := "/vendor/v3/team/member/" + tt.memberEmail + "/role/" + tt.roleID
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.mockStatusCode)
				if _, err := w.Write([]byte(tt.mockResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.AssignMemberRole(tt.memberEmail, tt.roleID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestInviteUser(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		policyID       string
		mockStatusCode int
		mockResponse   string
		expectError    bool
		expectedResp   *models.InviteUserResponse
	}{
		{
			name:           "successful invite",
			email:          "test@example.com",
			policyID:       "policy123",
			mockStatusCode: http.StatusCreated,
			mockResponse:   `{"id": "invite123", "email": "test@example.com", "policy_id": "policy123", "status": "pending"}`,
			expectError:    false,
			expectedResp: &models.InviteUserResponse{
				ID:       "invite123",
				Email:    "test@example.com",
				PolicyID: "policy123",
				Status:   "pending",
			},
		},
		{
			name:           "user already exists",
			email:          "existing@example.com",
			policyID:       "policy123",
			mockStatusCode: http.StatusConflict,
			mockResponse:   `{"error": "user already exists"}`,
			expectError:    true,
			expectedResp:   nil,
		},
		{
			name:           "invalid email",
			email:          "invalid-email",
			policyID:       "policy123",
			mockStatusCode: http.StatusBadRequest,
			mockResponse:   `{"error": "invalid email format"}`,
			expectError:    true,
			expectedResp:   nil,
		},
		{
			name:           "empty email",
			email:          "",
			policyID:       "policy123",
			mockStatusCode: http.StatusBadRequest,
			mockResponse:   "",
			expectError:    true,
			expectedResp:   nil,
		},
		{
			name:           "empty policy ID",
			email:          "test@example.com",
			policyID:       "",
			mockStatusCode: http.StatusBadRequest,
			mockResponse:   "",
			expectError:    true,
			expectedResp:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/vendor/v3/team/invite" {
					t.Errorf("Expected path /vendor/v3/team/invite, got %s", r.URL.Path)
				}

				// Verify request body for non-empty inputs
				if tt.email != "" && tt.policyID != "" {
					var reqBody models.InviteUserRequest
					if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
						t.Errorf("Failed to decode request body: %v", err)
						return
					}
					if reqBody.Email != tt.email {
						t.Errorf("Expected email %s, got %s", tt.email, reqBody.Email)
					}
					if reqBody.PolicyID != tt.policyID {
						t.Errorf("Expected policy ID %s, got %s", tt.policyID, reqBody.PolicyID)
					}
				}

				w.WriteHeader(tt.mockStatusCode)
				if _, err := w.Write([]byte(tt.mockResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			resp, err := client.InviteUser(tt.email, tt.policyID)

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

			if !reflect.DeepEqual(resp, tt.expectedResp) {
				t.Errorf("Expected response %+v, got %+v", tt.expectedResp, resp)
			}
		})
	}
}

func TestInviteUserWithContext(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		policyID       string
		mockStatusCode int
		mockResponse   string
		contextTimeout time.Duration
		expectError    bool
	}{
		{
			name:           "successful invite with context",
			email:          "test@example.com",
			policyID:       "policy123",
			mockStatusCode: http.StatusCreated,
			mockResponse:   `{"id": "invite123", "email": "test@example.com", "policy_id": "policy123", "status": "pending"}`,
			contextTimeout: 5 * time.Second,
			expectError:    false,
		},
		{
			name:           "context timeout",
			email:          "test@example.com",
			policyID:       "policy123",
			mockStatusCode: http.StatusCreated,
			mockResponse:   `{"id": "invite123"}`,
			contextTimeout: 1 * time.Millisecond, // Very short timeout
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.name == "context timeout" {
					// Simulate slow server for timeout test
					time.Sleep(10 * time.Millisecond)
				}

				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/vendor/v3/team/invite" {
					t.Errorf("Expected path /vendor/v3/team/invite, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.mockStatusCode)
				if _, err := w.Write([]byte(tt.mockResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", createTestLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTimeout)
			defer cancel()

			_, err = client.InviteUserWithContext(ctx, tt.email, tt.policyID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
