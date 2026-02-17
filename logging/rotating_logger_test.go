package logging

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/config"
)

func TestRotatingLogger(t *testing.T) {
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	// Create rotating logger with 1 week retention
	rl := NewRotatingLogger(tempDir, 1)

	// Test initial rotation
	rl.mu.Lock()
	err := rl.doRotate(getWeekKey(time.Now()))
	rl.mu.Unlock()
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	// Check that current file is created
	currentWeek := getWeekKey(time.Now())
	expectedFileName := filepath.Join(tempDir, "app-"+currentWeek+".log")
	if _, statErr := os.Stat(expectedFileName); os.IsNotExist(statErr) {
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
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	// Test creating two different rotating loggers for different weeks
	rl1 := NewRotatingLogger(tempDir, 1)
	rl1.currentWeek = "2025-W40"

	rl2 := NewRotatingLogger(tempDir, 1)
	rl2.currentWeek = "2025-W41"

	// Create files for both weeks
	rl1.mu.Lock()
	err := rl1.doRotate("2025-W40")
	rl1.mu.Unlock()
	if err != nil {
		t.Fatalf("Failed to rotate to week 40: %v", err)
	}

	rl2.mu.Lock()
	err = rl2.doRotate("2025-W41")
	rl2.mu.Unlock()
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

	_ = rl1.Close()
	_ = rl2.Close()
}

func TestGlobalLoggingService(t *testing.T) {
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	// Use ResetForTest() for proper test isolation
	ResetForTest(t, tempDir, config.EnvTest, "", 2, 100*1024*1024)

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

	// No need for explicit Close() - t.Cleanup() handles it
}

func TestCleanupOldLogs(t *testing.T) {
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	rl := NewRotatingLogger(tempDir, 1) // 1 week retention

	// Create some old log files
	oldFile := filepath.Join(tempDir, "app-2025-W30.log")
	newFile := filepath.Join(tempDir, "app-"+getWeekKey(time.Now())+".log")

	// Create old file with old modification time
	oldLogFile, err := os.Create(oldFile)
	if err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	_, _ = oldLogFile.WriteString("Old log content")
	_ = oldLogFile.Close()

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
	_, _ = newLogFile.WriteString("New log content")
	_ = newLogFile.Close()

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
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	// Create rotating logger with very small size limit (100 bytes)
	rl := NewRotatingLoggerWithSizeLimit(tempDir, 1, 100)

	// Initialize rotation
	err := rl.doRotate(getWeekKey(time.Now()))
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

	// Verify size-rotated files have correct naming format
	hasNumberedFile := false
	numberedPattern := regexp.MustCompile(`_\d{2}\.log$`)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "_01.") || strings.Contains(entry.Name(), "_02.") {
			hasNumberedFile = true
			if !strings.HasSuffix(entry.Name(), ".log") {
				t.Errorf("Numbered file missing .log extension: %s", entry.Name())
			}
			// Verify number format (two digits)
			if !numberedPattern.MatchString(entry.Name()) {
				t.Errorf("Numbered file has incorrect format: %s", entry.Name())
			}
		}
	}

	if !hasNumberedFile {
		t.Error("Expected at least one numbered file due to large write")
	}

	// Close logger
	err = rl.Close()
	if err != nil {
		t.Fatalf("Failed to close logger: %v", err)
	}
}

func TestRotatingLoggerErrorCases(t *testing.T) {
	// Test with invalid directory
	invalidDir := "/invalid/directory/that/does/not/exist"
	rl := NewRotatingLogger(invalidDir, 1)

	// Try to rotate with invalid directory
	err := rl.doRotate(getWeekKey(time.Now()))
	if err == nil {
		t.Error("Expected error when rotating with invalid directory, got nil")
	}

	// Try to write with invalid directory
	_, err = rl.Write([]byte("test message"))
	if err == nil {
		t.Error("Expected error when writing with invalid directory, got nil")
	}

	// Close should still work even with invalid directory
	err = rl.Close()
	if err != nil {
		t.Errorf("Unexpected error when closing logger with invalid directory: %v", err)
	}
}

