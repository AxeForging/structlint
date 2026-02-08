package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitGeneratesConfig tests that init creates a valid config file
func TestInitGeneratesConfig(t *testing.T) {
	bin := buildBinary(t)

	types := []string{"go", "node", "python", "generic"}
	for _, projType := range types {
		t.Run(projType, func(t *testing.T) {
			tmpDir := t.TempDir()

			out, err := runBinaryInDir(t, bin, tmpDir, "init", "--type", projType)
			if err != nil {
				t.Fatalf("init --type %s failed: %v\nOutput: %s", projType, err, out)
			}

			// Config file should exist
			configPath := filepath.Join(tmpDir, ".structlint.yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				t.Fatalf("config file not created for type %s", projType)
			}

			// Config should be valid YAML that validate can parse
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read config: %v", err)
			}

			if len(data) < 50 {
				t.Errorf("config too short for type %s: %d bytes", projType, len(data))
			}

			// Output should mention the project type
			if !strings.Contains(out, projType) {
				t.Errorf("output should mention project type %s: %s", projType, out)
			}
		})
	}
}

// TestInitAutoDetectsGo tests project type auto-detection for Go projects
func TestInitAutoDetectsGo(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create a go.mod to signal Go project
	writeTestFile(t, tmpDir, "go.mod", "module example.com/test\n\ngo 1.24\n")

	out, err := runBinaryInDir(t, bin, tmpDir, "init")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, out)
	}

	// Should detect Go project
	if !strings.Contains(out, "go") {
		t.Errorf("should detect Go project, got: %s", out)
	}

	// Config should contain Go-specific patterns
	data, err := os.ReadFile(filepath.Join(tmpDir, ".structlint.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "*.go") {
		t.Error("Go config should contain *.go pattern")
	}
}

// TestInitAutoDetectsNode tests project type auto-detection for Node projects
func TestInitAutoDetectsNode(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	writeTestFile(t, tmpDir, "package.json", `{"name": "test"}`)

	out, err := runBinaryInDir(t, bin, tmpDir, "init")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "node") {
		t.Errorf("should detect Node project, got: %s", out)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".structlint.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "*.js") {
		t.Error("Node config should contain *.js pattern")
	}
}

// TestInitAutoDetectsPython tests project type auto-detection for Python projects
func TestInitAutoDetectsPython(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	writeTestFile(t, tmpDir, "pyproject.toml", "[project]\nname = \"test\"\n")

	out, err := runBinaryInDir(t, bin, tmpDir, "init")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "python") {
		t.Errorf("should detect Python project, got: %s", out)
	}
}

// TestInitWontOverwriteExisting tests that init refuses to overwrite existing config
func TestInitWontOverwriteExisting(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create existing config
	writeTestFile(t, tmpDir, ".structlint.yaml", "existing: config\n")

	_, err := runBinaryInDir(t, bin, tmpDir, "init")
	if err == nil {
		t.Error("expected error when config already exists")
	}
}

// TestInitForceOverwrites tests that --force overwrites existing config
func TestInitForceOverwrites(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	writeTestFile(t, tmpDir, ".structlint.yaml", "old: config\n")

	out, err := runBinaryInDir(t, bin, tmpDir, "init", "--type", "go", "--force")
	if err != nil {
		t.Fatalf("init --force failed: %v\nOutput: %s", err, out)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".structlint.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if strings.Contains(string(data), "old: config") {
		t.Error("config should have been overwritten")
	}
}

// TestInitConfigIsValidatable tests that generated config can be used by validate
func TestInitConfigIsValidatable(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Init with Go type
	out, err := runBinaryInDir(t, bin, tmpDir, "init", "--type", "go")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, out)
	}

	// Create minimal Go project structure
	writeTestFile(t, tmpDir, "cmd/app/main.go", "package main\n")
	writeTestFile(t, tmpDir, "go.mod", "module example.com/test\n\ngo 1.24\n")
	writeTestFile(t, tmpDir, "README.md", "# Test\n")
	writeTestFile(t, tmpDir, ".gitignore", "bin/\n")

	// Validate should work with the generated config
	out, err = runBinaryInDir(t, bin, tmpDir, "validate")
	if err != nil {
		t.Logf("Validate output: %s", out)
		// Some violations are expected since we only created a minimal structure
		// The key thing is that the config parsed correctly (not a YAML error)
		if strings.Contains(out, "yaml") || strings.Contains(out, "parse") {
			t.Errorf("generated config has parsing errors: %s", out)
		}
	}
}

// TestInitUnknownType tests error for unknown project type
func TestInitUnknownType(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	out, err := runBinaryInDir(t, bin, tmpDir, "init", "--type", "rust")
	if err == nil {
		t.Error("expected error for unknown project type")
	}

	if !strings.Contains(out, "unknown project type") {
		t.Errorf("error should mention unknown project type: %s", out)
	}
}
