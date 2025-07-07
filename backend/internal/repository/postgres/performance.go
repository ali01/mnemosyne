// Package postgres provides performance optimization utilities
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// PerformanceOptimizer provides methods to optimize database performance
type PerformanceOptimizer struct {
	db *sqlx.DB
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(database *sqlx.DB) *PerformanceOptimizer {
	return &PerformanceOptimizer{db: database}
}

// OptimizeForLargeVault verifies the database is configured for large vaults (50K+ nodes)
// 
// IMPORTANT: For production use with large vaults, PostgreSQL must be properly configured
// at the server level. Session-level SET commands are not sufficient as they:
// - Only affect the current connection
// - Reset when connections return to the pool
// - Don't persist across application restarts
//
// See backend/docs/postgresql-tuning.md for required postgresql.conf settings.
func (p *PerformanceOptimizer) OptimizeForLargeVault(ctx context.Context) error {
	// Check current settings and log warnings if they're not optimal
	checks := []struct {
		setting      string
		recommended  string
		query        string
	}{
		{
			setting:     "work_mem",
			recommended: "256MB",
			query:       "SHOW work_mem",
		},
		{
			setting:     "shared_buffers",
			recommended: "2GB",
			query:       "SHOW shared_buffers",
		},
		{
			setting:     "random_page_cost",
			recommended: "1.1",
			query:       "SHOW random_page_cost",
		},
	}

	for _, check := range checks {
		var currentValue string
		if err := p.db.GetContext(ctx, &currentValue, check.query); err != nil {
			fmt.Printf("Warning: Could not check %s setting: %v\n", check.setting, err)
			continue
		}
		
		// Log if setting differs from recommended
		if currentValue != check.recommended {
			fmt.Printf("Performance warning: %s is set to '%s', recommended: '%s'\n", 
				check.setting, currentValue, check.recommended)
			fmt.Println("See backend/docs/postgresql-tuning.md for configuration instructions")
		}
	}

	// Ensure required extensions are available
	extensions := []string{"uuid-ossp"}
	for _, ext := range extensions {
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)`
		if err := p.db.GetContext(ctx, &exists, query, ext); err != nil {
			return fmt.Errorf("failed to check for extension %s: %w", ext, err)
		}
		if !exists {
			fmt.Printf("Warning: Required extension '%s' is not installed\n", ext)
		}
	}

	return nil
}

// AnalyzeTables updates table statistics for better query planning
func (p *PerformanceOptimizer) AnalyzeTables(ctx context.Context) error {
	tables := []string{"nodes", "edges", "node_positions", "parse_history", "vault_metadata"}
	
	for _, table := range tables {
		if _, err := p.db.ExecContext(ctx, fmt.Sprintf("ANALYZE %s", table)); err != nil {
			return fmt.Errorf("failed to analyze %s: %w", table, err)
		}
	}
	
	return nil
}

// VacuumTables performs maintenance on tables to reclaim space
func (p *PerformanceOptimizer) VacuumTables(ctx context.Context) error {
	tables := []string{"nodes", "edges", "node_positions", "parse_history", "vault_metadata"}
	
	for _, table := range tables {
		if _, err := p.db.ExecContext(ctx, fmt.Sprintf("VACUUM ANALYZE %s", table)); err != nil {
			return fmt.Errorf("failed to vacuum %s: %w", table, err)
		}
	}
	
	return nil
}

// GetQueryStats returns statistics about slow queries
func (p *PerformanceOptimizer) GetQueryStats(ctx context.Context) ([]QueryStat, error) {
	query := `
		SELECT 
			query,
			calls,
			total_time,
			mean_time,
			stddev_time,
			rows
		FROM pg_stat_statements
		WHERE query NOT LIKE '%pg_stat_statements%'
		ORDER BY mean_time DESC
		LIMIT 20
	`

	var stats []QueryStat
	err := p.db.SelectContext(ctx, &stats, query)
	if err != nil {
		// pg_stat_statements extension might not be enabled
		return nil, nil
	}

	return stats, nil
}

// QueryStat represents query performance statistics
type QueryStat struct {
	Query      string  `db:"query"`
	Calls      int64   `db:"calls"`
	TotalTime  float64 `db:"total_time"`
	MeanTime   float64 `db:"mean_time"`
	StddevTime float64 `db:"stddev_time"`
	Rows       int64   `db:"rows"`
}

// CreateMaterializedViews creates materialized views for expensive queries
func (p *PerformanceOptimizer) CreateMaterializedViews(ctx context.Context) error {
	views := []struct {
		name  string
		query string
	}{
		{
			name: "node_graph_metrics",
			query: `
				CREATE MATERIALIZED VIEW IF NOT EXISTS node_graph_metrics AS
				SELECT 
					n.id,
					n.title,
					n.node_type,
					COUNT(DISTINCT e_out.id) as out_degree_calc,
					COUNT(DISTINCT e_in.id) as in_degree_calc,
					COUNT(DISTINCT e_out.id) + COUNT(DISTINCT e_in.id) as total_degree
				FROM nodes n
				LEFT JOIN edges e_out ON n.id = e_out.source_id
				LEFT JOIN edges e_in ON n.id = e_in.target_id
				GROUP BY n.id, n.title, n.node_type
			`,
		},
		{
			name: "node_type_stats",
			query: `
				CREATE MATERIALIZED VIEW IF NOT EXISTS node_type_stats AS
				SELECT 
					node_type,
					COUNT(*) as node_count,
					AVG(in_degree) as avg_in_degree,
					AVG(out_degree) as avg_out_degree,
					AVG(centrality) as avg_centrality
				FROM nodes
				GROUP BY node_type
			`,
		},
	}

	for _, view := range views {
		if _, err := p.db.ExecContext(ctx, view.query); err != nil {
			return fmt.Errorf("failed to create view %s: %w", view.name, err)
		}
		
		// Create index on materialized view
		indexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_id ON %s(id)", view.name, view.name)
		if _, err := p.db.ExecContext(ctx, indexQuery); err != nil {
			return fmt.Errorf("failed to create index on view %s: %w", view.name, err)
		}
	}

	return nil
}

// RefreshMaterializedViews refreshes all materialized views
func (p *PerformanceOptimizer) RefreshMaterializedViews(ctx context.Context) error {
	views := []string{"node_graph_metrics", "node_type_stats"}
	
	for _, view := range views {
		if _, err := p.db.ExecContext(ctx, fmt.Sprintf("REFRESH MATERIALIZED VIEW CONCURRENTLY %s", view)); err != nil {
			// Try without CONCURRENTLY if it fails
			if _, err := p.db.ExecContext(ctx, fmt.Sprintf("REFRESH MATERIALIZED VIEW %s", view)); err != nil {
				return fmt.Errorf("failed to refresh view %s: %w", view, err)
			}
		}
	}
	
	return nil
}

// QueryOptimizer provides query optimization hints
type QueryOptimizer struct {
	batchSize int
}

// NewQueryOptimizer creates a new query optimizer with default settings
func NewQueryOptimizer() *QueryOptimizer {
	return &QueryOptimizer{
		batchSize: 1000, // Default batch size for operations
	}
}

// GetOptimalBatchSize returns the optimal batch size based on total items
func (q *QueryOptimizer) GetOptimalBatchSize(totalItems int) int {
	if totalItems < 100 {
		return totalItems
	}
	if totalItems < 1000 {
		return 100
	}
	if totalItems < 10000 {
		return 500
	}
	return q.batchSize
}

// ShouldUseTransaction determines if an operation should use a transaction
func (q *QueryOptimizer) ShouldUseTransaction(operationCount int) bool {
	return operationCount > 1
}

// EstimateQueryTime estimates query execution time based on complexity
func (q *QueryOptimizer) EstimateQueryTime(nodeCount, edgeCount int64) time.Duration {
	// Simple heuristic based on graph size
	baseTime := 10 * time.Millisecond
	nodeTime := time.Duration(nodeCount) * 10 * time.Microsecond
	edgeTime := time.Duration(edgeCount) * 5 * time.Microsecond
	
	return baseTime + nodeTime + edgeTime
}