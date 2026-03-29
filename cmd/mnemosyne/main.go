// Command mnemosyne starts the Mnemosyne graph visualizer for an Obsidian vault.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ali01/mnemosyne/internal/api"
	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/indexer"
	"github.com/ali01/mnemosyne/internal/store"
	"github.com/ali01/mnemosyne/internal/watcher"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Open database
	s, err := store.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()
	log.Printf("Database: %s", cfg.DBPath)

	// Create indexer
	idx, err := indexer.New(s, cfg.VaultPath, cfg.NodeClassification)
	if err != nil {
		log.Fatalf("Failed to create indexer: %v", err)
	}

	// Full index on startup
	if err := idx.FullIndex(); err != nil {
		log.Fatalf("Failed to index vault: %v", err)
	}

	// Start file watcher
	w, err := watcher.New(idx, cfg.VaultPath)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start watcher: %v", err)
	}
	defer w.Stop()

	// Start HTTP server with embedded frontend
	srv := api.NewServer(s, idx, api.EmbeddedFS(), cfg.Port)

	// Notify frontend when vault files change
	w.SetOnChange(srv.NotifyChange)

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\nShutting down...")
		w.Stop()
		s.Close()
		os.Exit(0)
	}()

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
