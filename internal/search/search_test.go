package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Parse tests ---

func TestParseEmpty(t *testing.T) {
	q, err := Parse("")
	require.NoError(t, err)
	assert.IsType(t, matchAll{}, q)
}

func TestParseStar(t *testing.T) {
	q, err := Parse("*")
	require.NoError(t, err)
	assert.IsType(t, matchAll{}, q)
}

func TestParsePathFilter(t *testing.T) {
	q, err := Parse("path:memex/concepts")
	require.NoError(t, err)
	assert.IsType(t, pathFilter{}, q)
}

func TestParseTagFilter(t *testing.T) {
	q, err := Parse("tag:#open-question")
	require.NoError(t, err)
	assert.IsType(t, tagFilter{}, q)
}

func TestParseTagFilterNoHash(t *testing.T) {
	q, err := Parse("tag:index")
	require.NoError(t, err)
	assert.IsType(t, tagFilter{}, q)
}

func TestParseFileFilter(t *testing.T) {
	q, err := Parse("file:readme")
	require.NoError(t, err)
	assert.IsType(t, fileFilter{}, q)
}

func TestParseFrontmatterFilter(t *testing.T) {
	q, err := Parse(`[author:"Ali Yahya"]`)
	require.NoError(t, err)
	assert.IsType(t, propFilter{}, q)
}

func TestParseFrontmatterUnquoted(t *testing.T) {
	q, err := Parse(`[status:draft]`)
	require.NoError(t, err)
	assert.IsType(t, propFilter{}, q)
}

func TestParseBareText(t *testing.T) {
	q, err := Parse("aviation")
	require.NoError(t, err)
	assert.IsType(t, textFilter{}, q)
}

func TestParseQuotedText(t *testing.T) {
	q, err := Parse(`"hello world"`)
	require.NoError(t, err)
	assert.IsType(t, textFilter{}, q)
}

func TestParseNOT(t *testing.T) {
	q, err := Parse("-tag:#archived")
	require.NoError(t, err)
	assert.IsType(t, notQuery{}, q)
}

func TestParseImplicitAND(t *testing.T) {
	q, err := Parse("path:memex tag:#index")
	require.NoError(t, err)
	assert.IsType(t, andQuery{}, q)
}

func TestParseOR(t *testing.T) {
	q, err := Parse("path:memex OR path:invest")
	require.NoError(t, err)
	assert.IsType(t, orQuery{}, q)
}

func TestParseParentheses(t *testing.T) {
	q, err := Parse("(path:a OR path:b) tag:#x")
	require.NoError(t, err)
	assert.IsType(t, andQuery{}, q)
}

func TestParseORDoesNotMatchWordPrefix(t *testing.T) {
	// "Oregon" should NOT be split into "OR" + "egon"
	q, err := Parse("Oregon")
	require.NoError(t, err)
	assert.IsType(t, textFilter{}, q)
}

func TestParseComplex(t *testing.T) {
	q, err := Parse(`(path:memex/ OR path:z-templates/) -tag:#archived`)
	require.NoError(t, err)
	assert.IsType(t, andQuery{}, q)
}

func TestParseError_UnterminatedQuote(t *testing.T) {
	_, err := Parse(`"unterminated`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unterminated")
}

func TestParseError_UnmatchedParen(t *testing.T) {
	_, err := Parse("(path:a")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "')'")
}

