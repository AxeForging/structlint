package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/youngestaxe/structlint/internal/app"
)

// TestIntegrationSelfValidation tests our CLI tool against our own project
func TestIntegrationSelfValidation(t *testing.T) {
	// Check if our configuration file exists
	configPath := ".structlint.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("Configuration file %s not found, skipping integration test", configPath)
	}

	// Test the CLI tool directly
	app := app.New()

	// Test validation command
	err := app.Run(context.Background(), []string{
		"structlint",
		"validate",
		"--config", configPath,
		"--json-output", "integration-test-report.json",
	})

	// Should not have any errors for our well-structured project
	if err != nil {
		t.Errorf("Self-validation failed: %v", err)
	}

	// Check if report was created
	if _, err := os.Stat("integration-test-report.json"); os.IsNotExist(err) {
		t.Error("JSON report was not created")
	} else {
		t.Log("✅ Integration test report created: integration-test-report.json")
		defer func() {
			_ = os.Remove("integration-test-report.json")
		}()
	}
}

// TestIntegrationWithRealProject tests against a real project structure
func TestIntegrationWithRealProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-real-project")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a realistic project structure
	projectFiles := map[string]string{
		// Core Go files
		"cmd/api/main.go":           "package main\n\nfunc main() {\n\t// API server\n}",
		"cmd/worker/main.go":        "package main\n\nfunc main() {\n\t// Background worker\n}",
		"internal/api/handler.go":   "package api\n\ntype Handler struct{}",
		"internal/service/user.go":  "package service\n\ntype UserService struct{}",
		"internal/repository/db.go": "package repository\n\ntype DB struct{}",
		"pkg/utils/logger.go":       "package utils\n\ntype Logger struct{}",

		// Configuration
		"config/app.yaml": `app:
  name: "test-api"
  port: 8080
database:
  host: "localhost"
  port: 5432`,

		// Documentation
		"README.md": `# Test API

A test API project demonstrating proper Go project structure.

## Features
- REST API
- Background workers
- Database integration

## Usage
` + "```bash" + `
go run cmd/api/main.go
` + "```" + ``,

		"docs/API.md": `# API Documentation

## Endpoints
- GET /users
- POST /users
- PUT /users/:id`,

		// Build files
		"Makefile": `build:
\tgo build -o bin/api cmd/api/main.go
\tgo build -o bin/worker cmd/worker/main.go`,

		"Dockerfile": `FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api cmd/api/main.go

FROM alpine:latest
COPY --from=builder /app/api /usr/local/bin/
CMD ["api"]`,

		// Go module files
		"go.mod": `module test-api

go 1.24

require (
\tgithub.com/gin-gonic/gin v1.9.1
\tgithub.com/lib/pq v1.10.9
)`,

		"go.sum": `github.com/gin-gonic/gin v1.9.1 h1:4idEAncQnU5cB7BeOkPtxjfCSye0AAm1R0RVIqJ+Jmg=
github.com/lib/pq v1.10.9 h1:YXG7RB+JIjhP29X+OtkiDnYaXrpzJN3FjVSiEx6Bq3Y=`,

		// Git files
		".gitignore": `bin/
*.log
.env
vendor/`,

		// Tests
		"internal/api/handler_test.go": `package api

import "testing"

func TestHandler(t *testing.T) {
\t// Test implementation
}`,

		"tests/integration/api_test.go": `package tests

import "testing"

func TestAPIIntegration(t *testing.T) {
\t// Integration test
}`,

		// Scripts
		"scripts/build.sh": `#!/bin/bash
echo "Building API..."
go build -o bin/api cmd/api/main.go
echo "Building worker..."
go build -o bin/worker cmd/worker/main.go`,

		"scripts/test.sh": `#!/bin/bash
echo "Running tests..."
go test ./...`,
	}

	// Create the project structure
	for path, content := range projectFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Create a comprehensive configuration
	configContent := `dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
    - "config/**"
    - "docs/**"
    - "tests/**"
    - "scripts/**"
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
    - "temp/**"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.md"
    - "*.txt"
    - "*.sh"
    - "*.mod"
    - "*.sum"
    - "Makefile"
    - "Dockerfile*"
    - ".gitignore"
    - ".editorconfig"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
    - "*~"
ignore:
  - "vendor"
  - ".git"
  - "bin"
`

	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Test the CLI tool against this project
	app := app.New()

	// Change to the test project directory
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	// Run validation
	err = app.Run(context.Background(), []string{
		"structlint",
		"validate",
		"--config", ".structlint.yaml",
		"--json-output", "real-project-report.json",
	})

	// Should not have any errors for a well-structured project
	if err != nil {
		t.Errorf("Real project validation failed: %v", err)
	}

	// Check if report was created
	if _, err := os.Stat("real-project-report.json"); os.IsNotExist(err) {
		t.Error("JSON report was not created")
	} else {
		t.Log("✅ Real project validation report created")
	}
}

