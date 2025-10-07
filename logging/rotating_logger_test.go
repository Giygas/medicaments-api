package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRotatingLogger(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "rotating-logger-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create rotating logger with 1 week retention
	rl := NewRotatingLogger(tempDir, 1)

	// Test initial rotation
	err = rl.rotateIfNeeded()
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	// Check that current file is created
	currentWeek := getWeekKey(time.Now())
	expectedFileName := filepath.Join(tempDir, "app-"+currentWeek+".log")
	if _, err := os.Stat(expectedFileName); os.IsNotExist(err) {
		t.Errorf("Expected log file %s was not created", expectedFileName)
	}

	// Test writing to log
	testMessage := "Test log message"
	_, err = rl.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("Failed to write to log: %v", err)
	}

	// Verify content was written
	content, err := os.ReadFile(expectedFileName)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), testMessage) {
		t.Errorf("Log file does not contain test message: %s", string(content))
	}

	// Test cleanup
	err = rl.cleanupOldLogs()
	if err != nil {
		t.Fatalf("Failed to cleanup old logs: %v", err)
	}

	// Close logger
	err = rl.Close()
	if err != nil {
		t.Fatalf("Failed to close logger: %v", err)
	}
}

func TestGetWeekKey(t *testing.T) {
	// Test week key generation
	testTime := time.Date(2025, 10, 7, 12, 0, 0, 0, time.UTC)
	weekKey := getWeekKey(testTime)

	// 2025-10-07 should be in week 41 of 2025
	expected := "2025-W41"
	if weekKey != expected {
		t.Errorf("Expected week key %s, got %s", expected, weekKey)
	}
}

func TestRotatingLoggerWithDifferentWeeks(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "rotating-logger-test-weeks")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test creating two different rotating loggers for different weeks
	rl1 := NewRotatingLogger(tempDir, 1)
	rl1.currentWeek = "2025-W40"

	rl2 := NewRotatingLogger(tempDir, 1)
	rl2.currentWeek = "2025-W41"

	// Create files for both weeks
	err = rl1.rotateIfNeeded()
	if err != nil {
		t.Fatalf("Failed to rotate to week 40: %v", err)
	}

	err = rl2.rotateIfNeeded()
	if err != nil {
		t.Fatalf("Failed to rotate to week 41: %v", err)
	}

	// Write to both files
	_, err = rl1.Write([]byte("Week 40 message"))
	if err != nil {
		t.Fatalf("Failed to write to week 40 log: %v", err)
	}

	_, err = rl2.Write([]byte("Week 41 message"))
	if err != nil {
		t.Fatalf("Failed to write to week 41 log: %v", err)
	}

	// Verify both files exist
	week40File := filepath.Join(tempDir, "app-2025-W40.log")
	week41File := filepath.Join(tempDir, "app-2025-W41.log")

	if _, err := os.Stat(week40File); os.IsNotExist(err) {
		t.Errorf("Expected week 40 log file %s was not created", week40File)
	}

	if _, err := os.Stat(week41File); os.IsNotExist(err) {
		t.Errorf("Expected week 41 log file %s was not created", week41File)
	}

	rl1.Close()
	rl2.Close()
}

func TestSetupLoggerWithRetention(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "setup-logger-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test setup with custom retention
	logger := SetupLoggerWithRetention(tempDir, 2)
	if logger == nil {
		t.Fatal("SetupLoggerWithRetention returned nil")
	}

	// Test that logger works
	logger.Info("Test message from rotating logger")

	// Check that log file was created
	currentWeek := getWeekKey(time.Now())
	expectedFileName := filepath.Join(tempDir, "app-"+currentWeek+".log")
	if _, err := os.Stat(expectedFileName); os.IsNotExist(err) {
		t.Errorf("Expected log file %s was not created", expectedFileName)
	}
}

func TestGlobalLoggingService(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "global-logger-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original service
	originalService := DefaultLoggingService
	defer func() {
		DefaultLoggingService = originalService
	}()

	// Test global service initialization
	InitLoggerWithRetention(tempDir, 2)
	if DefaultLoggingService == nil {
		t.Fatal("DefaultLoggingService was not initialized")
	}

	// Test that logger works
	Info("Test message from global logger")

	// Check that log file was created
	currentWeek := getWeekKey(time.Now())
	expectedFileName := filepath.Join(tempDir, "app-"+currentWeek+".log")
	if _, err := os.Stat(expectedFileName); os.IsNotExist(err) {
		t.Errorf("Expected log file %s was not created", expectedFileName)
	}

	// Test proper shutdown
	Close()
}

func TestCleanupOldLogs(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "cleanup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rl := NewRotatingLogger(tempDir, 1) // 1 week retention

	// Create some old log files
	oldFile := filepath.Join(tempDir, "app-2025-W30.log")
	newFile := filepath.Join(tempDir, "app-"+getWeekKey(time.Now())+".log")

	// Create old file with old modification time
	oldLogFile, err := os.Create(oldFile)
	if err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	oldLogFile.WriteString("Old log content")
	oldLogFile.Close()

	// Set modification time to 3 weeks ago
	threeWeeksAgo := time.Now().AddDate(0, 0, -21)
	err = os.Chtimes(oldFile, threeWeeksAgo, threeWeeksAgo)
	if err != nil {
		t.Fatalf("Failed to set old file modification time: %v", err)
	}

	// Create new file
	newLogFile, err := os.Create(newFile)
	if err != nil {
		t.Fatalf("Failed to create new log file: %v", err)
	}
	newLogFile.WriteString("New log content")
	newLogFile.Close()

	// Force cleanup by resetting lastCleanup time
	rl.lastCleanup = time.Now().Add(-25 * time.Hour)

	// Run cleanup
	err = rl.cleanupOldLogs()
	if err != nil {
		t.Fatalf("Failed to cleanup old logs: %v", err)
	}

	// Check that old file was deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("Old log file %s was not deleted", oldFile)
	}

	// Check that new file still exists
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Errorf("New log file %s was incorrectly deleted", newFile)
	}
}

func TestRotatingLoggerWithSizeLimit(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "size-limit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create rotating logger with very small size limit (100 bytes)
	rl := NewRotatingLoggerWithSizeLimit(tempDir, 1, 100)

	// Initialize rotation
	err = rl.rotateIfNeeded()
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	// Write a small message that should fit
	smallMessage := "Small message"
	_, err = rl.Write([]byte(smallMessage))
	if err != nil {
		t.Fatalf("Failed to write small message: %v", err)
	}

	// Write a large message that should trigger size-based rotation
	largeMessage := strings.Repeat("This is a very long log message that should trigger rotation. ", 10)
	_, err = rl.Write([]byte(largeMessage))
	if err != nil {
		t.Fatalf("Failed to write large message: %v", err)
	}

	// Check that multiple files were created (original + size-rotated)
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	logFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "app-") && strings.HasSuffix(entry.Name(), ".log") {
			logFiles++
		}
	}

	if logFiles < 2 {
		t.Errorf("Expected at least 2 log files due to size rotation, got %d", logFiles)
	}

	// Close logger
	err = rl.Close()
	if err != nil {
		t.Fatalf("Failed to close logger: %v", err)
	}
}
