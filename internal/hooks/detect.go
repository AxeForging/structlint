// Package hooks installs structlint into a repository's pre-commit hook
// chain. Every installer in this package must be idempotent (running twice
// is a no-op) and non-destructive (never overwrites user content it didn't
// put there).
package hooks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Type identifies which hook backend to target.
type Type string

const (
	TypeLefthook  Type = "lefthook"
	TypePreCommit Type = "pre-commit"
	TypeGit       Type = "git"
)

// Action names what the installer did (or would do under --dry-run).
type Action string

const (
	ActionInstalled        Action = "installed"
	ActionAlreadyInstalled Action = "already-installed"
	ActionRefused          Action = "refused"
)

// Result describes the outcome of an install.
type Result struct {
	Type    Type
	Action  Action
	File    string // absolute path of the file the installer targeted
	Reason  string // human-readable detail, especially on refusal
	Preview string // rendered file/preview when --dry-run
}

// HookRun is the command every hook target invokes.
const HookRun = "structlint validate --staged --silent"

// ErrNoTargetDetected is returned from Detect when no known hook framework
// is present. Callers should fall back to the raw git-hook installer.
var ErrNoTargetDetected = errors.New("no hook framework detected")

// Detect chooses a hook backend based on files already present in dir.
// lefthook wins over pre-commit; both win over the raw git-hook fallback.
func Detect(dir string) (Type, error) {
	if fileExists(filepath.Join(dir, "lefthook.yml")) || fileExists(filepath.Join(dir, "lefthook.yaml")) {
		return TypeLefthook, nil
	}
	if fileExists(filepath.Join(dir, ".pre-commit-config.yaml")) {
		return TypePreCommit, nil
	}
	return TypeGit, ErrNoTargetDetected
}

// Install runs the installer for the requested hook type.
func Install(dir string, t Type, dryRun bool) (Result, error) {
	switch t {
	case TypeLefthook:
		return InstallLefthook(dir, dryRun)
	case TypePreCommit:
		return InstallPreCommit(dir, dryRun)
	case TypeGit:
		return InstallGitHook(dir, dryRun)
	default:
		return Result{}, fmt.Errorf("unknown hook type: %s", t)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
