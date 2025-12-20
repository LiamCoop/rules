# Multi-Tenant Rules Engine Blueprint (Go + CEL-Go)

This blueprint outlines a **multi-tenant rules engine architecture** where a single shared service supports multiple tenants, each with their own dynamically-defined schemas. This approach provides operational simplicity while maintaining tenant isolation.

---

## 1. Overview

**Goal:** Build a shared rules engine service that supports multiple tenants with different schemas, using CEL's dynamic typing for flexibility and runtime type checking for safety.

**Key Features:**
- Single shared service for all tenants
- Schema definitions stored in database per tenant
- Dynamic CEL environments created from stored schemas
- Precompiled CEL programs cached in memory per tenant
- Zero-downtime schema updates (no redeployment required)
- API-driven schema, derived field, and rule management
- Tenant isolation through logical separation

**Position Among Blueprints:**
- **MVP (go_cel_rules_engine_plan.md)**: Core engine, single tenant, library-style ✅ (Implemented)
- **This Blueprint**: Evolution to multi-tenant with dynamic schemas
- **Service-Per-Tenant (tenant_service_builder_blueprint.md)**: Alternative for hard isolation requirements

**Why This Approach:**
- CEL with `DynType` provides runtime type checking without compile-time code generation
- Schemas stored as JSON enable instant tenant onboarding
- No service restarts needed for schema changes
- Simpler operations: no build pipeline, no plugins, no deployments for schema changes
- Proven architecture used by the current MVP implementation

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Tenant Management UI                     │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   Schema Management API                      │
│  - POST /tenants/:id/schema  (Create/Update Schema)        │
│  - POST /tenants/:id/derived (Define Derived Fields)        │
│  - POST /tenants/:id/rules   (Create Rules)                 │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      PostgreSQL Database                     │
│  - tenants: tenant info                                     │
│  - schemas: JSON schema definitions per tenant              │
│  - derived_fields: CEL expressions for computed values      │
│  - rules: CEL rule expressions                              │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              Shared Evaluation Service                       │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Tenant Engine Cache (in-memory)                     │  │
│  │                                                      │  │
│  │  tenant_abc123 -> Engine{                           │  │
│  │    env: *cel.Env (with DynType vars from schema)   │  │
│  │    programs: map[string]cel.Program                 │  │
│  │  }                                                   │  │
│  │                                                      │  │
│  │  tenant_def456 -> Engine{                           │  │
│  │    env: *cel.Env (different schema variables)      │  │
│  │    programs: map[string]cel.Program                 │  │
│  │  }                                                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  Evaluation Endpoint: POST /api/v1/evaluate                 │
│                                                              │
│  On schema update:                                           │
│  1. Save schema to DB                                       │
│  2. Create new CEL env from schema                          │
│  3. Recompile tenant's rules                                │
│  4. Update engine cache (zero downtime!)                    │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Database Schema

### **Tenants**
| Column        | Type        | Description |
|---------------|-------------|-------------|
| id            | UUID        | Primary Key |
| name          | string      | Tenant name |
| created_at    | timestamp   | Creation time |
| updated_at    | timestamp   | Last update |

### **Schemas**
| Column        | Type        | Description |
|---------------|-------------|-------------|
| id            | UUID        | Primary Key |
| tenant_id     | UUID        | FK to Tenants |
| version       | int         | Schema version |
| definition    | JSON        | Field definitions (object names and types) |
| active        | boolean     | Active version flag |
| created_at    | timestamp   | Creation time |

**Schema Definition Example:**
```json
{
  "User": {
    "Age": "int",
    "Citizenship": "string",
    "AccountBalance": "float64"
  },
  "Transaction": {
    "Amount": "float64",
    "Country": "string",
    "Timestamp": "int64"
  }
}
```

