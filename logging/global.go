package logging

import (
	"flag"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

type LoggingService struct {
	Logger         *slog.Logger
	RotatingLogger *RotatingLogger
}

var (
	DefaultLoggingService *LoggingService
	initOnce              sync.Once
	resetMu               sync.Mutex // Protects test-only reset
)

// InitLogger initializes global logger instance
func InitLogger(logDir string) {
	InitLoggerWithRetention(logDir, 4) // Default 4 weeks retention
}

// InitLoggerWithRetention initializes global logger with custom retention
func InitLoggerWithRetention(logDir string, retentionWeeks int) {
	InitLoggerWithRetentionAndSize(logDir, retentionWeeks, 100*1024*1024) // Default 100MB
}

// InitLoggerWithRetentionAndSize initializes the global logger with custom retention and size limit
func InitLoggerWithRetentionAndSize(logDir string, retentionWeeks int, maxFileSize int64) {
	initOnce.Do(func() {
		doInit(logDir, retentionWeeks, maxFileSize)
	})
}

// doInit contains the actual initialization logic (extracted for reuse by ResetForTest)
func doInit(logDir string, retentionWeeks int, maxFileSize int64) {
	// Handle empty log directory (common in tests)
	if logDir == "" {
		logDir = "logs" // Default directory
	}

	// Detect if running tests by checking for Go's test flags
	// All Go tests set these flags, even if not explicitly passed
	testV := flag.CommandLine.Lookup("test.v")
	testRun := flag.CommandLine.Lookup("test.run")
	isTest := testV != nil || testRun != nil

	// Determine log level: suppress console output during tests by default
	var consoleLevel slog.Level
	if isTest {
		consoleLevel = slog.LevelError // Only show errors during tests by default
		// Override to Info if -v is passed (for debugging)
		if testing.Verbose() {
			consoleLevel = slog.LevelInfo // Show all logs when verbose
		}
	} else {
		consoleLevel = slog.LevelInfo // Normal operation
	}

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If we can't create logs directory, just log to console
		consoleLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: consoleLevel,
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

	// Initialize rotating logger
	rotatingLogger.mu.Lock()
	err := rotatingLogger.doRotate(getWeekKey(time.Now()))
	rotatingLogger.mu.Unlock()
	if err != nil {
		// Fallback to console logger if rotation fails
		consoleLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: consoleLevel,
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
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		defer close(rotatingLogger.cleanupDone)

		for {
			select {
			case <-rotatingLogger.ctx.Done():
				// Context cancelled, exit gracefully
				return
			case <-ticker.C:
				if err := rotatingLogger.cleanupOldLogs(); err != nil {
					slog.Warn("Failed to cleanup old logs", "error", err)
				}
			}
		}
	}()

	// Create multi-handler that writes to both console and rotating file
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: consoleLevel,
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
}

// ResetForTest resets the global logger - ONLY for testing
// Provides proper test isolation by cleaning up resources and reinitializing
// Must be called with test.TempDir() and t.Cleanup() for automatic cleanup
func ResetForTest(t *testing.T, logDir string, retentionWeeks int, maxFileSize int64) {
	t.Helper() // Mark as test helper for better error reporting

	// Close existing logger to prevent resource leaks (goroutines, file handles)
	Close()

	// Clear service reference
	DefaultLoggingService = nil

	// Reset sync.Once to allow reinitialization
	initOnce = sync.Once{}

	// Reinitialize with new settings
	doInit(logDir, retentionWeeks, maxFileSize)

	// Register cleanup to run after test completes
	// This is key improvement over save/restore pattern
	t.Cleanup(func() {
		Close()
		DefaultLoggingService = nil
	})
}

// Close closes logging service and cleans up resources
func Close() {
	if DefaultLoggingService != nil && DefaultLoggingService.RotatingLogger != nil {
		if err := DefaultLoggingService.RotatingLogger.Close(); err != nil {
			slog.Warn("Failed to close rotating logger", "error", err)
		}
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
