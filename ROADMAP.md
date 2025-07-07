# Mnemosyne Development Roadmap

## Current Status
- ✅ Phase 1: Git Integration - COMPLETED
- ✅ Phase 2: Vault Parser - COMPLETED (94%+ test coverage)
- ✅ Phase 3: Graph Construction - COMPLETED (core components built)
- 🚧 Phase 3.5: Integration Layer - IN PROGRESS (current priority)
- ⏳ Phase 4: API Integration
- ⏳ Phase 5: Caching & Performance
- ⏳ Phase 6: File Watching & Live Updates
- ⏳ Phase 7: Production Readiness

## Immediate Next Steps

### Week 1: Complete Integration Layer (Phase 3.5)
1. **Day 1-2**: Create Repository Layer
   - [ ] Create `backend/internal/repository/` directory
   - [ ] Implement `node_repository.go` with basic CRUD operations
   - [ ] Implement `edge_repository.go` with graph queries
   - [ ] Write unit tests with mock database

2. **Day 3-4**: Create Service Layer
   - [ ] Create `backend/internal/service/` directory
   - [ ] Implement `vault_service.go` with parse orchestration
   - [ ] Add transaction support for atomic updates
   - [ ] Write integration tests with test database

3. **Day 5**: Wire Everything Together
   - [ ] Update `main.go` to initialize database and load config
   - [ ] Inject dependencies into API handlers
   - [ ] Replace `sample_graph.json` with real queries
   - [ ] Test end-to-end flow

### Week 2: Complete Basic API (Phase 4)
1. **Parse Management Endpoints**:
   - [ ] `POST /api/v1/vault/parse` - Trigger parsing
   - [ ] `GET /api/v1/vault/status` - Check progress

2. **Data Access Endpoints**:
   - [ ] `GET /api/v1/nodes/:id` - Get node with content
   - [ ] `GET /api/v1/search` - Search nodes by title/content
   - [ ] `PUT /api/v1/nodes/:id/position` - Persist positions

3. **Frontend Integration**:
   - [ ] Update frontend to use new endpoints
   - [ ] Add loading states during parsing
   - [ ] Test with real vault data


## Current Architecture & Data Flow

### Current Flow (Broken)
```
GitHub Vault → Git Clone ✓ → Parser ✓ → Graph Builder ✓ → ❌ (stops here)
Frontend ← API ← sample_graph.json (placeholder data)
```

### Target Flow
```
GitHub Vault → Git Clone → Parser → Graph Builder → Database → API → Frontend
                    ↑                                     ↓
                    └──────── Background Sync ←───────────┘
```

**Components Status**:
- ✓ = Implemented and tested
- ❌ = Missing integration
- The gap is between Graph Builder output and Database input

## MVP vs Future Features

### MVP (Minimum Viable Product) - Target: Functional System
**Goal**: Get a working system that can parse a vault, build a graph, and serve it via API

**Phases 1-4** (Must Have):
- ✅ **Phase 1**: Git Integration - Clone and sync vaults from GitHub
- ✅ **Phase 2**: Vault Parser - Extract nodes and links from markdown files
- ✅ **Phase 3**: Graph Construction - Build graph structure from parsed data
- 🚧 **Phase 3.5**: Integration Layer - Connect components to database and API
- ⏳ **Phase 4**: Basic API - Serve real vault data, search, node content

**MVP Success Criteria**:
- Can clone a GitHub-hosted Obsidian vault
- Can parse markdown files and extract graph structure
- Can store and retrieve graph data from PostgreSQL
- API serves actual vault data (not sample data)
- Frontend displays the graph with basic interactions
- Node positions can be saved and restored

### Post-MVP Features (Nice to Have)
**Goal**: Enhance performance, user experience, and capabilities

**Phases 5-7** (Future Enhancements):
- **Phase 5**: Caching & Performance
  - Redis caching for parsed graphs
  - Lazy loading for large vaults
  - Pagination and viewport filtering
  - Advanced metrics (PageRank, clustering)
  - Backend graph layout algorithms (force-directed, hierarchical, 3D)

