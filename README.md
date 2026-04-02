# Mnemosyne

A graph visualizer for Obsidian vaults. Runs as a single binary that reads your vault directly, indexes it into SQLite, and serves an interactive graph in the browser. Supports multiple vaults, each with multiple graphs defined by `GRAPH.yaml` marker files. File changes are detected automatically and reflected in real-time.

## Quick Start

### Prerequisites
- Go 1.23+
- Node.js 20+

### Build & Run

```bash
make            # Build frontend + Go binary
make run        # Build and run
```

On first run with no config, Mnemosyne prompts for your vault path and creates `~/.config/mnemosyne/config.yaml`.

```bash
./mnemosyne                 # Uses default config
./mnemosyne config.yaml     # Custom config path
./mnemosyne -p 8080         # Override port
./mnemosyne graphs          # List all graphs (active + archived)
./mnemosyne graphs delete 5 # Permanently delete a graph
```

Open http://localhost:5555 in your browser.

### Development

Run the frontend dev server (with hot reload) and Go backend simultaneously:

```bash
make dev
```

Frontend at http://localhost:5173, API at http://localhost:5555.

## Features

- **Multi-vault / multi-graph**: Configure multiple vaults, each with multiple graphs via `GRAPH.yaml` markers
- **Obsidian-style filtering**: Filter which nodes appear using Obsidian search syntax (`path:`, `tag:`, `file:`, `[field:value]`, boolean operators)
- **Group coloring**: Assign colors to node groups using the same search syntax
- **Live updates**: File changes detected via fsnotify, graph updates via SSE
- **Open in Obsidian**: Click any node to open the file directly in Obsidian via `obsidian://` URI protocol
- **Interactive graph**: Sigma.js with WebGL rendering, force-directed layout with community-based spatial grouping
- **Persistent layout**: Node positions saved per-graph to SQLite, persist across sessions
- **Graph archiving**: Deleting a GRAPH.yaml preserves positions in the database; re-adding it restores the graph with its saved layout
- **Search**: Full-text search via SQLite FTS5
- **Single binary**: Frontend embedded in the Go binary, no separate web server needed
- **CLI management**: List and permanently delete graphs via `mnemosyne graphs`

## Configuration

Global config at `~/.config/mnemosyne/config.yaml`:

```yaml
port: 5555              # Optional: HTTP port (default 5555)
home-graph: walros/memex  # Optional: default graph for root URL redirect
vaults:                 # Required: list of vault root paths
  - ~/home/walros
  - ~/home/research
```

Per-graph config in `GRAPH.yaml` (placed in any vault subdirectory):

```yaml
name: "Memex"                              # Optional: defaults to directory name
filter: "path:memex/ OR path:z-templates/" # Optional: Obsidian search query (default: show all)
groups:                                    # Optional: color groups, first match wins
  - query: "tag:#open-question"
    color: "#E5A84B"
  - query: "path:memex/concepts"
    color: "#5577CC"
  - query: '[author:"Ali Yahya"]'
    color: "#CC6655"
```

### Filter & Group Query Syntax

Follows [Obsidian search rules](https://obsidian.md/help/plugins/search):

| Operator | Meaning |
|----------|---------|
| `path:VALUE` | File path contains VALUE |
| `tag:#VALUE` | Node has this tag |
| `file:VALUE` | Filename contains VALUE |
| `[field:"value"]` | Frontmatter field match |
| bare text | Title or filename contains text |
| `*` | Match all (default) |
| space | AND (implicit) |
| `OR` | OR |
| `-` prefix | NOT |
| `(...)` | Grouping |

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
| `internal/store` | SQLite data access (nodes, edges, positions, graphs) |
| `internal/indexer` | Multi-vault parsing and database synchronization |
| `internal/discovery` | GRAPH.yaml scanning, nesting validation, graph membership |
| `internal/search` | Obsidian search query parser and evaluator |
| `internal/watcher` | Per-vault fsnotify watcher with debouncing |
| `internal/api` | net/http handlers, SSE, filter/group evaluation, static file serving |
| `internal/vault` | Markdown parser, WikiLink resolver, graph builder |
| `internal/config` | YAML configuration loading |
| `internal/models` | Data structures (VaultNode, VaultEdge, NodePosition, GraphInfo) |

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/graphs` | List all graphs with node counts |
| GET | `/api/v1/graphs/{id}` | Graph data (nodes with colors + edges + positions) |
| GET | `/api/v1/graphs/{id}/search?q=` | Full-text search within a graph |
| PUT | `/api/v1/graphs/{id}/positions` | Batch update positions |
| PUT | `/api/v1/graphs/{id}/positions/{nodeId}` | Update single position |
| GET | `/api/v1/nodes/{id}` | Single node metadata |
| POST | `/api/v1/reindex` | Trigger full re-index of all vaults |
| GET | `/api/v1/events` | SSE stream (graph-updated, graphs-changed) |

## License

MIT License - see LICENSE file for details.
