---
id: "project-mnemosyne"
tags: ["project", "active"]
related: ["graph-visualization", "obsidian", "knowledge-management"]
---
# Mnemosyne Project

A web-based graph visualizer for Obsidian vaults.

## Overview
Mnemosyne visualizes the connections between notes in an [[Obsidian]] vault as an interactive graph.

## Technical Stack
- Backend: [[Go]] with [[Gin Framework]]
- Frontend: [[SvelteKit]] with [[Sigma.js]]
- Database: [[PostgreSQL]]

## Key Features
- Handles up to 50,000 nodes
- Force-directed layout using [[concepts/Network#Graph Layout Algorithms|graph algorithms]]
- Real-time updates via [[WebSockets]]

## Related Concepts
- [[Graph Theory]] - Mathematical foundation
- [[concepts/~AI#Force-Directed Layouts]] - AI-based positioning