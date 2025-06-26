package vault

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test MarkdownFile
func createTestMarkdownFile(path, id, title string, tags []string, links []WikiLink) *MarkdownFile {
	frontmatter := &FrontmatterData{
		ID:   id,
		Tags: tags,
		Raw: map[string]interface{}{
			"id":    id,
			"title": title,
			"tags":  tags,
		},
	}

	return &MarkdownFile{
		Path:        path,
		Title:       title,
		Content:     "# " + title + "\n\nTest content",
		Frontmatter: frontmatter,
		Links:       links,
		FileInfo:    &testFileInfo{modTime: time.Now()},
	}
}

// Helper function to create a test NodeClassifier
func createTestNodeClassifier() *NodeClassifier {
	rules := []ClassificationRule{
		{
			Name:     "index_tag",
			Priority: 1,
			Matcher:  func(f *MarkdownFile) bool { return hasTag(f, "index") },
			NodeType: "index",
		},
		{
			Name:     "hub_prefix",
			Priority: 2,
			Matcher:  func(f *MarkdownFile) bool { return f.Path[0] == '~' },
			NodeType: "hub",
		},
	}

	classifier, _ := NewNodeClassifierWithRules(rules, "note")
	return classifier
}

// Helper function to find a node by ID in a slice of nodes
func findNodeByID(nodes []models.VaultNode, id string) *models.VaultNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func TestNewGraphBuilder(t *testing.T) {
	classifier := createTestNodeClassifier()

	t.Run("with default config", func(t *testing.T) {
		config := GraphBuilderConfig{}
		gb := NewGraphBuilder(classifier, config)

		assert.NotNil(t, gb)
		assert.Equal(t, classifier, gb.classifier)
		assert.Equal(t, 1.0, gb.config.DefaultWeight)
		assert.False(t, gb.config.SkipOrphans)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := GraphBuilderConfig{
			DefaultWeight: 2.5,
			SkipOrphans:   true,
		}
		gb := NewGraphBuilder(classifier, config)

		assert.NotNil(t, gb)
		assert.Equal(t, 2.5, gb.config.DefaultWeight)
		assert.True(t, gb.config.SkipOrphans)
	})

	t.Run("nil classifier allowed", func(t *testing.T) {
		gb := NewGraphBuilder(nil, GraphBuilderConfig{})
		assert.NotNil(t, gb)
		assert.Nil(t, gb.classifier)
	})
}

func TestBuildGraph_Simple(t *testing.T) {
	classifier := createTestNodeClassifier()
	gb := NewGraphBuilder(classifier, GraphBuilderConfig{})

	// Create test files
	file1 := createTestMarkdownFile("index.md", "index", "Index", []string{"index"}, []WikiLink{
		{Target: "concepts", DisplayText: "Concepts", LinkType: "wikilink"},
		{Target: "projects", DisplayText: "Projects", LinkType: "wikilink"},
	})

	file2 := createTestMarkdownFile("concepts.md", "concepts", "Concepts", []string{}, []WikiLink{
		{Target: "index", DisplayText: "Back to Index", LinkType: "wikilink"},
	})

	file3 := createTestMarkdownFile("projects.md", "projects", "Projects", []string{}, []WikiLink{
		{Target: "index", DisplayText: "Home", LinkType: "wikilink"},
	})

	// Create resolver and add mappings
	resolver := NewLinkResolver()
	resolver.AddFile(file1)
	resolver.AddFile(file2)
	resolver.AddFile(file3)

	parseResult := &ParseResult{
		Files: map[string]*MarkdownFile{
			"index":    file1,
			"concepts": file2,
			"projects": file3,
		},
		Resolver:        resolver,
		UnresolvedLinks: []UnresolvedLink{},
	}

	// Build graph
	result, err := gb.BuildGraph(parseResult)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify nodes
	assert.Len(t, result.Nodes, 3)

	// Find index node and verify its properties
	var indexNode *models.VaultNode
	for i := range result.Nodes {
		if result.Nodes[i].ID == "index" {
			indexNode = &result.Nodes[i]
			break
		}
	}
	require.NotNil(t, indexNode)
	assert.Equal(t, "Index", indexNode.Title)
	assert.Equal(t, "index", indexNode.NodeType)
	assert.Equal(t, 2, indexNode.OutDegree)
	assert.Equal(t, 2, indexNode.InDegree)

	// Verify edges
	assert.Len(t, result.Edges, 4) // 2 from index + 2 to index

	// Verify stats
	assert.Equal(t, 3, result.Stats.NodesCreated)
	assert.Equal(t, 4, result.Stats.EdgesCreated)
	assert.Equal(t, 0, result.Stats.FilesSkipped)
	assert.Len(t, result.DuplicateIDs, 0)
	assert.Equal(t, 0, result.Stats.UnresolvedLinks)
	assert.Equal(t, 0, result.Stats.OrphanedNodes)
}

