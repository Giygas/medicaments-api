# Guide de Développement

## Prérequis

- **Go 1.26+** avec support des modules
- **2GB RAM** recommandé pour le développement
- **Connexion internet** pour les mises à jour BDPM

## Démarrage Rapide

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
go run .
```

## Configuration

### Variables d'environnement

**Configuration serveur :**

```bash
ADDRESS=127.0.0.1            # Adresse d'écoute (défaut: localhost)
PORT=8000                       # Port du serveur
ENV=dev                         # Environnement (dev/production)
ALLOW_DIRECT_ACCESS=false       # Docker: true (autorise accès direct par IP)
```

_Pour une vue d'ensemble de l'architecture et du rôle de chaque composant, voir [Architecture du système](ARCHITECTURE.md)._

**Configuration logging :**

```bash
LOG_LEVEL=info                   # Niveau de log console (debug/info/warn/error)
                                 # Les fichiers capturent toujours DEBUG
LOG_RETENTION_WEEKS=4            # Nombre de semaines de conservation (1-52)
MAX_LOG_FILE_SIZE=104857600      # Taille max avant rotation (100MB)
```

**Limites optionnelles :**

```bash
MAX_REQUEST_BODY=1048576          # 1MB max corps de requête
MAX_HEADER_SIZE=1048576           # 1MB max taille headers
```

### Serveur de développement

- **Local (Go native)** : http://localhost:8000
- **Local (Docker)** : http://localhost:8030
- **Profiling pprof** : http://localhost:6060 (quand ENV=dev)
- **Documentation interactive** : http://localhost:8000/docs ou http://localhost:8030/docs (Docker)
- **Health endpoint** : http://localhost:8000/health ou http://localhost:8030/health (Docker)
- **Observabilité (Docker)** : Grafana http://localhost:3000, Prometheus http://localhost:9090
  - Géré via le submodule `observability/` (voir [DOCKER.md](../DOCKER.md))

## Commandes de Build

### Build standard

```bash
go build -o medicaments-api .
```

### Cross-platform build

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o medicaments-api-linux .

# Windows
GOOS=windows GOARCH=amd64 go build -o medicaments-api.exe .

# macOS
GOOS=darwin GOARCH=amd64 go build -o medicaments-api-darwin .
GOOS=darwin GOARCH=arm64 go build -o medicaments-api-darwin-arm64 .
```

_Pour plus d'informations sur les optimisations de performance et les benchmarks, voir [Performance et benchmarks](PERFORMANCE.md)._

### Docker Staging (Optionnel)

Pour un environnement de staging complet avec monitoring via submodule d'observabilité :

```bash
# Prérequis : Docker Engine 20.10+ ou Docker Desktop 4.0+

# Setup secrets (Grafana password)
make setup-secrets

# Initialiser le submodule d'observabilité (première fois seulement)
make obs-init

# Démarrer tous les services (API + observability)
make up

# Accéder à l'API
curl http://localhost:8030/health

# Accéder à Grafana (monitoring)
open http://localhost:3000  # Identifiants configurés via make setup-secrets

# Accéder à Prometheus (UI - métriques scrapées par Alloy)
open http://localhost:9090  # UI Prometheus - chercher http_request_total

# Voir les logs de l'application
make logs

# Voir les logs de la stack d'observabilité
make obs-logs

# Arrêter
make down
```

**Ports mappés :**

- API: 8030 (host) → 8000 (container)
- Grafana: 3000 (host) → 3000 (container)
- Prometheus: 9090 (host) → 9090 (container)

**Commandes d'observabilité :**

| Commande          | Description                                         |
| ----------------- | --------------------------------------------------- |
| `make obs-init`   | Initialiser le submodule (première fois)            |
| `make obs-up`     | Démarrer la stack d'observabilité                   |
| `make obs-down`   | Arrêter la stack d'observabilité                    |
| `make obs-logs`   | Voir les logs de la stack d'observabilité           |
| `make obs-status` | Vérifier le statut de la stack d'observabilité      |
| `make obs-update` | Mettre à jour le submodule vers la dernière version |

