-- Mnemosyne Database Schema
-- PostgreSQL schema for Obsidian vault graph visualization

-- Enable UUID extension for edge IDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Core tables for graph structure
CREATE TABLE IF NOT EXISTS nodes (
    id VARCHAR(20) PRIMARY KEY,              -- Frontmatter ID from vault
    file_path TEXT UNIQUE NOT NULL,          -- Relative file path in vault
    title TEXT NOT NULL,                     -- Extracted from filename or frontmatter
    content TEXT,                            -- Raw markdown content
    frontmatter JSONB,                       -- Complete frontmatter as JSON
    node_type VARCHAR(50),                   -- index, hub, question, concept, project, note, reference, default
    tags TEXT[],                             -- Array of tags from frontmatter
    in_degree INT DEFAULT 0,                 -- Number of incoming links
    out_degree INT DEFAULT 0,                -- Number of outgoing links
    centrality FLOAT DEFAULT 0.0,            -- PageRank or similar metric (0-1)
    created_at TIMESTAMP,                    -- File creation time
    updated_at TIMESTAMP,                    -- File modification time
    parsed_at TIMESTAMP DEFAULT NOW(),       -- When we parsed this file
    search_vector tsvector GENERATED ALWAYS AS (to_tsvector('english', COALESCE(title, '') || ' ' || COALESCE(content, ''))) STORED,
    CONSTRAINT check_node_type CHECK (
        node_type IN ('index', 'hub', 'concept', 'project', 'question', 'note', 'reference', 'default')
        OR node_type IS NULL
    ),
    CONSTRAINT check_centrality CHECK (centrality >= 0 AND centrality <= 1),
    CONSTRAINT check_degrees CHECK (in_degree >= 0 AND out_degree >= 0)
);

CREATE TABLE IF NOT EXISTS edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id VARCHAR(20) NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    target_id VARCHAR(20) NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    edge_type VARCHAR(50) NOT NULL DEFAULT 'wikilink',
    display_text TEXT,                           -- Alias text if present
    weight FLOAT DEFAULT 1.0,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(source_id, target_id, edge_type),
    CONSTRAINT check_edge_type CHECK (edge_type IN ('wikilink', 'embed')),
    CONSTRAINT check_weight CHECK (weight >= 0),
    CONSTRAINT check_no_self_loops CHECK (source_id != target_id)
);

CREATE TABLE IF NOT EXISTS node_positions (
    node_id VARCHAR(20) PRIMARY KEY,         -- No foreign key to allow positions to persist
    x FLOAT NOT NULL,
    y FLOAT NOT NULL,
    z FLOAT DEFAULT 0,
    locked BOOLEAN DEFAULT FALSE,            -- User can lock positions
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Metadata and sync tracking
CREATE TABLE IF NOT EXISTS vault_metadata (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS parse_history (
    id VARCHAR(100) PRIMARY KEY,             -- String ID (UUID or custom)
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    status VARCHAR(50) NOT NULL,             -- pending, running, completed, failed
    stats JSONB,                             -- ParseStats as JSON object
    error TEXT,                              -- Error message if failed
    CONSTRAINT check_status CHECK (status IN ('pending', 'running', 'completed', 'failed'))
);

-- Unresolved links tracking
CREATE TABLE IF NOT EXISTS unresolved_links (
    id SERIAL PRIMARY KEY,
    source_id VARCHAR(20) REFERENCES nodes(id) ON DELETE CASCADE,
    target_text TEXT NOT NULL,               -- The WikiLink text that couldn't be resolved
    link_type VARCHAR(50),                   -- wikilink, embed
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
--
-- IMPORTANT: For production deployments with existing data:
-- 1. Use CREATE INDEX CONCURRENTLY to avoid table locks:
--    CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_name ON table(column);
-- 2. Run ANALYZE after creating indexes to update query planner statistics:
--    ANALYZE nodes; ANALYZE edges;
-- 3. For initial schema creation on empty database, regular CREATE INDEX is fine
--
-- Note: CONCURRENTLY cannot be used within a transaction block
--
-- Index Design Notes:
-- - created_at indexes use DESC to optimize common "ORDER BY created_at DESC" queries
-- - file_path has UNIQUE constraint at table level, which creates a unique index
-- - All foreign key columns have indexes to speed up JOINs and CASCADE operations
CREATE INDEX IF NOT EXISTS idx_nodes_file_path ON nodes(file_path);
CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(node_type);
CREATE INDEX IF NOT EXISTS idx_nodes_tags ON nodes USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_nodes_created_at ON nodes(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_nodes_centrality ON nodes(centrality DESC);
CREATE INDEX IF NOT EXISTS idx_nodes_in_degree ON nodes(in_degree DESC);
CREATE INDEX IF NOT EXISTS idx_nodes_out_degree ON nodes(out_degree DESC);
CREATE INDEX IF NOT EXISTS idx_nodes_hubs ON nodes(id) WHERE node_type = 'hub';

CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
CREATE INDEX IF NOT EXISTS idx_edges_source_target ON edges(source_id, target_id);
CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(edge_type);
CREATE INDEX IF NOT EXISTS idx_edges_created_at ON edges(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_nodes_search ON nodes USING GIN(search_vector);

CREATE INDEX IF NOT EXISTS idx_parse_history_started_at ON parse_history(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_parse_history_status ON parse_history(status);

CREATE INDEX IF NOT EXISTS idx_unresolved_source ON unresolved_links(source_id);

-- Update trigger for node_positions
CREATE OR REPLACE FUNCTION update_node_position_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_node_position_timestamp
BEFORE UPDATE ON node_positions
FOR EACH ROW
EXECUTE FUNCTION update_node_position_timestamp();

-- Comments for documentation
COMMENT ON TABLE nodes IS 'Vault nodes representing markdown files in the knowledge graph';
COMMENT ON TABLE edges IS 'Connections between nodes (wikilinks and embeds)';
COMMENT ON TABLE parse_history IS 'History of vault parsing operations';

COMMENT ON COLUMN nodes.id IS 'Unique identifier from frontmatter (required)';
COMMENT ON COLUMN nodes.file_path IS 'Original file location in vault';
COMMENT ON COLUMN nodes.node_type IS 'Classification: index, hub, concept, project, question, note, reference, or default';
COMMENT ON COLUMN nodes.in_degree IS 'Number of incoming links';
COMMENT ON COLUMN nodes.out_degree IS 'Number of outgoing links';
COMMENT ON COLUMN nodes.centrality IS 'PageRank or similar metric (0-1)';
COMMENT ON COLUMN nodes.search_vector IS 'Pre-computed text search vector for title and content';

COMMENT ON COLUMN edges.id IS 'Auto-generated UUID';
COMMENT ON COLUMN edges.source_id IS 'Node ID of the link source';
COMMENT ON COLUMN edges.target_id IS 'Node ID of the link target';
COMMENT ON COLUMN edges.edge_type IS 'Type of connection: wikilink or embed';

COMMENT ON COLUMN parse_history.id IS 'Unique identifier (UUID or custom string)';
COMMENT ON COLUMN parse_history.stats IS 'JSON object containing ParseStats: total_files, parsed_files, total_nodes, total_edges, duration, unresolved_links';
COMMENT ON COLUMN parse_history.error IS 'Error message if parsing failed';

-- Update statistics for query planner after initial data load
-- Run these commands after populating the database:
-- ANALYZE nodes;
-- ANALYZE edges;
-- ANALYZE node_positions;
-- ANALYZE parse_history;
