package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtends_StringForm(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	// Minimal Go project matching go-standard preset.
	writeTestFile(t, dir, ".structlint.yaml", "extends: go-standard\n")
	writeTestFile(t, dir, "go.mod", "module t\n")
	writeTestFile(t, dir, "README.md", "# t\n")
	writeTestFile(t, dir, ".gitignore", "bin/\n")
	writeTestFile(t, dir, "cmd/app/main.go", "package main\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err != nil {
		t.Fatalf("extends: go-standard failed: err=%v out:\n%s", err, out)
	}
}

func TestExtends_PresetPlusOverride(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	// Extend go-standard and add "tools/**" as a new allowedPath.
	writeTestFile(t, dir, ".structlint.yaml", `extends: go-standard
dir_structure:
  allowedPaths:
    - "tools/**"
`)
	writeTestFile(t, dir, "go.mod", "module t\n")
	writeTestFile(t, dir, "README.md", "# t\n")
	writeTestFile(t, dir, ".gitignore", "bin/\n")
	writeTestFile(t, dir, "cmd/app/main.go", "package main\n")
	writeTestFile(t, dir, "tools/gen/main.go", "package gen\n") // only allowed via override

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err != nil {
		t.Fatalf("preset+override should pass, err=%v out:\n%s", err, out)
	}

	// A directory allowed by neither still fails.
	writeTestFile(t, dir, "other/x.go", "package other\n")
	out, err = runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err == nil {
		t.Fatalf("dir allowed by neither preset nor child should fail, out:\n%s", out)
	}
}

func TestExtends_RelativePath(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// Base at repo root.
	writeTestFile(t, dir, "base.yaml", `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.md", "*.yaml"]
ignore: [".git"]
`)
	// Extending file lives in sub/.
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, sub, ".structlint.yaml", "extends: ../base.yaml\n")
	writeTestFile(t, sub, "notes.md", "# notes\n")

	// Run from a different cwd (dir), pointing --path at sub/.
	out, err := runBinaryInDir(t, bin, dir, "validate", "--path", "sub", "--config", "sub/.structlint.yaml", "--silent")
	if err != nil {
		t.Fatalf("relative extends failed: err=%v out:\n%s", err, out)
	}
}

func TestExtends_ChildReplacesRuleByID(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeTestFile(t, dir, "parent.yaml", `dir_structure:
  allowedPaths: [".", "parent-only/**", "migrations/**"]
file_naming_pattern:
  allowed: ["*.sql", "*.md", "*.yaml", "*.go"]
placement:
  - id: sql-under-parent
    files: ["*.sql"]
    mustBeUnder: ["parent-only/**"]
ignore: [".git"]
`)
	// Child replaces rule id "sql-under-parent" with a different mustBeUnder.
	writeTestFile(t, dir, ".structlint.yaml", `extends: parent.yaml
placement:
  - id: sql-under-parent
    files: ["*.sql"]
    mustBeUnder: ["migrations/**"]
`)
	writeTestFile(t, dir, "migrations/001.sql", "-- ok\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err != nil {
		t.Fatalf("child-replaced rule should accept migrations/, err=%v out:\n%s", err, out)
	}

	// Adding an SQL file under parent-only/ should now be rejected (child rule wins).
	writeTestFile(t, dir, "parent-only/2.sql", "-- x\n")
	out, err = runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err == nil {
		t.Fatalf("parent-only .sql should be rejected under child rule, out:\n%s", out)
	}
}

func TestExtends_ListMergeOrder(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeTestFile(t, dir, "a.yaml", `dir_structure:
  allowedPaths: [".", "shared/**", "a-only/**"]
file_naming_pattern:
  allowed: ["*.md", "*.yaml"]
ignore: [".git"]
`)
	writeTestFile(t, dir, "b.yaml", `dir_structure:
  allowedPaths: ["shared/**", "b-only/**"]
file_naming_pattern:
  allowed: ["*.yaml", "*.go"]
`)
	writeTestFile(t, dir, ".structlint.yaml", `extends: [a.yaml, b.yaml]
`)
	writeTestFile(t, dir, "a-only/x.md", "x\n")
	writeTestFile(t, dir, "b-only/y.go", "package y\n")
	writeTestFile(t, dir, "shared/z.md", "z\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err != nil {
		t.Fatalf("list-merge should union a/b/shared: err=%v out:\n%s", err, out)
	}
}

func TestExtends_CycleDetected(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	writeTestFile(t, dir, "a.yaml", "extends: b.yaml\n")
	writeTestFile(t, dir, "b.yaml", "extends: a.yaml\n")
	writeTestFile(t, dir, ".structlint.yaml", "extends: a.yaml\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err == nil {
		t.Fatalf("expected cycle error, out:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "cycle") {
		t.Errorf("expected 'cycle' in error, got:\n%s", out)
	}
}

func TestExtends_DepthCapExceeded(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// Chain of 15 files, each extending the next.
	for i := 0; i < 15; i++ {
		content := ""
		if i < 14 {
			content = "extends: " + "chain" + string(rune('0'+((i+1)/10))) + string(rune('0'+((i+1)%10))) + ".yaml\n"
		}
		writeTestFile(t, dir, "chain"+string(rune('0'+(i/10)))+string(rune('0'+(i%10)))+".yaml", content)
	}
	writeTestFile(t, dir, ".structlint.yaml", "extends: chain00.yaml\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err == nil {
		t.Fatalf("expected depth error, out:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "too deep") {
		t.Errorf("expected 'too deep' in error, got:\n%s", out)
	}
}

func TestExtends_UnknownPresetErrors(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", "extends: rust-standard\n")

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err == nil {
		t.Fatalf("expected unknown-preset error, out:\n%s", out)
	}
	if !strings.Contains(out, "rust-standard") {
		t.Errorf("error should name the entry, got:\n%s", out)
	}
	if !strings.Contains(out, "go-standard") {
		t.Errorf("error should list valid presets, got:\n%s", out)
	}
}

func TestExtends_UnknownKeyStillStrict(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeTestFile(t, dir, ".structlint.yaml", `extends: go-standard
placment:
  - id: typo
`)

	out, err := runBinaryInDir(t, bin, dir, "validate", "--silent")
	if err == nil {
		t.Fatalf("strict parse should reject 'placment' typo even with extends, out:\n%s", out)
	}
}

func TestExtends_InitNeverEmitsExtends(t *testing.T) {
	bin := buildBinary(t)
	for _, kind := range []string{"go", "node", "python", "generic"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			out, err := runBinaryInDir(t, bin, dir, "init", "--type", kind)
			if err != nil {
				t.Fatalf("init --type %s failed: %v\n%s", kind, err, out)
			}
			got := readFile(t, filepath.Join(dir, ".structlint.yaml"))
			if strings.Contains(got, "extends:") {
				t.Errorf("init --type %s emitted 'extends:', which is forbidden:\n%s", kind, got)
			}
		})
	}
}
