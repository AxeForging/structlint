package hooks

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// InstallLefthook adds a `structlint` command under `pre-commit.commands` in
// lefthook.yml (or lefthook.yaml). Idempotent, comment-preserving, and
// refuses when the file uses YAML anchors/aliases (round-trip loses them).
func InstallLefthook(dir string, dryRun bool) (Result, error) {
	path, existing := findLefthookFile(dir)
	res := Result{Type: TypeLefthook, File: path}

	var root yaml.Node
	if existing {
		data, err := os.ReadFile(path)
		if err != nil {
			return res, fmt.Errorf("read lefthook config: %w", err)
		}
		if err := yaml.Unmarshal(data, &root); err != nil {
			return res, fmt.Errorf("parse lefthook config: %w", err)
		}
		if hasAnchorOrAlias(&root) {
			res.Action = ActionRefused
			res.Reason = "lefthook.yml uses YAML anchors/aliases; refusing to rewrite. " +
				"Add this snippet manually:\n\n" + lefthookSnippet()
			return res, nil
		}
	} else {
		root.Kind = yaml.DocumentNode
	}

	doc := ensureDocumentMapping(&root)
	preCommit := ensureChildMapping(doc, "pre-commit")
	commands := ensureChildMapping(preCommit, "commands")

	if hasMapKey(commands, "structlint") {
		res.Action = ActionAlreadyInstalled
		res.Reason = "structlint already present under pre-commit.commands"
		return res, nil
	}

	appendMapEntry(commands, "structlint", newStructlintCommandNode())

	out, err := marshalDocument(&root)
	if err != nil {
		return res, fmt.Errorf("re-encode lefthook config: %w", err)
	}

	if dryRun {
		res.Action = ActionInstalled
		res.Preview = string(out)
		return res, nil
	}

	if err := writeFileAtomic(path, out, 0o644); err != nil {
		return res, fmt.Errorf("write lefthook config: %w", err)
	}
	res.Action = ActionInstalled
	return res, nil
}

func findLefthookFile(dir string) (path string, existing bool) {
	for _, name := range []string{"lefthook.yml", "lefthook.yaml"} {
		p := filepath.Join(dir, name)
		if fileExists(p) {
			return p, true
		}
	}
	return filepath.Join(dir, "lefthook.yml"), false
}

func lefthookSnippet() string {
	return `pre-commit:
  commands:
    structlint:
      run: ` + HookRun + "\n"
}

func newStructlintCommandNode() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "run"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: HookRun},
		},
	}
}

// ensureDocumentMapping returns the root mapping node, creating it if the
// document was empty.
func ensureDocumentMapping(root *yaml.Node) *yaml.Node {
	if root.Kind == 0 {
		root.Kind = yaml.DocumentNode
	}
	if len(root.Content) == 0 {
		root.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		// yaml.v3 can produce a null scalar for an empty document.
		doc.Kind = yaml.MappingNode
		doc.Tag = "!!map"
		doc.Value = ""
	}
	return doc
}

// ensureChildMapping returns the mapping child under key, creating it if absent.
func ensureChildMapping(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			child := parent.Content[i+1]
			if child.Kind != yaml.MappingNode {
				child.Kind = yaml.MappingNode
				child.Tag = "!!map"
				child.Value = ""
			}
			return child
		}
	}
	child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		child,
	)
	return child
}

// appendMapEntry appends a key/value pair to a mapping node.
func appendMapEntry(m *yaml.Node, key string, value *yaml.Node) {
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

// hasMapKey reports whether a mapping already contains the given key.
func hasMapKey(m *yaml.Node, key string) bool {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return true
		}
	}
	return false
}

// hasAnchorOrAlias walks a yaml.Node tree looking for anchors or aliases;
// their presence means we can't safely round-trip the file.
func hasAnchorOrAlias(n *yaml.Node) bool {
	if n == nil {
		return false
	}
	if n.Anchor != "" || n.Kind == yaml.AliasNode {
		return true
	}
	for _, c := range n.Content {
		if hasAnchorOrAlias(c) {
			return true
		}
	}
	return false
}

func marshalDocument(root *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeFileAtomic writes data to path via a temp file + rename so a partial
// write can never leave the target half-updated.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".structlint-hook-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// errUnexpected is a sentinel for programmer errors we don't want to expose
// as raw yaml errors; kept unexported so callers can't type-assert on it.
var errUnexpected = errors.New("unexpected AST shape")

var _ = errUnexpected // reserved for future guardrails