- **Phase 6**: File Watching & Live Updates
  - WebSocket support for real-time updates
  - Incremental parsing for changed files
  - Git webhook integration

- **Phase 7**: Production Readiness
  - Docker containerization
  - Kubernetes deployment
  - Monitoring and alerting
  - Multi-vault support
  - Authentication and authorization

## Completed Phases Summary

### Phase 1: Git Integration ✅
- Git clone and sync from GitHub repositories
- SSH and HTTPS support with authentication
- Shallow cloning for performance
- Test utility: `cmd/test-git/main.go`

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

## Phase 3.5: Integration Layer (CURRENT PRIORITY)

### Overview
Connect the completed components (parser, graph builder, node classifier) to the database and API. This is the critical missing piece that bridges the gap between the functional components and the working system.

### Step 1: Create Repository Layer
**Directory**: `backend/internal/repository/`

**NodeRepository** (`node_repository.go`):
```go
type NodeRepository interface {
    CreateBatch(ctx context.Context, nodes []models.VaultNode) error
    UpsertBatch(ctx context.Context, nodes []models.VaultNode) error
    GetByIDs(ctx context.Context, ids []string) ([]models.VaultNode, error)
    GetAll(ctx context.Context) ([]models.VaultNode, error)
    Search(ctx context.Context, query string) ([]models.VaultNode, error)
    GetByType(ctx context.Context, nodeType string) ([]models.VaultNode, error)
    UpdatePositions(ctx context.Context, positions []models.Position) error
}
```

**EdgeRepository** (`edge_repository.go`):
```go
type EdgeRepository interface {
    CreateBatch(ctx context.Context, edges []models.VaultEdge) error
    GetByNode(ctx context.Context, nodeID string) ([]models.VaultEdge, error)
    GetSubgraph(ctx context.Context, nodeIDs []string) ([]models.VaultEdge, error)
    GetAll(ctx context.Context) ([]models.VaultEdge, error)
}
```

**VaultRepository** (`vault_repository.go`):
```go
type VaultRepository interface {
    SaveParseResult(ctx context.Context, result *vault.GraphBuildResult) error
    GetParseHistory(ctx context.Context, limit int) ([]models.ParseHistory, error)
    SaveParseStatus(ctx context.Context, status models.ParseStatus) error
}
```

### Step 2: Create Service Layer
**Directory**: `backend/internal/service/`

**VaultService** (`vault_service.go`):
```go
type VaultService interface {
    // Core operations
    ParseAndIndexVault(ctx context.Context) (*models.ParseResult, error)
    GetParseStatus(ctx context.Context) (*models.ParseStatus, error)

    // Graph operations
    GetFullGraph(ctx context.Context) (*models.Graph, error)
    GetSubgraph(ctx context.Context, nodeIDs []string) (*models.Graph, error)

    // Node operations
    GetNode(ctx context.Context, id string) (*models.VaultNode, error)
    SearchNodes(ctx context.Context, query string) ([]models.VaultNode, error)
    UpdateNodePositions(ctx context.Context, positions []models.Position) error
}
```

**Service Implementation Flow**:
1. Use GitManager to ensure vault is cloned/updated
2. Run VaultParser on the cloned directory
3. Use GraphBuilder to create graph structure
4. Calculate basic metrics (degree, node types)
5. Persist everything via repositories (in transaction)
6. Update parse status and handle errors

### Step 3: Update Main Server
**File**: `backend/cmd/server/main.go`

Initialize database connection, load configuration, create repository and service instances with dependency injection.

### Step 4: Update API Handlers
**File**: `backend/internal/api/graph_handlers.go`

Replace sample data reads with service calls and implement new endpoints for vault management.

### Success Criteria
- Full pipeline works: Git clone → Parse → Build → Store → API
- API serves real vault data instead of sample_graph.json
- Node positions persist across server restarts
- Error handling and recovery implemented

### Testing Strategy
1. **Unit Tests**: Mock repositories for service testing
2. **Integration Tests**: Test full pipeline with small test vault
3. **Performance Tests**: Ensure <30s for 50K nodes
4. **Error Tests**: Network failures, invalid vaults, database errors

