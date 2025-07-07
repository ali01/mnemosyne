// Package postgres implements PostgreSQL-based repository for edges
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

// EdgeRepository implements repository.EdgeRepository using PostgreSQL
// This implementation is stateless - all database operations receive the executor as a parameter
type EdgeRepository struct {
	// No fields - stateless
}

// NewEdgeRepository creates a new PostgreSQL-based edge repository
func NewEdgeRepository() repository.EdgeRepository {
	return &EdgeRepository{}
}

// Create inserts a new edge into the database
func (r *EdgeRepository) Create(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error {
	if edge.ID == "" {
		edge.ID = uuid.New().String()
	}

	edge.CreatedAt = time.Now()

	query := `
		INSERT INTO edges (id, source_id, target_id, edge_type, display_text, weight, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := exec.ExecContext(ctx, query, edge.ID, edge.SourceID, edge.TargetID,
		edge.EdgeType, edge.DisplayText, edge.Weight, edge.CreatedAt)
	if err != nil {
		return handlePostgresError(err, "edge")
	}

	return nil
}

// GetByID retrieves an edge by its ID
func (r *EdgeRepository) GetByID(exec repository.Executor, ctx context.Context, id string) (*models.VaultEdge, error) {
	var edge models.VaultEdge
	query := `SELECT * FROM edges WHERE id = $1`

	err := exec.GetContext(ctx, &edge, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &NotFoundError{Resource: "edge", ID: id}
		}
		return nil, fmt.Errorf("failed to get edge by ID: %w", err)
	}

	return &edge, nil
}

// Update updates an existing edge
func (r *EdgeRepository) Update(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error {
	query := `
		UPDATE edges
		SET source_id = $2, target_id = $3, edge_type = $4,
		    display_text = $5, weight = $6
		WHERE id = $1
	`

	result, err := exec.ExecContext(ctx, query, edge.ID, edge.SourceID, edge.TargetID,
		edge.EdgeType, edge.DisplayText, edge.Weight)
	if err != nil {
		return handlePostgresError(err, "edge")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return &NotFoundError{Resource: "edge", ID: edge.ID}
	}

	return nil
}

// Delete removes an edge by ID
func (r *EdgeRepository) Delete(exec repository.Executor, ctx context.Context, id string) error {
	query := `DELETE FROM edges WHERE id = $1`

	result, err := exec.ExecContext(ctx, query, id)
	if err != nil {
		return handlePostgresError(err, "edge")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return &NotFoundError{Resource: "edge", ID: id}
	}

	return nil
}

// CreateBatch inserts multiple edges efficiently
func (r *EdgeRepository) CreateBatch(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	if len(edges) == 0 {
		return nil
	}

	// Check if we're already in a transaction
	if tx, ok := exec.(*sqlx.Tx); ok {
		// Already in transaction, use COPY for best performance
		return r.createBatchWithCopy(ctx, tx, edges)
	}

	// Not in transaction, need to start one for batch operation
	sqlxDB, ok := exec.(*sqlx.DB)
	if !ok {
		// Fallback to individual inserts
		return r.createBatchIndividual(exec, ctx, edges)
	}

	// Use transaction for batch insert
	return db.WithTransaction(sqlxDB, ctx, func(tx *sqlx.Tx) error {
		return r.createBatchWithCopy(ctx, tx, edges)
	})
}

// createBatchWithCopy uses PostgreSQL COPY for efficient batch insert
func (r *EdgeRepository) createBatchWithCopy(ctx context.Context, tx *sqlx.Tx, edges []models.VaultEdge) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("edges",
		"id", "source_id", "target_id", "edge_type", "display_text", "weight", "created_at"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, edge := range edges {
		if edge.ID == "" {
			edge.ID = uuid.New().String()
		}
		edge.CreatedAt = now

		_, err = stmt.Exec(edge.ID, edge.SourceID, edge.TargetID,
			edge.EdgeType, edge.DisplayText, edge.Weight, edge.CreatedAt)
		if err != nil {
			return handlePostgresError(err, "edge")
		}
	}

	_, err = stmt.Exec()
	return err
}

// createBatchIndividual inserts edges one by one (fallback)
func (r *EdgeRepository) createBatchIndividual(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	for _, edge := range edges {
		if err := r.Create(exec, ctx, &edge); err != nil {
			return fmt.Errorf("failed to create edge %s: %w", edge.ID, err)
		}
	}
	return nil
}

// UpsertBatch inserts or updates multiple edges
func (r *EdgeRepository) UpsertBatch(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	if len(edges) == 0 {
		return nil
	}

	// Check if we're already in a transaction
	if tx, ok := exec.(*sqlx.Tx); ok {
		// Already in transaction
		return r.upsertBatchInTx(repository.Executor(tx), ctx, edges)
	}

	// Not in transaction, need to start one
	sqlxDB, ok := exec.(*sqlx.DB)
	if !ok {
		// Fallback to individual upserts
		return r.upsertBatchIndividual(exec, ctx, edges)
	}

	return db.WithTransaction(sqlxDB, ctx, func(tx *sqlx.Tx) error {
		return r.upsertBatchInTx(tx, ctx, edges)
	})
}

// upsertBatchInTx performs batch upsert within a transaction
func (r *EdgeRepository) upsertBatchInTx(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	query := `
		INSERT INTO edges (id, source_id, target_id, edge_type, display_text, weight, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			source_id = EXCLUDED.source_id,
			target_id = EXCLUDED.target_id,
			edge_type = EXCLUDED.edge_type,
			display_text = EXCLUDED.display_text,
			weight = EXCLUDED.weight
	`

	now := time.Now()
	for _, edge := range edges {
		if edge.ID == "" {
			edge.ID = uuid.New().String()
		}
		if edge.CreatedAt.IsZero() {
			edge.CreatedAt = now
		}

		_, err := exec.ExecContext(ctx, query, edge.ID, edge.SourceID, edge.TargetID,
			edge.EdgeType, edge.DisplayText, edge.Weight, edge.CreatedAt)
		if err != nil {
			return handlePostgresError(err, "edge")
		}
	}

	return nil
}

// upsertBatchIndividual performs individual upserts (fallback)
func (r *EdgeRepository) upsertBatchIndividual(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	for _, edge := range edges {
		_, err := r.GetByID(exec, ctx, edge.ID)
		if err != nil {
			if IsNotFound(err) {
				// Edge doesn't exist, create it
				if err := r.Create(exec, ctx, &edge); err != nil {
					return fmt.Errorf("failed to create edge %s: %w", edge.ID, err)
				}
			} else {
				return fmt.Errorf("failed to check edge %s: %w", edge.ID, err)
			}
		} else {
			// Edge exists, update it
			if err := r.Update(exec, ctx, &edge); err != nil {
				return fmt.Errorf("failed to update edge %s: %w", edge.ID, err)
			}
		}
	}
	return nil
}

// GetByNode retrieves all edges connected to a node
func (r *EdgeRepository) GetByNode(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	var edges []models.VaultEdge
	// Use UNION to leverage both indexes efficiently
	query := `
		SELECT * FROM edges WHERE source_id = $1
		UNION
		SELECT * FROM edges WHERE target_id = $2
	`

	err := exec.SelectContext(ctx, &edges, query, nodeID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges by node: %w", err)
	}

	return edges, nil
}

// GetBySourceAndTarget retrieves edges between two specific nodes
func (r *EdgeRepository) GetBySourceAndTarget(exec repository.Executor, ctx context.Context, sourceID, targetID string) ([]models.VaultEdge, error) {
	var edges []models.VaultEdge
	query := `SELECT * FROM edges WHERE source_id = $1 AND target_id = $2`

	err := exec.SelectContext(ctx, &edges, query, sourceID, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges by source and target: %w", err)
	}

	return edges, nil
}

// GetAll retrieves all edges with pagination
func (r *EdgeRepository) GetAll(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultEdge, error) {
	var edges []models.VaultEdge
	query := `SELECT * FROM edges ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	err := exec.SelectContext(ctx, &edges, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get all edges: %w", err)
	}

	return edges, nil
}

// Count returns the total number of edges
func (r *EdgeRepository) Count(exec repository.Executor, ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM edges`

	row := exec.QueryRowContext(ctx, query)
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count edges: %w", err)
	}

	return count, nil
}

// GetIncomingEdges retrieves all edges pointing to a node
func (r *EdgeRepository) GetIncomingEdges(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	var edges []models.VaultEdge
	query := `SELECT * FROM edges WHERE target_id = $1`

	err := exec.SelectContext(ctx, &edges, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming edges: %w", err)
	}

	return edges, nil
}

// GetOutgoingEdges retrieves all edges originating from a node
func (r *EdgeRepository) GetOutgoingEdges(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	var edges []models.VaultEdge
	query := `SELECT * FROM edges WHERE source_id = $1`

	err := exec.SelectContext(ctx, &edges, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing edges: %w", err)
	}

	return edges, nil
}

// DeleteAll removes all edges
func (r *EdgeRepository) DeleteAll(exec repository.Executor, ctx context.Context) error {
	query := `DELETE FROM edges`

	_, err := exec.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete all edges: %w", err)
	}

	return nil
}
