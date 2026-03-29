# Mnemosyne Development Roadmap

## Current Status
- ✅ Simplification: Single binary with SQLite, local vault reading, embedded frontend
- ✅ Vault Parser: 94%+ test coverage
- ✅ Graph Construction: Community detection, force-directed layout
- ✅ Live Updates: fsnotify watcher + SSE push to browser
- ✅ Interactive Visualization: Sigma.js with drag, hover, search

## Architecture

```
Vault Directory → fsnotify → Indexer → SQLite → net/http API → Embedded Svelte Frontend
                                                      ↓
                                                 SSE events → Browser reloads graph
```

**Single binary**. No PostgreSQL, no Git cloning, no Gin, no SvelteKit. Just `./mnemosyne config.yaml`.

## What's Done

### Core Pipeline ✅
- Markdown parser with WikiLink extraction (`[[Note]]`, `[[Note|Alias]]`, `[[Note#Section]]`)
- Concurrent file processing with worker pools
- Link resolution with multiple strategies (exact path, basename, normalized)
- Configurable node classification (tag, filename, path rules)
- Graph builder with duplicate handling and unresolved link tracking

### Data Layer ✅
- SQLite store with FTS5 full-text search
- Upsert operations for nodes, edges, positions
- Atomic bulk replace for full re-indexing
- Position persistence across re-indexes

### API ✅
- net/http handlers (Go 1.22+ ServeMux)
- Graph, node, content, search, position endpoints
- SSE endpoint for live graph updates
- CORS middleware
- Embedded static file serving with SPA fallback

### Frontend ✅
- Svelte + Vite (no SvelteKit)
- Sigma.js graph with WebGL rendering
- Louvain community detection with color coding
- Two-level ForceAtlas2 layout (meta-graph + per-community)
- Node hover highlighting with neighbor dimming
- Drag with neighbor displacement
- Dark theme UI with search, zoom controls
- Hash-based router (graph view + note detail view)
- SSE listener for live graph reload

### File Watching ✅
- fsnotify recursive directory watching
- Debounced change batching (500ms)
- Incremental indexing (only re-parse changed files)
- SSE notification to connected browsers

## What's Next

### Performance Optimization
- Lazy loading for node content (don't load full markdown until clicked)
- Pagination for large vaults (50K+ nodes)
- Streaming graph responses instead of loading all in memory

### Graph Features
- Semantic zoom (show different detail levels at different zoom levels)
- Filter by node type, tags, or degree
- Edge bundling to reduce visual clutter
- Shortest path highlighting between nodes

### UX
- Keyboard shortcuts for navigation
- Node details sidebar (show metadata without full page navigation)
- Minimap for large graphs
- Export graph as image

### Infrastructure
- Graceful shutdown with in-flight request draining
- Health check endpoint improvements
- Structured logging (JSON format)
