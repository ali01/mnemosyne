// Package vault provides functionality for parsing and processing vault files
package vault

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ali01/mnemosyne/internal/config"
)

// Classification rule priorities
const (
	PriorityTag      = 1
	PriorityFilename = 2
	PriorityPath     = 3
)

// ClassificationRule defines a rule for classifying nodes
type ClassificationRule struct {
	Name     string
	Priority int // Lower number = higher priority
	Matcher  func(file *MarkdownFile) bool
	NodeType string // Node type from configuration
}

// NodeClassifier classifies vault nodes based on configurable rules
type NodeClassifier struct {
	rules           []ClassificationRule
	nodeTypes       map[string]config.NodeTypeConfig // Optional: for access to display properties
	defaultNodeType string                           // Default node type when no rules match
}

// NewNodeClassifier creates a new node classifier with no rules
// For classification to work, use NewNodeClassifierFromConfig with a proper configuration
func NewNodeClassifier() *NodeClassifier {
	return &NodeClassifier{
		rules:           []ClassificationRule{},
		defaultNodeType: "",
	}
}

// NewNodeClassifierFromConfig creates a classifier from configuration
func NewNodeClassifierFromConfig(nodeClassConfig *config.NodeClassificationConfig) (*NodeClassifier, error) {
	rules, err := ConvertToClassificationRules(nodeClassConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert classification rules: %w", err)
	}

	// Sort rules by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	// Use configured default node type (empty string if not set)
	defaultType := nodeClassConfig.DefaultNodeType

	return &NodeClassifier{
		rules:           rules,
		nodeTypes:       nodeClassConfig.NodeTypes,
		defaultNodeType: defaultType,
	}, nil
}

// NewNodeClassifierWithRules creates a classifier with custom rules.
// Rules are validated before use to ensure they won't cause runtime errors.
// The rules are sorted by priority (ascending) to ensure correct evaluation order.
//
// Example:
//
//	rules := []ClassificationRule{
//	    {Name: "readme", Priority: 1, Matcher: isReadmeFile, NodeType: "index"},
//	    {Name: "archive", Priority: 3, Matcher: isInArchive, NodeType: "note"},
//	}
//	classifier, err := NewNodeClassifierWithRules(rules, "note")
//
// Returns an error if any rule fails validation.
func NewNodeClassifierWithRules(rules []ClassificationRule, defaultNodeType string) (*NodeClassifier, error) {
	// Validate custom rules
	if err := validateRules(rules); err != nil {
		return nil, err
	}

	// Sort rules by priority
	sortedRules := make([]ClassificationRule, len(rules))
	copy(sortedRules, rules)

	// Sort rules by priority using standard library
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority < sortedRules[j].Priority
	})

	return &NodeClassifier{
		rules:           sortedRules,
		defaultNodeType: defaultNodeType,
	}, nil
}

// ClassifyNode determines the type of a node based on classification rules.
// Rules are evaluated in priority order (lowest number = highest priority).
// The first matching rule determines the node type.
//
// Default classification priority:
//
//	PriorityTag (1): Tag-based rules (e.g., "index" tag)
//	PriorityFilename (2): Filename prefix rules (e.g., "~" prefix)
//	PriorityPath (3): Directory-based rules (e.g., "concepts/" directory)
//
// Returns the configured default node type if no rules match or if the file is nil.
func (nc *NodeClassifier) ClassifyNode(file *MarkdownFile) string {
	if file == nil {
		return nc.defaultNodeType
	}

	// Validate file path to prevent malicious patterns
	if !isValidPath(file.Path) {
		return nc.defaultNodeType
	}

	// Apply rules in priority order
	for _, rule := range nc.rules {
		if rule.Matcher(file) {
			return rule.NodeType
		}
	}

	// Default to configured default type if no rules match
	return nc.defaultNodeType
}

// hasTag checks if a file has a specific tag in its frontmatter.
// The comparison is case-insensitive and handles various tag formats:
//   - []string{"tag1", "tag2"}
//   - []any{"tag1", "tag2"}
//   - "single-tag"
//
// Returns false if frontmatter is nil or tags field is missing.
func hasTag(file *MarkdownFile, tag string) bool {
	if file.Frontmatter == nil {
		return false
	}

	// Check if tags field exists
	tagsInterface, exists := file.Frontmatter.Raw["tags"]
	if !exists {
		return false
	}

	// Handle different tag formats
	switch tags := tagsInterface.(type) {
	case []string:
		for _, t := range tags {
			if strings.EqualFold(t, tag) {
				return true
			}
		}
	case []any:
		for _, t := range tags {
			if str, ok := t.(string); ok && strings.EqualFold(str, tag) {
				return true
			}
		}
	case string:
		// Single tag as string
		return strings.EqualFold(tags, tag)
	}

	return false
}

