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
	@go test ./internal/... -count=1 -v > /tmp/mnemosyne-test-go.log 2>&1; \
	go_exit=$$?; \
	cd frontend && npx vitest run > /tmp/mnemosyne-test-frontend.log 2>&1; \
	fe_exit=$$?; \
	go_pass=$$(grep -c "^--- PASS:" /tmp/mnemosyne-test-go.log || true); \
	go_fail=$$(grep -c "^--- FAIL:" /tmp/mnemosyne-test-go.log || true); \
	fe_pass=$$(grep "Tests" /tmp/mnemosyne-test-frontend.log | grep -o "[0-9]* passed" | grep -o "[0-9]*" || echo 0); \
	fe_fail=$$(grep "Tests" /tmp/mnemosyne-test-frontend.log | grep -o "[0-9]* failed" | grep -o "[0-9]*" || echo 0); \
	total_pass=$$((go_pass + fe_pass)); \
	total_fail=$$((go_fail + fe_fail)); \
	total=$$((total_pass + total_fail)); \
	if [ $$total_fail -gt 0 ]; then \
		echo "\033[31m$$total_pass/$$total tests passed ($$total_fail failed)\033[0m"; \
		echo ""; \
		if [ $$go_fail -gt 0 ]; then \
			echo "\033[1mGo failures:\033[0m"; \
			awk '/^--- FAIL:/,/^(--- |=== |FAIL|ok )/' /tmp/mnemosyne-test-go.log; \
		fi; \
		if [ $$fe_fail -gt 0 ]; then \
			echo "\033[1mFrontend failures:\033[0m"; \
			cat /tmp/mnemosyne-test-frontend.log; \
		fi; \
		echo ""; \
		echo "Full logs: /tmp/mnemosyne-test-go.log, /tmp/mnemosyne-test-frontend.log"; \
		exit 1; \
	else \
		echo "\033[32m$$total_pass/$$total tests passed\033[0m"; \
	fi

clean:
	rm -rf mnemosyne frontend/dist frontend/node_modules
	rm -rf internal/api/static/*
	echo '<!doctype html><html><body><p>Frontend not built.</p></body></html>' > internal/api/static/index.html
