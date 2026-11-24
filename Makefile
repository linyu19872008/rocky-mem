.PHONY: build test lint doctor seed serve mcp-smoke verify-tier0 verify-tier1

BINARY ?= bin/rockymem
DSN ?= sqlite://$(CURDIR)/.local/rocky-mem.db

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/rockymem

test:
	go test ./...

lint:
	go vet ./...

doctor: build
	./$(BINARY) --dsn $(DSN) doctor

seed: build
	./scripts/seed-demo.sh

# Exclusive writer for .local/rocky-mem.db. Start this before Cursor MCP in multi-window use.
serve: build
	./scripts/serve-local.sh

mcp-smoke: build
	go run ./scripts/mcp_smoke.go

# Tier-0 gate: unit tests + protocol/workflow/audit adoption verify.
verify-tier0: build
	go test ./internal/adoption/ ./internal/core/ ./internal/project/ ./internal/textnorm/
	./$(BINARY) adopt-verify --write-verdict --json
	@echo "OK: tier-0 automated gate passed — complete live checklist in docs/adoption-verdict.md"

# Tier-1 gate: serve/lock/API + maintain/review/idempotency + retrieval eval.
verify-tier1: build
	go test ./internal/daemon/ ./internal/api/ ./internal/core/ ./internal/eval/ ./internal/project/ ./internal/textnorm/ ./internal/adoption/
	./$(BINARY) eval --min-recall 0.8
	@echo "OK: tier-1 automated gate passed"
