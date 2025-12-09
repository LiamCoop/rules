# MVP Rules Engine Requirements

This document specifies the functional and non-functional requirements for the MVP v1 rules engine. Each requirement is verifiable and uses specific language to enable test derivation.

**Requirement Language:**
- **SHALL** = Mandatory requirement
- **SHOULD** = Recommended but not mandatory
- **MAY** = Optional

---

## 1. Type System Requirements

### REQ-TYPE-001: Generated Types Directory
The rules engine **SHALL** reference type definitions from a `generated/` directory within the rules package.

**Verification:** Package imports successfully reference `generated` package and compilation succeeds.

### REQ-TYPE-002: Facts Structure
The `generated/types.go` file **SHALL** define a `Facts` struct that serves as the top-level container for all evaluation inputs.

**Verification:** `Facts` struct exists and can be instantiated.

### REQ-TYPE-003: Nested Object Support
The `Facts` struct **SHALL** support nested object types (e.g., `Facts.User.Citizenship`).

**Verification:** CEL expressions can access nested fields and evaluate correctly.

### REQ-TYPE-004: JSON Serialization
All types in `generated/types.go` **SHALL** support JSON serialization with appropriate struct tags.

**Verification:** `Facts` struct can be marshaled to and unmarshaled from JSON without data loss.

---

## 2. Rule Storage Requirements

### REQ-STORE-001: RuleStore Interface
The engine **SHALL** define a `RuleStore` interface with the following methods:
- `Add(rule *Rule) error`
- `Get(id string) (*Rule, error)`
- `ListActive() ([]*Rule, error)`
- `Update(rule *Rule) error`
- `Delete(id string) error`

**Verification:** Interface is defined with exactly these method signatures.

### REQ-STORE-002: Rule Structure
A `Rule` **SHALL** contain the following fields:
- `ID` (string)
- `Name` (string)
- `Expression` (string)
- `Active` (boolean)
- `CreatedAt` (timestamp)
- `UpdatedAt` (timestamp)

**Verification:** Rule struct has all required fields with correct types.

### REQ-STORE-003: In-Memory Implementation
The engine **SHALL** provide an `InMemoryRuleStore` implementation that stores rules in memory.

**Verification:** `InMemoryRuleStore` implements `RuleStore` interface and all methods work correctly.

### REQ-STORE-004: Unique Rule IDs
The `RuleStore` **SHALL** enforce unique rule IDs. Adding a rule with a duplicate ID **SHALL** return an error.

**Verification:** Attempting to add two rules with the same ID returns an error on the second attempt.

### REQ-STORE-005: Rule Not Found
The `RuleStore.Get()` method **SHALL** return an error when a rule ID does not exist.

**Verification:** Calling `Get()` with a non-existent ID returns an error.

### REQ-STORE-006: Timestamp Management
The `RuleStore` **SHALL** automatically set `CreatedAt` when adding a rule and update `UpdatedAt` when updating a rule.

**Verification:** After adding a rule, `CreatedAt` is set. After updating, `UpdatedAt` is greater than or equal to `CreatedAt`.

### REQ-STORE-007: Active Rules Filter
The `RuleStore.ListActive()` method **SHALL** return only rules where `Active` is true.

**Verification:** Given a store with active and inactive rules, `ListActive()` returns only active rules.

---

## 3. Rule Compilation Requirements

### REQ-COMPILE-001: CEL Environment Creation
The engine **SHALL** create a CEL environment with type information from `generated.Facts`.

**Verification:** CEL environment is created successfully and recognizes types from `Facts` struct.

### REQ-COMPILE-002: Expression Compilation
The engine **SHALL** compile CEL expressions into `cel.Program` instances.

**Verification:** Valid CEL expression compiles successfully without errors.

### REQ-COMPILE-003: Compilation Error Handling
The engine **SHALL** return a descriptive error when a CEL expression fails to compile.

**Verification:** Invalid CEL expression returns an error with message describing the issue.

### REQ-COMPILE-004: Type Checking
The engine **SHALL** perform type checking during compilation and reject expressions with type errors.

**Verification:** Expression with type mismatch (e.g., `User.Age == "string"`) returns compilation error.

### REQ-COMPILE-005: Program Caching
The engine **SHALL** cache compiled `cel.Program` instances keyed by rule ID to avoid recompilation.

**Verification:** After compiling a rule, subsequent evaluations use the cached program without recompiling.

### REQ-COMPILE-006: Compile All on Init
When creating a new engine, it **SHALL** compile all active rules from the store during initialization.

**Verification:** After engine initialization, all active rules are compiled and cached.

