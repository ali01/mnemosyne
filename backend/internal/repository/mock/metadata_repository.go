// Package mock provides in-memory implementations of repository interfaces for testing
package mock

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
)

// MetadataRepository is an in-memory implementation of repository.MetadataRepository
// that follows the stateless pattern
type MetadataRepository struct {
	mu       sync.RWMutex
	metadata map[string]*models.VaultMetadata
	history  map[string]*models.ParseHistory
}

// NewMetadataRepository creates a new mock metadata repository
func NewMetadataRepository() repository.MetadataRepository {
	return &MetadataRepository{
		metadata: make(map[string]*models.VaultMetadata),
		history:  make(map[string]*models.ParseHistory),
	}
}

// GetMetadata retrieves a metadata value by key
func (r *MetadataRepository) GetMetadata(exec repository.Executor, ctx context.Context, key string) (*models.VaultMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, exists := r.metadata[key]
	if !exists {
		return nil, repository.NewNotFoundError("metadata", key)
	}

	metaCopy := *meta
	return &metaCopy, nil
}

// SetMetadata sets a metadata value
func (r *MetadataRepository) SetMetadata(exec repository.Executor, ctx context.Context, metadata *models.VaultMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata.UpdatedAt = time.Now()
	metaCopy := *metadata
	r.metadata[metadata.Key] = &metaCopy
	return nil
}

// GetAllMetadata retrieves all metadata entries
func (r *MetadataRepository) GetAllMetadata(exec repository.Executor, ctx context.Context) ([]models.VaultMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.VaultMetadata, 0, len(r.metadata))
	for _, meta := range r.metadata {
		result = append(result, *meta)
	}

	return result, nil
}

// CreateParseRecord creates a new parse history record
func (r *MetadataRepository) CreateParseRecord(exec repository.Executor, ctx context.Context, record *models.ParseHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	recordCopy := *record
	if recordCopy.ID == "" {
		recordCopy.ID = uuid.New().String()
	}

	if _, exists := r.history[recordCopy.ID]; exists {
		return repository.NewDuplicateKeyError("parse_history", "id", recordCopy.ID)
	}

	r.history[recordCopy.ID] = &recordCopy

	// Update the input record ID to match real DB behavior
	record.ID = recordCopy.ID

	return nil
}

// GetLatestParse retrieves the most recent parse record
func (r *MetadataRepository) GetLatestParse(exec repository.Executor, ctx context.Context) (*models.ParseHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *models.ParseHistory
	for _, record := range r.history {
		if latest == nil || record.StartedAt.After(latest.StartedAt) {
			latest = record
		}
	}

	if latest == nil {
		return nil, repository.NewNotFoundError("parse_history", "latest")
	}

	latestCopy := *latest
	return &latestCopy, nil
}

// GetParseHistory retrieves parse history records
func (r *MetadataRepository) GetParseHistory(exec repository.Executor, ctx context.Context, limit int) ([]models.ParseHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect all records
	result := make([]models.ParseHistory, 0, len(r.history))
	for _, record := range r.history {
		result = append(result, *record)
	}

	// Sort by StartedAt descending (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.After(result[j].StartedAt)
	})

	// Apply limit
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	return result, nil
}

// UpdateParseStatus updates the status of a parse record
func (r *MetadataRepository) UpdateParseStatus(exec repository.Executor, ctx context.Context, id string, status models.ParseStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, exists := r.history[id]
	if !exists {
		return repository.NewNotFoundError("parse_history", id)
	}

	record.Status = status
	if status == models.ParseStatusCompleted || status == models.ParseStatusFailed {
		now := time.Now()
		record.CompletedAt = &now
	}

	return nil
}
