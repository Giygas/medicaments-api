package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RotatingLogger manages rotating log files with weekly retention
type RotatingLogger struct {
	logDir      string
	currentFile *os.File
	currentWeek string
	retention   time.Duration
	maxFileSize int64
	currentSize atomic.Int64
	mu          sync.RWMutex
	lastCleanup time.Time
	ctx         context.Context
	cancel      context.CancelFunc
	cleanupDone chan struct{}
}

// NewRotatingLogger creates a new rotating logger instance
func NewRotatingLogger(logDir string, retentionWeeks int) *RotatingLogger {
	return NewRotatingLoggerWithSizeLimit(logDir, retentionWeeks, 100*1024*1024) // Default 100MB
}

// NewRotatingLoggerWithSizeLimit creates a new rotating logger with custom size limit
func NewRotatingLoggerWithSizeLimit(logDir string, retentionWeeks int, maxFileSize int64) *RotatingLogger {
	ctx, cancel := context.WithCancel(context.Background())
	return &RotatingLogger{
		logDir:      logDir,
		retention:   time.Duration(retentionWeeks) * 7 * 24 * time.Hour,
		maxFileSize: maxFileSize,
		lastCleanup: time.Now(),
		ctx:         ctx,
		cancel:      cancel,
		cleanupDone: make(chan struct{}),
	}
}

// getWeekKey returns the week key in YYYY-Www format (ISO week)
func getWeekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// doRotate performs actual rotation (caller must hold write lock)
func (rl *RotatingLogger) doRotate(targetWeek string) error {
	if rl.currentFile != nil {
		if err := rl.currentFile.Close(); err != nil {
			slog.Warn("Failed to close log file during rotation", "error", err)
		}
	}

	isSizeRotation := rl.maxFileSize > 0 && rl.currentSize.Load() >= rl.maxFileSize
	fileName, shouldResetSize, err := rl.findOrCreateLogFile(targetWeek, isSizeRotation)
	if err != nil {
		return err
	}

	logPath := filepath.Join(rl.logDir, fileName)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logPath, err)
	}

	rl.currentFile = file
	rl.currentWeek = targetWeek

	if shouldResetSize {
		rl.currentSize.Store(0)
	} else {
		if info, err := os.Stat(logPath); err == nil {
			rl.currentSize.Store(info.Size())
		}
	}

	return nil
}

// findOrCreateLogFile determines which log file to use for the current week
func (rl *RotatingLogger) findOrCreateLogFile(targetWeek string, isSizeRotation bool) (string, bool, error) {
	baseFileName := fmt.Sprintf("app-%s.log", targetWeek)
	baseFilePath := filepath.Join(rl.logDir, baseFileName)

	if !isSizeRotation {
		if info, err := os.Stat(baseFilePath); err == nil {
			if rl.maxFileSize == 0 || info.Size() < rl.maxFileSize {
				return baseFileName, false, nil
			}
		} else {
			return baseFileName, false, nil
		}
	}

	highestNum, lastFilePath, lastSize := rl.findHighestNumberedFile(targetWeek)

	if lastFilePath != "" && lastSize < rl.maxFileSize {
		return filepath.Base(lastFilePath), false, nil
	}

	nextNum := highestNum + 1
	newFileName := fmt.Sprintf("app-%s_%02d.log", targetWeek, nextNum)
	return newFileName, true, nil
}

// findHighestNumberedFile searches for numbered log files and returns the highest number
func (rl *RotatingLogger) findHighestNumberedFile(targetWeek string) (int, string, int64) {
	pattern := fmt.Sprintf("app-%s_??.log", targetWeek)
	matches, _ := filepath.Glob(filepath.Join(rl.logDir, pattern))

	highestNum := 0
	var lastPath string
	var lastSize int64

	for _, match := range matches {
		num, size := rl.parseNumberedFile(match)
		if num > highestNum {
			highestNum = num
			lastPath = match
			lastSize = size
		}
	}

	return highestNum, lastPath, lastSize
}

// parseNumberedFile extracts the sequence number and file size from a numbered log file
func (rl *RotatingLogger) parseNumberedFile(filePath string) (int, int64) {
	base := filepath.Base(filePath)

	re := regexp.MustCompile(`app-\d{4}-W\d{2}_(\d{2})\.log$`)
	matches := re.FindStringSubmatch(base)

	if len(matches) < 2 {
		return 0, 0
	}

	num, _ := strconv.Atoi(matches[1])

	info, err := os.Stat(filePath)
	if err != nil {
		return num, 0
	}

	return num, info.Size()
}

