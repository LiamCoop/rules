package rules

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresRuleStore implements RuleStore backed by PostgreSQL
type PostgresRuleStore struct {
	db       *sql.DB
	tenantID string
}

// NewPostgresRuleStore creates a new PostgreSQL-backed RuleStore for a specific tenant
func NewPostgresRuleStore(db *sql.DB, tenantID string) *PostgresRuleStore {
	return &PostgresRuleStore{
		db:       db,
		tenantID: tenantID,
	}
}

// Add inserts a new rule into the database
func (s *PostgresRuleStore) Add(rule *Rule) error {
	// Check if rule already exists
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM rules WHERE id = $1 AND tenant_id = $2)
	`, rule.ID, s.tenantID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check rule existence: %w", err)
	}
	if exists {
		return fmt.Errorf("rule with ID %s already exists", rule.ID)
	}

	_, err = s.db.Exec(`
		INSERT INTO rules (id, tenant_id, name, expression, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, rule.ID, s.tenantID, rule.Name, rule.Expression, rule.Active,
		rule.CreatedAt, rule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert rule: %w", err)
	}

	return nil
}

// Get retrieves a rule by ID
func (s *PostgresRuleStore) Get(id string) (*Rule, error) {
	var rule Rule
	err := s.db.QueryRow(`
		SELECT id, name, expression, active, created_at, updated_at
		FROM rules
		WHERE id = $1 AND tenant_id = $2
	`, id, s.tenantID).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Expression,
		&rule.Active,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	return &rule, nil
}

// ListActive returns all active rules for the tenant
func (s *PostgresRuleStore) ListActive() ([]*Rule, error) {
	rows, err := s.db.Query(`
		SELECT id, name, expression, active, created_at, updated_at
		FROM rules
		WHERE tenant_id = $1 AND active = true
		ORDER BY created_at ASC
	`, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active rules: %w", err)
	}
	defer rows.Close()

	var rulesList []*Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.Name, &r.Expression, &r.Active,
			&r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rulesList = append(rulesList, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rulesList, nil
}

// Update modifies an existing rule
func (s *PostgresRuleStore) Update(rule *Rule) error {
	// Check if rule exists
	_, err := s.Get(rule.ID)
	if err != nil {
		return err
	}

	// Update the timestamp
	rule.UpdatedAt = time.Now()

	result, err := s.db.Exec(`
		UPDATE rules
		SET name = $1, expression = $2, active = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6
	`, rule.Name, rule.Expression, rule.Active, rule.UpdatedAt, rule.ID, s.tenantID)

	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("rule %s not found", rule.ID)
	}

	return nil
}

// Delete removes a rule from the database
func (s *PostgresRuleStore) Delete(id string) error {
	result, err := s.db.Exec(`
		DELETE FROM rules
		WHERE id = $1 AND tenant_id = $2
	`, id, s.tenantID)

	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("rule %s not found", id)
	}

	return nil
}
