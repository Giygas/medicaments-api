# Répertoire de Tests

Ce répertoire contient des tests spécialisés pour medicaments-api, organisés par objectif pour une meilleure maintenabilité.

## 📁 Organisation des Tests

### **Tests de Performance et Benchmarks**

| Fichier | Objectif | Commandes |
|------|---------|----------|
| `performance_benchmarks_test.go` | Benchmarks de performance de base avec logging de production | `go test ./tests -bench=. -benchmem`<br>`go test ./tests -bench=BenchmarkAlgorithmicPerformance -v`<br>`go test ./tests -bench=BenchmarkHTTPPerformance -v`<br>`go test ./tests -bench=BenchmarkRealWorldSearch -v`<br>`go test ./tests -bench=BenchmarkSustainedPerformance -v` |
| `documentation_claims_verification_test.go` | Valide toutes les revendications de la documentation contre les données réelles | `go test ./tests -run TestDocumentationClaimsVerification -v` |

### **Tests de Vérification & Validation**

| Fichier | Objectif | Commandes |
|------|---------|----------|
| `documentation_claims_verification_test.go` | Valide toutes les revendications de la documentation contre les données réelles | `go test ./tests -run TestDocumentationClaimsVerification -v` |

### **Tests d'Intégration**

| Fichier | Objectif | Commandes |
|------|---------|----------|
| `integration_test.go` | Tests d'intégration complets du pipeline | `go test ./tests -run TestIntegration -v` |
| `cross_file_consistency_integration_test.go` | Validation de la cohérence des données inter-fichiers | `go test ./tests -run TestIntegrationCrossFileConsistency -v` |

### **Tests des Endpoints API**

| Fichier | Objectif | Commandes |
|------|---------|----------|
| `endpoints_test.go` | Validation du comportement des endpoints API | `go test ./tests -run TestEndpoints -v` |
| `etag_test.go` | Tests du mécanisme de cache HTTP | `go test ./tests -run TestETagFunctionality -v` |

### **Tests de Fumée (Smoke Tests)**

| Fichier | Objectif | Commandes |
|------|---------|----------|
| `smoke_test.go` | Validation rapide et tests de fumée | `go test ./tests -run TestApplicationStartupSmoke -v` |

## 🚀 Commandes de Démarrage Rapide

### **Exécuter Tous les Tests**
```bash
# Tous les tests dans le répertoire tests
go test ./tests -v

# Tous les tests dans tout le projet
go test -v ./...
```

### **Tests de Performance**
```bash
# Tous les benchmarks avec environnement de production (performance optimale)
go test ./tests -bench=. -benchmem

# Benchmarks algorithmiques (niveau handler)
go test ./tests -bench=BenchmarkAlgorithmicPerformance -benchmem -v

# Benchmarks HTTP (niveau réseau)
go test ./tests -bench=BenchmarkHTTPPerformance -benchmem -v

# Benchmarks de recherche réels
go test ./tests -bench=BenchmarkRealWorldSearch -benchmem -v

# Benchmarks de performance soutenus
go test ./tests -bench=BenchmarkSustainedPerformance -benchmem -v
```

### **Vérification de la Documentation**
```bash
# Vérifier toutes les revendications de la documentation
go test ./tests -run TestDocumentationClaimsVerification -v
```

### **Tests d'Intégration**
```bash
# Tests complets du pipeline
go test ./tests -run TestIntegrationFullDataParsingPipeline -v
go test ./tests -run TestIntegrationConcurrentUpdates -v
go test ./tests -run TestIntegrationErrorHandling -v

# Cohérence inter-fichiers
go test ./tests -run TestIntegrationCrossFileConsistency -v
```

