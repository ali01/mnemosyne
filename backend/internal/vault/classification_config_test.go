package vault

import (
	"testing"

	"github.com/ali01/mnemosyne/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateClassificationConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  config.NodeClassificationConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"test": {
						DisplayName:    "Test",
						SizeMultiplier: 1.0,
					},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
					{
						Name:     "test_rule",
						Priority: 50,
						Type:     "tag",
						Pattern:  "test",
						NodeType: "test",
					},
				},
			},
			wantErr: "",
		},
		{
			name: "empty config is valid",
			config: config.NodeClassificationConfig{
				NodeTypes:           map[string]config.NodeTypeConfig{},
				ClassificationRules: []config.ClassificationRuleConfig{},
			},
			wantErr: "",
		},
		{
			name: "empty node type name",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"": {DisplayName: "Empty"},
				},
			},
			wantErr: "empty node type name",
		},
		{
			name: "missing display name",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"test": {SizeMultiplier: 1.0},
				},
			},
			wantErr: "missing display name",
		},
		{
			name: "invalid size multiplier",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"test": {
						DisplayName:    "Test",
						SizeMultiplier: -1.0,
					},
				},
			},
			wantErr: "invalid size multiplier",
		},
		{
			name: "empty rule name",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"test": {DisplayName: "Test", SizeMultiplier: 1.0},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
					{
						Name:     "",
						Priority: 50,
						Type:     "tag",
						Pattern:  "test",
					},
				},
			},
			wantErr: "empty rule name",
		},
		{
			name: "duplicate rule name",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"test": {DisplayName: "Test", SizeMultiplier: 1.0},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
					{Name: "dup", Priority: 50, Type: "tag", Pattern: "test", NodeType: "test"},
					{Name: "dup", Priority: 60, Type: "tag", Pattern: "test2", NodeType: "test"},
				},
			},
			wantErr: "duplicate rule name",
		},
		{
			name: "invalid priority",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"test": {DisplayName: "Test", SizeMultiplier: 1.0},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
					{
						Name:     "test",
						Priority: 101,
						Type:     "tag",
						Pattern:  "test",
						NodeType: "test",
					},
				},
			},
			wantErr: "invalid priority",
		},
		{
			name: "invalid rule type",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"test": {DisplayName: "Test", SizeMultiplier: 1.0},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
					{
						Name:     "test",
						Priority: 50,
						Type:     "invalid",
						Pattern:  "test",
						NodeType: "test",
					},
				},
			},
			wantErr: "invalid type",
		},
		{
			name: "empty pattern",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"test": {DisplayName: "Test", SizeMultiplier: 1.0},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
					{
						Name:     "test",
						Priority: 50,
						Type:     "tag",
						Pattern:  "",
						NodeType: "test",
					},
				},
			},
			wantErr: "empty pattern",
		},
		{
			name: "invalid regex",
			config: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"test": {DisplayName: "Test", SizeMultiplier: 1.0},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
					{
						Name:     "test",
						Priority: 50,
						Type:     "regex",
						Pattern:  "[invalid",
						NodeType: "test",
					},
				},
			},
			wantErr: "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClassificationConfig(&tt.config)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConvertToClassificationRules(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		nodeClassConfig := &config.NodeClassificationConfig{
			NodeTypes: map[string]config.NodeTypeConfig{
				"test1": {DisplayName: "Test1", SizeMultiplier: 1.0},
				"test2": {DisplayName: "Test2", SizeMultiplier: 1.0},
			},
			ClassificationRules: []config.ClassificationRuleConfig{
				{
					Name:     "tag_rule",
					Priority: 10,
					Type:     "tag",
					Pattern:  "test-tag",
					NodeType: "test1",
				},
				{
					Name:     "prefix_rule",
					Priority: 20,
					Type:     "filename_prefix",
					Pattern:  "TEST_",
					NodeType: "test2",
				},
			},
		}

		rules, err := ConvertToClassificationRules(nodeClassConfig)
		assert.NoError(t, err)
		assert.Len(t, rules, 2)

	// Test tag rule
	tagRule := rules[0]
	assert.Equal(t, "tag_rule", tagRule.Name)
	assert.Equal(t, 10, tagRule.Priority)
	assert.Equal(t, "test1", tagRule.NodeType)

	// Test the matcher functions
	fileWithTag := &MarkdownFile{
		Path: "test.md",
		Frontmatter: &FrontmatterData{
			Raw: map[string]any{
				"tags": []string{"test-tag", "other"},
			},
		},
	}
	assert.True(t, tagRule.Matcher(fileWithTag))

	// Test case insensitive matching
	fileWithTagUpper := &MarkdownFile{
		Path: "test.md",
		Frontmatter: &FrontmatterData{
			Raw: map[string]any{
				"tags": []string{"TEST-TAG"},
			},
		},
	}
	assert.True(t, tagRule.Matcher(fileWithTagUpper))

	// Test prefix rule
	prefixRule := rules[1]
	assert.Equal(t, "prefix_rule", prefixRule.Name)
	
	fileWithPrefix := &MarkdownFile{
		Path: "TEST_file.md",
	}
	assert.True(t, prefixRule.Matcher(fileWithPrefix))

	// Test case insensitive matching
	fileWithPrefixLower := &MarkdownFile{
		Path: "test_file.md",
	}
	assert.True(t, prefixRule.Matcher(fileWithPrefixLower))
	})

	t.Run("invalid configuration - missing node type", func(t *testing.T) {
		nodeClassConfig := &config.NodeClassificationConfig{
			NodeTypes: map[string]config.NodeTypeConfig{
				"test1": {DisplayName: "Test1", SizeMultiplier: 1.0},
			},
			ClassificationRules: []config.ClassificationRuleConfig{
				{
					Name:     "invalid_rule",
					Priority: 10,
					Type:     "tag",
					Pattern:  "test",
					NodeType: "undefined_type", // This type doesn't exist
				},
			},
		}

		rules, err := ConvertToClassificationRules(nodeClassConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined node type")
		assert.Nil(t, rules)
	})

	t.Run("invalid configuration - empty rule name", func(t *testing.T) {
		nodeClassConfig := &config.NodeClassificationConfig{
			NodeTypes: map[string]config.NodeTypeConfig{
				"test": {DisplayName: "Test", SizeMultiplier: 1.0},
			},
			ClassificationRules: []config.ClassificationRuleConfig{
				{
					Name:     "", // Empty name
					Priority: 10,
					Type:     "tag",
					Pattern:  "test",
					NodeType: "test",
				},
			},
		}

		rules, err := ConvertToClassificationRules(nodeClassConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty rule name")
		assert.Nil(t, rules)
	})
}

func TestCreateMatcher(t *testing.T) {
	tests := []struct {
		name        string
		ruleConfig  config.ClassificationRuleConfig
		testFile    *MarkdownFile
		shouldMatch bool
	}{
		{
			name: "tag matcher case insensitive",
			ruleConfig: config.ClassificationRuleConfig{
				Type:          "tag",
				Pattern:       "important",
			},
			testFile: &MarkdownFile{
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"IMPORTANT", "note"},
					},
				},
			},
			shouldMatch: true,
		},
		{
			name: "filename_suffix matcher",
			ruleConfig: config.ClassificationRuleConfig{
				Type:          "filename_suffix",
				Pattern:       "_draft",
			},
			testFile: &MarkdownFile{
				Path: "my_document_draft.md",
			},
			shouldMatch: true,
		},
		{
			name: "filename_match exact",
			ruleConfig: config.ClassificationRuleConfig{
				Type:          "filename_match",
				Pattern:       "README.md",
			},
			testFile: &MarkdownFile{
				Path: "docs/readme.md",
			},
			shouldMatch: true,
		},
		{
			name: "path_contains matcher",
			ruleConfig: config.ClassificationRuleConfig{
				Type:          "path_contains",
				Pattern:       "archive",
			},
			testFile: &MarkdownFile{
				Path: "notes/ARCHIVE/old-note.md",
			},
			shouldMatch: true,
		},
		{
			name: "regex matcher",
			ruleConfig: config.ClassificationRuleConfig{
				Type:          "regex",
				Pattern:       "^\\d{4}-\\d{2}-\\d{2}_",
			},
			testFile: &MarkdownFile{
				Path: "journal/2023-12-25_christmas.md",
			},
			shouldMatch: true,
		},
		{
			name: "regex matcher case insensitive",
			ruleConfig: config.ClassificationRuleConfig{
				Type:          "regex",
				Pattern:       "^TODO",
			},
			testFile: &MarkdownFile{
				Path: "todo-list.md",
			},
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := createMatcher(tt.ruleConfig)
			assert.NoError(t, err)
			assert.NotNil(t, matcher)
			
			result := matcher(tt.testFile)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

func TestNewNodeClassifierFromConfig(t *testing.T) {
	// Create test config
	nodeClassConfig := &config.NodeClassificationConfig{
		NodeTypes: map[string]config.NodeTypeConfig{
			"diary": {
				DisplayName:    "Diary Entry",
				Description:    "Personal diary entries",
				Color:          "#FFB6C1",
				SizeMultiplier: 0.8,
			},
			"meeting": {
				DisplayName:    "Meeting Note",
				Description:    "Meeting notes and minutes",
				Color:          "#98FB98",
				SizeMultiplier: 1.1,
			},
		},
		ClassificationRules: []config.ClassificationRuleConfig{
			{
				Name:          "diary_date_prefix",
				Priority:      15,
				Type:          "regex",
				Pattern:       "^\\d{4}-\\d{2}-\\d{2}_diary",
				NodeType:      "diary",
			},
			{
				Name:          "meeting_tag",
				Priority:      5,
				Type:          "tag",
				Pattern:       "meeting",
				NodeType:      "meeting",
			},
		},
	}
	
	// Create classifier from config
	classifier, err := NewNodeClassifierFromConfig(nodeClassConfig)
	assert.NoError(t, err)
	assert.NotNil(t, classifier)
	
	// Check that rules are sorted by priority
	assert.Len(t, classifier.rules, 2)
	assert.Equal(t, 5, classifier.rules[0].Priority)
	assert.Equal(t, 15, classifier.rules[1].Priority)
	
	// Test classification
	diaryFile := &MarkdownFile{
		Path: "2023-12-25_diary.md",
		Frontmatter: &FrontmatterData{
			Raw: map[string]any{},
		},
	}
	assert.Equal(t, "diary", classifier.ClassifyNode(diaryFile))
	
	meetingFile := &MarkdownFile{
		Path: "team-sync.md",
		Frontmatter: &FrontmatterData{
			Raw: map[string]any{
				"tags": []string{"MEETING", "weekly"},
			},
		},
	}
	assert.Equal(t, "meeting", classifier.ClassifyNode(meetingFile))
	
	// Test node type config access
	diaryConfig := classifier.GetNodeTypeConfig("diary")
	assert.NotNil(t, diaryConfig)
	assert.Equal(t, "Diary Entry", diaryConfig.DisplayName)
	assert.Equal(t, "#FFB6C1", diaryConfig.Color)
	assert.Equal(t, 0.8, diaryConfig.SizeMultiplier)
	
	// Test getting all node types
	allTypes := classifier.GetAvailableNodeTypes()
	assert.Len(t, allTypes, 2)
	assert.Contains(t, allTypes, "diary")
	assert.Contains(t, allTypes, "meeting")
}

func TestNodeClassifierConfigIntegration(t *testing.T) {
	// Test that default classifier has no rules (requires configuration)
	defaultClassifier := NewNodeClassifier()
	assert.NotNil(t, defaultClassifier)
	assert.Empty(t, defaultClassifier.rules)
	
	// Default classifier should not have node type configs
	assert.Nil(t, defaultClassifier.GetNodeTypeConfig("index"))
	assert.Empty(t, defaultClassifier.GetAvailableNodeTypes())
	
	// All files should be classified as empty string (default) with no rules
	testFile := &MarkdownFile{
		Path: "test.md",
		Frontmatter: &FrontmatterData{
			Raw: map[string]any{"tags": []string{"index"}},
		},
	}
	assert.Equal(t, "", defaultClassifier.ClassifyNode(testFile))
}