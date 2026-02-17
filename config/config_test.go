package config

import (
	"os"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	// Set valid environment variables
	_ = os.Setenv("PORT", "8002")
	_ = os.Setenv("ADDRESS", "127.0.0.1")
	_ = os.Setenv("ENV", "dev")
	_ = os.Setenv("LOG_LEVEL", "info")
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
	if cfg.Env != EnvDevelopment {
		t.Errorf("Expected env dev, got %s", cfg.Env)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected log level info, got %s", cfg.LogLevel)
	}
}

func TestLoadWithDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("ADDRESS")
	_ = os.Unsetenv("ENV")
	_ = os.Unsetenv("LOG_LEVEL")
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
	if cfg.Env != EnvDevelopment {
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
		_ = os.Setenv("PORT", tc.port)
		_ = os.Setenv("ADDRESS", "127.0.0.1")
		_ = os.Setenv("ENV", "dev")
		_ = os.Setenv("LOG_LEVEL", "info")

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
		_ = os.Setenv("PORT", "8002")
		_ = os.Setenv("ADDRESS", tc.address)
		_ = os.Setenv("ENV", "dev")
		_ = os.Setenv("LOG_LEVEL", "info")

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
		_ = os.Setenv("PORT", "8002")
		_ = os.Setenv("ADDRESS", "127.0.0.1")
		_ = os.Setenv("ENV", tc.env)
		_ = os.Setenv("LOG_LEVEL", "info")

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
		_ = os.Setenv("PORT", "8002")
		_ = os.Setenv("ADDRESS", "127.0.0.1")
		_ = os.Setenv("ENV", "dev")
		_ = os.Setenv("LOG_LEVEL", tc.logLevel)

		_, err := Load()
		if err == nil {
			t.Errorf("Expected error for log level %s, got nil", tc.logLevel)
		}
	}
}

func cleanupEnv() {
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("ADDRESS")
	_ = os.Unsetenv("ENV")
	_ = os.Unsetenv("LOG_LEVEL")
	_ = os.Unsetenv("ALLOW_DIRECT_ACCESS")
}

func TestParseEnvironment(t *testing.T) {
	tests := []struct {
		input    string
		expected Environment
		hasError bool
	}{
		{"dev", EnvDevelopment, false},
		{"development", EnvDevelopment, false},
		{"staging", EnvStaging, false},
		{"prod", EnvProduction, false},
		{"production", EnvProduction, false},
		{"test", EnvTest, false},
		{"invalid", EnvDevelopment, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			env, err := ParseEnvironment(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for %s, got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.input, err)
				}
				if env != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, env)
				}
			}
		})
	}
}

func TestEnvironmentString(t *testing.T) {
	tests := []struct {
		env      Environment
		expected string
	}{
		{EnvDevelopment, "dev"},
		{EnvStaging, "staging"},
		{EnvProduction, "prod"},
		{EnvTest, "test"},
	}

	for _, tt := range tests {
		if got := tt.env.String(); got != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, got)
		}
	}
}

func TestAllowDirectAccess(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		expectedValue bool
	}{
		{"ALLOW_DIRECT_ACCESS=true", "true", true},
		{"ALLOW_DIRECT_ACCESS=TRUE", "TRUE", true},
		{"ALLOW_DIRECT_ACCESS=1", "1", true},
		{"ALLOW_DIRECT_ACCESS=false", "false", false},
		{"ALLOW_DIRECT_ACCESS=FALSE", "FALSE", false},
		{"ALLOW_DIRECT_ACCESS=0", "0", false},
		{"ALLOW_DIRECT_ACCESS not set", "", false},
		{"ALLOW_DIRECT_ACCESS invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("PORT", "8002")
			_ = os.Setenv("ADDRESS", "127.0.0.1")
			_ = os.Setenv("ENV", "dev")
			_ = os.Setenv("LOG_LEVEL", "info")
			_ = os.Setenv("ALLOW_DIRECT_ACCESS", tt.envValue)
			defer cleanupEnv()

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if cfg.AllowDirectAccess != tt.expectedValue {
				t.Errorf("Expected AllowDirectAccess=%v for %s, got %v", tt.expectedValue, tt.name, cfg.AllowDirectAccess)
			}
		})
	}
}

func TestValidateAddress_0dot0dot0dot0dot0_WithoutAllowDirectAccess(t *testing.T) {
	cfg := &Config{
		Address:           "0.0.0.0",
		Port:              "8000",
		Env:               EnvDevelopment,
		LogLevel:          "info",
		LogRetentionWeeks: 4,
		MaxLogFileSize:    104857600,
		MaxRequestBody:    1048576,
		MaxHeaderSize:     1048576,
		AllowDirectAccess: false,
	}

	err := validateAddress(cfg)

	if err == nil {
		t.Error("Expected error for 0.0.0.0 when AllowDirectAccess is false")
	}

	expectedMsg := "Set ALLOW_DIRECT_ACCESS=true"
	if err != nil && !containsString(err.Error(), expectedMsg) {
		t.Errorf("Error message should mention ALLOW_DIRECT_ACCESS=true, got: %s", err.Error())
	}
}

func TestValidateAddress_0dot0dot0dot0dot0_WithAllowDirectAccess(t *testing.T) {
	cfg := &Config{
		Address:           "0.0.0.0",
		Port:              "8000",
		Env:               EnvDevelopment,
		LogLevel:          "info",
		LogRetentionWeeks: 4,
		MaxLogFileSize:    104857600,
		MaxRequestBody:    1048576,
		MaxHeaderSize:     1048576,
		AllowDirectAccess: true,
	}

	err := validateAddress(cfg)

	if err != nil {
		t.Errorf("Expected no error for 0.0.0.0 when AllowDirectAccess is true, got: %s", err.Error())
	}
}

func TestValidateAddress_IPv6Any_WithoutAllowDirectAccess(t *testing.T) {
	cfg := &Config{
		Address:           "::",
		Port:              "8000",
		Env:               EnvDevelopment,
		LogLevel:          "info",
		LogRetentionWeeks: 4,
		MaxLogFileSize:    104857600,
		MaxRequestBody:    1048576,
		MaxHeaderSize:     1048576,
		AllowDirectAccess: false,
	}

	err := validateAddress(cfg)

	if err == nil {
		t.Error("Expected error for :: when AllowDirectAccess is false")
	}
}

func TestValidateAddress_IPv6Any_WithAllowDirectAccess(t *testing.T) {
	cfg := &Config{
		Address:           "::",
		Port:              "8000",
		Env:               EnvDevelopment,
		LogLevel:          "info",
		LogRetentionWeeks: 4,
		MaxLogFileSize:    104857600,
		MaxRequestBody:    1048576,
		MaxHeaderSize:     1048576,
		AllowDirectAccess: true,
	}

	err := validateAddress(cfg)

	if err != nil {
		t.Errorf("Expected no error for :: when AllowDirectAccess is true, got: %s", err.Error())
	}
}

func TestValidateAddress_127dot0dot0dot1_AlwaysAllowed(t *testing.T) {
	cfg := &Config{
		Address:           "127.0.0.1",
		Port:              "8000",
		Env:               EnvDevelopment,
		LogLevel:          "info",
		LogRetentionWeeks: 4,
		MaxLogFileSize:    104857600,
		MaxRequestBody:    1048576,
		MaxHeaderSize:     1048576,
		AllowDirectAccess: false,
	}

	err := validateAddress(cfg)

	if err != nil {
		t.Errorf("127.0.0.1 should always be allowed, got error: %s", err.Error())
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
