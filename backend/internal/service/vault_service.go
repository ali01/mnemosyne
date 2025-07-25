// Package service provides business logic layer for the application
package service

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/git"
	"github.com/ali01/mnemosyne/internal/models"
	"github.com/ali01/mnemosyne/internal/repository"
	"github.com/ali01/mnemosyne/internal/vault"
)

// GitManagerInterface defines the interface for git operations needed by VaultService
type GitManagerInterface interface {
	Pull(ctx context.Context) error
	GetLocalPath() string
}

// Compile-time check that git.Manager implements GitManagerInterface
var _ GitManagerInterface = (*git.Manager)(nil)

// VaultService orchestrates the parsing pipeline, connecting Git integration,
// vault parser, and graph builder with the database layer
type VaultService struct {
	config          *config.Config
	gitManager      GitManagerInterface
	nodeService     NodeServiceInterface
	edgeService     EdgeServiceInterface
	metadataService *MetadataService // Use concrete type for parse history methods
	db              *sqlx.DB

	// State management
	mu              sync.Mutex
	currentParseID  string
	isParsing       bool
	parseStartTime  time.Time
	parseProgress   *models.ParseProgress
}

// NewVaultService creates a new vault service with all dependencies
func NewVaultService(
	cfg *config.Config,
	gitManager GitManagerInterface,
	nodeService NodeServiceInterface,
	edgeService EdgeServiceInterface,
	metadataService *MetadataService,
	database *sqlx.DB,
) *VaultService {
	return &VaultService{
		config:          cfg,
		gitManager:      gitManager,
		nodeService:     nodeService,
		edgeService:     edgeService,
		metadataService: metadataService,
		db:              database,
	}
}

// ParseAndIndexVault orchestrates the complete parsing pipeline
func (s *VaultService) ParseAndIndexVault(ctx context.Context) (history *models.ParseHistory, err error) {
	// Recover from panics and convert to errors
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during parse: %v", r)
			s.logger().Error("Parse panicked", "panic", r, "stack", debug.Stack())
			// Update parse history as failed
			if history != nil {
				s.updateParseHistoryError(ctx, history, err)
			}
			// Always release the lock on panic
			s.releaseParseLock()
		}
	}()

	// Acquire parse lock
	parseID, err := s.acquireParseLock()
	if err != nil {
		return nil, err
	}
	defer s.releaseParseLock()

	// Initialize parse history
	parseHistory, logger := s.initializeParseHistory(ctx, parseID)
	if parseHistory == nil {
		return nil, fmt.Errorf("failed to initialize parse history")
	}
	history = parseHistory // Set named return value for panic recovery

	// Setup error handler
	handleError := func(err error, message string) (*models.ParseHistory, error) {
		logger.Error(message, "error", err)
		s.updateParseHistoryError(ctx, parseHistory, err)
		return parseHistory, fmt.Errorf("%s: %w", message, err)
	}

	// Execute parsing pipeline
	if err := s.executeParsePipeline(ctx, parseHistory, logger); err != nil {
		return handleError(err, "parsing pipeline failed")
	}

	// Finalize parse history
	s.finalizeParseHistory(ctx, parseHistory, logger)
	return parseHistory, nil
}

// acquireParseLock ensures only one parse runs at a time
func (s *VaultService) acquireParseLock() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isParsing {
		return "", fmt.Errorf("parse already in progress")
	}

	s.isParsing = true
	s.currentParseID = uuid.New().String()
	s.parseStartTime = time.Now()
	return s.currentParseID, nil
}

// releaseParseLock releases the parse lock
func (s *VaultService) releaseParseLock() {
	s.setParsingState(false, "")
}

// initializeParseHistory creates a new parse history record
func (s *VaultService) initializeParseHistory(ctx context.Context, parseID string) (*models.ParseHistory, *slog.Logger) {
	logger := s.logger().With("parse_id", parseID)
	logger.Info("Starting vault parse")

	startTime := time.Now()
	parseHistory := &models.ParseHistory{
		ID:        parseID,
		StartedAt: startTime,
		Status:    models.ParseStatusRunning,
		Stats:     models.JSONStats{},
	}

	// Store the start time in the service state
	s.mu.Lock()
	s.parseStartTime = startTime
	s.mu.Unlock()

	if err := s.metadataService.CreateParseRecord(ctx, parseHistory); err != nil {
		logger.Error("Failed to create parse history", "error", err)
		return nil, nil
	}

	return parseHistory, logger
}

// updateParseHistoryError updates parse history with error information
func (s *VaultService) updateParseHistoryError(ctx context.Context, parseHistory *models.ParseHistory, err error) {
	errorStr := err.Error()
	parseHistory.Status = models.ParseStatusFailed
	parseHistory.Error = &errorStr
	completedAt := time.Now()
	parseHistory.CompletedAt = &completedAt

	// Update the status and error message in one call
	if updateErr := s.metadataService.UpdateParseStatus(ctx, parseHistory.ID, parseHistory.Status, &errorStr); updateErr != nil {
		s.logger().Error("Failed to update parse status on error", "error", updateErr)
	}
}

