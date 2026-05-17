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
	DirStructure      DirStructure      `yaml:"dir_structure" json:"dir_structure"`
	FileNamingPattern FileNamingPattern `yaml:"file_naming_pattern" json:"file_naming_pattern"`
	Ignore            []string          `yaml:"ignore" json:"ignore"`
	Placement         []PlacementRule   `yaml:"placement" json:"placement"`
	RequiredGroups    []RequiredGroup   `yaml:"requiredGroups" json:"requiredGroups"`
	Boundaries        []BoundaryRule    `yaml:"boundaries" json:"boundaries"`
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

// LoadConfig loads the configuration from a file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	ext := filepath.Ext(path)
	if ext == ".yaml" || ext == ".yml" {
		err = yaml.UnmarshalStrict(data, &config)
	} else {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&config)
	}

	if err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
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
