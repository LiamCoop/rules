# Multi-Tenant Engine Manager Requirements

This document specifies the functional and non-functional requirements for the Multi-Tenant Engine Manager component. These requirements extend the existing MVP and persistence requirements to support dynamic schema management and engine lifecycle operations.

**Requirement Language:**
- **SHALL** = Mandatory requirement
- **SHOULD** = Recommended but not mandatory
- **MAY** = Optional

**Relationship to Other Requirements:**
- Extends `mvp_requirements.md` (REQ-ENGINE-001, REQ-CONCUR-*)
- Extends `persistence_requirements.md` (REQ-TENANT-004, REQ-TENANT-005, REQ-TENANT-007)
- Adds new requirements specific to the multi-tenant engine manager

---

## 1. CEL Environment Management Requirements

### REQ-MANAGER-001: Dynamic CEL Environment Creation
The system **SHALL** provide a `CreateCELEnvFromSchema` function that creates CEL environments from schema definitions.

**Verification:** Function accepts a Schema map and returns a valid `*cel.Env`.

**Test:** `TestCreateCELEnvFromSchema` (multitenantengine/manager_test.go:54)

**Related Requirements:**
- REQ-COMPILE-001 (CEL Environment Creation)

---

### REQ-MANAGER-002: Schema-Based Variable Registration
The CEL environment **SHALL** register a variable for each top-level object in the schema using `cel.DynType`.

**Verification:** Schema with multiple objects creates CEL environment with all objects as variables.

**Test:** `TestCreateCELEnvFromSchema/multiple_objects_schema` (multitenantengine/manager_test.go:70)

---

### REQ-MANAGER-003: Empty Schema Support
The system **SHALL** support creating CEL environments from empty schemas.

**Verification:** Empty schema creates valid CEL environment that can compile basic expressions.

**Test:** `TestCreateCELEnvFromSchema/empty_schema` (multitenantengine/manager_test.go:81)

---

## 2. Tenant Loading and Initialization Requirements

### REQ-MANAGER-004: Load All Tenants from Database
The `MultiTenantEngineManager` **SHALL** provide a `LoadAllTenants()` method that initializes engines for all tenants with active schemas.

**Verification:** Calling `LoadAllTenants()` creates engine instances for all tenants in the database.

**Test:** `TestMultiTenantEngineManager_LoadAllTenants` (multitenantengine/manager_test.go:102)

**Related Requirements:**
- REQ-TENANT-004 (Multi-Tenant Engine Support)

---

### REQ-MANAGER-005: Tenant Schema Query
The `LoadAllTenants()` method **SHALL** query the database for tenants with active schemas only.

**Verification:** SQL query joins tenants and schemas tables with `active = true` filter.

**Test:** Code review of `multitenantengine/manager.go:61-66`

---

### REQ-MANAGER-006: Schema Deserialization
The system **SHALL** deserialize JSONB schema definitions into `Schema` type for each tenant.

**Verification:** JSONB from database is unmarshaled into Schema map structure.

**Test:** `TestMultiTenantEngineManager_LoadAllTenants` (multitenantengine/manager_test.go:102)

---

## 3. Engine Retrieval Requirements

### REQ-MANAGER-007: Get Engine by Tenant ID
The `MultiTenantEngineManager` **SHALL** provide a `GetEngine(tenantID)` method that returns the engine for a specific tenant.

**Verification:** Calling `GetEngine()` with valid tenant ID returns the correct engine instance.

**Test:** `TestMultiTenantEngineManager_LoadAllTenants` (multitenantengine/manager_test.go:133-145)

---

### REQ-MANAGER-008: Engine Not Found Error
The `GetEngine()` method **SHALL** return a descriptive error when the tenant does not exist.

**Verification:** Non-existent tenant ID returns error with message "tenant {id} not found".

**Test:** `TestMultiTenantEngineManager_GetEngineNotFound` (multitenantengine/manager_test.go:152)

**Related Requirements:**
- REQ-ERROR-002 (Descriptive Errors)

---

### REQ-MANAGER-009: List All Tenants
The `MultiTenantEngineManager` **SHALL** provide a `ListTenants()` method that returns all loaded tenant IDs.

**Verification:** Method returns slice of all tenant IDs currently in the manager.

**Test:** `TestMultiTenantEngineManager_LoadAllTenants` (multitenantengine/manager_test.go:125-131)

---

## 4. Tenant Creation Requirements

### REQ-MANAGER-010: Create New Tenant
The `MultiTenantEngineManager` **SHALL** provide a `CreateTenant(tenantID, schema)` method that initializes a new tenant engine.

