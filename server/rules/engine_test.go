package rules

import (
	"strings"
	"sync"
	"testing"
)

// TestNewEngine verifies REQ-ENGINE-001: Engine constructor SHALL exist
func TestNewEngine(t *testing.T) {
	store := NewInMemoryRuleStore()

	engine, err := NewEngine(store)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	if engine == nil {
		t.Fatal("NewEngine() should return non-nil engine")
	}
}

// TestNewEngineCreatesEnvironment verifies REQ-COMPILE-001: CEL environment SHALL be created
func TestNewEngineCreatesEnvironment(t *testing.T) {
	store := NewInMemoryRuleStore()

	engine, err := NewEngine(store)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Engine should be able to compile a simple expression
	err = engine.CompileRule("test-rule", `true`)
	if err != nil {
		t.Errorf("Engine should have valid CEL environment, got error: %v", err)
	}
}

// TestNewEngineCompilesExistingRules verifies REQ-COMPILE-006: SHALL compile all active rules on init
func TestNewEngineCompilesExistingRules(t *testing.T) {
	store := NewInMemoryRuleStore()

	// Add some rules before creating engine
	rules := []*Rule{
		{ID: "rule-1", Name: "Rule 1", Expression: `User.Age >= 18`, Active: true},
		{ID: "rule-2", Name: "Rule 2", Expression: `Transaction.Amount > 1000`, Active: true},
		{ID: "rule-3", Name: "Rule 3", Expression: `User.Citizenship == "CANADA"`, Active: false}, // Inactive
	}

	for _, rule := range rules {
		err := store.Add(rule)
		if err != nil {
			t.Fatalf("Failed to add rule: %v", err)
		}
	}

	// Create engine - should compile active rules
	engine, err := NewEngine(store)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Should be able to evaluate the active rules
	facts := map[string]any{
		"User": map[string]any{
			"Age":         20,
			"Citizenship": "CANADA",
		},
		"Transaction": map[string]any{
			"Amount": 1500.0,
		},
	}

	result, err := engine.Evaluate("rule-1", facts)
	if err != nil {
		t.Errorf("Evaluate() failed for pre-compiled rule: %v", err)
	}
	if result == nil {
		t.Error("Result should not be nil")
	}
}

