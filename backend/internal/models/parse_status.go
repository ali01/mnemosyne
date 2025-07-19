// Package models defines data structures for the mnemosyne graph visualizer.
// This file contains models for tracking vault parsing operations and providing
// real-time progress updates to the frontend during large-scale parsing operations.
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
	TotalFiles     int `json:"total_files,omitempty"`
	ProcessedFiles int `json:"parsed_files,omitempty"`  // Match ParseStats naming
	NodesCreated   int `json:"nodes_created,omitempty"`
	EdgesCreated   int `json:"edges_created,omitempty"`
	ErrorCount     int `json:"error_count,omitempty"`
}

// mapParseStatus safely converts internal ParseStatus constants to API response values
func mapParseStatus(status ParseStatus) string {
	// Map internal status to API response status with validation
	switch status {
	case ParseStatusIdle:
		return "idle"
	case ParseStatusPending:
		return "pending"
	case ParseStatusRunning:
		return "running"
	case ParseStatusCompleted:
		return "completed"
	case ParseStatusFailed:
		return "failed"
	default:
		// Fallback to idle for unknown status to ensure type safety
		return "idle"
	}
}

// NewParseStatusFromHistory creates a ParseStatusResponse from ParseHistory
func NewParseStatusFromHistory(history *ParseHistory) *ParseStatusResponse {
	if history == nil {
		return &ParseStatusResponse{
			Status: mapParseStatus(ParseStatusIdle),
		}
	}

	status := &ParseStatusResponse{
		Status:      mapParseStatus(history.Status),
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
