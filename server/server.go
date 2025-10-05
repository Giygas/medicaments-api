// Package server provides HTTP server management and lifecycle handling for the medicaments API.
// It includes server setup, middleware configuration, route management, and graceful shutdown
// capabilities with proper error handling and logging.
package server

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/health"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/validation"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Global server start time
var serverStartTime = time.Now()

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

// Server represents the HTTP server
type Server struct {
	server        *http.Server
	router        chi.Router
	dataContainer *data.DataContainer
	config        *config.Config
	httpHandler   interfaces.HTTPHandler
	healthChecker interfaces.HealthChecker
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, dataContainer *data.DataContainer) *Server {
	router := chi.NewRouter()

	// Create interface-based dependencies
	validator := validation.NewDataValidator()
	httpHandler := handlers.NewHTTPHandler(dataContainer, validator)
	healthChecker := health.NewHealthChecker(dataContainer)

	server := &Server{
		server: &http.Server{
			Handler:      router,
			Addr:         cfg.Address + ":" + cfg.Port,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		router:        router,
		dataContainer: dataContainer,
		config:        cfg,
		httpHandler:   httpHandler,
		healthChecker: healthChecker,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// setupMiddleware configures all middleware
func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(BlockDirectAccessMiddleware) // Put BEFORE RealIPMiddleware to see original RemoteAddr
	s.router.Use(RealIPMiddleware)
	s.router.Use(logging.LoggingMiddleware(logging.DefaultLoggingService.Logger))
	s.router.Use(middleware.RedirectSlashes)
	s.router.Use(middleware.Recoverer)
	s.router.Use(RequestSizeMiddleware(s.config))
	s.router.Use(RateLimitHandler)
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// API routes using clean interface-based handlers
	s.router.Get("/database", s.httpHandler.ServeAllMedicaments)
	s.router.Get("/database/{pageNumber}", s.httpHandler.ServePagedMedicaments)
	s.router.Get("/medicament/{element}", s.httpHandler.FindMedicament)
	s.router.Get("/medicament/id/{cis}", s.httpHandler.FindMedicamentByID)
	s.router.Get("/generiques/{libelle}", s.httpHandler.FindGeneriques)
	s.router.Get("/generiques/group/{groupId}", s.httpHandler.FindGeneriquesByGroupID)
	s.router.Get("/health", s.httpHandler.HealthCheck)

	// Documentation routes
	s.setupDocumentationRoutes()
}

// setupDocumentationRoutes configures documentation and static file routes
func (s *Server) setupDocumentationRoutes() {
	// Serve documentation with caching
	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "html/index.html")
	})

	// Serve OpenAPI specification
	s.router.Get("/docs/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		http.ServeFile(w, r, "html/docs/openapi.yaml")
	})

	// Serve Swagger UI documentation
	s.router.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "html/docs.html")
	})

	// Favicon
	s.router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
		w.Header().Set("Content-Type", "image/x-icon")
		http.ServeFile(w, r, "html/favicon.ico")
	})
}

// Start starts the server
func (s *Server) Start() error {
	// Start profiling server if in development mode
	if s.config.Env == "dev" {
		s.startProfilingServer()
	}

	logging.Info(fmt.Sprintf("Starting server at: %s:%s", s.config.Address, s.config.Port))
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	logging.Info("Shutting down server...")

	if err := s.server.Shutdown(ctx); err != nil {
		logging.Error("Server forced to shutdown", "error", err)
		// If graceful shutdown fails, force close
		if err := s.server.Close(); err != nil {
			logging.Error("Server close error", "error", err)
			return err
		}
	}

	// Wait a bit for any ongoing requests to complete
	logging.Info("Waiting for ongoing requests to complete...")
	time.Sleep(2 * time.Second)

	logging.Info("Server shutdown complete")
	return nil
}

// startProfilingServer starts the pprof profiling server in development mode
func (s *Server) startProfilingServer() {
	go func() {
		fmt.Println("Profiling server started at http://localhost:6060/debug/pprof/")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			fmt.Println("Profiling server failed: ", err)
		}
	}()
}

// HealthData represents health check response data
type HealthData struct {
	Status          string `json:"status"`
	Uptime          string `json:"uptime"`
	MemoryUsage     int    `json:"memory_usage_mb"`
	LastUpdate      string `json:"last_update"`
	NextUpdate      string `json:"next_update"`
	IsUpdating      bool   `json:"is_updating"`
	MedicamentCount int    `json:"medicament_count"`
	GeneriqueCount  int    `json:"generique_count"`
}

// GetHealthData returns current health statistics
func (s *Server) GetHealthData() HealthData {
	// Get memory statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsageMB := int(m.Alloc / 1024 / 1024)

	// Calculate uptime
	uptime := time.Since(serverStartTime)

	// Get health data from interface-based health checker
	status, details, err := s.healthChecker.HealthCheck()
	if err != nil {
		status = "unhealthy"
	}

	// Extract data from details
	data := details["data"].(map[string]interface{})

	return HealthData{
		Status:          status,
		Uptime:          formatUptimeHuman(uptime),
		MemoryUsage:     memoryUsageMB,
		LastUpdate:      details["last_update"].(string),
		NextUpdate:      data["next_update"].(string),
		IsUpdating:      data["is_updating"].(bool),
		MedicamentCount: int(data["medicaments"].(float64)),
		GeneriqueCount:  int(data["generiques"].(float64)),
	}
}
