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
	Port              string
	Address           string
	Env               string
	LogLevel          string
	LogRetentionWeeks int   // Number of weeks to keep log files
	MaxLogFileSize    int64 // Maximum log file size in bytes
	MaxRequestBody    int64 // Maximum request body size in bytes
	MaxHeaderSize     int64 // Maximum header size in bytes
}

// Load loads and validates configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnvWithDefault("PORT", "8000"),
		Address:           getEnvWithDefault("ADDRESS", "127.0.0.1"),
		Env:               getEnvWithDefault("ENV", "dev"),
		LogLevel:          getEnvWithDefault("LOG_LEVEL", "info"),
		LogRetentionWeeks: getIntEnvWithDefault("LOG_RETENTION_WEEKS", 4),         // 4 weeks default
		MaxLogFileSize:    getInt64EnvWithDefault("MAX_LOG_FILE_SIZE", 104857600), // 100MB default
		MaxRequestBody:    getInt64EnvWithDefault("MAX_REQUEST_BODY", 1048576),    // 1MB default
		MaxHeaderSize:     getInt64EnvWithDefault("MAX_HEADER_SIZE", 1048576),     // 1MB default
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

	// Validate MAX_REQUEST_BODY
	if err := validateSizeLimit(cfg.MaxRequestBody, "MAX_REQUEST_BODY"); err != nil {
		return fmt.Errorf("invalid MAX_REQUEST_BODY: %w", err)
	}

	// Validate MAX_HEADER_SIZE
	if err := validateSizeLimit(cfg.MaxHeaderSize, "MAX_HEADER_SIZE"); err != nil {
		return fmt.Errorf("invalid MAX_HEADER_SIZE: %w", err)
	}

	// Validate LOG_RETENTION_WEEKS
	if err := validateLogRetentionWeeks(cfg.LogRetentionWeeks); err != nil {
		return fmt.Errorf("invalid LOG_RETENTION_WEEKS: %w", err)
	}

	// Validate MAX_LOG_FILE_SIZE
	if err := validateMaxLogFileSize(cfg.MaxLogFileSize); err != nil {
		return fmt.Errorf("invalid MAX_LOG_FILE_SIZE: %w", err)
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

// validateSizeLimit validates size limit configuration values
func validateSizeLimit(size int64, configName string) error {
	if size <= 0 {
		return fmt.Errorf("%s must be positive, got: %d", configName, size)
	}

	if size > 100*1024*1024 { // 100MB
		return fmt.Errorf("%s is too large (max 100MB), got: %d bytes", configName, size)
	}

	return nil
}

// validateLogRetentionWeeks validates the LOG_RETENTION_WEEKS environment variable
func validateLogRetentionWeeks(weeks int) error {
	if weeks <= 0 {
		return fmt.Errorf("LOG_RETENTION_WEEKS must be positive, got: %d", weeks)
	}

	if weeks > 52 { // 1 year maximum
		return fmt.Errorf("LOG_RETENTION_WEEKS is too large (max 52 weeks), got: %d", weeks)
	}

	return nil
}

// validateMaxLogFileSize validates the MAX_LOG_FILE_SIZE environment variable
func validateMaxLogFileSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("MAX_LOG_FILE_SIZE must be positive, got: %d", size)
	}

	// Minimum 1MB, maximum 1GB
	if size < 1024*1024 {
		return fmt.Errorf("MAX_LOG_FILE_SIZE is too small (min 1MB), got: %d bytes", size)
	}

	if size > 1024*1024*1024 {
		return fmt.Errorf("MAX_LOG_FILE_SIZE is too large (max 1GB), got: %d bytes", size)
	}

	return nil
}

// getEnvWithDefault gets an environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnvWithDefault gets an environment variable as int with a default value
func getIntEnvWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getInt64EnvWithDefault gets an environment variable as int64 with a default value
func getInt64EnvWithDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
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
		"LOG_RETENTION_WEEKS",
		"MAX_LOG_FILE_SIZE",
		"MAX_REQUEST_BODY",
		"MAX_HEADER_SIZE",
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
