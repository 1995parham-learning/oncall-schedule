.PHONY: help build run run-memory test db-up db-down db-reset lint clean

help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building oncall-schedule..."
	@go build -o bin/oncall-schedule .

run: db-up ## Run the application with PostgreSQL
	@echo "Running with PostgreSQL..."
	@ONCALL_USE_DATABASE=true go run .

run-memory: ## Run the application with in-memory storage
	@echo "Running with in-memory storage..."
	@ONCALL_USE_DATABASE=false go run .

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

test-short: ## Run short tests
	@go test -short -v ./...

db-up: ## Start PostgreSQL database
	@echo "Starting PostgreSQL..."
	@docker-compose up -d postgres
	@echo "Waiting for database to be ready..."
	@sleep 2

db-down: ## Stop PostgreSQL database
	@echo "Stopping PostgreSQL..."
	@docker-compose down

db-reset: ## Reset PostgreSQL database (WARNING: deletes all data)
	@echo "Resetting database..."
	@docker-compose down -v
	@docker-compose up -d postgres
	@sleep 2

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out

deps: ## Install/update dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

.DEFAULT_GOAL := help
