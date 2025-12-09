//go:build integration

package multitenantengine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/liamcoop/rules/rules"
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
	migrationSQL, err := os.ReadFile("../migrations/000001_initial_schema.up.sql")
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

// createTenantWithSchema creates a tenant and schema in the database
func createTenantWithSchema(t *testing.T, db *sql.DB, tenantID string, schema Schema) {
	// Insert tenant
	_, err := db.Exec(`
		INSERT INTO tenants (id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, tenantID, tenantID+"-name")
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Insert schema
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO schemas (tenant_id, version, definition, active, created_at)
		VALUES ($1, 1, $2, true, NOW())
	`, tenantID, schemaJSON)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
}

// TestCreateCELEnvFromSchema verifies CEL environment creation from schema definitions
// Maps to: REQ-COMPILE-001 (CEL Environment Creation)
func TestCreateCELEnvFromSchema(t *testing.T) {
	tests := []struct {
		name          string
		schema        Schema
		testExpr      string
		shouldCompile bool
	}{
		{
			name: "single object schema",
			schema: Schema{
				"User": {
					"Age":  "int",
					"Name": "string",
				},
			},
			testExpr:      "User.Age > 18",
			shouldCompile: true,
		},
		{
			name: "multiple objects schema",
			schema: Schema{
				"User": {
					"Age": "int",
				},
				"Transaction": {
					"Amount": "double",
				},
			},
			testExpr:      "User.Age > 18 && Transaction.Amount > 100.0",
			shouldCompile: true,
		},
		{
			name:          "empty schema",
			schema:        Schema{},
			testExpr:      "true",
			shouldCompile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := CreateCELEnvFromSchema(tt.schema)
			if err != nil {
				t.Fatalf("Failed to create CEL environment: %v", err)
			}

			// Try to compile a test expression
			_, issues := env.Compile(tt.testExpr)
			if tt.shouldCompile && issues.Err() != nil {
				t.Errorf("Expected expression to compile, got error: %v", issues.Err())
			}
		})
	}
}

// TestMultiTenantEngineManager_LoadAllTenants verifies loading tenants from database
// Maps to: REQ-TENANT-004 (Multi-Tenant Engine Support)
func TestMultiTenantEngineManager_LoadAllTenants(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test tenants with schemas
	tenantA := uuid.New().String()
	schemaA := Schema{
		"User": {
			"Age": "int",
		},
	}
	createTenantWithSchema(t, db, tenantA, schemaA)

	tenantB := uuid.New().String()
	schemaB := Schema{
		"Transaction": {
			"Amount": "double",
		},
	}
	createTenantWithSchema(t, db, tenantB, schemaB)

	// Create manager and load tenants
	manager := NewMultiTenantEngineManager(db)
	err := manager.LoadAllTenants()
	if err != nil {
		t.Fatalf("Failed to load tenants: %v", err)
	}

	// Verify both tenants were loaded
	tenants := manager.ListTenants()
	if len(tenants) != 2 {
		t.Errorf("Expected 2 tenants, got %d", len(tenants))
	}

	// Verify we can get engines for both tenants
	engineA, err := manager.GetEngine(tenantA)
	if err != nil {
		t.Errorf("Failed to get engine for tenant A: %v", err)
	}
	if engineA == nil {
		t.Error("Engine A should not be nil")
	}

	engineB, err := manager.GetEngine(tenantB)
	if err != nil {
		t.Errorf("Failed to get engine for tenant B: %v", err)
	}
	if engineB == nil {
		t.Error("Engine B should not be nil")
	}
}

