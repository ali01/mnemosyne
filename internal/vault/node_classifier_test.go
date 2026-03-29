package vault

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeClassifier_ClassifyNode(t *testing.T) {
	tests := []struct {
		name     string
		file     *MarkdownFile
		expected string
	}{
		// Tag-based classification tests
		{
			name: "index tag classification",
			file: &MarkdownFile{
				Path: "index.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"index", "main"},
					},
				},
			},
			expected: "index",
		},
		{
			name: "open-question tag classification",
			file: &MarkdownFile{
				Path: "some-question.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"open-question"},
					},
				},
			},
			expected: "question",
		},
		{
			name: "case-insensitive tag matching",
			file: &MarkdownFile{
				Path: "index2.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"INDEX"},
					},
				},
			},
			expected: "index",
		},
		{
			name: "tags as interface slice",
			file: &MarkdownFile{
				Path: "question.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []any{"Open-Question", "research"},
					},
				},
			},
			expected: "question",
		},
		{
			name: "single tag as string",
			file: &MarkdownFile{
				Path: "single-tag.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": "index",
					},
				},
			},
			expected: "index",
		},

		// Filename prefix tests
		{
			name: "hub filename prefix",
			file: &MarkdownFile{
				Path: "~hub-page.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "hub",
		},
		{
			name: "hub prefix in subdirectory",
			file: &MarkdownFile{
				Path: "subdir/~another-hub.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "hub",
		},

		// Directory-based classification tests
		{
			name: "concepts directory",
			file: &MarkdownFile{
				Path: "concepts/some-concept.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "concept",
		},
		{
			name: "projects directory",
			file: &MarkdownFile{
				Path: "projects/my-project.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "project",
		},
		{
			name: "questions directory",
			file: &MarkdownFile{
				Path: "questions/research-question.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "question",
		},
		{
			name: "nested concepts directory",
			file: &MarkdownFile{
				Path: "knowledge/concepts/nested-concept.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "concept",
		},
		{
			name: "case-insensitive directory matching",
			file: &MarkdownFile{
				Path: "CONCEPTS/uppercase-concept.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "concept",
		},

		// Priority tests
		{
			name: "tag overrides directory",
			file: &MarkdownFile{
				Path: "concepts/index-page.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"index"},
					},
				},
			},
			expected: "index", // Tag has higher priority
		},
		{
			name: "tag overrides filename prefix",
			file: &MarkdownFile{
				Path: "~hub-with-tag.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"open-question"},
					},
				},
			},
			expected: "question", // Tag has higher priority
		},
		{
			name: "filename prefix overrides directory",
			file: &MarkdownFile{
				Path: "concepts/~concept-hub.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "hub", // Prefix has higher priority than directory
		},

		// Default classification
		{
			name: "default to note type",
			file: &MarkdownFile{
				Path: "random-note.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "note",
		},
		{
			name: "no frontmatter defaults to note",
			file: &MarkdownFile{
				Path: "no-frontmatter.md",
			},
			expected: "note",
		},
		{
			name: "empty tags defaults to note",
			file: &MarkdownFile{
				Path: "empty-tags.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{},
					},
				},
			},
			expected: "note",
		},

		// Edge cases
		{
			name: "nil frontmatter fields",
			file: &MarkdownFile{
				Path: "nil-fields.md",
				Frontmatter: &FrontmatterData{
					Raw: nil,
				},
			},
			expected: "note",
		},
	}

	// Create a classifier with test rules that match the expected behavior
	rules := []ClassificationRule{
		// Tag-based rules
		{
			Name:     "index-tag",
			Priority: 1,
			Matcher: func(file *MarkdownFile) bool {
				return hasTag(file, "index")
			},
			NodeType: "index",
		},
		{
			Name:     "open-question-tag",
			Priority: 1,
			Matcher: func(file *MarkdownFile) bool {
				return hasTag(file, "open-question")
			},
			NodeType: "question",
		},
		// Filename prefix rules
		{
			Name:     "hub-prefix",
			Priority: 25,
			Matcher: func(file *MarkdownFile) bool {
				filename := filepath.Base(file.Path)
				return strings.HasPrefix(filename, "~")
			},
			NodeType: "hub",
		},
		// Directory-based rules
		{
			Name:     "concepts-directory",
			Priority: 45,
			Matcher: func(file *MarkdownFile) bool {
				return isInDirectory(file.Path, "concepts")
			},
			NodeType: "concept",
		},
		{
			Name:     "projects-directory",
			Priority: 45,
			Matcher: func(file *MarkdownFile) bool {
				return isInDirectory(file.Path, "projects")
			},
			NodeType: "project",
		},
		{
			Name:     "questions-directory",
			Priority: 45,
			Matcher: func(file *MarkdownFile) bool {
				return isInDirectory(file.Path, "questions")
			},
			NodeType: "question",
		},
	}

	classifier, err := NewNodeClassifierWithRules(rules, "note")
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyNode(tt.file)
			assert.Equal(t, tt.expected, result, "Classification mismatch for %s", tt.file.Path)
		})
	}
}