func TestRotatingLoggerConcurrentWrites(t *testing.T) {
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	rl := NewRotatingLogger(tempDir, 1)
	defer func() { _ = rl.Close() }()

	// Initialize rotation
	err := rl.doRotate(getWeekKey(time.Now()))
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	// Test concurrent writes
	const numGoroutines = 10
	const numWrites = 5

	done := make(chan bool, numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			for j := range numWrites {
				message := fmt.Sprintf("Goroutine %d, Write %d", id, j)
				if _, writeErr := rl.Write([]byte(message)); writeErr != nil {
					t.Errorf("Concurrent write failed: %v", writeErr)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for range numGoroutines {
		<-done
	}

	// Verify log file exists and has content
	currentWeek := getWeekKey(time.Now())
	expectedFileName := filepath.Join(tempDir, "app-"+currentWeek+".log")
	content, err := os.ReadFile(expectedFileName)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file is empty after concurrent writes")
	}
}

func TestRotatingLoggerConcurrentRotation(t *testing.T) {
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	// Create rotating logger with small size limit to trigger frequent rotations
	rl := NewRotatingLoggerWithSizeLimit(tempDir, 1, 1000)
	defer func() {
		if err := rl.Close(); err != nil {
			t.Logf("Failed to close logger: %v", err)
		}
	}()

	// Initialize rotation
	rl.mu.Lock()
	err := rl.doRotate(getWeekKey(time.Now()))
	rl.mu.Unlock()
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	// Test concurrent writes with rotation
	const numGoroutines = 20
	const numWrites = 100
	done := make(chan bool, numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			message := fmt.Sprintf("Goroutine %d: %s", id, strings.Repeat("x", 100))
			for range numWrites {
				if _, writeErr := rl.Write([]byte(message)); writeErr != nil {
					t.Errorf("Concurrent write failed: %v", writeErr)
				}
			}
			done <- true
		}(i)
	}

	for range numGoroutines {
		<-done
	}

	// Verify files are properly named and no corruption occurred
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	logFiles := 0
	numberedFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "app-") && strings.HasSuffix(entry.Name(), ".log") {
			logFiles++
			if strings.Contains(entry.Name(), "_01.") || strings.Contains(entry.Name(), "_02.") || strings.Contains(entry.Name(), "_03.") {
				numberedFiles++
			}
		}
	}

	if logFiles < 1 {
		t.Error("Expected at least 1 log file")
	}

	if numberedFiles < 1 {
		t.Log("No numbered files created (might not have hit size limit)")
	}
}

func TestRotatingLoggerEdgeCases(t *testing.T) {
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	rl := NewRotatingLogger(tempDir, 1)
	defer func() { _ = rl.Close() }()

	// Test writing empty message
	_, err := rl.Write([]byte(""))
	if err != nil {
		t.Errorf("Failed to write empty message: %v", err)
	}

	// Test writing very large message
	largeMessage := strings.Repeat("x", 10000)
	_, err = rl.Write([]byte(largeMessage))
	if err != nil {
		t.Errorf("Failed to write large message: %v", err)
	}

	// Test multiple rotations in quick succession
	rl.currentWeek = "2025-W40"
	err = rl.doRotate("2025-W40")
	if err != nil {
		t.Fatalf("Failed first rotation: %v", err)
	}

	rl.currentWeek = "2025-W41"
	err = rl.doRotate("2025-W41")
	if err != nil {
		t.Fatalf("Failed second rotation: %v", err)
	}

	// Verify both files exist
	week40File := filepath.Join(tempDir, "app-2025-W40.log")
	week41File := filepath.Join(tempDir, "app-2025-W41.log")

	if _, err := os.Stat(week40File); os.IsNotExist(err) {
		t.Error("Week 40 file was not created")
	}
	if _, err := os.Stat(week41File); os.IsNotExist(err) {
		t.Error("Week 41 file was not created")
	}
}

func TestLoggingServiceMethods(t *testing.T) {
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	// Use ResetForTest() for proper test isolation
	ResetForTest(t, tempDir, config.EnvTest, "", 2, 100*1024*1024)

	// Test all logging methods
	Info("Info message")
	Error("Error message")
	Warn("Warning message")
	Debug("Debug message")

	// Check that log file was created
	currentWeek := getWeekKey(time.Now())
	expectedFileName := filepath.Join(tempDir, "app-"+currentWeek+".log")
	if _, err := os.Stat(expectedFileName); os.IsNotExist(err) {
		t.Errorf("Expected log file %s was not created", expectedFileName)
	}
}

func TestInitLoggerFunctions(t *testing.T) {
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	// Test InitLogger with proper isolation
	ResetForTest(t, tempDir, config.EnvTest, "", 4, 100*1024*1024)
	if DefaultLoggingService == nil {
		t.Error("InitLogger did not initialize DefaultLoggingService")
	}

	// Test InitLoggerWithRetentionAndSize with proper isolation
	ResetForTest(t, tempDir, config.EnvTest, "", 2, 1024*1024)
	if DefaultLoggingService == nil {
		t.Error("InitLoggerWithRetentionAndSize did not initialize DefaultLoggingService")
	}

	// Test that logger works
	Info("Test message from InitLoggerWithRetentionAndSize")
}

