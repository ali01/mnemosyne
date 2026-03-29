// Package indexer handles vault parsing and database synchronization.
package indexer

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/discovery"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/store"
	"github.com/ali01/mnemosyne/internal/vault"
)

// IndexManager coordinates indexing across multiple vaults.
type IndexManager struct {
	store  *store.Store
	vaults map[int]*vaultState
}

type vaultState struct {
	id         int
	path       string
	graphs     []registeredGraph
	classifier *vault.NodeClassifier
}

type registeredGraph struct {
	id       int
	rootPath string
}

// NewIndexManager creates a new index manager.
func NewIndexManager(s *store.Store) *IndexManager {
	return &IndexManager{
		store:  s,
		vaults: make(map[int]*vaultState),
	}
}

// RegisterVault discovers graphs and registers a vault for indexing.
// Returns the vault ID and the list of graph IDs.
func (m *IndexManager) RegisterVault(vaultPath string) (int, []int, error) {
	name := filepath.Base(vaultPath)
	vaultID, err := m.store.UpsertVault(name, vaultPath)
	if err != nil {
		return 0, nil, fmt.Errorf("upsert vault: %w", err)
	}

	defs, err := discovery.Discover(vaultPath)
	if err != nil {
		return 0, nil, fmt.Errorf("discover graphs in %s: %w", vaultPath, err)
	}

	// Build classifier from the first graph that has node_classification
	var classConfig *config.NodeClassificationConfig
	for _, d := range defs {
		if d.NodeClassification != nil {
			classConfig = d.NodeClassification
			break
		}
	}

	var classifier *vault.NodeClassifier
	if classConfig != nil {
		classifier, err = vault.NewNodeClassifierFromConfig(classConfig)
		if err != nil {
			return 0, nil, fmt.Errorf("create classifier: %w", err)
		}
	} else {
		classifier = vault.NewNodeClassifier()
	}

	// Upsert each graph, collect IDs
	var graphs []registeredGraph
	var graphIDs []int
	for _, d := range defs {
		gid, err := m.store.UpsertGraph(vaultID, d.Name, d.RootPath, d.RawConfig)
		if err != nil {
			return 0, nil, fmt.Errorf("upsert graph %q: %w", d.RootPath, err)
		}
		graphs = append(graphs, registeredGraph{id: gid, rootPath: d.RootPath})
		graphIDs = append(graphIDs, gid)
	}

	// Delete graphs that no longer have a GRAPH.yaml
	if err := m.store.DeleteStaleGraphs(vaultID, graphIDs); err != nil {
		return 0, nil, fmt.Errorf("delete stale graphs: %w", err)
	}

	m.vaults[vaultID] = &vaultState{
		id:         vaultID,
		path:       vaultPath,
		graphs:     graphs,
		classifier: classifier,
	}

	return vaultID, graphIDs, nil
}

// FullIndexVault parses an entire vault and replaces its data in the database.
func (m *IndexManager) FullIndexVault(vaultID int) error {
	vs, ok := m.vaults[vaultID]
	if !ok {
		return fmt.Errorf("vault %d not registered", vaultID)
	}

	start := time.Now()
	log.Printf("Starting full index of %s", vs.path)

	graph, err := parseAndBuild(vs.path, vs.classifier)
	if err != nil {
		return err
	}

	// Set vault_id on all nodes
	for i := range graph.Nodes {
		graph.Nodes[i].VaultID = vaultID
	}

	// Compute graph memberships
	memberships := computeMemberships(vs.graphs, graph.Nodes)

	if err := m.store.ReplaceVaultData(vaultID, graph.Nodes, graph.Edges, memberships); err != nil {
		return fmt.Errorf("store vault data: %w", err)
	}

	if err := m.store.SetMetadata(fmt.Sprintf("last_index_vault_%d", vaultID), time.Now().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("set metadata: %w", err)
	}

	log.Printf("Full index of %s completed in %v: %d nodes, %d edges, %d graphs",
		vs.path, time.Since(start), len(graph.Nodes), len(graph.Edges), len(vs.graphs))
	return nil
}

// FullIndexAll indexes all registered vaults.
func (m *IndexManager) FullIndexAll() error {
	for vaultID := range m.vaults {
		if err := m.FullIndexVault(vaultID); err != nil {
			return err
		}
	}
	return nil
}

// IndexFile incrementally indexes a single file. Returns affected graph IDs.
func (m *IndexManager) IndexFile(vaultID int, relPath string) ([]int, error) {
	vs, ok := m.vaults[vaultID]
	if !ok {
		return nil, fmt.Errorf("vault %d not registered", vaultID)
	}

	log.Printf("Incremental index: %s (vault %d)", relPath, vaultID)

	graph, err := parseAndBuild(vs.path, vs.classifier)
	if err != nil {
		return nil, err
	}

	var node *models.VaultNode
	for i := range graph.Nodes {
		if graph.Nodes[i].FilePath == relPath {
			node = &graph.Nodes[i]
			break
		}
	}
	if node == nil {
		return nil, nil
	}

	node.VaultID = vaultID
	if err := m.store.UpsertNode(node); err != nil {
		return nil, fmt.Errorf("upsert node: %w", err)
	}

	if err := m.store.DeleteEdgesBySource(node.ID); err != nil {
		return nil, fmt.Errorf("delete old edges: %w", err)
	}
	for _, e := range graph.Edges {
		if e.SourceID == node.ID {
			if err := m.store.UpsertEdge(&e); err != nil {
				return nil, fmt.Errorf("upsert edge: %w", err)
			}
		}
	}

	// Update graph memberships for this node
	var affectedGraphIDs []int
	for _, g := range vs.graphs {
		if discovery.IsUnderPath(relPath, g.rootPath) {
			affectedGraphIDs = append(affectedGraphIDs, g.id)
		}
	}
	if err := m.store.ReplaceGraphMemberships(node.ID, affectedGraphIDs); err != nil {
		return nil, fmt.Errorf("update memberships: %w", err)
	}

	return affectedGraphIDs, nil
}

