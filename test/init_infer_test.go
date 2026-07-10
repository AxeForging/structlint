package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildInferProject writes the given files into a tmp dir and returns the dir.
func buildInferProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		writeTestFile(t, dir, path, content)
	}
	return dir
}

func TestInitInfer_ValidatePassesOnSameTree(t *testing.T) {
	bin := buildBinary(t)
	trees := map[string]map[string]string{
		"go-like": {
			"go.mod":            "module t\n",
			"README.md":         "# t\n",
			"cmd/app/main.go":   "package main\n",
			"internal/foo/x.go": "package foo\n",
		},
		"node-like": {
			"package.json":        `{"name":"t"}`,
			"README.md":           "# t\n",
			"src/index.ts":        "export {}\n",
			"src/util/a.ts":       "export {}\n",
			"tests/index.test.ts": "test('x', () => {})\n",
		},
		"mixed": {
			"README.md":        "# t\n",
			"Makefile":         "all:\n\t@echo\n",
			"scripts/run.sh":   "#!/bin/sh\n",
			"data/samples.csv": "a,b\n",
			"docs/intro.md":    "# intro\n",
		},
	}
	for name, files := range trees {
		t.Run(name, func(t *testing.T) {
			dir := buildInferProject(t, files)
			out, err := runBinaryInDir(t, bin, dir, "init", "--infer")
			if err != nil {
				t.Fatalf("init --infer failed: err=%v out:\n%s", err, out)
			}
			out, err = runBinaryInDir(t, bin, dir, "validate", "--silent")
			if err != nil {
				t.Fatalf("validate should pass on the inferred tree, err=%v out:\n%s", err, out)
			}
		})
	}
}

func TestInitInfer_Depth1DirsBecomeGlobs(t *testing.T) {
	bin := buildBinary(t)
	dir := buildInferProject(t, map[string]string{
		"README.md":       "# t\n",
		"internal/x/x.go": "package x\n",
	})
	// Create an empty leaf dir (mkdir only).
	if err := writeEmptyDir(t, dir, "empty-leaf"); err != nil {
		t.Fatal(err)
	}
	if out, err := runBinaryInDir(t, bin, dir, "init", "--infer"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	got := readFile(t, filepath.Join(dir, ".structlint.yaml"))
	if !strings.Contains(got, `"internal/**"`) {
		t.Errorf("expected internal/** in generated config:\n%s", got)
	}
	if !strings.Contains(got, `"empty-leaf"`) {
		t.Errorf("expected bare empty-leaf in generated config:\n%s", got)
	}
	if strings.Contains(got, `"empty-leaf/**"`) {
		t.Errorf("empty-leaf should NOT have /**:\n%s", got)
	}
}

func TestInitInfer_ExtensionsAndExactNames(t *testing.T) {
	bin := buildBinary(t)
	dir := buildInferProject(t, map[string]string{
		"README.md":  "# t\n",
		"main.go":    "package main\n",
		"Makefile":   "all:\n\t@echo\n",
		".gitignore": "bin/\n",
	})
	if out, err := runBinaryInDir(t, bin, dir, "init", "--infer"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	got := readFile(t, filepath.Join(dir, ".structlint.yaml"))
	for _, want := range []string{`"*.go"`, `"*.md"`, `"Makefile"`, `".gitignore"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in inferred config:\n%s", want, got)
		}
	}
}

func TestInitInfer_RequiredOnlyFromCertainties(t *testing.T) {
	bin := buildBinary(t)
	t.Run("both present", func(t *testing.T) {
		dir := buildInferProject(t, map[string]string{
			"README.md": "# t\n",
			"go.mod":    "module t\n",
		})
		if out, err := runBinaryInDir(t, bin, dir, "init", "--infer"); err != nil {
			t.Fatalf("init: %v\n%s", err, out)
		}
		got := readFile(t, filepath.Join(dir, ".structlint.yaml"))
		if !strings.Contains(got, `"go.mod"`) || !strings.Contains(got, `"README.md"`) {
			t.Errorf("expected go.mod and README.md required, got:\n%s", got)
		}
	})
	t.Run("neither present", func(t *testing.T) {
		dir := buildInferProject(t, map[string]string{
			"main.go": "package main\n",
		})
		if out, err := runBinaryInDir(t, bin, dir, "init", "--infer"); err != nil {
			t.Fatalf("init: %v\n%s", err, out)
		}
		got := readFile(t, filepath.Join(dir, ".structlint.yaml"))
		if strings.Contains(got, "required:") {
			t.Errorf("did not expect a required: section with no certainties, got:\n%s", got)
		}
	})
}

func TestInitInfer_MutuallyExclusiveWithType(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	out, err := runBinaryInDir(t, bin, dir, "init", "--infer", "--type", "go")
	if err == nil {
		t.Fatalf("expected error when both --infer and --type used, out:\n%s", out)
	}
	if !strings.Contains(out, "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got:\n%s", out)
	}
}

func TestInitInfer_RespectsForceGuard(t *testing.T) {
	bin := buildBinary(t)
	dir := buildInferProject(t, map[string]string{
		".structlint.yaml": "dir_structure:\n  allowedPaths: [\".\"]\n",
		"README.md":        "# t\n",
	})
	out, err := runBinaryInDir(t, bin, dir, "init", "--infer")
	if err == nil {
		t.Fatalf("expected error when config exists and --force not passed, out:\n%s", out)
	}
	if !strings.Contains(out, "already exists") {
		t.Errorf("expected 'already exists' in error, got:\n%s", out)
	}
	out, err = runBinaryInDir(t, bin, dir, "init", "--infer", "--force")
	if err != nil {
		t.Fatalf("--force should overwrite: %v\n%s", err, out)
	}
}

func TestInitInfer_Deterministic(t *testing.T) {
	bin := buildBinary(t)
	dir := buildInferProject(t, map[string]string{
		"README.md":       "# t\n",
		"cmd/app/main.go": "package main\n",
		"internal/x.go":   "package internal\n",
		"docs/intro.md":   "# intro\n",
	})
	if out, err := runBinaryInDir(t, bin, dir, "init", "--infer"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	first := readFile(t, filepath.Join(dir, ".structlint.yaml"))
	if out, err := runBinaryInDir(t, bin, dir, "init", "--infer", "--force"); err != nil {
		t.Fatalf("init2: %v\n%s", err, out)
	}
	second := readFile(t, filepath.Join(dir, ".structlint.yaml"))
	if first != second {
		t.Errorf("expected byte-identical output across runs.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func writeEmptyDir(t *testing.T, root, name string) error {
	t.Helper()
	return os.MkdirAll(filepath.Join(root, name), 0o755)
}
