package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/scheduler"
	"github.com/giygas/medicaments-api/server"
	"github.com/joho/godotenv"
)

func main() {
	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	// Initialize structured logging
	logging.InitLogger("logs")

	// Load environment variables
	if err := loadEnvironment(); err != nil {
		logging.Error("Failed to load environment", "error", err)
		os.Exit(1)
	}

	// Log configuration on startup
	logging.Info("Configuration loaded successfully",
		"port", cfg.Port,
		"address", cfg.Address,
		"env", cfg.Env,
		"log_level", cfg.LogLevel,
		"max_request_body", cfg.MaxRequestBody,
		"max_header_size", cfg.MaxHeaderSize)

	// Initialize data container and parser
	dataContainer := data.NewDataContainer()
	parser := medicamentsparser.NewMedicamentsParser()

	// Initialize and start scheduler with dependency injection
	sched := scheduler.NewScheduler(dataContainer, parser)
	if err := sched.Start(); err != nil {
		logging.Error("Failed to start scheduler", "error", err)
		os.Exit(1)
	}
	defer sched.Stop()

	// Initialize and start server
	srv := server.NewServer(cfg, dataContainer)

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			logging.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Block until a signal is received
	<-quit
	logging.Info("Shutting down server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		logging.Error("Server shutdown failed", "error", err)
		os.Exit(1)
	}

	logging.Info("Server shutdown complete")
}

// loadEnvironment loads environment variables from .env file
func loadEnvironment() error {
	// Try to load .env file
	if err := godotenv.Load(); err != nil {
		// If failed, try loading from executable directory
		ex, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		exPath := filepath.Dir(ex)
		if err := os.Chdir(exPath); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}

		// Try again after changing directory
		if err := godotenv.Load(); err != nil {
			logging.Warn("Could not load .env file", "error", err)
		}
	}

	return nil
}
