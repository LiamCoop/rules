//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB creates a PostgreSQL testcontainer and runs migrations
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	host, err := postgres.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := postgres.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	connStr := fmt.Sprintf("postgres://postgres:password@%s:%s/testdb?sslmode=disable", host, port.Port())

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Run migrations
	migrationSQL, err := os.ReadFile("../../migrations/000001_initial_schema.up.sql")
	if err != nil {
		t.Fatalf("Failed to read migration file: %v", err)
	}

	if _, err := db.Exec(string(migrationSQL)); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		db.Close()
		postgres.Terminate(ctx)
	}

	return db, cleanup
}

// TestEndToEnd_CreateTenantAndEvaluateRule tests the complete workflow:
// 1. Create tenant
// 2. Create schema
// 3. Add rule
// 4. Evaluate rule
func TestEndToEnd_CreateTenantAndEvaluateRule(t *testing.T) {
	// Setup database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Start HTTP server
	server, err := NewServerWithDB(db)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Run server in background
	go func() {
		if err := http.ListenAndServe(":8080", server); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(500 * time.Millisecond)

	baseURL := "http://localhost:8080/api/v1"

	// Step 1: Create tenant
	t.Log("Step 1: Creating tenant...")
	createTenantReq := map[string]interface{}{
		"name": "Test Tenant",
	}
	tenantResp := makeRequest(t, "POST", baseURL+"/tenants", createTenantReq)
	tenantID := tenantResp["id"].(string)
	t.Logf("Created tenant: %s", tenantID)

	// Step 2: Create schema
	t.Log("Step 2: Creating schema...")
	createSchemaReq := map[string]interface{}{
		"definition": map[string]interface{}{
			"User": map[string]interface{}{
				"Age":  "int",
				"Name": "string",
			},
		},
	}
	schemaResp := makeRequest(t, "POST", baseURL+"/tenants/"+tenantID+"/schema", createSchemaReq)
	t.Logf("Created schema version: %v", schemaResp["version"])

	// Verify schema version is 1
	if version, ok := schemaResp["version"].(float64); !ok || version != 1 {
		t.Errorf("Expected schema version 1, got %v", schemaResp["version"])
	}

	// Step 3: Add rule
	t.Log("Step 3: Adding rule...")
	createRuleReq := map[string]interface{}{
		"name":       "adult-check",
		"expression": "User.Age >= 18",
		"active":     true,
	}
	ruleResp := makeRequest(t, "POST", baseURL+"/tenants/"+tenantID+"/rules", createRuleReq)
	ruleID := ruleResp["id"].(string)
	t.Logf("Created rule: %s", ruleID)

	// Step 4: Evaluate rule - adult user (should match)
	t.Log("Step 4a: Evaluating rule for adult user...")
	evaluateReq := map[string]interface{}{
		"tenantId": tenantID,
		"rules":    []string{ruleID},
		"facts": map[string]interface{}{
			"User": map[string]interface{}{
				"Age":  25,
				"Name": "John Doe",
			},
		},
	}
	evalResp := makeRequest(t, "POST", baseURL+"/evaluate", evaluateReq)

	// Check results array
	results, ok := evalResp["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatalf("Expected results array, got %v", evalResp)
	}

	firstResult := results[0].(map[string]interface{})
	if matched, ok := firstResult["Matched"].(bool); !ok || !matched {
		t.Errorf("Expected adult user to match rule, got matched=%v", firstResult["Matched"])
	}
	t.Logf("Evaluation result: %v", firstResult)

	// Step 4b: Evaluate rule - minor user (should not match)
	t.Log("Step 4b: Evaluating rule for minor user...")
	evaluateReq["facts"] = map[string]interface{}{
		"User": map[string]interface{}{
			"Age":  16,
			"Name": "Jane Doe",
		},
	}
	evalResp = makeRequest(t, "POST", baseURL+"/evaluate", evaluateReq)

	results, ok = evalResp["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatalf("Expected results array, got %v", evalResp)
	}

	firstResult = results[0].(map[string]interface{})
	if matched, ok := firstResult["Matched"].(bool); !ok || matched {
		t.Errorf("Expected minor user to not match rule, got matched=%v", firstResult["Matched"])
	}
	t.Logf("Evaluation result: %v", firstResult)

	// Step 5: List rules to verify it was stored
	t.Log("Step 5: Listing rules...")
	rulesResp := makeRequestNoBody(t, "GET", baseURL+"/tenants/"+tenantID+"/rules")
	rules, ok := rulesResp["rules"].([]interface{})
	if !ok || len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %v", rulesResp)
	}

	// Step 6: Get schema to verify it's stored
	t.Log("Step 6: Getting schema...")
	getSchemaResp := makeRequestNoBody(t, "GET", baseURL+"/tenants/"+tenantID+"/schema")
	if getSchemaResp["version"].(float64) != 1 {
		t.Errorf("Expected schema version 1, got %v", getSchemaResp["version"])
	}

	t.Log("End-to-end test completed successfully!")
}

