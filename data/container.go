// Package data provides thread-safe data storage and management for the medicaments API.
// It includes the DataContainer struct with atomic operations for zero-downtime updates
// and thread-safe access methods for medicaments and generiques data.
package data

import (
	"sync/atomic"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// Compile-time check to ensure DataContainer implements DataStore
var _ interfaces.DataStore = (*DataContainer)(nil)

// DataContainer holds all the data with atomic pointers for zero-downtime updates
type DataContainer struct {
	medicaments           atomic.Value // []entities.Medicament
	generiques            atomic.Value // []entities.GeneriqueList
	medicamentsMap        atomic.Value // map[int]entities.Medicament
	generiquesMap         atomic.Value // map[int][]entities.Generique
	presentationsCIP7Map  atomic.Value //map[int]entities.Presentation
	presentationsCIP13Map atomic.Value //map[int]entities.Presentation
	lastUpdated           atomic.Value // time.Time
	updating              atomic.Bool
	serverStartTime       atomic.Value // time.Time
}

// NewDataContainer creates a new DataContainer with empty data
func NewDataContainer() *DataContainer {
	dc := &DataContainer{}
	dc.medicaments.Store(make([]entities.Medicament, 0))
	dc.generiques.Store(make([]entities.GeneriqueList, 0))
	dc.medicamentsMap.Store(make(map[int]entities.Medicament))
	dc.generiquesMap.Store(make(map[int][]entities.Generique))
	dc.presentationsCIP7Map.Store(make(map[int]entities.Presentation))
	dc.presentationsCIP13Map.Store(make(map[int]entities.Presentation))
	dc.lastUpdated.Store(time.Time{})
	dc.serverStartTime.Store(time.Time{}) // Initialize with zero value
	return dc
}

// Thread-safe getters with type check

// GetMedicaments returns the list of medicaments
func (dc *DataContainer) GetMedicaments() []entities.Medicament {
	if v := dc.medicaments.Load(); v != nil {
		if medicaments, ok := v.([]entities.Medicament); ok {
			return medicaments
		}
	}

	logging.Warn("Medicaments list is empty or invalid")
	return []entities.Medicament{}
}

// GetGeneriques returns the list of generiques
func (dc *DataContainer) GetGeneriques() []entities.GeneriqueList {
	if v := dc.generiques.Load(); v != nil {
		if generiques, ok := v.([]entities.GeneriqueList); ok {
			return generiques
		}
	}

	logging.Warn("GeneriqueList is empty or invalid")
	return []entities.GeneriqueList{}
}

// GetMedicamentsMap returns the medicaments map for O(1) lookups
func (dc *DataContainer) GetMedicamentsMap() map[int]entities.Medicament {
	if v := dc.medicamentsMap.Load(); v != nil {
		if medicamentsMap, ok := v.(map[int]entities.Medicament); ok {
			return medicamentsMap
		}
	}

	logging.Warn("MedicamentsMap is empty or invalid")
	return make(map[int]entities.Medicament)
}

// GetGeneriquesMap returns the generiques map for O(1) lookups
func (dc *DataContainer) GetGeneriquesMap() map[int][]entities.Generique {
	if v := dc.generiquesMap.Load(); v != nil {
		if generiquesMap, ok := v.(map[int][]entities.Generique); ok {
			return generiquesMap
		}
	}

	logging.Warn("GeneriquesMap is empty or invalid")
	return make(map[int][]entities.Generique)
}

// GetPresentationsCIP7Map returns the generiques map for O(1) lookups
func (dc *DataContainer) GetPresentationsCIP7Map() map[int]entities.Presentation {
	if v := dc.presentationsCIP7Map.Load(); v != nil {
		if presentationsCIP7Map, ok := v.(map[int]entities.Presentation); ok {
			return presentationsCIP7Map
		}
	}

	logging.Warn("GeneriquesMap is empty or invalid")
	return make(map[int]entities.Presentation)
}

// GetPresentationsCIP13Map returns the generiques map for O(1) lookups
func (dc *DataContainer) GetPresentationsCIP13Map() map[int]entities.Presentation {
	if v := dc.presentationsCIP13Map.Load(); v != nil {
		if presentationsCIP13Map, ok := v.(map[int]entities.Presentation); ok {
			return presentationsCIP13Map
		}
	}

	logging.Warn("GeneriquesMap is empty or invalid")
	return make(map[int]entities.Presentation)
}

// GetLastUpdated returns the timestamp of the last data update
func (dc *DataContainer) GetLastUpdated() time.Time {
	if v := dc.lastUpdated.Load(); v != nil {
		if lastUpdated, ok := v.(time.Time); ok {
			return lastUpdated
		}
	}

	logging.Warn("Could not get the last updated value")
	return time.Time{}
}

// IsUpdating returns true if a data update is currently in progress
func (dc *DataContainer) IsUpdating() bool {
	return dc.updating.Load()
}

// SetServerStartTime sets the server start time
func (dc *DataContainer) SetServerStartTime(startTime time.Time) {
	dc.serverStartTime.Store(startTime)
}

// GetServerStartTime returns the server start time
func (dc *DataContainer) GetServerStartTime() time.Time {
	if v := dc.serverStartTime.Load(); v != nil {
		if startTime, ok := v.(time.Time); ok {
			return startTime
		}
	}

	logging.Warn("Could not get the server start time value")
	return time.Time{}
}

// UpdateData atomically updates all data in the container
func (dc *DataContainer) UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList,
	medicamentsMap map[int]entities.Medicament, generiquesMap map[int][]entities.Generique,
	presentationsCIP7Map map[int]entities.Presentation, presentationsCIP13Map map[int]entities.Presentation) {

	// Atomic swap (zero downtime replacement)
	dc.medicaments.Store(medicaments)
	dc.medicamentsMap.Store(medicamentsMap)
	dc.generiques.Store(generiques)
	dc.generiquesMap.Store(generiquesMap)
	dc.presentationsCIP7Map.Store(presentationsCIP7Map)
	dc.presentationsCIP13Map.Store(presentationsCIP13Map)
	dc.lastUpdated.Store(time.Now())
}

// BeginUpdate marks the start of a data update operation
// Returns true if update can proceed, false if another update is in progress
func (dc *DataContainer) BeginUpdate() bool {
	return dc.updating.CompareAndSwap(false, true)
}

// EndUpdate marks the end of a data update operation
func (dc *DataContainer) EndUpdate() {
	dc.updating.Store(false)
}
