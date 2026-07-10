package validator

import (
	"os"
	"path/filepath"
)

// Entry describes a single filesystem entry visited by Snapshot.
type Entry struct {
	RelPath string // slash-normalized; "." for the root
	Name    string
	IsDir   bool
	Abs     string
}

// Tree is an ignore-filtered snapshot of a project root, ready for
// rule enumeration without another filesystem walk.
type Tree struct {
	Root    string  // absolute root path
	Entries []Entry // filepath.Walk lexical order, ignore-filtered
	dirs    map[string]struct{}
	files   map[string]struct{}
	WalkErr error // first walk error, if any
}

// HasDir reports whether the tree contains a directory at relPath.
func (t *Tree) HasDir(relPath string) bool {
	if t == nil {
		return false
	}
	_, ok := t.dirs[relPath]
	return ok
}

// HasFile reports whether the tree contains a file at relPath.
func (t *Tree) HasFile(relPath string) bool {
	if t == nil {
		return false
	}
	_, ok := t.files[relPath]
	return ok
}

// Snapshot walks root once and returns an ignore-filtered Tree. Ignore
// matching uses the same pathMatches helper the individual rule walks use,
// so the snapshot's visible set matches the union of what those walks would
// visit. Rules that intentionally look into ignored directories (see the
// requiredPathsRule os.Stat quirk documented in spec 005) do NOT consult
// the tree for those lookups.
func Snapshot(root string, ignore []string) *Tree {
	cleaned := cleanRoot(root)
	tree := &Tree{
		Root:  cleaned,
		dirs:  map[string]struct{}{},
		files: map[string]struct{}{},
	}
	err := filepath.Walk(root, func(currentPath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			if tree.WalkErr == nil {
				tree.WalkErr = walkErr
			}
			return walkErr
		}
		rel := relativePath(cleaned, currentPath)
		for _, pat := range ignore {
			if pathMatches(rel, pat) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		entry := Entry{
			RelPath: rel,
			Name:    info.Name(),
			IsDir:   info.IsDir(),
			Abs:     currentPath,
		}
		tree.Entries = append(tree.Entries, entry)
		if info.IsDir() {
			tree.dirs[rel] = struct{}{}
		} else {
			tree.files[rel] = struct{}{}
		}
		return nil
	})
	if err != nil && tree.WalkErr == nil {
		tree.WalkErr = err
	}
	return tree
}
