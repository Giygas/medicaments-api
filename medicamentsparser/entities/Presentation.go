package entities

type Presentation struct {
	Cis                  int    `json:"cis"`
	Cip7                 int    `json:"cip7"`
	Libelle              string `json:"libelle"`
	StatusAdministratif  string `json:"statusAdministratif"`
	EtatComercialisation string `json:"etatComercialisation"`
	DateDeclaration      string `json:"dateDeclaration"`
	Cip13                int    `json:"cip13"`
	Agreement            string `json:"agreement"`
	TauxRemboursement    string `json:"tauxRemboursement"`
	// Prix keeps the legacy 0-when-absent contract until its sunset
	// (2026-12-31, v3.0.0), where it becomes *float64/null like the two
	// fields below. See CHANGELOG 2.2.0.
	Prix                   float64  `json:"prix"`
	PrixPublic             *float64 `json:"prixPublic"`
	HonorairesDispensation *float64 `json:"honorairesDispensation"`
}
