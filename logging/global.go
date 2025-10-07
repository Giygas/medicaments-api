package logging

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type LoggingService struct {
	Logger         *slog.Logger
	RotatingLogger *RotatingLogger
}

var DefaultLoggingService *LoggingService

// InitLogger initializes the global logger instance
func InitLogger(logDir string) {
	InitLoggerWithRetention(logDir, 4) // Default 4 weeks retention
}

// InitLoggerWithRetention initializes the global logger with custom retention
func InitLoggerWithRetention(logDir string, retentionWeeks int) {
	InitLoggerWithRetentionAndSize(logDir, retentionWeeks, 100*1024*1024) // Default 100MB
}

// InitLoggerWithRetentionAndSize initializes the global logger with custom retention and size limit
func InitLoggerWithRetentionAndSize(logDir string, retentionWeeks int, maxFileSize int64) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If we can't create logs directory, just log to console
		consoleLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		consoleLogger.Error("Failed to create logs directory", "error", err)
		DefaultLoggingService = &LoggingService{
			Logger:         consoleLogger,
			RotatingLogger: nil,
		}
		slog.SetDefault(consoleLogger)
		return
	}

	// Create rotating logger with size limit
	rotatingLogger := NewRotatingLoggerWithSizeLimit(logDir, retentionWeeks, maxFileSize)

	// Initialize the rotating logger
	if err := rotatingLogger.rotateIfNeeded(); err != nil {
		// Fallback to console logger if rotation fails
		consoleLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		consoleLogger.Error("Failed to initialize rotating logger", "error", err)
		DefaultLoggingService = &LoggingService{
			Logger:         consoleLogger,
			RotatingLogger: rotatingLogger,
		}
		slog.SetDefault(consoleLogger)
		return
	}

	// Start cleanup goroutine with proper cancellation
	go func() {
		defer close(rotatingLogger.cleanupDone)
		ticker := time.NewTicker(24 * time.Hour) // Check daily
		defer ticker.Stop()

		for {
			select {
			case <-rotatingLogger.ctx.Done():
				// Context cancelled, exit gracefully
				return
			case <-ticker.C:
				rotatingLogger.cleanupOldLogs()
			}
		}
	}()

	// Create multi-handler that writes to both console and rotating file
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	fileHandler := slog.NewJSONHandler(rotatingLogger, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Combine handlers - write to both
	multiHandler := &multiHandler{
		handlers: []slog.Handler{consoleHandler, fileHandler},
	}

	logger := slog.New(multiHandler)

	DefaultLoggingService = &LoggingService{
		Logger:         logger,
		RotatingLogger: rotatingLogger,
	}
	slog.SetDefault(logger)

	// Setup graceful shutdown to close log file
	setupGracefulShutdown()
}

// setupGracefulShutdown ensures log files are properly closed on exit
func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		if DefaultLoggingService != nil && DefaultLoggingService.RotatingLogger != nil {
			DefaultLoggingService.RotatingLogger.Close()
		}
		os.Exit(0)
	}()
}

// Close closes the logging service and cleans up resources
func Close() {
	if DefaultLoggingService != nil && DefaultLoggingService.RotatingLogger != nil {
		DefaultLoggingService.RotatingLogger.Close()
	}
}

// Package-level functions for direct access

func Info(msg string, args ...any) {
	if DefaultLoggingService == nil || DefaultLoggingService.Logger == nil {
		// Fallback to console logger if not initialized
		fallback := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		fallback.Info(msg, args...)
		return
	}
	DefaultLoggingService.Logger.Info(msg, args...)
}

func Error(msg string, args ...any) {
	if DefaultLoggingService == nil || DefaultLoggingService.Logger == nil {
		// Fallback to console logger if not initialized
		fallback := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
		fallback.Error(msg, args...)
		return
	}
	DefaultLoggingService.Logger.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	if DefaultLoggingService == nil || DefaultLoggingService.Logger == nil {
		// Fallback to console logger if not initialized
		fallback := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		}))
		fallback.Warn(msg, args...)
		return
	}
	DefaultLoggingService.Logger.Warn(msg, args...)
}

func Debug(msg string, args ...any) {
	if DefaultLoggingService == nil || DefaultLoggingService.Logger == nil {
		// Fallback to console logger if not initialized
		fallback := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		fallback.Debug(msg, args...)
		return
	}
	DefaultLoggingService.Logger.Debug(msg, args...)
}
