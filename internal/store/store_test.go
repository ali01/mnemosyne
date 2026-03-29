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

// createTestVault creates a vault and returns its ID.
func createTestVault(t *testing.T, s *Store, name, path string) int {
	t.Helper()
	id, err := s.UpsertVault(name, path)
	require.NoError(t, err)
	return id
}

// createTestGraph creates a graph and returns its ID.
func createTestGraph(t *testing.T, s *Store, vaultID int, name, rootPath string) int {
	t.Helper()
	id, err := s.UpsertGraph(vaultID, name, rootPath, "")
	require.NoError(t, err)
	return id
}

func testNode(vaultID int, id, title, path string) models.VaultNode {
	return models.VaultNode{
		ID:        id,
		VaultID:   vaultID,
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

// --- Vault tests ---

func TestUpsertVault(t *testing.T) {
	s := newTestStore(t)
	id, err := s.UpsertVault("walros", "/home/walros")
	require.NoError(t, err)
	assert.Greater(t, id, 0)
}

func TestUpsertVaultIdempotent(t *testing.T) {
	s := newTestStore(t)
	id1, _ := s.UpsertVault("walros", "/home/walros")
	id2, _ := s.UpsertVault("walros-renamed", "/home/walros")
	assert.Equal(t, id1, id2)
}

func TestGetVaults(t *testing.T) {
	s := newTestStore(t)
	s.UpsertVault("alpha", "/alpha")
	s.UpsertVault("beta", "/beta")

	vaults, err := s.GetVaults()
	require.NoError(t, err)
	assert.Len(t, vaults, 2)
	assert.Equal(t, "alpha", vaults[0].Name)
	assert.Equal(t, "beta", vaults[1].Name)
}

// --- Graph tests ---

func TestUpsertGraph(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")

	gid, err := s.UpsertGraph(vid, "root", "", "")
	require.NoError(t, err)
	assert.Greater(t, gid, 0)
}

func TestUpsertGraphIdempotent(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")

	id1, _ := s.UpsertGraph(vid, "root", "", "")
	id2, _ := s.UpsertGraph(vid, "root-updated", "", "{}")
	assert.Equal(t, id1, id2)
}

func TestGetGraphInfo(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "walros", "/walros")
	gid := createTestGraph(t, s, vid, "concepts", "concepts")

	g, err := s.GetGraphInfo(gid)
	require.NoError(t, err)
	assert.Equal(t, "concepts", g.Name)
	assert.Equal(t, "concepts", g.RootPath)
	assert.Equal(t, "walros", g.VaultName)
}

func TestGetGraphsByVault(t *testing.T) {
	s := newTestStore(t)
	v1 := createTestVault(t, s, "v1", "/v1")
	v2 := createTestVault(t, s, "v2", "/v2")

	createTestGraph(t, s, v1, "root", "")
	createTestGraph(t, s, v1, "concepts", "concepts")
	createTestGraph(t, s, v2, "root", "")

	graphs, err := s.GetGraphsByVault(v1)
	require.NoError(t, err)
	assert.Len(t, graphs, 2)
}

func TestGetAllGraphs(t *testing.T) {
	s := newTestStore(t)
	v1 := createTestVault(t, s, "v1", "/v1")
	v2 := createTestVault(t, s, "v2", "/v2")

	createTestGraph(t, s, v1, "root", "")
	createTestGraph(t, s, v2, "root", "")

	graphs, err := s.GetAllGraphs()
	require.NoError(t, err)
	assert.Len(t, graphs, 2)
}

func TestDeleteStaleGraphs(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")

	g1 := createTestGraph(t, s, vid, "root", "")
	g2 := createTestGraph(t, s, vid, "concepts", "concepts")
	createTestGraph(t, s, vid, "stale", "stale")

	require.NoError(t, s.DeleteStaleGraphs(vid, []int{g1, g2}))

	graphs, _ := s.GetGraphsByVault(vid)
	assert.Len(t, graphs, 2)
}

func TestDeleteGraphCascadesPositions(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gid := createTestGraph(t, s, vid, "root", "")
	n := testNode(vid, "n1", "N", "n.md")
	require.NoError(t, s.UpsertNode(&n))
	require.NoError(t, s.UpsertPosition(gid, &models.NodePosition{NodeID: "n1", X: 10, Y: 20}))

	require.NoError(t, s.DeleteGraph(gid))

	// Position should be gone
	graph, err := s.GetGraphData(gid)
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 0)
}

