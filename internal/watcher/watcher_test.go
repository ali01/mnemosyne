package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ali01/mnemosyne/internal/indexer"
	"github.com/ali01/mnemosyne/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestVault(t *testing.T) (string, *store.Store, *indexer.IndexManager, int, []int) {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "GRAPH.yaml"), "")
	writeFile(t, filepath.Join(dir, "note-a.md"), `---
id: "a"
---
# Note A
`)

	s, err := store.NewMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	m := indexer.NewIndexManager(s)
	vaultID, graphIDs, err := m.RegisterVault(dir)
	require.NoError(t, err)
	require.NoError(t, m.FullIndexVault(vaultID))

	return dir, s, m, vaultID, graphIDs
}

func TestWatcherDetectsNewFile(t *testing.T) {
	dir, s, m, vaultID, graphIDs := setupTestVault(t)

	w, err := New(m, vaultID, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	writeFile(t, filepath.Join(dir, "note-b.md"), `---
id: "b"
---
# Note B
`)

	assert.Eventually(t, func() bool {
		g, _ := s.GetGraphData(graphIDs[0])
		return len(g.Nodes) == 2
	}, 3*time.Second, 100*time.Millisecond, "expected 2 nodes after adding a file")
}

func TestWatcherDetectsFileModification(t *testing.T) {
	dir, s, m, vaultID, _ := setupTestVault(t)

	w, err := New(m, vaultID, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	writeFile(t, filepath.Join(dir, "note-a.md"), `---
id: "a"
---
# Updated Note A
New content here.
`)

	assert.Eventually(t, func() bool {
		node, err := s.GetNode("a")
		if err != nil {
			return false
		}
		return node.Content != ""
	}, 3*time.Second, 100*time.Millisecond, "expected node to be updated")
}

func TestWatcherDetectsFileDeletion(t *testing.T) {
	dir, s, m, vaultID, graphIDs := setupTestVault(t)

	// Add a second file before starting the watcher
	writeFile(t, filepath.Join(dir, "note-b.md"), `---
id: "b"
---
# Note B
`)
	require.NoError(t, m.FullIndexVault(vaultID))

	g, _ := s.GetGraphData(graphIDs[0])
	require.Len(t, g.Nodes, 2)

	w, err := New(m, vaultID, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	require.NoError(t, os.Remove(filepath.Join(dir, "note-b.md")))

	assert.Eventually(t, func() bool {
		g, _ := s.GetGraphData(graphIDs[0])
		return len(g.Nodes) == 1
	}, 10*time.Second, 200*time.Millisecond, "expected 1 node after deleting a file")
}

func TestWatcherIgnoresNonMarkdown(t *testing.T) {
	dir, s, m, vaultID, graphIDs := setupTestVault(t)

	w, err := New(m, vaultID, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	writeFile(t, filepath.Join(dir, "image.png"), "not markdown")

	time.Sleep(1 * time.Second)
	g, _ := s.GetGraphData(graphIDs[0])
	assert.Len(t, g.Nodes, 1, "non-markdown files should not trigger indexing")
}

func TestWatcherDebouncesBatchChanges(t *testing.T) {
	dir, s, m, vaultID, graphIDs := setupTestVault(t)

	w, err := New(m, vaultID, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	for i := 0; i < 5; i++ {
		writeFile(t, filepath.Join(dir, fmt.Sprintf("rapid-%d.md", i)), fmt.Sprintf(`---
id: "r%d"
---
# Rapid %d
`, i, i))
	}

	assert.Eventually(t, func() bool {
		g, _ := s.GetGraphData(graphIDs[0])
		return len(g.Nodes) == 6
	}, 5*time.Second, 100*time.Millisecond, "expected 6 nodes after batch creation")
}

func TestWatcherStartStop(t *testing.T) {
	dir, _, m, vaultID, _ := setupTestVault(t)

	w, err := New(m, vaultID, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	w.Stop()
}

func TestWatcherNewSubdirectory(t *testing.T) {
	dir, s, m, vaultID, graphIDs := setupTestVault(t)

	w, err := New(m, vaultID, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0o755)
	time.Sleep(200 * time.Millisecond)

	writeFile(t, filepath.Join(subdir, "sub-note.md"), `---
id: "sub"
---
# Sub Note
`)

	assert.Eventually(t, func() bool {
		g, _ := s.GetGraphData(graphIDs[0])
		return len(g.Nodes) == 2
	}, 3*time.Second, 100*time.Millisecond, "expected 2 nodes after adding file in subdirectory")
}

func TestWatcherOnChangeReceivesGraphIDs(t *testing.T) {
	dir, _, m, vaultID, graphIDs := setupTestVault(t)

	w, err := New(m, vaultID, dir)
	require.NoError(t, err)

	var receivedIDs []int
	w.SetOnChange(func(ids []int) {
		receivedIDs = ids
	})

	require.NoError(t, w.Start())
	defer w.Stop()

	writeFile(t, filepath.Join(dir, "note-c.md"), `---
id: "c"
---
# Note C
`)

	assert.Eventually(t, func() bool {
		return len(receivedIDs) > 0
	}, 3*time.Second, 100*time.Millisecond, "expected onChange to be called with graph IDs")

	assert.Contains(t, receivedIDs, graphIDs[0])
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