func TestBuildGraph_WithDuplicateIDs(t *testing.T) {
	// Test that duplicate detection works properly
	// This simulates a scenario where the parser might have multiple files with same ID
	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	// Override buildNodes to test duplicate detection logic
	files := map[string]*MarkdownFile{
		"note1": createTestMarkdownFile("folder1/note.md", "duplicate", "Note 1", nil, nil),
		"note2": createTestMarkdownFile("folder2/note.md", "duplicate", "Note 2", nil, nil),
		"note3": createTestMarkdownFile("unique.md", "unique", "Unique", nil, nil),
	}

	// Create a custom test that processes files in order
	stats := &GraphStats{}
	nodeMap, _, duplicatesMap, err := gb.buildNodes(files, stats)
	require.NoError(t, err)

	// Should have detected the duplicate
	assert.Contains(t, duplicatesMap, "duplicate")
	assert.Len(t, nodeMap, 2) // Only 2 unique IDs should be in the map
	assert.Equal(t, 2, stats.NodesCreated)

	// Verify duplicate tracking details
	dup := duplicatesMap["duplicate"]
	require.NotNil(t, dup)
	assert.Equal(t, "duplicate", dup.ID)
	assert.NotEmpty(t, dup.KeptPath)
	assert.Len(t, dup.SkippedPaths, 1)

	// Verify that only one of the duplicate files was processed
	duplicateNode, hasDuplicate := nodeMap["duplicate"]
	uniqueNode, hasUnique := nodeMap["unique"]
	assert.True(t, hasDuplicate)
	assert.True(t, hasUnique)
	assert.NotNil(t, duplicateNode)
	assert.NotNil(t, uniqueNode)
}

func TestBuildGraph_WithMissingID(t *testing.T) {
	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	fileWithID := createTestMarkdownFile("with-id.md", "valid", "Valid", nil, nil)
	fileWithoutID := &MarkdownFile{
		Path:        "without-id.md",
		Title:       "No ID",
		Content:     "Content",
		Frontmatter: &FrontmatterData{}, // No ID
	}

	parseResult := &ParseResult{
		Files: map[string]*MarkdownFile{
			"valid": fileWithID,
			"":      fileWithoutID, // Empty ID
		},
		Resolver: NewLinkResolver(),
	}

	result, err := gb.BuildGraph(parseResult)
	require.NoError(t, err)

	assert.Len(t, result.Nodes, 1) // Only file with valid ID
	assert.Equal(t, 1, result.Stats.NodesCreated)
	assert.Equal(t, 1, result.Stats.FilesSkipped)
}

func TestBuildGraph_WithUnresolvedLinks(t *testing.T) {
	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	file1 := createTestMarkdownFile("note.md", "note", "Note", nil, []WikiLink{
		{Target: "existing", LinkType: "wikilink"},
		{Target: "missing", LinkType: "wikilink"},
	})

	file2 := createTestMarkdownFile("existing.md", "existing", "Existing", nil, nil)

	resolver := NewLinkResolver()
	resolver.AddFile(file1)
	resolver.AddFile(file2)

	parseResult := &ParseResult{
		Files: map[string]*MarkdownFile{
			"note":     file1,
			"existing": file2,
		},
		Resolver: resolver,
		UnresolvedLinks: []UnresolvedLink{
			{
				SourceID:   "note",
				SourcePath: "note.md",
				Link:       WikiLink{Target: "missing", LinkType: "wikilink"},
			},
		},
	}

	result, err := gb.BuildGraph(parseResult)
	require.NoError(t, err)

	assert.Len(t, result.Nodes, 2)
	assert.Len(t, result.Edges, 1) // Only the resolved link
	// Stats counts graph-excluded links, not parser unresolved
	assert.Equal(t, 0, result.Stats.UnresolvedLinks)
	assert.Len(t, result.UnresolvedLinks, 1) // Parser unresolved links are preserved in Graph
}

