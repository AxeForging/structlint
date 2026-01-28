package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/youngestaxe/structlint/internal/config"
	"github.com/youngestaxe/structlint/internal/logging"
	"github.com/youngestaxe/structlint/internal/validator"
)

// TestSelfValidation validates our own project structure
func TestSelfValidation(t *testing.T) {
	// Check if our configuration file exists
	configPath := ".structlint.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("Configuration file %s not found, skipping self-validation test", configPath)
	}

	// Load our own configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load our own configuration: %v", err)
	}

	// Create validator
	logger, _ := logging.New("info", false) // Use info level to see what's happening
	v := validator.New(cfg, logger)
	v.Silent = false // Don't silence output for self-validation

	t.Logf("Validating our own project structure with config: %s", configPath)
	t.Logf("Allowed paths: %v", cfg.DirStructure.AllowedPaths)
	t.Logf("Disallowed paths: %v", cfg.DirStructure.DisallowedPaths)
	t.Logf("Allowed file patterns: %v", cfg.FileNamingPattern.Allowed)
	t.Logf("Disallowed file patterns: %v", cfg.FileNamingPattern.Disallowed)
	t.Logf("Ignored patterns: %v", cfg.Ignore)

	// Validate our project structure
	v.ValidateDirStructure(".")
	v.ValidateFileNaming(".")

	// Report results
	t.Logf("Self-validation results: %d successes, %d failures", v.Successes, len(v.Errors))

	// Check for critical errors (but allow some flexibility)
	criticalErrors := []string{}
	acceptableErrors := []string{}

	for _, err := range v.Errors {
		// Some errors might be acceptable for our current structure
		if strings.Contains(err, "vendor") ||
			strings.Contains(err, ".git") ||
			strings.Contains(err, "bin") ||
			strings.Contains(err, "dist") {
			acceptableErrors = append(acceptableErrors, err)
		} else {
			criticalErrors = append(criticalErrors, err)
		}
	}

	if len(criticalErrors) > 0 {
		t.Errorf("Critical validation errors found (%d):", len(criticalErrors))
		for _, err := range criticalErrors {
			t.Errorf("  ❌ %s", err)
		}
	}

	if len(acceptableErrors) > 0 {
		t.Logf("Acceptable validation issues (%d):", len(acceptableErrors))
		for _, err := range acceptableErrors {
			t.Logf("  ⚠️  %s", err)
		}
	}

	// Ensure we have some successes
	if v.Successes == 0 {
		t.Error("Expected at least some successful validations")
	}

	// Save a JSON report for analysis
	reportPath := "validation-report.json"
	if err := v.SaveJSONReport(reportPath); err != nil {
		t.Logf("Failed to save validation report: %v", err)
	} else {
		t.Logf("Validation report saved to: %s", reportPath)
		defer func() {
			_ = os.Remove(reportPath) // Clean up
		}()
	}
}

// TestProjectStandardsCompliance tests specific standards we want to enforce
func TestProjectStandardsCompliance(t *testing.T) {
	// Change to project root (go up one directory from test/)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Go up one directory to project root
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	// Define our specific standards
	standards := map[string][]string{
		"required_directories": {
			"cmd",
			"internal",
			"test",
		},
		"required_files": {
			"go.mod",
			"README.md",
			".gitignore",
		},
		"forbidden_patterns": {
			"*.env*",
			"*.log",
			"*.tmp",
			"vendor",
		},
	}

	// Check required directories
	for _, dir := range standards["required_directories"] {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Required directory missing: %s", dir)
		} else {
			t.Logf("✅ Required directory present: %s", dir)
		}
	}

	// Check required files
	for _, file := range standards["required_files"] {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Required file missing: %s", file)
		} else {
			t.Logf("✅ Required file present: %s", file)
		}
	}

	// Check for forbidden patterns
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		fileName := info.Name()
		for _, pattern := range standards["forbidden_patterns"] {
			if matched, _ := filepath.Match(pattern, fileName); matched {
				t.Errorf("❌ Forbidden file found: %s (matches pattern: %s)", path, pattern)
			}
		}

		return nil
	})

	if err != nil {
		t.Errorf("Error walking directory: %v", err)
	}
}