## Phase 4: API Integration

### Overview
Replace sample data with real vault data. Add management endpoints for vault operations.

### Key Components
1. **Updated API Handlers**: Serve data from database instead of sample JSON
2. **Vault Management Endpoints**: Parse control and status monitoring
3. **Data Access Endpoints**: Node content, search, and position persistence
4. **Markdown Renderer**: WikiLink to HTML conversion with proper linking

## Phase 5: Caching & Performance (Post-MVP)

### Overview
Optimize system performance for large vaults (50K+ nodes) through strategic caching and lazy loading.

### Components
- Redis integration for graph caching
- Lazy loading for node content
- Pagination for large result sets
- Query optimization with database indexes
- Background metrics calculation
- Advanced graph layout algorithms. Possibilities:
  - Force-directed layout (Fruchterman-Reingold)
  - Hierarchical layout based on node types
  - Community detection and clustering
  - Physics-based simulation with configurable forces
  - Persistent layout state with incremental updates

## Phase 6: File Watching & Live Updates (Post-MVP)

### Overview
Enable real-time synchronization between vault changes and graph visualization.

### Components
- File system watcher for local changes
- Git webhook receiver for remote updates
- WebSocket server for pushing updates
- Incremental parsing for changed files
- Conflict resolution for concurrent edits

## Phase 7: Production Readiness

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


## Success Metrics
- Graph construction completes in <30s for 50K nodes
- API response times <100ms for graph queries
- Test coverage remains >90%
- Zero data loss during re-indexing
- Memory usage <1GB for 50K nodes

### Quick Wins (Can Do Anytime)
- Add `/health` endpoint for basic monitoring
- Implement request logging middleware
- Add database connection pooling
- Create docker-compose for local development
- Add GitHub Actions for CI/CD

## Risk Mitigation

### Technical Risks

1. **Large Vault Performance (50K+ nodes)**
   - **Risk**: Initial graph load times out or uses too much memory
   - **Mitigation**:
     - Implement pagination from the start
     - Add `limit` and `offset` to graph endpoint
     - Stream large responses instead of loading all in memory
     - Monitor memory usage during development

2. **Git Operation Failures**
   - **Risk**: Network issues, large repositories, authentication problems
   - **Mitigation**:
     - Add configurable timeouts (default 5 minutes)
     - Implement shallow cloning for large repos
     - Cache successful clones and provide offline mode
     - Clear error messages for auth failures

3. **Database Connection Issues**
   - **Risk**: Connection pool exhaustion, network interruptions
   - **Mitigation**:
     - Configure connection pool limits
     - Implement health checks
     - Add circuit breaker for database operations
     - Cache recent data for read-only operations

4. **Concurrent Parse Operations**
   - **Risk**: Multiple parse requests causing conflicts
   - **Mitigation**:
     - Implement parse queue with single worker
     - Add status checks before starting new parse
     - Use database locks for critical sections
     - Return 409 Conflict for concurrent requests

### Operational Risks

1. **Data Loss During Re-indexing**
   - **Risk**: Losing user-saved node positions
   - **Mitigation**:
     - Never truncate node_positions table
     - Use upsert operations for nodes/edges
     - Backup positions before major operations
     - Add audit log for all mutations

2. **API Breaking Changes**
   - **Risk**: Frontend breaks when API changes
   - **Mitigation**:
     - Version API from start (`/api/v1/`)
     - Document all endpoints with OpenAPI
     - Deprecate endpoints before removal
     - Test frontend/backend together in CI

3. **Security Vulnerabilities**
   - **Risk**: Exposed credentials, injection attacks
   - **Mitigation**:
     - Never log sensitive configuration
     - Use prepared statements for all queries
     - Validate and sanitize all inputs
     - Regular dependency updates

### Performance Targets
- Parse 1,000 files: <5 seconds
- Parse 10,000 files: <30 seconds
- Graph API response: <100ms for 10K nodes
- Search response: <50ms with indexes
- Memory usage: <1GB for 50K nodes
