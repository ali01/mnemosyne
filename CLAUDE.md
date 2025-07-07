# CLAUDE.md

## Project Overview

**Mnemosyne** - A web-based graph visualizer for Obsidian vault concepts designed to handle large knowledge graphs. Details and goals of the project:
- The visualizer gets its data from a GitHub repository that contains the Obsidian vault.
- Scale to graphs with up to 50,000 nodes while remaining performant
- Display large graphs in a way that remains intelligible to the user. The visualizer needs to be capable of operating at various zoom levels so that the user can make sense of the whole thing.
- The user should be able to move the nodes around to organize the graph. The positions of the nodes need to be saved so that they persist across sessions.
- When a user clicks on a node representing a note, the visualizer will open a page with the rendered markdown of the note's contents.

## Implementation Status

⚠️ **IMPORTANT**: While core components are complete, they are NOT yet integrated. The API currently serves sample data from `backend/data/sample_graph.json`.

For current implementation status and roadmap, see `ROADMAP.md`.


## Working Directory

- **Project root**: `/Users/alive/home/mnemosyne`
- **Backend code**: `/Users/alive/home/mnemosyne/backend`
- **Frontend code**: `/Users/alive/home/mnemosyne/frontend`
- **Important**: Always verify current directory with `pwd` before running commands
- **Preferred approach**: Use absolute paths or chain commands with `&&` to maintain directory context

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

⚠️ **Note**: Backend currently serves sample data only. See `ROADMAP.md` for integration status and next steps.

```bash
# Backend Setup
cd backend
go mod download

# IMPORTANT: Edit config.yaml to point to your Obsidian vault
# Note: Git cloning works but data doesn't flow to API yet
cp config.example.yaml config.yaml
# Edit config.yaml - set git.url to your vault repository

# Database setup (schema auto-initializes)
createdb mnemosyne

# Run server (currently serves sample data from backend/data/sample_graph.json)
go run cmd/server/main.go
# API available at http://localhost:8080/api/v1/graph

# Frontend Setup (in another terminal)
cd frontend
npm install
npm run dev
# Opens http://localhost:5173 - displays sample graph, not your vault
```

### Testing & Quality Assurance
See the "Testing Strategy" section below for comprehensive testing information.

```bash
# Quick test commands:
cd backend
go test ./... -v                    # Run all tests
go test -bench=. ./internal/vault/  # Run benchmarks
golangci-lint run                   # Run linter
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

## Common Development Tasks

### Testing the Vault Parser (Works Today)
```bash
cd backend
go test -v ./internal/vault/...
# Expected: All tests pass with 94% coverage
```

### Testing Git Integration (Works Today)
```bash
cd backend
go run cmd/test-git/main.go config.yaml
# Expected: Successfully clones your configured vault
```

### Running the API with Sample Data (Works Today)
```bash
cd backend
go run cmd/server/main.go
# API serves sample data at http://localhost:8080/api/v1/graph
# Note: Returns data from backend/data/sample_graph.json, not your vault
```

### Viewing the Sample Graph (Works Today)
```bash
# Terminal 1: Start backend
cd backend && go run cmd/server/main.go

# Terminal 2: Start frontend
cd frontend && npm run dev

# Open http://localhost:5173 in browser
# Shows sample graph visualization (not your vault data)
```

### Parsing a Real Vault (NOT YET IMPLEMENTED)
```bash
# This requires completing Phase 3.5 (see ROADMAP.md)
# Once complete, it will be:
POST http://localhost:8080/api/v1/vault/parse
```

### Adding a New Node Type
1. Edit `backend/config.yaml` - Add to `node_classification.node_types`
2. Add classification rule in `node_classification.classification_rules`
3. Restart server to apply changes

### Debugging Why a File Wasn't Parsed
```bash
# Check if file has required 'id' in frontmatter
cd backend
grep -A5 "^---" path/to/file.md

