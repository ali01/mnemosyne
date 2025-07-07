// Package service implements the business logic layer
package service

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/repository/postgres"
)

// PositionService handles node position operations
type PositionService struct {
	db           *sqlx.DB
	positionRepo repository.PositionRepository
}

// NewPositionService creates a new position service
func NewPositionService(db *sqlx.DB) *PositionService {
	return &PositionService{
		db:           db,
		positionRepo: postgres.NewPositionRepository(),
	}
}

// NewPositionServiceWithRepo creates a new position service with a custom repository (for testing)
func NewPositionServiceWithRepo(db *sqlx.DB, repo repository.PositionRepository) *PositionService {
	return &PositionService{
		db:           db,
		positionRepo: repo,
	}
}

// GetNodePosition retrieves the position of a specific node
func (s *PositionService) GetNodePosition(ctx context.Context, nodeID string) (*models.NodePosition, error) {
	return s.positionRepo.GetByNodeID(s.db, ctx, nodeID)
}

// UpdateNodePosition updates or creates a node position
func (s *PositionService) UpdateNodePosition(ctx context.Context, position *models.NodePosition) error {
	return s.positionRepo.Upsert(s.db, ctx, position)
}

// UpdateNodePositions updates multiple node positions in a batch
func (s *PositionService) UpdateNodePositions(ctx context.Context, positions []models.NodePosition) error {
	return s.positionRepo.UpsertBatch(s.db, ctx, positions)
}

// GetAllPositions retrieves all node positions
func (s *PositionService) GetAllPositions(ctx context.Context) ([]models.NodePosition, error) {
	return s.positionRepo.GetAll(s.db, ctx)
}

// GetViewportPositions retrieves positions within a specified viewport
func (s *PositionService) GetViewportPositions(ctx context.Context, minX, maxX, minY, maxY float64) ([]models.NodePosition, error) {
	// Get all positions and filter in memory
	// In a production system, this would be done in the database for efficiency
	allPositions, err := s.positionRepo.GetAll(s.db, ctx)
	if err != nil {
		return nil, err
	}
	
	viewportPositions := make([]models.NodePosition, 0)
	for _, pos := range allPositions {
		if pos.X >= minX && pos.X <= maxX && pos.Y >= minY && pos.Y <= maxY {
			viewportPositions = append(viewportPositions, pos)
		}
	}
	
	return viewportPositions, nil
}

// DeleteNodePosition removes a node position
func (s *PositionService) DeleteNodePosition(ctx context.Context, nodeID string) error {
	return s.positionRepo.DeleteByNodeID(s.db, ctx, nodeID)
}