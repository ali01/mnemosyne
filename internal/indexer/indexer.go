// Package indexer handles vault parsing and database synchronization.
package indexer

import (
	"fmt"
	"log"
	"time"

	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/store"
	"github.com/ali01/mnemosyne/internal/vault"
)

// Indexer parses vault files and stores the graph in the database.
type Indexer struct {
	store      *store.Store
	vaultPath  string
	classifier *vault.NodeClassifier
}

// New creates an indexer for the given vault path.
func New(s *store.Store, vaultPath string, classConfig *config.NodeClassificationConfig) (*Indexer, error) {
	var classifier *vault.NodeClassifier
	if classConfig != nil {
		var err error
		classifier, err = vault.NewNodeClassifierFromConfig(classConfig)
		if err != nil {
			return nil, fmt.Errorf("create node classifier: %w", err)
		}
	} else {
		classifier = vault.NewNodeClassifier()
	}

	return &Indexer{
		store:      s,
		vaultPath:  vaultPath,
		classifier: classifier,
	}, nil
}

// FullIndex parses the entire vault and replaces all nodes and edges in the database.
func (idx *Indexer) FullIndex() error {
	start := time.Now()
	log.Printf("Starting full index of %s", idx.vaultPath)

	graph, err := idx.parseAndBuild()
	if err != nil {
		return err
	}

	if err := idx.store.ReplaceAllNodesAndEdges(graph.Nodes, graph.Edges); err != nil {
		return fmt.Errorf("store graph: %w", err)
	}

	if err := idx.store.SetMetadata("last_index", time.Now().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("set metadata: %w", err)
	}

	log.Printf("Full index completed in %v: %d nodes, %d edges",
		time.Since(start), len(graph.Nodes), len(graph.Edges))
	return nil
}

// IndexFile parses a single file and upserts its node and edges.
// It re-parses the entire vault to correctly resolve links, but only
// updates the changed node and its outgoing edges.
func (idx *Indexer) IndexFile(relPath string) error {
	log.Printf("Incremental index: %s", relPath)

	graph, err := idx.parseAndBuild()
	if err != nil {
		return err
	}

	// Find the node for this file
	var node *models.VaultNode
	for i := range graph.Nodes {
		if graph.Nodes[i].FilePath == relPath {
			node = &graph.Nodes[i]
			break
		}
	}
	if node == nil {
		// File doesn't produce a node (no frontmatter ID, etc.)
		return nil
	}

	if err := idx.store.UpsertNode(node); err != nil {
		return fmt.Errorf("upsert node: %w", err)
	}

	// Replace outgoing edges for this node
	if err := idx.store.DeleteEdgesBySource(node.ID); err != nil {
		return fmt.Errorf("delete old edges: %w", err)
	}
	for _, e := range graph.Edges {
		if e.SourceID == node.ID {
			if err := idx.store.UpsertEdge(&e); err != nil {
				return fmt.Errorf("upsert edge: %w", err)
			}
		}
	}

	return nil
}

// RemoveFile removes a node and all its edges by file path.
func (idx *Indexer) RemoveFile(relPath string) error {
	log.Printf("Removing file: %s", relPath)

	node, err := idx.store.GetNodeByPath(relPath)
	if err != nil {
		return nil
	}

	if err := idx.store.DeleteEdgesByNode(node.ID); err != nil {
		return fmt.Errorf("delete edges: %w", err)
	}
	if err := idx.store.DeleteNode(node.ID); err != nil {
		return fmt.Errorf("delete node: %w", err)
	}

	return nil
}

// parseAndBuild runs the vault parser and graph builder.
func (idx *Indexer) parseAndBuild() (*vault.Graph, error) {
	parser := vault.NewParser(idx.vaultPath, 4, 100)
	parseResult, err := parser.ParseVault()
	if err != nil {
		return nil, fmt.Errorf("parse vault: %w", err)
	}

	builder := vault.NewGraphBuilder(idx.classifier, vault.GraphBuilderConfig{
		DefaultWeight: 1.0,
		SkipOrphans:   false,
	})
	graph, err := builder.BuildGraph(parseResult)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	return graph, nil
}
