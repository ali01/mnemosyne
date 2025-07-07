// Package mock provides mock implementations of services for testing
package mock

import (
	"context"
	"errors"

	"github.com/ali01/mnemosyne/internal/models"
)

// MockNodeService is a mock implementation of NodeService for testing
type MockNodeService struct {
	nodes map[string]*models.VaultNode
}

// NewMockNodeService creates a new mock node service
func NewMockNodeService() *MockNodeService {
	return &MockNodeService{
		nodes: make(map[string]*models.VaultNode),
	}
}

// GetNode retrieves a node by ID
func (s *MockNodeService) GetNode(ctx context.Context, id string) (*models.VaultNode, error) {
	node, exists := s.nodes[id]
	if !exists {
		return nil, errors.New("node not found")
	}
	return node, nil
}

// GetAllNodes retrieves all nodes with pagination
func (s *MockNodeService) GetAllNodes(ctx context.Context, limit, offset int) ([]models.VaultNode, error) {
	var nodes []models.VaultNode
	count := 0
	for _, node := range s.nodes {
		if count >= offset && count < offset+limit {
			nodes = append(nodes, *node)
		}
		count++
	}
	return nodes, nil
}

// SearchNodes performs search on nodes
func (s *MockNodeService) SearchNodes(ctx context.Context, query string) ([]models.VaultNode, error) {
	var results []models.VaultNode
	for _, node := range s.nodes {
		if contains(node.Title, query) || contains(node.Content, query) {
			results = append(results, *node)
		}
	}
	return results, nil
}

// CountNodes returns the total number of nodes
func (s *MockNodeService) CountNodes(ctx context.Context) (int64, error) {
	return int64(len(s.nodes)), nil
}

// AddTestNode adds a node for testing
func (s *MockNodeService) AddTestNode(node *models.VaultNode) {
	s.nodes[node.ID] = node
}

// MockEdgeService is a mock implementation of EdgeService for testing
type MockEdgeService struct {
	edges map[string]*models.VaultEdge
}

// NewMockEdgeService creates a new mock edge service
func NewMockEdgeService() *MockEdgeService {
	return &MockEdgeService{
		edges: make(map[string]*models.VaultEdge),
	}
}

// GetEdgesByNode retrieves edges for a node
func (s *MockEdgeService) GetEdgesByNode(ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	var edges []models.VaultEdge
	for _, edge := range s.edges {
		if edge.SourceID == nodeID || edge.TargetID == nodeID {
			edges = append(edges, *edge)
		}
	}
	return edges, nil
}

// GetAllEdges retrieves all edges with pagination
func (s *MockEdgeService) GetAllEdges(ctx context.Context, limit, offset int) ([]models.VaultEdge, error) {
	var edges []models.VaultEdge
	count := 0
	for _, edge := range s.edges {
		if count >= offset && count < offset+limit {
			edges = append(edges, *edge)
		}
		count++
	}
	return edges, nil
}

// CountEdges returns the total number of edges
func (s *MockEdgeService) CountEdges(ctx context.Context) (int64, error) {
	return int64(len(s.edges)), nil
}

// AddTestEdge adds an edge for testing
func (s *MockEdgeService) AddTestEdge(edge *models.VaultEdge) {
	s.edges[edge.ID] = edge
}

// MockPositionService is a mock implementation of PositionService for testing
type MockPositionService struct {
	positions map[string]*models.NodePosition
}

// NewMockPositionService creates a new mock position service
func NewMockPositionService() *MockPositionService {
	return &MockPositionService{
		positions: make(map[string]*models.NodePosition),
	}
}

// GetPosition retrieves position for a node
func (s *MockPositionService) GetPosition(ctx context.Context, nodeID string) (*models.NodePosition, error) {
	pos, exists := s.positions[nodeID]
	if !exists {
		return nil, errors.New("position not found")
	}
	return pos, nil
}

// UpdatePosition updates position for a node
func (s *MockPositionService) UpdatePosition(ctx context.Context, nodeID string, position *models.NodePosition) error {
	if nodeID == "" {
		return errors.New("node ID is required")
	}
	s.positions[nodeID] = position
	return nil
}

// GetViewportPositions retrieves positions within viewport
func (s *MockPositionService) GetViewportPositions(ctx context.Context, minX, maxX, minY, maxY float64) ([]models.NodePosition, error) {
	var positions []models.NodePosition
	for nodeID, pos := range s.positions {
		if pos.X >= minX && pos.X <= maxX && pos.Y >= minY && pos.Y <= maxY {
			posCopy := *pos
			posCopy.NodeID = nodeID
			positions = append(positions, posCopy)
		}
	}
	return positions, nil
}

// SetTestPosition sets a position for testing
func (s *MockPositionService) SetTestPosition(nodeID string, x, y, z float64, locked bool) {
	s.positions[nodeID] = &models.NodePosition{
		NodeID: nodeID,
		X:      x,
		Y:      y,
		Z:      z,
		Locked: locked,
	}
}

// Helper function for string search
func contains(str, substr string) bool {
	return len(str) > 0 && len(substr) > 0 && 
		(str == substr || len(str) > len(substr) && 
		 (str[:len(substr)] == substr || 
		  contains(str[1:], substr)))
}

// MockParseService is a mock implementation of parse service for testing
type MockParseService struct {
	status string
}

// NewMockParseService creates a new mock parse service
func NewMockParseService() *MockParseService {
	return &MockParseService{
		status: "idle",
	}
}

// ParseVault simulates vault parsing
func (s *MockParseService) ParseVault(ctx context.Context) error {
	s.status = "parsing"
	return nil
}

// GetParseStatus returns parse status
func (s *MockParseService) GetParseStatus(ctx context.Context) (string, error) {
	return s.status, nil
}

// SetTestStatus sets status for testing
func (s *MockParseService) SetTestStatus(status string) {
	s.status = status
}