### **Tests d'Endpoints & Middleware**
```bash
# Tous les tests d'endpoints
go test ./tests -run TestEndpoints -v

# Tests de middleware
go test ./tests -run TestBlockDirectAccessMiddleware -v
go test ./tests -run TestRateLimiter -v
go test ./tests -run TestRealIPMiddleware -v
go test ./tests -run TestRequestSizeMiddleware -v
go test ./tests -run TestCompressionOptimization -v

# Tests ETag
go test ./tests -run TestETagFunctionality -v
```

### **Tests de Fumée (Smoke Tests)**
```bash
# Validation rapide
go test ./tests -run TestApplicationStartupSmoke -v
```

## 📊 Résumé de Performance

Les benchmarks de performance utilisent l'environnement de production (`config.EnvProduction`) pour une performance optimale. Cela signifie :

- **Logging console** : Niveau WARN et supérieurs uniquement (pas de sortie INFO/DEBUG sur la console)
- **Logging fichier** : Sortie JSON complète (tous les niveaux écrits dans les fichiers de logs rotatifs)
- **Résultat** : Élimine l'overhead d'E/S console pendant les benchmarks pour des mesures précises

### Exemple de Sortie du Rapport de Vérification

Lors de l'exécution de la vérification des revendications de la documentation :

```bash
go test ./tests -run TestDocumentationClaimsVerification -v
```

