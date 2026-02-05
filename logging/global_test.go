package logging

import (
	"log/slog"
	"testing"

	"github.com/giygas/medicaments-api/config"
)

func TestGetConsoleLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		env      config.Environment
		verbose  bool
		expected slog.Level
	}{
		{"dev", config.EnvDevelopment, false, slog.LevelInfo},
		{"test quiet", config.EnvTest, false, slog.LevelError},
		{"test verbose", config.EnvTest, true, slog.LevelInfo},
		{"prod", config.EnvProduction, false, slog.LevelWarn},
		{"staging", config.EnvStaging, false, slog.LevelWarn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConsoleLogLevel(tt.env, tt.verbose)
			if got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, got)
			}
		})
	}
}