### REQ-COMPILE-007: Tracing Support
Compiled programs **SHALL** be created with `cel.OptTrackState` to enable evaluation tracing.

**Verification:** Evaluation details include trace information showing evaluation steps.

---

## 4. Rule Evaluation Requirements

### REQ-EVAL-001: Evaluate Single Rule
The engine **SHALL** provide an `Evaluate(ruleID string, facts map[string]interface{}) (*EvaluationResult, error)` method.

**Verification:** Method exists and successfully evaluates a rule against provided facts.

### REQ-EVAL-002: Evaluate All Rules
The engine **SHALL** provide an `EvaluateAll(facts map[string]interface{}) ([]*EvaluationResult, error)` method that evaluates all active rules.

**Verification:** Method evaluates all active rules and returns results for each.

### REQ-EVAL-003: Boolean Result
Rule evaluation **SHALL** return a boolean match result indicating whether the rule expression evaluated to true.

**Verification:** A rule that matches returns `Matched: true`, a rule that doesn't match returns `Matched: false`.

### REQ-EVAL-004: Evaluation Result Structure
An `EvaluationResult` **SHALL** contain:
- `RuleID` (string)
- `RuleName` (string)
- `Matched` (boolean)
- `Error` (error, nullable)
- `Trace` (map[string]interface{}, optional)

**Verification:** `EvaluationResult` struct has all required fields.

### REQ-EVAL-005: Non-Boolean Expression Handling
If a rule expression evaluates to a non-boolean value, the engine **SHALL** treat it as `Matched: false`.

**Verification:** Rule expression returning an integer has `Matched: false` in the result.

### REQ-EVAL-006: Evaluation Error Handling
If evaluation fails, the engine **SHALL** return an `EvaluationResult` with the error populated and `Matched: false`.

**Verification:** Rule that fails evaluation (e.g., division by zero) returns result with error and `Matched: false`.

### REQ-EVAL-007: Continue on Error
When evaluating all rules, if one rule fails, the engine **SHALL** continue evaluating remaining rules.

**Verification:** Given 3 rules where the 2nd fails, `EvaluateAll()` returns 3 results with the 2nd containing an error.

### REQ-EVAL-008: Facts as Map
The engine **SHALL** accept facts as `map[string]interface{}` to support flexible data structures.

**Verification:** Evaluation succeeds with facts provided as a map.

### REQ-EVAL-009: Nested Field Access
The engine **SHALL** support accessing nested fields in facts (e.g., `User.Citizenship`).

**Verification:** Rule expression `User.Citizenship == "CANADA"` evaluates correctly against nested facts.

### REQ-EVAL-010: Missing Field Handling
If a rule references a field not present in facts, the engine **SHALL** return an evaluation error.

**Verification:** Rule referencing non-existent field returns evaluation error.

---

## 5. Derived Fields Requirements

### REQ-DERIVED-001: Precomputed Fields
The engine **SHALL** support derived fields that are precomputed and added to the facts map before evaluation.

**Verification:** Derived field in facts map can be referenced in rule expressions and evaluates correctly.

### REQ-DERIVED-002: Derived Field Type
The `DerivedField` type **SHALL** contain:
- `Name` (string)
- `Expression` (string, CEL expression)

**Verification:** `DerivedField` struct exists with these fields.

### REQ-DERIVED-003: Optional Derived Fields
Derived fields in the `Facts` struct **SHALL** use `omitempty` JSON tag to support optional inclusion.

**Verification:** Marshaling `Facts` without derived fields set omits them from JSON output.

---

## 6. Engine Management Requirements

### REQ-ENGINE-001: Engine Constructor
The engine **SHALL** provide a `NewEngine(store RuleStore) (*Engine, error)` constructor.

**Verification:** Constructor creates engine successfully with valid store.

### REQ-ENGINE-002: Add Rule Method
The engine **SHALL** provide `AddRule(rule *Rule) error` that validates, compiles, and stores the rule.

**Verification:** Adding a valid rule succeeds and the rule is stored and compiled.

### REQ-ENGINE-003: Add Rule Validation
The `AddRule` method **SHALL** validate that the rule compiles before adding to the store.

**Verification:** Adding a rule with invalid expression returns compilation error and does not store the rule.

### REQ-ENGINE-004: Add Rule Atomicity
If compilation succeeds but storage fails, the `AddRule` method **SHALL** remove the compiled program from cache.

**Verification:** Simulate storage failure after compilation; verify compiled program is not in cache.

### REQ-ENGINE-005: Update Rule Method
The engine **SHALL** provide `UpdateRule(rule *Rule) error` that recompiles and updates the rule.

**Verification:** Updating a rule with new expression compiles and updates successfully.

