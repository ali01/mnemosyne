// Package store provides SQLite-based data access for Mnemosyne.
package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	return &Store{db: db}, nil
}

// NewMemory creates an in-memory SQLite store for testing.
func NewMemory() (*Store, error) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// --- Nodes ---

// UpsertNode inserts or updates a node.
func (s *Store) UpsertNode(n *models.VaultNode) error {
	tags, _ := json.Marshal(n.Tags)
	meta, _ := json.Marshal(n.Metadata)

	_, err := s.db.Exec(`
		INSERT INTO nodes (id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at, parsed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			file_path=excluded.file_path, title=excluded.title, content=excluded.content,
			frontmatter=excluded.frontmatter, node_type=excluded.node_type, tags=excluded.tags,
			in_degree=excluded.in_degree, out_degree=excluded.out_degree,
			created_at=excluded.created_at, updated_at=excluded.updated_at, parsed_at=datetime('now')
	`, n.ID, n.FilePath, n.Title, n.Content, string(meta), n.NodeType, string(tags),
		n.InDegree, n.OutDegree,
		n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339))
	return err
}

// GetNode retrieves a single node by ID.
func (s *Store) GetNode(id string) (*models.VaultNode, error) {
	row := s.db.QueryRow(`SELECT id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at FROM nodes WHERE id = ?`, id)
	return scanNode(row)
}

// GetNodeByPath retrieves a node by its file path.
func (s *Store) GetNodeByPath(path string) (*models.VaultNode, error) {
	row := s.db.QueryRow(`SELECT id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at FROM nodes WHERE file_path = ?`, path)
	return scanNode(row)
}

// DeleteNode removes a node and its associated edges.
func (s *Store) DeleteNode(id string) error {
	_, err := s.db.Exec(`DELETE FROM nodes WHERE id = ?`, id)
	return err
}

// GetAllNodes returns all nodes (without content for performance).
func (s *Store) GetAllNodes() ([]models.VaultNode, error) {
	rows, err := s.db.Query(`SELECT id, file_path, title, '', frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at FROM nodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// SearchNodes performs full-text search on nodes.
func (s *Store) SearchNodes(query string) ([]models.VaultNode, error) {
	rows, err := s.db.Query(`
		SELECT n.id, n.file_path, n.title, '', n.frontmatter, n.node_type, n.tags, n.in_degree, n.out_degree, n.created_at, n.updated_at
		FROM nodes n
		JOIN nodes_fts fts ON n.rowid = fts.rowid
		WHERE nodes_fts MATCH ?
		ORDER BY rank
		LIMIT 50
	`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// --- Edges ---

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

// --- Positions ---

// GetAllPositions returns all saved node positions.
func (s *Store) GetAllPositions() ([]models.NodePosition, error) {
	rows, err := s.db.Query(`SELECT node_id, x, y, z, locked FROM node_positions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []models.NodePosition
	for rows.Next() {
		var p models.NodePosition
		if err := rows.Scan(&p.NodeID, &p.X, &p.Y, &p.Z, &p.Locked); err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

// UpsertPosition inserts or updates a single node position.
func (s *Store) UpsertPosition(p *models.NodePosition) error {
	_, err := s.db.Exec(`
		INSERT INTO node_positions (node_id, x, y, z, locked, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(node_id) DO UPDATE SET x=excluded.x, y=excluded.y, z=excluded.z, locked=excluded.locked, updated_at=datetime('now')
	`, p.NodeID, p.X, p.Y, p.Z, p.Locked)
	return err
}

// UpsertPositions batch-inserts or updates node positions.
func (s *Store) UpsertPositions(positions []models.NodePosition) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO node_positions (node_id, x, y, z, locked, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(node_id) DO UPDATE SET x=excluded.x, y=excluded.y, z=excluded.z, locked=excluded.locked, updated_at=datetime('now')
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range positions {
		if _, err := stmt.Exec(p.NodeID, p.X, p.Y, p.Z, p.Locked); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// --- Graph (combined query for API) ---

// GetGraph returns all nodes, edges, and positions for the graph API.
func (s *Store) GetGraph() (*models.Graph, map[string]models.NodePosition, error) {
	nodes, err := s.GetAllNodes()
	if err != nil {
		return nil, nil, fmt.Errorf("get nodes: %w", err)
	}

	edges, err := s.GetAllEdges()
	if err != nil {
		return nil, nil, fmt.Errorf("get edges: %w", err)
	}

	positions, err := s.GetAllPositions()
	if err != nil {
		return nil, nil, fmt.Errorf("get positions: %w", err)
	}

	posMap := make(map[string]models.NodePosition, len(positions))
	for _, p := range positions {
		posMap[p.NodeID] = p
	}

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

	return &models.Graph{Nodes: apiNodes, Edges: apiEdges}, posMap, nil
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

// ReplaceAllNodesAndEdges atomically replaces all nodes and edges in a transaction.
// Positions are preserved.
func (s *Store) ReplaceAllNodesAndEdges(nodes []models.VaultNode, edges []models.VaultEdge) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM edges`); err != nil {
		return fmt.Errorf("clear edges: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM nodes`); err != nil {
		return fmt.Errorf("clear nodes: %w", err)
	}

	nodeStmt, err := tx.Prepare(`
		INSERT INTO nodes (id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at, parsed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		return err
	}
	defer nodeStmt.Close()

	for _, n := range nodes {
		tags, _ := json.Marshal(n.Tags)
		meta, _ := json.Marshal(n.Metadata)
		if _, err := nodeStmt.Exec(n.ID, n.FilePath, n.Title, n.Content, string(meta), n.NodeType, string(tags),
			n.InDegree, n.OutDegree,
			n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339)); err != nil {
			return fmt.Errorf("insert node %s: %w", n.ID, err)
		}
	}

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
			continue // skip self-referential edges
		}
		if _, err := edgeStmt.Exec(e.ID, e.SourceID, e.TargetID, e.EdgeType, e.DisplayText, e.Weight); err != nil {
			return fmt.Errorf("insert edge %s->%s: %w", e.SourceID, e.TargetID, err)
		}
	}

	return tx.Commit()
}

// --- Helpers ---

type nodeScanner interface {
	Scan(dest ...any) error
}

func scanOneNode(sc nodeScanner) (models.VaultNode, error) {
	var n models.VaultNode
	var frontmatter, tags, nodeType, createdAt, updatedAt sql.NullString
	err := sc.Scan(&n.ID, &n.FilePath, &n.Title, &n.Content, &frontmatter, &nodeType, &tags, &n.InDegree, &n.OutDegree, &createdAt, &updatedAt)
	if err != nil {
		return n, err
	}
	n.NodeType = nodeType.String
	if frontmatter.Valid {
		json.Unmarshal([]byte(frontmatter.String), &n.Metadata)
	}
	if tags.Valid {
		json.Unmarshal([]byte(tags.String), &n.Tags)
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
