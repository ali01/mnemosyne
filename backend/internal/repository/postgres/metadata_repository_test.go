// Package postgres provides integration tests for metadata repository
package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// TestMetadataRepositoryStateless runs comprehensive integration tests for the PostgreSQL metadata repository
func TestMetadataRepositoryStateless(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database and repositories
	tdb, repos := CreateTestRepositories(t)
	defer tdb.Close()

	// Clean up before tests
	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	t.Run("SetMetadata_Create", func(t *testing.T) {
		metadata := &models.VaultMetadata{
			Key:       "test.key",
			Value:     "test value",
			UpdatedAt: time.Now(),
		}

		err := repos.Metadata.SetMetadata(tdb.DB, ctx, metadata)
		assert.NoError(t, err)

		// Verify metadata was created
		retrieved, err := repos.Metadata.GetMetadata(tdb.DB, ctx, "test.key")
		require.NoError(t, err)
		assert.Equal(t, metadata.Key, retrieved.Key)
		assert.Equal(t, metadata.Value, retrieved.Value)
		assert.NotZero(t, retrieved.UpdatedAt)
	})

	t.Run("SetMetadata_Update", func(t *testing.T) {
		// Create initial metadata
		initial := &models.VaultMetadata{
			Key:       "update.test",
			Value:     "initial value",
		}
		err := repos.Metadata.SetMetadata(tdb.DB, ctx, initial)
		require.NoError(t, err)
		
		// Get the actual initial timestamp
		initialRetrieved, err := repos.Metadata.GetMetadata(tdb.DB, ctx, "update.test")
		require.NoError(t, err)

		// Update the value
		time.Sleep(10 * time.Millisecond) // Ensure time difference
		updated := &models.VaultMetadata{
			Key:       "update.test",
			Value:     "updated value",
		}
		err = repos.Metadata.SetMetadata(tdb.DB, ctx, updated)
		assert.NoError(t, err)

		// Verify update
		retrieved, err := repos.Metadata.GetMetadata(tdb.DB, ctx, "update.test")
		require.NoError(t, err)
		assert.Equal(t, "updated value", retrieved.Value)
		assert.True(t, retrieved.UpdatedAt.After(initialRetrieved.UpdatedAt))
	})

	t.Run("GetMetadata_NotFound", func(t *testing.T) {
		_, err := repos.Metadata.GetMetadata(tdb.DB, ctx, "non.existent.key")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("GetAllMetadata", func(t *testing.T) {
		// Clean and create known set
		require.NoError(t, tdb.CleanTables(ctx))

		// Set multiple metadata entries
		entries := []models.VaultMetadata{
			{Key: "app.version", Value: "1.0.0", UpdatedAt: time.Now()},
			{Key: "app.environment", Value: "test", UpdatedAt: time.Now()},
			{Key: "vault.last_sync", Value: "2024-01-01T00:00:00Z", UpdatedAt: time.Now()},
		}

		for _, entry := range entries {
			err := repos.Metadata.SetMetadata(tdb.DB, ctx, &entry)
			require.NoError(t, err)
		}

		// Get all
		all, err := repos.Metadata.GetAllMetadata(tdb.DB, ctx)
		assert.NoError(t, err)
		assert.Len(t, all, 3)

		// Verify all keys are present
		keys := make(map[string]string)
		for _, meta := range all {
			keys[meta.Key] = meta.Value
		}
		assert.Equal(t, "1.0.0", keys["app.version"])
		assert.Equal(t, "test", keys["app.environment"])
		assert.Equal(t, "2024-01-01T00:00:00Z", keys["vault.last_sync"])
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		// Test handling of special characters in keys and values
		testCases := []models.VaultMetadata{
			{Key: "special.chars!@#$", Value: "value with spaces", UpdatedAt: time.Now()},
			{Key: "unicode.测试", Value: "中文值", UpdatedAt: time.Now()},
			{Key: "json.data", Value: `{"nested": {"key": "value"}}`, UpdatedAt: time.Now()},
			{Key: "multiline", Value: "line1\nline2\nline3", UpdatedAt: time.Now()},
		}

		for _, tc := range testCases {
			err := repos.Metadata.SetMetadata(tdb.DB, ctx, &tc)
			require.NoError(t, err)

			retrieved, err := repos.Metadata.GetMetadata(tdb.DB, ctx, tc.Key)
			require.NoError(t, err)
			assert.Equal(t, tc.Value, retrieved.Value)
		}
	})

	t.Run("CreateParseRecord", func(t *testing.T) {
		record := &models.ParseHistory{
			StartedAt:   time.Now(),
			CompletedAt: nil,
			Status:      models.ParseStatusRunning,
			Stats: models.JSONStats{
				TotalFiles:      150,
				ParsedFiles:     150,
				TotalNodes:      100,
				TotalEdges:      200,
				DurationMS:      5500,
				UnresolvedLinks: 5,
			},
		}

		err := repos.Metadata.CreateParseRecord(tdb.DB, ctx, record)
		assert.NoError(t, err)
		assert.NotEmpty(t, record.ID)
		assert.NotZero(t, record.StartedAt)
	})

	t.Run("GetLatestParse", func(t *testing.T) {
		// Clean up any existing parse records
		_, _ = tdb.ExecContext(ctx, "DELETE FROM parse_history")
		
		// Create multiple parse records
		now := time.Now()
		records := []models.ParseHistory{
			{
				StartedAt:   now.Add(-2 * time.Hour),
				CompletedAt: ptrTime(now.Add(-1*time.Hour - 50*time.Minute)),
				Status:      models.ParseStatusCompleted,
				Stats: models.JSONStats{
					TotalNodes: 10,
					TotalEdges: 20,
				},
			},
			{
				StartedAt:   now.Add(-1 * time.Hour),
				CompletedAt: ptrTime(now.Add(-30 * time.Minute)),
				Status:      models.ParseStatusCompleted,
				Stats: models.JSONStats{
					TotalNodes: 15,
					TotalEdges: 25,
				},
			},
			{
				StartedAt:   now.Add(-10 * time.Minute),
				CompletedAt: nil,
				Status:      models.ParseStatusRunning,
				Stats: models.JSONStats{
					TotalNodes: 5,
					TotalEdges: 10,
				},
			},
		}

		for _, record := range records {
			r := record // Copy to avoid pointer issues
			err := repos.Metadata.CreateParseRecord(tdb.DB, ctx, &r)
			require.NoError(t, err)
		}

		// Get latest
		latest, err := repos.Metadata.GetLatestParse(tdb.DB, ctx)
		assert.NoError(t, err)
		require.NotNil(t, latest)
		
		// Debug output
		t.Logf("Latest parse: Status=%s, TotalNodes=%d, TotalEdges=%d", 
			latest.Status, latest.Stats.TotalNodes, latest.Stats.TotalEdges)
		
		assert.Equal(t, models.ParseStatusRunning, latest.Status)
		// Check that we got the most recent record (the one with TotalNodes = 5)
		assert.Equal(t, 5, latest.Stats.TotalNodes)
		assert.Equal(t, 10, latest.Stats.TotalEdges)
		assert.Nil(t, latest.CompletedAt)
	})

	t.Run("GetLatestParse_NoRecords", func(t *testing.T) {
		// Clean parse history
		require.NoError(t, tdb.CleanTables(ctx))

		_, err := repos.Metadata.GetLatestParse(tdb.DB, ctx)
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("GetParseHistory", func(t *testing.T) {
		// Clean and create known history
		require.NoError(t, tdb.CleanTables(ctx))

		// Create parse records
		for i := 0; i < 10; i++ {
			record := &models.ParseHistory{
				StartedAt:   time.Now().Add(time.Duration(-i) * time.Hour),
				CompletedAt: ptrTime(time.Now().Add(time.Duration(-i)*time.Hour + 30*time.Minute)),
				Status:      models.ParseStatusCompleted,
				Stats: models.JSONStats{
					TotalNodes: i * 10,
					TotalEdges: i * 20,
					DurationMS: int64(i * 1000),
				},
			}
			err := repos.Metadata.CreateParseRecord(tdb.DB, ctx, record)
			require.NoError(t, err)
		}

		// Get limited history
		history, err := repos.Metadata.GetParseHistory(tdb.DB, ctx, 5)
		assert.NoError(t, err)
		assert.Len(t, history, 5)

		// Verify ordering (most recent first)
		for i := 1; i < len(history); i++ {
			assert.True(t, history[i-1].StartedAt.After(history[i].StartedAt))
		}
	})

	t.Run("UpdateParseStatus", func(t *testing.T) {
		// Create a parse record
		record := &models.ParseHistory{
			StartedAt: time.Now(),
			Status:    models.ParseStatusRunning,
			Stats: models.JSONStats{
				TotalNodes: 50,
				TotalEdges: 100,
			},
		}
		err := repos.Metadata.CreateParseRecord(tdb.DB, ctx, record)
		require.NoError(t, err)

		// Update status to completed
		err = repos.Metadata.UpdateParseStatus(tdb.DB, ctx, record.ID, models.ParseStatusCompleted)
		assert.NoError(t, err)

		// Verify update
		latest, err := repos.Metadata.GetLatestParse(tdb.DB, ctx)
		require.NoError(t, err)
		assert.Equal(t, models.ParseStatusCompleted, latest.Status)
		assert.Equal(t, record.ID, latest.ID)
	})

	t.Run("UpdateParseStatus_Failed", func(t *testing.T) {
		// Create a parse record
		record := &models.ParseHistory{
			StartedAt: time.Now(),
			Status:    models.ParseStatusRunning,
			Stats: models.JSONStats{
				TotalNodes: 25,
				TotalEdges: 50,
			},
		}
		err := repos.Metadata.CreateParseRecord(tdb.DB, ctx, record)
		require.NoError(t, err)

		// Update status to failed
		err = repos.Metadata.UpdateParseStatus(tdb.DB, ctx, record.ID, models.ParseStatusFailed)
		assert.NoError(t, err)

		// Verify update
		latest, err := repos.Metadata.GetLatestParse(tdb.DB, ctx)
		require.NoError(t, err)
		assert.Equal(t, models.ParseStatusFailed, latest.Status)
	})

	t.Run("UpdateParseStatus_NotFound", func(t *testing.T) {
		err := repos.Metadata.UpdateParseStatus(tdb.DB, ctx, "non-existent-id", models.ParseStatusCompleted)
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("ParseHistoryWithErrors", func(t *testing.T) {
		record := &models.ParseHistory{
			StartedAt:   time.Now(),
			CompletedAt: ptrTime(time.Now().Add(1 * time.Minute)),
			Status:      models.ParseStatusFailed,
			Stats: models.JSONStats{
				TotalNodes:      75,
				TotalEdges:      150,
				DurationMS:      60000,
				UnresolvedLinks: 3,
			},
			Error: ptrString("Failed to parse file: /path/to/file1.md; Invalid frontmatter in: /path/to/file2.md; Circular reference detected"),
		}

		err := repos.Metadata.CreateParseRecord(tdb.DB, ctx, record)
		assert.NoError(t, err)

		// Retrieve and verify errors
		retrieved, err := repos.Metadata.GetLatestParse(tdb.DB, ctx)
		require.NoError(t, err)
		assert.Equal(t, models.ParseStatusFailed, retrieved.Status)
		assert.NotNil(t, retrieved.Error)
		assert.Contains(t, *retrieved.Error, "file1.md")
	})

	t.Run("WithTransaction", func(t *testing.T) {
		// Test metadata operations in transaction
		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			// Set metadata in transaction
			meta := &models.VaultMetadata{
				Key:       "tx.test",
				Value:     "transaction value",
				UpdatedAt: time.Now(),
			}
			if err := repos.Metadata.SetMetadata(exec, ctx, meta); err != nil {
				return err
			}

			// Create parse record in same transaction
			record := &models.ParseHistory{
				StartedAt: time.Now(),
				Status:    models.ParseStatusRunning,
				Stats: models.JSONStats{
					TotalNodes: 123,
					TotalEdges: 456,
				},
			}
			if err := repos.Metadata.CreateParseRecord(exec, ctx, record); err != nil {
				return err
			}

			// Verify within transaction
			retrieved, err := repos.Metadata.GetMetadata(exec, ctx, "tx.test")
			if err != nil {
				return err
			}
			assert.Equal(t, "transaction value", retrieved.Value)

			return nil
		})

		require.NoError(t, err)

		// Verify committed
		meta, err := repos.Metadata.GetMetadata(tdb.DB, ctx, "tx.test")
		require.NoError(t, err)
		assert.Equal(t, "transaction value", meta.Value)
	})

	t.Run("WithTransaction_Rollback", func(t *testing.T) {
		// Test rollback
		err := repos.Transactions.WithTransaction(ctx, func(tx repository.Transaction) error {
			exec := tx.Executor()

			meta := &models.VaultMetadata{
				Key:       "rollback.test",
				Value:     "should not persist",
				UpdatedAt: time.Now(),
			}
			if err := repos.Metadata.SetMetadata(exec, ctx, meta); err != nil {
				return err
			}

			// Force rollback
			return assert.AnError
		})

		assert.Error(t, err)

		// Metadata should not exist
		_, err = repos.Metadata.GetMetadata(tdb.DB, ctx, "rollback.test")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("LargeValues", func(t *testing.T) {
		// Test with large values
		largeValue := ""
		for i := 0; i < 1000; i++ {
			largeValue += fmt.Sprintf("Line %d: This is a test of large metadata values.\n", i)
		}

		metadata := &models.VaultMetadata{
			Key:       "large.value",
			Value:     largeValue,
			UpdatedAt: time.Now(),
		}

		err := repos.Metadata.SetMetadata(tdb.DB, ctx, metadata)
		assert.NoError(t, err)

		retrieved, err := repos.Metadata.GetMetadata(tdb.DB, ctx, "large.value")
		require.NoError(t, err)
		assert.Equal(t, largeValue, retrieved.Value)
		assert.True(t, len(retrieved.Value) > 10000)
	})

	t.Run("ConcurrentMetadataUpdates", func(t *testing.T) {
		// Test concurrent updates to same key
		const numGoroutines = 10
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				meta := &models.VaultMetadata{
					Key:       "concurrent.key",
					Value:     fmt.Sprintf("value-%d", idx),
					UpdatedAt: time.Now(),
				}
				err := repos.Metadata.SetMetadata(tdb.DB, ctx, meta)
				errors <- err
			}(i)
		}

		// All updates should succeed
		for i := 0; i < numGoroutines; i++ {
			err := <-errors
			assert.NoError(t, err)
		}

		// Final value should be one of the updates
		final, err := repos.Metadata.GetMetadata(tdb.DB, ctx, "concurrent.key")
		require.NoError(t, err)
		assert.Contains(t, final.Value, "value-")
	})
}

// ptrTime is a helper to get a pointer to a time.Time
func ptrTime(t time.Time) *time.Time {
	return &t
}

// ptrString is a helper to get a pointer to a string
func ptrString(s string) *string {
	return &s
}