// RemoveFile removes a node by file path. Returns affected graph IDs.
func (m *IndexManager) RemoveFile(vaultID int, relPath string) ([]int, error) {
	vs, ok := m.vaults[vaultID]
	if !ok {
		return nil, fmt.Errorf("vault %d not registered", vaultID)
	}

	log.Printf("Removing file: %s (vault %d)", relPath, vaultID)

	node, err := m.store.GetNodeByVaultPath(vaultID, relPath)
	if err != nil {
		return nil, nil // not found, nothing to do
	}

	// Determine affected graphs before deletion
	var affectedGraphIDs []int
	for _, g := range vs.graphs {
		if discovery.IsUnderPath(relPath, g.rootPath) {
			affectedGraphIDs = append(affectedGraphIDs, g.id)
		}
	}

	if err := m.store.DeleteEdgesByNode(node.ID); err != nil {
		return nil, fmt.Errorf("delete edges: %w", err)
	}
	if err := m.store.DeleteNode(node.ID); err != nil {
		return nil, fmt.Errorf("delete node: %w", err)
	}

	return affectedGraphIDs, nil
}

// HandleGraphYAML handles creation or deletion of a GRAPH.yaml file.
// Returns affected graph IDs and whether the graph list changed.
func (m *IndexManager) HandleGraphYAML(vaultID int, relDir string, created bool) ([]int, error) {
	vs, ok := m.vaults[vaultID]
	if !ok {
		return nil, fmt.Errorf("vault %d not registered", vaultID)
	}

	if created {
		// Re-discover to validate no nesting
		defs, err := discovery.Discover(vs.path)
		if err != nil {
			return nil, fmt.Errorf("re-discover graphs: %w", err)
		}

		// Find the new graph def
		var newDef *discovery.GraphDef
		for i := range defs {
			if defs[i].RootPath == relDir {
				newDef = &defs[i]
				break
			}
		}
		if newDef == nil {
			return nil, nil
		}

		gid, err := m.store.UpsertGraph(vaultID, newDef.Name, newDef.RootPath, newDef.RawConfig)
		if err != nil {
			return nil, fmt.Errorf("upsert graph: %w", err)
		}

		vs.graphs = append(vs.graphs, registeredGraph{id: gid, rootPath: newDef.RootPath})

		// Populate memberships from existing nodes
		nodes, err := m.store.GetAllNodes()
		if err != nil {
			return nil, err
		}
		var nodeIDs []string
		for _, n := range nodes {
			if n.VaultID == vaultID && discovery.IsUnderPath(n.FilePath, newDef.RootPath) {
				nodeIDs = append(nodeIDs, n.ID)
			}
		}
		for _, nid := range nodeIDs {
			if err := m.store.ReplaceGraphMemberships(nid, []int{gid}); err != nil {
				return nil, err
			}
		}

		return []int{gid}, nil
	}

	// Deleted: find and remove the graph
	for i, g := range vs.graphs {
		if g.rootPath == relDir {
			if err := m.store.DeleteGraph(g.id); err != nil {
				return nil, fmt.Errorf("delete graph: %w", err)
			}
			vs.graphs = append(vs.graphs[:i], vs.graphs[i+1:]...)
			return []int{g.id}, nil
		}
	}

	return nil, nil
}

// GetVaultID returns the vault ID for a given path, or 0 if not registered.
func (m *IndexManager) GetVaultID(vaultPath string) int {
	for _, vs := range m.vaults {
		if vs.path == vaultPath {
			return vs.id
		}
	}
	return 0
}

// computeMemberships determines which nodes belong to which graphs.
func computeMemberships(graphs []registeredGraph, nodes []models.VaultNode) map[int][]string {
	memberships := make(map[int][]string)
	for _, g := range graphs {
		for _, n := range nodes {
			if discovery.IsUnderPath(n.FilePath, g.rootPath) {
				memberships[g.id] = append(memberships[g.id], n.ID)
			}
		}
	}
	return memberships
}

// parseAndBuild runs the vault parser and graph builder.
func parseAndBuild(vaultPath string, classifier *vault.NodeClassifier) (*vault.Graph, error) {
	parser := vault.NewParser(vaultPath, 4, 100)
	parseResult, err := parser.ParseVault()
	if err != nil {
		return nil, fmt.Errorf("parse vault: %w", err)
	}

	builder := vault.NewGraphBuilder(classifier, vault.GraphBuilderConfig{
		DefaultWeight: 1.0,
		SkipOrphans:   false,
	})
	graph, err := builder.BuildGraph(parseResult)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	return graph, nil
}
