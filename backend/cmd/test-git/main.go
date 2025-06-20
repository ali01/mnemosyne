package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ali01/mnemosyne/internal/git"
)

func main() {
	// Load config from YAML file
	configPath := "config.example.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	
	config, err := git.LoadConfigFromYAML(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	fmt.Printf("Git Manager Test\n")
	fmt.Printf("Repository: %s\n", config.RepoURL)
	fmt.Printf("Branch: %s\n", config.Branch)
	fmt.Printf("Local Path: %s\n", config.LocalPath)
	fmt.Println()
	
	// Create manager
	manager, err := git.NewManager(config)
	if err != nil {
		log.Fatalf("Failed to create git manager: %v", err)
	}
	
	// Set update callback
	manager.SetUpdateCallback(func(changedFiles []string) {
		fmt.Printf("Files changed: %d\n", len(changedFiles))
		for i, file := range changedFiles {
			if i < 10 {
				fmt.Printf("  - %s\n", file)
			}
		}
		if len(changedFiles) > 10 {
			fmt.Printf("  ... and %d more\n", len(changedFiles)-10)
		}
	})
	
	// Initialize (clone or open)
	ctx := context.Background()
	fmt.Println("Initializing repository...")
	if err := manager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	
	fmt.Println("Successfully initialized!")
	fmt.Printf("Last sync: %v\n", manager.GetLastSync())
	
	// Don't clean up the real vault clone
	if configPath == "config.example.yaml" {
		defer func() {
			fmt.Println("\nCleaning up test directory...")
			os.RemoveAll(config.LocalPath)
		}()
	}
}