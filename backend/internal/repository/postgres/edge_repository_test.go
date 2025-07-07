// Package postgres provides integration tests for edge repository
package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/models"
)

// TestEdgeRepositoryStateless runs integration tests for the PostgreSQL edge repository
func TestEdgeRepositoryStateless(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database connection
	cfg := db.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "mnemosyne_test",
		SSLMode:  "disable",
	}

	database, err := db.Connect(cfg)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}
	defer database.Close()

	// Create repositories
	nodeRepo := NewNodeRepository()
	edgeRepo := NewEdgeRepository()

	// Clean up before tests
	ctx := context.Background()
	_ = edgeRepo.DeleteAll(database, ctx)
	_ = nodeRepo.DeleteAll(database, ctx)

	// Create test nodes first (edges reference nodes)
	nodes := []models.VaultNode{
		{ID: "node-1", Title: "Node 1", FilePath: "/node/1.md"},
		{ID: "node-2", Title: "Node 2", FilePath: "/node/2.md"},
		{ID: "node-3", Title: "Node 3", FilePath: "/node/3.md"},
	}

	err = nodeRepo.CreateBatch(database, ctx, nodes)
	require.NoError(t, err)

	t.Run("Create", func(t *testing.T) {
		edge := &models.VaultEdge{
			ID:       "edge-1",
			SourceID: "node-1",
			TargetID: "node-2",
			EdgeType: "reference",
			DisplayText: "link text",
			Weight:   1.0,
		}

		err := edgeRepo.Create(database, ctx, edge)
		assert.NoError(t, err)
		assert.NotEmpty(t, edge.ID)
	})

	t.Run("GetByID", func(t *testing.T) {
		edge, err := edgeRepo.GetByID(database, ctx, "edge-1")
		assert.NoError(t, err)
		assert.Equal(t, "edge-1", edge.ID)
		assert.Equal(t, "node-1", edge.SourceID)
		assert.Equal(t, "node-2", edge.TargetID)
	})

	t.Run("CreateBatch", func(t *testing.T) {
		edges := []models.VaultEdge{
			{
				SourceID: "node-1",
				TargetID: "node-3",
				EdgeType: "reference",
				Weight:   1.0,
			},
			{
				SourceID: "node-2",
				TargetID: "node-3",
				EdgeType: "reference",
				Weight:   1.0,
			},
		}

		err := edgeRepo.CreateBatch(database, ctx, edges)
		assert.NoError(t, err)
	})

	t.Run("GetByNode", func(t *testing.T) {
		edges, err := edgeRepo.GetByNode(database, ctx, "node-3")
		assert.NoError(t, err)
		assert.Len(t, edges, 2)
	})

	t.Run("GetIncomingEdges", func(t *testing.T) {
		edges, err := edgeRepo.GetIncomingEdges(database, ctx, "node-3")
		assert.NoError(t, err)
		assert.Len(t, edges, 2)
	})

	t.Run("GetOutgoingEdges", func(t *testing.T) {
		edges, err := edgeRepo.GetOutgoingEdges(database, ctx, "node-1")
		assert.NoError(t, err)
		assert.Len(t, edges, 2)
	})

	t.Run("Count", func(t *testing.T) {
		count, err := edgeRepo.Count(database, ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3))
	})

	t.Run("Delete", func(t *testing.T) {
		err := edgeRepo.Delete(database, ctx, "edge-1")
		assert.NoError(t, err)

		_, err = edgeRepo.GetByID(database, ctx, "edge-1")
		assert.Error(t, err)
		assert.True(t, IsNotFound(err))
	})
}