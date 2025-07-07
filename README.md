# Mnemosyne

A web-based graph visualizer for Obsidian vault concepts, designed to handle large knowledge graphs with up to 50,000 nodes.

## Features

- **Vault Parsing**: Complete Obsidian vault support with WikiLink resolution
- **Interactive Visualization**: Sigma.js-powered graph with zoom, pan, and node manipulation
- **Git Integration**: Automatic cloning and syncing of GitHub-hosted vaults
- **High Performance**: Sub-millisecond parsing, handles thousands of files efficiently
- **PostgreSQL Storage**: Persistent storage for nodes, edges, and custom layouts

## Quick Start

### Prerequisites
- Go 1.23+
- Node.js 20+
- PostgreSQL 15+

### Installation

```bash
# Clone repository
git clone https://github.com/ali01/mnemosyne.git
cd mnemosyne

# Setup database
createdb mnemosyne

# Configure backend
cd backend
cp config.example.yaml config.yaml
# Edit config.yaml with your Obsidian vault repository

# Start backend
go run cmd/server/main.go

# Start frontend (new terminal)
cd frontend
npm install
npm run dev
```

Visit http://localhost:5173 to view the graph visualizer.

## Architecture Overview

**Backend**: Go server with Gin framework, PostgreSQL database, and Git integration for vault synchronization.

**Frontend**: SvelteKit application with Sigma.js for high-performance graph rendering using WebGL.

## Documentation

- **[CLAUDE.md](CLAUDE.md)**: Detailed development guide, architecture, and current implementation status
- **[ROADMAP.md](ROADMAP.md)**: Implementation phases and future development plans
- **[BENCHMARKS.md](BENCHMARKS.md)**: Performance metrics and optimization guidelines

## License

MIT License - see LICENSE file for details.
