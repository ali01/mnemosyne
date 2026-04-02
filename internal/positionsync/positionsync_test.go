package positionsync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func setupGraph(t *testing.T, s *store.Store, rootPath string) (vaultPath string, graphID int) {
	t.Helper()
	vaultPath = t.TempDir()
	vaultID, err := s.UpsertVault("test-vault", vaultPath)
	require.NoError(t, err)
	graphID, err = s.UpsertGraph(vaultID, "test-graph", rootPath, "")
	require.NoError(t, err)
	return vaultPath, graphID
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "_root_"},
		{"memex", "memex"},
		{"projects/alpha", "projects--alpha"},
		{"a/b/c", "a--b--c"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, sanitizePath(tt.input), "sanitizePath(%q)", tt.input)
	}
}

func TestFilePath(t *testing.T) {
	got := FilePath("/vault", "memex")
	want := filepath.Join("/vault", ".mnemosyne", "positions-memex.json")
	assert.Equal(t, want, got)

	got = FilePath("/vault", "")
	want = filepath.Join("/vault", ".mnemosyne", "positions-_root_.json")
	assert.Equal(t, want, got)

	got = FilePath("/vault", "projects/alpha")
	want = filepath.Join("/vault", ".mnemosyne", "positions-projects--alpha.json")
	assert.Equal(t, want, got)
}

func TestExportAndImport(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// Insert positions into DB
	positions := []models.NodePosition{
		{NodeID: "node-a", X: 10.5, Y: -20.3},
		{NodeID: "node-b", X: 42.0, Y: 100.0},
	}
	require.NoError(t, s.UpsertPositions(graphID, positions))

	// Export via syncer
	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	syncer.MarkDirty(graphID)
	syncer.Shutdown() // flushes immediately

	// Verify JSON file
	path := FilePath(vaultPath, "memex")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var pf positionFile
	require.NoError(t, json.Unmarshal(data, &pf))
	assert.Equal(t, 1, pf.MnemosyneVersion)
	assert.Equal(t, "memex", pf.RootPath)
	assert.Len(t, pf.Positions, 2)
	assert.Equal(t, 10.5, pf.Positions["node-a"].X)
	assert.Equal(t, -20.3, pf.Positions["node-a"].Y)
	assert.Equal(t, 42.0, pf.Positions["node-b"].X)

	// Now create a fresh DB and import
	s2 := newTestStore(t)
	vaultID2, err := s2.UpsertVault("test-vault", vaultPath)
	require.NoError(t, err)
	graphID2, err := s2.UpsertGraph(vaultID2, "test-graph", "memex", "")
	require.NoError(t, err)

	syncer2 := New(s2)
	syncer2.Register(graphID2, "memex", vaultPath)
	require.NoError(t, syncer2.ImportIfEmpty(graphID2))

	// Verify imported positions
	imported, err := s2.GetPositionsByGraph(graphID2)
	require.NoError(t, err)
	assert.Len(t, imported, 2)
	assert.Equal(t, 10.5, imported["node-a"].X)
	assert.Equal(t, -20.3, imported["node-a"].Y)
}

