# TODO: Mnemosyne Development Plan

## Current Status
- ✅ Phase 1: Git Integration - COMPLETED
- ✅ Phase 2: Vault Parser - COMPLETED (94%+ test coverage)
- 🚧 Phase 3: Graph Construction - IN PROGRESS
- ⏳ Phase 4: API Integration
- ⏳ Phase 5: Caching & Performance
- ⏳ Phase 6: File Watching & Live Updates

## Phase 3: Graph Construction

### Overview
Transform parsed markdown files into a graph structure suitable for visualization. Connect vault parser output to database storage and API endpoints.

### Step 1: Create VaultNode Model
**File**: `backend/internal/models/vault.go`

**Data Structure**:
```go
type VaultNode struct {
    ID          string                 // Required: from frontmatter
    Title       string                 // From frontmatter or filename fallback
    NodeType    string                 // Calculated: index/hub/concept/project/question/note
    Tags        []string               // From frontmatter tags field
    Content     string                 // Full markdown content
    Metadata    map[string]interface{} // All frontmatter fields
    FilePath    string                 // Original file location
    InDegree    int                    // Number of incoming links
    OutDegree   int                    // Number of outgoing links
    Centrality  float64                // PageRank or similar metric
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Purpose**: Bridge between file-centric parser output and node-centric graph visualization. Each node represents a concept/note in the knowledge graph with all necessary metadata for rendering and analysis.

### Step 2: Create VaultEdge Model
**File**: `backend/internal/models/vault.go`

**Data Structure**:
```go
type VaultEdge struct {
    ID          string    // Auto-generated UUID
    SourceID    string    // Node ID of link source
    TargetID    string    // Node ID of link target
    EdgeType    string    // "wikilink" or "embed"
    DisplayText string    // Link alias or section reference
    Weight      float64   // Default 1.0, for future use
    CreatedAt   time.Time
}
```

**Purpose**: Represent connections between ideas. Supports different link types and preserves context through display text. Enables both directed and bidirectional graph operations. All edges in the VaultEdge table are stored as directed (source→target). Bidirectional relationships are represented by creating two edges (A→B and B→A) when needed. This approach:
- Maintains simplicity in the data model
- Allows efficient queries for both incoming and outgoing links
- Supports future weighted edges where A→B might have different weight than B→A
- The EdgeRepository provides methods like `GetBidirectional()` to query relationships regardless of direction

### Step 3: Implement Node Type Calculator
**File**: `backend/internal/vault/node_classifier.go`

**Classification Rules** (priority order):
1. Explicit tags: `index` → "index", `open-question` → "question"
2. Filename prefix: `~` → "hub"
3. Directory path: `concepts/` → "concept", `projects/` → "project", etc.
4. Default: "note"

**Purpose**: Provide visual hierarchy through node types. Different types get different colors/sizes in visualization. Rules are configurable and extensible.

### Step 4: Create Graph Builder
**File**: `backend/internal/vault/graph_builder.go`

**Data Structures**:
```go
type GraphBuildResult struct {
    Nodes          []VaultNode
    Edges          []VaultEdge
    UnresolvedLinks []UnresolvedLink
    Stats          GraphBuildStats
}

type GraphBuildStats struct {
    NodesCreated    int
    EdgesCreated    int
    FilesSkipped    int           // Missing ID
    DuplicateIDs    []string
    UnresolvedLinks int
    BuildDuration   time.Duration
}
```

**Algorithm**:
1. First pass: Create VaultNode for each MarkdownFile with valid ID
2. Build ID→Node lookup map
3. Second pass: Create VaultEdges from resolved WikiLinks
4. Handle edge cases: deduplication, unresolved links, missing IDs

**Purpose**: Transform ParseResult into graph structure. Two-pass approach ensures all nodes exist before creating edges. Provides detailed statistics for monitoring.

### Step 5: Implement Graph Metrics Calculator
**File**: `backend/internal/vault/metrics.go`

**Data Structures**:
```go
type NodeMetrics struct {
    InDegree           int
    OutDegree          int
    PageRank           float64
    ClusteringCoeff    float64
    ConnectedComponent int
}

type GraphMetrics struct {
    TotalNodes         int
    TotalEdges         int
    AvgDegree          float64
    NumComponents      int
    Density            float64
    AvgClusteringCoeff float64
}

