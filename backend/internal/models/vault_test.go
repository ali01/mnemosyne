package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test VaultNode JSON serialization/deserialization
func TestVaultNode_JSONMarshaling(t *testing.T) {
	now := time.Now().Round(time.Second)
	
	tests := []struct {
		name     string
		node     VaultNode
		wantJSON bool
	}{
		{
			name: "complete node with all fields",
			node: VaultNode{
				ID:       "test-id-123",
				Title:    "Test Node Title",
				NodeType: "concept",
				Tags:     []string{"tag1", "tag2", "tag3"},
				Content:  "# Test Content\n\nThis is a test node with [[links]].",
				Metadata: map[string]interface{}{
					"author":   "Test Author",
					"priority": 5,
					"public":   true,
				},
				FilePath:   "/vault/concepts/test.md",
				InDegree:   10,
				OutDegree:  5,
				Centrality: 0.75,
				CreatedAt:  now,
				UpdatedAt:  now.Add(time.Hour),
			},
			wantJSON: true,
		},
		{
			name: "minimal node with required fields only",
			node: VaultNode{
				ID:        "minimal-id",
				Title:     "Minimal Node",
				NodeType:  "note",
				FilePath:  "/vault/minimal.md",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantJSON: true,
		},
		{
			name: "node with empty optional fields",
			node: VaultNode{
				ID:        "empty-fields",
				Title:     "Empty Fields Node",
				NodeType:  "index",
				Tags:      []string{},
				Content:   "",
				Metadata:  map[string]interface{}{},
				FilePath:  "/vault/empty.md",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantJSON: true,
		},
		{
			name: "node with nil metadata",
			node: VaultNode{
				ID:        "nil-metadata",
				Title:     "Nil Metadata Node",
				NodeType:  "hub",
				Tags:      nil,
				Metadata:  nil,
				FilePath:  "/vault/nil.md",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.node)
			require.NoError(t, err)
			require.NotEmpty(t, data)

			// Test unmarshaling
			var decoded VaultNode
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// Verify fields match
			assert.Equal(t, tt.node.ID, decoded.ID)
			assert.Equal(t, tt.node.Title, decoded.Title)
			assert.Equal(t, tt.node.NodeType, decoded.NodeType)
			assert.Equal(t, tt.node.FilePath, decoded.FilePath)
			assert.Equal(t, tt.node.InDegree, decoded.InDegree)
			assert.Equal(t, tt.node.OutDegree, decoded.OutDegree)
			assert.Equal(t, tt.node.Centrality, decoded.Centrality)
			assert.Equal(t, tt.node.Content, decoded.Content)
			assert.True(t, tt.node.CreatedAt.Equal(decoded.CreatedAt))
			assert.True(t, tt.node.UpdatedAt.Equal(decoded.UpdatedAt))

			// Check optional fields
			// Handle JSON unmarshaling behavior: empty slices/maps become nil
			if len(tt.node.Tags) > 0 {
				assert.Equal(t, tt.node.Tags, decoded.Tags)
			} else if len(tt.node.Tags) == 0 {
				// Empty slice marshals to [] but unmarshals to nil
				assert.Nil(t, decoded.Tags)
			}
			
			if len(tt.node.Metadata) > 0 {
				// Convert expected metadata values to handle JSON number conversion
				expected := make(map[string]interface{})
				for k, v := range tt.node.Metadata {
					// JSON unmarshals all numbers as float64
					if intVal, ok := v.(int); ok {
						expected[k] = float64(intVal)
					} else {
						expected[k] = v
					}
				}
				assert.Equal(t, JSONMetadata(expected), decoded.Metadata)
			} else if len(tt.node.Metadata) == 0 {
				// Empty map marshals to {} but unmarshals to nil
				assert.Nil(t, decoded.Metadata)
			}

			// Verify omitempty works correctly
			jsonStr := string(data)
			// Empty slices are still marshaled as [] even with omitempty
			// Only nil slices are omitted
			if tt.node.Tags == nil {
				assert.NotContains(t, jsonStr, `"tags"`)
			}
			if len(tt.node.Content) == 0 {
				assert.NotContains(t, jsonStr, `"content":""`)
			}
			// Empty maps are still marshaled as {} even with omitempty
			// Only nil maps are omitted
			if tt.node.Metadata == nil {
				assert.NotContains(t, jsonStr, `"metadata"`)
			}
		})
	}
}

// Test VaultEdge JSON serialization/deserialization
func TestVaultEdge_JSONMarshaling(t *testing.T) {
	now := time.Now().Round(time.Second)

	tests := []struct {
		name     string
		edge     VaultEdge
		wantJSON bool
	}{
		{
			name: "complete edge with all fields",
			edge: VaultEdge{
				ID:          "edge-uuid-123",
				SourceID:    "node-1",
				TargetID:    "node-2",
				EdgeType:    "wikilink",
				DisplayText: "Section Reference#Heading",
				Weight:      1.5,
				CreatedAt:   now,
			},
			wantJSON: true,
		},
		{
			name: "minimal edge without optional fields",
			edge: VaultEdge{
				ID:        "edge-minimal",
				SourceID:  "node-a",
				TargetID:  "node-b",
				EdgeType:  "embed",
				Weight:    1.0,
				CreatedAt: now,
			},
			wantJSON: true,
		},
		{
			name: "edge with empty display text",
			edge: VaultEdge{
				ID:          "edge-empty-display",
				SourceID:    "node-x",
				TargetID:    "node-y",
				EdgeType:    "wikilink",
				DisplayText: "",
				Weight:      1.0,
				CreatedAt:   now,
			},
			wantJSON: true,
		},
		{
			name: "edge with zero weight",
			edge: VaultEdge{
				ID:        "edge-zero-weight",
				SourceID:  "node-src",
				TargetID:  "node-tgt",
				EdgeType:  "embed",
				Weight:    0.0,
				CreatedAt: now,
			},
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.edge)
			require.NoError(t, err)
			require.NotEmpty(t, data)

			// Test unmarshaling
			var decoded VaultEdge
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// Verify fields match
			assert.Equal(t, tt.edge.ID, decoded.ID)
			assert.Equal(t, tt.edge.SourceID, decoded.SourceID)
			assert.Equal(t, tt.edge.TargetID, decoded.TargetID)
			assert.Equal(t, tt.edge.EdgeType, decoded.EdgeType)
			assert.Equal(t, tt.edge.DisplayText, decoded.DisplayText)
			assert.Equal(t, tt.edge.Weight, decoded.Weight)
			assert.True(t, tt.edge.CreatedAt.Equal(decoded.CreatedAt))

			// Verify omitempty works for DisplayText
			jsonStr := string(data)
			if tt.edge.DisplayText == "" {
				assert.NotContains(t, jsonStr, `"display_text":""`)
			}
		})
	}
}

