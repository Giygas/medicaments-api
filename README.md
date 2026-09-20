# API des Médicaments

[![Go Version](https://img.shields.io/badge/Go-1.26-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-AGPL%203.0-green.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Build Status](https://img.shields.io/github/actions/workflow/status/giygas/medicaments-api/tests.yml?branch=main)](https://github.com/giygas/medicaments-api/actions)
[![Coverage](https://img.shields.io/badge/coverage-78.5%25-brightgreen)](https://github.com/giygas/medicaments-api)
[![API](https://img.shields.io/badge/API-RESTful-orange)](https://medicaments-api.giygas.dev/docs)
[![Performance](https://img.shields.io/badge/performance-80K%2B%20req%2Fs-brightgreen)](https://medicaments-api.giygas.dev/health)
[![Uptime](https://img.shields.io/badge/uptime-99.9%25-brightgreen)](https://medicaments-api.giygas.dev/health)
[![Changelog](https://img.shields.io/badge/Changelog-v2.1.0-blue)](CHANGELOG.md)

API RESTful haute performance fournissant un accès programmatique aux données des médicaments français via une architecture basée sur 6 interfaces principales, parsing concurrent de 5 fichiers TSV BDPM, mises à jour atomiques zero-downtime, cache HTTP intelligent (ETag/Last-Modified), rate limiting par token bucket, et support Docker complet avec stack observabilité.

## Performance

L'API délivre des performances exceptionnelles : lookups O(1) par code CIS ou CIP atteignent **80K+ requêtes/seconde** en production avec latence moyenne < 5ms. Recherches regex atteignent **6,100 req/s** grâce aux noms normalisés pré-calculés.

## Fonctionnalités

### Points de terminaison (API v1)

**Nouveaux endpoints v1 (recommandés) :**

| Endpoint            | Description                    | Documentation                      |
| ------------------- | ------------------------------ | ---------------------------------- |
| `/v1/medicaments`   | Recherche & browse médicaments | [Full API](html/docs/openapi.yaml) |
| `/v1/generiques`    | Groupes génériques             | [Full API](html/docs/openapi.yaml) |
| `/v1/presentations` | Présentations par CIP          | [Full API](html/docs/openapi.yaml) |
| `/v1/diagnostics`   | Métriques système détaillées   | [Full API](html/docs/openapi.yaml) |
| `/health`           | Santé système simplifiée       | [Full API](html/docs/openapi.yaml) |
| `/`                 | Documentation SPA              | [Full API](html/docs/openapi.yaml) |
| `/docs`             | Swagger UI interactive         | [Full API](html/docs/openapi.yaml) |

**Endpoints legacy (supprimés le 31 juillet 2026) :**

Ces endpoints ont été supprimés le 31 juillet 2026. Ils répondent désormais `410 Gone`, accompagnés d'un header `Link; rel="successor-version"` pointant vers l'endpoint v1 de remplacement.

| Endpoint                     | Description        | Migration                            |
| ---------------------------- | ------------------ | ------------------------------------ |
| `GET /database`              | Base complète      | → `/v1/medicaments/export`           |
| `GET /database/{page}`       | Pagination         | → `/v1/medicaments?page={n}`         |
| `GET /medicament/{nom}`      | Recherche nom      | → `/v1/medicaments?search={nom}`     |
| `GET /medicament/id/{cis}`   | Recherche CIS      | → `/v1/medicaments/{cis}`            |
| `GET /medicament/cip/{cip}`  | Recherche CIP      | → `/v1/medicaments?cip={cip}`        |
| `GET /generiques/{libelle}`  | Génériques libellé | → `/v1/generiques?libelle={libelle}` |
| `GET /generiques/group/{id}` | Groupe générique   | → `/v1/generiques/{id}`              |

Voir le [Guide de Migration](docs/MIGRATION.md) pour les détails complets.

### Caractéristiques clés

- **15,811+ médicaments** avec 1,618-1,628 groupes génériques
- **RESTful v1 API** avec 9 endpoints optimisés
- **80K+ req/sec** pour les lookups O(1)
- **Mises à jour automatiques** : 2x/jour (6h et 18h)
- **Zero-downtime updates** via atomic operations
- **Rate limiting intelligent** : Token bucket avec coûts variables (5-200 tokens)
- **Cache HTTP** : ETag/Last-Modified pour optimisation
- **Recherche multi-mots** : Logique ET avec limite 6 mots

## Exemples Rapides

### Recherche de base (API v1)

```bash
# Production (HTTPS)
# Recherche par nom
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol"

# Recherche par CIS (Code Identifiant de Spécialité)
curl "https://medicaments-api.giygas.dev/v1/medicaments/61504672"

# Pagination (10 médicaments par page, défaut)
curl "https://medicaments-api.giygas.dev/v1/medicaments?page=1"

# Pagination avec pageSize personnalisé (50 médicaments par page)
curl "https://medicaments-api.giygas.dev/v1/medicaments?page=1&pageSize=50"

# Recherche par CIP via présentation
curl "https://medicaments-api.giygas.dev/v1/medicaments?cip=3400936403114"

# Export complet (~20MB)
curl "https://medicaments-api.giygas.dev/v1/medicaments/export"

# Local (Go native : port 8000, Docker : port 8030)
curl "http://localhost:8030/v1/medicaments?search=paracetamol"
curl "http://localhost:8030/health"
```

### Génériques (API v1)

```bash
# Génériques par libellé
curl "https://medicaments-api.giygas.dev/v1/generiques?libelle=paracetamol"

# Groupe générique par ID
curl "https://medicaments-api.giygas.dev/v1/generiques/1234"
```

### Présentations (API v1)

```bash
# Présentations par CIP
curl "https://medicaments-api.giygas.dev/v1/presentations/3400936403114"
```

### Recherche multi-mots

L'API supporte désormais la recherche multi-mots avec logique ET (tous les mots doivent être présents) :

```bash
# 2 mots - recherche précise
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol+500"

# 6 mots - recherche très précise
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol+500+mg+comprime+boite+20"
```

Note : Maximum de 6 mots par requête (protection DoS). Les mots peuvent être séparés par `+` ou espace.

### JavaScript/TypeScript

```javascript
// Recherche par nom
const response = await fetch(
  "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol",
);
const data = await response.json();

// Recherche par CIS
const response = await fetch(
  "https://medicaments-api.giygas.dev/v1/medicaments/61504672",
);
const data = await response.json();

// Pagination
const response = await fetch(
  "https://medicaments-api.giygas.dev/v1/medicaments?page=1",
);
const data = await response.json();
console.log(`Page ${data.page} of ${data.maxPage}`);

// Pagination avec pageSize personnalisé
const response2 = await fetch(
  "https://medicaments-api.giygas.dev/v1/medicaments?page=1&pageSize=50",
);
const data2 = await response2.json();
console.log(`Page ${data2.page} of ${data2.maxPage}, pageSize: ${data2.pageSize}`);
```

### Python

```python
import requests

# Recherche par nom
response = requests.get('https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol')
data = response.json()

# Recherche par CIS
response = requests.get('https://medicaments-api.giygas.dev/v1/medicaments/61504672')
data = response.json()

# Pagination
response = requests.get('https://medicaments-api.giygas.dev/v1/medicaments?page=1')
data = response.json()
print(f"Page {data['page']} of {data['maxPage']}")

# Pagination avec pageSize personnalisé
response2 = requests.get('https://medicaments-api.giygas.dev/v1/medicaments?page=1&pageSize=50')
data2 = response2.json()
print(f"Page {data2['page']} of {data2['maxPage']}, pageSize: {data2['pageSize']}")
```

## Sécurité et Robustesse

### Mesures de sécurité

- **Validation stricte** : 3-50 caractères alphanumériques + espaces (ASCII-only)
  - **Note** : Les données source BDPM sont en majuscules sans accents ni ponctuation (ex: IBUPROFENE, PARACETAMOL).
  - ⚠️ **Important** : Les apostrophes (`'`) et slash (`/`) sont acceptées. Les points consécutifs (`..`) sont bloqués.
  - **Recherche multi-mots** : Logique ET avec limite de 6 mots (protection DoS)
- **Protection injections** : `regexp.QuoteMeta` pour échappement
- **Rate limiting** : Token bucket (1000 tokens, 3/sec recharge, coûts variables 5-200 tokens selon endpoint)
  - Headers dans les réponses : `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Rate`, `Retry-After`
- **Limites de recherche** : Maximum 250 résultats pour médicaments, 100 pour génériques
  - Renvoie HTTP 400 si dépassé, avec message guidant vers `/v1/medicaments/export`
- **Middleware de protection** : Taille des requêtes et headers configurables
- **CORS configuré** : Géré via nginx en production

### Robustesse et résilience

- **Zero-downtime** : `atomic.Value` et `atomic.Bool` pour basculement
- **Logging structuré** : `slog` avec rotation de fichiers automatique
- **Monitoring proactif** : Alertes si > 25h sans mise à jour
- **Health checks** : Métriques détaillées (data+system), uptime, mises à jour
- **Graceful shutdown** : Timeout 30s + 2s pour finaliser requêtes
- **Concurrency safe** : `sync.RWMutex` et opérations atomiques

## Docker

```bash
# Initial setup (première fois)
make setup-secrets
make obs-init

# Build Docker image
make build

# Démarrer tous les services (API + observabilité)
make up

# Accès : http://localhost:8030
```

Pour la documentation complète Docker, voir [DOCKER.md](DOCKER.md)

## Documentation

- 📖 **[Spécification OpenAPI complète](html/docs/openapi.yaml)** - Définition complète de l'API avec exemples
- 🐳 **[Guide Docker complet](DOCKER.md)** - Setup Docker, stack observabilité, monitoring avancé
- 🏗️ **[Architecture du système](docs/ARCHITECTURE.md)** - Design des interfaces, flux de données, middleware stack
- ⚡ **[Performance et benchmarks](docs/PERFORMANCE.md)** - Mesures de performance, optimisations, profilage
- 🛠️ **[Guide de développement](docs/DEVELOPMENT.md)** - Setup, build, test, lint, configuration
- 📝 **[Guide de migration v1](docs/MIGRATION.md)** - Migration depuis les endpoints legacy vers v1
- 🧪 **[Guide de tests](docs/TESTING.md)** - Stratégies de tests, benchmarks, couverture

## Développement Local

Pour le guide de développement complet, voir [Guide de développement](docs/DEVELOPMENT.md).

### Démarrage Rapide

```bash
# Cloner et configurer
git clone https://github.com/giygas/medicaments-api.git
cd medicaments-api

# Installer les dépendances et configurer l'environnement
go mod tidy
cp .env.example .env

# Lancer le serveur de développement
go run .
```

### Commandes Principales

```bash
# Build
go build -o medicaments-api .

# Tests
go test -v ./...

# Formatage et analyse
gofmt -w .
go vet ./...
```

**Pour plus de détails sur le développement, les tests et les benchmarks, voir [Guide de développement](docs/DEVELOPMENT.md).**

### Fonctionnalités du serveur de développement

- **Serveur local** : `http://localhost:8000` (Go native) ou `http://localhost:8030` (Docker)
- **Profiling pprof** : `http://localhost:6060` (quand ENV=dev)
- **Documentation interactive** : `http://localhost:8000/docs` ou `http://localhost:8030/docs` (Docker)
- **Health endpoint** : `http://localhost:8000/health` ou `http://localhost:8030/health` (Docker)
- **Observabilité (Docker)** : Grafana `http://localhost:3000`, Prometheus `http://localhost:9090`
  - Géré via le submodule `observability/` (voir [DOCKER.md](DOCKER.md))
  - Voir [OBSERVABILITY.md](OBSERVABILITY.md) pour l'utilisation avec l'API

## Limitations et Conditions d'Utilisation

### Limitations techniques

Ce service est gratuit et fonctionne avec des ressources limitées :

- **Rate limiting** : 1000 tokens/IP, recharge 3 tokens/seconde
- **Coûts variables** : 5-200 tokens/requête selon endpoint
- **Data size** : ~20MB avec 60-90MB RAM stable (150MB startup)
- **Pas de SLA** : Service "as-is" sans garantie de disponibilité
- **Dépendance externe** : Mises à jour selon disponibilité source BDPM
- **Validation stricte** : 3-50 caractères alphanumériques + espaces (ASCII-only)

### Conditions d'utilisation

- **Usage non-commercial** : L'API est destinée à un usage personnel ou éducatif
- **Respect de la licence** : Les données restent soumises à la licence BDPM
- **Attribution requise** : Mention de la source obligatoire
- **Pas d'altération** : Interdiction de modifier les données originales

## Support et Contact

### Obtenir de l'aide

- **Documentation** : [https://medicaments-api.giygas.dev/docs](https://medicaments-api.giygas.dev/docs)
- **Swagger UI** : [https://medicaments-api.giygas.dev/docs](https://medicaments-api.giygas.dev/docs)
- **OpenAPI spec** : [https://medicaments-api.giygas.dev/docs/openapi.yaml](https://medicaments-api.giygas.dev/docs/openapi.yaml)
- **Issues** : [GitHub Issues](https://github.com/giygas/medicaments-api/issues)
- **Health check** : [https://medicaments-api.giygas.dev/health](https://medicaments-api.giygas.dev/health)

## Licence et Conformité

### Licence du logiciel

Ce projet est distribué sous **GNU AGPL-3.0**.

- [Voir la licence complète](https://www.gnu.org/licenses/agpl-3.0.html)
- Obligation de partage des modifications
- Utilisation commerciale soumise à conditions

### Licence des données

Les données médicales restent soumises à la licence de la
**Base de Données Publique des Médicaments (BDPM)**.

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

Pour l'historique complet des versions et des changements détaillés, consultez le [CHANGELOG.md](CHANGELOG.md).

### Versions

- **v1.2.0** (Février 2026) - Architecture Docker refactorée avec submodule observabilité, pageSize parameter, limites de recherche
- **v1.1.0** (Février 2026) - API RESTful v1, améliorations de performance 22-207%, métriques Prometheus
- **v1.0.0** (Décembre 2025) - Version initiale

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