func TestParseError_UnmatchedBracket(t *testing.T) {
	_, err := Parse("[author:foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "']'")
}

// --- Match tests ---

var testNode = &NodeData{
	FilePath: "memex/concepts/aviation.md",
	Title:    "Aviation",
	Tags:     []string{"index", "open-question"},
	Frontmatter: map[string]interface{}{
		"author": "Ali Yahya",
		"status": "draft",
		"year":   2024,
	},
}

func TestMatchAll(t *testing.T) {
	q, _ := Parse("*")
	assert.True(t, q.Match(testNode))
}

func TestMatchPathContains(t *testing.T) {
	q, _ := Parse("path:memex/concepts")
	assert.True(t, q.Match(testNode))

	q, _ = Parse("path:invest")
	assert.False(t, q.Match(testNode))
}

func TestMatchPathCaseInsensitive(t *testing.T) {
	q, _ := Parse("path:MEMEX")
	assert.True(t, q.Match(testNode))
}

func TestMatchPathPartialDir(t *testing.T) {
	// "concept" is a substring of "concepts" — this is how Obsidian works
	q, _ := Parse("path:concept")
	assert.True(t, q.Match(testNode))
}

func TestMatchTagWithHash(t *testing.T) {
	q, _ := Parse("tag:#index")
	assert.True(t, q.Match(testNode))
}

func TestMatchTagWithoutHash(t *testing.T) {
	q, _ := Parse("tag:index")
	assert.True(t, q.Match(testNode))
}

func TestMatchTagCaseInsensitive(t *testing.T) {
	q, _ := Parse("tag:#Index")
	assert.True(t, q.Match(testNode))
}

func TestMatchTagNotFound(t *testing.T) {
	q, _ := Parse("tag:#nonexistent")
	assert.False(t, q.Match(testNode))
}

func TestMatchTagNilTags(t *testing.T) {
	q, _ := Parse("tag:#foo")
	n := &NodeData{FilePath: "test.md", Title: "Test"}
	assert.False(t, q.Match(n))
}

func TestMatchFile(t *testing.T) {
	q, _ := Parse("file:aviation")
	assert.True(t, q.Match(testNode))

	q, _ = Parse("file:economics")
	assert.False(t, q.Match(testNode))
}

func TestMatchFileCaseInsensitive(t *testing.T) {
	q, _ := Parse("file:AVIATION")
	assert.True(t, q.Match(testNode))
}

func TestMatchFrontmatter(t *testing.T) {
	q, _ := Parse(`[author:"Ali Yahya"]`)
	assert.True(t, q.Match(testNode))
}

func TestMatchFrontmatterCaseInsensitive(t *testing.T) {
	q, _ := Parse(`[author:"ali yahya"]`)
	assert.True(t, q.Match(testNode))
}

func TestMatchFrontmatterNotFound(t *testing.T) {
	q, _ := Parse(`[author:"Bob"]`)
	assert.False(t, q.Match(testNode))
}

func TestMatchFrontmatterMissingKey(t *testing.T) {
	q, _ := Parse(`[editor:"foo"]`)
	assert.False(t, q.Match(testNode))
}

func TestMatchFrontmatterNumericValue(t *testing.T) {
	q, _ := Parse(`[year:"2024"]`)
	assert.True(t, q.Match(testNode))
}

func TestMatchFrontmatterNilMap(t *testing.T) {
	q, _ := Parse(`[author:"foo"]`)
	n := &NodeData{FilePath: "test.md", Title: "Test"}
	assert.False(t, q.Match(n))
}

func TestMatchFrontmatterUnquoted(t *testing.T) {
	q, _ := Parse(`[status:draft]`)
	assert.True(t, q.Match(testNode))
}

func TestMatchBareText(t *testing.T) {
	q, _ := Parse("Aviation")
	assert.True(t, q.Match(testNode))

	q, _ = Parse("economics")
	assert.False(t, q.Match(testNode))
}

func TestMatchBareTextMatchesFilename(t *testing.T) {
	q, _ := Parse("aviation.md")
	assert.True(t, q.Match(testNode))
}

func TestMatchNOT(t *testing.T) {
	q, _ := Parse("-tag:#archived")
	assert.True(t, q.Match(testNode)) // node doesn't have #archived

	q, _ = Parse("-tag:#index")
	assert.False(t, q.Match(testNode)) // node has #index
}

func TestMatchImplicitAND(t *testing.T) {
	q, _ := Parse("path:memex tag:#index")
	assert.True(t, q.Match(testNode))

	q, _ = Parse("path:memex tag:#nonexistent")
	assert.False(t, q.Match(testNode))
}

func TestMatchOR(t *testing.T) {
	q, _ := Parse("path:invest OR path:memex")
	assert.True(t, q.Match(testNode))

	q, _ = Parse("path:invest OR path:research")
	assert.False(t, q.Match(testNode))
}

func TestMatchParentheses(t *testing.T) {
	q, _ := Parse("(path:memex OR path:invest) tag:#index")
	assert.True(t, q.Match(testNode))

	q, _ = Parse("(path:invest OR path:research) tag:#index")
	assert.False(t, q.Match(testNode))
}

func TestMatchNOTWithAND(t *testing.T) {
	q, _ := Parse("-tag:#archived path:memex")
	assert.True(t, q.Match(testNode))

	q, _ = Parse("-tag:#index path:memex")
	assert.False(t, q.Match(testNode))
}

func TestMatchComplex(t *testing.T) {
	q, _ := Parse(`(path:memex/ OR path:z-templates/) -tag:#archived [author:"Ali Yahya"]`)
	assert.True(t, q.Match(testNode))
}

func TestMatchEmpty(t *testing.T) {
	q, _ := Parse("")
	assert.True(t, q.Match(testNode))
}

// --- Edge cases ---

func TestParseWhitespaceOnly(t *testing.T) {
	q, err := Parse("   ")
	require.NoError(t, err)
	assert.IsType(t, matchAll{}, q)
}

func TestMatchQuotedBareText(t *testing.T) {
	q, _ := Parse(`"open-question"`)
	// bare text matches against title or filename — "open-question" is not in title or filename
	assert.False(t, q.Match(testNode))
}

func TestMatchMultipleORs(t *testing.T) {
	q, _ := Parse("path:a OR path:b OR path:memex")
	assert.True(t, q.Match(testNode))
}

func TestMatchNestedParens(t *testing.T) {
	q, err := Parse("((path:memex))")
	require.NoError(t, err)
	assert.True(t, q.Match(testNode))
}

func TestMatchNotGrouped(t *testing.T) {
	q, _ := Parse("-(path:invest OR path:research)")
	assert.True(t, q.Match(testNode))

	q, _ = Parse("-(path:memex OR path:research)")
	assert.False(t, q.Match(testNode))
}

func TestTagWithStoredHash(t *testing.T) {
	// Tags stored with # prefix
	n := &NodeData{
		FilePath: "test.md",
		Title:    "Test",
		Tags:     []string{"#important", "#review"},
	}
	q, _ := Parse("tag:#important")
	assert.True(t, q.Match(n))

	q, _ = Parse("tag:important")
	assert.True(t, q.Match(n))
}

func TestPrecedence_ANDTighterThanOR(t *testing.T) {
	// "a b OR c" should parse as "(a AND b) OR c"
	// This means: if node matches both "a" AND "b", OR if node matches "c"
	node := &NodeData{
		FilePath: "c.md",
		Title:    "c",
		Tags:     []string{},
	}
	q, err := Parse("a b OR c")
	require.NoError(t, err)
	// "c" matches title, so the OR clause matches
	assert.True(t, q.Match(node))

	nodeAB := &NodeData{
		FilePath: "b.md",
		Title:    "a b",
	}
	// "a" and "b" both match title, so the AND clause matches
	assert.True(t, q.Match(nodeAB))
}

func TestEscapedQuote(t *testing.T) {
	q, err := Parse(`[title:"say \"hello\""]`)
	require.NoError(t, err)
	n := &NodeData{
		FilePath:    "test.md",
		Title:       "Test",
		Frontmatter: map[string]interface{}{"title": `say "hello"`},
	}
	assert.True(t, q.Match(n))
}
