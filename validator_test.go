package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AxeForging/structlint/internal/config"
	"github.com/AxeForging/structlint/internal/logging"
	"github.com/AxeForging/structlint/internal/validator"
)

func TestValidator_Complex(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-test-complex")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// 1. Create a more complex directory structure
	dirs := []string{
		"src/api/v1",
		"src/internal/pkg",
		"vendor/github.com/some/lib",
		"docs/swagger",
		"scripts",
		"tmp", // Disallowed directory
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	files := map[string]string{
		"src/main.go":                       "package main",
		"src/api/v1/auth.go":                "package v1",
		"src/internal/pkg/db.go":            "package pkg",
		"docs/swagger/api.yaml":             "openapi: 3.0.0",
		"scripts/build.sh":                  "#!/bin/bash",
		".env.local":                        "SECRET=123",  // Disallowed file
		"README.md":                         "# Project",   // Not in allowed list
		"vendor/github.com/some/lib/lib.go": "package lib", // Should be ignored
	}
	for path, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, path), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// 2. Create a more complex config file
	configContent := `dir_structure:
  allowedPaths:
    - "."
    - "src/**"
    - "docs/swagger"
    - "scripts"
  disallowedPaths:
    - "tmp"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.yaml"
    - "*.sh"
  disallowed:
    - "*.env*"
ignore:
  - "vendor"
  - "**/.git"
`
	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 3. Load config and run validator
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger, _ := logging.New("error", true) // silent logger for tests
	v := validator.New(cfg, logger)
	v.Silent = true // Test silent mode
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	v.ValidateDirStructure(".")
	v.ValidateFileNaming(".")

	// 4. Assert specific errors
	expectedErrors := map[string]bool{
		"Disallowed directory found: tmp":                  true,
		"Disallowed file naming pattern found: .env.local": true,
		"File not in allowed naming pattern: README.md":    true,
	}

	if len(v.Errors) != len(expectedErrors) {
		t.Errorf("Expected %d errors, but got %d. Errors: %v", len(expectedErrors), len(v.Errors), v.Errors)
	}

	for _, e := range v.Errors {
		// Normalize path separators for Windows compatibility
		normalizedError := strings.ReplaceAll(e, "\\", "/")
		if _, ok := expectedErrors[normalizedError]; !ok {
			t.Errorf("Unexpected error found: %s", normalizedError)
		}
	}

	// 5. Test JSON report
	jsonPath := filepath.Join(tmpDir, "report.json")
	if err := v.SaveJSONReport(jsonPath); err != nil {
		t.Fatalf("Failed to save JSON report: %v", err)
	}

	reportData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON report: %v", err)
	}

	var report validator.JSONReport
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Fatalf("Failed to unmarshal JSON report: %v", err)
	}

	if report.Failures != len(expectedErrors) {
		t.Errorf("Expected %d failures in JSON report, but got %d", len(expectedErrors), report.Failures)
	}

	// A simple check for the number of successes. A more robust test could check the exact number.
	if report.Successes <= 0 {
		t.Errorf("Expected a positive number of successes, but got %d", report.Successes)
	}
}
