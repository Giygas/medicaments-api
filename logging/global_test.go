package logging

import (
	"log/slog"
	"testing"

	"github.com/giygas/medicaments-api/config"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"invalid", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetConsoleLogLevel(t *testing.T) {
	tests := []struct {
		name        string
		env         config.Environment
		logLevelStr string
		verbose     bool
		expected    slog.Level
	}{
		{"dev defaults to info", config.EnvDevelopment, "", false, slog.LevelInfo},
		{"test quiet defaults to error", config.EnvTest, "", false, slog.LevelError},
		{"test verbose defaults to info", config.EnvTest, "", true, slog.LevelInfo},
		{"prod defaults to warn", config.EnvProduction, "", false, slog.LevelWarn},
		{"staging defaults to warn", config.EnvStaging, "", false, slog.LevelWarn},
		{"prod with debug override", config.EnvProduction, "debug", false, slog.LevelDebug},
		{"dev with error override", config.EnvDevelopment, "error", false, slog.LevelError},
		{"test with debug override (ignored)", config.EnvTest, "debug", false, slog.LevelError},
		{"test with debug override (ignored) verbose", config.EnvTest, "debug", true, slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConsoleLogLevel(tt.env, tt.logLevelStr, tt.verbose)
			if got != tt.expected {
				t.Errorf("GetConsoleLogLevel(%v, %q, %v) = %v, want %v", tt.env, tt.logLevelStr, tt.verbose, got, tt.expected)
			}
		})
	}
}

func TestGetFileLogLevel(t *testing.T) {
	got := GetFileLogLevel()
	if got != slog.LevelDebug {
		t.Errorf("GetFileLogLevel() = %v, want %v", got, slog.LevelDebug)
	}
}
