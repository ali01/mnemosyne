package service_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/service"
)

// mockPositionRepo implements repository.PositionRepository for testing
type mockPositionRepo struct {
	getByNodeIDFunc          func(exec repository.Executor, ctx context.Context, nodeID string) (*models.NodePosition, error)
	upsertFunc              func(exec repository.Executor, ctx context.Context, position *models.NodePosition) error
	upsertBatchFunc         func(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error
	deleteByNodeIDFunc      func(exec repository.Executor, ctx context.Context, nodeID string) error
	getAllFunc              func(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error)
}

func (m *mockPositionRepo) GetByNodeID(exec repository.Executor, ctx context.Context, nodeID string) (*models.NodePosition, error) {
	if m.getByNodeIDFunc != nil {
		return m.getByNodeIDFunc(exec, ctx, nodeID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPositionRepo) Upsert(exec repository.Executor, ctx context.Context, position *models.NodePosition) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(exec, ctx, position)
	}
	return errors.New("not implemented")
}

func (m *mockPositionRepo) UpsertBatch(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error {
	if m.upsertBatchFunc != nil {
		return m.upsertBatchFunc(exec, ctx, positions)
	}
	return errors.New("not implemented")
}

func (m *mockPositionRepo) DeleteByNodeID(exec repository.Executor, ctx context.Context, nodeID string) error {
	if m.deleteByNodeIDFunc != nil {
		return m.deleteByNodeIDFunc(exec, ctx, nodeID)
	}
	return errors.New("not implemented")
}

func (m *mockPositionRepo) GetAll(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(exec, ctx)
	}
	return nil, errors.New("not implemented")
}


// TestPositionService_GetNodePosition tests the GetNodePosition method
func TestPositionService_GetNodePosition(t *testing.T) {
	ctx := context.Background()
	
	testPosition := &models.NodePosition{
		NodeID:    "node-123",
		X:         100.5,
		Y:         200.5,
		Z:         1.0,
		Locked:    false,
		UpdatedAt: time.Now(),
	}
	
	tests := []struct {
		name         string
		nodeID       string
		mockFunc     func(exec repository.Executor, ctx context.Context, nodeID string) (*models.NodePosition, error)
		wantPosition *models.NodePosition
		wantErr      bool
	}{
		{
			name:   "successful retrieval",
			nodeID: "node-123",
			mockFunc: func(exec repository.Executor, ctx context.Context, nodeID string) (*models.NodePosition, error) {
				return testPosition, nil
			},
			wantPosition: testPosition,
			wantErr:      false,
		},
		{
			name:   "position not found",
			nodeID: "nonexistent",
			mockFunc: func(exec repository.Executor, ctx context.Context, nodeID string) (*models.NodePosition, error) {
				return nil, nil // Return nil, nil for not found
			},
			wantPosition: nil,
			wantErr:      false,
		},
		{
			name:   "database error",
			nodeID: "node-123",
			mockFunc: func(exec repository.Executor, ctx context.Context, nodeID string) (*models.NodePosition, error) {
				return nil, errors.New("database connection error")
			},
			wantPosition: nil,
			wantErr:      true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockPositionRepo{
				getByNodeIDFunc: tt.mockFunc,
			}
			
			svc := service.NewPositionServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			position, err := svc.GetNodePosition(ctx, tt.nodeID)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPosition, position)
			}
		})
	}
}

