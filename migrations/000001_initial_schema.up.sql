-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Tenants
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_name ON tenants(name);

-- Schemas
CREATE TABLE schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    definition JSONB NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_tenant_version UNIQUE(tenant_id, version)
);

CREATE INDEX idx_schemas_tenant_active ON schemas(tenant_id, active) WHERE active = true;
CREATE INDEX idx_schemas_definition ON schemas USING GIN(definition);

-- Rules
CREATE TABLE rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    expression TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_tenant_rule_name UNIQUE(tenant_id, name)
);

CREATE INDEX idx_rules_tenant_active ON rules(tenant_id, active) WHERE active = true;
CREATE INDEX idx_rules_created_at ON rules(created_at);

-- Derived Fields
CREATE TABLE derived_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    expression TEXT NOT NULL,
    dependencies JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_tenant_derived_name UNIQUE(tenant_id, name)
);

CREATE INDEX idx_derived_fields_tenant ON derived_fields(tenant_id);
CREATE INDEX idx_derived_fields_dependencies ON derived_fields USING GIN(dependencies);

-- Schema Changelog (audit trail)
CREATE TABLE schema_changelog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    schema_id UUID NOT NULL REFERENCES schemas(id) ON DELETE CASCADE,
    changed_by VARCHAR(255),
    change_type VARCHAR(50) NOT NULL,
    rules_recompiled INTEGER,
    rules_failed INTEGER,
    error_details JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_changelog_tenant ON schema_changelog(tenant_id, created_at DESC);
CREATE INDEX idx_changelog_schema ON schema_changelog(schema_id);

-- Trigger to update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rules_updated_at BEFORE UPDATE ON rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
