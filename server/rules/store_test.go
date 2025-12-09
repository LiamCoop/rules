package rules

import (
	"sync"
	"testing"
	"time"
)

// TestRuleStoreInterfaceExists verifies REQ-STORE-001: RuleStore interface SHALL exist with required methods
func TestRuleStoreInterfaceExists(t *testing.T) {
	// This test verifies at compile-time that RuleStore interface exists
	// and InMemoryRuleStore implements it
	var _ RuleStore = (*InMemoryRuleStore)(nil)

	t.Log("RuleStore interface exists and has correct method signatures")
}

// TestNewInMemoryRuleStore verifies REQ-STORE-003: InMemoryRuleStore can be created
func TestNewInMemoryRuleStore(t *testing.T) {
	store := NewInMemoryRuleStore()

	if store == nil {
		t.Fatal("NewInMemoryRuleStore() should return a non-nil store")
	}
}

// TestInMemoryRuleStoreAdd verifies basic Add functionality
func TestInMemoryRuleStoreAdd(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &Rule{
		ID:         "test-1",
		Name:       "Test Rule",
		Expression: `User.Age >= 18`,
		Active:     true,
	}

	err := store.Add(rule)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	// Verify rule was added by retrieving it
	retrieved, err := store.Get("test-1")
	if err != nil {
		t.Fatalf("Get() failed after Add(): %v", err)
	}

	if retrieved.ID != rule.ID {
		t.Errorf("Retrieved rule ID = %s, want %s", retrieved.ID, rule.ID)
	}

	if retrieved.Name != rule.Name {
		t.Errorf("Retrieved rule Name = %s, want %s", retrieved.Name, rule.Name)
	}
}

// TestInMemoryRuleStoreAddDuplicate verifies REQ-STORE-004: Duplicate IDs SHALL return error
func TestInMemoryRuleStoreAddDuplicate(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule1 := &Rule{
		ID:         "duplicate-id",
		Name:       "First Rule",
		Expression: `User.Age >= 18`,
		Active:     true,
	}

	rule2 := &Rule{
		ID:         "duplicate-id", // Same ID
		Name:       "Second Rule",
		Expression: `User.Age >= 21`,
		Active:     true,
	}

	// First add should succeed
	err := store.Add(rule1)
	if err != nil {
		t.Fatalf("First Add() should succeed: %v", err)
	}

	// Second add with same ID should fail
	err = store.Add(rule2)
	if err == nil {
		t.Fatal("Add() with duplicate ID should return error")
	}

	// Verify first rule is still there and unchanged
	retrieved, err := store.Get("duplicate-id")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Name != "First Rule" {
		t.Errorf("Rule should not have been overwritten, Name = %s, want 'First Rule'", retrieved.Name)
	}
}

// TestInMemoryRuleStoreGet verifies basic Get functionality
func TestInMemoryRuleStoreGet(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &Rule{
		ID:         "get-test",
		Name:       "Get Test Rule",
		Expression: `Transaction.Amount > 1000`,
		Active:     true,
	}

	err := store.Add(rule)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	retrieved, err := store.Get("get-test")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.ID != rule.ID {
		t.Errorf("ID = %s, want %s", retrieved.ID, rule.ID)
	}

	if retrieved.Expression != rule.Expression {
		t.Errorf("Expression = %s, want %s", retrieved.Expression, rule.Expression)
	}
}

// TestInMemoryRuleStoreGetNotFound verifies REQ-STORE-005: Get SHALL return error for non-existent ID
func TestInMemoryRuleStoreGetNotFound(t *testing.T) {
	store := NewInMemoryRuleStore()

	_, err := store.Get("non-existent-id")
	if err == nil {
		t.Fatal("Get() with non-existent ID should return error")
	}
}

// TestInMemoryRuleStoreTimestamps verifies REQ-STORE-006: Timestamps SHALL be managed automatically
func TestInMemoryRuleStoreTimestamps(t *testing.T) {
	store := NewInMemoryRuleStore()

	beforeAdd := time.Now()

	rule := &Rule{
		ID:         "timestamp-test",
		Name:       "Timestamp Test Rule",
		Expression: `true`,
		Active:     true,
		// CreatedAt and UpdatedAt not set - should be set by store
	}

	err := store.Add(rule)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	afterAdd := time.Now()

	// Retrieve and check timestamps
	retrieved, err := store.Get("timestamp-test")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// CreatedAt should be set and within reasonable bounds
	if retrieved.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set by Add()")
	}

	if retrieved.CreatedAt.Before(beforeAdd) || retrieved.CreatedAt.After(afterAdd) {
		t.Errorf("CreatedAt = %v, should be between %v and %v",
			retrieved.CreatedAt, beforeAdd, afterAdd)
	}

	// UpdatedAt should be set and equal to CreatedAt initially
	if retrieved.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set by Add()")
	}

	if !retrieved.UpdatedAt.Equal(retrieved.CreatedAt) {
		t.Errorf("UpdatedAt = %v, should equal CreatedAt = %v on creation",
			retrieved.UpdatedAt, retrieved.CreatedAt)
	}
}

