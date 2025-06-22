// Package api provides HTTP handlers for the Mnemosyne graph visualization API
package api

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/gin-gonic/gin"
)

// In-memory storage for node positions
var (
	nodePositions = make(map[string]models.Position)
	positionsMu   sync.RWMutex
)

func getGraph() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read from sample file
		data, err := os.ReadFile("data/sample_graph.json")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read sample data"})
			return
		}

		var graphData struct {
			Nodes []struct {
				ID       string          `json:"id"`
				Title    string          `json:"title"`
				Position models.Position `json:"position"`
				Type     string          `json:"type"`
				Level    int             `json:"level"`
			} `json:"nodes"`
			Edges []struct {
				ID     string  `json:"id"`
				Source string  `json:"source"`
				Target string  `json:"target"`
				Weight float64 `json:"weight"`
				Type   string  `json:"type"`
			} `json:"edges"`
		}

		if err := json.Unmarshal(data, &graphData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse sample data"})
			return
		}

		// Convert to models and apply any saved positions
		nodes := make([]models.Node, len(graphData.Nodes))
		positionsMu.RLock()
		for i, n := range graphData.Nodes {
			position := n.Position
			// Override with saved position if exists
			if savedPos, exists := nodePositions[n.ID]; exists {
				position = savedPos
			}

			nodes[i] = models.Node{
				ID:       n.ID,
				Title:    n.Title,
				Position: position,
				Level:    n.Level,
				Metadata: map[string]interface{}{
					"type": n.Type,
				},
			}
		}
		positionsMu.RUnlock()

		edges := make([]models.Edge, len(graphData.Edges))
		for i, e := range graphData.Edges {
			edges[i] = models.Edge{
				ID:     e.ID,
				Source: e.Source,
				Target: e.Target,
				Weight: e.Weight,
				Type:   e.Type,
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"nodes": nodes,
			"edges": edges,
			"level": "0",
		})
	}
}

func getViewportNodes() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For now, just return all nodes
		// In the future, this would filter based on viewport bounds
		getGraph()(c)
	}
}

func updateNodePosition() gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("id")

		var position models.Position
		if err := c.ShouldBindJSON(&position); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Save position in memory
		positionsMu.Lock()
		nodePositions[nodeID] = position
		positionsMu.Unlock()

		c.JSON(http.StatusOK, gin.H{
			"id":       nodeID,
			"position": position,
		})
	}
}

func getNode() gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("id")

		// Read the graph data to find the node
		data, err := os.ReadFile("data/sample_graph.json")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read data"})
			return
		}

		var graphData struct {
			Nodes []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				Type  string `json:"type"`
			} `json:"nodes"`
		}

		if err := json.Unmarshal(data, &graphData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse data"})
			return
		}

		for _, n := range graphData.Nodes {
			if n.ID == nodeID {
				c.JSON(http.StatusOK, gin.H{
					"id":    n.ID,
					"title": n.Title,
					"type":  n.Type,
				})
				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
	}
}

func getNodeContent() gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("id")

		// For now, return sample markdown content
		content := "# Sample Content\n\nThis is placeholder content for node " + nodeID + ".\n\n## Overview\n\nIn a real implementation, this would load the actual Obsidian note content."

		c.JSON(http.StatusOK, gin.H{
			"id":      nodeID,
			"content": content,
			"html":    "<h1>Sample Content</h1><p>This is placeholder content for node " + nodeID + ".</p>",
		})
	}
}