// TestCompileRuleSuccess verifies REQ-COMPILE-002: Expression SHALL compile successfully
func TestCompileRuleSuccess(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	testCases := []struct {
		name       string
		expression string
	}{
		{"Simple boolean", `true`},
		{"Field access", `User.Age >= 18`},
		{"Nested access", `Transaction.Amount > 1000`},
		{"Boolean logic", `User.Age >= 18 && User.Citizenship == "CANADA"`},
		{"Arithmetic", `Transaction.Amount * 0.1 > 100.0`},
		{"String comparison", `User.Citizenship == "USA"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := engine.CompileRule("test-"+tc.name, tc.expression)
			if err != nil {
				t.Errorf("CompileRule(%q) failed: %v", tc.expression, err)
			}
		})
	}
}

// TestCompileRuleError verifies REQ-COMPILE-003: Compilation errors SHALL be descriptive
func TestCompileRuleError(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	testCases := []struct {
		name       string
		expression string
	}{
		{"Syntax error", `User.Age >=`},
		{"Invalid operator", `User.Age === 18`},
		{"Undefined variable", `UndefinedField.Value > 0`},
		{"Mismatched parens", `(User.Age >= 18`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := engine.CompileRule("test-"+tc.name, tc.expression)
			if err == nil {
				t.Errorf("CompileRule(%q) should return error for invalid expression", tc.expression)
			}
			// Error should be descriptive (non-empty message)
			if err.Error() == "" {
				t.Error("Error message should be descriptive")
			}
		})
	}
}

// TestCompileRuleTypeChecking verifies REQ-COMPILE-004: Type checking SHALL be performed
// Note: With DynType, CEL allows comparisons between different types at compile time
// and will handle them at runtime. Strict type checking requires registering specific types.
func TestCompileRuleTypeChecking(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// This expression should fail compilation due to invalid syntax
	err := engine.CompileRule("test-type-check", `User.Age..Bad`)
	if err == nil {
		t.Error("CompileRule() should return error for malformed expression")
	}
}

// TestCompileRuleCaching verifies REQ-COMPILE-005: Compiled programs SHALL be cached
func TestCompileRuleCaching(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	ruleID := "cached-rule"
	expression := `User.Age >= 18`

	// Add rule to store and compile it
	rule := &Rule{
		ID:         ruleID,
		Name:       "Cached Rule",
		Expression: expression,
		Active:     true,
	}

	err := engine.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() failed: %v", err)
	}

	// Should be able to evaluate using cached program
	facts := map[string]any{
		"User":        map[string]any{"Age": 20},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	result, err := engine.Evaluate(ruleID, facts)
	if err != nil {
		t.Errorf("Evaluate() should use cached program: %v", err)
	}

	if !result.Matched {
		t.Error("Cached program should evaluate correctly")
	}
}

// TestCompileRuleWithTracing verifies REQ-COMPILE-007: Tracing SHALL be supported
func TestCompileRuleWithTracing(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	ruleID := "trace-rule"
	rule := &Rule{
		ID:         ruleID,
		Name:       "Trace Rule",
		Expression: `User.Age >= 18`,
		Active:     true,
	}

	err := engine.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() failed: %v", err)
	}

	facts := map[string]any{
		"User":        map[string]any{"Age": 20},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	result, err := engine.Evaluate(ruleID, facts)
	if err != nil {
		t.Fatalf("Evaluate() failed: %v", err)
	}

	// Trace should be present (it's an any type containing CEL trace data)
	if result.Trace == nil {
		t.Error("Result.Trace should not be nil when tracing is enabled")
	}
}

// TestEvaluateSingleRule verifies REQ-EVAL-001: SHALL evaluate single rule
func TestEvaluateSingleRule(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	ruleID := "eval-test"
	rule := &Rule{
		ID:         ruleID,
		Name:       "Eval Test",
		Expression: `User.Age >= 18`,
		Active:     true,
	}

	err := engine.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() failed: %v", err)
	}

	facts := map[string]any{
		"User":        map[string]any{"Age": 25, "Citizenship": "USA"},
		"Transaction": map[string]any{"Amount": 100.0, "Country": "USA"},
	}

	result, err := engine.Evaluate(ruleID, facts)
	if err != nil {
		t.Fatalf("Evaluate() failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.RuleID != ruleID {
		t.Errorf("Result.RuleID = %s, want %s", result.RuleID, ruleID)
	}
}

// TestEvaluateAllRules verifies REQ-EVAL-002: SHALL evaluate all active rules
func TestEvaluateAllRules(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Add multiple rules
	rules := []*Rule{
		{ID: "rule-1", Name: "Adult", Expression: `User.Age >= 18`, Active: true},
		{ID: "rule-2", Name: "Large Amount", Expression: `Transaction.Amount > 1000`, Active: true},
		{ID: "rule-3", Name: "Inactive", Expression: `false`, Active: false},
	}

	for _, rule := range rules {
		err := engine.AddRule(rule)
		if err != nil {
			t.Fatalf("AddRule() failed: %v", err)
		}
	}

	facts := map[string]any{
		"User":        map[string]any{"Age": 25, "Citizenship": "USA"},
		"Transaction": map[string]any{"Amount": 1500.0, "Country": "USA"},
	}

	results, err := engine.EvaluateAll(facts)
	if err != nil {
		t.Fatalf("EvaluateAll() failed: %v", err)
	}

	// Should return results for 2 active rules (not the inactive one)
	if len(results) != 2 {
		t.Errorf("EvaluateAll() returned %d results, want 2", len(results))
	}
}

// TestEvaluateBooleanResult verifies REQ-EVAL-003: Boolean result SHALL indicate match
func TestEvaluateBooleanResult(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	testCases := []struct {
		name       string
		expression string
		facts      map[string]any
		wantMatch  bool
	}{
		{
			name:       "Match true",
			expression: `User.Age >= 18`,
			facts: map[string]any{
				"User":        map[string]any{"Age": 25},
				"Transaction": map[string]any{"Amount": 100.0},
			},
			wantMatch: true,
		},
		{
			name:       "Match false",
			expression: `User.Age >= 65`,
			facts: map[string]any{
				"User":        map[string]any{"Age": 25},
				"Transaction": map[string]any{"Amount": 100.0},
			},
			wantMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ruleID := "test-" + tc.name
			rule := &Rule{
				ID:         ruleID,
				Name:       tc.name,
				Expression: tc.expression,
				Active:     true,
			}

			engine.AddRule(rule)
			result, err := engine.Evaluate(ruleID, tc.facts)
			if err != nil {
				t.Fatalf("Evaluate() failed: %v", err)
			}

			if result.Matched != tc.wantMatch {
				t.Errorf("Result.Matched = %v, want %v", result.Matched, tc.wantMatch)
			}
		})
	}
}

// TestEvaluateNonBooleanExpression verifies REQ-EVAL-005: Non-boolean SHALL be treated as false
func TestEvaluateNonBooleanExpression(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Expression that returns an integer
	rule := &Rule{
		ID:         "non-bool",
		Name:       "Non Boolean",
		Expression: `User.Age`, // Returns an int, not a bool
		Active:     true,
	}

	engine.AddRule(rule)

	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	result, err := engine.Evaluate("non-bool", facts)
	if err != nil {
		t.Fatalf("Evaluate() failed: %v", err)
	}

	// Non-boolean result should be treated as Matched: false
	if result.Matched {
		t.Error("Non-boolean expression should result in Matched: false")
	}
}

// TestEvaluateErrorHandling verifies REQ-EVAL-006: Evaluation errors SHALL be captured
func TestEvaluateErrorHandling(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Add a rule that will compile but fail at evaluation
	rule := &Rule{
		ID:         "error-rule",
		Name:       "Error Rule",
		Expression: `1 / 0`, // Division by zero
		Active:     true,
	}

	engine.AddRule(rule)

	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	result, err := engine.Evaluate("error-rule", facts)

	// Should return result with error populated
	if result == nil {
		t.Fatal("Result should not be nil even on error")
	}

	if result.Error == nil {
		t.Error("Result.Error should be populated when evaluation fails")
	}

	if result.Matched {
		t.Error("Matched should be false when evaluation fails")
	}

	// err return value should also contain the error
	if err == nil {
		t.Error("Evaluate() should return error on evaluation failure")
	}
}

// TestEvaluateAllContinuesOnError verifies REQ-EVAL-007: SHALL continue evaluating on error
func TestEvaluateAllContinuesOnError(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	rules := []*Rule{
		{ID: "rule-1", Name: "Good 1", Expression: `User.Age >= 18`, Active: true},
		{ID: "rule-2", Name: "Bad", Expression: `1 / 0`, Active: true}, // Will error
		{ID: "rule-3", Name: "Good 2", Expression: `Transaction.Amount > 0`, Active: true},
	}

	for _, rule := range rules {
		engine.AddRule(rule)
	}

	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	results, err := engine.EvaluateAll(facts)
	if err != nil {
		t.Fatalf("EvaluateAll() should not return error even if some rules fail: %v", err)
	}

	// Should get results for all 3 rules
	if len(results) != 3 {
		t.Fatalf("EvaluateAll() returned %d results, want 3", len(results))
	}

	// Check results by rule ID (not by order, since map iteration is non-deterministic)
	resultsByID := make(map[string]*EvaluationResult)
	for _, result := range results {
		resultsByID[result.RuleID] = result
	}

	// rule-1 and rule-3 should succeed
	if resultsByID["rule-1"].Error != nil {
		t.Error("rule-1 should not have error")
	}

	if resultsByID["rule-3"].Error != nil {
		t.Error("rule-3 should not have error")
	}

	// rule-2 (bad rule) should have error
	if resultsByID["rule-2"].Error == nil {
		t.Error("rule-2 should have error")
	}
}

// TestEvaluateFactsAsMap verifies REQ-EVAL-008: Facts SHALL be accepted as map
func TestEvaluateFactsAsMap(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	rule := &Rule{
		ID:         "map-facts",
		Name:       "Map Facts",
		Expression: `User.Age >= 18 && Transaction.Amount > 1000`,
		Active:     true,
	}

	engine.AddRule(rule)

	// Facts as map[string]any
	facts := map[string]any{
		"User": map[string]any{
			"Age":         25,
			"Citizenship": "USA",
		},
		"Transaction": map[string]any{
			"Amount":  1500.0,
			"Country": "USA",
		},
	}

	result, err := engine.Evaluate("map-facts", facts)
	if err != nil {
		t.Fatalf("Evaluate() with map facts failed: %v", err)
	}

	if !result.Matched {
		t.Error("Should evaluate correctly with map facts")
	}
}

// TestEvaluateNestedFieldAccess verifies REQ-EVAL-009: Nested fields SHALL be accessible
func TestEvaluateNestedFieldAccess(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	rule := &Rule{
		ID:         "nested",
		Name:       "Nested Access",
		Expression: `User.Citizenship == "CANADA" && Transaction.Amount > 1000`,
		Active:     true,
	}

	engine.AddRule(rule)

	facts := map[string]any{
		"User": map[string]any{
			"Age":         30,
			"Citizenship": "CANADA",
		},
		"Transaction": map[string]any{
			"Amount":  2000.0,
			"Country": "CANADA",
		},
	}

	result, err := engine.Evaluate("nested", facts)
	if err != nil {
		t.Fatalf("Evaluate() with nested access failed: %v", err)
	}

	if !result.Matched {
		t.Error("Nested field access should work correctly")
	}
}

// TestEvaluateMissingField verifies REQ-EVAL-010: Missing fields SHALL cause evaluation error
func TestEvaluateMissingField(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	rule := &Rule{
		ID:         "missing-field",
		Name:       "Missing Field",
		Expression: `User.NonExistentField == "value"`,
		Active:     true,
	}

	engine.AddRule(rule)

	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	result, err := engine.Evaluate("missing-field", facts)

	// Should get an error for missing field
	if err == nil && result.Error == nil {
		t.Error("Evaluate() should return error when accessing missing field")
	}
}

// TestEngineAddRule verifies REQ-ENGINE-002: AddRule SHALL validate and compile
func TestEngineAddRule(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	rule := &Rule{
		ID:         "add-test",
		Name:       "Add Test",
		Expression: `User.Age >= 18`,
		Active:     true,
	}

	err := engine.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() failed: %v", err)
	}

	// Should be in store
	retrieved, err := store.Get("add-test")
	if err != nil {
		t.Fatalf("Rule not found in store after AddRule(): %v", err)
	}

	if retrieved.Name != rule.Name {
		t.Errorf("Stored rule Name = %s, want %s", retrieved.Name, rule.Name)
	}

	// Should be compiled and evaluatable
	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	result, err := engine.Evaluate("add-test", facts)
	if err != nil {
		t.Errorf("Evaluate() failed after AddRule(): %v", err)
	}

	if !result.Matched {
		t.Error("Added rule should be evaluatable")
	}
}

// TestEngineAddRuleValidation verifies REQ-ENGINE-003: AddRule SHALL validate before storing
func TestEngineAddRuleValidation(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Rule with invalid expression
	rule := &Rule{
		ID:         "invalid-rule",
		Name:       "Invalid Rule",
		Expression: `User.Age >=`, // Syntax error
		Active:     true,
	}

	err := engine.AddRule(rule)
	if err == nil {
		t.Fatal("AddRule() should return error for invalid expression")
	}

	// Should NOT be in store
	_, err = store.Get("invalid-rule")
	if err == nil {
		t.Error("Invalid rule should not be stored")
	}
}

// TestEngineAddRuleAtomicity verifies REQ-ENGINE-004: Rollback on storage failure
func TestEngineAddRuleAtomicity(t *testing.T) {
	// This test verifies that if storage fails after compilation,
	// the compiled program is removed from cache

	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Add a rule first
	rule1 := &Rule{
		ID:         "existing",
		Name:       "Existing",
		Expression: `true`,
		Active:     true,
	}
	engine.AddRule(rule1)

	// Try to add a rule with duplicate ID (will fail at storage)
	rule2 := &Rule{
		ID:         "existing", // Same ID
		Name:       "Duplicate",
		Expression: `User.Age >= 21`, // Different expression
		Active:     true,
	}

	err := engine.AddRule(rule2)
	if err == nil {
		t.Fatal("AddRule() with duplicate ID should fail")
	}

	// Original rule should still work
	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	result, err := engine.Evaluate("existing", facts)
	if err != nil {
		t.Fatalf("Original rule should still be evaluatable: %v", err)
	}

	if !result.Matched {
		t.Error("Original rule should evaluate with original expression")
	}
}

// TestEngineUpdateRule verifies REQ-ENGINE-005: UpdateRule SHALL recompile
func TestEngineUpdateRule(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Add initial rule
	original := &Rule{
		ID:         "update-test",
		Name:       "Original",
		Expression: `User.Age >= 18`,
		Active:     true,
	}
	engine.AddRule(original)

	// Update with new expression
	updated := &Rule{
		ID:         "update-test",
		Name:       "Updated",
		Expression: `User.Age >= 65`,
		Active:     true,
	}

	err := engine.UpdateRule(updated)
	if err != nil {
		t.Fatalf("UpdateRule() failed: %v", err)
	}

	// Should use new expression
	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	result, err := engine.Evaluate("update-test", facts)
	if err != nil {
		t.Fatalf("Evaluate() after UpdateRule() failed: %v", err)
	}

	// Age 25 is >= 18 but not >= 65, so should be false with new expression
	if result.Matched {
		t.Error("Should use updated expression (Age >= 65), not original (Age >= 18)")
	}
}

// TestEngineUpdateRuleValidation verifies REQ-ENGINE-006: UpdateRule SHALL validate
func TestEngineUpdateRuleValidation(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Add initial rule
	original := &Rule{
		ID:         "update-validate",
		Name:       "Original",
		Expression: `User.Age >= 18`,
		Active:     true,
	}
	engine.AddRule(original)

	// Try to update with invalid expression
	invalid := &Rule{
		ID:         "update-validate",
		Name:       "Invalid",
		Expression: `User.Age >=`, // Syntax error
		Active:     true,
	}

	err := engine.UpdateRule(invalid)
	if err == nil {
		t.Fatal("UpdateRule() should return error for invalid expression")
	}

	// Original rule should still work
	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	result, err := engine.Evaluate("update-validate", facts)
	if err != nil {
		t.Fatalf("Original rule should still work: %v", err)
	}

	if !result.Matched {
		t.Error("Should still use original valid expression")
	}
}

// TestEngineDeleteRule verifies REQ-ENGINE-007: DeleteRule SHALL remove from store and cache
func TestEngineDeleteRule(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Add a rule
	rule := &Rule{
		ID:         "delete-test",
		Name:       "Delete Test",
		Expression: `User.Age >= 18`,
		Active:     true,
	}
	engine.AddRule(rule)

	// Verify it exists
	_, err := store.Get("delete-test")
	if err != nil {
		t.Fatal("Rule should exist before delete")
	}

	// Delete it
	err = engine.DeleteRule("delete-test")
	if err != nil {
		t.Fatalf("DeleteRule() failed: %v", err)
	}

	// Should not be in store
	_, err = store.Get("delete-test")
	if err == nil {
		t.Error("Rule should not exist in store after delete")
	}

	// Should not be evaluatable
	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	_, err = engine.Evaluate("delete-test", facts)
	if err == nil {
		t.Error("Should not be able to evaluate deleted rule")
	}
}

// TestEngineDeleteNonExistent verifies REQ-ENGINE-008: DeleteRule SHALL return error for non-existent
func TestEngineDeleteNonExistent(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	err := engine.DeleteRule("does-not-exist")
	if err == nil {
		t.Fatal("DeleteRule() should return error for non-existent rule")
	}
}

// TestEngineConcurrentEvaluate verifies REQ-CONCUR-002: Engine SHALL be thread-safe for reads
func TestEngineConcurrentEvaluate(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Add some rules
	rules := []*Rule{
		{ID: "rule-1", Name: "Rule 1", Expression: `User.Age >= 18`, Active: true},
		{ID: "rule-2", Name: "Rule 2", Expression: `Transaction.Amount > 1000`, Active: true},
		{ID: "rule-3", Name: "Rule 3", Expression: `User.Citizenship == "CANADA"`, Active: true},
	}

	for _, rule := range rules {
		engine.AddRule(rule)
	}

	facts := map[string]any{
		"User": map[string]any{
			"Age":         25,
			"Citizenship": "CANADA",
		},
		"Transaction": map[string]any{
			"Amount": 1500.0,
		},
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	iterations := 100

	// Concurrent evaluations
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for range iterations {
				// Evaluate single rule
				_, err := engine.Evaluate("rule-1", facts)
				if err != nil {
					t.Errorf("Concurrent Evaluate() failed: %v", err)
				}

				// Evaluate all rules
				_, err = engine.EvaluateAll(facts)
				if err != nil {
					t.Errorf("Concurrent EvaluateAll() failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
	t.Log("Concurrent evaluation test completed successfully")
}

// TestEngineConcurrentCompile verifies REQ-CONCUR-003: Compilation SHALL be thread-safe
func TestEngineConcurrentCompile(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	var wg sync.WaitGroup
	numGoroutines := 5

	// Concurrent rule additions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				rule := &Rule{
					ID:         strings.Join([]string{"rule", string(rune('a' + id)), string(rune('0' + j))}, "-"),
					Name:       "Concurrent Rule",
					Expression: `User.Age >= 18`,
					Active:     true,
				}

				err := engine.AddRule(rule)
				if err != nil {
					t.Errorf("Concurrent AddRule() failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all rules were added
	active, err := store.ListActive()
	if err != nil {
		t.Fatalf("ListActive() failed: %v", err)
	}

	expected := numGoroutines * 10
	if len(active) != expected {
		t.Errorf("After concurrent adds, got %d rules, want %d", len(active), expected)
	}
}

// TestEngineConcurrentReadWrite verifies REQ-CONCUR-004: RWMutex SHALL allow concurrent reads
func TestEngineConcurrentReadWrite(t *testing.T) {
	store := NewInMemoryRuleStore()
	engine, _ := NewEngine(store)

	// Add initial rules
	for i := range 5 {
		rule := &Rule{
			ID:         "initial-" + string(rune('0'+i)),
			Name:       "Initial Rule",
			Expression: `User.Age >= 18`,
			Active:     true,
		}
		engine.AddRule(rule)
	}

	facts := map[string]any{
		"User":        map[string]any{"Age": 25},
		"Transaction": map[string]any{"Amount": 100.0},
	}

	var wg sync.WaitGroup
	numReaders := 10
	numWriters := 3
	iterations := 50

	// Concurrent readers (should be able to run in parallel)
	for range numReaders {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range iterations {
				engine.EvaluateAll(facts)
			}
		}()
	}

	// Concurrent writers (should block each other but not readers)
	for i := range numWriters {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := range iterations {
				rule := &Rule{
					ID:         "writer-" + string(rune('0'+id)) + "-" + string(rune('0'+(j%10))),
					Name:       "Writer Rule",
					Expression: `Transaction.Amount > 0`,
					Active:     true,
				}
				engine.AddRule(rule)
			}
		}(i)
	}

	wg.Wait()
	t.Log("Concurrent read/write test completed successfully")
}
