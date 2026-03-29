// Package vault provides functionality for parsing and processing vault files
package vault

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/google/uuid"
)

// GraphBuilder transforms parsed vault data into a graph structure suitable for
// visualization and analysis. It implements a two-pass algorithm that first creates
// nodes from markdown files, then establishes edges based on wiki links between them.
//
// The builder handles various edge cases including:
// - Duplicate file IDs (keeps first occurrence)
// - Missing IDs (skips files)
// - Unresolved links (tracks but doesn't create edges)
// - Orphaned nodes (can be filtered based on configuration)
// - Self-referential links (creates valid edges)
//
// Thread Safety: GraphBuilder methods are NOT thread-safe. The builder assumes
// single-threaded execution. Concurrent access would require external synchronization.
type GraphBuilder struct {
	classifier *NodeClassifier
	config     GraphBuilderConfig
}

// GraphBuilderConfig contains configuration options for graph building.
// It allows customization of how the graph is constructed from vault data.
type GraphBuilderConfig struct {
	// DefaultWeight specifies the weight assigned to edges when no explicit weight is provided.
	// Typical values range from 0.5 to 2.0, with 1.0 being the standard default.
	// Higher weights indicate stronger connections between nodes.
	DefaultWeight float64

	// SkipOrphans determines whether nodes with no incoming or outgoing connections
	// should be excluded from the final graph. When true, isolated nodes are filtered out,
	// which can significantly reduce graph size for visualization purposes.
	SkipOrphans bool
}

// DuplicateID represents a file ID that appears in multiple vault files.
// Only the first occurrence is included in the graph; subsequent files are skipped.
type DuplicateID struct {
	// ID is the duplicate identifier found in multiple files
	ID string

	// KeptPath is the file path of the first occurrence (included in graph)
	KeptPath string

	// SkippedPaths contains all file paths that were skipped due to this duplicate
	SkippedPaths []string
}

// Graph represents the complete knowledge graph structure built from a parsed vault.
// It contains all nodes (representing vault files), edges (representing links between files),
// and metadata about the graph construction process. This structure is designed to be
// persisted to a database and used for visualization.
type Graph struct {
	// Nodes contains all vault files converted to graph nodes.
	// Each node represents a single markdown file with its metadata, content, and computed properties.
	Nodes []models.VaultNode

	// Edges contains all connections between nodes.
	// Each edge represents a wiki link from one file to another, preserving link context.
	Edges []models.VaultEdge

	// UnresolvedLinks tracks all links that couldn't be resolved to existing nodes.
	// This helps identify broken links or references to files outside the vault.
	UnresolvedLinks []UnresolvedLink

	// DuplicateIDs tracks all IDs that appear in multiple files.
	// This information is valuable for vault maintenance and debugging.
	DuplicateIDs []DuplicateID

	// Stats provides detailed metrics about the graph building process,
	// useful for debugging and understanding vault structure.
	Stats GraphStats
}

// GraphStats contains comprehensive statistics about the graph building process.
// These metrics help understand the vault's structure and identify potential issues
// like duplicate IDs or excessive orphaned nodes.
type GraphStats struct {
	// NodesCreated is the count of vault files successfully converted to graph nodes.
	// This excludes files without valid IDs or files that failed parsing.
	NodesCreated int

	// EdgesCreated is the count of unique edges created from wiki links.
	// Duplicate edges (same source, target, and type) are automatically deduplicated.
	EdgesCreated int

	// FilesSkipped tracks files that couldn't be processed, typically due to:
	// - Missing ID in frontmatter
	// - Parse errors
	// - File read errors
	FilesSkipped int

	// UnresolvedLinks counts links to files that exist but weren't included in the graph.
	// This typically happens when the target file has no ID in its frontmatter.
	// Note: Links to non-existent files are tracked separately in ParseResult.UnresolvedLinks.
	UnresolvedLinks int

	// OrphanedNodes counts nodes with neither incoming nor outgoing connections.
	// These represent isolated files that aren't part of the main knowledge graph.
	OrphanedNodes int

	// BuildDurationMS measures the total time taken to construct the graph in milliseconds.
	// Useful for performance monitoring and optimization of large vaults.
	BuildDurationMS int64
}

