.PHONY: test test-unit test-integration test-all test-race test-coverage help

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

test: test-unit ## Run unit tests (default)

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test ./rules -v

test-integration: ## Run integration tests only (requires Docker)
	@echo "Running integration tests..."
	go test ./rules -v -tags=integration -timeout=5m

test-all: ## Run all tests (unit + integration)
	@echo "Running all tests..."
	go test ./rules -v -tags=integration -timeout=5m

test-race: ## Run all tests with race detector
	@echo "Running tests with race detector..."
	go test ./rules -v -race -tags=integration -timeout=5m

test-coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	go test ./rules -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-integration: ## Generate integration test coverage report
	@echo "Generating integration test coverage report..."
	go test ./rules -tags=integration -coverprofile=coverage-integration.out -timeout=5m
	go tool cover -html=coverage-integration.out -o coverage-integration.html
	@echo "Integration coverage report generated: coverage-integration.html"

clean: ## Clean test artifacts and coverage reports
	@echo "Cleaning test artifacts..."
	rm -f coverage.out coverage-integration.out coverage.html coverage-integration.html

deps: ## Install test dependencies
	@echo "Installing test dependencies..."
	go get github.com/google/cel-go/cel
	go get github.com/testcontainers/testcontainers-go
	go get github.com/testcontainers/testcontainers-go/wait
	go get github.com/lib/pq

docker-pull: ## Pre-pull Docker images for integration tests
	@echo "Pulling PostgreSQL Docker image..."
	docker pull postgres:15-alpine

# Migration commands
migrate-up: ## Run database migrations (reads DATABASE_URL from .env)
	@./scripts/migrate.sh up

migrate-down: ## Rollback database migrations
	@./scripts/migrate.sh down

migrate-version: ## Show current migration version
	@./scripts/migrate.sh version

migrate-force: ## Force migration version (usage: make migrate-force VERSION=1)
	@./scripts/migrate.sh force $(VERSION)
