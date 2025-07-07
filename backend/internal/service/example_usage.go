// Package service demonstrates how the new architecture solves the transaction problem
package service

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/repository/postgres"
)

// ExampleService demonstrates the solution to the createTransactionRepositories problem
type ExampleService struct {
	db       *sqlx.DB
	nodeRepo repository.NodeRepository
	edgeRepo repository.EdgeRepository
	txMgr    repository.TransactionManager
}

// NewExampleService creates a new example service
func NewExampleService(database *sqlx.DB) *ExampleService {
	return &ExampleService{
		db:       database,
		nodeRepo: postgres.NewNodeRepository(),
		edgeRepo: postgres.NewEdgeRepository(),
		txMgr:    postgres.NewTransactionManager(database),
	}
}

// DemonstrateSimpleUsage shows basic non-transactional usage
func (s *ExampleService) DemonstrateSimpleUsage(ctx context.Context) error {
	// Direct usage with sqlx.DB (implements Executor interface)
	node := &models.VaultNode{
		Title:    "Example Node",
		FilePath: "/example.md",
	}
	
	// No type conversion needed - sqlx.DB implements Executor
	return s.nodeRepo.Create(s.db, ctx, node)
}

// DemonstrateTransactionUsage shows how transactions work now
func (s *ExampleService) DemonstrateTransactionUsage(ctx context.Context) error {
	// Using the transaction manager
	return s.txMgr.WithTransaction(ctx, func(tx repository.Transaction) error {
		// Get the executor from the transaction
		exec := tx.Executor() // This returns *sqlx.Tx which implements Executor
		
		// Create a node using the transaction
		node := &models.VaultNode{
			Title:    "Transaction Node",
			FilePath: "/transaction.md",
		}
		if err := s.nodeRepo.Create(exec, ctx, node); err != nil {
			return err
		}
		
		// Create an edge using the same transaction
		edge := &models.VaultEdge{
			SourceID:  node.ID,
			TargetID:  "other-node",
			EdgeType:  "link",
		}
		if err := s.edgeRepo.Create(exec, ctx, edge); err != nil {
			return err
		}
		
		// Both operations succeed or fail together
		return nil
	})
}

// DemonstrateDirectTransactionUsage shows direct transaction usage
func (s *ExampleService) DemonstrateDirectTransactionUsage(ctx context.Context) error {
	// Using sqlx transactions directly
	return db.WithTransaction(s.db, ctx, func(tx *sqlx.Tx) error {
		// tx implements Executor interface directly!
		// No type conversion needed
		
		node := &models.VaultNode{
			Title:    "Direct Transaction Node",
			FilePath: "/direct-tx.md",
		}
		if err := s.nodeRepo.Create(tx, ctx, node); err != nil {
			return err
		}
		
		// Can use the same tx for edges
		edge := &models.VaultEdge{
			SourceID:  node.ID,
			TargetID:  "another-node",
			EdgeType:  "reference",
		}
		return s.edgeRepo.Create(tx, ctx, edge)
	})
}

// CompareWithOldProblem shows what the old problem was
func CompareWithOldProblem() {
	/*
	// OLD PROBLEM: This didn't work
	func createTransactionRepositories(tx *sqlx.Tx) (...) {
		// COMPILE ERROR: cannot assign *sqlx.Tx to *db.DB
		txDB := &db.DB{DB: tx}  
		
		// These required *db.DB, not *sqlx.Tx
		return NewNodeRepository(txDB),
			   NewEdgeRepository(txDB),
			   ...
	}
	
	// NEW SOLUTION: No conversion needed!
	func (s *Service) UseTransaction(ctx context.Context) error {
		return db.WithTransaction(s.db, ctx, func(tx *sqlx.Tx) error {
			// tx implements Executor - use it directly!
			return s.nodeRepo.Create(tx, ctx, node)
		})
	}
	*/
	
	fmt.Println("The problem is solved by:")
	fmt.Println("1. Removing db.DB wrapper - use *sqlx.DB directly")
	fmt.Println("2. Repositories accept Executor interface as parameter")
	fmt.Println("3. Both *sqlx.DB and *sqlx.Tx implement Executor")
	fmt.Println("4. No type conversion needed!")
}