func TestBuildGraph_WithGraphExcludedLinks(t *testing.T) {
	// Test links to files that exist but aren't in the graph (no ID)
	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	fileWithID := createTestMarkdownFile("with-id.md", "source", "Source", nil, []WikiLink{
		{Target: "no-id", LinkType: "wikilink"},
	})
	fileWithoutID := &MarkdownFile{
		Path:        "no-id.md",
		Title:       "No ID",
		Content:     "Content",
		Frontmatter: &FrontmatterData{}, // No ID
	}

	resolver := NewLinkResolver()
	resolver.AddFile(fileWithID)
	resolver.AddFile(fileWithoutID)

	parseResult := &ParseResult{
		Files: map[string]*MarkdownFile{
			"source": fileWithID,
			"":       fileWithoutID, // Empty key since no ID
		},
		Resolver:        resolver,
		UnresolvedLinks: []UnresolvedLink{}, // No parser-level unresolved
	}

	result, err := gb.BuildGraph(parseResult)
	require.NoError(t, err)

	assert.Len(t, result.Nodes, 1)                   // Only the file with ID
	assert.Len(t, result.Edges, 0)                   // Link couldn't be created
	assert.Equal(t, 1, result.Stats.UnresolvedLinks) // Graph-excluded link counted
	assert.Len(t, result.UnresolvedLinks, 0)         // No parser-level unresolved
}

func TestBuildGraph_WithOrphanedNodes(t *testing.T) {
	t.Run("include orphans", func(t *testing.T) {
		gb := NewGraphBuilder(nil, GraphBuilderConfig{SkipOrphans: false})

		file1 := createTestMarkdownFile("connected.md", "connected", "Connected", nil, []WikiLink{
			{Target: "target", LinkType: "wikilink"},
		})
		file2 := createTestMarkdownFile("target.md", "target", "Target", nil, nil)
		file3 := createTestMarkdownFile("orphan.md", "orphan", "Orphan", nil, nil)

		resolver := NewLinkResolver()
		resolver.AddFile(file1)
		resolver.AddFile(file2)
		resolver.AddFile(file3)

		parseResult := &ParseResult{
			Files: map[string]*MarkdownFile{
				"connected": file1,
				"target":    file2,
				"orphan":    file3,
			},
			Resolver: resolver,
		}

		result, err := gb.BuildGraph(parseResult)
		require.NoError(t, err)

		assert.Len(t, result.Nodes, 3) // All nodes included
		assert.Equal(t, 1, result.Stats.OrphanedNodes)
	})

	t.Run("skip orphans", func(t *testing.T) {
		gb := NewGraphBuilder(nil, GraphBuilderConfig{SkipOrphans: true})

		file1 := createTestMarkdownFile("connected.md", "connected", "Connected", nil, []WikiLink{
			{Target: "target", LinkType: "wikilink"},
		})
		file2 := createTestMarkdownFile("target.md", "target", "Target", nil, nil)
		file3 := createTestMarkdownFile("orphan.md", "orphan", "Orphan", nil, nil)

		resolver := NewLinkResolver()
		resolver.AddFile(file1)
		resolver.AddFile(file2)
		resolver.AddFile(file3)

		parseResult := &ParseResult{
			Files: map[string]*MarkdownFile{
				"connected": file1,
				"target":    file2,
				"orphan":    file3,
			},
			Resolver: resolver,
		}

		result, err := gb.BuildGraph(parseResult)
		require.NoError(t, err)

		assert.Len(t, result.Nodes, 2) // Orphan excluded
		assert.Equal(t, 1, result.Stats.OrphanedNodes)
	})
}

