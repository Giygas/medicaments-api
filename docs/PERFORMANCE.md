# Performance et Optimisations

## Vue d'ensemble

L'API des Médicaments délivre des performances exceptionnelles grâce à des optimisations continues :

- **Lookups O(1)** par code CIS ou CIP atteignent **80K+ requêtes/seconde** en production
- **Recherches regex** atteignent **6,100 req/s** grâce aux noms normalisés pré-calculés
- **Mémoire stable** : 55-80MB avec 67.5MB médiane
- **Mises à jour atomiques** : Zero-downtime updates via atomic operations

*Pour comprendre l'architecture derrière ces performances, voir [Architecture du système](ARCHITECTURE.md).*

## Benchmarks Algorithmiques (Handler pur)

Ces benchmarks mesurent la logique pure du handler sans surcharge réseau (go test -bench).

### Résultats actuels

| Endpoint                         | Type de lookup | Reqs/sec | Latence | Allocs/op |
| -------------------------------- | -------------- | -------- | ------- | --------- |
| `/v1/medicaments/{cis}`          | O(1) lookup    | 400,000  | 3.0µs   | 38        |
| `/v1/generiques/{groupID}`       | O(1) lookup    | 200,000  | 5.0µs   | 37        |
| `/v1/generiques?libelle={nom}`   | Regex search   | 18,000   | 60µs    | 94        |
| `/v1/presentations/{cip}`        | O(1) lookup    | 430,000  | 2.0µs   | 63        |
| `/v1/medicaments?search={query}` | Regex search   | 1,600    | 600µs   | 94        |
| `/v1/medicaments?cip={code}`     | O(2) lookups   | 375,000  | 5.0µs   | 54        |
| `/v1/medicaments?page={n}`       | Pagination     | 40,000   | 30µs    | 38        |

**Note** : Benchmarks algorithmiques mesurent la logique pure du handler sans surcharge réseau.

*Pour les commandes de test détaillées et les stratégies de benchmarking, consultez le [Guide de tests](TESTING.md).*

## Performance en Production (HTTP complet)

Les résultats ci-dessous incluent l'overhead HTTP complet (middleware, logging, sérialisation, réseau).

### Conditions de test

- **Workers concurrents** : 300
- **Durée** : 3 secondes
- **Protocole** : HTTP/1.1 avec connexions persistantes
- **Middleware** : Stack complet activé

### Résultats

| Endpoint                         | HTTP Req/sec | Latence (avg) |
| -------------------------------- | ------------ | ------------- |
| `/v1/medicaments/{cis}`          | 78,000       | ~4ms          |
| `/v1/presentations/{cip}`        | 77,000       | ~4ms          |
| `/v1/medicaments?cip={code}`     | 75,000       | ~5ms          |
| `/v1/generiques?libelle={nom}`   | 36,000       | ~9ms          |
| `/v1/medicaments?page={n}`       | 41,000       | ~7ms          |
| `/v1/medicaments?search={query}` | 6,100        | ~50ms         |
| `/health`                        | 92,000       | ~3ms          |

## Exécuter les Benchmarks

### Lancer tous les benchmarks handlers

```bash
go test ./handlers -bench=. -benchmem -v
```

### Lancer tous les benchmarks tests complets

```bash
go test ./tests/ -bench=. -benchmem -run=^$
```

### Benchmark spécifique handler

```bash
go test -bench=BenchmarkMedicamentByCIS -benchmem -run=^$ ./handlers
go test -bench=BenchmarkMedicamentsExport -benchmem -run=^$ ./handlers
```

### Benchmark complet avec sous-tests

```bash
go test -bench=BenchmarkAlgorithmicPerformance -benchmem -run=^$ ./tests/
go test -bench=BenchmarkHTTPPerformance -benchmem -run=^$ ./tests/
```

### Sous-benchmark spécifique (exemple)

```bash
go test -bench=BenchmarkAlgorithmicPerformance/CISLookup -benchmem -run=^$ ./tests/
```

### Avec comptage multiple (plus fiable)

```bash
go test -bench=. -benchmem -count=3 -run=^$ ./handlers
```

### Benchmark avec profil CPU

```bash
go test -bench=. -benchmem -cpuprofile=cpu.prof -run=^$ ./handlers
go tool pprof cpu.prof
```

### Vérification des claims de documentation

```bash
go test ./tests/ -run TestDocumentationClaimsVerification -v
```

## Benchmarks v1 disponibles

