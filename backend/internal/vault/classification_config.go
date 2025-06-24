package vault

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/ali01/mnemosyne/internal/config"
)

// ConvertToClassificationRules converts config rules to ClassificationRule instances with validation
func ConvertToClassificationRules(nodeClassConfig *config.NodeClassificationConfig) ([]ClassificationRule, error) {
	// Validate the configuration first
	if err := validateClassificationConfig(nodeClassConfig); err != nil {
		return nil, fmt.Errorf("invalid classification config: %w", err)
	}
	rules := make([]ClassificationRule, 0, len(nodeClassConfig.ClassificationRules))

	for _, ruleConfig := range nodeClassConfig.ClassificationRules {
		// Validate node type exists
		if _, exists := nodeClassConfig.NodeTypes[ruleConfig.NodeType]; !exists {
			return nil, fmt.Errorf("rule '%s' references undefined node type '%s'", 
				ruleConfig.Name, ruleConfig.NodeType)
		}

		matcher, err := createMatcher(ruleConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create matcher for rule '%s': %w", 
				ruleConfig.Name, err)
		}

		rule := ClassificationRule{
			Name:     ruleConfig.Name,
			Priority: ruleConfig.Priority,
			Matcher:  matcher,
			NodeType: ruleConfig.NodeType,
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// createMatcher creates a matcher function based on the rule type
func createMatcher(ruleConfig config.ClassificationRuleConfig) (func(*MarkdownFile) bool, error) {
	switch ruleConfig.Type {
	case "tag":
		return func(file *MarkdownFile) bool {
			return hasTag(file, ruleConfig.Pattern)
		}, nil

	case "filename_prefix":
		return func(file *MarkdownFile) bool {
			filename := filepath.Base(file.Path)
			return strings.HasPrefix(strings.ToLower(filename), strings.ToLower(ruleConfig.Pattern))
		}, nil

	case "filename_suffix":
		return func(file *MarkdownFile) bool {
			filename := filepath.Base(file.Path)
			// Remove .md extension before checking suffix
			name := strings.TrimSuffix(filename, ".md")
			return strings.HasSuffix(strings.ToLower(name), strings.ToLower(ruleConfig.Pattern))
		}, nil

	case "filename_match":
		return func(file *MarkdownFile) bool {
			filename := filepath.Base(file.Path)
			return strings.EqualFold(filename, ruleConfig.Pattern)
		}, nil

	case "path_contains":
		return func(file *MarkdownFile) bool {
			return isInDirectory(file.Path, ruleConfig.Pattern)
		}, nil

	case "regex":
		// Always compile regex as case-insensitive
		re, err := regexp.Compile("(?i)" + ruleConfig.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern '%s': %w", ruleConfig.Pattern, err)
		}
		
		return func(file *MarkdownFile) bool {
			filename := filepath.Base(file.Path)
			return re.MatchString(filename)
		}, nil

	default:
		return nil, fmt.Errorf("unknown rule type: %s", ruleConfig.Type)
	}
}


// validateClassificationConfig validates the configuration
func validateClassificationConfig(config *config.NodeClassificationConfig) error {
	// Allow empty configuration - users might want no classification
	if len(config.NodeTypes) == 0 && len(config.ClassificationRules) == 0 {
		return nil
	}

	// Validate node types
	for name, nodeType := range config.NodeTypes {
		if name == "" {
			return fmt.Errorf("empty node type name")
		}
		if nodeType.DisplayName == "" {
			return fmt.Errorf("node type '%s' missing display name", name)
		}
		if nodeType.SizeMultiplier <= 0 {
			return fmt.Errorf("node type '%s' has invalid size multiplier: %f", 
				name, nodeType.SizeMultiplier)
		}
	}

	// Validate classification rules
	ruleNames := make(map[string]bool)
	for _, rule := range config.ClassificationRules {
		if rule.Name == "" {
			return fmt.Errorf("empty rule name")
		}
		if ruleNames[rule.Name] {
			return fmt.Errorf("duplicate rule name: %s", rule.Name)
		}
		ruleNames[rule.Name] = true

		if rule.Priority < 1 || rule.Priority > 100 {
			return fmt.Errorf("rule '%s' has invalid priority %d (must be 1-100)", 
				rule.Name, rule.Priority)
		}

		validTypes := []string{"tag", "filename_prefix", "filename_suffix", 
			"filename_match", "path_contains", "regex"}
		if !slices.Contains(validTypes, rule.Type) {
			return fmt.Errorf("rule '%s' has invalid type: %s", rule.Name, rule.Type)
		}

		if rule.Pattern == "" {
			return fmt.Errorf("rule '%s' has empty pattern", rule.Name)
		}

		// Validate regex patterns
		if rule.Type == "regex" {
			if _, err := regexp.Compile("(?i)" + rule.Pattern); err != nil {
				return fmt.Errorf("rule '%s' has invalid regex pattern: %w", rule.Name, err)
			}
		}
	}

	return nil
}