// Package scheduler provides automated data update scheduling and health monitoring
// for the medicaments API. It handles cron-based data updates, health checks,
// and coordinates data refresh operations with the data container using dependency injection.
package scheduler

import (
	"fmt"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
	"github.com/go-co-op/gocron"
)

// Compile-time check to ensure Scheduler implements Scheduler interface
var _ interfaces.Scheduler = (*Scheduler)(nil)

// Scheduler handles data updates and health monitoring using dependency injection
type Scheduler struct {
	dataStore interfaces.DataStore
	parser    interfaces.Parser
	scheduler *gocron.Scheduler
}

// NewScheduler creates a new scheduler instance with injected dependencies
func NewScheduler(dataStore interfaces.DataStore, parser interfaces.Parser) *Scheduler {
	return &Scheduler{
		dataStore: dataStore,
		parser:    parser,
		scheduler: gocron.NewScheduler(time.Local),
	}
}

// Start initializes the scheduler with data updates and health monitoring
func (s *Scheduler) Start() error {
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
func (s *Scheduler) Stop() {
	s.scheduler.Stop()
}

// updateData performs a complete data update using injected dependencies
func (s *Scheduler) updateData() error {
	// Prevent concurrent updates
	if !s.dataStore.BeginUpdate() {
		logging.Info("Update already in progress, skipping...")
		return nil
	}
	defer s.dataStore.EndUpdate()

	logging.Info(fmt.Sprintf("Starting database update at: %s", time.Now().Format(time.RFC3339)))
	start := time.Now()

	// Parse data using injected parser
	newMedicaments, newPresentationsCIP7Map, newPresentationsCIP13Map, err := s.parser.ParseAllMedicaments()
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

	validator := validation.NewDataValidator()
	report := validator.ReportDataQuality(newMedicaments, newGeneriques, newPresentationsCIP7Map, newPresentationsCIP13Map)

	// Log duplicate CIS
	if len(report.DuplicateCIS) > 0 {
		logging.Warn("Duplicate CIS detected",
			"total", len(report.DuplicateCIS),
			"cis_list", report.DuplicateCIS,
		)
	}

	// Log duplicate Group IDs
	if len(report.DuplicateGroupIDs) > 0 {
		logging.Warn("Duplicate group IDs detected",
			"total", len(report.DuplicateGroupIDs),
			"group_ids_list", report.DuplicateGroupIDs,
		)
	}

	// Log medicaments without compositions as WARN (with full CIS list)
	if report.MedicamentsWithoutCompositions > 0 {
		logging.Warn("Medicaments without compositions",
			"count", report.MedicamentsWithoutCompositions,
			"cis_list", report.MedicamentsWithoutCompositionsCIS,
		)
	}

	// Log presentations with orphaned CIS
	if report.PresentationsWithOrphanedCIS > 0 {
		logging.Warn("Presentations with orphaned CIS",
			"count", report.PresentationsWithOrphanedCIS,
			"cip_list", report.PresentationsWithOrphanedCISCIPList,
		)
	}

	// Atomic update using injected data store (including report)
	s.dataStore.UpdateData(newMedicaments, newGeneriques, newMedicamentsMap, newGeneriquesMap, newPresentationsCIP7Map, newPresentationsCIP13Map, report)

	elapsed := time.Since(start)
	logging.Info("Database update completed", "duration", elapsed.String(), "medicament_count", len(newMedicaments))

	return nil
}

// startHealthMonitoring monitors the health of the data updates
func (s *Scheduler) startHealthMonitoring() {
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