// edgeKey represents a unique edge for deduplication
type edgeKey struct {
	sourceID string
	targetID string
	edgeType string
}

// NewGraphBuilder creates a new graph builder with the given configuration.
// The classifier is used to categorize nodes based on file properties and frontmatter.
// If config.DefaultWeight is not specified or is invalid (<= 0), it defaults to 1.0.
//
// Example usage:
//
//	classifier := NewNodeClassifier(classificationConfig)
//	builder := NewGraphBuilder(classifier, GraphBuilderConfig{
//	    DefaultWeight: 1.0,
//	    SkipOrphans: true,
//	})
//	graph, err := builder.BuildGraph(parseResult)
func NewGraphBuilder(classifier *NodeClassifier, config GraphBuilderConfig) *GraphBuilder {
	// Set defaults
	if config.DefaultWeight <= 0 {
		config.DefaultWeight = 1.0
	}

	return &GraphBuilder{
		classifier: classifier,
		config:     config,
	}
}

// BuildGraph transforms a ParseResult into a graph structure using a two-pass algorithm.
//
// First pass (node creation):
// - Iterates through all parsed markdown files
// - Creates a VaultNode for each file with a valid ID
// - Tracks duplicate IDs and skips subsequent occurrences
// - Maintains a map for efficient link resolution in the second pass
//
// Second pass (edge creation):
// - Resolves wiki links to target node IDs
// - Creates edges with appropriate metadata (display text, timestamps)
// - Deduplicates edges based on source, target, and type
// - Updates node in/out degree counts
//
// The method is designed to handle large vaults efficiently (tested with 50,000+ nodes)
// and provides comprehensive statistics about the building process.
//
// Returns an error if parseResult is nil or if critical errors occur during processing.
func (gb *GraphBuilder) BuildGraph(parseResult *ParseResult) (*Graph, error) {
	if parseResult == nil {
		return nil, fmt.Errorf("parseResult cannot be nil")
	}

	startTime := time.Now()
	stats := &GraphStats{}

	log.Printf("Building graph from %d parsed files...", len(parseResult.Files))

	// Pass 1: Build nodes from files
	nodeMap, linkMap, duplicatesMap, err := gb.buildNodes(parseResult.Files, stats)
	if err != nil {
		return nil, fmt.Errorf("failed to build nodes from %d files: %w", len(parseResult.Files), err)
	}

	// Pass 2: Build edges from links
	edges, err := gb.buildEdges(nodeMap, linkMap, parseResult, stats)
	if err != nil {
		return nil, fmt.Errorf("failed to build edges from %d nodes: %w", len(nodeMap), err)
	}

	// Calculate final statistics and prepare result
	result := gb.finalizeResult(nodeMap, edges, parseResult.UnresolvedLinks, duplicatesMap, stats)

	duration := time.Since(startTime)
	stats.BuildDurationMS = duration.Milliseconds()
	result.Stats = *stats

	log.Printf("Graph building completed in %v", duration)
	log.Printf("Created: %d nodes, %d edges | Skipped: %d files | Orphaned: %d nodes",
		stats.NodesCreated, stats.EdgesCreated, stats.FilesSkipped, stats.OrphanedNodes)

	// Log unresolved links if any
	totalUnresolved := len(parseResult.UnresolvedLinks) + stats.UnresolvedLinks
	if totalUnresolved > 0 {
		log.Printf("Unresolved links: %d to non-existent files, %d to files without IDs",
			len(parseResult.UnresolvedLinks), stats.UnresolvedLinks)
	}

	return result, nil
}

