package validator

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AxeForging/structlint/internal/config"
)

// binarySniffLen is how many leading bytes are inspected for a NUL byte to
// decide a file is binary — the same heuristic git uses.
const binarySniffLen = 8000

// fileSizeRule enforces file_size.rules (spec 014). It runs last in the
// Registry so the order of every pre-existing rule is unchanged.
type fileSizeRule struct{}

func (fileSizeRule) Name() string { return "file_size" }

func (fileSizeRule) Run(ctx *RunContext) {
	fs := ctx.Cfg.FileSize
	if len(fs.Rules) == 0 {
		return
	}
	for _, e := range ctx.Tree.Entries {
		if e.IsDir || ctx.Skip(e.RelPath, false) {
			continue
		}
		if fileSizeExcluded(e.RelPath, fs.Exclude) {
			continue
		}
		rule, ok := firstFileSizeRule(e.RelPath, fs.Rules)
		if !ok {
			continue
		}
		n, binary, err := countFileLines(e.Abs)
		if err != nil {
			ctx.V.addViolation("walk_error", "error", e.RelPath, "file_size",
				fmt.Sprintf("Error reading file for size check: %s: %s", e.RelPath, err))
			continue
		}
		if binary {
			continue
		}
		if n > rule.MaxLines {
			msg := fmt.Sprintf("File too long: %s has %d lines (max %d)", e.RelPath, n, rule.MaxLines)
			ctx.V.addViolation("file_too_long", severity(rule.Severity), e.RelPath, rule.Pattern, msg)
			continue
		}
		ctx.V.printSuccess(fmt.Sprintf("File within line limit: %s (%d/%d)", e.RelPath, n, rule.MaxLines))
		ctx.V.Successes++
	}
}

// firstFileSizeRule returns the first rule whose pattern matches relPath.
// Order is the precedence: later rules are never consulted once one matches.
func firstFileSizeRule(relPath string, rules []config.FileSizeRule) (config.FileSizeRule, bool) {
	for _, rule := range rules {
		if fileSizeMatches(relPath, rule.Pattern) {
			return rule, true
		}
	}
	return config.FileSizeRule{}, false
}

func fileSizeExcluded(relPath string, exclude []string) bool {
	for _, pattern := range exclude {
		if fileSizeMatches(relPath, pattern) {
			return true
		}
	}
	return false
}

// fileSizeMatches applies the standard structlint glob semantics (relative
// path or base name) and additionally lets a leading "**/" match zero
// directories, so "**/*.py" also covers root-level files.
func fileSizeMatches(relPath, pattern string) bool {
	name := relPath
	if i := strings.LastIndex(relPath, "/"); i >= 0 {
		name = relPath[i+1:]
	}
	if matchesAnyFile(relPath, name, []string{pattern}) {
		return true
	}
	if rest, ok := strings.CutPrefix(pattern, "**/"); ok && rest != "" {
		return pathMatches(relPath, rest)
	}
	return false
}

// countFileLines opens path and counts its lines. Non-regular files (dirs
// reached through symlinks, devices, sockets) are reported as binary so
// they are skipped rather than read.
func countFileLines(path string) (int, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, false, err
	}
	if !info.Mode().IsRegular() {
		return 0, true, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = f.Close() }()
	return countLines(f)
}

// countLines counts lines the way `wc -l` does: the number of '\n' bytes.
// A final line without a trailing newline is therefore not counted, CRLF
// counts once, and a lone '\r' is not a line break. binary is true when a
// NUL byte appears in the first binarySniffLen bytes; counting stops there.
func countLines(r io.Reader) (lines int, binary bool, err error) {
	buf := make([]byte, 32*1024)
	sniffed := 0
	for {
		n, readErr := r.Read(buf)
		chunk := buf[:n]
		if sniffed < binarySniffLen && n > 0 {
			end := min(n, binarySniffLen-sniffed)
			if bytes.IndexByte(chunk[:end], 0) >= 0 {
				return 0, true, nil
			}
			sniffed += end
		}
		lines += bytes.Count(chunk, []byte{'\n'})
		if errors.Is(readErr, io.EOF) {
			return lines, false, nil
		}
		if readErr != nil {
			return 0, false, readErr
		}
	}
}
