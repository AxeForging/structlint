package test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/AxeForging/structlint/internal/validator"
)

// TestViolationCodesDoc_CoversRegistry is the keystone: every code in the
// canonical CodeDescriptions registry must have a heading in the frozen
// violation-codes doc. This locks the append-only contract that spec 011
// declares (agents key on codes; codes are documented; no drift).
func TestViolationCodesDoc_CoversRegistry(t *testing.T) {
	docPath := filepath.Join(repoRoot(t), "docs", "user", "violation-codes.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read violation-codes.md: %v", err)
	}
	body := string(data)
	for code := range validator.CodeDescriptions {
		needle := "`" + code + "`"
		if !strings.Contains(body, needle) {
			t.Errorf("violation-codes.md missing entry for code %q", code)
		}
	}
}

// TestViolationCodesDoc_NoUnknownCodes asserts that every code-formatted
// entry in the doc corresponds to a real registered code. Catches typos
// and stale renames in the doc itself.
func TestViolationCodesDoc_NoUnknownCodes(t *testing.T) {
	docPath := filepath.Join(repoRoot(t), "docs", "user", "violation-codes.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Heading form we use in the doc: `### \`code\``.
	re := regexp.MustCompile("### `([a-z_]+)`")
	matches := re.FindAllStringSubmatch(string(data), -1)
	if len(matches) == 0 {
		t.Fatal("no code headings found in violation-codes.md — regex mismatch or empty doc")
	}
	for _, m := range matches {
		code := m[1]
		if _, ok := validator.CodeDescriptions[code]; !ok {
			t.Errorf("violation-codes.md documents unknown code %q (not in CodeDescriptions registry)", code)
		}
	}
}

// TestSkillFile_ExistsWithFrontmatter asserts the shipped skill file exists
// with the frontmatter fields agent runtimes look for.
func TestSkillFile_ExistsWithFrontmatter(t *testing.T) {
	path := filepath.Join(repoRoot(t), "skills", "structlint", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	body := string(data)
	if !strings.HasPrefix(body, "---\n") {
		t.Errorf("SKILL.md must start with a frontmatter block (---); got: %q", body[:min(64, len(body))])
	}
	if !strings.Contains(body, "name: structlint") {
		t.Errorf("SKILL.md frontmatter must declare name: structlint")
	}
	if !strings.Contains(body, "description:") {
		t.Errorf("SKILL.md frontmatter must declare a description")
	}
}

// TestSkillFile_MentionsAllCodes asserts SKILL.md's decision table covers
// every registry code so agents have complete fix-or-config guidance.
func TestSkillFile_MentionsAllCodes(t *testing.T) {
	path := filepath.Join(repoRoot(t), "skills", "structlint", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(data)
	for code := range validator.CodeDescriptions {
		if !strings.Contains(body, "`"+code+"`") {
			t.Errorf("SKILL.md missing code %q from decision table", code)
		}
	}
}

// TestSelfValidation_AllowsSkillsDir builds the binary and validates the
// repo root, proving skills/** landed in .structlint.yaml in the same PR
// as the new directory (roadmap risk 5).
func TestSelfValidation_AllowsSkillsDir(t *testing.T) {
	bin := buildBinary(t)
	out, err := runBinaryInDir(t, bin, repoRoot(t), "validate", "--silent")
	if err != nil {
		t.Fatalf("self-validate must pass with skills/ present; err=%v out:\n%s", err, out)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
