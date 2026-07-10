package hooks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AxeForging/structlint/internal/build"
	"gopkg.in/yaml.v3"
)

// PreCommitRepoURL is the canonical URL pre-commit will point at.
const PreCommitRepoURL = "https://github.com/AxeForging/structlint"

// InstallPreCommit adds a repo entry for structlint to .pre-commit-config.yaml.
// Idempotent (skips when structlint is already listed) and refuses on
// anchors/aliases.
func InstallPreCommit(dir string, dryRun bool) (Result, error) {
	path := filepath.Join(dir, ".pre-commit-config.yaml")
	res := Result{Type: TypePreCommit, File: path}

	var root yaml.Node
	if fileExists(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return res, fmt.Errorf("read pre-commit config: %w", err)
		}
		if err := yaml.Unmarshal(data, &root); err != nil {
			return res, fmt.Errorf("parse pre-commit config: %w", err)
		}
		if hasAnchorOrAlias(&root) {
			res.Action = ActionRefused
			res.Reason = ".pre-commit-config.yaml uses YAML anchors/aliases; refusing to rewrite. " +
				"Add this snippet manually:\n\n" + preCommitSnippet()
			return res, nil
		}
	} else {
		root.Kind = yaml.DocumentNode
	}

	doc := ensureDocumentMapping(&root)
	repos := ensureChildSequence(doc, "repos")

	if repoAlreadyPresent(repos) {
		res.Action = ActionAlreadyInstalled
		res.Reason = "structlint already listed under repos"
		return res, nil
	}

	repos.Content = append(repos.Content, newPreCommitRepoNode())

	out, err := marshalDocument(&root)
	if err != nil {
		return res, fmt.Errorf("re-encode pre-commit config: %w", err)
	}

	if dryRun {
		res.Action = ActionInstalled
		res.Preview = string(out)
		return res, nil
	}

	if err := writeFileAtomic(path, out, 0o644); err != nil {
		return res, fmt.Errorf("write pre-commit config: %w", err)
	}
	res.Action = ActionInstalled
	return res, nil
}

// ensureChildSequence returns the sequence child under key, creating it if absent.
func ensureChildSequence(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			child := parent.Content[i+1]
			if child.Kind != yaml.SequenceNode {
				child.Kind = yaml.SequenceNode
				child.Tag = "!!seq"
				child.Value = ""
				child.Content = nil
			}
			return child
		}
	}
	child := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		child,
	)
	return child
}

// repoAlreadyPresent checks whether any repo entry already refers to structlint.
// Matches by canonical repo URL OR by any hooks[].id == "structlint" so a
// hand-crafted entry using a fork URL still counts.
func repoAlreadyPresent(repos *yaml.Node) bool {
	for _, entry := range repos.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(entry.Content); i += 2 {
			k := entry.Content[i].Value
			v := entry.Content[i+1]
			if k == "repo" && v.Kind == yaml.ScalarNode && strings.EqualFold(v.Value, PreCommitRepoURL) {
				return true
			}
			if k == "hooks" && v.Kind == yaml.SequenceNode {
				for _, hook := range v.Content {
					if hook.Kind != yaml.MappingNode {
						continue
					}
					for j := 0; j+1 < len(hook.Content); j += 2 {
						if hook.Content[j].Value == "id" && hook.Content[j+1].Value == "structlint" {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// preCommitRev returns the value we put in `rev:` for the pinned hook. Uses
// the ldflags-injected build version when available; falls back to "main"
// when running an unstamped dev binary.
func preCommitRev() string {
	v := strings.TrimSpace(build.Version)
	if v == "" || v == "dev" {
		return "main"
	}
	return v
}

func newPreCommitRepoNode() *yaml.Node {
	hook := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "id"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "structlint"},
		},
	}
	hooks := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: []*yaml.Node{hook},
	}
	entry := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "repo"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: PreCommitRepoURL},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "rev"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: preCommitRev()},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "hooks"},
			hooks,
		},
	}
	return entry
}

func preCommitSnippet() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "repos:\n")
	fmt.Fprintf(&b, "  - repo: %s\n", PreCommitRepoURL)
	fmt.Fprintf(&b, "    rev: %s\n", preCommitRev())
	fmt.Fprintf(&b, "    hooks:\n")
	fmt.Fprintf(&b, "      - id: structlint\n")
	return b.String()
}
