package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/service"
)

// TestParseAndIndexVault_Errors tests error scenarios that don't require a real database
func TestParseAndIndexVault_Errors(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*mockDependencies)
		expectedError string
		validateResult func(*testing.T, *models.ParseHistory, error)
	}{
		{
			name: "git pull failure",
			setupMocks: func(deps *mockDependencies) {
				deps.metadataRepo.createParseRecordFunc = func(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
					return nil
				}
				deps.metadataRepo.updateParseStatusFunc = func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus, errorMsg *string) error {
					return nil
				}
				deps.gitManager.pullFunc = func(ctx context.Context) error {
					return errors.New("git pull failed")
				}
			},
			expectedError: "parsing pipeline failed: failed to pull vault: git pull failed",
			validateResult: func(t *testing.T, history *models.ParseHistory, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "git pull failed")
				assert.NotNil(t, history)
			},
		},
		{
			name: "parse history creation failure",
			setupMocks: func(deps *mockDependencies) {
				deps.metadataRepo.createParseRecordFunc = func(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
					return errors.New("database error")
				}
			},
			expectedError: "failed to initialize parse history",
			validateResult: func(t *testing.T, history *models.ParseHistory, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to initialize parse history")
				assert.Nil(t, history)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := setupMockDependencies(t)
			tt.setupMocks(deps)

			// Create test vault with sample files
			testVaultDir := setupTestVault(t)
			deps.gitManager.getLocalPathFunc = func() string {
				return testVaultDir
			}

			vaultService := service.NewVaultService(
				deps.config,
				deps.gitManager,
				deps.nodeService,
				deps.edgeService,
				deps.metadataService,
				nil, // DB can be nil for these error tests
			)

			ctx := context.Background()
			history, err := vaultService.ParseAndIndexVault(ctx)

			tt.validateResult(t, history, err)
		})
	}
}

// TestParseAndIndexVault_GitPullTimeout tests timeout handling in Git operations
func TestParseAndIndexVault_GitPullTimeout(t *testing.T) {
	deps := setupMockDependencies(t)

	// Setup mock to simulate slow git pull
	deps.gitManager.pullFunc = func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			return nil
		}
	}

	deps.metadataRepo.createParseRecordFunc = func(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
		return nil
	}
	deps.metadataRepo.updateParseStatusFunc = func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus, errorMsg *string) error {
		return nil
	}

	vaultService := service.NewVaultService(
		deps.config,
		deps.gitManager,
		deps.nodeService,
		deps.edgeService,
		deps.metadataService,
		nil,
	)

	// Use a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	history, err := vaultService.ParseAndIndexVault(ctx)

	// Should handle timeout gracefully
	require.Error(t, err)
	assert.NotNil(t, history)
}

// TestVaultService_MethodsWithoutDB tests methods that don't require database
func TestVaultService_MethodsWithoutDB(t *testing.T) {
	t.Run("IsParseInProgress", func(t *testing.T) {
		deps := setupMockDependencies(t)
		vaultService := service.NewVaultService(
			deps.config,
			deps.gitManager,
			deps.nodeService,
			deps.edgeService,
			deps.metadataService,
			nil,
		)

		ctx := context.Background()
		inProgress, parseID, err := vaultService.IsParseInProgress(ctx)

		require.NoError(t, err)
		assert.False(t, inProgress)
		assert.Empty(t, parseID)
	})

	t.Run("GetVaultPath", func(t *testing.T) {
		deps := setupMockDependencies(t)

		expectedPath := "/path/to/vault"
		deps.metadataRepo.getMetadataFunc = func(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error) {
			if key == "vault_path" {
				return &models.VaultMetadata{
					Key:   "vault_path",
					Value: expectedPath,
				}, nil
			}
			return nil, errors.New("key not found")
		}

		vaultService := service.NewVaultService(
			deps.config,
			deps.gitManager,
			deps.nodeService,
			deps.edgeService,
			deps.metadataService,
			nil,
		)

		ctx := context.Background()
		path, err := vaultService.GetVaultPath(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedPath, path)
	})

	t.Run("GetVaultPath_NotFound", func(t *testing.T) {
		deps := setupMockDependencies(t)

		deps.metadataRepo.getMetadataFunc = func(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error) {
			return nil, nil // Not found
		}

		vaultService := service.NewVaultService(
			deps.config,
			deps.gitManager,
			deps.nodeService,
			deps.edgeService,
			deps.metadataService,
			nil,
		)

		ctx := context.Background()
		path, err := vaultService.GetVaultPath(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "vault path not found")
		assert.Empty(t, path)
	})
}

