# Guide de Tests

## Vue d'ensemble

Ce guide couvre les stratégies de tests pour l'API des Médicaments, incluant les tests unitaires, les tests d'intégration, les benchmarks et les tests de performance.

## Tests Rapides

### Exécuter tous les tests
```bash
go test -v
```

### Tests unitaires uniquement
```bash
go test -short -v
```

### Test spécifique
```bash
go test -run TestName -v
```

### Tests avec détection de race
```bash
go test -race -v
```

### Smoke tests
```bash
go test ./tests -run TestSmoke -v
```

## Tests v1

### Médicaments
```bash
go test ./handlers -run TestServeMedicamentsV1 -v
```

### Présentations
```bash
go test ./handlers -run TestServePresentationsV1 -v
```

### Génériques
```bash
go test ./handlers -run TestServeGeneriquesV1 -v
```

### Diagnostics
```bash
go test ./handlers -run TestServeDiagnosticsV1 -v
```

## Tests d'Intégration

### Pipeline complet de parsing
```bash
go test -run TestIntegrationFullDataParsingPipeline -v
```

### Mises à jour concurrentes
```bash
go test -run TestIntegrationConcurrentUpdates -v
```

### Utilisation mémoire
```bash
go test -run TestIntegrationMemoryUsage -v
```

## Tests du Parser

### Parser unitaires
```bash
go test ./medicamentsparser -v
```

### Couverture parser
```bash
go test ./medicamentsparser -coverprofile=parser_coverage.out
```

## Couverture

### Générer rapport de couverture
```bash
go test -coverprofile=coverage.out -v
```

### Générer HTML de couverture
```bash
go tool cover -html=coverage.out -o coverage.html
```

### Vérifier le pourcentage de couverture
```bash
go tool cover -func=coverage.out
```

### Couverture par package
```bash
go test ./handlers -coverprofile=handlers_coverage.out
go test ./medicamentsparser -coverprofile=parser_coverage.out
go test ./tests -coverprofile=tests_coverage.out
```

## Benchmarks

### Exécuter tous les benchmarks v1
```bash
go test ./handlers -bench=. -benchmem -v
```

### Benchmark spécifique v1
```bash
go test ./handlers -bench=BenchmarkMedicamentByCIS -benchmem -v
```

### Benchmark avec profil CPU
```bash
go test ./handlers -bench=. -benchmem -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

### Vérification des claims de documentation
```bash
go test -run=TestDocumentationClaimsVerification -v
```

### Benchmarks de pipeline d'intégration
```bash
go test ./tests -run TestIntegrationFullDataParsingPipeline -v
```

### Benchmarks complets avec sous-tests
```bash
# Tests algorithmiques
go test ./tests -bench=BenchmarkAlgorithmicPerformance -benchmem -run=^$

# Tests HTTP complets
go test ./tests -bench=BenchmarkHTTPPerformance -benchmem -run=^$

# Tests de recherche réels
go test ./tests -bench=BenchmarkRealWorldSearch -benchmem -run=^$

# Tests de charge soutenus
go test ./tests -bench=BenchmarkSustainedPerformance -benchmem -run=^$
```

## Stratégie de Tests

### Tests unitaires

**Objectif :** Feedback rapide avec mock data

**Caractéristiques :**
- Focus sur la logique individuelle des composants
- Utilisation de mocks pour les dépendances externes
- Exécutés rapidement (< 1 seconde)

**Exécution :**
```bash
go test -short -v
```

### Tests d'intégration

**Objectif :** Validation du pipeline de parsing réel

**Caractéristiques :**
- Pipeline de parsing réel (~15K médicaments)
- Détection complète des conditions de race
- Validation de l'intégrité des données
- Plus lents (10-30 secondes)

**Exécution :**
```bash
go test ./tests -run TestIntegrationFullDataParsingPipeline -v
go test ./tests -run TestIntegrationConcurrentUpdates -v
go test ./tests -run TestIntegrationMemoryUsage -v
```

### Tests de performance

**Objectif :** Benchmarking pour vérifier les claims de performance

**Caractéristiques :**
- Mesures de latence, throughput, et allocations
- Tolérances de 25% dans CI/CD pour variations d'environnement
- Tests non-bloquants dans le pipeline CI/CD

**Exécution :**
```bash
# Tous les benchmarks
go test ./handlers -bench=. -benchmem -v

