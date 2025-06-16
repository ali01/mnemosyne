# Mnemosyne

A web-based graph visualizer for Obsidian vault concepts, designed to handle large knowledge graphs with up to 50,000 nodes.

## Quick Start

1. **Clone the repository**:
   ```bash
   git clone https://github.com/ali01/mnemosyne.git
   cd mnemosyne
   ```

2. **Start the backend**:
   ```bash
   cd backend
   go mod download
   go run cmd/server/main.go
   ```

3. **Start the frontend** (in a new terminal):
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

4. **Open the visualizer**: Navigate to http://localhost:5173

## Features

- Interactive graph visualization with zoom controls
- Real-time node position updates
- Color-coded nodes by type (Core Concepts, Sub-concepts, Details)
- Sample graph with 30 nodes for testing

## Architecture

- **Backend**: Go + Gin API server with in-memory storage
- **Frontend**: SvelteKit + Sigma.js for graph visualization
- **Data**: Sample graph loaded from `backend/data/sample_graph.json`

See [CLAUDE.md](CLAUDE.md) for detailed development information.