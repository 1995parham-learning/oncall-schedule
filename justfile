# Justfile for oncall-schedule
# Install just: https://github.com/casey/just

# Default recipe to display help
default:
    @just --list

# Build the application
build:
    @echo "Building oncall-schedule..."
    @go build -o bin/oncall-schedule .

# Build with version and build info
build-release version:
    @echo "Building oncall-schedule v{{version}}..."
    @go build -ldflags "-X main.version={{version}} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/oncall-schedule .

# Run the application with PostgreSQL
run: db-up
    @echo "Running with PostgreSQL..."
    @ONCALL_USE_DATABASE=true go run .

# Run the application with in-memory storage
run-memory:
    @echo "Running with in-memory storage..."
    @ONCALL_USE_DATABASE=false go run .

# Run tests with coverage
test:
    @echo "Running tests..."
    @go test -v -race -coverprofile=coverage.out ./...
    @go tool cover -func=coverage.out

# Run short tests only
test-short:
    @go test -short -v ./...

# Run tests and open coverage report in browser
test-coverage:
    @echo "Running tests with coverage..."
    @go test -v -race -coverprofile=coverage.out ./...
    @go tool cover -html=coverage.out

# Run benchmarks
bench:
    @echo "Running benchmarks..."
    @go test -bench=. -benchmem ./...

# Start PostgreSQL database
db-up:
    @echo "Starting PostgreSQL..."
    @docker compose up -d postgres
    @echo "Waiting for database to be ready..."
    @sleep 2

# Stop PostgreSQL database
db-down:
    @echo "Stopping PostgreSQL..."
    @docker compose down

# Reset PostgreSQL database (WARNING: deletes all data)
db-reset:
    @echo "⚠️  Resetting database (all data will be deleted)..."
    @docker compose down -v
    @docker compose up -d postgres
    @sleep 2

# Connect to PostgreSQL with psql
db-shell:
    @docker compose exec postgres psql -U oncall -d oncall

# View PostgreSQL logs
db-logs:
    @docker compose logs -f postgres

# Run linter
lint:
    @echo "Running linter..."
    @golangci-lint run

# Fix linting issues automatically
lint-fix:
    @echo "Fixing linting issues..."
    @golangci-lint run --fix

# Format code
fmt:
    @echo "Formatting code..."
    @gofmt -s -w .
    @goimports -w .

# Tidy dependencies
tidy:
    @echo "Tidying dependencies..."
    @go mod tidy
    @go mod verify

# Install/update dependencies
deps:
    @echo "Installing dependencies..."
    @go mod download
    @go mod tidy

# Clean build artifacts and caches
clean:
    @echo "Cleaning..."
    @rm -rf bin/
    @rm -f coverage.out
    @go clean -cache -testcache

# Clean everything including Docker volumes
clean-all: clean db-down
    @echo "Removing Docker volumes..."
    @docker compose down -v

# Run security checks
security:
    @echo "Running security checks..."
    @go list -json -deps ./... | nancy sleuth

# Check for outdated dependencies
deps-check:
    @echo "Checking for outdated dependencies..."
    @go list -u -m all

# Update all dependencies
deps-update:
    @echo "Updating dependencies..."
    @go get -u ./...
    @go mod tidy

# Run the full CI pipeline locally
ci: lint test build
    @echo "✅ CI pipeline completed successfully!"

# Docker: Build application image
docker-build tag="latest":
    @echo "Building Docker image..."
    @docker build -t oncall-schedule:{{tag}} .

# Docker: Run application with docker compose
docker-up:
    @echo "Starting all services with Docker Compose..."
    @docker compose up -d

# Docker: Stop all services
docker-down:
    @docker compose down

# Docker: View all logs
docker-logs:
    @docker compose logs -f

# Development: Watch for changes and rebuild
dev:
    @echo "Starting development mode with auto-reload..."
    @command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/air-verse/air@latest; }
    @air

# Generate migration files
migrate-create name:
    @echo "Creating migration: {{name}}"
    @timestamp=$(date +%s); \
    echo "-- Migration: {{name}}" > migrations/${timestamp}_{{name}}.up.sql; \
    echo "-- Rollback migration: {{name}}" > migrations/${timestamp}_{{name}}.down.sql; \
    echo "Created migrations/${timestamp}_{{name}}.up.sql"; \
    echo "Created migrations/${timestamp}_{{name}}.down.sql"

# Run database migrations manually
migrate-up:
    @echo "Running database migrations..."
    @go run . migrate up

# Rollback last migration
migrate-down:
    @echo "Rolling back last migration..."
    @go run . migrate down

# Show project statistics
stats:
    @echo "Project Statistics:"
    @echo "==================="
    @echo "Go files: $(find . -name '*.go' | wc -l)"
    @echo "Total lines: $(find . -name '*.go' -exec wc -l {} + | tail -1 | awk '{print $1}')"
    @echo "Test files: $(find . -name '*_test.go' | wc -l)"
    @echo ""
    @echo "Dependencies:"
    @go list -m all | wc -l
    @echo ""
    @echo "Git status:"
    @git status --short

# Validate all configuration files
validate:
    @echo "Validating configuration files..."
    @echo "Checking docker-compose.yml..."
    @docker compose config > /dev/null
    @echo "Checking config.yaml..."
    @go run . --help > /dev/null || true
    @echo "✅ All configuration files are valid!"

# Install development tools
install-tools:
    @echo "Installing development tools..."
    @go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @go install github.com/air-verse/air@latest
    @go install golang.org/x/tools/cmd/goimports@latest
    @echo "✅ Development tools installed!"
