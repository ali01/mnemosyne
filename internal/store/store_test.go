package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func testNode(id, title, path string) models.VaultNode {
	return models.VaultNode{
		ID:        id,
		Title:     title,
		FilePath:  path,
		Content:   "# " + title + "\nSome content here.",
		NodeType:  "note",
		Tags:      models.StringArray{"tag1", "tag2"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func testEdge(source, target string) models.VaultEdge {
	return models.VaultEdge{
		SourceID: source,
		TargetID: target,
		EdgeType: "wikilink",
		Weight:   1.0,
	}
}

// --- Node tests ---

func TestUpsertAndGetNode(t *testing.T) {
	s := newTestStore(t)
	n := testNode("n1", "Test Node", "test.md")

	err := s.UpsertNode(&n)
	require.NoError(t, err)

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "n1", got.ID)
	assert.Equal(t, "Test Node", got.Title)
	assert.Equal(t, "test.md", got.FilePath)
	assert.Equal(t, "note", got.NodeType)
	assert.Contains(t, got.Content, "Some content here.")
}

func TestUpsertNodeUpdatesExisting(t *testing.T) {
	s := newTestStore(t)
	n := testNode("n1", "Original", "test.md")
	require.NoError(t, s.UpsertNode(&n))

	n.Title = "Updated"
	require.NoError(t, s.UpsertNode(&n))

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
}

func TestGetNodeNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetNode("nonexistent")
	assert.Error(t, err)
}

func TestGetNodeByPath(t *testing.T) {
	s := newTestStore(t)
	n := testNode("n1", "Test", "folder/test.md")
	require.NoError(t, s.UpsertNode(&n))

	got, err := s.GetNodeByPath("folder/test.md")
	require.NoError(t, err)
	assert.Equal(t, "n1", got.ID)
}

func TestDeleteNode(t *testing.T) {
	s := newTestStore(t)
	n := testNode("n1", "Test", "test.md")
	require.NoError(t, s.UpsertNode(&n))

	require.NoError(t, s.DeleteNode("n1"))

	_, err := s.GetNode("n1")
	assert.Error(t, err)
}

func TestDeleteNodeCascadesEdges(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e1", SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1}))

	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, edges, 1)

	require.NoError(t, s.DeleteNode("a"))

	edges, err = s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, edges, 0)
}

func TestGetAllNodes(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	nodes, err := s.GetAllNodes()
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestGetAllNodesEmpty(t *testing.T) {
	s := newTestStore(t)
	nodes, err := s.GetAllNodes()
	require.NoError(t, err)
	assert.Nil(t, nodes)
}

// --- Edge tests ---

func TestUpsertAndGetEdges(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	e := testEdge("a", "b")
	require.NoError(t, s.UpsertEdge(&e))

	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, edges, 1)
	assert.Equal(t, "a", edges[0].SourceID)
	assert.Equal(t, "b", edges[0].TargetID)
}

func TestUpsertEdgeGeneratesID(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	e := models.VaultEdge{SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1}
	require.NoError(t, s.UpsertEdge(&e))
	assert.NotEmpty(t, e.ID)
}

func TestDeleteEdgesBySource(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "c", Title: "C", FilePath: "c.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e1", SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e2", SourceID: "a", TargetID: "c", EdgeType: "wikilink", Weight: 1}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e3", SourceID: "b", TargetID: "c", EdgeType: "wikilink", Weight: 1}))

	require.NoError(t, s.DeleteEdgesBySource("a"))

	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, edges, 1)
	assert.Equal(t, "b", edges[0].SourceID)
}

func TestDeleteEdgesByNode(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "c", Title: "C", FilePath: "c.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e1", SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e2", SourceID: "c", TargetID: "b", EdgeType: "wikilink", Weight: 1}))

	require.NoError(t, s.DeleteEdgesByNode("b"))

	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, edges, 0)
}

// --- Position tests ---

func TestUpsertAndGetPositions(t *testing.T) {
	s := newTestStore(t)

	p := models.NodePosition{NodeID: "n1", X: 10.5, Y: 20.3, Z: 0}
	require.NoError(t, s.UpsertPosition(&p))

	positions, err := s.GetAllPositions()
	require.NoError(t, err)
	assert.Len(t, positions, 1)
	assert.Equal(t, "n1", positions[0].NodeID)
	assert.InDelta(t, 10.5, positions[0].X, 0.01)
	assert.InDelta(t, 20.3, positions[0].Y, 0.01)
}

