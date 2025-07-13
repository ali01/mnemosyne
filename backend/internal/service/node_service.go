// Package service provides business logic layer for the application
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/repository/postgres"
)

// NodeService provides business logic for node operations
type NodeService struct {
	db       *sqlx.DB
	nodeRepo repository.NodeRepository
}

// NewNodeService creates a new node service
func NewNodeService(database *sqlx.DB) *NodeService {
	return &NodeService{
		db:       database,
		nodeRepo: postgres.NewNodeRepository(),
	}
}

// NewNodeServiceWithRepo creates a new node service with a custom repository (for testing)
func NewNodeServiceWithRepo(database *sqlx.DB, repo repository.NodeRepository) *NodeService {
	return &NodeService{
		db:       database,
		nodeRepo: repo,
	}
}

// GetNode retrieves a node by ID
func (s *NodeService) GetNode(ctx context.Context, id string) (*models.VaultNode, error) {
	return s.nodeRepo.GetByID(s.db, ctx, id)
}

// GetNodeByPath retrieves a node by its file path
func (s *NodeService) GetNodeByPath(ctx context.Context, path string) (*models.VaultNode, error) {
	return s.nodeRepo.GetByPath(s.db, ctx, path)
}

// CreateNode creates a new node
func (s *NodeService) CreateNode(ctx context.Context, node *models.VaultNode) error {
	return s.nodeRepo.Create(s.db, ctx, node)
}

// UpdateNode updates an existing node
func (s *NodeService) UpdateNode(ctx context.Context, node *models.VaultNode) error {
	return s.nodeRepo.Update(s.db, ctx, node)
}

// DeleteNode removes a node by ID
func (s *NodeService) DeleteNode(ctx context.Context, id string) error {
	return s.nodeRepo.Delete(s.db, ctx, id)
}

// GetAllNodes retrieves all nodes with pagination
func (s *NodeService) GetAllNodes(ctx context.Context, limit, offset int) ([]models.VaultNode, error) {
	return s.nodeRepo.GetAll(s.db, ctx, limit, offset)
}

// GetNodesByType retrieves all nodes of a specific type
func (s *NodeService) GetNodesByType(ctx context.Context, nodeType string) ([]models.VaultNode, error) {
	return s.nodeRepo.GetByType(s.db, ctx, nodeType)
}

// SearchNodes performs full-text search on nodes
func (s *NodeService) SearchNodes(ctx context.Context, query string) ([]models.VaultNode, error) {
	return s.nodeRepo.Search(s.db, ctx, query)
}

// CountNodes returns the total number of nodes
func (s *NodeService) CountNodes(ctx context.Context) (int64, error) {
	return s.nodeRepo.Count(s.db, ctx)
}

// CreateNodeBatch creates multiple nodes in a single operation
func (s *NodeService) CreateNodeBatch(ctx context.Context, nodes []models.VaultNode) error {
	return s.nodeRepo.CreateBatch(s.db, ctx, nodes)
}

// UpdateNodeAndEdges updates a node and its edges in a transaction
func (s *NodeService) UpdateNodeAndEdges(ctx context.Context, node *models.VaultNode, edges []models.VaultEdge) error {
	// Use transaction for atomic update
	return db.WithTransaction(s.db, ctx, func(tx *sqlx.Tx) error {
		// Update the node
		node.UpdatedAt = time.Now()
		if err := s.nodeRepo.Update(tx, ctx, node); err != nil {
			return fmt.Errorf("failed to update node: %w", err)
		}

		// Update edges if provided
		if len(edges) > 0 {
			edgeRepo := postgres.NewEdgeRepository()
			if err := edgeRepo.UpsertBatch(tx, ctx, edges); err != nil {
				return fmt.Errorf("failed to update edges: %w", err)
			}
		}

		return nil
	})
}

// RebuildNodeGraph rebuilds the entire node graph from scratch
func (s *NodeService) RebuildNodeGraph(ctx context.Context, nodes []models.VaultNode) error {
	// Use transaction for atomic rebuild
	return db.WithTransaction(s.db, ctx, func(tx *sqlx.Tx) error {
		// Delete all existing nodes (edges will be cascade deleted)
		if err := s.nodeRepo.DeleteAll(tx, ctx); err != nil {
			return fmt.Errorf("failed to delete existing nodes: %w", err)
		}

		// Insert new nodes
		if len(nodes) > 0 {
			if err := s.nodeRepo.CreateBatch(tx, ctx, nodes); err != nil {
				return fmt.Errorf("failed to create nodes: %w", err)
			}
		}

		return nil
	})
}

// GetNodeWithEdges retrieves a node with all its edges
func (s *NodeService) GetNodeWithEdges(ctx context.Context, nodeID string) (*models.VaultNode, []models.VaultEdge, error) {
	// Get the node
	node, err := s.nodeRepo.GetByID(s.db, ctx, nodeID)
	if err != nil {
		return nil, nil, err
	}

	// Get edges
	edgeRepo := postgres.NewEdgeRepository()
	edges, err := edgeRepo.GetByNode(s.db, ctx, nodeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get edges: %w", err)
	}

	return node, edges, nil
}
