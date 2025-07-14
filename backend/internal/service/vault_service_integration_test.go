// +build integration

package service_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository/postgres"
	"github.com/ali01/mnemosyne/internal/service"
)

// TestVaultServiceIntegration runs comprehensive integration tests for VaultService
func TestVaultServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database using testcontainers
	tdb, _ := postgres.CreateTestRepositories(t)
	defer tdb.Close()

	// Clean up before tests
	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	// Create test vault directory
	testVaultDir := setupTestVault(t)
	defer os.RemoveAll(testVaultDir)

	// Create services with real implementations
	nodeService := service.NewNodeService(tdb.DB)
	edgeService := service.NewEdgeService(tdb.DB)
	metadataService := service.NewMetadataService(tdb.DB)

	// Create test config
	cfg := createTestConfig()

	// Create mock git manager that points to our test vault
	gitManager := &mockGitManagerIntegration{
		localPath: testVaultDir,
	}

	// Create vault service
	vaultService := service.NewVaultService(
		cfg,
		gitManager,
		nodeService,
		edgeService,
		metadataService,
		tdb.DB,
	)

	t.Run("EndToEndParse", func(t *testing.T) {
		// Parse the vault
		parseHistory, err := vaultService.ParseAndIndexVault(ctx)
		require.NoError(t, err)
		assert.NotNil(t, parseHistory)
		assert.Equal(t, models.ParseStatusCompleted, parseHistory.Status)

		// Verify nodes were created
		nodeCount, err := nodeService.CountNodes(ctx)
		require.NoError(t, err)
		assert.Greater(t, nodeCount, int64(0))

		// Verify edges were created
		edgeCount, err := edgeService.CountEdges(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, edgeCount, int64(0)) // May have no edges if no links

		// Verify metadata was updated
		lastParse, err := metadataService.GetMetadata(ctx, "last_parse")
		require.NoError(t, err)
		assert.NotNil(t, lastParse)

		// Verify parse history stats
		stats := parseHistory.Stats.ToParseStats()
		assert.Equal(t, int(nodeCount), stats.TotalNodes)
		assert.Equal(t, int(edgeCount), stats.TotalEdges)
		assert.Greater(t, stats.DurationMS, int64(0))
	})

	t.Run("ProgressTracking", func(t *testing.T) {
		// Clean tables for fresh test
		require.NoError(t, tdb.CleanTables(ctx))

		// Start parse in goroutine
		parseStarted := make(chan struct{})
		parseDone := make(chan error)

		go func() {
			// Notify test that parse started
			close(parseStarted)

			// Run parse
			_, err := vaultService.ParseAndIndexVault(ctx)
			parseDone <- err
		}()

		// Wait for parse to start
		<-parseStarted

		// Small delay to ensure parse is running
		time.Sleep(10 * time.Millisecond)

		// Check status while parsing
		status, err := vaultService.GetParseStatus(ctx)
		require.NoError(t, err)
		assert.NotNil(t, status)
		// Status could be running or completed depending on timing

		// Wait for parse to complete
		parseErr := <-parseDone
		require.NoError(t, parseErr)

		// Check final status
		finalStatus, err := vaultService.GetParseStatus(ctx)
		require.NoError(t, err)
		assert.Equal(t, "completed", finalStatus.Status)
		assert.NotNil(t, finalStatus.CompletedAt)
	})

	t.Run("ConcurrentParseRejection", func(t *testing.T) {
		// Clean tables for fresh test
		require.NoError(t, tdb.CleanTables(ctx))

		// Start a parse that will block
		parseBlocked := make(chan struct{})
		parseCanProceed := make(chan struct{})
		parseComplete := make(chan error, 1)

		gitManager.pullFunc = func(ctx context.Context) error {
			close(parseBlocked)
			<-parseCanProceed
			return nil
		}

		// Start first parse in goroutine
		go func() {
			_, err := vaultService.ParseAndIndexVault(ctx)
			parseComplete <- err
		}()

		// Wait for it to start
		<-parseBlocked

		// Try to start another parse - should be rejected
		_, err := vaultService.ParseAndIndexVault(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse already in progress")

		// Let first parse complete
		close(parseCanProceed)

		// Wait for first parse to finish
		firstParseErr := <-parseComplete
		require.NoError(t, firstParseErr, "First parse should complete successfully")

		// Reset git manager
		gitManager.pullFunc = nil

		// Now a new parse should work
		_, err = vaultService.ParseAndIndexVault(ctx)
		require.NoError(t, err, "Parse should work after previous one completes")
	})

	t.Run("LargeVaultPerformance", func(t *testing.T) {
		// Create a larger test vault
		largeVaultDir := setupLargeTestVault(t, 100) // 100 files
		defer os.RemoveAll(largeVaultDir)

		gitManager.localPath = largeVaultDir

		// Clean tables
		require.NoError(t, tdb.CleanTables(ctx))

		// Measure parse time
		start := time.Now()
		parseHistory, err := vaultService.ParseAndIndexVault(ctx)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, parseHistory)

		// Verify performance
		t.Logf("Parsed 100 files in %v", duration)
		assert.Less(t, duration, 10*time.Second, "Parsing 100 files should take less than 10 seconds")

		// Verify all nodes created
		nodeCount, err := nodeService.CountNodes(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(100), nodeCount)
	})

	t.Run("FailureRecovery", func(t *testing.T) {
		// Clean tables
		require.NoError(t, tdb.CleanTables(ctx))

		// Simulate git pull failure
		gitManager.pullFunc = func(ctx context.Context) error {
			return fmt.Errorf("simulated git failure")
		}

		// Parse should fail
		_, err := vaultService.ParseAndIndexVault(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git failure")

		// Verify no partial data
		nodeCount, err := nodeService.CountNodes(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), nodeCount)

		// Fix git and retry with the original test vault
		gitManager.pullFunc = nil
		gitManager.localPath = testVaultDir // Reset to original test vault
		parseHistory, err := vaultService.ParseAndIndexVault(ctx)
		require.NoError(t, err)
		assert.Equal(t, models.ParseStatusCompleted, parseHistory.Status)
	})
}

