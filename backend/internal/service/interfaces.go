// Package service defines interfaces for all services
package service

import (
	"context"

	"github.com/ali01/mnemosyne/internal/models"
)

// NodeServiceInterface defines the interface for node operations.
// It provides high-level business logic for managing vault nodes,
// including CRUD operations, searching, and type-based queries.
// The service layer handles coordination between repositories and
// ensures data consistency.
type NodeServiceInterface interface {
	GetNode(ctx context.Context, id string) (*models.VaultNode, error)
	GetNodeByPath(ctx context.Context, path string) (*models.VaultNode, error)
	CreateNode(ctx context.Context, node *models.VaultNode) error
	UpdateNode(ctx context.Context, node *models.VaultNode) error
	DeleteNode(ctx context.Context, id string) error
	GetAllNodes(ctx context.Context, limit, offset int) ([]models.VaultNode, error)
	GetNodesByType(ctx context.Context, nodeType string) ([]models.VaultNode, error)
	SearchNodes(ctx context.Context, query string) ([]models.VaultNode, error)
	CountNodes(ctx context.Context) (int64, error)
}

// EdgeServiceInterface defines the interface for edge operations.
// It manages the relationships between nodes in the knowledge graph,
// providing methods to create, query, and maintain edges. The service
// ensures referential integrity and handles edge-related business logic.
type EdgeServiceInterface interface {
	CreateEdge(ctx context.Context, edge *models.VaultEdge) error
	GetEdge(ctx context.Context, id string) (*models.VaultEdge, error)
	UpdateEdge(ctx context.Context, edge *models.VaultEdge) error
	DeleteEdge(ctx context.Context, id string) error
	GetAllEdges(ctx context.Context, limit, offset int) ([]models.VaultEdge, error)
	GetEdgesByNode(ctx context.Context, nodeID string) ([]models.VaultEdge, error)
	CountEdges(ctx context.Context) (int64, error)
}

// PositionServiceInterface defines the interface for position operations.
// It manages the spatial positioning of nodes in the graph visualization,
// supporting both individual and batch updates. The service enables
// viewport-based queries for efficient rendering of large graphs.
type PositionServiceInterface interface {
	GetNodePosition(ctx context.Context, nodeID string) (*models.NodePosition, error)
	UpdateNodePosition(ctx context.Context, position *models.NodePosition) error
	UpdateNodePositions(ctx context.Context, positions []models.NodePosition) error
	GetAllPositions(ctx context.Context) ([]models.NodePosition, error)
	GetViewportPositions(ctx context.Context, minX, maxX, minY, maxY float64) ([]models.NodePosition, error)
	DeleteNodePosition(ctx context.Context, nodeID string) error
}