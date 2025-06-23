-- Migration: Update schema for VaultNode and VaultEdge models
-- This migration aligns the database schema with the Go models defined in internal/models/vault.go

-- First, let's add missing columns to nodes table
ALTER TABLE nodes 
ADD COLUMN IF NOT EXISTS in_degree INT DEFAULT 0,
ADD COLUMN IF NOT EXISTS out_degree INT DEFAULT 0,
ADD COLUMN IF NOT EXISTS centrality FLOAT DEFAULT 0.0;

-- Update column names to match Go model (file_path instead of path)
ALTER TABLE nodes RENAME COLUMN path TO file_path;
ALTER TABLE nodes RENAME COLUMN modified_at TO updated_at;

-- Update edges table to use UUID for primary key
ALTER TABLE edges DROP CONSTRAINT IF EXISTS edges_pkey CASCADE;
ALTER TABLE edges DROP COLUMN IF EXISTS id;
ALTER TABLE edges ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();

-- Add additional indexes for better query performance
-- Index for filtering by node creation date (for time-based queries)
CREATE INDEX IF NOT EXISTS idx_nodes_created_at ON nodes(created_at);

-- Composite index for edge lookups (both directions)
CREATE INDEX IF NOT EXISTS idx_edges_source_target ON edges(source_id, target_id);

-- Index for edge type queries
CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(edge_type);

-- Index for finding nodes with high centrality (important nodes)
CREATE INDEX IF NOT EXISTS idx_nodes_centrality ON nodes(centrality DESC);

-- Index for degree-based queries
CREATE INDEX IF NOT EXISTS idx_nodes_in_degree ON nodes(in_degree DESC);
CREATE INDEX IF NOT EXISTS idx_nodes_out_degree ON nodes(out_degree DESC);

-- Partial index for hub nodes (frequently accessed)
CREATE INDEX IF NOT EXISTS idx_nodes_hubs ON nodes(id) WHERE node_type = 'hub';

-- Update constraints to match validation rules
ALTER TABLE nodes 
ADD CONSTRAINT check_node_type CHECK (node_type IN ('index', 'hub', 'concept', 'project', 'question', 'note') OR node_type IS NULL),
ADD CONSTRAINT check_centrality CHECK (centrality >= 0 AND centrality <= 1),
ADD CONSTRAINT check_degrees CHECK (in_degree >= 0 AND out_degree >= 0);

ALTER TABLE edges
ADD CONSTRAINT check_edge_type CHECK (edge_type IN ('wikilink', 'embed')),
ADD CONSTRAINT check_weight CHECK (weight >= 0),
ADD CONSTRAINT check_no_self_loops CHECK (source_id != target_id);

-- Add comments for documentation
COMMENT ON TABLE nodes IS 'Vault nodes representing markdown files in the knowledge graph';
COMMENT ON TABLE edges IS 'Connections between nodes (wikilinks and embeds)';

COMMENT ON COLUMN nodes.id IS 'Unique identifier from frontmatter (required)';
COMMENT ON COLUMN nodes.file_path IS 'Original file location in vault';
COMMENT ON COLUMN nodes.node_type IS 'Classification: index, hub, concept, project, question, or note';
COMMENT ON COLUMN nodes.in_degree IS 'Number of incoming links';
COMMENT ON COLUMN nodes.out_degree IS 'Number of outgoing links';
COMMENT ON COLUMN nodes.centrality IS 'PageRank or similar metric (0-1)';

COMMENT ON COLUMN edges.source_id IS 'Node ID of the link source';
COMMENT ON COLUMN edges.target_id IS 'Node ID of the link target';
COMMENT ON COLUMN edges.edge_type IS 'Type of connection: wikilink or embed';