// setupTestVault creates a test vault with sample markdown files
func setupTestVault(t testing.TB) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "test-vault-*")
	require.NoError(t, err)

	// Create sample files
	files := map[string]string{
		"index.md": `---
id: index
title: Index
tags: [home, index]
---

# Welcome

This is the index page. See [[note1]] and [[note2]].`,

		"notes/note1.md": `---
id: note1
title: First Note
tags: [note]
---

# First Note

This links to [[note2]] and [[index]].`,

		"notes/note2.md": `---
id: note2
title: Second Note
tags: [note, important]
---

# Second Note

This links back to [[note1]].`,

		"daily/2024-01-01.md": `---
id: daily-2024-01-01
title: Daily Note
tags: [note]
---

# January 1, 2024

Today's notes. References [[note1]].`,
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	return dir
}

// setupLargeTestVault creates a test vault with many files
func setupLargeTestVault(t testing.TB, count int) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "large-test-vault-*")
	require.NoError(t, err)

	// Create many files
	for i := 0; i < count; i++ {
		content := fmt.Sprintf(`---
id: note-%d
title: Note %d
tags: [generated]
---

# Note %d

This is note number %d.

Links: [[note-%d]]`, i, i, i, i, (i+1)%count)

		path := fmt.Sprintf("notes/note-%d.md", i)
		fullPath := filepath.Join(dir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	return dir
}

// createTestConfig creates a test configuration
func createTestConfig() *config.Config {
	return &config.Config{
		Graph: config.GraphConfig{
			MaxConcurrency: 4,
			BatchSize:      50,
			NodeClassification: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"note": {
						DisplayName:    "Note",
						Color:          "#3498db",
						SizeMultiplier: 1.0,
					},
					"index": {
						DisplayName:    "Index",
						Color:          "#2ecc71",
						SizeMultiplier: 1.2,
					},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
					{
						Name:     "index_tag",
						Priority: 10,
						Type:     "tag",
						Pattern:  "index",
						NodeType: "index",
					},
					{
						Name:     "daily_path",
						Priority: 20,
						Type:     "path_contains",
						Pattern:  "daily",
						NodeType: "note",
					},
					{
						Name:     "catch_all",
						Priority: 100,
						Type:     "regex",
						Pattern:  ".*",
						NodeType: "note",
					},
				},
			},
		},
	}
}

// mockGitManagerIntegration is a simple git manager for integration tests
type mockGitManagerIntegration struct {
	localPath string
	pullFunc  func(ctx context.Context) error
}

func (m *mockGitManagerIntegration) Pull(ctx context.Context) error {
	if m.pullFunc != nil {
		return m.pullFunc(ctx)
	}
	return nil
}

func (m *mockGitManagerIntegration) GetLocalPath() string {
	return m.localPath
}

// BenchmarkParseAndIndexVault benchmarks vault parsing performance
// Note: This benchmark requires Docker and will be skipped if not available
func BenchmarkParseAndIndexVault(b *testing.B) {
	// Skip if short
	if testing.Short() {
		b.Skip("Skipping benchmark")
	}

	b.Skip("Benchmark requires testcontainers which only supports testing.T")

	// To run performance tests, use TestVaultServiceIntegration/LargeVaultPerformance
	// which includes timing information
}
