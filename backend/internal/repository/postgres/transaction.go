// Package postgres implements transaction management for PostgreSQL
package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/repository"
)

// TransactionManager implements repository.TransactionManager using PostgreSQL
// This implementation works with stateless repositories
type TransactionManager struct {
	db *sqlx.DB
}

// NewTransactionManager creates a new PostgreSQL transaction manager
func NewTransactionManager(database *sqlx.DB) repository.TransactionManager {
	return &TransactionManager{db: database}
}

// WithTransaction executes a function within a database transaction
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(repository.Transaction) error) error {
	tx, err := tm.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create transaction object
	transaction := &postgresTransactionStateless{
		tx:         tx,
		committed:  false,
		rolledback: false,
	}

	// Ensure cleanup on panic
	defer func() {
		if p := recover(); p != nil {
			if err := transaction.Rollback(ctx); err != nil {
				// Log rollback error but don't mask the panic
				fmt.Printf("Failed to rollback transaction during panic: %v\n", err)
			}
			panic(p)
		}
	}()

	// Execute the function
	if err := fn(transaction); err != nil {
		if rollbackErr := transaction.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("transaction failed: %w; additionally, rollback failed: %v", err, rollbackErr)
		}
		return err
	}

	// Auto-commit if not already committed or rolled back
	if !transaction.committed && !transaction.rolledback {
		if err := transaction.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}

// postgresTransactionStateless implements repository.Transaction
type postgresTransactionStateless struct {
	tx         *sqlx.Tx
	committed  bool
	rolledback bool
}

// Executor returns the transaction as an Executor
func (t *postgresTransactionStateless) Executor() repository.Executor {
	return t.tx
}

// Commit commits the transaction
func (t *postgresTransactionStateless) Commit(ctx context.Context) error {
	if t.committed || t.rolledback {
		return fmt.Errorf("transaction already finished")
	}

	// Check if context is cancelled before committing
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before commit: %w", err)
	}

	if err := t.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	t.committed = true
	return nil
}

// Rollback rolls back the transaction
func (t *postgresTransactionStateless) Rollback(ctx context.Context) error {
	if t.committed || t.rolledback {
		return nil // Already finished
	}

	// Check context before rollback
	if err := ctx.Err(); err != nil {
		// Still attempt rollback even if context is cancelled
		// to clean up resources, but note the context error
		_ = t.tx.Rollback()
		t.rolledback = true
		return fmt.Errorf("context cancelled before rollback: %w", err)
	}

	if err := t.tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	t.rolledback = true
	return nil
}

// Alternative helper function for direct transaction usage
func WithTransactionStateless(database *sqlx.DB, ctx context.Context, fn func(*sqlx.Tx) error) error {
	return db.WithTransaction(database, ctx, fn)
}
