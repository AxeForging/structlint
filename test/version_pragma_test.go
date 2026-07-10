package test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinaryWithVersion builds a fresh structlint binary with a specific
// build.Version injected via ldflags. Used to prove the version pragma
// path fires end-to-end from a real CLI invocation.
func buildBinaryWithVersion(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "structlint-pinned")
	ldflags := "-X github.com/AxeForging/structlint/internal/build.Version=" + version
	cmd := exec.Command("go", "build", "-o", out, "-ldflags", ldflags, "./cmd/structlint")
	cmd.Dir = repoRoot(t)
	if raw, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build pinned binary: %v\n%s", err, string(raw))
	}
	return out
}

func TestVersionPragma_OldBinaryOnNewConfigFailsHelpfully(t *testing.T) {
	bin := buildBinaryWithVersion(t, "v0.5.0")
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", `# requires structlint >= v0.6.0
dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.md", "*.yaml"]
ignore: [".git"]
`)
	writeTestFile(t, dir, "README.md", "# t\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err == nil {
		t.Fatalf("expected failure: v0.5.0 binary should reject a >= v0.6.0 pragma, out:\n%s", out)
	}
	if !strings.Contains(out, "requires structlint >= v0.6.0") {
		t.Errorf("error should quote the required version, got:\n%s", out)
	}
	if !strings.Contains(out, "running version is v0.5.0") {
		t.Errorf("error should quote the running version, got:\n%s", out)
	}
	if !strings.Contains(out, "go install") {
		t.Errorf("error should suggest an upgrade path, got:\n%s", out)
	}
}

func TestVersionPragma_MatchingBinaryValidatesCleanly(t *testing.T) {
	bin := buildBinaryWithVersion(t, "v0.6.5")
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", `# requires structlint >= v0.6.0
dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.md", "*.yaml"]
ignore: [".git"]
`)
	writeTestFile(t, dir, "README.md", "# t\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err != nil {
		t.Fatalf("expected success: v0.6.5 satisfies >= v0.6.0, err=%v out:\n%s", err, out)
	}
}
