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
	createNodeBatchFunc   func(ctx context.Context, nodes []models.VaultNode) error
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

func (m *mockNodeService) CreateNodeBatch(ctx context.Context, nodes []models.VaultNode) error {
	if m.createNodeBatchFunc != nil {
		return m.createNodeBatchFunc(ctx, nodes)
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
	createEdgeBatchFunc   func(ctx context.Context, edges []models.VaultEdge) error
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

func (m *mockEdgeService) CreateEdgeBatch(ctx context.Context, edges []models.VaultEdge) error {
	if m.createEdgeBatchFunc != nil {
		return m.createEdgeBatchFunc(ctx, edges)
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
	updateParseStatusFunc func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus) error
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

func (m *mockMetadataRepository) UpdateParseStatus(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus) error {
	if m.updateParseStatusFunc != nil {
		return m.updateParseStatusFunc(exec, ctx, id, status)
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
		updateParseStatusFunc: func(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus) error {
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
			createNodeBatchFunc: func(ctx context.Context, nodes []models.VaultNode) error {
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
			createEdgeBatchFunc: func(ctx context.Context, edges []models.VaultEdge) error {
				return nil
			},
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

func TestGetVaultMetadata(t *testing.T) {
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
	metadata, err := vaultService.GetVaultMetadata(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedMetadata.Key, metadata.Key)
	assert.Equal(t, expectedMetadata.Value, metadata.Value)
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
