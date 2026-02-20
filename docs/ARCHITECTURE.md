# Architecture du Système

## Vue d'ensemble

L'API des Médicaments repose sur une architecture modulaire et performante basée sur 6 interfaces principales qui séparent clairement les responsabilités pour une maintenabilité optimale.

### Caractéristiques clés

- **Parsing concurrent** : Traitement parallèle de 5 fichiers TSV BDPM
- **Atomic operations** : Mises à jour zero-downtime via `atomic.Value` et `atomic.Bool`
- **Cache HTTP intelligent** : ETag/Last-Modified pour optimisation des réponses
- **Stateless architecture** : Facilite la montée en charge horizontale
- **Memory optimization** : 60-90MB RAM stable (67.5MB médiane)

## Architecture des Interfaces

L'architecture repose sur 6 interfaces principales :

### DataStore

Gère le stockage atomique des données en mémoire avec des opérations thread-safe via `atomic.Value`, garantissant des mises à jour zero-downtime.

**Responsabilités :**

- Stockage atomique des médicaments et génériques
- Maps O(1) pour lookups CIS et group ID (_voir [Performance et benchmarks](PERFORMANCE.md) pour les métriques_)
- Opérations thread-safe pour lecture/écriture concurrente
- Bascullement instantané sans interruption de service

### HTTPHandler

Orchestre les requêtes et route les appels vers les bons handlers sans assertions de type.

**Responsabilités :**

- Dispatch des requêtes vers les handlers appropriés
- Gestion des erreurs HTTP
- Sérialisation des réponses JSON
- Middleware stack orchestration

_Pour plus d'informations sur les endpoints v1, consultez le [Guide de migration v1](MIGRATION.md)._

### Parser

Télécharge et traite les 5 fichiers TSV BDPM en parallèle, construisant les maps pour lookups O(1) (CIS → médicament, groupe ID → générique).

**Responsabilités :**

- Téléchargement concurrent des fichiers TSV BDPM
- Parsing avec détection automatique de charset (UTF-8/ISO8859-1)
- Validation et nettoyage des données
- Construction des maps O(1) pour lookups rapides
- Gestion des erreurs de parsing avec statistiques

### Scheduler

Planifie les mises à jour automatiques (6h et 18h) en coordonnant le parsing et le stockage.

**Responsabilités :**

- Planification via gocron (6h et 18h)
- Coordination Parser → DataStore
- Gestion des mises à jour atomiques
- Monitoring des échecs de mise à jour

### HealthChecker

Surveille la fraîcheur des données et collecte les métriques système.

**Responsabilités :**

- Vérification de l'âge des données (alerte si > 25h)
- Collecte des métriques système (goroutines, mémoire, GC)
- État de santé (healthy/degraded/unhealthy)
- Endpoint `/v1/diagnostics` pour monitoring détaillé

### DataValidator

Assainit les entrées utilisateur et valide l'intégrité des données.

**Responsabilités :**

- Validation des paramètres de requête (3-50 chars, ASCII-only)
- Détection de patterns dangereux (SQL injection, XSS, etc.)
- Validation CIS/CIP/CIP13/CIP7 ranges
- Validation des limites de mots (max 6 pour recherche)
- Validation CheckDuplicateCIP pour intégrité des présentations
- **Limites de résultats** : Maximum 250 résultats pour recherche médicaments, 100 pour génériques
  - Retourne HTTP 429 si dépassé pour prévenir l'abus
  - Guide les utilisateurs vers `/export` pour le dataset complet

_Voir le [Guide de tests](TESTING.md) pour les stratégies de validation des tests._

## Flux de Données

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

### Pipeline de mise à jour

1. **Téléchargement** : 5 fichiers TSV BDPM téléchargés simultanément
2. **Parsing concurrent** : Chaque fichier parsé en parallèle dans sa propre goroutine
3. **Validation** : Données validées et cross-référencées
4. **Construction maps** : Création des maps O(1) (medicamentsMap, generiquesMap)
5. **Swap atomique** : Échange instantané via `atomic.Value`
6. **Nettoyage** : Anciennes structures libérées par GC

_Pour configurer et lancer le pipeline de parsing, consultez le [Guide de développement](DEVELOPMENT.md)._

## Middleware Stack Complet

Stack Chi v5 optimisée pour la sécurité et la performance :

1. **RequestID** - Traçabilité unique par requête
2. **BlockDirectAccess** - Bloque les accès directs non autorisés (désactivé avec ALLOW_DIRECT_ACCESS=true pour Docker)
3. **RealIP** - Détection IP réelle derrière les proxies
4. **Logging structuré** - Logs avec slog pour monitoring
5. **RedirectSlashes** - Normalisation des URLs
6. **Recoverer** - Gestion des paniques avec recovery
7. **RequestSize** - Limites taille corps/headers (configurable)
8. **RateLimiting** - Token bucket avec coûts variables par endpoint et limites de résultats de recherche
   - Limite 250 résultats pour `/v1/medicaments?search`, 100 pour `/v1/generiques?libelle`
   - Retourne HTTP 429 avec message guidant vers `/export`

### Ordre d'exécution

```go
RequestID → BlockDirectAccess → RealIP → Logging →
RedirectSlashes → Recoverer → RequestSize → RateLimiting → Handler
```

## Cache HTTP Intelligent

L'API implémente un système de cache HTTP efficace pour optimiser les performances et réduire la charge serveur.

### Stratégies de cache

**Ressources statiques (documentation, OpenAPI, favicon)**

- Headers `Cache-Control` avec durées adaptées
- 1 heure pour la documentation
- 1 an pour le favicon