func TestNodeClassifier_CustomRules(t *testing.T) {
	// Create custom rules
	customRules := []ClassificationRule{
		{
			Name:     "custom-tag",
			Priority: PriorityTag,
			Matcher: func(file *MarkdownFile) bool {
				return hasTag(file, "custom")
			},
			NodeType: "project",
		},
		{
			Name:     "readme-files",
			Priority: PriorityFilename,
			Matcher: func(file *MarkdownFile) bool {
				filename := filepath.Base(file.Path)
				return strings.ToLower(filename) == "readme.md"
			},
			NodeType: "index",
		},
		{
			Name:     "archive-directory",
			Priority: PriorityPath,
			Matcher: func(file *MarkdownFile) bool {
				return isInDirectory(file.Path, "archive")
			},
			NodeType: "note",
		},
	}

	classifier, err := NewNodeClassifierWithRules(customRules, "note")
	assert.NoError(t, err)

	tests := []struct {
		name     string
		file     *MarkdownFile
		expected string
	}{
		{
			name: "custom tag rule",
			file: &MarkdownFile{
				Path: "custom-file.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"custom"},
					},
				},
			},
			expected: "project",
		},
		{
			name: "readme file rule",
			file: &MarkdownFile{
				Path: "docs/README.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "index",
		},
		{
			name: "archive directory rule",
			file: &MarkdownFile{
				Path: "archive/old-note.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "note",
		},
		{
			name: "no matching rule defaults to note",
			file: &MarkdownFile{
				Path: "unmatched.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyNode(tt.file)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNodeClassifier_RulePriority(t *testing.T) {
	// Create rules with different priorities
	rules := []ClassificationRule{
		{
			Name:     "low-priority",
			Priority: 10,
			Matcher: func(file *MarkdownFile) bool {
				return true // Always matches
			},
			NodeType: "note",
		},
		{
			Name:     "high-priority",
			Priority: 1,
			Matcher: func(file *MarkdownFile) bool {
				return true // Always matches
			},
			NodeType: "index",
		},
		{
			Name:     "medium-priority",
			Priority: 5,
			Matcher: func(file *MarkdownFile) bool {
				return true // Always matches
			},
			NodeType: "concept",
		},
	}

	classifier, err := NewNodeClassifierWithRules(rules, "note")
	assert.NoError(t, err)

	file := &MarkdownFile{
		Path: "test.md",
		Frontmatter: &FrontmatterData{
			Raw: map[string]any{},
		},
	}

	// Should return the node type from the highest priority rule (lowest number)
	result := classifier.ClassifyNode(file)
	assert.Equal(t, "index", result, "Should use highest priority rule")
}

func TestHasTag(t *testing.T) {
	tests := []struct {
		name     string
		file     *MarkdownFile
		tag      string
		expected bool
	}{
		{
			name: "has tag in string slice",
			file: &MarkdownFile{
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"tag1", "tag2", "tag3"},
					},
				},
			},
			tag:      "tag2",
			expected: true,
		},
		{
			name: "missing tag",
			file: &MarkdownFile{
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"tag1", "tag2"},
					},
				},
			},
			tag:      "tag3",
			expected: false,
		},
		{
			name: "case insensitive match",
			file: &MarkdownFile{
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"TAG1", "Tag2"},
					},
				},
			},
			tag:      "tag1",
			expected: true,
		},
		{
			name: "tags as interface slice",
			file: &MarkdownFile{
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []any{"tag1", "tag2", 123}, // Mixed types
					},
				},
			},
			tag:      "tag1",
			expected: true,
		},
		{
			name: "single tag as string",
			file: &MarkdownFile{
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": "single-tag",
					},
				},
			},
			tag:      "single-tag",
			expected: true,
		},
		{
			name: "no tags field",
			file: &MarkdownFile{
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"title": "No tags",
					},
				},
			},
			tag:      "any",
			expected: false,
		},
		{
			name: "nil frontmatter",
			file: &MarkdownFile{
				Frontmatter: nil,
			},
			tag:      "any",
			expected: false,
		},
		{
			name: "empty tags slice",
			file: &MarkdownFile{
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{},
					},
				},
			},
			tag:      "any",
			expected: false,
		},
		{
			name: "non-string value in interface slice",
			file: &MarkdownFile{
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []any{123, true, "valid-tag"},
					},
				},
			},
			tag:      "valid-tag",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasTag(tt.file, tt.tag)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInDirectory(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		dirName  string
		expected bool
	}{
		{
			name:     "file in directory",
			filePath: "concepts/concept.md",
			dirName:  "concepts",
			expected: true,
		},
		{
			name:     "file in nested directory",
			filePath: "knowledge/concepts/nested.md",
			dirName:  "concepts",
			expected: true,
		},
		{
			name:     "file not in directory",
			filePath: "notes/note.md",
			dirName:  "concepts",
			expected: false,
		},
		{
			name:     "case insensitive match",
			filePath: "CONCEPTS/file.md",
			dirName:  "concepts",
			expected: true,
		},
		{
			name:     "directory name as substring not match",
			filePath: "misconception/file.md",
			dirName:  "concepts",
			expected: false,
		},
		{
			name:     "exact directory name match",
			filePath: "concepts/file.md",
			dirName:  "concepts",
			expected: true,
		},
		{
			name:     "trailing slash",
			filePath: "concepts/",
			dirName:  "concepts",
			expected: true,
		},
		{
			name:     "empty directory name",
			filePath: "concepts/file.md",
			dirName:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInDirectory(tt.filePath, tt.dirName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewNodeClassifierWithRules_Validation(t *testing.T) {
	tests := []struct {
		name    string
		rules   []ClassificationRule
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rules",
			rules: []ClassificationRule{
				{
					Name:     "rule1",
					Priority: 1,
					Matcher:  func(*MarkdownFile) bool { return true },
					NodeType: "index",
				},
			},
			wantErr: false,
		},
		{
			name:    "empty rules are valid",
			rules:   []ClassificationRule{},
			wantErr: false,
		},
		{
			name: "rule with empty name",
			rules: []ClassificationRule{
				{
					Name:     "",
					Priority: 1,
					Matcher:  func(*MarkdownFile) bool { return true },
					NodeType: "index",
				},
			},
			wantErr: true,
			errMsg:  "empty name",
		},
		{
			name: "duplicate rule names",
			rules: []ClassificationRule{
				{
					Name:     "duplicate",
					Priority: 1,
					Matcher:  func(*MarkdownFile) bool { return true },
					NodeType: "index",
				},
				{
					Name:     "duplicate",
					Priority: 2,
					Matcher:  func(*MarkdownFile) bool { return true },
					NodeType: "hub",
				},
			},
			wantErr: true,
			errMsg:  "duplicate name",
		},
		{
			name: "nil matcher function",
			rules: []ClassificationRule{
				{
					Name:     "nil-matcher",
					Priority: 1,
					Matcher:  nil,
					NodeType: "index",
				},
			},
			wantErr: true,
			errMsg:  "nil matcher function",
		},
		{
			name: "empty node type",
			rules: []ClassificationRule{
				{
					Name:     "empty-type",
					Priority: 1,
					Matcher:  func(*MarkdownFile) bool { return true },
					NodeType: "",
				},
			},
			wantErr: true,
			errMsg:  "empty node type",
		},
		{
			name: "negative priority",
			rules: []ClassificationRule{
				{
					Name:     "negative-priority",
					Priority: -1,
					Matcher:  func(*MarkdownFile) bool { return true },
					NodeType: "index",
				},
			},
			wantErr: true,
			errMsg:  "negative priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier, err := NewNodeClassifierWithRules(tt.rules, "note")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, classifier)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, classifier)
			}
		})
	}
}

