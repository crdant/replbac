// Test script to verify end-to-end member functionality
package main

import (
	"fmt"
	"log"
	"os"

	"replbac/internal/api"
	"replbac/internal/logging"
	"replbac/internal/models"
)

func main() {
	// Check if API token is provided
	apiToken := os.Getenv("REPLICATED_API_TOKEN")
	if apiToken == "" {
		fmt.Println("Please set REPLICATED_API_TOKEN environment variable")
		fmt.Println("Usage: REPLICATED_API_TOKEN=your_token go run test_member_roundtrip.go")
		os.Exit(1)
	}

	// Create logger
	logger := logging.NewLogger(os.Stderr, true)

	// Create API client
	client, err := api.NewClient(models.ReplicatedAPIEndpoint, apiToken, logger)
	if err != nil {
		log.Fatalf("Failed to create API client: %v", err)
	}

	fmt.Println("Testing member API integration...")

	// Test 1: Get team members directly
	fmt.Println("\n1. Testing GetTeamMembers...")
	members, err := client.GetTeamMembers()
	if err != nil {
		log.Fatalf("Failed to get team members: %v", err)
	}
	fmt.Printf("Found %d team members:\n", len(members))
	for _, member := range members {
		fmt.Printf("  - %s (Policy: %s)\n", member.ID, member.PolicyID)
	}

	// Test 2: Get roles with member correlation
	fmt.Println("\n2. Testing GetRoles with member correlation...")
	roles, err := client.GetRoles()
	if err != nil {
		log.Fatalf("Failed to get roles: %v", err)
	}
	fmt.Printf("Found %d roles:\n", len(roles))
	for _, role := range roles {
		if len(role.Members) > 0 {
			fmt.Printf("  - %s: %d members %v\n", role.Name, len(role.Members), role.Members)
		} else {
			fmt.Printf("  - %s: no members\n", role.Name)
		}
	}

	fmt.Println("\nMember API integration test completed successfully!")
}