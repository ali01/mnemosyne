# Repository Layer

This package provides the data access layer for the Mnemosyne application, implementing the Repository pattern for clean separation of concerns.

## Overview

The repository layer abstracts database operations and provides a clean interface for the service layer to interact with data storage. It supports both PostgreSQL (for production) and in-memory mock implementations (for testing).

## Structure

```
repository/
├── interfaces.go        # Core repository interfaces
├── errors.go           # Custom error types
├── factory.go          # Repository factory for DI
├── postgres/           # PostgreSQL implementations
│   ├── node_repository.go
│   ├── edge_repository.go
│   ├── position_repository.go
│   ├── metadata_repository.go
│   ├── transaction.go
│   ├── helpers.go
│   └── performance.go
└── mock/               # In-memory implementations for testing
    ├── node_repository.go
    ├── edge_repository.go
    ├── position_repository.go
    ├── metadata_repository.go
    └── transaction_manager.go
```

## Key Interfaces

### NodeRepository
- CRUD operations for vault nodes
- Batch operations for performance
- Search functionality
- Type-based filtering

### EdgeRepository
- CRUD operations for graph edges
- Graph traversal queries
- Relationship lookups

### PositionRepository
- Node position persistence for graph layout
- Batch updates for moving multiple nodes

### MetadataRepository
- Vault metadata storage
- Parse history tracking
- Key-value configuration

### TransactionManager
- Database transaction support
- Atomic operations across repositories
- Automatic rollback on errors

## Usage

### Basic Usage

```go
// Create repositories
nodeRepo := postgres.NewNodeRepository(db)
edgeRepo := postgres.NewEdgeRepository(db)

// Use in handlers
node, err := nodeRepo.GetByID(ctx, "node-123")
if err != nil {
    if repository.IsNotFound(err) {
        // Handle not found
    }
    // Handle other errors
}
```

### Transaction Usage

```go
err := txManager.WithTransaction(ctx, func(tx repository.Transaction) error {
    // Create node
    if err := tx.NodeRepository().Create(ctx, node); err != nil {
        return err
    }
    
    // Create edges
    for _, edge := range edges {
        if err := tx.EdgeRepository().Create(ctx, &edge); err != nil {
            return err
        }
    }
    
    return nil // Automatically commits
})
```

### Testing with Mocks

```go
// Create mock repositories for testing
nodeRepo := mock.NewNodeRepository()
edgeRepo := mock.NewEdgeRepository()

// Use exactly like real repositories
err := nodeRepo.Create(ctx, testNode)
```

## Performance Considerations

1. **Batch Operations**: Use `CreateBatch` and `UpsertBatch` for inserting multiple items
2. **Indexes**: Database indexes are created for common query patterns
3. **Full-text Search**: PostgreSQL's tsvector is used for efficient text search
4. **Connection Pooling**: Handled by the db package

## Error Handling

The repository layer provides custom error types:
- `ErrNotFound`: Resource not found
- `ErrDuplicateKey`: Unique constraint violation
- `ErrInvalidInput`: Validation errors
- `ErrConnection`: Database connection issues

Use helper functions to check error types:
```go
if repository.IsNotFound(err) {
    // Handle not found case
}
```

## Next Steps

The repository layer is complete and ready for use by the service layer (Phase 4), which will:
- Orchestrate parser → builder → repository pipeline
- Handle async operations
- Provide business logic layer