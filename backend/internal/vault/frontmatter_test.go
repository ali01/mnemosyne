package vault

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantData    *FrontmatterData
		wantContent string
		wantErr     bool
	}{
		{
			name: "valid frontmatter with all fields",
			content: `---
id: "test123"
tags: ["index", "concept"]
related: ["other-note"]
references: ["ref1", "ref2"]
custom: "value"
---
# Test Content

This is the body.`,
			wantData: &FrontmatterData{
				ID:         "test123",
				Tags:       []string{"index", "concept"},
				Related:    []string{"other-note"},
				References: []string{"ref1", "ref2"},
				Raw: map[string]interface{}{
					"id":         "test123",
					"tags":       []interface{}{"index", "concept"},
					"related":    []interface{}{"other-note"},
					"references": []interface{}{"ref1", "ref2"},
					"custom":     "value",
				},
			},
			wantContent: "# Test Content\n\nThis is the body.",
			wantErr:     false,
		},
		{
			name: "minimal frontmatter with only ID",
			content: `---
id: "minimal123"
---
Content here`,
			wantData: &FrontmatterData{
				ID:         "minimal123",
				Tags:       []string{},
				Related:    []string{},
				References: []string{},
				Raw: map[string]interface{}{
					"id": "minimal123",
				},
			},
			wantContent: "Content here",
			wantErr:     false,
		},
		{
			name:        "no frontmatter",
			content:     "# Just Content\n\nNo frontmatter here.",
			wantData:    nil,
			wantContent: "# Just Content\n\nNo frontmatter here.",
			wantErr:     false,
		},
		{
			name: "missing required ID field",
			content: `---
tags: ["test"]
---
Content`,
			wantData:    nil,
			wantContent: "",
			wantErr:     true, // ID is mandatory for all vault files
		},
		{
			name: "invalid YAML",
			content: `---
id: "test"
invalid: [unclosed
---
Content`,
			wantData:    nil,
			wantContent: "",
			wantErr:     true,
		},
		{
			name: "empty frontmatter",
			content: `---
---
Content`,
			wantData:    nil,
			wantContent: "",
			wantErr:     true, // Missing ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, content, err := ExtractFrontmatter(tt.content)

			if tt.wantErr {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, content)

			if tt.wantData == nil {
				assert.Nil(t, data)
			} else {
				require.NotNil(t, data)
				assert.Equal(t, tt.wantData.ID, data.ID)
				assert.Equal(t, tt.wantData.Tags, data.Tags)
				assert.Equal(t, tt.wantData.Related, data.Related)
				assert.Equal(t, tt.wantData.References, data.References)
			}
		})
	}
}

func TestFrontmatterData_HasTag(t *testing.T) {
	data := &FrontmatterData{
		Tags: []string{"index", "concept", "ai"},
	}

	assert.True(t, data.HasTag("index"))
	assert.True(t, data.HasTag("concept"))
	assert.False(t, data.HasTag("missing"))
	assert.False(t, data.HasTag(""))
}

