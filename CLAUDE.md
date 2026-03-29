# CLAUDE.md

## Project Overview

**Mnemosyne** - A single-binary graph visualizer for local Obsidian vaults. Supports multiple vaults, each with multiple graphs defined by `GRAPH.yaml` marker files. Reads vaults directly from disk, indexes into SQLite, serves a web UI, and watches for file changes in real-time.

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
- **File watching**: `fsnotify` with debounced incremental indexing (one watcher per vault)
- **Live updates**: Server-Sent Events (SSE) push graph changes to browser with graph IDs
- **Frontend**: Embedded in binary via `//go:embed`

### Key Packages
- `internal/store/` - SQLite data access (all queries in one file)
- `internal/indexer/` - IndexManager: multi-vault parsing and DB synchronization
- `internal/discovery/` - GRAPH.yaml scanning, graph membership (IsUnderPath)
- `internal/watcher/` - Per-vault fsnotify watcher with debouncing
- `internal/api/` - net/http handlers, SSE endpoint, static file serving
- `internal/vault/` - Markdown parser, WikiLink resolver, graph builder, node classifier
- `internal/models/` - Data structures (VaultNode, VaultEdge, NodePosition, Vault, GraphInfo)
- `internal/config/` - YAML configuration loading

### Multi-Vault / Multi-Graph Model
- **Config** at `~/.config/mnemosyne/config.yaml` defines `port` and `vaults` list
- Each vault directory can contain `GRAPH.yaml` files in subdirectories
- Each `GRAPH.yaml` marks its directory as a graph root (includes all `.md` files below)
- **No nested GRAPH.yaml**: sibling graphs OK, but ancestor/descendant is an error
- Each node belongs to at most one graph
- Positions are per-graph (independent layouts)
- `GRAPH.yaml` can optionally contain `node_classification` config

### Frontend (Svelte + Vite + Sigma.js)
- **Framework**: Plain Svelte (no SvelteKit) with Vite
- **Graph Rendering**: Sigma.js with WebGL
- **Layout**: ForceAtlas2 with Louvain community detection
- **Routing**: Hash-based client-side router
- **Key Files**:
  - `frontend/src/App.svelte` - Root component with router, auto-redirects to first graph
  - `frontend/src/lib/components/GraphVisualizer.svelte` - Main graph component (accepts graphId)
  - `frontend/src/lib/pages/GraphPage.svelte` - Graph page with SSE listener and graph selector
  - `frontend/src/lib/pages/GraphListPage.svelte` - Graph/vault picker landing page
  - `frontend/src/lib/pages/NotePage.svelte` - Note detail page
  - `frontend/src/lib/router.ts` - Hash-based router

### Frontend Routes
- `/#/` - Redirects to first graph, or shows GraphListPage
- `/#/graphs/:graphId` - Graph visualization for a specific graph
- `/#/graphs/:graphId/notes/:nodeId` - Note viewer with back-to-graph navigation

## Commands

### Build & Run
```bash
make build              # Build frontend + Go binary
./mnemosyne             # Run (reads ~/.config/mnemosyne/config.yaml)
./mnemosyne config.yaml # Run with custom config path
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

Global config at `~/.config/mnemosyne/config.yaml` (or CLI arg):

```yaml
port: 5555          # Optional: HTTP port (default 5555)
vaults:             # Required: list of vault root paths
  - ~/home/walros
  - ~/home/research
```

Per-graph config in `GRAPH.yaml` (placed in any vault subdirectory):

```yaml
name: "Custom Name"    # Optional: defaults to directory name
node_classification:   # Optional: node type rules
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

### Classification Rule Types
- `tag`: Match frontmatter tags
- `filename_prefix`: Match start of filename
- `filename_suffix`: Match end of filename (excluding .md)
- `filename_match`: Exact filename match
- `path_contains`: Directory name in path
- `regex`: Regular expression on filename

## Database Schema

SQLite with 7 tables:

```sql
vaults (id, name, path, created_at)
graphs (id, vault_id, name, root_path, config, created_at, updated_at)
nodes (id, vault_id, file_path, title, content, frontmatter, node_type, tags, in_degree, out_degree, created_at, updated_at, parsed_at)
edges (id, source_id, target_id, edge_type, display_text, weight, created_at)
graph_nodes (graph_id, node_id)  -- junction table
node_positions (graph_id, node_id, x, y, z, locked, updated_at)  -- per-graph positions
vault_metadata (key, value, updated_at)
```

Full-text search via FTS5 virtual table (`nodes_fts`) with automatic sync triggers.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/graphs` | List all graphs with node counts |
| GET | `/api/v1/graphs/{id}` | Graph-scoped nodes + edges + positions |
| GET | `/api/v1/graphs/{id}/search?q=` | Full-text search within a graph |
| PUT | `/api/v1/graphs/{id}/positions` | Batch update positions for a graph |
| PUT | `/api/v1/graphs/{id}/positions/{nodeId}` | Update single position |
| GET | `/api/v1/nodes/{id}` | Single node metadata |
| GET | `/api/v1/nodes/{id}/content` | Node markdown content |
| POST | `/api/v1/reindex` | Trigger full re-index of all vaults |
| GET | `/api/v1/events` | SSE stream (graph-updated with graphIds, graphs-changed) |

## Testing Strategy

All tests run against in-memory SQLite -- no external services needed.

### Test Coverage
- **Store**: 38 tests - Vault/graph CRUD, graph-scoped queries, vault-scoped replace, position independence
- **Discovery**: 12 tests - GRAPH.yaml scanning, nesting validation, IsUnderPath
- **Indexer**: 10 tests - Multi-vault indexing, sibling graphs, GRAPH.yaml lifecycle
- **Watcher**: 8 tests - File detection, GRAPH.yaml support, onChange with graph IDs
- **API**: 16 tests - All handlers, graph-scoped endpoints, CORS
- **Config**: 7 tests - Multi-vault format, defaults, validation
- **Vault Parser**: Existing tests with 94% coverage
- **Models**: Validation and serialization tests

### Running Tests
```bash
go test ./internal/... -count=1 -timeout=120s
```

## Key Design Decisions

1. **SQLite**: Embedded database, no external services. WAL mode for concurrent reads.
2. **Single binary**: Frontend embedded via `//go:embed`. `make build` produces one executable.
3. **Multi-vault**: Multiple vaults indexed into a single SQLite DB. Vault-scoped replace preserves other vaults' data.
4. **GRAPH.yaml**: Marker files define graphs. No nesting allowed (sibling graphs only). Each node belongs to at most one graph.
5. **Graph-scoped positions**: Each graph has independent node positions, preserved across re-indexes.
6. **Graph-scoped edges**: An edge only appears in a graph if both its source and target are members.
7. **fsnotify + SSE**: File changes trigger incremental re-index, SSE pushes graph IDs to browser for targeted refresh.
8. **No framework**: Uses Go standard library `net/http` (1.22+ ServeMux with path patterns).
9. **Flat architecture**: No repository pattern, no service layer. Direct SQL in store module.
10. **Frontmatter IDs**: Every markdown file needs a unique `id` field in frontmatter to be indexed. IDs must be globally unique across all vaults.
