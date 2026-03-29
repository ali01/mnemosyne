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

func newTestManager(t *testing.T) (*IndexManager, *store.Store) {
	t.Helper()
	s, err := store.NewMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return NewIndexManager(s), s
}

func TestRegisterVault(t *testing.T) {
	m, _ := newTestManager(t)

	// Add GRAPH.yaml to sample vault for discovery
	dir := t.TempDir()
	copyVault(t, sampleVault, dir)
	writeFile(t, filepath.Join(dir, "GRAPH.yaml"), "")

	vaultID, graphIDs, err := m.RegisterVault(dir)
	require.NoError(t, err)
	assert.Greater(t, vaultID, 0)
	assert.Len(t, graphIDs, 1)
}

func TestFullIndexVault(t *testing.T) {
	m, s := newTestManager(t)

	dir := t.TempDir()
	copyVault(t, sampleVault, dir)
	writeFile(t, filepath.Join(dir, "GRAPH.yaml"), "")

	vaultID, graphIDs, err := m.RegisterVault(dir)
	require.NoError(t, err)

	require.NoError(t, m.FullIndexVault(vaultID))

	graph, err := s.GetGraphData(graphIDs[0])
	require.NoError(t, err)
	assert.Greater(t, len(graph.Nodes), 0)
	t.Logf("Indexed %d nodes, %d edges", len(graph.Nodes), len(graph.Edges))
}

func TestFullIndexVaultIdempotent(t *testing.T) {
	m, s := newTestManager(t)

	dir := t.TempDir()
	copyVault(t, sampleVault, dir)
	writeFile(t, filepath.Join(dir, "GRAPH.yaml"), "")

	vaultID, graphIDs, _ := m.RegisterVault(dir)
	require.NoError(t, m.FullIndexVault(vaultID))

	g1, _ := s.GetGraphData(graphIDs[0])
	count1 := len(g1.Nodes)

	require.NoError(t, m.FullIndexVault(vaultID))
	g2, _ := s.GetGraphData(graphIDs[0])
	assert.Equal(t, count1, len(g2.Nodes))
}

func TestFullIndexPreservesPositions(t *testing.T) {
	m, s := newTestManager(t)

	dir := t.TempDir()
	copyVault(t, sampleVault, dir)
	writeFile(t, filepath.Join(dir, "GRAPH.yaml"), "")

	vaultID, graphIDs, _ := m.RegisterVault(dir)
	require.NoError(t, m.FullIndexVault(vaultID))

	gid := graphIDs[0]
	graph, _ := s.GetGraphData(gid)
	require.Greater(t, len(graph.Nodes), 0)

	require.NoError(t, s.UpsertPosition(gid, &models.NodePosition{NodeID: graph.Nodes[0].ID, X: 42, Y: 99}))
	require.NoError(t, m.FullIndexVault(vaultID))

	graph2, _ := s.GetGraphData(gid)
	for _, n := range graph2.Nodes {
		if n.ID == graph.Nodes[0].ID {
			assert.InDelta(t, 42, n.Position.X, 0.01)
			return
		}
	}
	t.Fatal("node not found after re-index")
}

func TestSiblingGraphs(t *testing.T) {
	m, s := newTestManager(t)

	dir := t.TempDir()
	// Create two sibling graphs
	writeFile(t, filepath.Join(dir, "concepts/GRAPH.yaml"), "")
	writeFile(t, filepath.Join(dir, "concepts/AI.md"), "---\nid: ai\n---\n# AI\nLinks to [[Network]]\n")
	writeFile(t, filepath.Join(dir, "concepts/Network.md"), "---\nid: net\n---\n# Network\n")
	writeFile(t, filepath.Join(dir, "projects/GRAPH.yaml"), "")
	writeFile(t, filepath.Join(dir, "projects/Mnemosyne.md"), "---\nid: mn\n---\n# Mnemosyne\nLinks to [[AI]]\n")

	vaultID, graphIDs, err := m.RegisterVault(dir)
	require.NoError(t, err)
	assert.Len(t, graphIDs, 2)

	require.NoError(t, m.FullIndexVault(vaultID))

	// Find which graph is which
	graphs, _ := s.GetGraphsByVault(vaultID)
	var conceptsGID, projectsGID int
	for _, g := range graphs {
		if g.RootPath == "concepts" {
			conceptsGID = g.ID
		} else if g.RootPath == "projects" {
			projectsGID = g.ID
		}
	}

	cGraph, _ := s.GetGraphData(conceptsGID)
	assert.Len(t, cGraph.Nodes, 2, "concepts graph should have AI and Network")
	// ai->net edge should exist within concepts
	assert.Greater(t, len(cGraph.Edges), 0)

	pGraph, _ := s.GetGraphData(projectsGID)
	assert.Len(t, pGraph.Nodes, 1, "projects graph should have Mnemosyne only")
	// mn->ai edge should NOT exist (AI is in a different graph)
	assert.Len(t, pGraph.Edges, 0)
}

