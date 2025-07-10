// Package postgres provides PostgreSQL implementations of repository interfaces
package postgres

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// handlePostgresError converts PostgreSQL-specific errors to repository errors
func handlePostgresError(err error, resource string) error {
	if err == nil {
		return nil
	}

	// Check for no rows error
	if err == sql.ErrNoRows {
		return &NotFoundError{Resource: resource}
	}

	// Check for PostgreSQL-specific errors
	if pgErr, ok := err.(*pq.Error); ok {
		switch pgErr.Code {
		case "23505": // unique_violation
			return &DuplicateKeyError{
				Resource: resource,
				Message:  pgErr.Detail,
			}
		case "23503": // foreign_key_violation
			return &ValidationError{
				Resource: resource,
				Field:    "foreign_key",
				Message:  pgErr.Detail,
			}
		case "22P02": // invalid_text_representation
			return &ValidationError{
				Resource: resource,
				Field:    "format",
				Message:  pgErr.Message,
			}
		}
	}

	// Wrap unhandled errors with context
	return fmt.Errorf("database operation failed for %s: %w", resource, err)
}

// IsNotFound checks if an error is a NotFoundError
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*NotFoundError)
	return ok
}
