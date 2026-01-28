SHELL := /usr/bin/env bash
APP := structlint
BIN := bin/$(APP)
PKG := github.com/youngestaxe/structlint
LDFLAGS := -s -w \
  -X $(PKG)/internal/build.Version=$$(git describe --tags --always --dirty) \
  -X $(PKG)/internal/build.Commit=$$(git rev-parse --short HEAD) \
  -X $(PKG)/internal/build.Date=$$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X $(PKG)/internal/build.BuiltBy=$$(whoami)

.PHONY: all build run test test-self lint tidy clean completion validate-self

all: build

build:
	@mkdir -p bin
	GOFLAGS="-trimpath" CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/$(APP)

run: build
	@$(BIN) $(ARGS)

test:
	go test ./...

test-self: build
	@echo "🔍 Validating our own project structure..."
	@$(BIN) validate --config .structlint.yaml --json-output validation-report.json
	@echo "📊 Validation report saved to validation-report.json"

lint:
	golangci-lint run --timeout=5m

tidy:
	go mod tidy

clean:
	rm -rf bin validation-report.json

completion:
	@mkdir -p dist/completion
	@$(BIN) completion bash       > dist/completion/$(APP).bash
	@$(BIN) completion zsh        > dist/completion/_$(APP)
	@$(BIN) completion fish       > dist/completion/$(APP).fish

validate-self: test-self

# Development helpers
fmt:
	go fmt ./...
	goimports -w .

format: fmt
	gofumpt -w .

check: lint test

# CI/CD helpers
ci: tidy check build test-self
