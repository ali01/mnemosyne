// Package config provides configuration management for the Mnemosyne application
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/git"
	"gopkg.in/yaml.v3"
)

// Config holds all application configuration loaded from YAML
type Config struct {
	Server   ServerConfig   `yaml:"server"`   // HTTP server settings
	Database DatabaseConfig `yaml:"database"` // PostgreSQL connection
	Git      git.Config     `yaml:"git"`      // Vault repository settings
	Graph    GraphConfig    `yaml:"graph"`    // Parsing and layout settings
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// GraphConfig holds graph processing configuration
type GraphConfig struct {
	// Layout algorithm settings
	Layout LayoutConfig `yaml:"layout"`

	// Cache settings
	Cache CacheConfig `yaml:"cache"`

	// Processing settings
	BatchSize      int `yaml:"batch_size"`      // Number of files to process at once
	MaxConcurrency int `yaml:"max_concurrency"` // Max concurrent goroutines

	// Node classification settings
	NodeClassification NodeClassificationConfig `yaml:"node_classification"`
}

// LayoutConfig holds graph layout algorithm configuration
type LayoutConfig struct {
	Algorithm       string  `yaml:"algorithm"`        // "force-directed", "hierarchical"
	Iterations      int     `yaml:"iterations"`       // Number of iterations for force-directed
	InitialTemp     float64 `yaml:"initial_temp"`     // Initial temperature
	CoolingRate     float64 `yaml:"cooling_rate"`     // Temperature cooling rate
	OptimalDistance float64 `yaml:"optimal_distance"` // Optimal node distance
}

// CacheConfig holds caching configuration
type CacheConfig struct {
	Enabled       bool          `yaml:"enabled"`
	TTL           time.Duration `yaml:"ttl"`
	MaxMemorySize int64         `yaml:"max_memory_size"` // Max memory for cache in bytes
}

// NodeClassificationConfig holds node type and classification rule configuration
type NodeClassificationConfig struct {
	NodeTypes           map[string]NodeTypeConfig      `yaml:"node_types"`
	ClassificationRules []ClassificationRuleConfig     `yaml:"classification_rules"`
	DefaultNodeType     string                         `yaml:"default_node_type,omitempty"`
}

// NodeTypeConfig defines the display properties for a node type
type NodeTypeConfig struct {
	DisplayName     string  `yaml:"display_name"`
	Description     string  `yaml:"description"`
	Color           string  `yaml:"color"`
	SizeMultiplier  float64 `yaml:"size_multiplier"`
}

// ClassificationRuleConfig defines a classification rule
type ClassificationRuleConfig struct {
	Name        string `yaml:"name"`
	Priority    int    `yaml:"priority"`
	Type        string `yaml:"type"` // tag, filename_prefix, filename_suffix, filename_match, path_contains, regex
	Pattern     string `yaml:"pattern"`
	NodeType    string `yaml:"node_type"`
	Description string `yaml:"description,omitempty"`
}

// DefaultConfig returns configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "mnemosyne",
			Password: "mnemosyne",
			DBName:   "mnemosyne",
			SSLMode:  "disable",
		},
		Git: git.Config{
			Branch:       "main",
			LocalPath:    "data/memex-clone",
			SyncInterval: 5 * time.Minute,
			AutoSync:     true,
			ShallowClone: true,
			SingleBranch: true,
		},
		Graph: GraphConfig{
			Layout: LayoutConfig{
				Algorithm:       "force-directed",
				Iterations:      500,
				InitialTemp:     1000.0,
				CoolingRate:     0.95,
				OptimalDistance: 100.0,
			},
			Cache: CacheConfig{
				Enabled:       true,
				TTL:           30 * time.Minute,
				MaxMemorySize: 100 * 1024 * 1024, // 100MB
			},
			BatchSize:      100,
			MaxConcurrency: 4,
			NodeClassification: NodeClassificationConfig{
				NodeTypes:           make(map[string]NodeTypeConfig),
				ClassificationRules: []ClassificationRuleConfig{},
			},
		},
	}
}

// LoadFromYAML loads configuration from a YAML file with defaults
func LoadFromYAML(path string) (*Config, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is controlled by application
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Start with defaults, overlay YAML values
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate Git config
	if err := c.Git.Validate(); err != nil {
		return fmt.Errorf("git config validation failed: %w", err)
	}

	if c.Graph.BatchSize <= 0 {
		return fmt.Errorf("graph batch size must be positive")
	}

	if c.Graph.MaxConcurrency <= 0 {
		return fmt.Errorf("graph max concurrency must be positive")
	}

	return nil
}

// GetDBConfig converts database config to db.Config
func (c *Config) GetDBConfig() db.Config {
	return db.Config{
		Host:     c.Database.Host,
		Port:     c.Database.Port,
		User:     c.Database.User,
		Password: c.Database.Password,
		DBName:   c.Database.DBName,
		SSLMode:  c.Database.SSLMode,
	}
}
