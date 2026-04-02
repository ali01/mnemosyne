// Package store provides SQLite-based data access for Mnemosyne.
package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store provides all database operations for Mnemosyne.
type Store struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at dbPath and initializes the schema.
func New(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	// Migrate: add archived column if missing (for databases created before this feature)
	db.Exec(`ALTER TABLE graphs ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`)

	return &Store{db: db}, nil
}

// NewMemory creates an in-memory SQLite store for testing.
func NewMemory() (*Store, error) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	// With :memory:, each connection gets its own database. Limit to one
	// connection so all queries share the same schema and data.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// --- Vault operations ---

// UpsertVault inserts or updates a vault, returning its ID.
func (s *Store) UpsertVault(name, path string) (int, error) {
	var id int
	err := s.db.QueryRow(`
		INSERT INTO vaults (name, path) VALUES (?, ?)
		ON CONFLICT(path) DO UPDATE SET name=excluded.name
		RETURNING id
	`, name, path).Scan(&id)
	return id, err
}

// GetVaults returns all registered vaults.
func (s *Store) GetVaults() ([]models.Vault, error) {
	rows, err := s.db.Query(`SELECT id, name, path FROM vaults ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vaults []models.Vault
	for rows.Next() {
		var v models.Vault
		if err := rows.Scan(&v.ID, &v.Name, &v.Path); err != nil {
			return nil, err
		}
		vaults = append(vaults, v)
	}
	return vaults, rows.Err()
}

// --- Graph operations ---

// UpsertGraph inserts or updates a graph definition, returning its ID.
// If the graph was previously archived, this unarchives it.
func (s *Store) UpsertGraph(vaultID int, name, rootPath, config string) (int, error) {
	var id int
	err := s.db.QueryRow(`
		INSERT INTO graphs (vault_id, name, root_path, config, archived, updated_at)
		VALUES (?, ?, ?, ?, 0, datetime('now'))
		ON CONFLICT(vault_id, root_path) DO UPDATE SET
			name=excluded.name, config=excluded.config, archived=0, updated_at=datetime('now')
		RETURNING id
	`, vaultID, name, rootPath, config).Scan(&id)
	return id, err
}

// ArchiveGraph soft-deletes a graph by setting archived=1.
// The graph remains in the DB and the indexer continues maintaining it.
func (s *Store) ArchiveGraph(graphID int) error {
	_, err := s.db.Exec(`UPDATE graphs SET archived = 1, updated_at = datetime('now') WHERE id = ?`, graphID)
	return err
}

// ArchiveStaleGraphs archives graphs for a vault that are not in the active list.
func (s *Store) ArchiveStaleGraphs(vaultID int, activeGraphIDs []int) error {
	if len(activeGraphIDs) == 0 {
		_, err := s.db.Exec(`UPDATE graphs SET archived = 1, updated_at = datetime('now') WHERE vault_id = ?`, vaultID)
		return err
	}
	placeholders := make([]string, len(activeGraphIDs))
	args := make([]interface{}, 0, len(activeGraphIDs)+1)
	args = append(args, vaultID)
	for i, id := range activeGraphIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(
		`UPDATE graphs SET archived = 1, updated_at = datetime('now') WHERE vault_id = ? AND id NOT IN (%s)`,
		strings.Join(placeholders, ","),
	)
	_, err := s.db.Exec(query, args...)
	return err
}

// PermanentlyDeleteGraph hard-deletes a graph and cascades to graph_nodes and node_positions.
func (s *Store) PermanentlyDeleteGraph(graphID int) error {
	_, err := s.db.Exec(`DELETE FROM graphs WHERE id = ?`, graphID)
	return err
}

// GetGraphInfo retrieves a single graph by ID.
func (s *Store) GetGraphInfo(graphID int) (*models.GraphInfo, error) {
	var g models.GraphInfo
	var config sql.NullString
	err := s.db.QueryRow(`
		SELECT g.id, g.vault_id, v.name, g.name, g.root_path, g.config, g.archived
		FROM graphs g JOIN vaults v ON v.id = g.vault_id
		WHERE g.id = ?
	`, graphID).Scan(&g.ID, &g.VaultID, &g.VaultName, &g.Name, &g.RootPath, &config, &g.Archived)
	if err != nil {
		return nil, err
	}
	g.Config = config.String
	return &g, nil
}

// GetGraphsByVault returns active (non-archived) graphs for a vault.
func (s *Store) GetGraphsByVault(vaultID int) ([]models.GraphInfo, error) {
	rows, err := s.db.Query(`
		SELECT g.id, g.vault_id, v.name, g.name, g.root_path, g.config, g.archived,
			(SELECT COUNT(*) FROM graph_nodes gn WHERE gn.graph_id = g.id)
		FROM graphs g JOIN vaults v ON v.id = g.vault_id
		WHERE g.vault_id = ? AND g.archived = 0
		ORDER BY g.root_path
	`, vaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGraphInfos(rows)
}

// GetAllGraphs returns all active (non-archived) graphs across all vaults.
func (s *Store) GetAllGraphs() ([]models.GraphInfo, error) {
	rows, err := s.db.Query(`
		SELECT g.id, g.vault_id, v.name, g.name, g.root_path, g.config, g.archived,
			(SELECT COUNT(*) FROM graph_nodes gn WHERE gn.graph_id = g.id)
		FROM graphs g JOIN vaults v ON v.id = g.vault_id
		WHERE g.archived = 0
		ORDER BY v.name, g.root_path
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGraphInfos(rows)
}

// GetAllGraphsIncludeArchived returns all graphs (active + archived) for CLI listing.
func (s *Store) GetAllGraphsIncludeArchived() ([]models.GraphInfo, error) {
	rows, err := s.db.Query(`
		SELECT g.id, g.vault_id, v.name, g.name, g.root_path, g.config, g.archived,
			(SELECT COUNT(*) FROM graph_nodes gn WHERE gn.graph_id = g.id)
		FROM graphs g JOIN vaults v ON v.id = g.vault_id
		ORDER BY v.name, g.root_path
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGraphInfos(rows)
}

// GetGraphsByVaultIncludeArchived returns all graphs (active + archived) for a vault.
func (s *Store) GetGraphsByVaultIncludeArchived(vaultID int) ([]models.GraphInfo, error) {
	rows, err := s.db.Query(`
		SELECT g.id, g.vault_id, v.name, g.name, g.root_path, g.config, g.archived,
			(SELECT COUNT(*) FROM graph_nodes gn WHERE gn.graph_id = g.id)
		FROM graphs g JOIN vaults v ON v.id = g.vault_id
		WHERE g.vault_id = ?
		ORDER BY g.root_path
	`, vaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGraphInfos(rows)
}

func scanGraphInfos(rows *sql.Rows) ([]models.GraphInfo, error) {
	var graphs []models.GraphInfo
	for rows.Next() {
		var g models.GraphInfo
		var config sql.NullString
		if err := rows.Scan(&g.ID, &g.VaultID, &g.VaultName, &g.Name, &g.RootPath, &config, &g.Archived, &g.NodeCount); err != nil {
			return nil, err
		}
		g.Config = config.String
		graphs = append(graphs, g)
	}
	return graphs, rows.Err()
}

// --- Node operations ---

// UpsertNode inserts or updates a node. VaultID must be set.
func (s *Store) UpsertNode(n *models.VaultNode) error {
	tags, err := json.Marshal(n.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags for node %s: %w", n.ID, err)
	}
	meta, err := json.Marshal(n.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata for node %s: %w", n.ID, err)
	}

	_, err = s.db.Exec(`
		INSERT INTO nodes (id, vault_id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at, parsed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			vault_id=excluded.vault_id, file_path=excluded.file_path, title=excluded.title,
			content=excluded.content, frontmatter=excluded.frontmatter, node_type=excluded.node_type,
			tags=excluded.tags, in_degree=excluded.in_degree, out_degree=excluded.out_degree,
			created_at=excluded.created_at, updated_at=excluded.updated_at, parsed_at=datetime('now')
	`, n.ID, n.VaultID, n.FilePath, n.Title, n.Content, string(meta), n.NodeType, string(tags),
		n.InDegree, n.OutDegree,
		n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339))
	return err
}

// GetNode retrieves a single node by ID.
func (s *Store) GetNode(id string) (*models.VaultNode, error) {
	row := s.db.QueryRow(`SELECT id, vault_id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at FROM nodes WHERE id = ?`, id)
	return scanNode(row)
}

// GetNodeByVaultPath retrieves a node by vault ID and file path.
func (s *Store) GetNodeByVaultPath(vaultID int, path string) (*models.VaultNode, error) {
	row := s.db.QueryRow(`SELECT id, vault_id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at FROM nodes WHERE vault_id = ? AND file_path = ?`, vaultID, path)
	return scanNode(row)
}

// DeleteNode removes a node and its associated edges.
func (s *Store) DeleteNode(id string) error {
	_, err := s.db.Exec(`DELETE FROM nodes WHERE id = ?`, id)
	return err
}

// GetAllNodes returns all nodes (without content for performance).
func (s *Store) GetAllNodes() ([]models.VaultNode, error) {
	rows, err := s.db.Query(`SELECT id, vault_id, file_path, title, '', frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at FROM nodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// --- Edge operations ---

// UpsertEdge inserts or updates an edge.
func (s *Store) UpsertEdge(e *models.VaultEdge) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	_, err := s.db.Exec(`
		INSERT INTO edges (id, source_id, target_id, edge_type, display_text, weight, created_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(source_id, target_id, edge_type) DO UPDATE SET
			display_text=excluded.display_text, weight=excluded.weight
	`, e.ID, e.SourceID, e.TargetID, e.EdgeType, e.DisplayText, e.Weight)
	return err
}

// GetAllEdges returns all edges.
func (s *Store) GetAllEdges() ([]models.VaultEdge, error) {
	rows, err := s.db.Query(`SELECT id, source_id, target_id, edge_type, display_text, weight FROM edges`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEdges(rows)
}

// DeleteEdgesBySource removes all edges originating from a node.
func (s *Store) DeleteEdgesBySource(sourceID string) error {
	_, err := s.db.Exec(`DELETE FROM edges WHERE source_id = ?`, sourceID)
	return err
}

// DeleteEdgesByNode removes all edges connected to a node (source or target).
func (s *Store) DeleteEdgesByNode(nodeID string) error {
	_, err := s.db.Exec(`DELETE FROM edges WHERE source_id = ? OR target_id = ?`, nodeID, nodeID)
	return err
}

// --- Graph-scoped data ---

// GetGraphData returns nodes, edges, and positions scoped to a specific graph.
func (s *Store) GetGraphData(graphID int) (*models.Graph, error) {
	// Nodes in this graph
	nodeRows, err := s.db.Query(`
		SELECT n.id, n.vault_id, n.file_path, n.title, '', n.frontmatter, n.node_type, n.tags,
			n.in_degree, n.out_degree, n.created_at, n.updated_at
		FROM nodes n
		JOIN graph_nodes gn ON gn.node_id = n.id
		WHERE gn.graph_id = ?
	`, graphID)
	if err != nil {
		return nil, fmt.Errorf("get graph nodes: %w", err)
	}
	defer nodeRows.Close()
	nodes, err := scanNodes(nodeRows)
	if err != nil {
		return nil, fmt.Errorf("scan graph nodes: %w", err)
	}

	// Edges where both endpoints are in this graph
	edgeRows, err := s.db.Query(`
		SELECT e.id, e.source_id, e.target_id, e.edge_type, e.display_text, e.weight
		FROM edges e
		WHERE e.source_id IN (SELECT node_id FROM graph_nodes WHERE graph_id = ?)
		  AND e.target_id IN (SELECT node_id FROM graph_nodes WHERE graph_id = ?)
	`, graphID, graphID)
	if err != nil {
		return nil, fmt.Errorf("get graph edges: %w", err)
	}
	defer edgeRows.Close()
	edges, err := scanEdges(edgeRows)
	if err != nil {
		return nil, fmt.Errorf("scan graph edges: %w", err)
	}

	// Positions for this graph
	posRows, err := s.db.Query(`SELECT node_id, x, y, z, locked FROM node_positions WHERE graph_id = ?`, graphID)
	if err != nil {
		return nil, fmt.Errorf("get graph positions: %w", err)
	}
	defer posRows.Close()
	posMap := make(map[string]models.NodePosition)
	for posRows.Next() {
		var p models.NodePosition
		if err := posRows.Scan(&p.NodeID, &p.X, &p.Y, &p.Z, &p.Locked); err != nil {
			return nil, err
		}
		posMap[p.NodeID] = p
	}
	if err := posRows.Err(); err != nil {
		return nil, err
	}

	// Assemble API response
	apiNodes := make([]models.Node, 0, len(nodes))
	for _, n := range nodes {
		pos := posMap[n.ID]
		apiNodes = append(apiNodes, models.Node{
			ID:       n.ID,
			Title:    n.Title,
			FilePath: n.FilePath,
			Position: models.Position{X: pos.X, Y: pos.Y, Z: pos.Z},
			Metadata: map[string]interface{}{"type": n.NodeType},
		})
	}

	apiEdges := make([]models.Edge, 0, len(edges))
	for _, e := range edges {
		apiEdges = append(apiEdges, models.Edge{
			ID:     e.ID,
			Source: e.SourceID,
			Target: e.TargetID,
			Weight: e.Weight,
			Type:   e.EdgeType,
		})
	}

	return &models.Graph{Nodes: apiNodes, Edges: apiEdges}, nil
}

// GraphDataRaw holds rich graph data for handler-side transformation.
type GraphDataRaw struct {
	Config    string
	Nodes     []models.VaultNode
	Edges     []models.VaultEdge
	Positions map[string]models.NodePosition
}

// GetGraphDataRaw returns full node/edge/position data plus graph config for a graph.
// Unlike GetGraphData, this returns VaultNode (with tags, frontmatter) for filter/group evaluation.
func (s *Store) GetGraphDataRaw(graphID int) (*GraphDataRaw, error) {
	// Graph config
	var config string
	err := s.db.QueryRow(`SELECT COALESCE(config, '') FROM graphs WHERE id = ?`, graphID).Scan(&config)
	if err != nil {
		return nil, fmt.Errorf("get graph config: %w", err)
	}

	// Nodes in this graph (full data including content for frontmatter)
	nodeRows, err := s.db.Query(`
		SELECT n.id, n.vault_id, n.file_path, n.title, '', n.frontmatter, n.node_type, n.tags,
			n.in_degree, n.out_degree, n.created_at, n.updated_at
		FROM nodes n
		JOIN graph_nodes gn ON gn.node_id = n.id
		WHERE gn.graph_id = ?
	`, graphID)
	if err != nil {
		return nil, fmt.Errorf("get graph nodes: %w", err)
	}
	defer nodeRows.Close()
	nodes, err := scanNodes(nodeRows)
	if err != nil {
		return nil, fmt.Errorf("scan graph nodes: %w", err)
	}

	// Edges where both endpoints are in this graph
	edgeRows, err := s.db.Query(`
		SELECT e.id, e.source_id, e.target_id, e.edge_type, e.display_text, e.weight
		FROM edges e
		WHERE e.source_id IN (SELECT node_id FROM graph_nodes WHERE graph_id = ?)
		  AND e.target_id IN (SELECT node_id FROM graph_nodes WHERE graph_id = ?)
	`, graphID, graphID)
	if err != nil {
		return nil, fmt.Errorf("get graph edges: %w", err)
	}
	defer edgeRows.Close()
	edges, err := scanEdges(edgeRows)
	if err != nil {
		return nil, fmt.Errorf("scan graph edges: %w", err)
	}

	// Positions
	posRows, err := s.db.Query(`SELECT node_id, x, y, z, locked FROM node_positions WHERE graph_id = ?`, graphID)
	if err != nil {
		return nil, fmt.Errorf("get graph positions: %w", err)
	}
	defer posRows.Close()
	posMap := make(map[string]models.NodePosition)
	for posRows.Next() {
		var p models.NodePosition
		if err := posRows.Scan(&p.NodeID, &p.X, &p.Y, &p.Z, &p.Locked); err != nil {
			return nil, err
		}
		posMap[p.NodeID] = p
	}
	if err := posRows.Err(); err != nil {
		return nil, err
	}

	return &GraphDataRaw{
		Config:    config,
		Nodes:     nodes,
		Edges:     edges,
		Positions: posMap,
	}, nil
}

// SearchInGraph performs full-text search scoped to a specific graph.
func (s *Store) SearchInGraph(graphID int, query string) ([]models.VaultNode, error) {
	rows, err := s.db.Query(`
		SELECT n.id, n.vault_id, n.file_path, n.title, '', n.frontmatter, n.node_type, n.tags,
			n.in_degree, n.out_degree, n.created_at, n.updated_at
		FROM nodes n
		JOIN nodes_fts fts ON n.rowid = fts.rowid
		JOIN graph_nodes gn ON gn.node_id = n.id
		WHERE nodes_fts MATCH ? AND gn.graph_id = ?
		ORDER BY rank
		LIMIT 50
	`, query, graphID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// --- Position operations (graph-scoped) ---

// UpsertPosition inserts or updates a single node position within a graph.
func (s *Store) UpsertPosition(graphID int, p *models.NodePosition) error {
	_, err := s.db.Exec(`
		INSERT INTO node_positions (graph_id, node_id, x, y, z, locked, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(graph_id, node_id) DO UPDATE SET
			x=excluded.x, y=excluded.y, z=excluded.z, locked=excluded.locked, updated_at=datetime('now')
	`, graphID, p.NodeID, p.X, p.Y, p.Z, p.Locked)
	return err
}

// UpsertPositions batch-inserts or updates node positions within a graph.
func (s *Store) UpsertPositions(graphID int, positions []models.NodePosition) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO node_positions (graph_id, node_id, x, y, z, locked, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(graph_id, node_id) DO UPDATE SET
			x=excluded.x, y=excluded.y, z=excluded.z, locked=excluded.locked, updated_at=datetime('now')
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range positions {
		if _, err := stmt.Exec(graphID, p.NodeID, p.X, p.Y, p.Z, p.Locked); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetPositionsByGraph returns all positions for a graph, keyed by node ID.
func (s *Store) GetPositionsByGraph(graphID int) (map[string]models.NodePosition, error) {
	rows, err := s.db.Query(`SELECT node_id, x, y, z, locked FROM node_positions WHERE graph_id = ?`, graphID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make(map[string]models.NodePosition)
	for rows.Next() {
		var p models.NodePosition
		if err := rows.Scan(&p.NodeID, &p.X, &p.Y, &p.Z, &p.Locked); err != nil {
			return nil, err
		}
		p.GraphID = graphID
		positions[p.NodeID] = p
	}
	return positions, rows.Err()
}

// GetPositionCount returns the number of positions stored for a graph.
func (s *Store) GetPositionCount(graphID int) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM node_positions WHERE graph_id = ?`, graphID).Scan(&count)
	return count, err
}

// --- Graph membership ---

// ReplaceGraphMemberships replaces all graph memberships for a single node.
func (s *Store) ReplaceGraphMemberships(nodeID string, graphIDs []int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM graph_nodes WHERE node_id = ?`, nodeID); err != nil {
		return err
	}

	if len(graphIDs) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO graph_nodes (graph_id, node_id) VALUES (?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, gid := range graphIDs {
			if _, err := stmt.Exec(gid, nodeID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// --- Metadata ---

// GetMetadata retrieves a metadata value by key.
func (s *Store) GetMetadata(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM vault_metadata WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetMetadata sets a metadata key-value pair.
func (s *Store) SetMetadata(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO vault_metadata (key, value, updated_at) VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=datetime('now')
	`, key, value)
	return err
}

// --- Bulk operations ---

// ReplaceVaultData atomically replaces all nodes, edges, and graph memberships for a vault.
// Positions are preserved (no FK from node_positions.node_id to nodes.id).
func (s *Store) ReplaceVaultData(vaultID int, nodes []models.VaultNode, edges []models.VaultEdge, memberships map[int][]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete graph_nodes for this vault's graphs (before nodes are deleted)
	if _, err := tx.Exec(`DELETE FROM graph_nodes WHERE graph_id IN (SELECT id FROM graphs WHERE vault_id = ?)`, vaultID); err != nil {
		return fmt.Errorf("clear graph_nodes: %w", err)
	}

	// Delete nodes for this vault (cascades to edges)
	if _, err := tx.Exec(`DELETE FROM nodes WHERE vault_id = ?`, vaultID); err != nil {
		return fmt.Errorf("clear vault nodes: %w", err)
	}

	// Insert nodes
	nodeStmt, err := tx.Prepare(`
		INSERT INTO nodes (id, vault_id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at, parsed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		return err
	}
	defer nodeStmt.Close()

	for _, n := range nodes {
		tags, err := json.Marshal(n.Tags)
		if err != nil {
			return fmt.Errorf("marshal tags for node %s: %w", n.ID, err)
		}
		meta, err := json.Marshal(n.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata for node %s: %w", n.ID, err)
		}
		if _, err := nodeStmt.Exec(n.ID, vaultID, n.FilePath, n.Title, n.Content, string(meta), n.NodeType, string(tags),
			n.InDegree, n.OutDegree,
			n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339)); err != nil {
			return fmt.Errorf("insert node %s: %w", n.ID, err)
		}
	}

	// Insert edges
	edgeStmt, err := tx.Prepare(`
		INSERT INTO edges (id, source_id, target_id, edge_type, display_text, weight, created_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		return err
	}
	defer edgeStmt.Close()

	for _, e := range edges {
		if e.ID == "" {
			e.ID = uuid.New().String()
		}
		if e.SourceID == e.TargetID {
			continue
		}
		if _, err := edgeStmt.Exec(e.ID, e.SourceID, e.TargetID, e.EdgeType, e.DisplayText, e.Weight); err != nil {
			return fmt.Errorf("insert edge %s->%s: %w", e.SourceID, e.TargetID, err)
		}
	}

	// Insert graph memberships
	if len(memberships) > 0 {
		memberStmt, err := tx.Prepare(`INSERT INTO graph_nodes (graph_id, node_id) VALUES (?, ?)`)
		if err != nil {
			return err
		}
		defer memberStmt.Close()

		for graphID, nodeIDs := range memberships {
			for _, nodeID := range nodeIDs {
				if _, err := memberStmt.Exec(graphID, nodeID); err != nil {
					return fmt.Errorf("insert graph_node %d:%s: %w", graphID, nodeID, err)
				}
			}
		}
	}

	return tx.Commit()
}

// --- Internal scan helpers ---

type nodeScanner interface {
	Scan(dest ...any) error
}

func scanOneNode(sc nodeScanner) (models.VaultNode, error) {
	var n models.VaultNode
	var frontmatter, tags, nodeType, createdAt, updatedAt sql.NullString
	err := sc.Scan(&n.ID, &n.VaultID, &n.FilePath, &n.Title, &n.Content, &frontmatter, &nodeType, &tags, &n.InDegree, &n.OutDegree, &createdAt, &updatedAt)
	if err != nil {
		return n, err
	}
	n.NodeType = nodeType.String
	if frontmatter.Valid {
		if err := json.Unmarshal([]byte(frontmatter.String), &n.Metadata); err != nil {
			return n, fmt.Errorf("unmarshal frontmatter for node %s: %w", n.ID, err)
		}
	}
	if tags.Valid {
		if err := json.Unmarshal([]byte(tags.String), &n.Tags); err != nil {
			return n, fmt.Errorf("unmarshal tags for node %s: %w", n.ID, err)
		}
	}
	if createdAt.Valid {
		n.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if updatedAt.Valid {
		n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}
	return n, nil
}

func scanNode(row *sql.Row) (*models.VaultNode, error) {
	n, err := scanOneNode(row)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func scanNodes(rows *sql.Rows) ([]models.VaultNode, error) {
	var nodes []models.VaultNode
	for rows.Next() {
		n, err := scanOneNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func scanEdges(rows *sql.Rows) ([]models.VaultEdge, error) {
	var edges []models.VaultEdge
	for rows.Next() {
		var e models.VaultEdge
		var displayText sql.NullString
		if err := rows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.EdgeType, &displayText, &e.Weight); err != nil {
			return nil, err
		}
		e.DisplayText = displayText.String
		edges = append(edges, e)
	}
	return edges, rows.Err()
}
