# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Mnemosyne** - A web-based graph visualizer for Obsidian vault concepts, designed to handle large knowledge graphs with up to 50,000 nodes.

## Architecture

### Backend (Go + Gin)
- **API Server**: RESTful API at `localhost:8080`
- **Data Storage**: In-memory with sample data from `backend/data/sample_graph.json`
- **Git Integration**: Clones and syncs Obsidian vaults from GitHub
- **Key Directories**:
  - `backend/cmd/server/` - Main server entry point
  - `backend/internal/api/` - HTTP handlers and routes
  - `backend/internal/models/` - Data models
  - `backend/internal/git/` - Git repository management
  - `backend/data/` - Sample graph data

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

go run cmd/server/main.go

# Frontend (in another terminal)
cd frontend
npm install
npm run dev
```

### Testing Git Integration
```bash
cd backend
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

## Key Design Decisions

1. **Simplified Architecture**: No database dependencies for easy setup and development
2. **In-Memory Storage**: Node positions are stored in memory during runtime
3. **Sample Data**: Graph data loaded from JSON file for testing
4. **Read-Only Vault Access**: The vault repository is cloned as read-only reference data
5. **Git Clone Strategy**: Using local Git clone for GitHub-hosted vaults (better performance, no rate limits)
6. **YAML Configuration**: Uses `config.yaml` instead of environment variables for cleaner configuration management

## Obsidian Vault Integration Plan

### Data Source Strategy: Git Clone
We're using the Git Clone approach for accessing GitHub-hosted Obsidian vaults:

**Advantages:**
- No API rate limits (can process 50k+ files efficiently)
- Fast local file reads after initial clone
- Full text search and analysis capabilities
- Offline operation support
- Simpler implementation

### Step-by-Step Implementation Plan

#### Phase 1: Git Integration Setup ✅ COMPLETED
1. **Add Git dependencies**
   ```bash
   go get github.com/go-git/go-git/v5
   go get gopkg.in/yaml.v3
   ```

2. **Create Git manager** (`backend/internal/git/`)
   - ✅ `manager.go`: Git operations (clone, pull, sync)
   - ✅ `config.go`: YAML configuration loading
   - ✅ `errors.go`: Error definitions
   - ✅ Thread-safe operations with mutex locks
   - ✅ Automatic retry and error handling
   - ✅ Force pull for read-only vault access

3. **Configuration setup**
   - ✅ YAML-based configuration (`config.yaml`)
   - ✅ Example configuration provided
   - ✅ Support for SSH authentication
   - ✅ Configurable sync intervals
   - ✅ Test program to verify Git operations

4. **Current limitations**
   - ⚠️ No real-time updates (relies on periodic sync)
   - ⚠️ No webhook support yet for instant GitHub updates

#### Phase 2: Vault Parser Implementation
1. **Create vault parser package** (`backend/internal/vault/`)
   - `parser.go`: Main parsing orchestrator
   - `markdown.go`: Markdown file processor
   - `wikilink.go`: WikiLink extraction (`[[Note]]` patterns)
   - `frontmatter.go`: YAML frontmatter parser

2. **WikiLink extraction patterns to handle**
   - Basic: `[[Note Name]]`
   - With alias: `[[Note Name|Display Text]]`
   - With heading: `[[Note Name#Heading]]`
   - Embeds: `![[Image.png]]` or `![[Another Note]]`
   - Full paths: `[[concepts/Network]]`

3. **File path resolution**
   - Map WikiLink text to actual file paths
   - Handle case-insensitive matching
   - Support both basename and full path references
   - Track unresolved links for broken reference detection

#### Phase 3: Graph Construction
1. **Data model implementation** (`backend/internal/models/vault.go`)
   ```go
   type VaultNode struct {
       ID          string              // File path as ID
       Title       string              // From filename or frontmatter
       Path        string              // Relative path in vault
       Content     string              // Raw markdown
       Frontmatter map[string]any      // Parsed YAML
       OutLinks    []string            // WikiLink targets
       InLinks     []string            // Backlinks
       Tags        []string            // From frontmatter
       NodeType    string              // Derived from tags/path
       CreatedAt   time.Time
       ModifiedAt  time.Time
   }
   ```

2. **Graph builder** (`backend/internal/vault/graph_builder.go`)
   - Convert vault files to graph nodes
   - Create edges from WikiLinks
   - Calculate node types:
     - Index nodes: files tagged with `index`
     - Hub nodes: files with `~` prefix
     - Question nodes: files tagged with `open-question`
   - Assign initial positions (force-directed layout)

