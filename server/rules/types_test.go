package rules

import (
	"errors"
	"testing"
	"time"
)

// TestRuleStructure verifies REQ-STORE-002: Rule SHALL contain all required fields
func TestRuleStructure(t *testing.T) {
	now := time.Now()

	rule := &Rule{
		ID:         "test-rule-1",
		Name:       "Test Rule",
		Expression: `User.Age >= 18`,
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Verify all fields are present and have correct types
	tests := []struct {
		name     string
		field    string
		value    any
		expected any
	}{
		{"ID field", "ID", rule.ID, "test-rule-1"},
		{"Name field", "Name", rule.Name, "Test Rule"},
		{"Expression field", "Expression", rule.Expression, `User.Age >= 18`},
		{"Active field", "Active", rule.Active, true},
		{"CreatedAt field", "CreatedAt", rule.CreatedAt, now},
		{"UpdatedAt field", "UpdatedAt", rule.UpdatedAt, now},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Rule.%s = %v, want %v", tt.field, tt.value, tt.expected)
			}
		})
	}
}

// TestRuleFieldTypes verifies that Rule fields have the correct Go types
func TestRuleFieldTypes(t *testing.T) {
	rule := &Rule{}

	// Use type assertions to verify field types at compile time
	var _ string = rule.ID
	var _ string = rule.Name
	var _ string = rule.Expression
	var _ bool = rule.Active
	var _ time.Time = rule.CreatedAt
	var _ time.Time = rule.UpdatedAt

	// If this compiles, types are correct
	t.Log("All Rule fields have correct types")
}

// TestEvaluationResultStructure verifies REQ-EVAL-004: EvaluationResult SHALL contain all required fields
func TestEvaluationResultStructure(t *testing.T) {
	testErr := errors.New("test error")
	trace := map[string]any{
		"step1": "evaluated",
		"step2": true,
	}

	result := &EvaluationResult{
		RuleID:   "rule-1",
		RuleName: "Test Rule",
		Matched:  true,
		Error:    testErr,
		Trace:    trace,
	}

	// Verify all fields are present and have correct values
	tests := []struct {
		name     string
		field    string
		value    any
		expected any
	}{
		{"RuleID field", "RuleID", result.RuleID, "rule-1"},
		{"RuleName field", "RuleName", result.RuleName, "Test Rule"},
		{"Matched field", "Matched", result.Matched, true},
		{"Error field", "Error", result.Error, testErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("EvaluationResult.%s = %v, want %v", tt.field, tt.value, tt.expected)
			}
		})
	}

	// Verify Trace is set (it's an any type so we just check it's not nil)
	if result.Trace == nil {
		t.Error("EvaluationResult.Trace should not be nil when set")
	}
}

// TestEvaluationResultFieldTypes verifies that EvaluationResult fields have the correct Go types
func TestEvaluationResultFieldTypes(t *testing.T) {
	result := &EvaluationResult{}

	// Use type assertions to verify field types at compile time
	var _ string = result.RuleID
	var _ string = result.RuleName
	var _ bool = result.Matched
	var _ error = result.Error
	var _ any = result.Trace

	// If this compiles, types are correct
	t.Log("All EvaluationResult fields have correct types")
}

// TestEvaluationResultWithNilError verifies that Error field can be nil
func TestEvaluationResultWithNilError(t *testing.T) {
	result := &EvaluationResult{
		RuleID:   "rule-1",
		RuleName: "Test Rule",
		Matched:  true,
		Error:    nil, // Should be allowed
		Trace:    nil, // Trace is also optional
	}

	if result.Error != nil {
		t.Errorf("EvaluationResult.Error = %v, want nil", result.Error)
	}

	if result.Trace != nil {
		t.Errorf("EvaluationResult.Trace = %v, want nil", result.Trace)
	}
}

// TestDerivedFieldStructure verifies REQ-DERIVED-002: DerivedField SHALL contain required fields
func TestDerivedFieldStructure(t *testing.T) {
	derived := &DerivedField{
		Name:       "isAdult",
		Expression: `User.Age >= 18`,
	}

	// Verify all fields are present and have correct values
	tests := []struct {
		name     string
		field    string
		value    any
		expected any
	}{
		{"Name field", "Name", derived.Name, "isAdult"},
		{"Expression field", "Expression", derived.Expression, `User.Age >= 18`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("DerivedField.%s = %v, want %v", tt.field, tt.value, tt.expected)
			}
		})
	}
}

// TestDerivedFieldFieldTypes verifies that DerivedField fields have the correct Go types
func TestDerivedFieldFieldTypes(t *testing.T) {
	derived := &DerivedField{}

	// Use type assertions to verify field types at compile time
	var _ string = derived.Name
	var _ string = derived.Expression

	// If this compiles, types are correct
	t.Log("All DerivedField fields have correct types")
}

// TestDerivedFieldCELExpression verifies that Expression can hold valid CEL expressions
func TestDerivedFieldCELExpression(t *testing.T) {
	testCases := []struct {
		name       string
		expression string
	}{
		{"Simple comparison", `User.Age >= 18`},
		{"Boolean logic", `User.Age >= 18 && User.Citizenship == "CANADA"`},
		{"Arithmetic", `Transaction.Amount * 0.1 > 100`},
		{"String operations", `User.Name.startsWith("John")`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			derived := &DerivedField{
				Name:       "testField",
				Expression: tc.expression,
			}

			if derived.Expression != tc.expression {
				t.Errorf("DerivedField.Expression = %q, want %q", derived.Expression, tc.expression)
			}
		})
	}
}