// buildNodes creates VaultNode objects from MarkdownFile objects (Pass 1)
// Note: This method is NOT thread-safe. It assumes single-threaded execution.
// If made concurrent in the future, the duplicate detection logic would need
// synchronization to prevent race conditions.
// Returns: nodeMap (ID -> VaultNode), linkMap (ID -> outgoing links), duplicatesMap
func (gb *GraphBuilder) buildNodes(files map[string]*MarkdownFile, stats *GraphStats) (
	map[string]*models.VaultNode,
	map[string][]WikiLink,
	map[string]*DuplicateID,
	error,
) {
	nodeMap := make(map[string]*models.VaultNode)
	linkMap := make(map[string][]WikiLink)         // Track outgoing links separately
	duplicatesMap := make(map[string]*DuplicateID) // Track duplicates by ID
	seenIDs := make(map[string]string)             // ID -> path mapping for first occurrence

	for _, file := range files {
		// Get ID from file
		id := ""
		if file.Frontmatter != nil {
			id = file.Frontmatter.ID
		}

		// Skip files without valid IDs
		if id == "" {
			stats.FilesSkipped++
			continue
		}

		// Check for duplicate IDs
		// TOCTOU Note: In concurrent scenarios, another goroutine could insert the same ID
		// between this check and the update below. Current single-threaded design prevents this.
		if existingPath, exists := seenIDs[id]; exists {
			// Track this duplicate
			dup := duplicatesMap[id]
			if dup == nil {
				dup = &DuplicateID{
					ID:           id,
					KeptPath:     existingPath,
					SkippedPaths: []string{},
				}
				duplicatesMap[id] = dup
			}
			dup.SkippedPaths = append(dup.SkippedPaths, file.Path)
			log.Printf("Warning: Duplicate ID '%s' found in files '%s' and '%s'. Keeping first occurrence.",
				id, existingPath, file.Path)
			continue
		}
		seenIDs[id] = file.Path

		// Create VaultNode from MarkdownFile
		node, err := gb.createNode(file, id)
		if err != nil {
			// Log error but continue processing other files
			stats.FilesSkipped++
			log.Printf("Warning: Failed to create node from file '%s' (ID: %s): %v", file.Path, id, err)
			continue
		}

		// Store node and links separately
		nodeMap[id] = node
		linkMap[id] = file.Links
		stats.NodesCreated++
	}

	return nodeMap, linkMap, duplicatesMap, nil
}

// buildEdges creates VaultEdge objects from WikiLinks (Pass 2)
func (gb *GraphBuilder) buildEdges(
	nodeMap map[string]*models.VaultNode,
	linkMap map[string][]WikiLink,
	parseResult *ParseResult,
	stats *GraphStats,
) ([]models.VaultEdge, error) {
	var edges []models.VaultEdge
	edgeSet := make(map[edgeKey]bool)   // For deduplication
	inDegreeMap := make(map[string]int) // Track in-degrees separately

	for sourceID, links := range linkMap {
		sourceNode := nodeMap[sourceID]
		outDegree := 0

		for _, link := range links {
			// Resolve the target node
			// Get source file path from node
			sourceFilePath := sourceNode.FilePath
			targetID, found := parseResult.Resolver.ResolveLink(link.Target, sourceFilePath)
			if !found {
				// This link was already counted as unresolved during parsing
				continue
			}

			// Check if target node exists in our graph
			_, exists := nodeMap[targetID]
			if !exists {
				// The file exists but wasn't included in the graph (e.g., missing ID)
				stats.UnresolvedLinks++
				continue
			}

			// Create edge
			edge, err := gb.createEdge(sourceID, targetID, link, sourceNode.UpdatedAt)
			if err != nil {
				// Log error but continue
				log.Printf("Warning: Failed to create edge from '%s' to '%s' (link: %s): %v",
					sourceID, targetID, link.Target, err)
				continue
			}

			// Deduplicate edges (same source, target, and type)
			key := edgeKey{
				sourceID: edge.SourceID,
				targetID: edge.TargetID,
				edgeType: edge.EdgeType,
			}
			if !edgeSet[key] {
				edges = append(edges, *edge)
				edgeSet[key] = true
				stats.EdgesCreated++
				outDegree++

				// Update in-degree for target node
				inDegreeMap[targetID]++
			}
		}

		// Update out-degree for source node
		sourceNode.OutDegree = outDegree
	}

	// Update in-degrees for all nodes
	for nodeID, node := range nodeMap {
		node.InDegree = inDegreeMap[nodeID]
	}

	return edges, nil
}