_Pour la documentation complète Docker, voir [DOCKER.md](../DOCKER.md)_

### Cross-platform build

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o medicaments-api-linux .

# Windows
GOOS=windows GOARCH=amd64 go build -o medicaments-api.exe .

# macOS
GOOS=darwin GOARCH=amd64 go build -o medicaments-api-darwin .
GOOS=darwin GOARCH=arm64 go build -o medicaments-api-darwin-arm64 .
```

## Tests

_Pour des stratégies de tests détaillées et les best practices, consultez le [Guide de tests](TESTING.md)._

### Tests rapides

**Exécuter tous les tests :**

```bash
go test -v
```

**Tests unitaires uniquement :**

```bash
go test -short -v
```

**Test spécifique :**

```bash
go test -run TestName -v
```

**Tests avec détection de race :**

```bash
go test -race -v
```

**Smoke tests :**

```bash
go test ./tests -run TestSmoke -v
```

### Smoke Tests

Les smoke tests sont des tests rapides de validation (< 2 secondes) qui vérifient que l'application démarre correctement et que les endpoints critiques fonctionnent.

#### Ce qui est testé

Les smoke tests valident :

1. **Démarrage de l'application** : Initialisation du logger et du conteneur de données
2. **Endpoint de santé** (`/health`) : Vérifie que le status est "healthy"
3. **Endpoint d'export** (`/v1/medicaments/export`) : Vérifie que les données sont sérialisées correctement
4. **Réponse 200 OK** : Confirme que tous les endpoints renvoient le code HTTP attendu

#### Utilisation recommandée

- **CI/CD rapide** : Exécutés comme validation avant les tests d'intégration plus longs
- **Développement** : Vérifier rapidement que les changements n'ont rien cassé
- **Docker** : Valider que l'application démarre correctement dans un conteneur

### Tests v1

**Médicaments :**

```bash
go test ./handlers -run TestServeMedicamentsV1 -v
```

**Présentations :**

```bash
go test ./handlers -run TestServePresentationsV1 -v
```

**Génériques :**

```bash
go test ./handlers -run TestServeGeneriquesV1 -v
```

**Diagnostics :**

```bash
go test ./handlers -run TestServeDiagnosticsV1 -v
```

### Tests d'intégration

**Pipeline complet de parsing :**

```bash
go test -run TestIntegrationFullDataParsingPipeline -v
```

**Mises à jour concurrentes :**

```bash
go test -run TestIntegrationConcurrentUpdates -v
```

**Utilisation mémoire :**

```bash
go test -run TestIntegrationMemoryUsage -v
```

### Tests du Parser

**Parser unitaires :**

```bash
go test ./medicamentsparser -v
```

**Couverture parser :**

```bash
go test ./medicamentsparser -coverprofile=parser_coverage.out
```

### Couverture

**Générer rapport de couverture :**

```bash
go test -coverprofile=coverage.out -v
```

**Générer HTML de couverture :**

```bash
go tool cover -html=coverage.out -o coverage.html
```

**Vérifier le pourcentage de couverture :**

```bash
go tool cover -func=coverage.out
```

## Benchmarks

### Exécuter tous les benchmarks handlers

```bash
go test ./handlers -bench=. -benchmem -v
```

### Exécuter tous les benchmarks tests complets

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

## Linting

### Formatage du code

```bash
# Formater tous les fichiers
gofmt -w .

# Vérifier si des fichiers ne sont pas formatés
gofmt -d .
```

### Analyse statique

```bash
# Vérifier les constructions suspectes
go vet ./...
```

### Linter approfondi (optionnel)

```bash
golangci-lint run
```

### Outils disponibles

- **go vet** : Vérifie les constructions suspectes, détecte le code inaccessible et les erreurs logiques
- **gofmt** : Formatage automatique du code Go pour standardisation
- **golangci-lint** : Linter plus approfondie (optionnel, à installer séparément)

## Monitoring en Développement

### Logs de développement

Les fichiers de logs sont stockés dans le dossier `logs/` :

```bash
# Consulter les logs actuels
tail -f logs/app-$(date +%Y-W%V).log

