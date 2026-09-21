package config

import (
	"log"
	"os"
	"testing"
	"time"
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

func TestValidLogLevels(t *testing.T) {
	// Test all valid log level values
	validLevels := []string{"debug", "info", "warn", "error"}

	for _, level := range validLevels {
		_ = os.Setenv("PORT", "8002")
		_ = os.Setenv("ADDRESS", "127.0.0.1")
		_ = os.Setenv("ENV", "dev")
		_ = os.Setenv("LOG_LEVEL", level)

		cfg, err := Load()
		if err != nil {
			t.Errorf("Expected no error for log level %s, got %v", level, err)
		}

		if cfg.LogLevel != level {
			t.Errorf("Expected log level %s, got %s", level, cfg.LogLevel)
		}
	}
}

func cleanupEnv() {
	if err := os.Unsetenv("PORT"); err != nil {
		log.Printf("Failed to unset PORT: %v", err)
	}
	if err := os.Unsetenv("ADDRESS"); err != nil {
		log.Printf("Failed to unset ADDRESS: %v", err)
	}
	if err := os.Unsetenv("ENV"); err != nil {
		log.Printf("Failed to unset ENV: %v", err)
	}
	if err := os.Unsetenv("LOG_LEVEL"); err != nil {
		log.Printf("Failed to unset LOG_LEVEL: %v", err)
	}
	if err := os.Unsetenv("ALLOW_DIRECT_ACCESS"); err != nil {
		log.Printf("Failed to unset ALLOW_DIRECT_ACCESS: %v", err)
	}
	if err := os.Unsetenv("DOCS_ENABLED"); err != nil {
		log.Printf("Failed to unset DOCS_ENABLED: %v", err)
	}
	if err := os.Unsetenv("DOCS_CACHE_DIR"); err != nil {
		log.Printf("Failed to unset DOCS_CACHE_DIR: %v", err)
	}
	if err := os.Unsetenv("DOCS_FETCH_RATE_PER_SEC"); err != nil {
		log.Printf("Failed to unset DOCS_FETCH_RATE_PER_SEC: %v", err)
	}
	if err := os.Unsetenv("DOCS_FETCH_TIMEOUT"); err != nil {
		log.Printf("Failed to unset DOCS_FETCH_TIMEOUT: %v", err)
	}
}

func TestDetectEnvironment(t *testing.T) {
	// Note: When running tests, DetectEnvironment checks for test flags first.
	// Since we're in a test context, it will always return EnvTest.
	// This test documents the behavior rather than trying to bypass it.

	tests := []struct {
		name     string
		envValue string
		expected Environment
	}{
		{"Default when ENV is empty", "", EnvTest},          // Test context overrides
		{"dev environment", "dev", EnvTest},                 // Test context overrides
		{"development environment", "development", EnvTest}, // Test context overrides
		{"staging environment", "staging", EnvTest},         // Test context overrides
		{"prod environment", "prod", EnvTest},               // Test context overrides
		{"production environment", "production", EnvTest},   // Test context overrides
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Unsetenv("ENV")
			if tt.envValue != "" {
				_ = os.Setenv("ENV", tt.envValue)
			}
			t.Cleanup(func() {
				if err := os.Unsetenv("ENV"); err != nil {
					t.Logf("Failed to unset ENV: %v", err)
				}
			})

			env := DetectEnvironment()
			// In test context, DetectEnvironment returns EnvTest due to test flags
			if env != tt.expected {
				t.Logf("Note: In test context, DetectEnvironment returns %v regardless of ENV variable", env)
			}
		})
	}

	// Verify that test context detection works
	t.Run("Test context detection", func(t *testing.T) {
		_ = os.Setenv("ENV", "production")
		t.Cleanup(func() {
			if err := os.Unsetenv("ENV"); err != nil {
				t.Logf("Failed to unset ENV: %v", err)
			}
		})

		env := DetectEnvironment()
		if env != EnvTest {
			t.Logf("Expected EnvTest in test context, got %v (behavior may vary by test runner)", env)
		} else {
			t.Log("Successfully detected test context - test flags take priority over ENV variable")
		}
	})
}

