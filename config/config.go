// Package config has the configuration file for the app
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration
type Config struct {
	Port     string
	Address  string
	Env      string
	LogLevel string
}

// Load loads and validates configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:     getEnvWithDefault("PORT", "8002"),
		Address:  getEnvWithDefault("ADDRESS", "127.0.0.1"),
		Env:      getEnvWithDefault("ENV", "dev"),
		LogLevel: getEnvWithDefault("LOG_LEVEL", "info"),
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// validateConfig validates all configuration values
func validateConfig(cfg *Config) error {
	// Validate PORT
	if err := validatePort(cfg.Port); err != nil {
		return fmt.Errorf("invalid PORT: %w", err)
	}

	// Validate ADDRESS
	if err := validateAddress(cfg.Address); err != nil {
		return fmt.Errorf("invalid ADDRESS: %w", err)
	}

	// Validate ENV
	if err := validateEnv(cfg.Env); err != nil {
		return fmt.Errorf("invalid ENV: %w", err)
	}

	// Validate LOG_LEVEL
	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return fmt.Errorf("invalid LOG_LEVEL: %w", err)
	}

	return nil
}

// validatePort validates the PORT environment variable
func validatePort(port string) error {
	if port == "" {
		return fmt.Errorf("PORT cannot be empty")
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("PORT must be a valid number: %w", err)
	}

	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535")
	}

	// Check for privileged ports
	if portNum < 1024 {
		return fmt.Errorf("PORT %d is privileged (less than 1024), use ports 1024-65535", portNum)
	}

	return nil
}

// validateAddress validates the ADDRESS environment variable
func validateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("ADDRESS cannot be empty")
	}

	// Check for localhost/loopback addresses first
	if address == "127.0.0.1" || address == "::1" || address == "localhost" {
		// This is acceptable for development
		return nil
	}

	// Check if it's a valid IP address
	if ip := net.ParseIP(address); ip == nil {
		return fmt.Errorf("ADDRESS must be a valid IP address or 'localhost', got: %s", address)
	}

	// Check for private network ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
	ip := net.ParseIP(address)
	if ip != nil && !ip.IsLoopback() && !ip.IsPrivate() {
		return fmt.Errorf("ADDRESS %s is a public IP, consider using private network ranges for security", address)
	}

	return nil
}

// validateEnv validates the ENV environment variable
func validateEnv(env string) error {
	if env == "" {
		return fmt.Errorf("ENV cannot be empty")
	}

	validEnvs := []string{"dev", "staging", "prod", "test"}
	env = strings.ToLower(env)

	for _, validEnv := range validEnvs {
		if env == validEnv {
			return nil
		}
	}

	return fmt.Errorf("ENV must be one of: %v, got: %s", validEnvs, env)
}

// validateLogLevel validates the LOG_LEVEL environment variable
func validateLogLevel(logLevel string) error {
	if logLevel == "" {
		return fmt.Errorf("LOG_LEVEL cannot be empty")
	}

	validLevels := []string{"debug", "info", "warn", "error"}
	logLevel = strings.ToLower(logLevel)

	for _, level := range validLevels {
		if logLevel == level {
			return nil
		}
	}

	return fmt.Errorf("LOG_LEVEL must be one of: %v, got: %s", validLevels, logLevel)
}

// getEnvWithDefault gets an environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvVars returns a list of all expected environment variables
func GetEnvVars() []string {
	return []string{
		"PORT",
		"ADDRESS",
		"ENV",
		"LOG_LEVEL",
	}
}

// ValidateAllEnvVars checks if all required environment variables are set
func ValidateAllEnvVars() error {
	requiredVars := []string{"PORT"} // Only PORT is truly required
	missingVars := []string{}

	for _, varName := range requiredVars {
		if os.Getenv(varName) == "" {
			missingVars = append(missingVars, varName)
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	return nil
}
