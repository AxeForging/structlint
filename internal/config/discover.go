package config

import (
	"os"
	"path/filepath"
)

// configNames are the file names structlint recognizes for its config,
// in per-directory priority order. The first that stats as a regular
// file wins for that directory.
var configNames = []string{
	".structlint.yaml",
	".structlint.yml",
	".structlint.json",
}

// Discover walks upward from startDir looking for a structlint config
// file. It stops after checking the first directory that contains a
// .git entry (file or directory — worktrees use a .git file) or at
// the filesystem root. Returns the absolute path of the first match,
// or "" when no match is found within the search boundary.
//
// The .git-containing directory is checked BEFORE the search stops,
// so a config sitting alongside .git is discoverable. This matches the
// behavior of most repo-scoped linters.
func Discover(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		for _, name := range configNames {
			candidate := filepath.Join(dir, name)
			info, err := os.Stat(candidate)
			if err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
		// Boundary check happens after the current dir has been inspected,
		// so the .git-containing directory is inclusive.
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return "", nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}
