.PHONY: build run help build-frontend dev test clean

build: build-frontend
	rm -rf internal/api/static/*
	cp -R frontend/dist/* internal/api/static/
	go build -o mnemosyne ./cmd/mnemosyne

run: build
	./mnemosyne

help:
	@echo "Available commands:"
	@echo "  make              - Build the mnemosyne binary (includes frontend)"
	@echo "  make run          - Build (if needed) and run the binary"
	@echo "  make build-frontend - Build only the frontend"
	@echo "  make dev          - Run frontend dev server + Go backend"
	@echo "  make test         - Run all tests (Go + frontend)"
	@echo "  make clean        - Clean build artifacts"

build-frontend:
	cd frontend && npm install && npm run build

dev:
	@echo "Starting dev servers..."
	@make -j2 dev-backend dev-frontend

dev-backend:
	go run ./cmd/mnemosyne

dev-frontend:
	cd frontend && npm run dev

test:
	go test ./internal/... -count=1
	cd frontend && npx vitest run

clean:
	rm -rf mnemosyne frontend/dist frontend/node_modules
	rm -rf internal/api/static/*
	echo '<!doctype html><html><body><p>Frontend not built.</p></body></html>' > internal/api/static/index.html
