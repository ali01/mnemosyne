# Current Workstream: Phase 3.5 - VaultService Implementation

## Overview

This document outlines the comprehensive implementation plan for completing Phase 3.5 of the Mnemosyne project. The primary goal is to create the VaultService that connects the existing, tested components (Git integration, vault parser, graph builder) with the database layer through the service/repository pattern.

## Current State Analysis

### What's Working
- **Git Integration**: Clones and syncs vaults from GitHub
- **Vault Parser**: Extracts nodes, edges, and metadata (94% test coverage)
- **Graph Builder**: Constructs graph from parsed data
- **Repository Layer**: Full CRUD operations for nodes, edges, positions, metadata
- **Service Layer**: Individual services for nodes, edges, positions, metadata
- **API Layer**: Endpoints exist but return 501 for parse operations

### The Gap
No orchestration layer connects these components. The VaultService will be this orchestration layer.

## Architecture Decisions

### 1. Service Layer Pattern
Follow the existing pattern where services:
- Accept context for cancellation/timeout
- Use repository interfaces for data access
- Return domain models (not database models)
- Handle business logic and validation
- Manage transactions when needed

### 2. Asynchronous vs Synchronous Parsing
**Decision**: Start with synchronous parsing, add async later
- Rationale: Simpler implementation, easier testing
- Future: Add job queue for background processing

### 3. Transaction Strategy
**Decision**: Single transaction for entire parse operation
- Ensures atomicity: all-or-nothing updates
- Rollback on any failure
- Clear vault data before inserting new graph

### 4. Progress Tracking
**Decision**: Update ParseHistory record with progress
- Store in database for persistence
- Enable status queries during parsing
- Track metrics: files processed, nodes created, etc.

### 5. Error Handling
**Decision**: Fail-fast with detailed error reporting
- Stop parsing on critical errors
- Continue on recoverable errors (single file issues)
- Log unresolved links but don't persist them

## Detailed Implementation Plan

### Step 1: Define Core Data Structures

#### ParseStatus Model
```go
// backend/internal/models/parse_status.go
type ParseStatus struct {
    Status      string    `json:"status"`       // "idle", "parsing", "completed", "failed"
    StartedAt   *time.Time `json:"started_at,omitempty"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
    Progress    *ParseProgress `json:"progress,omitempty"`
    Error       string    `json:"error,omitempty"`
}

type ParseProgress struct {
    TotalFiles     int `json:"total_files"`
    ProcessedFiles int `json:"processed_files"`
    NodesCreated   int `json:"nodes_created"`
    EdgesCreated   int `json:"edges_created"`
    ErrorCount     int `json:"error_count"`
}
```

### Step 2: Implement VaultService

#### Interface Definition
```go
// backend/internal/service/interfaces.go
type VaultServiceInterface interface {
    // Core parsing operations
    ParseAndIndexVault(ctx context.Context) (*models.ParseHistory, error)
    GetParseStatus(ctx context.Context) (*models.ParseStatus, error)
    GetLatestParseHistory(ctx context.Context) (*models.ParseHistory, error)

    // Vault information
    GetVaultMetadata(ctx context.Context) (*models.VaultMetadata, error)
}
```

#### Implementation Structure
```go
// backend/internal/service/vault_service.go
type VaultService struct {
    config           *config.Config
    gitManager       *git.Manager
    parser           *vault.Parser
    graphBuilder     *vault.GraphBuilder
    nodeService      NodeServiceInterface
    edgeService      EdgeServiceInterface
    metadataService  MetadataServiceInterface
    db               repository.Database

    // State management
    mu              sync.Mutex
    currentParseID  string
    isParsing       bool
}
```

#### Core Method Implementation Flow
```go
func (s *VaultService) ParseAndIndexVault(ctx context.Context) (*models.ParseHistory, error) {
    // 1. Check if parse is already running
    // 2. Create new ParseHistory record
    // 3. Clone/update vault from Git
    // 4. Parse vault files
    // 5. Build graph structure
    // 6. Start database transaction
    // 7. Clear existing data
    // 8. Store nodes in batches
    // 9. Store edges in batches
    // 10. Update vault metadata
    // 11. Commit transaction
    // 12. Update ParseHistory as completed
    // 13. Return ParseHistory
}
```

### Step 3: API Handler Implementation

#### Update Parse Endpoint
```go
// backend/internal/api/service_handlers.go
func (h *ServiceHandler) parseVault(c *gin.Context) {
    ctx := c.Request.Context()

    // Check if parse is already running
    status, err := h.vaultService.GetParseStatus(ctx)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to get parse status"})
        return
    }

    if status.Status == "parsing" {
        c.JSON(409, gin.H{"error": "Parse already in progress"})
        return
    }

    // Start parsing
    history, err := h.vaultService.ParseAndIndexVault(ctx)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, history)
}
```

### Step 4: Integration Points

#### Server Initialization
```go
// backend/cmd/server/main.go
// Add to server initialization:
vaultService := service.NewVaultService(
    cfg,
    gitManager,
    parser,
    graphBuilder,
    nodeService,
    edgeService,
    metadataService,
    db,
)