**Réponses API**

- `Last-Modified` : Date de dernière modification des données
- `ETag` : Hash SHA256 pour validation conditionnelle
- Réponses `304 Not Modified` sur requêtes répétées

### Compression

Compression gzip appliquée automatiquement :

- Réduit la taille des réponses JSON jusqu'à 80%
- Négociation content-encoding automatique
- Transparente pour le client

## Architecture Mémoire

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

### Caractéristiques mémoire

- **O(1) lookups** : Maps pour accès instantané par CIS ou group ID
- **Stabilité** : 55-80MB stable (67.5MB médiane)
- **Startup** : ~150MB pic initial après chargement des données
- **Zero-downtime** : Swap atomique sans allocation de structures temporaires

## Logging et Monitoring

### Rotation Automatique des Logs

L'API implémente un système de logging structuré avec rotation automatique :

**Fonctionnalités :**

- **Rotation Hebdomadaire** : Nouveau fichier chaque semaine (format ISO : `app-YYYY-Www.log`)
- **Rotation par Taille** : Rotation forcée si fichier dépasse `MAX_LOG_FILE_SIZE`
- **Nettoyage Automatique** : Suppression des fichiers plus anciens que `LOG_RETENTION_WEEKS`
- **Double Sortie** : Console (texte) + Fichier (JSON) pour faciliter le parsing
- **Arrêt Propre** : Fermeture gracieuse des fichiers avec context cancellation

**Configuration :**

```bash
LOG_RETENTION_WEEKS=4        # Nombre de semaines de conservation (1-52)
MAX_LOG_FILE_SIZE=104857600  # Taille max avant rotation (1MB-1GB, défaut: 100MB)
LOG_LEVEL=info               # Niveau de log console (les fichiers capturent toujours DEBUG)
```

**Structure des Fichiers :**

```
logs/
├── app-2025-W41.log              # Semaine en cours
├── app-2025-W40.log              # Semaine précédente
├── app-2025-W39.log              # 2 semaines précédentes
└── app-2025-W38_size_20251007_143022.log  # Rotation par taille
```

### Monitoring Intégré

#### Endpoint de santé simplifié

L'endpoint `/health` fournit une réponse rapide pour vérifier l'état de l'API :

```json
{
  "status": "healthy",
  "data": {
    "last_update": "2026-01-15T06:00:00Z",
    "data_age_hours": 2.5,
    "medicaments": 15420,
    "generiques": 5200,
    "is_updating": false
  }
}
```

**États de santé :**

- **`healthy`** : API opérationnelle avec données fraîches (< 24h)
- **`degraded`** : API opérationnelle mais données âgées (> 24h)
- **`unhealthy`** : API non opérationnelle (pas de médicaments en mémoire) - renvoie HTTP 503

#### Endpoint de diagnostics v1

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
    },
    "presentations_with_orphaned_cis": {
      "count": 6,
      "sample_cip": [3400935910882, 3400930279069]
    }
  }
}
```

**Métriques Clés - Diagnostics :**

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
  - `presentations_with_orphaned_cis` : Présentations référençant des CIS inexistants

_Pour la documentation complète de la stack d'observabilité (Grafana, Loki, Prometheus), consultez [OBSERVABILITY.md](OBSERVABILITY.md)._

_Pour la configuration Docker et le déploiement, consultez [DOCKER.md](DOCKER.md)._

## Stack Technique

### Architecture et Déploiement

Le projet utilise une architecture modulaire avec support Docker pour le staging et la production :

- **Environnement de développement** : Exécution directe (`go run .`)
- **Environnement Docker (staging)** : Conteneurs avec observabilité complète
- **Observabilité** : Stack séparée via submodule Git (Grafana, Loki, Prometheus, Alloy)

_Pour les détails de l'architecture Docker et de la configuration des conteneurs, consultez [DOCKER.md](DOCKER.md)._

_Pour la configuration et l'utilisation de la stack d'observabilité, consultez [OBSERVABILITY.md](OBSERVABILITY.md)._

### Core Technologies

- **Framework web** : Chi v5 avec middleware stack complet
- **Scheduling** : gocron pour les mises à jour automatiques (6h/18h)
- **Logging** : Structured logging avec slog et rotation de fichiers
- **Rate limiting** : juju/ratelimit (token bucket algorithm)
- **Atomic operations** : go.uber.org/atomic pour mises à jour zero-downtime
- **Configuration** : Validation d'environnement avec godotenv

### Data Processing

- **Encoding** : Support Windows-1252/UTF-8/ISO8859-1 → UTF-8 pour les fichiers TSV sources
- **Parsing** : Traitement concurrent de 5 fichiers TSV
- **Validation** : Validation stricte des données avec types forts
- **Memory** : Atomic operations pour zero-downtime updates

### Development & Operations

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

## Principes de Conception

L'architecture privilégie la simplicité, l'efficacité et la résilience :

### Atomic operations

- Mises à jour sans temps d'arrêt
- Swap instantané via `atomic.Value`
- Pas de downtime pour les utilisateurs

### Stateless architecture

- Facilite la montée en charge horizontale
- Chaque requête est indépendante
- Pas d'état partagé entre les requêtes

### Modular design

- Séparation claire des responsabilités
- Interface-based design pour testabilité
- Remplaçabilité des composants

### Memory optimization

- Cache intelligent pour des réponses rapides
- O(1) lookups pour les requêtes fréquentes
- Minimalisation des allocations

### Concurrency safety

- `sync.RWMutex` pour les accès concurrents
- Opérations atomiques pour les swaps de données
- Pas de race conditions dans le code critique
