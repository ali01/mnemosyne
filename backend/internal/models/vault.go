// Package models defines data structures for vault-specific graph nodes and edges
package models

import (
	"fmt"
	"time"
)

// VaultNode represents a node in the knowledge graph with vault-specific metadata
// Bridge between file-centric parser output and node-centric graph visualization
type VaultNode struct {
	ID         string       `json:"id" db:"id" validate:"required,min=1"`                     // Required: from frontmatter
	Title      string       `json:"title" db:"title" validate:"required,min=1"`               // From frontmatter or filename fallback
	NodeType   string       `json:"node_type" db:"node_type" validate:"omitempty,min=1"`      // Calculated node type from configuration
	Tags       StringArray  `json:"tags,omitempty" db:"tags" validate:"omitempty,dive,min=1"` // From frontmatter tags field
	Content    string       `json:"content,omitempty" db:"content"`                           // Full markdown content
	Metadata   JSONMetadata `json:"metadata,omitempty" db:"metadata"`                         // All frontmatter fields
	FilePath   string       `json:"file_path" db:"file_path" validate:"required,min=1"`       // Original file location
	InDegree   int          `json:"in_degree" db:"in_degree" validate:"min=0"`                // Number of incoming links
	OutDegree  int          `json:"out_degree" db:"out_degree" validate:"min=0"`              // Number of outgoing links
	Centrality float64      `json:"centrality" db:"centrality" validate:"min=0,max=1"`        // PageRank or similar metric
	CreatedAt  time.Time    `json:"created_at" db:"created_at" validate:"required"`
	UpdatedAt  time.Time    `json:"updated_at" db:"updated_at" validate:"required"`
}

// VaultEdge represents a connection between ideas in the knowledge graph
// Supports different link types and preserves context through display text
type VaultEdge struct {
	ID          string    `json:"id" db:"id" validate:"required,uuid4"`                                // Auto-generated UUID
	SourceID    string    `json:"source_id" db:"source_id" validate:"required,min=1"`                  // Node ID of link source
	TargetID    string    `json:"target_id" db:"target_id" validate:"required,min=1,nefield=SourceID"` // Node ID of link target
	EdgeType    string    `json:"edge_type" db:"edge_type" validate:"required,oneof=wikilink embed"`   // "wikilink" or "embed"
	DisplayText string    `json:"display_text,omitempty" db:"display_text"`                            // Link alias or section reference
	Weight      float64   `json:"weight" db:"weight" validate:"min=0"`                                 // Default 1.0, for future use
	CreatedAt   time.Time `json:"created_at" db:"created_at" validate:"required"`
}

// NodePosition represents the position of a node in 3D space with persistence metadata
// This extends the simple Position type with database-specific fields for tracking
type NodePosition struct {
	NodeID    string    `db:"node_id" json:"node_id" validate:"required,min=1"`
	X         float64   `db:"x" json:"x"`
	Y         float64   `db:"y" json:"y"`
	Z         float64   `db:"z" json:"z"`
	Locked    bool      `db:"locked" json:"locked"` // Whether user has manually positioned this node
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ToPosition converts NodePosition to the simple Position type for API responses
func (np *NodePosition) ToPosition() Position {
	return Position{
		X: np.X,
		Y: np.Y,
		Z: np.Z,
	}
}

// VaultMetadata stores key-value metadata about the vault
type VaultMetadata struct {
	Key       string    `db:"key" json:"key" validate:"required,min=1"`
	Value     string    `db:"value" json:"value"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ParseHistory tracks vault parsing operations
type ParseHistory struct {
	ID          string      `db:"id" json:"id" validate:"required"`
	StartedAt   time.Time   `db:"started_at" json:"started_at"`
	CompletedAt *time.Time  `db:"completed_at" json:"completed_at"`
	Status      ParseStatus `db:"status" json:"status" validate:"required"`
	Stats       JSONStats   `db:"stats" json:"stats"`
	Error       *string     `db:"error" json:"error,omitempty"`
}

// ParseStatus represents the status of a parse operation
type ParseStatus string

const (
	ParseStatusIdle      ParseStatus = "idle"      // No parse has been performed
	ParseStatusPending   ParseStatus = "pending"
	ParseStatusRunning   ParseStatus = "running"
	ParseStatusCompleted ParseStatus = "completed"
	ParseStatusFailed    ParseStatus = "failed"
)

// ParseStats contains statistics about a parse operation
type ParseStats struct {
	TotalFiles      int   `json:"total_files"`
	ParsedFiles     int   `json:"parsed_files"`
	TotalNodes      int   `json:"total_nodes"`
	TotalEdges      int   `json:"total_edges"`
	DurationMS      int64 `json:"duration_ms"` // Duration in milliseconds
	UnresolvedLinks int   `json:"unresolved_links"`
}

// Validate performs validation on VaultNode fields
func (n *VaultNode) Validate() error {
	if n.ID == "" {
		return fmt.Errorf("node ID is required")
	}
	if n.Title == "" {
		return fmt.Errorf("node title is required")
	}
	if n.FilePath == "" {
		return fmt.Errorf("node file path is required")
	}
	if n.InDegree < 0 {
		return fmt.Errorf("node in_degree cannot be negative")
	}
	if n.OutDegree < 0 {
		return fmt.Errorf("node out_degree cannot be negative")
	}
	if n.Centrality < 0 || n.Centrality > 1 {
		return fmt.Errorf("node centrality must be between 0 and 1")
	}
	return nil
}

// Validate performs validation on VaultEdge fields
func (e *VaultEdge) Validate() error {
	if e.SourceID == "" {
		return fmt.Errorf("edge source ID is required")
	}
	if e.TargetID == "" {
		return fmt.Errorf("edge target ID is required")
	}
	if e.SourceID == e.TargetID {
		return fmt.Errorf("self-referential edges are not allowed")
	}
	if e.EdgeType != "wikilink" && e.EdgeType != "embed" {
		return fmt.Errorf("edge type must be 'wikilink' or 'embed', got: %s", e.EdgeType)
	}
	if e.Weight < 0 {
		return fmt.Errorf("edge weight cannot be negative")
	}
	return nil
}
