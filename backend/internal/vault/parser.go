package vault

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Parser is the main vault parser that orchestrates the parsing process
// It coordinates reading markdown files, extracting metadata, and resolving links
type Parser struct {
	vaultPath   string        // Root path of the Obsidian vault
	resolver    *LinkResolver // Handles WikiLink resolution
	concurrency int           // Number of concurrent workers for parsing
	batchSize   int           // Number of files to process per batch
}

// ParseResult contains the complete parsed vault data
// This is the main output of the parsing process
type ParseResult struct {
	Files           map[string]*MarkdownFile // ID -> MarkdownFile mapping
	Resolver        *LinkResolver            // Link resolver with all mappings
	ParseErrors     []ParseError             // Errors encountered during parsing
	UnresolvedLinks []UnresolvedLink         // WikiLinks that couldn't be resolved
	Stats           ParseStats               // Statistics about the parsing process
}

// ParseError represents an error during parsing a specific file
type ParseError struct {
	FilePath string
	Error    error
}

// UnresolvedLink represents a WikiLink that couldn't be resolved to a target file
type UnresolvedLink struct {
	SourceID   string   // ID of the file containing the link
	SourcePath string   // Path of the file containing the link
	Link       WikiLink // The unresolved WikiLink
}

// ParseStats contains statistics about the parsing process
type ParseStats struct {
	TotalFiles      int           // Total markdown files found
	ParsedFiles     int           // Successfully parsed files
	FailedFiles     int           // Files that failed to parse
	TotalLinks      int           // Total WikiLinks found
	ResolvedLinks   int           // WikiLinks successfully resolved
	UnresolvedLinks int           // WikiLinks that couldn't be resolved
	StartTime       time.Time     // When parsing started
	EndTime         time.Time     // When parsing completed
	Duration        time.Duration // Total parsing duration
}

// NewParser creates a new vault parser with the specified configuration
func NewParser(vaultPath string, concurrency, batchSize int) *Parser {
	// Set defaults if not specified
	if concurrency <= 0 {
		concurrency = 4
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	return &Parser{
		vaultPath:   vaultPath,
		resolver:    NewLinkResolver(),
		concurrency: concurrency,
		batchSize:   batchSize,
	}
}

// ParseVault parses the entire vault and returns the result
// This is the main entry point for parsing an Obsidian vault
func (p *Parser) ParseVault() (*ParseResult, error) {
	// Initialize the result structure
	result := &ParseResult{
		Files:    make(map[string]*MarkdownFile),
		Resolver: p.resolver,
		Stats: ParseStats{
			StartTime: time.Now(),
		},
	}

	// Step 1: Discover all markdown files in the vault
	// This walks the directory tree and collects all .md file paths
	log.Printf("Scanning vault at %s for markdown files...", p.vaultPath)
	filePaths, err := p.collectMarkdownFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to collect markdown files: %w", err)
	}

	result.Stats.TotalFiles = len(filePaths)
	log.Printf("Found %d markdown files", len(filePaths))

	// Step 2: Parse all files concurrently
	// This reads each file, extracts frontmatter, and collects WikiLinks
	log.Printf("Parsing files with %d workers...", p.concurrency)
	p.processFilesConcurrently(filePaths, result)

	// Step 3: Resolve all WikiLinks to their target files
	// This matches link text to actual file IDs using various strategies
	log.Println("Resolving WikiLinks...")
	p.resolveAllLinks(result)

	// Step 4: Calculate final statistics
	result.Stats.EndTime = time.Now()
	result.Stats.Duration = result.Stats.EndTime.Sub(result.Stats.StartTime)

	log.Printf("Parsing completed in %v", result.Stats.Duration)
	log.Printf("Parsed: %d/%d files, Resolved: %d/%d links",
		result.Stats.ParsedFiles, result.Stats.TotalFiles,
		result.Stats.ResolvedLinks, result.Stats.TotalLinks)

	return result, nil
}

