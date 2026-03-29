package vault

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractWikiLinks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []WikiLink
	}{
		{
			name:    "basic wikilink",
			content: "This is a [[Test Link]] in content.",
			want: []WikiLink{
				{
					Raw:         "[[Test Link]]",
					Target:      "Test Link",
					DisplayText: "Test Link",
					Section:     "",
					LinkType:    "wikilink",
					Position:    10,
				},
			},
		},
		{
			name:    "wikilink with alias",
			content: "See [[Original Name|Display Name]] for details.",
			want: []WikiLink{
				{
					Raw:         "[[Original Name|Display Name]]",
					Target:      "Original Name",
					DisplayText: "Display Name",
					Section:     "",
					LinkType:    "wikilink",
					Position:    4,
				},
			},
		},
		{
			name:    "wikilink with section",
			content: "Check [[Document#Section]] for info.",
			want: []WikiLink{
				{
					Raw:         "[[Document#Section]]",
					Target:      "Document",
					DisplayText: "Document#Section", // Display includes section when no alias
					Section:     "Section",
					LinkType:    "wikilink",
					Position:    6,
				},
			},
		},
		{
			name:    "wikilink with section and alias",
			content: "Read [[Page#Heading|Custom Text]] carefully.",
			want: []WikiLink{
				{
					Raw:         "[[Page#Heading|Custom Text]]",
					Target:      "Page",
					DisplayText: "Custom Text",
					Section:     "Heading",
					LinkType:    "wikilink",
					Position:    5,
				},
			},
		},
		{
			name:    "embed link",
			content: "Here's an image: ![[photo.png]]",
			want: []WikiLink{
				{
					Raw:         "![[photo.png]]",
					Target:      "photo.png",
					DisplayText: "photo.png",
					Section:     "",
					LinkType:    "embed", // ! prefix indicates embedded content
					Position:    17,
				},
			},
		},
		{
			name:    "multiple links",
			content: "Link to [[Note A]] and [[Note B|B]] and ![[image.jpg]].",
			want: []WikiLink{
				{
					Raw:         "[[Note A]]",
					Target:      "Note A",
					DisplayText: "Note A",
					Section:     "",
					LinkType:    "wikilink",
					Position:    8,
				},
				{
					Raw:         "[[Note B|B]]",
					Target:      "Note B",
					DisplayText: "B",
					Section:     "",
					LinkType:    "wikilink",
					Position:    23,
				},
				{
					Raw:         "![[image.jpg]]",
					Target:      "image.jpg",
					DisplayText: "image.jpg",
					Section:     "",
					LinkType:    "embed",
					Position:    40,
				},
			},
		},
		{
			name:    "no links",
			content: "This content has no wiki links.",
			want:    []WikiLink{},
		},
		{
			name:    "complex path link",
			content: "See [[concepts/Network#Introduction|Networks]] for details.",
			want: []WikiLink{
				{
					Raw:         "[[concepts/Network#Introduction|Networks]]",
					Target:      "concepts/Network",
					DisplayText: "Networks",
					Section:     "Introduction",
					LinkType:    "wikilink",
					Position:    4,
				},
			},
		},
		{
			name:    "link with spaces",
			content: "Check [[ Spaced Link ]] here.",
			want: []WikiLink{
				{
					Raw:         "[[ Spaced Link ]]",
					Target:      "Spaced Link",
					DisplayText: "Spaced Link",
					Section:     "",
					LinkType:    "wikilink",
					Position:    6,
				},
			},
		},
		{
			name:    "nested brackets edge case",
			content: "This [[link [with] brackets]] is tricky.",
			want: []WikiLink{
				{
					Raw:         "[[link [with] brackets]]",
					Target:      "link [with] brackets",
					DisplayText: "link [with] brackets",
					Section:     "",
					LinkType:    "wikilink",
					Position:    5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractWikiLinks(tt.content)
			assert.Equal(t, len(tt.want), len(got), "number of links mismatch")

			for i := range tt.want {
				if i < len(got) {
					assert.Equal(t, tt.want[i].Raw, got[i].Raw)
					assert.Equal(t, tt.want[i].Target, got[i].Target)
					assert.Equal(t, tt.want[i].DisplayText, got[i].DisplayText)
					assert.Equal(t, tt.want[i].Section, got[i].Section)
					assert.Equal(t, tt.want[i].LinkType, got[i].LinkType)
					assert.Equal(t, tt.want[i].Position, got[i].Position)
				}
			}
		})
	}
}

