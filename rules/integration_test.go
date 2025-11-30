//go:build integration
// +build integration

package rules_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/liamcoop/rules/rules"

	_ "github.com/lib/pq"
)

// setupTestDB creates a PostgreSQL container and returns a connection
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "rules_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithStartupTimeout(60 * time.Second),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	host, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=test password=test dbname=rules_test sslmode=disable", host, port.Port())

	// Wait for connection to be available
	var db *sql.DB
	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	migrationSQL, err := os.ReadFile(filepath.Join("..", "migrations", "000001_initial_schema.up.sql"))
	if err != nil {
		// Try without the ../ prefix
		migrationSQL, err = os.ReadFile(filepath.Join("migrations", "000001_initial_schema.up.sql"))
		if err != nil {
			t.Fatalf("Failed to read migration file: %v", err)
		}
	}

	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		db.Close()
		postgresContainer.Terminate(ctx)
	}

	return db, cleanup
}

// createTenant helper function to create a tenant in the database
func createTenant(t *testing.T, db *sql.DB, name string) string {
	var tenantID string
	err := db.QueryRow(`
		INSERT INTO tenants (name) VALUES ($1) RETURNING id
	`, name).Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenantID
}

func TestPostgresRuleStore_BasicCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := createTenant(t, db, "test-tenant")
	store := rules.NewPostgresRuleStore(db, tenantID)

	// Test Add
	ruleID := uuid.New().String()
	rule := &rules.Rule{
		ID:         ruleID,
		Name:       "test-rule",
		Expression: "User.Age >= 18",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := store.Add(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Test Get
	retrieved, err := store.Get(ruleID)
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}
	if retrieved.Name != "test-rule" {
		t.Errorf("Expected name 'test-rule', got '%s'", retrieved.Name)
	}
	if retrieved.Expression != "User.Age >= 18" {
		t.Errorf("Expected expression 'User.Age >= 18', got '%s'", retrieved.Expression)
	}

	// Test ListActive
	activeRules, err := store.ListActive()
	if err != nil {
		t.Fatalf("Failed to list active rules: %v", err)
	}
	if len(activeRules) != 1 {
		t.Errorf("Expected 1 active rule, got %d", len(activeRules))
	}

	// Test Update
	rule.Name = "updated-rule"
	rule.Active = false
	err = store.Update(rule)
	if err != nil {
		t.Fatalf("Failed to update rule: %v", err)
	}

	updated, err := store.Get(ruleID)
	if err != nil {
		t.Fatalf("Failed to get updated rule: %v", err)
	}
	if updated.Name != "updated-rule" {
		t.Errorf("Expected name 'updated-rule', got '%s'", updated.Name)
	}
	if updated.Active {
		t.Error("Expected rule to be inactive after update")
	}

	// Verify it's not in active list
	activeRules, err = store.ListActive()
	if err != nil {
		t.Fatalf("Failed to list active rules: %v", err)
	}
	if len(activeRules) != 0 {
		t.Errorf("Expected 0 active rules, got %d", len(activeRules))
	}

	// Test Delete
	err = store.Delete(ruleID)
	if err != nil {
		t.Fatalf("Failed to delete rule: %v", err)
	}

	_, err = store.Get(ruleID)
	if err == nil {
		t.Error("Expected error when getting deleted rule, got nil")
	}
}

