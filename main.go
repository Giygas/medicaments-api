package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"
	_ "net/http/pprof"
)

// DataContainer holds all the data with atomic pointers for zero-downtime updates
type DataContainer struct {
	medicaments    atomic.Value // []entities.Medicament
	generiques     atomic.Value // []entities.GeneriqueList
	medicamentsMap atomic.Value // map[int]entities.Medicament
	generiquesMap  atomic.Value // map[int]entities.Generique
	lastUpdated    atomic.Value // time.Time
	updating       atomic.Bool
}

var dataContainer = &DataContainer{}
var serverStartTime = time.Now()

func scheduleMedicaments() {

	s := gocron.NewScheduler(time.Local)

	// Initial load
	if err := updateData(); err != nil {
		logging.Error("Failed to perform initial data load", "error", err)
		os.Exit(1)
	}

	// Schedule updates
	_, err := s.Every(1).Days().At("06:00;18:00").Do(func() {
		if err := updateData(); err != nil {
			logging.Error("Failed to update data", "error", err)
		}
	})

	if err != nil {
		logging.Error("Failed to schedule updates", "error", err)
		os.Exit(1)
	}

	s.StartAsync()

	// Health monitoring
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			lastUpdate := GetLastUpdated()
			if time.Since(lastUpdate) > 25*time.Hour {
				logging.Warn("Data hasn't been updated in over 25 hours")
			}
		}
	}()
}

// Thread-safe getters with type check

func GetMedicaments() []entities.Medicament {

	if v := dataContainer.medicaments.Load(); v != nil {
		if medicaments, ok := v.([]entities.Medicament); ok {
			return medicaments
		}
	}

	logging.Warn("Medicaments list is empty or invalid")

	return []entities.Medicament{}
}

func GetGeneriques() []entities.GeneriqueList {

	if v := dataContainer.generiques.Load(); v != nil {
		if generiques, ok := v.([]entities.GeneriqueList); ok {
			return generiques
		}
	}

	logging.Warn("GeneriqueList is empty or invalid")

	return []entities.GeneriqueList{}
}

func GetMedicamentsMap() map[int]entities.Medicament {

	if v := dataContainer.medicamentsMap.Load(); v != nil {
		if medicamentsMap, ok := v.(map[int]entities.Medicament); ok {
			return medicamentsMap
		}
	}

	logging.Warn("MedicamentsMap is empty or invalid")

	return make(map[int]entities.Medicament)
}

func GetGeneriquesMap() map[int]entities.Generique {

	if v := dataContainer.generiquesMap.Load(); v != nil {
		if generiquesMap, ok := v.(map[int]entities.Generique); ok {
			return generiquesMap
		}
	}

	logging.Warn("GeneriquesMap is empty or invalid")

	return make(map[int]entities.Generique)

}
func GetLastUpdated() time.Time {

	if v := dataContainer.lastUpdated.Load(); v != nil {
		if lastUpdated, ok := v.(time.Time); ok {
			return lastUpdated
		}
	}

	logging.Warn("Could not get the last updated value")

	return time.Time{}
}

func IsUpdating() bool {
	return dataContainer.updating.Load()
}

// calculateNextUpdate calculates the next scheduled update time based on the cron schedule (06:00;18:00)
func calculateNextUpdate() time.Time {
	now := time.Now()

	// Get today's 6:00 AM and 6:00 PM times
	sixAM := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
	sixPM := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())

	// If current time is before 6:00 AM, next update is 6:00 AM today
	if now.Before(sixAM) {
		return sixAM
	}

	// If current time is between 6:00 AM and 6:00 PM, next update is 6:00 PM today
	if now.Before(sixPM) {
		return sixPM
	}

	// If current time is after 6:00 PM, next update is 6:00 AM tomorrow
	tomorrow := now.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 6, 0, 0, 0, now.Location())
}

// formatUptimeHuman formats duration into a human-readable string
func formatUptimeHuman(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	parts = append(parts, fmt.Sprintf("%ds", seconds))

	return strings.Join(parts, " ")
}

func updateData() error {

	// Prevent concurrent updates
	if !dataContainer.updating.CompareAndSwap(false, true) {
		logging.Info("Update already in progress, skipping...")
		return nil
	}
	defer dataContainer.updating.Store(false)

	fmt.Println("Starting database update at:", time.Now())
	start := time.Now()

	// Parse data into temporary variables (not affecting current data)
	newMedicaments, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		logging.Error("Failed to parse medicaments", "error", err)
		return fmt.Errorf("failed to parse medicaments: %w", err)
	}

	// Create new maps
	newMedicamentsMap := make(map[int]entities.Medicament)
	for i := range newMedicaments {
		newMedicamentsMap[newMedicaments[i].Cis] = newMedicaments[i]
	}

	newGeneriques, newGeneriquesMap, err := medicamentsparser.GeneriquesParser(&newMedicaments, &newMedicamentsMap)
	if err != nil {
		logging.Error("Failed to parse generiques", "error", err)
		return fmt.Errorf("failed to parse generiques: %w", err)
	}

	// Atomic swap (zero downtime replacement)
	dataContainer.medicaments.Store(newMedicaments)
	dataContainer.medicamentsMap.Store(newMedicamentsMap)
	dataContainer.generiques.Store(newGeneriques)
	dataContainer.generiquesMap.Store(newGeneriquesMap)
	dataContainer.lastUpdated.Store(time.Now())

	elapsed := time.Since(start)
	logging.Info("Database update completed", "duration", elapsed.String(), "medicament_count", len(newMedicaments))

	return nil
}

