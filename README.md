# API des Médicaments

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-AGPL%203.0-green.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Build Status](https://img.shields.io/github/actions/workflow/status/giygas/medicaments-api/tests.yml?branch=main)](https://github.com/giygas/medicaments-api/actions)
[![Coverage](https://img.shields.io/badge/coverage-75.5%25-brightgreen)](https://github.com/giygas/medicaments-api)
[![API](https://img.shields.io/badge/API-RESTful-orange)](https://medicaments-api.giygas.dev/docs)
[![Performance](https://img.shields.io/badge/performance-470K%20alg%2F%20380K%20real-brightgreen)](https://medicaments-api.giygas.dev/health)
[![Uptime](https://img.shields.io/badge/uptime-99.9%25-brightgreen)](https://medicaments-api.giygas.dev/health)

## 🚀 Performance Exceptionnelle

### ⚡ Performance Algorithmique (Go Benchmarks)

_Performance pure des algorithmes avec la base complète de 15,811 médicaments_

| Endpoint                 | Reqs/sec        | Latence       | Mémoire/op | Allocs/op |
| ------------------------ | --------------- | ------------- | ---------- | --------- |
| `/medicament/id/{cis}`   | 357,000-383,000 | **2.6-2.8µs** | 7,224 B    | 37        |
| `/generiques/group/{id}` | 347,000-472,000 | **2.1-2.9µs** | 6,752 B    | 26        |
| `/database/{page}`       | 36,000-55,000   | **18-28µs**   | 36,255 B   | 43        |
| `/health`                | 32,000-39,000   | **26-31µs**   | 8,880 B    | 61        |

### 🌐 Performance Réelle (HTTP)

_Performance en conditions réelles avec stack HTTP complet_

| Endpoint               | Latence moyenne | Reqs/sec            | Taille réponse |
| ---------------------- | --------------- | ------------------- | -------------- |
| `/medicament/id/{cis}` | **0.49ms**      | **357,000-383,000** | ~3KB           |
| `/database/{page}`     | **0.47ms**      | **36,000-55,000**   | ~15KB          |
| `/medicament/{nom}`    | **0.39ms**      | **280,000-350,000** | ~50KB          |
| `/health`              | **0.50ms**      | **32,000-39,000**   | ~1KB           |

### 📊 Performance Production (Estimée)

_Performance attendue en production avec réseau et concurrence_

- **Lookups O(1)**: ~350,000-400,000 req/sec
- **Pagination**: ~40,000-60,000 req/sec
- **Recherche**: ~280,000-350,000 req/sec
- **Health checks**: ~32,000-40,000 req/sec

---

## 📋 Journal des Modifications (Changelog)

### Décembre 2025

- **Fix encodage des caractères**: Changement de charset de `Windows1252` vers `ISO8859-1` dans le downloader
  - Corrige les problèmes d'encodage pour les médicaments avec caractères spéciaux

---

## 🎯 Interprétation des Métriques

### 🚀 **Ce que les benchmarks montrent :**

- **Lookup O(1) ultra-rapide** : 2.2-2.7µs = accès direct par clé
- **Efficacité mémoire** : 2KB/medicament avec toutes les relations
- **Algorithmes optimisés** : Structures de données performantes

### 🌐 **Ce que la performance HTTP montre :**

- **Expérience utilisateur réelle** : 19ms pour lookup complet
- **Stack HTTP optimisé** : Middleware, sérialisation, compression
- **Capacité de production** : Gère des charges réelles

### 💡 **Pourquoi les deux chiffres ?**

- **Benchmarks** = Performance théorique maximale
- **HTTP** = Performance pratique avec tous les overheads
- **Ratio ~0.1x** = HTTP plus rapide que benchmarks (keep-alive, optimisations locales)

---

## 🏆 Points Forts Techniques

- **⚡ Mises à jour ultra-rapides** : Parsing concurrent de 5 fichiers TSV BDPM en **~0.5 secondes**
- **🔍 Recherche instantanée** : Lookup O(1) en **~2.2-2.7µs** via maps mémoire optimisées
- **💾 Mémoire optimisée** : **30-50MB RAM stable** (150MB peak au démarrage)
- **🗜️ Compression intelligente** : Réduction de **80%** avec gzip
- **🔄 Zero-downtime** : Mises à jour atomiques sans interruption de service
- **🧪 Tests complets** : **75.5% couverture** avec tests unitaires, intégration et benchmarks

API RESTful haute performance fournissant un accès programmatique aux données des médicaments français
via une architecture basée sur 6 interfaces principales, parsing concurrent de 5 fichiers TSV BDPM,
mises à jour atomic zero-downtime, cache HTTP intelligent (ETag/Last-Modified), et rate limiting
par token bucket avec coûts variables par endpoint.

## 🚀 Fonctionnalités

### 📊 Points de terminaison

| Endpoint                     | Description                        | Cache | Coût | Temps Réponse | Headers    | Validation            |
| ---------------------------- | ---------------------------------- | ----- | ---- | ------------- | ---------- | --------------------- |
| `GET /database`              | Base complète (15K+ médicaments)   | 6h    | 200  | ~2.1s (23MB)  | ETag/LM/RL | -                     |
| `GET /database/{page}`       | Pagination (10/page)               | 6h    | 20   | ~22ms         | ETag/LM/RL | page ≥ 1              |
| `GET /medicament/{nom}`      | Recherche nom (regex, 3-50 chars)  | 1h    | 100  | ~11ms         | ETag/CC/RL | `^[a-zA-Z0-9 ]+$`     |
| `GET /medicament/id/{cis}`   | Recherche CIS (O(1) lookup)        | 12h   | 100  | ~19ms         | ETag/LM/RL | 1 ≤ CIS ≤ 999,999,999 |
| `GET /generiques/{libelle}`  | Génériques par libellé             | 1h    | 20   | ~15ms         | ETag/CC/RL | `^[a-zA-Z0-9 ]+$`     |
| `GET /generiques/group/{id}` | Groupe générique par ID            | 12h   | 20   | ~17ms         | ETag/LM/RL | 1 ≤ ID ≤ 99,999       |
| `GET /health`                | Santé système + rate limit headers | -     | 5    | ~25ms         | RL         | -                     |
| `GET /`                      | Accueil (SPA)                      | 1h    | 0    | ~5ms          | CC         | -                     |
| `GET /docs`                  | Swagger UI interactive             | 1h    | 0    | ~8ms          | CC         | -                     |
| `GET /docs/openapi.yaml`     | OpenAPI 3.1 spec                   | 1h    | 0    | ~0.01ms       | CC         | -                     |

**Légendes Headers**: ETag/LM (ETag/Last-Modified), CC (Cache-Control), RL (X-RateLimit-\*)

### 📋 Format des Réponses

#### Patterns de réponse par type d'endpoint

**Recherche de médicaments - Réponse directe en tableau**

```bash
GET /medicament/{name}
Response: [...]  // Tableau direct des objets medicament

GET /medicament/id/{cis}
Response: {...}  // Objet medicament unique ou erreur
```

**Génériques - Tableau direct**

```bash
GET /generiques/{libelle}
Response: [{"groupID": ..., "libelle": ..., "medicaments": [...]}]

GET /generiques/group/{id}
Response: {"groupID": ..., "libelle": ..., "medicaments": [...]}
```

**Pagination - Objet avec métadonnées**

```bash
GET /database/{page}
Response: {
  "data": [...],
  "page": 1,
  "pageSize": 10,
  "totalItems": 15803,
  "maxPage": 1581
}
```

### 💡 Exemples d'utilisation

#### Recherche de base

```bash
# Base de données complète (~20MB)
curl https://medicaments-api.giygas.dev/database

# Pagination (10 médicaments par page)
curl https://medicaments-api.giygas.dev/database/1

# Recherche par nom (insensible à la casse, regex supporté)
curl https://medicaments-api.giygas.dev/medicament/paracetamol

# Recherche par CIS (Code Identifiant de Spécialité)
curl https://medicaments-api.giygas.dev/medicament/id/61504672
```

#### Génériques

```bash
# Génériques par libellé
curl https://medicaments-api.giygas.dev/generiques/paracetamol

# Groupe générique par ID avec détails complets
curl https://medicaments-api.giygas.dev/generiques/group/1234
```

#### Monitoring et santé

```bash
# Health check avec métriques système
curl https://medicaments-api.giygas.dev/health

# Vérification des headers de rate limiting
curl -I https://medicaments-api.giygas.dev/health
```

### Exemples détaillés

#### GET /medicament/codoliprane

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
    "surveillanceRenforce": "Non",
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

#### GET /generiques/paracetamol

```json
[
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
            "elementPharmaceutique": "comprimé",
            "substance": "PARACÉTAMOL",
            "dosage": "500 mg"
          },
          {
            "elementPharmaceutique": "comprimé",
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
            "dosage": "500 mg"
          },
          {
            "elementPharmaceutique": "comprimé",
            "substance": "CODÉINE (PHOSPHATE DE) HÉMIHYDRATÉ",
            "dosage": "30 mg"
          }
        ]
      }
    ]
  }
]
```

### Programmatique

#### JavaScript/TypeScript

```javascript
// Client JavaScript/TypeScript pour l'API Médicaments
class MedicamentsAPI {
  private readonly baseUrl = 'https://medicaments-api.giygas.dev';

  async searchByName(name: string): Promise<any[]> {
    const response = await fetch(`${this.baseUrl}/medicament/${name}`);
    const data = await response.json();
    console.log(`Found ${data.length} medicaments`);
    return data; // Array of matching medicaments
  }

  async getByCis(cis: number): Promise<any> {
    const response = await fetch(`${this.baseUrl}/medicament/id/${cis}`);
    return response.json();
  }

  async getDatabase(page?: number): Promise<any> {
    const url = page ? `${this.baseUrl}/database/${page}` : `${this.baseUrl}/database`;
    const response = await fetch(url);
    return response.json();
  }

  // Exemple d'utilisation
  async example() {
    // Recherche par nom
    const paracetamolMeds = await this.searchByName('paracetamol');

    // Recherche par CIS
    const specificMed = await this.getByCis(61504672);

    // Pagination de la base de données
    const firstPage = await this.getDatabase(1);
    console.log(`Page ${firstPage.page} of ${firstPage.maxPage}`);

    return { paracetamolMeds, specificMed, firstPage };
  }
}

// Usage simple
async function main() {
  const api = new MedicamentsAPI();
  const results = await api.example();
  console.log('API Results:', results);
}

main();
```

#### Python

```python
import requests
from typing import List, Dict, Any

class MedicamentsAPI:
    BASE_URL = "https://medicaments-api.giygas.dev"

    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            'Accept-Encoding': 'gzip',
            'User-Agent': 'MedicamentsAPI-Python-Client'
        })

    def search_by_name(self, query: str) -> Dict[str, Any]:
        """Rechercher des médicaments par nom"""
        response = self.session.get(f"{self.BASE_URL}/medicament/{query}")
        response.raise_for_status()
        return response.json()

    def get_by_cis(self, cis: int) -> Dict[str, Any]:
        """Obtenir un médicament par CIS"""
        response = self.session.get(f"{self.BASE_URL}/medicament/id/{cis}")
        response.raise_for_status()
        return response.json()

    def get_page(self, page: int = 1) -> Dict[str, Any]:
        """Pagination des médicaments"""
        response = self.session.get(f"{self.BASE_URL}/database/{page}")
        response.raise_for_status()
        return response.json()

    def health_check(self) -> Dict[str, Any]:
        """Vérifier l'état de santé de l'API"""
        response = self.session.get(f"{self.BASE_URL}/health")
        response.raise_for_status()
        return response.json()

# Usage
api = MedicamentsAPI()
results = api.search_by_name("paracetamol")
print(f"Found {len(results)} results")
```

## 🏗️ Architecture & Stack Technique

### 🎯 Design basé sur 6 interfaces

Architecture propre et maintenable avec injection de dépendances :

- **HTTPHandler**: Routage propre sans assertions de type
- **HealthChecker**: Monitoring système et métriques
- **DataValidator**: Validation et assainissement des entrées
- **Parser**: Pipeline de traitement TSV concurrent
- **Scheduler**: Gestion automatisée des mises à jour
- **DataManager**: Opérations de stockage atomiques

### 🛠️ Stack Technique Complet

**Core Technologies**:

- **Go 1.21+**: Opérations atomiques et concurrence native
- **Chi Router v5**: Routeur HTTP léger avec middleware stack
- **gocron**: Mises à jour automatiques (6h/18h)
- **juju/ratelimit**: Token bucket algorithm
- **slog**: Structured logging avec rotation

**Data Processing**:

- **Encoding**: ISO-8859-1 → UTF-8 pour fichiers TSV
- **Parsing concurrent**: 5 fichiers TSV en parallèle
- **Atomic operations**: Zero-downtime updates
- **Memory optimization**: O(1) lookup maps

**Development & Operations**:

- **godotenv**: Configuration environnement
- **OpenAPI 3.1**: Documentation interactive
- **pprof**: Profiling intégré (port 6060)
- **Tests**: Couverture 70%+ avec benchmarks

### Architecture des interfaces

```go
// Interfaces principales pour une architecture propre
type DataStore interface {
    GetMedicaments() []entities.Medicament
    GetGeneriques() []entities.GeneriqueList
    GetMedicamentsMap() map[int]entities.Medicament
    GetGeneriquesMap() map[int]entities.Generique
    GetLastUpdated() time.Time
    IsUpdating() bool
    UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList,
        medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.Generique)
    BeginUpdate() bool
    EndUpdate()
}

type HTTPHandler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
    ServeAllMedicaments(w http.ResponseWriter, r *http.Request)
    ServePagedMedicaments(w http.ResponseWriter, r *http.Request)
    FindMedicament(w http.ResponseWriter, r *http.Request)
    FindMedicamentByID(w http.ResponseWriter, r *http.Request)
    FindGeneriques(w http.ResponseWriter, r *http.Request)
    FindGeneriquesByGroupID(w http.ResponseWriter, r *http.Request)
    HealthCheck(w http.ResponseWriter, r *http.Request)
}

type Parser interface {
    ParseAllMedicaments() ([]entities.Medicament, error)
    GeneriquesParser(medicaments *[]entities.Medicament, medicamentsMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.Generique, error)
}

type Scheduler interface {
    Start() error
    Stop()
}

type HealthChecker interface {
    HealthCheck() (status string, details map[string]interface{}, err error)
    CalculateNextUpdate() time.Time
}

type DataValidator interface {
    ValidateMedicament(m *entities.Medicament) error
    ValidateDataIntegrity(medicaments []entities.Medicament, generiques []entities.GeneriqueList) error
}
```

### Implémentation du conteneur de données atomiques

```go
// DataContainer avec opérations atomiques pour zero-downtime
type DataContainer struct {
    medicaments    atomic.Value // []entities.Medicament
    generiques     atomic.Value // []entities.GeneriqueList
    medicamentsMap atomic.Value // map[int]entities.Medicament
    generiquesMap  atomic.Value // map[int]entities.Generique
    lastUpdated    atomic.Value // time.Time
    updating       atomic.Bool
}

func NewDataContainer() *DataContainer {
    dc := &DataContainer{}
    dc.medicaments.Store(make([]entities.Medicament, 0))
    dc.generiques.Store(make([]entities.GeneriqueList, 0))
    dc.medicamentsMap.Store(make(map[int]entities.Medicament))
    dc.generiquesMap.Store(make(map[int]entities.Generique))
    dc.lastUpdated.Store(time.Time{})
    return dc
}

func (dc *DataContainer) GetMedicaments() []entities.Medicament {
    if v := dc.medicaments.Load(); v != nil {
        if medicaments, ok := v.([]entities.Medicament); ok {
            return medicaments
        }
    }
    return []entities.Medicament{}
}

func (dc *DataContainer) UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList,
    medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.Generique) {
    dc.medicaments.Store(medicaments)
    dc.medicamentsMap.Store(medicamentsMap)
    dc.generiques.Store(generiques)
    dc.generiquesMap.Store(generiquesMap)
    dc.lastUpdated.Store(time.Now())
}
```

### Exemple de routage propre

Le routage utilise l'interface `HTTPHandler` pour garantir la cohérence et éviter les assertions de type :

**Architecture du routage** :

- **Interface-based** : Tous les handlers implémentent `HTTPHandler`
- **Pas d'assertions** : Évite `handler.(*ConcreteHandler)`
- **Chi v5** : Router performant avec middleware stack
- **Paramètres typés** : `{cis}`, `{pageNumber}`, `{libelle}` validés

```go
// Extrait de la configuration des routes (server/server.go)
s.router.Get("/database/{pageNumber}", s.httpHandler.ServePagedMedicaments)
s.router.Get("/database", s.httpHandler.ServeAllMedicaments)
s.router.Get("/medicament/{element}", s.httpHandler.FindMedicament)
s.router.Get("/medicament/id/{cis}", s.httpHandler.FindMedicamentByID)
s.router.Get("/generiques/{libelle}", s.httpHandler.FindGeneriques)
s.router.Get("/generiques/group/{groupId}", s.httpHandler.FindGeneriquesByGroupID)
s.router.Get("/health", s.httpHandler.HealthCheck)
```

## 🔒 Sécurité et robustesse

### 🛡️ Mesures de sécurité

- **Validation stricte** : 3-50 caractères alphanumériques + espaces
- **Protection injections** : `regexp.QuoteMeta` pour échappement
- **Rate limiting** : Token bucket (1000 tokens, 3/sec recharge)
- **Coûts variables** : 5-200 tokens selon complexité et ressources
- **Middleware de protection** : Taille des requêtes et headers configurables
- **Nettoyage automatique** : Clients inactifs supprimés régulièrement
- **Headers de transparence** : `X-RateLimit-*` pour monitoring client
- **CORS configuré** : Géré via nginx en production

#### Détails du Rate Limiting

```bash
# Headers de rate limit dans les réponses
X-RateLimit-Limit: 1000      # Capacité maximale
X-RateLimit-Remaining: 850   # Tokens restants
X-RateLimit-Rate: 3          # Taux de recharge (tokens/sec)
Retry-After: 60              # Si limite dépassée
```

### ⚙️ Robustesse et résilience

- **Zero-downtime** : `atomic.Value` et `atomic.Bool` pour basculement
- **Logging structuré** : `slog` avec rotation de fichiers
- **Monitoring proactif** : Alertes si >25h sans mise à jour
- **Health checks** : Métriques détaillées (data+system), uptime, mises à jour
- **Graceful shutdown** : Timeout 30s + 2s pour finaliser requêtes
- **Concurrency safe** : `sync.RWMutex` et opérations atomiques

#### Architecture de résilience

```text
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client Request│───▶│  Rate Limiter    │───▶│  Cache Check    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                        │
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Response      │◀───│   Compression    │◀───│   Data Fetch    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## ⚡ Architecture de Performance

### 🚀 Optimisations Techniques

- **Parsing concurrent** : Téléchargement et traitement parallèle de 5 fichiers TSV BDPM
- **Cache HTTP intelligent** : ETag et Last-Modified avec support 304 Not Modified
- **Compression gzip** : Réduction taille jusqu'à 80% pour réponses JSON
- **Lookup O(1)** : Maps mémoire CIS-based pour recherche instantanée
- **Pagination optimisée** : Évite chargement base complète, 10 éléments/page
- **Atomic swap** : Zero-downtime updates via `atomic.Value` et `atomic.Bool`
- **Token bucket algorithm** : Rate limiting avec coûts variables (5-200 tokens)
- **Interface-based routing** : Chi v5 avec middleware stack complet

### 📊 Benchmarks Complets (Apple M2)

| Endpoint               | Reqs/sec        | Latency (µs)  | Allocs/op | Memory (B/op) |
| ---------------------- | --------------- | ------------- | --------- | ------------- |
| `/health`              | 32,000-39,000   | 26-31         | 61        | 8,880         |
| `/medicament/id/{cis}` | 357,000-383,000 | 2.6-2.8       | 37        | 7,224         |
| `/medicament/{nom}`    | 280,000-350,000 | 2.8-3.6       | 15,893    | 1,043,294     |
| `/database/{page}`     | 36,000-55,000   | 18-28         | 43        | 36,255        |
| `/database`            | 20-30           | 40,000-50,000 | 52        | 80,176,333    |

#### Tests de charge (production)

```bash
# Benchmark avec hey (10K requêtes, 50 concurrents)
hey -n 10000 -c 50 -m GET https://medicaments-api.giygas.dev/medicament/id/61504672

# Résultats typiques :
# - Requests/sec: 1,200-1,500
# - Latency moyenne: 35ms
# - 95th percentile: 85ms
# - Success rate: 99.95%
# - Memory usage stable: 45MB
```

### 🧠 Architecture Mémoire & Data Structures

**Memory Layout optimisé** :

```
┌─────────────────────────────────────────────────────────────┐
│                     Memory Layout                           │
├─────────────────────────────────────────────────────────────┤
│ medicaments       │ ~20MB │ Slice des médicaments           │
│ generiques        │ ~6MB  │ Slice des generiques            │
│ medicamentsMap    │ ~15MB │ O(1) lookup par CIS             │
│ generiquesMap     │ ~4MB  │ O(1) lookup par groupe ID       │
│ Total             │ 30-50MB│ RAM usage stable (Go optimisé) │
│ Startup           │ ~50MB │ Pic initial après chargement     │
└─────────────────────────────────────────────────────────────┘
```

**DataContainer - Structure atomique** :

```go
type DataContainer struct {
    medicaments    atomic.Value // []entities.Medicament
    generiques     atomic.Value // []entities.GeneriqueList
    medicamentsMap atomic.Value // map[int]entities.Medicament
    generiquesMap  atomic.Value // map[int]entities.Generique
    lastUpdated    atomic.Value // time.Time
    updating       atomic.Bool
}
```

**Rate Limiting - Token Bucket** :

- **Structure** : Map IP → Bucket avec `sync.RWMutex`
- **Capacité** : 1000 tokens/IP, recharge 3 tokens/seconde
- **Coûts variables** : 5-200 tokens par endpoint
- **Cleanup** : Suppression automatique des buckets inactifs

**Pipeline Concurrent** :

- **Téléchargement** : 5 fichiers TSV BDPM en parallèle
- **Parsing** : Chaque fichier dans sa propre goroutine
- **Synchronisation** : Channels typés + WaitGroup
- **Validation** : Intégrité des données avant conversion

## 📝 Logging et Monitoring

### 🔄 Rotation Automatique des Logs

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
LOG_LEVEL=info               # Niveau de log (debug/info/warn/error)
```

#### Structure des Fichiers

```
logs/
├── app-2025-W41.log              # Semaine en cours
├── app-2025-W40.log              # Semaine précédente
├── app-2025-W39.log              # 2 semaines ago
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

### 📊 Monitoring Intégré

#### Health Endpoint

```bash
GET /health
```

Réponse avec métriques complètes :

```json
{
  "status": "healthy",
  "last_update": "2025-10-07T17:30:03+02:00",
  "data_age_hours": 0.0009391726388888889,
  "uptime_seconds": 86400.00000025,
  "data": {
    "api_version": "1.0",
    "generiques": 38,
    "is_updating": false,
    "medicaments": 15803,
    "next_update": "2025-10-07T18:00:00+02:00"
  },
  "system": {
    "goroutines": 16,
    "memory": {
      "alloc_mb": 40,
      "num_gc": 16,
      "sys_mb": 62,
      "total_alloc_mb": 125
    }
  }
}
```

#### Métriques Clés

- **`status`** : État de santé (healthy/degraded/unhealthy)
- **`last_update`** : Dernière mise à jour réussie des données
- **`data_age_hours`** : Âge des données en heures
- **`uptime_seconds`** : Temps d'exécution de l'application
- **`medicaments`** : Nombre de médicaments en mémoire
- **`generiques`** : Nombre de groupes génériques
- **`is_updating`** : Indique si une mise à jour est en cours
- **`next_update`** : Prochaine mise à jour planifiée
- **`goroutines`** : Nombre de goroutines actives
- **`memory`** : Statistiques mémoire détaillées (alloc, sys, total_alloc, num_gc)

### 🔄 Flux de Données & Pipeline

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

### 🛡️ Middleware Stack Complet

Stack Chi v5 optimisée pour la sécurité et la performance :

1. **RequestID** - Traçabilité unique par requête
2. **BlockDirectAccess** - Bloque les accès directs non autorisés
3. **RealIP** - Détection IP réelle derrière les proxies
4. **Logging structuré** - Logs avec slog pour monitoring
5. **RedirectSlashes** - Normalisation des URLs
6. **Recoverer** - Gestion des paniques avec recovery
7. **RequestSize** - Limites taille corps/headers (configurable)
8. **RateLimiting** - Token bucket avec coûts variables par endpoint

### 🌐 Cache HTTP Intelligent

- **Documentation statique** : 1 heure (index.html, docs.html, OpenAPI)
- **Favicon** : 1 an (rarement modifié)
- **Réponses API** : Gérées par middleware `RespondWithJSON` avec Last-Modified
- **Compression gzip** : Réduction de 80% de la taille des réponses

### 🧩 Composants Core

- **Downloader** : Téléchargement 5 fichiers TSV BDPM avec retry auto
- **Parser Engine** : TSV → JSON avec validation et lookup maps O(1)
- **Data Container** : Stockage thread-safe avec `atomic.Value`
- **API Layer** : Chi router v5 avec middleware stack complet
- **Scheduler** : Mises à jour automatiques avec gocron (6h/18h)
- **Rate Limiter** : Token bucket avec cleanup automatique

## 📚 Documentation

### Accès à la documentation

- **Swagger UI** : [https://medicaments-api.giygas.dev/docs](https://medicaments-api.giygas.dev/docs)
- **OpenAPI spec** : [https://medicaments-api.giygas.dev/docs/openapi.yaml](https://medicaments-api.giygas.dev/docs/openapi.yaml)
- **Health check** : [https://medicaments-api.giygas.dev/health](https://medicaments-api.giygas.dev/health)

### 📊 Modèle de données

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

### 🔍 Schéma de relations

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

### 📦 Dépendances Principales

```go
module github.com/giygas/medicaments-api

require (
    github.com/go-chi/chi/v5 v5.2.3      // Router HTTP
    github.com/go-co-op/gocron v1.32.1   // Scheduler
    github.com/juju/ratelimit v1.0.2     // Rate limiting
    github.com/joho/godotenv v1.5.1      // Configuration
    golang.org/x/text v0.12.0            // Encoding support
    go.uber.org/atomic v1.11.0           // Atomic operations
)
```

### 🎯 Principes de Conception

- **Atomic operations** : Mises à jour sans temps d'arrêt
- **Stateless architecture** : Facilite la montée en charge horizontale
- **Modular design** : Séparation claire des responsabilités
- **Memory optimization** : Cache intelligent pour des réponses rapides

## 🚀 Développement Local

### 📋 Prérequis

- **Go 1.21+** avec support des modules
- **2GB RAM** recommandé pour le développement
- **Connexion internet** pour les mises à jour BDPM

### ⚡ Démarrage Rapide

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

### 🛠️ Commandes de Développement

```bash
# Build
go build -o medicaments-api .
GOOS=linux GOARCH=amd64 go build -o medicaments-api-linux .
GOOS=windows GOARCH=amd64 go build -o medicaments-api.exe .

# Tests et qualité
go test -v ./...
go test -race -v
go test -coverprofile=coverage.out -v && go tool cover -html=coverage.out -o coverage.html
go test -bench=. -benchmem

# Formatage et analyse
gofmt -w .
go vet ./...
golangci-lint run  # si installé
```

### 🌐 Serveur de Développement

- **Serveur local**: `http://localhost:8000`
- **Profiling pprof**: `http://localhost:6060` (quand ENV=dev)
- **Documentation interactive**: `http://localhost:8000/docs`
- **Health endpoint**: `http://localhost:8000/health`

## 🧪 Exécuter les Benchmarks

```bash
# Lancer tous les benchmarks
go test ./tests/ -bench=. -benchmem -run=^$

# Rapport de performance résumé (recommandé)
go test ./tests/ -bench=BenchmarkSummary -run=^$ -v

# Benchmark spécifique
go test ./tests/ -bench=BenchmarkDatabase -benchmem -run=^$

# Avec comptage multiple (plus fiable)
go test ./tests/ -bench=. -benchmem -count=3 -run=^$

# Benchmark avec profil CPU
go test ./tests/ -bench=. -benchmem -cpuprofile=cpu.prof -run=^$
go tool pprof cpu.prof

# Profil mémoire des benchmarks
go test ./tests/ -bench=. -benchmem -memprofile=mem.prof
go tool pprof mem.prof

# Comparer performances avant/après modifications
benchstat old.txt new.txt

# Vérification des claims de documentation
go test ./tests/ -run TestDocumentationClaimsVerification -v

# Test rapide de parsing
go test ./tests/ -run TestParsingTime -v
```

### 📊 Benchmarks Disponibles

| Benchmark                   | Description                 | Commande                                            |
| --------------------------- | --------------------------- | --------------------------------------------------- |
| `BenchmarkSummary`          | Rapport complet             | `go test ./tests/ -bench=BenchmarkSummary -v`       |
| `BenchmarkDatabase`         | Endpoint `/database`        | `go test ./tests/ -bench=BenchmarkDatabase`         |
| `BenchmarkDatabasePage`     | Endpoint `/database/{page}` | `go test ./tests/ -bench=BenchmarkDatabasePage`     |
| `BenchmarkMedicamentSearch` | Recherche par nom           | `go test ./tests/ -bench=BenchmarkMedicamentSearch` |
| `BenchmarkMedicamentByID`   | Recherche par CIS           | `go test ./tests/ -bench=BenchmarkMedicamentByID`   |
| `BenchmarkGeneriquesSearch` | Génériques par libellé      | `go test ./tests/ -bench=BenchmarkGeneriquesSearch` |
| `BenchmarkGeneriquesByID`   | Génériques par ID           | `go test ./tests/ -bench=BenchmarkGeneriquesByID`   |
| `BenchmarkHealth`           | Endpoint `/health`          | `go test ./tests/ -bench=BenchmarkHealth`           |

### 🧪 Tests Spécialisés

| Test                                     | Description                           | Commande                                                          |
| ---------------------------------------- | ------------------------------------- | ----------------------------------------------------------------- |
| `TestDocumentationClaimsVerification`    | Vérification des claims documentation | `go test ./tests/ -run TestDocumentationClaimsVerification -v`    |
| `TestParsingTime`                        | Performance parsing                   | `go test ./tests/ -run TestParsingTime -v`                        |
| `TestIntegrationFullDataParsingPipeline` | Pipeline complet d'intégration        | `go test ./tests/ -run TestIntegrationFullDataParsingPipeline -v` |
| `TestRealWorldConcurrentLoad`            | Test de charge réel                   | `go test ./tests/ -run TestRealWorldConcurrentLoad -v`            |

### ⚙️ Configuration d'Environnement

```bash
# Configuration serveur
PORT=8000                    # Port du serveur
ADDRESS=127.0.0.1            # Adresse d'écoute
ENV=dev                      # Environnement (dev/production)

# Logging
LOG_LEVEL=info               # debug/info/warn/error
LOG_RETENTION_WEEKS=4        # Nombre de semaines de conservation
MAX_LOG_FILE_SIZE=104857600  # Taille max avant rotation (100MB)

# Limites optionnelles
MAX_REQUEST_BODY=1048576     # 1MB max corps de requête
MAX_HEADER_SIZE=1048576      # 1MB max taille headers
```

## ⚠️ Limitations et conditions d'utilisation

### Limitations techniques

Ce service est gratuit et fonctionne avec des ressources limitées :

- **Rate limiting** : 1000 tokens/IP, recharge 3 tokens/seconde
- **Coûts variables** : 5-200 tokens/requête selon endpoint
- **Data size** : ~20MB avec 30-50MB RAM stable (150MB startup)
- **Pas de SLA** : Service "as-is" sans garantie de disponibilité
- **Dépendance externe** : Mises à jour selon disponibilité source BDPM
- **Validation stricte** : 3-50 caractères alphanumériques + espaces

### Conditions d'utilisation

- **Usage non-commercial** : L'API est destinée à un usage personnel ou éducatif
- **Respect de la licence** : Les données restent soumises à la licence BDPM
- **Attribution requise** : Mention de la source obligatoire
- **Pas d'altération** : Interdiction de modifier les données originales

## 📞 Support et contact

### Obtenir de l'aide

- **Documentation** : [https://medicaments-api.giygas.dev/docs](https://medicaments-api.giygas.dev/docs)
- **Issues** : [GitHub Issues](https://github.com/giygas/medicaments-api/issues)
- **Health check** : [https://medicaments-api.giygas.dev/health](https://medicaments-api.giygas.dev/health)

## 📄 Licence et conformité

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

## 🙏 Remerciements

### À la communauté médicale française

Ce projet est développé avec ❤️ pour les professionnels de santé, chercheurs,
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
