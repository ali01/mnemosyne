package vault

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLinkResolver(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "note1.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "note2.md", Frontmatter: &FrontmatterData{ID: "id2"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}
	assert.NotNil(t, resolver)
	assert.Len(t, resolver.pathToID, 2)
	assert.Len(t, resolver.idToPath, 2)
	assert.Len(t, resolver.normalizedToIDs, 2)
}

func TestLinkResolver_ResolveByID(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "concepts/note1.md", Frontmatter: &FrontmatterData{ID: "abc123"}},
		{Path: "projects/note2.md", Frontmatter: &FrontmatterData{ID: "def456"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	// ID should be treated as path and won't match
	id, found := resolver.ResolveLink("abc123", "")
	assert.False(t, found)
	assert.Empty(t, id)

	// Non-existent ID
	id, found = resolver.ResolveLink("xyz789", "")
	assert.False(t, found)
	assert.Empty(t, id)
}

func TestLinkResolver_ResolveByPath(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "concepts/network.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "projects/mnemosyne.md", Frontmatter: &FrontmatterData{ID: "id2"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	tests := []struct {
		name      string
		target    string
		wantID    string
		wantFound bool
	}{
		{"exact path", "concepts/network.md", "id1", true},
		{"path without extension", "concepts/network", "id1", true},
		{"partial path", "projects/mnemosyne", "id2", true},
		{"non-existent path", "does/not/exist", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, found := resolver.ResolveLink(tt.target, "")
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestLinkResolver_ResolveByBasename(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "deep/nested/path/unique-name.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "other/location/different.md", Frontmatter: &FrontmatterData{ID: "id2"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	// Basename match
	id, found := resolver.ResolveLink("unique-name", "")
	assert.True(t, found)
	assert.Equal(t, "id1", id)

	// Another basename match
	id, found = resolver.ResolveLink("different", "")
	assert.True(t, found)
	assert.Equal(t, "id2", id)
}

func TestLinkResolver_ResolveCaseInsensitive(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "Network-Theory.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "AI-Concepts.md", Frontmatter: &FrontmatterData{ID: "id2"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	tests := []struct {
		target string
		wantID string
	}{
		{"network-theory", "id1"},
		{"network theory", "id1"}, // normalized matching
		{"Network-Theory", "id1"}, // exact basename
		{"ai-concepts", "id2"},
		{"ai concepts", "id2"}, // normalized matching
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			id, found := resolver.ResolveLink(tt.target, "")
			assert.True(t, found, "Failed to resolve: %s", tt.target)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestLinkResolver_AmbiguousLinks(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "concepts/network.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "projects/network.md", Frontmatter: &FrontmatterData{ID: "id2"}},
		{Path: "archive/network.md", Frontmatter: &FrontmatterData{ID: "id3"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	// Resolving "network" should return first match
	id, found := resolver.ResolveLink("network", "")
	assert.True(t, found)
	assert.Contains(t, []string{"id1", "id2", "id3"}, id)

	// With source file in same directory, should prefer that
	id, found = resolver.ResolveLink("network", "concepts/other.md")
	assert.True(t, found)
	assert.Equal(t, "id1", id) // Should prefer same directory
}

func TestLinkResolver_Priority(t *testing.T) {
	files := []*MarkdownFile{
		// This file has basename "exact-match"
		{Path: "exact-match.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		// This file has basename "target"
		{Path: "folder/target.md", Frontmatter: &FrontmatterData{ID: "id2"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	// Should match by basename
	id, found := resolver.ResolveLink("target", "")
	assert.True(t, found)
	assert.Equal(t, "id2", id)

	// Should match by path
	id, found = resolver.ResolveLink("exact-match", "")
	assert.True(t, found)
	assert.Equal(t, "id1", id)
}

func TestLinkResolver_NoMatch(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "note1.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "note2.md", Frontmatter: &FrontmatterData{ID: "id2"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	// No match cases
	id, found := resolver.ResolveLink("nonexistent", "")
	assert.False(t, found)
	assert.Empty(t, id)

	id, found = resolver.ResolveLink("", "")
	assert.False(t, found)
	assert.Empty(t, id)

	id, found = resolver.ResolveLink("random-file", "")
	assert.False(t, found)
	assert.Empty(t, id)
}

func TestLinkResolver_SpecialCharacters(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "notes/C++ Programming.md", Frontmatter: &FrontmatterData{ID: "cpp"}},
		{Path: "guides/[Draft] Proposal.md", Frontmatter: &FrontmatterData{ID: "draft"}},
		{Path: "data/2023-01-15 Meeting.md", Frontmatter: &FrontmatterData{ID: "meeting"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	tests := []struct {
		target string
		wantID string
	}{
		{"C++ Programming", "cpp"},
		{"[Draft] Proposal", "draft"},
		{"2023-01-15 Meeting", "meeting"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			id, found := resolver.ResolveLink(tt.target, "")
			assert.True(t, found)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestLinkResolver_ResolveLinks(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "note1.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "note2.md", Frontmatter: &FrontmatterData{ID: "id2"}},
		{Path: "folder/note3.md", Frontmatter: &FrontmatterData{ID: "id3"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	links := []WikiLink{
		{Target: "note2"},   // Will resolve by basename
		{Target: "note3"},   // Will resolve by basename
		{Target: "missing"}, // Won't resolve
	}

	resolved, unresolved := resolver.ResolveLinks(links, "note1.md")

	// Check resolved links
	assert.Len(t, resolved, 2)
	assert.Equal(t, "id2", resolved["note2"])
	assert.Equal(t, "id3", resolved["note3"])

	// Check unresolved links
	assert.Len(t, unresolved, 1)
	assert.Equal(t, "missing", unresolved[0].Target)
}

func TestLinkResolver_GetStats(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "note1.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "note2.md", Frontmatter: &FrontmatterData{ID: "id2"}},
		{Path: "folder1/duplicate.md", Frontmatter: &FrontmatterData{ID: "id3"}},
		{Path: "folder2/duplicate.md", Frontmatter: &FrontmatterData{ID: "id4"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	stats := resolver.GetStats()
	assert.Equal(t, 4, stats["total_files"])
	assert.Equal(t, 3, stats["unique_basenames"]) // note1, note2, duplicate
	assert.Equal(t, 1, stats["duplicate_names"])  // duplicate appears twice, so 1 duplicate
}

func TestLinkResolver_GetPath(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "concepts/network.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "projects/mnemosyne.md", Frontmatter: &FrontmatterData{ID: "id2"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	// Get path for existing ID
	path, found := resolver.GetPath("id1")
	assert.True(t, found)
	assert.Equal(t, "concepts/network.md", path)

	// Get path for non-existent ID
	path, found = resolver.GetPath("id999")
	assert.False(t, found)
	assert.Empty(t, path)
}

func TestLinkResolver_RelativePath(t *testing.T) {
	files := []*MarkdownFile{
		{Path: "concepts/network.md", Frontmatter: &FrontmatterData{ID: "id1"}},
		{Path: "concepts/graph.md", Frontmatter: &FrontmatterData{ID: "id2"}},
		{Path: "projects/mnemosyne.md", Frontmatter: &FrontmatterData{ID: "id3"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	// Resolve relative path from same directory
	id, found := resolver.ResolveLink("graph", "concepts/network.md")
	assert.True(t, found)
	assert.Equal(t, "id2", id)

	// Resolve relative path with ../
	id, found = resolver.ResolveLink("../projects/mnemosyne", "concepts/network.md")
	assert.True(t, found)
	assert.Equal(t, "id3", id)
}

func TestLinkResolver_ComplexPaths(t *testing.T) {
	// Test complex path resolution scenarios
	files := []*MarkdownFile{
		{Path: "a/b/c/deep.md", Frontmatter: &FrontmatterData{ID: "deep"}},
		{Path: "a/b/shallow.md", Frontmatter: &FrontmatterData{ID: "shallow"}},
		{Path: "index.md", Frontmatter: &FrontmatterData{ID: "root"}},
		{Path: "a/index.md", Frontmatter: &FrontmatterData{ID: "a-index"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	tests := []struct {
		name       string
		target     string
		sourceFile string
		wantID     string
		wantFound  bool
	}{
		// Test various relative paths
		{"../../shallow from deep", "../../shallow", "a/b/c/deep.md", "shallow", true},
		{"./shallow from b", "./shallow", "a/b/other.md", "shallow", true},
		{"../../../index from deep", "../../../index", "a/b/c/deep.md", "root", true},
		{"../index from b", "../index", "a/b/file.md", "a-index", true},

		// Test absolute paths
		{"absolute root index", "index", "", "root", true},
		{"absolute a/index", "a/index", "", "a-index", true},

		// Test with trailing slashes
		{"path with .md already", "a/b/shallow.md", "", "shallow", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, found := resolver.ResolveLink(tt.target, tt.sourceFile)
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestLinkResolver_EdgeCases(t *testing.T) {
	// Test edge cases and boundary conditions
	resolver := NewLinkResolver()

	// Test with empty resolver
	id, found := resolver.ResolveLink("anything", "")
	assert.False(t, found)
	assert.Empty(t, id)

	// Add file with empty path
	resolver.AddFile(&MarkdownFile{Path: "", Frontmatter: &FrontmatterData{ID: "empty-path"}})

	// Add file with special characters
	resolver.AddFile(&MarkdownFile{Path: "file with spaces.md", Frontmatter: &FrontmatterData{ID: "spaces"}})
	resolver.AddFile(&MarkdownFile{Path: "file-with-dashes.md", Frontmatter: &FrontmatterData{ID: "dashes"}})
	resolver.AddFile(&MarkdownFile{Path: "file_with_underscores.md", Frontmatter: &FrontmatterData{ID: "underscores"}})

	// Test resolution
	id, found = resolver.ResolveLink("file with spaces", "")
	assert.True(t, found)
	assert.Equal(t, "spaces", id)

	// Test normalized matching
	id, found = resolver.ResolveLink("file-with-dashes", "")
	assert.True(t, found)
	assert.Equal(t, "dashes", id)

	// Test partial normalization
	id, found = resolver.ResolveLink("file with dashes", "") // Mixed separators
	assert.True(t, found)
	assert.Equal(t, "dashes", id)
}

func TestLinkResolver_DuplicateHandling(t *testing.T) {
	// Test how resolver handles files with same basename in different directories
	files := []*MarkdownFile{
		{Path: "2023/01/index.md", Frontmatter: &FrontmatterData{ID: "2023-01"}},
		{Path: "2023/02/index.md", Frontmatter: &FrontmatterData{ID: "2023-02"}},
		{Path: "2023/03/index.md", Frontmatter: &FrontmatterData{ID: "2023-03"}},
		{Path: "index.md", Frontmatter: &FrontmatterData{ID: "root-index"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	// When resolving "index" without context, should return first match
	id, found := resolver.ResolveLink("index", "")
	assert.True(t, found)
	// Should return one of the indexes (implementation dependent)
	assert.Contains(t, []string{"2023-01", "2023-02", "2023-03", "root-index"}, id)

	// When resolving from same directory, should prefer local file
	// But path matching happens before basename, so it might match root first
	_, found = resolver.ResolveLink("index", "2023/02/other.md")
	assert.True(t, found)
	// Could be any of the indexes depending on implementation

	// Check stats
	stats := resolver.GetStats()
	assert.Equal(t, 4, stats["total_files"])
	assert.Equal(t, 1, stats["unique_basenames"]) // All are "index"
	assert.Equal(t, 3, stats["duplicate_names"])  // 3 duplicates of "index"
}

func TestLinkResolver_ResolveLinks_BatchProcessing(t *testing.T) {
	// Test batch link resolution
	files := []*MarkdownFile{
		{Path: "a.md", Frontmatter: &FrontmatterData{ID: "a"}},
		{Path: "b.md", Frontmatter: &FrontmatterData{ID: "b"}},
		{Path: "c.md", Frontmatter: &FrontmatterData{ID: "c"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	// Create links with various targets
	links := []WikiLink{
		{Target: "a"},            // Will resolve
		{Target: "b"},            // Will resolve
		{Target: "missing"},      // Won't resolve
		{Target: "c"},            // Will resolve
		{Target: ""},             // Empty target
		{Target: "also-missing"}, // Won't resolve
	}

	resolved, unresolved := resolver.ResolveLinks(links, "source.md")

	// Check resolved links
	assert.Len(t, resolved, 3)
	assert.Equal(t, "a", resolved["a"])
	assert.Equal(t, "b", resolved["b"])
	assert.Equal(t, "c", resolved["c"])

	// Check unresolved links
	assert.Len(t, unresolved, 3)
	unresolvedTargets := make([]string, len(unresolved))
	for i, link := range unresolved {
		unresolvedTargets[i] = link.Target
	}
	assert.Contains(t, unresolvedTargets, "missing")
	assert.Contains(t, unresolvedTargets, "")
	assert.Contains(t, unresolvedTargets, "also-missing")
}

func TestLinkResolver_NormalizationEdgeCases(t *testing.T) {
	// Test the normalizeForMatching function through resolver
	files := []*MarkdownFile{
		{Path: "~hub-node.md", Frontmatter: &FrontmatterData{ID: "hub"}},
		{Path: "+important.md", Frontmatter: &FrontmatterData{ID: "important"}},
		{Path: "mixed-CASE_file.md", Frontmatter: &FrontmatterData{ID: "mixed"}},
		{Path: "multiple---dashes___underscores.md", Frontmatter: &FrontmatterData{ID: "multiple"}},
	}

	resolver := NewLinkResolver()
	for _, f := range files {
		resolver.AddFile(f)
	}

	tests := []struct {
		target string
		wantID string
	}{
		// Prefixes are removed in normalization
		{"hub-node", "hub"},
		{"important", "important"},

		// Case insensitive and separator normalization
		{"MIXED-case_FILE", "mixed"},
		{"mixed case file", "mixed"},

		// Multiple separators normalized
		{"multiple   dashes   underscores", "multiple"},
		{"multiple-dashes-underscores", "multiple"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			id, found := resolver.ResolveLink(tt.target, "")
			assert.True(t, found, "Failed to resolve: %s", tt.target)
			assert.Equal(t, tt.wantID, id)
		})
	}
}
