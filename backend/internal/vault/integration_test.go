package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestClassifier creates a classifier with rules for integration tests
func createTestClassifier(t *testing.T) *NodeClassifier {
	rules := []ClassificationRule{
		{
			Name:     "index_tag",
			Priority: 1,
			Matcher:  func(f *MarkdownFile) bool { return hasTag(f, "index") },
			NodeType: "index",
		},
		{
			Name:     "open_question_tag",
			Priority: 1,
			Matcher:  func(f *MarkdownFile) bool { return hasTag(f, "open-question") },
			NodeType: "question",
		},
		{
			Name:     "hub_prefix",
			Priority: 2,
			Matcher:  func(f *MarkdownFile) bool { return strings.HasPrefix(filepath.Base(f.Path), "~") },
			NodeType: "hub",
		},
		{
			Name:     "concepts_dir",
			Priority: 3,
			Matcher:  func(f *MarkdownFile) bool { return isInDirectory(f.Path, "concepts") },
			NodeType: "concept",
		},
		{
			Name:     "projects_dir",
			Priority: 3,
			Matcher:  func(f *MarkdownFile) bool { return isInDirectory(f.Path, "projects") },
			NodeType: "project",
		},
		{
			Name:     "questions_dir",
			Priority: 3,
			Matcher:  func(f *MarkdownFile) bool { return isInDirectory(f.Path, "questions") },
			NodeType: "question",
		},
	}

	classifier, err := NewNodeClassifierWithRules(rules, "note")
	require.NoError(t, err)
	return classifier
}

