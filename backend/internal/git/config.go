// Package git provides Git repository management functionality for cloning and syncing Obsidian vaults
package git

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds Git repository configuration
type Config struct {
	// Repository details
	RepoURL   string `yaml:"url"`        // GitHub repository URL
	Branch    string `yaml:"branch"`     // Branch to track (default: main)
	LocalPath string `yaml:"local_path"` // Where to clone locally

	// Authentication
	SSHKeyPath string `yaml:"ssh_key_path"` // Path to SSH key (optional, will try defaults)

	// Sync settings
	AutoSync     bool          `yaml:"auto_sync"`     // Enable automatic syncing
	SyncInterval time.Duration `yaml:"sync_interval"` // How often to check for updates

	// Performance
	ShallowClone bool `yaml:"shallow_clone"` // Clone with depth=1 for faster initial clone
	SingleBranch bool `yaml:"single_branch"` // Clone only the specified branch
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Branch:       "main",
		LocalPath:    "./vault_clone",
		SyncInterval: 5 * time.Minute,
		AutoSync:     true,
		ShallowClone: true, // Shallow clone for faster performance
		SingleBranch: true, // Only clone the branch we need
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.RepoURL == "" {
		return ErrNoRepoURL
	}
	if c.LocalPath == "" {
		return ErrNoLocalPath
	}
	if c.Branch == "" {
		return fmt.Errorf("branch is required")
	}
	return nil
}

// LoadConfigFromYAML loads configuration from a YAML file
func LoadConfigFromYAML(path string) (*Config, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is controlled by application
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Start with defaults
	config := DefaultConfig()

	// Unmarshal YAML over defaults (non-zero values will override)
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadConfigFromYAMLOrDefault loads config from YAML file or returns default if file doesn't exist
func LoadConfigFromYAMLOrDefault(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	return LoadConfigFromYAML(path)
}