// executeParsePipeline runs the main parsing pipeline
func (s *VaultService) executeParsePipeline(ctx context.Context, parseHistory *models.ParseHistory, logger *slog.Logger) error {
	// Pull latest from Git
	vaultPath, err := s.pullLatestVault(ctx, logger)
	if err != nil {
		return fmt.Errorf("failed to pull vault: %w", err)
	}

	// Parse vault files
	parseResult, err := s.parseVaultFiles(ctx, vaultPath, logger)
	if err != nil {
		return fmt.Errorf("failed to parse vault: %w", err)
	}

	// Build graph structure
	graph, err := s.buildGraph(ctx, parseResult, logger)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	// Store graph in database
	if err := s.storeGraphInDatabase(ctx, parseResult, graph, logger); err != nil {
		return fmt.Errorf("failed to store graph: %w", err)
	}

	// Update parse history stats
	s.updateParseStats(parseHistory, parseResult, graph)
	return nil
}

// pullLatestVault pulls the latest changes from Git
func (s *VaultService) pullLatestVault(ctx context.Context, logger *slog.Logger) (string, error) {
	logger.Info("Pulling latest changes from git")
	if err := s.gitManager.Pull(ctx); err != nil {
		return "", err
	}

	vaultPath := s.gitManager.GetLocalPath()
	logger.Info("Git pull completed", "vault_path", vaultPath)
	return vaultPath, nil
}

// parseVaultFiles parses all markdown files in the vault
func (s *VaultService) parseVaultFiles(ctx context.Context, vaultPath string, logger *slog.Logger) (*vault.ParseResult, error) {
	parser := s.createParser(vaultPath)
	logger.Info("Starting vault parsing")

	parseResult, err := parser.ParseVault()
	if err != nil {
		return nil, err
	}

	// Count total links
	totalLinks := 0
	for _, file := range parseResult.Files {
		totalLinks += len(file.Links)
	}

	logger.Info("Vault parsing completed",
		"files_parsed", len(parseResult.Files),
		"links_found", totalLinks)

	// Update progress
	s.updateProgress(&models.ParseProgress{
		TotalFiles:     len(parseResult.Files),
		ProcessedFiles: len(parseResult.Files),
		NodesCreated:   0,
		EdgesCreated:   0,
		ErrorCount:     len(parseResult.UnresolvedLinks),
	})

	return parseResult, nil
}

// buildGraph builds the graph structure from parsed files
func (s *VaultService) buildGraph(ctx context.Context, parseResult *vault.ParseResult, logger *slog.Logger) (*vault.Graph, error) {
	graphBuilder, err := s.createGraphBuilder()
	if err != nil {
		return nil, err
	}

	logger.Info("Building graph from parsed data")
	graph, err := graphBuilder.BuildGraph(parseResult)
	if err != nil {
		return nil, err
	}

	logger.Info("Graph building completed",
		"nodes", len(graph.Nodes),
		"edges", len(graph.Edges),
		"orphans", graph.Stats.OrphanedNodes,
		"unresolved_links", graph.Stats.UnresolvedLinks)

	return graph, nil
}

// storeGraphInDatabase stores the graph in the database within a transaction
func (s *VaultService) storeGraphInDatabase(ctx context.Context, parseResult *vault.ParseResult, graph *vault.Graph, logger *slog.Logger) error {
	return db.WithTransaction(s.db, ctx, func(tx *sqlx.Tx) error {
		// Clear existing data
		if err := s.clearExistingGraphData(ctx, tx, logger); err != nil {
			return err
		}

		// Store nodes
		if err := s.storeNodes(ctx, tx, graph.Nodes, parseResult, logger); err != nil {
			return err
		}

		// Store edges
		if err := s.storeEdges(ctx, tx, graph.Edges, parseResult, graph.Nodes, logger); err != nil {
			return err
		}

		// Update metadata
		if err := s.updateVaultMetadata(tx, ctx, logger); err != nil {
			return err
		}

		return nil
	})
}

