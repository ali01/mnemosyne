// Package mock provides in-memory implementations of repository interfaces for testing
package mock

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// NodeRepository is an in-memory implementation of repository.NodeRepository
// that follows the stateless pattern
type NodeRepository struct {
	mu    sync.RWMutex
	nodes map[string]*models.VaultNode
}

// NewNodeRepository creates a new mock node repository
func NewNodeRepository() repository.NodeRepository {
	return &NodeRepository{
		nodes: make(map[string]*models.VaultNode),
	}
}

// Create inserts a new node
func (r *NodeRepository) Create(exec repository.Executor, ctx context.Context, node *models.VaultNode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a copy to avoid mutations
	nodeCopy := *node
	if nodeCopy.ID == "" {
		nodeCopy.ID = uuid.New().String()
	}

	if _, exists := r.nodes[nodeCopy.ID]; exists {
		return repository.NewDuplicateKeyError("node", "id", nodeCopy.ID)
	}

	r.nodes[nodeCopy.ID] = &nodeCopy

	// Update the input node ID to match real DB behavior
	node.ID = nodeCopy.ID

	return nil
}

// GetByID retrieves a node by its ID
func (r *NodeRepository) GetByID(exec repository.Executor, ctx context.Context, id string) (*models.VaultNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	node, exists := r.nodes[id]
	if !exists {
		return nil, repository.NewNotFoundError("node", id)
	}

	// Return a copy to avoid mutations
	nodeCopy := *node
	return &nodeCopy, nil
}

// Update updates an existing node
func (r *NodeRepository) Update(exec repository.Executor, ctx context.Context, node *models.VaultNode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[node.ID]; !exists {
		return repository.NewNotFoundError("node", node.ID)
	}

	// Update with a copy
	nodeCopy := *node
	r.nodes[node.ID] = &nodeCopy

	return nil
}

// Delete removes a node by ID
func (r *NodeRepository) Delete(exec repository.Executor, ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[id]; !exists {
		return repository.NewNotFoundError("node", id)
	}

	delete(r.nodes, id)
	return nil
}

// CreateBatch inserts multiple nodes
func (r *NodeRepository) CreateBatch(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create copies and check for duplicates
	nodeCopies := make([]models.VaultNode, len(nodes))
	for i := range nodes {
		nodeCopies[i] = nodes[i]
		if nodeCopies[i].ID == "" {
			nodeCopies[i].ID = uuid.New().String()
		}
		if _, exists := r.nodes[nodeCopies[i].ID]; exists {
			return repository.NewDuplicateKeyError("node", "id", nodeCopies[i].ID)
		}
	}

	// Insert all nodes
	for i := range nodeCopies {
		r.nodes[nodeCopies[i].ID] = &nodeCopies[i]
		// Update original slice to match DB behavior
		nodes[i].ID = nodeCopies[i].ID
	}

	return nil
}

// GetByIDs retrieves multiple nodes by their IDs
func (r *NodeRepository) GetByIDs(exec repository.Executor, ctx context.Context, ids []string) ([]models.VaultNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.VaultNode, 0, len(ids))
	for _, id := range ids {
		if node, exists := r.nodes[id]; exists {
			result = append(result, *node)
		}
	}

	return result, nil
}

// GetByPath retrieves a node by its file path
func (r *NodeRepository) GetByPath(exec repository.Executor, ctx context.Context, path string) (*models.VaultNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, node := range r.nodes {
		if node.FilePath == path {
			nodeCopy := *node
			return &nodeCopy, nil
		}
	}

	return nil, repository.NewNotFoundError("node", path)
}

// GetAll retrieves all nodes with pagination
func (r *NodeRepository) GetAll(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect all nodes
	allNodes := make([]models.VaultNode, 0, len(r.nodes))
	for _, node := range r.nodes {
		allNodes = append(allNodes, *node)
	}

	// Apply pagination
	start := offset
	if start > len(allNodes) {
		return []models.VaultNode{}, nil
	}

	end := start + limit
	if end > len(allNodes) {
		end = len(allNodes)
	}

	return allNodes[start:end], nil
}

// Count returns the total number of nodes
func (r *NodeRepository) Count(exec repository.Executor, ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return int64(len(r.nodes)), nil
}

// Search performs full-text search on nodes
func (r *NodeRepository) Search(exec repository.Executor, ctx context.Context, query string) ([]models.VaultNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	result := make([]models.VaultNode, 0)

	for _, node := range r.nodes {
		if strings.Contains(strings.ToLower(node.Title), query) ||
			strings.Contains(strings.ToLower(node.Content), query) {
			result = append(result, *node)
		}
	}

	return result, nil
}

// GetByType retrieves all nodes of a specific type
func (r *NodeRepository) GetByType(exec repository.Executor, ctx context.Context, nodeType string) ([]models.VaultNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.VaultNode, 0)
	for _, node := range r.nodes {
		if node.NodeType == nodeType {
			result = append(result, *node)
		}
	}

	return result, nil
}

// DeleteAll removes all nodes
func (r *NodeRepository) DeleteAll(exec repository.Executor, ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nodes = make(map[string]*models.VaultNode)
	return nil
}

// GetPathMap returns a map of normalized paths to node IDs
func (r *NodeRepository) GetPathMap(exec repository.Executor, ctx context.Context) (map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pathMap := make(map[string]string)
	for _, node := range r.nodes {
		normalizedPath := strings.ToLower(strings.TrimSuffix(node.FilePath, ".md"))
		pathMap[normalizedPath] = node.ID
	}

	return pathMap, nil
}

// UpdateMetrics updates node graph metrics
func (r *NodeRepository) UpdateMetrics(exec repository.Executor, ctx context.Context, id string, inDegree, outDegree int, centrality float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	node, exists := r.nodes[id]
	if !exists {
		return repository.NewNotFoundError("node", id)
	}

	node.InDegree = inDegree
	node.OutDegree = outDegree
	node.Centrality = centrality

	return nil
}

// UpsertBatch inserts or updates multiple nodes
func (r *NodeRepository) UpsertBatch(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range nodes {
		nodeCopy := nodes[i]
		if nodeCopy.ID == "" {
			nodeCopy.ID = uuid.New().String()
			// Update original slice to match DB behavior
			nodes[i].ID = nodeCopy.ID
		}
		r.nodes[nodeCopy.ID] = &nodeCopy
	}

	return nil
}
