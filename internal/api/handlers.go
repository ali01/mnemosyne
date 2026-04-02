package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/ali01/mnemosyne/internal/discovery"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/search"
	"github.com/ali01/mnemosyne/internal/store"
	"gopkg.in/yaml.v3"
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

	raw, err := s.store.GetGraphDataRaw(graphID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch graph"})
		return
	}

	graph := applyFilterAndGroups(raw)
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

	if s.positionSync != nil {
		s.positionSync.MarkDirty(graphID)
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

	if s.positionSync != nil {
		s.positionSync.MarkDirty(graphID)
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

// --- Filter and group evaluation ---

// graphConfig is the parsed structure of a GRAPH.yaml file for filter/group evaluation.
type graphConfig struct {
	Filter string               `yaml:"filter"`
	Groups []discovery.GroupDef `yaml:"groups"`
}

// applyFilterAndGroups evaluates the graph's filter and groups against its nodes,
// pruning filtered-out nodes and assigning group colors.
func applyFilterAndGroups(raw *store.GraphDataRaw) *models.Graph {
	var cfg graphConfig
	if raw.Config != "" {
		if err := yaml.Unmarshal([]byte(raw.Config), &cfg); err != nil {
			log.Printf("Failed to parse graph config: %v (using defaults)", err)
		}
	}

	// Parse filter query (default: match all)
	filterQuery, err := search.Parse(cfg.Filter)
	if err != nil {
		log.Printf("Invalid filter query %q: %v (showing all nodes)", cfg.Filter, err)
		filterQuery, _ = search.Parse("*")
	}

	// Parse group queries
	type parsedGroup struct {
		query search.Query
		color string
	}
	var groups []parsedGroup
	for _, g := range cfg.Groups {
		q, err := search.Parse(g.Query)
		if err != nil {
			log.Printf("Invalid group query %q: %v (skipping group)", g.Query, err)
			continue
		}
		groups = append(groups, parsedGroup{query: q, color: g.Color})
	}

	// Evaluate filter and groups for each node
	nodeSet := make(map[string]bool)
	apiNodes := make([]models.Node, 0, len(raw.Nodes))
	for _, n := range raw.Nodes {
		nd := search.NodeData{
			FilePath:    n.FilePath,
			Title:       n.Title,
			Tags:        []string(n.Tags),
			Frontmatter: map[string]interface{}(n.Metadata),
		}

		if !filterQuery.Match(&nd) {
			continue
		}

		nodeSet[n.ID] = true

		// First matching group determines color
		var color string
		for _, g := range groups {
			if g.query.Match(&nd) {
				color = g.color
				break
			}
		}

		pos := raw.Positions[n.ID]
		apiNodes = append(apiNodes, models.Node{
			ID:       n.ID,
			Title:    n.Title,
			FilePath: n.FilePath,
			Position: models.Position{X: pos.X, Y: pos.Y, Z: pos.Z},
			Color:    color,
		})
	}

	// Prune edges where either endpoint was filtered out
	apiEdges := make([]models.Edge, 0, len(raw.Edges))
	for _, e := range raw.Edges {
		if nodeSet[e.SourceID] && nodeSet[e.TargetID] {
			apiEdges = append(apiEdges, models.Edge{
				ID:     e.ID,
				Source: e.SourceID,
				Target: e.TargetID,
				Weight: e.Weight,
				Type:   e.EdgeType,
			})
		}
	}

	return &models.Graph{Nodes: apiNodes, Edges: apiEdges}
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
