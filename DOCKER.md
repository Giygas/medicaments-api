# Guide de Configuration Docker

**Guide complet pour exécuter medicaments-api dans Docker**

---

**🇫🇷 Français** | [🇬🇧 English](DOCKER.en.md)

---

## Table des Matières

- [Démarrage Rapide](#démarrage-rapide)
- [Commandes Essentielles](#commandes-essentielles)
- [Aperçu du Projet](#aperçu-du-projet)
  - [Ce Qui a Été Créé](#ce-qui-a-été-créé)
  - [Structure du Projet](#structure-du-projet)
  - [Configuration](#configuration)
- [Commandes Docker Compose](#commandes-docker-compose)
  - [Build et Exécution](#build-et-exécution)
  - [Voir les Logs](#voir-les-logs)
  - [Gestion des Conteneurs](#gestion-des-conteneurs)
- [Endpoints API](#endpoints-api)
- [Gestion des Données](#gestion-des-données)
- [Dépannage](#dépannage)
- [Utilisation Avancée](#utilisation-avancée)
- [Considérations de Sécurité](#considérations-de-sécurité)
- [Surveillance](#surveillance)
- [Nettoyage](#nettoyage)
- [Différences en Production](#différences-en-production)
- [Intégration CI/CD](#intégration-cicd)
- [Stack d'Observabilité](#stack-dobservabilité)
- [Support](#support)
- [Annexe](#annexe)

---

## Démarrage Rapide

### Prérequis

- Docker Engine 20.10+ ou Docker Desktop 4.0+
- Au moins 1Go d'espace disque disponible
- Connexion réseau pour le téléchargement des données BDPM
- Configuration des secrets : Exécuter `make setup-secrets` (crée `secrets/grafana_password.txt`)

### Configuration des Secrets et Observabilité (Première Étape Requise)

Avant d'exécuter les services Docker, vous devez configurer le secret du mot de passe Grafana et initialiser le submodule d'observabilité :

```bash
# Créer le secret du mot de passe Grafana
make setup-secrets

# Cela demande un mot de passe et crée secrets/grafana_password.txt avec des permissions sécurisées (600)

# Initialiser le submodule d'observabilité (première fois seulement)
make obs-init

# Cela clone le submodule observability-stack depuis GitHub
```

**Pourquoi des Secrets ?**

- Grafana nécessite un mot de passe administrateur pour un accès sécurisé
- Stocker les mots de passe dans des variables d'environnement ou des fichiers de configuration n'est pas sécurisé
- Les Docker secrets fournissent un moyen sécurisé de gérer les données sensibles
- Le répertoire `secrets/` est exclu du contrôle de version (`.gitignore`)

**Bonnes Pratiques de Sécurité :**

- ✅ Utiliser des mots de passe forts (minimum 12 caractères, majuscules/minuscules, chiffres, symboles)
- ✅ Ne jamais committer de secrets dans le contrôle de version
- ✅ Définir des permissions de fichiers restrictives (600)
- ❌ Ne pas réutiliser les mots de passe sur plusieurs services

### Démarrage Immédiat

```bash
# Docker Compose (recommandé)
docker compose up -d

# Voir les logs
docker compose logs -f

# Vérifier la santé
curl http://localhost:8030/health

# Arrêter
docker compose down
```

### Ce Qui se Passe au Premier Lancement

1. **Docker construit l'image** (~1-2 minutes)
2. **Le conteneur medicaments-api démarre** en tant qu'utilisateur non-root (UID 65534/nobody)
3. **Le conteneur grafana-alloy démarre** pour collecter les logs et métriques
4. **Le téléchargement des données BDPM** depuis les sources externes (~10-30 secondes)
5. **Le serveur HTTP démarre** sur le port 8000
6. **Le health check commence** après une période de démarrage de 10 secondes
7. **L'API est prête** sur http://localhost:8030
8. **La stack d'observabilité** (Loki, Prometheus, Grafana) est accessible via le submodule

---

## Commandes Essentielles

### 🚀 Démarrage & Arrêt

```bash
make up                          # Démarrer tous les services (API + observabilité)
make down                        # Arrêter tous les services
make restart                     # Redémarrer tous les services
```

### 📋 Logs

```bash
make logs                        # Suivre les logs de l'application en temps réel
make obs-logs                   # Suivre les logs de la stack d'observabilité
docker compose logs --tail=100   # 100 dernières lignes de tous les services
```

### 🔍 Statut & Santé

```bash
make ps                          # Statut de tous les conteneurs
make obs-status                  # Statut de la stack d'observabilité
curl http://localhost:8030/health # Vérification de santé
docker stats medicaments-api grafana-alloy # Utilisation des ressources
```

### 🛠️ Build & Rebuild

```bash
docker compose build             # Construire l'image
docker compose up -d --build     # Rebuild & démarrer
docker compose build --no-cache  # Build propre (sans cache)
```

### 🏗️ Builds Multi-Architecture

```bash
# Build pour l'architecture hôte (auto-détecté)
make build

# Forcer une architecture spécifique
make build-amd64
make build-arm64

# Démarrer les services
make up

# Voir toutes les commandes disponibles
make help
```

**Docker Compose (auto-détecte la plateforme) :**

```bash
docker compose up -d    # Construit pour votre plateforme native
docker compose build      # Construit pour votre plateforme native
```

**Plateformes Supportées :**

| Architecture | Description      | Plateformes Cibles                                     |
| ------------ | ---------------- | ------------------------------------------------------ |
| **amd64**    | Intel/AMD x86_64 | Serveurs Intel/AMD, instances cloud, Mac Intel         |
| **arm64**    | ARM 64-bit       | Apple Silicon (M1/M2/M3), Raspberry Pi 4, AWS Graviton |

**Note :** Utilisez l'option `--load` pour rendre l'image disponible localement. Sans cela, l'image existe uniquement dans le cache BuildKit.

---

## Aperçu du Projet

### Ce Qui a Été Créé

Les fichiers suivants ont été ajoutés pour configurer votre environnement Docker :

#### 1. **Dockerfile**

Build Docker multi-étapes optimisé pour la production :

- **Étape 1 - Builder** : `golang:1.26-alpine`
  - Utilise `syntax=docker/dockerfile:1` pour le support buildkit
  - Montages de cache pour les packages Go et le cache de build (rebuilds plus rapides)
  - Support multi-architecture via la variable BuildKit `$TARGETARCH`
  - Copie la documentation HTML depuis `/build/html`
- **Étape 2 - Runtime** : `scratch` (~8-10MB image finale, surface d'attaque minimale)
- **Sécurité** : Utilisateur non-root (UID 65534/nobody)
- **Health Check** : Intégré via l'instruction HEALTHCHECK avec la sous-commande healthcheck
- **Fichiers** : Copie le binaire, les certificats CA et la documentation HTML
- **Dépendances de Build** : Seulement `ca-certificates`

#### 2. **docker-compose.yml**

Orchestration Docker Compose (2 services principaux) :

- **Services** :
  - `medicaments-api` : Application principale
  - `grafana-alloy` : Collecteur de logs et métriques
- **Mapping de Ports** : 8030 (hôte) → 8000 (conteneur) pour API, 12345 pour Alloy metrics
- **Environnement** : Variables depuis `.env.docker`
- **Logs** : Persistants via un volume nommé (`logs_data:/app/logs`)
- **Sécurité** : Système de fichiers en lecture seule, no-new-privileges, tmpfs pour /app/files
- **Ressources** :
  - medicaments-api : limites 512MB/0.5CPU, réservations 256MB/0.25CPU
  - grafana-alloy : limites 256MB/0.5CPU, réservations 128MB/0.1CPU
- **Health Check** : Délégué au Dockerfile (intervalle 30s, timeout 5s, période de démarrage 10s, 3 tentatives)
- **Restart** : Politique `unless-stopped`
- **Réseau** : Utilise le réseau externe `obs-network` (créé par le submodule d'observabilité)
- **Labels de Conteneur** : Métadonnées pour l'identification et la gestion

#### 3. **observability/** (Submodule)

Submodule Git pour la stack d'observabilité :

- **Source** : https://github.com/Giygas/observability-stack.git
- **Services** :
  - `loki` : Agrégation et stockage des logs (30 jours)
  - `prometheus` : Stockage et interrogation des métriques (30 jours)
  - `grafana` : Visualisation et tableaux de bord
- **Réseau** : Crée le réseau externe `obs-network` partagé avec l'application
- **Secrets** : Gestion du mot de passe Grafana via `secrets/grafana_password.txt`
- **Configuration** : Fichiers de config dans `configs/` (loki, prometheus, grafana, dashboards)
- **Ressources** :
  - loki : limites 512MB/1.0CPU, réservations 256MB/0.2CPU
  - prometheus : limites 1G/1.0CPU, réservations 512MB/0.3CPU
  - grafana : limites 512MB/0.5CPU, réservations 256MB/0.1CPU

#### 4. **.dockerignore**

Optimise le contexte de build Docker :

- Exclut : logs, git, vendor, fichiers de test, \*.md (sauf README.md)
- Garde : code source et documentation HTML
- Réduit : temps de build et taille de l'image

#### 5. **.env.docker**

Configuration de l'environnement Docker :
| Variable | Valeur | Description |
|----------|-------|-------------|
| `ADDRESS` | `0.0.0.0` | Écouter sur toutes les interfaces dans le conteneur |
| `PORT` | `8000` | Port à l'intérieur du conteneur |
| `ENV` | `production` | Mode d'environnement |
| `ALLOW_DIRECT_ACCESS` | `true` | Autoriser la liaison à toutes les interfaces (Docker uniquement) |
| `LOG_LEVEL` | `info` | Niveau de logging (debug/info/warn/error) |
| `LOG_RETENTION_WEEKS` | `2` | Garder les logs pendant 2 semaines |
| `MAX_LOG_FILE_SIZE` | `52428800` | Rotation à 50MB |
| `MAX_REQUEST_BODY` | `2097152` | Corps de requête max 2MB |
| `MAX_HEADER_SIZE` | `2097152` | Taille d'en-tête max 2MB |
| `APP_VERSION` | `1.2.0` | Version de l'application |
| `ALLOY_CONFIG` | `config.alloy` | Configuration Alloy (local ou remote) |
| `PROMETHEUS_URL` | - | URL Prometheus distante (mode remote seulement) |
| `LOKI_URL` | - | URL Loki distante (mode remote seulement) |

#### 6. **Makefile**

Commandes de build et de développement unifiées :

- Auto-détecte l'architecture hôte (amd64 ou arm64)
- Fournit une interface unifiée pour Docker, les tests et le benchmarking
- Supporte le ciblage explicite d'architecture : `make build-amd64` ou `make build-arm64`
- **Observabilité** : Commandes pour gérer le submodule d'observabilité
- Opérations courantes : `make build`, `make up`, `make down`, `make logs`, `make test`, `make bench`
- Voir toutes les commandes : `make help`

#### 7. **.gitignore** (mis à jour)

Ajouté des exclusions complètes incluant :

- `.env.docker` et autres fichiers d'environnement
- Répertoire `secrets/` (gitignoré)
- Fichiers standard Git, CI/CD, IDE et OS
- Artefacts de test et fichiers de build

### Structure du Projet

```
medicaments-api/
├── Dockerfile              # Build Docker multi-étapes
├── docker-compose.yml      # Orchestration Docker Compose (2 services : medicaments-api + grafana-alloy)
├── .dockerignore          # Fichiers exclus du contexte de build
├── .env.docker             # Variables d'environnement Docker
├── Makefile               # Commandes de build et de développement unifiées
├── .gitmodules            # Configuration des submodules Git
├── logs/                  # Répertoire des logs persistants
├── html/                  # Fichiers de documentation (servis par l'API)
├── secrets/              # Docker secrets (gitignoré)
│   └── grafana_password.txt
├── configs/              # Configurations locales
│   └── alloy/            # Configurations Alloy (local & remote)
│       ├── config.alloy          # Mode local (défaut)
│       └── config.remote.alloy  # Mode remote (tunnel)
└── observability/         # Submodule Git pour la stack d'observabilité
    ├── docker-compose.yml         # Orchestration de la stack (loki + prometheus + grafana)
    ├── configs/                  # Configurations de la stack
    │   ├── alloy/
    │   ├── loki/
    │   ├── prometheus/
    │   └── grafana/
    ├── secrets/                 # Secrets de la stack (gitignoré)
    │   └── grafana_password.txt
    └── docs/                   # Documentation de la stack
        ├── README.md
        └── CONTRIBUTING.md
```

### Configuration

#### Mapping de Ports

- **Port Hôte** : 8030
- **Port Conteneur** : 8000

Accédez à l'API sur : `http://localhost:8030`

#### Limites de Ressources

Le conteneur de staging a les limites suivantes :

- **CPU** : 0.5 cœurs max, 0.25 cœurs réservés
- **Mémoire** : 512MB max, 256MB réservés

---

## Commandes Docker Compose

### Observabilité (Submodule)

Le submodule d'observabilité nécessite une initialisation avant la première utilisation :

```bash
# Initialiser le submodule (première fois seulement)
make obs-init

# Démarrer uniquement la stack d'observabilité
make obs-up

# Arrêter la stack d'observabilité
make obs-down

# Voir les logs de la stack d'observabilité
make obs-logs

# Vérifier le statut de la stack d'observabilité
make obs-status

# Mettre à jour le submodule vers la dernière version
make obs-update
```

### Build et Exécution

```bash
# Construire l'image Docker
make build

# Démarrer tous les services (API + observabilité)
make up

# Démarrer avec les logs
docker compose up

# Rebuild et démarrer
docker compose up -d --build
```

### Voir les Logs

```bash
# Suivre les logs en temps réel
docker compose logs -f

# Voir les logs pour les 100 dernières lignes
docker compose logs --tail=100

# Voir les logs avec horodatage
docker compose logs -f -t

# Voir les logs persistants depuis le volume nommé
docker compose exec medicaments-api ls -la /app/logs/
docker compose exec medicaments-api tail -f /app/logs/app-*.log
```

### Gestion des Conteneurs

```bash
# Vérifier le statut de tous les conteneurs
make ps

# Voir les informations détaillées du conteneur
docker inspect medicaments-api

# Voir l'utilisation des ressources
docker stats medicaments-api grafana-alloy

# Redémarrer tous les conteneurs
make restart

# Arrêter tous les conteneurs
make down

# Arrêter et supprimer les conteneurs et volumes
docker compose down -v

# Supprimer les conteneurs, volumes et images
docker compose down -v --rmi all
```

---

## Endpoints API

Accédez à tous les endpoints via `http://localhost:8030`

### Endpoints V1 (Recommandés)

```bash
# Vérification de santé
curl http://localhost:8030/health

# Obtenir tous les médicaments (paginés)
curl http://localhost:8030/v1/medicaments?page=1

# Rechercher par nom
curl http://localhost:8030/v1/medicaments?search=paracetamol

# Recherche par CIS
curl http://localhost:8030/v1/medicaments?cis=61504672

# Recherche par CIP
curl http://localhost:8030/v1/medicaments?cip=3400936403114

# Obtenir les génériques par libellé
curl http://localhost:8030/v1/generiques?libelle=paracetamol

# Obtenir les génériques par ID de groupe
curl http://localhost:8030/v1/generiques?group=1234

# Obtenir les présentations par CIP
curl http://localhost:8030/v1/presentations?cip=3400936403114

# Exporter toutes les données
curl http://localhost:8030/v1/medicaments/export
```

### Documentation

```bash
# Interface Swagger interactive
open http://localhost:8030/docs

# Spécification OpenAPI
curl http://localhost:8030/docs/openapi.yaml
```

---

## Gestion des Données

### Téléchargement des Données

L'application télécharge automatiquement les données BDPM depuis les sources externes :

- **Téléchargement Initial** : Se produit au démarrage du conteneur (prend 10-30 secondes)
- **Mises à Jour Automatiques** : Planifiées deux fois par jour (6h et 18h)
- **Zéro Downtime** : Les mises à jour n'interrompent pas l'accès à l'API

Surveiller le téléchargement des données :

```bash
# Regarder les logs pendant le démarrage
docker compose logs -f

# Vérifier le statut des données via l'endpoint de santé
curl http://localhost:8030/health | jq '.data'
```

### Vérifications de Santé

Le conteneur inclut un health check utilisant l'endpoint `/health` :

- **Intervalle** : 30 secondes
- **Timeout** : 5 secondes
- **Tentatives** : 3
- **Période de Démarrage** : 10 secondes

Vérifier le statut de santé :

```bash
# Vérifier le statut de santé Docker
docker compose ps

# Vérifier l'endpoint de santé
curl http://localhost:8030/health

# Exemple de réponse santé
{
  "status": "healthy",
  "last_update": "2025-02-08T12:00:00+01:00",
  "data_age_hours": 0.5,
  "medicaments": 15822,
  "generiques": 1645,
  "is_updating": false
}
```

### Diagnostics

Pour des métriques système détaillées et des informations sur l'intégrité des données, utilisez l'endpoint `/v1/diagnostics` :

**Ce qu'il retourne :**

- Métriques système : uptime, goroutines, utilisation mémoire
- Âge des données et prochaine mise à jour planifiée
- Contrôles d'intégrité des données : enregistrements orphelins, associations manquantes
- Codes CIS d'échantillon pour le dépannage

**Exemple d'utilisation :**

```bash
# Obtenir les diagnostics complets
curl http://localhost:8030/v1/diagnostics | jq

# Métriques système uniquement
curl http://localhost:8030/v1/diagnostics | jq '.system'

# Utilisation mémoire
curl http://localhost:8030/v1/diagnostics | jq '.system.memory'

# Résumé de l'intégrité des données
curl http://localhost:8030/v1/diagnostics | jq '.data_integrity'

# Uptime
curl http://localhost:8030/v1/diagnostics | jq '.uptime_seconds'
```

**Exemple de réponse :**

```json
{
  "timestamp": "2025-02-08T13:00:00+01:00",
  "uptime_seconds": 3600,
  "next_update": "2025-02-08T18:00:00+01:00",
  "data_age_hours": 0.3,
  "system": {
    "goroutines": 16,
    "memory": {
      "alloc_mb": 45,
      "num_gc": 20,
      "sys_mb": 65
    }
  },
  "data_integrity": {
    "medicaments_without_conditions": {"count": 3368, "sample_cis": [...]},
    "medicaments_without_generiques": {"count": 7714, "sample_cis": [...]},
    "medicaments_without_presentations": {"count": 1267, "sample_cis": [...]},
    "medicaments_without_compositions": {"count": 2, "sample_cis": [...]},
    "generique_only_cis": {"count": 2440, "sample_cis": [...]},
    "presentations_with_orphaned_cis": {"count": 6, "sample_cip": [...]}
  }
}
```

**Contrôles d'Intégrité des Données :**

| Contrôle                            | Description                                           | Champ d'Échantillon |
| ----------------------------------- | ----------------------------------------------------- | ------------------- |
| `medicaments_without_conditions`    | Médicaments sans données de condition                 | `sample_cis`        |
| `medicaments_without_generiques`    | Médicaments non dans les groupes de génériques        | `sample_cis`        |
| `medicaments_without_presentations` | Médicaments sans données de présentation              | `sample_cis`        |
| `medicaments_without_compositions`  | Médicaments sans données de composition               | `sample_cis`        |
| `generique_only_cis`                | Codes CIS uniquement dans les groupes de génériques   | `sample_cis`        |
| `presentations_with_orphaned_cis`   | Présentations référençant des médicaments inexistants | `sample_cip`        |

---

## Dépannage

### Le Conteneur Ne Démarre Pas

```bash
# Vérifier les erreurs
docker compose logs

# Vérifier que le port 8030 n'est pas utilisé
lsof -i :8030

# Vérifier l'espace disque
df -h

# Rebuild à partir de zéro
docker compose down
docker compose build --no-cache
docker compose up -d
```

### Échec du Health Check

```bash
# Vérifier que le conteneur est en cours d'exécution
docker compose ps

# Voir les logs du health check
docker inspect medicaments-api | jq '.[0].State.Health'

# Tester manuellement l'endpoint de santé
docker compose exec medicaments-api wget -O- http://localhost:8000/health

# Vérifier les erreurs de téléchargement de données
docker compose logs | grep -i error
```

### Problèmes de Téléchargement de Données

```bash
# Vérifier la connectivité réseau
docker compose exec medicaments-api wget -O- https://base-donnees-publique.medicaments.gouv.fr

# Voir les logs de téléchargement
docker compose logs | grep -i download

# Redémarrer pour déclencher le téléchargement
docker compose restart
```

### Les Logs Ne Sont Pas Persistants

```bash
# Vérifier le montage de volume
docker inspect medicaments-api | jq '.[0].Mounts'

# Vérifier les permissions du répertoire de logs
ls -la logs/

# Vérifier les logs dans le conteneur
docker compose exec medicaments-api ls -la /app/logs/
```

### Utilisation Mémoire Élevée

```bash
# Vérifier l'utilisation mémoire actuelle
docker stats medicaments-api

# Voir les métriques mémoire
curl http://localhost:8030/v1/diagnostics | jq '.system.memory'

# Redémarrer pour vider la mémoire
docker compose restart
```

### Conflits de Ports

Si le port 8030 est déjà utilisé :

```bash
# Changer le port dans docker-compose.yml
ports:
  - "8031:8000"  # Utiliser un port hôte différent

# Ou arrêter le service en conflit
lsof -i :8030
```

### Fichier de Secrets Manquant

Si vous rencontrez cette erreur :

```
ERROR: for grafana  Cannot create container for service grafana:
stat /path/to/secrets/grafana_password.txt: no such file or directory
```

**Solution :**

```bash
# Option 1 : Utiliser Make (recommandé)
make setup-secrets

# Option 2 : Créer manuellement
mkdir -p secrets
echo "your-secure-password" > secrets/grafana_password.txt
chmod 600 secrets/grafana_password.txt

# Option 3 : Valider les secrets existants
make validate-secrets
```

**Vérifier que les secrets fonctionnent :**

```bash
# Vérifier que le fichier existe avec les permissions correctes
ls -la secrets/grafana_password.txt

# Attendu : -rw------- 1 user group date secrets/grafana_password.txt
```

### Problèmes d'Observabilité

Pour un dépannage détaillé des problèmes de Grafana, Loki, Prometheus et Alloy, voir [OBSERVABILITY.md](OBSERVABILITY.md#troubleshooting).

**Commandes utiles :**

```bash
# Vérifier le statut du submodule
git submodule status

# Mettre à jour le submodule
make obs-update

# Réinitialiser le submodule en cas de problème
rm -rf .git/modules/observability
git submodule deinit -f observability
git submodule update --init --recursive observability
```

---

## Utilisation Avancée

### Variables d'Environnement Personnalisées

Créez un fichier `.env` personnalisé :

```bash
# Surcharge n'importe quelle variable d'environnement
LOG_LEVEL=debug
LOG_RETENTION_WEEKS=1
```

Ensuite utilisez-le :

```bash
docker compose --env-file .env.custom up -d
```

### Exécution de Plusieurs Instances

```bash
# Créer plusieurs fichiers compose
cp docker-compose.yml docker-compose.staging.yml

# Éditer le mapping de ports dans le nouveau fichier
# ports:
#   - "8031:8000"

# Démarrer les deux instances
docker compose -f docker-compose.yml up -d
docker compose -f docker-compose.staging.yml up -d
```

### Débogage

```bash
# Exécuter avec le mode debug
# Éditer .env.docker : LOG_LEVEL=debug
docker compose restart

# Voir les logs en temps réel
docker compose logs -f

# Entrer dans le shell du conteneur (l'image scratch n'a pas de shell - utiliser uniquement les logs)
# docker compose exec medicaments-api sh  # Non disponible

# Vérifier les processus (l'image scratch n'a pas de ps - utiliser l'endpoint de santé)
# docker compose exec medicaments-api ps aux  # Non disponible

# Surveiller les changements de fichiers
docker compose exec medicaments-api ls -la /app/logs/
```

### Tests de Performance

```bash
# Installer hey (outil de charge)
go install github.com/rakyll/hey@latest

# Tester l'endpoint de santé
hey -n 1000 -c 10 http://localhost:8030/health

# Tester la recherche de médicament
hey -n 1000 -c 10 http://localhost:8030/v1/medicaments?cis=61504672

# Tester l'endpoint de recherche
hey -n 100 -c 5 http://localhost:8030/v1/medicaments?search=paracetamol
```

---

## Considérations de Sécurité

### Utilisateur Non-Root

Le conteneur s'exécute en tant qu'utilisateur non-root (`UID 65534` / `nobody`) pour la sécurité :

```bash
# Vérifier l'utilisateur (l'image scratch peut ne pas avoir whoami)
# docker compose exec medicaments-api whoami  # Peut ne pas être disponible

# Vérifier l'ID utilisateur
docker compose exec medicaments-api id
```

### Stratégie d'Exposition des Ports

Pour la sécurité, certains services sont uniquement exposés en interne au réseau Docker :

| Service                   | Niveau d'Exposition | Rationale                                                |
| ------------------------- | ------------------- | -------------------------------------------------------- |
| medicaments-api (API)     | Hôte + Réseau       | Requis pour l'accès API externe                          |
| medicaments-api (metrics) | Réseau uniquement   | Scrapé par Alloy en interne                              |
| loki                      | Réseau uniquement   | Scrapé par Alloy en interne                              |
| grafana-alloy             | Hôte + Réseau       | Endpoint de débogage optionnel                           |
| prometheus                | Hôte + Réseau       | Requis pour l'interface Grafana et les requêtes externes |
| grafana                   | Hôte + Réseau       | Requis pour l'accès aux tableaux de bord                 |

**Avantages de l'exposition interne uniquement :**

- Réduit la surface d'attaque depuis l'accès externe
- Empêche le scraping non autorisé direct des métriques/logs
- Force l'accès via l'interface Grafana contrôlée
- Maintient la fonctionnalité d'observabilité dans le réseau Docker

**Pour accéder aux services internes uniquement pour le débogage :**

```bash
# Accéder à Loki depuis le réseau Docker
docker compose exec loki wget -O- 'http://localhost:3100/loki/api/v1/labels'

# Vérifier les logs dans les conteneurs
docker compose logs loki
docker compose logs grafana-alloy
```

### Isolement Réseau

Le conteneur utilise un réseau bridge personnalisé pour l'isolement :

```bash
# Lister les réseaux
docker network ls

# Inspecter le réseau
docker network inspect medicaments_medicaments-network
```

### Permissions de Volume

Le répertoire des logs appartient à `appuser` :

```bash
# Vérifier les permissions
ls -la logs/

# Corriger les permissions si nécessaire
sudo chown -R 1000:1000 logs/
```

---

## Surveillance

### Métriques de Conteneur

```bash
# Stats en temps réel
docker stats medicaments-api

# Métriques spécifiques
docker stats --no-stream medicaments-api
```

### Métriques d'Application

```bash
# Métriques de santé complètes
curl http://localhost:8030/health | jq

# Utilisation mémoire uniquement
curl -s http://localhost:8030/v1/diagnostics | jq '.system.memory'

# Âge des données
curl -s http://localhost:8030/health | jq '.data_age_hours'
```

### Surveillance des Logs

```bash
# Suivre les logs de l'application
docker compose logs -f

# Surveiller les erreurs
docker compose logs -f | grep -i error

# Surveiller les mises à jour de données
docker compose logs -f | grep -i update

# Compter les entrées de log
docker compose logs | wc -l
```

### Endpoint de Métriques Prometheus

L'application expose ses métriques Prometheus sur le port interne 9090, accessibles uniquement via le réseau Docker. Alloy collecte ces métriques automatiquement.

**Pour voir les métriques :**

1. Via Prometheus UI : http://localhost:9090 → cherchez `http_request_total`, `http_request_duration_seconds`, `http_request_in_flight`
2. Via Grafana : http://localhost:3000 → tableaux de bord préconfigurés
3. Via Alloy (développement) : `docker compose exec grafana-alloy wget -O- http://medicaments-api:9090/metrics`

**Métriques disponibles :**

- `http_request_total` - Total des requêtes HTTP avec les étiquettes méthode, chemin, statut
- `http_request_duration_seconds` - Histogramme de latence des requêtes
- `http_request_in_flight` - Requêtes en cours actuelles

Pour une configuration d'observabilité détaillée avec les tableaux de bord Grafana, les alertes et l'agrégation de logs, voir [OBSERVABILITY.md](OBSERVABILITY.md).

---

## Nettoyage

### Supprimer l'Environnement de Staging

```bash
# Arrêter et supprimer les conteneurs
docker compose down

# Supprimer les logs persistants (optionnel)
rm -rf logs/

# Supprimer les images Docker
docker rmi medicaments_medicaments-api

# Supprimer toutes les ressources inutilisées
docker system prune -a
```

### Nettoyer les Ressources Docker

```bash
# Supprimer les conteneurs arrêtés
docker container prune

# Supprimer les volumes inutilisés
docker volume prune

# Supprimer les images inutilisées
docker image prune

# Supprimer tout (utiliser avec précaution)
docker system prune -a --volumes
```

---

## Différences en Production

| Fonctionnalité            | Staging        | Production          |
| ------------------------- | -------------- | ------------------- |
| **Déploiement**           | Docker Compose | SSH + systemd       |
| **Port**                  | 8030           | 8000 (configurable) |
| **LOG_LEVEL**             | info           | info                |
| **LOG_RETENTION_WEEKS**   | 2              | 4                   |
| **MAX_LOG_FILE_SIZE**     | 50MB           | 100MB               |
| **Limites de Ressources** | 512MB/0.5CPU   | Aucune (systemd)    |
| **Emplacement des Logs**  | `./logs/`      | Logs serveur        |

---

## Intégration CI/CD

Cette configuration Docker peut être intégrée à votre pipeline CI/CD existant :

```bash
# Dans le pipeline CI/CD
docker compose -f docker-compose.yml -f docker-compose.ci.yml up -d

# Exécuter les tests
docker compose exec medicaments-api go test ./...

# Obtenir la couverture
docker compose exec medicaments-api go test -coverprofile=coverage.out ./...

# Nettoyage
docker compose down -v
```

---

## Stack d'Observabilité

Pour la configuration complète de l'observabilité (Grafana, Loki, Prometheus, Alloy), voir [OBSERVABILITY.md](OBSERVABILITY.md).

**Accès rapide :**

- Grafana : http://localhost:3000
- Prometheus : http://localhost:9090
- Identifiants : voir secrets/grafana_password.txt

---

## Support

Pour les problèmes ou questions :

1. Consultez la [section de dépannage](#dépannage) ci-dessus
2. Voir [OBSERVABILITY.md](OBSERVABILITY.md) pour les problèmes spécifiques à l'observabilité
3. Consulter la documentation du submodule d'observabilité : `observability/docs/README.md`
4. Consultez le README.md principal
5. Vérifiez les logs de l'application : `make logs` ou `make obs-logs`
6. Vérifiez le statut de santé : `curl http://localhost:8030/health`
7. Ouvrez une issue sur GitHub

### Observabilité-Stack Submodule

La stack d'observabilité est maintenue séparément dans le repository [Giygas/observability-stack](https://github.com/Giygas/observability-stack).

Pour les questions spécifiques à la stack d'observabilité :

- Documentation : `observability/docs/README.md`
- Contribution : `observability/docs/CONTRIBUTING.md`
- Issues : https://github.com/Giygas/observability-stack/issues

---

## Annexe

### Détails de l'Image Docker

- **Image de Base** : `scratch` (système de fichiers vide, surface d'attaque minimale)
- **Image de Builder** : `golang:1.26-alpine`
- **Taille de l'Image Finale** : ~8-10MB
- **Taille du Binaire** : ~8-10MB (statiquement lié, stripped)
- **Architectures Supportées** : amd64, arm64
- **Outil de Build** : Docker BuildKit avec détection automatique de plateforme ($TARGETOS/$TARGETARCH)

### Emplacements des Fichiers

| Type                     | Emplacement                                             |
| ------------------------ | ------------------------------------------------------- |
| **Binaire**              | `/app/medicaments-api`                                  |
| **Docs HTML**            | `/app/html/`                                            |
| **Logs**                 | `/app/logs/` (monté sur `logs_data`)                    |
| **Config API**           | Variables d'environnement (`.env.docker`)               |
| **Config Alloy**         | `./configs/alloy/config.alloy` ou `config.remote.alloy` |
| **Config Observabilité** | `./observability/configs/` (submodule)                  |

### Processus de Démarrage

1. **Initialisation** (première fois) : Le submodule `observability/` est initialisé via `make obs-init`
2. Le conteneur `medicaments-api` démarre en tant qu'utilisateur non-root (UID 65534/nobody)
3. La stack d'observabilité (`loki`, `prometheus`, `grafana`) démarre via le submodule
4. `grafana-alloy` démarre via docker-compose.yml de l'application, après le healthcheck de medicaments-api
5. L'application charge les variables d'environnement depuis `.env.docker`
6. Le système de logging est initialisé
7. Le conteneur de données et le parser sont créés
8. Le scheduler démarre (mises à jour 6h/18h)
9. Les données BDPM sont téléchargées depuis les sources externes
10. Le serveur HTTP démarre sur le port 8000
11. Le healthcheck Docker passe après une période de démarrage de 10s
12. Grafana Alloy commence à collecter les logs et les métriques depuis `/app/logs/`
13. Loki et Prometheus commencent à recevoir les données via Alloy

### Conseils

- Le conteneur télécharge les données BDPM au premier démarrage (10-30s)
- Le health check passe après une période de démarrage de ~10s
- Les logs persistent même après la suppression du conteneur (montage de volume)
- Utilisez `docker compose exec medicaments-api sh` pour entrer dans le conteneur (si disponible)
- Consultez la [section de dépannage](#dépannage) pour une aide détaillée
