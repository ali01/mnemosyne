// Package postgres provides test helpers for integration tests
package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/repository"
)

// TestDB provides a test database connection and cleanup
type TestDB struct {
	*sqlx.DB
	cleanup func()
}

// NewTestDB creates a new test database connection
// It reads configuration from environment variables or uses defaults
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Read config from environment or use defaults
	cfg := db.Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnvOrDefault("TEST_DB_USER", "postgres"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		DBName:   getEnvOrDefault("TEST_DB_NAME", "mnemosyne_test"),
		SSLMode:  getEnvOrDefault("TEST_DB_SSLMODE", "disable"),
	}

	// Try to connect
	database, err := db.Connect(cfg)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}

	// Run migrations if schema file exists
	schemaPath := "../../../db/schema.sql"
	if schemaSQL, err := os.ReadFile(schemaPath); err == nil {
		if err := db.ExecuteSchema(database, string(schemaSQL)); err != nil {
			t.Logf("Warning: Failed to execute schema: %v", err)
		}
	}

	return &TestDB{
		DB: database,
		cleanup: func() {
			database.Close()
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

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// CreateTestRepositories creates all repositories for testing
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