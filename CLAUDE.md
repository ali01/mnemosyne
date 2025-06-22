# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Mnemosyne** - A web-based graph visualizer for Obsidian vault concepts, designed to handle large knowledge graphs with up to 50,000 nodes.

## Architecture

### Backend (Go + Gin)
- **API Server**: RESTful API at `localhost:8080`
- **Database**: PostgreSQL for persistent storage
- **Git Integration**: Clones and syncs Obsidian vaults from GitHub
- **Key Directories**:
  - `backend/cmd/server/` - Main server entry point
  - `backend/internal/api/` - HTTP handlers and routes
  - `backend/internal/models/` - Data models
  - `backend/internal/git/` - Git repository management
  - `backend/internal/vault/` - Vault parser (COMPLETED - 94% coverage)
  - `backend/internal/db/` - Database connection and schema
  - `backend/internal/config/` - Configuration management
  - `backend/data/` - Sample graph data and cloned vault
  - `backend/scripts/` - Testing and build scripts
  - `.github/workflows/` - CI/CD pipeline configuration

### Frontend (SvelteKit + Sigma.js)
- **Framework**: SvelteKit with TypeScript
- **Graph Rendering**: Sigma.js with WebGL
- **Key Directories**:
  - `frontend/src/routes/` - Page components
  - `frontend/src/lib/components/` - Reusable components
  - `frontend/src/lib/stores/` - State management

## Commands

### Development Setup
```bash
# Backend
cd backend
go mod download

# Create config.yaml from example
cp config.example.yaml config.yaml
# Edit config.yaml to point to your Obsidian vault repository

# Set up PostgreSQL database
createdb mnemosyne
# The schema will be automatically initialized on first run

go run cmd/server/main.go

# Frontend (in another terminal)
cd frontend
npm install
npm run dev
```

### Testing & Quality Assurance
```bash
# Run all tests with coverage
cd backend
go test ./... -v -race -coverprofile=coverage.out

# Run integration tests
./scripts/test-integration.sh

# Run performance benchmarks
go test -bench=. -benchmem ./internal/vault/...

# Run linting (0 issues expected)
golangci-lint run --config=.golangci.yml

# Test Git integration
go run cmd/test-git/main.go config.yaml
```

### Build & Deploy
```bash
# Backend
cd backend
go build -o server cmd/server/main.go

# Frontend
cd frontend
npm run build
```

## CI/CD Pipeline & Quality Assurance

### Automated Testing Pipeline
- **GitHub Actions**: Comprehensive CI/CD with PostgreSQL service containers
- **Test Coverage**: 94%+ coverage with Codecov integration
- **Performance Testing**: Automated benchmarks with regression detection
- **Security Scanning**: Gosec and Trivy vulnerability detection
- **Code Quality**: golangci-lint with 0 issues (govet, errcheck, staticcheck, ineffassign, unused)
- **Multi-platform Builds**: Linux, macOS, Windows support
- **Docker Publishing**: Automated image builds to GitHub Container Registry

### Test Infrastructure
```bash
# Test files and coverage:
backend/internal/vault/
├── *_test.go           # 79 test functions, 165+ test cases
├── benchmark_test.go   # Performance benchmarks
├── integration_test.go # End-to-end testing (1000+ files)
└── testdata/          # Edge cases, unicode, malformed data

# Test results:
- Vault Parser: 94.0% coverage
- Database Module: 37.2% coverage  
- Frontmatter: 100% coverage
- WikiLink: 90.9% coverage
- Link Resolution: 65.7% coverage
```

### Performance Metrics
- **WikiLink extraction**: ~874ns for simple patterns, scales to 1000+ links
- **Link resolution**: Sub-20ns lookups even with 10,000 files
- **Frontmatter parsing**: ~4μs minimal, ~78μs for large frontmatter
- **Complete vault parsing**: ~2.6ms for 100 files with concurrency
- **Memory allocation**: Tracked and optimized for minimal GC pressure

## Key Design Decisions

1. **PostgreSQL Database**: Added for persistent storage of nodes, edges, and positions
2. **Frontmatter IDs**: Every markdown file must have a unique `id` field in frontmatter
3. **Read-Only Vault Access**: The vault repository is cloned as read-only reference data
4. **Git Clone Strategy**: Using local Git clone for GitHub-hosted vaults (better performance, no rate limits)
5. **YAML Configuration**: Uses `config.yaml` for all settings including database connection
6. **Zero-Defect Policy**: 0 linting issues, comprehensive error handling, 94%+ test coverage

### Current Vault
The project is configured to use Ali's memex vault (594 markdown files) at `backend/data/memex-clone/`.

### Data Source Strategy: Git Clone
We're using the Git Clone approach for accessing GitHub-hosted Obsidian vaults:

**Advantages:**
- No API rate limits (can process 50k+ files efficiently)
- Fast local file reads after initial clone
- Full text search and analysis capabilities
- Offline operation support
- Simpler implementation

## Implementation Progress

### Phase 1: Git Integration Setup - COMPLETED
1. **Git dependencies** added
2. **Git manager** (`backend/internal/git/`)
   - `manager.go`: Git operations (clone, pull, sync)
   - `config.go`: YAML configuration loading
   - `errors.go`: Error definitions
3. **Features implemented**:
   - Thread-safe operations with mutex locks
   - Automatic retry and error handling
   - Force pull for read-only vault access
   - SSH authentication support
   - Configurable sync intervals (default: 5 minutes)
   - Test program for verification

### Phase 2: Vault Parser Implementation - COMPLETED
1. **Database Setup** - COMPLETED
   - PostgreSQL schema created (`backend/internal/db/schema.sql`)
   - Connection management (`backend/internal/db/connection.go`)
   - Migration system (`backend/internal/db/migrations.go`)
   - Comprehensive configuration (`backend/internal/config/config.go`)

