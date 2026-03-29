package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ali01/mnemosyne/internal/models"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	graph, err := s.store.GetGraph()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch graph"})
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

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

func (s *Server) handleSearchNodes(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Query parameter 'q' is required"})
		return
	}

	nodes, err := s.store.SearchNodes(query)
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

func (s *Server) handleUpdatePosition(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var pos models.NodePosition
	if err := readJSON(r, &pos); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}
	pos.NodeID = id

	if err := s.store.UpsertPosition(&pos); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update position"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Position updated"})
}

func (s *Server) handleUpdatePositions(w http.ResponseWriter, r *http.Request) {
	var positions []models.NodePosition
	if err := readJSON(r, &positions); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if len(positions) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No positions provided"})
		return
	}

	if err := s.store.UpsertPositions(positions); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update positions"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Positions updated",
		"count":   len(positions),
	})
}

func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
	if s.indexer == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Indexer not configured"})
		return
	}

	if err := s.indexer.FullIndex(); err != nil {
		log.Printf("Reindex failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Reindex failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Reindex completed"})
}

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
