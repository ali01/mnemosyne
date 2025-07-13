package api

import (
	"testing"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeParseHistory(t *testing.T) {
	tests := []struct {
		name     string
		input    *models.ParseHistory
		expected map[string]interface{}
	}{
		{
			name: "basic parse history without error",
			input: &models.ParseHistory{
				ID:          "test-123",
				StartedAt:   time.Now(),
				CompletedAt: &time.Time{},
				Status:      models.ParseStatusCompleted,
				Stats: models.JSONStats{
					TotalFiles:  100,
					ParsedFiles: 95,
					TotalNodes:  200,
					TotalEdges:  150,
					DurationMS:  5000,
				},
			},
			expected: map[string]interface{}{
				"id":           "test-123",
				"status":       models.ParseStatusCompleted,
				"started_at":   true, // will check existence
				"completed_at": true,
				"stats":        true,
			},
		},
		{
			name: "parse history with git error",
			input: &models.ParseHistory{
				ID:        "test-456",
				StartedAt: time.Now(),
				Status:    models.ParseStatusFailed,
				Error:     stringPtr("failed to clone git repository from /Users/sensitive/path: authentication failed"),
			},
			expected: map[string]interface{}{
				"id":         "test-456",
				"status":     models.ParseStatusFailed,
				"started_at": true,
				"error": map[string]interface{}{
					"type":    "git_sync_error",
					"message": "An error occurred during vault processing. Please check logs for details.",
				},
			},
		},
		{
			name: "parse history with database error",
			input: &models.ParseHistory{
				ID:        "test-789",
				StartedAt: time.Now(),
				Status:    models.ParseStatusFailed,
				Error:     stringPtr("database connection lost at host 192.168.1.100:5432"),
			},
			expected: map[string]interface{}{
				"id":         "test-789",
				"status":     models.ParseStatusFailed,
				"started_at": true,
				"error": map[string]interface{}{
					"type":    "storage_error",
					"message": "An error occurred during vault processing. Please check logs for details.",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeParseHistory(tt.input)

			// Check expected fields
			assert.Equal(t, tt.expected["id"], result["id"])
			assert.Equal(t, tt.expected["status"], result["status"])

			// Check existence of timestamp fields
			if tt.expected["started_at"] != nil {
				assert.NotNil(t, result["started_at"])
			}
			if tt.expected["completed_at"] != nil {
				assert.NotNil(t, result["completed_at"])
			}

			// Check stats presence
			if tt.expected["stats"] != nil {
				stats, ok := result["stats"].(gin.H)
				assert.True(t, ok, "stats should be a gin.H map")
				assert.NotContains(t, stats, "file_paths") // ensure no paths exposed
			}

			// Check error sanitization
			if expectedError, ok := tt.expected["error"].(map[string]interface{}); ok {
				actualError, ok := result["error"].(gin.H)
				assert.True(t, ok, "error should be a gin.H map")
				assert.Equal(t, expectedError["type"], actualError["type"])
				assert.Equal(t, expectedError["message"], actualError["message"])
				// Ensure original error message is not exposed
				if tt.input.Error != nil {
					assert.NotContains(t, actualError["message"], *tt.input.Error)
				}
			}
		})
	}
}

func TestSanitizeParseStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    *models.ParseStatusResponse
		expected map[string]interface{}
	}{
		{
			name: "running status with progress",
			input: &models.ParseStatusResponse{
				Status:    "running",
				StartedAt: &time.Time{},
				Progress: &models.ParseProgress{
					TotalFiles:     1000,
					ProcessedFiles: 500,
					NodesCreated:   400,
					EdgesCreated:   300,
					ErrorCount:     5,
				},
			},
			expected: map[string]interface{}{
				"status":     "running",
				"started_at": true,
				"progress":   true,
			},
		},
		{
			name: "failed status with parse error",
			input: &models.ParseStatusResponse{
				Status:      "failed",
				StartedAt:   &time.Time{},
				CompletedAt: &time.Time{},
				Error:       "parse error: invalid frontmatter in /vault/sensitive/note.md",
			},
			expected: map[string]interface{}{
				"status":       "failed",
				"started_at":   true,
				"completed_at": true,
				"error": map[string]interface{}{
					"type":    "parse_error",
					"message": "An error occurred during vault processing. Please check logs for details.",
				},
			},
		},
		{
			name: "timeout error",
			input: &models.ParseStatusResponse{
				Status: "failed",
				Error:  "operation timeout after 300s",
			},
			expected: map[string]interface{}{
				"status": "failed",
				"error": map[string]interface{}{
					"type":    "timeout_error",
					"message": "An error occurred during vault processing. Please check logs for details.",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeParseStatus(tt.input)

			// Check expected fields
			assert.Equal(t, tt.expected["status"], result["status"])

			// Check timestamps
			if tt.expected["started_at"] != nil {
				assert.NotNil(t, result["started_at"])
			}
			if tt.expected["completed_at"] != nil {
				assert.NotNil(t, result["completed_at"])
			}

			// Check progress
			if tt.expected["progress"] != nil {
				progress, ok := result["progress"].(gin.H)
				assert.True(t, ok, "progress should be a gin.H map")
				assert.Contains(t, progress, "total_files")
				assert.Contains(t, progress, "processed_files")
			}

			// Check error sanitization
			if expectedError, ok := tt.expected["error"].(map[string]interface{}); ok {
				actualError, ok := result["error"].(gin.H)
				assert.True(t, ok, "error should be a gin.H map")
				assert.Equal(t, expectedError["type"], actualError["type"])
				assert.Equal(t, expectedError["message"], actualError["message"])
				// Ensure original error is not exposed
				assert.NotContains(t, actualError["message"], tt.input.Error)
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