### REQ-ENGINE-006: Update Rule Validation
The `UpdateRule` method **SHALL** validate that the new expression compiles before updating.

**Verification:** Updating with invalid expression returns error and preserves old rule.

### REQ-ENGINE-007: Delete Rule Method
The engine **SHALL** provide `DeleteRule(ruleID string) error` that removes the rule from store and cache.

**Verification:** After deletion, rule is not in store and compiled program is removed from cache.

### REQ-ENGINE-008: Delete Non-Existent Rule
The `DeleteRule` method **SHALL** return an error if the rule ID does not exist.

**Verification:** Deleting non-existent rule returns error.

---

## 7. Concurrency Requirements

### REQ-CONCUR-001: Thread-Safe Store
The `InMemoryRuleStore` **SHALL** be safe for concurrent access from multiple goroutines.

**Verification:** Concurrent Add/Get/Update/Delete operations do not cause data races or panics.

### REQ-CONCUR-002: Thread-Safe Engine
The `Engine` **SHALL** be safe for concurrent read access (evaluation) from multiple goroutines.

**Verification:** Concurrent `Evaluate()` calls on the same engine do not cause data races or panics.

### REQ-CONCUR-003: Thread-Safe Compilation
The `Engine` **SHALL** be safe for concurrent compilation operations (adding/updating rules).

**Verification:** Concurrent `AddRule()` and `UpdateRule()` calls do not cause data races or panics.

### REQ-CONCUR-004: Read-Write Lock
The `Engine` **SHALL** use read-write locks to allow concurrent reads while protecting writes.

**Verification:** Multiple concurrent `Evaluate()` calls can proceed while `AddRule()` blocks, and vice versa.

---

## 8. Error Handling Requirements

### REQ-ERROR-001: No Panics in Production Code
The engine **SHALL NOT** use `panic()` in production code paths (excluding example code).

**Verification:** Code review confirms no panic statements in non-example code.

### REQ-ERROR-002: Descriptive Errors
All error messages **SHALL** include context about what operation failed and why.

**Verification:** Error messages include operation name and reason for failure.

### REQ-ERROR-003: Error Wrapping
Errors **SHALL** be wrapped with additional context using `fmt.Errorf()` with `%w` verb.

**Verification:** Errors can be unwrapped to access original error using `errors.Unwrap()`.

### REQ-ERROR-004: Nil Checks
Methods that accept pointers **SHALL** check for nil and return appropriate errors.

**Verification:** Passing nil pointer to `AddRule()` returns error instead of panicking.

---

## 9. Performance Requirements

### REQ-PERF-001: No Recompilation on Evaluation
The engine **SHALL NOT** recompile rule expressions during evaluation.

**Verification:** Profiling shows no compilation activity during `Evaluate()` calls.

### REQ-PERF-002: Compilation Time Limit
Rule compilation **SHOULD** complete within 100ms for expressions under 1KB.

**Verification:** Benchmark test shows compilation time < 100ms for typical expressions.

### REQ-PERF-003: Evaluation Time Limit
Rule evaluation **SHOULD** complete within 1ms for simple expressions (< 10 operations).

**Verification:** Benchmark test shows evaluation time < 1ms for simple rules.

---

## 10. Documentation Requirements

### REQ-DOC-001: Public API Documentation
All public types, interfaces, and functions **SHALL** have Godoc comments.

**Verification:** `go doc` output shows documentation for all public symbols.

### REQ-DOC-002: Example Code
The package **SHALL** include at least one example function demonstrating basic usage.

**Verification:** Example code exists and runs successfully via `go test`.

### REQ-DOC-003: Interface Documentation
Each interface **SHALL** document the contract and expected behavior of its methods.

**Verification:** Interface Godoc includes method contracts and behavior descriptions.

---

## 11. Testing Requirements

### REQ-TEST-001: Unit Test Coverage
The package **SHALL** have unit tests for all public functions and methods.

**Verification:** `go test -cover` shows coverage for all public APIs.

### REQ-TEST-002: Error Path Testing
Unit tests **SHALL** cover error paths and edge cases.

**Verification:** Tests exist for invalid inputs, missing data, and error conditions.

### REQ-TEST-003: Concurrent Testing
Tests **SHALL** verify thread-safety using concurrent goroutines and race detector.

**Verification:** `go test -race` passes without data races.

### REQ-TEST-004: Example Tests
Example functions **SHALL** be tested using Go's example test framework.

**Verification:** `go test` runs and verifies example output.

---

## Requirement Traceability Matrix

