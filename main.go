package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/giygas/medicaments-api/ansmdocs"
	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/docstore"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/scheduler"
	"github.com/giygas/medicaments-api/server"
	"github.com/joho/godotenv"
)

//go:embed certigna-services-ca.pem
var certignaCA []byte

// Compile-time checks ensuring the concrete docstore and ansmdocs types
// satisfy the interfaces they are wired as. The repo convention places these
// assertions next to the implementation; here they live at the composition
// root because the implementation packages stay interface-agnostic.
var (
	_ interfaces.DocumentStore = (*docstore.DocumentStore)(nil)
	_ interfaces.ANSMFetcher   = (*ansmdocs.Fetcher)(nil)
)

// newCertignaHTTPClient creates an HTTP client that trusts the Certigna Services CA
// intermediate certificate, which is required to verify TLS connections to
// base-donnees-publique.medicaments.gouv.fr. The server does not send the full
// certificate chain, so the intermediate CA is bundled and added to the system cert pool.
// The timeout bounds a single request made through the returned client.
func newCertignaHTTPClient(timeout time.Duration) (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if ok := pool.AppendCertsFromPEM(certignaCA); !ok {
		return nil, fmt.Errorf("failed to parse Certigna intermediate CA cert")
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}, nil
}

// appVersion returns the application version reported in the ANSM fetcher
// User-Agent: the APP_VERSION environment variable when set (the Makefile
// and docker-compose export it), falling back to "dev".
func appVersion() string {
	if v := os.Getenv("APP_VERSION"); v != "" {
		return v
	}
	return "dev"
}

// initDocsDependencies constructs the ANSM document cache and fetcher when
// the feature is enabled (DOCS_ENABLED=true). When disabled it returns nil
// interfaces — the kill switch: the docs endpoints answer 501 and no
// directory scan, rate limiter or upstream request ever happens.
func initDocsDependencies(cfg *config.Config) (interfaces.DocumentStore, interfaces.ANSMFetcher, error) {
	if !cfg.DocsEnabled {
		logging.Info("ANSM documents feature disabled (DOCS_ENABLED=false): docs endpoints will answer 501")
		return nil, nil, nil
	}

	store, err := docstore.NewDocumentStore(cfg.DocsCacheDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize docs cache in %s: %w", cfg.DocsCacheDir, err)
	}

	// The ANSM upstream serves an incomplete TLS chain: reuse the bundled
	// Certigna intermediate CA, but bound each request by the configured
	// DOCS_FETCH_TIMEOUT instead of the parser's generous download budget.
	docsClient, err := newCertignaHTTPClient(cfg.DocsFetchTimeout)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ANSM docs HTTP client: %w", err)
	}

	fetcher := ansmdocs.NewFetcher(ansmdocs.Config{
		UserAgent:  fmt.Sprintf("medicaments-api/%s (+https://medicaments-api.giygas.dev)", appVersion()),
		Timeout:    cfg.DocsFetchTimeout,
		RatePerSec: cfg.DocsFetchRatePerSec,
		Client:     docsClient,
	})

	logging.Info("ANSM documents feature enabled",
		"cache_dir", cfg.DocsCacheDir,
		"fetch_rate_per_sec", cfg.DocsFetchRatePerSec,
		"fetch_timeout", cfg.DocsFetchTimeout.String())
	return store, fetcher, nil
}

func main() {
	// Check if running in healthcheck mode (for Docker HEALTHCHECK)
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("http://localhost:8000/health")
		if err != nil || resp.StatusCode != 200 {
			if resp != nil {
				_ = resp.Body.Close()
			}
			os.Exit(1)
		}
		_ = resp.Body.Close()
		os.Exit(0)
	}

	// Load environment variables first
	if err := loadEnvironment(); err != nil {
		fmt.Printf("Failed to load environment: %v\n", err)
		os.Exit(1)
	}

	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	// Initialize structured logging with rotating logs using config values
	logging.InitLoggerWithEnvironment("logs", cfg.Env, cfg.LogLevel, cfg.LogRetentionWeeks, cfg.MaxLogFileSize)

	// Log configuration on startup
	logging.Info("Configuration loaded successfully",
		"port", cfg.Port,
		"address", cfg.Address,
		"env", cfg.Env.String(),
		"log_level", cfg.LogLevel,
		"allow_direct_access", cfg.AllowDirectAccess,
		"max_request_body", cfg.MaxRequestBody,
		"max_header_size", cfg.MaxHeaderSize,
		"docs_enabled", cfg.DocsEnabled)

	// Initialize data container and parser
	dataContainer := data.NewDataContainer()

	// BDPM TSV downloads are large: keep the parser's generous 5-minute budget
	httpClient, err := newCertignaHTTPClient(5 * time.Minute)
	if err != nil {
		logging.Error("Failed to create HTTP client", "error", err)
		os.Exit(1)
	}
	parser := medicamentsparser.NewMedicamentsParser(httpClient)

	// ANSM documents (RCP/notice) dependencies: nil interfaces when the
	// feature is disabled act as the kill switch (docs endpoints answer
	// 501). They are threaded into the handler via WithDocuments so the
	// /v1/medicaments/{cis}/rcp and /notice routes can serve documents.
	docsStore, docsFetcher, err := initDocsDependencies(cfg)
	if err != nil {
		logging.Error("Failed to initialize ANSM docs dependencies", "error", err)
		os.Exit(1)
	}
	logging.Info("ANSM docs dependencies wired",
		"store_ready", docsStore != nil,
		"fetcher_ready", docsFetcher != nil)

	// Initialize and start scheduler with dependency injection
	sched := scheduler.NewScheduler(dataContainer, parser)
	if err := sched.Start(); err != nil {
		logging.Error("Failed to start scheduler", "error", err)
		os.Exit(1)
	}
	defer sched.Stop()

	// Initialize and start server, wiring the ANSM documents dependencies
	srv := server.NewServer(cfg, dataContainer, handlers.WithDocuments(docsStore, docsFetcher, cfg.DocsFetchTimeout))

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logging.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Block until a signal is received
	<-quit

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop rate limiter cleanup goroutine
	logging.Info("Stopping rate limiter cleanup goroutine...")
	server.StopRateLimiter()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		logging.Error("Server shutdown failed", "error", err)
		os.Exit(1)
	}

	logging.Info("Server shutdown complete")

	// Ensure all logs are flushed before exit
	logging.Close()
}

// loadEnvironment loads environment variables from .env file
func loadEnvironment() error {
	// Try to load .env file
	if err := godotenv.Load(); err != nil {
		// If failed, try loading from executable directory
		ex, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		exPath := filepath.Dir(ex)
		if err := os.Chdir(exPath); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}

		// Try again after changing directory
		if err := godotenv.Load(); err != nil {
			// Check if environment variables are already set (e.g., by Docker)
			if os.Getenv("PORT") == "" && os.Getenv("ENV") == "" {
				// Only warn if env vars aren't already configured
				logging.Warn("Could not load .env file", "error", err)
			}
		}
	}

	return nil
}
