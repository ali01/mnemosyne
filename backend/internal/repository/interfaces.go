// Package repository defines interfaces for data persistence operations
package repository

import (
	"context"

	"github.com/ali01/mnemosyne/internal/models"
)

// NodeRepository defines operations for VaultNode persistence.
// All methods accept an Executor as the first parameter, which can be
// either a database connection (*sqlx.DB) or a transaction (*sqlx.Tx).
// This allows the same repository to work in both transactional and
// non-transactional contexts.
type NodeRepository interface {
	// Basic CRUD operations
	Create(exec Executor, ctx context.Context, node *models.VaultNode) error
	GetByID(exec Executor, ctx context.Context, id string) (*models.VaultNode, error)
	Update(exec Executor, ctx context.Context, node *models.VaultNode) error
	Delete(exec Executor, ctx context.Context, id string) error

	// Batch operations for performance
	CreateBatch(exec Executor, ctx context.Context, nodes []models.VaultNode) error
	UpsertBatch(exec Executor, ctx context.Context, nodes []models.VaultNode) error

	// Query operations
	GetAll(exec Executor, ctx context.Context, limit, offset int) ([]models.VaultNode, error)
	GetByIDs(exec Executor, ctx context.Context, ids []string) ([]models.VaultNode, error)
	GetByType(exec Executor, ctx context.Context, nodeType string) ([]models.VaultNode, error)
	GetByPath(exec Executor, ctx context.Context, path string) (*models.VaultNode, error)
	Search(exec Executor, ctx context.Context, query string) ([]models.VaultNode, error)
	Count(exec Executor, ctx context.Context) (int64, error)

	// Bulk operations for rebuilding
	DeleteAll(exec Executor, ctx context.Context) error
}

// EdgeRepository defines operations for VaultEdge persistence.
// It manages the relationships between nodes in the knowledge graph,
// supporting both wikilinks and embeds. All methods use the Executor
// pattern for database flexibility.
type EdgeRepository interface {
	// Basic CRUD operations
	Create(exec Executor, ctx context.Context, edge *models.VaultEdge) error
	GetByID(exec Executor, ctx context.Context, id string) (*models.VaultEdge, error)
	Delete(exec Executor, ctx context.Context, id string) error

	// Batch operations
	CreateBatch(exec Executor, ctx context.Context, edges []models.VaultEdge) error
	UpsertBatch(exec Executor, ctx context.Context, edges []models.VaultEdge) error

	// Query operations
	GetByNode(exec Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error)
	GetBySourceAndTarget(exec Executor, ctx context.Context, sourceID, targetID string) ([]models.VaultEdge, error)
	GetAll(exec Executor, ctx context.Context, limit, offset int) ([]models.VaultEdge, error)
	Count(exec Executor, ctx context.Context) (int64, error)

	// Graph queries
	GetIncomingEdges(exec Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error)
	GetOutgoingEdges(exec Executor, ctx context.Context, nodeID string) ([]models.VaultEdge, error)

	// Bulk operations for rebuilding
	DeleteAll(exec Executor, ctx context.Context) error
}

// PositionRepository handles node position persistence for graph visualization.
// It stores the X, Y, Z coordinates of nodes and their lock status,
// allowing users to save custom layouts. The repository supports both
// individual and batch position updates for performance.
type PositionRepository interface {
	GetByNodeID(exec Executor, ctx context.Context, nodeID string) (*models.NodePosition, error)
	Upsert(exec Executor, ctx context.Context, position *models.NodePosition) error
	UpsertBatch(exec Executor, ctx context.Context, positions []models.NodePosition) error
	GetAll(exec Executor, ctx context.Context) ([]models.NodePosition, error)
	DeleteByNodeID(exec Executor, ctx context.Context, nodeID string) error
}

// MetadataRepository handles vault metadata and parse history.
// It manages key-value metadata about the vault (such as last sync time)
// and tracks the history of vault parsing operations including timing,
// statistics, and error information.
type MetadataRepository interface {
	// Vault metadata operations
	GetMetadata(exec Executor, ctx context.Context, key string) (*models.VaultMetadata, error)
	SetMetadata(exec Executor, ctx context.Context, metadata *models.VaultMetadata) error
	GetAllMetadata(exec Executor, ctx context.Context) ([]models.VaultMetadata, error)

	// Parse history operations
	CreateParseRecord(exec Executor, ctx context.Context, record *models.ParseHistory) error
	GetLatestParse(exec Executor, ctx context.Context) (*models.ParseHistory, error)
	GetParseHistory(exec Executor, ctx context.Context, limit int) ([]models.ParseHistory, error)
	UpdateParseStatus(exec Executor, ctx context.Context, id string, status models.ParseStatus) error
}

// TransactionManager handles database transactions.
// It provides a higher-level abstraction over database transactions,
// ensuring proper cleanup on errors and panics. The callback function
// receives a Transaction object that provides access to the underlying
// database executor.
type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(tx Transaction) error) error
}

// Transaction represents a database transaction with access to the transaction executor.
// It provides methods to retrieve the executor for use with repositories,
// and to manually commit or rollback the transaction if needed. Most users
// should rely on TransactionManager's automatic handling instead of calling
// Commit/Rollback directly.
type Transaction interface {
	// Executor returns the transaction executor to use with repositories
	Executor() Executor
	// Commit commits the transaction
	Commit(ctx context.Context) error
	// Rollback rolls back the transaction
	Rollback(ctx context.Context) error
}