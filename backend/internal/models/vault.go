// Package models defines data structures for vault-specific graph nodes and edges
package models

import "time"

// VaultNode represents a node in the knowledge graph with vault-specific metadata
// Bridge between file-centric parser output and node-centric graph visualization
type VaultNode struct {
	ID          string                 `json:"id" db:"id" validate:"required,min=1"`                    // Required: from frontmatter
	Title       string                 `json:"title" db:"title" validate:"required,min=1"`                 // From frontmatter or filename fallback
	NodeType    string                 `json:"node_type" db:"node_type" validate:"omitempty,oneof=index hub concept project question note"` // Calculated: index/hub/concept/project/question/note
	Tags        []string               `json:"tags,omitempty" db:"tags" validate:"omitempty,dive,min=1"` // From frontmatter tags field
	Content     string                 `json:"content,omitempty" db:"content"`                               // Full markdown content
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`                              // All frontmatter fields
	FilePath    string                 `json:"file_path" db:"file_path" validate:"required,min=1"`             // Original file location
	InDegree    int                    `json:"in_degree" db:"in_degree" validate:"min=0"`                      // Number of incoming links
	OutDegree   int                    `json:"out_degree" db:"out_degree" validate:"min=0"`                     // Number of outgoing links
	Centrality  float64                `json:"centrality" db:"centrality" validate:"min=0,max=1"`               // PageRank or similar metric
	CreatedAt   time.Time              `json:"created_at" db:"created_at" validate:"required"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at" validate:"required"`
}

// VaultEdge represents a connection between ideas in the knowledge graph
// Supports different link types and preserves context through display text
type VaultEdge struct {
	ID          string    `json:"id" db:"id" validate:"required,uuid4"`                    // Auto-generated UUID
	SourceID    string    `json:"source_id" db:"source_id" validate:"required,min=1"`             // Node ID of link source
	TargetID    string    `json:"target_id" db:"target_id" validate:"required,min=1,nefield=SourceID"` // Node ID of link target
	EdgeType    string    `json:"edge_type" db:"edge_type" validate:"required,oneof=wikilink embed"`   // "wikilink" or "embed"
	DisplayText string    `json:"display_text,omitempty" db:"display_text"`                          // Link alias or section reference
	Weight      float64   `json:"weight" db:"weight" validate:"min=0"`                         // Default 1.0, for future use
	CreatedAt   time.Time `json:"created_at" db:"created_at" validate:"required"`
}