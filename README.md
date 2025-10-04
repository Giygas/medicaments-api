# Medicaments API

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-AGPL%203.0-green.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![API Status](https://img.shields.io/badge/API-Production-00C853.svg)](https://medicamentsapi.giygas.dev/health)

Une API RESTful haute performance qui fournit un accès programmatique aux données publiques
des médicaments français via parsing concurrent de 5 sources BDPM, atomic zero-downtime updates,
cache HTTP intelligent, et rate limiting par token bucket.

## 🎯 Objectif

Les données officielles des médicaments français sont disponibles uniquement
au format TSV avec une structure complexe, ce qui rend leur utilisation
programmatique difficile. Cette API transforme ces données en JSON structuré
et les expose via une interface RESTful optimisée pour la production.

### 📋 Conformité BDPM

Le projet respecte intégralement les termes de la licence de la Base de Données Publique des
Médicaments :

- **Source exclusive** : [base-donnees-publique.medicaments.gouv.fr](https://base-donnees-publique.medicaments.gouv.fr)
- **Intégrité des données** : Aucune altération ou dénaturation du sens des données
- **Synchronisation automatique** : 2 fois par jour (6h et 18h) avec gocron
- **Transparence** : Source systématiquement mentionnée dans la documentation
- **Indépendance** : Projet non affilié à l'ANSM, HAS ou UNCAM

## 🚀 Fonctionnalités

### 📊 Points de terminaison

| Endpoint                          | Description                      | Cache     | Coût tokens | Performance |
| --------------------------------- | -------------------------------- | --------- | ----------- | ----------- |
| `GET /database`                   | Base de données complète         | 6 heures  | 200         | ~20MB       |
| `GET /database/{pageNumber}`      | Pagination (10 médicaments/page) | 6 heures  | 20          | <50ms       |
| `GET /medicament/{element}`       | Recherche par nom (regex)       | 1 heure   | 100         | <100ms      |
| `GET /medicament/id/{cis}`        | Recherche par identifiant CIS    | 12 heures | 100         | <50ms       |
| `GET /generiques/{libelle}`       | Recherche de génériques par nom  | 1 heure   | 20          | <100ms      |
| `GET /generiques/group/{groupId}` | Groupe de génériques par ID      | 12 heures | 20          | <50ms       |
| `GET /health`                     | État de santé avancé (data+system) | -         | 5           | <10ms       |
| `GET /`                           | Page d'accueil documentation     | 1 heure   | 0           | <20ms       |
| `GET /docs`                       | Documentation interactive Swagger | 1 heure   | 0           | <30ms       |
| `GET /docs/openapi.yaml`          | Spécification OpenAPI            | 1 heure   | 0           | <10ms       |
| `GET /favicon.ico`                | Favicon du site                  | 1 an      | 0           | <5ms        |

### 💡 Exemples d'utilisation

#### Recherche de base
```bash
# Base de données complète (~20MB)
curl -H "Accept-Encoding: gzip" https://medicamentsapi.giygas.dev/database

# Pagination (10 médicaments par page)
curl https://medicamentsapi.giygas.dev/database/1

# Recherche par nom (insensible à la casse, regex supporté)
curl https://medicamentsapi.giygas.dev/medicament/paracetamol

# Recherche par CIS (Code Identifiant de Spécialité)
curl https://medicamentsapi.giygas.dev/medicament/id/61504672
```

#### Génériques
```bash
# Génériques par libellé
curl https://medicamentsapi.giygas.dev/generiques/paracetamol

# Groupe générique par ID avec détails complets
curl https://medicamentsapi.giygas.dev/generiques/group/1234
```

#### Monitoring et santé
```bash
# Health check avec métriques système
curl https://medicamentsapi.giygas.dev/health

# Vérification des headers de rate limiting
curl -I https://medicamentsapi.giygas.dev/health
```

#### JavaScript/TypeScript
```javascript
// Recherche avec gestion des erreurs
async function searchMedicament(query) {
  try {
    const response = await fetch(`https://medicamentsapi.giygas.dev/medicament/${encodeURIComponent(query)}`);
    
    if (!response.ok) {
      if (response.status === 429) {
        const retryAfter = response.headers.get('Retry-After');
        throw new Error(`Rate limit exceeded. Retry after ${retryAfter} seconds`);
      }
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    
    const data = await response.json();
    console.log(`Found ${data.count} medicaments`);
    return data.results;
  } catch (error) {
    console.error('Search failed:', error.message);
  }
}

// Pagination
async function getMedicamentsPage(page = 1) {
  const response = await fetch(`https://medicamentsapi.giygas.dev/database/${page}`);
  const data = await response.json();
  console.log(`Page ${data.page} of ${data.maxPage}, ${data.totalItems} total items`);
  return data.data;
}
```

#### Python
```python
import requests
from typing import List, Dict, Any

class MedicamentsAPI:
    BASE_URL = "https://medicamentsapi.giygas.dev"
    
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
print(f"Found {results['count']} results")
```

## 🔒 Sécurité et robustesse

### 🛡️ Mesures de sécurité

- **Validation stricte des entrées** : 3-50 caractères alphanumériques + espaces uniquement
- **Protection contre les injections** : Utilisation de `regexp.QuoteMeta` pour échappement
- **Rate limiting renforcé** : Token bucket (1000 tokens initiaux, 3 tokens/sec recharge) avec protection anti-contournement
- **Coûts variables par endpoint** : 5-200 tokens selon la complexité et ressources consommées
- **Middleware de protection** : Taille des requêtes et headers configurables
- **Nettoyage automatique** : Clients inactifs supprimés toutes les 30 minutes
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

- **Zero-downtime updates** : `atomic.Value` et `atomic.Bool` pour basculement transparent
- **Logging structuré** : Utilisation de `slog` avec rotation de fichiers
- **Monitoring proactif** : Alertes si >25h sans mise à jour
- **Health checks** : Vérifications avec métriques détaillées (data+system), uptime humain, prochaines mises à jour
- **Graceful shutdown** : Timeout 30s + 2s pour finaliser les requêtes
- **Concurrency safe** : `sync.RWMutex` et operations atomiques

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

## ⚡ Performance et optimisations

### 🚀 Optimisations techniques

- **Parsing concurrent** : Téléchargement et traitement parallèles des 5 fichiers TSV
- **Cache HTTP intelligent** : Headers ETag et Last-Modified avec support 304 Not Modified
- **Compression gzip** : Réduction de taille jusqu'à 80% pour les réponses
- **Lookup O(1)** : Maps en mémoire pour recherche instantanée (medicamentsMap, generiquesMap, etc.)
- **Pagination optimisée** : Évite le chargement de la base complète
- **Atomic swap** : Zero-downtime updates sans interruption de service

### 📊 Métriques de performance

| Métrique | Valeur | Description |
|----------|--------|-------------|
| **Temps de réponse** | < 50ms | Recherche individuelle (O(1) via maps) |
| **Recherche complexe** | < 100ms | Par nom avec regex |
| **Pagination** | < 50ms | 10 éléments par page |
| **Mises à jour** | 1-2 min | Parsing concurrent de 5 fichiers TSV |
| **Disponibilité** | 99.9% | Avec redémarrage automatique |
| **Fraîcheur données** | 2x/jour | 6h et 18h automatique |
| **Dataset** | 20K+ médicaments | Données complètes BDPM |
| **RAM Usage** | 30-50MB | 150MB startup, optimisé 30-50MB stable |
| **Compression** | 80% | Réduction taille avec gzip |
| **Cache hit ratio** | > 90% | Avec ETag/Last-Modified |

#### Benchmark de performance

```bash
# Benchmark des temps de réponse (moyenne sur 1000 requêtes)
GET /medicament/id/61504672     → 23ms (cache hit)
GET /medicament/id/61504672     → 45ms (cache miss)
GET /medicament/paracetamol     → 67ms (recherche regex)
GET /database/1                 → 34ms (pagination)
GET /health                     → 8ms  (health check)
```

### 🧠 Architecture mémoire

```text
┌─────────────────────────────────────────────────────────────┐
│                     Memory Layout                           │
├─────────────────────────────────────────────────────────────┤
│ medicamentsMap    │ ~20MB │ O(1) lookup par CIS             │
│ generiquesMap     │ ~6MB  │ O(1) lookup par groupe ID       │
│ compositionsMap   │ ~12MB │ O(1) lookup par CIS             │
│ presentationsMap  │ ~8MB  │ O(1) lookup par CIS             │
│ conditionsMap     │ ~4MB  │ O(1) lookup par CIS             │
│ Total             │ 30-50MB│ RAM usage stable (Go optimisé) │
│ Startup           │ ~150MB│ Pic initial après chargement    │
└─────────────────────────────────────────────────────────────┘
```

## 🏗️ Architecture système

### 🔄 Flux de données

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

### 🧩 Composants détaillés

#### Core Components
- **Downloader** : Téléchargement concurrent des 5 fichiers TSV depuis BDPM avec retry automatique
- **Parser Engine** : Conversion TSV → JSON avec validation et création de lookup maps O(1)
- **Data Container** : Stockage thread-safe avec `atomic.Value` pour zero-downtime updates
- **API Layer** : Chi router v5 avec middleware stack complet

#### Infrastructure Components
- **Scheduler** : Mises à jour automatiques avec gocron (6h/18h) et monitoring
- **Rate Limiter** : Token bucket (juju/ratelimit) avec cleanup automatique
- **Cache System** : HTTP cache avancé avec ETag/Last-Modified
- **Configuration** : Validation d'environnement avec types forts
- **Logging** : Structured logging avec slog et rotation

#### Architecture détaillée

```text
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Layer (Chi)                        │
├─────────────────────────────────────────────────────────────┤
│ RequestID │ RealIP │ Logging │ RateLimit │ Recoverer │ Size │
├─────────────────────────────────────────────────────────────┤
│                    Route Handlers                           │
├─────────────────────────────────────────────────────────────┤
│  /database  │ /medicament  │ /generiques  │ /health       │
├─────────────────────────────────────────────────────────────┤
│                   Business Logic                            │
├─────────────────────────────────────────────────────────────┤
│  Validation │ Cache Check │ Rate Limit │ Response Format  │
├─────────────────────────────────────────────────────────────┤
│                  Data Access Layer                          │
├─────────────────────────────────────────────────────────────┤
│  medicamentsMap │ generiquesMap │ compositionsMap │ etc.   │
├─────────────────────────────────────────────────────────────┤
│                Atomic Data Container                        │
├─────────────────────────────────────────────────────────────┤
│  medicaments │ generiques │ lastUpdated │ updating (bool) │
└─────────────────────────────────────────────────────────────┘
```

## 📚 Documentation

### Accès à la documentation

- **Documentation interactive** : [https://medicamentsapi.giygas.dev/](https://medicamentsapi.giygas.dev/)
- **Swagger UI** : [https://medicamentsapi.giygas.dev/docs](https://medicamentsapi.giygas.dev/docs)
- **OpenAPI spec** : [https://medicamentsapi.giygas.dev/docs/openapi.yaml](https://medicamentsapi.giygas.dev/docs/openapi.yaml)
- **Health check** : [https://medicamentsapi.giygas.dev/health](https://medicamentsapi.giygas.dev/health)

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
  "surveillanceRenforce": "Non",
  "composition": [...],
  "generiques": [...],
  "presentation": [...],
  "conditions": [...]
}
```

#### Entités associées
- **Composition** : Substances actives, dosages, nature des composants
- **Presentation** : Présentations commerciales avec CIP7/CIP13, prix, taux de remboursement
- **Generique** : Groupes génériques avec libellés et types (Princeps/Générique)
- **Condition** : Conditions de prescription et de délivrance

Toutes les entités sont liées par le **CIS** (Code Identifiant de Spécialité) pour garantir la cohérence des données.

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

## 🛠️ Stack Technique

### Core Technologies
- **Langage** : Go 1.21+ avec atomic operations et concurrence native
- **Framework web** : Chi v5 avec middleware stack complet
- **Scheduling** : gocron pour les mises à jour automatiques (6h/18h)
- **Logging** : Structured logging avec slog et rotation de fichiers
- **Rate limiting** : juju/ratelimit (token bucket algorithm)

### Data Processing
- **Encoding** : Support Windows-1252 → UTF-8 pour les fichiers TSV sources
- **Parsing** : Traitement concurrent de 5 fichiers TSV
- **Validation** : Validation stricte des données avec types forts
- **Memory** : Atomic operations pour zero-downtime updates

### Development & Operations
- **Configuration** : Validation d'environnement avec godotenv
- **Tests** : Tests unitaires avec couverture de code et benchmarks
- **Documentation** : OpenAPI 3.0 avec Swagger UI interactive
- **Profiling** : pprof intégré pour le développement (port 6060)
- **Monitoring** : Health checks et métriques intégrées

### Dépendances principales
```go
module github.com/giygas/medicaments-api

require (
    github.com/go-chi/chi/v5 v5.2.3      // Router HTTP
    github.com/go-co-op/gocron v1.32.1   // Scheduler
    github.com/juju/ratelimit v1.0.2     // Rate limiting
    github.com/joho/godotenv v1.5.1      // Configuration
    golang.org/x/text v0.12.0            // Encoding support
)
```

## 🎯 Architecture et design patterns

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

## 🚀 Guide de démarrage rapide

### Installation locale

```bash
# Cloner le repository
git clone https://github.com/giygas/medicaments-api.git
cd medicaments-api

# Installer les dépendances
go mod tidy

# Configurer l'environnement
cp .env.example .env
# Éditer .env avec vos configurations

# Lancer le serveur
go run main.go
```

### Configuration requise
- **Go** : 1.21 ou supérieur
- **Mémoire** : 2GB RAM recommandé
- **Réseau** : Accès internet pour les mises à jour BDPM
- **Stockage** : 1GB d'espace disque

### Variables d'environnement
```bash
# Configuration serveur
PORT=8080
ADDRESS=0.0.0.0
ENV=production

# Logging
LOG_LEVEL=info

# Limites (optionnel)
MAX_REQUEST_BODY=1048576  # 1MB
MAX_HEADER_SIZE=1048576   # 1MB
```

## 🧪 Tests et qualité

### Exécuter les tests
```bash
# Tests unitaires
go test -v ./...

# Tests avec couverture
go test -coverprofile=coverage.out && go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchmem

# Tests de race condition
go test -race ./...
```

### Qualité du code
```bash
# Formatage du code
gofmt -w .

# Analyse statique
go vet ./...

# Linting (si installé)
golangci-lint run
```

## 🏭 Ingénierie de production

### Pratiques industrielles
Le projet intègre des pratiques industrielles modernes :

- **Observability** : Health checks, logging structuré, métriques intégrées
- **Security** : Validation des entrées, protection contre les abus, rate limiting
- **Reliability** : Graceful shutdown, gestion robuste des erreurs, retry automatique
- **Quality** : Code formaté, tests unitaires, documentation complète
- **Performance** : Optimisation mémoire, cache intelligent, compression

### Déploiement recommandé

#### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o medicaments-api .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/medicaments-api .
COPY --from=builder /app/html ./html
EXPOSE 8080
CMD ["./medicaments-api"]
```

#### Docker Compose
```yaml
version: '3.8'
services:
  medicaments-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ENV=production
      - PORT=8080
      - LOG_LEVEL=info
    volumes:
      - ./logs:/app/logs
    restart: unless-stopped
```

## ⚠️ Limitations et conditions d'utilisation

### Limitations techniques
Ce service est gratuit et fonctionne avec des ressources limitées :

- **Rate limiting** : 1000 tokens initiaux par IP, recharge de 3 tokens/seconde
- **Coûts variables** : 5-200 tokens par requête selon l'endpoint
- **Data size** : La base complète fait ~20MB avec 30-50MB RAM usage stable (150MB au démarrage) grâce à l'optimisation Go
- **Pas de SLA** : Service fourni "as-is" sans garantie de disponibilité
- **Dépendance externe** : Mises à jour dépendantes de la disponibilité de la source BDPM
- **Validation stricte** : 3-50 caractères alphanumériques + espaces pour les recherches

### Conditions d'utilisation
- **Usage non-commercial** : L'API est destinée à un usage personnel ou éducatif
- **Respect de la licence** : Les données restent soumises à la licence BDPM
- **Attribution requise** : Mention de la source obligatoire
- **Pas d'altération** : Interdiction de modifier les données originales

## 🤝 Contribuer

### Comment contribuer
1. Fork le repository
2. Créer une branche feature (`git checkout -b feature/amazing-feature`)
3. Commit les changements (`git commit -m 'Add amazing feature'`)
4. Push vers la branche (`git push origin feature/amazing-feature`)
5. Ouvrir une Pull Request

### Guidelines de contribution
- Respecter le style de code existant (gofmt)
- Ajouter des tests pour les nouvelles fonctionnalités
- Mettre à jour la documentation si nécessaire
- S'assurer que tous les tests passent

## 📞 Support et contact

### Obtenir de l'aide
- **Documentation** : [https://medicamentsapi.giygas.dev/docs](https://medicamentsapi.giygas.dev/docs)
- **Issues** : [GitHub Issues](https://github.com/giygas/medicaments-api/issues)
- **Health check** : [https://medicamentsapi.giygas.dev/health](https://medicamentsapi.giygas.dev/health)

### Signaler un problème
Pour signaler un bug ou une anomalie :
1. Vérifier l'état de santé de l'API
2. Consulter la documentation
3. Ouvrir une issue avec les détails suivants :
   - Endpoint concerné
   - Paramètres utilisés
   - Message d'erreur
   - Timestamp de la requête

## 📄 Licence et conformité

### Licence du logiciel
Ce projet est distribué sous **GNU AGPL-3.0**. 
- [Voir la licence complète](https://www.gnu.org/licenses/agpl-3.0.html)
- Obligation de partage des modifications
- Utilisation commerciale soumise à conditions

### Licence des données
Les données médicales restent soumises à la licence de la **Base de Données Publique des Médicaments**.

### Conformité BDPM
- **Source exclusive** : base-donnees-publique.medicaments.gouv.fr
- **Intégrité** : Aucune altération ou dénaturation du sens des données
- **Attribution** : Mention obligatoire de la source dans toute utilisation
- **Réutilisation** : Respect des conditions de réutilisation des données publiques

### Citation
Si vous utilisez cette API dans vos projets, merci de citer :
```
Données issues de la Base de Données Publique des Médicaments (BDPM)
API : https://medicamentsapi.giygas.dev/
Source : https://base-donnees-publique.medicaments.gouv.fr
```

---

## 🙏 Remerciements

### À la communauté médicale française
Ce projet est développé avec ❤️ pour les professionnels de santé, chercheurs, 
et développeurs qui ont besoin d'accéder aux données sur les médicaments 
disponibles en France.

### Sources officielles
- **ANSM** : Agence Nationale de Sécurité du Médicament
- **BDPM** : Base de Données Publique des Médicaments
- **HAS** : Haute Autorité de Santé
- **UNCAM** : Union Nationale des Caisses d'Assurance Maladie

### Contributeurs open source
Merci à tous les contributeurs des projets open source qui rendent 
cette API possible :
- Go et son écosystème
- Chi router
- La communauté des données publiques françaises

---

**⭐ Si ce projet vous est utile, n'hésitez pas à laisser une étoile sur GitHub !**
