# Mnemosyne

A graph visualizer for Obsidian vaults. Runs as a single binary that reads your vault directly, indexes it into SQLite, and serves an interactive graph in the browser. File changes are detected automatically and reflected in real-time.

## Quick Start

### Prerequisites
- Go 1.23+
- Node.js 20+

### Build & Run

```bash
make build
```

Edit `config.yaml` to point to your vault:

```yaml
vault_path: ~/path/to/your/obsidian/vault
```

Run:

```bash
./mnemosyne config.yaml
```

Open http://localhost:5555 in your browser.

### Development

Run the frontend dev server (with hot reload) and Go backend simultaneously:

```bash
make dev
```

Frontend at http://localhost:5173, API at http://localhost:5555.

## Features

- **Local vault reading**: Reads directly from your Obsidian vault directory
- **Live updates**: File changes detected via fsnotify, graph updates via SSE
- **Interactive graph**: Sigma.js with WebGL rendering, community detection, force-directed layout
- **Persistent layout**: Node positions saved to SQLite, persist across sessions
- **Search**: Full-text search via SQLite FTS5
- **Single binary**: Frontend embedded in the Go binary, no separate web server needed
- **Node classification**: Configurable rules to categorize nodes by tags, filenames, or paths

## Configuration

```yaml
vault_path: ~/my-vault          # Path to Obsidian vault (required)
port: 5555                       # HTTP port (default: 5555)
db_path: ~/.config/mnemosyne/mnemosyne.db  # SQLite database path (default)

node_classification:             # Optional
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

## Architecture

Single Go binary with embedded frontend:

```
Vault Directory → fsnotify watcher → Indexer → SQLite → net/http API → Embedded Frontend (Svelte + Sigma.js)
                                                              ↓
                                                         SSE events → Browser auto-reloads graph
```

### Key Packages

| Package | Purpose |
|---------|---------|
| `internal/store` | SQLite data access (nodes, edges, positions, metadata) |
| `internal/indexer` | Vault parsing and database synchronization |
| `internal/watcher` | fsnotify file watcher with debouncing |
| `internal/api` | net/http handlers, SSE, embedded static file serving |
| `internal/vault` | Markdown parser, WikiLink resolver, graph builder |
| `internal/config` | YAML configuration loading |
| `internal/models` | Data structures (VaultNode, VaultEdge, NodePosition) |

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/graph` | Full graph (nodes + edges + positions) |
| GET | `/api/v1/nodes/{id}` | Single node |
| GET | `/api/v1/nodes/{id}/content` | Node markdown content |
| GET | `/api/v1/nodes/search?q=` | Full-text search |
| PUT | `/api/v1/nodes/{id}/position` | Update node position |
| PUT | `/api/v1/nodes/positions` | Batch update positions |
| POST | `/api/v1/vault/reindex` | Trigger full re-index |
| GET | `/api/v1/events` | SSE stream for live updates |

## License

MIT License - see LICENSE file for details.
