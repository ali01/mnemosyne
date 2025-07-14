// Package postgres implements PostgreSQL-based repository for nodes
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// NodeRepository implements repository.NodeRepository using PostgreSQL
// This implementation is stateless - all database operations receive the executor as a parameter
type NodeRepository struct {
	// No fields - stateless
}

// NewNodeRepository creates a new PostgreSQL-based node repository
func NewNodeRepository() repository.NodeRepository {
	return &NodeRepository{}
}


// Create inserts a new node into the database
func (r *NodeRepository) Create(exec repository.Executor, ctx context.Context, node *models.VaultNode) error {
	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	now := time.Now()
	node.CreatedAt = now
	node.UpdatedAt = now

	query := `
		INSERT INTO nodes (id, title, node_type, tags, content, frontmatter, file_path,
		                  in_degree, out_degree, centrality, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := exec.ExecContext(ctx, query,
		node.ID, node.Title, node.NodeType, node.Tags, node.Content,
		ensureMetadata(node.Metadata), node.FilePath, node.InDegree, node.OutDegree,
		node.Centrality, node.CreatedAt, node.UpdatedAt,
	)
	if err != nil {
		return handlePostgresError(err, "node")
	}

	return nil
}

// GetByID retrieves a node by its ID
func (r *NodeRepository) GetByID(exec repository.Executor, ctx context.Context, id string) (*models.VaultNode, error) {
	var node models.VaultNode
	query := `SELECT id, title, COALESCE(node_type, '') AS node_type, tags, COALESCE(content, '') AS content, frontmatter AS metadata, file_path,
	          in_degree, out_degree, centrality, created_at, updated_at
	          FROM nodes WHERE id = $1`

	err := exec.GetContext(ctx, &node, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &NotFoundError{Resource: "node", ID: id}
		}
		return nil, fmt.Errorf("failed to get node by ID: %w", err)
	}

	return &node, nil
}

// Update updates an existing node
func (r *NodeRepository) Update(exec repository.Executor, ctx context.Context, node *models.VaultNode) error {
	node.UpdatedAt = time.Now()

	query := `
		UPDATE nodes
		SET title = $2, node_type = NULLIF($3, ''), tags = $4, content = NULLIF($5, ''),
		    frontmatter = $6, file_path = $7, in_degree = $8,
		    out_degree = $9, centrality = $10, updated_at = $11
		WHERE id = $1
	`

	result, err := exec.ExecContext(ctx, query,
		node.ID, node.Title, node.NodeType, node.Tags, node.Content,
		ensureMetadata(node.Metadata), node.FilePath, node.InDegree, node.OutDegree,
		node.Centrality, node.UpdatedAt,
	)
	if err != nil {
		return handlePostgresError(err, "node")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return &NotFoundError{Resource: "node", ID: node.ID}
	}

	return nil
}

// Delete removes a node by ID
func (r *NodeRepository) Delete(exec repository.Executor, ctx context.Context, id string) error {
	query := `DELETE FROM nodes WHERE id = $1`

	_, err := exec.ExecContext(ctx, query, id)
	if err != nil {
		return handlePostgresError(err, "node")
	}

	// Delete is idempotent - don't return error if node doesn't exist
	return nil
}

// CreateBatch inserts multiple nodes efficiently
func (r *NodeRepository) CreateBatch(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	if len(nodes) == 0 {
		return nil
	}

	// Check if we're already in a transaction
	if tx, ok := exec.(*sqlx.Tx); ok {
		// Already in transaction, use COPY for best performance
		return r.createBatchWithCopy(ctx, tx, nodes)
	}

	// Not in transaction, need to start one for batch operation
	sqlxDB, ok := exec.(*sqlx.DB)
	if !ok {
		// Fallback to individual inserts if we can't determine the type
		return r.createBatchIndividual(exec, ctx, nodes)
	}

	// Use transaction for batch insert
	return db.WithTransaction(sqlxDB, ctx, func(tx *sqlx.Tx) error {
		return r.createBatchWithCopy(ctx, tx, nodes)
	})
}

// createBatchWithCopy uses PostgreSQL COPY for efficient batch insert
func (r *NodeRepository) createBatchWithCopy(ctx context.Context, tx *sqlx.Tx, nodes []models.VaultNode) error {
	// Validate all nodes before starting the batch operation
	for i, node := range nodes {
		if err := r.validateNode(&node); err != nil {
			return fmt.Errorf("validation failed for node at index %d: %w", i, err)
		}
	}

	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("nodes",
		"id", "title", "node_type", "tags", "content", "frontmatter", "file_path",
		"in_degree", "out_degree", "centrality", "created_at", "updated_at"))
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, node := range nodes {
		if node.ID == "" {
			node.ID = uuid.New().String()
		}
		node.CreatedAt = now
		node.UpdatedAt = now

		// Ensure metadata is not nil for JSON column
		metadata := ensureMetadata(node.Metadata)
		// Convert metadata to JSON string for COPY
		metadataJSON, err := metadata.Value()
		if err != nil {
			return fmt.Errorf("failed to serialize metadata for node %s: %w", node.ID, err)
		}
		// COPY expects string, not []byte
		metadataStr := string(metadataJSON.([]byte))

		_, err = stmt.Exec(
			node.ID, node.Title, node.NodeType, node.Tags,
			node.Content, metadataStr, node.FilePath,
			node.InDegree, node.OutDegree, node.Centrality,
			node.CreatedAt, node.UpdatedAt,
		)
		if err != nil {
			return handlePostgresError(err, "node")
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}
	return nil
}

// createBatchIndividual inserts nodes one by one (fallback)
func (r *NodeRepository) createBatchIndividual(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	for _, node := range nodes {
		if err := r.Create(exec, ctx, &node); err != nil {
			return fmt.Errorf("failed to create node %s: %w", node.ID, err)
		}
	}
	return nil
}

// UpsertBatch inserts or updates multiple nodes
func (r *NodeRepository) UpsertBatch(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	if len(nodes) == 0 {
		return nil
	}

	// Check if we're already in a transaction
	if tx, ok := exec.(*sqlx.Tx); ok {
		// Already in transaction
		return r.upsertBatchInTx(repository.Executor(tx), ctx, nodes)
	}

	// Not in transaction, need to start one
	sqlxDB, ok := exec.(*sqlx.DB)
	if !ok {
		// Fallback to individual upserts
		return r.upsertBatchIndividual(exec, ctx, nodes)
	}

	return db.WithTransaction(sqlxDB, ctx, func(tx *sqlx.Tx) error {
		return r.upsertBatchInTx(tx, ctx, nodes)
	})
}

// upsertBatchInTx performs batch upsert within a transaction
func (r *NodeRepository) upsertBatchInTx(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	query := `
		INSERT INTO nodes (id, title, node_type, tags, content, frontmatter, file_path,
		                  in_degree, out_degree, centrality, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			node_type = EXCLUDED.node_type,
			tags = EXCLUDED.tags,
			content = EXCLUDED.content,
			frontmatter = EXCLUDED.frontmatter,
			file_path = EXCLUDED.file_path,
			in_degree = EXCLUDED.in_degree,
			out_degree = EXCLUDED.out_degree,
			centrality = EXCLUDED.centrality,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	for _, node := range nodes {
		if node.ID == "" {
			node.ID = uuid.New().String()
		}
		if node.CreatedAt.IsZero() {
			node.CreatedAt = now
		}
		node.UpdatedAt = now

		_, err := exec.ExecContext(ctx, query,
			node.ID, node.Title, node.NodeType, node.Tags, node.Content,
			ensureMetadata(node.Metadata), node.FilePath, node.InDegree, node.OutDegree,
			node.Centrality, node.CreatedAt, node.UpdatedAt,
		)
		if err != nil {
			return handlePostgresError(err, "node")
		}
	}

	return nil
}

// upsertBatchIndividual performs individual upserts (fallback)
func (r *NodeRepository) upsertBatchIndividual(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	for _, node := range nodes {
		// Try update first
		err := r.Update(exec, ctx, &node)
		if err != nil {
			if IsNotFound(err) {
				// Node doesn't exist, create it
				if err := r.Create(exec, ctx, &node); err != nil {
					return fmt.Errorf("failed to create node %s: %w", node.ID, err)
				}
			} else {
				return fmt.Errorf("failed to update node %s: %w", node.ID, err)
			}
		}
	}
	return nil
}

// GetAll retrieves all nodes with pagination
func (r *NodeRepository) GetAll(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultNode, error) {
	var nodes []models.VaultNode
	query := `SELECT id, title, COALESCE(node_type, '') AS node_type, tags, COALESCE(content, '') AS content, frontmatter AS metadata, file_path,
	          in_degree, out_degree, centrality, created_at, updated_at
	          FROM nodes ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	err := exec.SelectContext(ctx, &nodes, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get all nodes: %w", err)
	}

	return nodes, nil
}

// GetByIDs retrieves nodes by their IDs
func (r *NodeRepository) GetByIDs(exec repository.Executor, ctx context.Context, ids []string) ([]models.VaultNode, error) {
	if len(ids) == 0 {
		return []models.VaultNode{}, nil
	}

	var nodes []models.VaultNode
	query := `SELECT id, title, COALESCE(node_type, '') AS node_type, tags, COALESCE(content, '') AS content, frontmatter AS metadata, file_path,
	          in_degree, out_degree, centrality, created_at, updated_at
	          FROM nodes WHERE id = ANY($1)`

	err := exec.SelectContext(ctx, &nodes, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes by IDs: %w", err)
	}

	return nodes, nil
}

// GetByType retrieves all nodes of a specific type
func (r *NodeRepository) GetByType(exec repository.Executor, ctx context.Context, nodeType string) ([]models.VaultNode, error) {
	var nodes []models.VaultNode
	query := `SELECT id, title, COALESCE(node_type, '') AS node_type, tags, COALESCE(content, '') AS content, frontmatter AS metadata, file_path,
	          in_degree, out_degree, centrality, created_at, updated_at
	          FROM nodes WHERE node_type = $1 ORDER BY created_at DESC`

	err := exec.SelectContext(ctx, &nodes, query, nodeType)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes by type: %w", err)
	}

	return nodes, nil
}

// GetByPath retrieves a node by its file path
func (r *NodeRepository) GetByPath(exec repository.Executor, ctx context.Context, path string) (*models.VaultNode, error) {
	var node models.VaultNode
	query := `SELECT id, title, COALESCE(node_type, '') AS node_type, tags, COALESCE(content, '') AS content, frontmatter AS metadata, file_path,
	          in_degree, out_degree, centrality, created_at, updated_at
	          FROM nodes WHERE file_path = $1`

	err := exec.GetContext(ctx, &node, query, path)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &NotFoundError{Resource: "node", ID: path}
		}
		return nil, fmt.Errorf("failed to get node by path: %w", err)
	}

	return &node, nil
}

// Search performs full-text search on nodes
func (r *NodeRepository) Search(exec repository.Executor, ctx context.Context, query string) ([]models.VaultNode, error) {
	if query == "" {
		return []models.VaultNode{}, nil
	}

	var nodes []models.VaultNode
	searchQuery := `
		SELECT id, title, COALESCE(node_type, '') AS node_type, tags, COALESCE(content, '') AS content, frontmatter AS metadata, file_path,
		       in_degree, out_degree, centrality, created_at, updated_at
		FROM nodes
		WHERE search_vector @@ plainto_tsquery('english', $1)
		ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC
		LIMIT 100
	`

	err := exec.SelectContext(ctx, &nodes, searchQuery, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes: %w", err)
	}

	return nodes, nil
}

// Count returns the total number of nodes
func (r *NodeRepository) Count(exec repository.Executor, ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM nodes`

	row := exec.QueryRowContext(ctx, query)
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count nodes: %w", err)
	}

	return count, nil
}

// DeleteAll removes all nodes
func (r *NodeRepository) DeleteAll(exec repository.Executor, ctx context.Context) error {
	query := `DELETE FROM nodes`

	_, err := exec.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete all nodes: %w", err)
	}

	return nil
}

// validateNode checks that required fields are present
func (r *NodeRepository) validateNode(node *models.VaultNode) error {
	if node.Title == "" {
		return fmt.Errorf("node title is required")
	}
	if node.FilePath == "" {
		return fmt.Errorf("node file path is required")
	}
	// ID can be empty as it will be generated if not provided
	return nil
}

// ensureMetadata returns the node's metadata or an empty map if nil
func ensureMetadata(metadata models.JSONMetadata) models.JSONMetadata {
	if metadata == nil {
		return make(models.JSONMetadata)
	}
	return metadata
}
