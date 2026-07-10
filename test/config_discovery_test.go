package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const discoveryConfig = `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.go", "*.md", "*.yaml", "*.yml", "*.json"]
ignore: [".git"]
`

func TestDiscovery_FindsConfigInParent(t *testing.T) {
	bin := buildBinary(t)
	repo := t.TempDir()
	initGitRepo(t, repo)

	writeTestFile(t, repo, ".structlint.yaml", discoveryConfig)
	writeTestFile(t, repo, "internal/foo/main.go", "package foo\n")

	subdir := filepath.Join(repo, "internal", "foo")
	out, err := runBinaryInDir(t, bin, subdir, "validate", "--silent")
	if err != nil {
		t.Fatalf("expected discovery to succeed, out:\n%s\nerr: %v", out, err)
	}
}

func TestDiscovery_StopsAtGitBoundary(t *testing.T) {
	bin := buildBinary(t)
	outer := t.TempDir()

	// Config lives ABOVE the git repo. Discovery must stop at .git.
	writeTestFile(t, outer, ".structlint.yaml", discoveryConfig)
	inner := filepath.Join(outer, "repo")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, inner)
	writeTestFile(t, inner, "src/main.go", "package src\n")

	out, err := runBinaryInDir(t, bin, filepath.Join(inner, "src"), "validate", "--silent")
	if err == nil {
		t.Fatalf("expected config-not-found error (outer config must not leak past .git), out:\n%s", out)
	}
	if !strings.Contains(out, "configuration file not found") {
		t.Errorf("expected 'configuration file not found' message, got:\n%s", out)
	}
}

func TestDiscovery_GitDirItselfIsChecked(t *testing.T) {
	bin := buildBinary(t)
	repo := t.TempDir()
	initGitRepo(t, repo)
	writeTestFile(t, repo, ".structlint.yaml", discoveryConfig)
	writeTestFile(t, repo, "sub/x.go", "package sub\n")

	out, err := runBinaryInDir(t, bin, filepath.Join(repo, "sub"), "validate", "--silent")
	if err != nil {
		t.Fatalf("expected repo-root config to be discovered (inclusive .git stop), out:\n%s\nerr: %v", out, err)
	}
}

func TestDiscovery_StartDirConfigWins(t *testing.T) {
	bin := buildBinary(t)
	repo := t.TempDir()
	initGitRepo(t, repo)

	// Parent config allows *.go; nested strict config only allows *.md — the
	// nested one must win. If the parent's is (incorrectly) used, our .go
	// files pass; if the nested wins, they fail.
	writeTestFile(t, repo, ".structlint.yaml", discoveryConfig)
	strictNested := `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.md"]
ignore: [".git"]
`
	sub := filepath.Join(repo, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, sub, ".structlint.yaml", strictNested)
	writeTestFile(t, sub, "main.go", "package main\n")
	writeTestFile(t, sub, "notes.md", "# notes\n")

	out, err := runBinaryInDir(t, bin, sub, "validate")
	if err == nil {
		t.Fatalf("expected nested strict config to reject main.go, out:\n%s", out)
	}
	if !strings.Contains(out, "main.go") {
		t.Errorf("expected main.go in violation output, got:\n%s", out)
	}
}

func TestDiscovery_NamePriority(t *testing.T) {
	bin := buildBinary(t)
	repo := t.TempDir()
	initGitRepo(t, repo)

	writeTestFile(t, repo, ".structlint.yaml", discoveryConfig)
	// .yml file exists too — must NOT be used because .yaml wins.
	writeTestFile(t, repo, ".structlint.yml", "!!! invalid yaml because .yaml wins\n")
	writeTestFile(t, repo, "main.go", "package main\n")

	out, err := runBinaryInDir(t, bin, repo, "validate", "--silent")
	if err != nil {
		t.Fatalf("expected .yaml to win over .yml, out:\n%s\nerr: %v", out, err)
	}
}

func TestDiscovery_ExplicitConfigDisablesSearch(t *testing.T) {
	bin := buildBinary(t)
	repo := t.TempDir()
	initGitRepo(t, repo)

	writeTestFile(t, repo, ".structlint.yaml", discoveryConfig) // parent config
	sub := filepath.Join(repo, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, sub, "main.go", "package main\n")

	out, err := runBinaryInDir(t, bin, sub, "validate", "--config", "missing.yaml", "--silent")
	if err == nil {
		t.Fatalf("expected explicit --config missing.yaml to hard-fail, out:\n%s", out)
	}
	if !strings.Contains(out, "missing.yaml") {
		t.Errorf("error should mention the explicit path, got:\n%s", out)
	}
}

func TestDiscovery_NoConfigAnywhereErrors(t *testing.T) {
	bin := buildBinary(t)
	repo := t.TempDir()
	initGitRepo(t, repo)
	writeTestFile(t, repo, "main.go", "package main\n")

	out, err := runBinaryInDir(t, bin, repo, "validate")
	if err == nil {
		t.Fatalf("expected error when no config exists, out:\n%s", out)
	}
	if !strings.Contains(out, "structlint init") {
		t.Errorf("error should hint at `structlint init`, got:\n%s", out)
	}
}

func TestDiscovery_LogsChosenConfig(t *testing.T) {
	bin := buildBinary(t)
	repo := t.TempDir()
	initGitRepo(t, repo)
	writeTestFile(t, repo, ".structlint.yaml", discoveryConfig)
	writeTestFile(t, repo, "sub/x.go", "package sub\n")

	out, err := runBinaryInDir(t, bin, filepath.Join(repo, "sub"), "validate")
	if err != nil {
		t.Fatalf("expected success, err=%v out:\n%s", err, out)
	}
	if !strings.Contains(out, "using config") {
		t.Errorf("expected 'using config' log line, got:\n%s", out)
	}
	if !strings.Contains(out, ".structlint.yaml") {
		t.Errorf("expected chosen config path in log, got:\n%s", out)
	}
}
