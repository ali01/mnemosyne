// Package positionsync exports node positions from SQLite to JSON files
// inside each vault's .mnemosyne/ directory, and imports them back on startup
// when the DB has no positions for a graph (e.g. after DB rebuild).
package positionsync

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/store"
)

const (
	dirName       = ".mnemosyne"
	filePrefix    = "positions-"
	fileSuffix    = ".json"
	rootSentinel  = "_root_"
	formatVersion = 1
	debounceDur   = 5 * time.Second
)

// positionFile is the JSON structure written to disk.
type positionFile struct {
	MnemosyneVersion int                `json:"mnemosyne_version"`
	RootPath         string             `json:"root_path"`
	ExportedAt       time.Time          `json:"exported_at"`
	Positions        map[string]posXY   `json:"positions"`
}

type posXY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// graphRef identifies a graph for export/import.
type graphRef struct {
	graphID   int
	rootPath  string
	vaultPath string
}

// Syncer manages debounced position export and startup import.
type Syncer struct {
	store *store.Store

	mu     sync.Mutex
	graphs map[int]graphRef // graphID → ref
	dirty  map[int]bool     // graphIDs with pending export
	timer  *time.Timer
}

// New creates a new position syncer.
func New(s *store.Store) *Syncer {
	return &Syncer{
		store:  s,
		graphs: make(map[int]graphRef),
		dirty:  make(map[int]bool),
	}
}

// Register adds a graph so the syncer knows its vault path and root path.
func (s *Syncer) Register(graphID int, rootPath, vaultPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.graphs[graphID] = graphRef{
		graphID:   graphID,
		rootPath:  rootPath,
		vaultPath: vaultPath,
	}
}

// ImportIfEmpty imports positions from JSON for a graph if the DB has none.
func (s *Syncer) ImportIfEmpty(graphID int) error {
	s.mu.Lock()
	ref, ok := s.graphs[graphID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("graph %d not registered", graphID)
	}

	count, err := s.store.GetPositionCount(graphID)
	if err != nil {
		return fmt.Errorf("check position count: %w", err)
	}
	if count > 0 {
		return nil // DB already has positions
	}

	path := filePath(ref.vaultPath, ref.rootPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no file to import
		}
		return fmt.Errorf("read positions file: %w", err)
	}

	var pf positionFile
	if err := json.Unmarshal(data, &pf); err != nil {
		log.Printf("positionsync: corrupt JSON file %s, skipping import: %v", path, err)
		return nil
	}

	if pf.RootPath != ref.rootPath {
		log.Printf("positionsync: root_path mismatch in %s (file=%q, expected=%q), skipping import", path, pf.RootPath, ref.rootPath)
		return nil
	}

	if len(pf.Positions) == 0 {
		return nil
	}

	positions := make([]models.NodePosition, 0, len(pf.Positions))
	for nodeID, pos := range pf.Positions {
		positions = append(positions, models.NodePosition{
			NodeID: nodeID,
			X:      pos.X,
			Y:      pos.Y,
		})
	}

	if err := s.store.UpsertPositions(graphID, positions); err != nil {
		return fmt.Errorf("import positions: %w", err)
	}

	log.Printf("positionsync: imported %d positions for graph %d from %s", len(positions), graphID, path)
	return nil
}

// ExportIfMissing exports positions from DB to JSON if the file doesn't exist yet.
// Called on startup to back up pre-existing positions that were never exported.
func (s *Syncer) ExportIfMissing(graphID int) error {
	s.mu.Lock()
	ref, ok := s.graphs[graphID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("graph %d not registered", graphID)
	}

	path := filePath(ref.vaultPath, ref.rootPath)
	if _, err := os.Stat(path); err == nil {
		return nil // file already exists
	}

	count, err := s.store.GetPositionCount(graphID)
	if err != nil {
		return fmt.Errorf("check position count: %w", err)
	}
	if count == 0 {
		return nil // nothing to export
	}

	if err := s.exportGraph(graphID, ref); err != nil {
		return fmt.Errorf("initial export: %w", err)
	}

	log.Printf("positionsync: exported %d positions for graph %d to %s", count, graphID, path)
	return nil
}

// Sync ensures positions are consistent between DB and JSON on startup.
// Imports from JSON if DB is empty, exports to JSON if file is missing.
func (s *Syncer) Sync(graphID int) error {
	if err := s.ImportIfEmpty(graphID); err != nil {
		return err
	}
	return s.ExportIfMissing(graphID)
}

