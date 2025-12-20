# Go CEL Rules Engine Plan (MVP v1)

This document outlines the architecture for building a **core rules engine library** in Go using `cel-go`. This library will serve as the foundation for the hybrid multi-tenant service described in `hybrid_tenant_types_blueprint.md`.

**Key Design Principle:** Build clean abstractions and interfaces that can be reused by the hybrid service, while keeping the MVP simple and focused on single-tenant use cases.

---

## 1. Objectives

- Build a safe and efficient rules engine library in Go.
- Support precomputed derived fields (with path for future code generation).
- Enable caching of compiled CEL programs for performance.
- Provide observability and tracing for rule evaluation.
- Use `cel-go` as the expression language for safety and static checking.
- Design interfaces that support multi-tenant usage in future iterations.

---

## 2. Architecture Overview

### Package Structure

```
rules/
  ├── generated/           # Type definitions (hand-written initially, generated later)
  │   └── types.go        # Facts struct and schema for this tenant
  ├── engine.go           # Core Engine implementation
  ├── store.go            # RuleStore interface and implementations
  ├── compiler.go         # CEL compilation logic
  ├── evaluator.go        # Evaluation logic
  └── types.go            # Core types (Rule, EvaluationResult, etc.)
```

### Core Components

1. **Generated Types** (`generated/types.go`)
   - Defines the Facts struct for a tenant's schema
   - Hand-written for MVP, but structured to be code-generated later
   - Example: User, Transaction, and Facts structs

2. **RuleStore Interface** (`store.go`)
   - Stores rules with metadata: id, name, expression, active status
   - Provides CRUD operations for rules
   - MVP implementation: in-memory store
   - Future: can be backed by database

3. **Engine** (`engine.go`)
   - Central orchestrator for rule compilation and evaluation
   - Manages CEL environment and compiled programs cache
   - Handles derived field evaluation
   - Thread-safe for concurrent access

4. **Compiler** (`compiler.go`)
   - Uses `cel-go` to parse expressions into ASTs
   - Performs type-checking and validation
   - Returns compiled CEL programs

5. **Evaluator** (`evaluator.go`)
   - Evaluates compiled programs against input facts
   - Processes derived fields (precomputed in facts for MVP)
   - Returns evaluation results with optional tracing

6. **Core Types** (`types.go`)
   - Rule: Represents a rule definition
   - EvaluationResult: Contains match status and trace
   - Interfaces for extensibility

---

## 3. Core Type Definitions

### Rule Structure
```go
package rules

// Rule represents a single evaluation rule
type Rule struct {
    ID         string
    Name       string
    Expression string
    Active     bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// EvaluationResult contains the outcome of evaluating a rule
type EvaluationResult struct {
    RuleID    string
    RuleName  string
    Matched   bool
    Error     error
    Trace     map[string]interface{} // CEL evaluation trace (optional)
}

// DerivedField represents a computed field (for future code generation)
type DerivedField struct {
    Name       string
    Expression string // CEL expression for computing the field
}
```

---

## 4. Interface Definitions

### RuleStore Interface
```go
package rules

// RuleStore manages rule persistence and retrieval
type RuleStore interface {
    // Add a new rule
    Add(rule *Rule) error

    // Get a rule by ID
    Get(id string) (*Rule, error)

    // List all active rules
    ListActive() ([]*Rule, error)

    // Update an existing rule
    Update(rule *Rule) error

    // Delete a rule
    Delete(id string) error
}
```

---

## 5. Implementation Examples

### Generated Types (`generated/types.go`)

This file is hand-written for MVP but follows the pattern that will be code-generated in the hybrid service.

```go
// Code in generated/types.go
// For MVP: Hand-written
// For Hybrid Service: Auto-generated from tenant schema

package generated

// User represents user information
type User struct {
    Citizenship string  `json:"Citizenship"`
    Age         int     `json:"Age"`
}

// Transaction represents transaction data
type Transaction struct {
    Amount  float64 `json:"Amount"`
    Country string  `json:"Country"`
}

// Facts is the top-level container for all evaluation inputs
// Note: Derived fields like "isCanadian" are precomputed and added to this struct
type Facts struct {
    User        User        `json:"User"`
    Transaction Transaction `json:"Transaction"`

    // Derived fields (precomputed before evaluation)
    // Future: These could be generated from DerivedField definitions
    IsCanadian bool `json:"isCanadian,omitempty"`
}
```

### In-Memory RuleStore Implementation (`store.go`)

