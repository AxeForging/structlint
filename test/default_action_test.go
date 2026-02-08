package test

import (
	"strings"
	"testing"
)

// TestDefaultActionRunsValidate tests that running structlint without subcommand runs validate
func TestDefaultActionRunsValidate(t *testing.T) {
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

	// Run without subcommand - should run validate
	out, err := runBinaryInDir(t, bin, projectDir)
	if err != nil {
		t.Fatalf("default action failed: %v\nOutput: %s", err, out)
	}

	// Should show validation summary
	if !strings.Contains(out, "Validation Summary") {
		t.Errorf("expected validation summary in output: %s", out)
	}
}

// TestDefaultActionMissingConfigSuggestsInit tests helpful error when config is missing
func TestDefaultActionMissingConfigSuggestsInit(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	out, err := runBinaryInDir(t, bin, tmpDir)
	if err == nil {
		t.Error("expected error when config is missing")
	}

	if !strings.Contains(out, "structlint init") {
		t.Errorf("error should suggest 'structlint init': %s", out)
	}
}

// TestHelpShowsInitCommand tests that help output includes init command
func TestHelpShowsInitCommand(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "--help")
	if err != nil {
		t.Fatalf("help failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "init") {
		t.Errorf("help should show init command: %s", out)
	}
}
