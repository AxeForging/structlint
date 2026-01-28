package test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/youngestaxe/structlint/internal/config"
	"github.com/youngestaxe/structlint/internal/logging"
	"github.com/youngestaxe/structlint/internal/validator"
)

// TestLargeScaleViolations tests behavior with many violations
func TestLargeScaleViolations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "structlint-large-scale")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a project with many violations
	t.Log("Creating 1000+ files with violations...")

	// Create many .env files (disallowed)
	for i := 0; i < 100; i++ {
		envFile := filepath.Join(tmpDir, fmt.Sprintf(".env.%d", i))
		if err := os.WriteFile(envFile, []byte("SECRET=123"), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", envFile, err)
		}
	}

	// Create many .log files (disallowed)
	for i := 0; i < 100; i++ {
		logFile := filepath.Join(tmpDir, fmt.Sprintf("app.%d.log", i))
		if err := os.WriteFile(logFile, []byte("log content"), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", logFile, err)
		}
	}

	// Create many .tmp files (disallowed)
	for i := 0; i < 100; i++ {
		tmpFile := filepath.Join(tmpDir, fmt.Sprintf("temp.%d.tmp", i))
		if err := os.WriteFile(tmpFile, []byte("temp content"), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", tmpFile, err)
		}
	}

	// Create some allowed files for comparison
	allowedFiles := map[string]string{
		"cmd/app/main.go":     "package main",
		"internal/app/app.go": "package app",
		"go.mod":              "module test",
		"README.md":           "# Test",
	}

	for path, content := range allowedFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Create configuration that will catch violations
	configContent := `dir_structure:
  allowedPaths: ["cmd/**", "internal/**"]
  disallowedPaths: ["vendor/**"]
file_naming_pattern:
  allowed: ["*.go", "*.mod", "*.md"]
  disallowed: ["*.env*", "*.log", "*.tmp"]
ignore: ["vendor"]
`

	configPath := filepath.Join(tmpDir, ".structlint.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create validator
	logger, _ := logging.New("info", false) // Show all output
	v := validator.New(cfg, logger)
	v.Silent = false // Don't silence to see the behavior

	t.Logf("Validating %d+ files...", 300) // 300 violations + some allowed files

	// Change to test directory
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	// Run validation
	v.ValidateDirStructure(".")
	v.ValidateFileNaming(".")

	t.Logf("Results: %d successes, %d failures", v.Successes, len(v.Errors))

	// Check if we have the expected number of violations
	expectedViolations := 302 // 100 .env + 100 .log + 100 .tmp + 1 root dir + 1 config file
	if len(v.Errors) != expectedViolations {
		t.Errorf("Expected %d violations, got %d", expectedViolations, len(v.Errors))
	}

	// Test JSON report with many errors
	reportPath := "large-scale-report.json"
	if err := v.SaveJSONReport(reportPath); err != nil {
		t.Fatalf("Failed to save JSON report: %v", err)
	}

	t.Logf("Large-scale validation report saved to: %s", reportPath)
}