// TestInMemoryRuleStoreUpdate verifies Update functionality
func TestInMemoryRuleStoreUpdate(t *testing.T) {
	store := NewInMemoryRuleStore()

	original := &Rule{
		ID:         "update-test",
		Name:       "Original Name",
		Expression: `User.Age >= 18`,
		Active:     true,
	}

	err := store.Add(original)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	// Get original timestamps
	retrieved, _ := store.Get("update-test")
	originalCreatedAt := retrieved.CreatedAt
	originalUpdatedAt := retrieved.UpdatedAt

	// Wait a moment to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Update the rule
	updated := &Rule{
		ID:         "update-test",
		Name:       "Updated Name",
		Expression: `User.Age >= 21`,
		Active:     false,
	}

	err = store.Update(updated)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify changes
	retrieved, err = store.Get("update-test")
	if err != nil {
		t.Fatalf("Get() after Update() failed: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %s, want 'Updated Name'", retrieved.Name)
	}

	if retrieved.Expression != `User.Age >= 21` {
		t.Errorf("Expression = %s, want 'User.Age >= 21'", retrieved.Expression)
	}

	if retrieved.Active {
		t.Error("Active should be false after update")
	}

	// Verify CreatedAt unchanged
	if !retrieved.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("CreatedAt changed during Update, was %v, now %v",
			originalCreatedAt, retrieved.CreatedAt)
	}

	// Verify UpdatedAt changed
	if !retrieved.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("UpdatedAt = %v, should be after original %v",
			retrieved.UpdatedAt, originalUpdatedAt)
	}
}

// TestInMemoryRuleStoreUpdateNotFound verifies Update returns error for non-existent rule
func TestInMemoryRuleStoreUpdateNotFound(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &Rule{
		ID:         "does-not-exist",
		Name:       "Test",
		Expression: `true`,
		Active:     true,
	}

	err := store.Update(rule)
	if err == nil {
		t.Fatal("Update() with non-existent ID should return error")
	}
}

// TestInMemoryRuleStoreListActive verifies REQ-STORE-007: ListActive SHALL filter correctly
func TestInMemoryRuleStoreListActive(t *testing.T) {
	store := NewInMemoryRuleStore()

	// Add mix of active and inactive rules
	rules := []*Rule{
		{ID: "active-1", Name: "Active 1", Expression: `true`, Active: true},
		{ID: "inactive-1", Name: "Inactive 1", Expression: `true`, Active: false},
		{ID: "active-2", Name: "Active 2", Expression: `true`, Active: true},
		{ID: "inactive-2", Name: "Inactive 2", Expression: `true`, Active: false},
		{ID: "active-3", Name: "Active 3", Expression: `true`, Active: true},
	}

	for _, rule := range rules {
		err := store.Add(rule)
		if err != nil {
			t.Fatalf("Add() failed for %s: %v", rule.ID, err)
		}
	}

	// List active rules
	active, err := store.ListActive()
	if err != nil {
		t.Fatalf("ListActive() failed: %v", err)
	}

	// Should return exactly 3 active rules
	if len(active) != 3 {
		t.Fatalf("ListActive() returned %d rules, want 3", len(active))
	}

	// Verify all returned rules are active
	activeIDs := make(map[string]bool)
	for _, rule := range active {
		if !rule.Active {
			t.Errorf("ListActive() returned inactive rule: %s", rule.ID)
		}
		activeIDs[rule.ID] = true
	}

	// Verify correct rules were returned
	expectedIDs := []string{"active-1", "active-2", "active-3"}
	for _, id := range expectedIDs {
		if !activeIDs[id] {
			t.Errorf("ListActive() did not return expected rule: %s", id)
		}
	}
}

// TestInMemoryRuleStoreListActiveEmpty verifies ListActive works with no rules
func TestInMemoryRuleStoreListActiveEmpty(t *testing.T) {
	store := NewInMemoryRuleStore()

	active, err := store.ListActive()
	if err != nil {
		t.Fatalf("ListActive() failed: %v", err)
	}

	if len(active) != 0 {
		t.Errorf("ListActive() on empty store returned %d rules, want 0", len(active))
	}
}

// TestInMemoryRuleStoreListActiveAllInactive verifies ListActive with only inactive rules
func TestInMemoryRuleStoreListActiveAllInactive(t *testing.T) {
	store := NewInMemoryRuleStore()

	// Add only inactive rules
	for i := 1; i <= 3; i++ {
		rule := &Rule{
			ID:         "inactive-" + string(rune('0'+i)),
			Name:       "Inactive",
			Expression: `true`,
			Active:     false,
		}
		store.Add(rule)
	}

	active, err := store.ListActive()
	if err != nil {
		t.Fatalf("ListActive() failed: %v", err)
	}

	if len(active) != 0 {
		t.Errorf("ListActive() with all inactive returned %d rules, want 0", len(active))
	}
}

