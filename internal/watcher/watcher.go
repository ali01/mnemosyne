// Package watcher monitors an Obsidian vault for file changes and triggers re-indexing.
package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/ali01/mnemosyne/internal/indexer"
)

// Watcher monitors a single vault directory and triggers indexing on file changes.
type Watcher struct {
	indexer   *indexer.IndexManager
	vaultID   int
	vaultPath string
	watcher   *fsnotify.Watcher
	debounce  time.Duration
	done      chan struct{}
	wg        sync.WaitGroup
	onChange  func(graphIDs []int)  // called after changes are processed
	onGraphsChanged func()         // called when GRAPH.yaml added/removed

	mu      sync.Mutex
	pending map[string]fsnotify.Op
	timer   *time.Timer
}

// New creates a watcher for a single vault.
func New(idx *indexer.IndexManager, vaultID int, vaultPath string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		indexer:   idx,
		vaultID:   vaultID,
		vaultPath: vaultPath,
		watcher:   fw,
		debounce:  500 * time.Millisecond,
		done:      make(chan struct{}),
		pending:   make(map[string]fsnotify.Op),
	}, nil
}

// SetOnChange sets a callback that fires after vault changes are indexed.
func (w *Watcher) SetOnChange(fn func(graphIDs []int)) {
	w.onChange = fn
}

// SetOnGraphsChanged sets a callback that fires when the graph list changes.
func (w *Watcher) SetOnGraphsChanged(fn func()) {
	w.onGraphsChanged = fn
}

// Start begins watching the vault directory recursively.
func (w *Watcher) Start() error {
	if err := w.addRecursive(w.vaultPath); err != nil {
		return err
	}

	w.wg.Add(1)
	go w.loop()

	log.Printf("Watching vault at %s (vault %d)", w.vaultPath, w.vaultID)
	return nil
}

// Stop stops the watcher and waits for it to finish.
func (w *Watcher) Stop() {
	close(w.done)
	w.watcher.Close()
	w.wg.Wait()
}

func (w *Watcher) loop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (w *Watcher) isWatchable(name string) bool {
	return strings.HasSuffix(name, ".md") || filepath.Base(name) == "GRAPH.yaml"
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	if !w.isWatchable(event.Name) {
		// Watch new directories
		if event.Has(fsnotify.Create) {
			if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
				w.watcher.Add(event.Name)
			}
		}
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	switch {
	case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
		w.pending[event.Name] = fsnotify.Remove
	case event.Has(fsnotify.Create) || event.Has(fsnotify.Write):
		w.pending[event.Name] = fsnotify.Create
	}

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.debounce, w.flush)
}

func (w *Watcher) flush() {
	w.mu.Lock()
	pending := w.pending
	w.pending = make(map[string]fsnotify.Op)
	w.mu.Unlock()

	var allAffected []int
	graphsChanged := false

	// Process GRAPH.yaml changes first (affects membership of .md files)
	for path, op := range pending {
		if filepath.Base(path) != "GRAPH.yaml" {
			continue
		}
		relDir, err := filepath.Rel(w.vaultPath, filepath.Dir(path))
		if err != nil {
			continue
		}
		if relDir == "." {
			relDir = ""
		}

		created := op == fsnotify.Create
		affected, err := w.indexer.HandleGraphYAML(w.vaultID, relDir, created)
		if err != nil {
			log.Printf("Failed to handle GRAPH.yaml at %s: %v", relDir, err)
			continue
		}
		allAffected = append(allAffected, affected...)
		graphsChanged = true
	}

	// Process .md file changes
	for path, op := range pending {
		if !strings.HasSuffix(path, ".md") {
			continue
		}
		relPath, err := filepath.Rel(w.vaultPath, path)
		if err != nil {
			log.Printf("Failed to get relative path for %s: %v", path, err)
			continue
		}

		switch op {
		case fsnotify.Remove:
			affected, err := w.indexer.RemoveFile(w.vaultID, relPath)
			if err != nil {
				log.Printf("Failed to remove %s: %v", relPath, err)
			}
			allAffected = append(allAffected, affected...)
		case fsnotify.Create:
			affected, err := w.indexer.IndexFile(w.vaultID, relPath)
			if err != nil {
				log.Printf("Failed to index %s: %v", relPath, err)
			}
			allAffected = append(allAffected, affected...)
		}
	}

	if graphsChanged && w.onGraphsChanged != nil {
		w.onGraphsChanged()
	}

	if len(allAffected) > 0 && w.onChange != nil {
		w.onChange(dedupe(allAffected))
	}
}

func dedupe(ids []int) []int {
	seen := make(map[int]bool, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return w.watcher.Add(path)
		}
		return nil
	})
}