func TestIndexFile(t *testing.T) {
	m, s := newTestManager(t)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "GRAPH.yaml"), "")
	writeFile(t, filepath.Join(dir, "a.md"), "---\nid: a\n---\n# A\n")

	vaultID, graphIDs, _ := m.RegisterVault(dir)
	require.NoError(t, m.FullIndexVault(vaultID))

	g, _ := s.GetGraphData(graphIDs[0])
	assert.Len(t, g.Nodes, 1)

	// Add a new file
	writeFile(t, filepath.Join(dir, "b.md"), "---\nid: b\n---\n# B\n")
	affected, err := m.IndexFile(vaultID, "b.md")
	require.NoError(t, err)
	assert.Equal(t, graphIDs, affected)

	g, _ = s.GetGraphData(graphIDs[0])
	assert.Len(t, g.Nodes, 2)
}

func TestRemoveFile(t *testing.T) {
	m, s := newTestManager(t)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "GRAPH.yaml"), "")
	writeFile(t, filepath.Join(dir, "a.md"), "---\nid: a\n---\n# A\n")
	writeFile(t, filepath.Join(dir, "b.md"), "---\nid: b\n---\n# B\n")

	vaultID, graphIDs, _ := m.RegisterVault(dir)
	require.NoError(t, m.FullIndexVault(vaultID))

	g, _ := s.GetGraphData(graphIDs[0])
	assert.Len(t, g.Nodes, 2)

	os.Remove(filepath.Join(dir, "a.md"))
	affected, err := m.RemoveFile(vaultID, "a.md")
	require.NoError(t, err)
	assert.Equal(t, graphIDs, affected)

	g, _ = s.GetGraphData(graphIDs[0])
	assert.Len(t, g.Nodes, 1)
}

func TestTwoVaultsIndependent(t *testing.T) {
	m, s := newTestManager(t)

	dir1 := t.TempDir()
	writeFile(t, filepath.Join(dir1, "GRAPH.yaml"), "")
	writeFile(t, filepath.Join(dir1, "a.md"), "---\nid: v1a\n---\n# V1 A\n")

	dir2 := t.TempDir()
	writeFile(t, filepath.Join(dir2, "GRAPH.yaml"), "")
	writeFile(t, filepath.Join(dir2, "a.md"), "---\nid: v2a\n---\n# V2 A\n")

	v1, g1, _ := m.RegisterVault(dir1)
	v2, g2, _ := m.RegisterVault(dir2)

	require.NoError(t, m.FullIndexVault(v1))
	require.NoError(t, m.FullIndexVault(v2))

	graph1, _ := s.GetGraphData(g1[0])
	graph2, _ := s.GetGraphData(g2[0])
	assert.Len(t, graph1.Nodes, 1)
	assert.Len(t, graph2.Nodes, 1)
	assert.Equal(t, "v1a", graph1.Nodes[0].ID)
	assert.Equal(t, "v2a", graph2.Nodes[0].ID)

	// Re-index v1 should not affect v2
	require.NoError(t, m.FullIndexVault(v1))
	graph2After, _ := s.GetGraphData(g2[0])
	assert.Len(t, graph2After.Nodes, 1)
}

func TestHandleGraphYAMLCreate(t *testing.T) {
	m, s := newTestManager(t)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "concepts/GRAPH.yaml"), "")
	writeFile(t, filepath.Join(dir, "concepts/AI.md"), "---\nid: ai\n---\n# AI\n")
	writeFile(t, filepath.Join(dir, "projects/X.md"), "---\nid: px\n---\n# X\n")

	vaultID, _, _ := m.RegisterVault(dir)
	require.NoError(t, m.FullIndexVault(vaultID))

	// Now create a GRAPH.yaml for projects
	writeFile(t, filepath.Join(dir, "projects/GRAPH.yaml"), "")
	affected, err := m.HandleGraphYAML(vaultID, "projects", true)
	require.NoError(t, err)
	assert.Len(t, affected, 1)

	graphs, _ := s.GetGraphsByVault(vaultID)
	assert.Len(t, graphs, 2)
}

func TestHandleGraphYAMLDelete(t *testing.T) {
	m, s := newTestManager(t)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "concepts/GRAPH.yaml"), "")
	writeFile(t, filepath.Join(dir, "concepts/AI.md"), "---\nid: ai\n---\n# AI\n")

	vaultID, _, _ := m.RegisterVault(dir)
	require.NoError(t, m.FullIndexVault(vaultID))

	graphs, _ := s.GetGraphsByVault(vaultID)
	assert.Len(t, graphs, 1)

	os.Remove(filepath.Join(dir, "concepts/GRAPH.yaml"))
	affected, err := m.HandleGraphYAML(vaultID, "concepts", false)
	require.NoError(t, err)
	assert.Len(t, affected, 1)

	graphs, _ = s.GetGraphsByVault(vaultID)
	assert.Len(t, graphs, 0)
}

// --- helpers ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func copyVault(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	require.NoError(t, err)
}
