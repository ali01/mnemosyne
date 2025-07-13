// Package main is the entry point for the Mnemosyne HTTP server
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/ali01/mnemosyne/internal/api"
	"github.com/ali01/mnemosyne/internal/config"
	"github.com/ali01/mnemosyne/internal/db"
	"github.com/ali01/mnemosyne/internal/git"
	"github.com/ali01/mnemosyne/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Server panic recovered: %v", r)
			log.Printf("Stack trace:\n%s", debug.Stack())
			os.Exit(1)
		}
	}()

	// Load configuration
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.LoadFromYAML(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	dbConfig := db.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	database, err := db.Connect(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Database will be closed after all services are stopped

	// Initialize database schema
	if err := initializeDatabase(database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create Git manager
	gitManager, err := git.NewManager(&cfg.Git)
	if err != nil {
		log.Fatalf("Failed to create git manager: %v", err)
	}

	// Initialize git repository
	ctx := context.Background()
	if err := gitManager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize git repository: %v", err)
	}
	// Git manager will be stopped after services are stopped

	// Create services
	nodeService := service.NewNodeService(database)
	edgeService := service.NewEdgeService(database)
	positionService := service.NewPositionService(database)
	metadataService := service.NewMetadataService(database)

	// Create vault service
	vaultService := service.NewVaultService(
		cfg,
		gitManager,
		nodeService,
		edgeService,
		metadataService,
		database,
	)

	// Initialize router with services
	router := gin.Default()
	api.SetupRoutesWithServices(router, nodeService, edgeService, positionService, vaultService, cfg)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,
	}

	// Set up shutdown signal channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("HTTP server panic recovered: %v", r)
				log.Printf("Stack trace:\n%s", debug.Stack())
				// Signal main goroutine to shut down
				quit <- syscall.SIGTERM
			}
		}()

		log.Printf("Starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	// Proper shutdown sequence: stop services before their dependencies
	log.Println("Stopping services...")

	// Wait for any active parse operations to complete
	if vaultService != nil {
		if inProgress, parseID, _ := vaultService.IsParseInProgress(context.Background()); inProgress {
			log.Printf("Waiting for parse %s to complete...", parseID)
			// Give parse operations up to 30 seconds to complete
			if err := vaultService.WaitForParse(context.Background(), 30*time.Second); err != nil {
				log.Printf("Warning: Parse did not complete cleanly: %v", err)
			}
		}
	}

	// Stop git manager after services are done using it
	if gitManager != nil {
		log.Println("Stopping git manager...")
		gitManager.Stop()
	}

	// Close database connection last
	if database != nil {
		log.Println("Closing database connection...")
		if err := database.Close(); err != nil {
			log.Printf("Warning: Error closing database: %v", err)
		}
	}

	log.Println("Server exiting")
}

// initializeDatabase creates the database schema
func initializeDatabase(database *sqlx.DB) error {
	// Read and execute schema
	schemaPath := "internal/db/schema.sql"
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	if err := db.ExecuteSchema(database, string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}
