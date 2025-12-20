# Testing Guide

This document describes how to run the test suite for the rules engine.

## Unit Tests

Unit tests use in-memory storage and don't require external dependencies.

```bash
go test ./rules -v
```

To run with race detection:

```bash
go test ./rules -v -race
```

## Integration Tests

Integration tests use testcontainers to spin up a real PostgreSQL instance. These tests require Docker to be running.

### Prerequisites

1. **Docker**: Make sure Docker is installed and running
   ```bash
   docker --version
   ```

2. **Go dependencies**: Install required packages
   ```bash
   go get github.com/testcontainers/testcontainers-go
   go get github.com/testcontainers/testcontainers-go/wait
   go get github.com/lib/pq
   ```

### Running Integration Tests

Integration tests are tagged with `// +build integration` to separate them from unit tests.

Run integration tests:
```bash
go test ./rules -v -tags=integration
```

Run integration tests with race detection:
```bash
go test ./rules -v -race -tags=integration
```

Run integration tests with timeout (recommended, as container startup can be slow):
```bash
go test ./rules -v -tags=integration -timeout=5m
```

### What the Integration Tests Cover

1. **TestPostgresRuleStore_BasicCRUD**
   - Tests Add, Get, Update, Delete operations
   - Verifies active/inactive rule filtering
   - Validates timestamps

2. **TestPostgresRuleStore_TenantIsolation**
   - Creates two tenants with separate rule stores
   - Verifies tenants cannot access each other's rules
   - Ensures `ListActive()` only returns tenant-scoped rules

3. **TestPostgresRuleStore_DuplicateRuleID**
   - Tests that duplicate rule IDs are rejected

4. **TestPostgresRuleStore_UpdateNonExistent**
   - Verifies error handling when updating non-existent rules

5. **TestPostgresRuleStore_DeleteNonExistent**
   - Verifies error handling when deleting non-existent rules

6. **TestMultiTenantEngine_WithDatabase**
   - Full integration test: multiple tenants with engines
   - Tests rule compilation, evaluation, and tenant isolation at engine level
   - Verifies tenants can't evaluate each other's rules

7. **TestCascadingDelete**
   - Tests that deleting a tenant cascades to delete all their rules
   - Validates foreign key constraints

8. **TestRuleOrdering**
   - Verifies rules are returned in `created_at` ascending order
   - Tests ordering guarantees for rule evaluation

## Running All Tests

Run both unit and integration tests:
```bash
go test ./rules -v -tags=integration -race -timeout=5m
```

## CI/CD Considerations

For CI environments (GitHub Actions, GitLab CI, etc.):

1. Ensure Docker is available in the CI environment
2. Use the integration tag to run integration tests separately:
   ```yaml
   # Example GitHub Actions
   - name: Run Unit Tests
     run: go test ./rules -v -race

   - name: Run Integration Tests
     run: go test ./rules -v -tags=integration -timeout=5m
   ```

## Troubleshooting

### "Cannot connect to Docker daemon"
- Make sure Docker is running: `docker ps`
- Check Docker socket permissions

### "Timeout waiting for container to start"
- Increase timeout in test: `-timeout=10m`
- Check Docker resource limits (CPU/memory)
- Pull the image manually first: `docker pull postgres:15-alpine`

### "Migration file not found"
- Integration tests look for `migrations/001_initial_schema.sql`
- Make sure the file exists relative to the test file
- Check file path in `setupTestDB()` function

### Tests are slow
- First run will download the PostgreSQL Docker image (~80MB)
- Subsequent runs reuse the cached image
- Consider running integration tests separately from unit tests in CI

## Test Coverage

Generate coverage report:
```bash
go test ./rules -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Generate coverage for integration tests:
```bash
go test ./rules -tags=integration -coverprofile=coverage-integration.out
go tool cover -html=coverage-integration.out
```
