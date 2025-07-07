// Package service implements the business logic layer
package service

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/repository/postgres"
)

// MetadataService handles vault metadata and parse history operations
type MetadataService struct {
	db           *sqlx.DB
	metadataRepo repository.MetadataRepository
}

// NewMetadataService creates a new metadata service
func NewMetadataService(db *sqlx.DB) *MetadataService {
	return &MetadataService{
		db:           db,
		metadataRepo: postgres.NewMetadataRepository(),
	}
}

// GetMetadata retrieves a metadata value by key
func (s *MetadataService) GetMetadata(ctx context.Context, key string) (*models.VaultMetadata, error) {
	return s.metadataRepo.GetMetadata(s.db, ctx, key)
}

// SetMetadata sets a metadata value
func (s *MetadataService) SetMetadata(ctx context.Context, metadata *models.VaultMetadata) error {
	return s.metadataRepo.SetMetadata(s.db, ctx, metadata)
}

// GetAllMetadata retrieves all metadata entries
func (s *MetadataService) GetAllMetadata(ctx context.Context) ([]models.VaultMetadata, error) {
	return s.metadataRepo.GetAllMetadata(s.db, ctx)
}

// CreateParseRecord creates a new parse history record
func (s *MetadataService) CreateParseRecord(ctx context.Context, record *models.ParseHistory) error {
	return s.metadataRepo.CreateParseRecord(s.db, ctx, record)
}

// GetLatestParse retrieves the most recent parse record
func (s *MetadataService) GetLatestParse(ctx context.Context) (*models.ParseHistory, error) {
	return s.metadataRepo.GetLatestParse(s.db, ctx)
}

// GetParseHistory retrieves parse history records
func (s *MetadataService) GetParseHistory(ctx context.Context, limit int) ([]models.ParseHistory, error) {
	return s.metadataRepo.GetParseHistory(s.db, ctx, limit)
}

// UpdateParseStatus updates the status of a parse record
func (s *MetadataService) UpdateParseStatus(ctx context.Context, id string, status models.ParseStatus) error {
	return s.metadataRepo.UpdateParseStatus(s.db, ctx, id, status)
}