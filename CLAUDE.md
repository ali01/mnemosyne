# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Mnemosyne** - A web-based graph visualizer for Obsidian vault concepts, designed to handle large knowledge graphs with up to 50,000 nodes.

## Architecture

### Backend (Go + Gin)
- **API Server**: RESTful API at `localhost:8080`
- **Data Storage**: In-memory with sample data from `backend/data/sample_graph.json`
- **Key Directories**:
  - `backend/cmd/server/` - Main server entry point
  - `backend/internal/api/` - HTTP handlers and routes
  - `backend/internal/models/` - Data models
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
go run cmd/server/main.go

# Frontend (in another terminal)
cd frontend
npm install
npm run dev
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
4. **Read-Only Vault Access**: The `walros-obsidian` directory is read-only reference data

## Future Enhancements
- Add persistent storage for node positions
- Implement graph clustering algorithms
- Add Obsidian vault file parsing
- Implement viewport-based loading for large graphs