func TestBuildGraph_WithEmbedLinks(t *testing.T) {
	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	file1 := createTestMarkdownFile("note.md", "note", "Note", nil, []WikiLink{
		{Target: "image", LinkType: "embed", DisplayText: "Image"},
		{Target: "other", LinkType: "wikilink"},
	})

	file2 := createTestMarkdownFile("image.md", "image", "Image", nil, nil)
	file3 := createTestMarkdownFile("other.md", "other", "Other", nil, nil)

	resolver := NewLinkResolver()
	resolver.AddFile(file1)
	resolver.AddFile(file2)
	resolver.AddFile(file3)

	parseResult := &ParseResult{
		Files: map[string]*MarkdownFile{
			"note":  file1,
			"image": file2,
			"other": file3,
		},
		Resolver: resolver,
	}

	result, err := gb.BuildGraph(parseResult)
	require.NoError(t, err)

	// Find embed edge
	var embedEdge *models.VaultEdge
	for i := range result.Edges {
		if result.Edges[i].EdgeType == "embed" {
			embedEdge = &result.Edges[i]
			break
		}
	}

	require.NotNil(t, embedEdge)
	assert.Equal(t, "note", embedEdge.SourceID)
	assert.Equal(t, "image", embedEdge.TargetID)
	assert.Equal(t, "Image", embedEdge.DisplayText)
}

func TestBuildGraph_WithSelfReferentialLinks(t *testing.T) {
	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	file := createTestMarkdownFile("self.md", "self", "Self", nil, []WikiLink{
		{Target: "self", LinkType: "wikilink", DisplayText: "Self Reference"},
	})

	resolver := NewLinkResolver()
	resolver.AddFile(file)

	parseResult := &ParseResult{
		Files: map[string]*MarkdownFile{
			"self": file,
		},
		Resolver: resolver,
	}

	result, err := gb.BuildGraph(parseResult)
	require.NoError(t, err)

	assert.Len(t, result.Nodes, 1)
	assert.Len(t, result.Edges, 1)

	// Verify self-referential edge
	edge := result.Edges[0]
	assert.Equal(t, "self", edge.SourceID)
	assert.Equal(t, "self", edge.TargetID)

	// Verify degrees
	node := result.Nodes[0]
	assert.Equal(t, 1, node.OutDegree)
	assert.Equal(t, 1, node.InDegree)
}

func TestBuildGraph_EdgeDeduplicationWithinSingleFile(t *testing.T) {
	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	// Test case: Single file with duplicate links to same target
	links := []WikiLink{
		{Target: "target", LinkType: "wikilink"},
		{Target: "target", LinkType: "wikilink"},                                // Duplicate
		{Target: "target", LinkType: "wikilink", DisplayText: "Different text"}, // Still duplicate
	}

	file1 := createTestMarkdownFile("source.md", "source", "Source", nil, links)
	file2 := createTestMarkdownFile("target.md", "target", "Target", nil, nil)

	resolver := NewLinkResolver()
	resolver.AddFile(file1)
	resolver.AddFile(file2)

	parseResult := &ParseResult{
		Files: map[string]*MarkdownFile{
			"source": file1,
			"target": file2,
		},
		Resolver: resolver,
	}

	result, err := gb.BuildGraph(parseResult)
	require.NoError(t, err)

	// Should create only one edge despite multiple duplicate links
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, 1, result.Stats.EdgesCreated)

	// Verify the edge properties
	edge := result.Edges[0]
	assert.Equal(t, "source", edge.SourceID)
	assert.Equal(t, "target", edge.TargetID)
	assert.Equal(t, "wikilink", edge.EdgeType)

	// Verify node degrees
	sourceNode := findNodeByID(result.Nodes, "source")
	targetNode := findNodeByID(result.Nodes, "target")
	require.NotNil(t, sourceNode)
	require.NotNil(t, targetNode)

	assert.Equal(t, 1, sourceNode.OutDegree) // Only 1 edge despite multiple links
	assert.Equal(t, 1, targetNode.InDegree)
}

