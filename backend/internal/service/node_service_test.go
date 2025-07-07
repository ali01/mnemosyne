package service_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/service"
)

// mockNodeRepository implements repository.NodeRepository for testing
type mockNodeRepository struct {
	getByIDFunc      func(exec repository.Executor, ctx context.Context, id string) (*models.VaultNode, error)
	getByIDsFunc     func(exec repository.Executor, ctx context.Context, ids []string) ([]models.VaultNode, error)
	getByPathFunc    func(exec repository.Executor, ctx context.Context, path string) (*models.VaultNode, error)
	createFunc       func(exec repository.Executor, ctx context.Context, node *models.VaultNode) error
	updateFunc       func(exec repository.Executor, ctx context.Context, node *models.VaultNode) error
	deleteFunc       func(exec repository.Executor, ctx context.Context, id string) error
	getAllFunc       func(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultNode, error)
	getByTypeFunc    func(exec repository.Executor, ctx context.Context, nodeType string) ([]models.VaultNode, error)
	searchFunc       func(exec repository.Executor, ctx context.Context, query string) ([]models.VaultNode, error)
	countFunc        func(exec repository.Executor, ctx context.Context) (int64, error)
	createBatchFunc  func(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error
	upsertBatchFunc  func(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error
	deleteAllFunc    func(exec repository.Executor, ctx context.Context) error
}

func (m *mockNodeRepository) GetByID(exec repository.Executor, ctx context.Context, id string) (*models.VaultNode, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(exec, ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockNodeRepository) GetByIDs(exec repository.Executor, ctx context.Context, ids []string) ([]models.VaultNode, error) {
	if m.getByIDsFunc != nil {
		return m.getByIDsFunc(exec, ctx, ids)
	}
	return nil, errors.New("not implemented")
}

func (m *mockNodeRepository) GetByPath(exec repository.Executor, ctx context.Context, path string) (*models.VaultNode, error) {
	if m.getByPathFunc != nil {
		return m.getByPathFunc(exec, ctx, path)
	}
	return nil, errors.New("not implemented")
}

func (m *mockNodeRepository) Create(exec repository.Executor, ctx context.Context, node *models.VaultNode) error {
	if m.createFunc != nil {
		return m.createFunc(exec, ctx, node)
	}
	return errors.New("not implemented")
}

func (m *mockNodeRepository) Update(exec repository.Executor, ctx context.Context, node *models.VaultNode) error {
	if m.updateFunc != nil {
		return m.updateFunc(exec, ctx, node)
	}
	return errors.New("not implemented")
}

func (m *mockNodeRepository) Delete(exec repository.Executor, ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(exec, ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockNodeRepository) GetAll(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultNode, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(exec, ctx, limit, offset)
	}
	return nil, errors.New("not implemented")
}

func (m *mockNodeRepository) GetByType(exec repository.Executor, ctx context.Context, nodeType string) ([]models.VaultNode, error) {
	if m.getByTypeFunc != nil {
		return m.getByTypeFunc(exec, ctx, nodeType)
	}
	return nil, errors.New("not implemented")
}

func (m *mockNodeRepository) Search(exec repository.Executor, ctx context.Context, query string) ([]models.VaultNode, error) {
	if m.searchFunc != nil {
		return m.searchFunc(exec, ctx, query)
	}
	return nil, errors.New("not implemented")
}

func (m *mockNodeRepository) Count(exec repository.Executor, ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(exec, ctx)
	}
	return 0, errors.New("not implemented")
}

func (m *mockNodeRepository) CreateBatch(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	if m.createBatchFunc != nil {
		return m.createBatchFunc(exec, ctx, nodes)
	}
	return errors.New("not implemented")
}

func (m *mockNodeRepository) UpsertBatch(exec repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	if m.upsertBatchFunc != nil {
		return m.upsertBatchFunc(exec, ctx, nodes)
	}
	return errors.New("not implemented")
}

func (m *mockNodeRepository) DeleteAll(exec repository.Executor, ctx context.Context) error {
	if m.deleteAllFunc != nil {
		return m.deleteAllFunc(exec, ctx)
	}
	return errors.New("not implemented")
}


// TestNodeService_GetNode tests the GetNode method
func TestNodeService_GetNode(t *testing.T) {
	ctx := context.Background()
	
	testNode := &models.VaultNode{
		ID:       "test-123",
		Title:    "Test Node",
		FilePath: "/test/node.md",
		NodeType: "note",
		Content:  "Test content",
	}
	
	tests := []struct {
		name     string
		nodeID   string
		mockFunc func(exec repository.Executor, ctx context.Context, id string) (*models.VaultNode, error)
		wantNode *models.VaultNode
		wantErr  bool
	}{
		{
			name:   "successful retrieval",
			nodeID: "test-123",
			mockFunc: func(exec repository.Executor, ctx context.Context, id string) (*models.VaultNode, error) {
				return testNode, nil
			},
			wantNode: testNode,
			wantErr:  false,
		},
		{
			name:   "node not found",
			nodeID: "nonexistent",
			mockFunc: func(exec repository.Executor, ctx context.Context, id string) (*models.VaultNode, error) {
				return nil, sql.ErrNoRows
			},
			wantNode: nil,
			wantErr:  true,
		},
		{
			name:   "database error",
			nodeID: "test-123",
			mockFunc: func(exec repository.Executor, ctx context.Context, id string) (*models.VaultNode, error) {
				return nil, errors.New("database connection error")
			},
			wantNode: nil,
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := &mockNodeRepository{
				getByIDFunc: tt.mockFunc,
			}
			
			// Create service with mock
			svc := service.NewNodeServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			// Test GetNode
			node, err := svc.GetNode(ctx, tt.nodeID)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantNode, node)
			}
		})
	}
}

