package db_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/repository/postgres"
)

func TestConnect_InvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  db.Config
		wantErr bool
	}{
		{
			name: "empty host",
			config: db.Config{
				Host:     "",
				Port:     5432,
				User:     "test",
				Password: "test",
				DBName:   "test",
				SSLMode:  "disable",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: db.Config{
				Host:     "localhost",
				Port:     -1,
				User:     "test",
				Password: "test",
				DBName:   "test",
				SSLMode:  "disable",
			},
			wantErr: true,
		},
		{
			name: "invalid ssl mode",
			config: db.Config{
				Host:     "localhost",
				Port:     5432,
				User:     "test",
				Password: "test",
				DBName:   "test",
				SSLMode:  "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Connect(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecuteSchema_InvalidSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database
	tdb := postgres.CreateTestDB(t)
	defer tdb.Close()

	// Try to execute invalid SQL
	invalidSQL := "INVALID SQL STATEMENT;"
	_, err := tdb.Exec(invalidSQL)
	assert.Error(t, err)
}

func TestWithTransaction_NilDB(t *testing.T) {
	ctx := context.Background()
	err := db.WithTransaction(nil, ctx, func(tx *sqlx.Tx) error {
		return nil
	})
	assert.Error(t, err)
}

func TestWithTransaction_NilFunc(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database
	tdb := postgres.CreateTestDB(t)
	defer tdb.Close()

	ctx := context.Background()
	err := db.WithTransaction(tdb.DB, ctx, nil)
	assert.Error(t, err)
}

// TestSchemaContent verifies that the schema can be loaded
func TestSchemaContent(t *testing.T) {
	t.Run("embedded schema is accessible", func(t *testing.T) {
		// Verify the embedded schema is accessible and not empty
		require.NotEmpty(t, db.SchemaSQL)

		// Verify it contains expected SQL structures
		assert.Contains(t, db.SchemaSQL, "CREATE TABLE")
		assert.Contains(t, db.SchemaSQL, "nodes")
		assert.Contains(t, db.SchemaSQL, "edges")
		assert.Contains(t, db.SchemaSQL, "node_positions")
		assert.Contains(t, db.SchemaSQL, "vault_metadata")
	})
}
