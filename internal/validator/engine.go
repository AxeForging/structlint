package validator

import "github.com/AxeForging/structlint/internal/config"

// Rule is the pluggable unit of validation. Each family (dir structure,
// file naming, placement, ...) is a Rule. Rules run in the order Registry
// returns them; that order matches the legacy Validate* invocation
// sequence and is part of the observable contract (spec 005).
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
	// global (required-* families) do not consult Skip.
	Skip func(relPath string, isDir bool) bool
}

// Registry returns the seven rules in legacy execution order:
// dir_structure, file_naming, required_paths, required_files,
// placement, required_groups, boundaries. Spec 010's suggest command
// enumerates rules through this registry.
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
// against the same snapshot. Wire from validate.go instead of calling the
// seven Validate* methods individually.
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

// The seven rule types below are thin wrappers around the existing
// Validate* method bodies. They call the legacy method, which walks the
// tree itself. Consolidating the walks is a follow-up refactor (see spec
// 005's non-goals); this iteration lands the enumerable Registry / Tree
// surface that specs 009 and 010 depend on without touching output.

type dirStructureRule struct{}

func (dirStructureRule) Name() string { return "dir_structure" }
func (dirStructureRule) Run(ctx *RunContext) {
	ctx.V.ValidateDirStructure(ctx.Tree.Root)
}

type fileNamingRule struct{}

func (fileNamingRule) Name() string { return "file_naming" }
func (fileNamingRule) Run(ctx *RunContext) {
	ctx.V.ValidateFileNaming(ctx.Tree.Root)
}

type requiredPathsRule struct{}

func (requiredPathsRule) Name() string { return "required_paths" }
func (requiredPathsRule) Run(ctx *RunContext) {
	ctx.V.ValidateRequiredPaths(ctx.Tree.Root)
}

type requiredFilesRule struct{}

func (requiredFilesRule) Name() string { return "required_files" }
func (requiredFilesRule) Run(ctx *RunContext) {
	ctx.V.ValidateRequiredFiles(ctx.Tree.Root)
}

type placementRule struct{}

func (placementRule) Name() string { return "placement" }
func (placementRule) Run(ctx *RunContext) {
	ctx.V.ValidatePlacement(ctx.Tree.Root)
}

type requiredGroupsRule struct{}

func (requiredGroupsRule) Name() string { return "required_groups" }
func (requiredGroupsRule) Run(ctx *RunContext) {
	ctx.V.ValidateRequiredGroups(ctx.Tree.Root)
}

type boundariesRule struct{}

func (boundariesRule) Name() string { return "boundaries" }
func (boundariesRule) Run(ctx *RunContext) {
	ctx.V.ValidateBoundaries(ctx.Tree.Root)
}
