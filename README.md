# API des Médicaments

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-AGPL%203.0-green.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Build Status](https://img.shields.io/github/actions/workflow/status/giygas/medicaments-api/tests.yml?branch=main)](https://github.com/giygas/medicaments-api/actions)
[![Coverage](https://img.shields.io/badge/coverage-70%25-brightgreen)](https://github.com/giygas/medicaments-api)
[![API](https://img.shields.io/badge/API-RESTful-orange)](https://medicaments-api.giygas.dev/docs)
[![Performance](https://img.shields.io/badge/performance-80K%2B%20req%2Fs-brightgreen)](https://medicaments-api.giygas.dev/health)
[![Uptime](https://img.shields.io/badge/uptime-99.9%25-brightgreen)](https://medicaments-api.giygas.dev/health)

API RESTful haute performance fournissant un accès programmatique aux données des médicaments français
via une architecture basée sur 6 interfaces principales, parsing concurrent de 5 fichiers TSV BDPM,
mises à jour atomiques zero-downtime, cache HTTP intelligent (ETag/Last-Modified), et rate limiting
par token bucket avec coûts variables par endpoint.

## Performance

L'API délivre des performances exceptionnelles grâce à des optimisations continues. Les lookups O(1) par code CIS ou CIP atteignent **80K+ requêtes/seconde** en production avec une latence moyenne inférieure à 5ms. Les recherches regex (par nom) atteignent **6,100 req/s** grâce aux noms normalisés pré-calculés. L'architecture basée sur 6 interfaces principales avec parsing concurrent des 5 fichiers TSV BDPM garantit des mises à jour atomiques zero-downtime.

## Fonctionnalités

### Points de terminaison (API v1)

**Nouveaux endpoints v1 (recommandés) :**

| Endpoint                          | Description                        | Cache | Coût | Headers    | Validation              |
| --------------------------------- | ---------------------------------- | ----- | ---- | ---------- | ----------------------- |
| `/v1/medicaments/export`          | Export complet de la base          | 6h    | 200  | ETag/LM/RL | -                       |
| `/v1/medicaments?page={n}`        | Pagination (10/page)               | 6h    | 20   | ETag/LM/RL | page ≥ 1                |
| `/v1/medicaments?search={query}`  | Recherche multi-mots (1-6 mots, 3-50 chars)  | 1h    | 50   | ETag/CC/RL | 1-6 mots alphanumériques (séparés par + ou espace)       |
| `/v1/medicaments/{cis}`           | Recherche CIS (O(1) lookup)        | 12h   | 10   | ETag/LM/RL | 1 ≤ CIS ≤ 999,999,999   |
| `/v1/medicaments?cip={code}`      | Recherche CIP via présentation     | 12h   | 10   | ETag/LM/RL | 7 ou 13 chiffres        |
| `/v1/generiques?libelle={nom}`    | Génériques multi-mots (1-6 mots, 3-50 chars) | 1h    | 30   | ETag/CC/RL | 1-6 mots alphanumériques (séparés par + ou espace)       |
| `/v1/generiques?group={id}`       | Groupe générique par ID            | 12h   | 5    | ETag/LM/RL | 1 ≤ ID ≤ 99,999         |
| `/v1/presentations/{cip}`          | Présentations par CIP              | 12h   | 5    | ETag/LM/RL | 1 ≤ CIP ≤ 9,999,999,999 |
| `/v1/diagnostics`                | Diagnostics système (détails)    | 10s   | 30   | CC         | -                       |
| `/`                               | Accueil (SPA)                      | 1h    | 0    | CC         | -                       |
| `/docs`                           | Swagger UI interactive             | 1h    | 0    | CC         | -                       |
| `/health`                         | Santé système simplifiée            | -     | 5    | RL         | -                       |

**Légendes Headers**: ETag/LM (ETag/Last-Modified), CC (Cache-Control), RL (X-RateLimit-\*)

**Endpoints legacy (dépréciés - suppression juillet 2026) :**

Ces endpoints sont toujours disponibles mais seront supprimés le 31 juillet 2026. Veuillez migrer vers les endpoints v1 ci-dessus.

| Endpoint                     | Description                       | Cache | Coût | Headers    | Validation            |
| ---------------------------- | --------------------------------- | ----- | ---- | ---------- | --------------------- |
| `GET /database`              | Base complète (15,000+ médicaments)  | 6h    | 200  | ETag/LM/RL | -                     |
| `GET /database/{page}`       | Pagination (10/page)              | 6h    | 20   | ETag/LM/RL | page ≥ 1              |
| `GET /medicament/{nom}`      | Recherche nom (regex, 3-50 chars) | 1h    | 80   | ETag/CC/RL | `^[a-zA-Z0-9\s\-\.\+'àâäéèêëïîôöùûüÿç]+$`     |
| `GET /medicament/id/{cis}`   | Recherche CIS (O(1) lookup)       | 12h   | 10   | ETag/LM/RL | 1 ≤ CIS ≤ 999,999,999 |
| `GET /generiques/{libelle}`  | Génériques par libellé            | 1h    | 20   | ETag/CC/RL | `^[a-zA-Z0-9\s\-\.\+'àâäéèêëïîôöùûüÿç]+$`     |
| `GET /generiques/group/{id}` | Groupe générique par ID           | 12h   | 20   | ETag/LM/RL | 1 ≤ ID ≤ 99,999       |

### Guide de Migration vers v1

Les endpoints v1 utilisent des paramètres de requête au lieu de paramètres de chemin :

**Table de migration :**

| Endpoint Legacy               | Endpoint v1                              |
| ----------------------------- | ---------------------------------------- |
| `GET /medicament/paracetamol` | `GET /v1/medicaments?search=paracetamol` |
| `GET /medicament/id/61504672` | `GET /v1/medicaments/61504672`           |
| `GET /database/1`             | `GET /v1/medicaments?page=1`             |
| `GET /database`               | `GET /v1/medicaments/export`              |
| `GET /generiques/paracetamol` | `GET /v1/generiques?libelle=paracetamol` |
| `GET /generiques/group/1234`  | `GET /v1/generiques?group=1234`          |

**Règles v1 :**

- **Un seul paramètre** par requête : page, search, cip, libelle, ou group (CIS et export utilisent des paths séparés)
- **Maximum de 6 mots** : Les recherches multi-mots supportent jusqu'à 6 mots (logique ET)
- **Paramètres mutuellement exclusifs** : Les requêtes avec plusieurs paramètres retournent une erreur 400
- **Headers de dépréciation** : Les endpoints legacy renvoient les headers suivants :
  - `Deprecation: true`
  - `Sunset: 2026-07-31T23:59:59Z`
  - `Link: <https://medicaments-api.giygas.dev/v1/...>; rel="successor-version"`
  - `X-Deprecated: Use /v1/... instead`
  - `Warning: 299 - "Deprecated endpoint ..."`

**Exemples de migration :**

```javascript
// JavaScript/TypeScript
const response = await fetch('https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol');
const data = await response.json();

// Python
import requests
response = requests.get('https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol')
data = response.json()
```

### Format des Réponses

#### Patterns de réponse par type d'endpoint

**Recherche de médicaments - Réponse directe en tableau (v1)**

```bash
GET /v1/medicaments?search={query}
Response: [...]  // Tableau direct des objets medicament

GET /v1/medicaments/{cis}
Response: {...}  // Objet médicament unique ou erreur

GET /v1/medicaments?cip={code}
Response: {...}  // Objet médicament unique ou erreur
```

**Génériques - Tableau direct ou objet (v1)**

```bash
GET /v1/generiques?libelle={nom}
Response: [{"groupID": ..., "libelle": ..., "medicaments": [...]}]

GET /v1/generiques?group={id}
Response: {"groupID": ..., "libelle": ..., "medicaments": [...]}
```

**Pagination - Objet avec métadonnées (v1)**

```bash
GET /v1/medicaments?page={n}
Response: {
  "data": [...],
  "page": 1,
  "pageSize": 10,
  "totalItems": 15000,
  "maxPage": 1500
}

GET /v1/medicaments?export=all
Response: [...]  // Export complet en tableau
```

### Exemples d'utilisation

#### Recherche de base (API v1)

```bash
# Base de données complète (~20MB)
curl https://medicaments-api.giygas.dev/v1/medicaments/export

# Pagination (10 médicaments par page)
curl https://medicaments-api.giygas.dev/v1/medicaments?page=1

# Recherche par nom (insensible à la casse, regex supporté)
curl https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol

# Recherche par CIS (Code Identifiant de Spécialité)
curl https://medicaments-api.giygas.dev/v1/medicaments/61504672

# Recherche par CIP
curl https://medicaments-api.giygas.dev/v1/medicaments?cip=3400936403114
```

#### Génériques (API v1)

```bash
# Génériques par libellé
curl https://medicaments-api.giygas.dev/v1/generiques?libelle=paracetamol

# Groupe générique par ID avec détails complets
curl https://medicaments-api.giygas.dev/v1/generiques?group=1234
```

#### Recherche multi-mots

L'API supporte désormais la recherche multi-mots avec logique ET :
- Tous les mots doivent être présents dans le résultat (logique ET)
- Maximum de 6 mots par requête (protection DoS)
- Les mots peuvent être séparés par `+` ou espace dans l'URL

**Exemples :**
```bash
# 2 mots - recherche précise
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol+500"
curl "https://medicaments-api.giygas.dev/v1/generiques?libelle=paracetamol+500"

# 6 mots - recherche très précise
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol+500+mg+comprime+boite+20"
curl "https://medicaments-api.giygas.dev/v1/generiques?libelle=paracetamol+500+mg+comprime+effervescent"

# Erreur - trop de mots (7+ mots)
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol+500+mg+comprime+boite+20+extra"
# Réponse : {"error": "Bad Request", "message": "search query too complex: maximum 6 words allowed", "code": 400}
```

#### Présentations (API v1 - nouveau)

```bash
# Présentations par CIP
curl https://medicaments-api.giygas.dev/v1/presentations/3400936403114
```

### Exemples détaillés

#### GET /v1/medicaments?search=codoliprane

```json
[
  {
    "cis": 60904643,
    "elementPharmaceutique": "CODOLIPRANE 500 mg/30 mg, comprimé",
    "formePharmaceutique": "comprimé",
    "voiesAdministration": ["orale"],
    "statusAutorisation": "Autorisation active",
    "typeProcedure": "Procédure nationale",
    "etatComercialisation": "Commercialisée",
    "dateAMM": "10/05/2013",
    "titulaire": "OPELLA HEALTHCARE FRANCE",
    "surveillanceRenforcee": "Non",
    "composition": [
      {
        "cis": 60904643,
        "elementPharmaceutique": "comprimé",
        "codeSubstance": 2202,
        "denominationSubstance": "PARACÉTAMOL",
        "dosage": "500 mg",
        "referenceDosage": "un comprimé",
        "natureComposant": "SA"
      },
      {
        "cis": 60904643,
        "elementPharmaceutique": "comprimé",
        "codeSubstance": 1240,
        "denominationSubstance": "CAFÉINE",
        "dosage": "30 mg",
        "referenceDosage": "un comprimé",
        "natureComposant": "SA"
      }
    ],
    "generiques": [],
    "presentation": [
      {
        "cis": 60904643,
        "cip7": 3400936403114,
        "cip13": 3400936403114,
        "libelle": "CODOLIPRANE 500 mg/30 mg, comprimé (16)",
        "statusAdministratif": "Présentation active",
        "etatComercialisation": "Commercialisée",
        "dateDeclaration": "19/01/1965",
        "agreement": "non",
        "tauxRemboursement": "65%",
        "prix": 3.85
      }
    ],
    "conditions": []
  }
]
```

#### GET /v1/generiques?libelle=paracetamol

```json
[
  {
    "groupID": 1368,
    "libelle": "PARACETAMOL 400 mg + CAFEINE 50 mg + CODEINE (PHOSPHATE DE) HEMIHYDRATE 20 mg - PRONTALGINE, comprimé",
    "medicaments": [
      {
        "cis": 61644230,
        "elementPharmaceutique": "PRONTALGINE, comprimé",
        "formePharmaceutique": "comprimé",
        "type": "Princeps",
        "composition": [
          {
            "elementPharmaceutique": "comprimé",
            "substance": "CAFÉINE ANHYDRE",
            "dosage": "50,0 mg"
          },
          {
            "elementPharmaceutique": "comprimé",
            "substance": "PARACÉTAMOL",
            "dosage": "400,0 mg"
          },
          {
            "elementPharmaceutique": "comprimé",
            "substance": "PHOSPHATE DE CODÉINE HÉMIHYDRATÉ",
            "dosage": "20,0 mg"
          }
        ]
      },
      {
        "cis": 63399979,
        "elementPharmaceutique": "PARACETAMOL/CAFEINE/CODEINE ARROW 400 mg/50 mg/20 mg, comprimé",
        "formePharmaceutique": "comprimé",
        "type": "Générique",
        "composition": [
          {
            "elementPharmaceutique": "comprimé",
            "substance": "CAFÉINE ANHYDRE",
            "dosage": "50 mg"
          },
          {
            "elementPharmaceutique": "comprimé",
            "substance": "PARACÉTAMOL",
            "dosage": "400 mg"
          },
          {
            "elementPharmaceutique": "comprimé",
            "substance": "PHOSPHATE DE CODÉINE HÉMIHYDRATÉ",
            "dosage": "20 mg"
          }
        ]
      }
    ],
    "orphanCIS": [61586325]
  },
  {
    "groupID": 1643,
    "libelle": "PARACETAMOL 500 mg + CODEINE (PHOSPHATE DE) HEMIHYDRATE 30 mg - DAFALGAN CODEINE, comprimé pelliculé",
    "medicaments": [
      {
        "cis": 66003374,
        "elementPharmaceutique": "DAFALGAN CODEINE, comprimé pelliculé",
        "formePharmaceutique": "comprimé pelliculé",
        "type": "Princeps",
        "composition": [
          {
            "elementPharmaceutique": "comprimé pelliculé",
            "substance": "PARACÉTAMOL",
            "dosage": "500 mg"
          },
          {
            "elementPharmaceutique": "comprimé pelliculé",
            "substance": "CODÉINE (PHOSPHATE DE) HÉMIHYDRATÉ",
            "dosage": "30 mg"
          }
        ]
      },
      {
        "cis": 69458587,
        "elementPharmaceutique": "PARACETAMOL/CODEINE BIOGARAN 500 mg/30 mg, comprimé",
        "formePharmaceutique": "comprimé",
        "type": "Générique",
        "composition": [
          {
            "elementPharmaceutique": "comprimé",
            "substance": "PARACÉTAMOL",
            "dosage": "500,00 mg"
          },
          {
            "elementPharmaceutique": "comprimé",
            "substance": "PHOSPHATE DE CODÉINE HÉMIHYDRATÉ",
            "dosage": "30,00 mg"
          }
        ]
      }
    ],
    "orphanCIS": null
  }
]
```

**À propos du champ `orphanCIS`**

Le champ `orphanCIS` contient les codes CIS référencés dans un groupe générique mais pour lesquels aucune entrée médicament correspondante n'existe dans la base de données.
- Les médicaments avec des données complètes (composition, forme pharmaceutique, type) apparaissent dans le tableau `medicaments`
- Les CIS orphelins apparaissent dans le tableau `orphanCIS` sans détails supplémentaires
- Ce champ peut être :
  - Un tableau d'entiers : `[61586325, 60473805]`
  - Null : `null` (si le groupe ne contient aucun CIS orphelin)

### Programmatique

#### JavaScript/TypeScript

```javascript
// Client JavaScript/TypeScript pour l'API Médicaments v1
class MedicamentsApi {
  private readonly baseUrl = 'https://medicaments-api.giygas.dev';

  async searchByName(query: string): Promise<any[]> {
    const response = await fetch(`${this.baseUrl}/v1/medicaments?search=${query}`);
    const data = await response.json();
    console.log(`Found ${data.length} medicaments`);
    return data; // Tableau des médicaments correspondants
  }

  async getByCis(cis: number): Promise<any> {
    const response = await fetch(`${this.baseUrl}/v1/medicaments/${cis}`);
    return response.json();
  }

  async getByCip(cip: number): Promise<any> {
    const response = await fetch(`${this.baseUrl}/v1/medicaments?cip=${cip}`);
    return response.json();
  }

  async getPage(page: number): Promise<any> {
    const response = await fetch(`${this.baseUrl}/v1/medicaments?page=${page}`);
    return response.json();
  }

  async getDatabase(): Promise<any> {
    const response = await fetch(`${this.baseUrl}/v1/medicaments/export`);
    return response.json();
  }

  // Exemple d'utilisation
  async example() {
    // Recherche par nom
    const paracetamolMeds = await this.searchByName('paracetamol');

    // Recherche par CIS
    const specificMed = await this.getByCis(61504672);

    // Pagination de la base de données
    const firstPage = await this.getPage(1);
    console.log(`Page ${firstPage.page} of ${firstPage.maxPage}`);

    return { paracetamolMeds, specificMed, firstPage };
  }
}

// Usage simple
async function main() {
  const api = new MedicamentsApi();
  const results = await api.example();
  console.log('API Results:', results);
}

main();
```

#### Python

```python
import requests
from typing import List, Dict, Any

class MedicamentsApi:
    BASE_URL = "https://medicaments-api.giygas.dev"

    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            'Accept-Encoding': 'gzip',
            'User-Agent': 'MedicamentsAPI-Python-Client'
        })

    def search_by_name(self, query: str) -> Dict[str, Any]:
        """Rechercher des médicaments par nom"""
        response = self.session.get(f"{self.BASE_URL}/v1/medicaments?search={query}")
        response.raise_for_status()
        return response.json()

    def get_by_cis(self, cis: int) -> Dict[str, Any]:
        """Obtenir un médicament par CIS"""
        response = self.session.get(f"{self.BASE_URL}/v1/medicaments/{cis}")
        response.raise_for_status()
        return response.json()

    def get_by_cip(self, cip: int) -> Dict[str, Any]:
        """Obtenir un médicament par CIP"""
        response = self.session.get(f"{self.BASE_URL}/v1/medicaments?cip={cip}")
        response.raise_for_status()
        return response.json()

    def get_page(self, page: int) -> Dict[str, Any]:
        """Pagination des médicaments"""
        response = self.session.get(f"{self.BASE_URL}/v1/medicaments?page={page}")
        response.raise_for_status()
        return response.json()

    def get_database(self) -> Dict[str, Any]:
        """Exporter toute la base de données"""
        response = self.session.get(f"{self.BASE_URL}/v1/medicaments/export")
        response.raise_for_status()
        return response.json()

    def health_check(self) -> Dict[str, Any]:
        """Vérifier l'état de santé de l'API"""
        response = self.session.get(f"{self.BASE_URL}/health")
        response.raise_for_status()
        return response.json()

    def get_diagnostics(self) -> Dict[str, Any]:
        """Obtenir les diagnostics système détaillés"""
        response = self.session.get(f"{self.BASE_URL}/v1/diagnostics")
        response.raise_for_status()
        return response.json()

# Utilisation
api = MedicamentsApi()
results = api.search_by_name("paracetamol")
print(f"Found {len(results)} results")
```

## Architecture

### Architecture des interfaces

L'architecture repose sur 6 interfaces principales qui séparent clairement les responsabilités pour une maintenabilité optimale.

**Les 6 interfaces principales :**

- **DataStore** : Gère le stockage atomique des données en mémoire avec des opérations thread-safe via `atomic.Value`, garantissant des mises à jour zero-downtime
- **HTTPHandler** : Orchestre les requêtes et route les appels vers les bons handlers sans assertions de type
- **Parser** : Télécharge et traite les 5 fichiers TSV BDPM en parallèle, construisant les maps pour lookups O(1) (CIS → médicament, groupe ID → générique)
- **Scheduler** : Planifie les mises à jour automatiques (6h et 18h) en coordonnant le parsing et le stockage
- **HealthChecker** : Surveille la fraîcheur des données et collecte les métriques système
- **DataValidator** : Assainit les entrées utilisateur et valide l'intégrité des données

Cette approche basée sur interfaces permet de tester chaque composant indépendamment avec des mocks, de remplacer n'importe quelle partie sans impacter le reste, et d'étendre l'API avec de nouveaux endpoints sans modifications profondes.

## Sécurité et robustesse

### Mesures de sécurité

- **Validation stricte** : 3-50 caractères alphanumériques + espaces
  - **Recherche multi-mots** : Logique ET avec limite de 6 mots (protection DoS contre requêtes complexes)
- **Protection injections** : `regexp.QuoteMeta` pour échappement
- **Rate limiting** : Token bucket (1000 tokens, 3/sec recharge, coûts variables 5-200 tokens selon endpoint)
  - Headers dans les réponses : `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Rate`, `Retry-After`
- **Middleware de protection** : Taille des requêtes et headers configurables
- **Nettoyage automatique** : Clients inactifs supprimés régulièrement
- **Headers de transparence** : `X-RateLimit-*` pour monitoring client
- **CORS configuré** : Géré via nginx en production

### Robustesse et résilience

- **Zero-downtime** : `atomic.Value` et `atomic.Bool` pour basculement
- **Logging structuré** : `slog` avec rotation de fichiers
- **Monitoring proactif** : Alertes si >25h sans mise à jour
- **Health checks** : Métriques détaillées (data+system), uptime, mises à jour
- **Graceful shutdown** : Timeout 30s + 2s pour finaliser requêtes
- **Concurrency safe** : `sync.RWMutex` et opérations atomiques

## Optimisations techniques

### Exécuter les benchmarks

Les benchmarks mesurent les performances réelles des endpoints API avec des données réalistes (15,811 médicaments) :

```bash
# Lancer tous les benchmarks handlers
go test -bench=. -benchmem -run=^$ ./handlers

# Lancer tous les benchmarks tests complets
go test ./tests/ -bench=. -benchmem -run=^$

# Benchmark spécifique handler
go test -bench=BenchmarkMedicamentByCIS -benchmem -run=^$ ./handlers
go test -bench=BenchmarkMedicamentsExport -benchmem -run=^$ ./handlers

# Benchmark complet avec sous-tests
go test -bench=BenchmarkAlgorithmicPerformance -benchmem -run=^$ ./tests/
go test -bench=BenchmarkHTTPPerformance -benchmem -run=^$ ./tests/

# Sous-benchmark spécifique (exemple)
go test -bench=BenchmarkAlgorithmicPerformance/CISLookup -benchmem -run=^$ ./tests/

# Avec comptage multiple (plus fiable)
go test -bench=. -benchmem -count=3 -run=^$ ./handlers

# Benchmark avec profil CPU
go test -bench=. -benchmem -cpuprofile=cpu.prof -run=^$ ./handlers
go tool pprof cpu.prof

# Vérification des claims de documentation
go test ./tests/ -run TestDocumentationClaimsVerification -v
```

**Benchmarks v1 disponibles** :

| Benchmark                        | Endpoint v1                  | Type de lookup |
| -------------------------------- | ---------------------------- | -------------- |
| `BenchmarkMedicamentsExport`     | `/v1/medicaments?export=all` | Full export    |
| `BenchmarkMedicamentsPagination` | `/v1/medicaments?page={n}`   | Pagination     |
| `BenchmarkMedicamentsSearch`     | `/v1/medicaments?search={q}` | Regex search   |
| `BenchmarkMedicamentByCIS`       | `/v1/medicaments/{cis}`      | O(1) lookup    |
| `BenchmarkMedicamentByCIP`       | `/v1/medicaments?cip={code}` | O(2) lookups   |
| `BenchmarkGeneriquesSearch`      | `/v1/generiques?libelle={n}` | Regex search   |
| `BenchmarkGeneriqueByGroup`      | `/v1/generiques?group={id}`  | O(1) lookup    |
| `BenchmarkPresentationByCIP`     | `/v1/presentations/{cip}`    | O(1) lookup    |
| `BenchmarkAlgorithmicPerformance` | Test complet algorithmique       | Complet        |
| `BenchmarkHTTPPerformance`        | Test complet HTTP              | Complet        |
| `BenchmarkRealWorldSearch`       | Tests de recherche réels       | Complet        |
| `BenchmarkSustainedPerformance`  | Tests de charge soutenus      | Complet        |

**Notes sur les benchmarks complets** :

- `BenchmarkAlgorithmicPerformance` : Tests complets de performance algorithmique incluant CISLookup, GenericGroupLookup, Pagination, Search, et PresentationsLookup
- `BenchmarkHTTPPerformance` : Tests complets de performance HTTP incluant CISLookup, GenericGroupLookup, GenericSearch, et HealthCheck avec stack complète
- `BenchmarkRealWorldSearch` : Tests de recherche réels incluant CommonMedication, BrandName, ShortQuery, et autres scénarios
- `BenchmarkSustainedPerformance` : Tests de charge soutenus incluant ConcurrentLoad, MixedEndpoints, et MemoryUnderLoad

### Tests Spécialisés

| Test                                   | Description                           | Commande                                                          |
| -------------------------------------- | ------------------------------------- | ----------------------------------------------------------------- |
| TestDocumentationClaimsVerification    | Vérification des claims documentation | `go test ./tests/ -run TestDocumentationClaimsVerification -v`    |
| TestIntegrationFullDataParsingPipeline | Pipeline complet d'intégration        | `go test ./tests/ -run TestIntegrationFullDataParsingPipeline -v` |

**Interprétation des résultats** :

- `Reqs/sec` : Nombre de requêtes par seconde
- `Latence` : Temps moyen par opération
- `Mémoire/op` : Mémoire allouée par opération
- `Allocs/op` : Nombre d'allocations mémoire par opération
**Note** : Les benchmarks v1 mesurent le temps de sérialisation uniquement (sans réseau). L'export complet prend ~1.26ms pour sérialiser 15,811 médicaments, mais le transfert réseau prend plusieurs secondes pour ~20MB de données.

### Benchmarks algorithmiques (Handler performance)

| Endpoint                         | Reqs/sec | Latence      | Allocs/op |
| -------------------------------- | -------- | ------------ | --------- |
| `/v1/medicaments/{cis}`         | 400,000  | 3.0µs        | 38        |
| `/v1/generiques?group={id}`      | 200,000  | 5.0µs        | 37        |
| `/v1/generiques?libelle={nom}`   | 18,000   | 60µs         | 94        |
| `/v1/presentations/{cip}`        | 430,000  | 2.0µs        | 63        |
| `/v1/medicaments?cip={code}`     | 375,000  | 5.0µs        | 54        |
| `/v1/medicaments?page={n}`       | 40,000   | 30µs         | 38        |
| `/v1/medicaments?search={query}` | 1,600    | 600µs        | 94        |

**Note** : Benchmarks algorithmiques mesurent la logique pure du handler sans surcharge réseau (go test -bench).

### Performance en production

Les résultats ci-dessous incluent l'overhead HTTP complet (middleware, logging, sérialisation, réseau) :

| Endpoint                         | HTTP Req/sec | Latence (avg) |
| -------------------------------- | ------------ | -------------- |
| `/v1/medicaments/{cis}`         | 78,000       | ~4ms          |
| `/v1/presentations/{cip}`        | 77,000       | ~4ms          |
| `/v1/medicaments?cip={code}`    | 75,000       | ~5ms          |
| `/v1/generiques?libelle={nom}`   | 36,000       | ~9ms          |
| `/v1/medicaments?page={n}`       | 41,000       | ~7ms          |
| `/v1/medicaments?search={query}` | 6,100        | ~50ms         |

**Testing conditions**: 300 concurrent workers, 3-second duration, HTTP/1.1 with persistent connections, full middleware stack

**Note** : Tests de performance utilisent une tolérance de 25% pour tenir compte des variations d'environnement.

### Optimisations récentes

Les améliorations v1.1 ont été apportées à l'API pour augmenter considérablement le débit HTTP tout en maintenant une stabilité exceptionnelle.

**Noms normalisés pré-calculés**

Élimine les opérations de chaîne répétées (ToLower(), ReplaceAll()) pendant les recherches en calculant les versions normalisées une seule fois lors du parsing des données BDPM. La normalisation se produit une fois par médicament au lieu d'être exécutée à chaque requête de recherche.

Avantages :
- Réduction drastique des allocations mémoire par recherche (16,000 → 94)
- Lookups par chaîne directement au lieu de calculer à la volée
- Amélioration de la latence de recherche par un facteur important

**Logging environment-aware**

Réduit l'overhead I/O console en production en n'activant pas le logging debug/info en environnement de production et de test. Seuls les messages WARN et ERROR sont loggés dans ces environnements.

Avantages :
- Réduction de ~40% de l'overhead de logging en production
- Maintient les logs complets en développement
- Meilleure visibilité des problèmes réels (WARN/ERROR)

**Résultats combinés**

Ces deux optimisations travaillent ensemble pour améliorer le débit HTTP de 2-3x sur la plupart des endpoints :

- **Lookups O(1) (CIS, CIP)** : Amélioration significative du débit
- **Recherches regex** : Performance accrue grâce aux chaînes pré-normalisées
- **Stabilité maintenue** : Aucune régression sur les endpoints existants

### Architecture mémoire

```text
┌─────────────────────────────────────────────────────────────┐
│                     Memory Layout                           │
├─────────────────────────────────────────────────────────────┤
│ medicaments       │ ~20MB │ Slice des médicaments           │
│ generiques        │ ~6MB  │ Slice des generiques            │
│ medicamentsMap    │ ~15MB │ O(1) lookup par CIS             │
│ generiquesMap     │ ~4MB  │ O(1) lookup par groupe ID       │
 │ Total             │ 60-90MB│ RAM usage stable (Go optimisé)  │
│ Startup           │ ~150MB│ Pic initial après chargement     │
└─────────────────────────────────────────────────────────────┘
```

## Logging et Monitoring

### Rotation Automatique des Logs

L'API implémente un système de logging structuré avec rotation automatique :

#### Fonctionnalités

- **Rotation Hebdomadaire** : Nouveau fichier chaque semaine (format ISO : `app-YYYY-Www.log`)
- **Rotation par Taille** : Rotation forcée si fichier dépasse `MAX_LOG_FILE_SIZE`
- **Nettoyage Automatique** : Suppression des fichiers plus anciens que `LOG_RETENTION_WEEKS`
- **Double Sortie** : Console (texte) + Fichier (JSON) pour faciliter le parsing
- **Arrêt Propre** : Fermeture gracieuse des fichiers avec context cancellation

#### Configuration

```bash
# Configuration des logs
LOG_RETENTION_WEEKS=4        # Nombre de semaines de conservation (1-52)
MAX_LOG_FILE_SIZE=104857600  # Taille max avant rotation (1MB-1GB, défaut: 100MB)
LOG_LEVEL=info               # Niveau de log console (debug/info/warn/error) - les fichiers capturent toujours tous les niveaux
```

#### Structure des Fichiers

```
logs/
├── app-2025-W41.log              # Semaine en cours
├── app-2025-W40.log              # Semaine précédente
├── app-2025-W39.log              # 2 semaines précédentes
└── app-2025-W38_size_20251007_143022.log  # Rotation par taille
```

#### Format des Logs

```json
{
  "time": "2025-10-07T16:45:55.190+02:00",
  "level": "INFO",
  "msg": "Files downloaded and parsed successfully"
}
```

### Monitoring Intégré

### Endpoint de santé

L'endpoint `/health` fournit une réponse simplifiée pour vérifier l'état de l'API :

```json
{
  "status": "healthy",
  "data": {
    "last_update": "2026-01-15T06:00:00Z",
    "medicaments": 15420,
    "generiques": 5200,
    "is_updating": false
  }
}
```

#### États de santé

- **`healthy`** : API opérationnelle avec données fraîches (< 24h)
- **`degraded`** : API opérationnelle mais données âgées (> 24h)
- **`unhealthy`** : API non opérationnelle (pas de médicaments en mémoire) - renvoie HTTP 503

#### Métriques Clés

- **`status`** : État de santé (healthy/degraded/unhealthy)
- **`last_update`** : Dernière mise à jour réussie des données (ISO 8601)
- **`medicaments`** : Nombre de médicaments en mémoire
- **`generiques`** : Nombre de groupes génériques
- **`is_updating`** : Indique si une mise à jour est en cours

### Endpoint de diagnostics v1

L'endpoint `/v1/diagnostics` fournit des métriques détaillées pour le monitoring avancé :

```json
{
  "timestamp": "2026-01-15T14:30:45Z",
  "uptime_seconds": 86400,
  "next_update": "2026-01-15T18:00:00Z",
  "data_age_hours": 2.5,
  "system": {
    "goroutines": 45,
    "memory": {
      "alloc_mb": 130,
      "sys_mb": 150,
      "num_gc": 25
    }
  },
  "data_integrity": {
    "medicaments_without_conditions": {
      "count": 1250,
      "sample_cis": [60012345, 60023456]
    },
    "medicaments_without_generiques": {
      "count": 8450,
      "sample_cis": [61504672, 61001234]
    },
    "medicaments_without_presentations": {
      "count": 3200,
      "sample_cis": [62003456, 62004567]
    },
    "medicaments_without_compositions": {
      "count": 890,
      "sample_cis": [63005678, 63006789]
    },
    "generique_only_cis": {
      "count": 45,
      "sample_cis": [64007890, 64008901]
    }
  }
}
```

#### Métriques Clés - Diagnostics

- **`timestamp`** : Horodatage de la réponse (ISO 8601)
- **`uptime_seconds`** : Temps d'exécution de l'application en secondes
- **`next_update`** : Prochaine mise à jour planifiée (ISO 8601)
- **`data_age_hours`** : Âge des données en heures
- **`goroutines`** : Nombre de goroutines actives
- **`memory`** : Statistiques mémoire détaillées (alloc_mb, sys_mb, num_gc)
- **`data_integrity`** : Rapport de qualité des données avec catégories :
  - `medicaments_without_conditions` : Médicaments sans conditions de prescription
  - `medicaments_without_generiques` : Médicaments sans association générique
  - `medicaments_without_presentations` : Médicaments sans présentations
  - `medicaments_without_compositions` : Médicaments sans composition
  - `generique_only_cis` : CIS présents uniquement dans les génériques

#### Notes sur l'intégrité des données

- Les comptages représentent le nombre d'entrées affectées par chaque catégorie
- `sample_cis` contient des exemples de CIS pour chaque catégorie (limité à 2 exemples)
- Ces informations aident à identifier les incohérences potentielles dans les données BDPM

## Architecture système

### Flux de données

```text
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  BDPM TSV Files │───▶│ Concurrent       │───▶│ Parallel        │
│  (5 sources)    │    │ Downloader       │    │ Parsing (5x)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                         │
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   API Response  │◀───│   HTTP Cache     │◀───│   Atomic Store  │
│   (JSON/GZIP)   │    │   (ETag/LM)      │    │   (memory)      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Middleware Stack Complet

L'API utilise une stack de middleware Chi v5 optimisée pour la sécurité et la performance :

**Architecture des middleware** :

1. **RequestID** - Traçabilité unique par requête
2. **BlockDirectAccess** - Bloque les accès directs non autorisés
3. **RealIP** - Détection IP réelle derrière les proxies
4. **Logging structuré** - Logs avec slog pour monitoring
5. **RedirectSlashes** - Normalisation des URLs
6. **Recoverer** - Gestion des paniques avec recovery
7. **RequestSize** - Limites taille corps/headers (configurable)
8. **RateLimiting** - Token bucket avec coûts variables par endpoint

### Cache HTTP Intelligent

L'API implémente un système de cache HTTP efficace : les ressources statiques (documentation, OpenAPI, favicon) ont des headers `Cache-Control` avec des durées adaptées (1 heure pour la documentation, 1 an pour le favicon), tandis que les réponses API utilisent `Last-Modified` et `ETag` pour gérer le cache conditionnel (réponses 304 Not Modified sur requêtes répétées). La compression gzip est appliquée automatiquement, réduisant la taille des réponses JSON jusqu'à 80%.

## Documentation

### Accès à la documentation

- **Swagger UI** : [https://medicaments-api.giygas.dev/docs](https://medicaments-api.giygas.dev/docs)
- **OpenAPI spec** : [https://medicaments-api.giygas.dev/docs/openapi.yaml](https://medicaments-api.giygas.dev/docs/openapi.yaml)

### Modèle de données

L'API expose les données BDPM complètes avec les entités suivantes :

#### Entité principale : Medicament

```json
{
  "cis": 61504672,
  "elementPharmaceutique": "PARACETAMOL MYLAN 1 g, comprimé",
  "formePharmaceutique": "comprimé",
  "voiesAdministration": ["orale"],
  "statusAutorisation": "Autorisation active",
  "typeProcedure": "Procédure nationale",
  "etatComercialisation": "Commercialisée",
  "dateAMM": "2000-01-01",
  "titulaire": "MYLAN SAS",
  "surveillanceRenforcee": "Non",
  "composition": [...],
  "generiques": [...],
  "presentation": [...],
  "conditions": [...]
}
```

#### Entités associées

- **Composition** : Substances actives, dosages, nature des composants
- **Presentation** : Présentations avec CIP7/CIP13, prix, remboursement
- **Generique** : Groupes génériques avec libellés et types
- **Condition** : Conditions de prescription et délivrance

Toutes les entités sont liées par le **CIS** (Code Identifiant de Spécialité)
pour garantir la cohérence des données.

### Schéma de relations

```text
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Medicament    │───▶│  Composition    │───▶│   Substance     │
│     (CIS)       │    │   (CIS)         │    │   (Code)        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
          │                       │                       │
          ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Presentation   │    │   Generique     │    │   Condition     │
│   (CIS/CIP)     │    │   (CIS/Group)   │    │    (CIS)        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Stack Technique

### Core Technologies

- **Encoding** : Support Windows-1252 → UTF-8 pour les fichiers TSV sources
- **Framework web** : Chi v5 avec middleware stack complet
- **Scheduling** : gocron pour les mises à jour automatiques (6h/18h)
- **Logging** : Structured logging avec slog et rotation de fichiers
- **Rate limiting** : juju/ratelimit (token bucket algorithm)

### Data Processing

- **Encoding** : Support Windows-1252/UTF-8/ISO8859-1 → UTF-8 pour les fichiers TSV sources
- **Parsing** : Traitement concurrent de 5 fichiers TSV
- **Validation** : Validation stricte des données avec types forts
- **Memory** : Atomic operations pour zero-downtime updates

### Development & Operations

- **Configuration** : Validation d'environnement avec godotenv
- **Tests** : Tests unitaires avec couverture de code et benchmarks
- **Documentation** : OpenAPI 3.1 avec Swagger UI interactive
- **Profiling** : pprof intégré pour le développement (port 6060)
- **Monitoring** : Health checks et métriques intégrées

### Dépendances principales

```go
module github.com/giygas/medicaments-api

require (
    github.com/go-chi/chi/v5 v5.2.3      // Routeur HTTP
    github.com/go-co-op/gocron v1.32.1   // Planificateur
    github.com/juju/ratelimit v1.0.2     // Limitation de taux
    github.com/joho/godotenv v1.5.1      // Configuration
    golang.org/x/text v0.12.0            // Support d'encodage
    go.uber.org/atomic v1.11.0           // Opérations atomiques
)
```

## Architecture et design patterns

### Principes de conception

L'architecture privilégie la simplicité, l'efficacité et la résilience :

- **Atomic operations** : Mises à jour sans temps d'arrêt
- **Stateless architecture** : Facilite la montée en charge horizontale
- **Modular design** : Séparation claire des responsabilités
- **Memory optimization** : Cache intelligent pour des réponses rapides

### Design patterns appliqués

- **Singleton** : DataContainer pour gestion centralisée
- **Observer** : Health monitoring et logging
- **Strategy** : Rate limiting avec token bucket
- **Factory** : Parser creation et validation
- **Circuit breaker** : Gestion des erreurs de téléchargement

## Configuration développement local

### Prérequis

- **Go 1.21+** avec support des modules
- **2GB RAM** recommandé pour le développement
- **Connexion internet** pour les mises à jour BDPM

### Démarrage rapide

```bash
# Cloner et configurer
git clone https://github.com/giygas/medicaments-api.git
cd medicaments-api

# Installer les dépendances
go mod tidy

# Configurer l'environnement
cp .env.example .env
# Éditer .env avec vos paramètres

# Lancer le serveur de développement
go run main.go
```

### Commandes de développement

```bash
# Build pour la plateforme actuelle
go build -o medicaments-api .

# Builds multi-plateformes
GOOS=linux GOARCH=amd64 go build -o medicaments-api-linux .
GOOS=windows GOARCH=amd64 go build -o medicaments-api.exe .

# Lancer les tests
go test -v ./...

# Lancer avec couverture
go test -coverprofile=coverage.out -v
go tool cover -html=coverage.out -o coverage.html

# Lancer les benchmarks
go test -bench=. -benchmem

# Tests de race condition
go test -race -v

# Formatage du code
gofmt -w .
```

#### Analyse statique

```bash
# Analyse statique du code Go - détecte les problèmes potentiels
go vet ./...

# Formatage du code (standardisation)
gofmt -w .

# Vérification plus approfondie (si installé)
golangci-lint run
```

**Outils disponibles :**

- **go vet** : Vérifie les constructions suspectes, détecte le code inaccessible et les erreurs logiques, identifie les mauvaises utilisations des fonctions built-in, vérifie la conformité des interfaces, analyse les formats d'impression et les arguments
- **gofmt** : Formatage automatique du code Go pour standardisation
- **golangci-lint** : Linter plus approfondie (optionnel, à installer séparément)

### Configuration d'environnement

```bash
# Configuration serveur
PORT=8000                    # Port du serveur
ADDRESS=127.0.0.1            # Adresse d'écoute
ENV=dev                      # Environnement (dev/production)

# Logging
LOG_LEVEL=info               # Niveau de log console (les fichiers sont toujours en DEBUG)

# Limites optionnelles
MAX_REQUEST_BODY=1048576     # 1MB max corps de requête
MAX_HEADER_SIZE=1048576      # 1MB max taille headers
```

### Fonctionnalités du serveur de développement

- **Serveur local**: `http://localhost:8000`
- **Profiling pprof**: `http://localhost:6060` (quand ENV=dev)
- **Rechargement auto**: Utiliser `air` ou similaire pour hot reloading
- **Documentation interactive**: `http://localhost:8000/docs`
- **Health endpoint**: `http://localhost:8000/health`

## Limitations et conditions d'utilisation

### Limitations techniques

Ce service est gratuit et fonctionne avec des ressources limitées :

- **Rate limiting** : 1000 tokens/IP, recharge 3 tokens/seconde
- **Coûts variables** : 5-200 tokens/requête selon endpoint
- **Data size** : ~20MB avec 60-90MB RAM stable (150MB startup)
- **Pas de SLA** : Service "as-is" sans garantie de disponibilité
- **Dépendance externe** : Mises à jour selon disponibilité source BDPM
- **Validation stricte** : 3-50 caractères alphanumériques + espaces

### Conditions d'utilisation

- **Usage non-commercial** : L'API est destinée à un usage personnel ou éducatif
- **Respect de la licence** : Les données restent soumises à la licence BDPM
- **Attribution requise** : Mention de la source obligatoire
- **Pas d'altération** : Interdiction de modifier les données originales

## Support et contact

### Obtenir de l'aide

- **Documentation** : [https://medicaments-api.giygas.dev/docs](https://medicaments-api.giygas.dev/docs)
- **Issues** : [GitHub Issues](https://github.com/giygas/medicaments-api/issues)
- **Health check** : [https://medicaments-api.giygas.dev/health](https://medicaments-api.giygas.dev/health)

## Licence et conformité

### Licence du logiciel

Ce projet est distribué sous **GNU AGPL-3.0**.

- [Voir la licence complète](https://www.gnu.org/licenses/agpl-3.0.html)
- Obligation de partage des modifications
- Utilisation commerciale soumise à conditions

### Licence des données

Les données médicales restent soumises à la licence de la
**Base de Données Publique des Médicaments**.

### Conformité BDPM

- **Source exclusive** : base-donnees-publique.medicaments.gouv.fr
- **Intégrité** : Aucune altération ou dénaturation du sens des données
- **Attribution** : Mention obligatoire de la source dans toute utilisation
- **Réutilisation** : Respect des conditions de réutilisation des données publiques

### Citation

Si vous utilisez cette API dans vos projets, merci de citer :

```text
Données issues de la Base de Données Publique des Médicaments (BDPM)
API : https://medicaments-api.giygas.dev/
Source : https://base-donnees-publique.medicaments.gouv.fr
```

---

## Changelog

Décembre 2025
- Fix encodage des caractères: Changement de charset de Windows1252 vers détection automatique UTF-8/ISO8859-1 dans le downloader
- Corrige les problèmes d'encodage pour les médicaments avec caractères spéciaux
- Fix logging shutdown: Correction des logs pendant l'arrêt du serveur

Février 2026

**Nouvelles fonctionnalités**
- Endpoints v1 avec headers de dépréciation, caching ETag et coûts de token mis à jour
- Routes RESTful : `/v1/medicaments/{cis}`, `/v1/presentations/{cip}`, `/v1/medicaments/export`
- Maps de lookup CIP7/CIP13 avec détection de doublons
- Endpoint `/v1/diagnostics` : rapports d'intégrité des données et métriques système
- Champ orphanCIS dans les réponses génériques : codes CIS sans entrée médicament correspondante
- Endpoint `/health` simplifié : statut de santé et données de base uniquement
- Support du signe + dans les recherches : "paracetamol+cafeine" fonctionne comme "paracetamol cafeine"
- LOG_LEVEL fonctionnel : contrôle le niveau de log console/fichiers (fallback par environnement)

**Performance**
- Amélioration de 6-10x des recherches : noms normalisés pré-calculés réduisant allocations de 170x
- Validation des entrées optimisée 5-10x : pré-compilation regex, remplacement par string.Contains()
- Logging optimisé : skip /health et /metrics, réduction volume logs

**Corrections**
- Correction du logging lors de l'arrêt du serveur
- Endpoint /v1/medicaments : retourne 404 si non trouvé (au lieu de tableau vide)
- Race conditions corrigées dans le logger rotatif (fuites ressources + concurrence)
- Validation génériques stricte : groupID 1-9999 avec messages d'erreur clairs

**Refactoring et tests**
- Modernisation syntaxe, division des fichiers de tests et simplification des calculs
- Mise à jour de la suite de tests pour utiliser les endpoints v1
- Migration Go 1.24, ajout smoke tests, consolidation benchmarks

**Qualité des données**
- Rapports sur les médicaments sans conditions, génériques, présentations, ou composition
- Gestion des cas limites TSV : statistiques de skip pour lignes mal formatées

---

## Remerciements

### À la communauté médicale française

Ce projet est développé avec passion pour les professionnels de santé, chercheurs,
et développeurs qui ont besoin d'accéder aux données sur les médicaments
disponibles en France.

### Sources officielles

- **BDPM** : Base de Données Publique des Médicaments

### Contributeurs open source

Merci à tous les contributeurs des projets open source qui rendent
cette API possible :

- Go et son écosystème
- Chi router

---

**⭐ Si ce projet vous est utile, n'hésitez pas à laisser une étoile sur GitHub !**
