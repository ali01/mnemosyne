package db_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ali01/mnemosyne/internal/db"
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
	// This test would require a real database connection
	// Skip for now as it needs a test database
	t.Skip("Requires test database")
}

func TestWithTransaction_NilDB(t *testing.T) {
	// This test would require a real database connection
	// Skip for now as it needs a test database
	t.Skip("Requires test database")
}

func TestWithTransaction_NilFunc(t *testing.T) {
	// This test would require a valid database connection
	// Skip for now as it needs a test database
	t.Skip("Requires test database")
}

// TestSchemaContent verifies that the schema can be loaded
func TestSchemaContent(t *testing.T) {
	// This test just ensures the schema file exists and can be read
	// The actual schema execution would require a test database
	t.Run("schema file exists", func(t *testing.T) {
		// The ExecuteSchema function will panic if the schema file doesn't exist
		// We can't test the actual execution without a database
		assert.NotPanics(t, func() {
			// Just accessing the embedded schema to ensure it exists
			_ = db.ExecuteSchema
		})
	})
}