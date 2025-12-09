# Database Migrations Guide

This guide explains how to manage database migrations for the rules engine.

## Setup

### 1. Configure Database Connection

Create a `.env` file in the project root with your database URL:

```bash
cp .env.example .env
```

Edit `.env` and set your database URL:

```bash
DATABASE_URL=postgresql://postgres:mysecretpassword@localhost/rules
```

The format is: `postgresql://username:password@host:port/database`

### 2. Install Dependencies

```bash
go mod download
```

## Running Migrations

### Apply All Migrations (Up)

This will apply all pending migrations to bring your database up to date:

```bash
make migrate-up
```

Or using the script directly:

```bash
./scripts/migrate.sh up
```

### Rollback All Migrations (Down)

This will rollback all migrations, dropping all tables:

```bash
make migrate-down
```

**⚠️ WARNING:** This will delete all data! Use with caution.

### Check Migration Version

See what version your database is currently at:

```bash
make migrate-version
```

### Force Migration Version

If a migration fails partway through, you may need to force the version:

```bash
make migrate-force VERSION=1
```

## Migration Files

Migrations are stored in the `migrations/` directory with the following naming convention:

```
{version}_{description}.{direction}.sql
```

Examples:
- `000001_initial_schema.up.sql` - First migration (up)
- `000001_initial_schema.down.sql` - First migration rollback (down)

### Current Migrations

**000001_initial_schema** - Creates initial database schema:
- `tenants` table - Store tenant information
- `schemas` table - Store tenant schema definitions (JSONB)
- `rules` table - Store CEL rule expressions
- `derived_fields` table - Store computed field definitions
- `schema_changelog` table - Audit trail for schema changes
- Indexes for performance
- Triggers for automatic `updated_at` timestamps

## Creating New Migrations

When you need to modify the database schema:

### 1. Create Migration Files

Create two files for each migration (up and down):

```bash
# Create the up migration (applies changes)
touch migrations/000002_add_user_metadata.up.sql

# Create the down migration (reverts changes)
touch migrations/000002_add_user_metadata.down.sql
```

### 2. Write the Up Migration

Example `000002_add_user_metadata.up.sql`:

```sql
ALTER TABLE tenants
ADD COLUMN metadata JSONB DEFAULT '{}';

CREATE INDEX idx_tenants_metadata ON tenants USING GIN(metadata);
```

### 3. Write the Down Migration

Example `000002_add_user_metadata.down.sql`:

```sql
DROP INDEX IF EXISTS idx_tenants_metadata;

ALTER TABLE tenants
DROP COLUMN IF EXISTS metadata;
```

### 4. Test the Migration

Always test both directions:

```bash
# Apply the migration
make migrate-up

# Verify it worked
psql $DATABASE_URL -c "\d tenants"

# Rollback to test down migration
make migrate-down

# Re-apply
make migrate-up
```

## Troubleshooting

### "Dirty database version"

If a migration fails partway through, the database may be marked as "dirty":

```
error: Dirty database version 1. Fix and force version.
```

To fix:

1. Manually fix any partial changes in the database
2. Force the version:
   ```bash
   make migrate-force VERSION=1
   ```

### "No change" error

This is not actually an error - it means your database is already up to date:

```
No migrations to run (database is up to date)
```

### Connection refused

If you can't connect to the database:

1. Check that PostgreSQL is running:
   ```bash
   psql $DATABASE_URL -c "SELECT 1"
   ```

2. Verify your `.env` file has the correct URL

3. Check the database exists:
   ```bash
   psql postgresql://postgres:mysecretpassword@localhost/postgres -c "SELECT datname FROM pg_database"
   ```

4. Create the database if needed:
   ```bash
   psql postgresql://postgres:mysecretpassword@localhost/postgres -c "CREATE DATABASE rules"
   ```

### Permission denied

If you get permission errors:

1. Check database user has proper permissions:
   ```sql
   GRANT ALL PRIVILEGES ON DATABASE rules TO postgres;
   ```

2. Ensure the user can create extensions:
   ```sql
   ALTER USER postgres WITH SUPERUSER;
   ```

## Best Practices

### 1. Always Create Both Up and Down Migrations

Every up migration should have a corresponding down migration that perfectly reverses it.

### 2. Test Before Committing

Always test migrations on a local database before committing:

```bash
# Start fresh
make migrate-down
make migrate-up

# Verify schema
psql $DATABASE_URL -c "\dt"
```

### 3. Never Modify Existing Migrations

Once a migration has been applied to production, never modify it. Instead:
- Create a new migration to make changes
- Keep the history clean and linear

### 4. Use Transactions Carefully

PostgreSQL migrations run in transactions by default, but some operations like `CREATE INDEX CONCURRENTLY` cannot run in a transaction.

### 5. Backup Before Running

Always backup production data before running migrations:

```bash
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d_%H%M%S).sql
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Database Migrations

on:
  push:
    branches: [main]

jobs:
  migrate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run Migrations
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
        run: make migrate-up
```

## Manual Migration (Without Make)

If you prefer not to use the Makefile:

```bash
# Load environment variables
export $(cat .env | grep -v '^#' | xargs)

# Run migration
go run cmd/migrate/main.go -command up -database "$DATABASE_URL"
```

Or set the database URL directly:

```bash
go run cmd/migrate/main.go \
  -command up \
  -database "postgresql://postgres:mysecretpassword@localhost/rules"
```

## Schema Versioning

The migration tool tracks the current version in a `schema_migrations` table:

```sql
SELECT * FROM schema_migrations;
```

This table is automatically created and managed by golang-migrate. Never modify it manually.
