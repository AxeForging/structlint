package test

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

// TestRequiredDirectories tests validation of required directories
func TestRequiredDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-required-dirs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create some directories
	dirs := []string{
		"src",
		"docs",
		"scripts",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create config with required directories
	configContent := `dir_structure:
  allowedPaths: ["src/**", "docs", "scripts"]
  disallowedPaths: ["tmp"]
  requiredPaths: ["src", "docs", "scripts", "tests"]  # tests is missing
file_naming_pattern:
  allowed: ["*.go", "*.md"]
  disallowed: ["*.env*"]
ignore: []
`
	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config and run validator
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger, _ := logging.New("error", true) // silent logger for tests
	v := validator.New(cfg, logger)
	v.Silent = true // Suppress output for test

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	v.ValidateRequiredPaths(".")

	// Should have 1 error for missing "tests" directory
	expectedErrors := 1
	if len(v.Errors) != expectedErrors {
		t.Errorf("Expected %d errors, got %d. Errors: %v", expectedErrors, len(v.Errors), v.Errors)
	}

	// Check specific error message
	found := false
	for _, err := range v.Errors {
		if strings.Contains(err, "Required directory missing: tests") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected error message about missing 'tests' directory")
	}

	// Should have 3 successes for existing directories
	expectedSuccesses := 3
	if v.Successes != expectedSuccesses {
		t.Errorf("Expected %d successes, got %d", expectedSuccesses, v.Successes)
	}
}

// TestRequiredFiles tests validation of required file patterns
func TestRequiredFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-required-files")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create some files
	files := map[string]string{
		"main.go":     "package main",
		"README.md":   "# Project",
		"go.mod":      "module test",
		"config.yaml": "config: value",
	}
	for path, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, path), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Create config with required files
	configContent := `dir_structure:
  allowedPaths: ["."]
  disallowedPaths: []
file_naming_pattern:
  allowed: ["*.go", "*.md", "*.mod", "*.yaml"]
  disallowed: ["*.env*"]
  required: ["*.go", "README.md", "go.mod", "Dockerfile"]  # Dockerfile is missing
ignore: []
`
	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config and run validator
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger, _ := logging.New("error", true) // silent logger for tests
	v := validator.New(cfg, logger)
	v.Silent = true // Suppress output for test

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	v.ValidateRequiredFiles(".")

	// Should have 1 error for missing "Dockerfile"
	expectedErrors := 1
	if len(v.Errors) != expectedErrors {
		t.Errorf("Expected %d errors, got %d. Errors: %v", expectedErrors, len(v.Errors), v.Errors)
	}

	// Check specific error message
	found := false
	for _, err := range v.Errors {
		if strings.Contains(err, "Required file pattern missing: Dockerfile") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected error message about missing 'Dockerfile'")
	}

	// Should have 3 successes for existing required files
	expectedSuccesses := 3
	if v.Successes != expectedSuccesses {
		t.Errorf("Expected %d successes, got %d", expectedSuccesses, v.Successes)
	}
}

// TestRequiredWithGlobPatterns tests required files with glob patterns
func TestRequiredWithGlobPatterns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-required-glob")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "src"), 0o755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	// Create some files
	files := map[string]string{
		"src/main.go":  "package main",
		"src/utils.go": "package utils",
		"README.md":    "# Project",
		"docs/api.md":  "# API Docs",
	}
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Create config with glob patterns
	configContent := `dir_structure:
  allowedPaths: ["src/**", "docs/**"]
  disallowedPaths: []
file_naming_pattern:
  allowed: ["*.go", "*.md", ".structlint.yaml"]
  disallowed: ["*.env*"]
  required: ["src/*.go", "*.md", "tests/*_test.go"]  # tests/*_test.go is missing
ignore: []
`
	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config and run validator
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger, _ := logging.New("error", true) // silent logger for tests
	v := validator.New(cfg, logger)
	v.Silent = true // Suppress output for test

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	v.ValidateRequiredFiles(".")

	// Should have 1 error for missing "tests/*_test.go"
	expectedErrors := 1
	if len(v.Errors) != expectedErrors {
		t.Errorf("Expected %d errors, got %d. Errors: %v", expectedErrors, len(v.Errors), v.Errors)
	}

	// Check specific error message
	found := false
	for _, err := range v.Errors {
		if strings.Contains(err, "Required file pattern missing: tests/*_test.go") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected error message about missing 'tests/*_test.go'")
	}

	// Should have 2 successes for existing patterns (src/*.go and *.md)
	expectedSuccesses := 2
	if v.Successes != expectedSuccesses {
		t.Errorf("Expected %d successes, got %d", expectedSuccesses, v.Successes)
	}
}