# Vérifier la rotation des logs
ls -la logs/
```

### Profiling avec pprof

Quand `ENV=dev`, le serveur de profiling est disponible sur le port 6060 :

```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# Heap profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Allocation profiling
go tool pprof http://localhost:6060/debug/pprof/allocs
```

## Dépannage

### Vérifier la configuration

```bash
# Vérifier que .env existe
ls -la .env

# Vérifier les variables d'environnement
cat .env
```

### Consulter les logs

```bash
# Vérifier les erreurs récentes
grep -i "error\|warning" logs/app-*.log

# Vérifier les connexions BDPM
grep -i "download\|parse" logs/app-*.log
```

### Vérifier la connexion internet

```bash
# Tester la connexion à BDPM
curl -I https://base-donnees-publique.medicaments.gouv.fr
```

### Problèmes courants

**Le serveur ne démarre pas :**

- Vérifier que le port 8000 n'est pas déjà utilisé : `lsof -i :8000`
- Vérifier la configuration `.env`
- Consulter les logs dans `logs/`

**Les données ne sont pas mises à jour :**

- Vérifier la connexion internet
- Consulter les logs pour les erreurs de téléchargement
- Vérifier que l'URL BDPM est accessible

**Tests échouent :**

- Exécuter `go mod tidy` pour s'assurer que les dépendances sont à jour
- Vérifier que Go 1.26+ est installé : `go version`
- Exécuter `go test -race -v` pour détecter les race conditions

**Résultats de tests trop lénts :**

- Utiliser `go test -short` pour sauter les tests d'intégration lents
- Exécuter des tests spécifiques au lieu de tous les tests

**Les recherches retournent HTTP 429 :**

- Les recherches larges (> 250 médicaments ou > 100 génériques) renvoient une erreur 429
- Utilisez `/v1/medicaments/export` pour obtenir le dataset complet
- Réduisez la spécificité de la recherche (ex: "paracetamol 500" au lieu de "a")

## Bonnes Pratiques

### Organisation du code

- Suivre les conventions Go standard
- Séparer clairement les responsabilités
- Utiliser les interfaces définies dans `interfaces/`

### Tests

- Écrire des tests pour chaque nouvelle fonctionnalité
- Maintenir une couverture de code ≥ 75%
- Exécuter `go test -race` avant de commiter

**Pour les bonnes pratiques de tests détaillées, voir [Guide de tests - Bonnes Pratiques](TESTING.md#bonnes-pratiques).**

### Performance

- Profiler le code avec pprof pour identifier les goulots d'étranglement
- Utiliser les benchmarks pour valider les améliorations

**Pour les bonnes pratiques de performance détaillées, voir [Performance et benchmarks - Bonnes Pratiques de Performance](PERFORMANCE.md#bonnes-pratiques-de-performance).**

### Documentation

- Mettre à jour les commentaires GoDoc
- Mettre à jour l'OpenAPI spec lors de changements d'API
- Documenter les fonctions exportées
- Maintenir la documentation à jour avec le code

### Gestion des erreurs 429

- Les clients doivent gérer les réponses HTTP 429 de manière gracieuse
- Guider les utilisateurs vers `/v1/medicaments/export` pour les recherches larges
- Afficher un message explicite lorsque la limite est atteinte

## Workflow de Développement Recommandé

1. **Créer une branche** : `git checkout -b feature/nouvelle-fonctionnalite`
2. **Développer** : Écrire le code avec des tests
3. **Tester** : `go test -v ./...` et `go test -race ./...`
4. **Linting** : `gofmt -w .` et `go vet ./...`
5. **Commit** : `git commit -am` (message descriptif)
6. **Push** : `git push origin feature/nouvelle-fonctionnalite`
7. **Pull Request** : Ouvrir une PR pour review