Vous verrez une sortie comme :
```
=== COMPREHENSIVE DOCUMENTATION CLAIMS VERIFICATION ===

--- ALGORITHMIC PERFORMANCE VERIFICATION ---
  /v1/medicaments/{cis}: 441695.7 req/sec (revendiqué: 400000.0 req/sec, diff: 10.4%)
  /v1/medicaments/{cis}: 2.0 µs (revendiqué: 3.0 µs, diff: -33.3%)
  /v1/generiques/{groupID}: 244390.5 req/sec (revendiqué: 200000.0 req/sec, diff: 22.2%)
  /v1/generiques/{groupID}: 4.0 µs (revendiqué: 5.0 µs, diff: -20.0%)
  /v1/medicaments?page={n}: 40152.5 req/sec (revendiqué: 40000.0 req/sec, diff: 0.4%)
  /v1/medicaments?page={n}: 24.0 µs (revendiqué: 30.0 µs, diff: -20.0%)
  /v1/medicaments?search={query}: 1634.3 req/sec (revendiqué: 1600.0 req/sec, diff: 2.1%)
  /v1/medicaments?search={query}: 611.0 µs (revendiqué: 750.0 µs, diff: -18.5%)
  /v1/generiques?libelle={nom}: 16742.9 req/sec (revendiqué: 18000.0 req/sec, diff: -7.0%)
  /v1/generiques?libelle={nom}: 59.0 µs (revendiqué: 60.0 µs, diff: -1.7%)
  /v1/presentations?cip={code}: 438566.6 req/sec (revendiqué: 430000.0 req/sec, diff: 2.0%)
  /v1/presentations?cip={code}: 2.0 µs (revendiqué: 2.0 µs, diff: 0.0%)
  /v1/medicaments?cip={code}: 394485.4 req/sec (revendiqué: 375000.0 req/sec, diff: 5.2%)
  /v1/medicaments?cip={code}: 2.0 µs (revendiqué: 5.0 µs, diff: -60.0%)
  /health: 416206.4 req/sec (revendiqué: 400000.0 req/sec, diff: 4.1%)
  /health: 2.0 µs (revendiqué: 3.0 µs, diff: -33.3%)

--- HTTP PERFORMANCE VERIFICATION ---
  /v1/medicaments/{cis}: 90015.7 req/sec (revendiqué: 78000.0 req/sec, diff: 15.4%)
  /v1/medicaments?page={n}: 49463.0 req/sec (revendiqué: 41000.0 req/sec, diff: 20.6%)
  /v1/medicaments?search={query}: 7412.0 req/sec (revendiqué: 6100.0 req/sec, diff: 21.5%)
  /v1/generiques?libelle={nom}: 46865.7 req/sec (revendiqué: 36000.0 req/sec, diff: 30.2%)
  /v1/presentations?cip={code}: 91614.3 req/sec (revendiqué: 77000.0 req/sec, diff: 19.0%)
  /v1/medicaments?cip={code}: 92352.7 req/sec (revendiqué: 75000.0 req/sec, diff: 23.1%)
  /health: 114412.3 req/sec (revendiqué: 92000.0 req/sec, diff: 24.4%)

--- MEMORY USAGE VERIFICATION ---
  Application memory: 75.3 MB alloc, 158.1 MB sys (revendiqué: 70.0-90.0 MB)

--- PARSING PERFORMANCE VERIFICATION ---
  Parsing time: 0.5 seconds (revendiqué: 0.7 seconds)

=== VERIFICATION REPORT ===
  ✅ PASS /v1/medicaments/{cis} algorithmic throughput: 441695.7 req/sec (revendiqué: 400000.0 req/sec, diff: 10.4%)
  ✅ PASS /v1/medicaments/{cis} algorithmic latency: 2.0 µs (revendiqué: 3.0 µs, diff: -33.3%)
  ✅ PASS /v1/generiques/{groupID} algorithmic throughput: 244390.5 req/sec (revendiqué: 200000.0 req/sec, diff: 22.2%)
  ✅ PASS /v1/generiques/{groupID} algorithmic latency: 4.0 µs (revendiqué: 5.0 µs, diff: -20.0%)
  ✅ PASS /v1/medicaments?page={n} algorithmic throughput: 40152.5 req/sec (revendiqué: 40000.0 req/sec, diff: 0.4%)
  ✅ PASS /v1/medicaments?page={n} algorithmic latency: 24.0 µs (revendiqué: 30.0 µs, diff: -20.0%)
  ✅ PASS /v1/medicaments?search={query} algorithmic throughput: 1634.3 req/sec (revendiqué: 1600.0 req/sec, diff: 2.1%)
  ✅ PASS /v1/medicaments?search={query} algorithmic latency: 611.0 µs (revendiqué: 750.0 µs, diff: -18.5%)
  ✅ PASS /v1/generiques?libelle={nom} algorithmic throughput: 16742.9 req/sec (revendiqué: 18000.0 req/sec, diff: -7.0%)
  ✅ PASS /v1/generiques?libelle={nom} algorithmic latency: 59.0 µs (revendiqué: 60.0 µs, diff: -1.7%)
  ✅ PASS /v1/presentations?cip={code} algorithmic throughput: 438566.6 req/sec (revendiqué: 430000.0 req/sec, diff: 2.0%)
  ✅ PASS /v1/presentations?cip={code} algorithmic latency: 2.0 µs (revendiqué: 2.0 µs, diff: 0.0%)
  ✅ PASS /v1/medicaments?cip={code} algorithmic throughput: 394485.4 req/sec (revendiqué: 375000.0 req/sec, diff: 5.2%)
  ✅ PASS /v1/medicaments?cip={code} algorithmic latency: 2.0 µs (revendiqué: 5.0 µs, diff: -60.0%)
  ✅ PASS /health algorithmic throughput: 416206.4 req/sec (revendiqué: 400000.0 req/sec, diff: 4.1%)
  ✅ PASS /health algorithmic latency: 2.0 µs (revendiqué: 3.0 µs, diff: -33.3%)
  ✅ PASS /v1/medicaments/{cis} HTTP throughput: 90015.7 req/sec (revendiqué: 78000.0 req/sec, diff: 15.4%)
  ✅ PASS /v1/medicaments?page={n} HTTP throughput: 49463.0 req/sec (revendiqué: 41000.0 req/sec, diff: 20.6%)
  ✅ PASS /v1/medicaments?search={query} HTTP throughput: 7412.0 req/sec (revendiqué: 6100.0 req/sec, diff: 21.5%)
  ✅ PASS /v1/generiques?libelle={nom} HTTP throughput: 46865.7 req/sec (revendiqué: 36000.0 req/sec, diff: 30.2%)
  ✅ PASS /v1/presentations?cip={code} HTTP throughput: 91614.3 req/sec (revendiqué: 77000.0 req/sec, diff: 19.0%)
  ✅ PASS /v1/medicaments?cip={code} HTTP throughput: 92352.7 req/sec (revendiqué: 75000.0 req/sec, diff: 23.1%)
  ✅ PASS /health HTTP throughput: 114412.3 req/sec (revendiqué: 92000.0 req/sec, diff: 24.4%)
  ✅ PASS Application memory usage: 75.3 MB (revendiqué: 80.0 MB, diff: -5.9%)
  ✅ PASS Concurrent TSV parsing: 0.5 seconds (revendiqué: 0.7 seconds, diff: -30.9%)

SUMMARY: 25/25 claims verified (100.0%)
```

