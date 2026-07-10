package config

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed presets/*.yaml
var presetFS embed.FS

// presetNames maps a preset id (as it appears in `extends:`) to its
// embedded file. Keys are stable and part of the config surface.
var presetNames = map[string]string{
	"go-standard":     "presets/go-standard.yaml",
	"node-standard":   "presets/node-standard.yaml",
	"python-standard": "presets/python-standard.yaml",
	"generic":         "presets/generic.yaml",
}

const maxExtendsDepth = 10

// loadResolved parses path, resolves its extends chain depth-first with
// parents applied first, and returns the merged config. Validate() is NOT
// run here — the top-level LoadConfig runs it on the final merged result.
//
// visited tracks fully-resolved keys along the current chain for cycle
// detection. A key is a preset name as-is, or the absolute path for a file.
func loadResolved(path string, visited map[string]bool, depth int) (*Config, error) {
	if depth > maxExtendsDepth {
		return nil, fmt.Errorf("extends chain too deep (max %d)", maxExtendsDepth)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if visited[absPath] {
		return nil, fmt.Errorf("extends cycle detected: %s", describeCycle(visited, absPath))
	}

	cfg, err := parseConfigFile(path)
	if err != nil {
		return nil, err
	}

	if len(cfg.Extends) == 0 {
		return cfg, nil
	}

	// Mark this file as visited before recursing so cycles that pass
	// through it are detected. Unmark on return to allow diamond shapes
	// where two branches legitimately extend the same base.
	visited[absPath] = true
	defer delete(visited, absPath)

	extendingDir := filepath.Dir(absPath)
	merged := &Config{}
	for _, entry := range cfg.Extends {
		parent, err := resolveExtendEntry(entry, extendingDir, visited, depth+1)
		if err != nil {
			return nil, err
		}
		merged = mergeConfigs(merged, parent)
	}

	// Child (this file) overlays on top of all parents.
	cfg.Extends = nil
	return mergeConfigs(merged, cfg), nil
}

// resolveExtendEntry returns the merged config for a single extends entry —
// either a preset name (bytes come from the embedded FS) or a filesystem
// path relative to extendingDir.
func resolveExtendEntry(entry, extendingDir string, visited map[string]bool, depth int) (*Config, error) {
	if presetPath, ok := presetNames[entry]; ok {
		if visited["preset:"+entry] {
			return nil, fmt.Errorf("extends cycle detected: preset %q", entry)
		}
		visited["preset:"+entry] = true
		defer delete(visited, "preset:"+entry)

		data, err := presetFS.ReadFile(presetPath)
		if err != nil {
			return nil, fmt.Errorf("read preset %q: %w", entry, err)
		}
		cfg, err := parseConfigBytes(data, ".yaml")
		if err != nil {
			return nil, fmt.Errorf("parse preset %q: %w", entry, err)
		}
		if len(cfg.Extends) != 0 {
			// Presets themselves must not use extends — enforced at review time,
			// but reject at run time too so a broken embedded preset is obvious.
			return nil, fmt.Errorf("preset %q uses extends; presets must be flat", entry)
		}
		return cfg, nil
	}

	// Anything else is a filesystem path relative to the extending file.
	resolved := entry
	if !filepath.IsAbs(entry) {
		resolved = filepath.Join(extendingDir, entry)
	}
	if _, statErr := os.Stat(resolved); statErr != nil {
		return nil, fmt.Errorf("extends entry %q not found (looked for %s); "+
			"valid presets: %s", entry, resolved, presetsList())
	}
	return loadResolved(resolved, visited, depth)
}

func presetsList() string {
	names := make([]string, 0, len(presetNames))
	for name := range presetNames {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func describeCycle(visited map[string]bool, revisit string) string {
	nodes := make([]string, 0, len(visited)+1)
	for k := range visited {
		nodes = append(nodes, k)
	}
	sort.Strings(nodes)
	return strings.Join(nodes, " -> ") + " -> " + revisit
}

// mergeConfigs overlays child onto parent following the rules:
// - simple string slices → parent entries first, then child entries not
//   already present (exact-string dedup, order stable).
// - Placement/RequiredGroups/Boundaries → keyed by ID; same ID → child
//   rule replaces the parent's wholesale; new IDs append.
// - Extends is consumed by the resolver and empty here.
func mergeConfigs(parent, child *Config) *Config {
	out := &Config{
		DirStructure: DirStructure{
			AllowedPaths:    dedupAppend(parent.DirStructure.AllowedPaths, child.DirStructure.AllowedPaths),
			DisallowedPaths: dedupAppend(parent.DirStructure.DisallowedPaths, child.DirStructure.DisallowedPaths),
			RequiredPaths:   dedupAppend(parent.DirStructure.RequiredPaths, child.DirStructure.RequiredPaths),
		},
		FileNamingPattern: FileNamingPattern{
			Allowed:    dedupAppend(parent.FileNamingPattern.Allowed, child.FileNamingPattern.Allowed),
			Disallowed: dedupAppend(parent.FileNamingPattern.Disallowed, child.FileNamingPattern.Disallowed),
			Required:   dedupAppend(parent.FileNamingPattern.Required, child.FileNamingPattern.Required),
		},
		Ignore:         dedupAppend(parent.Ignore, child.Ignore),
		Placement:      mergePlacement(parent.Placement, child.Placement),
		RequiredGroups: mergeRequiredGroups(parent.RequiredGroups, child.RequiredGroups),
		Boundaries:     mergeBoundaries(parent.Boundaries, child.Boundaries),
	}
	return out
}

func dedupAppend(parent, child []string) []string {
	if len(parent) == 0 && len(child) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(parent)+len(child))
	out := make([]string, 0, len(parent)+len(child))
	for _, s := range parent {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range child {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func mergePlacement(parent, child []PlacementRule) []PlacementRule {
	byID := make(map[string]int)
	out := make([]PlacementRule, 0, len(parent)+len(child))
	for _, r := range parent {
		byID[r.ID] = len(out)
		out = append(out, r)
	}
	for _, r := range child {
		if idx, ok := byID[r.ID]; ok {
			out[idx] = r
			continue
		}
		byID[r.ID] = len(out)
		out = append(out, r)
	}
	return out
}

func mergeRequiredGroups(parent, child []RequiredGroup) []RequiredGroup {
	byID := make(map[string]int)
	out := make([]RequiredGroup, 0, len(parent)+len(child))
	for _, g := range parent {
		byID[g.ID] = len(out)
		out = append(out, g)
	}
	for _, g := range child {
		if idx, ok := byID[g.ID]; ok {
			out[idx] = g
			continue
		}
		byID[g.ID] = len(out)
		out = append(out, g)
	}
	return out
}

func mergeBoundaries(parent, child []BoundaryRule) []BoundaryRule {
	byID := make(map[string]int)
	out := make([]BoundaryRule, 0, len(parent)+len(child))
	for _, r := range parent {
		byID[r.ID] = len(out)
		out = append(out, r)
	}
	for _, r := range child {
		if idx, ok := byID[r.ID]; ok {
			out[idx] = r
			continue
		}
		byID[r.ID] = len(out)
		out = append(out, r)
	}
	return out
}
