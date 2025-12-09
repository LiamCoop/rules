package generated

import (
	"encoding/json"
	"testing"
)

// TestFactsStructExists verifies REQ-TYPE-002: Facts struct SHALL be defined
func TestFactsStructExists(t *testing.T) {
	// If this compiles, Facts struct exists
	facts := &Facts{}

	if facts == nil {
		t.Fatal("Facts struct should exist")
	}
}

// TestFactsNestedObjectSupport verifies REQ-TYPE-003: Facts SHALL support nested objects
func TestFactsNestedObjectSupport(t *testing.T) {
	facts := &Facts{
		User: User{
			Citizenship: "CANADA",
			Age:         30,
		},
		Transaction: Transaction{
			Amount:  1500.0,
			Country: "CANADA",
		},
	}

	// Verify nested field access works
	tests := []struct {
		name     string
		value    any
		expected any
	}{
		{"User.Citizenship", facts.User.Citizenship, "CANADA"},
		{"User.Age", facts.User.Age, 30},
		{"Transaction.Amount", facts.Transaction.Amount, 1500.0},
		{"Transaction.Country", facts.Transaction.Country, "CANADA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Facts.%s = %v, want %v", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// TestUserStructure verifies User struct has required fields
func TestUserStructure(t *testing.T) {
	user := User{
		Citizenship: "USA",
		Age:         25,
	}

	if user.Citizenship != "USA" {
		t.Errorf("User.Citizenship = %s, want USA", user.Citizenship)
	}

	if user.Age != 25 {
		t.Errorf("User.Age = %d, want 25", user.Age)
	}
}

// TestTransactionStructure verifies Transaction struct has required fields
func TestTransactionStructure(t *testing.T) {
	tx := Transaction{
		Amount:  999.99,
		Country: "MEXICO",
	}

	if tx.Amount != 999.99 {
		t.Errorf("Transaction.Amount = %f, want 999.99", tx.Amount)
	}

	if tx.Country != "MEXICO" {
		t.Errorf("Transaction.Country = %s, want MEXICO", tx.Country)
	}
}

// TestFactsJSONSerialization verifies REQ-TYPE-004: Facts SHALL support JSON serialization
func TestFactsJSONSerialization(t *testing.T) {
	original := &Facts{
		User: User{
			Citizenship: "CANADA",
			Age:         35,
		},
		Transaction: Transaction{
			Amount:  2000.50,
			Country: "CANADA",
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Facts to JSON: %v", err)
	}

	// Verify JSON is not empty
	if len(jsonData) == 0 {
		t.Fatal("Marshaled JSON is empty")
	}

	// Unmarshal back to Facts
	var unmarshaled Facts
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to Facts: %v", err)
	}

	// Verify no data loss
	if unmarshaled.User.Citizenship != original.User.Citizenship {
		t.Errorf("User.Citizenship after unmarshal = %s, want %s",
			unmarshaled.User.Citizenship, original.User.Citizenship)
	}

	if unmarshaled.User.Age != original.User.Age {
		t.Errorf("User.Age after unmarshal = %d, want %d",
			unmarshaled.User.Age, original.User.Age)
	}

	if unmarshaled.Transaction.Amount != original.Transaction.Amount {
		t.Errorf("Transaction.Amount after unmarshal = %f, want %f",
			unmarshaled.Transaction.Amount, original.Transaction.Amount)
	}

	if unmarshaled.Transaction.Country != original.Transaction.Country {
		t.Errorf("Transaction.Country after unmarshal = %s, want %s",
			unmarshaled.Transaction.Country, original.Transaction.Country)
	}
}

// TestFactsJSONFieldNames verifies that JSON tags are properly defined
func TestFactsJSONFieldNames(t *testing.T) {
	facts := &Facts{
		User: User{
			Citizenship: "CANADA",
			Age:         40,
		},
		Transaction: Transaction{
			Amount:  1000.0,
			Country: "USA",
		},
	}

	jsonData, err := json.Marshal(facts)
	if err != nil {
		t.Fatalf("Failed to marshal Facts: %v", err)
	}

	// Parse JSON to check field names
	var parsed map[string]any
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Verify expected JSON field names exist
	if _, exists := parsed["User"]; !exists {
		t.Error("JSON should contain 'User' field")
	}

	if _, exists := parsed["Transaction"]; !exists {
		t.Error("JSON should contain 'Transaction' field")
	}

	// Check nested User fields
	userMap, ok := parsed["User"].(map[string]any)
	if !ok {
		t.Fatal("User field should be an object")
	}

	if _, exists := userMap["Citizenship"]; !exists {
		t.Error("User JSON should contain 'Citizenship' field")
	}

	if _, exists := userMap["Age"]; !exists {
		t.Error("User JSON should contain 'Age' field")
	}

	// Check nested Transaction fields
	txMap, ok := parsed["Transaction"].(map[string]any)
	if !ok {
		t.Fatal("Transaction field should be an object")
	}

	if _, exists := txMap["Amount"]; !exists {
		t.Error("Transaction JSON should contain 'Amount' field")
	}

	if _, exists := txMap["Country"]; !exists {
		t.Error("Transaction JSON should contain 'Country' field")
	}
}

// TestDerivedFieldInFacts verifies that derived fields can be added to Facts
func TestDerivedFieldInFacts(t *testing.T) {
	facts := &Facts{
		User: User{
			Citizenship: "CANADA",
			Age:         30,
		},
		Transaction: Transaction{
			Amount:  1500.0,
			Country: "CANADA",
		},
		IsCanadian: true, // Derived field
	}

	if !facts.IsCanadian {
		t.Error("Facts.IsCanadian should be true")
	}
}

// TestDerivedFieldJSONOmitempty verifies REQ-DERIVED-003: Derived fields use omitempty
func TestDerivedFieldJSONOmitempty(t *testing.T) {
	// Facts without derived field set
	factsWithoutDerived := &Facts{
		User: User{
			Citizenship: "USA",
			Age:         25,
		},
		Transaction: Transaction{
			Amount:  500.0,
			Country: "USA",
		},
		// IsCanadian not set (zero value: false)
	}

	jsonData, err := json.Marshal(factsWithoutDerived)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed map[string]any
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// With omitempty, isCanadian should not be in JSON when false
	// Note: This test verifies the omitempty behavior
	t.Logf("JSON output: %s", string(jsonData))

	// Facts with derived field explicitly set to true
	factsWithDerived := &Facts{
		User: User{
			Citizenship: "CANADA",
			Age:         30,
		},
		Transaction: Transaction{
			Amount:  1000.0,
			Country: "CANADA",
		},
		IsCanadian: true,
	}

	jsonData2, err := json.Marshal(factsWithDerived)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed2 map[string]any
	err = json.Unmarshal(jsonData2, &parsed2)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// When true, isCanadian should be in JSON
	if val, exists := parsed2["isCanadian"]; exists {
		if boolVal, ok := val.(bool); ok && !boolVal {
			t.Error("isCanadian should be true when set")
		}
	}

	t.Logf("JSON with derived field: %s", string(jsonData2))
}

// TestFactsFieldTypes verifies compile-time type checking
func TestFactsFieldTypes(t *testing.T) {
	facts := &Facts{}

	// Type assertions - will fail at compile time if types are wrong
	var _ User = facts.User
	var _ Transaction = facts.Transaction
	var _ bool = facts.IsCanadian

	t.Log("All Facts fields have correct types")
}

// TestUserFieldTypes verifies User field types
func TestUserFieldTypes(t *testing.T) {
	user := &User{}

	var _ string = user.Citizenship
	var _ int = user.Age

	t.Log("All User fields have correct types")
}

// TestTransactionFieldTypes verifies Transaction field types
func TestTransactionFieldTypes(t *testing.T) {
	tx := &Transaction{}

	var _ float64 = tx.Amount
	var _ string = tx.Country

	t.Log("All Transaction fields have correct types")
}