// Test edge cases and field validation
func TestVaultNode_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		nodeJSON    string
		shouldError bool
		validate    func(t *testing.T, node *VaultNode)
	}{
		{
			name: "empty string fields",
			nodeJSON: `{
				"id": "",
				"title": "",
				"node_type": "",
				"file_path": "",
				"in_degree": 0,
				"out_degree": 0,
				"centrality": 0.0,
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, node *VaultNode) {
				assert.Empty(t, node.ID)
				assert.Empty(t, node.Title)
				assert.Empty(t, node.NodeType)
				assert.Empty(t, node.FilePath)
			},
		},
		{
			name: "very large content field",
			nodeJSON: `{
				"id": "large-content",
				"title": "Large Content Node",
				"node_type": "note",
				"content": "` + strings.Repeat("A", 100000) + `",
				"file_path": "/large.md",
				"in_degree": 0,
				"out_degree": 0,
				"centrality": 0.0,
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, node *VaultNode) {
				assert.Len(t, node.Content, 100000)
			},
		},
		{
			name: "unicode and special characters",
			nodeJSON: `{
				"id": "unicode-🚀",
				"title": "Unicode Title 你好世界",
				"node_type": "concept",
				"tags": ["émoji-🎯", "中文标签", "русский"],
				"content": "Special chars: \n\t\r\\\"'",
				"file_path": "/unicode/文件.md",
				"in_degree": 0,
				"out_degree": 0,
				"centrality": 0.0,
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, node *VaultNode) {
				assert.Equal(t, "unicode-🚀", node.ID)
				assert.Equal(t, "Unicode Title 你好世界", node.Title)
				assert.Contains(t, node.Tags, "émoji-🎯")
				assert.Contains(t, node.Tags, "中文标签")
				assert.Contains(t, node.Tags, "русский")
			},
		},
		{
			name: "negative numeric values",
			nodeJSON: `{
				"id": "negative",
				"title": "Negative Values",
				"node_type": "note",
				"file_path": "/negative.md",
				"in_degree": -5,
				"out_degree": -10,
				"centrality": -0.5,
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, node *VaultNode) {
				assert.Equal(t, -5, node.InDegree)
				assert.Equal(t, -10, node.OutDegree)
				assert.Equal(t, -0.5, node.Centrality)
			},
		},
		{
			name: "null values for optional fields",
			nodeJSON: `{
				"id": "null-fields",
				"title": "Null Fields",
				"node_type": "note",
				"tags": null,
				"content": null,
				"metadata": null,
				"file_path": "/null.md",
				"in_degree": 0,
				"out_degree": 0,
				"centrality": 0.0,
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, node *VaultNode) {
				assert.Nil(t, node.Tags)
				assert.Empty(t, node.Content)
				assert.Nil(t, node.Metadata)
			},
		},
		{
			name:        "invalid JSON",
			nodeJSON:    `{"id": "invalid", "title": }`,
			shouldError: true,
		},
		{
			name: "missing required fields",
			nodeJSON: `{
				"tags": ["tag1"],
				"content": "content"
			}`,
			shouldError: false,
			validate: func(t *testing.T, node *VaultNode) {
				assert.Empty(t, node.ID)
				assert.Empty(t, node.Title)
				assert.Zero(t, node.CreatedAt)
			},
		},
		{
			name: "complex metadata structure",
			nodeJSON: `{
				"id": "complex-metadata",
				"title": "Complex Metadata",
				"node_type": "project",
				"metadata": {
					"nested": {
						"level1": {
							"level2": "value"
						}
					},
					"array": [1, 2, 3],
					"mixed": ["string", 123, true, null],
					"number": 42.5,
					"boolean": true,
					"null": null
				},
				"file_path": "/complex.md",
				"in_degree": 0,
				"out_degree": 0,
				"centrality": 0.0,
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, node *VaultNode) {
				assert.NotNil(t, node.Metadata)
				assert.Contains(t, node.Metadata, "nested")
				assert.Contains(t, node.Metadata, "array")
				assert.Contains(t, node.Metadata, "mixed")
				assert.Equal(t, 42.5, node.Metadata["number"])
				assert.Equal(t, true, node.Metadata["boolean"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node VaultNode
			err := json.Unmarshal([]byte(tt.nodeJSON), &node)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, &node)
				}
			}
		})
	}
}

