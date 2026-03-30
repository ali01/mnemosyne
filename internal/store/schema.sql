-- Mnemosyne SQLite Schema

CREATE TABLE IF NOT EXISTS vaults (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    path TEXT UNIQUE NOT NULL,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    vault_id INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    frontmatter TEXT,          -- JSON object stored as text
    node_type TEXT,
    tags TEXT,                 -- JSON array stored as text
    in_degree INTEGER DEFAULT 0,
    out_degree INTEGER DEFAULT 0,
    created_at TEXT,
    updated_at TEXT,
    parsed_at TEXT DEFAULT (datetime('now')),
    UNIQUE(vault_id, file_path)
);

CREATE TABLE IF NOT EXISTS edges (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    edge_type TEXT NOT NULL DEFAULT 'wikilink',
    display_text TEXT,
    weight REAL DEFAULT 1.0,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(source_id, target_id, edge_type)
);

CREATE TABLE IF NOT EXISTS graphs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    vault_id INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL DEFAULT '',  -- relative to vault, '' for vault root
    config TEXT,                          -- GRAPH.yaml contents as JSON
    archived INTEGER NOT NULL DEFAULT 0, -- 1 = GRAPH.yaml deleted, graph hidden but maintained
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    UNIQUE(vault_id, root_path)
);

CREATE TABLE IF NOT EXISTS graph_nodes (
    graph_id INTEGER NOT NULL REFERENCES graphs(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    PRIMARY KEY (graph_id, node_id)
);

CREATE TABLE IF NOT EXISTS node_positions (
    graph_id INTEGER NOT NULL REFERENCES graphs(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    x REAL NOT NULL DEFAULT 0,
    y REAL NOT NULL DEFAULT 0,
    z REAL DEFAULT 0,
    locked INTEGER DEFAULT 0,
    updated_at TEXT DEFAULT (datetime('now')),
    PRIMARY KEY (graph_id, node_id)
);

CREATE TABLE IF NOT EXISTS vault_metadata (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at TEXT DEFAULT (datetime('now'))
);

-- FTS5 virtual table for full-text search
CREATE VIRTUAL TABLE IF NOT EXISTS nodes_fts USING fts5(
    title,
    content,
    content=nodes,
    content_rowid=rowid
);

-- Triggers to keep FTS in sync
CREATE TRIGGER IF NOT EXISTS nodes_fts_insert AFTER INSERT ON nodes BEGIN
    INSERT INTO nodes_fts(rowid, title, content)
    VALUES (new.rowid, new.title, new.content);
END;

CREATE TRIGGER IF NOT EXISTS nodes_fts_delete AFTER DELETE ON nodes BEGIN
    INSERT INTO nodes_fts(nodes_fts, rowid, title, content)
    VALUES ('delete', old.rowid, old.title, old.content);
END;

CREATE TRIGGER IF NOT EXISTS nodes_fts_update AFTER UPDATE ON nodes BEGIN
    INSERT INTO nodes_fts(nodes_fts, rowid, title, content)
    VALUES ('delete', old.rowid, old.title, old.content);
    INSERT INTO nodes_fts(rowid, title, content)
    VALUES (new.rowid, new.title, new.content);
END;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_nodes_vault ON nodes(vault_id);
CREATE INDEX IF NOT EXISTS idx_nodes_file_path ON nodes(file_path);
CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(node_type);
CREATE INDEX IF NOT EXISTS idx_nodes_created_at ON nodes(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
CREATE INDEX IF NOT EXISTS idx_edges_source_target ON edges(source_id, target_id);
CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(edge_type);

CREATE INDEX IF NOT EXISTS idx_graph_nodes_node ON graph_nodes(node_id);