// isInDirectory checks if a file path contains a specific directory name at any level.
// The comparison is case-insensitive and works with normalized paths.
//
// Examples:
//   - isInDirectory("/vault/concepts/idea.md", "concepts") returns true
//   - isInDirectory("/vault/CONCEPTS/idea.md", "concepts") returns true (case-insensitive)
//   - isInDirectory("/vault/misconception/idea.md", "concepts") returns false
//
// Returns false if dirName is empty.
func isInDirectory(filePath, dirName string) bool {
	// Empty directory name should not match anything
	if dirName == "" {
		return false
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(filePath)

	// Walk up the directory tree using filepath operations
	for dir := cleanPath; dir != "." && dir != "/"; dir = filepath.Dir(dir) {
		// Check if the current directory name matches (case-insensitive)
		if strings.EqualFold(filepath.Base(dir), dirName) {
			return true
		}
		
		// filepath.Dir returns "." for the current directory when it can't go up further
		// Avoid infinite loop by checking if we've reached the top
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return false
}

// validateRules checks that classification rules are valid and safe to use.
//
// Validation checks:
//   - Rule names must be non-empty
//   - Rule names must be unique (no duplicates)
//   - Matcher functions must not be nil
//   - NodeType values must be valid (one of the predefined types)
//   - Priority values must be non-negative
//
// Returns a RuleValidationError with details about the first validation failure,
// or nil if all rules are valid. Empty rule sets are considered valid.
func validateRules(rules []ClassificationRule) error {
	if len(rules) == 0 {
		return nil // Empty rules are valid
	}

	// Check for duplicate names
	names := make(map[string]bool)
	for i, rule := range rules {
		// Validate rule name
		if rule.Name == "" {
			return &RuleValidationError{
				Index:  i,
				Reason: "empty name",
			}
		}

		if names[rule.Name] {
			return &RuleValidationError{
				RuleName: rule.Name,
				Reason:   "duplicate name",
			}
		}
		names[rule.Name] = true

		// Validate matcher function
		if rule.Matcher == nil {
			return &RuleValidationError{
				RuleName: rule.Name,
				Reason:   "nil matcher function",
			}
		}

		// Validate node type is not empty
		if rule.NodeType == "" {
			return &RuleValidationError{
				RuleName: rule.Name,
				Reason:   "empty node type",
			}
		}

		// Validate priority (should be positive)
		if rule.Priority < 0 {
			return &RuleValidationError{
				RuleName: rule.Name,
				Reason:   "negative priority",
			}
		}
	}

	return nil
}

// isValidPath checks if a file path is safe and doesn't contain malicious patterns.
// It prevents directory traversal attacks and other path-based exploits.
// All paths must be relative to the vault root (no leading slash).
// Only Unix-style paths are allowed (no backslashes).
func isValidPath(path string) bool {
	if path == "" {
		return false
	}

	// Reject Windows-style paths
	if strings.Contains(path, "\\") {
		return false
	}

	// Reject absolute paths - all paths must be relative
	if strings.HasPrefix(path, "/") {
		return false
	}

	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)

	// Prevent directory traversal - cleaned path should not contain ..
	// after normalization
	if strings.Contains(cleanPath, "..") {
		return false
	}

	// Prevent null bytes which can cause issues with file operations
	if strings.Contains(path, "\x00") {
		return false
	}

	// Flag suspiciously long paths that might indicate an attack
	if len(path) > 500 {
		return false
	}

	return true
}

// GetNodeTypeConfig returns the display configuration for a node type
// Returns nil if the classifier wasn't loaded from config or type doesn't exist
func (nc *NodeClassifier) GetNodeTypeConfig(nodeType string) *config.NodeTypeConfig {
	if nc.nodeTypes == nil {
		return nil
	}
	cfg, exists := nc.nodeTypes[nodeType]
	if !exists {
		return nil
	}
	return &cfg
}

// GetAvailableNodeTypes returns all configured node types
// Returns empty map if classifier wasn't loaded from config
func (nc *NodeClassifier) GetAvailableNodeTypes() map[string]config.NodeTypeConfig {
	if nc.nodeTypes == nil {
		return make(map[string]config.NodeTypeConfig)
	}
	// Return a copy to prevent modification
	result := make(map[string]config.NodeTypeConfig, len(nc.nodeTypes))
	for k, v := range nc.nodeTypes {
		result[k] = v
	}
	return result
}