// TestInMemoryRuleStoreDelete verifies Delete functionality
func TestInMemoryRuleStoreDelete(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &Rule{
		ID:         "delete-test",
		Name:       "Delete Test",
		Expression: `true`,
		Active:     true,
	}

	err := store.Add(rule)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	// Verify rule exists
	_, err = store.Get("delete-test")
	if err != nil {
		t.Fatalf("Get() before Delete() failed: %v", err)
	}

	// Delete the rule
	err = store.Delete("delete-test")
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify rule no longer exists
	_, err = store.Get("delete-test")
	if err == nil {
		t.Fatal("Get() after Delete() should return error")
	}
}

// TestInMemoryRuleStoreDeleteNotFound verifies Delete returns error for non-existent rule
func TestInMemoryRuleStoreDeleteNotFound(t *testing.T) {
	store := NewInMemoryRuleStore()

	err := store.Delete("does-not-exist")
	if err == nil {
		t.Fatal("Delete() with non-existent ID should return error")
	}
}

// TestInMemoryRuleStoreConcurrentAdd verifies REQ-CONCUR-001: Store SHALL be thread-safe
func TestInMemoryRuleStoreConcurrentAdd(t *testing.T) {
	store := NewInMemoryRuleStore()

	var wg sync.WaitGroup
	numGoroutines := 10
	rulesPerGoroutine := 10

	// Concurrent adds
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < rulesPerGoroutine; j++ {
				rule := &Rule{
					ID:         string(rune('a'+goroutineID)) + "-" + string(rune('0'+j)),
					Name:       "Concurrent Rule",
					Expression: `true`,
					Active:     true,
				}

				err := store.Add(rule)
				if err != nil {
					t.Errorf("Concurrent Add() failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all rules were added
	active, err := store.ListActive()
	if err != nil {
		t.Fatalf("ListActive() after concurrent adds failed: %v", err)
	}

	expected := numGoroutines * rulesPerGoroutine
	if len(active) != expected {
		t.Errorf("After concurrent adds, got %d rules, want %d", len(active), expected)
	}
}

// TestInMemoryRuleStoreConcurrentReadWrite verifies concurrent reads and writes
func TestInMemoryRuleStoreConcurrentReadWrite(t *testing.T) {
	store := NewInMemoryRuleStore()

	// Pre-populate with some rules
	for i := 0; i < 10; i++ {
		rule := &Rule{
			ID:         "rule-" + string(rune('0'+i)),
			Name:       "Test Rule",
			Expression: `true`,
			Active:     true,
		}
		store.Add(rule)
	}

	var wg sync.WaitGroup
	numReaders := 5
	numWriters := 3
	iterations := 100

	// Concurrent readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				_, err := store.Get("rule-5")
				if err != nil {
					t.Errorf("Concurrent Get() failed: %v", err)
				}

				_, err = store.ListActive()
				if err != nil {
					t.Errorf("Concurrent ListActive() failed: %v", err)
				}
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				rule := &Rule{
					ID:         "writer-" + string(rune('0'+writerID)) + "-" + string(rune('0'+(j%10))),
					Name:       "Writer Rule",
					Expression: `true`,
					Active:     j%2 == 0,
				}

				// Add or update
				if j%2 == 0 {
					store.Add(rule)
				} else {
					// Try to update existing rule
					store.Update(&Rule{
						ID:         "rule-5",
						Name:       "Updated",
						Expression: `false`,
						Active:     true,
					})
				}
			}
		}(i)
	}

	wg.Wait()

	// Should complete without panics or data races
	t.Log("Concurrent read/write test completed successfully")
}

// TestInMemoryRuleStoreConcurrentDelete verifies concurrent deletes are safe
func TestInMemoryRuleStoreConcurrentDelete(t *testing.T) {
	store := NewInMemoryRuleStore()

	// Pre-populate with rules
	numRules := 20
	for i := 0; i < numRules; i++ {
		rule := &Rule{
			ID:         "delete-" + string(rune('a'+i)),
			Name:       "To Delete",
			Expression: `true`,
			Active:     true,
		}
		store.Add(rule)
	}

	var wg sync.WaitGroup

	// Multiple goroutines trying to delete the same rules
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < numRules; j++ {
				id := "delete-" + string(rune('a'+j))
				// Delete may fail if already deleted by another goroutine
				// That's okay - we just verify no panic/race
				store.Delete(id)
			}
		}()
	}

	wg.Wait()

	// All rules should be deleted
	active, err := store.ListActive()
	if err != nil {
		t.Fatalf("ListActive() after concurrent deletes failed: %v", err)
	}

	if len(active) != 0 {
		t.Errorf("After concurrent deletes, got %d rules, want 0", len(active))
	}
}
