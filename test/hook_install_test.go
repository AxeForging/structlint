package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const existingLefthook = `# user's own config, keep me
pre-commit:
  parallel: true
  commands:
    gofmt:
      run: gofmt -w {staged_files}
      stage_fixed: true
`

const existingPreCommit = `# managed by the team; keep comments
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
`

const existingPreCommitHook = `#!/bin/sh
# user's own hook
echo "running my checks"
`

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

func TestHookInstall_Lefthook_FreshFile(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// lefthook.yml doesn't exist yet, but explicit --type takes over auto-detection
	out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "lefthook")
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	got := readFile(t, filepath.Join(dir, "lefthook.yml"))
	if !strings.Contains(got, "structlint:") {
		t.Errorf("expected structlint entry, got:\n%s", got)
	}
	if !strings.Contains(got, "structlint validate --staged --silent") {
		t.Errorf("expected staged validate command, got:\n%s", got)
	}
}

func TestHookInstall_Lefthook_ExistingWithoutStructlint(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, "lefthook.yml", existingLefthook)

	out, err := runBinaryInDir(t, bin, dir, "hook", "install")
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	got := readFile(t, filepath.Join(dir, "lefthook.yml"))
	// User content preserved
	if !strings.Contains(got, "gofmt:") {
		t.Errorf("existing gofmt entry lost:\n%s", got)
	}
	if !strings.Contains(got, "parallel: true") {
		t.Errorf("existing parallel key lost:\n%s", got)
	}
	if !strings.Contains(got, "structlint:") {
		t.Errorf("structlint entry not added:\n%s", got)
	}
}