func TestNodeClassifier_EmptyRules(t *testing.T) {
	// Test classifier with no rules
	classifier, err := NewNodeClassifierWithRules([]ClassificationRule{}, "note")
	assert.NoError(t, err)

	file := &MarkdownFile{
		Path: "test.md",
		Frontmatter: &FrontmatterData{
			Raw: map[string]any{
				"tags": []string{"index"},
			},
		},
	}

	// Should default to note type
	result := classifier.ClassifyNode(file)
	assert.Equal(t, "note", result, "Should default to note type with no rules")
}

func TestNodeClassifier_ComplexPriorityScenarios(t *testing.T) {
	// Create rules for testing priority scenarios
	rules := []ClassificationRule{
		{
			Name:     "index-tag",
			Priority: 1,
			Matcher: func(file *MarkdownFile) bool {
				return hasTag(file, "index")
			},
			NodeType: "index",
		},
		{
			Name:     "open-question-tag",
			Priority: 1,
			Matcher: func(file *MarkdownFile) bool {
				return hasTag(file, "open-question")
			},
			NodeType: "question",
		},
		{
			Name:     "hub-prefix",
			Priority: 25,
			Matcher: func(file *MarkdownFile) bool {
				filename := filepath.Base(file.Path)
				return strings.HasPrefix(filename, "~")
			},
			NodeType: "hub",
		},
		{
			Name:     "concepts-directory",
			Priority: 45,
			Matcher: func(file *MarkdownFile) bool {
				return isInDirectory(file.Path, "concepts")
			},
			NodeType: "concept",
		},
	}

	classifier, err := NewNodeClassifierWithRules(rules, "note")
	assert.NoError(t, err)

	tests := []struct {
		name     string
		file     *MarkdownFile
		expected string
		reason   string
	}{
		{
			name: "multiple tags - first match wins",
			file: &MarkdownFile{
				Path: "multi-tag.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"note", "index", "concept"},
					},
				},
			},
			expected: "index",
			reason:   "index tag should be matched even with other tags present",
		},
		{
			name: "all classification criteria present",
			file: &MarkdownFile{
				Path: "concepts/~hub-index.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{
						"tags": []string{"open-question"},
					},
				},
			},
			expected: "question",
			reason:   "tag should override both prefix and directory",
		},
		{
			name: "similar directory names",
			file: &MarkdownFile{
				Path: "project-concepts/file.md",
				Frontmatter: &FrontmatterData{
					Raw: map[string]any{},
				},
			},
			expected: "note",
			reason:   "should not match partial directory names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyNode(tt.file)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

// Benchmark tests
func BenchmarkNodeClassifier_ClassifyNode(b *testing.B) {
	// Create a classifier with simple rules for benchmarking
	rules := []ClassificationRule{
		{
			Name:     "index-tag",
			Priority: 1,
			Matcher: func(file *MarkdownFile) bool {
				return hasTag(file, "index")
			},
			NodeType: "index",
		},
		{
			Name:     "hub-prefix",
			Priority: 25,
			Matcher: func(file *MarkdownFile) bool {
				filename := filepath.Base(file.Path)
				return strings.HasPrefix(filename, "~")
			},
			NodeType: "hub",
		},
		{
			Name:     "concepts-directory",
			Priority: 45,
			Matcher: func(file *MarkdownFile) bool {
				return isInDirectory(file.Path, "concepts")
			},
			NodeType: "concept",
		},
	}

	classifier, err := NewNodeClassifierWithRules(rules, "note")
	if err != nil {
		b.Fatal(err)
	}

	// Create various test files
	files := []*MarkdownFile{
		{
			Path: "index.md",
			Frontmatter: &FrontmatterData{
				Raw: map[string]any{
					"tags": []string{"index"},
				},
			},
		},
		{
			Path: "~hub.md",
			Frontmatter: &FrontmatterData{
				Raw: map[string]any{},
			},
		},
		{
			Path: "concepts/concept.md",
			Frontmatter: &FrontmatterData{
				Raw: map[string]any{},
			},
		},
		{
			Path: "note.md",
			Frontmatter: &FrontmatterData{
				Raw: map[string]any{},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		file := files[i%len(files)]
		_ = classifier.ClassifyNode(file)
	}
}

func BenchmarkNodeClassifier_WithManyRules(b *testing.B) {
	// Create many rules to test performance with large rule sets
	rules := make([]ClassificationRule, 100)
	for i := 0; i < 100; i++ {
		priority := i
		rules[i] = ClassificationRule{
			Name:     fmt.Sprintf("rule-%d", i),
			Priority: priority,
			Matcher: func(file *MarkdownFile) bool {
				return false // Never matches
			},
			NodeType: "note",
		}
	}

	// Add one matching rule at the end
	rules = append(rules, ClassificationRule{
		Name:     "matching-rule",
		Priority: 101,
		Matcher: func(file *MarkdownFile) bool {
			return true
		},
		NodeType: "index",
	})

	classifier, err := NewNodeClassifierWithRules(rules, "note")
	if err != nil {
		b.Fatal(err)
	}

	file := &MarkdownFile{
		Path: "test.md",
		Frontmatter: &FrontmatterData{
			Raw: map[string]any{},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = classifier.ClassifyNode(file)
	}
}
