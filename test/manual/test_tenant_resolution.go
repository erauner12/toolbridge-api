package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Test the /v1/auth/tenant endpoint
// This script assumes you already have an ID token from OIDC login
//
// Usage:
//   1. Set BACKEND_URL (e.g., http://localhost:8080 or https://toolbridgeapi.erauner.dev)
//   2. Set ID_TOKEN from your OIDC login
//   3. Run: go run test_tenant_resolution.go

func main() {
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}

	idToken := os.Getenv("ID_TOKEN")
	if idToken == "" {
		fmt.Println("ERROR: ID_TOKEN environment variable is required")
		fmt.Println()
		fmt.Println("To get an ID token:")
		fmt.Println("  1. Run: go run test/manual/standard_oidc_tenant.go")
		fmt.Println("  2. Copy the ID token from the output")
		fmt.Println("  3. Export it: export ID_TOKEN='<token>'")
		fmt.Println("  4. Run this script again")
		os.Exit(1)
	}

	fmt.Printf("Testing /v1/auth/tenant endpoint at %s\n", backendURL)
	fmt.Println()

	// Call the tenant resolution endpoint
	req, err := http.NewRequest("GET", backendURL+"/v1/auth/tenant", nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Authorization", "Bearer "+idToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error calling endpoint: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Response Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Println()

	if resp.StatusCode == 200 {
		// Parse and pretty-print the response
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			fmt.Printf("Raw response: %s\n", string(body))
			os.Exit(1)
		}

		prettyJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println("✓ SUCCESS - Tenant resolved:")
		fmt.Println(string(prettyJSON))
		fmt.Println()

		// Extract key fields
		if tenantID, ok := result["tenant_id"].(string); ok {
			fmt.Printf("Tenant ID: %s\n", tenantID)
		}
		if orgName, ok := result["organization_name"].(string); ok {
			fmt.Printf("Organization Name: %s\n", orgName)
		}
		if requiresSelection, ok := result["requires_selection"].(bool); ok && requiresSelection {
			fmt.Println("⚠ User belongs to multiple organizations - selection required")
		}
	} else {
		fmt.Printf("✗ FAILED - Status %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
		os.Exit(1)
	}
}
