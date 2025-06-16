-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Nodes table
CREATE TABLE nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    file_path TEXT UNIQUE NOT NULL,
    content TEXT,
    position JSONB NOT NULL DEFAULT '{"x": 0, "y": 0}',
    cluster_id UUID,
    level INTEGER NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Edges table
CREATE TABLE edges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    target UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    weight DECIMAL(10,4) DEFAULT 1.0,
    type VARCHAR(50) NOT NULL DEFAULT 'reference',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source, target, type)
);

-- Clusters table
CREATE TABLE clusters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    level INTEGER NOT NULL,
    center_node UUID REFERENCES nodes(id) ON DELETE SET NULL,
    node_count INTEGER NOT NULL DEFAULT 0,
    position JSONB NOT NULL DEFAULT '{"x": 0, "y": 0}',
    radius DECIMAL(10,4) NOT NULL DEFAULT 0,
    computed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Cluster memberships
CREATE TABLE cluster_nodes (
    cluster_id UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    PRIMARY KEY (cluster_id, node_id)
);

-- Layout snapshots for different zoom levels
CREATE TABLE layout_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    level INTEGER NOT NULL,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_nodes_position ON nodes USING GIN (position);
CREATE INDEX idx_nodes_cluster_id ON nodes(cluster_id);
CREATE INDEX idx_nodes_level ON nodes(level);
CREATE INDEX idx_nodes_file_path ON nodes(file_path);
CREATE INDEX idx_nodes_title_trgm ON nodes USING GIN (title gin_trgm_ops);

CREATE INDEX idx_edges_source ON edges(source);
CREATE INDEX idx_edges_target ON edges(target);
CREATE INDEX idx_edges_type ON edges(type);

CREATE INDEX idx_clusters_level ON clusters(level);
CREATE INDEX idx_cluster_nodes_node ON cluster_nodes(node_id);

-- Updated timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_nodes_updated_at BEFORE UPDATE ON nodes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();