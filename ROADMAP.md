# Mnemosyne Development Roadmap

## Current Status
- ✅ Phase 1: Git Integration - COMPLETED
- ✅ Phase 2: Vault Parser - COMPLETED (94%+ test coverage)
- ✅ Phase 3: Graph Construction - COMPLETED (core components built)
- 🚧 Phase 3.5: Integration Layer - PARTIALLY COMPLETE (repository layer done, VaultService missing)
- ⏳ Phase 4: Caching & Performance
- ⏳ Phase 5: File Watching & Live Updates
- ⏳ Phase 6: Production Readiness


## Current Architecture & Data Flow

### Current Flow (Disconnected)
```
GitHub Vault → Git Clone ✓ → Parser ✓ → Graph Builder ✓ → ❌ (stops here)

Database ✓ → API ✓ → Frontend ✓
  ↑
Manual data entry only
```

### Target Flow
```
GitHub Vault → Git Clone → Parser → Graph Builder → VaultService → Database → API → Frontend
                    ↑                                                    ↓
                    └──────── Background Sync ←────────────────────────┘
```

**Components Status**:
- ✓ = Implemented and tested
- ❌ = Missing implementation
- **Key Gap**: No connection between parsing components and database
- **Working**: Individual components work in isolation; API serves data from database via manual entry


## Completed Phases Summary

### Phase 1: Git Integration ✅
- Git clone and sync from GitHub repositories
- SSH and HTTPS support with authentication
- Shallow cloning for performance
- Test utility: `cmd/sync-vault/main.go`

### Phase 2: Vault Parser ✅
- Extract nodes, links, and frontmatter from markdown files
- Support for WikiLinks: `[[Note]]`, `[[Note|Alias]]`, `[[Note#Section]]`
- Concurrent file processing with worker pools
- 94%+ test coverage with comprehensive edge cases

### Phase 3: Graph Construction ✅
- VaultNode & VaultEdge models for graph representation
- Configurable node classification system
- Two-pass graph building algorithm
- Handles duplicates, missing IDs, and unresolved links

## Phase 3.5: Integration Layer (PARTIALLY COMPLETE)

### Overview
Connect the completed components (parser, graph builder, node classifier) to the database and API. This is the critical missing piece that bridges the gap between the functional components and the working system.

### Step 1: Create Repository Layer ✅ COMPLETED
**Directory**: `backend/internal/repository/`

**Implemented Repositories**:
- ✅ `NodeRepository` - Full CRUD operations with search and pagination
- ✅ `EdgeRepository` - Graph queries with UNION optimization for performance
- ✅ `PositionRepository` - Node position persistence for visualization
- ✅ `MetadataRepository` - Vault metadata and parse history tracking
- ✅ Mock implementations for all repositories
- ✅ Transaction support with snapshot-based rollback
- ✅ Comprehensive unit tests with 95%+ coverage

**Performance Optimizations**:
- Generated search vectors for full-text search
- UNION queries for efficient edge retrieval
- Batch operations with PostgreSQL COPY
- Optimized indexes for pagination patterns

### Step 2: Create Service Layer ⚠️ PARTIALLY COMPLETE
**Directory**: `backend/internal/service/`

**Completed Individual Services**:
- ✅ `NodeService` - Search, CRUD operations with repository pattern
- ✅ `EdgeService` - Graph queries with optimized edge retrieval
- ✅ `PositionService` - Node position persistence for visualization
- ✅ `MetadataService` - Parse history and vault metadata management

**Missing VaultService** (required for parsing orchestration):
```go
type VaultService interface {
    // Core operations - NOT IMPLEMENTED
    ParseAndIndexVault(ctx context.Context) (*models.ParseResult, error)
    GetParseStatus(ctx context.Context) (*models.ParseStatus, error)
    // Graph operations would use existing services
}
```

### Step 3: Update Main Server ⚠️ PARTIALLY COMPLETE
**File**: `backend/cmd/server/main.go`

✅ **Completed**:
- Database connection initialization
- Configuration loading with timeout settings
- Individual service instances with dependency injection
- Panic recovery for production robustness

❌ **Missing**: VaultService creation and parsing orchestration

### Step 4: Update API Handlers ⚠️ PARTIALLY COMPLETE
**File**: `backend/internal/api/service_handlers.go`

✅ **Completed API Endpoints**:
- `GET /api/v1/graph` - Full graph data with pagination and validation
- `GET /api/v1/nodes/:id` - Individual node retrieval with timeout handling
- `GET /api/v1/nodes/:id/content` - Node content and metadata
- `GET /api/v1/search` - Full-text search with optimized queries
- `PUT /api/v1/nodes/:id/position` - Node position updates
- Request timeout handling with configurable timeouts
- Input validation for pagination parameters
- Consistent error handling with timeout detection

⚠️ **Partially Implemented API Endpoints** (endpoints exist but return 501 Not Implemented):
- `POST /api/v1/vault/parse` - Route exists but requires VaultService
- `GET /api/v1/vault/status` - Route exists but requires VaultService

❌ **Future Enhancements**:
- Markdown Renderer: WikiLink to HTML conversion with proper linking

### Current Success Criteria Status
- ✅ API serves data from database via service layer
- ✅ Node positions persist across server restarts
- ✅ Comprehensive error handling and recovery
- ❌ Full pipeline does NOT work: Git clone → Parse → Build → Store → API
- ❌ Automated vault parsing not implemented


