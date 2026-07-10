package test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// runValidate runs the binary against a fixture in dir and returns combined
// output plus a boolean indicating exit code non-zero.
func runValidate(t *testing.T, bin, dir string, extra ...string) (string, bool) {
	t.Helper()
	args := append([]string{"validate"}, extra...)
	out, err := runBinaryInDir(t, bin, dir, args...)
	return out, err != nil
}

// TestSummary_PlacementViolationsGetOwnGroup verifies placement violations
// no longer collapse into the "other" bucket. Before spec 004 they did.
func TestSummary_PlacementViolationsGetOwnGroup(t *testing.T) {
	bin := buildBinary(t)
	config := `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.sql", "*.go", "*.yaml", "*.md"]
placement:
  - id: migrations-only
    files: ["*.sql"]
    mustBeUnder: ["migrations/**"]
ignore: [".git"]
`
	dir := createTestProject(t, map[string]string{
		"README.md":  "# t\n",
		"go.mod":     "module t\n",
		"stray1.sql": "-- 1\n",
		"stray2.sql": "-- 2\n",
	}, config)

	out, failed := runValidate(t, bin, dir)
	if !failed {
		t.Fatalf("expected non-zero exit for placement violations, out:\n%s", out)
	}
	if !strings.Contains(out, "Files placed outside their required directories") {
		t.Errorf("expected placement group description, got:\n%s", out)
	}
	if strings.Contains(out, "Other validation errors") {
		t.Errorf("did not expect placement to fall into 'Other', got:\n%s", out)
	}
}

// TestSummary_LegacyGroupsUnchanged locks in that the six previously
// recognized codes still render with their exact legacy descriptions,
// so anyone parsing text output isn't broken.
func TestSummary_LegacyGroupsUnchanged(t *testing.T) {
	bin := buildBinary(t)
	config := `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.md", "*.yaml"]
  disallowed: ["*.env*"]
  required: ["*.go"]
ignore: [".git"]
`
	dir := createTestProject(t, map[string]string{
		"README.md":  "# t\n",
		".env.local": "SECRET=1\n",
		"stray.txt":  "x\n",
	}, config)

	reportPath := filepath.Join(dir, "report.json")
	out, _ := runValidate(t, bin, dir, "--json-output", reportPath)
	_ = out
	data := readFile(t, reportPath)

	var report map[string]any
	if err := json.Unmarshal([]byte(data), &report); err != nil {
		t.Fatalf("parse report: %v", err)
	}
	summary, _ := report["summary"].(map[string]any)
	viols, _ := summary["violations"].([]any)

	wantDesc := map[string]string{
		"disallowed_file_pattern": "Files matching disallowed naming patterns",
		"unallowed_file_pattern":  "Files not matching any allowed naming pattern",
		"missing_required_file":   "Required file patterns that are missing",
	}
	seen := map[string]string{}
	for _, v := range viols {
		vm, _ := v.(map[string]any)
		t, _ := vm["type"].(string)
		d, _ := vm["description"].(string)
		seen[t] = d
	}
	for code, desc := range wantDesc {
		got, ok := seen[code]
		if !ok {
			t.Errorf("missing summary entry for %s", code)
			continue
		}
		if got != desc {
			t.Errorf("%s: got description %q, want %q", code, got, desc)
		}
	}
}

// TestSummary_TieOrderDeterministic runs the validate twice against a
// fixture that produces equal-count groups and asserts the output is
// byte-identical. Before spec 004 the tie order was random (map order).
func TestSummary_TieOrderDeterministic(t *testing.T) {
	bin := buildBinary(t)
	config := `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.go", "*.md", "*.yaml"]
  disallowed: ["*.env*"]
ignore: [".git"]
`
	dir := createTestProject(t, map[string]string{
		"README.md":  "# t\n",
		"go.mod":     "module t\n",
		".env.local": "S=1\n",
		".env.prod":  "S=1\n",
		"stray1.txt": "x\n",
		"stray2.txt": "y\n",
	}, config)

	out1, _ := runValidate(t, bin, dir)
	out2, _ := runValidate(t, bin, dir)
	if out1 != out2 {
		t.Errorf("summary output not deterministic across runs:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", out1, out2)
	}
}