# Benchmarks spécifiques
go test ./handlers -bench=BenchmarkMedicamentByCIS -benchmem -v
```

### Tests de fumée (Smoke Tests)

**Objectif :** Validation rapide pour CI/CD

**Caractéristiques :**
- Tests critiques de fonctionnalité principale
- Exécutés rapidement (< 10 secondes)
- Échouent rapidement si un problème majeur existe

**Exécution :**
```bash
go test ./tests -run TestSmoke -v
```

## Couverture Cible

### Exigences minimales

- **Couverture globale** : 75% minimum
- **Couverture handlers** : 75% minimum
- **Couverture parser** : 75% minimum

### Rapport actuel (v1.1.0)

- **Couverture globale** : 78.5%
- **Handlers** : 85.6%
- **Medicaments Parser** : 84.2%

*Voir [Performance et benchmarks](PERFORMANCE.md) pour comprendre l'impact de la couverture sur les performances.*

### Vérifier la couverture

```bash
# Générer le rapport de couverture
go test -coverprofile=coverage.out ./...

# Vérifier la couverture totale
go tool cover -func=coverage.out | grep total

# Générer un rapport HTML détaillé
go tool cover -html=coverage.out -o coverage.html
```

## Benchmarks Disponibles

*Pour des métriques de performance détaillées et l'analyse des optimisations, voir [Performance et benchmarks](PERFORMANCE.md).*

### Benchmarks v1 (handlers)

| Benchmark | Endpoint v1 | Type de lookup |
|-----------|--------------|----------------|
| `BenchmarkMedicamentsExport` | `/v1/medicaments?export=all` | Full export |
| `BenchmarkMedicamentsPagination` | `/v1/medicaments?page={n}` | Pagination |
| `BenchmarkMedicamentsSearch` | `/v1/medicaments?search={q}` | Regex search |
| `BenchmarkMedicamentByCIS` | `/v1/medicaments/{cis}` | O(1) lookup |
| `BenchmarkMedicamentByCIP` | `/v1/medicaments?cip={code}` | O(2) lookups |
| `BenchmarkGeneriquesSearch` | `/v1/generiques?libelle={n}` | Regex search |
| `BenchmarkGeneriqueByGroup` | `/v1/generiques/{groupID}` | O(1) lookup |
| `BenchmarkPresentationByCIP` | `/v1/presentations/{cip}` | O(1) lookup |

### Benchmarks complets (tests)

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkAlgorithmicPerformance` | Tests complets de performance algorithmique (CISLookup, GenericGroupLookup, Pagination, Search, PresentationsLookup) |
| `BenchmarkHTTPPerformance` | Tests complets de performance HTTP (CISLookup, GenericGroupLookup, GenericSearch, HealthCheck avec stack complète) |
| `BenchmarkRealWorldSearch` | Tests de recherche réels (CommonMedication, BrandName, ShortQuery, etc.) |
| `BenchmarkSustainedPerformance` | Tests de charge soutenus (ConcurrentLoad, MixedEndpoints, MemoryUnderLoad) |

## CI/CD

### Workflow GitHub Actions

Le pipeline CI/CD exécute :

1. **Tests unitaires avec détection de race** : `go test -race -v ./...`
2. **Tests d'intégration complets** : `go test ./tests/ -v`
3. **Benchmarks non-bloquants** : `go test ./handlers -bench=. -benchmem`
4. **Vérification de couverture** : `go test -coverprofile=coverage.out` avec vérification ≥ 75%
5. **Formatage du code** : `gofmt -d .`
6. **Analyse statique** : `go vet ./...`
7. **Sécurité** : Scans gosec et govulncheck (workflow `security.yml`)
8. **Déploiement** : Build et déploiement automatique si tous les tests passent

### Workflows supplémentaires

#### security.yml - Analyse de sécurité

Exécuté automatiquement sur chaque pull request vers `main` :

- **gosec** : Analyse statique de sécurité du code Go
  - Bloque le pipeline sur sévérité HIGH ou CRITICAL
  - Génère rapport JSON téléchargeable comme artifact

- **govulncheck** : Détection de vulnérabilités connues
  - Vérifie les dépendances contre la base de données Go
  - Bloque le pipeline si des vulnérabilités sont trouvées

#### rollback.yml - Rollback en production

Workflow manuel pour restaurer une version précédente :

- Sélection de version par tag (ex: `v1.1.0`) ou répertoire de backup
- Restauration automatique du binaire et fichiers HTML
- Arrêt propre du service avant restauration
- Health check post-rollback
- Listing des backups disponibles avant sélection

