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
