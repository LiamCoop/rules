# Rules Engine Service Blueprint (Go + CEL-Go)

This blueprint provides a detailed guide for building a flexible rules engine as a service using Go and `cel-go`. It includes the API spec, database schema suggestions, CEL-Go integration notes, and evaluation workflow.

---

## 1. Overview

**Goal:** Create a service that allows users to define their data model, add derived values, create rules, and evaluate them via an HTTP API.

**Key Features:**
- UI-driven data model management
- Support for derived fields
- User-defined rules using CEL expressions
- Real-time evaluation via API
- AST caching for performance
- Tracing and explainability
- Multi-tenant support (optional)

---

## 2. High-Level Architecture

```
+-------------------+
|   User Interface  |
|  (Web / React)    |
+---------+---------+
          |
          v
+-------------------+
| Rule & Model API  | <-- HTTP/REST/gRPC
+---------+---------+
          |
          v
+-------------------+
| Rules Engine      |  <- CEL-Go runtime
| Compiler / AST    |
| Derived Fields    |
+---------+---------+
          |
          v
+-------------------+
| Evaluation Cache  |
| (Compiled ASTs)   |
+---------+---------+
          |
          v
+-------------------+
|  Results / Audit  |
+-------------------+
```

**Workflow:**
1. User defines a data model.
2. User optionally defines derived fields.
3. User creates rules referencing fields and derived values.
4. Backend compiles rules to ASTs and stores them.
5. Evaluation API receives facts and executes rules.
6. Returns evaluation results and traces.

---

## 3. Database Schema Suggestion

### **Models**
| Column        | Type        | Description |
|---------------|------------|-------------|
| id            | UUID       | Primary Key |
| name          | string     | Model name |
| owner_id      | UUID       | User/tenant |
| schema        | JSON       | Field definitions |
| created_at    | timestamp  | Creation time |
| updated_at    | timestamp  | Last update |

### **Derived Fields**
| Column        | Type        | Description |
|---------------|------------|-------------|
| id            | UUID       | Primary Key |
| model_id      | UUID       | FK to Models |
| name          | string     | Derived field name |
| expression    | string     | CEL expression |
| created_at    | timestamp  | Creation time |
| updated_at    | timestamp  | Last update |

### **Rules**
| Column        | Type        | Description |
|---------------|------------|-------------|
| id            | UUID       | Primary Key |
| name          | string     | Rule name |
| model_id      | UUID       | FK to Model |
| expression    | string     | CEL expression referencing fields/derived fields |
| compiled_ast  | binary     | Serialized AST (optional caching) |
| active        | boolean    | Rule active status |
| created_at    | timestamp  | Creation time |
| updated_at    | timestamp  | Last update |

---

## 4. API Specification

### **Data Model Endpoints**
- `POST /models` → Create a new data model
- `GET /models` → List existing models
- `PUT /models/:id` → Update model
- `DELETE /models/:id` → Remove model

### **Derived Field Endpoints**
- `POST /models/:id/derived` → Add derived field
- `GET /models/:id/derived` → List derived fields
- `PUT /models/:id/derived/:derivedId` → Update
- `DELETE /models/:id/derived/:derivedId` → Delete

### **Rule Endpoints**
- `POST /rules` → Create a rule
- `GET /rules` → List rules
- `PUT /rules/:id` → Update a rule
- `DELETE /rules/:id` → Remove a rule

### **Evaluation Endpoint**
- `POST /evaluate`
  - Request:
    ```json
    {
      "rules": ["rule1", "rule2"],
      "facts": { ... }
    }
    ```
  - Response:
    ```json
    {
      "results": [
        {"ruleId": "rule1", "matched": true, "trace": [...]},
        {"ruleId": "rule2", "matched": false, "trace": [...]}
      ]
    }
    ```

---

## 5. CEL-Go Integration Notes

1. **Environment Setup:**
   - Declare variables representing the user-defined data model.
   - Register any custom functions needed.

2. **Rule Compilation:**
   - Compile rules and derived expressions to ASTs when created or updated.
   - Optionally serialize ASTs and store in DB for caching.

3. **Derived Rules Handling:**
   - Precompute derived fields at fact ingestion OR
   - Compile derived expressions as reusable AST nodes/functions.

4. **Runtime Evaluation:**
   - Load compiled ASTs.
   - Evaluate against facts using `program.Eval(facts)`.
   - Optionally include evaluation trace for explainability.

5. **Caching:**
   - Keep in-memory cache of compiled ASTs keyed by rule ID/version.
   - Invalidate cache on rule/derived field update.

---

## 6. Evaluation Workflow

1. API receives evaluation request with facts and rule identifiers.
2. For each rule:
   - Retrieve compiled AST from cache (or compile on-demand).
   - Evaluate AST against the facts.
   - Include derived fields in evaluation if required.
   - Collect result and optional trace.
3. Aggregate results and return JSON response.

---

## 7. Example Go Code Snippet (from earlier)

```go
package rules

import (
    "fmt"
    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/checker/decls"
    "github.com/google/cel-go/common/types/ref"
)

// Example facts structure
type Facts struct {
    User struct {
        Citizenship string
        Age         int
    }
    Tx struct {
        Amount float64
        Country string
    }
}

func makeEnv() (*cel.Env, error) {
    env, err := cel.NewEnv(
        cel.Declarations(
            decls.NewIdent("user", decls.NewObjectType("User"), nil),
            decls.NewIdent("tx", decls.NewObjectType("Tx"), nil),
        ),
        cel.Types(&Facts{}),
    )
    if err != nil {
        return nil, err
    }
    return env, nil
}

func compileRule(env *cel.Env, expr string) (cel.Program, error) {
    ast, issues := env.Compile(expr)
    if issues != nil && issues.Err() != nil {
        return nil, fmt.Errorf("compile error: %w", issues.Err())
    }
    prog, err := env.Program(ast, cel.EvalOptions(cel.OptTrackState))
    if err != nil {
        return nil, err
    }
    return prog, nil
}

func evalRule(prog cel.Program, facts map[string]interface{}) (ref.Val, *cel.EvalDetails, error) {
    out, details, err := prog.Eval(facts)
    if err != nil {
        return nil, nil, err
    }
    return out, details, nil
}

func Example() {
    env, _ := makeEnv()
    expr := `user.Citizenship == "CANADA" && tx.Amount > 1000`
    program, _ := compileRule(env, expr)

    facts := map[string]interface{}{
        "user": map[string]interface{}{ "Citizenship": "CANADA", "Age": 30 },
        "tx": map[string]interface{}{ "Amount": 1500, "Country": "CANADA" },
    }

    result, details, _ := evalRule(program, facts)
    fmt.Println("Result:", result)
    fmt.Println("Trace:", details.State())
}
```

---

## 8. Next Steps

- Implement REST API scaffolding in Go (e.g., using Gin or Echo).
- Build database schema for models, derived fields, and rules.
- Integrate CEL-Go compilation and evaluation pipeline.
- Implement caching of compiled ASTs.
- Build front-end UI for model, derived fields, and rule management.
- Add authentication/authorization for multi-tenant support.
- Add testing and audit/logging of evaluations.

