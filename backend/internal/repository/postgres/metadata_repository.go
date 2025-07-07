// Package postgres implements PostgreSQL-based repository for metadata and parse history
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// MetadataRepository implements repository.MetadataRepository using PostgreSQL without state
type MetadataRepository struct {
	// No fields - stateless
}

// NewMetadataRepository creates a new stateless PostgreSQL-based metadata repository
func NewMetadataRepository() repository.MetadataRepository {
	return &MetadataRepository{}
}

// GetMetadata retrieves a metadata value by key
func (r *MetadataRepository) GetMetadata(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error) {
	var metadata models.VaultMetadata
	query := `SELECT key, value, updated_at FROM vault_metadata WHERE key = $1`

	err := exec.GetContext(ctx, &metadata, query, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &NotFoundError{Resource: "metadata", ID: key}
		}
		return nil, err
	}

	return &metadata, nil
}

// SetMetadata sets a metadata value
func (r *MetadataRepository) SetMetadata(exec repository.Executor, ctx context.Context, metadata *models.VaultMetadata) error {
	metadata.UpdatedAt = time.Now()

	query := `
		INSERT INTO vault_metadata (key, value, updated_at)
		VALUES (:key, :value, :updated_at)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = EXCLUDED.updated_at
	`

	_, err := exec.NamedExecContext(ctx, query, metadata)
	if err != nil {
		return handlePostgresError(err, "metadata")
	}

	return nil
}

// GetAllMetadata retrieves all metadata entries
func (r *MetadataRepository) GetAllMetadata(exec repository.Executor, ctx context.Context) ([]models.VaultMetadata, error) {
	query := `SELECT key, value, updated_at FROM vault_metadata ORDER BY key`

	var metadata []models.VaultMetadata
	err := exec.SelectContext(ctx, &metadata, query)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// CreateParseRecord creates a new parse history record
func (r *MetadataRepository) CreateParseRecord(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}

	query := `
		INSERT INTO parse_history (id, started_at, completed_at, status, stats, error)
		VALUES (:id, :started_at, :completed_at, :status, :stats, :error)
	`

	_, err := exec.NamedExecContext(ctx, query, record)
	if err != nil {
		return handlePostgresError(err, "parse_history")
	}

	return nil
}

// GetLatestParse retrieves the most recent parse record
func (r *MetadataRepository) GetLatestParse(exec repository.Executor, ctx context.Context) (*models.ParseHistory, error) {
	var record models.ParseHistory
	query := `
		SELECT id, started_at, completed_at, status, stats, error 
		FROM parse_history 
		ORDER BY started_at DESC 
		LIMIT 1
	`

	err := exec.GetContext(ctx, &record, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &NotFoundError{Resource: "parse_history"}
		}
		return nil, err
	}

	return &record, nil
}

// GetParseHistory retrieves parse history records
func (r *MetadataRepository) GetParseHistory(exec repository.Executor, ctx context.Context, limit int) ([]models.ParseHistory, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, started_at, completed_at, status, stats, error 
		FROM parse_history 
		ORDER BY started_at DESC 
		LIMIT $1
	`

	var records []models.ParseHistory
	err := exec.SelectContext(ctx, &records, query, limit)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// UpdateParseStatus updates the status of a parse record
func (r *MetadataRepository) UpdateParseStatus(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus) error {
	var query string
	var args []interface{}

	switch status {
	case models.ParseStatusCompleted, models.ParseStatusFailed:
		// Set completed_at when finishing
		query = `UPDATE parse_history SET status = $1, completed_at = $2 WHERE id = $3`
		args = []interface{}{status, time.Now(), id}
	default:
		// Just update status
		query = `UPDATE parse_history SET status = $1 WHERE id = $2`
		args = []interface{}{status, id}
	}

	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return handlePostgresError(err, "parse_history")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return &NotFoundError{Resource: "parse_history", ID: id}
	}

	return nil
}