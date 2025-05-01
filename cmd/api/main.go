package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/go-alpine-saas-starter/internal/config"
	_ "github.com/yourusername/go-alpine-saas-starter/internal/database" // Keep for sqlc generated types, alias not needed directly here
	"github.com/yourusername/go-alpine-saas-starter/internal/server"
)

func main() {
	fmt.Println("Starting Go Alpine SaaS Starter...")

	// Create context that listens for termination signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
	}()

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// TODO: Set up logger (Phase 0) - for now use standard log
	log.Println("Configuration loaded successfully")

	// Connect to the database
	log.Println("Connecting to database...")
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Database connection established successfully")

	// Note: The following line is commented out because sqlc code generation hasn't been run yet
	// When sqlc is run, it will generate the New() function to create queries
	// queries := database.New(db)
	// Repositories will be created here in later phases
	_ = "queries will be used in Phase 3-4"

	// At this point, we would set up repositories and handlers, but that's for Phase 3-4
	// We'll just create a simple endpoint to verify our setup

	// Initialize the router with middleware and routes
	router := server.New()

	// Add health check route
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// Test the database connection
		err := db.Ping(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, "Database connection error: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Status: OK\nDatabase: Connected")
	})

	// Set up the HTTP server
	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Server listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v\n", err)
		}
	}()

	// Wait for termination signal
	<-ctx.Done()

	// Graceful shutdown
	log.Println("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped gracefully")
}
