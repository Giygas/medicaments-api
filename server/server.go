// Package server provides HTTP server management and lifecycle handling for the medicaments API.
// It includes server setup, middleware configuration, route management, and graceful shutdown
// capabilities with proper error handling and logging.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/health"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/metrics"
	"github.com/giygas/medicaments-api/validation"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the HTTP server
type Server struct {
	server        *http.Server
	router        chi.Router
	dataContainer *data.DataContainer
	config        *config.Config
	httpHandler   interfaces.HTTPHandler
	healthChecker interfaces.HealthChecker
	startTime     time.Time

	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc

	metricsServer   *http.Server
	profilingServer *http.Server
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, dataContainer *data.DataContainer) *Server {
	router := chi.NewRouter()

	// Dependencies
	validator := validation.NewDataValidator()
	healthChecker := health.NewHealthChecker(dataContainer)
	httpHandler := handlers.NewHTTPHandler(dataContainer, validator, healthChecker)

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	server := &Server{
		server: &http.Server{
			Handler:      router,
			Addr:         cfg.Address + ":" + cfg.Port,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		router:         router,
		dataContainer:  dataContainer,
		config:         cfg,
		httpHandler:    httpHandler,
		healthChecker:  healthChecker,
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// setupMiddleware configures all middleware
func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(BlockDirectAccessMiddleware(s.config.AllowDirectAccess)) // Put BEFORE RealIPMiddleware to see original RemoteAddr
	s.router.Use(RealIPMiddleware)
	s.router.Use(logging.LoggingMiddleware(logging.DefaultLoggingService.Logger))
	s.router.Use(middleware.RedirectSlashes)
	s.router.Use(middleware.Recoverer)
	s.router.Use(RequestSizeMiddleware(s.config))

	s.router.Use(metrics.Metrics)

	// Disable rate limiting in test mode to measure true throughput performance
	if s.config.Env != config.EnvTest {
		s.router.Use(RateLimitHandler)
	}
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Old API routes
	s.router.Get("/database", s.httpHandler.ExportMedicaments)
	s.router.Get("/database/{pageNumber}", s.httpHandler.ServePagedMedicaments)
	s.router.Get("/medicament/cip/{cip}", s.httpHandler.FindMedicamentByCIP)
	s.router.Get("/medicament/id/{cis}", s.httpHandler.FindMedicamentByCIS)
	s.router.Get("/medicament/{element}", s.httpHandler.FindMedicament) // General string search
	s.router.Get("/generiques/group/{groupId}", s.httpHandler.FindGeneriquesByGroupID)
	s.router.Get("/generiques/{libelle}", s.httpHandler.FindGeneriques)
	// This will stay between in all versions
	s.router.Get("/health", s.httpHandler.HealthCheck)

	// Documentation routes
	s.setupDocumentationRoutes()

	// V1 routes
	s.router.Get("/v1/medicaments/export", s.httpHandler.ExportMedicaments)
	s.router.Get("/v1/medicaments", s.httpHandler.ServeMedicamentsV1)
	s.router.Get("/v1/medicaments/{cis}", s.httpHandler.FindMedicamentByCIS)
	s.router.Get("/v1/presentations/{cip}", s.httpHandler.ServePresentationsV1)
	s.router.Get("/v1/generiques/{groupID}", s.httpHandler.FindGeneriquesByGroupID)
	s.router.Get("/v1/generiques", s.httpHandler.ServeGeneriquesV1)
	s.router.Get("/v1/diagnostics", s.httpHandler.ServeDiagnosticsV1)

	// Will get a 404 otherwise
	s.router.Get("/v1/presentations/", s.httpHandler.ServePresentationsMissingCIP)

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

	// Cache test page
	s.router.Get("/cache-test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache") // Don't cache the test page itself
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "cache_test.html")
	})
}

// Start starts the server
func (s *Server) Start() error {
	// Set the actual start time when server begins listening
	s.startTime = time.Now()
	s.dataContainer.SetServerStartTime(s.startTime)

	// Start profiling server if in development mode
	if s.config.Env == config.EnvDevelopment {
		s.startProfilingServer()
	}

	s.startMetricsServer()

	logging.Info(fmt.Sprintf("Starting server at: %s:%s", s.config.Address, s.config.Port))
	return s.server.ListenAndServe()
}

// Start metrics server
func (s *Server) startMetricsServer() {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	metricsAddr := func() string {
		if s.config.AllowDirectAccess {
			return "0.0.0.0:9090"
		}
		return "127.0.0.1:9090"
	}()

	s.metricsServer = &http.Server{
		Addr:              metricsAddr,
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logging.Info("Metrics: http://" + metricsAddr + "/metrics")

	go func() {
		if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Error("Metrics server failed", "error", err)
		}
	}()

	go func() {
		<-s.shutdownCtx.Done()
		logging.Info("Shutting down metrics server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.metricsServer.Shutdown(shutdownCtx); err != nil {
			logging.Error("Metrics server shutdown error", "error", err)
		}
	}()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	logging.Info("Shutting down server...")

	s.shutdownCancel()

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

	return nil
}

// startProfilingServer starts the pprof profiling server in development mode
func (s *Server) startProfilingServer() {
	profilingAddr := func() string {
		if s.config.AllowDirectAccess {
			return "0.0.0.0:6060"
		}
		return "127.0.0.1:6060"
	}()

	s.profilingServer = &http.Server{
		Addr:              profilingAddr,
		Handler:           nil,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logging.Info("Profiling server started at http://" + profilingAddr + "/debug/pprof/")

	go func() {
		if err := s.profilingServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Warn("Profiling server failed", "error", err)
		}
	}()

	go func() {
		<-s.shutdownCtx.Done()
		logging.Info("Shutting down profiling server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.profilingServer.Shutdown(shutdownCtx); err != nil {
			logging.Warn("Profiling server shutdown error", "error", err)
		}
	}()
}