# Run parser on specific directory
go test -v ./internal/vault -run TestParseDirectory
```

### Checking Database Schema
```bash
# The schema auto-initializes on first run
psql mnemosyne -c "\dt"  # List tables
psql mnemosyne -c "\d nodes"  # Show nodes table structure
```

## Understanding the Codebase

For implementation status and component details, see `ROADMAP.md`.

### Key Files to Understand
- `ROADMAP.md` - Detailed implementation phases and current progress
- `backend/config.yaml` - All configuration options with examples
- `backend/data/sample_graph.json` - Current API data source (temporary)
- `backend/internal/models/vault.go` - Core data structures (VaultNode, VaultEdge)
- `backend/scripts/test-integration.sh` - Integration test setup (awaiting integration)

### Architecture Decisions
- **Two Model Sets**: `models.Node/Edge` (API) vs `models.VaultNode/VaultEdge` (database)
  - API models are simplified for frontend consumption
  - Vault models contain full metadata and content
- **Repository Pattern**: See Phase 3.5 in ROADMAP.md
  - Isolates database logic from business logic
  - Enables testing with mock repositories
- **Service Layer**: See Phase 3.5 in ROADMAP.md
  - Orchestrates parser → builder → database pipeline
  - Handles async processing and status tracking

## Key Design Decisions

1. **PostgreSQL Database**: Added for persistent storage of nodes, edges, and positions
2. **Frontmatter IDs**: Every markdown file has a unique `id` field in frontmatter
3. **Read-Only Vault Access**: The vault repository is cloned as read-only reference data
4. **Git Clone Strategy**: Using local Git clone for GitHub-hosted vaults. This bypasses API rate limits and enables fast local file reads once the vault is cloned.
5. **YAML Configuration**: Uses `config.yaml` for all settings including database connection

### Current Vault
The project is configured to use Ali's memex vault (594 markdown files) at `backend/data/memex-clone/`.

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

## Testing Strategy

### Current Test Coverage
- **Vault Parser**: 94% coverage ✓
  - Comprehensive unit tests for all parsing functions
  - Edge case handling (circular links, unicode, invalid YAML)
  - Performance benchmarks included
- **Graph Builder**: Comprehensive tests ✓
  - Tests for duplicate handling, orphan nodes, metrics
  - Mock data for predictable testing
- **Models**: Full validation tests ✓
  - VaultNode and VaultEdge validation
  - JSON serialization/deserialization

### Missing Tests
- **Integration Tests**: No integration to test yet
  - `test-integration.sh` exists but has no actual tests
  - Waiting for service/repository implementation
- **API Tests**: Currently using sample data
  - Need tests with real database queries
  - Mock repositories not yet implemented
- **End-to-End Tests**: Blocked by integration gap
  - Cannot test full flow: Git → Parser → Database → API

### Running Tests
```bash
# Unit tests with coverage
cd backend
go test ./... -v -race -coverprofile=coverage.out
go tool cover -html=coverage.out  # View coverage report

# Run specific package tests
go test -v ./internal/vault/...  # Test vault parser only

# Benchmarks (see BENCHMARKS.md for detailed results)
go test -bench=. -benchmem ./internal/vault/...

# Linting (0 issues expected)
golangci-lint run --config=.golangci.yml

# Integration tests (placeholder for now)
./scripts/test-integration.sh
```

### Test Data Locations
- **Sample vault**: `backend/data/testdata/sample_vault/`
- **Edge cases**: `backend/data/testdata/edge_cases/`
- **Invalid files**: `backend/data/testdata/invalid_files/`
- **Sample graph**: `backend/data/sample_graph.json` (API placeholder)

## Performance Expectations

Based on comprehensive benchmarks (see `BENCHMARKS.md` for full results):

### Target Scale: 50,000 Nodes
- **Parsing time**: ~457ms for typical documents (9.15 μs × 50,000)
- **Memory usage**: ~1GB for full vault in memory (19.1 KB × 50,000)
- **Link resolution**: O(1) constant time regardless of vault size
- **Database queries**: Should be <100ms with proper indexes

### Current Performance Characteristics
- **WikiLink extraction**: ~874ns for simple patterns
- **Frontmatter parsing**: ~4μs minimal, ~78μs for large metadata
- **Link resolution**: 12-23ns lookups even with 10,000 files
- **Concurrent processing**: Linear speedup to CPU core count

### Known Limitations
- **API loads entire graph**: No pagination or viewport filtering yet
- **Content field**: Should implement lazy loading for files >100KB
- **Memory scaling**: Linear with content size (consider streaming for huge vaults)

### Performance Monitoring
```bash
# Run benchmarks
cd backend
go test -bench=. -benchmem ./internal/vault/...

# CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./internal/vault/...
go tool pprof cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof ./internal/vault/...
go tool pprof mem.prof
```

## Configuration Structure
Backend configuration is in `backend/config.yaml`.

### Key Configuration Sections
- **server**: API host and port settings
- **database**: PostgreSQL connection parameters
- **git**: Repository URL, branch, and sync settings
- **graph**: Processing options, layout algorithm, cache settings
- **node_classification**: Node types and classification rules (see section below)

## Node Classification

The node classification system categorizes vault files into different types for visualization. Node types and classification rules are configured directly in `config.yaml`.

### Node Types
Each node type can have:
- `display_name`: Human-readable name
- `description`: Purpose of this node type
- `color`: Hex color for visualization
- `size_multiplier`: Relative size in graph (1.0 = normal)

### Classification Rules
Rules are evaluated in priority order (lower number = higher priority):
- **Priority 1-20**: Tag-based rules (e.g., frontmatter tags)
- **Priority 21-40**: Filename-based rules (e.g., prefix/suffix)
- **Priority 41-60**: Path-based rules (e.g., directory names)
- **Priority 61+**: Custom patterns

**Note**: All pattern matching is case-insensitive by design.

### Rule Types
- `tag`: Match frontmatter tags (case-insensitive)
- `filename_prefix`: Match start of filename (case-insensitive)
- `filename_suffix`: Match end of filename excluding .md (case-insensitive)
- `filename_match`: Exact filename match (case-insensitive)
- `path_contains`: Directory name anywhere in path (case-insensitive)
- `regex`: Regular expression on filename (case-insensitive)

### Example: Adding a Custom Node Type
```yaml
node_classification:
  node_types:
    research:
      display_name: "Research"
      description: "Research notes and findings"
      color: "#9B59B6"
      size_multiplier: 1.4

  classification_rules:
    - name: "research_tag"
      priority: 5
      type: "tag"
      pattern: "research"
      node_type: "research"
```