func TestImportSkipsWhenDBHasPositions(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// Write a JSON file with different positions
	dir := filepath.Join(vaultPath, dirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	pf := positionFile{
		MnemosyneVersion: 1,
		RootPath:         "memex",
		ExportedAt:       time.Now().UTC(),
		Positions: map[string]posXY{
			"node-a": {X: 999, Y: 999},
		},
	}
	data, _ := json.Marshal(pf)
	require.NoError(t, os.WriteFile(FilePath(vaultPath, "memex"), data, 0o644))

	// DB already has positions
	require.NoError(t, s.UpsertPositions(graphID, []models.NodePosition{
		{NodeID: "node-a", X: 1, Y: 2},
	}))

	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	require.NoError(t, syncer.ImportIfEmpty(graphID))

	// Should keep DB values, not JSON values
	positions, err := s.GetPositionsByGraph(graphID)
	require.NoError(t, err)
	assert.Equal(t, 1.0, positions["node-a"].X)
	assert.Equal(t, 2.0, positions["node-a"].Y)
}

func TestImportSkipsCorruptJSON(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// Write corrupt JSON
	dir := filepath.Join(vaultPath, dirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(FilePath(vaultPath, "memex"), []byte("{invalid json"), 0o644))

	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	require.NoError(t, syncer.ImportIfEmpty(graphID)) // should not error

	// No positions imported
	count, err := s.GetPositionCount(graphID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestImportSkipsRootPathMismatch(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// Write JSON with wrong root_path
	dir := filepath.Join(vaultPath, dirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	pf := positionFile{
		MnemosyneVersion: 1,
		RootPath:         "wrong-path",
		ExportedAt:       time.Now().UTC(),
		Positions:        map[string]posXY{"node-a": {X: 1, Y: 2}},
	}
	data, _ := json.Marshal(pf)
	require.NoError(t, os.WriteFile(FilePath(vaultPath, "memex"), data, 0o644))

	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	require.NoError(t, syncer.ImportIfEmpty(graphID))

	count, err := s.GetPositionCount(graphID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestImportSkipsMissingFile(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	require.NoError(t, syncer.ImportIfEmpty(graphID)) // no file, no error
}

func TestDeleteFile(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// Export some positions
	require.NoError(t, s.UpsertPositions(graphID, []models.NodePosition{
		{NodeID: "node-a", X: 1, Y: 2},
	}))
	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	syncer.MarkDirty(graphID)
	syncer.Shutdown()

	path := FilePath(vaultPath, "memex")
	_, err := os.Stat(path)
	require.NoError(t, err, "file should exist after export")

	syncer.DeleteFile(graphID, "memex", vaultPath)
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "file should be removed")
}

func TestDeleteFileNoOp(t *testing.T) {
	s := newTestStore(t)
	syncer := New(s)
	// Should not panic or error when file doesn't exist
	syncer.DeleteFile(999, "memex", t.TempDir())
}

func TestRootGraphFileName(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "")

	require.NoError(t, s.UpsertPositions(graphID, []models.NodePosition{
		{NodeID: "node-a", X: 5, Y: 10},
	}))

	syncer := New(s)
	syncer.Register(graphID, "", vaultPath)
	syncer.MarkDirty(graphID)
	syncer.Shutdown()

	path := FilePath(vaultPath, "")
	assert.Contains(t, path, "positions-_root_.json")
	_, err := os.Stat(path)
	require.NoError(t, err)
}

func TestExportSkipsEmptyPositions(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// No positions in DB
	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	syncer.MarkDirty(graphID)
	syncer.Shutdown()

	// No file should be created
	path := FilePath(vaultPath, "memex")
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "no file for empty positions")
}

func TestExportIfMissing(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// DB has positions but no JSON file
	require.NoError(t, s.UpsertPositions(graphID, []models.NodePosition{
		{NodeID: "node-a", X: 10, Y: 20},
		{NodeID: "node-b", X: 30, Y: 40},
	}))

	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	require.NoError(t, syncer.ExportIfMissing(graphID))

	// JSON file should now exist
	path := FilePath(vaultPath, "memex")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var pf positionFile
	require.NoError(t, json.Unmarshal(data, &pf))
	assert.Len(t, pf.Positions, 2)
	assert.Equal(t, 10.0, pf.Positions["node-a"].X)
}

func TestExportIfMissingSkipsWhenFileExists(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// Create JSON file first
	dir := filepath.Join(vaultPath, dirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	pf := positionFile{
		MnemosyneVersion: 1,
		RootPath:         "memex",
		ExportedAt:       time.Now().UTC(),
		Positions:        map[string]posXY{"node-a": {X: 999, Y: 999}},
	}
	data, _ := json.Marshal(pf)
	require.NoError(t, os.WriteFile(FilePath(vaultPath, "memex"), data, 0o644))

	// DB has different positions
	require.NoError(t, s.UpsertPositions(graphID, []models.NodePosition{
		{NodeID: "node-a", X: 1, Y: 2},
	}))

	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	require.NoError(t, syncer.ExportIfMissing(graphID))

	// Should NOT overwrite existing file
	data2, err := os.ReadFile(FilePath(vaultPath, "memex"))
	require.NoError(t, err)
	var pf2 positionFile
	require.NoError(t, json.Unmarshal(data2, &pf2))
	assert.Equal(t, 999.0, pf2.Positions["node-a"].X)
}

func TestExportIfMissingSkipsEmptyDB(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// No positions in DB, no JSON file
	syncer := New(s)
	syncer.Register(graphID, "memex", vaultPath)
	require.NoError(t, syncer.ExportIfMissing(graphID))

	// No file should be created
	_, err := os.Stat(FilePath(vaultPath, "memex"))
	assert.True(t, os.IsNotExist(err))
}

func TestSyncImportsThenExports(t *testing.T) {
	s := newTestStore(t)
	vaultPath := t.TempDir()
	vaultID, err := s.UpsertVault("vault", vaultPath)
	require.NoError(t, err)

	// Two graphs: one has a JSON file, one has DB positions
	gidA, err := s.UpsertGraph(vaultID, "graphA", "alpha", "")
	require.NoError(t, err)
	gidB, err := s.UpsertGraph(vaultID, "graphB", "beta", "")
	require.NoError(t, err)

	// graphA: JSON file exists, DB empty → should import
	dir := filepath.Join(vaultPath, dirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	pf := positionFile{
		MnemosyneVersion: 1,
		RootPath:         "alpha",
		ExportedAt:       time.Now().UTC(),
		Positions:        map[string]posXY{"node-a": {X: 5, Y: 10}},
	}
	data, _ := json.Marshal(pf)
	require.NoError(t, os.WriteFile(FilePath(vaultPath, "alpha"), data, 0o644))

	// graphB: DB has positions, no JSON file → should export
	require.NoError(t, s.UpsertPositions(gidB, []models.NodePosition{
		{NodeID: "node-b", X: 20, Y: 30},
	}))

	syncer := New(s)
	syncer.Register(gidA, "alpha", vaultPath)
	syncer.Register(gidB, "beta", vaultPath)

	require.NoError(t, syncer.Sync(gidA))
	require.NoError(t, syncer.Sync(gidB))

	// graphA: positions should be in DB (imported)
	posA, err := s.GetPositionsByGraph(gidA)
	require.NoError(t, err)
	assert.Len(t, posA, 1)
	assert.Equal(t, 5.0, posA["node-a"].X)

	// graphB: JSON file should exist (exported)
	dataB, err := os.ReadFile(FilePath(vaultPath, "beta"))
	require.NoError(t, err)
	var pfB positionFile
	require.NoError(t, json.Unmarshal(dataB, &pfB))
	assert.Equal(t, 20.0, pfB.Positions["node-b"].X)
}

func TestMarkDirtyAutoRegisters(t *testing.T) {
	s := newTestStore(t)
	vaultPath, graphID := setupGraph(t, s, "memex")

	// Insert positions into DB
	require.NoError(t, s.UpsertPositions(graphID, []models.NodePosition{
		{NodeID: "node-a", X: 10, Y: 20},
	}))

	// Create syncer WITHOUT registering the graph
	syncer := New(s)
	syncer.MarkDirty(graphID)
	syncer.Shutdown()

	// Should have auto-registered and exported
	path := FilePath(vaultPath, "memex")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var pf positionFile
	require.NoError(t, json.Unmarshal(data, &pf))
	assert.Equal(t, 10.0, pf.Positions["node-a"].X)
}

func TestMultipleGraphsExport(t *testing.T) {
	s := newTestStore(t)
	vaultPath := t.TempDir()
	vaultID, err := s.UpsertVault("vault", vaultPath)
	require.NoError(t, err)

	gid1, err := s.UpsertGraph(vaultID, "graph1", "memex", "")
	require.NoError(t, err)
	gid2, err := s.UpsertGraph(vaultID, "graph2", "invest", "")
	require.NoError(t, err)

	require.NoError(t, s.UpsertPositions(gid1, []models.NodePosition{{NodeID: "a", X: 1, Y: 2}}))
	require.NoError(t, s.UpsertPositions(gid2, []models.NodePosition{{NodeID: "b", X: 3, Y: 4}}))

	syncer := New(s)
	syncer.Register(gid1, "memex", vaultPath)
	syncer.Register(gid2, "invest", vaultPath)
	syncer.MarkDirty(gid1)
	syncer.MarkDirty(gid2)
	syncer.Shutdown()

	// Both files exist
	_, err = os.Stat(FilePath(vaultPath, "memex"))
	require.NoError(t, err)
	_, err = os.Stat(FilePath(vaultPath, "invest"))
	require.NoError(t, err)
}
