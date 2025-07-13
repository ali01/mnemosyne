// Package api provides HTTP handlers with service access
package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

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

// createParse creates a new vault parse job (POST /vault/parses)
func (h *ServiceHandler) createParse(c *gin.Context) {
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
		// If we can't get status, check if it's because there's no parse history yet
		// In that case, we can proceed with a new parse
		if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get parse status"})
			return
		}
		// No parse history exists yet, we can proceed
	} else if status.Status == string(models.ParseStatusRunning) {
		c.JSON(http.StatusConflict, gin.H{"error": "Parse already in progress"})
		return
	}

	// Start parsing
	history, err := h.vaultService.ParseAndIndexVault(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": sanitizeError(err)})
		return
	}

	// Return created status with location header
	c.Header("Location", "/api/v1/vault/parses/"+history.ID)
	c.JSON(http.StatusCreated, sanitizeParseHistory(history))
}

// getLatestParse returns the status of the latest parse operation (GET /vault/parses/latest)
func (h *ServiceHandler) getLatestParse(c *gin.Context) {
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

	c.JSON(http.StatusOK, sanitizeParseStatus(status))
}

// getParseById returns the status of a specific parse by ID (GET /vault/parses/:id)
func (h *ServiceHandler) getParseById(c *gin.Context) {
	// Use request context
	ctx := c.Request.Context()
	parseID := c.Param("id")

	// Check if vault service is available
	if h.vaultService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Vault service not configured"})
		return
	}

	// TODO: Implement GetParseById in VaultService
	// For now, return not implemented
	_ = ctx // Mark as used
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Get parse by ID not yet implemented",
		"id":    parseID,
	})
}

// listParses returns a list of all parse history (GET /vault/parses)
func (h *ServiceHandler) listParses(c *gin.Context) {
	// Use request context
	ctx := c.Request.Context()

	// Check if vault service is available
	if h.vaultService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Vault service not configured"})
		return
	}

	// Get pagination parameters
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter. Must be between 1 and 100"})
		return
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter. Must be 0 or greater"})
		return
	}

	// TODO: Implement ListParses in VaultService
	// For now, return not implemented
	_ = ctx // Mark as used
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "List parses not yet implemented",
		"limit": limit,
		"offset": offset,
	})
}

// calculateNodeLevel calculates the level of a node based on its properties
func calculateNodeLevel(node models.VaultNode) int {
	// TODO: Implement actual level calculation based on node properties
	// For now, return a default level
	return 1
}

// sanitizeParseHistory removes sensitive information from ParseHistory for API responses
func sanitizeParseHistory(history *models.ParseHistory) gin.H {
	// Return only essential information without exposing system paths or detailed errors
	response := gin.H{
		"id":          history.ID,
		"started_at":  history.StartedAt,
		"status":      history.Status,
	}

	// Add completed_at if available
	if history.CompletedAt != nil {
		response["completed_at"] = history.CompletedAt
	}

	// Add sanitized stats - only counts, no paths or sensitive data
	// Check if Stats has any meaningful data
	if !history.Stats.IsEmpty() {
		stats := history.Stats.ToParseStats()
		response["stats"] = gin.H{
			"total_files":      stats.TotalFiles,
			"parsed_files":     stats.ParsedFiles,
			"total_nodes":      stats.TotalNodes,
			"total_edges":      stats.TotalEdges,
			"duration_ms":      stats.DurationMS,
			"unresolved_links": stats.UnresolvedLinks,
		}
	}

	// Sanitize error messages - provide generic error types instead of detailed messages
	if history.Error != nil && *history.Error != "" {
		// Map detailed errors to generic error types
		var errorType string
		errorMsg := *history.Error

		switch {
		case strings.Contains(errorMsg, "git"):
			errorType = "git_sync_error"
		case strings.Contains(errorMsg, "parse"):
			errorType = "parse_error"
		case strings.Contains(errorMsg, "database"):
			errorType = "storage_error"
		case strings.Contains(errorMsg, "timeout"):
			errorType = "timeout_error"
		default:
			errorType = "processing_error"
		}

		response["error"] = gin.H{
			"type": errorType,
			"message": "An error occurred during vault processing. Please check logs for details.",
		}
	}

	return response
}

// sanitizeParseStatus removes sensitive information from ParseStatusResponse for API responses
func sanitizeParseStatus(status *models.ParseStatusResponse) gin.H {
	// Return only essential information without exposing system details
	response := gin.H{
		"status": status.Status,
	}

	// Add timestamps if available
	if status.StartedAt != nil {
		response["started_at"] = status.StartedAt
	}
	if status.CompletedAt != nil {
		response["completed_at"] = status.CompletedAt
	}

	// Add sanitized progress information
	if status.Progress != nil {
		response["progress"] = gin.H{
			"total_files":     status.Progress.TotalFiles,
			"processed_files": status.Progress.ProcessedFiles,
			"nodes_created":   status.Progress.NodesCreated,
			"edges_created":   status.Progress.EdgesCreated,
			"error_count":     status.Progress.ErrorCount,
		}
	}

	// Sanitize error messages - same approach as ParseHistory
	if status.Error != "" {
		var errorType string

		switch {
		case strings.Contains(status.Error, "git"):
			errorType = "git_sync_error"
		case strings.Contains(status.Error, "parse"):
			errorType = "parse_error"
		case strings.Contains(status.Error, "database"):
			errorType = "storage_error"
		case strings.Contains(status.Error, "timeout"):
			errorType = "timeout_error"
		default:
			errorType = "processing_error"
		}

		response["error"] = gin.H{
			"type": errorType,
			"message": "An error occurred during vault processing. Please check logs for details.",
		}
	}

	return response
}

// sanitizeError removes sensitive information from error messages
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Map detailed errors to generic messages
	// Check specific cases first before more general patterns
	switch {
	case strings.Contains(errMsg, "already in progress"):
		return "Parse already in progress"
	case strings.Contains(errMsg, "panic"):
		return "An unexpected error occurred during parsing"
	case strings.Contains(errMsg, "timeout"):
		return "Operation timed out"
	case strings.Contains(errMsg, "git"):
		return "Failed to sync vault repository"
	case strings.Contains(errMsg, "parse"):
		return "Failed to parse vault files"
	case strings.Contains(errMsg, "database"), strings.Contains(errMsg, "postgres"):
		return "Storage operation failed"
	default:
		// Remove any file paths or system-specific information
		// This is a catch-all that returns a generic message
		return "Vault processing failed"
	}
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
		// Don't expose internal error details, use the provided message
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
	}
}
