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

// Watcher monitors a vault directory and triggers indexing on file changes.
type Watcher struct {
	indexer   *indexer.Indexer
	vaultPath string
	watcher   *fsnotify.Watcher
	debounce  time.Duration
	done      chan struct{}
	wg        sync.WaitGroup
	onChange  func() // called after changes are processed

	mu      sync.Mutex
	pending map[string]fsnotify.Op
	timer   *time.Timer
}

// New creates a watcher for the given vault path.
func New(idx *indexer.Indexer, vaultPath string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		indexer:   idx,
		vaultPath: vaultPath,
		watcher:   fw,
		debounce:  500 * time.Millisecond,
		done:      make(chan struct{}),
		pending:   make(map[string]fsnotify.Op),
	}, nil
}

// SetOnChange sets a callback that fires after vault changes are indexed.
func (w *Watcher) SetOnChange(fn func()) {
	w.onChange = fn
}

// Start begins watching the vault directory recursively.
func (w *Watcher) Start() error {
	if err := w.addRecursive(w.vaultPath); err != nil {
		return err
	}

	w.wg.Add(1)
	go w.loop()

	log.Printf("Watching vault at %s", w.vaultPath)
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

func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Only care about markdown files
	if !strings.HasSuffix(event.Name, ".md") {
		// But watch new directories
		if event.Has(fsnotify.Create) {
			if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
				w.watcher.Add(event.Name)
			}
		}
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Map the event to a simplified operation
	switch {
	case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
		w.pending[event.Name] = fsnotify.Remove
	case event.Has(fsnotify.Create) || event.Has(fsnotify.Write):
		w.pending[event.Name] = fsnotify.Create
	}

	// Reset debounce timer
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

	for path, op := range pending {
		relPath, err := filepath.Rel(w.vaultPath, path)
		if err != nil {
			log.Printf("Failed to get relative path for %s: %v", path, err)
			continue
		}

		switch op {
		case fsnotify.Remove:
			if err := w.indexer.RemoveFile(relPath); err != nil {
				log.Printf("Failed to remove %s: %v", relPath, err)
			}
		case fsnotify.Create:
			if err := w.indexer.IndexFile(relPath); err != nil {
				log.Printf("Failed to index %s: %v", relPath, err)
			}
		}
	}

	if len(pending) > 0 && w.onChange != nil {
		w.onChange()
	}
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return w.watcher.Add(path)
		}
		return nil
	})
}