**Verification:** Calling `CreateTenant()` creates engine instance that can be retrieved via `GetEngine()`.

**Test:** `TestMultiTenantEngineManager_CreateTenant` (multitenantengine/manager_test.go:164)

**Related Requirements:**
- REQ-TENANT-007 (Tenant Lifecycle Management)
- REQ-ENGINE-001 (Engine Constructor)

---

### REQ-MANAGER-011: Tenant-Scoped RuleStore Creation
The `CreateTenant()` method **SHALL** create a tenant-scoped `PostgresRuleStore` for the new tenant.

**Verification:** Engine uses RuleStore that filters by tenant ID.

**Test:** Code review of `multitenantengine/manager.go:108`

**Related Requirements:**
- REQ-TENANT-002 (Tenant-Scoped Queries)

---

### REQ-MANAGER-012: Tenant Registration
The `CreateTenant()` method **SHALL** register the new tenant in the internal cache for retrieval.

**Verification:** After creation, tenant appears in `ListTenants()` results.

**Test:** `TestMultiTenantEngineManager_CreateTenant` (multitenantengine/manager_test.go:189-196)

---

## 5. Schema Update Requirements

### REQ-MANAGER-013: Zero-Downtime Schema Updates
The `MultiTenantEngineManager` **SHALL** provide an `UpdateTenantSchema()` method that updates schemas without downtime.

**Verification:** Schema update creates new engine and atomically swaps it while existing evaluations complete.

**Test:** `TestMultiTenantEngineManager_UpdateTenantSchema` (multitenantengine/manager_test.go:202)

**Related Requirements:**
- REQ-TENANT-004 (Multi-Tenant Engine Support)

---

### REQ-MANAGER-014: Schema Versioning
The `UpdateTenantSchema()` method **SHALL** increment the schema version and save to database.

**Verification:** New schema is inserted with incremented version number.

**Test:** Code review of `multitenantengine/manager.go:169-176`

**Related Requirements:**
- REQ-SCHEMA-003 (Schemas Table)

---

### REQ-MANAGER-015: Deactivate Previous Schemas
The `UpdateTenantSchema()` method **SHALL** set previous schemas to `active = false` before inserting new schema.

**Verification:** SQL UPDATE sets all previous schemas to inactive.

**Test:** Code review of `multitenantengine/manager.go:155-162`

---

### REQ-MANAGER-016: Atomic Engine Swap
The `UpdateTenantSchema()` method **SHALL** atomically replace the old engine with the new engine.

**Verification:** Engine swap happens under write lock to prevent race conditions.

**Test:** Code review of `multitenantengine/manager.go:200-205`

**Related Requirements:**
- REQ-CONCUR-003 (Thread-Safe Compilation)

---

### REQ-MANAGER-017: Backward Compatible Rule Execution
After schema update, existing rules **SHALL** continue to work if they remain compatible with the new schema.

**Verification:** Rule compiled under old schema evaluates successfully after schema update adds new fields.

**Test:** `TestMultiTenantEngineManager_UpdateTenantSchema` (multitenantengine/manager_test.go:239-254)

---

### REQ-MANAGER-018: Create Tenant on Update
The `UpdateTenantSchema()` method **SHALL** create the tenant if it doesn't exist in the manager.

**Verification:** Calling `UpdateTenantSchema()` on nonexistent tenant creates the tenant.

**Test:** `TestMultiTenantEngineManager_UpdateNonexistentTenant` (multitenantengine/manager_test.go:320)

---

## 6. Tenant Deletion Requirements

### REQ-MANAGER-019: Delete Tenant from Cache
The `MultiTenantEngineManager` **SHALL** provide a `DeleteTenant(tenantID)` method that removes the tenant from the cache.

**Verification:** After deletion, `GetEngine()` returns error for the tenant.

**Test:** `TestMultiTenantEngineManager_DeleteTenant` (multitenantengine/manager_test.go:301)

**Related Requirements:**
- REQ-TENANT-007 (Tenant Lifecycle Management)

---

### REQ-MANAGER-020: Delete Nonexistent Tenant Error
The `DeleteTenant()` method **SHALL** return an error when attempting to delete a nonexistent tenant.

**Verification:** Deleting nonexistent tenant returns descriptive error.

**Test:** `TestMultiTenantEngineManager_DeleteTenant` (multitenantengine/manager_test.go:315-318)

**Related Requirements:**
- REQ-ERROR-002 (Descriptive Errors)