3. **Graph metrics calculation**
   - Node degree (in/out connections)
   - Centrality scores
   - Connected components
   - Clustering coefficient

#### Phase 4: API Integration
1. **Update graph handler** (`backend/internal/api/graph_handlers.go`)
   - Replace `ReadFile("sample_graph.json")` with vault parser
   - Add caching layer for parsed graph
   - Implement incremental updates

2. **New endpoints**
   - `GET /api/v1/vault/status` - Clone/sync status
   - `POST /api/v1/vault/sync` - Trigger manual sync
   - `GET /api/v1/nodes/:id/content` - Get actual markdown
   - `GET /api/v1/search?q=term` - Search vault content

3. **Response format**
   ```json
   {
     "nodes": [{
       "id": "concepts/Network",
       "title": "Network",
       "position": {"x": 100, "y": 200},
       "type": "concept",
       "level": 1,
       "metadata": {
         "tags": ["index"],
         "inDegree": 5,
         "outDegree": 3
       }
     }],
     "edges": [{
       "id": "e1",
       "source": "concepts/Network",
       "target": "concepts/ai/~AI",
       "type": "wikilink",
       "weight": 1.0
     }]
   }
   ```

#### Phase 5: Caching & Performance
1. **Multi-level cache**
   - In-memory cache for hot data (LRU)
   - Disk cache for parsed graph
   - Cache invalidation on Git pull

2. **Incremental parsing**
   - Track file modifications since last parse
   - Only reparse changed files
   - Update graph incrementally

3. **Background workers**
   - Git sync worker (5-minute intervals)
   - Graph metrics calculator
   - Cache warmer

#### Phase 6: File Watching & Live Updates
1. **File system watcher**
   - Watch cloned repository for changes
   - Detect file modifications/additions/deletions
   - Trigger incremental parsing

2. **WebSocket notifications** (future)
   - Notify frontend of graph updates
   - Send specific node/edge changes
   - Enable real-time collaboration

### Vault Data Structure
- **Nodes**: Each markdown file becomes a node
- **Edges**: WikiLinks (`[[Note Name]]`) create directed edges
- **Metadata**: YAML frontmatter provides tags, references, related notes
- **Special nodes**: 
  - Index nodes (tagged with `index`) - higher visual prominence
  - Topic overviews (prefix `~`) - central hub positioning
  - Open questions (tagged with `open-question`) - different color

### Directory Structure
```
backend/
├── cmd/
│   ├── server/             # Main server entry point
│   └── test-git/           # Git integration test program
├── config.example.yaml     # Example configuration file
├── internal/
│   ├── api/                # HTTP handlers and routes
│   ├── git/                # Git integration (COMPLETED)
│   │   ├── config.go       # YAML configuration & loading
│   │   ├── errors.go       # Error definitions
│   │   └── manager.go      # Git operations (clone, pull, sync)
│   ├── models/             # Data models
│   └── vault/              # Vault parser (TO BE IMPLEMENTED)
│       ├── parser.go       # Main parser
│       ├── markdown.go     # Markdown processor
│       ├── wikilink.go     # Link extraction
│       ├── frontmatter.go  # YAML parsing
│       └── graph_builder.go # Graph construction
└── data/
    └── sample_graph.json   # Sample data for testing
```

### Testing Strategy
1. Unit tests for WikiLink extraction
2. Integration tests with sample vault
3. Performance tests with large vaults (50k+ files)
4. E2E tests for API endpoints

## Current Status

### ✅ Completed: Phase 1 - Git Integration
- Git manager with clone, pull, and sync capabilities
- YAML-based configuration system
- SSH authentication support
- Automatic background sync with configurable intervals
- Thread-safe operations
- Test program for verification

### ⚠️ Current Limitations
- No real-time updates (relies on periodic sync every 5 minutes)
- No webhook support yet for instant GitHub updates

### 🚧 Next Steps
- Phase 2: Implement vault parser to extract nodes and edges from markdown files
- Phase 3: Build graph structure from parsed vault
- Phase 4: Update API to serve real vault data instead of sample JSON
- Phase 5: Add caching and performance optimizations
- Phase 6: Implement webhooks for real-time updates

### 📋 Future Enhancements
- Add persistent storage for node positions
- Implement graph clustering algorithms
- Implement viewport-based loading for large graphs
- Implement search functionality across vault content