func TestIntegration_CompleteVaultParsing(t *testing.T) {
	// Create a complete vault structure for end-to-end testing
	tempDir := t.TempDir()

	// Create directory structure
	dirs := []string{
		"concepts",
		"projects",
		"references",
		"prototypes",
		"daily",
		"assets",
	}

	for _, dir := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(tempDir, dir), 0o750))
	}

	// Create interconnected files
	files := map[string]string{
		"index.md": `---
id: "index"
tags: ["index", "home"]
---
# Knowledge Base Index

This is the main index linking to:
- [[concepts/~graph-theory]] - Core concepts
- [[projects/mnemosyne]] - Main project
- [[daily/2023-01-01]] - Daily notes
`,
		"concepts/~graph-theory.md": `---
id: "graph-theory"
tags: ["hub", "concept"]
---
# Graph Theory Hub

Key concepts:
- [[concepts/nodes-and-edges]]
- [[concepts/algorithms#dijkstra|Dijkstra's Algorithm]]
- See also [[projects/mnemosyne#architecture]]
`,
		"concepts/nodes-and-edges.md": `---
id: "nodes-edges"
tags: ["concept", "fundamental"]
---
# Nodes and Edges

Basic building blocks of [[~graph-theory]].
Used in [[projects/mnemosyne]].
`,
		"concepts/algorithms.md": `---
id: "algorithms"
tags: ["concept"]
---
# Graph Algorithms

## Dijkstra
Shortest path algorithm.

## BFS
Breadth-first search.

Links back to [[~graph-theory]].
`,
		"projects/mnemosyne.md": `---
id: "mnemosyne"
tags: ["project", "active"]
aliases: ["Memory Palace", "Graph Visualizer"]
metadata:
  status: "in-progress"
  priority: "high"
---
# Mnemosyne Project

## Architecture
Uses [[concepts/nodes-and-edges]] for graph representation.
Implements [[concepts/algorithms#dijkstra]] for pathfinding.

## References
- [[references/paper-2023]]
- [[prototypes/demo-v1]]

![[assets/architecture.png]]
`,
		"references/paper-2023.md": `---
id: "paper-2023"
tags: ["reference", "academic"]
---
# Graph Visualization Paper 2023

Relevant to [[projects/mnemosyne]].
Discusses [[concepts/algorithms]].
`,
		"prototypes/demo-v1.md": `---
id: "demo-v1"
tags: ["prototype"]
---
# Demo Version 1

Prototype for [[projects/mnemosyne]].
Tests [[concepts/nodes-and-edges]] visualization.
`,
		"daily/2023-01-01.md": `---
id: "daily-2023-01-01"
tags: ["daily", "open-question"]
---
# 2023-01-01

Worked on [[projects/mnemosyne]].
Question: How to optimize [[concepts/algorithms#bfs]]?

Links to explore:
- [[non-existent-note]]
- [[another-missing-file]]
`,
	}

	// Write all files
	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	// Also create a non-markdown file that should be ignored
	require.NoError(t, os.WriteFile(
		filepath.Join(tempDir, "assets", "architecture.png"),
		[]byte("fake image data"),
		0o600,
	))

	// Parse the vault
	parser := NewParser(tempDir, 4, 10) // Use concurrency
	result, err := parser.ParseVault()

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify parsing statistics
	assert.Equal(t, 8, result.Stats.TotalFiles)
	assert.Equal(t, 8, result.Stats.ParsedFiles)
	assert.Equal(t, 0, result.Stats.FailedFiles)

	// Verify all files were parsed
	assert.Len(t, result.Files, 8)

	// Create test classifier
	classifier := createTestClassifier(t)

	// Verify specific file relationships
	indexFile := result.Files["index"]
	require.NotNil(t, indexFile)
	assert.Equal(t, "index", indexFile.GetNodeType(classifier))
	assert.Len(t, indexFile.Links, 3)

	// Verify hub node
	graphTheoryFile := result.Files["graph-theory"]
	require.NotNil(t, graphTheoryFile)
	assert.Equal(t, "hub", graphTheoryFile.GetNodeType(classifier))

	// Verify project file with aliases
	mnemosyneFile := result.Files["mnemosyne"]
	require.NotNil(t, mnemosyneFile)
	assert.Equal(t, "project", mnemosyneFile.GetNodeType(classifier))
	aliases, ok := mnemosyneFile.Frontmatter.GetStringSlice("aliases")
	assert.True(t, ok)
	assert.Contains(t, aliases, "Memory Palace")
	assert.Contains(t, aliases, "Graph Visualizer")

	// Verify open question
	dailyFile := result.Files["daily-2023-01-01"]
	require.NotNil(t, dailyFile)
	assert.Equal(t, "question", dailyFile.GetNodeType(classifier))

	// Test link resolution
	resolved, unresolved := result.Resolver.ResolveLinks(dailyFile.Links, dailyFile.Path)
	assert.Len(t, resolved, 2)   // mnemosyne and algorithms should resolve
	assert.Len(t, unresolved, 2) // Two non-existent notes

	// Verify cross-directory link resolution
	id, found := result.Resolver.ResolveLink("~graph-theory", "concepts/algorithms.md")
	assert.True(t, found)
	assert.Equal(t, "graph-theory", id)

	// Verify section links
	var sectionLink *WikiLink
	for _, link := range mnemosyneFile.Links {
		if link.Section == "dijkstra" {
			sectionLink = &link
			break
		}
	}
	require.NotNil(t, sectionLink)
	assert.Equal(t, "concepts/algorithms", sectionLink.Target)
	assert.Equal(t, "dijkstra", sectionLink.Section)
	assert.Equal(t, "concepts/algorithms#dijkstra", sectionLink.DisplayText) // No custom alias

	// Verify embed link
	var embedLink *WikiLink
	for _, link := range mnemosyneFile.Links {
		if link.LinkType == "embed" {
			embedLink = &link
			break
		}
	}
	require.NotNil(t, embedLink)
	assert.Equal(t, "assets/architecture.png", embedLink.Target)

	// Verify resolver statistics
	stats := result.Resolver.GetStats()
	assert.Equal(t, 8, stats["total_files"])
	assert.Greater(t, stats["unique_basenames"], 0)
}

