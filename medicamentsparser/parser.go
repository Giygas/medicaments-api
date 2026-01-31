package medicamentsparser

import (
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// Compile-time check to ensure MedicamentsParser implements Parser interface
var _ interfaces.Parser = (*MedicamentsParser)(nil)

// MedicamentsParser implements the Parser interface
type MedicamentsParser struct{}

// NewMedicamentsParser creates a new MedicamentsParser instance
func NewMedicamentsParser() *MedicamentsParser {
	return &MedicamentsParser{}
}

// ParseAllMedicaments implements the Parser interface
func (p *MedicamentsParser) ParseAllMedicaments() ([]entities.Medicament, map[int]entities.Presentation, map[int]entities.Presentation, error) {
	return ParseAllMedicaments()
}

// GeneriquesParser implements the Parser interface
func (p *MedicamentsParser) GeneriquesParser(medicaments *[]entities.Medicament, medicamentsMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.GeneriqueList, error) {
	return GeneriquesParser(medicaments, medicamentsMap)
}
