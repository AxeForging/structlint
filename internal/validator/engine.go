package validator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/AxeForging/structlint/internal/config"
)

// Rule is the pluggable unit of validation. Each family (dir structure,
// file naming, placement, ...) is a Rule. Rules run in Registry order —
// that order matches the legacy Validate* invocation sequence and is
// part of the observable contract enforced by the parity goldens.
type Rule interface {
	Name() string
	Run(ctx *RunContext)
}

// RunContext bundles the state a rule needs.
type RunContext struct {
	Cfg  *config.Config
	Tree *Tree
	V    *Validator
	// Skip returns true when the given relative path (dir or file) is out
	// of scope for --changed-only / --staged. Rules that are meant to be
	// global (required-* families) never call Skip.
	Skip func(relPath string, isDir bool) bool
}

// Registry returns the seven rules in legacy execution order:
// dir_structure, file_naming, required_paths, required_files,
// placement, required_groups, boundaries.
func Registry(cfg *config.Config) []Rule {
	return []Rule{
		dirStructureRule{},
		fileNamingRule{},
		requiredPathsRule{},
		requiredFilesRule{},
		placementRule{},
		requiredGroupsRule{},
		boundariesRule{},
	}
}

// Run snapshots the tree once, then executes each rule in Registry order
// against that single snapshot. This is the single-walk architecture:
// rules do not perform their own filepath.Walk — they iterate ctx.Tree.
func (v *Validator) Run(path string) {
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
	for _, rule := range Registry(v.Config) {
		rule.Run(ctx)
	}
}

// skipTracker tracks path prefixes whose subtree should be skipped from
// iteration. Used to preserve filepath.SkipDir semantics when we hit a
// disallowed directory in the middle of a single walk.
type skipTracker struct {
	prefixes []string
}

// mark adds relPath as the root of a skipped subtree.
func (s *skipTracker) mark(relPath string) {
	if relPath == "" || relPath == "." {
		return
	}
	s.prefixes = append(s.prefixes, relPath)
}

// under reports whether relPath is inside any skipped subtree.
func (s *skipTracker) under(relPath string) bool {
	for _, p := range s.prefixes {
		if relPath == p || strings.HasPrefix(relPath, p+"/") {
			return true
		}
	}
	return false
}

// dirStructureRule validates directory structure by iterating Tree.Entries.
// Preserves filepath.SkipDir semantics: hitting a disallowed directory
// suppresses violations for its entire subtree (matching the legacy walk).
type dirStructureRule struct{}

func (dirStructureRule) Name() string { return "dir_structure" }

