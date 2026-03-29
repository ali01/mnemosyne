// Package config provides configuration management for Mnemosyne.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Port      int      `yaml:"port"`
	Vaults    []string `yaml:"vaults"`
	HomeGraph string   `yaml:"home-graph,omitempty"` // e.g. "walros/memex"
}

// NodeClassificationConfig holds node type and classification rule configuration.
// Lives in GRAPH.yaml files, not the global config.
type NodeClassificationConfig struct {
	NodeTypes           map[string]NodeTypeConfig  `yaml:"node_types"`
	ClassificationRules []ClassificationRuleConfig `yaml:"classification_rules"`
	DefaultNodeType     string                     `yaml:"default_node_type,omitempty"`
}

// NodeTypeConfig defines the display properties for a node type.
type NodeTypeConfig struct {
	DisplayName    string  `yaml:"display_name"`
	Description    string  `yaml:"description"`
	Color          string  `yaml:"color"`
	SizeMultiplier float64 `yaml:"size_multiplier"`
}

// ClassificationRuleConfig defines a classification rule.
type ClassificationRuleConfig struct {
	Name        string `yaml:"name"`
	Priority    int    `yaml:"priority"`
	Type        string `yaml:"type"` // tag, filename_prefix, filename_suffix, filename_match, path_contains, regex
	Pattern     string `yaml:"pattern"`
	NodeType    string `yaml:"node_type"`
	Description string `yaml:"description,omitempty"`
}

// DefaultConfigPath returns the default config file location.
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mnemosyne", "config.yaml")
}

// DBPath returns the fixed database path.
func DBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mnemosyne", "mnemosyne.db")
}

// Load reads and parses a config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		Port: 5555,
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Vaults) == 0 {
		return nil, fmt.Errorf("at least one vault path is required in 'vaults'")
	}

	for i, v := range cfg.Vaults {
		cfg.Vaults[i] = expandHome(v)
	}

	return cfg, nil
}

// CreateDefault writes a new config file with the given vault path.
func CreateDefault(cfgPath, vaultPath string) error {
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	cfg := Config{Port: 5555, Vaults: []string{vaultPath}}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(cfgPath, data, 0o644)
}

// ExpandHome expands a leading ~/ to the user's home directory.
func ExpandHome(path string) string {
	return expandHome(path)
}

func expandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
