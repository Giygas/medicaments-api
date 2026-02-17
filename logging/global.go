package logging

import (
	"flag"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/config"
)

type LoggingService struct {
	Logger         *slog.Logger
	RotatingLogger *RotatingLogger
}

var (
	DefaultLoggingService *LoggingService
	initOnce              sync.Once
	fallbackLogger        *slog.Logger
)

func init() {
	fallbackLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

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
	env := config.DetectEnvironment()
	InitLoggerWithEnvironment(logDir, env, "", retentionWeeks, maxFileSize)
}

// InitLoggerWithEnvironment initializes logger with explicit environment control
func InitLoggerWithEnvironment(logDir string, env config.Environment, logLevelStr string, retentionWeeks int, maxFileSize int64) {
	initOnce.Do(func() {
		doInit(logDir, env, logLevelStr, retentionWeeks, maxFileSize)
	})
}

// parseLogLevel converts a string log level to slog.Level
// Returns slog.LevelInfo as a safe default for invalid values
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo // Safe default
	}
}

// GetConsoleLogLevel returns the appropriate console log level for an environment
// If logLevelStr is provided and not in test environment, it overrides the environment default
func GetConsoleLogLevel(env config.Environment, logLevelStr string, isVerbose bool) slog.Level {
	// Use LOG_LEVEL override if provided (except in test environment)
	if logLevelStr != "" && env != config.EnvTest {
		return parseLogLevel(logLevelStr)
	}

	switch env {
	case config.EnvDevelopment:
		return slog.LevelInfo // Full output in dev
	case config.EnvTest:
		if isVerbose {
			return slog.LevelInfo // Verbose tests show all logs
		}
		return slog.LevelError // Errors only in tests
	case config.EnvStaging, config.EnvProduction:
		return slog.LevelWarn // WARN and errors in staging/prod
	default:
		return slog.LevelInfo // Default to info
	}
}

// GetFileLogLevel returns the file log level
// Files always log at DEBUG level to capture all information for debugging
func GetFileLogLevel() slog.Level {
	return slog.LevelDebug
}

// doInit contains the actual initialization logic (extracted for reuse by ResetForTest)
func doInit(logDir string, env config.Environment, logLevelStr string, retentionWeeks int, maxFileSize int64) {
	// Handle empty log directory (common in tests)
	if logDir == "" {
		logDir = "logs" // Default directory
	}

	// Determine if running tests (for verbose mode)
	// Use recover to handle calls to testing.Verbose() before flags are parsed
	isVerbose := false
	if flag := flag.CommandLine.Lookup("test.v"); flag != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// testing.Verbose() called before Parse() - default to false (ERROR-only in tests)
					isVerbose = false
				}
			}()
			isVerbose = testing.Verbose()
		}()
	}

	// Determine console and file log levels based on environment and LOG_LEVEL
	consoleLevel := GetConsoleLogLevel(env, logLevelStr, isVerbose)
	fileLevel := GetFileLogLevel()

	// Log detected environment for debugging (skip in tests to avoid noise)
	if env != config.EnvTest {
		consoleLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: consoleLevel,
		}))
		consoleLogger.Info("Initializing logger",
			"environment", env.String(),
			"console_level", consoleLevel.String(),
			"file_level", fileLevel.String(),
			"log_directory", logDir)
	}

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0750); err != nil {
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

	// Start cleanup goroutine with proper cancellation.
	//
	// Shutdown protocol:
	// 1. Close() calls rotatingLogger.cancel() to signal shutdown
	// 2. This goroutine receives on rotatingLogger.ctx.Done() and returns
	// 3. defer close(cleanupDone) signals that cleanup is finished
	// 4. Close() waits on cleanupDone with configurable timeout (default 5s)
	// 5. Use SetShutdownTimeout() in tests to avoid slow test execution
	rotatingLogger.cleanupStarted = true
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
		Level: fileLevel,
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
func ResetForTest(t *testing.T, logDir string, env config.Environment, logLevelStr string, retentionWeeks int, maxFileSize int64) {
	t.Helper() // Mark as test helper for better error reporting

	// Close existing logger to prevent resource leaks (goroutines, file handles)
	Close()

	// Clear service reference
	DefaultLoggingService = nil

	// Reset sync.Once to allow reinitialization
	initOnce = sync.Once{}

	// Reinitialize with new settings
	doInit(logDir, env, logLevelStr, retentionWeeks, maxFileSize)

	// Set short shutdown timeout for tests to avoid slow test execution
	if DefaultLoggingService != nil && DefaultLoggingService.RotatingLogger != nil {
		DefaultLoggingService.RotatingLogger.SetShutdownTimeout(100 * time.Millisecond)
	}

	// Register cleanup to run after test completes
	// This is key improvement over save/restore pattern
	t.Cleanup(func() {
		Close()
		DefaultLoggingService = nil
	})
}

// ResetForBenchmark resets the global logger - ONLY for benchmarks
// Provides proper benchmark isolation by cleaning up resources and reinitializing
func ResetForBenchmark(b *testing.B, logDir string, env config.Environment, logLevelStr string, retentionWeeks int, maxFileSize int64) {
	b.Helper() // Mark as benchmark helper for better error reporting

	// Close existing logger to prevent resource leaks (goroutines, file handles)
	Close()

	// Clear service reference
	DefaultLoggingService = nil

	// Reset sync.Once to allow reinitialization
	initOnce = sync.Once{}

	// Reinitialize with new settings
	doInit(logDir, env, logLevelStr, retentionWeeks, maxFileSize)

	// Set short shutdown timeout for benchmarks to avoid slow execution
	if DefaultLoggingService != nil && DefaultLoggingService.RotatingLogger != nil {
		DefaultLoggingService.RotatingLogger.SetShutdownTimeout(100 * time.Millisecond)
	}

	// Register cleanup to run after benchmark completes
	b.Cleanup(func() {
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
		fallbackLogger.Info(msg, args...)
		return
	}
	DefaultLoggingService.Logger.Info(msg, args...)
}

func Error(msg string, args ...any) {
	if DefaultLoggingService == nil || DefaultLoggingService.Logger == nil {
		fallbackLogger.Error(msg, args...)
		return
	}
	DefaultLoggingService.Logger.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	if DefaultLoggingService == nil || DefaultLoggingService.Logger == nil {
		fallbackLogger.Warn(msg, args...)
		return
	}
	DefaultLoggingService.Logger.Warn(msg, args...)
}

func Debug(msg string, args ...any) {
	if DefaultLoggingService == nil || DefaultLoggingService.Logger == nil {
		fallbackLogger.Debug(msg, args...)
		return
	}
	DefaultLoggingService.Logger.Debug(msg, args...)
}
