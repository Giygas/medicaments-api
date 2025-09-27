package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/giygas/medicamentsfr/medicamentsparser"
	"github.com/giygas/medicamentsfr/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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

func scheduleMedicaments() {
	s := gocron.NewScheduler(time.Local)

	// Initial load
	if err := updateData(); err != nil {
		log.Fatal("Failed to perform initial data load:", err)
	}

	// Schedule updates
	_, err := s.Every(1).Days().At("06:00;18:00").Do(func() {
		if err := updateData(); err != nil {
			log.Printf("Failed to update data: %v", err)
		}
	})

	if err != nil {
		log.Fatal("Failed to schedule updates:", err)
	}

	s.StartAsync()

	// Health monitoring
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			lastUpdate := GetLastUpdated()
			if time.Since(lastUpdate) > 25*time.Hour {
				log.Println("WARNING: Data hasn't been updated in over 25 hours")
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

	log.Printf("Warning: Medicaments list is empty or invalid")

	return []entities.Medicament{}
}

func GetGeneriques() []entities.GeneriqueList {
	if v := dataContainer.generiques.Load(); v != nil {
		if generiques, ok := v.([]entities.GeneriqueList); ok {
			return generiques
		}
	}

	log.Printf("Warning: GeneriqueList is empty or invalid")

	return []entities.GeneriqueList{}
}

func GetMedicamentsMap() map[int]entities.Medicament {
	if v := dataContainer.medicamentsMap.Load(); v != nil {
		if medicamentsMap, ok := v.(map[int]entities.Medicament); ok {
			return medicamentsMap
		}
	}

	log.Printf("Warning: MedicamentsMap is empty or invalid")

	return make(map[int]entities.Medicament)
}

func GetGeneriquesMap() map[int]entities.Generique {
	if v := dataContainer.generiquesMap.Load(); v != nil {
		if generiquesMap, ok := v.(map[int]entities.Generique); ok {
			return generiquesMap
		}
	}

	log.Printf("Warning: GeneriquesMap is empty or invalid")

	return make(map[int]entities.Generique)

}
func GetLastUpdated() time.Time {
	if v := dataContainer.lastUpdated.Load(); v != nil {
		if lastUpdated, ok := v.(time.Time); ok {
			return lastUpdated
		}
	}

	log.Printf("Warning: could not get the last updated value")

	return time.Time{}
}

func IsUpdating() bool {
	return dataContainer.updating.Load()
}

func updateData() error {
	// Prevent concurrent updates
	if !dataContainer.updating.CompareAndSwap(false, true) {
		log.Println("Update already in progress, skipping...")
		return nil
	}
	defer dataContainer.updating.Store(false)

	fmt.Println("Starting database update at:", time.Now())
	start := time.Now()

	// Parse data into temporary variables (not affecting current data)
	newMedicaments := medicamentsparser.ParseAllMedicaments()

	// Create new maps
	newMedicamentsMap := make(map[int]entities.Medicament)
	for i := range newMedicaments {
		newMedicamentsMap[newMedicaments[i].Cis] = newMedicaments[i]
	}

	newGeneriques, newGeneriquesMap := medicamentsparser.GeneriquesParser(&newMedicaments, &newMedicamentsMap)

	// Atomic swap (zero downtime replacement)
	dataContainer.medicaments.Store(newMedicaments)
	dataContainer.medicamentsMap.Store(newMedicamentsMap)
	dataContainer.generiques.Store(newGeneriques)
	dataContainer.generiquesMap.Store(newGeneriquesMap)
	dataContainer.lastUpdated.Store(time.Now())

	elapsed := time.Since(start)
	log.Printf("Database update completed in %s. Loaded %d medicaments", elapsed, len(newMedicaments))
	os.Stdout.Sync()

	return nil
}

func init() {
	// Initialize stores with empty data
	dataContainer.medicaments.Store(make([]entities.Medicament, 0))
	dataContainer.generiques.Store(make([]entities.GeneriqueList, 0))
	dataContainer.medicamentsMap.Store(make(map[int]entities.Medicament))
	dataContainer.generiquesMap.Store(make(map[int]entities.Generique))
	dataContainer.lastUpdated.Store(time.Time{})

	// Get the working directory and read the env variables
	err := godotenv.Load()
	if err != nil {
		// If failed, try loading from executable directory
		ex, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}

		exPath := filepath.Dir(ex)

		err = os.Chdir(exPath)
		if err != nil {
			log.Fatal(err)
		}

	}
	go scheduleMedicaments()
}

func main() {

	portString := os.Getenv("PORT")
	if portString == "" {
		log.Fatal("PORT is not found in the environment")
	}
	adressString := os.Getenv("ADDRESS")
	if adressString == "" {
		adressString = "127.0.0.1" // default to localhost
	}
	environment := os.Getenv("ENV")
	if environment == "" {
		environment = "dev" //default as dev
	}

	router := chi.NewRouter()

	router.Use(middleware.RedirectSlashes)
	router.Use(middleware.RequestID)
	router.Use(realIPMiddleware)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(blockDirectAccessMiddleware)

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	router.Use(rateLimitHandler)

	server := &http.Server{
		Handler:      router,
		Addr:         adressString + ":" + portString,
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
	if environment == "dev" {
		go func() {
			log.Println("Profiling server started at http://localhost:6060/debug/pprof/")
			log.Fatal(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Serve documentation with caching
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// Set caching headers for HTML
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		http.ServeFile(w, r, "html/index.html")
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
		fmt.Printf("Starting server at %s:%v\n", adressString, portString)
		fmt.Printf("Will be accessible via nginx at: http://your-server/medicamentsfr\n")

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed to start: %v\n", err)
		}
	}()

	// Block until a signal is received
	<-quit
	log.Println("Shutting down server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		// If graceful shutdown fails, force close
		if err := server.Close(); err != nil {
			log.Printf("Server close error: %v", err)
		}
	} else {
		log.Println("Server exited gracefully")
	}

	// Wait a bit for any ongoing requests to complete
	log.Println("Waiting for ongoing requests to complete...")
	time.Sleep(2 * time.Second)

	log.Println("Server shutdown complete")
}

// Health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"status":           "healthy",
		"medicament_count": len(GetMedicaments()),
		"generique_count":  len(GetGeneriques()),
		"last_updated":     GetLastUpdated(),
		"updating":         IsUpdating(),
	}

	respondWithJSON(w, 200, status)
}