// TestNodeService_CreateNode tests the CreateNode method
func TestNodeService_CreateNode(t *testing.T) {
	ctx := context.Background()
	
	testNode := &models.VaultNode{
		ID:       "new-node",
		Title:    "New Node",
		FilePath: "/new/node.md",
		NodeType: "note",
		Content:  "New content",
	}
	
	tests := []struct {
		name     string
		node     *models.VaultNode
		mockFunc func(exec repository.Executor, ctx context.Context, node *models.VaultNode) error
		wantErr  bool
	}{
		{
			name: "successful creation",
			node: testNode,
			mockFunc: func(exec repository.Executor, ctx context.Context, node *models.VaultNode) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "duplicate node",
			node: testNode,
			mockFunc: func(exec repository.Executor, ctx context.Context, node *models.VaultNode) error {
				return errors.New("duplicate key value")
			},
			wantErr: true,
		},
		{
			name: "validation error",
			node: &models.VaultNode{
				// Missing required fields
			},
			mockFunc: func(exec repository.Executor, ctx context.Context, node *models.VaultNode) error {
				return errors.New("validation failed")
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := &mockNodeRepository{
				createFunc: tt.mockFunc,
			}
			
			// Create service with mock
			svc := service.NewNodeServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			// Test CreateNode
			err := svc.CreateNode(ctx, tt.node)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNodeService_SearchNodes tests the SearchNodes method
func TestNodeService_SearchNodes(t *testing.T) {
	ctx := context.Background()
	
	searchResults := []models.VaultNode{
		{ID: "1", Title: "Search Result 1", Content: "Contains search term"},
		{ID: "2", Title: "Search Result 2", Content: "Also contains search term"},
	}
	
	tests := []struct {
		name        string
		query       string
		mockFunc    func(exec repository.Executor, ctx context.Context, query string) ([]models.VaultNode, error)
		wantResults []models.VaultNode
		wantErr     bool
	}{
		{
			name:  "successful search",
			query: "search term",
			mockFunc: func(exec repository.Executor, ctx context.Context, query string) ([]models.VaultNode, error) {
				return searchResults, nil
			},
			wantResults: searchResults,
			wantErr:     false,
		},
		{
			name:  "empty results",
			query: "nonexistent",
			mockFunc: func(exec repository.Executor, ctx context.Context, query string) ([]models.VaultNode, error) {
				return []models.VaultNode{}, nil
			},
			wantResults: []models.VaultNode{},
			wantErr:     false,
		},
		{
			name:  "search error",
			query: "search",
			mockFunc: func(exec repository.Executor, ctx context.Context, query string) ([]models.VaultNode, error) {
				return nil, errors.New("search failed")
			},
			wantResults: nil,
			wantErr:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := &mockNodeRepository{
				searchFunc: tt.mockFunc,
			}
			
			// Create service with mock
			svc := service.NewNodeServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			// Test SearchNodes
			results, err := svc.SearchNodes(ctx, tt.query)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResults, results)
			}
		})
	}
}

// TestNodeService_UpdateNodeAndEdges tests the UpdateNodeAndEdges method with transactions
func TestNodeService_UpdateNodeAndEdges(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		// This test would require a test database to properly test transactions
		// For now, we'll skip the actual transaction testing
		t.Skip("Transaction testing requires test database")
	})
}

