// Package postgres provides PostgreSQL-specific error types
package postgres

import (
	"fmt"

	"github.com/ali01/mnemosyne/internal/repository"
)

// NotFoundError is returned when a resource is not found
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s with ID '%s' not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

func (e *NotFoundError) Is(target error) bool {
	return target == repository.ErrNotFound
}

// DuplicateKeyError is returned when a unique constraint is violated
type DuplicateKeyError struct {
	Resource string
	Field    string
	Value    string
	Message  string
}

func (e *DuplicateKeyError) Error() string {
	if e.Field != "" && e.Value != "" {
		return fmt.Sprintf("%s with %s '%s' already exists", e.Resource, e.Field, e.Value)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s duplicate key: %s", e.Resource, e.Message)
	}
	return fmt.Sprintf("%s duplicate key violation", e.Resource)
}

func (e *DuplicateKeyError) Is(target error) bool {
	return target == repository.ErrDuplicateKey
}

// ValidationError is returned when input validation fails
type ValidationError struct {
	Resource string
	Field    string
	Message  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s validation failed: %s - %s", e.Resource, e.Field, e.Message)
}

func (e *ValidationError) Is(target error) bool {
	return target == repository.ErrInvalidInput
}