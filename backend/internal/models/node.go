package models

import (
	"time"
)

type Node struct {
	ID          string                 `json:"id" db:"id"`
	Title       string                 `json:"title" db:"title"`
	FilePath    string                 `json:"file_path" db:"file_path"`
	Content     string                 `json:"content,omitempty" db:"content"`
	Position    Position               `json:"position" db:"position"`
	ClusterID   *string                `json:"cluster_id,omitempty" db:"cluster_id"`
	Level       int                    `json:"level" db:"level"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z,omitempty"`
}

type Edge struct {
	ID        string    `json:"id" db:"id"`
	Source    string    `json:"source" db:"source"`
	Target    string    `json:"target" db:"target"`
	Weight    float64   `json:"weight" db:"weight"`
	Type      string    `json:"type" db:"type"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Cluster struct {
	ID           string    `json:"id" db:"id"`
	Level        int       `json:"level" db:"level"`
	CenterNode   string    `json:"center_node" db:"center_node"`
	NodeCount    int       `json:"node_count" db:"node_count"`
	Position     Position  `json:"position" db:"position"`
	Radius       float64   `json:"radius" db:"radius"`
	ComputedAt   time.Time `json:"computed_at" db:"computed_at"`
}