func TestUpsertPositionUpdates(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertPosition(&models.NodePosition{NodeID: "n1", X: 1, Y: 2}))
	require.NoError(t, s.UpsertPosition(&models.NodePosition{NodeID: "n1", X: 99, Y: 88}))

	positions, err := s.GetAllPositions()
	require.NoError(t, err)
	assert.Len(t, positions, 1)
	assert.InDelta(t, 99, positions[0].X, 0.01)
	assert.InDelta(t, 88, positions[0].Y, 0.01)
}

func TestUpsertPositionsBatch(t *testing.T) {
	s := newTestStore(t)
	batch := []models.NodePosition{
		{NodeID: "a", X: 1, Y: 2},
		{NodeID: "b", X: 3, Y: 4},
		{NodeID: "c", X: 5, Y: 6},
	}
	require.NoError(t, s.UpsertPositions(batch))

	positions, err := s.GetAllPositions()
	require.NoError(t, err)
	assert.Len(t, positions, 3)
}

// --- Search tests ---

func TestSearchNodes(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "n1", Title: "Aviation History", FilePath: "aviation.md",
		Content: "The history of powered flight begins with the Wright brothers.",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "n2", Title: "Economics", FilePath: "econ.md",
		Content: "Supply and demand are fundamental concepts.",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	results, err := s.SearchNodes("aviation")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "n1", results[0].ID)
}

func TestSearchNodesNoResults(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "n1", Title: "Test", FilePath: "test.md", Content: "hello",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	results, err := s.SearchNodes("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, results)
}

// --- Metadata tests ---

func TestSetAndGetMetadata(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.SetMetadata("last_index", "2024-01-01"))

	val, err := s.GetMetadata("last_index")
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01", val)
}

func TestGetMetadataNotFound(t *testing.T) {
	s := newTestStore(t)
	val, err := s.GetMetadata("missing")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestSetMetadataUpdates(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.SetMetadata("key", "v1"))
	require.NoError(t, s.SetMetadata("key", "v2"))

	val, err := s.GetMetadata("key")
	require.NoError(t, err)
	assert.Equal(t, "v2", val)
}

// --- Bulk operations ---

func TestReplaceAllNodesAndEdges(t *testing.T) {
	s := newTestStore(t)

	// Insert initial data
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "old", Title: "Old", FilePath: "old.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	// Replace with new data
	nodes := []models.VaultNode{
		{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "b", Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	edges := []models.VaultEdge{
		{SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1},
	}
	require.NoError(t, s.ReplaceAllNodesAndEdges(nodes, edges))

	allNodes, err := s.GetAllNodes()
	require.NoError(t, err)
	assert.Len(t, allNodes, 2)

	allEdges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, allEdges, 1)

	// Old node should be gone
	_, err = s.GetNode("old")
	assert.Error(t, err)
}

func TestReplaceAllSkipsSelfReferentialEdges(t *testing.T) {
	s := newTestStore(t)

	nodes := []models.VaultNode{
		{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	edges := []models.VaultEdge{
		{SourceID: "a", TargetID: "a", EdgeType: "wikilink", Weight: 1},
	}
	require.NoError(t, s.ReplaceAllNodesAndEdges(nodes, edges))

	allEdges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, allEdges, 0)
}

func TestReplaceAllPreservesPositions(t *testing.T) {
	s := newTestStore(t)

	// Save a position
	require.NoError(t, s.UpsertPosition(&models.NodePosition{NodeID: "a", X: 42, Y: 99}))

	// Replace nodes (position table is independent)
	nodes := []models.VaultNode{
		{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	require.NoError(t, s.ReplaceAllNodesAndEdges(nodes, nil))

	positions, err := s.GetAllPositions()
	require.NoError(t, err)
	assert.Len(t, positions, 1)
	assert.InDelta(t, 42, positions[0].X, 0.01)
}

// --- Tags and metadata round-trip ---

func TestTagsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	n := testNode("n1", "Test", "test.md")
	n.Tags = models.StringArray{"alpha", "beta", "gamma"}
	require.NoError(t, s.UpsertNode(&n))

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, models.StringArray{"alpha", "beta", "gamma"}, got.Tags)
}

func TestNilTagsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	n := models.VaultNode{ID: "n1", Title: "T", FilePath: "t.md", Tags: nil, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, s.UpsertNode(&n))

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Nil(t, got.Tags)
}

func TestNullNodeType(t *testing.T) {
	s := newTestStore(t)
	n := models.VaultNode{ID: "n1", Title: "T", FilePath: "t.md", NodeType: "", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, s.UpsertNode(&n))

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "", got.NodeType)
}

