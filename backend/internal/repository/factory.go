// Package repository provides a factory for creating repository instances
package repository

import "context"

// Repositories holds all repository instances
type Repositories struct {
	Nodes        NodeRepository
	Edges        EdgeRepository
	Positions    PositionRepository
	Metadata     MetadataRepository
	Transactions TransactionManager
}

// NewRepositories creates a new Repositories struct with the provided implementations.
// The context parameter allows for initialization operations that might require
// database access, such as ensuring tables exist or setting up prepared statements.
func NewRepositories(
	ctx context.Context,
	nodes NodeRepository,
	edges EdgeRepository,
	positions PositionRepository,
	metadata MetadataRepository,
	transactions TransactionManager,
) (*Repositories, error) {
	// Validate inputs
	if nodes == nil {
		return nil, &ValidationError{Field: "nodes", Message: "NodeRepository cannot be nil"}
	}
	if edges == nil {
		return nil, &ValidationError{Field: "edges", Message: "EdgeRepository cannot be nil"}
	}
	if positions == nil {
		return nil, &ValidationError{Field: "positions", Message: "PositionRepository cannot be nil"}
	}
	if metadata == nil {
		return nil, &ValidationError{Field: "metadata", Message: "MetadataRepository cannot be nil"}
	}
	if transactions == nil {
		return nil, &ValidationError{Field: "transactions", Message: "TransactionManager cannot be nil"}
	}

	return &Repositories{
		Nodes:        nodes,
		Edges:        edges,
		Positions:    positions,
		Metadata:     metadata,
		Transactions: transactions,
	}, nil
}