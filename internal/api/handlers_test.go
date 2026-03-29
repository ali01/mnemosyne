package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	s, err := store.NewMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	srv := NewServer(s, nil, nil, 0)
	return srv, s
}

func seedGraph(t *testing.T, s *store.Store) {
	t.Helper()
	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "a", Title: "Aviation", FilePath: "aviation.md", NodeType: "hub",
		Content: "All about aviation.", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "b", Title: "Economics", FilePath: "econ.md", NodeType: "note",
		Content: "Supply and demand.", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{
		ID: "e1", SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1,
	}))
	require.NoError(t, s.UpsertPosition(&models.NodePosition{NodeID: "a", X: 10, Y: 20}))
}

func doRequest(handler http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// --- Health ---

func TestHealth(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "GET", "/api/v1/health", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "ok", resp["status"])
}

// --- Graph ---

func TestGetGraph(t *testing.T) {
	srv, s := newTestServer(t)
	seedGraph(t, s)

	w := doRequest(srv.Handler(), "GET", "/api/v1/graph", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var graph models.Graph
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &graph))
	assert.Len(t, graph.Nodes, 2)
	assert.Len(t, graph.Edges, 1)

	// Check position was applied
	for _, n := range graph.Nodes {
		if n.ID == "a" {
			assert.InDelta(t, 10, n.Position.X, 0.01)
			assert.InDelta(t, 20, n.Position.Y, 0.01)
		}
	}
}

func TestGetGraphEmpty(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "GET", "/api/v1/graph", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var graph models.Graph
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &graph))
	assert.Empty(t, graph.Nodes)
	assert.Empty(t, graph.Edges)
}

// --- Node ---

func TestGetNode(t *testing.T) {
	srv, s := newTestServer(t)
	seedGraph(t, s)

	w := doRequest(srv.Handler(), "GET", "/api/v1/nodes/a", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var node models.Node
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &node))
	assert.Equal(t, "a", node.ID)
	assert.Equal(t, "Aviation", node.Title)
}

func TestGetNodeNotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "GET", "/api/v1/nodes/nonexistent", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Node Content ---

func TestGetNodeContent(t *testing.T) {
	srv, s := newTestServer(t)
	seedGraph(t, s)

	w := doRequest(srv.Handler(), "GET", "/api/v1/nodes/a/content", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "All about aviation.", resp["content"])
}

// --- Search ---

func TestSearchNodes(t *testing.T) {
	srv, s := newTestServer(t)
	seedGraph(t, s)

	w := doRequest(srv.Handler(), "GET", "/api/v1/nodes/search?q=aviation", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	nodes := resp["nodes"].([]interface{})
	assert.Len(t, nodes, 1)
}

func TestSearchNodesMissingQuery(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "GET", "/api/v1/nodes/search", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Positions ---

func TestUpdatePosition(t *testing.T) {
	srv, s := newTestServer(t)

	pos := models.NodePosition{X: 42, Y: 99}
	w := doRequest(srv.Handler(), "PUT", "/api/v1/nodes/mynode/position", pos)
	assert.Equal(t, http.StatusOK, w.Code)

	positions, _ := s.GetAllPositions()
	assert.Len(t, positions, 1)
	assert.Equal(t, "mynode", positions[0].NodeID)
	assert.InDelta(t, 42, positions[0].X, 0.01)
}

func TestUpdatePositionsBatch(t *testing.T) {
	srv, s := newTestServer(t)

	batch := []models.NodePosition{
		{NodeID: "a", X: 1, Y: 2},
		{NodeID: "b", X: 3, Y: 4},
	}
	w := doRequest(srv.Handler(), "PUT", "/api/v1/nodes/positions", batch)
	assert.Equal(t, http.StatusOK, w.Code)

	positions, _ := s.GetAllPositions()
	assert.Len(t, positions, 2)
}

func TestUpdatePositionsEmpty(t *testing.T) {
	srv, _ := newTestServer(t)

	w := doRequest(srv.Handler(), "PUT", "/api/v1/nodes/positions", []models.NodePosition{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Reindex ---

func TestReindexNoIndexer(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "POST", "/api/v1/vault/reindex", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- CORS ---

func TestCORSPreflight(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "OPTIONS", "/api/v1/graph", nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSHeaders(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "GET", "/api/v1/health", nil)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