// clearExistingGraphData removes all existing nodes and edges
func (s *VaultService) clearExistingGraphData(ctx context.Context, tx *sqlx.Tx, logger *slog.Logger) error {
	logger.Info("Clearing existing graph data")

	// IMPORTANT: Never delete node_positions table - user-saved positions must persist
	// Only clear edges and nodes, in that order due to foreign key constraints

	// Delete all edges first (due to foreign key constraints)
	if _, err := tx.ExecContext(ctx, "DELETE FROM edges"); err != nil {
		return fmt.Errorf("failed to clear edges: %w", err)
	}

	// Delete all nodes
	if _, err := tx.ExecContext(ctx, "DELETE FROM nodes"); err != nil {
		return fmt.Errorf("failed to clear nodes: %w", err)
	}

	// Clear unresolved links from previous parses
	if _, err := tx.ExecContext(ctx, "DELETE FROM unresolved_links"); err != nil {
		// Log warning but don't fail - table might not have any data
		logger.Warn("Failed to clear unresolved_links", "error", err)
	}

	return nil
}

// storeNodes stores nodes in batches
func (s *VaultService) storeNodes(ctx context.Context, tx *sqlx.Tx, nodes []models.VaultNode, parseResult *vault.ParseResult, logger *slog.Logger) error {
	logger.Info("Storing nodes in batches")
	batchSize := 1000

	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}

		batch := nodes[i:end]
		if err := s.nodeService.CreateNodeBatchTx(tx, ctx, batch); err != nil {
			return fmt.Errorf("failed to create node batch: %w", err)
		}

		s.updateProgress(&models.ParseProgress{
			TotalFiles:     len(parseResult.Files),
			ProcessedFiles: len(parseResult.Files),
			NodesCreated:   end,
			EdgesCreated:   0,
			ErrorCount:     len(parseResult.UnresolvedLinks),
		})

		logger.Debug("Stored node batch", "batch_start", i, "batch_end", end)
	}

	return nil
}

// storeEdges stores edges in batches
func (s *VaultService) storeEdges(ctx context.Context, tx *sqlx.Tx, edges []models.VaultEdge, parseResult *vault.ParseResult, nodes []models.VaultNode, logger *slog.Logger) error {
	logger.Info("Storing edges in batches")
	batchSize := 1000

	for i := 0; i < len(edges); i += batchSize {
		end := i + batchSize
		if end > len(edges) {
			end = len(edges)
		}

		batch := edges[i:end]
		if err := s.edgeService.CreateEdgeBatchTx(tx, ctx, batch); err != nil {
			return fmt.Errorf("failed to create edge batch: %w", err)
		}

		s.updateProgress(&models.ParseProgress{
			TotalFiles:     len(parseResult.Files),
			ProcessedFiles: len(parseResult.Files),
			NodesCreated:   len(nodes),
			EdgesCreated:   end,
			ErrorCount:     len(parseResult.UnresolvedLinks),
		})

		logger.Debug("Stored edge batch", "batch_start", i, "batch_end", end)
	}

	return nil
}

// updateVaultMetadata updates the last parse timestamp
func (s *VaultService) updateVaultMetadata(tx repository.Executor, ctx context.Context, logger *slog.Logger) error {
	logger.Info("Updating vault metadata")
	metadata := &models.VaultMetadata{
		Key:       "last_parse",
		Value:     time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now(),
	}
	return s.metadataService.SetMetadataTx(tx, ctx, metadata)
}

// updateParseStats updates the parse history with final statistics
func (s *VaultService) updateParseStats(parseHistory *models.ParseHistory, parseResult *vault.ParseResult, graph *vault.Graph) {
	// Check for nil inputs to prevent panic
	if parseHistory == nil {
		s.logger().Error("Cannot update stats for nil parse history")
		return
	}
	if parseResult == nil || graph == nil {
		s.logger().Error("Cannot update stats with nil parse result or graph")
		return
	}

	duration := time.Since(parseHistory.StartedAt)
	stats := models.ParseStats{
		TotalFiles:      len(parseResult.Files),
		ParsedFiles:     len(parseResult.Files),
		TotalNodes:      len(graph.Nodes),
		TotalEdges:      len(graph.Edges),
		DurationMS:      duration.Milliseconds(),
		UnresolvedLinks: graph.Stats.UnresolvedLinks,
	}
	parseHistory.Stats = models.JSONStats(stats)
}

// finalizeParseHistory marks the parse as completed
func (s *VaultService) finalizeParseHistory(ctx context.Context, parseHistory *models.ParseHistory, logger *slog.Logger) {
	completedAt := time.Now()
	parseHistory.CompletedAt = &completedAt
	parseHistory.Status = models.ParseStatusCompleted

	if err := s.metadataService.UpdateParseStatus(ctx, parseHistory.ID, parseHistory.Status, nil); err != nil {
		logger.Error("Failed to update parse status", "error", err)
	}

	stats := parseHistory.Stats.ToParseStats()
	logger.Info("Vault parsing completed successfully",
		"duration", time.Since(parseHistory.StartedAt),
		"nodes", stats.TotalNodes,
		"edges", stats.TotalEdges)
}

