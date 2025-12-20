package rules

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
)

// Engine manages CEL environment and rule compilation/evaluation
// Satisfies REQ-CONCUR-002: Thread-safe for concurrent reads (RWMutex)
// Satisfies REQ-CONCUR-003: Thread-safe for concurrent compilation
// Satisfies REQ-CONCUR-004: Uses RWMutex for concurrent reads
type Engine struct {
	env      *cel.Env
	store    RuleStore
	cache    RulesCache              // cache for active rules list
	programs map[string]cel.Program // ruleID -> compiled program
	mu       sync.RWMutex
}

// NewEngine creates a new rules engine with a default CEL environment
// Satisfies REQ-ENGINE-001: Engine constructor
// Satisfies REQ-COMPILE-001: Creates CEL environment
// Satisfies REQ-COMPILE-006: Compiles all active rules on initialization
func NewEngine(store RuleStore) (*Engine, error) {
	// Create CEL environment with variable declarations for map-based facts
	// We're using map[string]any for facts, so we declare the top-level objects as dynamic types
	env, err := cel.NewEnv(
		cel.Variable("User", cel.DynType),
		cel.Variable("Transaction", cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return NewEngineWithEnv(env, store)
}

// NewEngineWithEnv creates a new rules engine with a custom CEL environment
// This allows multi-tenant deployments to use schema-specific environments
func NewEngineWithEnv(env *cel.Env, store RuleStore) (*Engine, error) {
	en := &Engine{
		env:      env,
		store:    store,
		cache:    NewInMemoryRulesCache(DefaultCacheConfig()),
		programs: make(map[string]cel.Program),
	}

	if err := en.CompileAllRules(); err != nil {
		return nil, fmt.Errorf("failed to compile rules: %w", err)
	}

	return en, nil
}

// CompileRule compiles a single rule expression to a CEL program
// Satisfies REQ-COMPILE-002: Compiles CEL expressions
// Satisfies REQ-COMPILE-003: Returns descriptive compilation errors
// Satisfies REQ-COMPILE-004: Performs type checking
// Satisfies REQ-COMPILE-005: Caches compiled programs
// Satisfies REQ-COMPILE-007: Enables tracing with OptTrackState
// Satisfies REQ-SEC-001: Applies cost limit to prevent runaway expressions
func (en *Engine) CompileRule(ruleID, expression string) error {
	ast, issues := en.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("compile error: %w", issues.Err())
	}

	// REQ-SEC-001: Apply cost limit and enable tracking
	// Cost limit of 1,000,000 prevents resource exhaustion from malicious/complex expressions
	prog, err := en.env.Program(ast,
		cel.EvalOptions(cel.OptTrackState),
		cel.CostLimit(1000000),
	)
	if err != nil {
		return fmt.Errorf("program creation error: %w", err)
	}

	en.mu.Lock()
	en.programs[ruleID] = prog
	en.mu.Unlock()

	return nil
}

// Evaluate evaluates a single rule against the provided facts
// Satisfies REQ-EVAL-001: Evaluate single rule
// Satisfies REQ-EVAL-003: Returns boolean match result
// Satisfies REQ-EVAL-005: Non-boolean expressions treated as false
// Satisfies REQ-EVAL-006: Evaluation errors are captured
func (en *Engine) Evaluate(ruleID string, facts map[string]any) (*EvaluationResult, error) {
	rule, err := en.store.Get(ruleID)
	if err != nil {
		return nil, err
	}

	en.mu.RLock()
	prog, exists := en.programs[ruleID]
	en.mu.RUnlock()

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

// CompileAllRules compiles all active rules from the store
// Also populates the cache with the active rules list
func (en *Engine) CompileAllRules() error {
	rules, err := en.store.ListActive()
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if err := en.CompileRule(rule.ID, rule.Expression); err != nil {
			return fmt.Errorf("failed to compile rule %s: %w", rule.ID, err)
		}
	}

	// Populate cache with active rules
	en.cache.Set(rules)

	return nil
}

// AddRule adds a new rule to the store and compiles it
// Satisfies REQ-ENGINE-002: Validates, compiles, and stores rule
// Satisfies REQ-ENGINE-003: Validates rule compiles before adding to store
// Satisfies REQ-ENGINE-004: Removes compiled program if store fails (atomicity)
func (en *Engine) AddRule(r *Rule) error {
	// Check if rule already exists before compiling (to avoid overwriting existing programs)
	_, err := en.store.Get(r.ID)
	if err == nil {
		// Rule already exists
		return fmt.Errorf("rule with ID %s already exists", r.ID)
	}

	// Validate that the rule compiles
	if err := en.CompileRule(r.ID, r.Expression); err != nil {
		return fmt.Errorf("rule validation failed: %w", err)
	}

	// Then add to store
	if err := en.store.Add(r); err != nil {
		// Remove from compiled programs if store fails
		en.mu.Lock()
		delete(en.programs, r.ID)
		en.mu.Unlock()
		return err
	}

	// Invalidate cache since rules list changed
	en.cache.Invalidate()

	return nil
}

// UpdateRule updates an existing rule and recompiles it
// Satisfies REQ-ENGINE-005: Recompiles rule on update
// Satisfies REQ-ENGINE-006: Validates new expression before updating
func (en *Engine) UpdateRule(r *Rule) error {
	// Compile the new expression to validate it
	if err := en.CompileRule(r.ID, r.Expression); err != nil {
		return fmt.Errorf("rule validation failed: %w", err)
	}

	// Update in store
	if err := en.store.Update(r); err != nil {
		return err
	}

	// Invalidate cache since rule metadata might have changed
	en.cache.Invalidate()

	return nil
}

// DeleteRule removes a rule from the store and compiled programs
// Satisfies REQ-ENGINE-007: Removes rule from store and cache
func (en *Engine) DeleteRule(ruleID string) error {
	if err := en.store.Delete(ruleID); err != nil {
		return err
	}

	en.mu.Lock()
	delete(en.programs, ruleID)
	en.mu.Unlock()

	// Invalidate cache since rules list changed
	en.cache.Invalidate()

	return nil
}

// EvaluateAll evaluates all active rules against the provided facts
// Satisfies REQ-EVAL-002: Evaluates all active rules
// Satisfies REQ-EVAL-007: Continues evaluating even if some rules fail
// Uses cache to avoid database query on every evaluation
func (en *Engine) EvaluateAll(facts map[string]any) ([]*EvaluationResult, error) {
	// Try to get rules from cache first
	rules := en.cache.Get()

	// If cache miss, fetch from database and populate cache
	if rules == nil {
		var err error
		rules, err = en.store.ListActive()
		if err != nil {
			return nil, err
		}
		en.cache.Set(rules)
	}

	results := make([]*EvaluationResult, 0, len(rules))
	for _, rule := range rules {
		// Use cached rule data instead of fetching from DB
		// This eliminates 10-100 DB queries per evaluation request
		en.mu.RLock()
		prog, exists := en.programs[rule.ID]
		en.mu.RUnlock()

		if !exists {
			results = append(results, &EvaluationResult{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Matched:  false,
				Error:    fmt.Errorf("rule %s is not compiled", rule.ID),
			})
			continue
		}

		out, details, err := prog.Eval(facts)
		if err != nil {
			results = append(results, &EvaluationResult{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Matched:  false,
				Error:    err,
			})
			continue
		}

		matched := false
		if boolVal, ok := out.Value().(bool); ok {
			matched = boolVal
		}

		results = append(results, &EvaluationResult{
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Matched:  matched,
			Trace:    details.State(),
		})
	}

	return results, nil
}
