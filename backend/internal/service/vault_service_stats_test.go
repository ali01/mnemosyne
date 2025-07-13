package service

import (
	"testing"
	"time"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/vault"
	"github.com/stretchr/testify/assert"
)

func TestVaultService_UpdateParseStats(t *testing.T) {
	svc := &VaultService{}

	t.Run("nil parse history", func(t *testing.T) {
		// Should not panic with nil parse history
		parseResult := &vault.ParseResult{
			Files: map[string]*vault.MarkdownFile{
				"test-id": {Path: "test.md"},
			},
		}
		graph := &vault.Graph{
			Nodes: []models.VaultNode{{ID: "1"}},
			Stats: vault.GraphStats{UnresolvedLinks: 5},
		}

		// This should not panic
		svc.updateParseStats(nil, parseResult, graph)
	})

	t.Run("nil parse result", func(t *testing.T) {
		// Should not panic with nil parse result
		parseHistory := &models.ParseHistory{
			ID:        "test-123",
			StartedAt: time.Now(),
		}
		graph := &vault.Graph{
			Nodes: []models.VaultNode{{ID: "1"}},
		}

		// This should not panic
		svc.updateParseStats(parseHistory, nil, graph)

		// Stats should not be set
		assert.Equal(t, models.JSONStats{}, parseHistory.Stats)
	})

	t.Run("nil graph", func(t *testing.T) {
		// Should not panic with nil graph
		parseHistory := &models.ParseHistory{
			ID:        "test-123",
			StartedAt: time.Now(),
		}
		parseResult := &vault.ParseResult{
			Files: map[string]*vault.MarkdownFile{
				"test-id": {Path: "test.md"},
			},
		}

		// This should not panic
		svc.updateParseStats(parseHistory, parseResult, nil)

		// Stats should not be set
		assert.Equal(t, models.JSONStats{}, parseHistory.Stats)
	})

	t.Run("successful stats update with unresolved links", func(t *testing.T) {
		startTime := time.Now().Add(-5 * time.Second)
		parseHistory := &models.ParseHistory{
			ID:        "test-123",
			StartedAt: startTime,
		}

		parseResult := &vault.ParseResult{
			Files: map[string]*vault.MarkdownFile{
				"id1": {Path: "file1.md"},
				"id2": {Path: "file2.md"},
				"id3": {Path: "file3.md"},
			},
		}

		graph := &vault.Graph{
			Nodes: []models.VaultNode{
				{ID: "1"},
				{ID: "2"},
			},
			Edges: []models.VaultEdge{
				{ID: "e1", SourceID: "1", TargetID: "2"},
			},
			Stats: vault.GraphStats{
				NodesCreated:    2,
				EdgesCreated:    1,
				UnresolvedLinks: 7, // This should be tracked
			},
		}

		// Update stats
		svc.updateParseStats(parseHistory, parseResult, graph)

		// Verify stats were set correctly
		stats := parseHistory.Stats.ToParseStats()
		assert.Equal(t, 3, stats.TotalFiles)
		assert.Equal(t, 3, stats.ParsedFiles)
		assert.Equal(t, 2, stats.TotalNodes)
		assert.Equal(t, 1, stats.TotalEdges)
		assert.Equal(t, 7, stats.UnresolvedLinks, "UnresolvedLinks should be tracked from graph.Stats")
		assert.True(t, stats.DurationMS >= 5000, "Duration should be at least 5000ms")
	})

	t.Run("empty graph still tracks stats", func(t *testing.T) {
		parseHistory := &models.ParseHistory{
			ID:        "test-456",
			StartedAt: time.Now(),
		}

		parseResult := &vault.ParseResult{
			Files: map[string]*vault.MarkdownFile{},
		}

		graph := &vault.Graph{
			Nodes: []models.VaultNode{},
			Edges: []models.VaultEdge{},
			Stats: vault.GraphStats{
				UnresolvedLinks: 0,
			},
		}

		// Update stats
		svc.updateParseStats(parseHistory, parseResult, graph)

		// Verify empty stats
		stats := parseHistory.Stats.ToParseStats()
		assert.Equal(t, 0, stats.TotalFiles)
		assert.Equal(t, 0, stats.ParsedFiles)
		assert.Equal(t, 0, stats.TotalNodes)
		assert.Equal(t, 0, stats.TotalEdges)
		assert.Equal(t, 0, stats.UnresolvedLinks)
	})
}
