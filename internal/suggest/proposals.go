// Package suggest turns validator violations into actionable proposals —
// either a config change (rendered as a unified diff against the actual
// config file) or a filesystem action (git mv, create). It never writes
// anything; the caller decides what to apply.
package suggest

// Kind identifies what shape a proposal takes.
type Kind string

const (
	KindConfigAdd Kind = "config_add"
	KindMove      Kind = "move"
	KindCreate    Kind = "create"
	KindNote      Kind = "note"
)

// Proposal describes one recommended change. Depending on Kind, different
// fields are populated:
//   - config_add: Section, Value
//   - move:       From, To, Command
//   - create:     Path
//   - note:       (no fields beyond Reason/Paths)
type Proposal struct {
	Kind    Kind     `json:"kind"`
	Section string   `json:"section,omitempty"`
	Value   string   `json:"value,omitempty"`
	From    string   `json:"from,omitempty"`
	To      string   `json:"to,omitempty"`
	Command string   `json:"command,omitempty"`
	Path    string   `json:"path,omitempty"`
	Reason  string   `json:"reason"`
	Paths   []string `json:"paths"`
}

// Report is the versioned output surface. Callers agree on shape via the
// version field; breaking changes bump it.
type Report struct {
	Version    int        `json:"version"`
	ConfigPath string     `json:"configPath"`
	Proposals  []Proposal `json:"proposals"`
	ConfigDiff string     `json:"configDiff,omitempty"`
}