func TestGetUniqueTargets(t *testing.T) {
	links := []WikiLink{
		{Target: "Note A"},
		{Target: "Note B"},
		{Target: "Note A"}, // Duplicate
		{Target: "Note C"},
		{Target: "Note B"}, // Duplicate
	}

	targets := GetUniqueTargets(links)
	assert.Len(t, targets, 3)

	// Check all unique targets are present
	targetMap := make(map[string]bool)
	for _, target := range targets {
		targetMap[target] = true
	}
	assert.True(t, targetMap["Note A"])
	assert.True(t, targetMap["Note B"])
	assert.True(t, targetMap["Note C"])
}

func TestFilterByType(t *testing.T) {
	links := []WikiLink{
		{Target: "Note 1", LinkType: "wikilink"},
		{Target: "image.png", LinkType: "embed"},
		{Target: "Note 2", LinkType: "wikilink"},
		{Target: "doc.pdf", LinkType: "embed"},
	}

	wikilinks := FilterByType(links, "wikilink")
	assert.Len(t, wikilinks, 2)
	assert.Equal(t, "Note 1", wikilinks[0].Target)
	assert.Equal(t, "Note 2", wikilinks[1].Target)

	embeds := FilterByType(links, "embed")
	assert.Len(t, embeds, 2)
	assert.Equal(t, "image.png", embeds[0].Target)
	assert.Equal(t, "doc.pdf", embeds[1].Target)
}

func TestNormalizeTarget(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple", "simple"},
		{"  Spaces  ", "spaces"},
		{"UPPERCASE", "uppercase"},
		{"Mixed Case", "mixed case"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeTarget(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseWikiLink_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		innerContent string
		isEmbed      bool
		position     int
		want         WikiLink
	}{
		{
			name:         "empty inner content",
			raw:          "[[]]",
			innerContent: "",
			isEmbed:      false,
			position:     0,
			want: WikiLink{
				Raw:         "[[]]",
				Target:      "",
				DisplayText: "",
				Section:     "",
				LinkType:    "wikilink",
				Position:    0,
			},
		},
		{
			name:         "only section",
			raw:          "[[#Section]]",
			innerContent: "#Section",
			isEmbed:      false,
			position:     10,
			want: WikiLink{
				Raw:         "[[#Section]]",
				Target:      "",
				DisplayText: "#Section",
				Section:     "Section",
				LinkType:    "wikilink",
				Position:    10,
			},
		},
		{
			name:         "only alias separator",
			raw:          "[[|]]",
			innerContent: "|",
			isEmbed:      false,
			position:     5,
			want: WikiLink{
				Raw:         "[[|]]",
				Target:      "",
				DisplayText: "",
				Section:     "",
				LinkType:    "wikilink",
				Position:    5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWikiLink(tt.raw, tt.innerContent, tt.isEmbed, tt.position)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractWikiLinks_PerformanceEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "unicode characters",
			content: "Link to [[café société]] and [[日本語ノート]].",
			want:    2,
		},
		{
			name:    "multiple pipe characters",
			content: "Link [[target|display|with|pipes]].",
			want:    1,
		},
		{
			name:    "consecutive links no space",
			content: "[[link1]][[link2]][[link3]]",
			want:    3,
		},
		{
			name:    "many links in large document",
			content: strings.Repeat("Text with [[link]] and more. ", 1000),
			want:    1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractWikiLinks(tt.content)
			assert.Len(t, got, tt.want)
		})
	}
}
