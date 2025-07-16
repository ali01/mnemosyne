// Package models defines data structures for the application
package models

import "time"

// ParseStatusResponse represents the current status of a parse operation
// This is used for API responses to check parsing progress
type ParseStatusResponse struct {
	Status      string         `json:"status"`                    // "idle", "pending", "running", "completed", "failed"
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Progress    *ParseProgress `json:"progress,omitempty"`
	Error       string         `json:"error,omitempty"`
}

// ParseProgress represents real-time progress of a parse operation
type ParseProgress struct {
	TotalFiles     int `json:"total_files"`
	ProcessedFiles int `json:"processed_files"`
	NodesCreated   int `json:"nodes_created"`
	EdgesCreated   int `json:"edges_created"`
	ErrorCount     int `json:"error_count"`
}

// NewParseStatusFromHistory creates a ParseStatusResponse from ParseHistory
func NewParseStatusFromHistory(history *ParseHistory) *ParseStatusResponse {
	if history == nil {
		return &ParseStatusResponse{
			Status: string(ParseStatusIdle),
		}
	}

	status := &ParseStatusResponse{
		Status:      string(history.Status),
		StartedAt:   &history.StartedAt,
		CompletedAt: history.CompletedAt,
	}

	// Convert error pointer to string
	if history.Error != nil {
		status.Error = *history.Error
	}

	// Convert stats to progress if available
	// Progress should be available for all active parsing operations
	if history.Status == ParseStatusRunning {
		stats := history.Stats.ToParseStats()
		status.Progress = &ParseProgress{
			TotalFiles:     stats.TotalFiles,
			ProcessedFiles: stats.ParsedFiles,
			NodesCreated:   stats.TotalNodes,
			EdgesCreated:   stats.TotalEdges,
			ErrorCount:     0, // Could be calculated from unresolved links
		}
	}

	return status
}
