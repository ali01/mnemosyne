# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Mnemosyne** - A web-based graph visualizer for Obsidian vault concepts, designed to handle up to 50,000 nodes with intelligent clustering and zoom-level abstractions.

## Architecture

### Backend (Go + Gin)
- **API Server**: RESTful API at `localhost:8080`
- **Database**: PostgreSQL with graph-optimized schema
- **Cache**: Redis for computed layouts and session state
- **Key Directories**:
  - `backend/cmd/server/` - Main server entry point
  - `backend/internal/api/` - HTTP handlers and routes
  - `backend/internal/graph/` - Graph algorithms and clustering
  - `backend/internal/storage/` - Database and cache interfaces

### Frontend (SvelteKit + Sigma.js)
- **Framework**: SvelteKit with TypeScript
- **Graph Rendering**: Sigma.js with WebGL
- **Client Storage**: IndexedDB for offline caching
- **Key Directories**:
  - `frontend/src/routes/` - Page components
  - `frontend/src/lib/components/` - Reusable components
  - `frontend/src/lib/stores/` - State management

## Commands

### Development Setup
```bash
# Start database and cache
docker-compose up -d

# Backend
cd backend
go mod download
go run cmd/server/main.go

# Frontend (in another terminal)
cd frontend
npm install
npm run dev
```

### Database
```bash
# Run migrations (requires golang-migrate)
migrate -path database/migrations -database "postgresql://mnemosyne:mnemosyne@localhost:5432/mnemosyne?sslmode=disable" up

# Connect to database
docker exec -it mnemosyne-postgres psql -U mnemosyne
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

1. **Graph Clustering**: Multi-level clustering computed server-side for performance
2. **Viewport Loading**: Dynamic node loading based on zoom level and viewport bounds
3. **Position Persistence**: Node positions stored in PostgreSQL with JSONB
4. **Read-Only Vault Access**: The `walros-obsidian` directory is read-only reference data

## Environment Variables
Copy `.env.example` to `.env` and configure as needed.