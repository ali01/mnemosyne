// Package postgres provides integration tests for position repository
package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// TestPositionRepositoryStateless runs comprehensive integration tests for the PostgreSQL position repository
func TestPositionRepositoryStateless(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database and repositories
	tdb, repos := CreateTestRepositories(t)
	defer tdb.Close()

	// Clean up before tests
	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	// Create test nodes that positions will reference
	testNodes := []models.VaultNode{
		{ID: "pos-node-1", Title: "Position Node 1", FilePath: "/pos/1.md", NodeType: "note"},
		{ID: "pos-node-2", Title: "Position Node 2", FilePath: "/pos/2.md", NodeType: "note"},
		{ID: "pos-node-3", Title: "Position Node 3", FilePath: "/pos/3.md", NodeType: "note"},
	}
	err := repos.Nodes.CreateBatch(tdb.DB, ctx, testNodes)
	require.NoError(t, err)

	t.Run("Upsert_Create", func(t *testing.T) {
		position := &models.NodePosition{
			NodeID:    "pos-node-1",
			X:         100.5,
			Y:         200.5,
			Z:         0,
			Locked:    false,
			UpdatedAt: time.Now(),
		}

		err := repos.Positions.Upsert(tdb.DB, ctx, position)
		assert.NoError(t, err)

		// Verify position was created
		retrieved, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "pos-node-1")
		require.NoError(t, err)
		assert.Equal(t, position.NodeID, retrieved.NodeID)
		assert.Equal(t, position.X, retrieved.X)
		assert.Equal(t, position.Y, retrieved.Y)
		assert.Equal(t, position.Z, retrieved.Z)
		assert.Equal(t, position.Locked, retrieved.Locked)
		assert.NotZero(t, retrieved.UpdatedAt)
	})

	t.Run("Upsert_Update", func(t *testing.T) {
		// Create initial position
		initial := &models.NodePosition{
			NodeID:    "pos-node-2",
			X:         50,
			Y:         75,
			Z:         0,
			Locked:    false,
			UpdatedAt: time.Now(),
		}
		err := repos.Positions.Upsert(tdb.DB, ctx, initial)
		require.NoError(t, err)

		// Get the actual initial timestamp from DB
		initialRetrieved, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "pos-node-2")
		require.NoError(t, err)

		// Sleep to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		// Update the position
		updated := &models.NodePosition{
			NodeID:    "pos-node-2",
			X:         150,
			Y:         175,
			Z:         10,
			Locked:    true,
			UpdatedAt: time.Now().Add(1 * time.Second),
		}
		err = repos.Positions.Upsert(tdb.DB, ctx, updated)
		assert.NoError(t, err)

		// Verify update
		retrieved, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "pos-node-2")
		require.NoError(t, err)
		assert.Equal(t, updated.X, retrieved.X)
		assert.Equal(t, updated.Y, retrieved.Y)
		assert.Equal(t, updated.Z, retrieved.Z)
		assert.Equal(t, updated.Locked, retrieved.Locked)
		assert.True(t, retrieved.UpdatedAt.After(initialRetrieved.UpdatedAt))
	})

	t.Run("Upsert_NonExistentNode", func(t *testing.T) {
		// Try to create position for non-existent node
		position := &models.NodePosition{
			NodeID:    "non-existent-node",
			X:         100,
			Y:         100,
			UpdatedAt: time.Now(),
		}

		err := repos.Positions.Upsert(tdb.DB, ctx, position)
		assert.Error(t, err)
		// Should fail due to foreign key constraint
	})

	t.Run("GetByNodeID_NotFound", func(t *testing.T) {
		_, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "no-position-node")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("UpsertBatch", func(t *testing.T) {
		positions := []models.NodePosition{
			{
				NodeID:    "pos-node-1",
				X:         300,
				Y:         400,
				Z:         5,
				Locked:    true,
				UpdatedAt: time.Now(),
			},
			{
				NodeID:    "pos-node-3",
				X:         500,
				Y:         600,
				Z:         0,
				Locked:    false,
				UpdatedAt: time.Now(),
			},
		}

		err := repos.Positions.UpsertBatch(tdb.DB, ctx, positions)
		assert.NoError(t, err)

		// Verify all positions
		for _, pos := range positions {
			retrieved, err := repos.Positions.GetByNodeID(tdb.DB, ctx, pos.NodeID)
			require.NoError(t, err)
			assert.Equal(t, pos.X, retrieved.X)
			assert.Equal(t, pos.Y, retrieved.Y)
			assert.Equal(t, pos.Z, retrieved.Z)
			assert.Equal(t, pos.Locked, retrieved.Locked)
		}
	})

	t.Run("UpsertBatch_MixedCreateUpdate", func(t *testing.T) {
		// Ensure pos-node-1 has a position
		initial := &models.NodePosition{
			NodeID:    "pos-node-1",
			X:         0,
			Y:         0,
			UpdatedAt: time.Now(),
		}
		err := repos.Positions.Upsert(tdb.DB, ctx, initial)
		require.NoError(t, err)

		// Batch with mix of updates and creates
		positions := []models.NodePosition{
			{
				NodeID:    "pos-node-1", // Update existing
				X:         111,
				Y:         222,
				UpdatedAt: time.Now(),
			},
			{
				NodeID:    "pos-node-2", // Create new
				X:         333,
				Y:         444,
				UpdatedAt: time.Now(),
			},
		}

		err = repos.Positions.UpsertBatch(tdb.DB, ctx, positions)
		assert.NoError(t, err)

		// Verify both
		pos1, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "pos-node-1")
		require.NoError(t, err)
		assert.Equal(t, float64(111), pos1.X)

		pos2, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "pos-node-2")
		require.NoError(t, err)
		assert.Equal(t, float64(333), pos2.X)
	})

	t.Run("UpsertBatch_PartialFailure", func(t *testing.T) {
		// First create a position with known values
		initialPos := &models.NodePosition{
			NodeID:    "pos-node-1",
			X:         100,
			Y:         200,
			UpdatedAt: time.Now(),
		}
		err := repos.Positions.Upsert(tdb.DB, ctx, initialPos)
		require.NoError(t, err)

		// Try to update with a batch that includes an invalid node
		positions := []models.NodePosition{
			{
				NodeID:    "pos-node-1",
				X:         700,
				Y:         800,
				UpdatedAt: time.Now(),
			},
			{
				NodeID:    "invalid-node", // This will fail
				X:         900,
				Y:         1000,
				UpdatedAt: time.Now(),
			},
		}

		err = repos.Positions.UpsertBatch(tdb.DB, ctx, positions)
		assert.Error(t, err)

		// In a transaction, nothing should be updated
		pos1, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "pos-node-1")
		require.NoError(t, err)
		// Position should still have original values due to transaction rollback
		assert.Equal(t, float64(100), pos1.X)
		assert.Equal(t, float64(200), pos1.Y)
	})

	t.Run("GetAll", func(t *testing.T) {
		// Clean positions and create known set
		require.NoError(t, tdb.CleanTables(ctx))

		// Recreate nodes
		err := repos.Nodes.CreateBatch(tdb.DB, ctx, testNodes)
		require.NoError(t, err)

		// Create positions
		positions := []models.NodePosition{
			{NodeID: "pos-node-1", X: 10, Y: 20, UpdatedAt: time.Now()},
			{NodeID: "pos-node-2", X: 30, Y: 40, UpdatedAt: time.Now()},
			{NodeID: "pos-node-3", X: 50, Y: 60, UpdatedAt: time.Now()},
		}
		err = repos.Positions.UpsertBatch(tdb.DB, ctx, positions)
		require.NoError(t, err)

		// Get all
		allPositions, err := repos.Positions.GetAll(tdb.DB, ctx)
		assert.NoError(t, err)
		assert.Len(t, allPositions, 3)

		// Verify all node IDs are present
		nodeIDs := make(map[string]bool)
		for _, pos := range allPositions {
			nodeIDs[pos.NodeID] = true
		}
		assert.True(t, nodeIDs["pos-node-1"])
		assert.True(t, nodeIDs["pos-node-2"])
		assert.True(t, nodeIDs["pos-node-3"])
	})

	t.Run("DeleteByNodeID", func(t *testing.T) {
		// Create a position
		position := &models.NodePosition{
			NodeID:    "pos-node-1",
			X:         123,
			Y:         456,
			UpdatedAt: time.Now(),
		}
		err := repos.Positions.Upsert(tdb.DB, ctx, position)
		require.NoError(t, err)

		// Verify it exists
		_, err = repos.Positions.GetByNodeID(tdb.DB, ctx, "pos-node-1")
		require.NoError(t, err)

		// Delete
		err = repos.Positions.DeleteByNodeID(tdb.DB, ctx, "pos-node-1")
		assert.NoError(t, err)

		// Verify deletion
		_, err = repos.Positions.GetByNodeID(tdb.DB, ctx, "pos-node-1")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("DeleteByNodeID_NotFound", func(t *testing.T) {
		// Delete non-existent position (should be idempotent)
		err := repos.Positions.DeleteByNodeID(tdb.DB, ctx, "non-existent-position")
		assert.NoError(t, err)
	})

	t.Run("CascadeDelete", func(t *testing.T) {
		// Create node with position
		node := &models.VaultNode{
			ID:       "cascade-node",
			Title:    "Cascade Test",
			FilePath: "/cascade/test.md",
		}
		err := repos.Nodes.Create(tdb.DB, ctx, node)
		require.NoError(t, err)

		position := &models.NodePosition{
			NodeID:    "cascade-node",
			X:         100,
			Y:         200,
			UpdatedAt: time.Now(),
		}
		err = repos.Positions.Upsert(tdb.DB, ctx, position)
		require.NoError(t, err)

		// Delete the node
		err = repos.Nodes.Delete(tdb.DB, ctx, "cascade-node")
		require.NoError(t, err)

		// Position should be gone too (CASCADE)
		_, err = repos.Positions.GetByNodeID(tdb.DB, ctx, "cascade-node")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("WithTransaction", func(t *testing.T) {
		// Test positions in transaction
		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			// Create node in transaction
			node := &models.VaultNode{
				ID:       "tx-pos-node",
				Title:    "Transaction Position Node",
				FilePath: "/tx/pos.md",
			}
			if err := repos.Nodes.Create(exec, ctx, node); err != nil {
				return err
			}

			// Create position in same transaction
			position := &models.NodePosition{
				NodeID:    "tx-pos-node",
				X:         777,
				Y:         888,
				UpdatedAt: time.Now(),
			}
			if err := repos.Positions.Upsert(exec, ctx, position); err != nil {
				return err
			}

			// Verify within transaction
			retrieved, err := repos.Positions.GetByNodeID(exec, ctx, "tx-pos-node")
			if err != nil {
				return err
			}
			assert.Equal(t, float64(777), retrieved.X)

			return nil
		})

		require.NoError(t, err)

		// Verify committed
		pos, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "tx-pos-node")
		require.NoError(t, err)
		assert.Equal(t, float64(777), pos.X)
	})

	t.Run("WithTransaction_Rollback", func(t *testing.T) {
		// Ensure node exists
		node := &models.VaultNode{
			ID:       "rollback-pos-node",
			Title:    "Rollback Position Node",
			FilePath: "/rollback/pos.md",
		}
		err := repos.Nodes.Create(tdb.DB, ctx, node)
		require.NoError(t, err)

		// Try to create position in failed transaction
		err = repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			position := &models.NodePosition{
				NodeID:    "rollback-pos-node",
				X:         999,
				Y:         1111,
				UpdatedAt: time.Now(),
			}
			if err := repos.Positions.Upsert(exec, ctx, position); err != nil {
				return err
			}

			// Force rollback
			return assert.AnError
		})

		assert.Error(t, err)

		// Position should not exist
		_, err = repos.Positions.GetByNodeID(tdb.DB, ctx, "rollback-pos-node")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("LockedPositions", func(t *testing.T) {
		// Test locked position behavior
		node := &models.VaultNode{
			ID:       "locked-node",
			Title:    "Locked Position Node",
			FilePath: "/locked/node.md",
		}
		err := repos.Nodes.Create(tdb.DB, ctx, node)
		require.NoError(t, err)

		// Create locked position
		position := &models.NodePosition{
			NodeID:    "locked-node",
			X:         100,
			Y:         100,
			Locked:    true,
			UpdatedAt: time.Now(),
		}
		err = repos.Positions.Upsert(tdb.DB, ctx, position)
		require.NoError(t, err)

		// Verify locked state persists
		retrieved, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "locked-node")
		require.NoError(t, err)
		assert.True(t, retrieved.Locked)
		assert.Equal(t, float64(100), retrieved.X)
		assert.Equal(t, float64(100), retrieved.Y)
	})

	t.Run("PrecisionHandling", func(t *testing.T) {
		// Test floating point precision
		node := &models.VaultNode{
			ID:       "precision-node",
			Title:    "Precision Test",
			FilePath: "/precision/test.md",
		}
		err := repos.Nodes.Create(tdb.DB, ctx, node)
		require.NoError(t, err)

		position := &models.NodePosition{
			NodeID:    "precision-node",
			X:         123.456789,
			Y:         -987.654321,
			Z:         0.000001,
			UpdatedAt: time.Now(),
		}
		err = repos.Positions.Upsert(tdb.DB, ctx, position)
		require.NoError(t, err)

		retrieved, err := repos.Positions.GetByNodeID(tdb.DB, ctx, "precision-node")
		require.NoError(t, err)
		assert.InDelta(t, 123.456789, retrieved.X, 0.000001)
		assert.InDelta(t, -987.654321, retrieved.Y, 0.000001)
		assert.InDelta(t, 0.000001, retrieved.Z, 0.0000001)
	})
}
