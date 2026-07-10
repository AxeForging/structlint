package hooks

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	gitHookMarkerStart = "# >>> structlint hook >>>"
	gitHookMarkerEnd   = "# <<< structlint hook <<<"
	gitHookShebang     = "#!/bin/sh"
)

// InstallGitHook writes or updates a marker-blocked snippet in the git
// pre-commit hook. Respects core.hooksPath via `git rev-parse`.
func InstallGitHook(dir string, dryRun bool) (Result, error) {
	hooksDir, err := resolveHooksDir(dir)
	if err != nil {
		return Result{Type: TypeGit}, err
	}

	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return Result{Type: TypeGit, File: hooksDir}, fmt.Errorf("create hooks dir: %w", err)
	}

	path := filepath.Join(hooksDir, "pre-commit")
	res := Result{Type: TypeGit, File: path}
	block := gitHookBlock()

	var newContent []byte
	var already bool
	switch {
	case !fileExists(path):
		newContent = []byte(gitHookShebang + "\n\n" + block + "\n")
	default:
		existing, err := os.ReadFile(path)
		if err != nil {
			return res, fmt.Errorf("read pre-commit hook: %w", err)
		}
		updated, unchanged := mergeGitHookBlock(existing, block)
		if unchanged {
			already = true
			newContent = existing
		} else {
			newContent = updated
		}
	}

	if already {
		res.Action = ActionAlreadyInstalled
		res.Reason = "marker block already up to date"
		return res, nil
	}

	if dryRun {
		res.Action = ActionInstalled
		res.Preview = string(newContent)
		return res, nil
	}

	if err := writeFileAtomic(path, newContent, 0o755); err != nil {
		return res, fmt.Errorf("write pre-commit hook: %w", err)
	}
	res.Action = ActionInstalled
	return res, nil
}

// resolveHooksDir asks git where its hooks live so we honor core.hooksPath
// and worktrees. Returns an actionable error when git is unavailable or
// dir isn't inside a repository.
func resolveHooksDir(dir string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", errors.New("git not found in PATH; install git or use --type lefthook|pre-commit")
	}
	cmd := exec.Command("git", "rev-parse", "--git-path", "hooks")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git rev-parse failed): run `git init` first")
	}
	rel := strings.TrimSpace(string(out))
	if rel == "" {
		return "", errors.New("git rev-parse returned empty hooks path")
	}
	if filepath.IsAbs(rel) {
		return rel, nil
	}
	return filepath.Join(dir, rel), nil
}

func gitHookBlock() string {
	return strings.Join([]string{
		gitHookMarkerStart,
		"# Managed by `structlint hook install`. Edit inside markers only.",
		"if command -v structlint >/dev/null 2>&1; then",
		"  " + HookRun + " || exit 1",
		"fi",
		gitHookMarkerEnd,
	}, "\n")
}

// mergeGitHookBlock returns the file contents with the block ensured in place.
// When the file already contains a block bounded by our markers, it's
// replaced with the current block; otherwise the block is appended.
// The second return value is true when the file was already up-to-date.
func mergeGitHookBlock(existing []byte, block string) ([]byte, bool) {
	startIdx := bytes.Index(existing, []byte(gitHookMarkerStart))
	if startIdx == -1 {
		// No existing block. Preserve trailing newline discipline.
		buf := append([]byte{}, existing...)
		if len(buf) > 0 && buf[len(buf)-1] != '\n' {
			buf = append(buf, '\n')
		}
		buf = append(buf, '\n')
		buf = append(buf, []byte(block)...)
		buf = append(buf, '\n')
		return buf, false
	}
	endIdx := bytes.Index(existing, []byte(gitHookMarkerEnd))
	if endIdx == -1 || endIdx < startIdx {
		// Malformed (opening marker without closing). Append a fresh block
		// rather than trying to guess where it ends.
		buf := append([]byte{}, existing...)
		if len(buf) > 0 && buf[len(buf)-1] != '\n' {
			buf = append(buf, '\n')
		}
		buf = append(buf, '\n')
		buf = append(buf, []byte(block)...)
		buf = append(buf, '\n')
		return buf, false
	}
	endMarkerFull := endIdx + len(gitHookMarkerEnd)
	before := existing[:startIdx]
	after := existing[endMarkerFull:]
	rebuilt := append([]byte{}, before...)
	rebuilt = append(rebuilt, []byte(block)...)
	rebuilt = append(rebuilt, after...)
	unchanged := bytes.Equal(rebuilt, existing)
	return rebuilt, unchanged
}
