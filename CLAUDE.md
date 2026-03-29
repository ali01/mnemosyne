# CLAUDE.md

## Project Overview

**Mnemosyne** - A single-binary graph visualizer for local Obsidian vaults. Reads the vault directly from disk, indexes into SQLite, serves a web UI, and watches for file changes in real-time.

CRITICAL: When you encounter a file reference (e.g., ROADMAP.md), use your Read tool to load it on a need-to-know basis.

## Working Directory

- **Project root**: `/Users/alive/home/mnemosyne`
- **Go code**: `/Users/alive/home/mnemosyne/internal/` and `/Users/alive/home/mnemosyne/cmd/`
- **Frontend code**: `/Users/alive/home/mnemosyne/frontend`

## Code Style Guidelines

### Indentation
- **Frontend (JavaScript/TypeScript/Svelte)**: 2 spaces
- **Go**: Tabs (Go standard)

## Architecture

### Single Binary (Go + net/http)
- **Entry point**: `cmd/mnemosyne/main.go`
- **HTTP server**: Go 1.22+ `net/http.ServeMux` (no frameworks)
- **Database**: SQLite via `modernc.org/sqlite` (pure Go, no CGo)
- **File watching**: `fsnotify` with debounced incremental indexing
- **Live updates**: Server-Sent Events (SSE) push graph changes to browser
- **Frontend**: Embedded in binary via `//go:embed`

### Key Packages
- `internal/store/` - SQLite data access (all queries in one file)
- `internal/indexer/` - Vault parsing and DB synchronization
- `internal/watcher/` - fsnotify file watcher with debouncing
- `internal/api/` - net/http handlers, SSE endpoint, static file serving
- `internal/vault/` - Markdown parser, WikiLink resolver, graph builder, node classifier
- `internal/models/` - Data structures (VaultNode, VaultEdge, NodePosition)
- `internal/config/` - YAML configuration loading

### Frontend (Svelte + Vite + Sigma.js)
- **Framework**: Plain Svelte (no SvelteKit) with Vite
- **Graph Rendering**: Sigma.js with WebGL
- **Layout**: ForceAtlas2 with Louvain community detection
- **Routing**: Simple hash-based client-side router (2 routes: `/` and `/notes/:id`)
- **Key Files**:
  - `frontend/src/App.svelte` - Root component with router
  - `frontend/src/lib/components/GraphVisualizer.svelte` - Main graph component
  - `frontend/src/lib/pages/GraphPage.svelte` - Graph page with SSE listener
  - `frontend/src/lib/pages/NotePage.svelte` - Note detail page
  - `frontend/src/lib/router.ts` - Hash-based router

## Commands

### Build & Run
```bash
make build              # Build frontend + Go binary
./mnemosyne config.yaml # Run the binary
```

### Development
```bash
make dev                # Run frontend dev server + Go backend in parallel
make test               # Run all Go tests
```

### Testing
```bash
go test ./internal/... -count=1          # All tests
go test ./internal/store/... -v          # Store tests only
go test ./internal/vault/... -v          # Vault parser tests
go test -bench=. -benchmem ./internal/vault/...  # Benchmarks
```

## Configuration

`config.yaml` at project root:

```yaml
vault_path: ~/path/to/vault    # Required: path to Obsidian vault
port: 5555                      # Optional: HTTP port (default 5555)
db_path: ~/.config/mnemosyne/mnemosyne.db  # Optional: SQLite path

node_classification:             # Optional: node type rules
  default_node_type: note
  node_types:
    hub:
      display_name: "Hub"
      color: "#4ECDC4"
      size_multiplier: 1.5
  classification_rules:
    - name: hub_prefix
      priority: 2
      type: filename_prefix
      pattern: "~"
      node_type: hub
```

### Rule Types
- `tag`: Match frontmatter tags
- `filename_prefix`: Match start of filename
- `filename_suffix`: Match end of filename (excluding .md)
- `filename_match`: Exact filename match
- `path_contains`: Directory name in path
- `regex`: Regular expression on filename

## Database Schema

SQLite with 4 tables:

```sql
nodes (id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at, parsed_at)
edges (id, source_id, target_id, edge_type, display_text, weight, created_at)
node_positions (node_id, x, y, z, locked, updated_at)
vault_metadata (key, value, updated_at)
```

Full-text search via FTS5 virtual table (`nodes_fts`) with automatic sync triggers.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/graph` | Full graph data |
| GET | `/api/v1/nodes/{id}` | Single node |
| GET | `/api/v1/nodes/{id}/content` | Node markdown content |
| GET | `/api/v1/nodes/search?q=` | Full-text search |
| PUT | `/api/v1/nodes/{id}/position` | Update node position |
| PUT | `/api/v1/nodes/positions` | Batch update positions |
| POST | `/api/v1/vault/reindex` | Trigger full re-index |
| GET | `/api/v1/events` | SSE stream for live graph updates |

## Testing Strategy

All tests run against in-memory SQLite — no external services needed.

### Test Coverage
- **Store**: 34 tests - CRUD, search, bulk operations, file persistence
- **Indexer**: 10 tests - Full/incremental indexing with temp vaults
- **Watcher**: 7 tests - File create/modify/delete detection, debouncing
- **API**: 14 tests - All handlers, CORS, error cases
- **Config**: 5 tests - Loading, defaults, validation, path expansion
- **Vault Parser**: Existing tests with 94% coverage
- **Models**: Validation and serialization tests

### Running Tests
```bash
go test ./internal/... -count=1 -timeout=60s
```

## Key Design Decisions

1. **SQLite**: Embedded database, no external services. WAL mode for concurrent reads.
2. **Single binary**: Frontend embedded via `//go:embed`. `make build` produces one executable.
3. **Local vault**: Reads directly from filesystem. No Git, no cloning, no network.
4. **fsnotify + SSE**: File changes trigger incremental re-index, SSE pushes updates to browser.
5. **No framework**: Uses Go standard library `net/http` (1.22+ ServeMux with path patterns).
6. **Flat architecture**: No repository pattern, no service layer. Direct SQL in store module.
7. **Frontmatter IDs**: Every markdown file needs a unique `id` field in frontmatter to be indexed.
