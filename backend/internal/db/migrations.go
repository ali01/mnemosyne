package db

import (
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

//go:embed schema.sql
var schemaSQL string

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	SQL         string
	AppliedAt   time.Time
}

// InitializeSchema creates the initial database schema
func (db *DB) InitializeSchema() error {
	// Create migrations table if it doesn't exist
	migrationTableSQL := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INT PRIMARY KEY,
		description TEXT,
		applied_at TIMESTAMP DEFAULT NOW()
	);`

	if _, err := db.Exec(migrationTableSQL); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Check if schema is already initialized
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM schema_migrations WHERE version = 1")
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if count > 0 {
		log.Println("Database schema already initialized")
		return nil
	}

	// Execute the schema
	log.Println("Initializing database schema...")
	if execErr := db.ExecuteSchema(schemaSQL); execErr != nil {
		return fmt.Errorf("failed to initialize schema: %w", execErr)
	}

	// Record the migration
	_, err = db.Exec(
		"INSERT INTO schema_migrations (version, description) VALUES ($1, $2)",
		1, "Initial schema",
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}

// GetAppliedMigrations returns all applied migrations
func (db *DB) GetAppliedMigrations() ([]Migration, error) {
	var migrations []Migration
	err := db.Select(&migrations,
		"SELECT version, description, applied_at FROM schema_migrations ORDER BY version")
	return migrations, err
}

// RunMigration applies a specific migration
func (db *DB) RunMigration(version int, description string, sql string) error {
	// Check if migration already applied
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if count > 0 {
		log.Printf("Migration %d already applied", version)
		return nil
	}

	// Apply migration in transaction
	return db.Transaction(func(tx *sqlx.Tx) error {
		// Execute migration SQL
		if _, err := tx.Exec(sql); err != nil {
			return fmt.Errorf("failed to execute migration %d: %w", version, err)
		}

		// Record migration
		_, err := tx.Exec(
			"INSERT INTO schema_migrations (version, description) VALUES ($1, $2)",
			version, description,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %w", version, err)
		}

		log.Printf("Applied migration %d: %s", version, description)
		return nil
	})
}

// Future migrations can be added here
var migrations = []struct {
	Version     int
	Description string
	SQL         string
}{
	// Example future migration:
	// {
	//     Version:     2,
	//     Description: "Add full-text search configuration",
	//     SQL: `
	//         ALTER TABLE nodes ADD COLUMN search_vector tsvector;
	//         CREATE INDEX idx_nodes_search_vector ON nodes USING GIN(search_vector);
	//     `,
	// },
}

// RunAllMigrations applies all pending migrations
func (db *DB) RunAllMigrations() error {
	// First ensure base schema exists
	if err := db.InitializeSchema(); err != nil {
		return err
	}

	// Then run any additional migrations
	for _, m := range migrations {
		if err := db.RunMigration(m.Version, m.Description, m.SQL); err != nil {
			return err
		}
	}

	return nil
}
