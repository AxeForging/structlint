package validator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AxeForging/structlint/internal/config"
	"github.com/gobwas/glob"
)

// Validator holds the configuration and validation results.
type Validator struct {
	Config          *config.Config
	Errors          []string
	Violations      []Violation
	Successes       int
	Logger          *slog.Logger
	Silent          bool
	GroupViolations bool
	Verbose         bool // Show all allowed files, not just violations
	ChangedOnly     bool
	changedPaths    map[string]bool
}

// New creates a new Validator.
func New(cfg *config.Config, logger *slog.Logger) *Validator {
	return &Validator{
		Config:          cfg,
		Errors:          []string{},
		Violations:      []Violation{},
		Successes:       0,
		Logger:          logger,
		Silent:          false,
		GroupViolations: true,  // Default to grouping
		Verbose:         false, // Default to quiet mode - only show violations
	}
}

// ValidateDirStructure is a thin wrapper kept for external callers (e.g. the
// root validator_test.go). It snapshots the tree and runs the corresponding
// rule against it — same output as v.Run() would produce.
func (v *Validator) ValidateDirStructure(path string) {
	v.runSingleRule(path, dirStructureRule{})
}

// runSingleRule is the shared entry point for the legacy Validate* wrappers.
// It snapshots root once and runs exactly one rule so wrapper behavior
// stays parity-safe with the modern v.Run() path.
func (v *Validator) runSingleRule(path string, rule Rule) {
	tree := Snapshot(path, v.Config.Ignore)
	ctx := &RunContext{
		Cfg:  v.Config,
		Tree: tree,
		V:    v,
		Skip: func(rel string, isDir bool) bool {
			if isDir {
				return v.shouldSkipChangedDir(rel)
			}
			return v.shouldSkipChanged(rel)
		},
	}
	rule.Run(ctx)
}

// matches checks if a path matches a glob pattern.
func matches(path, pattern string) bool {
	g, err := glob.Compile(pattern)
	if err != nil {
		return false
	}
	return g.Match(path)
}

// ValidateFileNaming is a thin wrapper — see runSingleRule.
func (v *Validator) ValidateFileNaming(path string) {
	v.runSingleRule(path, fileNamingRule{})
}

// PrintSummary prints a summary of the validation results.
func (v *Validator) PrintSummary() {
	if v.Silent {
		return
	}

	// Always show a concise summary
	fmt.Println("\n--- Validation Summary ---")
	fmt.Printf("✓ %d files/directories passed validation\n", v.Successes)
	fmt.Printf("✗ %d violations found\n", len(v.Errors))

	if len(v.Errors) == 0 {
		fmt.Println("🎉 All files and directories comply with the rules!")
		return
	}

	// Use grouped summary if enabled or if there are many violations
	if v.GroupViolations || len(v.Errors) > 10 {
		v.PrintGroupedSummary()
		return
	}

	// For few errors, show detailed list
	fmt.Println("\nViolations:")
	for _, err := range v.Errors {
		fmt.Printf("- %s\n", err)
	}
}

// ValidateRequiredPaths is a thin wrapper — see runSingleRule.
func (v *Validator) ValidateRequiredPaths(path string) {
	v.validateRequiredPathsDirect(path)
}

// validateRequiredPathsDirect stats each requiredPath directly. Preserves
// the quirk that requiredPaths inside an ignored directory still count
// as present — this is the legacy behavior locked by parity goldens.
func (v *Validator) validateRequiredPathsDirect(path string) {
	for _, requiredPath := range v.Config.DirStructure.RequiredPaths {
		fullPath := filepath.Join(path, requiredPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			msg := fmt.Sprintf("Required directory missing: %s", requiredPath)
			v.addViolation("missing_required_directory", "error", requiredPath, "dir_structure.requiredPaths", msg)
		} else {
			v.printSuccess(fmt.Sprintf("Required directory found: %s", requiredPath))
			v.Successes++
		}
	}
}

// ValidateRequiredFiles is a thin wrapper — see runSingleRule.
func (v *Validator) ValidateRequiredFiles(path string) {
	v.runSingleRule(path, requiredFilesRule{})
}

// ValidatePlacement is a thin wrapper — see runSingleRule.
func (v *Validator) ValidatePlacement(path string) {
	v.runSingleRule(path, placementRule{})
}

// ValidateRequiredGroups is a thin wrapper — see runSingleRule.
func (v *Validator) ValidateRequiredGroups(path string) {
	v.validateRequiredGroupsDirect(path)
}