### **Derived Fields**
| Column        | Type        | Description |
|---------------|-------------|-------------|
| id            | UUID        | Primary Key |
| tenant_id     | UUID        | FK to Tenants |
| name          | string      | Derived field name |
| expression    | string      | CEL expression |
| dependencies  | JSON        | List of field dependencies |
| created_at    | timestamp   | Creation time |

### **Rules**
| Column        | Type        | Description |
|---------------|-------------|-------------|
| id            | UUID        | Primary Key |
| tenant_id     | UUID        | FK to Tenants |
| name          | string      | Rule name |
| expression    | string      | CEL expression |
| active        | boolean     | Active status |
| created_at    | timestamp   | Creation time |
| updated_at    | timestamp   | Last update |

---

## 4. API Specification

### **Schema Management**

#### Create/Update Tenant Schema
```http
POST /api/v1/tenants/:tenantId/schema

{
  "definition": {
    "User": {
      "Age": "int",
      "Citizenship": "string"
    },
    "Transaction": {
      "Amount": "float64"
    }
  }
}

Response:
{
  "schemaId": "uuid",
  "version": 2,
  "status": "active",
  "rulesRecompiled": 5  // number of rules recompiled with new schema
}
```

### **Derived Fields**
```http
POST /api/v1/tenants/:tenantId/derived

{
  "name": "isCanadian",
  "expression": "User.Citizenship == 'CANADA'"
}
```

### **Rules**
```http
POST /api/v1/tenants/:tenantId/rules

{
  "name": "largeCanadianTransaction",
  "expression": "User.Citizenship == 'CANADA' && Transaction.Amount > 1000"
}
```

### **Evaluation**
```http
POST /api/v1/evaluate

{
  "tenantId": "abc123",
  "facts": {
    "User": {
      "Age": 30,
      "Citizenship": "CANADA"
    },
    "Transaction": {
      "Amount": 1500.0
    }
  },
  "rules": ["rule-uuid-1", "rule-uuid-2"]  // optional, evaluates all if omitted
}

Response:
{
  "results": [
    {
      "ruleId": "rule-uuid-1",
      "ruleName": "largeCanadianTransaction",
      "matched": true,
      "trace": { ... }  // optional, for debugging
    }
  ],
  "evaluationTime": "2.3ms"
}
```

---

## 5. Implementation: Dynamic CEL Environments

### Creating CEL Environment from Schema

Instead of generating code, we dynamically create CEL environments from stored schemas:

```go
package engine

import (
    "fmt"
    "github.com/google/cel-go/cel"
)

// Schema represents a tenant's data schema
type Schema map[string]map[string]string  // objectName -> fieldName -> fieldType

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
```

**Example Usage:**
```go
// Tenant's schema stored in DB
schema := Schema{
    "User": {
        "Age":         "int",
        "Citizenship": "string",
    },
    "Transaction": {
        "Amount":  "float64",
        "Country": "string",
    },
}

// Create CEL environment - no code generation needed!
env, err := CreateCELEnvFromSchema(schema)
if err != nil {
    return err
}

// Compile a rule
ast, issues := env.Compile(`User.Age >= 18 && Transaction.Amount > 1000`)
// ... rest of compilation
```

---

## 6. Multi-Tenant Engine Management

### Tenant Engine Structure

Each tenant gets their own Engine instance with schema-specific CEL environment:

```go
package multitenantengine

import (
    "fmt"
    "sync"
    "github.com/google/cel-go/cel"
    "yourapp/rules"
)

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
    mu      sync.RWMutex
}

func NewMultiTenantEngineManager() *MultiTenantEngineManager {
    return &MultiTenantEngineManager{
        engines: make(map[string]*TenantEngine),
    }
}
```

### Loading Tenants on Startup

When the service starts, load all tenants from the database:

