package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Config represents the main configuration structure.
type Config struct {
	Extends           ExtendsList       `yaml:"extends" json:"extends"`
	DirStructure      DirStructure      `yaml:"dir_structure" json:"dir_structure"`
	FileNamingPattern FileNamingPattern `yaml:"file_naming_pattern" json:"file_naming_pattern"`
	Ignore            []string          `yaml:"ignore" json:"ignore"`
	Placement         []PlacementRule   `yaml:"placement" json:"placement"`
	RequiredGroups    []RequiredGroup   `yaml:"requiredGroups" json:"requiredGroups"`
	Boundaries        []BoundaryRule    `yaml:"boundaries" json:"boundaries"`
}

// ExtendsList accepts either a single string or a list of strings. Each
// entry is a built-in preset name or a filesystem path relative to the
// extending config file.
type ExtendsList []string

// UnmarshalYAML supports the scalar-or-sequence shape.
func (e *ExtendsList) UnmarshalYAML(unmarshal func(any) error) error {
	var single string
	if err := unmarshal(&single); err == nil {
		*e = ExtendsList{single}
		return nil
	}
	var list []string
	if err := unmarshal(&list); err != nil {
		return err
	}
	*e = ExtendsList(list)
	return nil
}

// UnmarshalJSON mirrors UnmarshalYAML for JSON configs.
func (e *ExtendsList) UnmarshalJSON(data []byte) error {
	trim := bytes.TrimSpace(data)
	if len(trim) > 0 && trim[0] == '"' {
		var single string
		if err := json.Unmarshal(data, &single); err != nil {
			return err
		}
		*e = ExtendsList{single}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	*e = ExtendsList(list)
	return nil
}

// DirStructure represents the directory structure validation rules.
type DirStructure struct {
	AllowedPaths    []string `yaml:"allowedPaths" json:"allowedPaths"`
	DisallowedPaths []string `yaml:"disallowedPaths" json:"disallowedPaths"`
	RequiredPaths   []string `yaml:"requiredPaths" json:"requiredPaths"`
}

// FileNamingPattern represents the file naming pattern validation rules.
type FileNamingPattern struct {
	Allowed    []string `yaml:"allowed" json:"allowed"`
	Disallowed []string `yaml:"disallowed" json:"disallowed"`
	Required   []string `yaml:"required" json:"required"`
}

// PlacementRule requires matching files to live under one of the allowed roots.
type PlacementRule struct {
	ID          string   `yaml:"id" json:"id"`
	Files       []string `yaml:"files" json:"files"`
	MustBeUnder []string `yaml:"mustBeUnder" json:"mustBeUnder"`
	Severity    string   `yaml:"severity" json:"severity"`
}

// RequiredGroup supports higher-level required-file contracts.
type RequiredGroup struct {
	ID               string   `yaml:"id" json:"id"`
	OneOf            []string `yaml:"oneOf" json:"oneOf"`
	EachDirMatching  string   `yaml:"eachDirMatching" json:"eachDirMatching"`
	MustContain      []string `yaml:"mustContain" json:"mustContain"`
	MustContainOneOf []string `yaml:"mustContainOneOf" json:"mustContainOneOf"`
	RequireMatch     bool     `yaml:"requireMatch" json:"requireMatch"`
	Severity         string   `yaml:"severity" json:"severity"`
}

// BoundaryRule blocks imports across configured source boundaries.
type BoundaryRule struct {
	ID           string   `yaml:"id" json:"id"`
	From         string   `yaml:"from" json:"from"`
	CannotImport []string `yaml:"cannotImport" json:"cannotImport"`
	Severity     string   `yaml:"severity" json:"severity"`
}

// LoadConfig loads the configuration from a file, resolving any `extends`
// chain (built-in presets or paths relative to the extending file) and
// merging parents-first. Strict parsing is preserved throughout.
func LoadConfig(path string) (*Config, error) {
	cfg, err := loadResolved(path, map[string]bool{}, 0)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// parseConfigFile reads and strict-parses a single file. It does NOT resolve
// extends or run Validate — that's loadResolved's job.
func parseConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Version pragma is checked BEFORE strict parsing so users of newer
	// features (e.g. `extends`) get an actionable "upgrade required"
	// error instead of a raw yaml `field ... not found` message.
	if err := enforceRequiresComment(path, data); err != nil {
		return nil, err
	}
	return parseConfigBytes(data, filepath.Ext(path))
}

func parseConfigBytes(data []byte, ext string) (*Config, error) {
	var cfg Config
	var err error
	if ext == ".yaml" || ext == ".yml" || ext == "" {
		err = yaml.UnmarshalStrict(data, &cfg)
	} else {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&cfg)
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate catches malformed rule definitions before repository walking starts.
func (c *Config) Validate() error {
	for i, rule := range c.Placement {
		if rule.ID == "" {
			return fmt.Errorf("placement[%d] missing id", i)
		}
		if len(rule.Files) == 0 {
			return fmt.Errorf("placement[%s] must define files", rule.ID)
		}
		if len(rule.MustBeUnder) == 0 {
			return fmt.Errorf("placement[%s] must define mustBeUnder", rule.ID)
		}
	}

	for i, group := range c.RequiredGroups {
		if group.ID == "" {
			return fmt.Errorf("requiredGroups[%d] missing id", i)
		}
		if len(group.OneOf) == 0 && group.EachDirMatching == "" {
			return fmt.Errorf("requiredGroups[%s] must define oneOf or eachDirMatching", group.ID)
		}
		if group.EachDirMatching != "" && len(group.MustContain) == 0 && len(group.MustContainOneOf) == 0 {
			return fmt.Errorf("requiredGroups[%s] must define mustContain or mustContainOneOf", group.ID)
		}
	}

	for i, rule := range c.Boundaries {
		if rule.ID == "" {
			return fmt.Errorf("boundaries[%d] missing id", i)
		}
		if rule.From == "" {
			return fmt.Errorf("boundaries[%s] must define from", rule.ID)
		}
		if len(rule.CannotImport) == 0 {
			return fmt.Errorf("boundaries[%s] must define cannotImport", rule.ID)
		}
	}

	return nil
}
