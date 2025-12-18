.PHONY: help build run-api test test-unit test-coverage test-verbose test-config test-logger test-clean clean migrate-up migrate-down migrate-create docker-up docker-down dev

# Load environment variables from .env file
include .env
export

# Variables
APP_NAME=job-api-service
BINARY_DIR=bin
BINARY_NAME=api-service
DOCKER_COMPOSE=docker compose -f docker/docker-compose.yml
MIGRATIONS_DIR=migrations
DATABASE_URL=postgresql://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=$(DATABASE_SSLMODE)

## help: Display this help message
help:
	@echo "Available commands:"
	@echo "  make build         - Build the API service binary"
	@echo "  make run-api       - Run the API service"
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

## build: Build the API service binary
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BINARY_DIR)
	@go build -o ./$(BINARY_DIR)/$(BINARY_NAME) ./cmd/api-service/main.go
	@echo "Build complete: $(BINARY_DIR)/$(BINARY_NAME)"

## run-api: Run the API service
run-api:
	@echo "Starting $(APP_NAME)..."
	@go run cmd/api-service/main.go

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

## setup: Initial project setup
setup: install-tools docker-up
	@echo "Waiting for database to be ready..."
	@sleep 5
	@$(MAKE) migrate-up
	@echo "Setup complete! You can now run: make run-api"
