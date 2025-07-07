package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessMarkdownReader(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		path     string
		wantFile *MarkdownFile
		wantErr  bool
	}{
		{
			name: "complete markdown file",
			content: `---
id: "test123"
tags: ["index", "concept"]
---
# Test Document

This contains a [[Link]] to another note.`,
			path: "test/document.md",
			wantFile: &MarkdownFile{
				Path:  "test/document.md",
				Title: "document",
				Content: `---
id: "test123"
tags: ["index", "concept"]
---
# Test Document

This contains a [[Link]] to another note.`,
				Frontmatter: &FrontmatterData{
					ID:   "test123",
					Tags: []string{"index", "concept"},
				},
				Links: []WikiLink{
					{
						Raw:         "[[Link]]",
						Target:      "Link",
						DisplayText: "Link",
						LinkType:    "wikilink",
						Position:    65,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "file without frontmatter",
			content: `# Simple Note

Just content, no frontmatter.`,
			path: "simple.md",
			wantFile: &MarkdownFile{
				Path:        "simple.md",
				Title:       "simple",
				Content:     "# Simple Note\n\nJust content, no frontmatter.",
				Frontmatter: nil,
				Links:       []WikiLink{},
			},
			wantErr: false,
		},
		{
			name: "invalid frontmatter",
			content: `---
tags: ["test"]
---
Content`,
			path:     "invalid.md",
			wantFile: nil,
			wantErr:  true, // Missing required ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			file, err := ProcessMarkdownReader(reader, tt.path)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, file)

			assert.Equal(t, tt.wantFile.Path, file.Path)
			assert.Equal(t, tt.wantFile.Title, file.Title)
			assert.Equal(t, tt.wantFile.Content, file.Content)

			if tt.wantFile.Frontmatter != nil {
				require.NotNil(t, file.Frontmatter)
				assert.Equal(t, tt.wantFile.Frontmatter.ID, file.Frontmatter.ID)
				assert.Equal(t, tt.wantFile.Frontmatter.Tags, file.Frontmatter.Tags)
			} else {
				assert.Nil(t, file.Frontmatter)
			}

			assert.Equal(t, len(tt.wantFile.Links), len(file.Links))
			for i := range tt.wantFile.Links {
				if i < len(file.Links) {
					assert.Equal(t, tt.wantFile.Links[i].Target, file.Links[i].Target)
				}
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"simple.md", "simple"},
		{"my-document.md", "my-document"},
		{"my_document.md", "my_document"},
		{"~hub-node.md", "~hub-node"},
		{"path/to/document.md", "document"},
		{"UPPERCASE.md", "UPPERCASE"},
		{"multiple---dashes.md", "multiple---dashes"},
		{"", ""},
		{"no-extension", "no-extension"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := extractTitle(tt.path, nil)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestMarkdownFile_GetID(t *testing.T) {
	// With frontmatter
	file := &MarkdownFile{
		Frontmatter: &FrontmatterData{ID: "test123"},
	}
	assert.Equal(t, "test123", file.GetID())

	// Without frontmatter
	file = &MarkdownFile{Frontmatter: nil}
	assert.Equal(t, "", file.GetID())
}

func TestMarkdownFile_GetTags(t *testing.T) {
	// With tags
	file := &MarkdownFile{
		Frontmatter: &FrontmatterData{
			Tags: []string{"tag1", "tag2"},
		},
	}
	assert.Equal(t, []string{"tag1", "tag2"}, file.GetTags())

	// Without frontmatter
	file = &MarkdownFile{Frontmatter: nil}
	assert.Equal(t, []string{}, file.GetTags())

	// With empty tags
	file = &MarkdownFile{
		Frontmatter: &FrontmatterData{Tags: []string{}},
	}
	assert.Equal(t, []string{}, file.GetTags())
}

func TestMarkdownFile_GetNodeType(t *testing.T) {
	// Create a classifier with rules matching the old hardcoded logic
	rules := []ClassificationRule{
		{
			Name:     "index_tag",
			Priority: 1,
			Matcher:  func(f *MarkdownFile) bool { return f.Frontmatter != nil && f.Frontmatter.HasTag("index") },
			NodeType: "index",
		},
		{
			Name:     "open_question_tag",
			Priority: 1,
			Matcher:  func(f *MarkdownFile) bool { return f.Frontmatter != nil && f.Frontmatter.HasTag("open-question") },
			NodeType: "question",
		},
		{
			Name:     "hub_prefix",
			Priority: 2,
			Matcher:  func(f *MarkdownFile) bool { return strings.HasPrefix(filepath.Base(f.Path), "~") },
			NodeType: "hub",
		},
		{
			Name:     "concepts_dir",
			Priority: 3,
			Matcher:  func(f *MarkdownFile) bool { return isInDirectory(f.Path, "concepts") },
			NodeType: "concept",
		},
		{
			Name:     "projects_dir",
			Priority: 3,
			Matcher:  func(f *MarkdownFile) bool { return isInDirectory(f.Path, "projects") },
			NodeType: "project",
		},
	}

	classifier, err := NewNodeClassifierWithRules(rules, "note")
	require.NoError(t, err)

	tests := []struct {
		name     string
		file     *MarkdownFile
		expected string
	}{
		{
			name: "index tag",
			file: &MarkdownFile{
				Path: "test.md",
				Frontmatter: &FrontmatterData{
					Tags: []string{"index", "other"},
				},
			},
			expected: "index",
		},
		{
			name: "open-question tag",
			file: &MarkdownFile{
				Path: "question.md",
				Frontmatter: &FrontmatterData{
					Tags: []string{"open-question"},
				},
			},
			expected: "question",
		},
		{
			name: "hub prefix",
			file: &MarkdownFile{
				Path: "concepts/~hub.md",
			},
			expected: "hub",
		},
		{
			name: "concepts directory",
			file: &MarkdownFile{
				Path: "concepts/network.md",
			},
			expected: "concept",
		},
		{
			name: "projects directory",
			file: &MarkdownFile{
				Path: "projects/mnemosyne.md",
			},
			expected: "project",
		},
		{
			name: "default type - no classifier",
			file: &MarkdownFile{
				Path: "random/file.md",
			},
			expected: "note", // Changed from "default" to "note"
		},
		{
			name: "nil classifier returns note",
			file: &MarkdownFile{
				Path: "test.md",
				Frontmatter: &FrontmatterData{
					Tags: []string{"index"},
				},
			},
			expected: "note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil classifier returns note" {
				// Test with nil classifier
				got := tt.file.GetNodeType(nil)
				assert.Equal(t, tt.expected, got)
			} else {
				// Test with configured classifier
				got := tt.file.GetNodeType(classifier)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestMarkdownFile_Timestamps(t *testing.T) {
	// Create a mock FileInfo
	mockTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	file := &MarkdownFile{
		FileInfo: &mockFileInfo{modTime: mockTime},
	}

	assert.Equal(t, mockTime, file.GetCreatedAt())
	assert.Equal(t, mockTime, file.GetModifiedAt())

	// Without FileInfo
	file = &MarkdownFile{FileInfo: nil}
	now := time.Now()

	createdAt := file.GetCreatedAt()
	assert.WithinDuration(t, now, createdAt, time.Second)

	modifiedAt := file.GetModifiedAt()
	assert.WithinDuration(t, now, modifiedAt, time.Second)
}

func TestProcessMarkdownFile_Errors(t *testing.T) {
	// Test various error conditions
	tempDir := t.TempDir()

	// Test non-existent file
	_, err := ProcessMarkdownFile(tempDir, "non-existent.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")

	// Test directory instead of file
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0o750))
	_, err = ProcessMarkdownFile(tempDir, "subdir")
	assert.Error(t, err)
}

func TestExtractTitle_EdgeCases(t *testing.T) {
	// Test additional edge cases for title extraction
	tests := []struct {
		input string
		want  string
	}{
		// Special prefixes
		{"++priority.md", "++priority"},
		{"--archived.md", "--archived"},
		{"~!special.md", "~!special"},

		// Unicode and special characters
		{"café-société.md", "café-société"},
		{"2023-01-15_meeting.md", "2023-01-15_meeting"},
		{"file.multiple.dots.md", "file.multiple.dots"},

		// Very long filename
		{strings.Repeat("very-long-", 20) + "name.md", strings.Repeat("very-long-", 20) + "name"},

		// Path separators
		{"some/path/to/file.md", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractTitle(tt.input, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractTitle_WithFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		frontmatter *FrontmatterData
		expected    string
	}{
		{
			name: "frontmatter title overrides filename",
			path: "my-file.md",
			frontmatter: &FrontmatterData{
				Raw: map[string]any{"title": "Custom Title"},
			},
			expected: "Custom Title",
		},
		{
			name: "empty frontmatter title falls back to filename",
			path: "my-file.md",
			frontmatter: &FrontmatterData{
				Raw: map[string]any{"title": ""},
			},
			expected: "my-file",
		},
		{
			name: "no frontmatter uses filename",
			path: "my-file.md",
			frontmatter: nil,
			expected: "my-file",
		},
		{
			name: "frontmatter without title field uses filename",
			path: "another-file.md",
			frontmatter: &FrontmatterData{
				Raw: map[string]any{"other": "value"},
			},
			expected: "another-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.path, tt.frontmatter)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestMarkdownFile_ComplexContent(t *testing.T) {
	// Test processing complex markdown content
	content := `---
id: complex-note
tags: 
  - nested
  - yaml
  - "quoted tag"
aliases:
  - "Complex Note"
  - ComplexDoc
metadata:
  author: Test Author
  date: 2023-01-01
  custom:
    nested: value
---

# Complex Note

This note has various markdown features:

## Links
- [[simple-link]]
- [[path/to/note|Custom Display]]
- [[note#section]]
- ![[embedded-image.png]]
- [[note#section|Display with Section]]

## Code blocks
` + "```" + `python
def hello():
    print("Hello")
` + "```" + `

## Nested content
> Blockquote with [[link-in-quote]]
> > Nested quote

- List with [[link-in-list]]
  - Nested item with [[nested-link]]

## Tables
| Column 1 | Column 2 |
|----------|----------|
| [[link1]] | [[link2]] |

## Escaped links
\[[not-a-link]]
` + "`[[also-not-a-link]]`" + `

## Special characters in links
- [[C++ Programming]]
- [[[Draft] Proposal]]
- [[2023-01-15 Meeting Notes]]
`

	reader := strings.NewReader(content)
	file, err := ProcessMarkdownReader(reader, "complex.md")
	require.NoError(t, err)

	// Verify frontmatter
	assert.Equal(t, "complex-note", file.GetID())
	assert.Contains(t, file.GetTags(), "nested")
	assert.Contains(t, file.GetTags(), "yaml")
	assert.Contains(t, file.GetTags(), "quoted tag")

	// Verify custom metadata access
	aliases, ok := file.Frontmatter.GetStringSlice("aliases")
	assert.True(t, ok)
	assert.Contains(t, aliases, "Complex Note")
	assert.Contains(t, aliases, "ComplexDoc")

	// Verify links extracted correctly (escaped links should be included unfortunately)
	assert.Len(t, file.Links, 15) // WikiLink regex doesn't exclude escaped links

	// Check specific links
	linkTargets := make(map[string]bool)
	for _, link := range file.Links {
		linkTargets[link.Target] = true
	}

	assert.True(t, linkTargets["simple-link"])
	assert.True(t, linkTargets["path/to/note"])
	assert.True(t, linkTargets["link-in-quote"])
	assert.True(t, linkTargets["link-in-list"])
	assert.True(t, linkTargets["C++ Programming"])
	assert.True(t, linkTargets["2023-01-15 Meeting Notes"])

	// Unfortunately escaped links are extracted (regex limitation)
	assert.True(t, linkTargets["not-a-link"])
	assert.True(t, linkTargets["also-not-a-link"])
}

func TestMarkdownFile_BinaryContent(t *testing.T) {
	// Test handling of binary/invalid UTF-8 content
	tempDir := t.TempDir()
	binaryFile := filepath.Join(tempDir, "binary.md")

	// Write some binary data
	binaryData := []byte{0xFF, 0xFE, 0x00, 0x00, 0x41, 0x00}
	require.NoError(t, os.WriteFile(binaryFile, binaryData, 0o600))

	// Should handle binary data gracefully
	file, err := ProcessMarkdownFile(tempDir, "binary.md")
	assert.NoError(t, err) // Should not error
	assert.NotNil(t, file)
	assert.Equal(t, "binary", file.Title)
}

func TestMarkdownFile_LargeContent(t *testing.T) {
	// Test with very large content
	var builder strings.Builder
	builder.WriteString(`---
id: large-test
---
`)

	// Generate 10MB of content
	line := strings.Repeat("This is a test line. ", 50) + "\n"
	for i := 0; i < 10000; i++ {
		builder.WriteString(line)
		if i%100 == 0 {
			builder.WriteString(fmt.Sprintf("Link to [[note%d]] here.\n", i))
		}
	}

	reader := strings.NewReader(builder.String())
	file, err := ProcessMarkdownReader(reader, "large.md")

	require.NoError(t, err)
	assert.Equal(t, "large-test", file.GetID())
	assert.Len(t, file.Links, 100)
	assert.True(t, len(file.Content) > 10*1024*1024) // Content preserved
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	modTime time.Time
}

func (m *mockFileInfo) Name() string       { return "test.md" }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0o644 }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }
