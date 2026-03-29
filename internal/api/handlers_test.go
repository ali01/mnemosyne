package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

// seedGraph creates a vault, a graph, two nodes, an edge, and a position.
// Returns the graph ID.
func seedGraph(t *testing.T, s *store.Store) int {
	t.Helper()
	vid, err := s.UpsertVault("test", "/test")
	require.NoError(t, err)
	gid, err := s.UpsertGraph(vid, "root", "", "")
	require.NoError(t, err)

	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "a", VaultID: vid, Title: "Aviation", FilePath: "aviation.md", NodeType: "hub",
		Content: "All about aviation.", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.UpsertNode(&models.VaultNode{
		ID: "b", VaultID: vid, Title: "Economics", FilePath: "econ.md", NodeType: "note",
		Content: "Supply and demand.", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.UpsertEdge(&models.VaultEdge{
		ID: "e1", SourceID: "a", TargetID: "b", EdgeType: "wikilink", Weight: 1,
	}))
	require.NoError(t, s.ReplaceGraphMemberships("a", []int{gid}))
	require.NoError(t, s.ReplaceGraphMemberships("b", []int{gid}))
	require.NoError(t, s.UpsertPosition(gid, &models.NodePosition{NodeID: "a", X: 10, Y: 20}))

	return gid
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
}

// --- Graph List ---

func TestListGraphs(t *testing.T) {
	srv, s := newTestServer(t)
	seedGraph(t, s)

	w := doRequest(srv.Handler(), "GET", "/api/v1/graphs", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Graphs []models.GraphInfo `json:"graphs"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Graphs, 1)
	assert.Equal(t, "root", resp.Graphs[0].Name)
	assert.Equal(t, 2, resp.Graphs[0].NodeCount)
}

func TestListGraphsEmpty(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "GET", "/api/v1/graphs", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Graphs []models.GraphInfo `json:"graphs"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Graphs)
}

// --- Graph Data ---

func TestGetGraphData(t *testing.T) {
	srv, s := newTestServer(t)
	gid := seedGraph(t, s)

	w := doRequest(srv.Handler(), "GET", "/api/v1/graphs/"+strconv.Itoa(gid), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var graph models.Graph
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &graph))
	assert.Len(t, graph.Nodes, 2)
	assert.Len(t, graph.Edges, 1)

	for _, n := range graph.Nodes {
		if n.ID == "a" {
			assert.InDelta(t, 10, n.Position.X, 0.01)
		}
	}
}

func TestGetGraphDataInvalidID(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "GET", "/api/v1/graphs/abc", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Search ---

func TestSearchInGraph(t *testing.T) {
	srv, s := newTestServer(t)
	gid := seedGraph(t, s)

	w := doRequest(srv.Handler(), "GET", "/api/v1/graphs/"+strconv.Itoa(gid)+"/search?q=aviation", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Nodes []models.Node `json:"nodes"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Nodes, 1)
}

func TestSearchInGraphMissingQuery(t *testing.T) {
	srv, s := newTestServer(t)
	gid := seedGraph(t, s)

	w := doRequest(srv.Handler(), "GET", "/api/v1/graphs/"+strconv.Itoa(gid)+"/search", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
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

func TestGetNodeContent(t *testing.T) {
	srv, s := newTestServer(t)
	seedGraph(t, s)

	w := doRequest(srv.Handler(), "GET", "/api/v1/nodes/a/content", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "All about aviation.", resp["content"])
}

// --- Positions ---

func TestUpdateGraphPosition(t *testing.T) {
	srv, s := newTestServer(t)
	gid := seedGraph(t, s)

	pos := models.NodePosition{X: 42, Y: 99}
	w := doRequest(srv.Handler(), "PUT", "/api/v1/graphs/"+strconv.Itoa(gid)+"/positions/a", pos)
	assert.Equal(t, http.StatusOK, w.Code)

	graph, _ := s.GetGraphData(gid)
	for _, n := range graph.Nodes {
		if n.ID == "a" {
			assert.InDelta(t, 42, n.Position.X, 0.01)
		}
	}
}

func TestUpdateGraphPositionsBatch(t *testing.T) {
	srv, s := newTestServer(t)
	gid := seedGraph(t, s)

	batch := []models.NodePosition{
		{NodeID: "a", X: 1, Y: 2},
		{NodeID: "b", X: 3, Y: 4},
	}
	w := doRequest(srv.Handler(), "PUT", "/api/v1/graphs/"+strconv.Itoa(gid)+"/positions", batch)
	assert.Equal(t, http.StatusOK, w.Code)

	graph, _ := s.GetGraphData(gid)
	assert.Len(t, graph.Nodes, 2)
}

func TestUpdateGraphPositionsEmpty(t *testing.T) {
	srv, s := newTestServer(t)
	gid := seedGraph(t, s)

	w := doRequest(srv.Handler(), "PUT", "/api/v1/graphs/"+strconv.Itoa(gid)+"/positions", []models.NodePosition{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Reindex ---

func TestReindexNoIndexer(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "POST", "/api/v1/reindex", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- CORS ---

func TestCORSPreflight(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "OPTIONS", "/api/v1/graphs", nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSHeaders(t *testing.T) {
	srv, _ := newTestServer(t)
	w := doRequest(srv.Handler(), "GET", "/api/v1/health", nil)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