func TestBuildGraph_NilInput(t *testing.T) {
	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	result, err := gb.BuildGraph(nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parseResult cannot be nil")
}

func TestCreateNode_WithMetadata(t *testing.T) {
	gb := NewGraphBuilder(createTestNodeClassifier(), GraphBuilderConfig{})

	frontmatter := &FrontmatterData{
		ID:   "test",
		Tags: []string{"tag1", "tag2"},
		Raw: map[string]interface{}{
			"id":           "test",
			"title":        "Custom Title",
			"tags":         []string{"tag1", "tag2"},
			"author":       "Test Author",
			"date":         "2023-01-01",
			"custom_field": 42,
		},
	}

	file := &MarkdownFile{
		Path:        "test.md",
		Title:       "Filename Title",
		Content:     "Test content",
		Frontmatter: frontmatter,
		FileInfo:    &testFileInfo{modTime: time.Now()},
	}

	node, err := gb.createNode(file, "test")
	require.NoError(t, err)

	assert.Equal(t, "test", node.ID)
	assert.Equal(t, "Custom Title", node.Title) // Prefers frontmatter title
	assert.Equal(t, []string{"tag1", "tag2"}, node.Tags)
	assert.Equal(t, "Test content", node.Content)
	assert.Equal(t, "test.md", node.FilePath)
	assert.Equal(t, "note", node.NodeType) // Default from classifier

	// Verify metadata
	assert.NotNil(t, node.Metadata)
	assert.Equal(t, "Test Author", node.Metadata["author"])
	assert.Equal(t, "2023-01-01", node.Metadata["date"])
	assert.Equal(t, 42, node.Metadata["custom_field"])
}

func TestCreateEdge_WithVariations(t *testing.T) {
	gb := NewGraphBuilder(nil, GraphBuilderConfig{DefaultWeight: 1.5})
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("basic edge", func(t *testing.T) {
		link := WikiLink{
			Target:      "target",
			DisplayText: "Custom Display",
			LinkType:    "wikilink",
		}

		edge, err := gb.createEdge("source", "target", link, testTime)
		require.NoError(t, err)

		assert.NotEmpty(t, edge.ID) // UUID generated
		assert.Equal(t, "source", edge.SourceID)
		assert.Equal(t, "target", edge.TargetID)
		assert.Equal(t, "wikilink", edge.EdgeType)
		assert.Equal(t, "Custom Display", edge.DisplayText)
		assert.Equal(t, 1.5, edge.Weight)
		assert.Equal(t, testTime, edge.CreatedAt)
	})

	t.Run("edge with section", func(t *testing.T) {
		link := WikiLink{
			Target:   "target",
			Section:  "Section Name",
			LinkType: "wikilink",
		}

		edge, err := gb.createEdge("source", "target", link, testTime)
		require.NoError(t, err)

		assert.Equal(t, "Section Name", edge.DisplayText)
	})

	t.Run("edge with both display text and section", func(t *testing.T) {
		link := WikiLink{
			Target:      "target",
			DisplayText: "Custom",
			Section:     "Section",
			LinkType:    "wikilink",
		}

		edge, err := gb.createEdge("source", "target", link, testTime)
		require.NoError(t, err)

		assert.Equal(t, "Custom", edge.DisplayText) // DisplayText takes precedence
	})

	t.Run("empty sourceID", func(t *testing.T) {
		link := WikiLink{
			Target:   "target",
			LinkType: "wikilink",
		}

		edge, err := gb.createEdge("", "target", link, testTime)
		require.Error(t, err)
		assert.Nil(t, edge)
		assert.Contains(t, err.Error(), "sourceID cannot be empty")
	})

	t.Run("empty targetID", func(t *testing.T) {
		link := WikiLink{
			Target:   "target",
			LinkType: "wikilink",
		}

		edge, err := gb.createEdge("source", "", link, testTime)
		require.Error(t, err)
		assert.Nil(t, edge)
		assert.Contains(t, err.Error(), "targetID cannot be empty")
	})

	t.Run("both IDs empty", func(t *testing.T) {
		link := WikiLink{
			Target:   "target",
			LinkType: "wikilink",
		}

		edge, err := gb.createEdge("", "", link, testTime)
		require.Error(t, err)
		assert.Nil(t, edge)
		// Should fail on first validation
		assert.Contains(t, err.Error(), "sourceID cannot be empty")
	})
}

func TestBuildGraph_DeterministicOutput(t *testing.T) {
	// Test that multiple runs produce identical output order
	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	// Create files with IDs that would sort differently than creation order
	files := map[string]*MarkdownFile{
		"zebra": createTestMarkdownFile("zebra.md", "zebra", "Zebra", nil, []WikiLink{
			{Target: "alpha", LinkType: "wikilink"},
		}),
		"alpha": createTestMarkdownFile("alpha.md", "alpha", "Alpha", nil, []WikiLink{
			{Target: "beta", LinkType: "wikilink"},
		}),
		"beta": createTestMarkdownFile("beta.md", "beta", "Beta", nil, []WikiLink{
			{Target: "zebra", LinkType: "wikilink"},
		}),
	}

	resolver := NewLinkResolver()
	for _, file := range files {
		resolver.AddFile(file)
	}

	// Build graph multiple times
	var results []*Graph
	for i := 0; i < 5; i++ {
		parseResult := &ParseResult{
			Files:    files,
			Resolver: resolver,
		}

		result, err := gb.BuildGraph(parseResult)
		require.NoError(t, err)
		results = append(results, result)
	}

	// Verify all results have identical order
	for i := 1; i < len(results); i++ {
		// Check nodes order
		require.Equal(t, len(results[0].Nodes), len(results[i].Nodes))
		for j := 0; j < len(results[0].Nodes); j++ {
			assert.Equal(t, results[0].Nodes[j].ID, results[i].Nodes[j].ID,
				"Node order differs at index %d between run 0 and run %d", j, i)
		}

		// Check edges order
		require.Equal(t, len(results[0].Edges), len(results[i].Edges))
		for j := 0; j < len(results[0].Edges); j++ {
			assert.Equal(t, results[0].Edges[j].SourceID, results[i].Edges[j].SourceID,
				"Edge source order differs at index %d between run 0 and run %d", j, i)
			assert.Equal(t, results[0].Edges[j].TargetID, results[i].Edges[j].TargetID,
				"Edge target order differs at index %d between run 0 and run %d", j, i)
		}
	}

	// Verify nodes are sorted by ID
	assert.Equal(t, "alpha", results[0].Nodes[0].ID)
	assert.Equal(t, "beta", results[0].Nodes[1].ID)
	assert.Equal(t, "zebra", results[0].Nodes[2].ID)
}

func TestBuildGraph_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	gb := NewGraphBuilder(nil, GraphBuilderConfig{})

	// Create a large number of interconnected files
	fileCount := 1000
	files := make(map[string]*MarkdownFile)
	resolver := NewLinkResolver()

	for i := 0; i < fileCount; i++ {
		id := fmt.Sprintf("file%d", i)

		// Create links to next few files (circular)
		var links []WikiLink
		for j := 1; j <= 3; j++ {
			targetID := fmt.Sprintf("file%d", (i+j)%fileCount)
			links = append(links, WikiLink{
				Target:   targetID,
				LinkType: "wikilink",
			})
		}

		file := createTestMarkdownFile(
			fmt.Sprintf("file%d.md", i),
			id,
			fmt.Sprintf("File %d", i),
			nil,
			links,
		)

		files[id] = file
		resolver.AddFile(file)
	}

	parseResult := &ParseResult{
		Files:    files,
		Resolver: resolver,
	}

	start := time.Now()
	result, err := gb.BuildGraph(parseResult)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, result.Nodes, fileCount)
	assert.Len(t, result.Edges, fileCount*3) // Each file has 3 outgoing links

	// Performance assertion
	assert.Less(t, duration, 5*time.Second,
		"Building graph with %d nodes should take less than 5 seconds", fileCount)
	t.Logf("Built graph with %d nodes and %d edges in %v", fileCount, fileCount*3, duration)
}

// testFileInfo for testing (avoiding conflict with markdown_test.go)
type testFileInfo struct {
	modTime time.Time
}

func (m *testFileInfo) Name() string       { return "test.md" }
func (m *testFileInfo) Size() int64        { return 0 }
func (m *testFileInfo) Mode() os.FileMode  { return 0644 }
func (m *testFileInfo) ModTime() time.Time { return m.modTime }
func (m *testFileInfo) IsDir() bool        { return false }
func (m *testFileInfo) Sys() interface{}   { return nil }

