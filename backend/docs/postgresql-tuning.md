# PostgreSQL Tuning Guide for Mnemosyne

This guide provides recommended PostgreSQL configuration settings for optimal performance with large Obsidian vaults (50,000+ nodes).

## Required PostgreSQL Version

- **Minimum**: PostgreSQL 11 (for `websearch_to_tsquery` support)
- **Recommended**: PostgreSQL 14 or later

## Configuration Settings

Add these settings to your `postgresql.conf` file:

### Memory Settings

```ini
# Shared memory for caching (25% of system RAM recommended)
shared_buffers = 2GB

# Memory for each query operation
work_mem = 256MB

# Memory for maintenance operations (CREATE INDEX, VACUUM, etc.)
maintenance_work_mem = 512MB

# Memory for autovacuum operations
autovacuum_work_mem = 256MB
```

### Query Optimization

```ini
# Optimize for SSDs (lower = faster random access)
random_page_cost = 1.1

# Enable parallel query execution
max_parallel_workers_per_gather = 4
max_parallel_workers = 8

# Enable JIT compilation for complex queries
jit = on
jit_above_cost = 100000

# Checkpoint settings for write-heavy operations
checkpoint_timeout = 15min
checkpoint_completion_target = 0.9
```

### Connection Pooling

```ini
# Maximum connections (adjust based on your needs)
max_connections = 200

# Enable connection pooling statistics
track_activities = on
track_counts = on
```

### Full-Text Search Optimization

```ini
# Increase default statistics target for better query plans
default_statistics_target = 200

# Enable query plan caching
plan_cache_mode = auto
```

## Required Extensions

Enable these extensions for optimal performance:

```sql
-- For query performance analysis
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- For UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

## Application-Level Settings

The application will automatically:
- Create appropriate indexes for full-text search
- Run ANALYZE after bulk imports
- Use connection pooling

## Monitoring

Monitor these metrics:
- Query execution time (via pg_stat_statements)
- Cache hit ratio (should be > 95%)
- Index usage statistics
- Connection pool efficiency

## Applying Changes

1. Edit `/etc/postgresql/14/main/postgresql.conf` (path may vary)
2. Restart PostgreSQL: `sudo systemctl restart postgresql`
3. Verify settings: `SHOW ALL;` in psql

## Notes

- These settings are optimized for a dedicated database server with at least 8GB RAM
- Adjust `shared_buffers` and `work_mem` based on your available RAM
- Monitor performance and adjust as needed