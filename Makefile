.PHONY: help dev-setup dev-backend dev-frontend dev build test clean

help:
	@echo "Available commands:"
	@echo "  make dev-setup    - Install dependencies"
	@echo "  make dev          - Run both backend and frontend in development mode"
	@echo "  make dev-backend  - Run backend server"
	@echo "  make dev-frontend - Run frontend development server"
	@echo "  make build        - Build both backend and frontend for production"
	@echo "  make test         - Run all tests"
	@echo "  make clean        - Clean build artifacts"

dev-setup:
	@echo "Setting up development environment..."
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