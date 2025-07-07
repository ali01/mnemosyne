package mock

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// mockExecutorForTest is a simple mock executor for testing
type mockExecutorForTest struct{}

func (e *mockExecutorForTest) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return nil
}

func (e *mockExecutorForTest) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return nil
}

func (e *mockExecutorForTest) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return &mockResult{}, nil
}

func (e *mockExecutorForTest) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return &mockResult{}, nil
}

func (e *mockExecutorForTest) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (e *mockExecutorForTest) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

func (e *mockExecutorForTest) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, nil
}

// mockResult implements sql.Result
type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}

func TestMockNodeRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewNodeRepository()
	exec := &mockExecutorForTest{}

	// Test Create
	node := &models.VaultNode{
		ID:       "test-1",
		Title:    "Test Node",
		FilePath: "/test.md",
	}

	err := repo.Create(exec, ctx, node)
	if err != nil {
		t.Errorf("Create failed: %v", err)
	}

	// Test GetByID
	retrieved, err := repo.GetByID(exec, ctx, "test-1")
	if err != nil {
		t.Errorf("GetByID failed: %v", err)
	}

	if retrieved.Title != "Test Node" {
		t.Errorf("Expected title 'Test Node', got '%s'", retrieved.Title)
	}

	// Test Search
	results, err := repo.Search(exec, ctx, "Test")
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(results))
	}

	// Test Delete
	err = repo.Delete(exec, ctx, "test-1")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(exec, ctx, "test-1")
	if err == nil {
		t.Errorf("Expected error after delete")
	}
}

func TestMockTransactionManager(t *testing.T) {
	ctx := context.Background()
	tm := NewTransactionManager()

	// Test transaction with commit
	err := tm.WithTransaction(ctx, func(tx repository.Transaction) error {
		// The mock transaction doesn't actually do anything,
		// but this tests the interface
		return nil
	})

	if err != nil {
		t.Errorf("Transaction failed: %v", err)
	}
}