| Benchmark                         | Endpoint v1                  | Type de lookup |
| --------------------------------- | ---------------------------- | -------------- |
| `BenchmarkMedicamentsExport`      | `/v1/medicaments/export` | Full export    |
| `BenchmarkMedicamentsPagination`  | `/v1/medicaments?page={n}`   | Pagination     |
| `BenchmarkMedicamentsSearch`      | `/v1/medicaments?search={q}` | Regex search   |
| `BenchmarkMedicamentByCIS`        | `/v1/medicaments/{cis}`      | O(1) lookup    |
| `BenchmarkMedicamentByCIP`        | `/v1/medicaments?cip={code}` | O(2) lookups   |
| `BenchmarkGeneriquesSearch`       | `/v1/generiques?libelle={n}` | Regex search   |
| `BenchmarkGeneriqueByGroup`       | `/v1/generiques/{groupID}`  | O(1) lookup    |
| `BenchmarkPresentationByCIP`      | `/v1/presentations/{cip}`    | O(1) lookup    |
| `BenchmarkAlgorithmicPerformance` | Test complet algorithmique   | Complet        |
| `BenchmarkHTTPPerformance`        | Test complet HTTP            | Complet        |
| `BenchmarkRealWorldSearch`        | Tests de recherche réels     | Complet        |
| `BenchmarkSustainedPerformance`   | Tests de charge soutenus     | Complet        |

### Notes sur les benchmarks complets

- **`BenchmarkAlgorithmicPerformance`** : Tests complets de performance algorithmique incluant CISLookup, GenericGroupLookup, Pagination, Search, et PresentationsLookup
- **`BenchmarkHTTPPerformance`** : Tests complets de performance HTTP incluant CISLookup, GenericGroupLookup, GenericSearch, et HealthCheck avec stack complète
- **`BenchmarkRealWorldSearch`** : Tests de recherche réels incluant CommonMedication, BrandName, ShortQuery, et autres scénarios
- **`BenchmarkSustainedPerformance`** : Tests de charge soutenus incluant ConcurrentLoad, MixedEndpoints, et MemoryUnderLoad

## Tests Spécialisés

| Test                                     | Description                           | Commande                                                          |
| ---------------------------------------- | ------------------------------------- | ----------------------------------------------------------------- |
| `TestDocumentationClaimsVerification`    | Vérification des claims documentation | `go test ./tests/ -run TestDocumentationClaimsVerification -v`    |
| `TestParsingTime`                        | Performance parsing                   | `go test ./tests/ -run TestParsingTime -v`                        |
| `TestIntegrationFullDataParsingPipeline` | Pipeline complet d'intégration        | `go test ./tests/ -run TestIntegrationFullDataParsingPipeline -v` |
| `TestRealWorldConcurrentLoad`            | Test de charge réel                   | `go test ./tests/ -run TestRealWorldConcurrentLoad -v`            |

## Optimisations v1.1.0

Les améliorations v1.1.0 ont été apportées à l'API pour augmenter considérablement le débit HTTP tout en maintenant une stabilité exceptionnelle.

### Noms normalisés pré-calculés

Élimine les opérations de chaîne répétées (ToLower(), ReplaceAll()) pendant les recherches en calculant les versions normalisées une seule fois lors du parsing des données BDPM.

**Avantages :**

- Réduction drastique des allocations mémoire par recherche (16,000 → 94)
- Lookups par chaîne directement au lieu de calculer à la volée
- Amélioration de la latence de recherche par un facteur important

### Logging environment-aware

Réduit l'overhead I/O console en production en n'activant pas le logging debug/info en environnement de production et de test. Seuls les messages WARN et ERROR sont loggés dans ces environnements.

**Avantages :**

- Réduction de ~40% de l'overhead de logging en production
- Maintient les logs complets en développement
- Meilleure visibilité des problèmes réels (WARN/ERROR)

### Optimisation de la validation

- Pré-compilation des regex au niveau package
- Remplacement des patterns de danger par `string.Contains()` (5-10x plus rapide que regex)
- Validation CIP/CIS directe via `strconv.Atoi()` sans regex

*Pour les bonnes pratiques de développement et la configuration, voir [Guide de développement](DEVELOPMENT.md).*

### Résultats combinés

Ces deux optimisations travaillent ensemble pour améliorer le débit HTTP de 2-5x sur la plupart des endpoints :

