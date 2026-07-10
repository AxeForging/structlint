package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type suggestReport struct {
	Version    int              `json:"version"`
	ConfigPath string           `json:"configPath"`
	Proposals  []suggestPropose `json:"proposals"`
	ConfigDiff string           `json:"configDiff"`
}

type suggestPropose struct {
	Kind    string   `json:"kind"`
	Section string   `json:"section"`
	Value   string   `json:"value"`
	From    string   `json:"from"`
	To      string   `json:"to"`
	Command string   `json:"command"`
	Path    string   `json:"path"`
	Reason  string   `json:"reason"`
	Paths   []string `json:"paths"`
}

func runSuggestJSON(t *testing.T, bin, dir string) suggestReport {
	t.Helper()
	out, err := runBinaryInDir(t, bin, dir, "suggest", "--format", "json")
	if err != nil {
		t.Fatalf("suggest --format json failed: %v\n%s", err, out)
	}
	var r suggestReport
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("parse suggest json: %v\nraw:\n%s", err, out)
	}
	return r
}

func TestSuggest_JSONContractVersion(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.md"]
ignore: [".git"]
`)
	writeTestFile(t, dir, "README.md", "# t\n")
	writeTestFile(t, dir, "stray.txt", "x\n")

	r := runSuggestJSON(t, bin, dir)
	if r.Version != 1 {
		t.Errorf("expected version 1, got %d", r.Version)
	}
	if r.ConfigPath == "" {
		t.Errorf("expected configPath in report")
	}
	if len(r.Proposals) == 0 {
		t.Errorf("expected non-empty proposals for a violating tree")
	}
}

func TestSuggest_PerCodeProposals(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", `dir_structure:
  allowedPaths:
    - "."
    - "src/**"
  requiredPaths:
    - "src"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.md"
    - "*.yaml"
  disallowed:
    - "*.env*"
  required:
    - "README.md"
placement:
  - id: sql-under-migrations
    files: ["*.sql"]
    mustBeUnder: ["migrations/**"]
ignore: [".git"]
`)
	writeTestFile(t, dir, "src/main.go", "package main\n")
	writeTestFile(t, dir, "README.md", "# t\n")
	writeTestFile(t, dir, "tools/gen.go", "package tools\n") // unallowed_directory
	writeTestFile(t, dir, "notes.txt", "x\n")                // unallowed_file_pattern
	writeTestFile(t, dir, ".env.local", "S=1\n")             // disallowed_file_pattern (note only)
	writeTestFile(t, dir, "stray.sql", "-- x\n")             // placement_violation

	r := runSuggestJSON(t, bin, dir)
	kinds := map[string]int{}
	sections := map[string]string{}
	var moves []suggestPropose
	var notes []suggestPropose
	for _, p := range r.Proposals {
		kinds[p.Kind]++
		if p.Kind == "config_add" {
			sections[p.Section+"|"+p.Value] = p.Reason
		}
		if p.Kind == "move" {
			moves = append(moves, p)
		}
		if p.Kind == "note" {
			notes = append(notes, p)
		}
	}
	if kinds["config_add"] < 2 {
		t.Errorf("expected ≥2 config_add proposals (tools/**, *.txt), got %v", kinds)
	}
	if kinds["move"] < 1 {
		t.Errorf("expected ≥1 move for placement_violation, got %v", kinds)
	}
	if kinds["note"] < 1 {
		t.Errorf("expected ≥1 note for disallowed_file_pattern, got %v", kinds)
	}
	if _, ok := sections["dir_structure.allowedPaths|tools/**"]; !ok {
		t.Errorf("expected tools/** config_add, got sections=%v", sections)
	}
	if _, ok := sections["file_naming_pattern.allowed|*.txt"]; !ok {
		t.Errorf("expected *.txt config_add, got sections=%v", sections)
	}
	for _, m := range moves {
		if !strings.HasPrefix(m.Command, "git mv ") {
			t.Errorf("move command should start with git mv, got %q", m.Command)
		}
	}
	for _, n := range notes {
		if !strings.Contains(n.Reason, "deliberate") {
			t.Errorf("disallowed note should mention 'deliberate', got %q", n.Reason)
		}
	}
}

func TestSuggest_DisallowedNeverLoosened(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.md"]
  disallowed: ["*.env*"]
ignore: [".git"]
`)
	writeTestFile(t, dir, "README.md", "# t\n")
	writeTestFile(t, dir, ".env.local", "S=1\n")

	r := runSuggestJSON(t, bin, dir)
	for _, p := range r.Proposals {
		if p.Section == "file_naming_pattern.disallowed" ||
			strings.Contains(p.Section, "disallowedPaths") {
			t.Errorf("suggest must never propose loosening disallowed, got proposal: %+v", p)
		}
		if p.Kind == "config_add" && p.Value == "*.env*" {
			t.Errorf("suggest must never propose *.env* as allowed, got proposal: %+v", p)
		}
	}
	if strings.Contains(r.ConfigDiff, `+    - "*.env*"`) {
		t.Errorf("configDiff must not add *.env* to allowed:\n%s", r.ConfigDiff)
	}
}

