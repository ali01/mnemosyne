// Package api provides HTTP handlers with service access
package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/service"
	"github.com/gin-gonic/gin"
)

// ServiceHandler holds dependencies for HTTP handlers using services
type ServiceHandler struct {
	nodeService     *service.NodeService
	edgeService     *service.EdgeService
	positionService *service.PositionService
	vaultService    service.VaultServiceInterface
	cfg             *config.Config
}

// getGraph returns the full graph data from the database
func (h *ServiceHandler) getGraph(c *gin.Context) {
	// Use request context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.cfg.Server.RequestTimeout)
	defer cancel()

	// Get pagination parameters with validation
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	if err != nil || limit < 1 || limit > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter. Must be between 1 and 10000"})
		return
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter. Must be 0 or greater"})
		return
	}

	// Get all nodes
	vaultNodes, err := h.nodeService.GetAllNodes(ctx, limit, offset)
	if err != nil {
		handleError(c, err, "Failed to fetch nodes")
		return
	}

	// Get all edges
	vaultEdges, err := h.edgeService.GetAllEdges(ctx, limit, offset)
	if err != nil {
		handleError(c, err, "Failed to fetch edges")
		return
	}

	// Get all positions
	positions, err := h.positionService.GetAllPositions(ctx)
	if err != nil {
		// Positions are optional, so we just log and continue
		positions = []models.NodePosition{}
	}

	// Convert to API format
	nodes := make([]models.Node, 0, len(vaultNodes))
	for _, vn := range vaultNodes {
		nodes = append(nodes, models.Node{
			ID:       vn.ID,
			Title:    vn.Title,
			FilePath: vn.FilePath,
			Level:    calculateNodeLevel(vn), // TODO: Implement level calculation
			Position: models.Position{
				X: 0, // Will be set from positions
				Y: 0,
				Z: 0,
			},
		})
	}

	// Apply positions
	posMap := make(map[string]models.NodePosition)
	for _, pos := range positions {
		posMap[pos.NodeID] = pos
	}

	for i := range nodes {
		if pos, ok := posMap[nodes[i].ID]; ok {
			nodes[i].Position.X = pos.X
			nodes[i].Position.Y = pos.Y
			nodes[i].Position.Z = pos.Z
		}
	}

	// Convert edges
	edges := make([]models.Edge, 0, len(vaultEdges))
	for _, ve := range vaultEdges {
		edges = append(edges, models.Edge{
			ID:     ve.ID,
			Source: ve.SourceID,
			Target: ve.TargetID,
			Type:   ve.EdgeType,
			Weight: ve.Weight,
		})
	}

	// Build graph response
	graph := models.Graph{
		Nodes: nodes,
		Edges: edges,
	}

	c.JSON(http.StatusOK, graph)
}

// getNode returns a single node by ID
func (h *ServiceHandler) getNode(c *gin.Context) {
	// Use request context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.cfg.Server.RequestTimeout)
	defer cancel()
	nodeID := c.Param("id")

	node, err := h.nodeService.GetNode(ctx, nodeID)
	if err != nil {
		handleError(c, err, "Node not found")
		return
	}

	// Convert to API format
	apiNode := models.Node{
		ID:       node.ID,
		Title:    node.Title,
		FilePath: node.FilePath,
		Level:    calculateNodeLevel(*node),
	}

	c.JSON(http.StatusOK, apiNode)
}

// getNodeContent returns the markdown content of a node
func (h *ServiceHandler) getNodeContent(c *gin.Context) {
	// Use request context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.cfg.Server.RequestTimeout)
	defer cancel()
	nodeID := c.Param("id")

	node, err := h.nodeService.GetNode(ctx, nodeID)
	if err != nil {
		handleError(c, err, "Node not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content":  node.Content,
		"metadata": node.Metadata,
	})
}

// updateNodePosition updates the position of a node
func (h *ServiceHandler) updateNodePosition(c *gin.Context) {
	// Use request context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.cfg.Server.RequestTimeout)
	defer cancel()
	nodeID := c.Param("id")

	var position models.NodePosition
	if err := c.ShouldBindJSON(&position); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	position.NodeID = nodeID
	err := h.positionService.UpdateNodePosition(ctx, &position)
	if err != nil {
		handleError(c, err, "Failed to update position")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Position updated successfully"})
}

// searchNodes performs full-text search on nodes
func (h *ServiceHandler) searchNodes(c *gin.Context) {
	// Use request context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.cfg.Server.RequestTimeout)
	defer cancel()
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	nodes, err := h.nodeService.SearchNodes(ctx, query)
	if err != nil {
		handleError(c, err, "Search failed")
		return
	}

	// Convert to API format
	apiNodes := make([]models.Node, 0, len(nodes))
	for _, node := range nodes {
		apiNodes = append(apiNodes, models.Node{
			ID:       node.ID,
			Title:    node.Title,
			FilePath: node.FilePath,
			Level:    calculateNodeLevel(node),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"results": apiNodes,
		"count":   len(apiNodes),
	})
}

// getViewportNodes returns nodes within a specific viewport
func (h *ServiceHandler) getViewportNodes(c *gin.Context) {
	// TODO: Implement viewport filtering
	h.getGraph(c) // For now, return all nodes
}

// parseVault triggers a vault parsing operation
func (h *ServiceHandler) parseVault(c *gin.Context) {
	// Use request context
	ctx := c.Request.Context()

	// Check if vault service is available
	if h.vaultService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Vault service not configured"})
		return
	}

	// Check if parse is already running
	status, err := h.vaultService.GetParseStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get parse status"})
		return
	}

	if status.Status == "parsing" {
		c.JSON(http.StatusConflict, gin.H{"error": "Parse already in progress"})
		return
	}

	// Start parsing
	history, err := h.vaultService.ParseAndIndexVault(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// getParseStatus returns the status of the latest parse operation
func (h *ServiceHandler) getParseStatus(c *gin.Context) {
	// Use request context
	ctx := c.Request.Context()

	// Check if vault service is available
	if h.vaultService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Vault service not configured"})
		return
	}

	// Get parse status
	status, err := h.vaultService.GetParseStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get parse status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// calculateNodeLevel calculates the level of a node based on its properties
func calculateNodeLevel(node models.VaultNode) int {
	// TODO: Implement actual level calculation based on node properties
	// For now, return a default level
	return 1
}

// handleError processes errors and returns appropriate HTTP responses
func handleError(c *gin.Context, err error, message string) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "Request timeout"})
	case errors.Is(err, context.Canceled):
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "Request canceled"})
	case service.IsNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": message})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
	}
}
