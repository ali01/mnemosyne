DROP TRIGGER IF EXISTS update_nodes_updated_at ON nodes;
DROP FUNCTION IF EXISTS update_updated_at();

DROP TABLE IF EXISTS layout_snapshots;
DROP TABLE IF EXISTS cluster_nodes;
DROP TABLE IF EXISTS clusters;
DROP TABLE IF EXISTS edges;
DROP TABLE IF EXISTS nodes;

DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "uuid-ossp";