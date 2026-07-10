package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type preCommitHook struct {
	ID            string `yaml:"id"`
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	Entry         string `yaml:"entry"`
	Language      string `yaml:"language"`
	PassFilenames bool   `yaml:"pass_filenames"`
	AlwaysRun     bool   `yaml:"always_run"`
}

const expectedEntry = "structlint validate --staged --silent"

func loadPreCommitHooks(t *testing.T) []preCommitHook {
	t.Helper()
	path := filepath.Join(repoRoot(t), ".pre-commit-hooks.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var hooks []preCommitHook
	if err := yaml.Unmarshal(data, &hooks); err != nil {
		t.Fatalf("parse .pre-commit-hooks.yaml: %v", err)
	}
	return hooks
}

func TestPreCommitHooks_ParsesWithRequiredKeys(t *testing.T) {
	hooks := loadPreCommitHooks(t)
	if len(hooks) != 1 {
		t.Fatalf("expected exactly 1 hook, got %d", len(hooks))
	}
	h := hooks[0]
	if h.ID != "structlint" {
		t.Errorf("id: got %q, want structlint", h.ID)
	}
	if h.Language != "golang" {
		t.Errorf("language: got %q, want golang", h.Language)
	}
	if h.PassFilenames {
		t.Errorf("pass_filenames must be false")
	}
	if !h.AlwaysRun {
		t.Errorf("always_run must be true")
	}
	if h.Entry != expectedEntry {
		t.Errorf("entry: got %q, want %q", h.Entry, expectedEntry)
	}
}

func TestPreCommitHooks_EntryFlagsExistOnBinary(t *testing.T) {
	bin := buildBinary(t)
	out, _ := runBinary(t, bin, "validate", "--help")
	hooks := loadPreCommitHooks(t)
	entry := hooks[0].Entry
	// Drift guard: every flag referenced in `entry` must exist on the binary.
	for _, tok := range strings.Fields(entry) {
		if !strings.HasPrefix(tok, "--") {
			continue
		}
		if !strings.Contains(out, tok) {
			t.Errorf("packaged entry references %s but `validate --help` doesn't advertise it:\n%s", tok, out)
		}
	}
}

func TestPreCommitHooks_EntryFailsOnStagedViolation(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeTestFile(t, dir, ".structlint.yaml", stagedTestConfig)
	writeTestFile(t, dir, "README.md", "# test\n")
	writeTestFile(t, dir, ".gitignore", "\n")
	initGitRepo(t, dir)
	gitCommit(t, dir, "initial")

	// Stage a forbidden file — the packaged entry should fail.
	writeTestFile(t, dir, ".env.local", "SECRET=1\n")
	gitStage(t, dir, ".env.local")

	hooks := loadPreCommitHooks(t)
	entryArgs := strings.Fields(hooks[0].Entry)
	// entry starts with "structlint"; the rest are args passed to the binary.
	if len(entryArgs) < 2 || entryArgs[0] != "structlint" {
		t.Fatalf("unexpected entry shape: %v", entryArgs)
	}
	args := entryArgs[1:]

	out, err := runBinaryInDir(t, bin, dir, args...)
	if err == nil {
		t.Fatalf("packaged entry should have failed on staged .env.local, output:\n%s", out)
	}

	// Reset the index and verify success.
	runGit(t, dir, "reset", "HEAD", "--", ".env.local")
	out, err = runBinaryInDir(t, bin, dir, args...)
	if err != nil {
		t.Fatalf("packaged entry should pass after unstaging, err=%v output:\n%s", err, out)
	}
}