// --- Node tests ---

func TestUpsertAndGetNode(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	n := testNode(vid, "n1", "Test Node", "test.md")

	err := s.UpsertNode(&n)
	require.NoError(t, err)

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "n1", got.ID)
	assert.Equal(t, vid, got.VaultID)
	assert.Equal(t, "Test Node", got.Title)
	assert.Equal(t, "test.md", got.FilePath)
	assert.Equal(t, "note", got.NodeType)
	assert.Contains(t, got.Content, "Some content here.")
}

func TestUpsertNodeUpdatesExisting(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	n := testNode(vid, "n1", "Original", "test.md")
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

func TestGetNodeByVaultPath(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	n := testNode(vid, "n1", "Test", "folder/test.md")
	require.NoError(t, s.UpsertNode(&n))

	got, err := s.GetNodeByVaultPath(vid, "folder/test.md")
	require.NoError(t, err)
	assert.Equal(t, "n1", got.ID)
}

func TestDeleteNode(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	n := testNode(vid, "n1", "Test", "test.md")
	require.NoError(t, s.UpsertNode(&n))

	require.NoError(t, s.DeleteNode("n1"))

	_, err := s.GetNode("n1")
	assert.Error(t, err)
}

func TestDeleteNodeCascadesEdges(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", VaultID: vid, Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", VaultID: vid, Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e1", SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1}))

	require.NoError(t, s.DeleteNode("a"))

	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, edges, 0)
}

func TestDeleteNodeCascadesGraphNodes(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gid := createTestGraph(t, s, vid, "root", "")
	n := testNode(vid, "n1", "N", "n.md")
	require.NoError(t, s.UpsertNode(&n))
	require.NoError(t, s.ReplaceGraphMemberships("n1", []int{gid}))

	require.NoError(t, s.DeleteNode("n1"))

	graph, _ := s.GetGraphData(gid)
	assert.Len(t, graph.Nodes, 0)
}

func TestGetAllNodes(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", VaultID: vid, Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", VaultID: vid, Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	nodes, err := s.GetAllNodes()
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
}

// --- Edge tests ---

func TestUpsertAndGetEdges(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", VaultID: vid, Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", VaultID: vid, Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	e := testEdge("a", "b")
	require.NoError(t, s.UpsertEdge(&e))

	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, edges, 1)
	assert.Equal(t, "a", edges[0].SourceID)
}

func TestDeleteEdgesBySource(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", VaultID: vid, Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", VaultID: vid, Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "c", VaultID: vid, Title: "C", FilePath: "c.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e1", SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e2", SourceID: "a", TargetID: "c", EdgeType: "wikilink", Weight: 1}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e3", SourceID: "b", TargetID: "c", EdgeType: "wikilink", Weight: 1}))

	require.NoError(t, s.DeleteEdgesBySource("a"))

	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, edges, 1)
	assert.Equal(t, "b", edges[0].SourceID)
}

// --- Graph-scoped data tests ---

func TestGetGraphDataScopesNodes(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gRoot := createTestGraph(t, s, vid, "root", "")
	gConcepts := createTestGraph(t, s, vid, "concepts", "concepts")

	// Nodes: index.md (root only), concepts/AI.md (both), projects/X.md (root only)
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "idx", VaultID: vid, Title: "Index", FilePath: "index.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "ai", VaultID: vid, Title: "AI", FilePath: "concepts/AI.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "proj", VaultID: vid, Title: "Project", FilePath: "projects/X.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	// Root graph has all 3 nodes, concepts graph has only AI
	require.NoError(t, s.ReplaceGraphMemberships("idx", []int{gRoot}))
	require.NoError(t, s.ReplaceGraphMemberships("ai", []int{gRoot, gConcepts}))
	require.NoError(t, s.ReplaceGraphMemberships("proj", []int{gRoot}))

	rootGraph, err := s.GetGraphData(gRoot)
	require.NoError(t, err)
	assert.Len(t, rootGraph.Nodes, 3)

	conceptsGraph, err := s.GetGraphData(gConcepts)
	require.NoError(t, err)
	assert.Len(t, conceptsGraph.Nodes, 1)
	assert.Equal(t, "ai", conceptsGraph.Nodes[0].ID)
}

