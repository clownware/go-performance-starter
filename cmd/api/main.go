package main

import (
	"context"
	"fmt"
	"log/slog"
	"github.com/joho/godotenv"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/go-alpine-saas-starter/internal/config"
	_ "github.com/yourusername/go-alpine-saas-starter/internal/database" // Keep for sqlc generated types, alias not needed directly here
	"github.com/yourusername/go-alpine-saas-starter/internal/middleware"
	"github.com/yourusername/go-alpine-saas-starter/internal/server"
)

func main() {
	slog.Info("Starting Go Alpine SaaS Starter...")

	// Set global log level to Debug to see diagnostic messages
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelDebug)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	// Manually load .env and set environment variables
	envMap, err := godotenv.Read() // Read .env into a map
	if err != nil {
		slog.Warn("Error reading .env file, relying on existing environment variables", "error", err)
	} else {
		for key, value := range envMap {
			if os.Getenv(key) == "" { // Only set if not already set in the OS environment
				err := os.Setenv(key, value)
				if err != nil {
					// Log setting error, but continue
					slog.Warn("Error setting env variable from .env", "key", key, "error", err)
				}
			}
		}
		slog.Info(".env file processed successfully")
	}

	// Create context that listens for termination signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Info("Shutdown signal received")
		cancel()
	}()

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// TODO: Set up logger (Phase 0) - for now use standard log
	slog.Info("Configuration loaded successfully")

	// Connect to the database
	slog.Info("Connecting to database...")
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Database connection established successfully")

	// Note: The following line is commented out because sqlc code generation hasn't been run yet
	// When sqlc is run, it will generate the New() function to create queries
	// queries := database.New(db)
	// Repositories will be created here in later phases
	_ = "queries will be used in Phase 3-4"

	// At this point, we would set up repositories and handlers, but that's for Phase 3-4
	// We'll just create a simple endpoint to verify our setup

	// Initialize the router with middleware and routes
	srv, err := server.New(cfg, db) // Pass db connection to server
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}
	
	// Start memory metrics collector (updates every 30 seconds)
	middleware.StartMemoryMetricsCollector(30 * time.Second)
	slog.Info("Memory metrics collector started")

	// Set up the HTTP server
	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv,
	}

	// Start the server in a goroutine
	go func() {
		slog.Info("Server listening on " + addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Error starting server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for termination signal
	<-ctx.Done()

	// Graceful shutdown
	slog.Info("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped gracefully")
}
