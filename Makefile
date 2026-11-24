.PHONY: build test lint doctor seed mcp-smoke verify-tier0

BINARY ?= bin/rockymem

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/rockymem

test:
	go test ./...

lint:
	go vet ./...

doctor: build
	./$(BINARY) doctor

seed: build
	./scripts/seed-demo.sh

mcp-smoke: build
	go run ./scripts/mcp_smoke.go

# Tier-0 gate: unit tests + protocol/workflow/audit adoption verify.
verify-tier0: build
	go test ./internal/adoption/ ./internal/core/ ./internal/project/ ./internal/textnorm/
	./$(BINARY) adopt-verify --write-verdict --json
	@echo "OK: tier-0 automated gate passed — complete live checklist in docs/adoption-verdict.md"