```go
package rules

import (
    "fmt"
    "sync"
    "time"
)

// InMemoryRuleStore implements RuleStore using an in-memory map
type InMemoryRuleStore struct {
    rules map[string]*Rule
    mu    sync.RWMutex
}

// NewInMemoryRuleStore creates a new in-memory rule store
func NewInMemoryRuleStore() *InMemoryRuleStore {
    return &InMemoryRuleStore{
        rules: make(map[string]*Rule),
    }
}

func (s *InMemoryRuleStore) Add(rule *Rule) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.rules[rule.ID]; exists {
        return fmt.Errorf("rule with ID %s already exists", rule.ID)
    }

    rule.CreatedAt = time.Now()
    rule.UpdatedAt = time.Now()
    s.rules[rule.ID] = rule
    return nil
}

func (s *InMemoryRuleStore) Get(id string) (*Rule, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    rule, exists := s.rules[id]
    if !exists {
        return nil, fmt.Errorf("rule with ID %s not found", id)
    }
    return rule, nil
}

func (s *InMemoryRuleStore) ListActive() ([]*Rule, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var active []*Rule
    for _, rule := range s.rules {
        if rule.Active {
            active = append(active, rule)
        }
    }
    return active, nil
}

func (s *InMemoryRuleStore) Update(rule *Rule) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.rules[rule.ID]; !exists {
        return fmt.Errorf("rule with ID %s not found", rule.ID)
    }

    rule.UpdatedAt = time.Now()
    s.rules[rule.ID] = rule
    return nil
}

func (s *InMemoryRuleStore) Delete(id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.rules[id]; !exists {
        return fmt.Errorf("rule with ID %s not found", id)
    }

    delete(s.rules, id)
    return nil
}
```

### Engine Implementation (`engine.go`)

The Engine manages the CEL environment and compiled programs, providing high-level evaluation APIs.

```go
package rules

import (
    "fmt"
    "sync"

    "github.com/google/cel-go/cel"
    "github.com/your-module/rules/generated" // Reference to generated types
)

// Engine manages CEL environment and rule compilation/evaluation
type Engine struct {
    env      *cel.Env
    store    RuleStore
    programs map[string]cel.Program // ruleID -> compiled program
    mu       sync.RWMutex
}

// NewEngine creates a new rules engine
func NewEngine(store RuleStore) (*Engine, error) {
    // Create CEL environment with generated types
    env, err := cel.NewEnv(
        cel.Types(&generated.Facts{}),
        cel.Variable("User", cel.ObjectType("generated.User")),
        cel.Variable("Transaction", cel.ObjectType("generated.Transaction")),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create CEL environment: %w", err)
    }

    engine := &Engine{
        env:      env,
        store:    store,
        programs: make(map[string]cel.Program),
    }

    // Compile all active rules on initialization
    if err := engine.CompileAllRules(); err != nil {
        return nil, fmt.Errorf("failed to compile rules: %w", err)
    }

    return engine, nil
}

// CompileRule compiles a single rule expression to a CEL program
func (e *Engine) CompileRule(ruleID string, expression string) error {
    ast, issues := e.env.Compile(expression)
    if issues != nil && issues.Err() != nil {
        return fmt.Errorf("compile error: %w", issues.Err())
    }

    prog, err := e.env.Program(ast, cel.EvalOptions(cel.OptTrackState))
    if err != nil {
        return fmt.Errorf("program creation error: %w", err)
    }

    e.mu.Lock()
    e.programs[ruleID] = prog
    e.mu.Unlock()

    return nil
}

// CompileAllRules compiles all active rules from the store
func (e *Engine) CompileAllRules() error {
    rules, err := e.store.ListActive()
    if err != nil {
        return err
    }

    for _, rule := range rules {
        if err := e.CompileRule(rule.ID, rule.Expression); err != nil {
            return fmt.Errorf("failed to compile rule %s: %w", rule.ID, err)
        }
    }

    return nil
}

// Evaluate evaluates a single rule against the provided facts
func (e *Engine) Evaluate(ruleID string, facts map[string]interface{}) (*EvaluationResult, error) {
    rule, err := e.store.Get(ruleID)
    if err != nil {
        return nil, err
    }

    e.mu.RLock()
    prog, exists := e.programs[ruleID]
    e.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("rule %s is not compiled", ruleID)
    }

    out, details, err := prog.Eval(facts)
    if err != nil {
        return &EvaluationResult{
            RuleID:   ruleID,
            RuleName: rule.Name,
            Matched:  false,
            Error:    err,
        }, err
    }

    matched := false
    if boolVal, ok := out.Value().(bool); ok {
        matched = boolVal
    }

    return &EvaluationResult{
        RuleID:   ruleID,
        RuleName: rule.Name,
        Matched:  matched,
        Trace:    details.State(),
    }, nil
}

// EvaluateAll evaluates all active rules against the provided facts
func (e *Engine) EvaluateAll(facts map[string]interface{}) ([]*EvaluationResult, error) {
    rules, err := e.store.ListActive()
    if err != nil {
        return nil, err
    }

    results := make([]*EvaluationResult, 0, len(rules))
    for _, rule := range rules {
        result, err := e.Evaluate(rule.ID, facts)
        if err != nil {
            // Continue evaluating other rules even if one fails
            result = &EvaluationResult{
                RuleID:   rule.ID,
                RuleName: rule.Name,
                Matched:  false,
                Error:    err,
            }
        }
        results = append(results, result)
    }

    return results, nil
}

// AddRule adds a new rule to the store and compiles it
func (e *Engine) AddRule(rule *Rule) error {
    // First, validate that the rule compiles
    if err := e.CompileRule(rule.ID, rule.Expression); err != nil {
        return fmt.Errorf("rule validation failed: %w", err)
    }

    // Then add to store
    if err := e.store.Add(rule); err != nil {
        // Remove from compiled programs if store fails
        e.mu.Lock()
        delete(e.programs, rule.ID)
        e.mu.Unlock()
        return err
    }

    return nil
}

// UpdateRule updates an existing rule and recompiles it
func (e *Engine) UpdateRule(rule *Rule) error {
    // Compile the new expression
    if err := e.CompileRule(rule.ID, rule.Expression); err != nil {
        return fmt.Errorf("rule validation failed: %w", err)
    }

    // Update in store
    return e.store.Update(rule)
}

// DeleteRule removes a rule from the store and compiled programs
func (e *Engine) DeleteRule(ruleID string) error {
    if err := e.store.Delete(ruleID); err != nil {
        return err
    }

    e.mu.Lock()
    delete(e.programs, ruleID)
    e.mu.Unlock()

    return nil
}
```