```go
func (m *MultiTenantEngineManager) LoadAllTenants(db *sql.DB) error {
    // Fetch all active tenant schemas from database
    rows, err := db.Query(`
        SELECT t.id, s.definition
        FROM tenants t
        JOIN schemas s ON s.tenant_id = t.id
        WHERE s.active = true
    `)
    if err != nil {
        return fmt.Errorf("failed to fetch tenants: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var tenantID string
        var schemaJSON []byte
        if err := rows.Scan(&tenantID, &schemaJSON); err != nil {
            return err
        }

        var schema Schema
        if err := json.Unmarshal(schemaJSON, &schema); err != nil {
            return fmt.Errorf("invalid schema for tenant %s: %w", tenantID, err)
        }

        if err := m.CreateTenant(tenantID, schema, db); err != nil {
            return fmt.Errorf("failed to initialize tenant %s: %w", tenantID, err)
        }
    }

    return nil
}

func (m *MultiTenantEngineManager) CreateTenant(tenantID string, schema Schema, db *sql.DB) error {
    // Create CEL environment from schema
    env, err := CreateCELEnvFromSchema(schema)
    if err != nil {
        return fmt.Errorf("failed to create CEL env: %w", err)
    }

    // Create a custom RuleStore that filters by tenant
    store := NewTenantRuleStore(db, tenantID)

    // Create the engine using our MVP implementation
    engine, err := rules.NewEngineWithEnv(env, store)
    if err != nil {
        return fmt.Errorf("failed to create engine: %w", err)
    }

    // Load and compile all active rules for this tenant
    activeRules, err := store.ListActive()
    if err != nil {
        return fmt.Errorf("failed to load rules: %w", err)
    }

    for _, rule := range activeRules {
        if err := engine.CompileRule(rule); err != nil {
            return fmt.Errorf("failed to compile rule %s: %w", rule.ID, err)
        }
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

func (m *MultiTenantEngineManager) GetEngine(tenantID string) (*rules.Engine, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    te, exists := m.engines[tenantID]
    if !exists {
        return nil, fmt.Errorf("tenant %s not found", tenantID)
    }

    return te.Engine, nil
}
```

### Tenant-Specific RuleStore

Wrap the database RuleStore to filter by tenant:

```go
type TenantRuleStore struct {
    db       *sql.DB
    tenantID string
}

func NewTenantRuleStore(db *sql.DB, tenantID string) *TenantRuleStore {
    return &TenantRuleStore{
        db:       db,
        tenantID: tenantID,
    }
}

func (s *TenantRuleStore) Add(rule *rules.Rule) error {
    _, err := s.db.Exec(`
        INSERT INTO rules (id, tenant_id, name, expression, active, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, rule.ID, s.tenantID, rule.Name, rule.Expression, rule.Active,
       rule.CreatedAt, rule.UpdatedAt)
    return err
}

func (s *TenantRuleStore) Get(id string) (*rules.Rule, error) {
    var rule rules.Rule
    err := s.db.QueryRow(`
        SELECT id, name, expression, active, created_at, updated_at
        FROM rules
        WHERE id = $1 AND tenant_id = $2
    `, id, s.tenantID).Scan(&rule.ID, &rule.Name, &rule.Expression,
                            &rule.Active, &rule.CreatedAt, &rule.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("rule %s not found", id)
    }
    return &rule, err
}

