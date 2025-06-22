package vault

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontmatterData represents parsed YAML frontmatter
type FrontmatterData struct {
	ID         string                 `yaml:"id"`
	Tags       []string               `yaml:"tags"`
	Related    []string               `yaml:"related"`
	References []string               `yaml:"references"`
	Raw        map[string]interface{} // Preserves all frontmatter fields
}

var (
	// Matches YAML frontmatter between --- markers
	frontmatterRegex = regexp.MustCompile(`(?s)^---\s*\n(.*?)---\s*\n`)
)

// ExtractFrontmatter parses YAML frontmatter and returns remaining content
func ExtractFrontmatter(content string) (*FrontmatterData, string, error) {
	matches := frontmatterRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, content, nil // No frontmatter
	}

	// Extract YAML content
	yamlContent := matches[1]

	// Remove frontmatter from content
	contentWithoutFrontmatter := strings.TrimPrefix(content, matches[0])

	// Parse YAML into raw map first
	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter YAML: %w", err)
	}

	// Also parse into structured data
	var data FrontmatterData
	if err := yaml.Unmarshal([]byte(yamlContent), &data); err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter structure: %w", err)
	}

	// Store raw data for preservation (ensure it's not nil)
	if raw == nil {
		raw = make(map[string]interface{})
	}
	data.Raw = raw

	// Validate required fields
	if data.ID == "" {
		return nil, "", fmt.Errorf("frontmatter missing required 'id' field")
	}

	// Ensure slices are non-nil
	if data.Tags == nil {
		data.Tags = []string{}
	}
	if data.Related == nil {
		data.Related = []string{}
	}
	if data.References == nil {
		data.References = []string{}
	}

	return &data, contentWithoutFrontmatter, nil
}

// HasTag checks if the frontmatter contains a specific tag
func (f *FrontmatterData) HasTag(tag string) bool {
	if f == nil {
		return false
	}
	for _, t := range f.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// GetString retrieves a string value from raw frontmatter
func (f *FrontmatterData) GetString(key string) (string, bool) {
	if f == nil || f.Raw == nil {
		return "", false
	}

	val, exists := f.Raw[key]
	if !exists {
		return "", false
	}

	str, ok := val.(string)
	return str, ok
}

// GetStringSlice retrieves a string slice from raw frontmatter
func (f *FrontmatterData) GetStringSlice(key string) ([]string, bool) {
	if f == nil || f.Raw == nil {
		return nil, false
	}

	val, exists := f.Raw[key]
	if !exists {
		return nil, false
	}

	// Handle different YAML representations of arrays
	switch v := val.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result, true
	case []string:
		return v, true
	default:
		return nil, false
	}
}