// validateRequiredGroupsDirect implements requiredGroups using stat-based
// helpers (existsAt, existsAny, matchingDirs) which intentionally see
// through ignored directories. Preserving that behavior is a parity
// requirement locked by the goldens.
func (v *Validator) validateRequiredGroupsDirect(path string) {
	root := cleanRoot(path)
	for _, group := range v.Config.RequiredGroups {
		if len(group.OneOf) > 0 {
			if existsAny(root, group.OneOf, v.Config.Ignore) {
				v.Successes++
			} else {
				msg := fmt.Sprintf("Required group missing one of: %s", strings.Join(group.OneOf, ", "))
				v.addViolation("missing_required_group", severity(group.Severity), group.ID, group.ID, msg)
			}
		}
		if group.EachDirMatching == "" {
			continue
		}
		matches := matchingDirs(root, group.EachDirMatching, v.Config.Ignore)
		if len(matches) == 0 && group.RequireMatch {
			msg := fmt.Sprintf("Required group matched no directories: %s", group.EachDirMatching)
			v.addViolation("missing_required_group_match", severity(group.Severity), group.EachDirMatching, group.ID, msg)
		}
		for _, dir := range matches {
			for _, required := range group.MustContain {
				if !existsAt(root, filepath.ToSlash(filepath.Join(dir, required))) {
					msg := fmt.Sprintf("Directory %s missing required file: %s", dir, required)
					v.addViolation("missing_group_file", severity(group.Severity), filepath.ToSlash(filepath.Join(dir, required)), group.ID, msg)
				} else {
					v.Successes++
				}
			}
			if len(group.MustContainOneOf) > 0 {
				found := false
				for _, required := range group.MustContainOneOf {
					if existsAt(root, filepath.ToSlash(filepath.Join(dir, required))) {
						found = true
						break
					}
				}
				if found {
					v.Successes++
				} else {
					msg := fmt.Sprintf("Directory %s missing one of: %s", dir, strings.Join(group.MustContainOneOf, ", "))
					v.addViolation("missing_group_file", severity(group.Severity), dir, group.ID, msg)
				}
			}
		}
	}
}

// ValidateBoundaries is a thin wrapper — see runSingleRule.
func (v *Validator) ValidateBoundaries(path string) {
	v.runSingleRule(path, boundariesRule{})
}

// LoadChangedPaths populates the changed-file set used by --changed-only,
// diffing against HEAD (working tree). Kept for backward compatibility.
func (v *Validator) LoadChangedPaths(path string) {
	v.LoadChangedPathsMode(path, false)
}

// LoadChangedPathsMode populates the changed-file set. When staged is true it
// uses `git diff --cached` (staged index) instead of HEAD.
func (v *Validator) LoadChangedPathsMode(path string, staged bool) {
	v.ChangedOnly = true
	root := cleanRoot(path)
	changed := map[string]bool{}
	args := []string{"diff", "--name-only", "--diff-filter=ACMRT"}
	if staged {
		args = append(args, "--cached")
	} else {
		args = append(args, "HEAD")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		v.changedPaths = changed
		return
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		p := normalizePath(scanner.Text())
		if p != "" {
			changed[p] = true
		}
	}
	v.changedPaths = changed
}

// ApplyBaseline suppresses violations already recorded in a previous JSON report.
func (v *Validator) ApplyBaseline(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report JSONReport
	if err := json.Unmarshal(data, &report); err != nil {
		return err
	}
	known := map[string]bool{}
	for _, violation := range report.Violations {
		known[violationKey(violation)] = true
	}
	if len(known) == 0 {
		for _, errText := range report.Errors {
			known[errText] = true
		}
	}
	var keptViolations []Violation
	var keptErrors []string
	for _, violation := range v.Violations {
		if known[violationKey(violation)] || known[violation.Message] {
			continue
		}
		keptViolations = append(keptViolations, violation)
		keptErrors = append(keptErrors, violation.Message)
	}
	v.Violations = keptViolations
	v.Errors = keptErrors
	return nil
}

func (v *Validator) addViolation(code, sev, path, rule, message string) {
	if sev == "" {
		sev = "error"
	}
	violation := Violation{
		Code:     code,
		Severity: sev,
		Path:     normalizePath(path),
		Rule:     rule,
		Message:  message,
	}
	v.printError(message)
	v.Violations = append(v.Violations, violation)
	v.Errors = append(v.Errors, message)
}

func (v *Validator) sortedViolations() []Violation {
	violations := append([]Violation(nil), v.Violations...)
	sort.SliceStable(violations, func(i, j int) bool {
		if violations[i].Path == violations[j].Path {
			if violations[i].Code == violations[j].Code {
				return violations[i].Rule < violations[j].Rule
			}
			return violations[i].Code < violations[j].Code
		}
		return violations[i].Path < violations[j].Path
	})
	return violations
}

func (v *Validator) shouldSkipChanged(relPath string) bool {
	if !v.ChangedOnly {
		return false
	}
	if len(v.changedPaths) == 0 {
		return true
	}
	return !v.changedPaths[normalizePath(relPath)]
}

// shouldSkipChangedDir returns true when a directory falls outside the
// changed-file scope. A directory is in-scope if it equals a changed path
// or is an ancestor of one; "." (root) is always in-scope so the walk starts.
func (v *Validator) shouldSkipChangedDir(relPath string) bool {
	if !v.ChangedOnly {
		return false
	}
	if len(v.changedPaths) == 0 {
		return true
	}
	rel := normalizePath(relPath)
	if rel == "." || rel == "" {
		return false
	}
	prefix := rel + "/"
	for changed := range v.changedPaths {
		if changed == rel || strings.HasPrefix(changed, prefix) {
			return false
		}
	}
	return true
}

func (v *Validator) printSuccess(message string) {
	if !v.Silent && v.Logger != nil && v.Verbose {
		v.Logger.Info("✓ " + message)
	}
}

func (v *Validator) printError(message string) {
	if !v.Silent && v.Logger != nil {
		v.Logger.Error("✗ " + message)
	}
}
