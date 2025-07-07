# Production Deployment Guide

This guide covers best practices for deploying Mnemosyne to production, particularly focusing on database schema creation and index management for large-scale deployments.

## Database Schema Deployment

### Initial Deployment (Empty Database)

For a fresh installation with no existing data:

```bash
# Create the database
createdb mnemosyne

# Apply the schema
psql mnemosyne < backend/internal/db/schema.sql
```

The schema file uses `CREATE INDEX IF NOT EXISTS` which is safe for initial deployments.

### Adding Indexes to Existing Production Database

When adding indexes to a production database with existing data, use `CREATE INDEX CONCURRENTLY` to avoid table locks:

```sql
-- Example: Adding a new index without blocking reads/writes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_nodes_new_column
  ON nodes(new_column);

-- After creating indexes, update statistics
ANALYZE nodes;
```

**Important Notes:**
- `CONCURRENTLY` cannot be used within a transaction
- The operation takes longer but doesn't block table access
- Monitor for long-running queries before adding indexes
- Consider doing this during low-traffic periods

### Large-Scale Deployment Considerations

For deployments targeting 50,000+ nodes:

1. **Set appropriate lock timeouts** to prevent long-running operations:
   ```sql
   SET lock_timeout = '5s';
   ```

2. **Create indexes before bulk data import** when possible

3. **Use parallel index creation** (PostgreSQL 11+):
   ```sql
   SET max_parallel_maintenance_workers = 4;
   ```

4. **Update table statistics** after major data changes:
   ```sql
   -- Update statistics for all tables
   ANALYZE nodes;
   ANALYZE edges;
   ANALYZE node_positions;
   ANALYZE parse_history;

   -- Or analyze entire database
   ANALYZE;
   ```

## Performance Optimization

### Query Planning

The search vector column uses a generated column for optimal performance:
- Pre-computed at write time
- Includes both title and content
- GIN indexed for fast full-text search

### Index Usage

The schema includes indexes optimized for common query patterns:
- `idx_edges_source` and `idx_edges_target` for UNION queries
- `idx_nodes_search` for full-text search
- Various indexes on foreign keys for JOIN performance

### Connection Pooling

Configure appropriate connection pool settings:
```yaml
database:
  max_connections: 25
  max_idle_connections: 5
  connection_max_lifetime: 1h
```


## Monitoring

Monitor these key metrics:
- Index usage: `pg_stat_user_indexes`
- Slow queries: Enable `log_min_duration_statement`
- Table bloat: Use `pg_stat_user_tables`
- Connection count: Monitor against `max_connections`

## Backup Strategy

1. **Regular backups**: Use `pg_dump` with custom format:
   ```bash
   pg_dump -Fc mnemosyne > mnemosyne_$(date +%Y%m%d).dump
   ```

2. **Point-in-time recovery**: Enable WAL archiving for critical deployments

3. **Test restores**: Regularly verify backup integrity

## Migration Strategy

For schema updates in production:

1. **Test migrations** on a staging environment first
2. **Use reversible migrations** when possible
3. **Plan for rollback** scenarios
4. **Consider blue-green deployments** for zero-downtime updates

## Security Considerations

1. **Use separate roles** for application and admin access
2. **Enable SSL** for database connections
3. **Restrict network access** to database port
4. **Regular security updates** for PostgreSQL