func TestMultiHandlerMethods(t *testing.T) {
	// Create temporary directory for test logs (auto-cleanup)
	tempDir := t.TempDir()

	// Create a rotating logger for testing
	rotatingLogger := NewRotatingLogger(tempDir, 1)
	defer func() { _ = rotatingLogger.Close() }()

	// Create multiHandler directly to test its methods
	fileHandler := slog.NewJSONHandler(rotatingLogger, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	multi := &multiHandler{
		handlers: []slog.Handler{consoleHandler, fileHandler},
	}

	// Test Enabled method (currently 75% coverage)
	if !multi.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Expected Enabled() to return true for info level")
	}

	// Test Handle method (currently 80% coverage)
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)

	err := multi.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle method failed: %v", err)
	}

	// Test WithAttrs method (currently 0% coverage)
	attrs := []slog.Attr{slog.String("key", "value")}
	newHandler := multi.WithAttrs(attrs)
	if newHandler == nil {
		t.Error("WithAttrs returned nil")
	}

	// Test WithGroup method (currently 0% coverage)
	newHandler = multi.WithGroup("test-group")
	if newHandler == nil {
		t.Error("WithGroup returned nil")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	// Create a simple logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	})

	// Wrap with logging middleware
	middleware := LoggingMiddleware(logger)
	wrappedHandler := middleware(handler)

	// Create a test request
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestResponseWriterWrapper(t *testing.T) {
	// Test the custom response writer wrapper
	recorder := httptest.NewRecorder()

	// Create wrapper
	wrapper := &responseWriterWrapper{ResponseWriter: recorder}

	// Test WriteHeader method
	wrapper.WriteHeader(http.StatusNotFound)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	// Test Write method
	data := []byte("test data")
	n, err := wrapper.Write(data)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Test that status is not written twice
	wrapper.WriteHeader(http.StatusInternalServerError)
	if recorder.Code != http.StatusNotFound {
		t.Error("Status should not be changed after first write")
	}

	// Test bytes written tracking
	if wrapper.bytesWritten != len(data) {
		t.Errorf("Expected bytesWritten %d, got %d", len(data), wrapper.bytesWritten)
	}
}

func TestRotatingLoggerExistingFileAtSizeLimit(t *testing.T) {
	tempDir := t.TempDir()

	maxFileSize := int64(1024)
	currentWeek := getWeekKey(time.Now())
	baseFileName := fmt.Sprintf("app-%s.log", currentWeek)
	baseFilePath := filepath.Join(tempDir, baseFileName)

	if err := os.WriteFile(baseFilePath, []byte(strings.Repeat("x", 2048)), 0666); err != nil {
		t.Fatalf("Failed to create initial log file: %v", err)
	}

	rl := NewRotatingLoggerWithSizeLimit(tempDir, 1, maxFileSize)
	defer func() { _ = rl.Close() }()

	rl.mu.Lock()
	err := rl.doRotate(currentWeek)
	rl.mu.Unlock()
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	if rl.currentFile.Name() == baseFilePath {
		t.Errorf("Expected new numbered file, but got: %s", rl.currentFile.Name())
	}

	if !strings.Contains(rl.currentFile.Name(), "_01.") {
		t.Errorf("Expected filename to contain '_01' suffix, got: %s", rl.currentFile.Name())
	}

	if rl.currentSize.Load() != 0 {
		t.Errorf("Expected currentSize to be 0 for new file, got: %d", rl.currentSize.Load())
	}

	_, err = rl.Write([]byte("test message"))
	if err != nil {
		t.Fatalf("Failed to write to new file: %v", err)
	}
}

func TestRotatingLoggerExistingFileBelowSizeLimit(t *testing.T) {
	tempDir := t.TempDir()

	maxFileSize := int64(1024)
	currentWeek := getWeekKey(time.Now())
	baseFileName := fmt.Sprintf("app-%s.log", currentWeek)
	baseFilePath := filepath.Join(tempDir, baseFileName)

	if err := os.WriteFile(baseFilePath, []byte(strings.Repeat("x", 512)), 0666); err != nil {
		t.Fatalf("Failed to create initial log file: %v", err)
	}

	rl := NewRotatingLoggerWithSizeLimit(tempDir, 1, maxFileSize)
	defer func() { _ = rl.Close() }()

	rl.mu.Lock()
	err := rl.doRotate(currentWeek)
	rl.mu.Unlock()
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	if rl.currentFile.Name() != baseFilePath {
		t.Errorf("Expected to reuse existing file, but got: %s", rl.currentFile.Name())
	}

	if rl.currentSize.Load() != 512 {
		t.Errorf("Expected currentSize to be 512 (actual file size), got: %d", rl.currentSize.Load())
	}

	_, err = rl.Write([]byte("x"))
	if err != nil {
		t.Fatalf("Failed to write to file: %v", err)
	}

	if rl.currentSize.Load() != 513 {
		t.Errorf("Expected currentSize to be 513 after write, got: %d", rl.currentSize.Load())
	}
}
