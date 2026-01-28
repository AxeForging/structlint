package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Config represents the main configuration structure.
type Config struct {
	DirStructure      DirStructure      `yaml:"dir_structure" json:"dir_structure"`
	FileNamingPattern FileNamingPattern `yaml:"file_naming_pattern" json:"file_naming_pattern"`
	Ignore            []string          `yaml:"ignore" json:"ignore"`
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

// LoadConfig loads the configuration from a file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	ext := filepath.Ext(path)
	if ext == ".yaml" || ext == ".yml" {
		err = yaml.Unmarshal(data, &config)
	} else {
		err = json.Unmarshal(data, &config)
	}

	if err != nil {
		return nil, err
	}

	return &config, nil
}