## Phase 4: Caching & Performance (Post-MVP)

### Overview
Continue optimizing system performance for large vaults (50K+ nodes). Core database and pagination optimizations are complete; remaining work focuses on caching, lazy loading, and advanced graph algorithms.

### Already Implemented Performance Features
- ✅ **Pagination**: Complete with validation and repository support
- ✅ **Database Optimization**: Comprehensive indexes, materialized views, performance tuning
- ✅ **Batch Operations**: PostgreSQL COPY optimization for bulk operations
- ✅ **Query Performance**: Advanced indexing strategies and performance monitoring

### Components Still Needed
- Redis integration for graph caching
- Lazy loading for node content
- ⚠️ Background metrics calculation (infrastructure exists, algorithms missing)
- Advanced graph layout algorithms. Possibilities:
  - Force-directed layout (Fruchterman-Reingold)
  - Hierarchical layout based on node types
  - Community detection and clustering
  - Physics-based simulation with configurable forces
  - Persistent layout state with incremental updates

### Repository Layer Performance Enhancements
- **Streaming support for large result sets**: `StreamAll()` method to process nodes one-by-one without loading all into memory
- **Bulk existence checks**: `ExistsByIDs()` to avoid N+1 queries when validating node references
- **Filtered counting methods**: `CountByType()` and `CountByTags()` for efficient metrics without loading full data
- **Batch size limits**: Chunking for batch operations to prevent overwhelming PostgreSQL
  - Based on feedback from PR #12 (comment r2193148495), implement chunking in `CreateBatch` and `UpsertBatch` methods
  - Process nodes in configurable chunks (e.g., 1000 nodes at a time) to avoid memory issues
  - Prevent long-running transactions that could cause timeouts with 50k+ nodes
  - Add progress tracking for large batch operations
- **Read replica support**: Interface updates to support read/write splitting for scale

### Graph-Specific Query Operations
Based on feedback from PR #12 (comment r2193145588), add specialized graph traversal methods to EdgeRepository:
- **Get nodes within N hops**: `GetNodesWithinDistance(ctx, nodeID, distance)` for proximity analysis
- **Find shortest paths**: `GetShortestPath(ctx, sourceID, targetID)` for connection analysis
- **Connected components**: `GetConnectedComponents(ctx)` for identifying isolated subgraphs
- **Batch edge existence**: `ExistsBatch(ctx, edges)` for efficient validation of multiple edges
These operations would enable richer graph visualizations and analysis capabilities

## Phase 5: File Watching & Live Updates (Post-MVP)

### Overview
Enable real-time synchronization between vault changes and graph visualization.

### Components
- File system watcher for local changes
- Git webhook receiver for remote updates
- WebSocket server for pushing updates
- Incremental parsing for changed files
- Conflict resolution for concurrent edits

## Phase 6: Production Readiness

### Overview
Prepare the system for production deployment with proper operations, monitoring, and security.

### Step 1: Configuration Management
**Environment-based Configuration**:
- Development, staging, production configs
- Environment variable overrides
- Secret management (database passwords, API keys)
- Feature flags for gradual rollout

### Step 2: Observability
**Structured Logging**:
- JSON-formatted logs with trace IDs
- Log levels: DEBUG, INFO, WARN, ERROR
- Request/response logging with sanitization
- Performance metrics in logs

**Metrics & Monitoring**:
- Prometheus metrics endpoint
- Key metrics:
  - Parse duration and success rate
  - API latency (p50, p95, p99)
  - Database query performance
  - Git sync success/failure rates
  - Active graph sessions
- Grafana dashboards

**Health Checks**:
- `/health` - Basic liveness check
- `/ready` - Readiness probe (database, git connectivity)
- `/metrics` - Prometheus metrics

### Step 3: Deployment
**Containerization**:
```dockerfile
# Multi-stage build for smaller images
FROM golang:1.21-alpine AS builder
# Build steps...

FROM alpine:latest
# Runtime configuration...
```

**Kubernetes Manifests**:
- Deployment with rolling updates
- ConfigMap for configuration
- Secret for sensitive data
- Service for load balancing
- Ingress for external access
- HorizontalPodAutoscaler for scaling

**Database Migrations**:
- Migration tool integration (golang-migrate)
- Version control for schema changes
- Rollback procedures
- Zero-downtime migrations

### Step 4: Security
**API Security**:
- Rate limiting per IP/user
- CORS configuration
- Request size limits
- Input validation and sanitization

**Vault Security**:
- Read-only access to vault repositories
- SSH key management for private repos
- Audit logging for all operations

### Step 5: Error Handling & Resilience
**Circuit Breakers**:
- Git operations timeout and retry
- Database connection pooling
- Graceful degradation when services unavailable

**Error Recovery**:
- Automatic retry with exponential backoff
- Dead letter queues for failed operations
- Manual intervention procedures

**Backup & Disaster Recovery**:
- Database backup strategy
- Point-in-time recovery
- Disaster recovery procedures

### Success Criteria
- [ ] Zero-downtime deployments
- [ ] <1s health check response time
- [ ] Automatic scaling based on load
- [ ] Complete audit trail of operations
- [ ] Recovery from component failures
- [ ] Monitoring alerts for critical issues
