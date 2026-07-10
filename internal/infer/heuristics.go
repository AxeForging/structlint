// Package infer generates a baseline .structlint.yaml from the actual
// tree instead of a canned template. Rule of thumb: what we emit must
// make `validate` pass on the same tree. Tightening — removing entries,
// adding `disallowed` — is a later intentional act by the user.
package infer

import (
	"sort"
	"strings"

	"github.com/AxeForging/structlint/internal/validator"
)

// DefaultIgnore is the ignore set applied during the walk. It is also
// emitted in the generated config so what we skipped stays skipped at
// validate time.
var DefaultIgnore = []string{
	".git",
	"node_modules",
	"vendor",
	"dist",
	"build",
	"bin",
}

// AllowedPaths returns "." plus one entry per depth-1 directory —
// `name/**` when the dir has children, bare `name` when empty.
// Files at the root do not produce entries (they're covered by
// file_naming_pattern). Sorted ascending for determinism.
func AllowedPaths(t *validator.Tree) []string {
	depth1Dirs := map[string]bool{}
	nonEmpty := map[string]bool{}
	for _, e := range t.Entries {
		if e.RelPath == "." || e.RelPath == "" {
			continue
		}
		parts := strings.Split(e.RelPath, "/")
		top := parts[0]
		if len(parts) == 1 {
			// Depth-1 entry: only directories count as top-level path entries.
			if e.IsDir {
				depth1Dirs[top] = true
			}
			continue
		}
		// Nested under a depth-1 dir → mark it non-empty.
		depth1Dirs[top] = true
		nonEmpty[top] = true
	}
	out := []string{"."}
	names := make([]string, 0, len(depth1Dirs))
	for name := range depth1Dirs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if nonEmpty[name] {
			out = append(out, name+"/**")
		} else {
			out = append(out, name)
		}
	}
	return out
}

// AllowedFilePatterns returns:
//   - one "*.ext" per unique extension seen anywhere
//   - one exact name per extensionless file (Makefile, LICENSE, .gitignore …)
//
// Both categories are sorted, deduped. Extensions come first, then names.
func AllowedFilePatterns(t *validator.Tree) []string {
	exts := map[string]bool{}
	names := map[string]bool{}
	for _, e := range t.Entries {
		if e.IsDir {
			continue
		}
		// Split on the last dot. A leading dot with no other dots (e.g.
		// ".gitignore") is treated as an extensionless exact name — that
		// matches the user intuition for dotfiles.
		name := e.Name
		lastDot := strings.LastIndex(name, ".")
		if lastDot <= 0 { // 0 → leading dot (dotfile); -1 → no dot at all
			names[name] = true
			continue
		}
		exts["*"+name[lastDot:]] = true
	}
	out := make([]string, 0, len(exts)+len(names))
	extList := mapKeys(exts)
	sort.Strings(extList)
	out = append(out, extList...)
	nameList := mapKeys(names)
	sort.Strings(nameList)
	out = append(out, nameList...)
	return out
}

// RequiredFiles seeds `required` only from certainties present at the
// tree root: go.mod and README.md. Nothing speculative — a wrong entry
// makes the primary "validate passes on the same tree" property fail.
func RequiredFiles(t *validator.Tree) []string {
	var out []string
	for _, candidate := range []string{"go.mod", "README.md"} {
		if t.HasFile(candidate) {
			out = append(out, candidate)
		}
	}
	return out
}

func mapKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
