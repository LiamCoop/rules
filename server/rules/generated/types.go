// Code in generated/types.go
// For MVP: Hand-written
// For Hybrid Service: Auto-generated from tenant schema
//
// Satisfies REQ-TYPE-001: Generated types directory structure

package generated

// User represents user information
type User struct {
	Citizenship string `json:"Citizenship"`
	Age         int    `json:"Age"`
}

// Transaction represents transaction data
type Transaction struct {
	Amount  float64 `json:"Amount"`
	Country string  `json:"Country"`
}

// Facts is the top-level container for all evaluation inputs
// Satisfies REQ-TYPE-002: Facts struct definition
// Satisfies REQ-TYPE-003: Nested object support (User, Transaction)
// Satisfies REQ-TYPE-004: JSON serialization via struct tags
type Facts struct {
	User        User        `json:"User"`
	Transaction Transaction `json:"Transaction"`

	// Derived fields (precomputed before evaluation)
	// Satisfies REQ-DERIVED-003: omitempty tag for optional derived fields
	// Future: These could be generated from DerivedField definitions
	IsCanadian bool `json:"isCanadian,omitempty"`
}
