.PHONY: help build build-api build-worker run-api run-worker test test-unit test-coverage test-verbose test-config test-logger test-clean clean migrate-up migrate-down migrate-create docker-up docker-down dev ci-lint ci-test ci-build ci-build-api ci-build-worker ci install-lint

# Load environment variables from .env file
include .env
export

# Variables
APP_NAME=job-api-service
WORKER_NAME=job-worker-service
BINARY_DIR=bin
API_BINARY_NAME=api-service
WORKER_BINARY_NAME=worker-service
DOCKER_COMPOSE=docker compose -f docker/docker-compose.yml
MIGRATIONS_DIR=migrations
DATABASE_URL=postgresql://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=$(DATABASE_SSLMODE)

## help: Display this help message
help:
	@echo "Available commands:"
	@echo "  make build         - Build all services (API + Worker)"
	@echo "  make build-api     - Build the API service binary"
	@echo "  make build-worker  - Build the Worker service binary"
	@echo "  make run-api       - Run the API service"
	@echo "  make run-worker    - Run the Worker service"
	@echo ""
	@echo "Testing:"
	@echo "  make test          - Run all tests with coverage"
	@echo "  make test-unit     - Run unit tests only (fast)"
	@echo "  make test-coverage - Run tests with detailed coverage report"
	@echo "  make test-verbose  - Run tests with verbose output"
	@echo "  make test-config   - Test config package only"
	@echo "  make test-logger   - Test logger package only"
	@echo "  make test-clean    - Remove test artifacts"
	@echo ""
	@echo "Development:"
	@echo "  make dev           - Run with hot reload (requires air)"
	@echo "  make clean         - Remove build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up     - Start PostgreSQL and RabbitMQ containers"
	@echo "  make docker-down   - Stop and remove containers"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up    - Run database migrations"
	@echo "  make migrate-down  - Rollback last migration"
	@echo "  make migrate-create NAME=migration_name - Create new migration files"
	@echo ""
	@echo "CI/CD:"
	@echo "  make ci            - Run all CI checks locally"
	@echo "  make ci-lint       - Run linter"
	@echo "  make ci-test       - Run tests for CI"
	@echo "  make ci-build      - Build all services for CI"
	@echo "  make ci-build-api  - Build API service for CI"
	@echo "  make ci-build-worker - Build Worker service for CI"
	@echo "  make install-lint  - Install golangci-lint"
	@echo ""

## build: Build all services
build: build-api build-worker
	@echo "All services built successfully"

## build-api: Build the API service binary
build-api:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BINARY_DIR)
	@go build -o ./$(BINARY_DIR)/$(API_BINARY_NAME) ./cmd/api-service/main.go
	@echo "Build complete: $(BINARY_DIR)/$(API_BINARY_NAME)"

## build-worker: Build the Worker service binary
build-worker:
	@echo "Building $(WORKER_NAME)..."
	@mkdir -p $(BINARY_DIR)
	@go build -o ./$(BINARY_DIR)/$(WORKER_BINARY_NAME) ./cmd/worker-service/main.go
	@echo "Build complete: $(BINARY_DIR)/$(WORKER_BINARY_NAME)"

## run-api: Run the API service
run-api:
	@echo "Starting $(APP_NAME)..."
	@go run cmd/api-service/main.go

## run-worker: Run the Worker service
run-worker:
	@echo "Starting $(WORKER_NAME)..."
	@go run cmd/worker-service/main.go

## test: Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-unit: Run unit tests only (short mode)
test-unit:
	@echo "Running unit tests..."
	@go test -v -short -race ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out
	@echo ""
	@echo "HTML coverage report: coverage.html"
	@go tool cover -html=coverage.out -o coverage.html

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v -race ./...

## test-config: Run config package tests only
test-config:
	@echo "Testing config package..."
	@go test -v ./internal/config/...

## test-logger: Run logger package tests only
test-logger:
	@echo "Testing logger package..."
	@go test -v ./shared/logger/...

## test-clean: Remove test artifacts
test-clean:
	@echo "Cleaning test artifacts..."
	@rm -f coverage.out coverage.html
	@echo "Test artifacts removed"

## dev: Run with hot reload using air
dev:
	@echo "Starting development mode with hot reload..."
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	@air

## clean: Remove build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## docker-up: Start PostgreSQL and RabbitMQ containers
docker-up:
	@echo "Starting Docker services..."
	@$(DOCKER_COMPOSE) up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@$(DOCKER_COMPOSE) ps

## docker-down: Stop and remove containers
docker-down:
	@echo "Stopping Docker services..."
	@$(DOCKER_COMPOSE) down
	@echo "Services stopped"

## docker-logs: View logs from Docker services
docker-logs:
	@$(DOCKER_COMPOSE) logs -f

## migrate-up: Run all pending migrations
migrate-up:
	@echo "Running migrations..."
	@migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up
	@echo "Migrations complete"

## migrate-down: Rollback last migration
migrate-down:
	@echo "Rolling back last migration..."
	@migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1
	@echo "Rollback complete"

## migrate-force: Force migration version (use with caution)
migrate-force:
	@echo "Forcing migration version to $(VERSION)..."
	@migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" force $(VERSION)

## migrate-version: Show current migration version
migrate-version:
	@migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" version

## migrate-create: Create new migration files
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(NAME)..."
	@migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)
	@echo "Migration files created"

## db-shell: Connect to PostgreSQL database
db-shell:
	@echo "Connecting to database..."
	@PGPASSWORD=$(DATABASE_PASSWORD) psql -h $(DATABASE_HOST) -p $(DATABASE_PORT) -U $(DATABASE_USER) -d $(DATABASE_NAME)

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/cosmtrek/air@latest
	@echo "Tools installed"

## install-lint: Install golangci-lint
install-lint:
	@echo "Installing golangci-lint..."
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin latest
	@echo "golangci-lint installed"

## ci-lint: Run linter (CI)
ci-lint:
	@echo "Running linter..."
	@golangci-lint run --timeout=5m

## ci-test: Run tests for CI environment
ci-test:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

## ci-build: Build all services for CI
ci-build: ci-build-api ci-build-worker
	@echo "All services built for CI"

## ci-build-api: Build API service for CI
ci-build-api:
	@echo "Building API service for CI..."
	@mkdir -p $(BINARY_DIR)
	@go build -v -o ./$(BINARY_DIR)/$(API_BINARY_NAME) ./cmd/api-service/main.go
	@echo "API build complete: $(BINARY_DIR)/$(API_BINARY_NAME)"

## ci-build-worker: Build Worker service for CI
ci-build-worker:
	@echo "Building Worker service for CI..."
	@mkdir -p $(BINARY_DIR)
	@go build -v -o ./$(BINARY_DIR)/$(WORKER_BINARY_NAME) ./cmd/worker-service/main.go
	@echo "Worker build complete: $(BINARY_DIR)/$(WORKER_BINARY_NAME)"

## ci: Run all CI checks locally
ci: ci-lint ci-test ci-build
	@echo ""
	@echo "✅ All CI checks passed!"
	@echo ""
	@echo "Coverage report: coverage.out"
	@go tool cover -func=coverage.out | grep total:

## setup: Initial project setup
setup: install-tools docker-up
	@echo "Waiting for database to be ready..."
	@sleep 5
	@$(MAKE) migrate-up
	@echo "Setup complete! You can now run: make run-api"
