package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"replbac/internal/logging"
	"replbac/internal/models"
)

// TestGetTeamMembersWithPolicyId tests the integration with the correct member API endpoint
func TestGetTeamMembersWithPolicyId(t *testing.T) {
	// Mock response based on the real API structure you discovered
	mockResponse := `[
		{
			"id": "ada+shortrib@replicated.com",
			"createdAt": "2025-03-07T20:41:53Z",
			"lastActiveAt": null,
			"policyId": "policy-123"
		},
		{
			"id": "user2@example.com", 
			"createdAt": "2025-03-06T15:30:00Z",
			"lastActiveAt": "2025-03-07T10:15:30Z",
			"policyId": "policy-456"
		}
	]`

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify correct endpoint is called
		expectedPath := "/v1/team/members"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		
		// Verify headers
		if r.Header.Get("Authorization") == "" {
			t.Error("Missing Authorization header")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Missing or incorrect Accept header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	// Create client
	var buf bytes.Buffer
	logger := logging.NewLogger(&buf, false)
	client, err := NewClient(server.URL, "test-token", logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetTeamMembers
	members, err := client.GetTeamMembers()
	if err != nil {
		t.Fatalf("GetTeamMembers failed: %v", err)
	}

	// Verify results
	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}

	// Check first member
	if members[0].ID != "ada+shortrib@replicated.com" {
		t.Errorf("Expected ID 'ada+shortrib@replicated.com', got '%s'", members[0].ID)
	}
	if members[0].PolicyID != "policy-123" {
		t.Errorf("Expected PolicyID 'policy-123', got '%s'", members[0].PolicyID)
	}

	// Check second member  
	if members[1].ID != "user2@example.com" {
		t.Errorf("Expected ID 'user2@example.com', got '%s'", members[1].ID)
	}
	if members[1].PolicyID != "policy-456" {
		t.Errorf("Expected PolicyID 'policy-456', got '%s'", members[1].PolicyID)
	}
}

// TestGetRolesWithMembers tests that GetRoles properly correlates members with roles
func TestGetRolesWithMembers(t *testing.T) {
	// Mock role/policy response
	policiesResponse := `{
		"policies": [
			{
				"id": "policy-123",
				"teamId": "test-team",
				"name": "admin", 
				"description": "Admin policy",
				"definition": "{\"v1\":{\"name\":\"admin\",\"resources\":{\"allowed\":[\"*\"],\"denied\":[]}}}",
				"createdAt": "2023-01-01T00:00:00Z",
				"modifiedAt": null,
				"readOnly": false
			},
			{
				"id": "policy-456",
				"teamId": "test-team",
				"name": "viewer",
				"description": "Viewer policy", 
				"definition": "{\"v1\":{\"name\":\"viewer\",\"resources\":{\"allowed\":[\"read\"],\"denied\":[]}}}",
				"createdAt": "2023-01-01T00:00:00Z",
				"modifiedAt": null,
				"readOnly": false
			}
		]
	}`

	// Mock members response
	membersResponse := `[
		{
			"id": "admin@example.com",
			"createdAt": "2025-03-07T20:41:53Z", 
			"lastActiveAt": null,
			"policyId": "policy-123"
		},
		{
			"id": "user1@example.com",
			"createdAt": "2025-03-06T15:30:00Z",
			"lastActiveAt": "2025-03-07T10:15:30Z", 
			"policyId": "policy-456"
		},
		{
			"id": "user2@example.com",
			"createdAt": "2025-03-05T12:00:00Z",
			"lastActiveAt": "2025-03-07T09:45:15Z",
			"policyId": "policy-456" 
		}
	]`

	// Create test server that handles both endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		switch r.URL.Path {
		case "/vendor/v3/policies":
			w.Write([]byte(policiesResponse))
		case "/v1/team/members":
			w.Write([]byte(membersResponse))
		default:
			t.Errorf("Unexpected endpoint called: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client
	var buf bytes.Buffer
	logger := logging.NewLogger(&buf, false)
	client, err := NewClient(server.URL, "test-token", logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetRoles - should now include member information
	roles, err := client.GetRoles()
	if err != nil {
		t.Fatalf("GetRoles failed: %v", err)
	}

	// Verify results
	if len(roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(roles))
	}

	// Find admin role and verify it has the correct member
	var adminRole *models.Role
	for i := range roles {
		if roles[i].Name == "admin" {
			adminRole = &roles[i]
			break
		}
	}
	if adminRole == nil {
		t.Fatal("Admin role not found")
	}
	if len(adminRole.Members) != 1 {
		t.Errorf("Expected admin role to have 1 member, got %d", len(adminRole.Members))
	}
	if len(adminRole.Members) > 0 && adminRole.Members[0] != "admin@example.com" {
		t.Errorf("Expected admin role member 'admin@example.com', got '%s'", adminRole.Members[0])
	}

	// Find viewer role and verify it has the correct members
	var viewerRole *models.Role
	for i := range roles {
		if roles[i].Name == "viewer" {
			viewerRole = &roles[i]
			break
		}
	}
	if viewerRole == nil {
		t.Fatal("Viewer role not found")
	}
	if len(viewerRole.Members) != 2 {
		t.Errorf("Expected viewer role to have 2 members, got %d", len(viewerRole.Members))
	}
	
	// Check that both viewer members are present (order may vary)
	expectedMembers := map[string]bool{
		"user1@example.com": false,
		"user2@example.com": false,
	}
	for _, member := range viewerRole.Members {
		if _, exists := expectedMembers[member]; exists {
			expectedMembers[member] = true
		}
	}
	for member, found := range expectedMembers {
		if !found {
			t.Errorf("Expected viewer role member '%s' not found", member)
		}
	}
}