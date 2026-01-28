SHELL := /usr/bin/env bash
APP := structlint
BIN := bin/$(APP)
PKG := github.com/AxeForging/structlint
DIST_DIR := dist

# Cross-compilation targets
GOOS_ARCH := linux/amd64 linux/arm64 linux/386 linux/arm darwin/amd64 darwin/arm64 windows/amd64 windows/arm64 windows/386

# Version information - can be overridden by environment variable
ifeq ($(origin VERSION), environment)
  # VERSION is set from environment
else
  VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
endif

BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILT_BY := $(shell whoami)

# Build flags
LDFLAGS := -s -w \
  -X $(PKG)/internal/build.Version=$(VERSION) \
  -X $(PKG)/internal/build.Commit=$(GIT_COMMIT) \
  -X $(PKG)/internal/build.Date=$(BUILD_TIME) \
  -X $(PKG)/internal/build.BuiltBy=$(BUILT_BY)

.PHONY: all build build-all run test test-self lint tidy clean completion validate-self fmt format check ci version tag release-check

all: build

# Build for current platform
build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/$(APP)
	@echo "Built $(BIN) ($(VERSION))"

# Build for all platforms
build-all:
	@echo "Building binaries for all platforms..."
	@echo "Version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"
	@mkdir -p $(DIST_DIR)
	@for t in $(GOOS_ARCH); do \
		os=$${t%/*}; arch=$${t#*/}; \
		bin_name=$(APP)-$${os}-$${arch}; \
		if [ "$$os" = "windows" ]; then bin_name="$${bin_name}.exe"; fi; \
		bin_path=$(DIST_DIR)/$$bin_name; \
		echo "  Building for $$os/$$arch..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o $$bin_path ./cmd/$(APP); \
	done
	@echo "Build complete. Binaries in $(DIST_DIR)/"

run: build
	@$(BIN) $(ARGS)

test:
	go test ./... -v

test-self: build
	@echo "Validating our own project structure..."
	@$(BIN) validate --config .structlint.yaml --json-output validation-report.json
	@echo "Validation report saved to validation-report.json"

lint:
	golangci-lint run --timeout=5m

tidy:
	go mod tidy

clean:
	rm -rf bin $(DIST_DIR) validation-report.json *.json

completion: build
	@mkdir -p $(DIST_DIR)/completion
	@$(BIN) completion bash > $(DIST_DIR)/completion/$(APP).bash
	@$(BIN) completion zsh  > $(DIST_DIR)/completion/_$(APP)
	@$(BIN) completion fish > $(DIST_DIR)/completion/$(APP).fish
	@echo "Shell completions generated in $(DIST_DIR)/completion/"

validate-self: test-self

# Development helpers
fmt:
	go fmt ./...

format: fmt
	@command -v gofumpt >/dev/null 2>&1 && gofumpt -w . || echo "gofumpt not installed, skipping"
	@command -v goimports >/dev/null 2>&1 && goimports -w . || echo "goimports not installed, skipping"

check: lint test

# CI/CD helpers
ci: tidy check build test-self

# Version information
version:
	@echo "Version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Built by: $(BUILT_BY)"

# Create a git tag for release
tag:
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "Error: Cannot tag dev version. Set VERSION env var (e.g., VERSION=v1.0.0 make tag)"; \
		exit 1; \
	fi
	@echo "Creating git tag: $(VERSION)"
	git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Tag created. Push with: git push origin $(VERSION)"

# Build and test before release
release-check: build-all
	@echo "Running tests..."
	go test ./... -v
	@echo "All tests passed. Ready for release $(VERSION)"