func TestGetGraphDataScopesEdges(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gRoot := createTestGraph(t, s, vid, "root", "")
	gConcepts := createTestGraph(t, s, vid, "concepts", "concepts")

	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "idx", VaultID: vid, Title: "Index", FilePath: "index.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "ai", VaultID: vid, Title: "AI", FilePath: "concepts/AI.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "net", VaultID: vid, Title: "Network", FilePath: "concepts/Net.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	// Edges: idx->ai, ai->net, idx->net
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e1", SourceID: "idx", TargetID: "ai", EdgeType: "wikilink", Weight: 1}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e2", SourceID: "ai", TargetID: "net", EdgeType: "wikilink", Weight: 1}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{ID: "e3", SourceID: "idx", TargetID: "net", EdgeType: "wikilink", Weight: 1}))

	// Root: all 3 nodes. Concepts: ai + net only.
	require.NoError(t, s.ReplaceGraphMemberships("idx", []int{gRoot}))
	require.NoError(t, s.ReplaceGraphMemberships("ai", []int{gRoot, gConcepts}))
	require.NoError(t, s.ReplaceGraphMemberships("net", []int{gRoot, gConcepts}))

	// Root graph should have all 3 edges
	rootGraph, err := s.GetGraphData(gRoot)
	require.NoError(t, err)
	assert.Len(t, rootGraph.Edges, 3)

	// Concepts graph: only ai->net (idx is not in concepts)
	conceptsGraph, err := s.GetGraphData(gConcepts)
	require.NoError(t, err)
	assert.Len(t, conceptsGraph.Edges, 1)
	assert.Equal(t, "ai", conceptsGraph.Edges[0].Source)
	assert.Equal(t, "net", conceptsGraph.Edges[0].Target)
}

func TestGetGraphDataPositionsIndependent(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	g1 := createTestGraph(t, s, vid, "g1", "g1")
	g2 := createTestGraph(t, s, vid, "g2", "g2")

	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "n1", VaultID: vid, Title: "N", FilePath: "n.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.ReplaceGraphMemberships("n1", []int{g1, g2}))

	// Different positions in each graph
	require.NoError(t, s.UpsertPosition(g1, &models.NodePosition{NodeID: "n1", X: 10, Y: 20}))
	require.NoError(t, s.UpsertPosition(g2, &models.NodePosition{NodeID: "n1", X: 99, Y: 88}))

	graph1, _ := s.GetGraphData(g1)
	graph2, _ := s.GetGraphData(g2)

	assert.InDelta(t, 10, graph1.Nodes[0].Position.X, 0.01)
	assert.InDelta(t, 99, graph2.Nodes[0].Position.X, 0.01)
}

func TestGetGraphDataEmptyGraph(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gid := createTestGraph(t, s, vid, "empty", "empty")

	graph, err := s.GetGraphData(gid)
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 0)
	assert.Len(t, graph.Edges, 0)
}

// --- Search tests ---

func TestSearchInGraph(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	g1 := createTestGraph(t, s, vid, "g1", "g1")
	g2 := createTestGraph(t, s, vid, "g2", "g2")

	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "n1", VaultID: vid, Title: "Aviation History", FilePath: "aviation.md",
		Content: "The history of powered flight.", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "n2", VaultID: vid, Title: "Economics", FilePath: "econ.md",
		Content: "Supply and demand.", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// n1 in g1 only, n2 in g2 only
	require.NoError(t, s.ReplaceGraphMemberships("n1", []int{g1}))
	require.NoError(t, s.ReplaceGraphMemberships("n2", []int{g2}))

	results, err := s.SearchInGraph(g1, "aviation")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "n1", results[0].ID)

	// Searching g2 for aviation should return nothing
	results, err = s.SearchInGraph(g2, "aviation")
	require.NoError(t, err)
	assert.Nil(t, results)
}