// TestNodeService_RebuildNodeGraph tests the RebuildNodeGraph method
func TestNodeService_RebuildNodeGraph(t *testing.T) {
	t.Run("successful rebuild", func(t *testing.T) {
		// This test would require a test database to properly test transactions
		t.Skip("Transaction testing requires test database")
	})
}

// TestNodeService_GetNodeWithEdges tests the GetNodeWithEdges method
func TestNodeService_GetNodeWithEdges(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		// This test requires creating a service with both node and edge repos
		// Skipping for now as it needs refactoring of the service
		t.Skip("Requires service refactoring to inject edge repository")
	})
}

// TestNodeService_Pagination tests pagination functionality
func TestNodeService_Pagination(t *testing.T) {
	ctx := context.Background()
	
	// Create 25 test nodes
	allNodes := make([]models.VaultNode, 25)
	for i := 0; i < 25; i++ {
		allNodes[i] = models.VaultNode{
			ID:       fmt.Sprintf("node-%d", i),
			Title:    fmt.Sprintf("Node %d", i),
			FilePath: fmt.Sprintf("/node%d.md", i),
			NodeType: "note",
		}
	}
	
	tests := []struct {
		name       string
		limit      int
		offset     int
		wantCount  int
		wantFirst  string
		wantLast   string
	}{
		{
			name:      "first page",
			limit:     10,
			offset:    0,
			wantCount: 10,
			wantFirst: "node-0",
			wantLast:  "node-9",
		},
		{
			name:      "second page",
			limit:     10,
			offset:    10,
			wantCount: 10,
			wantFirst: "node-10",
			wantLast:  "node-19",
		},
		{
			name:      "last page",
			limit:     10,
			offset:    20,
			wantCount: 5,
			wantFirst: "node-20",
			wantLast:  "node-24",
		},
		{
			name:      "beyond last page",
			limit:     10,
			offset:    30,
			wantCount: 0,
			wantFirst: "",
			wantLast:  "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockNodeRepository{
				getAllFunc: func(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultNode, error) {
					end := offset + limit
					if end > len(allNodes) {
						end = len(allNodes)
					}
					if offset >= len(allNodes) {
						return []models.VaultNode{}, nil
					}
					return allNodes[offset:end], nil
				},
			}
			
			svc := service.NewNodeServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			nodes, err := svc.GetAllNodes(ctx, tt.limit, tt.offset)
			require.NoError(t, err)
			assert.Len(t, nodes, tt.wantCount)
			
			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantFirst, nodes[0].ID)
				assert.Equal(t, tt.wantLast, nodes[len(nodes)-1].ID)
			}
		})
	}
}

// TestNodeService_ConcurrentAccess tests concurrent access to the service
func TestNodeService_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	
	// Create a mock repository that's safe for concurrent access
	mockRepo := &mockNodeRepository{
		countFunc: func(exec repository.Executor, ctx context.Context) (int64, error) {
			time.Sleep(10 * time.Millisecond) // Simulate some work
			return 100, nil
		},
	}
	
	svc := service.NewNodeServiceWithRepo(&sqlx.DB{}, mockRepo)
	
	// Run 10 concurrent operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			count, err := svc.CountNodes(ctx)
			assert.NoError(t, err)
			assert.Equal(t, int64(100), count)
			done <- true
		}()
	}
	
	// Wait for all operations to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}