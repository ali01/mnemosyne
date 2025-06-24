// Package main provides a Git repository sync utility for Mnemosyne
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/git"
)

const (
	// ANSI color codes
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

func main() {
	// Header
	fmt.Printf("%s%sMnemosyne Vault Sync Utility%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s%s\n\n", colorGray, strings.Repeat("─", 40), colorReset)

	// Load config from YAML file
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// Load configuration
	fmt.Printf("%s→%s Loading configuration from %s%s%s...\n", colorBlue, colorReset, colorYellow, configPath, colorReset)
	fullConfig, err := config.LoadFromYAML(configPath)
	if err != nil {
		fmt.Printf("%s✗ Failed to load config:%s %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}
	fmt.Printf("%s✓%s Configuration loaded successfully\n\n", colorGreen, colorReset)

	// Get git configuration
	gitConfig := &fullConfig.Git

	// Display repository information
	fmt.Printf("%sRepository Details:%s\n", colorBold, colorReset)
	fmt.Printf("  %sURL:%s      %s\n", colorGray, colorReset, gitConfig.RepoURL)
	fmt.Printf("  %sBranch:%s   %s\n", colorGray, colorReset, gitConfig.Branch)
	fmt.Printf("  %sPath:%s     %s\n", colorGray, colorReset, gitConfig.LocalPath)
	fmt.Printf("  %sShallow:%s  %v\n", colorGray, colorReset, gitConfig.ShallowClone)
	fmt.Printf("\n")

	// Check if directory already exists
	repoExists := false
	if _, err := os.Stat(gitConfig.LocalPath); err == nil {
		repoExists = true
		fmt.Printf("%s📂 Repository exists:%s %s\n", colorCyan, colorReset, gitConfig.LocalPath)
		fmt.Printf("  %s→%s Will pull latest changes from remote\n\n", colorBlue, colorReset)
	} else {
		fmt.Printf("%s🆕 Repository not found locally%s\n", colorYellow, colorReset)
		fmt.Printf("  %s→%s Will clone from: %s\n\n", colorBlue, colorReset, gitConfig.RepoURL)
	}

	// Create manager
	fmt.Printf("%s→%s Creating git manager...\n", colorBlue, colorReset)
	manager, err := git.NewManager(gitConfig)
	if err != nil {
		fmt.Printf("%s✗ Failed to create git manager:%s %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	// Set update callback for sync operations
	manager.SetUpdateCallback(func(changedFiles []string) {
		fmt.Printf("\n%s↻ Files updated:%s %d\n", colorCyan, colorReset, len(changedFiles))
		for i, file := range changedFiles {
			if i < 10 {
				fmt.Printf("  %s•%s %s\n", colorGray, colorReset, file)
			}
		}
		if len(changedFiles) > 10 {
			fmt.Printf("  %s... and %d more%s\n", colorGray, len(changedFiles)-10, colorReset)
		}
	})

	// Initialize (clone or open)
	ctx := context.Background()
	startTime := time.Now()
	
	if repoExists {
		fmt.Printf("%s→%s Syncing repository (pulling latest changes)...\n", colorBlue, colorReset)
	} else {
		fmt.Printf("%s→%s Cloning repository...\n", colorBlue, colorReset)
	}
	
	if err := manager.Initialize(ctx); err != nil {
		fmt.Printf("%s✗ Failed to sync:%s %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	
	if repoExists {
		fmt.Printf("%s✓%s Repository synced successfully in %s%.2fs%s\n", 
			colorGreen, colorReset, colorYellow, duration.Seconds(), colorReset)
	} else {
		fmt.Printf("%s✓%s Repository cloned successfully in %s%.2fs%s\n", 
			colorGreen, colorReset, colorYellow, duration.Seconds(), colorReset)
	}

	// Display final status
	fmt.Printf("\n%s%s%s\n", colorGray, strings.Repeat("─", 40), colorReset)
	fmt.Printf("%sStatus:%s\n", colorBold, colorReset)
	fmt.Printf("  %sLocation:%s   %s\n", colorGray, colorReset, gitConfig.LocalPath)
	fmt.Printf("  %sLast sync:%s  %s\n", colorGray, colorReset, manager.GetLastSync().Format("2006-01-02 15:04:05"))
	
	fmt.Printf("\n%s💡 Tip:%s Run this command again to sync with latest changes.\n", colorYellow, colorReset)
	
	fmt.Printf("\n%s✨ Done!%s The vault is ready at: %s%s%s\n", 
		colorGreen, colorReset, colorCyan, gitConfig.LocalPath, colorReset)
}
