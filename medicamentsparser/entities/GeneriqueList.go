package entities

// GeneriqueList represents a group of generique medicaments with the same libelle.
// It includes medicaments with full data and a list of orphan CIS without medicament entries.
type GeneriqueList struct {
	GroupID           int                   `json:"groupID"`
	Libelle           string                `json:"libelle"`
	LibelleNormalized string                `json:"-"` // Pre-computed: ToLower() + ReplaceAll("+", " ")
	Medicaments       []GeneriqueMedicament `json:"medicaments"`
	OrphanCIS         []int                 `json:"orphanCIS"`
}

type GeneriqueMedicament struct {
	Cis                 int                    `json:"cis"`
	Denomination        string                 `json:"elementPharmaceutique"`
	FormePharmaceutique string                 `json:"formePharmaceutique"`
	Type                string                 `json:"type"`
	Composition         []GeneriqueComposition `json:"composition"`
}

type GeneriqueComposition struct {
	ElementPharmaceutique string `json:"elementPharmaceutique"`
	DenominationSubstance string `json:"substance"`
	Dosage                string `json:"dosage"`
}