func TestGetNodeReturnsContent(t *testing.T) {
	s := newTestStore(t)
	n := testNode("n1", "Test", "test.md")
	n.Content = "Full markdown content here"
	require.NoError(t, s.UpsertNode(&n))

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "Full markdown content here", got.Content)

	// GetAllNodes should NOT return content
	all, err := s.GetAllNodes()
	require.NoError(t, err)
	assert.Equal(t, "", all[0].Content)
}

// --- Edge upsert on conflict ---

func TestUpsertEdgeUpdatesOnConflict(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	e1 := models.VaultEdge{ID: "e1", SourceID: "a", TargetID: "b", EdgeType: "wikilink", DisplayText: "original", Weight: 1}
	require.NoError(t, s.UpsertEdge(&e1))

	e2 := models.VaultEdge{ID: "e2", SourceID: "a", TargetID: "b", EdgeType: "wikilink", DisplayText: "updated", Weight: 2}
	require.NoError(t, s.UpsertEdge(&e2))

	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, edges, 1)
	assert.Equal(t, "updated", edges[0].DisplayText)
	assert.InDelta(t, 2.0, edges[0].Weight, 0.01)
}

// --- Batch positions edge cases ---

func TestUpsertPositionsEmpty(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertPositions(nil))
	require.NoError(t, s.UpsertPositions([]models.NodePosition{}))

	positions, err := s.GetAllPositions()
	require.NoError(t, err)
	assert.Nil(t, positions)
}

// --- File-based store ---

func TestNewFileStore(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/sub/mnemosyne.db"

	s, err := New(dbPath)
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "n1", Title: "T", FilePath: "t.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "n1", got.ID)
}

func TestFileStorePersists(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/mnemosyne.db"

	// Write
	s1, err := New(dbPath)
	require.NoError(t, err)
	require.NoError(t, s1.UpsertNode(&models.VaultNode{ID: "n1", Title: "T", FilePath: "t.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s1.Close())

	// Re-open and read
	s2, err := New(dbPath)
	require.NoError(t, err)
	defer s2.Close()

	got, err := s2.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "n1", got.ID)
}

// --- Search edge cases ---

func TestSearchWithSpecialCharacters(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "n1", Title: "C++ Programming", FilePath: "cpp.md",
		Content: "Templates and move semantics",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// FTS5 treats special chars as separators, so "C" should match
	results, err := s.SearchNodes("programming")
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

// --- Large batch ---

func TestReplaceAllLargeBatch(t *testing.T) {
	s := newTestStore(t)

	nodes := make([]models.VaultNode, 500)
	for i := range nodes {
		nodes[i] = models.VaultNode{
			ID:        fmt.Sprintf("n%d", i),
			Title:     fmt.Sprintf("Node %d", i),
			FilePath:  fmt.Sprintf("node%d.md", i),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	edges := make([]models.VaultEdge, 400)
	for i := range edges {
		edges[i] = models.VaultEdge{
			SourceID: fmt.Sprintf("n%d", i),
			TargetID: fmt.Sprintf("n%d", i+1),
			EdgeType: "wikilink",
			Weight:   1,
		}
	}

	require.NoError(t, s.ReplaceAllNodesAndEdges(nodes, edges))

	allNodes, err := s.GetAllNodes()
	require.NoError(t, err)
	assert.Len(t, allNodes, 500)

	allEdges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, allEdges, 400)
}

// --- Graph API ---

func TestGetGraph(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", Title: "A", FilePath: "a.md", NodeType: "hub", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", Title: "B", FilePath: "b.md", NodeType: "note", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e1", SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1}))
	require.NoError(t, s.UpsertPosition(&models.NodePosition{NodeID: "a", X: 10, Y: 20}))

	graph, err := s.GetGraph()
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 2)
	assert.Len(t, graph.Edges, 1)
	// Check position was applied to the node
	for _, n := range graph.Nodes {
		if n.ID == "a" {
			assert.InDelta(t, 10, n.Position.X, 0.01)
		}
	}
	assert.Equal(t, "hub", graph.Nodes[0].Metadata["type"])
}
