// Package mock provides in-memory implementations of repository interfaces for testing
package mock

import (
	"context"
	"sync"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// PositionRepository is an in-memory implementation of repository.PositionRepository
// that follows the stateless pattern
type PositionRepository struct {
	mu        sync.RWMutex
	positions map[string]*models.NodePosition
}

// NewPositionRepository creates a new mock position repository
func NewPositionRepository() repository.PositionRepository {
	return &PositionRepository{
		positions: make(map[string]*models.NodePosition),
	}
}

// GetByNodeID retrieves the position of a specific node
func (r *PositionRepository) GetByNodeID(exec repository.Executor, ctx context.Context, nodeID string) (*models.NodePosition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pos, exists := r.positions[nodeID]
	if !exists {
		return nil, repository.NewNotFoundError("position", nodeID)
	}

	posCopy := *pos
	return &posCopy, nil
}

// Upsert inserts or updates a node position
func (r *PositionRepository) Upsert(exec repository.Executor, ctx context.Context, position *models.NodePosition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	posCopy := *position
	posCopy.UpdatedAt = time.Now()
	r.positions[posCopy.NodeID] = &posCopy
	return nil
}

// UpsertBatch inserts or updates multiple node positions
func (r *PositionRepository) UpsertBatch(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, pos := range positions {
		posCopy := pos
		posCopy.UpdatedAt = now
		r.positions[posCopy.NodeID] = &posCopy
	}

	return nil
}

// GetAll retrieves all node positions
func (r *PositionRepository) GetAll(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.NodePosition, 0, len(r.positions))
	for _, pos := range r.positions {
		result = append(result, *pos)
	}

	return result, nil
}

// DeleteByNodeID removes a node position
func (r *PositionRepository) DeleteByNodeID(exec repository.Executor, ctx context.Context, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.positions[nodeID]; !exists {
		return repository.NewNotFoundError("position", nodeID)
	}

	delete(r.positions, nodeID)
	return nil
}
