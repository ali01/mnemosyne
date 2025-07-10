// Package postgres implements PostgreSQL-based repository for node positions
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// PositionRepository implements repository.PositionRepository using PostgreSQL without state
type PositionRepository struct {
	// No fields - stateless
}

// NewPositionRepository creates a new stateless PostgreSQL-based position repository
func NewPositionRepository() repository.PositionRepository {
	return &PositionRepository{}
}

// GetByNodeID retrieves the position of a specific node
func (r *PositionRepository) GetByNodeID(exec repository.Executor, ctx context.Context, nodeID string) (*models.NodePosition, error) {
	var position models.NodePosition
	query := `
		SELECT node_id, x, y, z, locked, updated_at
		FROM node_positions
		WHERE node_id = $1
	`

	err := exec.GetContext(ctx, &position, query, nodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &NotFoundError{Resource: "position", ID: nodeID}
		}
		return nil, err
	}

	return &position, nil
}

// Upsert inserts or updates a node position
func (r *PositionRepository) Upsert(exec repository.Executor, ctx context.Context, position *models.NodePosition) error {
	position.UpdatedAt = time.Now()

	query := `
		INSERT INTO node_positions (node_id, x, y, z, locked, updated_at)
		VALUES (:node_id, :x, :y, :z, :locked, :updated_at)
		ON CONFLICT (node_id) DO UPDATE SET
			x = EXCLUDED.x,
			y = EXCLUDED.y,
			z = EXCLUDED.z,
			locked = EXCLUDED.locked,
			updated_at = EXCLUDED.updated_at
	`

	_, err := exec.NamedExecContext(ctx, query, position)
	if err != nil {
		return handlePostgresError(err, "position")
	}

	return nil
}

// UpsertBatch inserts or updates multiple node positions
func (r *PositionRepository) UpsertBatch(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error {
	if len(positions) == 0 {
		return nil
	}

	// Check if we're already in a transaction
	if tx, ok := exec.(*sqlx.Tx); ok {
		// Already in transaction
		return r.upsertBatchInTx(tx, ctx, positions)
	}

	// Not in transaction, need to start one for atomic batch operation
	sqlxDB, ok := exec.(*sqlx.DB)
	if !ok {
		// Fallback to individual upserts
		return r.upsertBatchIndividual(exec, ctx, positions)
	}

	// Use transaction for batch upsert
	return db.WithTransaction(sqlxDB, ctx, func(tx *sqlx.Tx) error {
		return r.upsertBatchInTx(tx, ctx, positions)
	})
}

// upsertBatchInTx performs batch upsert within a transaction
func (r *PositionRepository) upsertBatchInTx(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error {
	query := `
		INSERT INTO node_positions (node_id, x, y, z, locked, updated_at)
		VALUES (:node_id, :x, :y, :z, :locked, :updated_at)
		ON CONFLICT (node_id) DO UPDATE SET
			x = EXCLUDED.x,
			y = EXCLUDED.y,
			z = EXCLUDED.z,
			locked = EXCLUDED.locked,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	for i := range positions {
		positions[i].UpdatedAt = now

		_, err := exec.NamedExecContext(ctx, query, &positions[i])
		if err != nil {
			return handlePostgresError(err, "position")
		}
	}

	return nil
}

// upsertBatchIndividual performs individual upserts (fallback)
func (r *PositionRepository) upsertBatchIndividual(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error {
	for _, position := range positions {
		if err := r.Upsert(exec, ctx, &position); err != nil {
			return fmt.Errorf("failed to upsert position for node %s: %w", position.NodeID, err)
		}
	}
	return nil
}

// GetAll retrieves all node positions
func (r *PositionRepository) GetAll(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error) {
	query := `
		SELECT node_id, x, y, z, locked, updated_at
		FROM node_positions
		ORDER BY node_id
	`

	var positions []models.NodePosition
	err := exec.SelectContext(ctx, &positions, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all positions: %w", err)
	}

	return positions, nil
}

// DeleteByNodeID removes a node position
func (r *PositionRepository) DeleteByNodeID(exec repository.Executor, ctx context.Context, nodeID string) error {
	query := `DELETE FROM node_positions WHERE node_id = $1`

	_, err := exec.ExecContext(ctx, query, nodeID)
	if err != nil {
		return handlePostgresError(err, "position")
	}

	// Delete is idempotent - don't return error if position doesn't exist
	return nil
}
