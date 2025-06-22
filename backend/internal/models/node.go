// Package models defines data structures for graph nodes, edges, and positions
package models

// Node represents a single node in the knowledge graph
type Node struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	FilePath string                 `json:"file_path,omitempty"`
	Content  string                 `json:"content,omitempty"`
	Position Position               `json:"position"`
	Level    int                    `json:"level"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Position represents the 3D coordinates of a node in the graph visualization
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z,omitempty"`
}

// Edge represents a connection between two nodes in the knowledge graph
type Edge struct {
	ID     string  `json:"id"`
	Source string  `json:"source"`
	Target string  `json:"target"`
	Weight float64 `json:"weight"`
	Type   string  `json:"type"`
}