func init() {

	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		logging.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

	// Initialize stores with empty data
	dataContainer.medicaments.Store(make([]entities.Medicament, 0))
	dataContainer.generiques.Store(make([]entities.GeneriqueList, 0))
	dataContainer.medicamentsMap.Store(make(map[int]entities.Medicament))
	dataContainer.generiquesMap.Store(make(map[int]entities.Generique))
	dataContainer.lastUpdated.Store(time.Time{})

	// Get the working directory and read the env variables
	err = godotenv.Load()
	if err != nil {
		// If failed, try loading from executable directory
		ex, err := os.Executable()
		if err != nil {
			logging.Error("Failed to get executable path", "error", err)
			os.Exit(1)
		}

		exPath := filepath.Dir(ex)

		err = os.Chdir(exPath)
		if err != nil {
			logging.Error("Failed to change directory", "error", err)
			os.Exit(1)
		}

	}

	// Log configuration on startup
	logging.Info("Configuration loaded successfully",
		"port", cfg.Port,
		"address", cfg.Address,
		"env", cfg.Env,
		"log_level", cfg.LogLevel,
		"max_request_body", cfg.MaxRequestBody,
		"max_header_size", cfg.MaxHeaderSize)

	go scheduleMedicaments()
}

func main() {

	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	// Initialize slog for structured logging to console and file
	logging.InitLogger("logs")

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(realIPMiddleware)
	router.Use(logging.LoggingMiddleware(logging.DefaultLoggingService.Logger))
	router.Use(middleware.RedirectSlashes)
	router.Use(middleware.Recoverer)
	router.Use(blockDirectAccessMiddleware)
	router.Use(requestSizeMiddleware(cfg))

	// CORS headers are now handled by nginx configuration

	router.Use(rateLimitHandler)

	server := &http.Server{
		Handler:      router,
		Addr:         cfg.Address + ":" + cfg.Port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// API routes
	router.Get("/database/{pageNumber}", servePagedMedicaments)
	router.Get("/database", serveAllMedicaments)
	router.Get("/medicament/{element}", findMedicament)
	router.Get("/medicament/id/{cis}", findMedicamentByID)
	router.Get("/generiques/{libelle}", findGeneriques)
	router.Get("/generiques/group/{groupId}", findGeneriquesByGroupID)
	router.Get("/health", healthCheck)

	// Profiling endpoint (accessible at /debug/pprof/) - only for local dev
	if cfg.Env == "dev" {
		go func() {
			fmt.Println("Profiling server started at http://localhost:6060/debug/pprof/")
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				fmt.Println("Profiling server failed: ", err)
				os.Exit(1)
			}
		}()
	}

	// Serve documentation with caching
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// Set caching headers for HTML
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		http.ServeFile(w, r, "html/index.html")
	})

	// Serve OpenAPI specification
	router.Get("/docs/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		http.ServeFile(w, r, "html/openapi.yaml")
	})

	// Serve Swagger UI documentation
	router.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "html/docs.html")
	})

	// Favicon
	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		// Long cache for favicon since it rarely changes
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
		w.Header().Set("Content-Type", "image/x-icon")

		http.ServeFile(w, r, "html/favicon.ico")
	})

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	// Register the channel to receive specific signals
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		logging.Info(fmt.Sprintf("Starting server at: %s:%s", cfg.Address, cfg.Port))

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logging.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Block until a signal is received
	<-quit
	logging.Info("Shutting down server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logging.Error("Server forced to shutdown", "error", err)
		// If graceful shutdown fails, force close
		if err := server.Close(); err != nil {
			logging.Error("Server close error", "error", err)
		}
	}

	// Wait a bit for any ongoing requests to complete
	logging.Info("Waiting for ongoing requests to complete...")
	time.Sleep(2 * time.Second)

	logging.Info("Server shutdown complete")
}

// Health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
	// Get memory statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsageMB := int(m.Alloc / 1024 / 1024)

	// Calculate uptime
	uptime := time.Since(serverStartTime)
	uptimeSeconds := int(uptime.Seconds())

	// Determine health status based on various factors
	status := "healthy"
	lastUpdate := GetLastUpdated()

	// Check if data is stale (more than 25 hours)
	if time.Since(lastUpdate) > 25*time.Hour {
		status = "degraded"
	}

	// Check if currently updating
	if IsUpdating() {
		status = "degraded"
	}

	// Check if no data available
	if len(GetMedicaments()) == 0 {
		status = "unhealthy"
	}

	// Calculate additional metrics
	dataAgeHours := time.Since(lastUpdate).Hours()
	nextUpdate := calculateNextUpdate()
	uptimeHuman := formatUptimeHuman(uptime)

	healthStatus := map[string]any{
		"status":         status,
		"uptime_seconds": uptimeSeconds,
		"uptime_human":   uptimeHuman,
		"last_updated":   GetLastUpdated(),
		"data": map[string]any{
			"medicament_count": len(GetMedicaments()),
			"generique_count":  len(GetGeneriques()),
			"data_age_hours":   dataAgeHours,
			"next_update":      nextUpdate,
		},
		"system": map[string]any{
			"updating":        IsUpdating(),
			"memory_usage_mb": memoryUsageMB,
			"goroutines":      runtime.NumGoroutine(),
		},
	}

	respondWithJSON(w, 200, healthStatus)
}
