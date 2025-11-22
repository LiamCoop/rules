package rules

import "time"

// Rule represents a single evaluation rule
// Satisfies REQ-STORE-002: Rule SHALL contain all required fields
type Rule struct {
    ID         string
    Name       string
    Expression string
    Active     bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// EvaluationResult contains the outcome of evaluating a rule
// Satisfies REQ-EVAL-004: EvaluationResult SHALL contain all required fields
type EvaluationResult struct {
    RuleID   string
    RuleName string
    Matched  bool
    Error    error
    Trace    any // CEL evaluation trace (optional)
}

// DerivedField represents a computed field (for future code generation)
// Satisfies REQ-DERIVED-002: DerivedField SHALL contain required fields
type DerivedField struct {
    Name       string
    Expression string // CEL expression for computing the field
}