// collectMarkdownFiles walks the vault directory and collects all .md files
// It returns a slice of relative paths to all markdown files
func (p *Parser) collectMarkdownFiles() ([]string, error) {
	var files []string

	// Walk the directory tree starting from vault root
	err := filepath.Walk(p.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and files (like .git, .obsidian)
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir // Don't descend into hidden directories
			}
			return nil // Skip hidden files
		}

		// Collect markdown files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			// Convert to relative path for consistency
			relPath, err := filepath.Rel(p.vaultPath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

// processFilesConcurrently processes files using worker goroutines
// This enables parallel processing for better performance with large vaults
func (p *Parser) processFilesConcurrently(filePaths []string, result *ParseResult) {
	var wg sync.WaitGroup
	var mu sync.Mutex // Protects shared result data

	// Create a channel with all file paths to process
	// Workers will pull from this channel
	workCh := make(chan string, len(filePaths))
	for _, path := range filePaths {
		workCh <- path
	}
	close(workCh) // Close channel to signal no more work

	// Start worker goroutines
	for i := 0; i < p.concurrency; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()

			// Each worker processes files from the work channel
			for path := range workCh {
				// Process individual markdown file
				file, err := ProcessMarkdownFile(p.vaultPath, path)

				// Update results (with mutex for thread safety)
				mu.Lock()
				var shouldLogProgress bool
				var currentProgress int

				if err != nil {
					// Record parse error
					result.ParseErrors = append(result.ParseErrors, ParseError{
						FilePath: path,
						Error:    err,
					})
					result.Stats.FailedFiles++
				} else {
					// Store successfully parsed file
					id := file.GetID()
					result.Files[id] = file
					// Register file with resolver for link resolution
					p.resolver.AddFile(file)
					result.Stats.ParsedFiles++
					result.Stats.TotalLinks += len(file.Links)
				}

				// Check if we should log progress (inside mutex)
				if result.Stats.ParsedFiles%100 == 0 {
					shouldLogProgress = true
					currentProgress = result.Stats.ParsedFiles + result.Stats.FailedFiles
				}
				mu.Unlock()

				// Log progress outside mutex to avoid holding lock during I/O
				if shouldLogProgress {
					log.Printf("Progress: %d/%d files parsed",
						currentProgress, result.Stats.TotalFiles)
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
}

// resolveAllLinks resolves all WikiLinks in the parsed files
// This matches link text to actual file IDs using the resolver
func (p *Parser) resolveAllLinks(result *ParseResult) {
	// Iterate through all parsed files
	for id, file := range result.Files {
		// Check each WikiLink in the file
		for _, link := range file.Links {
			// Try to resolve the link target to a file ID
			// The resolver uses multiple strategies:
			// 1. Exact path match
			// 2. Relative path resolution
			// 3. Basename matching
			// 4. Fuzzy/normalized matching
			_, found := p.resolver.ResolveLink(link.Target, file.Path)
			if found {
				result.Stats.ResolvedLinks++
			} else {
				// Track unresolved links for debugging
				result.UnresolvedLinks = append(result.UnresolvedLinks, UnresolvedLink{
					SourceID:   id,
					SourcePath: file.Path,
					Link:       link,
				})
				result.Stats.UnresolvedLinks++
			}
		}
	}
}

// GetFile retrieves a parsed file by its ID
func (r *ParseResult) GetFile(id string) (*MarkdownFile, bool) {
	file, found := r.Files[id]
	return file, found
}

// GetFileByPath retrieves a parsed file by its path
// This is useful when you have a path but not the ID
func (r *ParseResult) GetFileByPath(path string) (*MarkdownFile, bool) {
	// Normalize the path by removing .md extension
	pathWithoutExt := strings.TrimSuffix(path, ".md")

	// Linear search through files (could be optimized with additional index)
	for _, file := range r.Files {
		filePathWithoutExt := strings.TrimSuffix(file.Path, ".md")
		if filePathWithoutExt == pathWithoutExt {
			return file, true
		}
	}
	return nil, false
}
