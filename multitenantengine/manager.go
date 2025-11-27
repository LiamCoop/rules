package multitenantengine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/liamcoop/rules/rules"
)

// Schema represents a tenant's data schema
// Maps object names to field definitions
type Schema map[string]map[string]string

// TenantEngine wraps a rules.Engine with tenant-specific metadata
type TenantEngine struct {
	TenantID string
	Schema   Schema
	Engine   *rules.Engine
	mu       sync.RWMutex
}

// MultiTenantEngineManager manages engines for all tenants
type MultiTenantEngineManager struct {
	engines map[string]*TenantEngine
	db      *sql.DB
	mu      sync.RWMutex
}

// NewMultiTenantEngineManager creates a new manager instance
func NewMultiTenantEngineManager(db *sql.DB) *MultiTenantEngineManager {
	return &MultiTenantEngineManager{
		engines: make(map[string]*TenantEngine),
		db:      db,
	}
}

// CreateCELEnvFromSchema creates a CEL environment with variables defined by the schema
func CreateCELEnvFromSchema(schema Schema) (*cel.Env, error) {
	var opts []cel.EnvOption

	// Create a CEL variable for each top-level object in the schema
	// Using DynType allows flexible runtime type checking
	for objectName := range schema {
		opts = append(opts, cel.Variable(objectName, cel.DynType))
	}

	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return env, nil
}

// LoadAllTenants loads all tenants from the database and initializes their engines
func (m *MultiTenantEngineManager) LoadAllTenants() error {
	// Fetch all active tenant schemas from database
	rows, err := m.db.Query(`
		SELECT t.id, s.definition
		FROM tenants t
		JOIN schemas s ON s.tenant_id = t.id
		WHERE s.active = true
	`)
	if err != nil {
		return fmt.Errorf("failed to fetch tenants: %w", err)
	}
	defer rows.Close()

	tenantsLoaded := 0
	for rows.Next() {
		var tenantID string
		var schemaJSON []byte
		if err := rows.Scan(&tenantID, &schemaJSON); err != nil {
			return fmt.Errorf("failed to scan tenant row: %w", err)
		}

		var schema Schema
		if err := json.Unmarshal(schemaJSON, &schema); err != nil {
			return fmt.Errorf("invalid schema for tenant %s: %w", tenantID, err)
		}

		if err := m.CreateTenant(tenantID, schema); err != nil {
			return fmt.Errorf("failed to initialize tenant %s: %w", tenantID, err)
		}

		tenantsLoaded++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating tenant rows: %w", err)
	}

	return nil
}

// CreateTenant creates a new tenant engine with the given schema
func (m *MultiTenantEngineManager) CreateTenant(tenantID string, schema Schema) error {
	// Create CEL environment from schema
	env, err := CreateCELEnvFromSchema(schema)
	if err != nil {
		return fmt.Errorf("failed to create CEL env: %w", err)
	}

	// Create a custom RuleStore that filters by tenant
	store := rules.NewPostgresRuleStore(m.db, tenantID)

	// Create the engine using the schema-specific environment
	engine, err := rules.NewEngineWithEnv(env, store)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// Store in cache
	m.mu.Lock()
	m.engines[tenantID] = &TenantEngine{
		TenantID: tenantID,
		Schema:   schema,
		Engine:   engine,
	}
	m.mu.Unlock()

	return nil
}

// GetEngine retrieves the engine for a specific tenant
func (m *MultiTenantEngineManager) GetEngine(tenantID string) (*rules.Engine, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	te, exists := m.engines[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}

	return te.Engine, nil
}

// UpdateTenantSchema updates a tenant's schema and recompiles all rules
// This operation is zero-downtime: creates new engine and atomically swaps it
func (m *MultiTenantEngineManager) UpdateTenantSchema(tenantID string, newSchema Schema) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingEngine, exists := m.engines[tenantID]
	if !exists {
		m.mu.Unlock()
		defer m.mu.Lock()
		return m.CreateTenant(tenantID, newSchema)
	}

	// Step 1: Save new schema to database
	_, err := m.db.Exec(`
		UPDATE schemas
		SET active = false
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return fmt.Errorf("failed to deactivate old schemas: %w", err)
	}

	schemaJSON, err := json.Marshal(newSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	var newVersion int
	err = m.db.QueryRow(`
		INSERT INTO schemas (tenant_id, version, definition, active, created_at)
		SELECT $1, COALESCE(MAX(version), 0) + 1, $2, true, NOW()
		FROM schemas
		WHERE tenant_id = $1
		RETURNING version
	`, tenantID, schemaJSON).Scan(&newVersion)
	if err != nil {
		return fmt.Errorf("failed to save new schema: %w", err)
	}

	// Step 2: Create new CEL environment
	env, err := CreateCELEnvFromSchema(newSchema)
	if err != nil {
		return fmt.Errorf("failed to create new CEL env: %w", err)
	}

	// Step 3: Create new Engine instance
	store := rules.NewPostgresRuleStore(m.db, tenantID)
	newEngine, err := rules.NewEngineWithEnv(env, store)
	if err != nil {
		return fmt.Errorf("failed to create new engine: %w", err)
	}

	// Step 4: Get compilation stats
	activeRules, err := store.ListActive()
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	// Step 5: Atomically swap the engine
	m.engines[tenantID] = &TenantEngine{
		TenantID: tenantID,
		Schema:   newSchema,
		Engine:   newEngine,
	}

	// Log success (in production, use structured logging)
	_ = existingEngine // Keep reference to avoid breaking change detection
	_ = activeRules

	return nil
}

// ListTenants returns all loaded tenant IDs
func (m *MultiTenantEngineManager) ListTenants() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenants := make([]string, 0, len(m.engines))
	for tenantID := range m.engines {
		tenants = append(tenants, tenantID)
	}
	return tenants
}

// DeleteTenant removes a tenant's engine from the cache
// Note: This does not delete the tenant from the database
func (m *MultiTenantEngineManager) DeleteTenant(tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.engines[tenantID]; !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	delete(m.engines, tenantID)
	return nil
}
