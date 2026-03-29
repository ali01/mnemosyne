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

func setupTestVault(t *testing.T) (string, *store.Store, *indexer.Indexer) {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "note-a.md"), `---
id: "a"
---
# Note A
`)

	s, err := store.NewMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	idx, err := indexer.New(s, dir, nil)
	require.NoError(t, err)

	require.NoError(t, idx.FullIndex())
	return dir, s, idx
}

func TestWatcherDetectsNewFile(t *testing.T) {
	dir, s, idx := setupTestVault(t)

	w, err := New(idx, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	// Create a new file
	writeFile(t, filepath.Join(dir, "note-b.md"), `---
id: "b"
---
# Note B
`)

	// Wait for debounce + processing
	assert.Eventually(t, func() bool {
		nodes, _ := s.GetAllNodes()
		return len(nodes) == 2
	}, 3*time.Second, 100*time.Millisecond, "expected 2 nodes after adding a file")
}

func TestWatcherDetectsFileModification(t *testing.T) {
	dir, s, idx := setupTestVault(t)

	w, err := New(idx, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	// Modify existing file
	writeFile(t, filepath.Join(dir, "note-a.md"), `---
id: "a"
---
# Updated Note A
New content here.
`)

	// Wait for debounce + processing
	assert.Eventually(t, func() bool {
		node, err := s.GetNode("a")
		if err != nil {
			return false
		}
		return node.Content != ""
	}, 3*time.Second, 100*time.Millisecond, "expected node to be updated")
}

func TestWatcherDetectsFileDeletion(t *testing.T) {
	dir, s, idx := setupTestVault(t)

	// Add a second file before starting the watcher
	writeFile(t, filepath.Join(dir, "note-b.md"), `---
id: "b"
---
# Note B
`)
	require.NoError(t, idx.FullIndex())

	nodes, _ := s.GetAllNodes()
	require.Len(t, nodes, 2)

	w, err := New(idx, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	// Delete the file
	require.NoError(t, os.Remove(filepath.Join(dir, "note-b.md")))

	assert.Eventually(t, func() bool {
		nodes, _ := s.GetAllNodes()
		return len(nodes) == 1
	}, 10*time.Second, 200*time.Millisecond, "expected 1 node after deleting a file")
}

func TestWatcherIgnoresNonMarkdown(t *testing.T) {
	dir, s, idx := setupTestVault(t)

	w, err := New(idx, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	// Create a non-markdown file
	writeFile(t, filepath.Join(dir, "image.png"), "not markdown")

	// Wait a bit, then check nothing changed
	time.Sleep(1 * time.Second)
	nodes, _ := s.GetAllNodes()
	assert.Len(t, nodes, 1, "non-markdown files should not trigger indexing")
}

func TestWatcherDebouncesBatchChanges(t *testing.T) {
	dir, s, idx := setupTestVault(t)

	w, err := New(idx, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	// Rapidly create multiple files
	for i := 0; i < 5; i++ {
		writeFile(t, filepath.Join(dir, fmt.Sprintf("rapid-%d.md", i)), fmt.Sprintf(`---
id: "r%d"
---
# Rapid %d
`, i, i))
	}

	assert.Eventually(t, func() bool {
		nodes, _ := s.GetAllNodes()
		return len(nodes) == 6 // 1 original + 5 new
	}, 5*time.Second, 100*time.Millisecond, "expected 6 nodes after batch creation")
}

func TestWatcherStartStop(t *testing.T) {
	dir, _, idx := setupTestVault(t)

	w, err := New(idx, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	w.Stop()
	// Should not panic or hang
}

func TestWatcherNewSubdirectory(t *testing.T) {
	dir, s, idx := setupTestVault(t)

	w, err := New(idx, dir)
	require.NoError(t, err)
	require.NoError(t, w.Start())
	defer w.Stop()

	// Create a subdirectory with a file
	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0o755)

	// Small delay for the dir watcher to register
	time.Sleep(200 * time.Millisecond)

	writeFile(t, filepath.Join(subdir, "sub-note.md"), `---
id: "sub"
---
# Sub Note
`)

	assert.Eventually(t, func() bool {
		nodes, _ := s.GetAllNodes()
		return len(nodes) == 2
	}, 3*time.Second, 100*time.Millisecond, "expected 2 nodes after adding file in subdirectory")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
