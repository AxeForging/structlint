package suggest

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AxeForging/structlint/internal/config"
	"github.com/AxeForging/structlint/internal/validator"
)

// Analyze runs the validator against root using cfg, then turns each
// resulting violation into a proposal. Return value's ConfigDiff is
// populated when any config_add proposals were emitted.
func Analyze(cfg *config.Config, configPath, root string) (*Report, error) {
	// Run the engine ourselves so we control the reporter and don't
	// double-print. A tiny Reporter satisfies the same surface Validator
	// uses; here we borrow *Validator directly and just don't call any of
	// its print methods (Silent covers that).
	v := validator.New(cfg, nil)
	v.Silent = true
	v.Run(root)

	report := &Report{
		Version:    1,
		ConfigPath: configPath,
	}
	proposals := build(cfg, v.Violations)
	report.Proposals = dedupeAndSort(proposals)

	// Build the unified diff from config_add proposals only.
	if configPath != "" {
		diff, err := buildConfigDiff(configPath, report.Proposals)
		if err != nil {
			return nil, err
		}
		report.ConfigDiff = diff
	}
	return report, nil
}

// build produces proposals in the same order the engine emitted violations,
// deduping later in Analyze.
func build(cfg *config.Config, violations []validator.Violation) []Proposal {
	var out []Proposal
	for _, v := range violations {
		p, ok := propose(cfg, v)
		if !ok {
			continue
		}
		out = append(out, p)
	}
	return out
}

// propose maps a single violation to a proposal.
func propose(cfg *config.Config, v validator.Violation) (Proposal, bool) {
	reason := fmt.Sprintf("%s: %s", v.Code, v.Message)
	switch v.Code {
	case "unallowed_directory":
		return Proposal{
			Kind:    KindConfigAdd,
			Section: "dir_structure.allowedPaths",
			Value:   generalizeDirGlob(v.Path),
			Reason:  reason,
			Paths:   []string{v.Path},
		}, true
	case "unallowed_file_pattern":
		return Proposal{
			Kind:    KindConfigAdd,
			Section: "file_naming_pattern.allowed",
			Value:   fileNameToPattern(v.Path),
			Reason:  reason,
			Paths:   []string{v.Path},
		}, true
	case "disallowed_directory", "disallowed_file_pattern":
		// NEVER auto-loosen a deliberate prohibition.
		return Proposal{
			Kind:   KindNote,
			Reason: reason + " — the rule is deliberate; review it or remove the path",
			Paths:  []string{v.Path},
		}, true
	case "placement_violation":
		from, to, cmd := placementMove(cfg, v)
		return Proposal{
			Kind:    KindMove,
			From:    from,
			To:      to,
			Command: cmd,
			Reason:  reason,
			Paths:   []string{v.Path},
		}, true
	case "missing_required_directory", "missing_required_file", "missing_group_file":
		return Proposal{
			Kind:   KindCreate,
			Path:   v.Path,
			Reason: reason,
			Paths:  []string{v.Path},
		}, true
	default:
		// boundary_violation, missing_required_group*, parse_error, walk_error.
		return Proposal{
			Kind:   KindNote,
			Reason: reason + " — no mechanical fix; needs human judgment",
			Paths:  []string{v.Path},
		}, true
	}
}

// generalizeDirGlob turns a concrete directory path (e.g. "tools/gen") into
// a top-level allowedPaths entry ("tools/**"). Root-level directories that
// don't nest still get /** since we can't know from a single violation
// whether they'll ever have children.
func generalizeDirGlob(path string) string {
	if path == "" || path == "." {
		return "."
	}
	top := strings.SplitN(filepath.ToSlash(path), "/", 2)[0]
	return top + "/**"
}

// fileNameToPattern maps a violating file to its allowed pattern:
// extensions become "*.ext", extensionless names stay exact.
func fileNameToPattern(path string) string {
	base := filepath.Base(path)
	dot := strings.LastIndex(base, ".")
	if dot <= 0 {
		return base
	}
	return "*" + base[dot:]
}

func placementMove(cfg *config.Config, v validator.Violation) (from, to, cmd string) {
	from = v.Path
	for _, rule := range cfg.Placement {
		if rule.ID != v.Rule || len(rule.MustBeUnder) == 0 {
			continue
		}
		targetGlob := rule.MustBeUnder[0]
		targetDir := strings.TrimSuffix(strings.TrimSuffix(targetGlob, "**"), "/")
		to = filepath.ToSlash(filepath.Join(targetDir, filepath.Base(from)))
		break
	}
	if to == "" {
		to = from // no rule match; caller sees identical from/to and picks
	}
	cmd = fmt.Sprintf("git mv %s %s", from, to)
	return from, to, cmd
}

// dedupeAndSort collapses duplicate config_add proposals (many files, one
// entry) and sorts everything for deterministic output: config_add first
// (by section, then value), then move (by from), then create (by path),
// then note (by first path).
func dedupeAndSort(proposals []Proposal) []Proposal {
	seenAdd := map[string]int{}
	var out []Proposal
	for _, p := range proposals {
		if p.Kind == KindConfigAdd {
			key := p.Section + "|" + p.Value
			if idx, ok := seenAdd[key]; ok {
				out[idx].Paths = append(out[idx].Paths, p.Paths...)
				continue
			}
			seenAdd[key] = len(out)
			out = append(out, p)
			continue
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return kindOrder(out[i].Kind) < kindOrder(out[j].Kind)
		}
		switch out[i].Kind {
		case KindConfigAdd:
			if out[i].Section != out[j].Section {
				return out[i].Section < out[j].Section
			}
			return out[i].Value < out[j].Value
		case KindMove:
			return out[i].From < out[j].From
		case KindCreate:
			return out[i].Path < out[j].Path
		}
		return firstPath(out[i]) < firstPath(out[j])
	})
	for i := range out {
		if len(out[i].Paths) > 1 {
			sort.Strings(out[i].Paths)
		}
	}
	return out
}

func kindOrder(k Kind) int {
	switch k {
	case KindConfigAdd:
		return 0
	case KindMove:
		return 1
	case KindCreate:
		return 2
	default:
		return 3
	}
}

func firstPath(p Proposal) string {
	if len(p.Paths) == 0 {
		return ""
	}
	return p.Paths[0]
}
