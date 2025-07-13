// +build integration

package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/api"
	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository/postgres"
	"github.com/ali01/mnemosyne/internal/service"
)

// TestVaultAPIIntegration tests the complete parsing pipeline through the API
func TestVaultAPIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database
	tdb, _ := postgres.CreateTestRepositories(t)
	defer tdb.Close()

	ctx := context.Background()
	require.NoError(t, tdb.CleanTables(ctx))

	// Track if we've parsed data
	var dataParsed bool

	// Create test vault
	testVaultDir := setupTestVaultForAPI(t)
	defer os.RemoveAll(testVaultDir)

	// Create services
	nodeService := service.NewNodeService(tdb.DB)
	edgeService := service.NewEdgeService(tdb.DB)
	positionService := service.NewPositionService(tdb.DB)
	metadataService := service.NewMetadataService(tdb.DB)

	// Create mock git manager
	gitManager := &mockGitManager{
		localPath: testVaultDir,
	}

	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			RequestTimeout: 30 * time.Second,
		},
		Graph: config.GraphConfig{
			MaxConcurrency: 4,
			BatchSize:      100,
			NodeClassification: config.NodeClassificationConfig{
				NodeTypes: map[string]config.NodeTypeConfig{
					"default": {
						DisplayName:    "Default",
						Color:          "#999999",
						SizeMultiplier: 1.0,
					},
				},
				ClassificationRules: []config.ClassificationRuleConfig{
					{
						Name:     "default",
						Priority: 100,
						Type:     "regex",
						Pattern:  ".*",
						NodeType: "default",
					},
				},
			},
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

	// Setup API router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api.SetupRoutesWithServices(router, nodeService, edgeService, positionService, vaultService, cfg)

	// Helper to ensure data is parsed
	ensureDataParsed := func(t *testing.T) {
		if dataParsed {
			return
		}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/vault/parses", nil)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		dataParsed = true
	}

	// Run tests sequentially to avoid interference
	t.Run("CompleteParsingPipeline", func(t *testing.T) {
		// Trigger parsing via API
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/vault/parses", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Logf("Response body: %s", w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/api/v1/vault/parses/")

		// Extract parse ID from response
		var parseResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &parseResponse)
		require.NoError(t, err)
		if parseResponse["id"] == nil {
			t.Fatalf("No parse ID in response: %v", parseResponse)
		}
		parseID := parseResponse["id"].(string)
		assert.NotEmpty(t, parseID)
		dataParsed = true

		// Check parse status
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/api/v1/vault/parses/latest", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var statusResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &statusResponse)
		require.NoError(t, err)
		assert.Equal(t, "completed", statusResponse["status"])

		// Verify graph data is available
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/api/v1/graph", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var graphResponse models.Graph
		err = json.Unmarshal(w.Body.Bytes(), &graphResponse)
		require.NoError(t, err)
		assert.Len(t, graphResponse.Nodes, 3)
		assert.GreaterOrEqual(t, len(graphResponse.Edges), 2) // At least 2 edges from the links
	})

	t.Run("ParseAlreadyInProgress", func(t *testing.T) {
		// Skip cleaning - reuse existing data

		// Set up blocking git pull
		blockChan := make(chan struct{})
		releaseChan := make(chan struct{})
		gitManager.pullFunc = func(ctx context.Context) error {
			close(blockChan)
			<-releaseChan
			return nil
		}

		// Start first parse
		go func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/vault/parses", nil)
			router.ServeHTTP(w, req)
		}()

		// Wait for first parse to start
		<-blockChan

		// Try second parse - should fail with 409 Conflict
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/vault/parses", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "Parse already in progress", errorResponse["error"])

		// Release the first parse
		close(releaseChan)
		gitManager.pullFunc = nil
	})

	t.Run("NodeContentRetrieval", func(t *testing.T) {
		// Ensure data is parsed
		ensureDataParsed(t)

		// Get all nodes to find one with content
		nodes, err := nodeService.GetAllNodes(ctx, 10, 0)
		require.NoError(t, err)
		require.NotEmpty(t, nodes, "No nodes found in database")

		// Get node content
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/nodes/%s/content", nodes[0].ID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var contentResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &contentResponse)
		require.NoError(t, err)
		assert.NotEmpty(t, contentResponse["content"])
		assert.NotNil(t, contentResponse["metadata"])
	})

	t.Run("NodePositionUpdate", func(t *testing.T) {
		// Ensure data is parsed
		ensureDataParsed(t)

		// Get a node
		nodes, err := nodeService.GetAllNodes(ctx, 1, 0)
		require.NoError(t, err)
		require.NotEmpty(t, nodes)

		// Update position
		position := models.NodePosition{
			X:      100.5,
			Y:      200.5,
			Z:      1.0,
			Locked: true,
		}

		body, _ := json.Marshal(position)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/nodes/%s/position", nodes[0].ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify position was saved
		savedPos, err := positionService.GetNodePosition(ctx, nodes[0].ID)
		require.NoError(t, err)
		assert.Equal(t, 100.5, savedPos.X)
		assert.Equal(t, 200.5, savedPos.Y)
		assert.Equal(t, 1.0, savedPos.Z)
		assert.True(t, savedPos.Locked)
	})

	t.Run("SearchNodes", func(t *testing.T) {
		// Ensure data is parsed
		ensureDataParsed(t)

		// Search for nodes
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/nodes/search?q=Note", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var searchResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &searchResponse)
		require.NoError(t, err)
		assert.Equal(t, "Note", searchResponse["query"])

		results := searchResponse["results"].([]interface{})
		assert.GreaterOrEqual(t, len(results), 1)
	})
}

// setupTestVaultForAPI creates a test vault with sample files
func setupTestVaultForAPI(t testing.TB) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "api-test-vault-*")
	require.NoError(t, err)

	files := map[string]string{
		"index.md": `---
id: home
title: Home Page
tags: [home]
---

# Welcome Home

This is the home page. See [[note1]] for more info.`,

		"note1.md": `---
id: note1
title: First Note
tags: [note]
---

# First Note

This is the first note. Links to [[note2]].`,

		"note2.md": `---
id: note2
title: Second Note
tags: [note, important]
---

# Second Note

This is the second note.`,
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	return dir
}

// mockGitManager for API tests
type mockGitManager struct {
	localPath string
	pullFunc  func(ctx context.Context) error
}

func (m *mockGitManager) Pull(ctx context.Context) error {
	if m.pullFunc != nil {
		return m.pullFunc(ctx)
	}
	return nil
}

func (m *mockGitManager) GetLocalPath() string {
	return m.localPath
}
