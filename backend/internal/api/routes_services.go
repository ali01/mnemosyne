package api

import (
	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/service"
	"github.com/gin-gonic/gin"
)

// SetupRoutesWithServices configures all API routes with service access
func SetupRoutesWithServices(router *gin.Engine, nodeService *service.NodeService, edgeService *service.EdgeService, positionService *service.PositionService, vaultService service.VaultServiceInterface, cfg *config.Config) {
	router.Use(CORSMiddleware())

	// Create handler with services
	h := &ServiceHandler{
		nodeService:     nodeService,
		edgeService:     edgeService,
		positionService: positionService,
		vaultService:    vaultService,
		cfg:             cfg,
	}

	api := router.Group("/api/v1")
	{
		api.GET("/health", healthCheck)

		// Graph endpoints
		api.GET("/graph", h.getGraph)
		api.GET("/graph/viewport", h.getViewportNodes)

		// Node endpoints
		api.GET("/nodes/:id", h.getNode)
		api.GET("/nodes/:id/content", h.getNodeContent)
		api.PUT("/nodes/:id/position", h.updateNodePosition)
		api.GET("/nodes/search", h.searchNodes)

		// Vault parse endpoints (RESTful resource-based)
		// TODO(CL): Consider adding DELETE /vault/parses/:id to cancel running parses
		api.POST("/vault/parses", h.createParse)           // Create new parse job
		api.GET("/vault/parses", h.listParses)            // List all parse history
		api.GET("/vault/parses/latest", h.getLatestParse) // Get latest parse status
		api.GET("/vault/parses/:id", h.getParseById)      // Get specific parse status
	}
}

// CORSMiddleware returns a Gin middleware that handles Cross-Origin Resource Sharing (CORS)
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}