func TestSuggest_ConfigDiffAppliesAndValidatePasses(t *testing.T) {
	bin := buildBinary(t)
	if _, err := exec.LookPath("patch"); err != nil {
		t.Skip("patch not available; skipping round-trip test")
	}
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", `dir_structure:
  allowedPaths:
    - "."
    - "src/**"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.md"
    - "*.yaml"
ignore: [".git"]
`)
	writeTestFile(t, dir, "README.md", "# t\n")
	writeTestFile(t, dir, "src/main.go", "package main\n")
	writeTestFile(t, dir, "tools/gen.go", "package tools\n")

	r := runSuggestJSON(t, bin, dir)
	if r.ConfigDiff == "" {
		t.Fatalf("expected non-empty configDiff, report=%+v", r)
	}
	// Apply the diff with patch. Keep the patch file OUTSIDE the fixture
	// dir so it doesn't become an unallowed_file_pattern violation itself.
	patchDir := t.TempDir()
	patchPath := filepath.Join(patchDir, "suggest.patch")
	if err := os.WriteFile(patchPath, []byte(r.ConfigDiff), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("patch", "-p1", "-i", patchPath)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("patch failed: %v\n%s", err, string(out))
	}
	// Now validate must pass for the config_add-mapped violations.
	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err != nil {
		t.Fatalf("expected validate to pass after applying diff; err=%v out:\n%s", err, out)
	}
}

func TestSuggest_ExitZeroWithProposals(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.md"]
ignore: [".git"]
`)
	writeTestFile(t, dir, "README.md", "# t\n")
	writeTestFile(t, dir, "stray.txt", "x\n")

	out, err := runBinaryInDir(t, bin, dir, "suggest")
	if err != nil {
		t.Fatalf("suggest must exit 0 even with proposals; err=%v out:\n%s", err, out)
	}
	if !strings.Contains(out, "Config additions") {
		t.Errorf("expected text output to include Config additions, got:\n%s", out)
	}
}

func TestSuggest_ExitOneOnOperationalError(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	// No config file anywhere → LoadConfigForContext errors → exit != 0.
	writeTestFile(t, dir, "README.md", "# t\n")

	out, err := runBinaryInDir(t, bin, dir, "suggest")
	if err == nil {
		t.Fatalf("expected non-zero exit on operational error, out:\n%s", out)
	}
}

func TestSuggest_MoveEmitsGitMv(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.sql", "*.md", "*.yaml"]
placement:
  - id: sql-under-migrations
    files: ["*.sql"]
    mustBeUnder: ["migrations/**"]
ignore: [".git"]
`)
	writeTestFile(t, dir, "README.md", "# t\n")
	writeTestFile(t, dir, "stray.sql", "-- x\n")

	r := runSuggestJSON(t, bin, dir)
	found := false
	for _, p := range r.Proposals {
		if p.Kind == "move" {
			found = true
			if !strings.HasPrefix(p.Command, "git mv ") {
				t.Errorf("expected git mv command, got %q", p.Command)
			}
			if p.From != "stray.sql" {
				t.Errorf("expected from=stray.sql, got %q", p.From)
			}
			if !strings.HasPrefix(p.To, "migrations/") {
				t.Errorf("expected to under migrations/, got %q", p.To)
			}
		}
	}
	if !found {
		t.Errorf("expected a move proposal, got proposals=%+v", r.Proposals)
	}
}
