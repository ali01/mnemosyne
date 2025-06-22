-- Mnemosyne Database Schema
-- PostgreSQL schema for Obsidian vault graph visualization

-- Core tables for graph structure
CREATE TABLE IF NOT EXISTS nodes (
    id VARCHAR(20) PRIMARY KEY,          -- Frontmatter ID from vault
    path TEXT UNIQUE NOT NULL,           -- Relative file path in vault
    title TEXT NOT NULL,                 -- Extracted from filename
    content TEXT,                        -- Raw markdown content
    frontmatter JSONB,                   -- Complete frontmatter as JSON
    node_type VARCHAR(50),               -- index, hub, question, concept, reference, default
    tags TEXT[],                         -- Array of tags from frontmatter
    created_at TIMESTAMP,                -- File creation time
    modified_at TIMESTAMP,               -- File modification time
    parsed_at TIMESTAMP DEFAULT NOW()    -- When we parsed this file
);

CREATE TABLE IF NOT EXISTS edges (
    id SERIAL PRIMARY KEY,
    source_id VARCHAR(20) NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    target_id VARCHAR(20) NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    edge_type VARCHAR(50) DEFAULT 'wikilink',  -- wikilink, embed, reference
    display_text TEXT,                          -- Alias text if present
    weight FLOAT DEFAULT 1.0,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(source_id, target_id, edge_type)
);

CREATE TABLE IF NOT EXISTS node_positions (
    node_id VARCHAR(20) PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
    x FLOAT NOT NULL,
    y FLOAT NOT NULL,
    z FLOAT DEFAULT 0,
    locked BOOLEAN DEFAULT FALSE,        -- User can lock positions
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Metadata and sync tracking
CREATE TABLE IF NOT EXISTS vault_metadata (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS parse_history (
    id SERIAL PRIMARY KEY,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    files_processed INT,
    nodes_created INT,
    edges_created INT,
    errors TEXT[],
    status VARCHAR(50)  -- parsing, completed, failed
);

-- Unresolved links tracking
CREATE TABLE IF NOT EXISTS unresolved_links (
    id SERIAL PRIMARY KEY,
    source_id VARCHAR(20) REFERENCES nodes(id) ON DELETE CASCADE,
    target_text TEXT NOT NULL,           -- The WikiLink text that couldn't be resolved
    link_type VARCHAR(50),               -- wikilink, embed
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_nodes_path ON nodes(path);
CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(node_type);
CREATE INDEX IF NOT EXISTS idx_nodes_tags ON nodes USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
CREATE INDEX IF NOT EXISTS idx_nodes_content_search ON nodes USING GIN(to_tsvector('english', content));
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