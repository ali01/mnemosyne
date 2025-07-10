// Package db provides database setup and transaction helpers for PostgreSQL
package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Connect creates a new database connection with connection pooling
func Connect(cfg Config) (*sqlx.DB, error) {
	// Build PostgreSQL DSN
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	// Connect and verify
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Connected to PostgreSQL database: %s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, cfg.DBName)
	return db, nil
}

// ExecuteSchema runs the schema SQL file
func ExecuteSchema(db *sqlx.DB, schemaSQL string) error {
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}
	return nil
}

// WithTransaction executes a function within a database transaction with context support
// Automatically handles commit/rollback and panics
func WithTransaction(db *sqlx.DB, ctx context.Context, fn func(*sqlx.Tx) error) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	if fn == nil {
		return fmt.Errorf("transaction function is nil")
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback on panic
	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				// Log rollback error but don't mask the original panic
				log.Printf("Failed to rollback transaction during panic: %v", rbErr)
			}
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %w; additionally, rollback failed: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