func (dirStructureRule) Run(ctx *RunContext) {
	if ctx.Tree.WalkErr != nil {
		ctx.V.addViolation("walk_error", "error", ctx.Tree.Root, "filesystem",
			fmt.Sprintf("Error walking directory: %s", ctx.Tree.WalkErr))
	}
	skipped := skipTracker{}
	for _, e := range ctx.Tree.Entries {
		if !e.IsDir {
			continue
		}
		if skipped.under(e.RelPath) {
			continue
		}
		if ctx.Skip(e.RelPath, true) {
			// Not part of the changed-set; skip its subtree too, matching
			// the legacy behavior where the walker returned SkipDir.
			skipped.mark(e.RelPath)
			continue
		}
		// Check against disallowed paths.
		matched := false
		for _, disallowed := range ctx.Cfg.DirStructure.DisallowedPaths {
			if pathMatches(e.RelPath, disallowed) {
				msg := fmt.Sprintf("Disallowed directory found: %s", e.RelPath)
				ctx.V.addViolation("disallowed_directory", "error", e.RelPath, disallowed, msg)
				skipped.mark(e.RelPath)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		// Check against allowed paths.
		isAllowed := false
		for _, allowed := range ctx.Cfg.DirStructure.AllowedPaths {
			if pathMatches(e.RelPath, allowed) || isParentOfPattern(e.RelPath, allowed) {
				isAllowed = true
				break
			}
		}
		if isAllowed {
			ctx.V.printSuccess(fmt.Sprintf("Allowed directory found: %s", e.RelPath))
			ctx.V.Successes++
		} else {
			msg := fmt.Sprintf("Directory not in allowed list: %s", e.RelPath)
			ctx.V.addViolation("unallowed_directory", "error", e.RelPath, "dir_structure.allowedPaths", msg)
		}
	}
}

// fileNamingRule validates file naming patterns via Tree iteration.
type fileNamingRule struct{}

func (fileNamingRule) Name() string { return "file_naming" }

func (fileNamingRule) Run(ctx *RunContext) {
	if ctx.Tree.WalkErr != nil {
		ctx.V.addViolation("walk_error", "error", ctx.Tree.Root, "filesystem",
			fmt.Sprintf("Error walking directory: %s", ctx.Tree.WalkErr))
	}
	for _, e := range ctx.Tree.Entries {
		if e.IsDir {
			continue
		}
		if ctx.Skip(e.RelPath, false) {
			continue
		}
		// Disallowed patterns first — matching a disallowed pattern short-
		// circuits, matching legacy behavior of `return nil`.
		matched := false
		for _, disallowed := range ctx.Cfg.FileNamingPattern.Disallowed {
			if pathMatches(e.Name, disallowed) || pathMatches(e.RelPath, disallowed) {
				msg := fmt.Sprintf("Disallowed file naming pattern found: %s", e.RelPath)
				ctx.V.addViolation("disallowed_file_pattern", "error", e.RelPath, disallowed, msg)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		isAllowed := false
		for _, allowed := range ctx.Cfg.FileNamingPattern.Allowed {
			if pathMatches(e.Name, allowed) || pathMatches(e.RelPath, allowed) {
				isAllowed = true
				break
			}
		}
		if isAllowed {
			ctx.V.printSuccess(fmt.Sprintf("Allowed file naming pattern found: %s", e.RelPath))
			ctx.V.Successes++
		} else {
			msg := fmt.Sprintf("File not in allowed naming pattern: %s", e.RelPath)
			ctx.V.addViolation("unallowed_file_pattern", "error", e.RelPath, "file_naming_pattern.allowed", msg)
		}
	}
}

// requiredPathsRule intentionally does NOT use the Tree — the legacy code
// stats the joined path directly, so a required path inside an ignored
// directory still counts as present. Preserving that quirk is spec 005's
// parity requirement.
type requiredPathsRule struct{}

func (requiredPathsRule) Name() string { return "required_paths" }

func (requiredPathsRule) Run(ctx *RunContext) {
	ctx.V.validateRequiredPathsDirect(ctx.Tree.Root)
}

// requiredFilesRule finds required patterns via Tree lookup. Unlike required
// paths, the legacy code walked with ignore filtering, so using the Tree
// matches behavior.
type requiredFilesRule struct{}

func (requiredFilesRule) Name() string { return "required_files" }

func (requiredFilesRule) Run(ctx *RunContext) {
	for _, requiredFile := range ctx.Cfg.FileNamingPattern.Required {
		found := false
		for _, e := range ctx.Tree.Entries {
			if e.IsDir {
				continue
			}
			if pathMatches(e.Name, requiredFile) || pathMatches(e.RelPath, requiredFile) {
				found = true
				break
			}
		}
		if found {
			ctx.V.printSuccess(fmt.Sprintf("Required file pattern found: %s", requiredFile))
			ctx.V.Successes++
		} else {
			msg := fmt.Sprintf("Required file pattern missing: %s", requiredFile)
			ctx.V.addViolation("missing_required_file", "error", requiredFile, requiredFile, msg)
		}
	}
}

// placementRule iterates Tree files and applies placement rules. Preserves
// the counter quirk from the legacy walk: v.Successes++ is called for
// every (file, rule) pair that passes — a file matching N placement rules
// contributes N successes.
type placementRule struct{}

func (placementRule) Name() string { return "placement" }

func (placementRule) Run(ctx *RunContext) {
	for _, e := range ctx.Tree.Entries {
		if e.IsDir {
			continue
		}
		if ctx.Skip(e.RelPath, false) {
			continue
		}
		for _, rule := range ctx.Cfg.Placement {
			if !matchesAnyFile(e.RelPath, e.Name, rule.Files) {
				continue
			}
			if underAny(e.RelPath, rule.MustBeUnder) {
				ctx.V.Successes++
				continue
			}
			msg := fmt.Sprintf("File placement violation: %s must be under %s",
				e.RelPath, strings.Join(rule.MustBeUnder, ", "))
			ctx.V.addViolation("placement_violation", severity(rule.Severity), e.RelPath, rule.ID, msg)
		}
	}
}

// requiredGroupsRule uses stat-based helpers (existsAt, existsAny,
// matchingDirs) which look through ignored directories on purpose — same
// quirk as requiredPathsRule.
type requiredGroupsRule struct{}

func (requiredGroupsRule) Name() string { return "required_groups" }

func (requiredGroupsRule) Run(ctx *RunContext) {
	ctx.V.validateRequiredGroupsDirect(ctx.Tree.Root)
}

// boundariesRule iterates supported source files in the Tree, parses
// imports, and reports violations. Preserves the counter quirk: Successes++
// is incremented once per (file, matching rule) pair even when that pair
// produced boundary violations, matching legacy behavior.
type boundariesRule struct{}

func (boundariesRule) Name() string { return "boundaries" }

func (boundariesRule) Run(ctx *RunContext) {
	modulePath := readGoModule(ctx.Tree.Root)
	for _, e := range ctx.Tree.Entries {
		if e.IsDir {
			continue
		}
		if !isSupportedBoundaryFile(e.RelPath) {
			continue
		}
		if ctx.Skip(e.RelPath, false) {
			continue
		}
		for _, rule := range ctx.Cfg.Boundaries {
			if !pathMatches(e.RelPath, rule.From) {
				continue
			}
			imports, err := sourceImports(filepath.Join(ctx.Tree.Root, e.RelPath), e.RelPath)
			if err != nil {
				ctx.V.addViolation("parse_error", "error", e.RelPath, rule.ID,
					fmt.Sprintf("Failed to parse imports: %s", err))
				continue
			}
			for _, imp := range imports {
				localImport := importToLocalPath(modulePath, imp, e.RelPath)
				for _, forbidden := range rule.CannotImport {
					if pathMatches(imp, forbidden) || pathMatches(localImport, forbidden) {
						msg := fmt.Sprintf("Boundary violation: %s imports %s", e.RelPath, imp)
						ctx.V.addViolation("boundary_violation", severity(rule.Severity), e.RelPath, rule.ID, msg)
					}
				}
			}
			ctx.V.Successes++
		}
	}
}