| Endpoint                         | Avant     | Après      | Amélioration |
| -------------------------------- | --------- | ---------- | ------------ |
| `/v1/presentations/{cip}`        | 35K req/s | 77K req/s  | +120%        |
| `/v1/medicaments/{cis}`          | 13K req/s | 78K req/s  | +500%        |
| `/v1/medicaments?cip={code}`     | 35K req/s | 75K req/s  | +114%        |
| `/v1/medicaments?page={n}`       | 20K req/s | 41K req/s  | +105%        |
| `/v1/generiques?libelle={nom}`   | 5K req/s  | 36K req/s  | +620%        |
| `/v1/medicaments?search={query}` | 1K req/s  | 6.1K req/s | +510%        |
| `/health`                        | 30K req/s | 92K req/s  | +207%        |

**Memory** : 55-80MB avec 67.5MB moyen

### Améliorations spécifiques

- **Lookups O(1) (CIS, CIP)** : Amélioration significative du débit
- **Recherches regex** : Performance accrue grâce aux chaînes pré-normalisées
- **Stabilité maintenue** : Aucune régression sur les endpoints existants

## Architecture Mémoire

Pour les détails de l'architecture mémoire, voir [Architecture du système - Architecture Mémoire](ARCHITECTURE.md#architecture-mémoire).

## Interprétation des Résultats

### Métriques clés

- **Reqs/sec** : Nombre de requêtes par seconde
- **Latence** : Temps moyen par opération
- **Mémoire/op** : Mémoire allouée par opération
- **Allocs/op** : Nombre d'allocations mémoire par opération

### Notes importantes

- Les benchmarks v1 mesurent le temps de sérialisation uniquement (sans réseau)
- L'export complet prend ~1.26ms pour sérialiser 15,811 médicaments
- Le transfert réseau prend plusieurs secondes pour ~20MB de données
- Les tests de production incluent l'overhead HTTP complet (middleware, logging, sérialisation, réseau)

### Limites de recherche et protection contre l'abus

Les endpoints de recherche v1 ont des limites de résultats pour prévenir l'abus :
- **Médicaments** : Maximum 250 résultats par recherche
- **Génériques** : Maximum 100 résultats par recherche

Lorsqu'une recherche dépasse ces limites, l'API retourne **HTTP 400 Bad Request** avec un message guidant vers `/v1/medicaments/export`.

**Raisonnement :**
- Empêche de télécharger l'ensemble du dataset via des recherches larges multiples
- Force l'utilisation appropriée du endpoint `/export` pour le dataset complet (~20MB)
- Protège contre les abus de rate limiting (1000 tokens, 3/sec recharge)

## CI/CD

### Tests benchmarks

Les benchmarks sont exécutés dans le pipeline CI/CD avec :

- **Tests benchmarks non-bloquants** : Ne font pas échouer le pipeline
- **Tolérance de 25%** : Pour tenir compte des variations d'environnement
- **Vérification automatique** : Alertes sur les régressions significatives

### Exécuter localement avant push

```bash
# Test complet
go test -race -v ./...

# Vérifier couverture
go test -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total

# Exécuter les benchmarks
go test ./handlers -bench=. -benchmem -v
```

## Profiling et Débogage

### Profiling avec pprof

Quand `ENV=dev`, le serveur de profiling est disponible sur le port 6060 :

```bash
# CPU profiling (30 secondes)
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30

# Heap profiling
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/goroutine

# Allocation profiling
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/allocs
```

### Profiling de benchmark

```bash
# Générer un profil CPU depuis un benchmark
go test -bench=BenchmarkMedicamentByCIS -cpuprofile=cpu.prof -run=^$ ./handlers

# Analyser le profil
go tool pprof -http=:8080 cpu.prof
```

## Bonnes Pratiques de Performance

### Éviter les allocations inutiles

- Réutiliser les buffers quand possible
- Pré-allouer les slices avec capacity connue
- Éviter les concaténations de chaînes dans les boucles

### Utiliser les maps pour les lookups O(1)

- Maps pour les lookups fréquents (CIS, CIP, group ID)
- Éviter les itérations linéaires sur les slices grandes

### Profiler avant d'optimiser

- Mesurer avec pprof pour identifier les goulots d'étranglement
- Utiliser des benchmarks pour valider les améliorations
- Ne pas optimiser prématurément

### Gérer la concurrence correctement

- Utiliser `sync.RWMutex` pour les accès partagés
- Utiliser `atomic.Value` pour les swaps de données
- Éviter les race conditions avec `go test -race`