func TestIntegration_ErrorRecovery(t *testing.T) {
	// Test that parser continues despite individual file errors
	tempDir := t.TempDir()

	// Create files with various problems
	files := map[string]string{
		// Valid file
		"valid.md": `---
id: "valid"
---
This is valid content with [[links]].`,

		// Missing ID
		"missing-id.md": `---
tags: ["test"]
---
This file is missing the required ID.`,

		// Invalid YAML
		"broken-yaml.md": `---
id: "test"
tags: [unclosed
---
Broken YAML frontmatter.`,

		// Empty file
		"empty.md": ``,

		// Just frontmatter markers
		"markers-only.md": `---
---`,

		// Valid file referencing broken ones
		"referencer.md": `---
id: "referencer"
---
Links to [[valid]], [[missing-id]], [[broken-yaml]], [[empty]], and [[markers-only]].`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	// Parse vault
	parser := NewParser(tempDir, 2, 50)
	result, err := parser.ParseVault()

	// Should not error at parser level
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check statistics
	assert.Equal(t, 6, result.Stats.TotalFiles)
	assert.Equal(t, 4, result.Stats.ParsedFiles) // valid, referencer, empty, and markers-only (latter two have no frontmatter but that's ok)
	assert.Equal(t, 2, result.Stats.FailedFiles) // Only missing-id and broken-yaml fail

	// Verify error details
	assert.Len(t, result.ParseErrors, 2) // Only missing-id and broken-yaml
	errorPaths := make(map[string]bool)
	for _, e := range result.ParseErrors {
		errorPaths[e.FilePath] = true
		assert.NotNil(t, e.Error)
	}
	assert.True(t, errorPaths["missing-id.md"])
	assert.True(t, errorPaths["broken-yaml.md"])

	// Verify valid files were still processed
	validFile := result.Files["valid"]
	require.NotNil(t, validFile)

	referencerFile := result.Files["referencer"]
	require.NotNil(t, referencerFile)
	assert.Len(t, referencerFile.Links, 5)

	// Check that files were parsed
	// Files without frontmatter have empty ID and overwrite each other
	// Only the last one parsed remains
	emptyIDFile := result.Files[""]
	assert.NotNil(t, emptyIDFile) // One of empty.md or markers-only.md
	assert.Nil(t, emptyIDFile.Frontmatter)

	// Test link resolution for broken files
	// Note: empty.md and markers-only.md can resolve by basename even though they have no/empty ID
	resolved, unresolved := result.Resolver.ResolveLinks(referencerFile.Links, referencerFile.Path)
	assert.Len(t, resolved, 3)   // valid, empty, markers-only all resolve
	assert.Len(t, unresolved, 2) // missing-id and broken-yaml don't resolve
}

func TestIntegration_PerformanceWithManyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a vault with many interconnected files
	tempDir := t.TempDir()
	fileCount := 1000

	// Create files in batches to avoid too many open files
	batchSize := 100
	for batch := 0; batch < fileCount/batchSize; batch++ {
		for i := 0; i < batchSize; i++ {
			idx := batch*batchSize + i
			content := fmt.Sprintf(`---
id: "note-%d"
tags: ["category-%d", "batch-%d"]
---
# Note %d

Links to:
- [[note-%d]]
- [[note-%d]]
- [[note-%d]]
- [[non-existent-%d]]
`, idx, idx%10, batch, idx,
				(idx+1)%fileCount,
				(idx+10)%fileCount,
				(idx+100)%fileCount,
				idx)

			path := filepath.Join(tempDir, fmt.Sprintf("note-%d.md", idx))
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		}
	}

	// Time the parsing
	start := time.Now()
	parser := NewParser(tempDir, 8, 50) // Higher concurrency
	result, err := parser.ParseVault()
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, fileCount, result.Stats.ParsedFiles)
	assert.Equal(t, fileCount*4, result.Stats.TotalLinks) // 4 links per file

	// Performance assertions
	t.Logf("Parsed %d files in %v", fileCount, duration)
	assert.Less(t, duration, 30*time.Second, "Parsing should complete within 30 seconds")

	// Verify resolver efficiency
	startResolve := time.Now()
	testFile := result.Files["note-500"]
	require.NotNil(t, testFile)
	resolved, _ := result.Resolver.ResolveLinks(testFile.Links, testFile.Path)
	resolveTime := time.Since(startResolve)

	assert.Len(t, resolved, 3) // 3 valid links should resolve
	assert.Less(t, resolveTime, 10*time.Millisecond, "Link resolution should be fast")
}

func TestIntegration_CircularReferences(t *testing.T) {
	// Test handling of circular references
	tempDir := t.TempDir()

	files := map[string]string{
		"a.md": `---
id: "a"
---
Links to [[b]] and [[c]].`,
		"b.md": `---
id: "b"
---
Links to [[c]] and [[a]].`,
		"c.md": `---
id: "c"
---
Links to [[a]] and [[b]].`,
		"self-ref.md": `---
id: "self"
---
This note links to [[self-ref]] (itself).`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	parser := NewParser(tempDir, 2, 10)
	result, err := parser.ParseVault()

	require.NoError(t, err)
	assert.Equal(t, 4, result.Stats.ParsedFiles)

	// Verify all circular references resolve correctly
	for _, file := range result.Files {
		resolved, unresolved := result.Resolver.ResolveLinks(file.Links, file.Path)
		assert.Len(t, unresolved, 0, "All links should resolve in circular reference test")

		// For self-referencing file
		if file.GetID() == "self" {
			assert.Equal(t, "self", resolved["self-ref"])
		}
	}
}
