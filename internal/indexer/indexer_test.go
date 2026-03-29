package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleVault = "testdata/sample_vault"

func newTestIndexer(t *testing.T) (*Indexer, *store.Store) {
	t.Helper()
	s, err := store.NewMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	idx, err := New(s, sampleVault, nil)
	require.NoError(t, err)
	return idx, s
}

func TestFullIndex(t *testing.T) {
	idx, s := newTestIndexer(t)

	err := idx.FullIndex()
	require.NoError(t, err)

	nodes, err := s.GetAllNodes()
	require.NoError(t, err)
	assert.Greater(t, len(nodes), 0, "should have indexed some nodes")

	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	// The sample vault has inter-linked files, so we expect some edges
	t.Logf("Indexed %d nodes, %d edges", len(nodes), len(edges))

	// Verify metadata was set
	lastIndex, err := s.GetMetadata("last_index")
	require.NoError(t, err)
	assert.NotEmpty(t, lastIndex)
}

func TestFullIndexIdempotent(t *testing.T) {
	idx, s := newTestIndexer(t)

	require.NoError(t, idx.FullIndex())
	nodes1, _ := s.GetAllNodes()

	require.NoError(t, idx.FullIndex())
	nodes2, _ := s.GetAllNodes()

	assert.Equal(t, len(nodes1), len(nodes2), "re-indexing should produce same node count")
}

func TestFullIndexPreservesPositions(t *testing.T) {
	idx, s := newTestIndexer(t)
	require.NoError(t, idx.FullIndex())

	nodes, _ := s.GetAllNodes()
	require.Greater(t, len(nodes), 0)

	// Save a position for the first node
	require.NoError(t, s.UpsertPosition(&models.NodePosition{NodeID: nodes[0].ID, X: 42, Y: 99}))

	// Re-index
	require.NoError(t, idx.FullIndex())

	// Position should be preserved
	positions, err := s.GetAllPositions()
	require.NoError(t, err)
	assert.Len(t, positions, 1)
	assert.InDelta(t, 42, positions[0].X, 0.01)
}

func TestIndexFile(t *testing.T) {
	idx, s := newTestIndexer(t)

	// First do a full index so nodes exist for edge references
	require.NoError(t, idx.FullIndex())

	initialNodes, _ := s.GetAllNodes()

	// Re-index a specific file
	err := idx.IndexFile("index.md")
	require.NoError(t, err)

	afterNodes, _ := s.GetAllNodes()
	assert.Equal(t, len(initialNodes), len(afterNodes), "node count should remain the same")
}

func TestIndexFileNonexistent(t *testing.T) {
	idx, _ := newTestIndexer(t)

	// Indexing a path that doesn't produce a node should not error
	err := idx.IndexFile("nonexistent.md")
	// The parser will parse the whole vault; if the file doesn't exist
	// in the parse result, IndexFile just returns nil
	require.NoError(t, err)
}

func TestRemoveFile(t *testing.T) {
	idx, s := newTestIndexer(t)
	require.NoError(t, idx.FullIndex())

	nodesBefore, _ := s.GetAllNodes()
	require.Greater(t, len(nodesBefore), 0)

	// Remove the index file
	err := idx.RemoveFile("index.md")
	require.NoError(t, err)

	nodesAfter, _ := s.GetAllNodes()
	assert.Equal(t, len(nodesBefore)-1, len(nodesAfter))
}

func TestRemoveFileNonexistent(t *testing.T) {
	idx, _ := newTestIndexer(t)

	// Removing a file that doesn't exist in DB should not error
	err := idx.RemoveFile("doesnt-exist.md")
	require.NoError(t, err)
}

func TestIndexWithTempVault(t *testing.T) {
	// Create a temp vault with known content
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "note-a.md"), `---
id: "a"
tags: ["test"]
---
# Note A
Links to [[note-b]]
`)
	writeFile(t, filepath.Join(dir, "note-b.md"), `---
id: "b"
---
# Note B
Content of note B.
`)

	s, err := store.NewMemory()
	require.NoError(t, err)
	defer s.Close()

	idx, err := New(s, dir, nil)
	require.NoError(t, err)

	require.NoError(t, idx.FullIndex())

	nodes, err := s.GetAllNodes()
	require.NoError(t, err)
	assert.Len(t, nodes, 2)

	// Check node A
	nodeA, err := s.GetNode("a")
	require.NoError(t, err)
	assert.NotEmpty(t, nodeA.Title)

	// Check edge from a -> b
	edges, err := s.GetAllEdges()
	require.NoError(t, err)
	assert.Greater(t, len(edges), 0, "should have edge from a to b")

	foundEdge := false
	for _, e := range edges {
		if e.SourceID == "a" && e.TargetID == "b" {
			foundEdge = true
			break
		}
	}
	assert.True(t, foundEdge, "expected edge from a to b")
}

func TestIncrementalAddFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "note-a.md"), `---
id: "a"
---
# Note A
`)

	s, err := store.NewMemory()
	require.NoError(t, err)
	defer s.Close()

	idx, err := New(s, dir, nil)
	require.NoError(t, err)

	require.NoError(t, idx.FullIndex())
	nodes, _ := s.GetAllNodes()
	assert.Len(t, nodes, 1)

	// Add a new file to the vault
	writeFile(t, filepath.Join(dir, "note-b.md"), `---
id: "b"
---
# Note B
`)

	// Incremental index of the new file
	require.NoError(t, idx.IndexFile("note-b.md"))

	nodes, _ = s.GetAllNodes()
	assert.Len(t, nodes, 2)
}

func TestIncrementalRemoveFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "note-a.md"), `---
id: "a"
---
# Note A
`)
	writeFile(t, filepath.Join(dir, "note-b.md"), `---
id: "b"
---
# Note B
Links to [[note-a]]
`)

	s, err := store.NewMemory()
	require.NoError(t, err)
	defer s.Close()

	idx, err := New(s, dir, nil)
	require.NoError(t, err)

	require.NoError(t, idx.FullIndex())
	nodes, _ := s.GetAllNodes()
	assert.Len(t, nodes, 2)

	// Remove note-a from disk and from DB
	os.Remove(filepath.Join(dir, "note-a.md"))
	require.NoError(t, idx.RemoveFile("note-a.md"))

	nodes, _ = s.GetAllNodes()
	assert.Len(t, nodes, 1)
	assert.Equal(t, "b", nodes[0].ID)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