// TestEndToEnd_SchemaUpdate tests zero-downtime schema updates
func TestEndToEnd_SchemaUpdate(t *testing.T) {
	// Setup database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Start HTTP server
	server, err := NewServerWithDB(db)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		if err := http.ListenAndServe(":8081", server); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	baseURL := "http://localhost:8081/api/v1"

	// Create tenant
	createTenantReq := map[string]interface{}{
		"name": "Schema Update Test Tenant",
	}
	tenantResp := makeRequest(t, "POST", baseURL+"/tenants", createTenantReq)
	tenantID := tenantResp["id"].(string)

	// Create initial schema
	createSchemaReq := map[string]interface{}{
		"definition": map[string]interface{}{
			"User": map[string]interface{}{
				"Age": "int",
			},
		},
	}
	makeRequest(t, "POST", baseURL+"/tenants/"+tenantID+"/schema", createSchemaReq)

	// Add rule with initial schema
	createRuleReq := map[string]interface{}{
		"name":       "adult-check",
		"expression": "User.Age >= 18",
		"active":     true,
	}
	ruleResp := makeRequest(t, "POST", baseURL+"/tenants/"+tenantID+"/rules", createRuleReq)
	ruleID := ruleResp["id"].(string)

	// Update schema to add Email field
	t.Log("Updating schema to add Email field...")
	updateSchemaReq := map[string]interface{}{
		"definition": map[string]interface{}{
			"User": map[string]interface{}{
				"Age":   "int",
				"Email": "string",
			},
		},
	}
	schemaResp := makeRequest(t, "PUT", baseURL+"/tenants/"+tenantID+"/schema", updateSchemaReq)

	if version, ok := schemaResp["version"].(float64); !ok || version != 2 {
		t.Errorf("Expected schema version 2 after update, got %v", schemaResp["version"])
	}

	// Verify old rule still works after schema update
	t.Log("Verifying old rule still works after schema update...")
	evaluateReq := map[string]interface{}{
		"tenantId": tenantID,
		"rules":    []string{ruleID},
		"facts": map[string]interface{}{
			"User": map[string]interface{}{
				"Age": 25,
			},
		},
	}
	evalResp := makeRequest(t, "POST", baseURL+"/evaluate", evaluateReq)

	results, ok := evalResp["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatalf("Expected results array, got %v", evalResp)
	}

	firstResult := results[0].(map[string]interface{})
	if matched, ok := firstResult["Matched"].(bool); !ok || !matched {
		t.Errorf("Old rule should still work after schema update, got matched=%v", firstResult["Matched"])
	}

	t.Log("Schema update test completed successfully!")
}

// TestEndToEnd_CreateSchemaConflict tests that you can't create a schema twice
func TestEndToEnd_CreateSchemaConflict(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	server, err := NewServerWithDB(db)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		if err := http.ListenAndServe(":8082", server); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	baseURL := "http://localhost:8082/api/v1"

	// Create tenant
	createTenantReq := map[string]interface{}{
		"name": "Conflict Test Tenant",
	}
	tenantResp := makeRequest(t, "POST", baseURL+"/tenants", createTenantReq)
	tenantID := tenantResp["id"].(string)

	// Create schema
	createSchemaReq := map[string]interface{}{
		"definition": map[string]interface{}{
			"User": map[string]interface{}{
				"Age": "int",
			},
		},
	}
	makeRequest(t, "POST", baseURL+"/tenants/"+tenantID+"/schema", createSchemaReq)

	// Try to create schema again - should get 409 Conflict
	t.Log("Attempting to create schema again (should fail)...")
	resp, err := makeHTTPRequest("POST", baseURL+"/tenants/"+tenantID+"/schema", createSchemaReq)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected 409 Conflict, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Conflict response: %s", string(body))
}

// Helper function to make HTTP requests with JSON body
func makeRequest(t *testing.T, method, url string, body interface{}) map[string]interface{} {
	resp, err := makeHTTPRequest(method, url, body)
	if err != nil {
		t.Fatalf("Failed to make %s request to %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return result
}

// Helper function to make HTTP requests without body
func makeRequestNoBody(t *testing.T, method, url string) map[string]interface{} {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make %s request to %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return result
}

// Helper function to make raw HTTP requests
func makeHTTPRequest(method, url string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}
