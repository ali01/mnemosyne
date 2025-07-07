// Package postgres provides integration tests for transaction manager
package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// TestTransactionManager runs comprehensive integration tests for the PostgreSQL transaction manager
func TestTransactionManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database and repositories
	tdb, repos := CreateTestRepositories(t)
	defer tdb.Close()

	// Clean up before tests
	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	t.Run("CommitTransaction", func(t *testing.T) {
		// Use transaction to create node and edge atomically
		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			// Create nodes
			node1 := &models.VaultNode{
				ID:       "tx-commit-node-1",
				Title:    "Transaction Node 1",
				FilePath: "/tx/commit/1.md",
			}
			node2 := &models.VaultNode{
				ID:       "tx-commit-node-2",
				Title:    "Transaction Node 2",
				FilePath: "/tx/commit/2.md",
			}

			if err := repos.Nodes.Create(exec, ctx, node1); err != nil {
				return err
			}
			if err := repos.Nodes.Create(exec, ctx, node2); err != nil {
				return err
			}

			// Create edge
			edge := &models.VaultEdge{
				ID:       "tx-commit-edge-1",
				SourceID: node1.ID,
				TargetID: node2.ID,
				EdgeType: "wikilink",
			}
			return repos.Edges.Create(exec, ctx, edge)
		})

		assert.NoError(t, err)

		// Verify data was committed
		node, err := repos.Nodes.GetByID(tdb.DB, ctx, "tx-commit-node-1")
		require.NoError(t, err)
		assert.Equal(t, "Transaction Node 1", node.Title)

		edge, err := repos.Edges.GetByID(tdb.DB, ctx, "tx-commit-edge-1")
		require.NoError(t, err)
		assert.Equal(t, "tx-commit-node-1", edge.SourceID)
		assert.Equal(t, "tx-commit-node-2", edge.TargetID)
	})

	t.Run("RollbackTransaction", func(t *testing.T) {
		// Use transaction that will fail
		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			// Create a node
			node := &models.VaultNode{
				ID:       "tx-rollback-node",
				Title:    "Should be rolled back",
				FilePath: "/tx/rollback/1.md",
			}

			if err := repos.Nodes.Create(exec, ctx, node); err != nil {
				return err
			}

			// Force an error to trigger rollback
			return fmt.Errorf("intentional error for rollback")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "intentional error")

		// Verify node was not created
		_, err = repos.Nodes.GetByID(tdb.DB, ctx, "tx-rollback-node")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("NestedOperations", func(t *testing.T) {
		// Test complex operations within a transaction
		var nodesCreated int
		var edgesCreated int

		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			// Create multiple nodes
			nodes := []models.VaultNode{
				{ID: "nested-1", Title: "Nested 1", FilePath: "/nested/1.md"},
				{ID: "nested-2", Title: "Nested 2", FilePath: "/nested/2.md"},
				{ID: "nested-3", Title: "Nested 3", FilePath: "/nested/3.md"},
			}

			if err := repos.Nodes.CreateBatch(exec, ctx, nodes); err != nil {
				return err
			}
			nodesCreated = len(nodes)

			// Create edges between them
			edges := []models.VaultEdge{
				{ID: "nested-edge-1", SourceID: "nested-1", TargetID: "nested-2", EdgeType: "wikilink"},
				{ID: "nested-edge-2", SourceID: "nested-2", TargetID: "nested-3", EdgeType: "wikilink"},
			}

			if err := repos.Edges.CreateBatch(exec, ctx, edges); err != nil {
				return err
			}
			edgesCreated = len(edges)

			// Verify within transaction
			count, err := repos.Nodes.Count(exec, ctx)
			if err != nil {
				return err
			}
			assert.GreaterOrEqual(t, count, int64(nodesCreated))

			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, nodesCreated)
		assert.Equal(t, 2, edgesCreated)

		// Verify outside transaction
		for i := 1; i <= 3; i++ {
			node, err := repos.Nodes.GetByID(tdb.DB, ctx, fmt.Sprintf("nested-%d", i))
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("Nested %d", i), node.Title)
		}
	})

	t.Run("IsolationBetweenTransactions", func(t *testing.T) {
		// Start two concurrent transactions to test isolation
		tx1Started := make(chan bool)
		tx1Proceed := make(chan bool)
		tx2Done := make(chan bool)

		var tx1Err, tx2Err error

		// Transaction 1: Creates a node and waits
		go func() {
			tx1Err = repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
				exec := tx.Executor()

				node := &models.VaultNode{
					ID:       "isolation-node",
					Title:    "Transaction 1 Node",
					FilePath: "/isolation/tx1.md",
				}

				if err := repos.Nodes.Create(exec, ctx, node); err != nil {
					return err
				}

				// Signal that tx1 has created the node
				tx1Started <- true

				// Wait for signal to proceed
				<-tx1Proceed

				return nil
			})
		}()

		// Wait for tx1 to create its node
		<-tx1Started

		// Transaction 2: Try to read the node (should not see it due to isolation)
		go func() {
			tx2Err = repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
				exec := tx.Executor()

				// Should not see the node from tx1
				_, err := repos.Nodes.GetByID(exec, ctx, "isolation-node")
				if err != nil && repository.IsNotFound(err) {
					// Expected - good isolation
					tx2Done <- true
					return nil
				}

				// If we see the node, isolation is broken
				tx2Done <- false
				return errors.New("isolation broken: saw uncommitted data")
			})
		}()

		// Wait for tx2 to complete
		isolated := <-tx2Done
		assert.True(t, isolated, "Transaction isolation was not maintained")

		// Let tx1 complete
		tx1Proceed <- true

		// Wait for both to finish
		assert.NoError(t, tx1Err)
		assert.NoError(t, tx2Err)

		// Now the node should be visible
		node, err := repos.Nodes.GetByID(tdb.DB, ctx, "isolation-node")
		require.NoError(t, err)
		assert.Equal(t, "Transaction 1 Node", node.Title)
	})

	t.Run("ManualCommit", func(t *testing.T) {
		var committed bool

		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			node := &models.VaultNode{
				ID:       "manual-commit-node",
				Title:    "Manual Commit Test",
				FilePath: "/manual/commit.md",
			}

			if err := repos.Nodes.Create(exec, ctx, node); err != nil {
				return err
			}

			// Manually commit
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			committed = true

			// After manual commit, further operations should fail
			// but we'll just return success
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, committed)

		// Verify node was committed
		node, err := repos.Nodes.GetByID(tdb.DB, ctx, "manual-commit-node")
		require.NoError(t, err)
		assert.Equal(t, "Manual Commit Test", node.Title)
	})

	t.Run("ManualRollback", func(t *testing.T) {
		var rolledBack bool

		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			node := &models.VaultNode{
				ID:       "manual-rollback-node",
				Title:    "Manual Rollback Test",
				FilePath: "/manual/rollback.md",
			}

			if err := repos.Nodes.Create(exec, ctx, node); err != nil {
				return err
			}

			// Manually rollback
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			rolledBack = true

			// Return nil - the transaction is already rolled back
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, rolledBack)

		// Verify node was not created
		_, err = repos.Nodes.GetByID(tdb.DB, ctx, "manual-rollback-node")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("BatchOperationsInTransaction", func(t *testing.T) {
		// Test large batch operations within a transaction
		const batchSize = 100

		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			// Create batch of nodes
			nodes := make([]models.VaultNode, batchSize)
			for i := 0; i < batchSize; i++ {
				nodes[i] = models.VaultNode{
					ID:       fmt.Sprintf("batch-tx-%d", i),
					Title:    fmt.Sprintf("Batch Node %d", i),
					FilePath: fmt.Sprintf("/batch/tx/%d.md", i),
					NodeType: "note",
				}
			}

			if err := repos.Nodes.CreateBatch(exec, ctx, nodes); err != nil {
				return err
			}

			// Create edges between consecutive nodes
			edges := make([]models.VaultEdge, batchSize-1)
			for i := 0; i < batchSize-1; i++ {
				edges[i] = models.VaultEdge{
					ID:       fmt.Sprintf("batch-edge-%d", i),
					SourceID: fmt.Sprintf("batch-tx-%d", i),
					TargetID: fmt.Sprintf("batch-tx-%d", i+1),
					EdgeType: "sequence",
				}
			}

			if err := repos.Edges.CreateBatch(exec, ctx, edges); err != nil {
				return err
			}

			// Verify counts within transaction
			nodeCount, err := repos.Nodes.Count(exec, ctx)
			if err != nil {
				return err
			}
			assert.GreaterOrEqual(t, nodeCount, int64(batchSize))

			edgeCount, err := repos.Edges.Count(exec, ctx)
			if err != nil {
				return err
			}
			assert.GreaterOrEqual(t, edgeCount, int64(batchSize-1))

			return nil
		})

		assert.NoError(t, err)

		// Verify batch was committed
		// Check a few samples
		for i := 0; i < 5; i++ {
			idx := i * 20 // Sample every 20th node
			node, err := repos.Nodes.GetByID(tdb.DB, ctx, fmt.Sprintf("batch-tx-%d", idx))
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("Batch Node %d", idx), node.Title)
		}
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		// Test that specific errors are properly propagated
		testErr := errors.New("specific test error")

		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			// Create a valid node first
			node := &models.VaultNode{
				ID:       "error-prop-node",
				Title:    "Error Propagation Test",
				FilePath: "/error/prop.md",
			}

			if err := repos.Nodes.Create(exec, ctx, node); err != nil {
				return err
			}

			// Return specific error
			return testErr
		})

		// Should get our specific error back
		assert.Equal(t, testErr, err)

		// Node should not exist due to rollback
		_, err = repos.Nodes.GetByID(tdb.DB, ctx, "error-prop-node")
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("ConcurrentTransactions", func(t *testing.T) {
		// Test multiple concurrent transactions
		const numGoroutines = 10
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
					exec := tx.Executor()

					// Each goroutine creates its own node
					node := &models.VaultNode{
						ID:       fmt.Sprintf("concurrent-%d", idx),
						Title:    fmt.Sprintf("Concurrent Node %d", idx),
						FilePath: fmt.Sprintf("/concurrent/%d.md", idx),
					}

					return repos.Nodes.Create(exec, ctx, node)
				})
				errors <- err
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-errors
			assert.NoError(t, err)
		}

		// Verify all nodes were created
		for i := 0; i < numGoroutines; i++ {
			node, err := repos.Nodes.GetByID(tdb.DB, ctx, fmt.Sprintf("concurrent-%d", i))
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("Concurrent Node %d", i), node.Title)
		}
	})
}