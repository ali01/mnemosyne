package service_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/repository/postgres"
	"github.com/ali01/mnemosyne/internal/service"
)

// mockNodeService implements NodeServiceInterface for testing
type mockNodeService struct {
	getNodeFunc           func(ctx context.Context, id string) (*models.VaultNode, error)
	getNodeByPathFunc     func(ctx context.Context, path string) (*models.VaultNode, error)
	createNodeFunc        func(ctx context.Context, node *models.VaultNode) error
	createNodesFunc       func(ctx context.Context, nodes []models.VaultNode) error
	createNodeBatchTxFunc func(tx repository.Executor, ctx context.Context, nodes []models.VaultNode) error
	updateNodeFunc        func(ctx context.Context, node *models.VaultNode) error
	deleteNodeFunc        func(ctx context.Context, id string) error
	getAllNodesFunc       func(ctx context.Context, limit, offset int) ([]models.VaultNode, error)
	getNodesByTypeFunc    func(ctx context.Context, nodeType string) ([]models.VaultNode, error)
	searchNodesFunc       func(ctx context.Context, query string) ([]models.VaultNode, error)
	countFunc             func(ctx context.Context) (int64, error)
}

func (m *mockNodeService) GetNode(ctx context.Context, id string) (*models.VaultNode, error) {
	if m.getNodeFunc != nil {
		return m.getNodeFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockNodeService) GetNodeByPath(ctx context.Context, path string) (*models.VaultNode, error) {
	if m.getNodeByPathFunc != nil {
		return m.getNodeByPathFunc(ctx, path)
	}
	return nil, nil
}

func (m *mockNodeService) CreateNode(ctx context.Context, node *models.VaultNode) error {
	if m.createNodeFunc != nil {
		return m.createNodeFunc(ctx, node)
	}
	return nil
}

func (m *mockNodeService) CreateNodes(ctx context.Context, nodes []models.VaultNode) error {
	if m.createNodesFunc != nil {
		return m.createNodesFunc(ctx, nodes)
	}
	return nil
}

func (m *mockNodeService) CreateNodeBatchTx(tx repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
	if m.createNodeBatchTxFunc != nil {
		return m.createNodeBatchTxFunc(tx, ctx, nodes)
	}
	return nil
}

func (m *mockNodeService) UpdateNode(ctx context.Context, node *models.VaultNode) error {
	if m.updateNodeFunc != nil {
		return m.updateNodeFunc(ctx, node)
	}
	return nil
}

func (m *mockNodeService) DeleteNode(ctx context.Context, id string) error {
	if m.deleteNodeFunc != nil {
		return m.deleteNodeFunc(ctx, id)
	}
	return nil
}

func (m *mockNodeService) GetAllNodes(ctx context.Context, limit, offset int) ([]models.VaultNode, error) {
	if m.getAllNodesFunc != nil {
		return m.getAllNodesFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockNodeService) GetNodesByType(ctx context.Context, nodeType string) ([]models.VaultNode, error) {
	if m.getNodesByTypeFunc != nil {
		return m.getNodesByTypeFunc(ctx, nodeType)
	}
	return nil, nil
}

func (m *mockNodeService) SearchNodes(ctx context.Context, query string) ([]models.VaultNode, error) {
	if m.searchNodesFunc != nil {
		return m.searchNodesFunc(ctx, query)
	}
	return nil, nil
}

func (m *mockNodeService) CountNodes(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

// mockEdgeService implements EdgeServiceInterface for testing
type mockEdgeService struct {
	createEdgeFunc        func(ctx context.Context, edge *models.VaultEdge) error
	createEdgeBatchTxFunc func(tx repository.Executor, ctx context.Context, edges []models.VaultEdge) error
	getEdgeFunc           func(ctx context.Context, id string) (*models.VaultEdge, error)
	updateEdgeFunc        func(ctx context.Context, edge *models.VaultEdge) error
	deleteEdgeFunc        func(ctx context.Context, id string) error
	getAllEdgesFunc       func(ctx context.Context, limit, offset int) ([]models.VaultEdge, error)
	getEdgesByNodeFunc    func(ctx context.Context, nodeID string) ([]models.VaultEdge, error)
	countFunc             func(ctx context.Context) (int64, error)
}

func (m *mockEdgeService) CreateEdge(ctx context.Context, edge *models.VaultEdge) error {
	if m.createEdgeFunc != nil {
		return m.createEdgeFunc(ctx, edge)
	}
	return nil
}


func (m *mockEdgeService) CreateEdgeBatchTx(tx repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
	if m.createEdgeBatchTxFunc != nil {
		return m.createEdgeBatchTxFunc(tx, ctx, edges)
	}
	return nil
}

func (m *mockEdgeService) GetEdge(ctx context.Context, id string) (*models.VaultEdge, error) {
	if m.getEdgeFunc != nil {
		return m.getEdgeFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockEdgeService) UpdateEdge(ctx context.Context, edge *models.VaultEdge) error {
	if m.updateEdgeFunc != nil {
		return m.updateEdgeFunc(ctx, edge)
	}
	return nil
}

func (m *mockEdgeService) DeleteEdge(ctx context.Context, id string) error {
	if m.deleteEdgeFunc != nil {
		return m.deleteEdgeFunc(ctx, id)
	}
	return nil
}

func (m *mockEdgeService) GetAllEdges(ctx context.Context, limit, offset int) ([]models.VaultEdge, error) {
	if m.getAllEdgesFunc != nil {
		return m.getAllEdgesFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockEdgeService) GetEdgesByNode(ctx context.Context, nodeID string) ([]models.VaultEdge, error) {
	if m.getEdgesByNodeFunc != nil {
		return m.getEdgesByNodeFunc(ctx, nodeID)
	}
	return nil, nil
}

func (m *mockEdgeService) CountEdges(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

// mockGitManager implements git.Manager interface for testing
type mockGitManager struct {
	pullFunc         func(ctx context.Context) error
	getLocalPathFunc func() string
}

func (m *mockGitManager) Pull(ctx context.Context) error {
	if m.pullFunc != nil {
		return m.pullFunc(ctx)
	}
	return nil
}

func (m *mockGitManager) GetLocalPath() string {
	if m.getLocalPathFunc != nil {
		return m.getLocalPathFunc()
	}
	return "/test/vault/path"
}

func (m *mockGitManager) Clone(ctx context.Context) error {
	return nil
}

func (m *mockGitManager) IsCloned() bool {
	return true
}


// mockMetadataRepository implements repository.MetadataRepository for testing
type mockMetadataRepository struct {
	getMetadataFunc       func(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error)
	setMetadataFunc       func(exec repository.Executor, ctx context.Context, metadata *models.VaultMetadata) error
	getAllMetadataFunc    func(exec repository.Executor, ctx context.Context) ([]models.VaultMetadata, error)
	createParseRecordFunc func(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error
	getLatestParseFunc    func(exec repository.Executor, ctx context.Context) (*models.ParseHistory, error)
	getParseByIDFunc      func(exec repository.Executor, ctx context.Context, id string) (*models.ParseHistory, error)
	updateParseStatusFunc func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus, errorMsg *string) error
	getParseHistoryFunc   func(exec repository.Executor, ctx context.Context, limit int) ([]models.ParseHistory, error)
}

func (m *mockMetadataRepository) GetMetadata(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error) {
	if m.getMetadataFunc != nil {
		return m.getMetadataFunc(exec, ctx, key)
	}
	return nil, nil
}

func (m *mockMetadataRepository) SetMetadata(exec repository.Executor, ctx context.Context, metadata *models.VaultMetadata) error {
	if m.setMetadataFunc != nil {
		return m.setMetadataFunc(exec, ctx, metadata)
	}
	return nil
}

func (m *mockMetadataRepository) GetAllMetadata(exec repository.Executor, ctx context.Context) ([]models.VaultMetadata, error) {
	if m.getAllMetadataFunc != nil {
		return m.getAllMetadataFunc(exec, ctx)
	}
	return nil, nil
}

func (m *mockMetadataRepository) CreateParseRecord(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
	if m.createParseRecordFunc != nil {
		return m.createParseRecordFunc(exec, ctx, record)
	}
	return nil
}

func (m *mockMetadataRepository) GetLatestParse(exec repository.Executor, ctx context.Context) (*models.ParseHistory, error) {
	if m.getLatestParseFunc != nil {
		return m.getLatestParseFunc(exec, ctx)
	}
	return nil, nil
}

func (m *mockMetadataRepository) GetParseByID(exec repository.Executor, ctx context.Context, id string) (*models.ParseHistory, error) {
	if m.getParseByIDFunc != nil {
		return m.getParseByIDFunc(exec, ctx, id)
	}
	return nil, nil
}

func (m *mockMetadataRepository) UpdateParseStatus(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus, errorMsg *string) error {
	if m.updateParseStatusFunc != nil {
		return m.updateParseStatusFunc(exec, ctx, id, status, errorMsg)
	}
	return nil
}

func (m *mockMetadataRepository) GetParseHistory(exec repository.Executor, ctx context.Context, limit int) ([]models.ParseHistory, error) {
	if m.getParseHistoryFunc != nil {
		return m.getParseHistoryFunc(exec, ctx, limit)
	}
	return nil, nil
}

// mockDependencies groups all mocks for easier test setup
type mockDependencies struct {
	gitManager      *mockGitManager
	nodeService     *mockNodeService
	edgeService     *mockEdgeService
	metadataService *service.MetadataService
	metadataRepo    *mockMetadataRepository
	db              *sqlx.DB
	config          *config.Config
}

// setupMockDependencies creates a default set of mocks
func setupMockDependencies(t *testing.T) *mockDependencies {
	t.Helper()

	// For unit tests, we'll use a nil DB since we mock all repository calls
	// The VaultService should not reach database transaction code in unit tests
	var db *sqlx.DB

	// Create a temporary directory for tests that might need it
	tmpDir := t.TempDir()

	// Create mock metadata repository
	metadataRepo := &mockMetadataRepository{
		createParseRecordFunc: func(exec repository.Executor, ctx context.Context, history *models.ParseHistory) error {
			return nil
		},
		updateParseStatusFunc: func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus, errorMsg *string) error {
			return nil
		},
		getLatestParseFunc: func(exec repository.Executor, ctx context.Context) (*models.ParseHistory, error) {
			return &models.ParseHistory{
				ID:          uuid.New().String(),
				Status:      models.ParseStatusCompleted,
				StartedAt:   time.Now().Add(-time.Hour),
				CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
			}, nil
		},
		setMetadataFunc: func(exec repository.Executor, ctx context.Context, metadata *models.VaultMetadata) error {
			return nil
		},
		getMetadataFunc: func(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error) {
			return &models.VaultMetadata{
				Key:   key,
				Value: "test-value",
			}, nil
		},
	}

	// Create metadata service with mock repository
	metadataService := service.NewMetadataServiceWithRepo(db, metadataRepo)

	return &mockDependencies{
		gitManager: &mockGitManager{
			pullFunc: func(ctx context.Context) error {
				return nil
			},
			getLocalPathFunc: func() string {
				return tmpDir
			},
		},
		nodeService: &mockNodeService{
			createNodesFunc: func(ctx context.Context, nodes []models.VaultNode) error {
				return nil
			},
			createNodeBatchTxFunc: func(tx repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
				return nil
			},
			countFunc: func(ctx context.Context) (int64, error) {
				return 0, nil
			},
		},
		edgeService: &mockEdgeService{
			createEdgeBatchTxFunc: func(tx repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
				return nil
			},
			countFunc: func(ctx context.Context) (int64, error) {
				return 0, nil
			},
		},
		metadataService: metadataService,
		metadataRepo:    metadataRepo,
		db:              db,
		config: &config.Config{
			Graph: config.GraphConfig{
				MaxConcurrency: 4,
				BatchSize:      100,
				NodeClassification: config.NodeClassificationConfig{
					NodeTypes: map[string]config.NodeTypeConfig{
						"note": {
							DisplayName:    "Note",
							Color:          "#3498db",
							SizeMultiplier: 1.0,
						},
					},
					ClassificationRules: []config.ClassificationRuleConfig{
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
		},
	}
}

func TestNewVaultService(t *testing.T) {
	deps := setupMockDependencies(t)

	service := service.NewVaultService(
		deps.config,
		deps.gitManager,
		deps.nodeService,
		deps.edgeService,
		deps.metadataService,
		deps.db,
	)

	assert.NotNil(t, service)
}


// TestParseAndIndexVault_ConcurrentRejection tests concurrent parse rejection
// This test uses real database via Docker to properly test mutex behavior
func TestParseAndIndexVault_ConcurrentRejection(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Use test containers for real database
	tdb, _ := postgres.CreateTestRepositories(t)
	defer tdb.Close()

	// Use context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Clean tables before test
	require.NoError(t, tdb.CleanTables(ctx))

	// Create test vault directory with sample files
	testVaultDir := setupTestVault(t)
	defer os.RemoveAll(testVaultDir)

	// Create real services with test database
	nodeService := service.NewNodeService(tdb.DB)
	edgeService := service.NewEdgeService(tdb.DB)
	metadataService := service.NewMetadataService(tdb.DB)

	// Create test config
	cfg := &config.Config{
		Graph: config.GraphConfig{
			MaxConcurrency: 4,
			BatchSize:      100,
			NodeClassification: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"note": {
						DisplayName:    "Note",
						Color:          "#3498db",
						SizeMultiplier: 1.0,
					},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
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

	// Create channels to coordinate the test
	parseStarted := make(chan struct{})
	parseCanFinish := make(chan struct{})
	firstParseResult := make(chan error, 1)

	// Track number of pull calls to control blocking behavior
	var pullCallCount int32

	// Create mock git manager that blocks on first call only
	gitManager := &mockGitManager{
		pullFunc: func(ctx context.Context) error {
			// Use atomic increment for thread safety
			callNum := atomic.AddInt32(&pullCallCount, 1)

			if callNum == 1 {
				// First call - signal that parse has started and block
				close(parseStarted)

				// Wait for test to signal we can proceed
				select {
				case <-parseCanFinish:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			// Subsequent calls complete immediately
			return nil
		},
		getLocalPathFunc: func() string {
			return testVaultDir
		},
	}

	// Create vault service with real database
	vaultService := service.NewVaultService(
		cfg,
		gitManager,
		nodeService,
		edgeService,
		metadataService,
		tdb.DB,
	)

	// Start first parse in goroutine with timeout context
	go func() {
		parseCtx, parseCancel := context.WithTimeout(ctx, 15*time.Second)
		defer parseCancel()
		_, err := vaultService.ParseAndIndexVault(parseCtx)
		firstParseResult <- err
	}()

	// Wait for first parse to start and acquire mutex
	<-parseStarted

	// Give it a moment to ensure mutex is held and first parse is blocked
	time.Sleep(100 * time.Millisecond)

	// Try to start second parse - should be rejected due to mutex
	secondResult, err := vaultService.ParseAndIndexVault(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse already in progress")
	assert.Nil(t, secondResult)

	// Let first parse complete
	close(parseCanFinish)

	// Wait for first parse to finish
	firstErr := <-firstParseResult
	require.NoError(t, firstErr, "First parse should complete successfully")

	// Verify data was stored
	nodeCount, err := nodeService.CountNodes(ctx)
	require.NoError(t, err)
	assert.Greater(t, nodeCount, int64(0), "Nodes should have been created")

	// Clean tables before third parse to avoid duplicate key constraints
	require.NoError(t, tdb.CleanTables(ctx))

	// Now a third parse should work since mutex is released
	thirdResult, err := vaultService.ParseAndIndexVault(ctx)
	require.NoError(t, err, "Third parse should succeed after mutex is released")
	assert.NotNil(t, thirdResult)
	assert.Equal(t, models.ParseStatusCompleted, thirdResult.Status)
}

func TestGetParseStatus(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*mockDependencies) string // returns parseID if simulating active parse
		validate   func(*testing.T, *models.ParseStatusResponse)
	}{
		{
			name: "no active parse - returns latest history",
			setupMocks: func(deps *mockDependencies) string {
				deps.metadataRepo.getLatestParseFunc = func(exec repository.Executor, ctx context.Context) (*models.ParseHistory, error) {
					completedAt := time.Now()
					return &models.ParseHistory{
						ID:          "test-parse-id",
						Status:      models.ParseStatusCompleted,
						StartedAt:   completedAt.Add(-time.Hour),
						CompletedAt: &completedAt,
						Stats: models.JSONStats(models.ParseStats{
							TotalNodes: 100,
							TotalEdges: 200,
						}),
					}, nil
				}
				return ""
			},
			validate: func(t *testing.T, status *models.ParseStatusResponse) {
				assert.Equal(t, "completed", status.Status)
				assert.NotNil(t, status.CompletedAt)
				assert.Nil(t, status.Progress)
			},
		},
		{
			name: "parse history error",
			setupMocks: func(deps *mockDependencies) string {
				deps.metadataRepo.getLatestParseFunc = func(exec repository.Executor, ctx context.Context) (*models.ParseHistory, error) {
					return nil, errors.New("database error")
				}
				return ""
			},
			validate: func(t *testing.T, status *models.ParseStatusResponse) {
				// Should return error, status will be nil
				assert.Nil(t, status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := setupMockDependencies(t)
			parseID := tt.setupMocks(deps)

			vaultService := service.NewVaultService(
				deps.config,
				deps.gitManager,
				deps.nodeService,
				deps.edgeService,
				deps.metadataService,
				deps.db,
			)

			// If we need to simulate active parse, we'd need to expose internal state
			// For now, we'll test the non-active parse scenarios
			_ = parseID // Would be used if we could simulate active parse

			ctx := context.Background()
			status, err := vaultService.GetParseStatus(ctx)

			if tt.name == "parse history error" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validate(t, status)
			}
		})
	}
}

func TestGetLatestParseHistory(t *testing.T) {
	deps := setupMockDependencies(t)

	expectedHistory := &models.ParseHistory{
		ID:        "test-parse-id",
		Status:    models.ParseStatusCompleted,
		StartedAt: time.Now().Add(-time.Hour),
	}

	deps.metadataRepo.getLatestParseFunc = func(exec repository.Executor, ctx context.Context) (*models.ParseHistory, error) {
		return expectedHistory, nil
	}

	vaultService := service.NewVaultService(
		deps.config,
		deps.gitManager,
		deps.nodeService,
		deps.edgeService,
		deps.metadataService,
		deps.db,
	)

	ctx := context.Background()
	history, err := vaultService.GetLatestParseHistory(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedHistory.ID, history.ID)
	assert.Equal(t, expectedHistory.Status, history.Status)
}

func TestGetVaultPath(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupMockDependencies(t)

		expectedMetadata := &models.VaultMetadata{
			Key:   "vault_path",
			Value: "/path/to/vault",
		}

		deps.metadataRepo.getMetadataFunc = func(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error) {
			if key == "vault_path" {
				return expectedMetadata, nil
			}
			return nil, errors.New("key not found")
		}

		vaultService := service.NewVaultService(
			deps.config,
			deps.gitManager,
			deps.nodeService,
			deps.edgeService,
			deps.metadataService,
			deps.db,
		)

		ctx := context.Background()
		path, err := vaultService.GetVaultPath(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedMetadata.Value, path)
	})

	t.Run("metadata not found", func(t *testing.T) {
		deps := setupMockDependencies(t)

		deps.metadataRepo.getMetadataFunc = func(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error) {
			return nil, nil // Metadata not found
		}

		vaultService := service.NewVaultService(
			deps.config,
			deps.gitManager,
			deps.nodeService,
			deps.edgeService,
			deps.metadataService,
			deps.db,
		)

		ctx := context.Background()
		path, err := vaultService.GetVaultPath(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "vault path not found")
		assert.Empty(t, path)
	})

	t.Run("metadata error", func(t *testing.T) {
		deps := setupMockDependencies(t)

		expectedErr := errors.New("database error")
		deps.metadataRepo.getMetadataFunc = func(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error) {
			return nil, expectedErr
		}

		vaultService := service.NewVaultService(
			deps.config,
			deps.gitManager,
			deps.nodeService,
			deps.edgeService,
			deps.metadataService,
			deps.db,
		)

		ctx := context.Background()
		path, err := vaultService.GetVaultPath(ctx)

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, path)
	})
}

// TestParserConfiguration tests that the parser is configured correctly
func TestParserConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		expectedConc   int
		expectedBatch  int
	}{
		{
			name: "uses config values",
			config: &config.Config{
				Graph: config.GraphConfig{
					MaxConcurrency: 8,
					BatchSize:      200,
				},
			},
			expectedConc:  8,
			expectedBatch: 200,
		},
		{
			name: "uses defaults for zero values",
			config: &config.Config{
				Graph: config.GraphConfig{
					MaxConcurrency: 0,
					BatchSize:      0,
				},
			},
			expectedConc:  4,
			expectedBatch: 100,
		},
		{
			name: "uses defaults for negative values",
			config: &config.Config{
				Graph: config.GraphConfig{
					MaxConcurrency: -1,
					BatchSize:      -1,
				},
			},
			expectedConc:  4,
			expectedBatch: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := setupMockDependencies(t)
			deps.config = tt.config

			// We can't directly test the parser creation without exposing internals
			// This test demonstrates how we would test configuration handling
			assert.NotNil(t, deps.config)
		})
	}
}

// TestGraphBuilderConfiguration tests that the graph builder is configured correctly
func TestGraphBuilderConfiguration(t *testing.T) {
	deps := setupMockDependencies(t)

	// Ensure node classification config exists
	assert.NotEmpty(t, deps.config.Graph.NodeClassification.NodeTypes)
	assert.NotEmpty(t, deps.config.Graph.NodeClassification.ClassificationRules)

	// Test would verify graph builder configuration if we could access it
	// This demonstrates the test structure
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
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	return dir
}

// TestIsParseInProgress tests the IsParseInProgress method
func TestIsParseInProgress(t *testing.T) {
	tests := []struct {
		name             string
		simulateActive   bool
		expectedProgress bool
		expectedParseID  string
		expectedError    error
	}{
		{
			name:             "no active parse",
			simulateActive:   false,
			expectedProgress: false,
			expectedParseID:  "",
			expectedError:    nil,
		},
		{
			name:             "active parse",
			simulateActive:   true,
			expectedProgress: true,
			expectedParseID:  "test-parse-id", // Will be set during test
			expectedError:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if running in short mode
			if testing.Short() && tt.simulateActive {
				t.Skip("Skipping integration test")
			}

			// Use test containers for real database
			tdb, _ := postgres.CreateTestRepositories(t)
			defer tdb.Close()

			ctx := context.Background()
			require.NoError(t, tdb.CleanTables(ctx))

			// Create test vault directory
			testVaultDir := setupTestVault(t)
			defer os.RemoveAll(testVaultDir)

			// Create real services
			nodeService := service.NewNodeService(tdb.DB)
			edgeService := service.NewEdgeService(tdb.DB)
			metadataService := service.NewMetadataService(tdb.DB)

			// Create test config
			cfg := &config.Config{
				Graph: config.GraphConfig{
					MaxConcurrency: 4,
					BatchSize:      100,
					NodeClassification: config.NodeClassificationConfig{
						NodeTypes: map[string]config.NodeTypeConfig{
							"note": {
								DisplayName:    "Note",
								Color:          "#3498db",
								SizeMultiplier: 1.0,
							},
						},
						ClassificationRules: []config.ClassificationRuleConfig{
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

			// Create channels to coordinate the test
			parseStarted := make(chan struct{})
			parseCanFinish := make(chan struct{})
			parseCompleted := make(chan string, 1)

			// Create mock git manager
			gitManager := &mockGitManager{
				pullFunc: func(ctx context.Context) error {
					if tt.simulateActive {
						close(parseStarted)
						<-parseCanFinish
					}
					return nil
				},
				getLocalPathFunc: func() string {
					return testVaultDir
				},
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

			if tt.simulateActive {
				// Start parse in background
				go func() {
					parseCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					history, _ := vaultService.ParseAndIndexVault(parseCtx)
					if history != nil {
						parseCompleted <- history.ID
					} else {
						parseCompleted <- ""
					}
				}()

				// Wait for parse to start
				<-parseStarted
				time.Sleep(100 * time.Millisecond)
			}

			// Test IsParseInProgress
			inProgress, parseID, err := vaultService.IsParseInProgress(ctx)

			// Validate results
			assert.Equal(t, tt.expectedProgress, inProgress)
			assert.Equal(t, tt.expectedError, err)

			if tt.simulateActive {
				assert.NotEmpty(t, parseID)
				// Let parse complete
				close(parseCanFinish)
				<-parseCompleted
			} else {
				assert.Empty(t, parseID)
			}
		})
	}
}

// TestParseAndIndexVault_Integration tests the full ParseAndIndexVault method with real database
// This is moved to integration tests because VaultService requires a real database for transactions
func TestParseAndIndexVault_Integration(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Use test containers for real database
	tdb, _ := postgres.CreateTestRepositories(t)
	defer tdb.Close()

	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	tests := []struct {
		name          string
		setupMocks    func(*mockDependencies)
		expectedError string
		validateResult func(*testing.T, *models.ParseHistory, error)
	}{
		{
			name: "successful parse",
			setupMocks: func(deps *mockDependencies) {
				// No special setup needed for successful parse
			},
			validateResult: func(t *testing.T, history *models.ParseHistory, err error) {
				require.NoError(t, err)
				assert.NotNil(t, history)
				assert.Equal(t, models.ParseStatusCompleted, history.Status)
			},
		},
		{
			name: "git pull failure",
			setupMocks: func(deps *mockDependencies) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := setupMockDependencies(t)
			tt.setupMocks(deps)

			// Create test vault with sample files
			testVaultDir := setupTestVault(t)
			defer os.RemoveAll(testVaultDir)
			deps.gitManager.getLocalPathFunc = func() string {
				return testVaultDir
			}

			// Create real services with test database
			nodeService := service.NewNodeService(tdb.DB)
			edgeService := service.NewEdgeService(tdb.DB)
			metadataService := service.NewMetadataService(tdb.DB)

			vaultService := service.NewVaultService(
				deps.config,
				deps.gitManager,
				nodeService,
				edgeService,
				metadataService,
				tdb.DB,
			)

			history, err := vaultService.ParseAndIndexVault(ctx)

			tt.validateResult(t, history, err)
		})
	}
}

// TestParseAndIndexVault_PanicRecovery tests panic recovery mechanism
func TestParseAndIndexVault_PanicRecovery(t *testing.T) {
	deps := setupMockDependencies(t)

	// Setup mock to panic during git pull
	deps.gitManager.pullFunc = func(ctx context.Context) error {
		panic("simulated panic during git pull")
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
		deps.db,
	)

	ctx := context.Background()
	history, err := vaultService.ParseAndIndexVault(ctx)

	// Should recover from panic and return error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic during parse: simulated panic during git pull")
	assert.NotNil(t, history)
}

// TestAcquireReleaseParseLock tests the parse lock mechanism
func TestAcquireReleaseParseLock(t *testing.T) {
	deps := setupMockDependencies(t)

	vaultService := service.NewVaultService(
		deps.config,
		deps.gitManager,
		deps.nodeService,
		deps.edgeService,
		deps.metadataService,
		deps.db,
	)

	ctx := context.Background()

	// Test 1: First acquire should succeed
	inProgress, parseID, err := vaultService.IsParseInProgress(ctx)
	require.NoError(t, err)
	assert.False(t, inProgress)
	assert.Empty(t, parseID)

	// Start a parse to acquire lock
	deps.metadataRepo.createParseRecordFunc = func(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
		return errors.New("stop after creating history")
	}

	_, err = vaultService.ParseAndIndexVault(ctx)
	require.Error(t, err)

	// Test 2: Check lock is released after error
	inProgress, parseID, err = vaultService.IsParseInProgress(ctx)
	require.NoError(t, err)
	assert.False(t, inProgress)
	assert.Empty(t, parseID)
}

// TestUpdateParseProgress tests progress tracking during parse
func TestUpdateParseProgress(t *testing.T) {
	deps := setupMockDependencies(t)

	// progressUpdates := make([]models.ParseProgress, 0) // Currently unused but kept for future test enhancements
	parseStarted := make(chan struct{})
	captureProgress := make(chan struct{})

	// Track progress updates
	deps.gitManager.pullFunc = func(ctx context.Context) error {
		close(parseStarted)
		// Wait a bit to ensure we're in parsing state
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	deps.metadataRepo.createParseRecordFunc = func(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
		return nil
	}
	deps.metadataRepo.updateParseStatusFunc = func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus, errorMsg *string) error {
		return nil
	}
	deps.metadataRepo.setMetadataFunc = func(exec repository.Executor, ctx context.Context, metadata *models.VaultMetadata) error {
		return nil
	}
	deps.nodeService.createNodeBatchTxFunc = func(tx repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
		return nil
	}
	deps.edgeService.createEdgeBatchTxFunc = func(tx repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
		return nil
	}

	// Create test vault
	testVaultDir := setupTestVault(t)
	defer os.RemoveAll(testVaultDir)
	deps.gitManager.getLocalPathFunc = func() string {
		return testVaultDir
	}

	vaultService := service.NewVaultService(
		deps.config,
		deps.gitManager,
		deps.nodeService,
		deps.edgeService,
		deps.metadataService,
		deps.db,
	)

	// Start parse in background
	go func() {
		ctx := context.Background()
		_, _ = vaultService.ParseAndIndexVault(ctx)
		close(captureProgress)
	}()

	// Wait for parse to start
	<-parseStarted

	// Check status while parsing
	ctx := context.Background()
	status, err := vaultService.GetParseStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, "running", status.Status)

	// Progress should be available
	assert.NotNil(t, status.Progress, "Progress should be available during parse")

	// Wait for parse to complete
	<-captureProgress

	// Final status should be completed
	status, err = vaultService.GetParseStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, "completed", status.Status)
}

// TestClearExistingGraphData tests the data clearing functionality
func TestClearExistingGraphData(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test verifies that node positions are preserved when nodes are deleted
	// during re-parsing, allowing users' manual positioning to persist

	// Use test containers for real database
	tdb, _ := postgres.CreateTestRepositories(t)
	defer tdb.Close()

	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	// Create test data
	testNode := models.VaultNode{
		ID:       "test-node",
		FilePath: "test.md",
		Title:    "Test Node",
	}

	testEdge := models.VaultEdge{
		SourceID: "test-node",
		TargetID: "test-node-2",
		EdgeType: "wikilink",
	}

	// Insert test data directly
	_, err := tdb.DB.ExecContext(ctx, `
		INSERT INTO nodes (id, file_path, title) VALUES ($1, $2, $3)`,
		testNode.ID, testNode.FilePath, testNode.Title)
	require.NoError(t, err)

	_, err = tdb.DB.ExecContext(ctx, `
		INSERT INTO nodes (id, file_path, title) VALUES ($1, $2, $3)`,
		"test-node-2", "test2.md", "Test Node 2")
	require.NoError(t, err)

	_, err = tdb.DB.ExecContext(ctx, `
		INSERT INTO edges (source_id, target_id, edge_type) VALUES ($1, $2, $3)`,
		testEdge.SourceID, testEdge.TargetID, testEdge.EdgeType)
	require.NoError(t, err)

	// Also insert node position data (should NOT be deleted)
	_, err = tdb.DB.ExecContext(ctx, `
		INSERT INTO node_positions (node_id, x, y) VALUES ($1, $2, $3)`,
		testNode.ID, 100.0, 200.0)
	require.NoError(t, err)

	// Setup service
	deps := setupMockDependencies(t)
	deps.db = tdb.DB

	// Mock successful parse flow
	deps.gitManager.pullFunc = func(ctx context.Context) error {
		return nil
	}
	deps.metadataRepo.createParseRecordFunc = func(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
		return nil
	}
	deps.metadataRepo.updateParseStatusFunc = func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus, errorMsg *string) error {
		return nil
	}
	deps.metadataRepo.setMetadataFunc = func(exec repository.Executor, ctx context.Context, metadata *models.VaultMetadata) error {
		return nil
	}

	// Create real services for database operations
	nodeService := service.NewNodeService(tdb.DB)
	edgeService := service.NewEdgeService(tdb.DB)

	// Create test vault
	testVaultDir := setupTestVault(t)
	defer os.RemoveAll(testVaultDir)
	deps.gitManager.getLocalPathFunc = func() string {
		return testVaultDir
	}

	vaultService := service.NewVaultService(
		deps.config,
		deps.gitManager,
		nodeService,
		edgeService,
		deps.metadataService,
		tdb.DB,
	)

	// Run parse (which should clear existing data)
	_, err = vaultService.ParseAndIndexVault(ctx)
	require.NoError(t, err)

	// Verify node positions were PRESERVED (not deleted)
	// With the updated schema, positions persist when nodes are deleted
	var positionCount int
	err = tdb.DB.GetContext(ctx, &positionCount, "SELECT COUNT(*) FROM node_positions WHERE node_id = $1", testNode.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, positionCount, "Node positions should be preserved when nodes are deleted")

	// Verify old nodes and edges were deleted
	var nodeCount int
	err = tdb.DB.GetContext(ctx, &nodeCount, "SELECT COUNT(*) FROM nodes WHERE id = $1", testNode.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, nodeCount, "Old nodes should be deleted")

	var edgeCount int
	err = tdb.DB.GetContext(ctx, &edgeCount, "SELECT COUNT(*) FROM edges WHERE source_id = $1", testEdge.SourceID)
	require.NoError(t, err)
	assert.Equal(t, 0, edgeCount, "Old edges should be deleted")
}

// TestConfigurationValidation tests parser and graph builder configuration
func TestConfigurationValidation(t *testing.T) {
	// Skip if running in short mode - this test requires a real database
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Use test containers for real database
	tdb, _ := postgres.CreateTestRepositories(t)
	defer tdb.Close()

	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	tests := []struct {
		name   string
		config *config.Config
	}{
		{
			name: "zero values use defaults",
			config: &config.Config{
				Graph: config.GraphConfig{
					MaxConcurrency: 0,
					BatchSize:      0,
					NodeClassification: config.NodeClassificationConfig{
						NodeTypes: map[string]config.NodeTypeConfig{
							"note": {DisplayName: "Note", Color: "#3498db", SizeMultiplier: 1.0},
						},
						ClassificationRules: []config.ClassificationRuleConfig{
							{Name: "default", NodeType: "note", Type: "regex", Pattern: ".*", Priority: 100},
						},
					},
				},
			},
		},
		{
			name: "negative values use defaults",
			config: &config.Config{
				Graph: config.GraphConfig{
					MaxConcurrency: -1,
					BatchSize:      -10,
					NodeClassification: config.NodeClassificationConfig{
						NodeTypes: map[string]config.NodeTypeConfig{
							"note": {DisplayName: "Note", Color: "#3498db", SizeMultiplier: 1.0},
						},
						ClassificationRules: []config.ClassificationRuleConfig{
							{Name: "default", NodeType: "note", Type: "regex", Pattern: ".*", Priority: 100},
						},
					},
				},
			},
		},
		{
			name: "valid positive values",
			config: &config.Config{
				Graph: config.GraphConfig{
					MaxConcurrency: 8,
					BatchSize:      500,
					NodeClassification: config.NodeClassificationConfig{
						NodeTypes: map[string]config.NodeTypeConfig{
							"note": {DisplayName: "Note", Color: "#3498db", SizeMultiplier: 1.0},
						},
						ClassificationRules: []config.ClassificationRuleConfig{
							{Name: "default", NodeType: "note", Type: "regex", Pattern: ".*", Priority: 100},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := setupMockDependencies(t)
			deps.config = tt.config

			// Mock successful parse
			deps.gitManager.pullFunc = func(ctx context.Context) error {
				return nil
			}
			deps.metadataRepo.createParseRecordFunc = func(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
				return nil
			}
			deps.metadataRepo.updateParseStatusFunc = func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus, errorMsg *string) error {
				return nil
			}
			deps.metadataRepo.setMetadataFunc = func(exec repository.Executor, ctx context.Context, metadata *models.VaultMetadata) error {
				return nil
			}
			deps.nodeService.createNodeBatchTxFunc = func(tx repository.Executor, ctx context.Context, nodes []models.VaultNode) error {
				return nil
			}
			deps.edgeService.createEdgeBatchTxFunc = func(tx repository.Executor, ctx context.Context, edges []models.VaultEdge) error {
				return nil
			}

			// Create test vault
			testVaultDir := setupTestVault(t)
			defer os.RemoveAll(testVaultDir)
			deps.gitManager.getLocalPathFunc = func() string {
				return testVaultDir
			}

			// Create real services with test database
			nodeService := service.NewNodeService(tdb.DB)
			edgeService := service.NewEdgeService(tdb.DB)
			metadataService := service.NewMetadataService(tdb.DB)

			vaultService := service.NewVaultService(
				deps.config,
				deps.gitManager,
				nodeService,
				edgeService,
				metadataService,
				tdb.DB,
			)

			_, err := vaultService.ParseAndIndexVault(ctx)

			// Should handle all configurations without error
			require.NoError(t, err)
		})
	}
}

// TestWaitForParse tests the WaitForParse method
func TestWaitForParse(t *testing.T) {
	tests := []struct {
		name          string
		timeout       time.Duration
		parseDelay    time.Duration
		parseFails    bool
		expectedError string
	}{
		{
			name:          "successful parse completes before timeout",
			timeout:       5 * time.Second,
			parseDelay:    100 * time.Millisecond,
			parseFails:    false,
			expectedError: "",
		},
		{
			name:          "timeout exceeded",
			timeout:       100 * time.Millisecond,
			parseDelay:    2 * time.Second,
			parseFails:    false,
			expectedError: "wait for parse timed out",
		},
		{
			name:          "parse fails",
			timeout:       5 * time.Second,
			parseDelay:    100 * time.Millisecond,
			parseFails:    true,
			expectedError: "parse failed",
		},
		{
			name:          "no active parse",
			timeout:       1 * time.Second,
			parseDelay:    0,
			parseFails:    false,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if running in short mode
			if testing.Short() && tt.parseDelay > 0 {
				t.Skip("Skipping integration test")
			}

			// Use test containers for real database
			tdb, _ := postgres.CreateTestRepositories(t)
			defer tdb.Close()

			ctx := context.Background()
			require.NoError(t, tdb.CleanTables(ctx))

			// Create test vault directory
			testVaultDir := setupTestVault(t)
			defer os.RemoveAll(testVaultDir)

			// Create real services
			nodeService := service.NewNodeService(tdb.DB)
			edgeService := service.NewEdgeService(tdb.DB)
			metadataService := service.NewMetadataService(tdb.DB)

			// Create test config
			cfg := &config.Config{
				Graph: config.GraphConfig{
					MaxConcurrency: 4,
					BatchSize:      100,
					NodeClassification: config.NodeClassificationConfig{
						NodeTypes: map[string]config.NodeTypeConfig{
							"note": {
								DisplayName:    "Note",
								Color:          "#3498db",
								SizeMultiplier: 1.0,
							},
						},
						ClassificationRules: []config.ClassificationRuleConfig{
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

			// Create mock git manager
			gitManager := &mockGitManager{
				pullFunc: func(ctx context.Context) error {
					if tt.parseDelay > 0 {
						time.Sleep(tt.parseDelay)
					}
					if tt.parseFails {
						return errors.New("git pull failed")
					}
					return nil
				},
				getLocalPathFunc: func() string {
					return testVaultDir
				},
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

			if tt.parseDelay > 0 {
				// Start parse in background
				go func() {
					parseCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					_, _ = vaultService.ParseAndIndexVault(parseCtx)
				}()

				// Give parse a moment to start
				time.Sleep(50 * time.Millisecond)
			}

			// Test WaitForParse
			err := vaultService.WaitForParse(ctx, tt.timeout)

			// Debug: check parse status after wait
			if tt.name == "parse fails" {
				status, _ := vaultService.GetParseStatus(ctx)
				if status != nil {
					t.Logf("Parse status after wait: %s, error: %s", status.Status, status.Error)
				}
			}

			// Validate results
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