type GraphStatistics struct {
    Metrics        GraphMetrics
    NodeTypeCount  map[string]int    // Count by type
    OrphanedNodes  []string          // Nodes with no connections
    TopNodes       []string          // By PageRank
    LastUpdated    time.Time
}
```

**Algorithms**:
- **Degree**: Count edges per node
- **PageRank**: Power iteration with damping factor 0.85
- **Components**: DFS to find disconnected subgraphs
- **Clustering**: Ratio of edges between node's neighbors

**Purpose**: Provide insights for visualization (node sizing) and analysis (finding key concepts). Metrics help identify important nodes and graph structure.

### Step 6: Create Database Repositories
**Files**: `backend/internal/db/node_repository.go`, `edge_repository.go`

**NodeRepository Methods**:
- `CreateBatch(nodes []VaultNode) error` - Bulk insert with COPY
- `UpsertBatch(nodes []VaultNode) error` - Insert or update
- `GetByIDs(ids []string) ([]VaultNode, error)`
- `Search(query string) ([]VaultNode, error)` - Full-text search
- `GetByType(nodeType string) ([]VaultNode, error)`

**EdgeRepository Methods**:
- `CreateBatch(edges []VaultEdge) error`
- `GetByNode(nodeID string) ([]VaultEdge, error)`
- `GetSubgraph(nodeIDs []string) ([]VaultEdge, error)`
- `GetBidirectional(nodeID1, nodeID2 string) ([]VaultEdge, error)`

**Purpose**: Efficient persistence with bulk operations for thousands of nodes/edges. Repository pattern isolates database logic from business logic.

### Step 7: Integration Service
**File**: `backend/internal/vault/service.go`

**Core Methods**:
```go
type VaultService interface {
    ParseAndIndex() (*IndexResult, error)
    GetStatus() (*IndexStatus, error)
    RefreshMetrics() error
    GetNode(id string) (*VaultNode, error)
    SearchNodes(query string) ([]VaultNode, error)
}

type IndexStatus struct {
    Status      string    // "idle", "parsing", "indexing", "complete"
    Progress    float64   // 0.0 to 1.0
    CurrentStep string
    Error       error
    StartedAt   time.Time
    UpdatedAt   time.Time
}
```

**Workflow**:
1. Parse vault files
2. Build graph from parse result
3. Calculate metrics
4. Store in database (transaction)
5. Track progress and handle errors

**Purpose**: Single entry point orchestrating the complete pipeline. Provides async processing, status tracking, and error recovery. Implements circuit breaker for resilience.

### Step 8: Comprehensive Testing

**Test Coverage**:
- **Unit Tests**: Each component in isolation
  - Model validation and serialization
  - Classification rules and edge cases
  - Metric calculations with known graphs
- **Integration Tests**: Complete pipeline
  - Various vault structures (circular refs, orphans)
  - Error injection and recovery
  - Concurrent processing safety
- **Performance Tests**: Scalability validation
  - Synthetic vaults: 100, 1K, 10K, 50K nodes
  - Memory profiling and leak detection
  - Query performance with indices

**Purpose**: Maintain 90%+ coverage standard. Ensure correctness, reliability, and performance at scale.

### Step 9: Implement Layout Algorithm
**File**: `backend/internal/layout/force_directed.go`

**Data Structures**:
```go
type Position struct {
    NodeID string
    X      float64
    Y      float64
    Z      float64  // For future 3D layouts
    Locked bool     // User-pinned nodes
}

type LayoutConfig struct {
    Algorithm   string  // "force-directed", "hierarchical"
    Iterations  int     // Default: 500
    Temperature float64 // Initial temperature for annealing
    Gravity     float64 // Pull towards center
}
```

**Algorithm**: Fruchterman-Reingold force-directed layout
- Repulsive forces between all nodes
- Attractive forces along edges
- Initial positioning based on node types (index at center, hubs in inner ring)
- Store final positions in node_positions table

**Purpose**: Generate aesthetically pleasing graph layouts that reveal structure. Node types influence initial positions for better organization.

## Phase 4: API Integration

### Overview
Replace sample data with real vault data. Add management endpoints for vault operations.

### Updates Required:
1. **Modify** `graph_handlers.go` - Serve from database instead of JSON
2. **Add endpoints**:
   - `GET /api/v1/vault/status` - Parsing status
   - `POST /api/v1/vault/parse` - Trigger parsing
   - `GET /api/v1/nodes/:id/content` - Markdown with rendered links
   - `GET /api/v1/search?q=term` - Full-text search
3. **Implement Markdown renderer** (`backend/internal/render/markdown.go`):
   - Convert WikiLinks to clickable HTML links
   - Support for sections and aliases
   - Syntax highlighting for code blocks


## Success Metrics
- Graph construction completes in <30s for 50K nodes
- API response times <100ms for graph queries
- Test coverage remains >90%
- Zero data loss during re-indexing

## Future Optimizations

### Content Field Lazy Loading
**Problem**: With 50K nodes, loading full markdown content for all nodes uses significant memory (~100MB for 2KB average content) and slows API responses.

**Proposed Solution**: Implement lazy loading for the `Content` field in VaultNode to improve performance and reduce memory usage.

**Options**:
1. **Separate Content Table**: Store content in `node_content` table, load on demand
2. **Field Selection API**: Support `?fields=id,title,node_type` to exclude content unless requested
3. **Nullable Content**: Make Content a pointer with `ContentLoaded` flag

**Benefits**:
- Reduce initial graph load time by 80-90%
- Lower memory footprint for large vaults
- Faster API responses for graph visualization
- Better scalability to 50K+ nodes

**Implementation**: Defer until after Phase 4 when we have real-world performance metrics.

### Caching
- Consider caching classification results for large vaults

## Code Quality Improvements
### Minor Issues
  2. Test Independence: Some tests might benefit from setup/teardown helpers
  3. Benchmarks: Could add more realistic benchmarks with actual file operations

## Scratch

- add validation and database constraints to node.go like the ones in vault.go
- tool to strip trailing whitespace
