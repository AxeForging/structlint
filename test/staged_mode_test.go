package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const stagedTestConfig = `dir_structure:
  allowedPaths:
    - "."
    - "src/**"
  disallowedPaths:
    - "tmp/**"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.md"
    - "*.yaml"
    - ".gitignore"
  disallowed:
    - "*.env*"
ignore:
  - ".git"
`

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init", "--quiet")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "commit.gpgsign", "false")
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Prevent hooks from firing inside test repos.
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

// gitStage runs `git add` for the given paths.
func gitStage(t *testing.T, dir string, paths ...string) {
	t.Helper()
	args := append([]string{"add", "--"}, paths...)
	runGit(t, dir, args...)
}

// gitCommit stages all and commits with the given message.
func gitCommit(t *testing.T, dir, message string) {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "--quiet", "--allow-empty", "-m", message)
}

func TestStagedMode_CatchesStagedViolation(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeTestFile(t, dir, ".structlint.yaml", stagedTestConfig)
	writeTestFile(t, dir, "README.md", "# test\n")
	writeTestFile(t, dir, ".gitignore", "\n")
	initGitRepo(t, dir)
	gitCommit(t, dir, "initial")

	writeTestFile(t, dir, ".env.local", "SECRET=1\n")
	gitStage(t, dir, ".env.local")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--staged")
	if err == nil {
		t.Fatalf("expected non-zero exit for staged violation, output:\n%s", out)
	}
	if !strings.Contains(out, ".env.local") {
		t.Errorf("expected output to mention .env.local, got:\n%s", out)
	}
}

func TestStagedMode_IgnoresUnstagedViolation(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeTestFile(t, dir, ".structlint.yaml", stagedTestConfig)
	writeTestFile(t, dir, "README.md", "# test\n")
	writeTestFile(t, dir, ".gitignore", "\n")
	initGitRepo(t, dir)
	gitCommit(t, dir, "initial")

	// Working-tree-only file — never staged.
	writeTestFile(t, dir, ".env.local", "SECRET=1\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--staged")
	if err != nil {
		t.Fatalf("expected zero exit (nothing staged), got err=%v output:\n%s", err, out)
	}
}

func TestStagedMode_IgnoresPreExistingDirViolation(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeTestFile(t, dir, ".structlint.yaml", stagedTestConfig)
	writeTestFile(t, dir, "README.md", "# test\n")
	writeTestFile(t, dir, ".gitignore", "\n")
	writeTestFile(t, dir, "src/main.go", "package main\n")
	writeTestFile(t, dir, "tmp/leftover.go", "package tmp\n")
	initGitRepo(t, dir)
	gitCommit(t, dir, "initial")

	// Stage a change to an allowed file only. tmp/ pre-exists and remains
	// disallowed on a full run — but staged mode should ignore it.
	writeTestFile(t, dir, "src/main.go", "package main\n\n// change\n")
	gitStage(t, dir, "src/main.go")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--staged")
	if err != nil {
		t.Fatalf("expected zero exit (pre-existing tmp/ out of scope), got err=%v output:\n%s", err, out)
	}
	if strings.Contains(out, "tmp") {
		t.Errorf("did not expect tmp/ in staged-mode output, got:\n%s", out)
	}
}

func TestStagedMode_NoGitRepoIsGraceful(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeTestFile(t, dir, ".structlint.yaml", stagedTestConfig)
	writeTestFile(t, dir, "README.md", "# test\n")
	writeTestFile(t, dir, ".gitignore", "\n")
	writeTestFile(t, dir, ".env.local", "SECRET=1\n")

	// No .git — git command fails, changed set is empty, staged mode
	// treats every path as out of scope. Should exit 0 without crashing.
	out, err := runBinaryInDir(t, bin, dir, "validate", "--staged")
	if err != nil {
		t.Fatalf("expected zero exit (no git → empty changed set), got err=%v output:\n%s", err, out)
	}
}

func TestChangedOnly_DirFilterAlsoApplies(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeTestFile(t, dir, ".structlint.yaml", stagedTestConfig)
	writeTestFile(t, dir, "README.md", "# test\n")
	writeTestFile(t, dir, ".gitignore", "\n")
	writeTestFile(t, dir, "src/main.go", "package main\n")
	writeTestFile(t, dir, "tmp/leftover.go", "package tmp\n")
	initGitRepo(t, dir)
	gitCommit(t, dir, "initial")

	// Modify an allowed file and commit; then leave a working-tree change
	// on src/main.go so `git diff HEAD` returns just that file.
	writeTestFile(t, dir, "src/main.go", "package main\n// v2\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--changed-only")
	if err != nil {
		t.Fatalf("expected zero exit for --changed-only after dir filter fix, got err=%v output:\n%s", err, out)
	}
	if strings.Contains(out, "tmp") {
		t.Errorf("did not expect tmp/ in --changed-only output, got:\n%s", out)
	}
}

// pathExists is used to sanity-check test setups.
func pathExists(t *testing.T, p string) bool {
	t.Helper()
	_, err := os.Stat(p)
	return err == nil
}

func TestStagedMode_SetupSanity(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.txt", "x")
	initGitRepo(t, dir)
	if !pathExists(t, filepath.Join(dir, ".git")) {
		t.Fatal("expected .git after initGitRepo")
	}
}
