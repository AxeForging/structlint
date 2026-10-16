package config

import (
	"fmt"

	"github.com/gobwas/glob"
)

// FileSize holds file-level size limits (spec 014). Rules are ordered: the
// first rule whose pattern matches a file decides its limit.
type FileSize struct {
	Rules   []FileSizeRule `yaml:"rules" json:"rules"`
	Exclude []string       `yaml:"exclude" json:"exclude"`
}

// FileSizeRule caps the line count of files matching Pattern.
type FileSizeRule struct {
	Pattern  string `yaml:"pattern" json:"pattern"`
	MaxLines int    `yaml:"maxLines" json:"maxLines"`
	Severity string `yaml:"severity" json:"severity"`
}

// validate rejects rules that can never behave sensibly. Patterns are
// compiled here so a typo fails at load time instead of silently matching
// nothing during the walk.
func (f FileSize) validate() error {
	for i, rule := range f.Rules {
		if rule.Pattern == "" {
			return fmt.Errorf("file_size.rules[%d]: pattern is required", i)
		}
		if _, err := glob.Compile(rule.Pattern); err != nil {
			return fmt.Errorf("file_size.rules[%d]: invalid pattern %q: %v", i, rule.Pattern, err)
		}
		if rule.MaxLines <= 0 {
			return fmt.Errorf("file_size.rules[%d] (%s): maxLines must be greater than 0, got %d", i, rule.Pattern, rule.MaxLines)
		}
		switch rule.Severity {
		case "", "error", "warning":
		default:
			return fmt.Errorf("file_size.rules[%d] (%s): severity must be \"error\" or \"warning\", got %q", i, rule.Pattern, rule.Severity)
		}
	}
	for i, pattern := range f.Exclude {
		if pattern == "" {
			return fmt.Errorf("file_size.exclude[%d]: pattern is required", i)
		}
		if _, err := glob.Compile(pattern); err != nil {
			return fmt.Errorf("file_size.exclude[%d]: invalid pattern %q: %v", i, pattern, err)
		}
	}
	return nil
}

// mergeFileSize overlays child onto parent for `extends`. Because the first
// matching rule wins, child rules go first so a child can override a
// parent's limit; parent rules whose pattern the child redeclares are
// dropped, the rest follow in their original order. Exclude is a dedup
// append like every other string list.
func mergeFileSize(parent, child FileSize) FileSize {
	var rules []FileSizeRule
	if len(parent.Rules)+len(child.Rules) > 0 {
		rules = make([]FileSizeRule, 0, len(parent.Rules)+len(child.Rules))
		declared := make(map[string]struct{}, len(child.Rules))
		for _, r := range child.Rules {
			declared[r.Pattern] = struct{}{}
			rules = append(rules, r)
		}
		for _, r := range parent.Rules {
			if _, ok := declared[r.Pattern]; ok {
				continue
			}
			rules = append(rules, r)
		}
	}
	return FileSize{
		Rules:   rules,
		Exclude: dedupAppend(parent.Exclude, child.Exclude),
	}
}