// TestPositionService_UpdateNodePosition tests the UpdateNodePosition method
func TestPositionService_UpdateNodePosition(t *testing.T) {
	ctx := context.Background()
	
	testPosition := &models.NodePosition{
		NodeID:    "node-456",
		X:         300.0,
		Y:         400.0,
		Z:         2.0,
		Locked:    true,
		UpdatedAt: time.Now(),
	}
	
	tests := []struct {
		name     string
		position *models.NodePosition
		mockFunc func(exec repository.Executor, ctx context.Context, position *models.NodePosition) error
		wantErr  bool
	}{
		{
			name:     "successful update",
			position: testPosition,
			mockFunc: func(exec repository.Executor, ctx context.Context, position *models.NodePosition) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:     "validation error - missing node ID",
			position: &models.NodePosition{X: 100, Y: 200},
			mockFunc: func(exec repository.Executor, ctx context.Context, position *models.NodePosition) error {
				return errors.New("node ID is required")
			},
			wantErr: true,
		},
		{
			name:     "database error",
			position: testPosition,
			mockFunc: func(exec repository.Executor, ctx context.Context, position *models.NodePosition) error {
				return errors.New("update failed")
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockPositionRepo{
				upsertFunc: tt.mockFunc,
			}
			
			svc := service.NewPositionServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			err := svc.UpdateNodePosition(ctx, tt.position)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPositionService_GetViewportPositions tests the GetViewportPositions method
func TestPositionService_GetViewportPositions(t *testing.T) {
	ctx := context.Background()
	
	allPositions := []models.NodePosition{
		{NodeID: "node-1", X: 50, Y: 50},     // Outside viewport
		{NodeID: "node-2", X: 150, Y: 150},   // Inside viewport
		{NodeID: "node-3", X: 250, Y: 250},   // Inside viewport
		{NodeID: "node-4", X: 350, Y: 350},   // Outside viewport
		{NodeID: "node-5", X: 200, Y: 200},   // Inside viewport
	}
	
	tests := []struct {
		name          string
		minX, maxX    float64
		minY, maxY    float64
		mockFunc      func(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error)
		wantPositions []models.NodePosition
		wantErr       bool
	}{
		{
			name: "successful viewport query",
			minX: 100, maxX: 300,
			minY: 100, maxY: 300,
			mockFunc: func(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error) {
				return allPositions, nil
			},
			wantPositions: []models.NodePosition{
				{NodeID: "node-2", X: 150, Y: 150},
				{NodeID: "node-3", X: 250, Y: 250},
				{NodeID: "node-5", X: 200, Y: 200},
			},
			wantErr: false,
		},
		{
			name: "empty viewport",
			minX: 500, maxX: 600,
			minY: 500, maxY: 600,
			mockFunc: func(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error) {
				return allPositions, nil
			},
			wantPositions: []models.NodePosition{},
			wantErr:       false,
		},
		{
			name: "database error",
			minX: 0, maxX: 100,
			minY: 0, maxY: 100,
			mockFunc: func(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error) {
				return nil, errors.New("query failed")
			},
			wantPositions: nil,
			wantErr:       true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockPositionRepo{
				getAllFunc: tt.mockFunc,
			}
			
			svc := service.NewPositionServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			positions, err := svc.GetViewportPositions(ctx, tt.minX, tt.maxX, tt.minY, tt.maxY)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPositions, positions)
			}
		})
	}
}

// TestPositionService_BatchUpdate tests batch position updates
func TestPositionService_BatchUpdate(t *testing.T) {
	ctx := context.Background()
	
	positions := []models.NodePosition{
		{NodeID: "node-1", X: 100, Y: 100, Z: 0, Locked: false},
		{NodeID: "node-2", X: 200, Y: 200, Z: 1, Locked: true},
		{NodeID: "node-3", X: 300, Y: 300, Z: 2, Locked: false},
	}
	
	tests := []struct {
		name      string
		positions []models.NodePosition
		mockFunc  func(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error
		wantErr   bool
	}{
		{
			name:      "successful batch update",
			positions: positions,
			mockFunc: func(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:      "empty batch",
			positions: []models.NodePosition{},
			mockFunc: func(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:      "batch update error",
			positions: positions,
			mockFunc: func(exec repository.Executor, ctx context.Context, positions []models.NodePosition) error {
				return errors.New("batch update failed")
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockPositionRepo{
				upsertBatchFunc: tt.mockFunc,
			}
			
			svc := service.NewPositionServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			err := svc.UpdateNodePositions(ctx, tt.positions)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPositionService_DeleteNodePosition tests deleting a node position
func TestPositionService_DeleteNodePosition(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name     string
		nodeID   string
		mockFunc func(exec repository.Executor, ctx context.Context, nodeID string) error
		wantErr  bool
	}{
		{
			name:   "successful deletion",
			nodeID: "node-to-delete",
			mockFunc: func(exec repository.Executor, ctx context.Context, nodeID string) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:   "position not found",
			nodeID: "nonexistent",
			mockFunc: func(exec repository.Executor, ctx context.Context, nodeID string) error {
				// Still returns nil as it's idempotent
				return nil
			},
			wantErr: false,
		},
		{
			name:   "database error",
			nodeID: "node-123",
			mockFunc: func(exec repository.Executor, ctx context.Context, nodeID string) error {
				return errors.New("delete failed")
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockPositionRepo{
				deleteByNodeIDFunc: tt.mockFunc,
			}
			
			svc := service.NewPositionServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			err := svc.DeleteNodePosition(ctx, tt.nodeID)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPositionService_GetAllPositions tests retrieving all positions
func TestPositionService_GetAllPositions(t *testing.T) {
	ctx := context.Background()
	
	allPositions := []models.NodePosition{
		{NodeID: "node-1", X: 100, Y: 100, Z: 0, Locked: false},
		{NodeID: "node-2", X: 200, Y: 200, Z: 1, Locked: true},
		{NodeID: "node-3", X: 300, Y: 300, Z: 2, Locked: false},
	}
	
	tests := []struct {
		name          string
		mockFunc      func(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error)
		wantPositions []models.NodePosition
		wantErr       bool
	}{
		{
			name: "successful retrieval",
			mockFunc: func(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error) {
				return allPositions, nil
			},
			wantPositions: allPositions,
			wantErr:       false,
		},
		{
			name: "empty positions",
			mockFunc: func(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error) {
				return []models.NodePosition{}, nil
			},
			wantPositions: []models.NodePosition{},
			wantErr:       false,
		},
		{
			name: "database error",
			mockFunc: func(exec repository.Executor, ctx context.Context) ([]models.NodePosition, error) {
				return nil, errors.New("query failed")
			},
			wantPositions: nil,
			wantErr:       true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockPositionRepo{
				getAllFunc: tt.mockFunc,
			}
			
			svc := service.NewPositionServiceWithRepo(&sqlx.DB{}, mockRepo)
			
			positions, err := svc.GetAllPositions(ctx)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPositions, positions)
			}
		})
	}
}

// TestPositionService_ConcurrentUpdates tests concurrent position updates
func TestPositionService_ConcurrentUpdates(t *testing.T) {
	ctx := context.Background()
	
	var updateCount int32
	var mu sync.Mutex
	mockRepo := &mockPositionRepo{
		upsertFunc: func(exec repository.Executor, ctx context.Context, position *models.NodePosition) error {
			mu.Lock()
			updateCount++
			mu.Unlock()
			time.Sleep(10 * time.Millisecond) // Simulate some work
			return nil
		},
	}
	
	svc := service.NewPositionServiceWithRepo(&sqlx.DB{}, mockRepo)
	
	// Run 10 concurrent updates
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			position := &models.NodePosition{
				NodeID: fmt.Sprintf("node-%d", idx),
				X:      float64(idx * 100),
				Y:      float64(idx * 100),
			}
			err := svc.UpdateNodePosition(ctx, position)
			assert.NoError(t, err)
			done <- true
		}(i)
	}
	
	// Wait for all updates to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify all updates were called
	assert.Equal(t, int32(10), updateCount)
}