func TestFrontmatterData_GetString(t *testing.T) {
	data := &FrontmatterData{
		Raw: map[string]interface{}{
			"title":  "Test Title",
			"number": 123,
			"bool":   true,
		},
	}

	// Valid string field
	val, ok := data.GetString("title")
	assert.True(t, ok)
	assert.Equal(t, "Test Title", val)

	// Non-string field
	val, ok = data.GetString("number")
	assert.False(t, ok)
	assert.Equal(t, "", val)

	// Missing field
	val, ok = data.GetString("missing")
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestFrontmatterData_GetStringSlice(t *testing.T) {
	data := &FrontmatterData{
		Raw: map[string]interface{}{
			"tags1":  []interface{}{"tag1", "tag2", "tag3"},
			"tags2":  []string{"tag4", "tag5"},
			"mixed":  []interface{}{"string", 123, true},
			"single": "not-a-slice",
		},
	}

	// Interface slice
	val, ok := data.GetStringSlice("tags1")
	assert.True(t, ok)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, val)

	// String slice
	val, ok = data.GetStringSlice("tags2")
	assert.True(t, ok)
	assert.Equal(t, []string{"tag4", "tag5"}, val)

	// Mixed types - non-string values are silently ignored
	val, ok = data.GetStringSlice("mixed")
	assert.True(t, ok)
	assert.Equal(t, []string{"string"}, val)

	// Non-slice field
	val, ok = data.GetStringSlice("single")
	assert.False(t, ok)
	assert.Nil(t, val)

	// Missing field
	val, ok = data.GetStringSlice("missing")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestExtractFrontmatter_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantErr   bool
		checkData func(t *testing.T, data *FrontmatterData)
	}{
		{
			name: "frontmatter with unicode",
			content: `---
id: "unicode-test"
title: "café société"
tags: ["日本語", "中文", "emoji-🎉"]
---
Content with émojis 🌟`,
			wantErr: false,
			checkData: func(t *testing.T, data *FrontmatterData) {
				assert.Equal(t, "unicode-test", data.ID)
				assert.Equal(t, []string{"日本語", "中文", "emoji-🎉"}, data.Tags)
				title, ok := data.GetString("title")
				assert.True(t, ok)
				assert.Equal(t, "café société", title)
			},
		},
		{
			name: "very large frontmatter",
			content: func() string {
				content := "---\nid: \"large\"\n"
				for i := 0; i < 100; i++ {
					content += fmt.Sprintf("field%d: \"value%d\"\n", i, i)
				}
				content += "---\nContent"
				return content
			}(),
			wantErr: false,
			checkData: func(t *testing.T, data *FrontmatterData) {
				assert.Equal(t, "large", data.ID)
				// Check a few fields
				val, ok := data.GetString("field50")
				assert.True(t, ok)
				assert.Equal(t, "value50", val)
			},
		},
		{
			name: "ID as number",
			content: `---
id: 12345
tags: ["numeric"]
---
Content`,
			wantErr: false,
			checkData: func(t *testing.T, data *FrontmatterData) {
				assert.Equal(t, "12345", data.ID) // Should convert to string
			},
		},
		{
			name: "empty string ID",
			content: `---
id: ""
---
Content`,
			wantErr: true,
		},
		{
			name: "null ID",
			content: `---
id: null
---
Content`,
			wantErr: true,
		},
		{
			name: "ID with whitespace",
			content: `---
id: "  test  "
---
Content`,
			wantErr: false,
			checkData: func(t *testing.T, data *FrontmatterData) {
				assert.Equal(t, "  test  ", data.ID) // Preserves whitespace
			},
		},
		{
			name: "tags as single string",
			content: `---
id: "test"
tags: "single-tag"
---
Content`,
			wantErr: true, // Tags must be an array
		},
		{
			name: "malformed yaml with tabs",
			content: `---
id: "test"
	tags: ["bad-indent"]
---
Content`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, content, err := ExtractFrontmatter(tt.content)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, data)
			assert.NotEmpty(t, content)

			if tt.checkData != nil {
				tt.checkData(t, data)
			}
		})
	}
}

func TestFrontmatterData_GetString_TypeConversion(t *testing.T) {
	data := &FrontmatterData{
		Raw: map[string]interface{}{
			"string":     "value",
			"number":     123,
			"float":      45.67,
			"bool":       true,
			"nil":        nil,
			"empty":      "",
			"whitespace": "  \n\t  ",
		},
	}

	tests := []struct {
		key     string
		wantVal string
		wantOk  bool
	}{
		{"string", "value", true},
		{"number", "", false}, // GetString doesn't convert types
		{"float", "", false},  // GetString doesn't convert types
		{"bool", "", false},   // GetString doesn't convert types
		{"nil", "", false},
		{"empty", "", true},
		{"whitespace", "  \n\t  ", true},
		{"missing", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			val, ok := data.GetString(tt.key)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

func TestFrontmatterData_NilSafety(t *testing.T) {
	// Test nil FrontmatterData pointer
	var nilData *FrontmatterData
	assert.False(t, nilData.HasTag("test"))

	val, ok := nilData.GetString("test")
	assert.False(t, ok)
	assert.Empty(t, val)

	slice, ok := nilData.GetStringSlice("test")
	assert.False(t, ok)
	assert.Nil(t, slice)

	// Test with nil Raw map
	data := &FrontmatterData{}
	assert.False(t, data.HasTag("test"))

	val, ok = data.GetString("test")
	assert.False(t, ok)
	assert.Empty(t, val)

	slice, ok = data.GetStringSlice("test")
	assert.False(t, ok)
	assert.Nil(t, slice)

	// Test with nil Tags
	data = &FrontmatterData{
		Raw: map[string]interface{}{},
	}
	assert.False(t, data.HasTag("test"))
	assert.Empty(t, data.Tags)
}
