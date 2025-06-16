.PHONY: help dev-setup dev-backend dev-frontend dev db-up db-down db-migrate build test clean

help:
	@echo "Available commands:"
	@echo "  make dev-setup    - Install dependencies and set up development environment"
	@echo "  make dev          - Run both backend and frontend in development mode"
	@echo "  make dev-backend  - Run backend server"
	@echo "  make dev-frontend - Run frontend development server"
	@echo "  make db-up        - Start PostgreSQL and Redis containers"
	@echo "  make db-down      - Stop database containers"
	@echo "  make db-migrate   - Run database migrations"
	@echo "  make build        - Build both backend and frontend for production"
	@echo "  make test         - Run all tests"
	@echo "  make clean        - Clean build artifacts"

dev-setup: db-up
	@echo "Setting up development environment..."
	@if [ ! -f .env ]; then cp .env.example .env; fi
	cd backend && go mod download
	cd frontend && npm install
	@echo "Setup complete! Run 'make dev' to start development servers"

dev:
	@echo "Starting development servers..."
	@make -j2 dev-backend dev-frontend

dev-backend:
	cd backend && go run cmd/server/main.go

dev-frontend:
	cd frontend && npm run dev

db-up:
	docker-compose up -d postgres redis
	@echo "Waiting for databases to be ready..."
	@sleep 5

db-down:
	docker-compose down

db-migrate:
	@echo "Running database migrations..."
	migrate -path database/migrations -database "postgresql://mnemosyne:mnemosyne@localhost:5432/mnemosyne?sslmode=disable" up

build:
	@echo "Building backend..."
	cd backend && go build -o ../dist/server cmd/server/main.go
	@echo "Building frontend..."
	cd frontend && npm run build

test:
	@echo "Running backend tests..."
	cd backend && go test ./...
	@echo "Running frontend tests..."
	cd frontend && npm test

clean:
	rm -rf dist/
	rm -rf frontend/.svelte-kit/
	rm -rf frontend/build/