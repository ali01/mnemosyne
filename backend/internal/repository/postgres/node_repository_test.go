// Package postgres provides integration tests for node repository
package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// TestNodeRepositoryStateless runs comprehensive integration tests for the PostgreSQL node repository
func TestNodeRepositoryStateless(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database and repositories
	tdb, repos := CreateTestRepositories(t)
	defer tdb.Close()

	// Clean up before tests
	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	t.Run("Create", func(t *testing.T) {
		node := &models.VaultNode{
			ID:       "test-node-1",
			Title:    "Test Node",
			FilePath: "/test/path.md",
			NodeType: "note",
			Tags:     []string{"test", "integration"},
			Content:  "Test content",
			Metadata: map[string]interface{}{
				"author": "test",
			},
		}

		err := repos.Nodes.Create(tdb.DB, ctx, node)
		assert.NoError(t, err)

		// Verify the node was created
		retrieved, err := repos.Nodes.GetByID(tdb.DB, ctx, node.ID)
		require.NoError(t, err)
		assert.Equal(t, node.ID, retrieved.ID)
		assert.Equal(t, node.Title, retrieved.Title)
		assert.Equal(t, node.Tags, retrieved.Tags)
		assert.Equal(t, node.Content, retrieved.Content)
		assert.Equal(t, node.NodeType, retrieved.NodeType)
		assert.NotZero(t, retrieved.CreatedAt)
		assert.NotZero(t, retrieved.UpdatedAt)
	})

	t.Run("Create_DuplicateID", func(t *testing.T) {
		node := &models.VaultNode{
			ID:       "duplicate-node",
			Title:    "First Node",
			FilePath: "/test/dup1.md",
		}

		// Create first node
		err := repos.Nodes.Create(tdb.DB, ctx, node)
		require.NoError(t, err)

		// Try to create duplicate
		node2 := &models.VaultNode{
			ID:       "duplicate-node",
			Title:    "Second Node",
			FilePath: "/test/dup2.md",
		}
		err = repos.Nodes.Create(tdb.DB, ctx, node2)
		assert.Error(t, err)
		assert.True(t, repository.IsDuplicateKey(err))
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		_, err := repos.Nodes.GetByID(tdb.DB, ctx, "non-existent-node")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("Update", func(t *testing.T) {
		node := &models.VaultNode{
			ID:       "test-node-update",
			Title:    "Original Title",
			FilePath: "/test/update.md",
			NodeType: "note",
		}

		// Create the node
		err := repos.Nodes.Create(tdb.DB, ctx, node)
		require.NoError(t, err)

		// Update the node
		node.Title = "Updated Title"
		node.Tags = []string{"updated", "modified"}
		node.Content = "Updated content"
		node.NodeType = "reference"
		
		time.Sleep(10 * time.Millisecond) // Ensure UpdatedAt changes
		err = repos.Nodes.Update(tdb.DB, ctx, node)
		assert.NoError(t, err)

		// Verify the update
		retrieved, err := repos.Nodes.GetByID(tdb.DB, ctx, node.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", retrieved.Title)
		assert.Equal(t, []string{"updated", "modified"}, retrieved.Tags)
		assert.Equal(t, "Updated content", retrieved.Content)
		assert.Equal(t, "reference", retrieved.NodeType)
		assert.True(t, retrieved.UpdatedAt.After(retrieved.CreatedAt))
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		node := &models.VaultNode{
			ID:    "non-existent-update",
			Title: "Ghost Node",
		}
		err := repos.Nodes.Update(tdb.DB, ctx, node)
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("Delete", func(t *testing.T) {
		node := &models.VaultNode{
			ID:       "test-node-delete",
			Title:    "To Be Deleted",
			FilePath: "/test/delete.md",
		}

		// Create and verify
		err := repos.Nodes.Create(tdb.DB, ctx, node)
		require.NoError(t, err)
		
		_, err = repos.Nodes.GetByID(tdb.DB, ctx, node.ID)
		require.NoError(t, err)

		// Delete
		err = repos.Nodes.Delete(tdb.DB, ctx, node.ID)
		assert.NoError(t, err)

		// Verify deletion
		_, err = repos.Nodes.GetByID(tdb.DB, ctx, node.ID)
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		err := repos.Nodes.Delete(tdb.DB, ctx, "non-existent-delete")
		// Delete is idempotent - no error expected
		assert.NoError(t, err)
	})

	t.Run("CreateBatch", func(t *testing.T) {
		nodes := []models.VaultNode{
			{ID: "batch-1", Title: "Batch Node 1", FilePath: "/batch/1.md"},
			{ID: "batch-2", Title: "Batch Node 2", FilePath: "/batch/2.md"},
			{ID: "batch-3", Title: "Batch Node 3", FilePath: "/batch/3.md"},
		}

		err := repos.Nodes.CreateBatch(tdb.DB, ctx, nodes)
		assert.NoError(t, err)

		// Verify all were created
		for _, node := range nodes {
			retrieved, err := repos.Nodes.GetByID(tdb.DB, ctx, node.ID)
			require.NoError(t, err)
			assert.Equal(t, node.Title, retrieved.Title)
		}
	})

	t.Run("CreateBatch_PartialFailure", func(t *testing.T) {
		// Create one node first
		existing := &models.VaultNode{
			ID:       "batch-existing",
			Title:    "Existing Node",
			FilePath: "/batch/existing.md",
		}
		err := repos.Nodes.Create(tdb.DB, ctx, existing)
		require.NoError(t, err)

		// Try batch with duplicate
		nodes := []models.VaultNode{
			{ID: "batch-new-1", Title: "New Node 1", FilePath: "/batch/new1.md"},
			{ID: "batch-existing", Title: "Duplicate", FilePath: "/batch/dup.md"}, // This will fail
			{ID: "batch-new-2", Title: "New Node 2", FilePath: "/batch/new2.md"},
		}

		err = repos.Nodes.CreateBatch(tdb.DB, ctx, nodes)
		assert.Error(t, err)

		// In a transaction, none should be created
		_, err = repos.Nodes.GetByID(tdb.DB, ctx, "batch-new-1")
		assert.Error(t, err)
		_, err = repos.Nodes.GetByID(tdb.DB, ctx, "batch-new-2")
		assert.Error(t, err)
	})

	t.Run("UpsertBatch", func(t *testing.T) {
		// Create initial nodes
		initial := []models.VaultNode{
			{ID: "upsert-1", Title: "Initial 1", FilePath: "/upsert/1.md", Content: "v1"},
			{ID: "upsert-2", Title: "Initial 2", FilePath: "/upsert/2.md", Content: "v1"},
		}
		err := repos.Nodes.CreateBatch(tdb.DB, ctx, initial)
		require.NoError(t, err)

		// Upsert with updates and new nodes
		upserted := []models.VaultNode{
			{ID: "upsert-1", Title: "Updated 1", FilePath: "/upsert/1.md", Content: "v2"}, // Update
			{ID: "upsert-3", Title: "New 3", FilePath: "/upsert/3.md", Content: "v1"},     // Insert
		}

		err = repos.Nodes.UpsertBatch(tdb.DB, ctx, upserted)
		assert.NoError(t, err)

		// Verify updates
		node1, err := repos.Nodes.GetByID(tdb.DB, ctx, "upsert-1")
		require.NoError(t, err)
		assert.Equal(t, "Updated 1", node1.Title)
		assert.Equal(t, "v2", node1.Content)

		// Verify unchanged
		node2, err := repos.Nodes.GetByID(tdb.DB, ctx, "upsert-2")
		require.NoError(t, err)
		assert.Equal(t, "Initial 2", node2.Title)

		// Verify new
		node3, err := repos.Nodes.GetByID(tdb.DB, ctx, "upsert-3")
		require.NoError(t, err)
		assert.Equal(t, "New 3", node3.Title)
	})

	t.Run("GetAll", func(t *testing.T) {
		// Clean and create test data
		require.NoError(t, tdb.CleanTables(ctx))
		
		nodes := []models.VaultNode{
			{ID: "all-1", Title: "Node 1", FilePath: "/all/1.md"},
			{ID: "all-2", Title: "Node 2", FilePath: "/all/2.md"},
			{ID: "all-3", Title: "Node 3", FilePath: "/all/3.md"},
		}
		err := repos.Nodes.CreateBatch(tdb.DB, ctx, nodes)
		require.NoError(t, err)

		// Test pagination
		result, err := repos.Nodes.GetAll(tdb.DB, ctx, 2, 0)
		assert.NoError(t, err)
		assert.Len(t, result, 2)

		result, err = repos.Nodes.GetAll(tdb.DB, ctx, 2, 2)
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		// Get all
		result, err = repos.Nodes.GetAll(tdb.DB, ctx, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("GetByIDs", func(t *testing.T) {
		// Create test nodes
		nodes := []models.VaultNode{
			{ID: "ids-1", Title: "Node 1", FilePath: "/ids/1.md"},
			{ID: "ids-2", Title: "Node 2", FilePath: "/ids/2.md"},
			{ID: "ids-3", Title: "Node 3", FilePath: "/ids/3.md"},
		}
		err := repos.Nodes.CreateBatch(tdb.DB, ctx, nodes)
		require.NoError(t, err)

		// Get subset
		result, err := repos.Nodes.GetByIDs(tdb.DB, ctx, []string{"ids-1", "ids-3", "non-existent"})
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		
		ids := make([]string, len(result))
		for i, node := range result {
			ids[i] = node.ID
		}
		assert.Contains(t, ids, "ids-1")
		assert.Contains(t, ids, "ids-3")
	})

	t.Run("GetByType", func(t *testing.T) {
		// Clean and create typed nodes
		require.NoError(t, tdb.CleanTables(ctx))
		
		nodes := []models.VaultNode{
			{ID: "type-1", Title: "Note 1", NodeType: "note", FilePath: "/type/1.md"},
			{ID: "type-2", Title: "Reference 1", NodeType: "reference", FilePath: "/type/2.md"},
			{ID: "type-3", Title: "Note 2", NodeType: "note", FilePath: "/type/3.md"},
			{ID: "type-4", Title: "Task 1", NodeType: "task", FilePath: "/type/4.md"},
		}
		err := repos.Nodes.CreateBatch(tdb.DB, ctx, nodes)
		require.NoError(t, err)

		// Get by type
		notes, err := repos.Nodes.GetByType(tdb.DB, ctx, "note")
		assert.NoError(t, err)
		assert.Len(t, notes, 2)
		for _, node := range notes {
			assert.Equal(t, "note", node.NodeType)
		}

		refs, err := repos.Nodes.GetByType(tdb.DB, ctx, "reference")
		assert.NoError(t, err)
		assert.Len(t, refs, 1)
		assert.Equal(t, "Reference 1", refs[0].Title)
	})

	t.Run("GetByPath", func(t *testing.T) {
		node := &models.VaultNode{
			ID:       "path-test",
			Title:    "Path Test",
			FilePath: "/unique/path/to/file.md",
		}
		err := repos.Nodes.Create(tdb.DB, ctx, node)
		require.NoError(t, err)

		// Find by path
		found, err := repos.Nodes.GetByPath(tdb.DB, ctx, "/unique/path/to/file.md")
		assert.NoError(t, err)
		assert.Equal(t, node.ID, found.ID)

		// Not found
		_, err = repos.Nodes.GetByPath(tdb.DB, ctx, "/non/existent/path.md")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("Search", func(t *testing.T) {
		// Clean and create searchable nodes
		require.NoError(t, tdb.CleanTables(ctx))
		
		nodes := []models.VaultNode{
			{
				ID:       "search-1",
				Title:    "PostgreSQL Tutorial",
				Content:  "Learn about PostgreSQL database features",
				FilePath: "/search/1.md",
			},
			{
				ID:       "search-2",
				Title:    "MySQL Guide",
				Content:  "MySQL is another database system",
				FilePath: "/search/2.md",
			},
			{
				ID:       "search-3",
				Title:    "Database Design",
				Content:  "Best practices for designing databases with PostgreSQL",
				FilePath: "/search/3.md",
			},
		}
		err := repos.Nodes.CreateBatch(tdb.DB, ctx, nodes)
		require.NoError(t, err)

		// Search by title
		results, err := repos.Nodes.Search(tdb.DB, ctx, "PostgreSQL")
		assert.NoError(t, err)
		assert.Len(t, results, 2) // Should find search-1 and search-3

		// Search by content
		results, err = repos.Nodes.Search(tdb.DB, ctx, "database")
		assert.NoError(t, err)
		assert.Len(t, results, 3) // All three contain "database"

		// No results
		results, err = repos.Nodes.Search(tdb.DB, ctx, "nonexistentterm")
		assert.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("Count", func(t *testing.T) {
		// Clean and add known number of nodes
		require.NoError(t, tdb.CleanTables(ctx))
		
		nodes := []models.VaultNode{
			{ID: "count-1", Title: "Node 1", FilePath: "/count/1.md"},
			{ID: "count-2", Title: "Node 2", FilePath: "/count/2.md"},
			{ID: "count-3", Title: "Node 3", FilePath: "/count/3.md"},
		}
		err := repos.Nodes.CreateBatch(tdb.DB, ctx, nodes)
		require.NoError(t, err)

		count, err := repos.Nodes.Count(tdb.DB, ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("DeleteAll", func(t *testing.T) {
		// Create some nodes
		nodes := []models.VaultNode{
			{ID: "delete-all-1", Title: "Node 1", FilePath: "/del/1.md"},
			{ID: "delete-all-2", Title: "Node 2", FilePath: "/del/2.md"},
		}
		err := repos.Nodes.CreateBatch(tdb.DB, ctx, nodes)
		require.NoError(t, err)

		// Verify they exist
		count, err := repos.Nodes.Count(tdb.DB, ctx)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))

		// Delete all
		err = repos.Nodes.DeleteAll(tdb.DB, ctx)
		assert.NoError(t, err)

		// Verify empty
		count, err = repos.Nodes.Count(tdb.DB, ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("WithTransaction", func(t *testing.T) {
		// Test using nodes in a transaction
		err := db.WithTransaction(tdb.DB, ctx, func(tx *sqlx.Tx) error {
			node := &models.VaultNode{
				ID:       "tx-node",
				Title:    "Transaction Node",
				FilePath: "/tx/node.md",
			}
			
			// Create using transaction
			if err := repos.Nodes.Create(tx, ctx, node); err != nil {
				return err
			}

			// Verify within transaction
			retrieved, err := repos.Nodes.GetByID(tx, ctx, node.ID)
			if err != nil {
				return err
			}
			assert.Equal(t, node.Title, retrieved.Title)

			// Update within transaction
			node.Title = "Updated in Transaction"
			return repos.Nodes.Update(tx, ctx, node)
		})

		require.NoError(t, err)

		// Verify changes persisted
		node, err := repos.Nodes.GetByID(tdb.DB, ctx, "tx-node")
		require.NoError(t, err)
		assert.Equal(t, "Updated in Transaction", node.Title)
	})

	t.Run("WithTransaction_Rollback", func(t *testing.T) {
		// Test rollback on error
		err := db.WithTransaction(tdb.DB, ctx, func(tx *sqlx.Tx) error {
			node := &models.VaultNode{
				ID:       "rollback-node",
				Title:    "Should be rolled back",
				FilePath: "/rollback/node.md",
			}
			
			// Create in transaction
			if err := repos.Nodes.Create(tx, ctx, node); err != nil {
				return err
			}

			// Force error to trigger rollback
			return assert.AnError
		})

		assert.Error(t, err)

		// Verify node was not created
		_, err = repos.Nodes.GetByID(tdb.DB, ctx, "rollback-node")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})
}