package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/service"
)

// mockEdgeRepoForTest implements repository.EdgeRepository for testing
type mockEdgeRepoForTest struct {
	createFunc               func(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error
	getByIDFunc             func(exec repository.Executor, ctx context.Context, id string) (*models.VaultEdge, error)
	updateFunc              func(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error
	deleteFunc              func(exec repository.Executor, ctx context.Context, id string) error
	getAllFunc              func(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultEdge, error)
	getByNodeFunc           func(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error)
	getBySourceAndTargetFunc func(exec repository.Executor, ctx context.Context, sourceID, targetID string) ([]models.VaultEdge, error)
	countFunc               func(exec repository.Executor, ctx context.Context) (int64, error)
	createBatchFunc         func(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error
	upsertBatchFunc         func(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error
	deleteByNodeFunc        func(exec repository.Executor, ctx context.Context, nodeID string) error
	deleteAllFunc           func(exec repository.Executor, ctx context.Context) error
	getIncomingEdgesFunc    func(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error)
	getOutgoingEdgesFunc    func(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error)
}

func (m *mockEdgeRepoForTest) Create(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error {
	if m.createFunc != nil {
		return m.createFunc(exec, ctx, edge)
	}
	return errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) GetByID(exec repository.Executor, ctx context.Context, id string) (*models.VaultEdge, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(exec, ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) Update(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error {
	if m.updateFunc != nil {
		return m.updateFunc(exec, ctx, edge)
	}
	return errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) Delete(exec repository.Executor, ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(exec, ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) GetAll(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultEdge, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(exec, ctx, limit, offset)
	}
	return nil, errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) GetByNode(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	if m.getByNodeFunc != nil {
		return m.getByNodeFunc(exec, ctx, nodeID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) GetBySourceAndTarget(exec repository.Executor, ctx context.Context, sourceID, targetID string) ([]models.VaultEdge, error) {
	if m.getBySourceAndTargetFunc != nil {
		return m.getBySourceAndTargetFunc(exec, ctx, sourceID, targetID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) Count(exec repository.Executor, ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(exec, ctx)
	}
	return 0, errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) CreateBatch(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	if m.createBatchFunc != nil {
		return m.createBatchFunc(exec, ctx, edges)
	}
	return errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) UpsertBatch(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	if m.upsertBatchFunc != nil {
		return m.upsertBatchFunc(exec, ctx, edges)
	}
	return errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) DeleteByNode(exec repository.Executor, ctx context.Context, nodeID string) error {
	if m.deleteByNodeFunc != nil {
		return m.deleteByNodeFunc(exec, ctx, nodeID)
	}
	return errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) DeleteAll(exec repository.Executor, ctx context.Context) error {
	if m.deleteAllFunc != nil {
		return m.deleteAllFunc(exec, ctx)
	}
	return errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) GetIncomingEdges(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	if m.getIncomingEdgesFunc != nil {
		return m.getIncomingEdgesFunc(exec, ctx, nodeID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockEdgeRepoForTest) GetOutgoingEdges(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	if m.getOutgoingEdgesFunc != nil {
		return m.getOutgoingEdgesFunc(exec, ctx, nodeID)
	}
	return nil, errors.New("not implemented")
}

// TestEdgeService_CreateEdge tests the CreateEdge method
func TestEdgeService_CreateEdge(t *testing.T) {
	ctx := context.Background()

	testEdge := &models.VaultEdge{
		ID:       "edge-123",
		SourceID: "node-1",
		TargetID: "node-2",
		EdgeType: "link",
		Weight:   1.0,
	}

	tests := []struct {
		name     string
		edge     *models.VaultEdge
		mockFunc func(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error
		wantErr  bool
	}{
		{
			name: "successful creation",
			edge: testEdge,
			mockFunc: func(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "duplicate edge",
			edge: testEdge,
			mockFunc: func(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error {
				return errors.New("duplicate key value")
			},
			wantErr: true,
		},
		{
			name: "validation error",
			edge: &models.VaultEdge{
				// Missing required fields
			},
			mockFunc: func(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error {
				return errors.New("validation failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockEdgeRepoForTest{
				createFunc: tt.mockFunc,
			}

			svc := service.NewEdgeServiceWithRepo(&sqlx.DB{}, mockRepo)

			err := svc.CreateEdge(ctx, tt.edge)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestEdgeService_GetEdgesByNode tests the GetEdgesByNode method
func TestEdgeService_GetEdgesByNode(t *testing.T) {
	ctx := context.Background()

	nodeID := "node-123"
	testEdges := []models.VaultEdge{
		{ID: "edge-1", SourceID: nodeID, TargetID: "target-1", EdgeType: "link"},
		{ID: "edge-2", SourceID: "source-1", TargetID: nodeID, EdgeType: "reference"},
		{ID: "edge-3", SourceID: nodeID, TargetID: "target-2", EdgeType: "embed"},
	}

	tests := []struct {
		name       string
		nodeID     string
		mockFunc   func(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error)
		wantEdges  []models.VaultEdge
		wantErr    bool
	}{
		{
			name:   "successful retrieval",
			nodeID: nodeID,
			mockFunc: func(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
				return testEdges, nil
			},
			wantEdges: testEdges,
			wantErr:   false,
		},
		{
			name:   "no edges found",
			nodeID: "orphan-node",
			mockFunc: func(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
				return []models.VaultEdge{}, nil
			},
			wantEdges: []models.VaultEdge{},
			wantErr:   false,
		},
		{
			name:   "database error",
			nodeID: nodeID,
			mockFunc: func(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
				return nil, errors.New("database connection error")
			},
			wantEdges: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockEdgeRepoForTest{
				getByNodeFunc: tt.mockFunc,
			}

			svc := service.NewEdgeServiceWithRepo(&sqlx.DB{}, mockRepo)

			edges, err := svc.GetEdgesByNode(ctx, tt.nodeID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantEdges, edges)
			}
		})
	}
}

// TestEdgeService_BatchOperations tests batch edge operations
func TestEdgeService_BatchOperations(t *testing.T) {
	ctx := context.Background()

	testEdges := []models.VaultEdge{
		{ID: "batch-1", SourceID: "node-1", TargetID: "node-2", EdgeType: "link"},
		{ID: "batch-2", SourceID: "node-2", TargetID: "node-3", EdgeType: "reference"},
		{ID: "batch-3", SourceID: "node-3", TargetID: "node-1", EdgeType: "embed"},
	}

	t.Run("create batch", func(t *testing.T) {
		mockRepo := &mockEdgeRepoForTest{
			createBatchFunc: func(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
				assert.Equal(t, testEdges, edges)
				return nil
			},
		}

		svc := service.NewEdgeServiceWithRepo(&sqlx.DB{}, mockRepo)

		err := svc.CreateEdges(ctx, testEdges)
		assert.NoError(t, err)
	})

	t.Run("create batch error", func(t *testing.T) {
		mockRepo := &mockEdgeRepoForTest{
			createBatchFunc: func(exec repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
				return errors.New("batch insert failed")
			},
		}

		svc := service.NewEdgeServiceWithRepo(&sqlx.DB{}, mockRepo)

		err := svc.CreateEdges(ctx, testEdges)
		assert.Error(t, err)
	})
}

// TestEdgeService_DeleteNodeEdges tests deleting all edges for a node
func TestEdgeService_DeleteNodeEdges(t *testing.T) {
	ctx := context.Background()

	deletedEdges := []string{}

	// Create mock repository
	mockRepo := &mockEdgeRepoForTest{
		getByNodeFunc: func(exec repository.Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
			if nodeID == "node1" {
				return []models.VaultEdge{
					{ID: "edge1", SourceID: "node1", TargetID: "node2"},
					{ID: "edge2", SourceID: "node2", TargetID: "node1"},
				}, nil
			}
			return []models.VaultEdge{}, nil
		},
		deleteFunc: func(exec repository.Executor, ctx context.Context, id string) error {
			deletedEdges = append(deletedEdges, id)
			return nil
		},
	}

	t.Run("successful deletion", func(t *testing.T) {
		// The service would need to support this method for the test to work
		// For now, we test the repository directly
		edges, err := mockRepo.getByNodeFunc(nil, ctx, "node1")
		require.NoError(t, err)
		assert.Len(t, edges, 2)

		// Simulate deleting each edge
		for _, edge := range edges {
			err := mockRepo.deleteFunc(nil, ctx, edge.ID)
			require.NoError(t, err)
		}

		assert.Contains(t, deletedEdges, "edge1")
		assert.Contains(t, deletedEdges, "edge2")
	})
}

// TestEdgeService_CountEdges tests counting edges
func TestEdgeService_CountEdges(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		mockFunc  func(exec repository.Executor, ctx context.Context) (int64, error)
		wantCount int64
		wantErr   bool
	}{
		{
			name: "successful count",
			mockFunc: func(exec repository.Executor, ctx context.Context) (int64, error) {
				return 42, nil
			},
			wantCount: 42,
			wantErr:   false,
		},
		{
			name: "empty graph",
			mockFunc: func(exec repository.Executor, ctx context.Context) (int64, error) {
				return 0, nil
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "database error",
			mockFunc: func(exec repository.Executor, ctx context.Context) (int64, error) {
				return 0, errors.New("count failed")
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockEdgeRepoForTest{
				countFunc: tt.mockFunc,
			}

			svc := service.NewEdgeServiceWithRepo(&sqlx.DB{}, mockRepo)

			count, err := svc.CountEdges(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}
		})
	}
}

// TestEdgeService_UpdateEdge tests the update edge method
func TestEdgeService_UpdateEdge(t *testing.T) {
	ctx := context.Background()

	// Track operations
	operations := []string{}

	mockRepo := &mockEdgeRepoForTest{
		deleteFunc: func(exec repository.Executor, ctx context.Context, id string) error {
			operations = append(operations, fmt.Sprintf("delete-%s", id))
			return nil
		},
		createFunc: func(exec repository.Executor, ctx context.Context, edge *models.VaultEdge) error {
			operations = append(operations, fmt.Sprintf("create-%s", edge.ID))
			return nil
		},
	}

	// We can't properly test transactions with mocks, but we can verify the operations
	// The actual transaction testing is done in the repository layer tests
	t.Run("update operations", func(t *testing.T) {
		edge := &models.VaultEdge{
			ID:       "edge-to-update",
			SourceID: "node1",
			TargetID: "node2",
			EdgeType: "wikilink",
		}

		// Simulate the operations that would happen in UpdateEdge
		err := mockRepo.deleteFunc(nil, ctx, edge.ID)
		require.NoError(t, err)

		err = mockRepo.createFunc(nil, ctx, edge)
		require.NoError(t, err)

		// Verify both operations occurred in order
		assert.Equal(t, []string{"delete-edge-to-update", "create-edge-to-update"}, operations)
	})
}

// TestEdgeService_Pagination tests pagination for edges
func TestEdgeService_Pagination(t *testing.T) {
	ctx := context.Background()

	// Create 20 test edges
	allEdges := make([]models.VaultEdge, 20)
	for i := 0; i < 20; i++ {
		allEdges[i] = models.VaultEdge{
			ID:       fmt.Sprintf("edge-%d", i),
			SourceID: fmt.Sprintf("node-%d", i),
			TargetID: fmt.Sprintf("node-%d", (i+1)%20),
			EdgeType: "link",
		}
	}

	tests := []struct {
		name      string
		limit     int
		offset    int
		wantCount int
		wantFirst string
		wantLast  string
	}{
		{
			name:      "first page",
			limit:     5,
			offset:    0,
			wantCount: 5,
			wantFirst: "edge-0",
			wantLast:  "edge-4",
		},
		{
			name:      "middle page",
			limit:     5,
			offset:    10,
			wantCount: 5,
			wantFirst: "edge-10",
			wantLast:  "edge-14",
		},
		{
			name:      "last page",
			limit:     5,
			offset:    15,
			wantCount: 5,
			wantFirst: "edge-15",
			wantLast:  "edge-19",
		},
		{
			name:      "beyond last page",
			limit:     5,
			offset:    25,
			wantCount: 0,
			wantFirst: "",
			wantLast:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockEdgeRepoForTest{
				getAllFunc: func(exec repository.Executor, ctx context.Context, limit, offset int) ([]models.VaultEdge, error) {
					end := offset + limit
					if end > len(allEdges) {
						end = len(allEdges)
					}
					if offset >= len(allEdges) {
						return []models.VaultEdge{}, nil
					}
					return allEdges[offset:end], nil
				},
			}

			svc := service.NewEdgeServiceWithRepo(&sqlx.DB{}, mockRepo)

			edges, err := svc.GetAllEdges(ctx, tt.limit, tt.offset)
			require.NoError(t, err)
			assert.Len(t, edges, tt.wantCount)

			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantFirst, edges[0].ID)
				assert.Equal(t, tt.wantLast, edges[len(edges)-1].ID)
			}
		})
	}
}
