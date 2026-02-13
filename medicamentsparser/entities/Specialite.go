package entities

type Specialite struct {
	Cis                   int      `json:"cis"`
	Denomination          string   `json:"elementPharmaceutique"`
	FormePharmaceutique   string   `json:"formePharmaceutique"`
	VoiesAdministration   []string `json:"voiesAdministration"`
	StatusAutorisation    string   `json:"statusAutorisation"`
	TypeProcedure         string   `json:"typeProcedure"`
	EtatComercialisation  string   `json:"etatComercialisation"`
	DateAMM               string   `json:"dateAMM"`
	Titulaire             string   `json:"titulaire"`
	SurveillanceRenforcee string   `json:"surveillanceRenforcee"`
}
