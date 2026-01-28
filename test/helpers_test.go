package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

var (
	testBinaryPath string
	buildOnce      sync.Once
	buildErr       error
)

// buildBinary compiles the structlint binary once and returns its path.
// Uses sync.Once to ensure we only build once per test run.
func buildBinary(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		tmpDir := os.TempDir()
		binName := "structlint-test"
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		testBinaryPath = filepath.Join(tmpDir, binName)

		cmd := exec.Command("go", "build", "-o", testBinaryPath, "./cmd/structlint")
		cmd.Dir = repoRoot(t)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = err
			t.Logf("build output: %s", string(out))
		}
	})

	if buildErr != nil {
		t.Fatalf("failed to build binary: %v", buildErr)
	}

	return testBinaryPath
}

// repoRoot returns the project root directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	// test files are in repo/test, so go up one level
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(wd)
}

// runBinary executes the structlint binary with the given arguments.
func runBinary(t *testing.T, bin string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runBinaryInDir executes the structlint binary in a specific directory.
func runBinaryInDir(t *testing.T, bin, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// writeTestFile creates a file with the given content in a directory.
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

// createTestProject sets up a temporary project with files and config.
func createTestProject(t *testing.T, files map[string]string, configContent string) string {
	t.Helper()
	tmpDir := t.TempDir()

	for path, content := range files {
		writeTestFile(t, tmpDir, path, content)
	}

	if configContent != "" {
		writeTestFile(t, tmpDir, ".structlint.yaml", configContent)
	}

	return tmpDir
}
