// Package service implements the business logic layer
package service

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/repository/postgres"
)

// EdgeService handles edge operations
type EdgeService struct {
	db       *sqlx.DB
	edgeRepo repository.EdgeRepository
}

// NewEdgeService creates a new edge service
func NewEdgeService(db *sqlx.DB) *EdgeService {
	return &EdgeService{
		db:       db,
		edgeRepo: postgres.NewEdgeRepository(),
	}
}

// NewEdgeServiceWithRepo creates a new edge service with a custom repository (for testing)
func NewEdgeServiceWithRepo(db *sqlx.DB, repo repository.EdgeRepository) *EdgeService {
	return &EdgeService{
		db:       db,
		edgeRepo: repo,
	}
}

// CreateEdge creates a new edge
func (s *EdgeService) CreateEdge(ctx context.Context, edge *models.VaultEdge) error {
	return s.edgeRepo.Create(s.db, ctx, edge)
}

// GetEdge retrieves an edge by ID
func (s *EdgeService) GetEdge(ctx context.Context, id string) (*models.VaultEdge, error) {
	return s.edgeRepo.GetByID(s.db, ctx, id)
}

// UpdateEdge updates an existing edge by deleting and recreating it
func (s *EdgeService) UpdateEdge(ctx context.Context, edge *models.VaultEdge) error {
	// Since EdgeRepository doesn't have Update, we'll delete and recreate
	return db.WithTransaction(s.db, ctx, func(tx *sqlx.Tx) error {
		if err := s.edgeRepo.Delete(tx, ctx, edge.ID); err != nil {
			return err
		}
		return s.edgeRepo.Create(tx, ctx, edge)
	})
}

// DeleteEdge removes an edge by ID
func (s *EdgeService) DeleteEdge(ctx context.Context, id string) error {
	return s.edgeRepo.Delete(s.db, ctx, id)
}

// CreateEdges creates multiple edges efficiently
func (s *EdgeService) CreateEdges(ctx context.Context, edges []models.VaultEdge) error {
	return s.edgeRepo.CreateBatch(s.db, ctx, edges)
}

// UpsertEdges inserts or updates multiple edges
func (s *EdgeService) UpsertEdges(ctx context.Context, edges []models.VaultEdge) error {
	return s.edgeRepo.UpsertBatch(s.db, ctx, edges)
}

// GetEdgesByNode retrieves all edges connected to a node
func (s *EdgeService) GetEdgesByNode(ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	return s.edgeRepo.GetByNode(s.db, ctx, nodeID)
}

// GetEdgesBetweenNodes retrieves edges between two specific nodes
func (s *EdgeService) GetEdgesBetweenNodes(ctx context.Context, sourceID, targetID string) ([]models.VaultEdge, error) {
	return s.edgeRepo.GetBySourceAndTarget(s.db, ctx, sourceID, targetID)
}

// GetAllEdges retrieves all edges with pagination
func (s *EdgeService) GetAllEdges(ctx context.Context, limit, offset int) ([]models.VaultEdge, error) {
	return s.edgeRepo.GetAll(s.db, ctx, limit, offset)
}

// CountEdges returns the total number of edges
func (s *EdgeService) CountEdges(ctx context.Context) (int64, error) {
	return s.edgeRepo.Count(s.db, ctx)
}

// CreateEdgeBatch creates multiple edges efficiently
func (s *EdgeService) CreateEdgeBatch(ctx context.Context, edges []models.VaultEdge) error {
	return s.edgeRepo.CreateBatch(s.db, ctx, edges)
}

// DeleteNodeEdges removes all edges connected to a specific node
func (s *EdgeService) DeleteNodeEdges(ctx context.Context, nodeID string) error {
	// This would typically be implemented in the repository
	// For now, we'll use a transaction to delete both incoming and outgoing edges
	return db.WithTransaction(s.db, ctx, func(tx *sqlx.Tx) error {
		// Get all edges for the node
		edges, err := s.edgeRepo.GetByNode(tx, ctx, nodeID)
		if err != nil {
			return err
		}
		
		// Delete each edge
		for _, edge := range edges {
			if err := s.edgeRepo.Delete(tx, ctx, edge.ID); err != nil {
				return err
			}
		}
		
		return nil
	})
}

// GetIncomingEdges retrieves all edges pointing to a node
func (s *EdgeService) GetIncomingEdges(ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	return s.edgeRepo.GetIncomingEdges(s.db, ctx, nodeID)
}

// GetOutgoingEdges retrieves all edges originating from a node
func (s *EdgeService) GetOutgoingEdges(ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	return s.edgeRepo.GetOutgoingEdges(s.db, ctx, nodeID)
}

// DeleteAllEdges removes all edges (use with caution)
func (s *EdgeService) DeleteAllEdges(ctx context.Context) error {
	return s.edgeRepo.DeleteAll(s.db, ctx)
}

// CreateNodeWithEdges creates a node and its edges in a transaction
func (s *EdgeService) CreateNodeWithEdges(ctx context.Context, node *models.VaultNode, edges []models.VaultEdge) error {
	return db.WithTransaction(s.db, ctx, func(tx *sqlx.Tx) error {
		// This would need NodeRepository too - just an example of transactional usage
		// In practice, you might want a higher-level service that coordinates multiple services
		return s.edgeRepo.CreateBatch(tx, ctx, edges)
	})
}