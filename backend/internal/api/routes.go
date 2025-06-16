package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func SetupRoutes(router *gin.Engine, db *sql.DB, redis *redis.Client) {
	router.Use(CORSMiddleware())

	api := router.Group("/api/v1")
	{
		api.GET("/health", healthCheck)
		
		api.GET("/graph", getGraph(db, redis))
		api.GET("/graph/viewport", getViewportNodes(db, redis))
		api.PUT("/nodes/:id/position", updateNodePosition(db, redis))
		api.GET("/nodes/:id", getNode(db))
		api.GET("/nodes/:id/content", getNodeContent(db))
		
		api.GET("/clusters/:level", getClusters(db, redis))
		api.POST("/clusters/compute", computeClusters(db, redis))
	}
}

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