// --- Position tests ---

func TestUpsertAndGetPositions(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gid := createTestGraph(t, s, vid, "root", "")

	require.NoError(t, s.UpsertPosition(gid, &models.NodePosition{NodeID: "n1", X: 10.5, Y: 20.3}))

	graph, err := s.GetGraphData(gid)
	require.NoError(t, err)
	// No nodes in graph (no membership), but position was saved.
	// Test via a node that IS in the graph.
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "n1", VaultID: vid, Title: "N", FilePath: "n.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.ReplaceGraphMemberships("n1", []int{gid}))

	graph, err = s.GetGraphData(gid)
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 1)
	assert.InDelta(t, 10.5, graph.Nodes[0].Position.X, 0.01)
	assert.InDelta(t, 20.3, graph.Nodes[0].Position.Y, 0.01)
}

func TestUpsertPositionsBatch(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gid := createTestGraph(t, s, vid, "root", "")

	batch := []models.NodePosition{
		{NodeID: "a", X: 1, Y: 2},
		{NodeID: "b", X: 3, Y: 4},
	}
	require.NoError(t, s.UpsertPositions(gid, batch))

	// Verify by adding nodes to graph and reading back
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", VaultID: vid, Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", VaultID: vid, Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.ReplaceGraphMemberships("a", []int{gid}))
	require.NoError(t, s.ReplaceGraphMemberships("b", []int{gid}))

	graph, err := s.GetGraphData(gid)
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 2)
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

// --- Bulk operations ---

