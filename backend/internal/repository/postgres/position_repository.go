// Package postgres implements PostgreSQL-based repository for node positions
package postgres

import (
	"context"
	"database/sql"
	"time"

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
		return nil, err
	}

	return positions, nil
}

// DeleteByNodeID removes a node position
func (r *PositionRepository) DeleteByNodeID(exec repository.Executor, ctx context.Context, nodeID string) error {
	query := `DELETE FROM node_positions WHERE node_id = $1`

	result, err := exec.ExecContext(ctx, query, nodeID)
	if err != nil {
		return handlePostgresError(err, "position")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return &NotFoundError{Resource: "position", ID: nodeID}
	}

	return nil
}