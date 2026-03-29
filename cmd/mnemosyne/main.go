// Command mnemosyne starts the Mnemosyne graph visualizer for an Obsidian vault.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	s, err := store.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()
	log.Printf("Database: %s", cfg.DBPath)

	idx, err := indexer.New(s, cfg.VaultPath, cfg.NodeClassification)
	if err != nil {
		log.Fatalf("Failed to create indexer: %v", err)
	}

	if err := idx.FullIndex(); err != nil {
		log.Fatalf("Failed to index vault: %v", err)
	}

	w, err := watcher.New(idx, cfg.VaultPath)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start watcher: %v", err)
	}
	defer w.Stop()

	srv := api.NewServer(s, idx, api.EmbeddedFS(), cfg.Port)
	w.SetOnChange(srv.NotifyChange)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: srv.Handler(),
	}

	// Graceful shutdown on signal
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\nShutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
	}()

	log.Printf("Listening on http://localhost:%d", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