// GetParseStatus returns the current parse status with progress information
func (s *VaultService) GetParseStatus(ctx context.Context) (*models.ParseStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If currently parsing, return in-progress status with progress
	if s.isParsing && s.currentParseID != "" {
		history := &models.ParseHistory{
			ID:        s.currentParseID,
			Status:    models.ParseStatusRunning,
			StartedAt: s.parseStartTime, // Use the actual parse start time
		}

		status := models.NewParseStatusFromHistory(history)
		if s.parseProgress != nil {
			status.Progress = s.parseProgress
		}
		return status, nil
	}

	// Otherwise, get latest parse history from metadata service
	history, err := s.GetLatestParseHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest parse history: %w", err)
	}

	return models.NewParseStatusFromHistory(history), nil
}

// GetLatestParseHistory retrieves the most recent parse history record
func (s *VaultService) GetLatestParseHistory(ctx context.Context) (*models.ParseHistory, error) {
	return s.metadataService.GetLatestParse(ctx)
}

// IsParseInProgress checks if a parse operation is currently running
// Returns (inProgress, parseID, error) where parseID is empty if no parse is running
func (s *VaultService) IsParseInProgress(ctx context.Context) (bool, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.isParsing, s.currentParseID, nil
}

// WaitForParse waits for any active parse operation to complete or timeout
// Returns nil if parse completes successfully, error if parse fails or timeout occurs
func (s *VaultService) WaitForParse(ctx context.Context, timeout time.Duration) error {
	// Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Polling interval
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Track the initial parse ID if there's one running
	_, initialParseID, _ := s.IsParseInProgress(ctx)
	hasSeenParse := initialParseID != ""

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for parse timed out after %v", timeout)
		case <-ticker.C:
			// Check if parse is still in progress
			inProgress, currentParseID, err := s.IsParseInProgress(ctx)
			if err != nil {
				return fmt.Errorf("failed to check parse progress: %w", err)
			}

			// Track if we've seen a parse running
			if currentParseID != "" {
				hasSeenParse = true
			}

			// If no parse is running, check the latest status
			if !inProgress {
				// If we never saw a parse running and there's no parse history, return success
				if !hasSeenParse {
					_, err := s.GetParseStatus(ctx)
					if err != nil && strings.Contains(err.Error(), "not found") {
						return nil
					}
				}

				// Get the latest parse status
				status, err := s.GetParseStatus(ctx)
				if err != nil {
					return fmt.Errorf("failed to get parse status: %w", err)
				}

				// Check if the last parse failed
				if status.Status == "failed" && status.Error != "" {
					return fmt.Errorf("parse failed: %s", status.Error)
				}

				// Parse completed successfully
				return nil
			}

			// Log progress if parse is still running
			if currentParseID != "" {
				s.logger().Debug("Waiting for parse to complete", "parse_id", currentParseID)
			}
		}
	}
}

// GetVaultPath retrieves the vault path from metadata
func (s *VaultService) GetVaultPath(ctx context.Context) (string, error) {
	// Get the vault path metadata
	metadata, err := s.metadataService.GetMetadata(ctx, "vault_path")
	if err != nil {
		return "", err
	}
	if metadata == nil {
		return "", fmt.Errorf("vault path not found in metadata")
	}
	return metadata.Value, nil
}

// Helper methods

// updateProgress updates the current parse progress (thread-safe)
func (s *VaultService) updateProgress(progress *models.ParseProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parseProgress = progress
}

// setParsingState updates the parsing state (thread-safe)
func (s *VaultService) setParsingState(isParsing bool, parseID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isParsing = isParsing
	s.currentParseID = parseID
	if !isParsing {
		s.parseProgress = nil
		s.parseStartTime = time.Time{} // Reset to zero value
	}
}

// createParser creates a configured vault parser
func (s *VaultService) createParser(vaultPath string) *vault.Parser {
	concurrency := s.config.Graph.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 4 // Default
	}

	batchSize := s.config.Graph.BatchSize
	if batchSize <= 0 {
		batchSize = 100 // Default
	}

	return vault.NewParser(vaultPath, concurrency, batchSize)
}

// createGraphBuilder creates a configured graph builder
func (s *VaultService) createGraphBuilder() (*vault.GraphBuilder, error) {
	// Create node classifier from config
	classifier, err := vault.NewNodeClassifierFromConfig(&s.config.Graph.NodeClassification)
	if err != nil {
		return nil, fmt.Errorf("failed to create node classifier: %w", err)
	}

	// Create graph builder config
	builderConfig := vault.GraphBuilderConfig{
		DefaultWeight: 1.0,    // Default edge weight
		SkipOrphans:   false,  // Include nodes without connections
	}

	return vault.NewGraphBuilder(classifier, builderConfig), nil
}

// logger returns a logger with service context
func (s *VaultService) logger() *slog.Logger {
	return slog.Default().With("service", "vault")
}
