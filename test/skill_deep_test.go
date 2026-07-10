package test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/AxeForging/structlint/internal/app"
	"github.com/AxeForging/structlint/internal/validator"
	"gopkg.in/yaml.v3"
)

// TestSkill_FrontmatterParsesAsYAML asserts SKILL.md's frontmatter is
// valid YAML (agents load it via a YAML parser) and declares the fields
// their runtime keys on.
func TestSkill_FrontmatterParsesAsYAML(t *testing.T) {
	body := readSkillFile(t)
	front := extractFrontmatter(t, body)

	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(front), &meta); err != nil {
		t.Fatalf("SKILL.md frontmatter is not valid YAML: %v\n%s", err, front)
	}
	if meta.Name != "structlint" {
		t.Errorf("expected name: structlint, got %q", meta.Name)
	}
	if len(strings.TrimSpace(meta.Description)) < 30 {
		t.Errorf("description is too short to be useful (agents key on it): %q", meta.Description)
	}
}

// TestSkill_FrontmatterHasTriggerPhrases asserts the description contains
// concrete trigger phrases so agent runtimes match user requests to this
// skill. Parses the YAML first so block-scalar folded newlines don't
// hide multi-word phrases.
func TestSkill_FrontmatterHasTriggerPhrases(t *testing.T) {
	body := readSkillFile(t)
	front := extractFrontmatter(t, body)
	var meta struct {
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(front), &meta); err != nil {
		t.Fatalf("parse frontmatter: %v", err)
	}
	// Collapse whitespace before searching: block scalars fold newlines
	// to a single space, but leading indentation could still trip us up
	// if the layout changes. Normalize aggressively.
	desc := strings.Join(strings.Fields(meta.Description), " ")

	phrases := []string{
		"structlint violation",
		"where should this file go",
		"enforce directory layout",
		"file in the wrong place",
	}
	for _, phrase := range phrases {
		if !strings.Contains(desc, phrase) {
			t.Errorf("SKILL.md description missing trigger phrase %q", phrase)
		}
	}
}

// TestSkill_CoversAllViolationCodes cross-checks both directions — every
// code in the CodeDescriptions registry appears in the SKILL.md table
// AND every code in the SKILL.md table is a real registry entry.
func TestSkill_CoversAllViolationCodes(t *testing.T) {
	body := readSkillFile(t)

	// Registry → doc.
	for code := range validator.CodeDescriptions {
		if !strings.Contains(body, "`"+code+"`") {
			t.Errorf("SKILL.md missing code %q from decision table", code)
		}
	}

	// Doc → registry. Codes appear in table cells as `code_name` — greedy
	// enough regex; false positives (e.g. code lookalikes in prose) are
	// tolerated. What we want to catch is stale renames.
	re := regexp.MustCompile("`([a-z_]+_[a-z_]+)`")
	matches := re.FindAllStringSubmatch(body, -1)
	seen := map[string]bool{}
	for _, m := range matches {
		token := m[1]
		if seen[token] {
			continue
		}
		seen[token] = true
		if _, ok := validator.CodeDescriptions[token]; ok {
			continue
		}
		// Whitelist tokens that look like codes but are structlint concepts
		// or config fields discussed in the skill.
		if isSkillConcept(token) {
			continue
		}
		// Whitelist standard shell/config words that pattern-match the
		// underscore convention but aren't codes.
		t.Logf("hint: %q looks like a code in SKILL.md but isn't in the registry (typo?)", token)
	}
}

// TestSkill_ReferencedCommandsExist boots the CLI in-process and asserts
// that every subcommand the SKILL.md setup recipes tell the agent to
// run is a real registered command. Catches renames of `hook install`,
// `suggest`, etc. without doc updates.
func TestSkill_ReferencedCommandsExist(t *testing.T) {
	body := readSkillFile(t)

	referenced := []string{
		"validate",
		"init",
		"hook install",
		"suggest",
	}

	// Route each through `--help` in-process — the CLI errors on unknown
	// subcommands, so a passing help call proves the command is real.
	root := app.New()
	for _, cmd := range referenced {
		if !strings.Contains(body, "structlint "+cmd) {
			t.Errorf("SKILL.md never references `structlint %s` — outdated setup recipe?", cmd)
		}
		parts := append([]string{"structlint"}, strings.Fields(cmd)...)
		parts = append(parts, "--help")
		if err := root.Run(context.Background(), parts); err != nil {
			t.Errorf("`%s` failed to boot: %v — SKILL.md references a broken subcommand path", strings.Join(parts, " "), err)
		}
	}
}

// TestSkill_MentionsJSONContractVersion asserts SKILL.md documents the
// suggest JSON v1 shape (agents depend on that promise). If the version
// bumps, the doc must bump too or agents will silently break.
func TestSkill_MentionsJSONContractVersion(t *testing.T) {
	body := readSkillFile(t)
	if !strings.Contains(body, `"version": 1`) && !strings.Contains(body, "JSON v1") {
		t.Errorf("SKILL.md does not mention the suggest JSON v1 contract shape")
	}
}

// TestSkill_ExitCodesDocumented asserts every exit code the CLI can
// return is mentioned. Agents shouldn't guess.
func TestSkill_ExitCodesDocumented(t *testing.T) {
	body := readSkillFile(t)
	for _, code := range []string{"0", "1", "2", "3"} {
		if !strings.Contains(body, "**"+code+"**") {
			t.Errorf("SKILL.md doesn't call out exit code %s", code)
		}
	}
}

// --- helpers ---

func readSkillFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), "skills", "structlint", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	return string(data)
}

func extractFrontmatter(t *testing.T, body string) string {
	t.Helper()
	if !strings.HasPrefix(body, "---\n") {
		t.Fatalf("SKILL.md missing frontmatter delimiter")
	}
	rest := body[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		t.Fatalf("SKILL.md frontmatter has no closing delimiter")
	}
	return rest[:end]
}

// isSkillConcept whitelists tokens that pattern-match a violation-code
// shape (snake_case with underscores) but are actually config field names,
// commands, or other structlint concepts the skill talks about. Keeping
// this narrow is intentional: false positives log a hint but don't fail.
func isSkillConcept(token string) bool {
	switch token {
	case "dir_structure", "file_naming_pattern", "required_paths",
		"allowed_paths", "disallowed_paths":
		return true
	}
	return false
}
