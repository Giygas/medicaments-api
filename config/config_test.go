package config

import (
	"os"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	// Set valid environment variables
	os.Setenv("PORT", "8002")
	os.Setenv("ADDRESS", "127.0.0.1")
	os.Setenv("ENV", "dev")
	os.Setenv("LOG_LEVEL", "info")
	defer cleanupEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Port != "8002" {
		t.Errorf("Expected port 8002, got %s", cfg.Port)
	}
	if cfg.Address != "127.0.0.1" {
		t.Errorf("Expected address 127.0.0.1, got %s", cfg.Address)
	}
	if cfg.Env != "dev" {
		t.Errorf("Expected env dev, got %s", cfg.Env)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected log level info, got %s", cfg.LogLevel)
	}
}

func TestLoadWithDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	os.Unsetenv("PORT")
	os.Unsetenv("ADDRESS")
	os.Unsetenv("ENV")
	os.Unsetenv("LOG_LEVEL")
	defer cleanupEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Port != "8000" {
		t.Errorf("Expected default port 8000, got %s", cfg.Port)
	}
	if cfg.Address != "127.0.0.1" {
		t.Errorf("Expected default address 127.0.0.1, got %s", cfg.Address)
	}
	if cfg.Env != "dev" {
		t.Errorf("Expected default env dev, got %s", cfg.Env)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected default log level info, got %s", cfg.LogLevel)
	}
}

func TestInvalidPort(t *testing.T) {
	// Test invalid port values (excluding empty string since it uses default)
	testCases := []struct {
		port     string
		expected string
	}{
		{"abc", "PORT must be a valid number"},
		{"0", "PORT must be between 1 and 65535"},
		{"65536", "PORT must be between 1 and 65535"},
		{"80", "PORT 80 is privileged"},
	}

	for _, tc := range testCases {
		os.Setenv("PORT", tc.port)
		os.Setenv("ADDRESS", "127.0.0.1")
		os.Setenv("ENV", "dev")
		os.Setenv("LOG_LEVEL", "info")

		_, err := Load()
		if err == nil {
			t.Errorf("Expected error for port %s, got nil", tc.port)
		}
	}
}

func TestInvalidAddress(t *testing.T) {
	// Test invalid address values (excluding empty string since it uses default)
	testCases := []struct {
		address  string
		expected string
	}{
		{"invalid", "ADDRESS must be a valid IP address"},
	}

	for _, tc := range testCases {
		os.Setenv("PORT", "8002")
		os.Setenv("ADDRESS", tc.address)
		os.Setenv("ENV", "dev")
		os.Setenv("LOG_LEVEL", "info")

		_, err := Load()
		if err == nil {
			t.Errorf("Expected error for address %s, got nil", tc.address)
		}
	}
}

func TestInvalidEnv(t *testing.T) {
	// Test invalid env values (excluding empty string since it uses default)
	testCases := []struct {
		env      string
		expected string
	}{
		{"invalid", "ENV must be one of"},
	}

	for _, tc := range testCases {
		os.Setenv("PORT", "8002")
		os.Setenv("ADDRESS", "127.0.0.1")
		os.Setenv("ENV", tc.env)
		os.Setenv("LOG_LEVEL", "info")

		_, err := Load()
		if err == nil {
			t.Errorf("Expected error for env %s, got nil", tc.env)
		}
	}
}

func TestInvalidLogLevel(t *testing.T) {
	// Test invalid log level values (excluding empty string since it uses default)
	testCases := []struct {
		logLevel string
		expected string
	}{
		{"invalid", "LOG_LEVEL must be one of"},
	}

	for _, tc := range testCases {
		os.Setenv("PORT", "8002")
		os.Setenv("ADDRESS", "127.0.0.1")
		os.Setenv("ENV", "dev")
		os.Setenv("LOG_LEVEL", tc.logLevel)

		_, err := Load()
		if err == nil {
			t.Errorf("Expected error for log level %s, got nil", tc.logLevel)
		}
	}
}

func cleanupEnv() {
	os.Unsetenv("PORT")
	os.Unsetenv("ADDRESS")
	os.Unsetenv("ENV")
	os.Unsetenv("LOG_LEVEL")
}
