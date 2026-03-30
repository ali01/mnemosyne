# Mnemosyne Development Roadmap

## Current Status
- ✅ Single binary with SQLite, local vault reading, embedded frontend
- ✅ Multi-vault / multi-graph support with GRAPH.yaml markers
- ✅ Obsidian-style graph filtering and group coloring
- ✅ Vault parser with 94%+ test coverage
- ✅ Graph construction with community-based spatial layout
- ✅ Live updates via fsnotify + SSE
- ✅ Interactive visualization with Sigma.js
- ✅ Full test suite (Go + frontend)

## Architecture

```
Vault Directory → fsnotify → Indexer → SQLite → net/http API → Embedded Svelte Frontend
                                                      ↓
                                                 SSE events → Browser reloads graph
```

**Single binary**. No PostgreSQL, no Git cloning, no Gin, no SvelteKit. Just `./mnemosyne`.

## What's Done

### Core Pipeline ✅
- Markdown parser with WikiLink extraction (`[[Note]]`, `[[Note|Alias]]`, `[[Note#Section]]`)
- Concurrent file processing with worker pools
- Link resolution with multiple strategies (exact path, basename, normalized)
- Graph builder with duplicate handling and unresolved link tracking

### Multi-Vault / Multi-Graph ✅
- Config at `~/.config/mnemosyne/config.yaml` with `port`, `vaults`, `home-graph`
- GRAPH.yaml marker files define graph boundaries (no nesting allowed)
- Per-vault file watchers with GRAPH.yaml lifecycle support
- Graph-scoped positions, edges, and search
- Path-based URL routing (`/{vaultName}/{graphPath}`)
- Interactive config bootstrap on first run

### Filtering & Grouping ✅
- Obsidian search query parser (`path:`, `tag:`, `file:`, `[field:value]`, boolean logic)
- GRAPH.yaml `filter` field to control which nodes appear
- GRAPH.yaml `groups` field to assign hex colors to matching nodes
- Evaluated at API serving time (positions survive filter changes)

### Data Layer ✅
- SQLite store with FTS5 full-text search
- Upsert operations for nodes, edges, positions
- Atomic bulk replace for full re-indexing
- Position persistence across re-indexes
- Graph-scoped data retrieval with filter/group evaluation

### API ✅
- net/http handlers (Go 1.22+ ServeMux)
- Graph-scoped endpoints for data, search, positions
- SSE endpoint with typed events (graph-updated, graphs-changed)
- CORS middleware
- Embedded static file serving with SPA fallback
- `--port` / `-p` CLI flag to override config port

### Frontend ✅
- Svelte + Vite (no SvelteKit)
- Sigma.js graph with WebGL rendering
- Louvain community detection for spatial layout (not coloring)
- Group-based node coloring from API response
- Two-level ForceAtlas2 layout (meta-graph + per-community seeding)
- Node hover highlighting with neighbor dimming
- Drag with neighbor displacement
- Dark theme UI with search, zoom controls
- Path-based router with graph selector dropdown
- SSE listener for live graph reload
- Vitest test suite with 136 tests

### File Watching ✅
- Per-vault fsnotify recursive directory watching
- Debounced change batching (500ms)
- Incremental indexing (only re-parse changed files)
- GRAPH.yaml create/delete detection with graph lifecycle management
- SSE notification to connected browsers with graph IDs

## What's Next

### Performance Optimization
- Lazy loading for node content (don't load full markdown until clicked)
- Pagination for large vaults (50K+ nodes)
- Streaming graph responses instead of loading all in memory

### Graph Features
- Semantic zoom (show different detail levels at different zoom levels)
- Edge bundling to reduce visual clutter
- Shortest path highlighting between nodes

### UX
- Keyboard shortcuts for navigation
- Node details sidebar (show metadata without full page navigation)
- Minimap for large graphs
- Export graph as image
- Groups legend overlay

### Infrastructure
- Graceful shutdown with in-flight request draining
- Structured logging (JSON format)