serviceHandler := api.NewServiceHandler(
    nodeService,
    edgeService,
    positionService,
    metadataService,
    vaultService, // Add this
)
```

### Step 5: Testing Strategy

#### Unit Tests
1. **VaultService Tests**
   - Mock all dependencies
   - Test successful parse flow
   - Test error scenarios
   - Test concurrent parse prevention
   - Test transaction rollback

#### Integration Tests
1. **End-to-End Parse Test**
   - Use test vault with known structure
   - Verify all nodes and edges created
   - Verify parse statistics are accurate
   - Confirm API returns correct data

2. **Performance Tests**
   - Test with 1k, 10k, 50k node vaults
   - Measure parse time and memory usage
   - Verify batch operations work efficiently

### Step 6: Error Handling & Recovery

#### Error Categories
1. **Fatal Errors** (stop parsing)
   - Git clone/pull failures
   - Database connection lost
   - Out of memory
   - Transaction failures

2. **Recoverable Errors** (continue parsing)
   - Single file parse errors
   - Invalid frontmatter
   - Malformed links

#### Recovery Mechanisms
- Automatic retry for transient errors
- Rollback on fatal errors
- Detailed error logging
- Parse history tracks all errors

### Step 7: Performance Optimizations

#### Batch Processing
- Process nodes in chunks of 1000
- Use COPY for bulk inserts
- Implement progress tracking per batch

#### Memory Management
- Stream large files instead of loading
- Clear parsed data after batch insert
- Use sync.Pool for reusable objects

#### Concurrent Processing
- Parse files with worker pool
- Configurable worker count
- Coordinate through channels

### Step 8: Monitoring & Observability

#### Metrics to Track
- Parse duration
- Nodes/edges created per second
- Memory usage during parse
- Error rates by type
- Files with missing IDs or unresolved links (logged only)

#### Logging Strategy
- Structured logging with context
- Log levels: DEBUG, INFO, WARN, ERROR
- Include parse ID in all logs
- Performance metrics in logs

## Implementation Timeline

### Phase 1: Core Infrastructure (1-2 days)
1. Create ParseStatus model
2. Create VaultService skeleton with interface
3. Set up basic structure

### Phase 2: Service Implementation (3-4 days)
1. Implement ParseAndIndexVault method
2. Add progress tracking
3. Implement status and metadata methods
4. Add transaction management

### Phase 3: API Integration (1-2 days)
1. Update API handlers
2. Wire services in main.go
3. Add request validation
4. Test API endpoints

### Phase 4: Testing (2-3 days)
1. Write comprehensive unit tests
2. Create integration tests
3. Add performance benchmarks
4. Test error scenarios

### Phase 5: Optimization (1-2 days)
1. Implement batch processing
2. Add concurrent parsing
3. Optimize database queries
4. Profile and tune performance

## Success Criteria

1. **Functional Requirements**
   - Git vault successfully cloned/updated
   - All markdown files parsed
   - Graph structure built and stored
   - API returns complete graph data
   - Node positions persist

2. **Performance Requirements**
   - Parse 1k nodes in <5 seconds
   - Parse 10k nodes in <30 seconds
   - Parse 50k nodes in <5 minutes
   - Memory usage <1GB for 50k nodes

3. **Reliability Requirements**
   - Graceful error handling
   - Transaction atomicity
   - No data corruption
   - Clear error messages

4. **Observability Requirements**
   - Parse progress visible
   - Errors logged with context
   - Performance metrics available
   - System health checkable

## Risk Mitigation

### Technical Risks

1. **Large Vault Performance**
   - Risk: OOM or timeout with 50k+ nodes
   - Mitigation: Batch processing, streaming, monitoring

2. **Concurrent Parse Requests**
   - Risk: Data corruption or conflicts
   - Mitigation: Mutex lock, status checks

3. **Transaction Size Limits**
   - Risk: PostgreSQL transaction too large
   - Mitigation: Chunked operations, tuning

4. **Git Operation Failures**
   - Risk: Network issues, auth problems
   - Mitigation: Retries, clear error messages

### Operational Risks

1. **Data Loss**
   - Risk: Parsing overwrites user data
   - Mitigation: Never delete positions table

2. **Parse Interruption**
   - Risk: Partial data in database
   - Mitigation: Transaction rollback

3. **Resource Exhaustion**
   - Risk: Parse consumes all CPU/memory
   - Mitigation: Resource limits, monitoring

## Next Steps After Phase 3.5

Once VaultService is complete and tested:

1. **Phase 4 Prerequisites**
   - Performance baseline established
   - Bottlenecks identified
   - Monitoring in place

2. **Phase 5 Prerequisites**
   - Incremental parsing design
   - WebSocket infrastructure
   - Change detection strategy

3. **Production Prerequisites**
   - Load testing completed
   - Security review done
   - Operations runbook created

This completes the comprehensive plan for implementing Phase 3.5. The VaultService will be the critical integration layer that enables the full parsing pipeline from Git repositories to the visualization API.