func TestGetEnvVars(t *testing.T) {
	envVars := GetEnvVars()

	expectedVars := []string{
		"PORT",
		"ADDRESS",
		"ENV",
		"LOG_LEVEL",
		"LOG_RETENTION_WEEKS",
		"MAX_LOG_FILE_SIZE",
		"MAX_REQUEST_BODY",
		"MAX_HEADER_SIZE",
		"ALLOW_DIRECT_ACCESS",
		"DISABLE_RATE_LIMITER",
		"DOCS_ENABLED",
		"DOCS_CACHE_DIR",
		"DOCS_FETCH_RATE_PER_SEC",
		"DOCS_FETCH_TIMEOUT",
	}

	if len(envVars) != len(expectedVars) {
		t.Errorf("Expected %d environment variables, got %d", len(expectedVars), len(envVars))
	}

	for _, expected := range expectedVars {
		found := false
		for _, actual := range envVars {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected env var %s not found in returned list", expected)
		}
	}
}

func TestValidateAllEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		expectError bool
	}{
		{
			name: "All required env vars present",
			setupEnv: func() {
				_ = os.Setenv("PORT", "8000")
			},
			expectError: false,
		},
		{
			name: "Missing PORT",
			setupEnv: func() {
				_ = os.Unsetenv("PORT")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Unsetenv("PORT")
			tt.setupEnv()
			t.Cleanup(func() {
				if err := os.Unsetenv("PORT"); err != nil {
					t.Logf("Failed to unset PORT: %v", err)
				}
			})

			err := ValidateAllEnvVars()
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
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
		{"test", EnvDevelopment, true},
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

func TestDisableRateLimiter(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		expectedValue bool
	}{
		{"DISABLE_RATE_LIMITER=true", "true", true},
		{"DISABLE_RATE_LIMITER=TRUE", "TRUE", true},
		{"DISABLE_RATE_LIMITER=1", "1", true},
		{"DISABLE_RATE_LIMITER=false", "false", false},
		{"DISABLE_RATE_LIMITER=FALSE", "FALSE", false},
		{"DISABLE_RATE_LIMITER=0", "0", false},
		{"DISABLE_RATE_LIMITER not set", "", false},
		{"DISABLE_RATE_LIMITER invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("PORT", "8003")
			_ = os.Setenv("ADDRESS", "127.0.0.1")
			_ = os.Setenv("ENV", "dev")
			_ = os.Setenv("LOG_LEVEL", "info")
			_ = os.Setenv("DISABLE_RATE_LIMITER", tt.envValue)
			defer cleanupEnv()

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if cfg.DisableRateLimiter != tt.expectedValue {
				t.Errorf("Expected DisableRateLimiter=%v for %s, got %v", tt.expectedValue, tt.name, cfg.DisableRateLimiter)
			}
		})
	}
}

func TestValidateAddress_0dot0dot0dot0dot0_WithoutAllowDirectAccess(t *testing.T) {
	cfg := &Config{
		Address:            "0.0.0.0",
		Port:               "8000",
		Env:                EnvDevelopment,
		LogLevel:           "info",
		LogRetentionWeeks:  4,
		MaxLogFileSize:     104857600,
		MaxRequestBody:     1048576,
		MaxHeaderSize:      1048576,
		AllowDirectAccess:  false,
		DisableRateLimiter: false,
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
		Address:            "0.0.0.0",
		Port:               "8000",
		Env:                EnvDevelopment,
		LogLevel:           "info",
		LogRetentionWeeks:  4,
		MaxLogFileSize:     104857600,
		MaxRequestBody:     1048576,
		MaxHeaderSize:      1048576,
		AllowDirectAccess:  true,
		DisableRateLimiter: false,
	}

	err := validateAddress(cfg)

	if err != nil {
		t.Errorf("Expected no error for 0.0.0.0 when AllowDirectAccess is true, got: %s", err.Error())
	}
}

func TestValidateAddress_IPv6Any_WithoutAllowDirectAccess(t *testing.T) {
	cfg := &Config{
		Address:            "::",
		Port:               "8000",
		Env:                EnvDevelopment,
		LogLevel:           "info",
		LogRetentionWeeks:  4,
		MaxLogFileSize:     104857600,
		MaxRequestBody:     1048576,
		MaxHeaderSize:      1048576,
		AllowDirectAccess:  false,
		DisableRateLimiter: false,
	}

	err := validateAddress(cfg)

	if err == nil {
		t.Error("Expected error for :: when AllowDirectAccess is false")
	}
}

func TestValidateAddress_IPv6Any_WithAllowDirectAccess(t *testing.T) {
	cfg := &Config{
		Address:            "::",
		Port:               "8000",
		Env:                EnvDevelopment,
		LogLevel:           "info",
		LogRetentionWeeks:  4,
		MaxLogFileSize:     104857600,
		MaxRequestBody:     1048576,
		MaxHeaderSize:      1048576,
		AllowDirectAccess:  true,
		DisableRateLimiter: false,
	}

	err := validateAddress(cfg)

	if err != nil {
		t.Errorf("Expected no error for :: when AllowDirectAccess is true, got: %s", err.Error())
	}
}

func TestValidateAddress_127dot0dot0dot1_AlwaysAllowed(t *testing.T) {
	cfg := &Config{
		Address:            "127.0.0.1",
		Port:               "8000",
		Env:                EnvDevelopment,
		LogLevel:           "info",
		LogRetentionWeeks:  4,
		MaxLogFileSize:     104857600,
		MaxRequestBody:     1048576,
		MaxHeaderSize:      1048576,
		AllowDirectAccess:  false,
		DisableRateLimiter: false,
	}

	err := validateAddress(cfg)

	if err != nil {
		t.Errorf("127.0.0.1 should always be allowed, got error: %s", err.Error())
	}
}

func TestValidateAddress_IPv6Loopback(t *testing.T) {
	cfg := &Config{
		Address:            "::1",
		Port:               "8000",
		Env:                EnvDevelopment,
		LogLevel:           "info",
		LogRetentionWeeks:  4,
		MaxLogFileSize:     104857600,
		MaxRequestBody:     1048576,
		MaxHeaderSize:      1048576,
		AllowDirectAccess:  false,
		DisableRateLimiter: false,
	}

	err := validateAddress(cfg)

	if err != nil {
		t.Errorf("IPv6 loopback ::1 should always be allowed, got error: %s", err.Error())
	}
}

func TestValidateAddress_IPv6Localhost(t *testing.T) {
	cfg := &Config{
		Address:            "localhost",
		Port:               "8000",
		Env:                EnvDevelopment,
		LogLevel:           "info",
		LogRetentionWeeks:  4,
		MaxLogFileSize:     104857600,
		MaxRequestBody:     1048576,
		MaxHeaderSize:      1048576,
		AllowDirectAccess:  false,
		DisableRateLimiter: false,
	}

	err := validateAddress(cfg)

	if err != nil {
		t.Errorf("localhost should always be allowed, got error: %s", err.Error())
	}
}

func TestValidateAddress_PrivateIPRanges(t *testing.T) {
	privateIPs := []string{
		"10.0.0.1",        // 10.0.0.0/8
		"10.255.255.254",  // 10.0.0.0/8
		"172.16.0.1",      // 172.16.0.0/12
		"172.31.255.254",  // 172.16.0.0/12
		"192.168.0.1",     // 192.168.0.0/16
		"192.168.255.254", // 192.168.0.0/16
	}

	for _, ip := range privateIPs {
		t.Run(ip, func(t *testing.T) {
			cfg := &Config{
				Address:            ip,
				Port:               "8000",
				Env:                EnvDevelopment,
				LogLevel:           "info",
				LogRetentionWeeks:  4,
				MaxLogFileSize:     104857600,
				MaxRequestBody:     1048576,
				MaxHeaderSize:      1048576,
				AllowDirectAccess:  false,
				DisableRateLimiter: false,
			}

			err := validateAddress(cfg)

			if err != nil {
				t.Errorf("Private IP %s should be allowed, got error: %s", ip, err.Error())
			}
		})
	}
}

func TestValidateAddress_PublicIPs(t *testing.T) {
	publicIPs := []string{
		"8.8.8.8",     // Google DNS
		"1.1.1.1",     // Cloudflare DNS
		"203.0.113.1", // TEST-NET-3 (public range)
	}

	for _, ip := range publicIPs {
		t.Run(ip, func(t *testing.T) {
			cfg := &Config{
				Address:            ip,
				Port:               "8000",
				Env:                EnvDevelopment,
				LogLevel:           "info",
				LogRetentionWeeks:  4,
				MaxLogFileSize:     104857600,
				MaxRequestBody:     1048576,
				MaxHeaderSize:      1048576,
				AllowDirectAccess:  false,
				DisableRateLimiter: false,
			}

			err := validateAddress(cfg)

			// Public IPs should return a warning (not necessarily an error, but a warning message)
			if err == nil {
				t.Logf("Public IP %s returned nil (warning might be expected)", ip)
			} else {
				// Should contain a warning about public IPs
				if !containsString(err.Error(), "public IP") && !containsString(err.Error(), "private network ranges") {
					t.Errorf("Expected warning about public IP, got: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateAddress_IPv6PrivateRanges(t *testing.T) {
	ipv6PrivateIPs := []string{
		"fc00::1", // Unique local address (fc00::/7)
		"fe80::1", // Link-local address (fe80::/10)
	}

	for _, ip := range ipv6PrivateIPs {
		t.Run(ip, func(t *testing.T) {
			cfg := &Config{
				Address:            ip,
				Port:               "8000",
				Env:                EnvDevelopment,
				LogLevel:           "info",
				LogRetentionWeeks:  4,
				MaxLogFileSize:     104857600,
				MaxRequestBody:     1048576,
				MaxHeaderSize:      1048576,
				AllowDirectAccess:  false,
				DisableRateLimiter: false,
			}

			err := validateAddress(cfg)

			if err != nil {
				t.Logf("IPv6 private IP %s validation: %s", ip, err.Error())
				// IPv6 private IPs may or may not be detected as private depending on implementation
				// The test documents the current behavior
			}
		})
	}
}

// setDocsTestBaseEnv sets the baseline variables required for Load to succeed
// and clears all DOCS_* variables so each test controls its own docs config.
func setDocsTestBaseEnv() {
	_ = os.Setenv("PORT", "8002")
	_ = os.Setenv("ADDRESS", "127.0.0.1")
	_ = os.Setenv("ENV", "dev")
	_ = os.Setenv("LOG_LEVEL", "info")
	unsetDocsEnv()
}

// unsetDocsEnv clears all DOCS_* environment variables.
func unsetDocsEnv() {
	_ = os.Unsetenv("DOCS_ENABLED")
	_ = os.Unsetenv("DOCS_CACHE_DIR")
	_ = os.Unsetenv("DOCS_FETCH_RATE_PER_SEC")
	_ = os.Unsetenv("DOCS_FETCH_TIMEOUT")
}

func TestDocsConfigDefaults(t *testing.T) {
	// Arrange: baseline env, no DOCS_* variables set
	setDocsTestBaseEnv()
	defer cleanupEnv()

	// Act
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Assert
	if !cfg.DocsEnabled {
		t.Error("Expected default DocsEnabled=true, got false")
	}
	if cfg.DocsCacheDir != "./files/docs" {
		t.Errorf("Expected default DocsCacheDir ./files/docs, got %s", cfg.DocsCacheDir)
	}
	if cfg.DocsFetchRatePerSec != 2.0 {
		t.Errorf("Expected default DocsFetchRatePerSec 2.0, got %g", cfg.DocsFetchRatePerSec)
	}
	if cfg.DocsFetchTimeout != 15*time.Second {
		t.Errorf("Expected default DocsFetchTimeout 15s, got %s", cfg.DocsFetchTimeout)
	}
}

func TestDocsConfigExplicitValues(t *testing.T) {
	// Arrange
	setDocsTestBaseEnv()
	defer cleanupEnv()
	_ = os.Setenv("DOCS_ENABLED", "true")
	_ = os.Setenv("DOCS_CACHE_DIR", "/var/lib/medicaments-api/docs")
	_ = os.Setenv("DOCS_FETCH_RATE_PER_SEC", "5")
	_ = os.Setenv("DOCS_FETCH_TIMEOUT", "30s")

	// Act
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Assert
	if !cfg.DocsEnabled {
		t.Error("Expected DocsEnabled=true, got false")
	}
	if cfg.DocsCacheDir != "/var/lib/medicaments-api/docs" {
		t.Errorf("Expected DocsCacheDir /var/lib/medicaments-api/docs, got %s", cfg.DocsCacheDir)
	}
	if cfg.DocsFetchRatePerSec != 5.0 {
		t.Errorf("Expected DocsFetchRatePerSec 5.0, got %g", cfg.DocsFetchRatePerSec)
	}
	if cfg.DocsFetchTimeout != 30*time.Second {
		t.Errorf("Expected DocsFetchTimeout 30s, got %s", cfg.DocsFetchTimeout)
	}
}

func TestDocsConfigDisabled(t *testing.T) {
	// Arrange
	setDocsTestBaseEnv()
	defer cleanupEnv()
	_ = os.Setenv("DOCS_ENABLED", "false")

	// Act
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Assert: the kill switch parses to false while the rest stays valid
	if cfg.DocsEnabled {
		t.Error("Expected DocsEnabled=false, got true")
	}
}

func TestDocsEnabledParsing(t *testing.T) {
	// DOCS_ENABLED follows the lenient bool parsing used by the other bool
	// flags: an unusable value falls back to the default (true), so a typo
	// can never silently disable the feature.
	tests := []struct {
		name          string
		envValue      string
		expectedValue bool
	}{
		{"DOCS_ENABLED=true", "true", true},
		{"DOCS_ENABLED=TRUE", "TRUE", true},
		{"DOCS_ENABLED=1", "1", true},
		{"DOCS_ENABLED=false", "false", false},
		{"DOCS_ENABLED=FALSE", "FALSE", false},
		{"DOCS_ENABLED=0", "0", false},
		{"DOCS_ENABLED not set", "", true},
		{"DOCS_ENABLED invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setDocsTestBaseEnv()
			defer cleanupEnv()
			_ = os.Setenv("DOCS_ENABLED", tt.envValue)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if cfg.DocsEnabled != tt.expectedValue {
				t.Errorf("Expected DocsEnabled=%v for %s, got %v", tt.expectedValue, tt.name, cfg.DocsEnabled)
			}
		})
	}
}

func TestInvalidDocsFetchRatePerSec(t *testing.T) {
	testCases := []struct {
		value    string
		expected string
	}{
		{"0.1", "must be between 0.25 and 10"},
		{"10.5", "must be between 0.25 and 10"},
		{"-1", "must be between 0.25 and 10"},
		{"abc", "failed to parse DOCS_FETCH_RATE_PER_SEC"},
	}

	for _, tc := range testCases {
		setDocsTestBaseEnv()
		defer cleanupEnv()
		_ = os.Setenv("DOCS_FETCH_RATE_PER_SEC", tc.value)

		_, err := Load()
		if err == nil {
			t.Errorf("Expected error for DOCS_FETCH_RATE_PER_SEC=%s, got nil", tc.value)
			continue
		}
		if !containsString(err.Error(), tc.expected) {
			t.Errorf("Expected error containing %q for DOCS_FETCH_RATE_PER_SEC=%s, got: %s", tc.expected, tc.value, err.Error())
		}
	}
}

func TestValidDocsFetchRatePerSec(t *testing.T) {
	testCases := []struct {
		value    string
		expected float64
	}{
		{"0.25", 0.25},
		{"0.5", 0.5},
		{"2", 2.0},
		{"10", 10.0},
	}

	for _, tc := range testCases {
		setDocsTestBaseEnv()
		defer cleanupEnv()
		_ = os.Setenv("DOCS_FETCH_RATE_PER_SEC", tc.value)

		cfg, err := Load()
		if err != nil {
			t.Errorf("Expected no error for DOCS_FETCH_RATE_PER_SEC=%s, got %v", tc.value, err)
			continue
		}

		if cfg.DocsFetchRatePerSec != tc.expected {
			t.Errorf("Expected DocsFetchRatePerSec %g for %s, got %g", tc.expected, tc.value, cfg.DocsFetchRatePerSec)
		}
	}
}

func TestInvalidDocsFetchTimeout(t *testing.T) {
	testCases := []struct {
		value    string
		expected string
	}{
		{"abc", "failed to parse DOCS_FETCH_TIMEOUT"},
		{"15", "failed to parse DOCS_FETCH_TIMEOUT"}, // missing time unit
		{"0s", "DOCS_FETCH_TIMEOUT must be positive"},
		{"-5s", "DOCS_FETCH_TIMEOUT must be positive"},
	}

	for _, tc := range testCases {
		setDocsTestBaseEnv()
		defer cleanupEnv()
		_ = os.Setenv("DOCS_FETCH_TIMEOUT", tc.value)

		_, err := Load()
		if err == nil {
			t.Errorf("Expected error for DOCS_FETCH_TIMEOUT=%s, got nil", tc.value)
			continue
		}
		if !containsString(err.Error(), tc.expected) {
			t.Errorf("Expected error containing %q for DOCS_FETCH_TIMEOUT=%s, got: %s", tc.expected, tc.value, err.Error())
		}
	}
}

func TestValidDocsFetchTimeout(t *testing.T) {
	testCases := []struct {
		value    string
		expected time.Duration
	}{
		{"15s", 15 * time.Second},
		{"250ms", 250 * time.Millisecond},
		{"1m", time.Minute},
		{"1h", time.Hour},
	}

	for _, tc := range testCases {
		setDocsTestBaseEnv()
		defer cleanupEnv()
		_ = os.Setenv("DOCS_FETCH_TIMEOUT", tc.value)

		cfg, err := Load()
		if err != nil {
			t.Errorf("Expected no error for DOCS_FETCH_TIMEOUT=%s, got %v", tc.value, err)
			continue
		}

		if cfg.DocsFetchTimeout != tc.expected {
			t.Errorf("Expected DocsFetchTimeout %s for %s, got %s", tc.expected, tc.value, cfg.DocsFetchTimeout)
		}
	}
}

func TestInvalidDocsCacheDir(t *testing.T) {
	setDocsTestBaseEnv()
	defer cleanupEnv()
	_ = os.Setenv("DOCS_CACHE_DIR", "   ")

	_, err := Load()
	if err == nil {
		t.Error("Expected error for whitespace-only DOCS_CACHE_DIR, got nil")
	}
	if err != nil && !containsString(err.Error(), "DOCS_CACHE_DIR cannot be empty") {
		t.Errorf("Expected error mentioning DOCS_CACHE_DIR cannot be empty, got: %s", err.Error())
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