2. **Vault Parser Components** - COMPLETED
   - `backend/internal/vault/frontmatter.go` - YAML frontmatter parser
   - `backend/internal/vault/wikilink.go` - WikiLink extractor
   - `backend/internal/vault/markdown.go` - Markdown file processor
   - `backend/internal/vault/parser.go` - Main parsing orchestrator
   - `backend/internal/vault/resolver.go` - Link resolution system

3. **Features implemented**:
   - Extracts frontmatter with required `id` field
   - Parses all WikiLink formats: `[[Note]]`, `[[Note|Alias]]`, `[[Note#Section]]`, `![[Embed]]`
   - Multi-strategy link resolution (exact path, basename, fuzzy matching)
   - Concurrent file processing with configurable workers
   - Comprehensive error tracking and statistics

4. **Testing & Quality Assurance** - COMPLETED
   - **79 test functions** with 165+ test cases across all modules
   - **94%+ test coverage** with comprehensive edge case handling
   - **Performance benchmarks**: Sub-millisecond parsing for 100+ files
   - **Integration tests**: End-to-end vault parsing with 1000+ files
   - **Error handling**: Unicode, malformed input, concurrent operations
   - **0 linting issues**: All code quality warnings resolved

### Phase 3: Graph Construction - TODO
1. **Data model implementation** (`backend/internal/models/vault.go`)
   - VaultNode structure using frontmatter ID
   - VaultEdge for WikiLinks
   - Graph statistics

2. **Graph builder** (`backend/internal/vault/graph_builder.go`)
   - Convert parsed files to nodes
   - Create edges from resolved WikiLinks
   - Calculate node types based on tags and paths
   - Compute graph metrics

3. **Layout algorithm** (`backend/internal/layout/`)
   - Force-directed layout (Fruchterman-Reingold)
   - Initial positioning based on node types
   - Store positions in database

### Phase 4: API Integration - TODO
1. **Update graph handlers** to use vault parser instead of sample JSON
2. **Add new endpoints**:
   - `GET /api/v1/vault/status` - Parse status
   - `POST /api/v1/vault/parse` - Trigger parsing
   - `GET /api/v1/nodes/:id/content` - Get markdown with rendered links
   - `GET /api/v1/search?q=term` - Full-text search

3. **Markdown rendering** with clickable WikiLinks

### Phase 5: Caching & Performance - TODO
1. **Multi-level cache** (memory → database)
2. **Incremental parsing** for changed files
3. **Background workers** for sync and metrics

### Phase 6: File Watching & Live Updates - TODO
1. **File system watcher** for vault changes
2. **WebSocket notifications** to frontend

## Database Schema

```sql
-- Core tables
nodes (id, path, title, content, frontmatter, node_type, tags, timestamps)
edges (id, source_id, target_id, edge_type, display_text, weight)
node_positions (node_id, x, y, z, locked, updated_at)

-- Metadata tables
vault_metadata (key, value, updated_at)
parse_history (id, started_at, completed_at, stats, status)
unresolved_links (id, source_id, target_text, link_type)

-- Indexes for performance
Multiple indexes on foreign keys, node types, tags, and full-text search
```

## Configuration Structure

```yaml
server:
  host: localhost
  port: 8080

database:
  host: localhost
  port: 5432
  user: mnemosyne
  password: mnemosyne
  dbname: mnemosyne
  sslmode: disable

git:
  url: git@github.com:ali01/memex.git
  branch: main
  local_path: data/memex-clone
  auto_sync: true
  sync_interval: 5m

graph:
  layout:
    algorithm: force-directed
    iterations: 500
  cache:
    enabled: true
    ttl: 30m
  batch_size: 100
  max_concurrency: 4
```

## Current Status & Next Steps

### Recently Completed
- **Phase 2 Vault Parser**: Fully implemented with 94%+ test coverage
- **Testing Infrastructure**: 79 test functions, performance benchmarks, integration tests
- **CI/CD Pipeline**: Automated testing, security scanning, code quality checks
- **Code Quality**: 0 linting issues, comprehensive error handling
- **Documentation**: Package comments, exported variable documentation

### Phase 3: Graph Construction (Next Priority)
1. **Create VaultNode model** in `backend/internal/models/vault.go`
   - Convert parsed MarkdownFile to graph-ready VaultNode structure
   - Map frontmatter IDs to node identifiers
   - Calculate node types based on tags and file paths

2. **Implement graph builder** (`backend/internal/vault/graph_builder.go`)
   - Convert ParseResult to Node/Edge collections
   - Create edges from resolved WikiLinks
   - Assign node types (index, hub, concept, reference, project)
   - Generate graph statistics and metrics

3. **Database repositories** for CRUD operations
   - Node repository with full-text search capabilities
   - Edge repository with relationship queries
   - Position repository for layout persistence
   - Batch operations for performance

### Phase 4: API & Visualization (Upcoming)
1. **Update API handlers** to serve real vault data instead of sample JSON
2. **Implement force-directed layout** algorithm (Fruchterman-Reingold)
3. **Add new endpoints** for vault operations and search
4. **Markdown rendering** with clickable WikiLinks

## Important Notes

- Every markdown file in the vault MUST have an `id` field in frontmatter
- The vault at `backend/data/memex-clone/` has 594 markdown files
- Node types are determined by:
  - `index` tag → index node
  - `~` prefix in filename → hub node
  - `open-question` tag → question node
  - Directory path → concept, reference, project, prototype
- WikiLinks create directed edges between nodes
- The system is designed to scale to 50,000+ nodes