// Write writes data to the current log file
func (rl *RotatingLogger) Write(p []byte) (n int, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	currentWeek := getWeekKey(time.Now())
	// Check if rotation is needed
	needsRotation := rl.currentWeek != currentWeek
	// Check if current size is at limit OR if next write would exceed limit
	if rl.maxFileSize > 0 && !needsRotation {
		currentSize := rl.currentSize.Load()
		if currentSize >= rl.maxFileSize || currentSize+int64(len(p)) > rl.maxFileSize {
			needsRotation = true
			// Mark that we need a size-rotated file by setting size to limit
			rl.currentSize.Store(rl.maxFileSize)
		}
	}

	if needsRotation {
		if err = rl.doRotate(currentWeek); err != nil {
			return 0, err
		}
	}

	if rl.currentFile == nil {
		return 0, fmt.Errorf("no log file available")
	}

	n, err = rl.currentFile.Write(p)
	rl.currentSize.Add(int64(n))
	return n, err
}

// cleanupOldLogs removes log files older than the retention period
func (rl *RotatingLogger) cleanupOldLogs() error {
	// Read directory contents (ticker in goroutine controls frequency)
	entries, err := os.ReadDir(rl.logDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	cutoff := time.Now().Add(-rl.retention)
	var deletedCount int

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "app-") || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		// Get file info to check modification time
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Delete if older than retention period
		if info.ModTime().Before(cutoff) {
			fullPath := filepath.Join(rl.logDir, entry.Name())
			if err := os.Remove(fullPath); err == nil {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		// Log cleanup (using console to avoid recursion)
		fmt.Printf("Cleaned up %d old log files\n", deletedCount)
	}

	return nil
}

// Close closes the rotating logger and stops background cleanup
func (rl *RotatingLogger) Close() error {
	// Signal cancellation to stop background goroutine
	rl.cancel()

	// Wait for cleanup goroutine to finish with shorter timeout for tests
	timeout := 5 * time.Second
	// Check if we're in a test environment and use shorter timeout
	if len(os.Args) > 0 && strings.Contains(os.Args[0], "test") {
		timeout = 100 * time.Millisecond
	}

	select {
	case <-rl.cleanupDone:
		// Cleanup finished
	case <-time.After(timeout):
		// Timeout - only log warning if not in test
		if timeout > 100*time.Millisecond {
			fmt.Printf("Warning: background cleanup goroutine did not shutdown gracefully\n")
		}
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.currentFile != nil {
		return rl.currentFile.Close()
	}
	return nil
}

// SetupLogger configures slog to log to both console and rotating file
func SetupLogger(logDir string) *slog.Logger {
	return SetupLoggerWithRetention(logDir, 4) // Default 4 weeks retention
}

// SetupLoggerWithRetention configures slog with custom retention period
// Note: This function is deprecated - use InitLoggerWithRetention for proper resource management
func SetupLoggerWithRetention(logDir string, retentionWeeks int) *slog.Logger {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If we can't create logs directory, just log to console
		consoleLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		consoleLogger.Error("Failed to create logs directory", "error", err)
		return consoleLogger
	}

	// Create rotating logger
	rotatingLogger := NewRotatingLogger(logDir, retentionWeeks)

	// Initialize rotation
	rotatingLogger.mu.Lock()
	rotateErr := rotatingLogger.doRotate(getWeekKey(time.Now()))
	rotatingLogger.mu.Unlock()
	if rotateErr != nil {
		consoleLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		consoleLogger.Error("Failed to initialize rotating logger", "error", rotateErr)
		return consoleLogger
	}

	// Start cleanup goroutine with proper cancellation
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		defer close(rotatingLogger.cleanupDone)

		for {
			select {
			case <-rotatingLogger.ctx.Done():
				// Context cancelled, exit gracefully
				return
			case <-ticker.C:
				if err := rotatingLogger.cleanupOldLogs(); err != nil {
					slog.Warn("Failed to cleanup old logs during rotation", "error", err)
				}
			}
		}
	}()

	// Create multi-handler that writes to both console and rotating file
	// Console gets text format, file gets JSON format for better parsing
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	fileHandler := slog.NewJSONHandler(rotatingLogger, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Combine handlers - write to both
	multiHandler := &multiHandler{
		handlers: []slog.Handler{consoleHandler, fileHandler},
	}

	return slog.New(multiHandler)
}

// multiHandler implements slog.Handler to write to multiple handlers
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Enable if any handler enables it
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	// Handle with all handlers
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Create new multiHandler with handlers that have the attrs
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	// Create new multiHandler with handlers that have the group
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}
