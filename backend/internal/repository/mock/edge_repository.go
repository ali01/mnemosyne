// Package mock provides in-memory implementations of repository interfaces for testing
package mock

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// EdgeRepository is an in-memory implementation of repository.EdgeRepository
// that follows the stateless pattern
type EdgeRepository struct {
	mu    sync.RWMutex
	edges map[string]*models.VaultEdge
}

// NewEdgeRepository creates a new mock edge repository
func NewEdgeRepository() repository.EdgeRepository {
	return &EdgeRepository{
		edges: make(map[string]*models.VaultEdge),
	}
}

// Create inserts a new edge
func (r *EdgeRepository) Create(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if edge.ID == "" {
		edge.ID = uuid.New().String()
	}

	if _, exists := r.edges[edge.ID]; exists {
		return repository.NewDuplicateKeyError("edge", "id", edge.ID)
	}

	edgeCopy := *edge
	r.edges[edge.ID] = &edgeCopy
	return nil
}

// GetByID retrieves an edge by its ID
func (r *EdgeRepository) GetByID(exec repository.Executor, ctx context.Context, id string) (*models.VaultEdge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	edge, exists := r.edges[id]
	if !exists {
		return nil, repository.NewNotFoundError("edge", id)
	}

	edgeCopy := *edge
	return &edgeCopy, nil
}

// Delete removes an edge by ID
func (r *EdgeRepository) Delete(exec repository.Executor, ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.edges[id]; !exists {
		return repository.NewNotFoundError("edge", id)
	}

	delete(r.edges, id)
	return nil
}

// CreateBatch inserts multiple edges
func (r *EdgeRepository) CreateBatch(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create copies and check for duplicates
	edgeCopies := make([]models.VaultEdge, len(edges))
	for i := range edges {
		edgeCopies[i] = edges[i]
		if edgeCopies[i].ID == "" {
			edgeCopies[i].ID = uuid.New().String()
		}
		if _, exists := r.edges[edgeCopies[i].ID]; exists {
			return repository.NewDuplicateKeyError("edge", "id", edgeCopies[i].ID)
		}
	}

	// Insert all edges
	for i := range edgeCopies {
		r.edges[edgeCopies[i].ID] = &edgeCopies[i]
	}

	return nil
}

// UpsertBatch inserts or updates multiple edges
func (r *EdgeRepository) UpsertBatch(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, edge := range edges {
		edgeCopy := edge
		if edgeCopy.ID == "" {
			edgeCopy.ID = uuid.New().String()
		}
		r.edges[edgeCopy.ID] = &edgeCopy
	}

	return nil
}

// GetByNode retrieves all edges connected to a node
func (r *EdgeRepository) GetByNode(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.VaultEdge, 0)
	for _, edge := range r.edges {
		if edge.SourceID == nodeID || edge.TargetID == nodeID {
			result = append(result, *edge)
		}
	}

	return result, nil
}

// GetBySourceAndTarget retrieves edges between two specific nodes
func (r *EdgeRepository) GetBySourceAndTarget(exec repository.Executor, ctx context.Context, sourceID, targetID string) ([]models.VaultEdge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.VaultEdge, 0)
	for _, edge := range r.edges {
		if edge.SourceID == sourceID && edge.TargetID == targetID {
			result = append(result, *edge)
		}
	}

	return result, nil
}

// GetAll retrieves all edges with pagination
func (r *EdgeRepository) GetAll(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultEdge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect all edges
	allEdges := make([]models.VaultEdge, 0, len(r.edges))
	for _, edge := range r.edges {
		allEdges = append(allEdges, *edge)
	}

	// Apply pagination
	start := offset
	if start > len(allEdges) {
		return []models.VaultEdge{}, nil
	}

	end := start + limit
	if end > len(allEdges) {
		end = len(allEdges)
	}

	return allEdges[start:end], nil
}

// Count returns the total number of edges
func (r *EdgeRepository) Count(exec repository.Executor, ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return int64(len(r.edges)), nil
}

// GetIncomingEdges retrieves all edges pointing to a node
func (r *EdgeRepository) GetIncomingEdges(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.VaultEdge, 0)
	for _, edge := range r.edges {
		if edge.TargetID == nodeID {
			result = append(result, *edge)
		}
	}

	return result, nil
}

// GetOutgoingEdges retrieves all edges originating from a node
func (r *EdgeRepository) GetOutgoingEdges(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.VaultEdge, 0)
	for _, edge := range r.edges {
		if edge.SourceID == nodeID {
			result = append(result, *edge)
		}
	}

	return result, nil
}

// DeleteAll removes all edges
func (r *EdgeRepository) DeleteAll(exec repository.Executor, ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.edges = make(map[string]*models.VaultEdge)
	return nil
}