---

### REQ-MANAGER-021: Cache-Only Deletion
The `DeleteTenant()` method **SHALL** only remove the tenant from the cache, not from the database.

**Verification:** Method does not execute database DELETE statements.

**Test:** Code review of `multitenantengine/manager.go:228-237`

---

## 7. Tenant Isolation Requirements

### REQ-MANAGER-022: Engine Isolation per Tenant
Each tenant's engine **SHALL** be isolated and unable to access other tenants' rules.

**Verification:** Tenant A's engine cannot evaluate Tenant B's rules.

**Test:** `TestMultiTenantEngineManager_TenantIsolation` (multitenantengine/manager_test.go:271)

**Related Requirements:**
- REQ-TENANT-001 (Tenant Isolation)
- REQ-TENANT-005 (Cross-Tenant Evaluation Prevention)

---

### REQ-MANAGER-023: Schema Isolation per Tenant
Each tenant **SHALL** have its own schema definition independent of other tenants.

**Verification:** Tenants can have different schemas with different object types.

**Test:** `TestMultiTenantEngineManager_TenantIsolation` (multitenantengine/manager_test.go:271)

---

## 8. Concurrency Requirements

### REQ-MANAGER-024: Thread-Safe Read Operations
The `MultiTenantEngineManager` **SHALL** support concurrent read operations (`GetEngine`, `ListTenants`) without blocking.

**Verification:** Multiple goroutines can call `GetEngine()` concurrently without errors.

**Test:** `TestMultiTenantEngineManager_Concurrency` (multitenantengine/manager_test.go:310)

**Related Requirements:**
- REQ-CONCUR-002 (Thread-Safe Engine)
- REQ-CONCUR-004 (Read-Write Lock)

---

### REQ-MANAGER-025: Thread-Safe Write Operations
The `MultiTenantEngineManager` **SHALL** support concurrent write operations (`CreateTenant`, `UpdateTenantSchema`) with proper locking.

**Verification:** Write operations use locks to prevent data races.

**Test:** Code review of `multitenantengine/manager.go` (mutex usage)

**Related Requirements:**
- REQ-CONCUR-003 (Thread-Safe Compilation)

---

### REQ-MANAGER-026: Read-Write Lock Implementation
The `MultiTenantEngineManager` **SHALL** use `sync.RWMutex` to allow concurrent reads while protecting writes.

**Verification:** Code uses `RLock()/RUnlock()` for reads and `Lock()/Unlock()` for writes.

**Test:** Code review of `multitenantengine/manager.go:29,117,130,144,228`

**Related Requirements:**
- REQ-CONCUR-004 (Read-Write Lock)

---

## 9. Error Handling Requirements

### REQ-MANAGER-027: Schema Compilation Errors
The `CreateTenant()` and `UpdateTenantSchema()` methods **SHALL** return descriptive errors when schema creates invalid CEL environment.

**Verification:** Invalid schema returns error with context about CEL environment creation failure.

**Test:** Error handling in `multitenantengine/manager.go:102-105,182-185`

**Related Requirements:**
- REQ-ERROR-002 (Descriptive Errors)
- REQ-ERROR-003 (Error Wrapping)

---

### REQ-MANAGER-028: Engine Creation Errors
The `CreateTenant()` and `UpdateTenantSchema()` methods **SHALL** return descriptive errors when engine creation fails.

**Verification:** Engine creation failure returns wrapped error with context.

**Test:** Error handling in `multitenantengine/manager.go:111-114,189-192`

**Related Requirements:**
- REQ-ERROR-003 (Error Wrapping)

---

### REQ-MANAGER-029: Database Query Errors
The `LoadAllTenants()` and `UpdateTenantSchema()` methods **SHALL** return descriptive errors for database failures.

**Verification:** Database errors are wrapped with context about the operation.

**Test:** Error handling throughout `multitenantengine/manager.go`

**Related Requirements:**
- REQ-ERROR-002 (Descriptive Errors)
- REQ-ERROR-003 (Error Wrapping)

---

## Requirement Traceability Matrix

