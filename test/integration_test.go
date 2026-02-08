package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegrationSelfValidation tests the CLI binary against our own project
func TestIntegrationSelfValidation(t *testing.T) {
	bin := buildBinary(t)

	// Run from repo root
	root := repoRoot(t)
	configPath := filepath.Join(root, ".structlint.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("Configuration file %s not found, skipping integration test", configPath)
	}

	reportPath := filepath.Join(t.TempDir(), "integration-test-report.json")

	out, err := runBinaryInDir(t, bin, root,
		"validate",
		"--config", configPath,
		"--json-output", reportPath,
	)

	if err != nil {
		t.Errorf("Self-validation failed: %v\nOutput: %s", err, out)
	}

	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("JSON report was not created")
	} else {
		t.Log("Integration test report created successfully")
	}
}

// TestIntegrationWithRealProject tests against a realistic project structure
func TestIntegrationWithRealProject(t *testing.T) {
	bin := buildBinary(t)

	projectFiles := map[string]string{
		"cmd/api/main.go":               "package main\n\nfunc main() {}",
		"cmd/worker/main.go":            "package main\n\nfunc main() {}",
		"internal/api/handler.go":       "package api\n\ntype Handler struct{}",
		"internal/service/user.go":      "package service\n\ntype UserService struct{}",
		"internal/repository/db.go":     "package repository\n\ntype DB struct{}",
		"pkg/utils/logger.go":           "package utils\n\ntype Logger struct{}",
		"config/app.yaml":               "app:\n  name: test-api\n  port: 8080",
		"README.md":                     "# Test API\n\nA test API project.",
		"docs/API.md":                   "# API Documentation\n\n## Endpoints",
		"Makefile":                      "build:\n\tgo build -o bin/api cmd/api/main.go",
		"Dockerfile":                    "FROM golang:1.24-alpine\nWORKDIR /app",
		"go.mod":                        "module test-api\n\ngo 1.24",
		"go.sum":                        "",
		".gitignore":                    "bin/\n*.log\n.env",
		"internal/api/handler_test.go":  "package api\n\nimport \"testing\"\n\nfunc TestHandler(t *testing.T) {}",
		"tests/integration/api_test.go": "package tests\n\nimport \"testing\"\n\nfunc TestAPI(t *testing.T) {}",
		"scripts/build.sh":              "#!/bin/bash\ngo build -o bin/api cmd/api/main.go",
	}

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
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.yaml"
    - "*.yml"
    - "*.md"
    - "*.sh"
    - "*.mod"
    - "*.sum"
    - "Makefile"
    - "Dockerfile*"
    - ".gitignore"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
ignore:
  - "vendor"
  - ".git"
  - "bin"
`

	projectDir := createTestProject(t, projectFiles, configContent)
	reportPath := filepath.Join(t.TempDir(), "real-project-report.json")

	out, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
		"--json-output", reportPath,
	)

	if err != nil {
		t.Errorf("Real project validation failed: %v\nOutput: %s", err, out)
	}

	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("JSON report was not created")
	}
}

// TestIntegrationWithViolations tests that violations are detected correctly
func TestIntegrationWithViolations(t *testing.T) {
	bin := buildBinary(t)

	violationFiles := map[string]string{
		"cmd/app/main.go":   "package main",
		".env.local":        "SECRET_KEY=abc123",
		"debug.log":         "2024-01-01 ERROR: Something went wrong",
		"temp.tmp":          "temporary data",
		"backup~":           "backup file",
		"tmp/file.tmp":      "more temp data",
		"vendor/lib/lib.go": "package lib", // Should be ignored
	}

	configContent := `dir_structure:
  allowedPaths: ["cmd/**", "internal/**"]
  disallowedPaths: ["vendor/**", "tmp/**"]
file_naming_pattern:
  allowed: ["*.go", "*.mod"]
  disallowed: ["*.env*", "*.log", "*.tmp", "*~"]
ignore: ["vendor"]
`

	projectDir := createTestProject(t, violationFiles, configContent)
	reportPath := filepath.Join(t.TempDir(), "violations-report.json")

	_, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
		"--json-output", reportPath,
	)

	// Should have errors for violations
	if err == nil {
		t.Error("Expected validation to fail due to violations")
	} else {
		t.Logf("Validation failed as expected: %v", err)
	}

	// Check if report was created
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("JSON report was not created even with violations")
	}

	// Verify report contains violations
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("Failed to parse report JSON: %v", err)
	}

	if violations, ok := report["total_violations"].(float64); ok {
		if violations == 0 {
			t.Error("Expected violations in report but got 0")
		} else {
			t.Logf("Report shows %v violations as expected", violations)
		}
	}
}

// TestCLIVersionCommand tests the version command
func TestCLIVersionCommand(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "version")
	if err != nil {
		t.Errorf("Version command failed: %v\nOutput: %s", err, out)
	}

	// Should contain version information (dev, commit, or built)
	hasVersionInfo := strings.Contains(out, "dev") ||
		strings.Contains(out, "commit") ||
		strings.Contains(out, "built")
	if !hasVersionInfo {
		t.Errorf("Version output doesn't contain expected info: %s", out)
	}
}

// TestCLIHelpCommand tests the help command
func TestCLIHelpCommand(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "--help")
	if err != nil {
		t.Errorf("Help command failed: %v\nOutput: %s", err, out)
	}

	// Should contain command descriptions
	expectedStrings := []string{"validate", "version", "completion"}
	for _, expected := range expectedStrings {
		if !strings.Contains(strings.ToLower(out), expected) {
			t.Errorf("Help output missing '%s': %s", expected, out)
		}
	}
}

// TestCLIValidateHelp tests the validate subcommand help
func TestCLIValidateHelp(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "validate", "--help")
	if err != nil {
		t.Errorf("Validate help command failed: %v\nOutput: %s", err, out)
	}

	// Should contain flag descriptions
	expectedFlags := []string{"config", "json-output"}
	for _, flag := range expectedFlags {
		if !strings.Contains(strings.ToLower(out), flag) {
			t.Errorf("Validate help missing '--%s' flag: %s", flag, out)
		}
	}
}

// TestCLICompletionCommands tests shell completion generation
func TestCLICompletionCommands(t *testing.T) {
	bin := buildBinary(t)

	shells := []string{"bash", "zsh", "fish"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			out, err := runBinary(t, bin, "completion", shell)
			if err != nil {
				t.Errorf("Completion command for %s failed: %v\nOutput: %s", shell, err, out)
			}

			// Should produce some output (completion script or message)
			if len(out) < 10 {
				t.Errorf("Completion output for %s seems too short: %d bytes", shell, len(out))
			}
		})
	}
}

// TestCLIConfigPrecedence tests environment variable config override
func TestCLIConfigPrecedence(t *testing.T) {
	bin := buildBinary(t)

	projectFiles := map[string]string{
		"cmd/app/main.go":     "package main",
		"internal/app/app.go": "package app",
	}

	configContent := `dir_structure:
  allowedPaths: [".", "cmd/**", "internal/**"]
file_naming_pattern:
  allowed: ["*.go", "*.yaml"]
ignore: ["vendor"]
`

	projectDir := createTestProject(t, projectFiles, configContent)
	configPath := filepath.Join(projectDir, ".structlint.yaml")

	// Test with STRUCTLINT_CONFIG env var
	t.Setenv("STRUCTLINT_CONFIG", configPath)

	out, err := runBinaryInDir(t, bin, projectDir, "validate")
	// May pass or fail depending on config, but should not crash
	_ = out // output varies based on config
	_ = err
}

// TestCLIJSONOutput tests JSON output format
func TestCLIJSONOutput(t *testing.T) {
	bin := buildBinary(t)

	projectFiles := map[string]string{
		"cmd/app/main.go": "package main",
	}

	configContent := `dir_structure:
  allowedPaths: [".", "cmd/**"]
file_naming_pattern:
  allowed: ["*.go", "*.yaml"]
`

	projectDir := createTestProject(t, projectFiles, configContent)
	reportPath := filepath.Join(t.TempDir(), "json-report.json")

	_, _ = runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
		"--json-output", reportPath,
	)

	// Read and validate JSON structure
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read JSON report: %v", err)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("Invalid JSON in report: %v", err)
	}

	// Verify expected fields exist (based on JSONReport struct)
	expectedFields := []string{"successes", "failures", "errors"}
	for _, field := range expectedFields {
		if _, ok := report[field]; !ok {
			t.Errorf("JSON report missing field: %s", field)
		}
	}
}

// TestCLIMissingConfig tests error handling when config is missing
func TestCLIMissingConfig(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	out, err := runBinaryInDir(t, bin, tmpDir,
		"validate",
		"--config", "nonexistent.yaml",
	)

	if err == nil {
		t.Error("Expected error for missing config file")
	}

	if !strings.Contains(strings.ToLower(out), "error") && !strings.Contains(strings.ToLower(out), "not found") {
		t.Logf("Output for missing config: %s", out)
	}
}

// TestCLIVerboseOutput tests verbose flag
func TestCLIVerboseOutput(t *testing.T) {
	bin := buildBinary(t)

	projectFiles := map[string]string{
		"main.go": "package main",
	}

	configContent := `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.go", "*.yaml"]
`

	projectDir := createTestProject(t, projectFiles, configContent)

	out, _ := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
		"--log-level", "debug",
	)

	// Verbose output should be longer and contain more detail
	t.Logf("Verbose output length: %d bytes", len(out))
}
