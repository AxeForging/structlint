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

// ValidateDirStructure validates the directory structure.
func (v *Validator) ValidateDirStructure(path string) {
	root := cleanRoot(path)
	err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath := relativePath(root, currentPath)

		// Check if the path should be ignored
		for _, ignored := range v.Config.Ignore {
			if pathMatches(relPath, ignored) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			if v.shouldSkipChangedDir(relPath) {
				return filepath.SkipDir
			}

			// Check against disallowed paths
			for _, disallowed := range v.Config.DirStructure.DisallowedPaths {
				if pathMatches(relPath, disallowed) {
					msg := fmt.Sprintf("Disallowed directory found: %s", relPath)
					v.addViolation("disallowed_directory", "error", relPath, disallowed, msg)
					return filepath.SkipDir // Skip validating contents of disallowed directories
				}
			}

			// Check against allowed paths
			isAllowed := false
			for _, allowed := range v.Config.DirStructure.AllowedPaths {
				if pathMatches(relPath, allowed) || isParentOfPattern(relPath, allowed) {
					isAllowed = true
					break
				}
			}

			if isAllowed {
				msg := fmt.Sprintf("Allowed directory found: %s", relPath)
				v.printSuccess(msg)
				v.Successes++
			} else {
				msg := fmt.Sprintf("Directory not in allowed list: %s", relPath)
				v.addViolation("unallowed_directory", "error", relPath, "dir_structure.allowedPaths", msg)
			}
		}
		return nil
	})
	if err != nil {
		v.addViolation("walk_error", "error", path, "filesystem", fmt.Sprintf("Error walking directory: %s", err))
	}
}

// matches checks if a path matches a glob pattern.
func matches(path, pattern string) bool {
	g, err := glob.Compile(pattern)
	if err != nil {
		return false
	}
	return g.Match(path)
}

// ValidateFileNaming validates the file naming conventions.
func (v *Validator) ValidateFileNaming(path string) {
	root := cleanRoot(path)
	err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath := relativePath(root, currentPath)

		// Check if the path should be ignored
		for _, ignored := range v.Config.Ignore {
			if pathMatches(relPath, ignored) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !info.IsDir() {
			if v.shouldSkipChanged(relPath) {
				return nil
			}
			fileName := info.Name()

			// Check against disallowed patterns
			for _, disallowed := range v.Config.FileNamingPattern.Disallowed {
				if pathMatches(fileName, disallowed) || pathMatches(relPath, disallowed) {
					msg := fmt.Sprintf("Disallowed file naming pattern found: %s", relPath)
					v.addViolation("disallowed_file_pattern", "error", relPath, disallowed, msg)
					return nil
				}
			}

			// Check against allowed patterns
			isAllowed := false
			for _, allowed := range v.Config.FileNamingPattern.Allowed {
				if pathMatches(fileName, allowed) || pathMatches(relPath, allowed) {
					isAllowed = true
					break
				}
			}
			if isAllowed {
				msg := fmt.Sprintf("Allowed file naming pattern found: %s", relPath)
				v.printSuccess(msg)
				v.Successes++
			} else {
				msg := fmt.Sprintf("File not in allowed naming pattern: %s", relPath)
				v.addViolation("unallowed_file_pattern", "error", relPath, "file_naming_pattern.allowed", msg)
			}
		}
		return nil
	})
	if err != nil {
		v.addViolation("walk_error", "error", path, "filesystem", fmt.Sprintf("Error walking directory: %s", err))
	}
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

// ValidateRequiredPaths validates that all required directories exist.
func (v *Validator) ValidateRequiredPaths(path string) {
	for _, requiredPath := range v.Config.DirStructure.RequiredPaths {
		fullPath := filepath.Join(path, requiredPath)

		// Check if the required path exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			msg := fmt.Sprintf("Required directory missing: %s", requiredPath)
			v.addViolation("missing_required_directory", "error", requiredPath, "dir_structure.requiredPaths", msg)
		} else {
			v.printSuccess(fmt.Sprintf("Required directory found: %s", requiredPath))
			v.Successes++
		}
	}
}