// TestVaultService_ProgressTracking tests progress tracking functionality
func TestVaultService_ProgressTracking(t *testing.T) {
	deps := setupMockDependencies(t)

	// Setup to track progress during parse
	parseInProgress := make(chan struct{})
	parseDone := make(chan struct{})

	deps.gitManager.pullFunc = func(ctx context.Context) error {
		close(parseInProgress)
		// Simulate some work
		time.Sleep(200 * time.Millisecond)
		return errors.New("stop after git pull for testing")
	}

	deps.metadataRepo.createParseRecordFunc = func(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
		return nil
	}
	deps.metadataRepo.updateParseStatusFunc = func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus, errorMsg *string) error {
		return nil
	}

	vaultService := service.NewVaultService(
		deps.config,
		deps.gitManager,
		deps.nodeService,
		deps.edgeService,
		deps.metadataService,
		nil,
	)

	// Start parse in background
	go func() {
		ctx := context.Background()
		_, _ = vaultService.ParseAndIndexVault(ctx)
		close(parseDone)
	}()

	// Wait for parse to start
	<-parseInProgress

	// Check status while parsing
	ctx := context.Background()
	status, err := vaultService.GetParseStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, "running", status.Status)

	// Progress should be available
	assert.NotNil(t, status.Progress)

	// Wait for parse to complete
	<-parseDone

	// Progress should have been available during parsing
	assert.NotNil(t, status.StartedAt)
}

// TestVaultService_WaitForParseNoActiveOperation tests WaitForParse when no parse is running
func TestVaultService_WaitForParseNoActiveOperation(t *testing.T) {
	deps := setupMockDependencies(t)

	// Setup mock to return no parse history
	deps.metadataRepo.getLatestParseFunc = func(exec repository.Executor, ctx context.Context) (*models.ParseHistory, error) {
		return nil, errors.New("not found")
	}

	vaultService := service.NewVaultService(
		deps.config,
		deps.gitManager,
		deps.nodeService,
		deps.edgeService,
		deps.metadataService,
		nil,
	)

	ctx := context.Background()
	err := vaultService.WaitForParse(ctx, 1*time.Second)

	// Should return immediately when no parse is active
	require.NoError(t, err)
}

// TestVaultService_ConfigDefaults tests configuration default handling
func TestVaultService_ConfigDefaults(t *testing.T) {
	tests := []struct {
		name          string
		maxConcurrency int
		batchSize     int
	}{
		{
			name:          "zero values",
			maxConcurrency: 0,
			batchSize:     0,
		},
		{
			name:          "negative values",
			maxConcurrency: -1,
			batchSize:     -1,
		},
		{
			name:          "valid values",
			maxConcurrency: 8,
			batchSize:     500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := setupMockDependencies(t)
			deps.config.Graph.MaxConcurrency = tt.maxConcurrency
			deps.config.Graph.BatchSize = tt.batchSize

			vaultService := service.NewVaultService(
				deps.config,
				deps.gitManager,
				deps.nodeService,
				deps.edgeService,
				deps.metadataService,
				nil,
			)

			// The service should be created without error
			assert.NotNil(t, vaultService)
		})
	}
}