// TestMultiTenantEngineManager_CreateTenant verifies creating new tenants
// Maps to: REQ-TENANT-007 (Tenant Lifecycle Management)
func TestMultiTenantEngineManager_CreateTenant(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := uuid.New().String()

	// Create tenant in database first
	_, err := db.Exec(`
		INSERT INTO tenants (id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, tenantID, "test-tenant")
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	manager := NewMultiTenantEngineManager(db)

	schema := Schema{
		"User": {
			"Age":  "int",
			"Name": "string",
		},
	}

	err = manager.CreateTenant(tenantID, schema)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Verify tenant was added to manager
	engine, err := manager.GetEngine(tenantID)
	if err != nil {
		t.Errorf("Failed to get engine for new tenant: %v", err)
	}
	if engine == nil {
		t.Error("Engine should not be nil for new tenant")
	}

	// Verify tenant appears in list
	tenants := manager.ListTenants()
	found := false
	for _, id := range tenants {
		if id == tenantID {
			found = true
			break
		}
	}
	if !found {
		t.Error("New tenant should appear in tenant list")
	}
}

// TestMultiTenantEngineManager_GetEngineNotFound verifies error handling for missing tenants
// Maps to: REQ-ERROR-002 (Descriptive Errors)
func TestMultiTenantEngineManager_GetEngineNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewMultiTenantEngineManager(db)

	_, err := manager.GetEngine("nonexistent-tenant")
	if err == nil {
		t.Error("Expected error when getting nonexistent tenant")
	}

	expectedMsg := "tenant nonexistent-tenant not found"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// TestMultiTenantEngineManager_UpdateTenantSchema verifies zero-downtime schema updates
// Maps to: REQ-TENANT-004 (Multi-Tenant Engine Support)
func TestMultiTenantEngineManager_UpdateTenantSchema(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := uuid.New().String()

	// Create initial schema
	initialSchema := Schema{
		"User": {
			"Age": "int",
		},
	}
	createTenantWithSchema(t, db, tenantID, initialSchema)

	manager := NewMultiTenantEngineManager(db)
	err := manager.LoadAllTenants()
	if err != nil {
		t.Fatalf("Failed to load tenants: %v", err)
	}

	// Add a rule using the initial schema
	store := rules.NewPostgresRuleStore(db, tenantID)
	ruleID := uuid.New().String()
	rule := &rules.Rule{
		ID:         ruleID,
		Name:       "age-check",
		Expression: "User.Age >= 18",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = store.Add(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Update schema to add new field
	newSchema := Schema{
		"User": {
			"Age":  "int",
			"Name": "string",
		},
		"Transaction": {
			"Amount": "double",
		},
	}

	err = manager.UpdateTenantSchema(tenantID, newSchema)
	if err != nil {
		t.Fatalf("Failed to update tenant schema: %v", err)
	}

	// Verify old rule still works
	engine, err := manager.GetEngine(tenantID)
	if err != nil {
		t.Fatalf("Failed to get engine after update: %v", err)
	}

	facts := map[string]interface{}{
		"User": map[string]interface{}{
			"Age": 25,
		},
	}
	result, err := engine.Evaluate(ruleID, facts)
	if err != nil {
		t.Errorf("Old rule should still work after schema update: %v", err)
	}
	if !result.Matched {
		t.Error("Expected rule to match")
	}

	// Verify schema was updated in database
	var schemaJSON []byte
	err = db.QueryRow(`
		SELECT definition FROM schemas
		WHERE tenant_id = $1 AND active = true
	`, tenantID).Scan(&schemaJSON)
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}

	var savedSchema Schema
	err = json.Unmarshal(schemaJSON, &savedSchema)
	if err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	// Verify Transaction object exists in new schema
	if _, exists := savedSchema["Transaction"]; !exists {
		t.Error("Updated schema should include Transaction object")
	}
}

// TestMultiTenantEngineManager_TenantIsolation verifies tenant isolation
// Maps to: REQ-TENANT-001 (Tenant Isolation), REQ-TENANT-005 (Cross-Tenant Evaluation Prevention)
func TestMultiTenantEngineManager_TenantIsolation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create two tenants
	tenantA := uuid.New().String()
	schemaA := Schema{
		"User": {
			"Age": "int",
		},
	}
	createTenantWithSchema(t, db, tenantA, schemaA)

	tenantB := uuid.New().String()
	schemaB := Schema{
		"Transaction": {
			"Amount": "double",
		},
	}
	createTenantWithSchema(t, db, tenantB, schemaB)

	// Load tenants
	manager := NewMultiTenantEngineManager(db)
	err := manager.LoadAllTenants()
	if err != nil {
		t.Fatalf("Failed to load tenants: %v", err)
	}

	// Add rules for each tenant
	storeA := rules.NewPostgresRuleStore(db, tenantA)
	ruleAID := uuid.New().String()
	ruleA := &rules.Rule{
		ID:         ruleAID,
		Name:       "adult-check",
		Expression: "User.Age >= 18",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = storeA.Add(ruleA)
	if err != nil {
		t.Fatalf("Failed to add rule A: %v", err)
	}

	storeB := rules.NewPostgresRuleStore(db, tenantB)
	ruleBID := uuid.New().String()
	ruleB := &rules.Rule{
		ID:         ruleBID,
		Name:       "large-transaction",
		Expression: "Transaction.Amount > 1000.0",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = storeB.Add(ruleB)
	if err != nil {
		t.Fatalf("Failed to add rule B: %v", err)
	}

	// Reload engines to pick up new rules
	err = manager.LoadAllTenants()
	if err != nil {
		t.Fatalf("Failed to reload tenants: %v", err)
	}

	// Get engines
	engineA, err := manager.GetEngine(tenantA)
	if err != nil {
		t.Fatalf("Failed to get engine A: %v", err)
	}

	engineB, err := manager.GetEngine(tenantB)
	if err != nil {
		t.Fatalf("Failed to get engine B: %v", err)
	}

	// Verify tenant A can evaluate its own rule
	factsA := map[string]interface{}{
		"User": map[string]interface{}{
			"Age": 25,
		},
	}
	resultA, err := engineA.Evaluate(ruleAID, factsA)
	if err != nil {
		t.Errorf("Tenant A should be able to evaluate its own rule: %v", err)
	}
	if !resultA.Matched {
		t.Error("Expected rule A to match")
	}

	// Verify tenant B can evaluate its own rule
	factsB := map[string]interface{}{
		"Transaction": map[string]interface{}{
			"Amount": 1500.0,
		},
	}
	resultB, err := engineB.Evaluate(ruleBID, factsB)
	if err != nil {
		t.Errorf("Tenant B should be able to evaluate its own rule: %v", err)
	}
	if !resultB.Matched {
		t.Error("Expected rule B to match")
	}

	// Verify tenant A cannot evaluate tenant B's rule
	_, err = engineA.Evaluate(ruleBID, factsB)
	if err == nil {
		t.Error("Tenant A should not be able to evaluate tenant B's rule")
	}

	// Verify tenant B cannot evaluate tenant A's rule
	_, err = engineB.Evaluate(ruleAID, factsA)
	if err == nil {
		t.Error("Tenant B should not be able to evaluate tenant A's rule")
	}
}

// TestMultiTenantEngineManager_Concurrency verifies thread safety
// Maps to: REQ-CONCUR-001, REQ-CONCUR-002 (Thread-Safe Operations)
func TestMultiTenantEngineManager_Concurrency(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create initial tenant
	tenantID := uuid.New().String()
	schema := Schema{
		"User": {
			"Age": "int",
		},
	}
	createTenantWithSchema(t, db, tenantID, schema)

	manager := NewMultiTenantEngineManager(db)
	err := manager.LoadAllTenants()
	if err != nil {
		t.Fatalf("Failed to load tenants: %v", err)
	}

	// Run concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent reads (GetEngine)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.GetEngine(tenantID)
			if err != nil {
				t.Errorf("Concurrent GetEngine failed: %v", err)
			}
		}()
	}

	// Concurrent ListTenants
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.ListTenants()
		}()
	}

	wg.Wait()
}

// TestMultiTenantEngineManager_DeleteTenant verifies tenant deletion
// Maps to: REQ-TENANT-007 (Tenant Lifecycle Management)
func TestMultiTenantEngineManager_DeleteTenant(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := uuid.New().String()
	schema := Schema{
		"User": {
			"Age": "int",
		},
	}
	createTenantWithSchema(t, db, tenantID, schema)

	manager := NewMultiTenantEngineManager(db)
	err := manager.LoadAllTenants()
	if err != nil {
		t.Fatalf("Failed to load tenants: %v", err)
	}

	// Verify tenant exists
	_, err = manager.GetEngine(tenantID)
	if err != nil {
		t.Fatalf("Tenant should exist before deletion: %v", err)
	}

	// Delete tenant
	err = manager.DeleteTenant(tenantID)
	if err != nil {
		t.Fatalf("Failed to delete tenant: %v", err)
	}

	// Verify tenant no longer exists in manager
	_, err = manager.GetEngine(tenantID)
	if err == nil {
		t.Error("Tenant should not exist after deletion")
	}

	// Verify error when deleting nonexistent tenant
	err = manager.DeleteTenant("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting nonexistent tenant")
	}
}

// TestMultiTenantEngineManager_UpdateNonexistentTenant verifies creating tenant via update
// Maps to: REQ-ENGINE-001 (Engine Constructor)
func TestMultiTenantEngineManager_UpdateNonexistentTenant(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := uuid.New().String()

	// Create tenant in database
	_, err := db.Exec(`
		INSERT INTO tenants (id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, tenantID, "new-tenant")
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	manager := NewMultiTenantEngineManager(db)

	schema := Schema{
		"User": {
			"Age": "int",
		},
	}

	// Update schema for nonexistent tenant (should create it)
	err = manager.UpdateTenantSchema(tenantID, schema)
	if err != nil {
		t.Fatalf("Failed to update schema for nonexistent tenant: %v", err)
	}

	// Verify tenant was created
	engine, err := manager.GetEngine(tenantID)
	if err != nil {
		t.Errorf("Tenant should exist after schema update: %v", err)
	}
	if engine == nil {
		t.Error("Engine should not be nil")
	}
}