// ValidateRequiredFiles validates that all required files exist.
func (v *Validator) ValidateRequiredFiles(path string) {
	root := cleanRoot(path)
	for _, requiredFile := range v.Config.FileNamingPattern.Required {
		// Check if any file matching the pattern exists
		found := false
		err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath := relativePath(root, currentPath)

			// Check if the path should be ignored
			for _, ignored := range v.Config.Ignore {
				if pathMatches(relPath, ignored) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if !info.IsDir() {
				// For required file patterns, we need to check both the filename and the relative path
				fileName := info.Name()
				// Check if either the filename or the relative path matches the pattern
				if pathMatches(fileName, requiredFile) || pathMatches(relPath, requiredFile) {
					found = true
					return filepath.SkipAll // Stop walking once we find a match
				}
			}
			return nil
		})
		if err != nil {
			v.addViolation("walk_error", "error", requiredFile, "file_naming_pattern.required", fmt.Sprintf("Error checking for required file %s: %s", requiredFile, err))
			continue
		}

		if found {
			v.printSuccess(fmt.Sprintf("Required file pattern found: %s", requiredFile))
			v.Successes++
		} else {
			msg := fmt.Sprintf("Required file pattern missing: %s", requiredFile)
			v.addViolation("missing_required_file", "error", requiredFile, requiredFile, msg)
		}
	}
}

// ValidatePlacement validates file placement rules.
func (v *Validator) ValidatePlacement(path string) {
	root := cleanRoot(path)
	_ = filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			v.addViolation("walk_error", "error", currentPath, "placement", fmt.Sprintf("Error walking directory: %s", err))
			return nil
		}
		relPath := relativePath(root, currentPath)
		for _, ignored := range v.Config.Ignore {
			if pathMatches(relPath, ignored) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if info.IsDir() || v.shouldSkipChanged(relPath) {
			return nil
		}
		for _, rule := range v.Config.Placement {
			if !matchesAnyFile(relPath, info.Name(), rule.Files) {
				continue
			}
			if underAny(relPath, rule.MustBeUnder) {
				v.Successes++
				continue
			}
			msg := fmt.Sprintf("File placement violation: %s must be under %s", relPath, strings.Join(rule.MustBeUnder, ", "))
			v.addViolation("placement_violation", severity(rule.Severity), relPath, rule.ID, msg)
		}
		return nil
	})
}

// ValidateRequiredGroups validates one-of and per-directory requirements.
func (v *Validator) ValidateRequiredGroups(path string) {
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

// ValidateBoundaries validates import boundaries for supported source files.
func (v *Validator) ValidateBoundaries(path string) {
	root := cleanRoot(path)
	modulePath := readGoModule(root)
	_ = filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			v.addViolation("walk_error", "error", currentPath, "boundaries", fmt.Sprintf("Error walking directory: %s", err))
			return nil
		}
		relPath := relativePath(root, currentPath)
		for _, ignored := range v.Config.Ignore {
			if pathMatches(relPath, ignored) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if info.IsDir() || !isSupportedBoundaryFile(relPath) || v.shouldSkipChanged(relPath) {
			return nil
		}
		for _, rule := range v.Config.Boundaries {
			if !pathMatches(relPath, rule.From) {
				continue
			}
			imports, err := sourceImports(currentPath, relPath)
			if err != nil {
				v.addViolation("parse_error", "error", relPath, rule.ID, fmt.Sprintf("Failed to parse imports: %s", err))
				continue
			}
			for _, imp := range imports {
				localImport := importToLocalPath(modulePath, imp, relPath)
				for _, forbidden := range rule.CannotImport {
					if pathMatches(imp, forbidden) || pathMatches(localImport, forbidden) {
						msg := fmt.Sprintf("Boundary violation: %s imports %s", relPath, imp)
						v.addViolation("boundary_violation", severity(rule.Severity), relPath, rule.ID, msg)
					}
				}
			}
			v.Successes++
		}
		return nil
	})
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
