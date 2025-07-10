// Package mock provides in-memory implementations of repository interfaces for testing
package mock

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// TransactionManager is an in-memory implementation of repository.TransactionManager
// that provides proper transaction semantics with rollback support
type TransactionManager struct {
	mu        sync.Mutex
	nodes     *NodeRepository
	edges     *EdgeRepository
	positions *PositionRepository
	metadata  *MetadataRepository
}

// NewTransactionManager creates a new mock transaction manager
func NewTransactionManager() repository.TransactionManager {
	return &TransactionManager{
		nodes:     NewNodeRepository().(*NodeRepository),
		edges:     NewEdgeRepository().(*EdgeRepository),
		positions: NewPositionRepository().(*PositionRepository),
		metadata:  NewMetadataRepository().(*MetadataRepository),
	}
}

// WithTransaction executes a function within a mock transaction with proper rollback support
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(repository.Transaction) error) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Create transaction with snapshots
	tx := &mockTransaction{
		executor:  &mockExecutor{},
		manager:   tm,
		committed: false,
		rolledback: false,
	}

	// Take snapshots of current state
	tx.takeSnapshots()

	// Execute the function
	if err := fn(tx); err != nil {
		// Rollback on error
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("transaction failed: %w; additionally, rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("transaction failed: %w", err)
	}

	// Auto-commit if not already committed or rolled back
	if !tx.committed && !tx.rolledback {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}

// mockTransaction implements repository.Transaction with rollback support
type mockTransaction struct {
	executor   repository.Executor
	manager    *TransactionManager
	committed  bool
	rolledback bool

	// Snapshots for rollback
	snapshot struct {
		nodes     map[string]*models.VaultNode
		edges     map[string]*models.VaultEdge
		positions map[string]*models.NodePosition
		metadata  map[string]*models.VaultMetadata
		history   map[string]*models.ParseHistory
	}
}

// Executor returns the transaction executor
func (t *mockTransaction) Executor() repository.Executor {
	return t.executor
}

// Commit commits the transaction
func (t *mockTransaction) Commit(ctx context.Context) error {
	if t.committed || t.rolledback {
		return fmt.Errorf("transaction already finished")
	}
	t.committed = true
	// Changes are already applied, just mark as committed
	return nil
}

// Rollback rolls back the transaction by restoring snapshots
func (t *mockTransaction) Rollback(ctx context.Context) error {
	if t.committed || t.rolledback {
		return nil // Already finished
	}

	t.rolledback = true
	t.restoreSnapshots()
	return nil
}

// takeSnapshots creates deep copies of all repository data
func (t *mockTransaction) takeSnapshots() {
	// Lock all repositories for reading
	t.manager.nodes.mu.RLock()
	t.manager.edges.mu.RLock()
	t.manager.positions.mu.RLock()
	t.manager.metadata.mu.RLock()
	defer t.manager.nodes.mu.RUnlock()
	defer t.manager.edges.mu.RUnlock()
	defer t.manager.positions.mu.RUnlock()
	defer t.manager.metadata.mu.RUnlock()

	// Deep copy nodes
	t.snapshot.nodes = make(map[string]*models.VaultNode)
	for k, v := range t.manager.nodes.nodes {
		nodeCopy := *v
		t.snapshot.nodes[k] = &nodeCopy
	}

	// Deep copy edges
	t.snapshot.edges = make(map[string]*models.VaultEdge)
	for k, v := range t.manager.edges.edges {
		edgeCopy := *v
		t.snapshot.edges[k] = &edgeCopy
	}

	// Deep copy positions
	t.snapshot.positions = make(map[string]*models.NodePosition)
	for k, v := range t.manager.positions.positions {
		posCopy := *v
		t.snapshot.positions[k] = &posCopy
	}

	// Deep copy metadata
	t.snapshot.metadata = make(map[string]*models.VaultMetadata)
	for k, v := range t.manager.metadata.metadata {
		metaCopy := *v
		t.snapshot.metadata[k] = &metaCopy
	}

	// Deep copy history
	t.snapshot.history = make(map[string]*models.ParseHistory)
	for k, v := range t.manager.metadata.history {
		historyCopy := *v
		t.snapshot.history[k] = &historyCopy
	}
}

// restoreSnapshots restores all repository data from snapshots
func (t *mockTransaction) restoreSnapshots() {
	// Lock all repositories for writing
	t.manager.nodes.mu.Lock()
	t.manager.edges.mu.Lock()
	t.manager.positions.mu.Lock()
	t.manager.metadata.mu.Lock()
	defer t.manager.nodes.mu.Unlock()
	defer t.manager.edges.mu.Unlock()
	defer t.manager.positions.mu.Unlock()
	defer t.manager.metadata.mu.Unlock()

	// Restore nodes
	t.manager.nodes.nodes = make(map[string]*models.VaultNode)
	for k, v := range t.snapshot.nodes {
		nodeCopy := *v
		t.manager.nodes.nodes[k] = &nodeCopy
	}

	// Restore edges
	t.manager.edges.edges = make(map[string]*models.VaultEdge)
	for k, v := range t.snapshot.edges {
		edgeCopy := *v
		t.manager.edges.edges[k] = &edgeCopy
	}

	// Restore positions
	t.manager.positions.positions = make(map[string]*models.NodePosition)
	for k, v := range t.snapshot.positions {
		posCopy := *v
		t.manager.positions.positions[k] = &posCopy
	}

	// Restore metadata
	t.manager.metadata.metadata = make(map[string]*models.VaultMetadata)
	for k, v := range t.snapshot.metadata {
		metaCopy := *v
		t.manager.metadata.metadata[k] = &metaCopy
	}

	// Restore history
	t.manager.metadata.history = make(map[string]*models.ParseHistory)
	for k, v := range t.snapshot.history {
		historyCopy := *v
		t.manager.metadata.history[k] = &historyCopy
	}
}

// mockExecutor implements repository.Executor for testing
type mockExecutor struct{}

// GetContext implements Executor
func (e *mockExecutor) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return fmt.Errorf("mock executor: GetContext not implemented")
}

// SelectContext implements Executor
func (e *mockExecutor) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return fmt.Errorf("mock executor: SelectContext not implemented")
}

// ExecContext implements Executor
func (e *mockExecutor) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, fmt.Errorf("mock executor: ExecContext not implemented")
}

// NamedExecContext implements Executor
func (e *mockExecutor) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return nil, fmt.Errorf("mock executor: NamedExecContext not implemented")
}

// QueryContext implements Executor
func (e *mockExecutor) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("mock executor: QueryContext not implemented")
}

// QueryRowContext implements Executor
func (e *mockExecutor) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

// PrepareContext implements Executor
func (e *mockExecutor) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, fmt.Errorf("mock executor: PrepareContext not implemented")
}