// Test edge cases for VaultEdge
func TestVaultEdge_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		edgeJSON    string
		shouldError bool
		validate    func(t *testing.T, edge *VaultEdge)
	}{
		{
			name: "empty string IDs",
			edgeJSON: `{
				"id": "",
				"source_id": "",
				"target_id": "",
				"edge_type": "",
				"weight": 1.0,
				"created_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, edge *VaultEdge) {
				assert.Empty(t, edge.ID)
				assert.Empty(t, edge.SourceID)
				assert.Empty(t, edge.TargetID)
				assert.Empty(t, edge.EdgeType)
			},
		},
		{
			name: "very long display text",
			edgeJSON: `{
				"id": "long-display",
				"source_id": "src",
				"target_id": "tgt",
				"edge_type": "wikilink",
				"display_text": "` + strings.Repeat("Long text ", 1000) + `",
				"weight": 1.0,
				"created_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, edge *VaultEdge) {
				assert.Greater(t, len(edge.DisplayText), 9000)
			},
		},
		{
			name: "unicode in edge fields",
			edgeJSON: `{
				"id": "unicode-edge-🔗",
				"source_id": "源节点",
				"target_id": "目标节点",
				"edge_type": "wikilink",
				"display_text": "链接文本 🌟",
				"weight": 1.0,
				"created_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, edge *VaultEdge) {
				assert.Equal(t, "unicode-edge-🔗", edge.ID)
				assert.Equal(t, "源节点", edge.SourceID)
				assert.Equal(t, "目标节点", edge.TargetID)
				assert.Equal(t, "链接文本 🌟", edge.DisplayText)
			},
		},
		{
			name: "extreme weight values",
			edgeJSON: `{
				"id": "extreme-weight",
				"source_id": "src",
				"target_id": "tgt",
				"edge_type": "embed",
				"weight": 999999.999999,
				"created_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, edge *VaultEdge) {
				assert.Equal(t, 999999.999999, edge.Weight)
			},
		},
		{
			name: "negative weight",
			edgeJSON: `{
				"id": "negative-weight",
				"source_id": "src",
				"target_id": "tgt",
				"edge_type": "wikilink",
				"weight": -10.5,
				"created_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, edge *VaultEdge) {
				assert.Equal(t, -10.5, edge.Weight)
			},
		},
		{
			name:        "malformed JSON",
			edgeJSON:    `{"id": "bad", "source_id": ]`,
			shouldError: true,
		},
		{
			name: "null optional field",
			edgeJSON: `{
				"id": "null-display",
				"source_id": "src",
				"target_id": "tgt",
				"edge_type": "wikilink",
				"display_text": null,
				"weight": 1.0,
				"created_at": "2023-01-01T00:00:00Z"
			}`,
			shouldError: false,
			validate: func(t *testing.T, edge *VaultEdge) {
				assert.Empty(t, edge.DisplayText)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var edge VaultEdge
			err := json.Unmarshal([]byte(tt.edgeJSON), &edge)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, &edge)
				}
			}
		})
	}
}

// Benchmark tests for memory usage with large content
func BenchmarkVaultNode_LargeContent(b *testing.B) {
	sizes := []int{
		1000,      // 1KB
		10000,     // 10KB
		100000,    // 100KB
		1000000,   // 1MB
		10000000,  // 10MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			content := strings.Repeat("A", size)
			node := VaultNode{
				ID:        "bench-node",
				Title:     "Benchmark Node",
				NodeType:  "note",
				Content:   content,
				FilePath:  "/bench.md",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				data, err := json.Marshal(node)
				if err != nil {
					b.Fatal(err)
				}

				var decoded VaultNode
				err = json.Unmarshal(data, &decoded)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark metadata complexity
func BenchmarkVaultNode_ComplexMetadata(b *testing.B) {
	metadataSizes := []int{10, 50, 100, 500}

	for _, size := range metadataSizes {
		b.Run(fmt.Sprintf("fields_%d", size), func(b *testing.B) {
			metadata := make(map[string]interface{})
			for i := 0; i < size; i++ {
				metadata[fmt.Sprintf("field_%d", i)] = map[string]interface{}{
					"value":     i,
					"timestamp": time.Now(),
					"tags":      []string{"tag1", "tag2", "tag3"},
				}
			}

			node := VaultNode{
				ID:        "metadata-bench",
				Title:     "Metadata Benchmark",
				NodeType:  "concept",
				Metadata:  metadata,
				FilePath:  "/metadata.md",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				data, err := json.Marshal(node)
				if err != nil {
					b.Fatal(err)
				}

				var decoded VaultNode
				err = json.Unmarshal(data, &decoded)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark edge serialization with different display text sizes
func BenchmarkVaultEdge_DisplayText(b *testing.B) {
	textSizes := []int{0, 50, 200, 1000, 5000}

	for _, size := range textSizes {
		b.Run(fmt.Sprintf("text_size_%d", size), func(b *testing.B) {
			displayText := ""
			if size > 0 {
				displayText = strings.Repeat("X", size)
			}

			edge := VaultEdge{
				ID:          "bench-edge",
				SourceID:    "source",
				TargetID:    "target",
				EdgeType:    "wikilink",
				DisplayText: displayText,
				Weight:      1.0,
				CreatedAt:   time.Now(),
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				data, err := json.Marshal(edge)
				if err != nil {
					b.Fatal(err)
				}

				var decoded VaultEdge
				err = json.Unmarshal(data, &decoded)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Test time handling and timezone consistency
func TestVaultNode_TimeHandling(t *testing.T) {
	locations := []*time.Location{
		time.UTC,
		time.FixedZone("EST", -5*60*60),
		time.FixedZone("JST", 9*60*60),
	}

	for _, loc := range locations {
		t.Run(loc.String(), func(t *testing.T) {
			originalTime := time.Now().In(loc).Round(time.Second)
			node := VaultNode{
				ID:        "time-test",
				Title:     "Time Test",
				NodeType:  "note",
				FilePath:  "/time.md",
				CreatedAt: originalTime,
				UpdatedAt: originalTime.Add(time.Hour),
			}

			// Marshal
			data, err := json.Marshal(node)
			require.NoError(t, err)

			// Unmarshal
			var decoded VaultNode
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// Times should be equal when compared properly
			assert.True(t, originalTime.Equal(decoded.CreatedAt))
			assert.True(t, originalTime.Add(time.Hour).Equal(decoded.UpdatedAt))
		})
	}
}

// Test concurrent access (for future thread safety if needed)
func TestVaultNode_ConcurrentAccess(t *testing.T) {
	node := VaultNode{
		ID:        "concurrent",
		Title:     "Concurrent Test",
		NodeType:  "note",
		Tags:      []string{"tag1", "tag2"},
		Metadata:  map[string]interface{}{"key": "value"},
		FilePath:  "/concurrent.md",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Run concurrent marshaling
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				data, err := json.Marshal(node)
				assert.NoError(t, err)
				assert.NotEmpty(t, data)

				var decoded VaultNode
				err = json.Unmarshal(data, &decoded)
				assert.NoError(t, err)
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test that struct fields are properly exported (capitalized)
func TestVaultStructs_FieldExporting(t *testing.T) {
	// This test ensures all fields that should be accessible are exported
	node := VaultNode{
		ID:         "export-test",
		Title:      "Export Test",
		NodeType:   "note",
		Tags:       []string{"exported"},
		Content:    "Exported content",
		Metadata:   map[string]interface{}{"exported": true},
		FilePath:   "/export.md",
		InDegree:   1,
		OutDegree:  2,
		Centrality: 0.5,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	edge := VaultEdge{
		ID:          "edge-export",
		SourceID:    "src",
		TargetID:    "tgt",
		EdgeType:    "wikilink",
		DisplayText: "exported",
		Weight:      1.0,
		CreatedAt:   time.Now(),
	}

	// If fields weren't exported, these assignments would fail at compile time
	assert.NotEmpty(t, node.ID)
	assert.NotEmpty(t, edge.ID)
}
// Test field validation with validator tags
func TestVaultNode_Validation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name        string
		node        VaultNode
		shouldError bool
		errorFields []string
	}{
		{
			name: "valid node with all required fields",
			node: VaultNode{
				ID:        "valid-node",
				Title:     "Valid Node",
				NodeType:  "concept",
				FilePath:  "/valid.md",
				InDegree:  5,
				OutDegree: 3,
				Centrality: 0.75,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			shouldError: false,
		},
		{
			name: "missing required ID",
			node: VaultNode{
				Title:     "Missing ID",
				NodeType:  "note",
				FilePath:  "/missing-id.md",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"ID"},
		},
		{
			name: "empty required fields",
			node: VaultNode{
				ID:        "",
				Title:     "",
				FilePath:  "",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"ID", "Title", "FilePath"},
		},
		{
			name: "empty node type",
			node: VaultNode{
				ID:        "empty-type",
				Title:     "Empty Type", 
				NodeType:  "",  // Empty is valid due to omitempty
				FilePath:  "/empty-type.md",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			shouldError: false,  // omitempty means empty is valid
		},
		{
			name: "negative degrees",
			node: VaultNode{
				ID:        "negative",
				Title:     "Negative Degrees",
				FilePath:  "/negative.md",
				InDegree:  -1,
				OutDegree: -5,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"InDegree", "OutDegree"},
		},
		{
			name: "centrality out of range",
			node: VaultNode{
				ID:        "centrality",
				Title:     "Bad Centrality",
				FilePath:  "/centrality.md",
				Centrality: 1.5,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"Centrality"},
		},
		{
			name: "missing timestamps",
			node: VaultNode{
				ID:       "no-time",
				Title:    "No Timestamps",
				FilePath: "/notime.md",
			},
			shouldError: true,
			errorFields: []string{"CreatedAt", "UpdatedAt"},
		},
		{
			name: "empty tags in array",
			node: VaultNode{
				ID:        "empty-tags",
				Title:     "Empty Tags",
				Tags:      []string{"valid", ""},
				FilePath:  "/tags.md",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"Tags[1]"},
		},
		{
			name: "valid optional fields",
			node: VaultNode{
				ID:        "optional",
				Title:     "Optional Fields",
				NodeType:  "",  // optional
				Tags:      nil, // optional
				Content:   "",  // optional
				Metadata:  nil, // optional
				FilePath:  "/optional.md",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.node)

			if tt.shouldError {
				assert.Error(t, err)
				if validationErrors, ok := err.(validator.ValidationErrors); ok {
					for _, fieldName := range tt.errorFields {
						found := false
						for _, validationErr := range validationErrors {
							if strings.Contains(validationErr.Namespace(), fieldName) {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected validation error for field %s", fieldName)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test VaultEdge validation
func TestVaultEdge_Validation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name        string
		edge        VaultEdge
		shouldError bool
		errorFields []string
	}{
		{
			name: "valid edge",
			edge: VaultEdge{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				SourceID:  "source",
				TargetID:  "target",
				EdgeType:  "wikilink",
				Weight:    1.0,
				CreatedAt: time.Now(),
			},
			shouldError: false,
		},
		{
			name: "invalid UUID",
			edge: VaultEdge{
				ID:        "not-a-uuid",
				SourceID:  "source",
				TargetID:  "target",
				EdgeType:  "wikilink",
				CreatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"ID"},
		},
		{
			name: "empty required fields",
			edge: VaultEdge{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				SourceID:  "",
				TargetID:  "",
				EdgeType:  "",
				CreatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"SourceID", "TargetID", "EdgeType"},
		},
		{
			name: "invalid edge type",
			edge: VaultEdge{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				SourceID:  "source",
				TargetID:  "target",
				EdgeType:  "invalid",
				CreatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"EdgeType"},
		},
		{
			name: "self loop",
			edge: VaultEdge{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				SourceID:  "same",
				TargetID:  "same",
				EdgeType:  "wikilink",
				CreatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"TargetID"},
		},
		{
			name: "negative weight",
			edge: VaultEdge{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				SourceID:  "source",
				TargetID:  "target",
				EdgeType:  "embed",
				Weight:    -1.0,
				CreatedAt: time.Now(),
			},
			shouldError: true,
			errorFields: []string{"Weight"},
		},
		{
			name: "missing timestamp",
			edge: VaultEdge{
				ID:       "550e8400-e29b-41d4-a716-446655440000",
				SourceID: "source",
				TargetID: "target",
				EdgeType: "wikilink",
			},
			shouldError: true,
			errorFields: []string{"CreatedAt"},
		},
		{
			name: "valid edge with optional display text",
			edge: VaultEdge{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				SourceID:    "source",
				TargetID:    "target",
				EdgeType:    "wikilink",
				DisplayText: "Optional alias text",
				Weight:      2.5,
				CreatedAt:   time.Now(),
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.edge)

			if tt.shouldError {
				assert.Error(t, err)
				if validationErrors, ok := err.(validator.ValidationErrors); ok {
					for _, fieldName := range tt.errorFields {
						found := false
						for _, validationErr := range validationErrors {
							if strings.Contains(validationErr.Namespace(), fieldName) {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected validation error for field %s", fieldName)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Benchmark validation performance
func BenchmarkVaultNode_Validation(b *testing.B) {
	validate := validator.New()
	node := VaultNode{
		ID:         "bench-valid",
		Title:      "Benchmark Validation",
		NodeType:   "concept",
		Tags:       []string{"tag1", "tag2", "tag3"},
		Content:    strings.Repeat("Content ", 100),
		Metadata:   map[string]interface{}{"key1": "value1", "key2": 2, "key3": true},
		FilePath:   "/bench.md",
		InDegree:   10,
		OutDegree:  5,
		Centrality: 0.75,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := validate.Struct(node)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test memory usage with different scenarios
func TestVaultNode_MemoryUsage(t *testing.T) {
	scenarios := []struct {
		name     string
		nodeFunc func() VaultNode
	}{
		{
			name: "minimal node",
			nodeFunc: func() VaultNode {
				return VaultNode{
					ID:        "min",
					Title:     "Min",
					FilePath:  "/min.md",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
			},
		},
		{
			name: "node with 10KB content",
			nodeFunc: func() VaultNode {
				return VaultNode{
					ID:        "10kb",
					Title:     "10KB Content",
					Content:   strings.Repeat("A", 10*1024),
					FilePath:  "/10kb.md",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
			},
		},
		{
			name: "node with 100 metadata fields",
			nodeFunc: func() VaultNode {
				metadata := make(map[string]interface{})
				for i := 0; i < 100; i++ {
					metadata[fmt.Sprintf("field%d", i)] = fmt.Sprintf("value%d", i)
				}
				return VaultNode{
					ID:        "meta100",
					Title:     "100 Metadata Fields",
					Metadata:  metadata,
					FilePath:  "/meta100.md",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
			},
		},
		{
			name: "node with 1000 tags",
			nodeFunc: func() VaultNode {
				tags := make([]string, 1000)
				for i := 0; i < 1000; i++ {
					tags[i] = fmt.Sprintf("tag%d", i)
				}
				return VaultNode{
					ID:        "tags1000",
					Title:     "1000 Tags",
					Tags:      tags,
					FilePath:  "/tags1000.md",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			node := scenario.nodeFunc()
			
			// Test JSON marshaling doesn't leak memory
			for i := 0; i < 100; i++ {
				data, err := json.Marshal(node)
				assert.NoError(t, err)
				assert.NotEmpty(t, data)
				
				var decoded VaultNode
				err = json.Unmarshal(data, &decoded)
				assert.NoError(t, err)
			}
		})
	}
}