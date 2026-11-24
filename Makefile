.PHONY: build test lint doctor seed mcp-smoke

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
