// Package postgres provides test helpers for integration tests
//
// Test Helper Usage Guide:
// - Use CreateTestDB() when you only need a database connection
// - Use CreateTestRepositories() when you need both database and repositories
// - All helpers handle container setup and cleanup automatically
package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/repository"
)

// TestDB provides a test database connection and cleanup
type TestDB struct {
	*sqlx.DB
	cleanup func()
}

// NewTestDB creates a new test database connection
// It uses testcontainers to spin up a PostgreSQL instance
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	ctx := context.Background()

	// Check if Docker is available before attempting to create containers
	dockerClient, err := testcontainers.NewDockerClientWithOpts(ctx)
	if err != nil {
		t.Skipf("Docker is not available: %v", err)
	}
	defer dockerClient.Close()

	// Also check Docker daemon is responsive
	_, err = dockerClient.Ping(ctx)
	if err != nil {
		t.Skipf("Docker daemon is not responsive: %v", err)
	}

	// Catch any unexpected panics from testcontainers
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Failed to start PostgreSQL container (unexpected panic): %v", r)
		}
	}()

	// Create PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("mnemosyne_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Skipf("Failed to start PostgreSQL container (is Docker running?): %v", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database
	dbx, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run embedded schema
	if err := db.ExecuteSchema(dbx, db.SchemaSQL); err != nil {
		t.Fatalf("Failed to execute schema: %v", err)
	}

	return &TestDB{
		DB: dbx,
		cleanup: func() {
			dbx.Close()
			if err := pgContainer.Terminate(ctx); err != nil {
				t.Logf("Failed to terminate container: %v", err)
			}
		},
	}
}

// Close closes the test database connection
func (tdb *TestDB) Close() {
	if tdb.cleanup != nil {
		tdb.cleanup()
	}
}

// CleanTables truncates all tables for a fresh test
func (tdb *TestDB) CleanTables(ctx context.Context) error {
	tables := []string{
		"edges",
		"node_positions",
		"nodes",
		"parse_history",
		"vault_metadata",
		"unresolved_links",
	}

	for _, table := range tables {
		if _, err := tdb.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			// Ignore errors if table doesn't exist
			continue
		}
	}

	return nil
}


// CreateTestDB creates a test database for simple database tests
// Use this when you only need a database connection without repositories
func CreateTestDB(t *testing.T) *TestDB {
	t.Helper()
	return NewTestDB(t)
}

// CreateTestRepositories creates all repositories for testing
// Use this when you need both database and repository instances
func CreateTestRepositories(t *testing.T) (*TestDB, *TestRepositories) {
	t.Helper()

	tdb := NewTestDB(t)

	repos := &TestRepositories{
		Nodes:        NewNodeRepository(),
		Edges:        NewEdgeRepository(),
		Positions:    NewPositionRepository(),
		Metadata:     NewMetadataRepository(),
		Transactions: NewTransactionManager(tdb.DB),
	}

	return tdb, repos
}

// TestRepositories holds all repository instances for testing
type TestRepositories struct {
	Nodes        repository.NodeRepository
	Edges        repository.EdgeRepository
	Positions    repository.PositionRepository
	Metadata     repository.MetadataRepository
	Transactions repository.TransactionManager
}
