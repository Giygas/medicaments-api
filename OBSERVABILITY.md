# Guide de la Stack d'Observabilité

**Guide complet de la stack d'observabilité Grafana, Loki, Prometheus et Alloy**

---

**🇫🇷 Français** | [🇬🇧 English](OBSERVABILITY.en.md)

---

## Table des matières

- [Vue d'ensemble](#vue-densemble)
- [Architecture](#architecture)
- [Architecture des Ports](#architecture-des-ports)
- [Services](#services)
  - [Grafana Alloy](#grafana-alloy)
  - [Loki](#loki)
  - [Prometheus](#prometheus)
  - [Grafana](#grafana)
- [Points d'Accès](#points-daccès)
- [Format des Logs](#format-des-logs)
- [Métriques Collectées](#métriques-collectées)
- [Identifiants par Défaut](#identifiants-par-défaut)
- [Utilisation des Ressources](#utilisation-des-ressources)
- [Fichiers de Configuration](#fichiers-de-configuration)
- [Dépannage](#dépannage)
- [Nettoyage](#nettoyage)
- [Alerting Prometheus](#alerting-prometheus)
  - [Règles d'Alerte](#règles-dalerte)
  - [Visualisation des Alertes](#visualisation-des-alertes)
  - [Personnalisation des Alertes](#personnalisation-des-alertes)
  - [Monitoring des Health Checks](#monitoring-des-health-checks)
- [Sujets Avancés](#sujets-avancés)

---

## Vue d'ensemble

Le setup staging inclut une stack d'observabilité complète avec Grafana, Loki, Prometheus et Alloy pour le monitoring des logs et des métriques.

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

---

## Architecture

```
medicaments-api (logs + metrics)
          ↓
grafana-alloy (collector)
          ↓         ↓
        loki    prometheus
          ↓         ↓
          grafana (visualization)
```

**Flux de données :**

1. **medicaments-api** génère des logs (vers `/app/logs/`) et des métriques (vers l'endpoint `/metrics`)
2. **Grafana Alloy** lit les fichiers de logs et scrape les endpoints de métriques
3. **Loki** stocke les logs provenant d'Alloy
4. **Prometheus** stocke les métriques provenant d'Alloy via remote_write
5. **Grafana** interroge à la fois Loki et Prometheus pour la visualisation

---

## Architecture des Ports

| Service         | Port Conteneur | Port Hôte | Accès Externe                | Communication Interne |
| --------------- | -------------- | --------- | ---------------------------- | --------------------- |
| medicaments-api | 8000 (API)     | 8030      | http://localhost:8030        | medicaments-api:8000  |
| medicaments-api | 9090 (metrics) | interne   | N/A                          | medicaments-api:9090  |
| grafana-alloy   | 12345          | 12345     | http://localhost:12345/metrics | grafana-alloy:12345  |
| loki            | 3100           | interne   | N/A                          | loki:3100             |
| prometheus      | 9090           | 9090      | http://localhost:9090        | prometheus:9090       |
| grafana         | 3000           | 3000      | http://localhost:3000        | grafana:3000          |

**Points clés :**

- Grafana se connecte à Prometheus sur `prometheus:9090` (port conteneur)
- L'accès externe à Prometheus se fait via `localhost:9090` (mappage de port hôte)
- Toute la communication service-à-service utilise les ports conteneurs dans le réseau Docker
- Les ports hôte sont uniquement pour accéder aux services depuis la machine hôte
- Certains services (métriques de medicaments-api, Loki) sont uniquement exposés en interne au réseau Docker pour la sécurité

---

## Services

### Grafana Alloy

Collecte les logs et métriques de medicaments-api et les métriques système.

- **Image** : `grafana/alloy:v1.4.0`
- **Configuration** : `observability/alloy/config.alloy`
- **Port** : 12345 (métriques d'Alloy)
- **Fonctions** :
  - Lire les logs du répertoire `./logs/`
  - Scraper les métriques de l'application depuis `medicaments-api:9090/metrics` (toutes les 30s)
  - Collecter les métriques système via l'exporter Unix (toutes les 60s)
  - Transférer vers le Loki et Prometheus locaux
  - Filtrer les métriques du runtime Go (conserver uniquement les métriques HTTP et système)
- **Utilisation des ressources** : ~150MB RAM

**Points forts de la configuration :**

```alloy
// Collecte des logs
loki.source.file "read_logs" {
  targets    = [{__path__ = "/var/log/app/*.log"}]
  forward_to = [loki.write.local.receiver]
}

// Scraping des métriques
prometheus.scrape "medicaments" {
  targets    = [{__address__ = "medicaments-api:9090"}]
  forward_to = [prometheus.remote_write.local.receiver]
  scrape_interval = "30s"
}

// Métriques système
prometheus.exporter.unix "system" {
  collectors = ["cpu", "meminfo", "filesystem", "network"]
}
```

### Loki

Agrégation et stockage des logs.

- **Image** : `grafana/loki:2.9.10`
- **Configuration** : `observability/loki/config.yaml`
- **Port** : 3100 (interne uniquement - exposé au réseau Docker)
- **Stockage** : Système de fichiers (chunks dans `/loki/chunks`, règles dans `/loki/rules`)
- **Rétention** : 30 jours (720 heures)
- **Volume de données** : `loki-data`
- **Utilisation des ressources** : ~100MB RAM + ~100MB disque
- **Health Check** : Disponible via l'endpoint `/ready`

**Points forts de la configuration :**

```yaml
limits_config:
  retention_period: 720h    # 30 jours
  ingestion_rate_mb: 16    # 16MB/sec

schema_config:
  configs:
    - from: 2024-01-01
      store: tsdb
      object_store: filesystem
      schema: v13

storage_config:
  filesystem:
    directory: /loki/chunks
  ruler:
    storage:
      type: local
      local:
        directory: /loki/rules
```

**Important :** Le type de stockage du ruler doit être explicitement défini à `local` dans Loki 2.9+ pour éviter les erreurs de démarrage.

### Prometheus

Stockage et interrogation des métriques.

- **Image** : `prom/prometheus:v2.48.0`
- **Configuration** : `observability/prometheus/prometheus.yml`
- **Port** : 9090 (hôte et conteneur)
  - Le port hôte 9090 fournit un accès externe à l'UI Prometheus
  - Le port conteneur 9090 est utilisé pour la communication service-à-service
- **Rétention** : 30 jours (720 heures)
- **Volume de données** : `prometheus-data`
- **Utilisation des ressources** : ~150MB RAM + ~200MB disque
- **Scraping** : Reçoit les métriques via `remote_write` de Grafana Alloy (pas besoin de `scrape_configs`)

**Points forts de la configuration :**

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'medicaments-staging'

# Rétention des données
retention:
  time: 720h  # 30 jours

# Remote write depuis Alloy
remote_write:
  - url: http://localhost:9090/api/v1/write

# Règles d'alerte
rule_files:
  - '/etc/prometheus/rules/*.yml'
```

### Grafana

Visualisation des logs et métriques.

- **Image** : `grafana/grafana:10.2.4`
- **Port** : 3000
- **Identifiants par défaut** : giygas/paquito (à changer après la première connexion)
- **Volume de données** : `grafana-data`
- **Utilisation des ressources** : ~200MB RAM + ~50MB disque
- **Auto-Provisioning** : Les datasources sont configurées automatiquement

**Auto-Provisioning :**

- **Datasources** : Configurées automatiquement depuis `observability/grafana/provisioning/datasources/`
  - Loki : `observability/grafana/provisioning/datasources/loki.yml`
  - Prometheus : `observability/grafana/provisioning/datasources/prometheus.yml`
- **Tableaux de bord** : Importés automatiquement depuis `observability/grafana/provisioning/dashboards/`

---

## Points d'Accès

```bash
# UI Grafana (visualisation)
open http://localhost:3000

# UI Prometheus (navigation des métriques)
open http://localhost:9090

# Métriques de medicaments-api (métriques de l'application)
# Disponible uniquement via le réseau Docker pour le scraping interne
curl http://localhost:9090/metrics

# Métriques d'Alloy (statut du collecteur)
curl http://localhost:12345/metrics
```

**Note** : Loki et les métriques de medicaments-api sont uniquement exposés en interne au réseau Docker.
Ils sont scrapés par Alloy et ne sont pas directement accessibles depuis la machine hôte pour la sécurité.

---

## Format des Logs

Votre application devrait générer des logs JSON avec les champs `level` et `path` pour un parsing correct :

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

**Parsing des Logs par Alloy :**

La configuration Alloy utilise `stage.json` pour parser les logs JSON structurés :

```alloy
loki.source.file "read_logs" {
  targets    = [{__path__ = "/var/log/app/*.log"}]
  forward_to = [loki.process.process_logs.receiver]
}

loki.process "process_logs" {
  stage.json {}
  stage.labels {
    values = {
      level   = "level",
      path    = "path",
      status  = "status"
    }
  }
  forward_to = [loki.write.local.receiver]
}
```

Si les logs sont en texte brut, mettez à jour `alloy/config.alloy` pour supprimer le bloc `stage.json` et utiliser le parsing regex.

---

## Métriques Collectées

### Depuis l'Application (endpoint `/metrics`)

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

### Depuis les Métriques Système (via `prometheus.exporter.unix`)

Alloy collecte les métriques système suivantes :

- **Métriques CPU** : Utilisation CPU du processus et du système
  - `process_cpu_seconds_total`
  - `node_cpu_seconds_total`

- **Métriques Mémoire** : Utilisation mémoire du processus et du système
  - `process_resident_memory_bytes`
  - `process_virtual_memory_bytes`
  - `node_memory_MemAvailable_bytes`
  - `node_memory_MemTotal_bytes`

- **Descripteurs de fichiers** : Descripteurs de fichiers ouverts
  - `process_open_fds`
  - `process_max_fds`

- **Métriques Réseau** : Statistiques d'E/S réseau
  - `node_network_receive_bytes_total`
  - `node_network_transmit_bytes_total`

- **Métriques Disque** : Statistiques d'E/S disque et système de fichiers
  - `node_filesystem_size_bytes`
  - `node_filesystem_avail_bytes`
  - `node_filesystem_read_bytes_total`

**Note :** La configuration Alloy filtre les métriques du runtime Go des scrapers d'application et système, conservant uniquement les métriques HTTP et système pertinentes.

---

## Identifiants par Défaut

**Grafana :**

- Nom d'utilisateur : `giygas` (depuis `.env.docker`)
- Mot de passe : Stocké dans `secrets/grafana_password.txt` (créé via `make setup-secrets`)
- **Important** : Changez le mot de passe après la première connexion (Configuration → Utilisateurs → Changer le mot de passe)

**Autres Services :**

- Aucune authentification requise (réseau local uniquement)

---

## Utilisation des Ressources

| Service         | RAM        | Disque        | Rétention      |
| --------------- | ---------- | ------------- | -------------- |
| medicaments-api | ~50MB      | ~20MB         | N/A            |
| grafana-alloy   | ~150MB     | ~10MB         | N/A            |
| loki            | ~100MB     | ~100MB (données) | 30 jours      |
| prometheus      | ~150MB     | ~200MB (données) | 30 jours (720h) |
| grafana         | ~200MB     | ~50MB         | N/A            |
| **Total**       | **~650MB** | **~380MB**    | 30 jours (les deux) |

---

## Fichiers de Configuration

| Fichier                                                             | Objectif                                                                                               |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `observability/alloy/config.alloy`                                   | Configuration Alloy (collecte logs + métriques, filtre les métriques runtime Go)                          |
| `observability/loki/config.yaml`                                    | Configuration Loki (stockage logs, stockage ruler filesystem, rétention 30 jours, taux d'ingestion 16MB/sec) |
| `observability/prometheus/prometheus.yml`                           | Configuration Prometheus (stockage métriques, rétention 30 jours, règles d'alerte)                       |
| `observability/prometheus/alerts/medicaments-api.yml`                | Règles d'alerte Prometheus (service en panne, taux d'erreurs élevé, latence élevée)                     |
| `observability/grafana/provisioning/datasources/loki.yml`           | Configuration automatique de la datasource Loki                                                         |
| `observability/grafana/provisioning/datasources/prometheus.yml`     | Configuration automatique de la datasource Prometheus                                                   |
| `observability/grafana/provisioning/dashboards/dashboard.yml`        | Import automatique des tableaux de bord Grafana                                                         |
| `observability/grafana/dashboards/api-health.json`                   | Tableau de bord de santé de l'API préconfiguré                                                         |

---

## Dépannage

### Grafana ne peut pas se connecter aux datasources

```bash
# Vérifier que les conteneurs sont en cours d'exécution
docker-compose ps

# Vérifier la connectivité réseau
# Note : Grafana se connecte à Prometheus sur le port conteneur 9090
docker-compose exec grafana wget -O- http://loki:3100/ready
docker-compose exec grafana wget -O- http://prometheus:9090/-/ready

# Vérifier la configuration des datasources
docker-compose logs grafana | grep -i datasource

# Vérifier la configuration de la datasource Prometheus
cat observability/grafana/provisioning/datasources/prometheus.yml
# Devrait afficher : url: http://prometheus:9090
```

### Les logs n'apparaissent pas dans Grafana

```bash
# Vérifier qu'Alloy lit les logs
docker-compose logs grafana-alloy | grep -i logs

# Vérifier que les fichiers de logs existent
docker-compose exec grafana-alloy ls -la /var/log/app/

# Vérifier que Loki reçoit les logs
docker-compose logs loki | grep -i received

# Interroger les logs depuis le réseau Docker
docker-compose exec loki wget -O- 'http://localhost:3100/loki/api/v1/labels'
```

### Les métriques n'apparaissent pas dans Grafana

```bash
# Vérifier qu'Alloy scrape les métriques
docker-compose logs grafana-alloy | grep -i scrape

# Vérifier que l'endpoint de métriques est accessible
docker-compose exec grafana-alloy wget -O- http://medicaments-api:9090/metrics

# Vérifier que Prometheus reçoit les métriques
docker-compose logs prometheus | grep -i received

# Tester la requête Prometheus
curl 'http://localhost:9090/api/v1/query?query=http_request_total'
```

### Loki échoue au démarrage avec une erreur de stockage

```bash
# Vérifier les logs de Loki pour les erreurs de configuration de stockage
docker-compose logs loki | grep -i "storage\|ruler"

# Erreur courante : "field filesystem not found in type base.RuleStoreConfig"
# Cela se produit dans Loki 2.9+ quand le type de stockage ruler n'est pas explicitement spécifié

# Correction : S'assurer que la section ruler de loki/config.yaml a un stockage local explicite :
#   ruler:
#     storage:
#       type: local
#       local:
#         directory: /loki/rules

# Redémarrer Loki après avoir corrigé la configuration
docker-compose restart loki
```

### Utilisation élevée des ressources

```bash
# Vérifier l'utilisation des ressources pour tous les services
docker stats medicaments-api grafana-alloy loki prometheus grafana

# Vérifier l'utilisation du disque pour les volumes
docker system df -v

# Réduire la rétention si nécessaire (éditer loki/config.yaml ou prometheus/prometheus.yml)
```

### Problèmes de Communication entre Services

**Grafana ne peut pas se connecter à Prometheus :**

```bash
# Vérifier la configuration de la datasource Grafana
cat observability/grafana/provisioning/datasources/prometheus.yml

# S'assurer qu'elle utilise le port conteneur (9090)
# Correct : url: http://prometheus:9090
# Incorrect : url: http://prometheus:9090

# Redémarrer Grafana pour recharger la configuration de la datasource
docker-compose restart grafana

# Vérifier la connectivité depuis le conteneur Grafana
docker-compose exec grafana wget -O- http://prometheus:9090/-/ready
```

**Les métriques n'apparaissent pas dans Grafana :**

```bash
# Vérifier si Alloy scrape les métriques de medicaments-api
docker-compose logs grafana-alloy | grep -i "medicaments-api:9090"

# Vérifier que l'endpoint de métriques de medicaments-api est accessible
curl http://localhost:9090/metrics

# Vérifier si Prometheus reçoit les métriques d'Alloy
docker-compose logs prometheus | grep -i "received from Alloy"

# Tester la requête Prometheus pour les métriques de l'application
curl 'http://localhost:9090/api/v1/query?query=http_request_total'
```

---

## Nettoyage

```bash
# Arrêter uniquement les services d'observabilité
docker-compose stop grafana-alloy loki prometheus grafana

# Supprimer les services d'observabilité (conserve les volumes)
docker-compose rm -f grafana-alloy loki prometheus grafana

# Supprimer les services d'observabilité et toutes les données (SUPPRIME TOUT)
docker-compose down -v

# Supprimer uniquement les volumes d'observabilité
docker volume rm medicaments-api_loki-data medicaments-api_prometheus-data medicaments-api_grafana-data
```

---

## Alerting Prometheus

La stack de monitoring inclut des règles d'alerte Prometheus qui détectent automatiquement les problèmes et affichent les alertes dans Grafana.

### Règles d'Alerte

**Emplacement des règles d'alerte :** `observability/prometheus/alerts/medicaments-api.yml`

#### Alertes Critiques

| Alerte            | Description            | Seuil                  | Durée |
| ----------------- | ---------------------- | ---------------------- | ----- |
| ServiceDown       | Service inaccessible   | `up == 0`              | 5m    |
| High5xxErrorRate  | Trop d'erreurs serveur | Taux 5xx > 5%          | 5m    |
| HighTotalErrorRate | Trop d'erreurs au total | Taux 4xx+5xx > 10%     | 5m    |

#### Alertes d'Avertissement

| Alerte            | Description            | Seuil                       | Durée |
| ----------------- | ---------------------- | --------------------------- | ----- |
| HighLatencyP95    | Temps de réponse lents | Latence P95 > 200ms         | 10m   |
| HighRequestRate   | Volume de trafic élevé | Taux de requêtes > 1000/sec | 5m    |
| Sustained4xxRate  | Taux d'erreurs client élevé | Taux 4xx > 5%           | 10m   |

### Visualisation des Alertes dans Grafana

1. Naviguez vers `http://localhost:3000`
2. Allez dans **Alerting** → **Alert Rules** (dans la barre latérale gauche)
3. Filtrez par job `medicaments-api`
4. Visualisez les alertes actives, les alertes masquées et l'historique des alertes

### Personnalisation des Alertes

Éditez `observability/prometheus/alerts/medicaments-api.yml` pour ajuster les seuils :

```yaml
# Exemple : Changer le seuil de latence P95
- alert: HighLatencyP95
  expr: |
    histogram_quantile(0.95,
      rate(http_request_duration_seconds_bucket{job="medicaments-api"}[10m])
    ) > 0.5  # Changer de 0.2 (200ms) à 0.5 (500ms)
  for: 10m
  annotations:
    summary: "Latence P95 élevée détectée"
    description: "La latence P95 est de {{ $value }}s pour le job {{ $labels.job }}"
```

Après édition, rechargez la configuration Prometheus :

```bash
# Redémarrer Prometheus pour appliquer les modifications
docker-compose restart prometheus

# Ou utiliser SIGHUP pour le rechargement à chaud (si configuré)
docker exec prometheus kill -HUP 1
```

### Monitoring des Health Checks

Pour le monitoring des métriques système et de l'intégrité des données, utilisez l'endpoint `/v1/diagnostics` dans les alertes Grafana. L'endpoint `/health` est utilisé uniquement pour le statut de santé des données.

Créez un panneau d'alerte Grafana basé sur les données de diagnostics :

1. Allez dans **Dashboards** → **medicaments-api Health**
2. Ajoutez un nouveau panneau ou éditez un existant
3. Configurez une alerte basée sur le statut de santé ou l'âge des données
4. Configurez les conditions d'alerte (par exemple, `data_age_hours > 24`)

**Utilisation des endpoints :**
- `/health` → Statut de santé des données (nombre de médicaments, nombre de génériques, âge des données)
- `/v1/diagnostics` → Métriques système + intégrité des données (uptime, mémoire, vérifications d'intégrité des données)

---

## Sujets Avancés

### Ajout de Tableaux de Bord Personnalisés

1. Créez un nouveau tableau de bord JSON dans `observability/grafana/dashboards/`
2. Mettez à jour `observability/grafana/provisioning/dashboards/dashboard.yml` :

```yaml
apiVersion: 1

providers:
  - name: 'medicaments-api'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
      foldersFromFilesStructure: true
```

3. Redémarrez Grafana :

```bash
docker-compose restart grafana
```

### Parsers de Logs Personnalisés

Modifiez `observability/alloy/config.alloy` pour ajouter un parsing de logs personnalisé :

```alloy
loki.process "custom_parser" {
  stage.regex {
    expression: "^(?P<timestamp>\\S+) (?P<level>\\S+) (?P<message>.*)$"
  }
  stage.labels {
    values = {
      timestamp = "timestamp",
      level     = "level"
    }
  }
  forward_to = [loki.write.local.receiver]
}
```

### Réduction de la Rétention

Pour réduire l'utilisation du disque, éditez les paramètres de rétention :

**Loki** (`observability/loki/config.yaml`) :

```yaml
limits_config:
  retention_period: 168h  # 7 jours (au lieu de 720h)
```

**Prometheus** (`observability/prometheus/prometheus.yml`) :

```yaml
retention:
  time: 168h  # 7 jours (au lieu de 720h)
```

Puis redémarrez :

```bash
docker-compose restart loki prometheus
```

### Export des Métriques

Pour exporter les métriques vers un Prometheus externe :

```yaml
# Dans observability/prometheus/prometheus.yml
remote_write:
  - url: https://external-prometheus.example.com/api/v1/write
    basic_auth:
      username: ${EXTERNAL_PROMETHEUS_USER}
      password: ${EXTERNAL_PROMETHEUS_PASSWORD}
```

---

**Dernière mise à jour : 2026-02-17**
