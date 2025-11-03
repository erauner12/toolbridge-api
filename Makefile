.PHONY: help dev test test-unit test-integration test-smoke build docker-build docker-up docker-down clean

# Default target
help:
	@echo "ToolBridge API - Development Commands"
	@echo ""
	@echo "  make dev              - Start local dev server"
	@echo "  make test             - Run all tests (unit + integration)"
	@echo "  make test-unit        - Run unit tests only (fast, no DB)"
	@echo "  make test-integration - Run integration tests (requires DB)"
	@echo "  make test-smoke       - Run smoke tests against running server"
	@echo "  make build            - Build binary"
	@echo "  make docker-build     - Build Docker image"
	@echo "  make docker-up        - Start Postgres via docker-compose"
	@echo "  make docker-down      - Stop Postgres"
	@echo "  make migrate          - Run migrations against local DB"
	@echo "  make clean            - Clean build artifacts"

# Local development server
dev:
	@echo "Starting dev server..."
	DATABASE_URL=postgres://toolbridge:dev-password@localhost:5432/toolbridge?sslmode=disable \
	JWT_HS256_SECRET=dev-secret \
	ENV=dev \
	go run ./cmd/server

# Run all tests (unit + integration)
test:
	@echo "Running all tests..."
	TEST_DATABASE_URL=postgres://toolbridge:dev-password@localhost:5432/toolbridge?sslmode=disable \
	go test -v -race -cover ./...

# Run unit tests only (no database required)
test-unit:
	@echo "Running unit tests..."
	go test -v -short -race -cover ./...

# Run integration tests (requires database)
test-integration:
	@echo "Running integration tests..."
	TEST_DATABASE_URL=postgres://toolbridge:dev-password@localhost:5432/toolbridge?sslmode=disable \
	go test -v -race -cover ./internal/httpapi/...

# Run smoke tests against running server
test-smoke:
	@echo "Running smoke tests..."
	@./scripts/smoke-test.sh

# Build binary
build:
	@echo "Building server..."
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

# Build Docker image
docker-build:
	docker build -t toolbridge-api:latest .

# Start Postgres with docker-compose
docker-up:
	docker-compose up -d postgres
	@echo "Waiting for Postgres to be ready..."
	@sleep 3
	@echo "Postgres is ready at localhost:5432"

# Stop docker-compose services
docker-down:
	docker-compose down

# Run migrations manually (requires psql)
migrate:
	@echo "Running migrations..."
	PGPASSWORD=dev-password psql -h localhost -U toolbridge -d toolbridge -f migrations/0001_init.sql

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod download
	go mod tidy