### Exécuter localement avant push

*Pour le développement local et la configuration, voir [Guide de développement](DEVELOPMENT.md).*

```bash
# Test complet avec détection de race
go test -race -v ./...

# Vérifier couverture
go test -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total

# Exécuter les benchmarks
go test ./handlers -bench=. -benchmem -v

# Vérifier le formatage
gofmt -d .

# Analyse statique
go vet ./...
```

### Tolérance CI/CD

Les benchmarks ont une tolérance de 25% pour tenir compte des variations d'environnement :
- Variations de CPU
- Variations de mémoire
- Variations de charge système

## Dépannage

### Tests lents

**Symptôme :** Les tests prennent trop de temps

**Solution :** Utiliser `go test -short` pour sauter les tests d'intégration lents

```bash
go test -short -v
```

### Race conditions détectées

**Symptôme :** `go test -race -v` rapporte des race conditions

**Solution :** Identifier les accès concurrents non-synchronisés avec le rapport détaillé

```bash
go test -race -v
```

### Couverture faible

**Symptôme :** Couverture < 75%

**Solution :** Identifier les fichiers avec peu de couverture

```bash
# Générer un rapport HTML
go tool cover -html=coverage.out

# Identifier les fichiers avec peu de couverture
go tool cover -func=coverage.out | sort -k3 -n
```

### Benchmarks variables

**Symptôme :** Les résultats des benchmarks varient beaucoup

**Solution :** Les benchmarks peuvent varier selon la charge système. La tolérance CI/CD de 25% est normale.

```bash
# Exécuter plusieurs fois pour une moyenne
go test -bench=. -benchmem -count=3 -v
```

### Tests échouent avec des erreurs de connexion

**Symptôme :** Tests échouent avec des erreurs de téléchargement BDPM

**Solution :** Vérifier la connexion internet et l'accès à la BDPM

```bash
curl -I https://base-donnees-publique.medicaments.gouv.fr
```

## Bonnes Pratiques

### Écrire des tests pour chaque nouvelle fonctionnalité

- Tests unitaires pour la logique business
- Tests d'intégration pour les endpoints
- Benchmarks pour les chemins critiques de performance

### Maintenir une couverture élevée

- Viser ≥ 75% de couverture
- Tester les cas limites et les erreurs
- Utiliser les tests de table pour les cas multiples

### Exécuter les tests avec détection de race

```bash
go test -race -v ./...
```

Cela permet de détecter les problèmes de concurrence avant qu'ils ne surviennent en production.

### Utiliser les mocks pour les tests unitaires

- Mock des dépendances externes
- Tester la logique isolée
- Tests rapides et fiables

### Profiler les tests lents

```bash
# Profil CPU des tests
go test -cpuprofile=cpu.prof -v

# Analyser le profil
go tool pprof cpu.prof
```

### Documenter les tests complexes

- Ajouter des commentaires pour expliquer la logique
- Utiliser des noms de tests descriptifs
- Documenter les assertions importantes

## Référence des Commandes de Test

### Tests v1

```bash
# Tous les tests v1
go test ./handlers -run "TestServe.*V1" -v

# Tests medicaments
go test ./handlers -run TestServeMedicamentsV1 -v

# Tests présentations
go test ./handlers -run TestServePresentationsV1 -v

# Tests génériques
go test ./handlers -run TestServeGeneriquesV1 -v

# Tests diagnostics
go test ./handlers -run TestServeDiagnosticsV1 -v
```

### Tests d'intégration

```bash
# Pipeline complet
go test ./tests -run TestIntegrationFullDataParsingPipeline -v

# Concurrent updates
go test ./tests -run TestIntegrationConcurrentUpdates -v

# Memory usage
go test ./tests -run TestIntegrationMemoryUsage -v
```

### Benchmarks

```bash
# Tous les benchmarks
go test ./handlers -bench=. -benchmem -v

# Benchmark spécifique
go test ./handlers -bench=BenchmarkMedicamentByCIS -benchmem -v

# Avec profil CPU
go test ./handlers -bench=. -benchmem -cpuprofile=cpu.prof -v
```

### Couverture

```bash
# Rapport de couverture
go test -coverprofile=coverage.out -v

# HTML de couverture
go tool cover -html=coverage.out -o coverage.html

# Vérifier couverture
go tool cover -func=coverage.out | grep total
```
