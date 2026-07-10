package suggest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// buildConfigDiff produces a unified diff that, applied against the
// original config text, adds the values from every config_add proposal
// under the correct section. We NEVER re-marshal the YAML: re-marshal
// would destroy comments, ordering, and quoting, making the diff
// unappliable to the user's real file.
//
// The strategy is line-based:
//  1. Read the config file.
//  2. For each unique (section, value), locate the section's list in the
//     text and insert `- "value"` after the last existing item at the
//     same indent.
//  3. Emit a `diff -u` between original and modified text.
//
// Sections missing from the file get appended at the end.
func buildConfigDiff(configPath string, proposals []Proposal) (string, error) {
	adds := groupBySection(proposals)
	if len(adds) == 0 {
		return "", nil
	}
	origBytes, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read config for diff: %w", err)
	}
	orig := string(origBytes)
	modified := orig
	for section, values := range adds {
		modified = insertUnderSection(modified, section, values)
	}
	if modified == orig {
		return "", nil
	}
	return unifiedDiff(configPath, orig, modified), nil
}

func groupBySection(proposals []Proposal) map[string][]string {
	out := map[string][]string{}
	for _, p := range proposals {
		if p.Kind != KindConfigAdd || p.Value == "" || p.Section == "" {
			continue
		}
		out[p.Section] = append(out[p.Section], p.Value)
	}
	return out
}

// sectionListStart matches a YAML key at the start of a line, at any indent.
// We use it to locate the beginning of a nested list under a header.
var listItemRE = regexp.MustCompile(`^(\s*)- `)

// insertUnderSection appends `- "value"` lines under the given nested key
// path (e.g. "dir_structure.allowedPaths"). It looks for the innermost key
// as a line ending in `:` at some indent, then walks forward while lines
// are `- ` items at a deeper indent. Values are inserted after the last
// existing item at the same indent.
func insertUnderSection(text, section string, values []string) string {
	parts := strings.Split(section, ".")
	if len(parts) == 0 {
		return text
	}
	target := parts[len(parts)-1]
	lines := strings.Split(text, "\n")
	// Find the "target:" line. We accept any depth match; if the config has
	// two headers with the same leaf name we take the first (users rarely
	// duplicate section names, and this fallback is documented).
	targetIdx := -1
	for i, line := range lines {
		if strings.HasSuffix(strings.TrimRight(line, " \t"), target+":") {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		// Append at end.
		return text + fmt.Sprintf("\n# added by structlint suggest\n%s:\n%s\n",
			target, renderItems(values, "  "))
	}
	// Walk forward collecting list items directly under target.
	itemIndent := ""
	lastItemIdx := targetIdx
	for i := targetIdx + 1; i < len(lines); i++ {
		match := listItemRE.FindStringSubmatch(lines[i])
		if match == nil {
			// Blank line? Continue past it inside the same block.
			if strings.TrimSpace(lines[i]) == "" {
				continue
			}
			// Non-item line at any deeper indent means we've left the block.
			break
		}
		itemIndent = match[1]
		lastItemIdx = i
	}
	if itemIndent == "" {
		// Empty list or `[]` syntax; guess a reasonable indent.
		itemIndent = strings.Repeat(" ", leadingSpaces(lines[targetIdx])+2)
	}
	newItems := make([]string, 0, len(values))
	for _, v := range values {
		newItems = append(newItems, fmt.Sprintf(`%s- %q`, itemIndent, v))
	}
	// Splice after lastItemIdx.
	updated := make([]string, 0, len(lines)+len(newItems))
	updated = append(updated, lines[:lastItemIdx+1]...)
	updated = append(updated, newItems...)
	updated = append(updated, lines[lastItemIdx+1:]...)
	return strings.Join(updated, "\n")
}

func leadingSpaces(s string) int {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return i
		}
	}
	return len(s)
}

func renderItems(values []string, indent string) string {
	var b bytes.Buffer
	for _, v := range values {
		fmt.Fprintf(&b, "%s- %q\n", indent, v)
	}
	return b.String()
}

// unifiedDiff produces a `diff -u`-style patch between orig and modified.
// Only the changed hunks are emitted; unchanged head/tail are elided.
func unifiedDiff(path, orig, modified string) string {
	origLines := strings.Split(orig, "\n")
	modLines := strings.Split(modified, "\n")

	rel := filepath.ToSlash(path)
	var out bytes.Buffer
	fmt.Fprintf(&out, "--- a/%s\n", rel)
	fmt.Fprintf(&out, "+++ b/%s\n", rel)

	// Diff via the simple additive-only algorithm since we only ever INSERT
	// lines; no removals, no in-line edits. Walk both, emit context around
	// the inserted lines.
	i, j := 0, 0
	const contextLines = 3
	for i < len(origLines) || j < len(modLines) {
		// Fast-forward through matching lines.
		for i < len(origLines) && j < len(modLines) && origLines[i] == modLines[j] {
			i++
			j++
		}
		if i >= len(origLines) && j >= len(modLines) {
			break
		}
		// Detect the block of insertions in modified until they realign with
		// origLines[i].
		insertStart := j
		for j < len(modLines) && (i >= len(origLines) || modLines[j] != origLines[i]) {
			j++
		}
		// Emit hunk header with context.
		hunkStartOrig := max0(i - contextLines)
		hunkStartMod := max0(insertStart - contextLines)
		origHunk := origLines[hunkStartOrig:i]
		modInserts := modLines[insertStart:j]
		modTrailContext := []string{}
		trailEnd := min(i+contextLines, len(origLines))
		if trailEnd > i {
			modTrailContext = origLines[i:trailEnd]
		}
		origHunkLen := len(origHunk) + len(modTrailContext)
		modHunkLen := len(origHunk) + len(modInserts) + len(modTrailContext)
		fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n",
			hunkStartOrig+1, origHunkLen,
			hunkStartMod+1, modHunkLen,
		)
		for _, l := range origHunk {
			fmt.Fprintf(&out, " %s\n", l)
		}
		for _, l := range modInserts {
			fmt.Fprintf(&out, "+%s\n", l)
		}
		for _, l := range modTrailContext {
			fmt.Fprintf(&out, " %s\n", l)
		}
		i = trailEnd
		j += len(modTrailContext)
	}
	return out.String()
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
