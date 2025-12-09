-- Drop triggers
DROP TRIGGER IF EXISTS update_rules_updated_at ON rules;
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS schema_changelog;
DROP TABLE IF EXISTS derived_fields;
DROP TABLE IF EXISTS rules;
DROP TABLE IF EXISTS schemas;
DROP TABLE IF EXISTS tenants;

-- Drop extension
DROP EXTENSION IF EXISTS "pgcrypto";
