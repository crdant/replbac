package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"replbac/internal/models"
)

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
			client, err := NewClient(tt.baseURL, tt.apiToken)

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
					Name: "admin",
					Resources: models.Resources{
						Allowed: []string{"**/*"},
						Denied:  []string{"kots/app/*/delete"},
					},
				},
				{
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
				if r.URL.Path != "/vendor/v3/policies" {
					t.Errorf("Expected path /vendor/v3/policies, got %s", r.URL.Path)
				}

				// Verify authorization header
				authHeader := r.Header.Get("Authorization")
				if authHeader != "test-token" {
					t.Errorf("Expected Authorization header 'test-token', got '%s'", authHeader)
				}

				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token")
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
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token")
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
				expectedPath := "/vendor/v3/policy/" + role.Name
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token")
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				expectedPath := "/vendor/v3/policy/" + tt.roleName
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token")
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
				expectedPath := "/vendor/v3/policy/" + tt.roleName
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token")
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