// TestRequiredWithIgnore tests that ignored paths are not checked for required files
func TestRequiredWithIgnore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-required-ignore")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "vendor"), 0o755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	// Create files including one in ignored directory
	files := map[string]string{
		"main.go":       "package main",
		"vendor/lib.go": "package lib", // This should be ignored
		"README.md":     "# Project",
	}
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Create config with required files and ignore
	configContent := `dir_structure:
  allowedPaths: ["."]
  disallowedPaths: []
file_naming_pattern:
  allowed: ["*.go", "*.md"]
  disallowed: ["*.env*"]
  required: ["*.go", "README.md"]  # Both should be found
ignore: ["vendor"]
`
	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config and run validator
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger, _ := logging.New("error", true) // silent logger for tests
	v := validator.New(cfg, logger)
	v.Silent = true // Suppress output for test

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	v.ValidateRequiredFiles(".")

	// Should have 0 errors since both required patterns are found (vendor is ignored)
	expectedErrors := 0
	if len(v.Errors) != expectedErrors {
		t.Errorf("Expected %d errors, got %d. Errors: %v", expectedErrors, len(v.Errors), v.Errors)
	}

	// Should have 2 successes for existing patterns
	expectedSuccesses := 2
	if v.Successes != expectedSuccesses {
		t.Errorf("Expected %d successes, got %d", expectedSuccesses, v.Successes)
	}
}

// TestRequiredIntegration tests the complete validation flow with required fields
func TestRequiredIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-required-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a complete project structure
	projectStructure := map[string]string{
		"cmd/app/main.go":     "package main",
		"internal/app/app.go": "package app",
		"README.md":           "# Project",
		"go.mod":              "module test",
		"docs/api.md":         "# API",
		// Missing: tests/ directory and Dockerfile
	}

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

	// Create config with all validation types
	configContent := `dir_structure:
  allowedPaths: ["cmd/**", "internal/**", "docs", "."]
  disallowedPaths: ["tmp"]
  requiredPaths: ["cmd", "internal", "tests"]  # tests is missing
file_naming_pattern:
  allowed: ["*.go", "*.md", "*.mod", ".structlint.yaml"]
  disallowed: ["*.env*", "*.log"]
  required: ["*.go", "README.md", "go.mod", "Dockerfile"]  # Dockerfile is missing
ignore: ["vendor"]
`
	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config and run validator
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger, _ := logging.New("error", true) // silent logger for tests
	v := validator.New(cfg, logger)
	v.Silent = true // Suppress output for test

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	// Run all validations
	v.ValidateDirStructure(".")
	v.ValidateFileNaming(".")
	v.ValidateRequiredPaths(".")
	v.ValidateRequiredFiles(".")

	// Should have 2 errors for missing required items
	expectedErrors := 2
	if len(v.Errors) != expectedErrors {
		t.Errorf("Expected %d errors, got %d. Errors: %v", expectedErrors, len(v.Errors), v.Errors)
	}

	// Check for specific error messages
	missingTests := false
	missingDockerfile := false
	for _, err := range v.Errors {
		if strings.Contains(err, "Required directory missing: tests") {
			missingTests = true
		}
		if strings.Contains(err, "Required file pattern missing: Dockerfile") {
			missingDockerfile = true
		}
	}

	if !missingTests {
		t.Error("Expected error message about missing 'tests' directory")
	}
	if !missingDockerfile {
		t.Error("Expected error message about missing 'Dockerfile'")
	}

	// Test JSON report
	jsonPath := filepath.Join(tmpDir, "required-report.json")
	if err := v.SaveJSONReport(jsonPath); err != nil {
		t.Fatalf("Failed to save JSON report: %v", err)
	}

	// Read and verify JSON report
	reportData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON report: %v", err)
	}

	var report validator.JSONReport
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Fatalf("Failed to unmarshal JSON report: %v", err)
	}

	if report.Failures != expectedErrors {
		t.Errorf("Expected %d failures in JSON report, but got %d", expectedErrors, report.Failures)
	}

	// Check that the summary includes the new violation types
	foundMissingDir := false
	foundMissingFile := false
	for _, violation := range report.Summary.Violations {
		if violation.Type == "missing_required_directory" {
			foundMissingDir = true
		}
		if violation.Type == "missing_required_file" {
			foundMissingFile = true
		}
	}

	if !foundMissingDir {
		t.Error("Expected missing_required_directory violation type in summary")
	}
	if !foundMissingFile {
		t.Error("Expected missing_required_file violation type in summary")
	}
}

// TestRequiredEmptyConfig tests behavior with empty required fields
func TestRequiredEmptyConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-required-empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create config with empty required fields
	configContent := `dir_structure:
  allowedPaths: ["."]
  disallowedPaths: []
  requiredPaths: []  # Empty
file_naming_pattern:
  allowed: ["*.go"]
  disallowed: ["*.env*"]
  required: []  # Empty
ignore: []
`
	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config and run validator
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger, _ := logging.New("error", true) // silent logger for tests
	v := validator.New(cfg, logger)
	v.Silent = true // Suppress output for test

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	v.ValidateRequiredPaths(".")
	v.ValidateRequiredFiles(".")

	// Should have 0 errors since no required items are specified
	expectedErrors := 0
	if len(v.Errors) != expectedErrors {
		t.Errorf("Expected %d errors, got %d. Errors: %v", expectedErrors, len(v.Errors), v.Errors)
	}

	// Should have 0 successes since no required items are specified
	expectedSuccesses := 0
	if v.Successes != expectedSuccesses {
		t.Errorf("Expected %d successes, got %d", expectedSuccesses, v.Successes)
	}
}
