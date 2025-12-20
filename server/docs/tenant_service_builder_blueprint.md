# Tenant Service Builder Blueprint (Go + CEL-Go)

This blueprint outlines a **multi-tenant, service-per-tenant rules engine architecture** where each tenant gets a deployed service with their own data model, derived fields, and compiled CEL rules.

---

## 1. Overview

**Goal:** Allow each tenant to have a fully isolated rules engine service with strongly typed Go structs, precompiled CEL rules, and runtime evaluation via HTTP API.

**Key Features:**
- UI-driven tenant registration and model creation
- Derived field support with dependency resolution
- Tenant-specific code generation and compilation
- Deployment orchestration (containers/pods)
- Precompiled ASTs for high-performance evaluation
- Multi-tenant isolation and scalability

---

## 2. High-Level Architecture

```
+-------------------------+
| Tenant Registration UI  |
+------------+------------+
             |
             v
+-------------------------+
| Master Service Builder   |
| - Accept schema/rules   |
| - Generate Go code      |
| - Compile tenant binary |
| - Deploy tenant service |
+------------+------------+
             |
             v
+-------------------------+
| Tenant Service (per tenant) |
| - Generated structs         |
| - Derived fields            |
| - CEL-Go rules compiled      |
| - Evaluation API             |
+-------------------------+
```

**Flow:**
1. Tenant registers via UI.
2. Defines schema, derived fields, and rules.
3. Master service generates Go code + CEL setup.
4. Compiles tenant service binary or container image.
5. Deploys tenant service.
6. Tenant calls their service to evaluate facts via API.

---

## 3. Master Service Responsibilities

### a) Schema & Rules Intake
- Receive tenant schema, derived fields, and rules via REST API.
- Validate schema (check types, field names) and detect cyclic dependencies in derived fields.

### b) Code Generation
- Generate Go structs for tenant schema.
- Generate derived field functions or inline expressions.
- Generate rules code (or CEL expression storage for compilation).
- Include `Facts` wrapper struct with all top-level fields.

### c) Compilation
- Build tenant-specific Go binary or plugin.
- Precompile CEL rules to ASTs at startup.
- Include optional evaluation tracing code.

### d) Deployment
- Build a container image per tenant or deploy binary as pod/service.
- Optionally, use Kubernetes for scaling and lifecycle management.

### e) Orchestration
- Maintain metadata mapping tenant ID → deployed service URL/version.
- Monitor tenant services for health and restarts.
- Manage upgrades or schema changes per tenant.

---

## 4. Tenant Service Structure

### a) Generated Go Code
- Structs for data model
- Derived field functions
- CEL environment initialization with `cel.Types(&Facts{})`
- Precompiled rules stored in memory

### b) API Endpoints
**Evaluation:**
- `POST /evaluate`
  - Accepts facts JSON
  - Optionally: rules to evaluate (UUID or internal IDs)
  - Returns matched rules + trace

**Optional Management Endpoints:**
- `GET /rules` → List compiled rules
- `GET /derived` → List derived fields

### c) CEL Evaluation
- On startup, precompile CEL expressions per tenant
- Cache compiled ASTs in memory
- Evaluate facts via API on request
- Return evaluation results + optional trace for explainability

---

## 5. Derived Field Handling

1. **Dependency resolution:**
   - Build a DAG of derived fields.
   - Topologically sort before evaluation.
2. **Inlining or functions:**
   - Inline derived field expressions recursively OR
   - Implement derived fields as Go functions returning values based on `Facts`
3. **Cycle detection:**
   - Validate schema + derived fields at creation/update time to reject cycles.

---

## 6. Code Generation Strategy

- **Templates:** Use `text/template` to generate Go structs and derived field functions.
- **Formatting:** Use `go/format` to ensure clean, idiomatic Go code.
- **Build:** Compile generated code into tenant-specific binary or container image.
- **CEL-Go Integration:**
  - Import generated structs
  - Initialize CEL environment with typed `Facts`
  - Precompile rules into ASTs

Example generated `Facts` struct:
```go
type User struct {
    Age int
    Citizenship string
}

type Tx struct {
    Amount float64
    Country string
}

type Facts struct {
    User User
    Tx Tx
}
```

---

## 7. Deployment Options

- **Kubernetes:** Pod/Deployment per tenant, service discovery via DNS
- **Docker:** Individual container per tenant, managed by orchestrator
- **Serverless:** Lambda or function per tenant (more advanced)

- **Shared base image:** Skeleton service with rules engine; inject tenant-specific Go code + CEL rules at build time

---

## 8. Pros / Cons of Service-Per-Tenant

| Pros | Cons |
|------|------|
| Strong type safety for tenant schema | Higher infrastructure overhead |
| Isolation between tenants | More CI/CD complexity |
| Easier caching & precompilation | Increased deployment management |
| Derived field dependency resolution simplified | Slower onboarding for new tenants (compile+deploy) |

---

## 9. Next Steps

1. Implement master service REST API for tenant onboarding, schema, and rule intake.
2. Build code generation templates for models, derived fields, and CEL rules.
3. Set up compilation pipeline for tenant binaries or container images.
4. Design deployment orchestration (Kubernetes preferred).
5. Implement tenant evaluation API with CEL-Go precompiled ASTs.
6. Add monitoring, logging, and multi-tenant metadata tracking.
7. Optional: Hot-reload or dynamic plugin support for tenant updates.

