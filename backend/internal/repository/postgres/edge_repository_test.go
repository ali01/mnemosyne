// Package postgres provides integration tests for edge repository
package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// TestEdgeRepository runs integration tests for the PostgreSQL edge repository
func TestEdgeRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database and repositories
	tdb, repos := CreateTestRepositories(t)
	defer tdb.Close()

	// Clean up before tests
	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	// Create test nodes first (edges reference nodes)
	nodes := []models.VaultNode{
		{ID: "node-1", Title: "Node 1", FilePath: "/node/1.md", NodeType: "note"},
		{ID: "node-2", Title: "Node 2", FilePath: "/node/2.md", NodeType: "note"},
		{ID: "node-3", Title: "Node 3", FilePath: "/node/3.md", NodeType: "note"},
	}

	err := repos.Nodes.CreateBatch(tdb.DB, ctx, nodes)
	require.NoError(t, err)

	t.Run("Create", func(t *testing.T) {
		edge := &models.VaultEdge{
			// ID will be auto-generated as UUID
			SourceID:    "node-1",
			TargetID:    "node-2",
			EdgeType:    "wikilink",
			DisplayText: "Link to Node 2",
			Weight:      1.0,
		}

		err := repos.Edges.Create(tdb.DB, ctx, edge)
		assert.NoError(t, err)
		assert.NotEmpty(t, edge.ID)

		// Since ID is auto-generated, we need to query by source and target
		edges, err := repos.Edges.GetBySourceAndTarget(tdb.DB, ctx, "node-1", "node-2")
		require.NoError(t, err)
		assert.Len(t, edges, 1)
		assert.Equal(t, edge.SourceID, edges[0].SourceID)
		assert.Equal(t, edge.TargetID, edges[0].TargetID)
		assert.Equal(t, edge.EdgeType, edges[0].EdgeType)
	})

	t.Run("CreateBatch", func(t *testing.T) {
		// Clean up from previous tests
		_ = repos.Edges.DeleteAll(tdb.DB, ctx)
		edges := []models.VaultEdge{
			{SourceID: "node-1", TargetID: "node-2", EdgeType: "wikilink"},
			{SourceID: "node-2", TargetID: "node-3", EdgeType: "embed"},
			{SourceID: "node-3", TargetID: "node-1", EdgeType: "wikilink"},
		}

		err := repos.Edges.CreateBatch(tdb.DB, ctx, edges)
		assert.NoError(t, err)

		// Verify all were created by counting total edges
		count, err := repos.Edges.Count(tdb.DB, ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("GetByNode", func(t *testing.T) {
		// Clean up from previous tests
		_ = repos.Edges.DeleteAll(tdb.DB, ctx)
		// Create edges connecting to node-2
		edges := []models.VaultEdge{
			{SourceID: "node-1", TargetID: "node-2", EdgeType: "wikilink"},
			{SourceID: "node-2", TargetID: "node-3", EdgeType: "wikilink"},
			{SourceID: "node-3", TargetID: "node-2", EdgeType: "embed"},
		}
		err := repos.Edges.CreateBatch(tdb.DB, ctx, edges)
		require.NoError(t, err)

		// Get all edges connected to node-2
		connectedEdges, err := repos.Edges.GetByNode(tdb.DB, ctx, "node-2")
		assert.NoError(t, err)
		assert.Len(t, connectedEdges, 3)
	})

	t.Run("GetIncomingEdges", func(t *testing.T) {
		// Clean up from previous tests
		_ = repos.Edges.DeleteAll(tdb.DB, ctx)

		// Create edges with node-2 as target
		edges := []models.VaultEdge{
			{SourceID: "node-1", TargetID: "node-2", EdgeType: "wikilink"},
			{SourceID: "node-3", TargetID: "node-2", EdgeType: "embed"},
			{SourceID: "node-2", TargetID: "node-3", EdgeType: "wikilink"}, // This one should not be included
		}
		err := repos.Edges.CreateBatch(tdb.DB, ctx, edges)
		require.NoError(t, err)

		// Get incoming edges to node-2
		incoming, err := repos.Edges.GetIncomingEdges(tdb.DB, ctx, "node-2")
		assert.NoError(t, err)
		assert.Len(t, incoming, 2)
	})

	t.Run("GetOutgoingEdges", func(t *testing.T) {
		// Clean up from previous tests
		_ = repos.Edges.DeleteAll(tdb.DB, ctx)

		// Create edges with node-2 as source
		edges := []models.VaultEdge{
			{SourceID: "node-2", TargetID: "node-1", EdgeType: "wikilink"},
			{SourceID: "node-2", TargetID: "node-3", EdgeType: "embed"},
			{SourceID: "node-1", TargetID: "node-2", EdgeType: "wikilink"}, // This one should not be included
		}
		err := repos.Edges.CreateBatch(tdb.DB, ctx, edges)
		require.NoError(t, err)

		// Get outgoing edges from node-2
		outgoing, err := repos.Edges.GetOutgoingEdges(tdb.DB, ctx, "node-2")
		assert.NoError(t, err)
		assert.Len(t, outgoing, 2)
	})

	t.Run("GetBySourceAndTarget", func(t *testing.T) {
		// Clean up from previous tests
		_ = repos.Edges.DeleteAll(tdb.DB, ctx)
		// Create specific edge
		edge := &models.VaultEdge{
			SourceID: "node-1",
			TargetID: "node-3",
			EdgeType: "wikilink",
		}
		err := repos.Edges.Create(tdb.DB, ctx, edge)
		require.NoError(t, err)

		// Find edges between node-1 and node-3
		edges, err := repos.Edges.GetBySourceAndTarget(tdb.DB, ctx, "node-1", "node-3")
		assert.NoError(t, err)
		assert.Len(t, edges, 1)
		assert.Equal(t, "node-1", edges[0].SourceID)
		assert.Equal(t, "node-3", edges[0].TargetID)
	})

	t.Run("Delete", func(t *testing.T) {
		// Clean up from previous tests
		_ = repos.Edges.DeleteAll(tdb.DB, ctx)
		edge := &models.VaultEdge{
			SourceID: "node-1",
			TargetID: "node-2",
			EdgeType: "wikilink",
		}

		// Create edge
		err := repos.Edges.Create(tdb.DB, ctx, edge)
		require.NoError(t, err)

		// Get the created edge to get its ID
		edges, err := repos.Edges.GetBySourceAndTarget(tdb.DB, ctx, "node-1", "node-2")
		require.NoError(t, err)
		require.Len(t, edges, 1)
		createdEdge := edges[0]

		// Delete the edge
		err = repos.Edges.Delete(tdb.DB, ctx, createdEdge.ID)
		assert.NoError(t, err)

		// Verify deletion
		_, err = repos.Edges.GetByID(tdb.DB, ctx, createdEdge.ID)
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("Count", func(t *testing.T) {
		// Clean up from previous tests
		_ = repos.Edges.DeleteAll(tdb.DB, ctx)
		// Get current count
		count, err := repos.Edges.Count(tdb.DB, ctx)
		assert.NoError(t, err)
		currentCount := count

		// Add more edges
		edges := []models.VaultEdge{
			{SourceID: "node-1", TargetID: "node-2", EdgeType: "wikilink"},
			{SourceID: "node-2", TargetID: "node-3", EdgeType: "embed"},
		}
		err = repos.Edges.CreateBatch(tdb.DB, ctx, edges)
		require.NoError(t, err)

		// Verify count increased
		newCount, err := repos.Edges.Count(tdb.DB, ctx)
		assert.NoError(t, err)
		assert.Equal(t, currentCount+2, newCount)
	})
}