| Requirement ID | Category | Priority | Verification Method | Test Reference |
|---------------|----------|----------|---------------------|----------------|
| REQ-MANAGER-001 | CEL Environment | MUST | Function test | TestCreateCELEnvFromSchema |
| REQ-MANAGER-002 | CEL Environment | MUST | Schema test | TestCreateCELEnvFromSchema/multiple_objects_schema |
| REQ-MANAGER-003 | CEL Environment | MUST | Empty schema test | TestCreateCELEnvFromSchema/empty_schema |
| REQ-MANAGER-004 | Tenant Loading | MUST | Integration test | TestMultiTenantEngineManager_LoadAllTenants |
| REQ-MANAGER-005 | Tenant Loading | MUST | Code review | manager.go:61-66 |
| REQ-MANAGER-006 | Tenant Loading | MUST | Deserialization test | TestMultiTenantEngineManager_LoadAllTenants |
| REQ-MANAGER-007 | Engine Retrieval | MUST | Retrieval test | TestMultiTenantEngineManager_LoadAllTenants |
| REQ-MANAGER-008 | Engine Retrieval | MUST | Error test | TestMultiTenantEngineManager_GetEngineNotFound |
| REQ-MANAGER-009 | Engine Retrieval | MUST | List test | TestMultiTenantEngineManager_LoadAllTenants |
| REQ-MANAGER-010 | Tenant Creation | MUST | Creation test | TestMultiTenantEngineManager_CreateTenant |
| REQ-MANAGER-011 | Tenant Creation | MUST | Code review | manager.go:108 |
| REQ-MANAGER-012 | Tenant Creation | MUST | Registration test | TestMultiTenantEngineManager_CreateTenant |
| REQ-MANAGER-013 | Schema Update | MUST | Update test | TestMultiTenantEngineManager_UpdateTenantSchema |
| REQ-MANAGER-014 | Schema Update | MUST | Code review | manager.go:169-176 |
| REQ-MANAGER-015 | Schema Update | MUST | Code review | manager.go:155-162 |
| REQ-MANAGER-016 | Schema Update | MUST | Code review | manager.go:200-205 |
| REQ-MANAGER-017 | Schema Update | MUST | Compatibility test | TestMultiTenantEngineManager_UpdateTenantSchema |
| REQ-MANAGER-018 | Schema Update | MUST | Nonexistent test | TestMultiTenantEngineManager_UpdateNonexistentTenant |
| REQ-MANAGER-019 | Tenant Deletion | MUST | Deletion test | TestMultiTenantEngineManager_DeleteTenant |
| REQ-MANAGER-020 | Tenant Deletion | MUST | Error test | TestMultiTenantEngineManager_DeleteTenant |
| REQ-MANAGER-021 | Tenant Deletion | MUST | Code review | manager.go:228-237 |
| REQ-MANAGER-022 | Isolation | MUST | Isolation test | TestMultiTenantEngineManager_TenantIsolation |
| REQ-MANAGER-023 | Isolation | MUST | Schema test | TestMultiTenantEngineManager_TenantIsolation |
| REQ-MANAGER-024 | Concurrency | MUST | Concurrent test | TestMultiTenantEngineManager_Concurrency |
| REQ-MANAGER-025 | Concurrency | MUST | Code review | manager.go |
| REQ-MANAGER-026 | Concurrency | MUST | Code review | manager.go:29,117,130,144,228 |
| REQ-MANAGER-027 | Error Handling | MUST | Code review | manager.go:102-105,182-185 |
| REQ-MANAGER-028 | Error Handling | MUST | Code review | manager.go:111-114,189-192 |
| REQ-MANAGER-029 | Error Handling | MUST | Code review | manager.go |

---

## Summary Statistics

- **Total Requirements:** 29
- **MUST Requirements:** 29
- **SHOULD Requirements:** 0
- **Categories:** 9
  - CEL Environment Management: 3 requirements
  - Tenant Loading and Initialization: 3 requirements
  - Engine Retrieval: 3 requirements
  - Tenant Creation: 3 requirements
  - Schema Update: 6 requirements
  - Tenant Deletion: 3 requirements
  - Tenant Isolation: 2 requirements
  - Concurrency: 3 requirements
  - Error Handling: 3 requirements

---

## Implementation Status

All requirements in this document have been **implemented and verified** as of the completion of the multi-tenant engine manager test suite:

- ✅ **8 integration tests** covering all multi-tenant manager requirements
- ✅ **MultiTenantEngineManager** fully implemented with all lifecycle methods
- ✅ **Dynamic schema management** with zero-downtime updates
- ✅ **Thread safety** verified with concurrent operation tests
- ✅ **Tenant isolation** verified with cross-tenant tests

**Test Coverage:**
- Total tests: 80 (63 unit + 9 persistence integration + 8 multi-tenant manager integration)
- All tests passing

**Next Phase:** HTTP API layer implementation (see `cmd/server/main.go` for server foundation).
