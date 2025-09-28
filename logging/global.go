package logging

import (
	"log/slog"
	"os"
)

type LoggingService struct {
	Logger *slog.Logger
}

var DefaultLoggingService *LoggingService

// InitLogger initializes the global logger instance
func InitLogger(logDir string) {
	DefaultLoggingService = &LoggingService{
		Logger: SetupLogger(logDir),
	}
	slog.SetDefault(DefaultLoggingService.Logger)
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
