// Package repository provides custom error types for repository operations
package repository

import (
	"errors"
	"fmt"
)

// Common repository errors
var (
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("resource not found")

	// ErrDuplicateKey is returned when a unique constraint is violated
	ErrDuplicateKey = errors.New("duplicate key violation")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")

	// ErrConnection is returned when database connection fails
	ErrConnection = errors.New("database connection error")

	// ErrTransaction is returned when a transaction operation fails
	ErrTransaction = errors.New("transaction error")

	// ErrBatchOperation is returned when a batch operation partially fails
	ErrBatchOperation = errors.New("batch operation error")
)

// NotFoundError provides detailed information about missing resources
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID '%s' not found", e.Resource, e.ID)
}

func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// DuplicateKeyError provides information about unique constraint violations
type DuplicateKeyError struct {
	Resource string
	Field    string
	Value    string
}

func (e *DuplicateKeyError) Error() string {
	return fmt.Sprintf("%s with %s '%s' already exists", e.Resource, e.Field, e.Value)
}

func (e *DuplicateKeyError) Is(target error) bool {
	return target == ErrDuplicateKey
}

// ValidationError provides detailed validation failure information
type ValidationError struct {
	Resource string
	Field    string
	Message  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s validation failed: %s - %s", e.Resource, e.Field, e.Message)
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidInput
}

// BatchOperationError provides information about batch operation failures
type BatchOperationError struct {
	Operation      string
	TotalItems     int
	FailedItems    int
	FailedIndices  []int
	UnderlyingErrs []error
}

func (e *BatchOperationError) Error() string {
	return fmt.Sprintf("batch %s failed: %d/%d items failed", e.Operation, e.FailedItems, e.TotalItems)
}

func (e *BatchOperationError) Is(target error) bool {
	return target == ErrBatchOperation
}

// Unwrap returns the underlying errors for error chain inspection
func (e *BatchOperationError) Unwrap() []error {
	return e.UnderlyingErrs
}

// Helper functions for creating specific errors

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(resource, id string) error {
	return &NotFoundError{Resource: resource, ID: id}
}

// NewDuplicateKeyError creates a new DuplicateKeyError
func NewDuplicateKeyError(resource, field, value string) error {
	return &DuplicateKeyError{Resource: resource, Field: field, Value: value}
}

// NewValidationError creates a new ValidationError
func NewValidationError(resource, field, message string) error {
	return &ValidationError{Resource: resource, Field: field, Message: message}
}

// NewBatchOperationError creates a new BatchOperationError
func NewBatchOperationError(operation string, totalItems, failedItems int, failedIndices []int, errs []error) error {
	return &BatchOperationError{
		Operation:      operation,
		TotalItems:     totalItems,
		FailedItems:    failedItems,
		FailedIndices:  failedIndices,
		UnderlyingErrs: errs,
	}
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsDuplicateKey checks if an error is a duplicate key error
func IsDuplicateKey(err error) bool {
	return errors.Is(err, ErrDuplicateKey)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsConnectionError checks if an error is a connection error
func IsConnectionError(err error) bool {
	return errors.Is(err, ErrConnection)
}