## 6. Usage Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/your-module/rules"
    "github.com/your-module/rules/generated"
)

func main() {
    // 1. Create rule store
    store := rules.NewInMemoryRuleStore()

    // 2. Add some rules
    rule1 := &rules.Rule{
        ID:         "rule-1",
        Name:       "Large Canadian Transaction",
        Expression: `User.Citizenship == "CANADA" && Transaction.Amount > 1000`,
        Active:     true,
    }

    rule2 := &rules.Rule{
        ID:         "rule-2",
        Name:       "Senior Citizen",
        Expression: `User.Age >= 65`,
        Active:     true,
    }

    // 3. Create engine (this will compile all rules)
    engine, err := rules.NewEngine(store)
    if err != nil {
        log.Fatal(err)
    }

    // 4. Add rules (engine validates and compiles them)
    if err := engine.AddRule(rule1); err != nil {
        log.Fatal(err)
    }
    if err := engine.AddRule(rule2); err != nil {
        log.Fatal(err)
    }

    // 5. Prepare facts (with precomputed derived fields)
    facts := map[string]interface{}{
        "User": map[string]interface{}{
            "Citizenship": "CANADA",
            "Age":         70,
        },
        "Transaction": map[string]interface{}{
            "Amount": 1500.0,
            "Country": "CANADA",
        },
        // Derived field (precomputed)
        "isCanadian": true,
    }

    // 6. Evaluate all rules
    results, err := engine.EvaluateAll(facts)
    if err != nil {
        log.Fatal(err)
    }

    // 7. Process results
    for _, result := range results {
        if result.Error != nil {
            fmt.Printf("Rule %s failed: %v\n", result.RuleName, result.Error)
            continue
        }

        fmt.Printf("Rule: %s\n", result.RuleName)
        fmt.Printf("  Matched: %v\n", result.Matched)
        fmt.Printf("  Trace: %v\n", result.Trace)
    }
}
```

**Output:**
```
Rule: Large Canadian Transaction
  Matched: true
  Trace: map[...]
Rule: Senior Citizen
  Matched: true
  Trace: map[...]
