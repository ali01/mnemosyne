// Package repository defines interfaces for data persistence operations
package repository

import (
	"context"
	"database/sql"
)

// Executor defines database operations that can be performed by both
// sqlx.DB and sqlx.Tx. This allows repositories to work with both
// regular connections and transactions transparently.
//
// Both *sqlx.DB and *sqlx.Tx implement this interface naturally,
// so no wrapper types are needed.
type Executor interface {
	// Query operations - for reading data
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	
	// Execution operations - for writes/updates
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	
	// Query operations that return sql types
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	
	// Prepared statements
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}