### Interprétation du Rapport

**Indicateurs de Statut :**
- ✅ PASS = Répond ou dépasse la revendication (dans la tolérance)
- ❌ FAIL = En dessous du seuil minimum (plus de tolérance en dessous de la revendication)

**Sections de Performance :**
- **Algorithmic** : Benchmarks de niveau handler avec sous-ensemble de données (~500 éléments)
- **HTTP** : Benchmarks de niveau réseau avec ensemble complet de données (~15K+ éléments)
- **Memory** : Utilisation mémoire de l'application sous charge
- **Parsing** : Temps de traitement des fichiers TSV en parallèle

**Métriques :**
- **Throughput** : Requêtes par seconde (plus élevé est meilleur)
- **Latency** : Microsecondes par opération (plus bas est meilleur)
- **Diff** : Différence en pourcentage de la valeur revendiquée
  - Positif = Mesuré plus élevé que revendiqué (bon !)
  - Négatif = Mesuré plus bas que revendiqué (dans la tolérance c'est OK)

**Paramètres de Tolérance :**
- Revendications algorithmiques : 20% de tolérance (30% pour les endpoints de recherche)
- Revendications de throughput HTTP : 25% de tolérance (pour la variance réseau)
- Revendication mémoire : Plage de 70-90 MB (80 MB moyen)
- Temps de parsing : 100% de tolérance (pour la variabilité CI)

**Impact de l'Environnement :**
L'utilisation de `config.EnvProduction` assure que les benchmarks s'exécutent avec un logging de type production :
- Console : WARN et supérieurs uniquement (overhead d'E/S minimal)
- Fichier : Sortie JSON complète (tous les niveaux capturés)
- Résultat : Mesures de performance plus précises

### Revendications de Performance Actuelles

Les optimisations récentes ont considérablement amélioré la performance :

**1. Noms Normalisés Pré-calculés** (commit précédent) :
- Ajout du champ `DenominationNormalized` à l'entité Medicament
- Ajout du champ `LibelleNormalized` à l'entité GeneriqueList
- La normalisation se produit une fois pendant le parsing au lieu de chaque requête
- **Résultat** : Amélioration de 10x de la performance de recherche

**2. Logging Sensible à l'Environnement** (commit actuel) :
- Les environnements de production/test utilisent un logging console réduit
- Console : WARN/ERREUR uniquement (vs INFO en dev)
- Fichier : Sortie JSON complète toujours
- **Résultat** : Élimine l'overhead d'E/S console pendant les benchmarks

**Effet Combiné** : Amélioration de 2-3x du throughput HTTP sur la plupart des endpoints

### Revendications de Performance Actuelles (Documentation)

**Benchmarks Algorithmiques** (niveau handler avec sous-ensemble de données ~500 éléments) :
- `/v1/medicaments/{cis}` : 400,000 req/sec, 3.0µs latence
- `/v1/generiques/{groupID}` : 200,000 req/sec, 5.0µs latence
- `/v1/medicaments?page={n}` : 40,000 req/sec, 30.0µs latence
- `/v1/medicaments?search={query}` : 1,600 req/sec, 750.0µs latence
- `/v1/generiques?libelle={nom}` : 18,000 req/sec, 60.0µs latence
- `/v1/presentations?cip={code}` : 430,000 req/sec, 2.0µs latence
- `/v1/medicaments?cip={code}` : 375,000 req/sec, 5.0µs latence
- `/health` : 400,000 req/sec, 3.0µs latence

**Benchmarks HTTP** (niveau réseau avec ensemble complet de données ~15K+ éléments) :
- `/v1/medicaments/{cis}` : 78,000 req/sec
- `/v1/medicaments?page={n}` : 41,000 req/sec
- `/v1/medicaments?search={query}` : 6,100 req/sec
- `/v1/generiques?libelle={nom}` : 36,000 req/sec
- `/v1/presentations?cip={code}` : 77,000 req/sec
- `/v1/medicaments?cip={code}` : 75,000 req/sec
- `/health` : 92,000 req/sec

**Utilisation Mémoire** : 70-90 MB (80 MB médian)
**Parsing Concurrent** : ~0.5-0.7 secondes pour l'ensemble complet de données

## 📋 Couverture de Tests

### Types de Tests

- **Tests Unitaires** : Dans les répertoires de paquets individuels (`*_test.go`)
- **Tests d'Intégration** : `integration_test.go`, `cross_file_consistency_integration_test.go`
- **Tests de Performance** : `performance_benchmarks_test.go`
- **Vérification de Documentation** : `documentation_claims_verification_test.go`
- **Tests d'Endpoints** : `endpoints_test.go`
- **Tests de Middleware** : Dans `server/middleware_test.go`
- **Tests ETag** : `etag_test.go`
- **Tests de Fumée** : `smoke_test.go`

## 📝 Notes

- Tous les tests utilisent `package main` pour accéder au code de l'application principale
- Les tests sont organisés par objectif, pas par taille de fichier
- Les benchmarks de performance utilisent l'environnement de production pour des mesures optimales
- Les tests d'intégration utilisent des données BDPM réelles pour des tests authentiques
- La vérification de documentation assure l'exactitude des revendications publiques

## 🔧 Développement

### Exécution des Benchmarks

Lors de l'exécution des benchmarks de performance, ils utilisent automatiquement l'environnement de production :

```go
// Toutes les fonctions de benchmark initialisent avec le logging de production
logging.InitLoggerWithEnvironment("", config.EnvProduction, 4, 100*1024*1024)
```

Cela assure :
- Pas d'overhead d'E/S console (WARN/ERREUR sur la console uniquement)
- Le logging fichier capture toute la sortie pour analyse
- Mesures de performance précises (environnement de type production)

### Ajout de Nouveaux Tests

1. **Benchmarks de performance** → Ajouter à `performance_benchmarks_test.go`
2. **Tests d'intégration** → Ajouter à `integration_test.go` ou créer un nouveau fichier
3. **Nouvelle vérification** → Ajouter à `documentation_claims_verification_test.go`
4. **Tests unitaires** → Garder dans les répertoires de paquets respectifs

### Lignes Directrices d'Organisation des Tests

- **Garder les tests organisés par objectif** : Performance, Intégration, Vérification, Endpoint, Fumée
- **Utiliser des noms de tests descriptifs** : Clarifier ce que chaque test valide
- **Tester les cas limites** : Inclure à la fois le chemin heureux et les scénarios d'erreur
- **Utiliser des helpers de test** : Configuration commune/nettoyage dans des fonctions helper
- **Éviter la pollution des tests** : Nettoyer les ressources dans le nettoyage de tests (defer, t.Cleanup())
- **Utiliser l'environnement de production pour les benchmarks** : Assure des mesures de performance réalistes

Cette organisation garde le répertoire racine propre tout en maintenant une couverture de tests complète.
