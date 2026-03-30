// Command mnemosyne starts the Mnemosyne graph visualizer for Obsidian vaults.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/ali01/mnemosyne/internal/api"
	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/indexer"
	"github.com/ali01/mnemosyne/internal/store"
	"github.com/ali01/mnemosyne/internal/watcher"
)

func main() {
	portFlag := flag.Int("port", 0, "HTTP port to listen on (overrides config file)")
	flag.IntVar(portFlag, "p", 0, "HTTP port to listen on (shorthand)")
	flag.Parse()

	// Subcommand dispatch
	if flag.NArg() > 0 && flag.Arg(0) == "graphs" {
		cmdGraphs(flag.Args()[1:])
		return
	}

	cfgPath := config.DefaultConfigPath()
	if flag.NArg() > 0 {
		cfgPath = flag.Arg(0)
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

	// Command-line port flag overrides config file
	if *portFlag != 0 {
		cfg.Port = *portFlag
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

	srv := api.NewServer(s, idx, api.EmbeddedFS(), cfg.Port, cfg.HomeGraph)

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

func cmdGraphs(args []string) {
	if len(args) > 0 && args[0] == "delete" {
		cmdGraphsDelete(args[1:])
		return
	}

	dbPath := config.DBPath()
	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()

	graphs, err := s.GetAllGraphsIncludeArchived()
	if err != nil {
		log.Fatalf("Failed to list graphs: %v", err)
	}

	if len(graphs) == 0 {
		fmt.Println("No graphs found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tVault\tName\tRoot Path\tNodes\tStatus")
	fmt.Fprintln(w, "--\t-----\t----\t---------\t-----\t------")
	for _, g := range graphs {
		status := "active"
		if g.Archived {
			status = "archived"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%s\n", g.ID, g.VaultName, g.Name, g.RootPath, g.NodeCount, status)
	}
	w.Flush()
}

func cmdGraphsDelete(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: mnemosyne graphs delete <graph-id>")
		os.Exit(1)
	}

	graphID, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid graph ID: %s\n", args[0])
		os.Exit(1)
	}

	dbPath := config.DBPath()
	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()

	info, err := s.GetGraphInfo(graphID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Graph %d not found.\n", graphID)
		os.Exit(1)
	}

	if err := s.PermanentlyDeleteGraph(graphID); err != nil {
		log.Fatalf("Failed to delete graph: %v", err)
	}

	fmt.Printf("Permanently deleted graph %d (%s/%s).\n", info.ID, info.VaultName, info.Name)
}