func TestHookInstall_Lefthook_Idempotent(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, "lefthook.yml", existingLefthook)

	if out, err := runBinaryInDir(t, bin, dir, "hook", "install"); err != nil {
		t.Fatalf("first install failed: %v\n%s", err, out)
	}
	first := readFile(t, filepath.Join(dir, "lefthook.yml"))

	out2, err := runBinaryInDir(t, bin, dir, "hook", "install")
	if err != nil {
		t.Fatalf("second install failed: %v\n%s", err, out2)
	}
	if !strings.Contains(out2, "already installed") {
		t.Errorf("expected 'already installed' output, got:\n%s", out2)
	}
	second := readFile(t, filepath.Join(dir, "lefthook.yml"))
	if first != second {
		t.Errorf("file changed on second run (not idempotent).\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestHookInstall_Lefthook_DryRun(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, "lefthook.yml", existingLefthook)
	before := readFile(t, filepath.Join(dir, "lefthook.yml"))

	out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "structlint:") {
		t.Errorf("dry-run preview missing structlint entry:\n%s", out)
	}
	after := readFile(t, filepath.Join(dir, "lefthook.yml"))
	if before != after {
		t.Errorf("dry-run modified file on disk:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestHookInstall_Lefthook_RefusesOnAnchors(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, "lefthook.yml", `defaults: &defaults
  parallel: true

pre-commit:
  <<: *defaults
  commands:
    gofmt:
      run: gofmt -w {staged_files}
`)
	before := readFile(t, filepath.Join(dir, "lefthook.yml"))

	out, err := runBinaryInDir(t, bin, dir, "hook", "install")
	if err == nil {
		t.Fatalf("expected refusal to exit non-zero, output:\n%s", out)
	}
	if !strings.Contains(out, "anchors/aliases") {
		t.Errorf("expected refusal reason mentioning anchors/aliases, got:\n%s", out)
	}
	after := readFile(t, filepath.Join(dir, "lefthook.yml"))
	if before != after {
		t.Errorf("refusal path modified file:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestHookInstall_PreCommit_FreshFile(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "pre-commit")
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	got := readFile(t, filepath.Join(dir, ".pre-commit-config.yaml"))
	if !strings.Contains(got, "AxeForging/structlint") {
		t.Errorf("expected structlint repo entry, got:\n%s", got)
	}
	if !strings.Contains(got, "id: structlint") {
		t.Errorf("expected id: structlint, got:\n%s", got)
	}
}

func TestHookInstall_PreCommit_ExistingWithoutStructlint(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, ".pre-commit-config.yaml", existingPreCommit)

	out, err := runBinaryInDir(t, bin, dir, "hook", "install")
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	got := readFile(t, filepath.Join(dir, ".pre-commit-config.yaml"))
	if !strings.Contains(got, "trailing-whitespace") {
		t.Errorf("existing trailing-whitespace hook lost:\n%s", got)
	}
	if !strings.Contains(got, "AxeForging/structlint") {
		t.Errorf("structlint entry not added:\n%s", got)
	}
}

func TestHookInstall_PreCommit_Idempotent(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	if out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "pre-commit"); err != nil {
		t.Fatalf("first install failed: %v\n%s", err, out)
	}
	first := readFile(t, filepath.Join(dir, ".pre-commit-config.yaml"))
	out2, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "pre-commit")
	if err != nil {
		t.Fatalf("second install failed: %v\n%s", err, out2)
	}
	if !strings.Contains(out2, "already installed") {
		t.Errorf("expected 'already installed' output, got:\n%s", out2)
	}
	second := readFile(t, filepath.Join(dir, ".pre-commit-config.yaml"))
	if first != second {
		t.Errorf("file changed on second run.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestHookInstall_Git_FreshFile(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "git")
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	got := readFile(t, hookPath)
	if !strings.Contains(got, "structlint validate --staged --silent") {
		t.Errorf("expected staged validate command in hook, got:\n%s", got)
	}
	if !strings.Contains(got, "structlint hook >>>") {
		t.Errorf("expected marker block, got:\n%s", got)
	}
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("expected pre-commit hook to be executable, got mode %v", info.Mode())
	}
}

func TestHookInstall_Git_ExistingWithoutMarkers(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Pre-existing user hook without structlint markers.
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte(existingPreCommitHook), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "git")
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	got := readFile(t, hookPath)
	if !strings.Contains(got, `echo "running my checks"`) {
		t.Errorf("user content lost:\n%s", got)
	}
	if !strings.Contains(got, "structlint hook >>>") {
		t.Errorf("marker block missing:\n%s", got)
	}
}

func TestHookInstall_Git_Idempotent(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	if out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "git"); err != nil {
		t.Fatalf("first install failed: %v\n%s", err, out)
	}
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	first := readFile(t, hookPath)

	out2, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "git")
	if err != nil {
		t.Fatalf("second install failed: %v\n%s", err, out2)
	}
	if !strings.Contains(out2, "already installed") {
		t.Errorf("expected 'already installed' output, got:\n%s", out2)
	}
	second := readFile(t, hookPath)
	if first != second {
		t.Errorf("hook changed on second run.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestHookInstall_Git_ReplacesOutdatedMarkerBlock(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	outdated := `#!/bin/sh
# my hook
echo "before"

# >>> structlint hook >>>
# outdated block from an older install
structlint validate || exit 1
# <<< structlint hook <<<

echo "after"
`
	if err := os.WriteFile(hookPath, []byte(outdated), 0o755); err != nil {
		t.Fatal(err)
	}

	if out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "git"); err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	got := readFile(t, hookPath)
	if strings.Contains(got, "outdated block from an older install") {
		t.Errorf("outdated block was not replaced:\n%s", got)
	}
	if !strings.Contains(got, "structlint validate --staged --silent") {
		t.Errorf("current invocation missing:\n%s", got)
	}
	// Content outside markers preserved.
	if !strings.Contains(got, `echo "before"`) || !strings.Contains(got, `echo "after"`) {
		t.Errorf("content outside markers lost:\n%s", got)
	}
}

func TestHookInstall_Git_NoRepo(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--type", "git")
	if err == nil {
		t.Fatalf("expected error when not in a git repo, output:\n%s", out)
	}
	if !strings.Contains(out, "not a git repository") && !strings.Contains(out, "git init") {
		t.Errorf("expected helpful error mentioning git init, got:\n%s", out)
	}
}

func TestHookInstall_AutoDetect(t *testing.T) {
	bin := buildBinary(t)

	cases := []struct {
		name  string
		files map[string]string
		want  string // substring in output identifying the type
	}{
		{
			name:  "lefthook_wins",
			files: map[string]string{"lefthook.yml": "pre-commit:\n  commands: {}\n", ".pre-commit-config.yaml": "repos: []\n"},
			want:  "lefthook.yml",
		},
		{
			name:  "precommit_when_no_lefthook",
			files: map[string]string{".pre-commit-config.yaml": "repos: []\n"},
			want:  ".pre-commit-config.yaml",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tc.files {
				writeTestFile(t, dir, name, content)
			}
			out, err := runBinaryInDir(t, bin, dir, "hook", "install", "--dry-run")
			if err != nil {
				t.Fatalf("install failed: %v\n%s", err, out)
			}
			if !strings.Contains(out, tc.want) {
				t.Errorf("expected output to mention %q, got:\n%s", tc.want, out)
			}
		})
	}
}