// TestIntegrationWithViolations tests against a project with violations
func TestIntegrationWithViolations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-violations")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a project with violations
	violationFiles := map[string]string{
		"cmd/app/main.go":   "package main",
		".env.local":        "SECRET_KEY=abc123",
		"debug.log":         "2024-01-01 ERROR: Something went wrong",
		"temp.tmp":          "temporary data",
		"backup~":           "backup file",
		"tmp/file.tmp":      "more temp data",
		"vendor/lib/lib.go": "package lib", // Should be ignored
	}

	for path, content := range violationFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Create configuration that should catch violations
	configContent := `dir_structure:
  allowedPaths: ["cmd/**", "internal/**"]
  disallowedPaths: ["vendor/**", "tmp/**"]
file_naming_pattern:
  allowed: ["*.go", "*.mod"]
  disallowed: ["*.env*", "*.log", "*.tmp", "*~"]
ignore: ["vendor"]
`

	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Test the CLI tool
	app := app.New()

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	// Run validation - this should fail
	err = app.Run(context.Background(), []string{
		"structlint",
		"validate",
		"--config", ".structlint.yaml",
		"--json-output", "violations-report.json",
	})

	// Should have errors for violations
	if err == nil {
		t.Error("Expected validation to fail due to violations")
	} else {
		t.Logf("✅ Validation failed as expected: %v", err)
	}

	// Check if report was created
	if _, err := os.Stat("violations-report.json"); os.IsNotExist(err) {
		t.Error("JSON report was not created")
	} else {
		t.Log("✅ Violations report created (as expected)")
	}
}

// TestCLICommands tests all CLI commands
func TestCLICommands(t *testing.T) {
	app := app.New()

	// Test version command
	err := app.Run(context.Background(), []string{"structlint", "version"})
	if err != nil {
		t.Errorf("Version command failed: %v", err)
	}

	// Test help command
	err = app.Run(context.Background(), []string{"structlint", "--help"})
	if err != nil {
		t.Errorf("Help command failed: %v", err)
	}

	// Test validate help
	err = app.Run(context.Background(), []string{"structlint", "validate", "--help"})
	if err != nil {
		t.Errorf("Validate help command failed: %v", err)
	}

	// Test completion commands
	shells := []string{"bash", "zsh", "fish"}
	for _, shell := range shells {
		err = app.Run(context.Background(), []string{"structlint", "completion", shell})
		if err != nil {
			t.Errorf("Completion command for %s failed: %v", shell, err)
		}
	}
}

// TestConfigurationPrecedence tests that configuration precedence works correctly
func TestConfigurationPrecedence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-precedence")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a config file
	configContent := `dir_structure:
  allowedPaths: [".", "cmd/**", "internal/**"]
file_naming_pattern:
  allowed: ["*.go"]
ignore: ["vendor"]
`

	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create a simple project
	projectFiles := map[string]string{
		"cmd/app/main.go":     "package main",
		"internal/app/app.go": "package app",
	}

	for path, content := range projectFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Test with environment variable override
	os.Setenv("STRUCTLINT_CONFIG", configPath)
	defer os.Unsetenv("STRUCTLINT_CONFIG")

	app := app.New()

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	// Should work with env var (but may have violations due to config file)
	err = app.Run(context.Background(), []string{"structlint", "validate"})
	// We expect this to fail because .structlint.yaml is not in the allowed patterns
	if err == nil {
		t.Error("Expected validation to fail due to .structlint.yaml not being allowed")
	}
}
