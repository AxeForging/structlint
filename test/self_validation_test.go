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

// TestSelfValidation validates the repository through the same built binary a
// user and CI execute. This catches CLI wiring and rule-registry regressions
// that direct calls to individual validator methods would miss.
func TestSelfValidation(t *testing.T) {
	root := repoRoot(t)
	bin := buildBinary(t)
	out, err := runBinaryInDir(t, bin, root, "validate", "--config", ".structlint.yaml")
	if err != nil {
		t.Fatalf("built binary rejected repository:\n%s", out)
	}
	if !strings.Contains(out, "0 violations found") {
		t.Fatalf("expected successful self-validation summary, got:\n%s", out)
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

		// Skip directories, but prune whole subtrees we know are ignored.
		if info.IsDir() {
			if info.Name() == "testdata" || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
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
	configPath := filepath.Join(repoRoot(t), ".structlint.yaml")

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

func TestSelfPolicyContainsArchitectureAndRepositoryContracts(t *testing.T) {
	cfg, err := config.LoadConfig(filepath.Join(repoRoot(t), ".structlint.yaml"))
	if err != nil {
		t.Fatalf("load self policy: %v", err)
	}

	boundaryIDs := make(map[string]bool, len(cfg.Boundaries))
	for _, rule := range cfg.Boundaries {
		boundaryIDs[rule.ID] = true
	}
	for _, id := range []string{
		"build-is-leaf",
		"logging-is-leaf",
		"config-does-not-import-features",
		"validator-does-not-import-orchestration",
	} {
		if !boundaryIDs[id] {
			t.Errorf("self policy missing architectural boundary %q", id)
		}
	}

	groupIDs := make(map[string]bool, len(cfg.RequiredGroups))
	for _, group := range cfg.RequiredGroups {
		groupIDs[group.ID] = true
	}
	for _, id := range []string{"release-configuration", "public-schema", "shipped-agent-skill", "commands-have-entrypoints"} {
		if !groupIDs[id] {
			t.Errorf("self policy missing repository contract %q", id)
		}
	}
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