func TestPostgresRuleStore_TenantIsolation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create two tenants
	tenantA := createTenant(t, db, "tenant-a")
	tenantB := createTenant(t, db, "tenant-b")

	storeA := rules.NewPostgresRuleStore(db, tenantA)
	storeB := rules.NewPostgresRuleStore(db, tenantB)

	// Add rules for tenant A
	ruleAID := uuid.New().String()
	ruleA := &rules.Rule{
		ID:         ruleAID,
		Name:       "tenant-a-rule",
		Expression: "User.Age >= 18",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err := storeA.Add(ruleA)
	if err != nil {
		t.Fatalf("Failed to add rule for tenant A: %v", err)
	}

	// Add rules for tenant B
	ruleBID := uuid.New().String()
	ruleB := &rules.Rule{
		ID:         ruleBID,
		Name:       "tenant-b-rule",
		Expression: "Transaction.Amount > 1000",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = storeB.Add(ruleB)
	if err != nil {
		t.Fatalf("Failed to add rule for tenant B: %v", err)
	}

	// Verify tenant A can't see tenant B's rules
	_, err = storeA.Get(ruleBID)
	if err == nil {
		t.Error("Tenant A should not be able to see tenant B's rule")
	}

	// Verify tenant B can't see tenant A's rules
	_, err = storeB.Get(ruleAID)
	if err == nil {
		t.Error("Tenant B should not be able to see tenant A's rule")
	}

	// Verify each tenant sees only their own rules
	rulesA, err := storeA.ListActive()
	if err != nil {
		t.Fatalf("Failed to list rules for tenant A: %v", err)
	}
	if len(rulesA) != 1 {
		t.Errorf("Expected tenant A to have 1 rule, got %d", len(rulesA))
	}
	if rulesA[0].Name != "tenant-a-rule" {
		t.Errorf("Expected tenant A rule name 'tenant-a-rule', got '%s'", rulesA[0].Name)
	}

	rulesB, err := storeB.ListActive()
	if err != nil {
		t.Fatalf("Failed to list rules for tenant B: %v", err)
	}
	if len(rulesB) != 1 {
		t.Errorf("Expected tenant B to have 1 rule, got %d", len(rulesB))
	}
	if rulesB[0].Name != "tenant-b-rule" {
		t.Errorf("Expected tenant B rule name 'tenant-b-rule', got '%s'", rulesB[0].Name)
	}
}

func TestPostgresRuleStore_DuplicateRuleID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := createTenant(t, db, "test-tenant")
	store := rules.NewPostgresRuleStore(db, tenantID)

	ruleID := uuid.New().String()
	rule := &rules.Rule{
		ID:         ruleID,
		Name:       "test-rule",
		Expression: "User.Age >= 18",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Add first rule
	err := store.Add(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Try to add duplicate
	err = store.Add(rule)
	if err == nil {
		t.Error("Expected error when adding duplicate rule, got nil")
	}
}

func TestPostgresRuleStore_UpdateNonExistent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := createTenant(t, db, "test-tenant")
	store := rules.NewPostgresRuleStore(db, tenantID)

	ruleID := uuid.New().String()
	rule := &rules.Rule{
		ID:         ruleID,
		Name:       "test-rule",
		Expression: "User.Age >= 18",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := store.Update(rule)
	if err == nil {
		t.Error("Expected error when updating non-existent rule, got nil")
	}
}

func TestPostgresRuleStore_DeleteNonExistent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := createTenant(t, db, "test-tenant")
	store := rules.NewPostgresRuleStore(db, tenantID)

	nonExistentID := uuid.New().String()
	err := store.Delete(nonExistentID)
	if err == nil {
		t.Error("Expected error when deleting non-existent rule, got nil")
	}
}

func TestMultiTenantEngine_WithDatabase(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create two tenants
	tenantA := createTenant(t, db, "tenant-a")
	tenantB := createTenant(t, db, "tenant-b")

	// Create stores
	storeA := rules.NewPostgresRuleStore(db, tenantA)
	storeB := rules.NewPostgresRuleStore(db, tenantB)

	// Create engines
	engineA, err := rules.NewEngine(storeA)
	if err != nil {
		t.Fatalf("Failed to create engine A: %v", err)
	}

	engineB, err := rules.NewEngine(storeB)
	if err != nil {
		t.Fatalf("Failed to create engine B: %v", err)
	}

	// Add rules for tenant A
	ruleAID := uuid.New().String()
	ruleA := &rules.Rule{
		ID:         ruleAID,
		Name:       "adult-check",
		Expression: "User.Age >= 18",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = engineA.AddRule(ruleA)
	if err != nil {
		t.Fatalf("Failed to add rule to engine A: %v", err)
	}

	// Add rules for tenant B
	ruleBID := uuid.New().String()
	ruleB := &rules.Rule{
		ID:         ruleBID,
		Name:       "large-transaction",
		Expression: "Transaction.Amount > 1000.0",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = engineB.AddRule(ruleB)
	if err != nil {
		t.Fatalf("Failed to add rule to engine B: %v", err)
	}

	// Evaluate for tenant A
	factsA := map[string]interface{}{
		"User": map[string]interface{}{
			"Age": 25,
		},
	}
	resultA, err := engineA.Evaluate(ruleAID, factsA)
	if err != nil {
		t.Fatalf("Failed to evaluate rule A: %v", err)
	}
	if !resultA.Matched {
		t.Error("Expected rule A to match for adult user")
	}

	// Evaluate for tenant B
	factsB := map[string]interface{}{
		"Transaction": map[string]interface{}{
			"Amount": 1500.0,
		},
	}
	resultB, err := engineB.Evaluate(ruleBID, factsB)
	if err != nil {
		t.Fatalf("Failed to evaluate rule B: %v", err)
	}
	if !resultB.Matched {
		t.Error("Expected rule B to match for large transaction")
	}

	// Verify tenant A can't evaluate tenant B's rule
	_, err = engineA.Evaluate(ruleBID, factsB)
	if err == nil {
		t.Error("Tenant A should not be able to evaluate tenant B's rule")
	}

	// Verify tenant B can't evaluate tenant A's rule
	_, err = engineB.Evaluate(ruleAID, factsA)
	if err == nil {
		t.Error("Tenant B should not be able to evaluate tenant A's rule")
	}
}

func TestCascadingDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := createTenant(t, db, "test-tenant")
	store := rules.NewPostgresRuleStore(db, tenantID)

	// Add a rule
	ruleID := uuid.New().String()
	rule := &rules.Rule{
		ID:         ruleID,
		Name:       "test-rule",
		Expression: "User.Age >= 18",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err := store.Add(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Delete the tenant
	_, err = db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	if err != nil {
		t.Fatalf("Failed to delete tenant: %v", err)
	}

	// Verify rule was cascade deleted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM rules WHERE tenant_id = $1", tenantID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rules: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 rules after tenant deletion, got %d", count)
	}
}

func TestRuleOrdering(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tenantID := createTenant(t, db, "test-tenant")
	store := rules.NewPostgresRuleStore(db, tenantID)

	// Add rules in specific order
	for i := 1; i <= 5; i++ {
		ruleID := uuid.New().String()
		rule := &rules.Rule{
			ID:         ruleID,
			Name:       fmt.Sprintf("rule-%d", i),
			Expression: "User.Age >= 18",
			Active:     true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err := store.Add(rule)
		if err != nil {
			t.Fatalf("Failed to add rule %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Retrieve rules
	rulesList, err := store.ListActive()
	if err != nil {
		t.Fatalf("Failed to list rules: %v", err)
	}

	if len(rulesList) != 5 {
		t.Fatalf("Expected 5 rules, got %d", len(rulesList))
	}

	// Verify rules are in order by created_at
	for i := 0; i < len(rulesList)-1; i++ {
		if rulesList[i].CreatedAt.After(rulesList[i+1].CreatedAt) {
			t.Error("Rules are not ordered by created_at ascending")
		}
	}
}
