# apps/server Makefile — command entrypoints per AGENTS.md.
# All go tooling is version-pinned in go.mod via the `tool` directive.

MIGRATE_URL ?= postgres://go_shell:go_shell@localhost:5432/go_shell?sslmode=disable
MIGRATIONS_DIR := db/migrations
BIN := bin/server

.PHONY: dev build test lint typecheck gen migrate-up migrate-down clean

dev: ## Run the API server locally (SIGTERM/SIGINT drains gracefully).
	go run ./cmd/server

build: ## Compile all packages and link the server binary.
	go build ./...
	go build -o $(BIN) ./cmd/server

test: ## Run all unit tests.
	go test ./...

lint: ## Static checks: gofmt drift + go vet.
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi
	go vet ./...

typecheck: ## Compilation is Go's typecheck.
	go build ./...

gen: ## Regenerate code (buf pinned; proto sources arrive in Phase 1). Must stay idempotent.
	@go tool buf --version
	@echo "gen: nothing to generate in Phase 0"

migrate-up: ## Apply all pending migrations (no-op until Phase 2).
	@if ls $(MIGRATIONS_DIR)/*.up.sql >/dev/null 2>&1; then \
		go tool migrate -path $(MIGRATIONS_DIR) -database "$(MIGRATE_URL)" up; \
	else \
		echo "migrate-up: no migrations yet (first migration lands in Phase 2)"; \
	fi

migrate-down: ## Roll back the most recent migration (no-op until Phase 2).
	@if ls $(MIGRATIONS_DIR)/*.up.sql >/dev/null 2>&1; then \
		go tool migrate -path $(MIGRATIONS_DIR) -database "$(MIGRATE_URL)" down 1; \
	else \
		echo "migrate-down: no migrations yet (first migration lands in Phase 2)"; \
	fi

clean: ## Remove build artifacts.
	rm -rf bin
