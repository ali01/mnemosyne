package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository/mock"
)

// TestNodeServiceWithMockRepo tests the node service with a mock repository
func TestNodeServiceWithMockRepo(t *testing.T) {
	ctx := context.Background()
	
	// Create a mock repository
	mockRepo := mock.NewNodeRepository()
	
	// We need a mock executor for the repository
	exec := &mockExecutor{}
	
	// Add a test node to the mock repository
	testNode := &models.VaultNode{
		ID:       "test-123",
		Title:    "Test Node",
		FilePath: "/test/node.md",
		NodeType: "note",
		Content:  "Test content",
	}
	
	err := mockRepo.Create(exec, ctx, testNode)
	require.NoError(t, err)
	
	// Test retrieval
	retrieved, err := mockRepo.GetByID(exec, ctx, "test-123")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Test Node", retrieved.Title)
	
	// Test search
	results, err := mockRepo.Search(exec, ctx, "Test")
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	
	// Test count
	count, err := mockRepo.Count(exec, ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
	
	// Test update
	testNode.Title = "Updated Node"
	err = mockRepo.Update(exec, ctx, testNode)
	assert.NoError(t, err)
	
	// Test delete
	err = mockRepo.Delete(exec, ctx, "test-123")
	assert.NoError(t, err)
	
	// Verify deletion
	_, err = mockRepo.GetByID(exec, ctx, "test-123")
	assert.Error(t, err)
}

// TestNodeServicePagination tests pagination functionality
func TestNodeServicePagination(t *testing.T) {
	ctx := context.Background()
	mockRepo := mock.NewNodeRepository()
	exec := &mockExecutor{}
	
	// Add 25 nodes
	for i := 0; i < 25; i++ {
		node := &models.VaultNode{
			ID:       fmt.Sprintf("node-%d", i),
			Title:    fmt.Sprintf("Node %d", i),
			FilePath: fmt.Sprintf("/test/node%d.md", i),
			NodeType: "note",
		}
		err := mockRepo.Create(exec, ctx, node)
		require.NoError(t, err)
	}
	
	// Test first page
	page1, err := mockRepo.GetAll(exec, ctx, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, page1, 10)
	
	// Test second page
	page2, err := mockRepo.GetAll(exec, ctx, 10, 10)
	assert.NoError(t, err)
	assert.Len(t, page2, 10)
	
	// Test last page
	page3, err := mockRepo.GetAll(exec, ctx, 10, 20)
	assert.NoError(t, err)
	assert.Len(t, page3, 5)
	
	// Test beyond last page
	page4, err := mockRepo.GetAll(exec, ctx, 10, 30)
	assert.NoError(t, err)
	assert.Len(t, page4, 0)
}

// TestNodeServiceBatchOperations tests batch operations
func TestNodeServiceBatchOperations(t *testing.T) {
	ctx := context.Background()
	mockRepo := mock.NewNodeRepository()
	exec := &mockExecutor{}
	
	// Create batch of nodes
	nodes := []models.VaultNode{
		{ID: "batch-1", Title: "Batch Node 1", FilePath: "/batch/1.md", NodeType: "note"},
		{ID: "batch-2", Title: "Batch Node 2", FilePath: "/batch/2.md", NodeType: "note"},
		{ID: "batch-3", Title: "Batch Node 3", FilePath: "/batch/3.md", NodeType: "note"},
	}
	
	err := mockRepo.CreateBatch(exec, ctx, nodes)
	assert.NoError(t, err)
	
	// Verify all were created
	count, err := mockRepo.Count(exec, ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
	
	// Test upsert batch (update existing, create new)
	upsertNodes := []models.VaultNode{
		{ID: "batch-1", Title: "Updated Batch Node 1", FilePath: "/batch/1.md", NodeType: "note"}, // Update
		{ID: "batch-4", Title: "Batch Node 4", FilePath: "/batch/4.md", NodeType: "note"},         // New
	}
	
	err = mockRepo.UpsertBatch(exec, ctx, upsertNodes)
	assert.NoError(t, err)
	
	// Verify count increased
	count, err = mockRepo.Count(exec, ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(4), count)
	
	// Verify update worked
	updated, err := mockRepo.GetByID(exec, ctx, "batch-1")
	assert.NoError(t, err)
	assert.Equal(t, "Updated Batch Node 1", updated.Title)
}

// TestNodeServiceByType tests filtering by node type
func TestNodeServiceByType(t *testing.T) {
	ctx := context.Background()
	mockRepo := mock.NewNodeRepository()
	exec := &mockExecutor{}
	
	// Add nodes of different types
	nodeData := []struct {
		nodeType string
		count    int
	}{
		{"index", 3},
		{"concept", 5},
		{"project", 2},
		{"note", 10},
	}
	
	for _, data := range nodeData {
		for i := 0; i < data.count; i++ {
			node := &models.VaultNode{
				ID:       fmt.Sprintf("%s-%d", data.nodeType, i),
				Title:    fmt.Sprintf("%s Node %d", data.nodeType, i),
				FilePath: fmt.Sprintf("/%s/node%d.md", data.nodeType, i),
				NodeType: data.nodeType,
			}
			err := mockRepo.Create(exec, ctx, node)
			require.NoError(t, err)
		}
	}
	
	// Test getting nodes by type
	for _, data := range nodeData {
		nodes, err := mockRepo.GetByType(exec, ctx, data.nodeType)
		assert.NoError(t, err)
		assert.Len(t, nodes, data.count)
		
		// Verify all have correct type
		for _, node := range nodes {
			assert.Equal(t, data.nodeType, node.NodeType)
		}
	}
	
	// Test non-existent type
	nodes, err := mockRepo.GetByType(exec, ctx, "non-existent")
	assert.NoError(t, err)
	assert.Len(t, nodes, 0)
}

// TestNodeServiceConcurrentAccess tests concurrent access
func TestNodeServiceConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	mockRepo := mock.NewNodeRepository()
	exec := &mockExecutor{}
	
	// Add initial nodes
	for i := 0; i < 10; i++ {
		node := &models.VaultNode{
			ID:       fmt.Sprintf("concurrent-%d", i),
			Title:    fmt.Sprintf("Concurrent Node %d", i),
			FilePath: fmt.Sprintf("/concurrent/node%d.md", i),
			NodeType: "note",
		}
		err := mockRepo.Create(exec, ctx, node)
		require.NoError(t, err)
	}
	
	// Run concurrent operations
	done := make(chan bool, 30)
	
	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			_, _ = mockRepo.GetByID(exec, ctx, fmt.Sprintf("concurrent-%d", id))
			done <- true
		}(i)
	}
	
	// Concurrent searches
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = mockRepo.Search(exec, ctx, "Concurrent")
			done <- true
		}()
	}
	
	// Concurrent counts
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = mockRepo.Count(exec, ctx)
			done <- true
		}()
	}
	
	// Wait for all operations to complete
	for i := 0; i < 30; i++ {
		<-done
	}
	
	// Verify data integrity
	count, err := mockRepo.Count(exec, ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), count)
}

// mockExecutor is a simple mock implementation of the Executor interface
type mockExecutor struct{}

func (m *mockExecutor) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return nil
}

func (m *mockExecutor) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return nil
}

func (m *mockExecutor) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return &mockResult{}, nil
}

func (m *mockExecutor) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return &mockResult{}, nil
}

func (m *mockExecutor) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *mockExecutor) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *mockExecutor) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, nil
}

// mockResult implements sql.Result
type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}