func TestReplaceVaultData(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gid := createTestGraph(t, s, vid, "root", "")

	nodes := []models.VaultNode{
		{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "b", Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	edges := []models.VaultEdge{
		{SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1},
	}
	memberships := map[int][]string{
		gid: {"a", "b"},
	}

	require.NoError(t, s.ReplaceVaultData(vid, nodes, edges, memberships))

	graph, err := s.GetGraphData(gid)
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 2)
	assert.Len(t, graph.Edges, 1)
}

func TestReplaceVaultDataPreservesOtherVault(t *testing.T) {
	s := newTestStore(t)
	v1 := createTestVault(t, s, "v1", "/v1")
	v2 := createTestVault(t, s, "v2", "/v2")
	g1 := createTestGraph(t, s, v1, "root", "")
	g2 := createTestGraph(t, s, v2, "root", "")

	// Insert data for v2
	require.NoError(t, s.ReplaceVaultData(v2, []models.VaultNode{
		{ID: "v2n1", Title: "V2 Node", FilePath: "v2.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}, nil, map[int][]string{g2: {"v2n1"}}))

	// Replace v1 data (should not touch v2)
	require.NoError(t, s.ReplaceVaultData(v1, []models.VaultNode{
		{ID: "v1n1", Title: "V1 Node", FilePath: "v1.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}, nil, map[int][]string{g1: {"v1n1"}}))

	// v2 data should still exist
	graph2, err := s.GetGraphData(g2)
	require.NoError(t, err)
	assert.Len(t, graph2.Nodes, 1)
	assert.Equal(t, "v2n1", graph2.Nodes[0].ID)
}

func TestReplaceVaultDataPreservesPositions(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gid := createTestGraph(t, s, vid, "root", "")

	// Insert initial data + positions
	require.NoError(t, s.ReplaceVaultData(vid, []models.VaultNode{
		{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}, nil, map[int][]string{gid: {"a"}}))
	require.NoError(t, s.UpsertPosition(gid, &models.NodePosition{NodeID: "a", X: 42, Y: 99}))

	// Re-replace (same node ID)
	require.NoError(t, s.ReplaceVaultData(vid, []models.VaultNode{
		{ID: "a", Title: "A Updated", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}, nil, map[int][]string{gid: {"a"}}))

	graph, err := s.GetGraphData(gid)
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 1)
	assert.InDelta(t, 42, graph.Nodes[0].Position.X, 0.01)
}

func TestReplaceVaultDataSkipsSelfReferentialEdges(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")

	nodes := []models.VaultNode{
		{ID: "a", Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	edges := []models.VaultEdge{
		{SourceID: "a", TargetID: "a", EdgeType: "wikilink", Weight: 1},
	}
	require.NoError(t, s.ReplaceVaultData(vid, nodes, edges, nil))

	allEdges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Len(t, allEdges, 0)
}

// --- Graph membership tests ---

func TestReplaceGraphMemberships(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	g1 := createTestGraph(t, s, vid, "g1", "g1")
	g2 := createTestGraph(t, s, vid, "g2", "g2")

	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "n1", VaultID: vid, Title: "N", FilePath: "n.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	// Initially in g1 only
	require.NoError(t, s.ReplaceGraphMemberships("n1", []int{g1}))
	graph1, _ := s.GetGraphData(g1)
	graph2, _ := s.GetGraphData(g2)
	assert.Len(t, graph1.Nodes, 1)
	assert.Len(t, graph2.Nodes, 0)

	// Move to both g1 and g2
	require.NoError(t, s.ReplaceGraphMemberships("n1", []int{g1, g2}))
	graph1, _ = s.GetGraphData(g1)
	graph2, _ = s.GetGraphData(g2)
	assert.Len(t, graph1.Nodes, 1)
	assert.Len(t, graph2.Nodes, 1)
}

func TestGraphNodeCountInList(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gid := createTestGraph(t, s, vid, "root", "")

	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "a", VaultID: vid, Title: "A", FilePath: "a.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "b", VaultID: vid, Title: "B", FilePath: "b.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s.ReplaceGraphMemberships("a", []int{gid}))
	require.NoError(t, s.ReplaceGraphMemberships("b", []int{gid}))

	graphs, err := s.GetAllGraphs()
	require.NoError(t, err)
	assert.Len(t, graphs, 1)
	assert.Equal(t, 2, graphs[0].NodeCount)
}

// --- Tags and metadata round-trip ---

func TestTagsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	n := testNode(vid, "n1", "Test", "test.md")
	n.Tags = models.StringArray{"alpha", "beta", "gamma"}
	require.NoError(t, s.UpsertNode(&n))

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, models.StringArray{"alpha", "beta", "gamma"}, got.Tags)
}

// --- Large batch ---

func TestReplaceVaultDataLargeBatch(t *testing.T) {
	s := newTestStore(t)
	vid := createTestVault(t, s, "v", "/v")
	gid := createTestGraph(t, s, vid, "root", "")

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

	allNodeIDs := make([]string, len(nodes))
	for i, n := range nodes {
		allNodeIDs[i] = n.ID
	}
	memberships := map[int][]string{gid: allNodeIDs}

	require.NoError(t, s.ReplaceVaultData(vid, nodes, edges, memberships))

	graph, err := s.GetGraphData(gid)
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 500)
	assert.Len(t, graph.Edges, 400)
}

// --- File-based store ---

func TestNewFileStore(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/sub/mnemosyne.db"

	s, err := New(dbPath)
	require.NoError(t, err)
	defer s.Close()

	vid, _ := s.UpsertVault("test", "/test")
	require.NoError(t, s.UpsertNode(&models.VaultNode{ID: "n1", VaultID: vid, Title: "T", FilePath: "t.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	got, err := s.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "n1", got.ID)
}

func TestFileStorePersists(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/mnemosyne.db"

	s1, err := New(dbPath)
	require.NoError(t, err)
	vid, _ := s1.UpsertVault("test", "/test")
	require.NoError(t, s1.UpsertNode(&models.VaultNode{ID: "n1", VaultID: vid, Title: "T", FilePath: "t.md", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, s1.Close())

	s2, err := New(dbPath)
	require.NoError(t, err)
	defer s2.Close()

	got, err := s2.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "n1", got.ID)
}
