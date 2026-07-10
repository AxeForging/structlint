package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitTypeUsesPresetBody asserts that `init --type <T>` writes a file
// whose body (below the header comment) is byte-identical to the embedded
// preset that `extends: <T>-standard` (or `generic`) resolves to.
// This locks in the deduplication of spec 007's presets and init's
// templates — they were duplicated string literals before this refactor.
func TestInitTypeUsesPresetBody(t *testing.T) {
	bin := buildBinary(t)

	cases := map[string]string{
		"go":      "go-standard.yaml",
		"node":    "node-standard.yaml",
		"python":  "python-standard.yaml",
		"generic": "generic.yaml",
	}

	for typ, presetFile := range cases {
		t.Run(typ, func(t *testing.T) {
			dir := t.TempDir()
			if out, err := runBinaryInDir(t, bin, dir, "init", "--type", typ); err != nil {
				t.Fatalf("init --type %s: %v\n%s", typ, err, out)
			}
			generated := readFile(t, filepath.Join(dir, ".structlint.yaml"))

			// Strip the first line (the header comment) — the rest must
			// match the preset byte-for-byte.
			nl := strings.Index(generated, "\n")
			if nl < 0 {
				t.Fatalf("generated config has no newline: %q", generated)
			}
			body := generated[nl+1:]

			presetPath := filepath.Join(repoRoot(t), "internal", "config", "presets", presetFile)
			presetData, err := os.ReadFile(presetPath)
			if err != nil {
				t.Fatalf("read preset %s: %v", presetPath, err)
			}
			if body != string(presetData) {
				t.Errorf("init --type %s body drifted from preset %s\n--- got body ---\n%s\n--- preset ---\n%s",
					typ, presetFile, body, string(presetData))
			}
		})
	}
}

// TestInitTypeAndExtendsProduceEquivalentValidation is the semantic
// twin of the byte test above: init --type <T> and a config with
// `extends: <T>-standard` should validate the same tree the same way.
func TestInitTypeAndExtendsProduceEquivalentValidation(t *testing.T) {
	bin := buildBinary(t)

	// Minimal go-shaped project the go-standard preset would pass on.
	files := map[string]string{
		"go.mod":            "module t\n",
		"README.md":         "# t\n",
		".gitignore":        "bin/\n",
		"cmd/app/main.go":   "package main\n\nfunc main() {}\n",
		"internal/foo/x.go": "package foo\n",
	}

	// init --type go path.
	dirA := t.TempDir()
	for p, content := range files {
		writeTestFile(t, dirA, p, content)
	}
	if out, err := runBinaryInDir(t, bin, dirA, "init", "--type", "go", "--force"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	outA, errA := runBinaryInDir(t, bin, dirA, "validate", "--silent")

	// extends: go-standard path.
	dirB := t.TempDir()
	for p, content := range files {
		writeTestFile(t, dirB, p, content)
	}
	writeTestFile(t, dirB, ".structlint.yaml", "extends: go-standard\n")
	outB, errB := runBinaryInDir(t, bin, dirB, "validate", "--silent")

	if (errA == nil) != (errB == nil) {
		t.Errorf("validation outcomes diverge:\n  init --type go: err=%v\n  extends: go-standard: err=%v\n\n  outA=%s\n  outB=%s",
			errA, errB, outA, outB)
	}
}