// MarkDirty marks a graph as having unsaved position changes.
// A debounced timer will flush dirty graphs to disk.
// Automatically registers unknown graphs by querying the store.
func (s *Syncer) MarkDirty(graphID int) {
	s.mu.Lock()
	_, ok := s.graphs[graphID]
	s.mu.Unlock()

	if !ok {
		ref, err := s.resolveGraph(graphID)
		if err != nil {
			log.Printf("positionsync: cannot resolve graph %d for export: %v", graphID, err)
			return
		}
		s.mu.Lock()
		s.graphs[graphID] = ref
		s.mu.Unlock()
		log.Printf("positionsync: auto-registered graph %d (%s)", graphID, ref.rootPath)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.dirty[graphID] = true

	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(debounceDur, func() {
		s.flush()
	})
}

// resolveGraph looks up a graph's root path and vault path from the store.
func (s *Syncer) resolveGraph(graphID int) (graphRef, error) {
	info, err := s.store.GetGraphInfo(graphID)
	if err != nil {
		return graphRef{}, fmt.Errorf("get graph info: %w", err)
	}
	vaults, err := s.store.GetVaults()
	if err != nil {
		return graphRef{}, fmt.Errorf("get vaults: %w", err)
	}
	for _, v := range vaults {
		if v.ID == info.VaultID {
			return graphRef{
				graphID:   graphID,
				rootPath:  info.RootPath,
				vaultPath: v.Path,
			}, nil
		}
	}
	return graphRef{}, fmt.Errorf("vault %d not found for graph %d", info.VaultID, graphID)
}

// Shutdown flushes all dirty positions and stops the syncer.
func (s *Syncer) Shutdown() {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.mu.Unlock()

	s.flush()
}

// DeleteFile removes the positions JSON file for a graph.
func (s *Syncer) DeleteFile(graphID int, rootPath, vaultPath string) {
	path := filePath(vaultPath, rootPath)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("positionsync: failed to remove %s: %v", path, err)
	}
}

// flush exports all dirty graphs to JSON files.
func (s *Syncer) flush() {
	s.mu.Lock()
	toFlush := make(map[int]graphRef)
	for id := range s.dirty {
		if ref, ok := s.graphs[id]; ok {
			toFlush[id] = ref
		}
	}
	s.dirty = make(map[int]bool)
	s.mu.Unlock()

	for id, ref := range toFlush {
		if err := s.exportGraph(id, ref); err != nil {
			log.Printf("positionsync: export graph %d failed: %v", id, err)
		}
	}
}

// exportGraph writes a single graph's positions to its JSON file.
func (s *Syncer) exportGraph(graphID int, ref graphRef) error {
	positions, err := s.store.GetPositionsByGraph(graphID)
	if err != nil {
		return fmt.Errorf("get positions: %w", err)
	}

	if len(positions) == 0 {
		return nil // nothing to export
	}

	posMap := make(map[string]posXY, len(positions))
	for nodeID, p := range positions {
		posMap[nodeID] = posXY{X: p.X, Y: p.Y}
	}

	pf := positionFile{
		MnemosyneVersion: formatVersion,
		RootPath:         ref.rootPath,
		ExportedAt:       time.Now().UTC(),
		Positions:        posMap,
	}

	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal positions: %w", err)
	}

	dir := filepath.Join(ref.vaultPath, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create .mnemosyne dir: %w", err)
	}

	path := filePath(ref.vaultPath, ref.rootPath)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write positions file: %w", err)
	}

	return nil
}

// filePath returns the path to the positions JSON file for a graph.
func filePath(vaultPath, rootPath string) string {
	return filepath.Join(vaultPath, dirName, filePrefix+sanitizePath(rootPath)+fileSuffix)
}

// sanitizePath converts a root path to a safe filename component.
// Empty root path (vault-root graph) becomes "_root_".
// Slashes become "--".
func sanitizePath(rootPath string) string {
	if rootPath == "" {
		return rootSentinel
	}
	return strings.ReplaceAll(rootPath, "/", "--")
}

// FilePath is exported for use by cmdGraphsDelete.
func FilePath(vaultPath, rootPath string) string {
	return filePath(vaultPath, rootPath)
}
