package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AxeForging/structlint/internal/config"
	"github.com/AxeForging/structlint/internal/logging"
	"github.com/AxeForging/structlint/internal/validator"
)

// TestProjectStructureValidation tests our own project structure against defined standards
func TestProjectStructureValidation(t *testing.T) {
	// Define our project structure standards
	configContent := `dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "test/**"
    - "docs/**"
    - "scripts/**"
    - "bin/**"
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
    - "temp/**"
    - ".git/**"
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
    - ".golangci.yml"
    - ".goreleaser.yaml"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
    - "*.temp"
    - "*~"
    - "*.swp"
    - "*.swo"
ignore:
  - ".git"
  - "vendor"
  - "node_modules"
  - "bin"
  - "dist"
  - "*.log"
  - "*.tmp"
`

	// Create temporary config file
	tmpDir, err := os.MkdirTemp("", "structlint-project-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create validator
	logger, _ := logging.New("error", true)
	v := validator.New(cfg, logger)
	v.Silent = true

	// Validate our current project structure
	v.ValidateDirStructure(".")
	v.ValidateFileNaming(".")

	// Check for any validation errors
	if len(v.Errors) > 0 {
		t.Errorf("Project structure validation failed with %d errors:", len(v.Errors))
		for _, err := range v.Errors {
			t.Errorf("  - %s", err)
		}
	}

	// Ensure we have some successes
	if v.Successes == 0 {
		t.Error("Expected at least some successful validations")
	}

	t.Logf("Project validation: %d successes, %d failures", v.Successes, len(v.Errors))
}

// TestGoProjectStandards tests a comprehensive Go project structure
func TestGoProjectStandards(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-go-project")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a comprehensive Go project structure
	projectStructure := map[string]string{
		// Root files
		"go.mod":           "module test-project\ngo 1.24",
		"go.sum":           "// dependencies",
		"README.md":        "# Test Project",
		"LICENSE":          "MIT License",
		"Makefile":         "build: go build",
		"Dockerfile":       "FROM golang:1.24",
		".gitignore":       "bin/\n*.log",
		".editorconfig":    "root = true",
		".golangci.yml":    "linters:\n  enable:\n    - govet",
		".goreleaser.yaml": "version: 2",

		// Command structure
		"cmd/app/main.go": "package main\nfunc main() {}",

		// Internal packages
		"internal/app/app.go":         "package app",
		"internal/config/config.go":   "package config",
		"internal/handler/handler.go": "package handler",

		// API structure
		"api/v1/users.go": "package v1",
		"api/v2/users.go": "package v2",

		// Tests
		"test/integration_test.go": "package test",
		"internal/app/app_test.go": "package app",

		// Documentation
		"docs/api.md":    "API Documentation",
		"docs/deploy.md": "Deployment Guide",

		// Scripts
		"scripts/build.sh":  "#!/bin/bash\necho 'Building...'",
		"scripts/deploy.sh": "#!/bin/bash\necho 'Deploying...'",

		// Configuration
		"config/app.yaml": "app:\n  name: test",
		"config/app.json": `{"app": {"name": "test"}}`,
	}

	// Create the project structure
	for path, content := range projectStructure {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Add some disallowed files to test validation
	disallowedFiles := map[string]string{
		".env.local":    "SECRET=123",
		"debug.log":     "debug information",
		"temp.tmp":      "temporary file",
		"backup~":       "backup file",
		"vendor/lib.go": "vendor code", // Should be ignored
	}

	for path, content := range disallowedFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Create comprehensive configuration
	configContent := `dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "api/**"
    - "test/**"
    - "docs/**"
    - "scripts/**"
    - "config/**"
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
    - ".golangci.yml"
    - ".goreleaser.yaml"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
    - "*~"
ignore:
  - "vendor"
  - ".git"
`

	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load and run validation
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger, _ := logging.New("error", true)
	v := validator.New(cfg, logger)
	v.Silent = true

	// Change to test directory
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	v.ValidateDirStructure(".")
	v.ValidateFileNaming(".")

	// Expected errors for disallowed files
	expectedErrors := []string{
		"Disallowed file naming pattern found: .env.local",
		"Disallowed file naming pattern found: debug.log",
		"Disallowed file naming pattern found: temp.tmp",
		"Disallowed file naming pattern found: backup~",
	}

	// Check that we got the expected errors
	foundErrors := make(map[string]bool)
	for _, err := range v.Errors {
		foundErrors[err] = true
	}

	for _, expectedErr := range expectedErrors {
		if !foundErrors[expectedErr] {
			t.Errorf("Expected error not found: %s", expectedErr)
		}
	}

	// Check that vendor files are ignored (should not appear in errors)
	for _, err := range v.Errors {
		if strings.Contains(err, "vendor") {
			t.Errorf("Vendor files should be ignored but found error: %s", err)
		}
	}

	t.Logf("Go project validation: %d successes, %d failures", v.Successes, len(v.Errors))
}

// TestMicroserviceStructure tests a microservice project structure
func TestMicroserviceStructure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-microservice")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create microservice structure
	structure := map[string]string{
		"cmd/api/main.go":                 "package main",
		"cmd/worker/main.go":              "package main",
		"internal/api/handler.go":         "package api",
		"internal/service/user.go":        "package service",
		"internal/repository/db.go":       "package repository",
		"internal/middleware/auth.go":     "package middleware",
		"pkg/utils/logger.go":             "package utils",
		"api/openapi.yaml":                "openapi: 3.0.0",
		"deployments/docker-compose.yml":  "version: '3'",
		"deployments/k8s/deployment.yaml": "apiVersion: apps/v1",
		"scripts/migrate.sh":              "#!/bin/bash",
		"scripts/test.sh":                 "#!/bin/bash",
		"docs/README.md":                  "# API Documentation",
		"docs/ARCHITECTURE.md":            "# Architecture",
		"tests/integration/api_test.go":   "package tests",
		"tests/unit/service_test.go":      "package tests",
	}

	for path, content := range structure {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Microservice-specific configuration
	configContent := `dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
    - "api/**"
    - "deployments/**"
    - "scripts/**"
    - "docs/**"
    - "tests/**"
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.yaml"
    - "*.yml"
    - "*.md"
    - "*.sh"
    - "*.mod"
    - "*.sum"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
ignore:
  - "vendor"
  - ".git"
`

	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger, _ := logging.New("error", true)
	v := validator.New(cfg, logger)
	v.Silent = true

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	v.ValidateDirStructure(".")
	v.ValidateFileNaming(".")

	// Should have no errors for a well-structured microservice
	if len(v.Errors) > 0 {
		t.Errorf("Microservice structure validation failed with %d errors:", len(v.Errors))
		for _, err := range v.Errors {
			t.Errorf("  - %s", err)
		}
	}

	t.Logf("Microservice validation: %d successes, %d failures", v.Successes, len(v.Errors))
}

// TestConfigurationFormats tests different configuration file formats
func TestConfigurationFormats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-config-formats")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Test YAML configuration
	yamlConfig := `dir_structure:
  allowedPaths: ["cmd/**", "internal/**"]
  disallowedPaths: ["vendor/**"]
file_naming_pattern:
  allowed: ["*.go", "*.yaml"]
  disallowed: ["*.env*"]
ignore: ["vendor", ".git"]
`

	yamlPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlConfig), 0o644); err != nil {
		t.Fatalf("Failed to write YAML config: %v", err)
	}

	cfg, err := config.LoadConfig(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	if len(cfg.DirStructure.AllowedPaths) != 2 {
		t.Errorf("Expected 2 allowed paths, got %d", len(cfg.DirStructure.AllowedPaths))
	}

	// Test JSON configuration
	jsonConfig := `{
  "dir_structure": {
    "allowedPaths": ["cmd/**", "internal/**"],
    "disallowedPaths": ["vendor/**"]
  },
  "file_naming_pattern": {
    "allowed": ["*.go", "*.json"],
    "disallowed": ["*.env*"]
  },
  "ignore": ["vendor", ".git"]
}`

	jsonPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(jsonPath, []byte(jsonConfig), 0o644); err != nil {
		t.Fatalf("Failed to write JSON config: %v", err)
	}

	cfg2, err := config.LoadConfig(jsonPath)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	if len(cfg2.DirStructure.AllowedPaths) != 2 {
		t.Errorf("Expected 2 allowed paths, got %d", len(cfg2.DirStructure.AllowedPaths))
	}

	t.Log("Configuration format tests passed")
}
