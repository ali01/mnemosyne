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
	VaultPath          string                  `yaml:"vault_path"`
	Port               int                     `yaml:"port"`
	DBPath             string                  `yaml:"db_path"`
	NodeClassification *NodeClassificationConfig `yaml:"node_classification,omitempty"`
}

// NodeClassificationConfig holds node type and classification rule configuration.
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

// Load reads and parses a config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		Port:   5555,
		DBPath: defaultDBPath(),
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.VaultPath == "" {
		return nil, fmt.Errorf("vault_path is required")
	}

	// Expand ~ in paths
	cfg.VaultPath = expandHome(cfg.VaultPath)
	cfg.DBPath = expandHome(cfg.DBPath)

	return cfg, nil
}

func defaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mnemosyne", "mnemosyne.db")
}

func expandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