| Requirement ID | Category | Priority | Verification Method |
|---------------|----------|----------|---------------------|
| REQ-TYPE-001 | Type System | MUST | Compilation test |
| REQ-TYPE-002 | Type System | MUST | Struct existence test |
| REQ-TYPE-003 | Type System | MUST | CEL evaluation test |
| REQ-TYPE-004 | Type System | MUST | JSON marshal/unmarshal test |
| REQ-STORE-001 | Storage | MUST | Interface compliance test |
| REQ-STORE-002 | Storage | MUST | Struct field test |
| REQ-STORE-003 | Storage | MUST | Implementation test |
| REQ-STORE-004 | Storage | MUST | Duplicate ID test |
| REQ-STORE-005 | Storage | MUST | Not found test |
| REQ-STORE-006 | Storage | MUST | Timestamp test |
| REQ-STORE-007 | Storage | MUST | Filter test |
| REQ-COMPILE-001 | Compilation | MUST | Environment creation test |
| REQ-COMPILE-002 | Compilation | MUST | Successful compilation test |
| REQ-COMPILE-003 | Compilation | MUST | Compilation error test |
| REQ-COMPILE-004 | Compilation | MUST | Type checking test |
| REQ-COMPILE-005 | Compilation | MUST | Cache verification test |
| REQ-COMPILE-006 | Compilation | MUST | Initialization test |
| REQ-COMPILE-007 | Compilation | MUST | Trace details test |
| REQ-EVAL-001 | Evaluation | MUST | Single evaluation test |
| REQ-EVAL-002 | Evaluation | MUST | Batch evaluation test |
| REQ-EVAL-003 | Evaluation | MUST | Match result test |
| REQ-EVAL-004 | Evaluation | MUST | Result structure test |
| REQ-EVAL-005 | Evaluation | MUST | Non-boolean test |
| REQ-EVAL-006 | Evaluation | MUST | Error handling test |
| REQ-EVAL-007 | Evaluation | MUST | Continue-on-error test |
| REQ-EVAL-008 | Evaluation | MUST | Map facts test |
| REQ-EVAL-009 | Evaluation | MUST | Nested access test |
| REQ-EVAL-010 | Evaluation | MUST | Missing field test |
| REQ-DERIVED-001 | Derived Fields | MUST | Precomputed evaluation test |
| REQ-DERIVED-002 | Derived Fields | MUST | Struct existence test |
| REQ-DERIVED-003 | Derived Fields | MUST | JSON omitempty test |
| REQ-ENGINE-001 | Engine Mgmt | MUST | Constructor test |
| REQ-ENGINE-002 | Engine Mgmt | MUST | Add rule test |
| REQ-ENGINE-003 | Engine Mgmt | MUST | Add validation test |
| REQ-ENGINE-004 | Engine Mgmt | MUST | Rollback test |
| REQ-ENGINE-005 | Engine Mgmt | MUST | Update rule test |
| REQ-ENGINE-006 | Engine Mgmt | MUST | Update validation test |
| REQ-ENGINE-007 | Engine Mgmt | MUST | Delete rule test |
| REQ-ENGINE-008 | Engine Mgmt | MUST | Delete error test |
| REQ-CONCUR-001 | Concurrency | MUST | Race detector test |
| REQ-CONCUR-002 | Concurrency | MUST | Race detector test |
| REQ-CONCUR-003 | Concurrency | MUST | Race detector test |
| REQ-CONCUR-004 | Concurrency | MUST | Lock behavior test |
| REQ-ERROR-001 | Error Handling | MUST | Code review |
| REQ-ERROR-002 | Error Handling | MUST | Error message test |
| REQ-ERROR-003 | Error Handling | MUST | Error wrapping test |
| REQ-ERROR-004 | Error Handling | MUST | Nil pointer test |
| REQ-PERF-001 | Performance | MUST | Profiling test |
| REQ-PERF-002 | Performance | SHOULD | Benchmark test |
| REQ-PERF-003 | Performance | SHOULD | Benchmark test |
| REQ-DOC-001 | Documentation | MUST | Godoc review |
| REQ-DOC-002 | Documentation | MUST | Example test |
| REQ-DOC-003 | Documentation | MUST | Godoc review |
| REQ-TEST-001 | Testing | MUST | Coverage report |
| REQ-TEST-002 | Testing | MUST | Test review |
| REQ-TEST-003 | Testing | MUST | Race detector |
| REQ-TEST-004 | Testing | MUST | Example test run |

---

## Summary Statistics

- **Total Requirements:** 58
- **MUST Requirements:** 55
- **SHOULD Requirements:** 2
- **Categories:** 11

Each requirement is designed to be independently verifiable and will map directly to one or more test cases.