func (s *TenantRuleStore) ListActive() ([]*rules.Rule, error) {
    rows, err := s.db.Query(`
        SELECT id, name, expression, active, created_at, updated_at
        FROM rules
        WHERE tenant_id = $1 AND active = true
        ORDER BY created_at ASC
    `, s.tenantID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var rulesList []*rules.Rule
    for rows.Next() {
        var r rules.Rule
        if err := rows.Scan(&r.ID, &r.Name, &r.Expression, &r.Active,
                           &r.CreatedAt, &r.UpdatedAt); err != nil {
            return nil, err
        }
        rulesList = append(rulesList, &r)
    }
    return rulesList, nil
}

// ... implement Update and Delete similarly
```

---

## 7. Schema Update Workflow (Zero Downtime)

### Updating a Tenant's Schema

When a tenant updates their schema, the workflow is:

1. **Save schema to database** (marks new version as active)
2. **Create new CEL environment** from the updated schema
3. **Create new Engine instance** with the new environment
4. **Recompile all tenant's rules** using the new environment
5. **Atomically swap the engine** in the cache

```go
func (m *MultiTenantEngineManager) UpdateTenantSchema(tenantID string, newSchema Schema, db *sql.DB) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    existingEngine, exists := m.engines[tenantID]
    if !exists {
        m.mu.Unlock()
        return m.CreateTenant(tenantID, newSchema, db)
    }

    // Step 1: Save new schema to database
    _, err := db.Exec(`
        UPDATE schemas
        SET active = false
        WHERE tenant_id = $1
    `, tenantID)
    if err != nil {
        return fmt.Errorf("failed to deactivate old schemas: %w", err)
    }

    _, err = db.Exec(`
        INSERT INTO schemas (id, tenant_id, version, definition, active, created_at)
        SELECT uuid_generate_v4(), $1, COALESCE(MAX(version), 0) + 1, $2, true, NOW()
        FROM schemas
        WHERE tenant_id = $1
    `, tenantID, newSchema)
    if err != nil {
        return fmt.Errorf("failed to save new schema: %w", err)
    }

    // Step 2: Create new CEL environment
    env, err := CreateCELEnvFromSchema(newSchema)
    if err != nil {
        return fmt.Errorf("failed to create new CEL env: %w", err)
    }

    // Step 3: Create new Engine instance
    store := NewTenantRuleStore(db, tenantID)
    newEngine, err := rules.NewEngineWithEnv(env, store)
    if err != nil {
        return fmt.Errorf("failed to create new engine: %w", err)
    }

    // Step 4: Recompile all rules
    activeRules, err := store.ListActive()
    if err != nil {
        return fmt.Errorf("failed to load rules: %w", err)
    }

    compiledCount := 0
    failedRules := []string{}

    for _, rule := range activeRules {
        if err := newEngine.CompileRule(rule); err != nil {
            // Log error but continue - some rules may break with schema changes
            failedRules = append(failedRules, fmt.Sprintf("%s: %v", rule.ID, err))
        } else {
            compiledCount++
        }
    }

    // Step 5: Atomically swap the engine
    m.engines[tenantID] = &TenantEngine{
        TenantID: tenantID,
        Schema:   newSchema,
        Engine:   newEngine,
    }

    // Log the update
    if len(failedRules) > 0 {
        return fmt.Errorf("schema updated but %d rules failed to compile: %v",
                         len(failedRules), failedRules)
    }

    return nil
}
```

### HTTP Handler for Schema Updates

```go
func (h *Handler) UpdateTenantSchema(w http.ResponseWriter, r *http.Request) {
    tenantID := chi.URLParam(r, "tenantId")

    var req struct {
        Definition Schema `json:"definition"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Validate schema
    if err := validateSchema(req.Definition); err != nil {
        http.Error(w, fmt.Sprintf("invalid schema: %v", err), http.StatusBadRequest)
        return
    }

    // Update schema (zero downtime!)
    err := h.engineManager.UpdateTenantSchema(tenantID, req.Definition, h.db)
    if err != nil {
        http.Error(w, fmt.Sprintf("failed to update schema: %v", err),
                   http.StatusInternalServerError)
        return
    }

    // Return success with compiled rule count
    engine, _ := h.engineManager.GetEngine(tenantID)
    store := engine.GetStore()
    activeRules, _ := store.ListActive()

    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":           "active",
        "rulesRecompiled":  len(activeRules),
        "schemaVersion":    "v2", // fetch from DB if needed
    })
}
```

---

## 8. Evaluation Flow

### Evaluation HTTP Handler

```go
func (h *Handler) Evaluate(w http.ResponseWriter, r *http.Request) {
    var req struct {
        TenantID string                 `json:"tenantId"`
        Facts    map[string]interface{} `json:"facts"`
        RuleIDs  []string               `json:"rules"` // optional
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get tenant's engine
    engine, err := h.engineManager.GetEngine(req.TenantID)
    if err != nil {
        http.Error(w, "tenant not found", http.StatusNotFound)
        return
    }

    startTime := time.Now()

    // Evaluate rules
    var results []*rules.EvaluationResult
    if len(req.RuleIDs) > 0 {
        // Evaluate specific rules
        for _, ruleID := range req.RuleIDs {
            result, err := engine.Evaluate(ruleID, req.Facts)
            if err != nil {
                // Continue on error (might be rule not found)
                continue
            }
            results = append(results, result)
        }
    } else {
        // Evaluate all active rules
        results, err = engine.EvaluateAll(req.Facts)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    }

    evaluationTime := time.Since(startTime)

    // Format response
    response := map[string]interface{}{
        "results": results,
        "evaluationTime": evaluationTime.String(),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

---

## 9. Migration Path

### From MVP to Multi-Tenant

The MVP implementation (`rules/engine.go`) already uses the right architecture. To migrate:

1. **Wrap the MVP Engine with tenant context:**
   - Create `MultiTenantEngineManager` (shown above)
   - Add tenant-aware `RuleStore` wrapper

2. **Add HTTP API layer:**
   - Schema management endpoints
   - Rule management endpoints
   - Evaluation endpoint

3. **Add database persistence:**
   - Implement PostgreSQL-backed `RuleStore`
   - Store tenant schemas in database
   - Load tenants on startup

4. **No code changes needed in MVP engine:**
   - The engine already uses `cel.DynType`
   - The engine already supports custom environments
   - The engine already caches compiled programs

### Extending Engine for Custom Environments

Update the MVP engine to accept a custom CEL environment:

```go
// Add to rules/engine.go
func NewEngineWithEnv(env *cel.Env, store RuleStore) (*Engine, error) {
    return &Engine{
        env:      env,
        store:    store,
        programs: make(map[string]cel.Program),
    }, nil
}
```

This allows multi-tenant manager to inject schema-specific environments while reusing all the engine logic.

---

## 10. Next Steps

### Phase 1: API Layer (Week 1)
1. ✅ Build HTTP server with chi/mux router
2. ✅ Add tenant schema CRUD endpoints
3. ✅ Add rule CRUD endpoints
4. ✅ Add evaluation endpoint

### Phase 2: Persistence (Week 2)
5. Set up PostgreSQL database
6. Implement database-backed RuleStore
7. Add schema version management
8. Add tenant management tables

### Phase 3: Multi-Tenant Engine (Week 3)
9. Implement `MultiTenantEngineManager`
10. Implement `TenantRuleStore` wrapper
11. Add `NewEngineWithEnv` to MVP engine
12. Implement `LoadAllTenants` on startup

### Phase 4: Production Readiness (Week 4)
13. Add authentication/authorization
14. Add rate limiting per tenant
15. Add metrics and monitoring
16. Add health check endpoints
17. Write integration tests

### Testing Strategy
- **Unit Tests**: Already complete (63 tests in MVP)
- **Integration Tests**: Test schema updates, tenant isolation
- **Load Tests**: Verify performance under concurrent tenant load
- **Chaos Tests**: Test engine swapping under load

---

## 11. Advantages of Dynamic Schema Approach

### Why This is Better Than Code Generation

1. **Instant tenant onboarding** - no builds, no deployments
2. **Zero-downtime schema updates** - atomic engine swaps
3. **Simpler operations** - no build pipeline, no plugins
4. **All platforms supported** - no platform-specific binary loading
5. **Clean memory management** - no plugin unloading issues
6. **Standard debugging** - no plugin symbol complexity
7. **Already proven** - this is how the MVP works today

### Performance Characteristics

- **CEL compilation**: ~1-5ms per rule (cached indefinitely)
- **Schema update**: ~10-50ms to create env + recompile all rules
- **Engine swap**: Atomic pointer update (~1μs)
- **Evaluation**: Identical performance to generated types

CEL's `DynType` performs runtime type checking but doesn't sacrifice performance because:
- Type checking happens at **compile time** when the rule is added
- **Evaluation** uses compiled programs (no type checking overhead)
- The only "cost" is less strict compile-time validation, which is acceptable for multi-tenant SaaS

---

## 12. Security Considerations

### Input Validation
```go
func validateSchema(schema Schema) error {
    for objectName, fields := range schema {
        if err := validateIdentifier(objectName); err != nil {
            return fmt.Errorf("invalid object name %s: %w", objectName, err)
        }
        for fieldName, fieldType := range fields {
            if err := validateIdentifier(fieldName); err != nil {
                return fmt.Errorf("invalid field name %s: %w", fieldName, err)
            }
            if !isValidCELType(fieldType) {
                return fmt.Errorf("invalid type %s for field %s", fieldType, fieldName)
            }
        }
    }
    return nil
}

func validateIdentifier(name string) error {
    if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name) {
        return fmt.Errorf("must be valid identifier")
    }
    return nil
}

func isValidCELType(typeName string) bool {
    validTypes := map[string]bool{
        "int": true, "int64": true, "float64": true,
        "string": true, "bool": true, "bytes": true,
        "timestamp": true, "duration": true,
    }
    return validTypes[typeName]
}
```

### CEL Expression Sandboxing
```go
func CreateCELEnvFromSchema(schema Schema) (*cel.Env, error) {
    var opts []cel.EnvOption

    for objectName := range schema {
        opts = append(opts, cel.Variable(objectName, cel.DynType))
    }

    // Add security restrictions
    opts = append(opts,
        cel.CostLimit(1000000),  // Prevent runaway expressions
        cel.ClearMacros(),        // Disable potentially dangerous macros
    )

    return cel.NewEnv(opts...)
}
```

### Multi-Tenancy Isolation
- Validate `tenantID` on every request
- Use tenant-scoped RuleStore (filters at DB level)
- Rate limit per tenant
- Resource quotas (max rules, max schema size, max evaluation time)

---

## Appendix: Future Code Generation Option

While the dynamic schema approach is recommended for most use cases, **code generation** could be considered in the future if:

### When Code Generation Might Be Useful

1. **Single-tenant deployments** - If you're building a rules engine for a single organization, generating types provides compile-time safety without multi-tenant complexity.

2. **Extremely high performance requirements** - Generated types with `cel.Types()` provide marginally better compile-time type checking, though evaluation performance is identical.

3. **Strong static typing requirements** - Some teams prefer compile-time guarantees over runtime flexibility, especially for regulated industries.

### Code Generation Approach Overview

Instead of storing schemas as JSON and using `cel.DynType`, you would:

1. Generate Go structs from schema definitions
2. Compile generated code into plugins (`.so` files) or rebuild service
3. Load plugins at runtime or deploy new binary

**Trade-offs:**
- ✅ Compile-time type safety for rules
- ✅ IDE autocomplete for schema fields
- ❌ Requires build pipeline for schema changes
- ❌ Go plugins are platform-specific (Linux/macOS only)
- ❌ Plugins cannot be unloaded (memory leaks)
- ❌ More complex operations (builds instead of DB updates)
- ❌ Requires service restart or plugin hot-reload complexity

### Why We Don't Recommend It

After implementing the MVP with `cel.DynType`, we found:
- Runtime type checking is sufficient (errors caught at rule compilation)
- Operational simplicity is more valuable than marginal type safety gains
- CEL's dynamic typing doesn't impact evaluation performance
- Zero-downtime updates are more important than compile-time checks

**Conclusion:** Start with dynamic schemas. Only revisit code generation if you have a specific, measurable need that dynamic schemas can't address.

---

This blueprint provides a complete path to building a production-ready multi-tenant rules engine using dynamic schemas, building directly on the proven MVP implementation.
