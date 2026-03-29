// Command mnemosyne starts the Mnemosyne graph visualizer for Obsidian vaults.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ali01/mnemosyne/internal/api"
	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/indexer"
	"github.com/ali01/mnemosyne/internal/store"
	"github.com/ali01/mnemosyne/internal/watcher"
)

func main() {
	cfgPath := config.DefaultConfigPath()
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	// Bootstrap config if it doesn't exist
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := bootstrapConfig(cfgPath); err != nil {
			log.Fatalf("Failed to create config: %v", err)
		}
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbPath := config.DBPath()
	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()
	log.Printf("Database: %s", dbPath)

	idx := indexer.NewIndexManager(s)

	// Register and index all vaults
	var watchers []*watcher.Watcher
	for _, vaultPath := range cfg.Vaults {
		vaultID, _, err := idx.RegisterVault(vaultPath)
		if err != nil {
			log.Fatalf("Failed to register vault %s: %v", vaultPath, err)
		}

		if err := idx.FullIndexVault(vaultID); err != nil {
			log.Fatalf("Failed to index vault %s: %v", vaultPath, err)
		}

		w, err := watcher.New(idx, vaultID, vaultPath)
		if err != nil {
			log.Fatalf("Failed to create watcher for %s: %v", vaultPath, err)
		}
		watchers = append(watchers, w)
	}

	srv := api.NewServer(s, idx, api.EmbeddedFS(), cfg.Port)

	// Start watchers with SSE notification
	for _, w := range watchers {
		w.SetOnChange(func(graphIDs []int) {
			srv.NotifyChange(graphIDs)
		})
		w.SetOnGraphsChanged(func() {
			srv.NotifyGraphsChanged()
		})
		if err := w.Start(); err != nil {
			log.Fatalf("Failed to start watcher: %v", err)
		}
	}
	defer func() {
		for _, w := range watchers {
			w.Stop()
		}
	}()

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

func bootstrapConfig(cfgPath string) error {
	fmt.Println("Welcome to Mnemosyne!")
	fmt.Println()
	fmt.Print("Enter path to your Obsidian vault: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	vaultPath := strings.TrimSpace(line)
	if vaultPath == "" {
		return fmt.Errorf("vault path cannot be empty")
	}

	vaultPath = config.ExpandHome(vaultPath)

	info, err := os.Stat(vaultPath)
	if err != nil {
		return fmt.Errorf("vault path %q does not exist", vaultPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("vault path %q is not a directory", vaultPath)
	}

	if err := config.CreateDefault(cfgPath, vaultPath); err != nil {
		return err
	}

	fmt.Printf("Config created at %s\n\n", cfgPath)
	return nil
}