// createNode creates a VaultNode from a MarkdownFile
func (gb *GraphBuilder) createNode(file *MarkdownFile, id string) (*models.VaultNode, error) {
	title := file.Title

	// Determine node type using classifier
	nodeType := ""
	if gb.classifier != nil {
		nodeType = gb.classifier.ClassifyNode(file)
	}

	// Extract tags
	tags := file.GetTags()

	// Get timestamps
	createdAt := file.GetCreatedAt()
	modifiedAt := file.GetModifiedAt()

	// Build metadata map from frontmatter
	var metadata models.JSONMetadata
	if file.Frontmatter != nil && file.Frontmatter.Raw != nil {
		metadata = make(models.JSONMetadata)
		for k, v := range file.Frontmatter.Raw {
			metadata[k] = v
		}
	}

	node := &models.VaultNode{
		ID:         id,
		Title:      title,
		NodeType:   nodeType,
		Tags:       tags,
		Content:    file.Content,
		Metadata:   metadata,
		FilePath:   file.Path,
		InDegree:   0, // Will be calculated in edge building
		OutDegree:  0, // Will be calculated in edge building
		Centrality: 0, // Will be calculated by metrics calculator
		CreatedAt:  createdAt,
		UpdatedAt:  modifiedAt,
	}

	return node, nil
}

// validateEdgeIDs validates that both source and target IDs are non-empty
func validateEdgeIDs(sourceID, targetID string, link WikiLink) error {
	if sourceID == "" {
		return fmt.Errorf("sourceID cannot be empty for link to '%s'", link.Target)
	}
	if targetID == "" {
		return fmt.Errorf("targetID cannot be empty for link '%s' from source '%s'",
			link.Target, sourceID)
	}
	return nil
}

// createEdge creates a VaultEdge from a WikiLink
func (gb *GraphBuilder) createEdge(
	sourceID, targetID string,
	link WikiLink,
	timestamp time.Time,
) (*models.VaultEdge, error) {
	// Validate inputs
	if err := validateEdgeIDs(sourceID, targetID, link); err != nil {
		return nil, err
	}

	// Generate UUID for edge
	edgeID := uuid.New().String()

	// Determine display text
	displayText := link.DisplayText
	if displayText == "" && link.Section != "" {
		displayText = link.Section
	}

	edge := &models.VaultEdge{
		ID:          edgeID,
		SourceID:    sourceID,
		TargetID:    targetID,
		EdgeType:    link.LinkType,
		DisplayText: displayText,
		Weight:      gb.config.DefaultWeight,
		CreatedAt:   timestamp, // Use source file's timestamp for reproducibility
	}

	return edge, nil
}

// finalizeResult prepares the final Graph
func (gb *GraphBuilder) finalizeResult(
	nodeMap map[string]*models.VaultNode,
	edges []models.VaultEdge,
	unresolvedLinks []UnresolvedLink,
	duplicatesMap map[string]*DuplicateID,
	stats *GraphStats,
) *Graph {
	// Extract nodes from map
	nodes := make([]models.VaultNode, 0, len(nodeMap))
	for _, node := range nodeMap {
		// Count orphaned nodes
		if node.InDegree == 0 && node.OutDegree == 0 {
			stats.OrphanedNodes++
			if gb.config.SkipOrphans {
				continue
			}
		}
		nodes = append(nodes, *node)
	}

	// Convert duplicates map to slice
	duplicates := make([]DuplicateID, 0, len(duplicatesMap))
	for _, dup := range duplicatesMap {
		duplicates = append(duplicates, *dup)
	}

	// Sort all slices for deterministic output
	// Sort nodes by ID
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	// Sort edges by source ID, then target ID, then edge type
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].SourceID != edges[j].SourceID {
			return edges[i].SourceID < edges[j].SourceID
		}
		if edges[i].TargetID != edges[j].TargetID {
			return edges[i].TargetID < edges[j].TargetID
		}
		return edges[i].EdgeType < edges[j].EdgeType
	})

	// Sort duplicate IDs by ID
	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].ID < duplicates[j].ID
	})

	return &Graph{
		Nodes:           nodes,
		Edges:           edges,
		UnresolvedLinks: unresolvedLinks,
		DuplicateIDs:    duplicates,
		Stats:           *stats,
	}
}