```

---

## 7. Derived Fields Approach

### MVP: Precomputed Derived Fields

For the MVP, derived fields are computed **before** evaluation and added to the facts map.

**Example:**
```go
// Application code computes derived fields
func computeDerivedFields(facts map[string]interface{}) {
    user := facts["User"].(map[string]interface{})

    // Compute isCanadian
    isCanadian := user["Citizenship"] == "CANADA"
    facts["isCanadian"] = isCanadian

    // Compute isAdult
    age := user["Age"].(int)
    facts["isAdult"] = age >= 18

    // These can now be referenced in rules:
    // - "isCanadian && Transaction.Amount > 1000"
    // - "isAdult && Transaction.Type == 'CREDIT'"
}
```

### Future: Code-Generated Derived Fields

In the hybrid service, derived fields will be:
1. Defined via API with CEL expressions
2. Stored in database with dependency information
3. Generated as Go functions in `generated/types.go`
4. Compiled as separate CEL programs
5. Evaluated in topological order before rule evaluation

**Example Generated Code (Future):**
```go
// Auto-generated in generated/types.go
func (f *Facts) ComputeDerivedFields() error {
    // Derived field: isCanadian
    f.IsCanadian = f.User.Citizenship == "CANADA"

    // Derived field: isAdult (depends on User.Age)
    f.IsAdult = f.User.Age >= 18

    // Derived field: isEligible (depends on other derived fields)
    f.IsEligible = f.IsCanadian && f.IsAdult

    return nil
}
```

**Benefits of Generation:**
- Type safety for derived fields
- Automatic dependency resolution
- Validation at code generation time
- Same performance as hand-written code

---

## 8. Migration Path to Hybrid Service

This MVP is designed to evolve into the hybrid multi-tenant service:

### What Stays the Same
- Core `Engine` implementation
- `RuleStore` interface (swap in DB implementation)
- Compilation and evaluation logic
- `EvaluationResult` structure

### What Changes
1. **Multiple Engines**: One `Engine` instance per tenant
2. **Generated Types**: `generated/types.go` becomes auto-generated per tenant schema
3. **Tenant Cache**: Wrap `Engine` in a `TenantCache` structure
4. **HTTP API**: Add REST endpoints around the engine
5. **Code Generator**: Add template-based code generation from schema definitions

### Migration Steps
1. Extract core engine into reusable package ✅ (this MVP)
2. Add HTTP API layer (see `hybrid_tenant_types_blueprint.md`)
3. Implement code generator from schema definitions
4. Add tenant management and multi-tenancy support
5. Deploy as shared service with per-tenant engines

---

## 9. Recommended Next Steps

### Phase 1: Core Implementation
1. ✅ **Define interfaces** - RuleStore, Engine (covered in this plan)
2. ✅ **Implement in-memory RuleStore** (covered in this plan)
3. ✅ **Implement Engine** with compilation and evaluation (covered in this plan)
4. **Create `generated/types.go`** - Hand-write example tenant schema
5. **Write unit tests** for store, engine, compilation, evaluation

### Phase 2: Derived Fields
6. **Implement precomputed derived fields** - Add helper function to compute them
7. **Test derived field evaluation** - Ensure they work in rule expressions
8. **Document derived field patterns** - Guidelines for users

### Phase 3: Production Readiness
9. **Add security** - CEL function restrictions, cost limits, expression validation
10. **Add metrics** - Evaluation time, rule hit rates, compilation errors
11. **Add logging** - Structured logs for debugging and auditing
12. **Error handling** - Comprehensive error types and messages

### Phase 4: Testing
13. **Unit tests** - All core components (80%+ coverage)
14. **Integration tests** - End-to-end rule evaluation scenarios
15. **Benchmark tests** - Performance characteristics
16. **Security tests** - Validation of sandboxing and limits

### Phase 5: Documentation
17. **API documentation** - Godoc for all public types and functions
18. **Usage examples** - Multiple real-world scenarios
19. **Migration guide** - How to move to hybrid service later

---

## 10. Key Design Decisions

### ✅ Decisions Made

1. **Derived Fields**: Precomputed for MVP, path to code generation later
2. **Type Definitions**: In `generated/` directory to establish pattern
3. **Rule Storage**: Interface-based, in-memory for MVP
4. **Engine Architecture**: Single engine with CEL environment and program cache
5. **Thread Safety**: Read-write locks for concurrent access
6. **Error Handling**: Return errors, don't panic (except in examples)

### 🔄 Deferred to Hybrid Service

1. **Multi-tenancy**: Single tenant for MVP
2. **HTTP API**: Library only, no REST endpoints yet
3. **Database**: In-memory only, DB implementation later
4. **Code Generation**: Manual for MVP, automated later
5. **Monitoring**: Basic errors only, full observability later

---

This plan provides a complete foundation for building a rules engine that can evolve into the hybrid multi-tenant service architecture.

