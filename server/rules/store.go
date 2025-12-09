package rules

import (
    "fmt"
    "sync"
    "time"
)

// RuleStore manages rule persistence and retrieval
// Satisfies REQ-STORE-001: RuleStore interface with required methods
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

// InMemoryRuleStore implements RuleStore using an in-memory map
// Satisfies REQ-STORE-003: In-memory RuleStore implementation
// Satisfies REQ-CONCUR-001: Thread-safe with RWMutex
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

// Add adds a new rule to the store
// Satisfies REQ-STORE-004: Enforces unique rule IDs
// Satisfies REQ-STORE-006: Sets CreatedAt and UpdatedAt timestamps
func (s *InMemoryRuleStore) Add(rule *Rule) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.rules[rule.ID]; exists {
        return fmt.Errorf("rule with ID %s already exists", rule.ID)
    }

    now := time.Now()
    rule.CreatedAt = now
    rule.UpdatedAt = now
    s.rules[rule.ID] = rule
    return nil
}

// Get retrieves a rule by ID
// Satisfies REQ-STORE-005: Returns error when rule ID does not exist
func (s *InMemoryRuleStore) Get(id string) (*Rule, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    rule, exists := s.rules[id]
    if !exists {
        return nil, fmt.Errorf("rule with ID %s not found", id)
    }
    return rule, nil
}

// ListActive returns all active rules
// Satisfies REQ-STORE-007: Filters to return only active rules
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

// Update updates an existing rule
// Satisfies REQ-STORE-006: Updates UpdatedAt timestamp, preserves CreatedAt
func (s *InMemoryRuleStore) Update(rule *Rule) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    existing, exists := s.rules[rule.ID]
    if !exists {
        return fmt.Errorf("rule with ID %s not found", rule.ID)
    }

    // Preserve original CreatedAt timestamp
    rule.CreatedAt = existing.CreatedAt
    rule.UpdatedAt = time.Now()
    s.rules[rule.ID] = rule
    return nil
}

// Delete removes a rule from the store
func (s *InMemoryRuleStore) Delete(id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.rules[id]; !exists {
        return fmt.Errorf("rule with ID %s not found", id)
    }

    delete(s.rules, id)
    return nil
}

