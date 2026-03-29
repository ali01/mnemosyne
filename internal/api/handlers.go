package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/ali01/mnemosyne/internal/models"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Graph listing and data ---

func (s *Server) handleListGraphs(w http.ResponseWriter, r *http.Request) {
	graphs, err := s.store.GetAllGraphs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch graphs"})
		return
	}
	if graphs == nil {
		graphs = []models.GraphInfo{}
	}
	resp := map[string]interface{}{"graphs": graphs}
	if s.homeGraph != "" {
		resp["home_graph"] = s.homeGraph
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetGraphData(w http.ResponseWriter, r *http.Request) {
	graphID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid graph ID"})
		return
	}

	graph, err := s.store.GetGraphData(graphID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch graph"})
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *Server) handleSearchInGraph(w http.ResponseWriter, r *http.Request) {
	graphID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid graph ID"})
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Query parameter 'q' is required"})
		return
	}

	nodes, err := s.store.SearchInGraph(graphID, query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Search failed"})
		return
	}

	apiNodes := make([]models.Node, 0, len(nodes))
	for _, n := range nodes {
		apiNodes = append(apiNodes, models.Node{
			ID:       n.ID,
			Title:    n.Title,
			FilePath: n.FilePath,
			Metadata: map[string]interface{}{"type": n.NodeType},
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"nodes": apiNodes})
}

// --- Graph-scoped positions ---

func (s *Server) handleUpdateGraphPosition(w http.ResponseWriter, r *http.Request) {
	graphID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid graph ID"})
		return
	}
	nodeID := r.PathValue("nodeId")

	var pos models.NodePosition
	if err := readJSON(r, &pos); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}
	pos.NodeID = nodeID

	if err := s.store.UpsertPosition(graphID, &pos); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update position"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Position updated"})
}

func (s *Server) handleUpdateGraphPositions(w http.ResponseWriter, r *http.Request) {
	graphID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid graph ID"})
		return
	}

	var positions []models.NodePosition
	if err := readJSON(r, &positions); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if len(positions) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No positions provided"})
		return
	}

	if err := s.store.UpsertPositions(graphID, positions); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update positions"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Positions updated",
		"count":   len(positions),
	})
}

// --- Node content (global, not graph-scoped) ---

func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	node, err := s.store.GetNode(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Node not found"})
		return
	}

	writeJSON(w, http.StatusOK, models.Node{
		ID:       node.ID,
		Title:    node.Title,
		FilePath: node.FilePath,
		Metadata: map[string]interface{}{"type": node.NodeType},
	})
}

func (s *Server) handleGetNodeContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	node, err := s.store.GetNode(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Node not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"title":    node.Title,
		"content":  node.Content,
		"metadata": node.Metadata,
	})
}

// --- Reindex ---

func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
	if s.indexer == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Indexer not configured"})
		return
	}

	if err := s.indexer.FullIndexAll(); err != nil {
		log.Printf("Reindex failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Reindex failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Reindex completed"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(nil, r.Body, 10<<20) // 10MB limit
	return json.NewDecoder(r.Body).Decode(v)
}