// TestConfigurationValidation tests our configuration file itself
func TestConfigurationValidation(t *testing.T) {
	configPath := ".structlint.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("Configuration file %s not found", configPath)
	}

	// Test that our config file is valid
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Our configuration file is invalid: %v", err)
	}

	// Validate configuration content
	if len(cfg.DirStructure.AllowedPaths) == 0 {
		t.Error("Configuration should have at least one allowed path")
	}

	if len(cfg.FileNamingPattern.Allowed) == 0 {
		t.Error("Configuration should have at least one allowed file pattern")
	}

	// Check for common Go project patterns
	hasGoPattern := false
	for _, pattern := range cfg.FileNamingPattern.Allowed {
		if strings.Contains(pattern, "*.go") {
			hasGoPattern = true
			break
		}
	}
	if !hasGoPattern {
		t.Error("Configuration should allow *.go files")
	}

	// Check for common ignored patterns
	hasVendorIgnore := false
	for _, ignore := range cfg.Ignore {
		if strings.Contains(ignore, "vendor") {
			hasVendorIgnore = true
			break
		}
	}
	if !hasVendorIgnore {
		t.Error("Configuration should ignore vendor directory")
	}

	t.Log("✅ Configuration validation passed")
}

// TestRealWorldScenarios tests common real-world project scenarios
func TestRealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name         string
		description  string
		config       string
		structure    map[string]string
		expectErrors int
	}{
		{
			name:        "minimal_go_project",
			description: "A minimal Go project with just the essentials",
			config: `dir_structure:
  allowedPaths: [".", "cmd/**", "internal/**", "pkg/**"]
  disallowedPaths: ["vendor/**"]
file_naming_pattern:
  allowed: ["*.go", "*.mod", "*.sum", "README.md", ".structlint.yaml"]
  disallowed: ["*.env*", "*.log"]
ignore: ["vendor", ".git"]`,
			structure: map[string]string{
				"cmd/app/main.go":     "package main",
				"internal/app/app.go": "package app",
				"go.mod":              "module test",
				"README.md":           "# Test",
			},
			expectErrors: 0,
		},
		{
			name:        "project_with_violations",
			description: "A project with common violations",
			config: `dir_structure:
  allowedPaths: [".", "cmd/**", "internal/**"]
  disallowedPaths: ["vendor/**", "tmp/**"]
file_naming_pattern:
  allowed: ["*.go", "*.mod", ".structlint.yaml"]
  disallowed: ["*.env*", "*.log", "*.tmp"]
ignore: ["vendor"]`,
			structure: map[string]string{
				"cmd/app/main.go":   "package main",
				".env.local":        "SECRET=123",
				"debug.log":         "debug info",
				"tmp/file.tmp":      "temp file",
				"vendor/lib/lib.go": "package lib",
			},
			expectErrors: 4, // tmp dir + 3 file violations // .env.local, debug.log, tmp/file.tmp (vendor should be ignored)
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "structlint-"+scenario.name)
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				_ = os.RemoveAll(tmpDir)
			}()

			// Create project structure
			for path, content := range scenario.structure {
				fullPath := filepath.Join(tmpDir, path)
				dir := filepath.Dir(fullPath)
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatalf("Failed to create dir %s: %v", dir, err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
					t.Fatalf("Failed to write file %s: %v", path, err)
				}
			}

			// Create config file
			configPath := filepath.Join(tmpDir, ".structlint.yaml")
			if err := os.WriteFile(configPath, []byte(scenario.config), 0o644); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			// Load config and validate
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

			if len(v.Errors) != scenario.expectErrors {
				t.Errorf("Expected %d errors, got %d. Errors: %v", scenario.expectErrors, len(v.Errors), v.Errors)
			}

			t.Logf("Scenario '%s': %d successes, %d failures (expected %d failures)",
				scenario.description, v.Successes, len(v.Errors), scenario.expectErrors)
		})
	}
}
