# Guide de l'Observabilité - medicaments-api

**Guide pour configurer et utiliser la stack d'observabilité avec medicaments-api**

---

**🇫🇷 Français** | [🇬🇧 English](OBSERVABILITY.en.md)

---

## Vue d'ensemble

Le setup staging inclut une stack d'observabilité complète avec Grafana, Loki, Prometheus et Alloy pour le monitoring des logs et des métriques.

**Architecture :**

La stack d'observabilité est organisée via un **submodule Git** séparé :

- **docker-compose.yml** (application) : Contient `medicaments-api` et `grafana-alloy`
- **observability/** (submodule) : Contient `loki`, `prometheus`, et `grafana`

Les deux composants sont connectés via le réseau externe `obs-network` créé par le submodule.

**Composants :**

- **Grafana Alloy** : Agent de collecte qui rassemble les logs et les métriques
- **Loki** : Agrégation et stockage des logs
- **Prometheus** : Stockage et interrogation des métriques
- **Grafana** : Visualisation et tableaux de bord

**Avantages :**

- Visualisation centralisée et recherche de logs
- Monitoring en temps réel des métriques
- Alertes sur la santé et la performance des services
- Tableaux de bord préconfigurés pour des insights rapides

## Table des matières

- [Vue d'ensemble](#vue-densemble)
- [Configuration Rapide](#configuration-rapide)
- [Architecture](#architecture)
- [Modes de Configuration](#modes-de-configuration)
  - [Mode Local (Défaut)](#mode-local-défaut)
  - [Mode Remote (Production)](#mode-remote-production)
- [Configuration de l'Application](#configuration-de-lapplication)
  - [Configuration Grafana Alloy](#configuration-grafana-alloy)
  - [Variables d'Environnement](#variables-denvironnement)
- [Gestion du Submodule](#gestion-du-submodule)
- [Documentation du Submodule](#documentation-du-submodule)
- [Métriques de l'Application](#métriques-de-lapplication)
- [Format des Logs](#format-des-logs)
- [Points d'Accès](#points-daccès)
- [Dépannage](#dépannage)

---

## Configuration Rapide

### Prérequis

- Docker installé
- Git installé
- Permissions pour exécuter `make`

### Installation

```bash
# 1. Initialiser le submodule (première fois seulement)
make obs-init

# 2. Configurer les secrets
make setup-secrets

# 3. Démarrer tous les services
make up
```

### Vérification

```bash
# Vérifier le statut
make ps

# Accéder à Grafana
open http://localhost:3000

# Identifiants : admin / (mot de passe dans observability/secrets/grafana_password.txt)
```

---

## Architecture

### Diagramme Global

```
┌───────────────────────────────────────────────────────────┐
│   medicaments-api/                                        │
│   (docker-compose.yml)                                    │
│                                                           │
│  ┌─────────────────┐       ┌─────────────────┐            │
│  │ medicaments-api │◀──────│ grafana-alloy   │            │
│  │  (logs/metrics) │       │   (collector)   │            │
│  └────────┬────────┘       └────────┬────────┘            │
│           │ logs volume             │                     │
│           └────────────────▶────────┘                     │
│                                  │  │                     │
│                    Network: obs-network (external)        │
└──────────────────────────────────┼──┼─────────────────────┘
                                   │  │
┌──────────────────────────────────┼──┼─────────────────────┐
│    observability/                │  │                     │
│    (git submodule)               │  │                     │
│                                  │  │                     │
│                       ┌──────────┘  └─────────┐           │
│                       │                       │           │
│                  ┌────▼────┐           ┌──────▼─────┐     │
│                  │  loki   │           │ prometheus │     │
│                  │ (logs)  │           │  (metrics) │     │
│                  └────┬────┘           └──────┬─────┘     │
│                       └──────────┬────────────┘           │
│                                  │                        │
│                           ┌──────▼────────┐               │
│                           │    grafana    │               │
│                           │(visualization)│               │
│                           └───────────────┘               │
└───────────────────────────────────────────────────────────┘
```

**Flux de données :**

1. **medicaments-api** génère des logs (vers `/app/logs/`) et des métriques (vers l'endpoint `/metrics`)
2. **Grafana Alloy** lit les fichiers de logs et scrape les endpoints de métriques
3. **Loki** stocke les logs provenant d'Alloy
4. **Prometheus** stocke les métriques provenant d'Alloy via remote_write
5. **Grafana** interroge à la fois Loki et Prometheus pour la visualisation

**Réseau :**

- Le réseau `obs-network` est **externe** et créé par le submodule `observability/`
- Les deux fichiers `docker-compose.yml` utilisent ce réseau pour la communication inter-conteneurs
- Le réseau est partagé entre l'application et la stack d'observabilité

---

## Modes de Configuration

### Mode Local (Défaut)

Dans le mode local, Alloy se connecte directement aux services d'observabilité via le DNS du conteneur Docker :

**Configuration :**

- Variable d'environnement : `ALLOY_CONFIG=config.alloy` (ou laisser vide)
- Grafana Alloy connecte à :
  - `http://loki:3100` pour les logs
  - `http://prometheus:9090` pour les métriques

**Utilisation recommandée :**

- Développement local
- Staging sur la même machine
- Tests et validation

**Démarrage :**

```bash
make setup-secrets  # Configuration des secrets (première fois)
make obs-init        # Initialisation du submodule (première fois)
make up              # Démarrage de tous les services
```

### Mode Remote (Production)

Dans le mode remote, Alloy se connecte à des endpoints distants via des tunnels sécurisés :

**Configuration :**

- Variable d'environnement : `ALLOY_CONFIG=config.remote.alloy`
- Variables d'environnement requises :
  - `PROMETHEUS_URL` : URL distante de Prometheus (ex: `https://prometheus-obs.example.com/api/v1/write`)
  - `LOKI_URL` : URL distante de Loki (ex: `https://loki-obs.example.com/loki/api/v1/push`)
- Options de tunnel :
  - **Cloudflare Tunnel** : URLs HTTPS avec Cloudflare Access
  - **Tailscale VPN** : URLs HTTP avec adresse IP privée
  - **VPN/Mesh** : URLs HTTP avec réseau privé

**Utilisation recommandée :**

- Production avec infrastructure centralisée
- Multi-site monitoring
- Environnements cloud

**Configuration exemple :**

```bash
# .env.docker
ALLOY_CONFIG=config.remote.alloy

# Cloudflare Tunnel
PROMETHEUS_URL=https://prometheus-obs.yourdomain.com/api/v1/write
LOKI_URL=https://loki-obs.yourdomain.com/loki/api/v1/push

# Cloudflare Access (optionnel)
CF_ACCESS_CLIENT_ID=your_client_id
CF_ACCESS_CLIENT_SECRET=your_client_secret

# Tailscale VPN
# PROMETHEUS_URL=http://100.x.x.x:9090/api/v1/write
# LOKI_URL=http://100.x.x.x:3100/loki/api/v1/push
```

**Démarrage :**

```bash
make setup-secrets  # Configuration des secrets (première fois)
make obs-init        # Initialisation du submodule (première fois)
make up              # Démarrage de tous les services
```

**Protection contre les pannes :**
Le mode remote utilise le buffer WAL (Write-Ahead Log) d'Alloy pour protéger les données pendant les pannes réseau :

- Buffer de 2.5GB
- Protection variable selon le volume de données (plusieurs heures à plusieurs jours selon le trafic)
- Reprise automatique lors de la restauration de la connexion

---

## Configuration de l'Application

### Configuration Grafana Alloy

medicaments-api utilise Grafana Alloy pour collecter les logs et les métriques. Les configurations Alloy sont situées dans `configs/alloy/` :

| Fichier                    | Mode        | Description                                            |
| -------------------------- | ----------- | ------------------------------------------------------ |
| `configs/alloy/config.alloy` | Local       | Connexion directe à Loki/Prometheus via le réseau Docker |
| `configs/alloy/config.remote.alloy` | Remote      | Connexion via tunnel avec auth et WAL buffering        |

**Fonctions principales de la configuration Alloy :**

- **Collecte des logs** : Lecture des fichiers JSON depuis `/var/log/app/*.log`
- **Parsing des logs** : Extraction des labels (level, path, status, duration_ms)
- **Filtrage DEBUG** : Suppression des logs DEBUG avant stockage dans Loki
- **Scraping des métriques** : Récupération des métriques HTTP depuis `medicaments-api:9090/metrics`
- **Métriques système** : Collecte des métriques CPU, mémoire, disque, réseau

**Note** : Les configurations Alloy sont spécifiques à medicaments-api et ne font pas partie du submodule.

### Variables d'Environnement

Les variables d'environnement pour l'observabilité sont définies dans `.env.docker` :

| Variable                    | Valeur Par Défaut       | Description                                          |
| --------------------------- | ------------------------ | ---------------------------------------------------- |
| `ALLOY_CONFIG`             | `config.alloy`           | Fichier de configuration Alloy à utiliser               |
| `PROMETHEUS_URL`          | -                        | URL distante de Prometheus (mode remote seulement)      |
| `LOKI_URL`                 | -                        | URL distante de Loki (mode remote seulement)            |
| `CF_ACCESS_CLIENT_ID`      | -                        | Cloudflare Access client ID (optionnel, mode remote)    |
| `CF_ACCESS_CLIENT_SECRET`    | -                        | Cloudflare Access client secret (optionnel, mode remote)  |

**Note** : En mode local, seules `ALLOY_CONFIG` est requis. En mode remote, les variables de tunnel sont nécessaires.

---

## Gestion du Submodule

Le submodule d'observabilité est géré via les commandes Make suivantes :

### Commandes Disponibles

| Commande          | Description                                         |
| ----------------- | --------------------------------------------------- |
| `make obs-init`   | Initialiser le submodule (première fois)            |
| `make obs-up`     | Démarrer la stack d'observabilité                   |
| `make obs-down`   | Arrêter la stack d'observabilité                    |
| `make obs-logs`   | Voir les logs de la stack d'observabilité           |
| `make obs-status` | Vérifier le statut de la stack d'observabilité      |
| `make obs-update` | Mettre à jour le submodule vers la dernière version |

### Structure du Submodule

```
observability/
├── docker-compose.yml         # Orchestration de la stack (loki + prometheus + grafana)
├── configs/                  # Configurations de la stack
│   ├── loki/
│   ├── prometheus/
│   └── grafana/
├── secrets/                 # Secrets de la stack (gitignoré)
│   └── grafana_password.txt
└── docs/                   # Documentation complète de la stack
    ├── README.md
    ├── local-setup.md
    ├── remote-setup.md
    └── tunnels.md
```

### Réinitialisation du Submodule

En cas de problème avec le submodule, vous pouvez le réinitialiser :

```bash
# Supprimer le submodule
rm -rf .git/modules/observability
git submodule deinit -f observability
git rm -f observability

# Réinitialiser
make obs-init
```

---

## Documentation du Submodule

Pour la documentation complète de la stack d'observabilité, consultez les fichiers dans le submodule :

### Documentation Principale

- **[observability/README.md](observability/README.md)** : Guide principal de la stack d'observabilité
  - Vue d'ensemble et architecture
  - Quick start et modes de fonctionnement
  - Configuration et commandes Make

### Guides de Configuration

- **[observability/docs/local-setup.md](observability/docs/local-setup.md)** : Guide de configuration en mode local (submodule)
  - Ajout du submodule à votre application
  - Configuration du réseau Docker partagé
  - Démarrage et utilisation

- **[observability/docs/remote-setup.md](observability/docs/remote-setup.md)** : Guide de configuration en mode remote (tunnel)
  - Configuration des tunnels (Cloudflare, Tailscale, WireGuard)
  - Authentification et sécurité
  - Connexion d'applications distantes

- **[observability/docs/tunnels.md](observability/docs/tunnels.md)** : Guide détaillé des tunnels
  - Configuration Cloudflare Tunnel avec Cloudflare Access
  - Configuration Tailscale VPN
  - Configuration WireGuard

### Ressources du Submodule

- **Repository GitHub** : https://github.com/Giygas/observability-stack
- **Documentation** : [observability/docs/](observability/docs/)
- **Contribution** : [observability/CONTRIBUTING.md](observability/CONTRIBUTING.md)

**Note** : Pour les questions spécifiques à la stack d'observabilité (configuration, dépannage, alertes, tableaux de bord), consultez la documentation du submodule mentionnée ci-dessus.

---

## Métriques de l'Application

medicaments-api expose des métriques Prometheus sur le port 9090 (interne au réseau Docker).

### Métriques HTTP

Via `metrics/metrics.go` :

#### `http_request_total`

- **Type** : Counter
- **Labels** : `method`, `path`, `status`
- **Description** : Total des requêtes HTTP
- **Exemple** : `http_request_total{method="GET",path="/v1/medicaments",status="200"}`

#### `http_request_duration_seconds`

- **Type** : Histogram
- **Labels** : `method`, `path`
- **Description** : Histogramme de latence des requêtes
- **Buckets** : .001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5
- **Exemple** : `http_request_duration_seconds_sum{method="GET",path="/v1/medicaments"}`

#### `http_request_in_flight`

- **Type** : Gauge
- **Description** : Requêtes actuellement en cours
- **Exemple** : `http_request_in_flight`

### Visualisation des Métriques

Les métriques sont visualisées dans Grafana via les tableaux de bord préconfigurés fournis par le submodule.

**Accès aux métriques :**

- **Grafana** : http://localhost:3000 → Dashboards → medicaments-api Health
- **Prometheus** : http://localhost:9090 → requête directe des métriques
- **Alloy** : http://localhost:12345/metrics (métriques du collecteur)

---

## Format des Logs

medicaments-api génère des logs JSON structurés avec les champs suivants :

### Format attendu

```json
{
  "time": "2025-02-08T12:00:00Z",
  "level": "info",
  "path": "/v1/medicaments?search=paracetamol",
  "msg": "Request completed",
  "status": 200,
  "duration_ms": 15
}
```

### Champs Requis

| Champ      | Type   | Description                       | Exemple                         |
| ---------- | ------ | --------------------------------- | ------------------------------- |
| `time`     | string | Horodatage ISO 8601                | `"2025-02-08T12:00:00Z"`      |
| `level`    | string | Niveau de log (DEBUG/INFO/WARN/ERROR) | `"info"`                        |
| `path`     | string | Chemin de la requête HTTP           | `"/v1/medicaments?search=paracetamol"` |
| `msg`      | string | Message du log                     | `"Request completed"`            |
| `status`   | number | Code de statut HTTP                | `200`                           |
| `duration_ms` | number | Durée de la requête en ms         | `15`                            |

### Parsing par Alloy

La configuration Alloy utilise `stage.json` pour parser les logs JSON structurés et extraire les labels :

- **Labels extraits** : `level`, `path`, `status`, `duration_ms`
- **Filtrage** : Les logs de niveau `DEBUG` sont supprimés avant stockage dans Loki
- **Timestamp** : Utilisation du champ `time` avec format RFC3339Nano

### Modification du Format

Si vous modifiez le format des logs dans votre application, mettez à jour `configs/alloy/config.alloy` :

```alloy
// Pour les logs en texte brut, remplacer stage.json par :
loki.process "process_logs" {
  stage.regex {
    expression: "^(?P<timestamp>\\S+) (?P<level>\\S+) (?P<message>.*)$"
  }
  // ... configuration restante
}
```

---

## Points d'Accès

### Services de l'Application

| Service         | Port Hôte | URL                                     | Description                             |
| --------------- | ---------- | --------------------------------------- | ------------------------------------- |
| medicaments-api | 8030       | http://localhost:8030                     | API principale                        |
| grafana-alloy   | 12345      | http://localhost:12345/metrics           | Métriques du collecteur Alloy         |

### Services d'Observabilité (Submodule)

| Service    | Port Hôte | URL                                     | Description                             |
| ---------- | ---------- | --------------------------------------- | ------------------------------------- |
| loki       | Interne    | N/A                                     | Logs (accès via Grafana uniquement)      |
| prometheus | 9090       | http://localhost:9090                     | Interface Prometheus                   |
| grafana    | 3000       | http://localhost:3000                     | Interface Grafana avec tableaux de bord |

### Identifiants

**Grafana :**

- Nom d'utilisateur : `admin` (configurable via `GRAFANA_ADMIN_USER` dans le submodule)
- Mot de passe : Stocké dans `observability/secrets/grafana_password.txt` (créé via `make setup-secrets`)
- **Important** : Changez le mot de passe après la première connexion

**Autres Services :**

- Aucune authentification requise (réseau local uniquement)

---

## Dépannage

### Problèmes de Submodule

**Le submodule n'est pas initialisé :**

```bash
# Erreur : "fatal: not a git repository" ou "network obs-network not found"
# Solution : Initialiser le submodule
make obs-init
make up
```

**Le submodule est désynchronisé :**

```bash
# Erreur : Les services d'observabilité ne démarrent pas
# Solution : Mettre à jour le submodule
make obs-update
make obs-down
make obs-up
```

### Problèmes de Démarrage

**Le réseau obs-network n'existe pas :**

```bash
# Erreur : "network obs-network not found"
# Solution : Démarrer la stack d'observabilité en premier
make obs-up
make up
```

**Les services d'observabilité ne démarrent pas :**

```bash
# Vérifier les logs de la stack d'observabilité
make obs-logs

# Vérifier le statut de tous les conteneurs
docker compose ps

# Redémarrer les services d'observabilité
make obs-down
make obs-up
```

### Problèmes de Logs

**Les logs n'apparaissent pas dans Grafana :**

```bash
# Vérifier qu'Alloy lit les logs
docker compose logs grafana-alloy | grep -i logs

# Vérifier que les fichiers de logs existent
docker compose exec grafana-alloy ls -la /var/log/app/

# Vérifier que Loki reçoit les logs
make obs-logs | grep loki | grep -i received
```

### Problèmes de Métriques

**Les métriques n'apparaissent pas dans Grafana :**

```bash
# Vérifier qu'Alloy scrape les métriques
docker compose logs grafana-alloy | grep -i scrape

# Vérifier que l'endpoint de métriques est accessible
docker compose exec grafana-alloy wget -O- http://medicaments-api:9090/metrics

# Vérifier que Prometheus reçoit les métriques
make obs-logs | grep prometheus | grep -i received

# Tester la requête Prometheus
curl 'http://localhost:9090/api/v1/query?query=http_request_total'
```

### Dépannage Avancé

Pour des problèmes plus complexes concernant :

- Configuration détaillée de Loki, Prometheus, Grafana
- Alertes et règles Prometheus
- Tableaux de bord personnalisés
- Configuration des tunnels (Cloudflare, Tailscale, WireGuard)
- Performance et optimisation

**Consultez la documentation du submodule :**

- **[observability/README.md](observability/README.md)** : Guide principal
- **[observability/docs/local-setup.md](observability/docs/local-setup.md)** : Mode local
- **[observability/docs/remote-setup.md](observability/docs/remote-setup.md)** : Mode remote
- **[observability/docs/tunnels.md](observability/docs/tunnels.md)** : Tunnels

**Ou reportez l'issue sur le repository du submodule :**
https://github.com/Giygas/observability-stack/issues

---

## Ressources

### Documentation de medicaments-api

- **[DOCKER.md](DOCKER.md)** : Guide complet de Docker pour medicaments-api
- **[README.md](README.md)** : Vue d'ensemble du projet

### Documentation du Submodule

- **[observability/README.md](observability/README.md)** : Documentation principale
- **[observability/docs/local-setup.md](observability/docs/local-setup.md)** : Guide de configuration local
- **[observability/docs/remote-setup.md](observability/docs/remote-setup.md)** : Guide de configuration remote
- **[observability/docs/tunnels.md](observability/docs/tunnels.md)** : Guide des tunnels

### Liens Externes

- **Grafana** : https://grafana.com/docs/
- **Loki** : https://grafana.com/docs/loki/latest/
- **Prometheus** : https://prometheus.io/docs/
- **Grafana Alloy** : https://grafana.com/docs/alloy/
