.PHONY: help dev test build docker-build docker-up docker-down clean

# Default target
help:
	@echo "ToolBridge API - Development Commands"
	@echo ""
	@echo "  make dev          - Start local dev server"
	@echo "  make test         - Run tests"
	@echo "  make build        - Build binary"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-up    - Start Postgres via docker-compose"
	@echo "  make docker-down  - Stop Postgres"
	@echo "  make migrate      - Run migrations against local DB"
	@echo "  make clean        - Clean build artifacts"

# Local development server
dev:
	@echo "Starting dev server..."
	DATABASE_URL=postgres://toolbridge:dev-password@localhost:5432/toolbridge?sslmode=disable \
	JWT_HS256_SECRET=dev-secret \
	ENV=dev \
	go run ./cmd/server

# Run tests
test:
	go test -v -race -cover ./...

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
