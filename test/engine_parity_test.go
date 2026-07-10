package test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// parityFixtures returns the fixture names under test/testdata/parity/,
// excluding non-fixture entries.
func parityFixtures(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "test", "testdata", "parity")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read parity dir: %v", err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "goldens" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), ".structlint.yaml")); err != nil {
			continue
		}
		out = append(out, e.Name())
	}
	return out
}

// runFixture invokes the binary against a fixture and returns stdout+stderr
// combined and the exit code. Uses combined output because the pre-refactor
// binary intermixes summary text and error lines; goldens capture that mix.
func runFixture(t *testing.T, bin, name string, extra ...string) (string, int) {
	t.Helper()
	root := repoRoot(t)
	fixture := filepath.Join(root, "test", "testdata", "parity", name)
	config := filepath.Join(fixture, ".structlint.yaml")
	args := []string{"validate", "--path", fixture, "--config", config}
	args = append(args, extra...)
	out, err := runBinary(t, bin, args...)
	code := 0
	if err != nil {
		code = 1 // urfave/cli returns 1 on Action error; goldens confirm this
	}
	return out, code
}

func readGolden(t *testing.T, kind, name string) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), "test", "testdata", "parity", "goldens", name+"."+kind)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	return string(data)
}

func TestEngineParity_TextOutput(t *testing.T) {
	bin := buildBinary(t)
	for _, name := range parityFixtures(t) {
		t.Run(name, func(t *testing.T) {
			got, _ := runFixture(t, bin, name)
			want := readGolden(t, "text", name)
			if got != want {
				t.Errorf("text output drift for fixture %q\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
			}
		})
	}
}

func TestEngineParity_JSONOutput(t *testing.T) {
	bin := buildBinary(t)
	for _, name := range parityFixtures(t) {
		t.Run(name, func(t *testing.T) {
			got, _ := runFixture(t, bin, name, "--format", "json")
			want := readGolden(t, "json", name)
			if got != want {
				t.Errorf("json output drift for fixture %q\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
			}
		})
	}
}

func TestEngineParity_ExitCodes(t *testing.T) {
	bin := buildBinary(t)
	for _, name := range parityFixtures(t) {
		t.Run(name, func(t *testing.T) {
			_, code := runFixture(t, bin, name)
			want := strings.TrimSpace(readGolden(t, "exit", name))
			wantCode, err := strconv.Atoi(want)
			if err != nil {
				t.Fatalf("bad exit golden: %v", err)
			}
			if code != wantCode {
				t.Errorf("exit code drift for %q: got %d, want %d", name, code, wantCode)
			}
		})
	}
}
