package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
	router.Use(CORSMiddleware())

	api := router.Group("/api/v1")
	{
		api.GET("/health", healthCheck)
		
		api.GET("/graph", getGraph())
		api.GET("/graph/viewport", getViewportNodes())
		api.PUT("/nodes/:id/position", updateNodePosition())
		api.GET("/nodes/:id", getNode())
		api.GET("/nodes/:id/content", getNodeContent())
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