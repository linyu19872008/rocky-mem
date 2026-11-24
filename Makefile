.PHONY: build test lint doctor seed serve backup mcp-smoke verify-tier0 verify-tier1 verify-tier2 verify-tier3

BINARY ?= bin/rockymem
DSN ?= sqlite://$(CURDIR)/.local/rocky-mem.db
BACKUP_DIR ?= $(CURDIR)/.local/backups
BACKUP_RETAIN ?= 14

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

# Snapshot store (safe while serve is up). Override BACKUP_RETAIN / BACKUP_DIR as needed.
backup: build
	./$(BINARY) --dsn $(DSN) backup --dir $(BACKUP_DIR) --retain $(BACKUP_RETAIN)

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

# Tier-2 gate: backup + term-map + short-query + packages above.
verify-tier2: build
	go test ./internal/daemon/ ./internal/api/ ./internal/core/ ./internal/eval/ ./internal/termmap/ ./internal/project/ ./internal/textnorm/ ./internal/adoption/ ./internal/store/sqlite/
	./$(BINARY) eval --min-recall 0.8
	@echo "OK: tier-2 automated gate passed"

# Tier-3 gate: store conformance + web/cookie auth + prior gates.
verify-tier3: build
	go test ./internal/store/conformance/ ./internal/store/sqlite/ ./internal/api/ ./internal/core/ ./internal/eval/ ./internal/termmap/ ./internal/daemon/ ./internal/adoption/
	./$(BINARY) eval --min-recall 0.8
	@echo "OK: tier-3 automated gate passed"
