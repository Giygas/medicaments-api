package scheduler

import (
	"fmt"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-co-op/gocron"
)

// SchedulerWithDI is a version of the scheduler that uses dependency injection
// This demonstrates how interfaces improve testability and flexibility
type SchedulerWithDI struct {
	dataStore interfaces.DataStore
	parser    interfaces.Parser
	scheduler *gocron.Scheduler
}

// NewSchedulerWithDI creates a new scheduler with injected dependencies
func NewSchedulerWithDI(dataStore interfaces.DataStore, parser interfaces.Parser) *SchedulerWithDI {
	return &SchedulerWithDI{
		dataStore: dataStore,
		parser:    parser,
		scheduler: gocron.NewScheduler(time.Local),
	}
}

// Start initializes the scheduler with data updates and health monitoring
func (s *SchedulerWithDI) Start() error {
	// Initial load
	if err := s.updateData(); err != nil {
		logging.Error("Failed to perform initial data load", "error", err)
		return fmt.Errorf("initial data load failed: %w", err)
	}

	// Schedule updates at 06:00 and 18:00 daily
	_, err := s.scheduler.Every(1).Days().At("06:00;18:00").Do(func() {
		if err := s.updateData(); err != nil {
			logging.Error("Failed to update data", "error", err)
		}
	})

	if err != nil {
		logging.Error("Failed to schedule updates", "error", err)
		return fmt.Errorf("failed to schedule updates: %w", err)
	}

	s.scheduler.StartAsync()

	// Start health monitoring
	s.startHealthMonitoring()

	return nil
}

// Stop stops the scheduler
func (s *SchedulerWithDI) Stop() {
	s.scheduler.Stop()
}

// updateData performs a complete data update using injected dependencies
func (s *SchedulerWithDI) updateData() error {
	// Prevent concurrent updates
	if !s.dataStore.BeginUpdate() {
		logging.Info("Update already in progress, skipping...")
		return nil
	}
	defer s.dataStore.EndUpdate()

	fmt.Println("Starting database update at:", time.Now())
	start := time.Now()

	// Parse data using injected parser
	newMedicaments, err := s.parser.ParseAllMedicaments()
	if err != nil {
		logging.Error("Failed to parse medicaments", "error", err)
		return fmt.Errorf("failed to parse medicaments: %w", err)
	}

	// Create new maps
	newMedicamentsMap := make(map[int]entities.Medicament)
	for i := range newMedicaments {
		newMedicamentsMap[newMedicaments[i].Cis] = newMedicaments[i]
	}

	newGeneriques, newGeneriquesMap, err := s.parser.GeneriquesParser(&newMedicaments, &newMedicamentsMap)
	if err != nil {
		logging.Error("Failed to parse generiques", "error", err)
		return fmt.Errorf("failed to parse generiques: %w", err)
	}

	// Atomic update using injected data store
	s.dataStore.UpdateData(newMedicaments, newGeneriques, newMedicamentsMap, newGeneriquesMap)

	elapsed := time.Since(start)
	logging.Info("Database update completed", "duration", elapsed.String(), "medicament_count", len(newMedicaments))

	return nil
}

// startHealthMonitoring monitors the health of the data updates
func (s *SchedulerWithDI) startHealthMonitoring() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			lastUpdate := s.dataStore.GetLastUpdated()
			if time.Since(lastUpdate) > 25*time.Hour {
				logging.Warn("Data hasn't been updated in over 25 hours")
			}
		}
	}()
}
