package entities

type GeneriqueList struct {
	GroupID     int                   `json:"groupID"`
	Libelle     string                `json:"libelle"`
	Medicaments []GeneriqueMedicament `json:"medicaments"`
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
