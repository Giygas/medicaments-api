# Documents ANSM : RCP & Notices

## Vue d'ensemble

L'API expose deux points de terminaison en lecture seule qui servent le **RCP**
(Résumé des Caractéristiques du Produit) et la **notice patient** de n'importe
quel médicament, au format **JSON sectionné** conçu pour la consommation mobile
(liste de rubriques avec identifiants stables plutôt qu'un bloc HTML monolithique).

Les documents proviennent de la Base de Données Publique des Médicaments (ANSM) :

```
https://base-donnees-publique.medicaments.gouv.fr/medicament/{cis}/extrait
```

Le fonctionnement est **paresseux et permanent** :

1. la première requête pour un `(cis, type)` donné déclenche un unique appel
   amont (dédupliqué par singleflight) ;
2. le JSON final est compressé (gzip) puis écrit sur disque de façon atomique ;
3. toutes les requêtes suivantes sont servies depuis le cache disque, sans
   jamais re-contacter l'ANSM ;
4. un document définitivement absent en amont est enregistré comme
   **tombstone** (cache négatif) et ne sera **jamais re-téléchargé**.

Le HTML source est assaini avec une liste blanche stricte
(`bluemonday`) : seules les balises de contenu (`p`, `strong`, `em`, `ul`,
`ol`, `li`, `table`, `tr`, `td`, `th`, `sub`, `sup`, `h3`, `h4`, `br`, …) et
les attributs `colspan`/`rowspan` survivent. Tout le reste (scripts, styles,
classes, identifiants, gestionnaires d'événements, liens externes) est retiré.

## Points de terminaison

| Endpoint                          | Coût (tokens) | Cache                      |
| --------------------------------- | ------------- | -------------------------- |
| `GET /v1/medicaments/{cis}/rcp`   | 20            | 1h client / 24h CDN (ETag) |
| `GET /v1/medicaments/{cis}/notice`| 20            | 1h client / 24h CDN (ETag) |

### Exemples

```bash
# RCP d'un médicament par son code CIS
curl "https://medicaments-api.giygas.dev/v1/medicaments/60016308/rcp"

# Notice patient
curl "https://medicaments-api.giygas.dev/v1/medicaments/60016308/notice"
```

```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
ETag: "9f2c1e0a5b8d4c3f7a6e2d1b0c9f8e7d6a5b4c3f2e1d0c9b8a7f6e5d4c3b2a1"
Cache-Control: public, max-age=3600, s-maxage=86400
Last-Modified: Mon, 21 Sep 2026 10:00:00 GMT
```

## Format de réponse (JSON sectionné)

```json
{
  "cis": "60016308",
  "type": "rcp",
  "titre": "CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable",
  "miseAJour": "2025-11-07",
  "source": "ANSM - Base de données publique des médicaments",
  "sections": [
    { "id": "1", "titre": "1. DENOMINATION DU MEDICAMENT", "contenu": "<p>CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable</p>" },
    { "id": "4.2", "titre": "4.2. Posologie et mode d'administration", "contenu": "<p>Réservé à l'adulte.</p><h4>Posologie</h4>…" }
  ]
}
```

| Champ       | Description                                                                                  |
| ----------- | -------------------------------------------------------------------------------------------- |
| `cis`       | Code CIS canonique du médicament (sans zéros de tête)                                        |
| `type`      | `"rcp"` ou `"notice"`                                                                        |
| `titre`     | Dénomination du médicament (titre `<h3>` de la page ANSM)                                    |
| `miseAJour` | Date de mise à jour ANSM au format `AAAA-MM-JJ` (ligne « ANSM - Mis à jour le : JJ/MM/AAAA ») |
| `source`    | Attribution obligatoire (voir section « Attribution — licence Etalab 2.0 ») |
| `sections`  | Rubriques du document, dans l'ordre du document source                                       |

Chaque section porte trois champs : `id` (identifiant stable, voir ci-dessous),
`titre` (intitulé lisible de la rubrique) et `contenu` (HTML assaini par liste
blanche ; les sous-rubriques de niveau 3/4 sont rétrogradées en `<h4>` interne
à la section parente).

### Sémantique des identifiants de section

Les ID sont **stables par construction** pour permettre l'ancrage applicatif
(signets, synchronisation offline) :

- **RCP** : numérotation canonique du RCP (`1` à `12`, `4.1`–`4.9`, `5.1`–`5.3`,
  `6.1`–`6.6`). L'ID est d'abord le numéro explicite en tête du titre de la
  rubrique (`4.2. Posologie…` → `4.2`), sinon la table des matières canonique
  est mise en correspondance par titre, sinon repli sur le slug. Les rubriques
  mères structurelles vides immédiatement suivies de leur première fille
  (`4` suivi de `4.1`) sont fondues dans leurs filles.
- **Notices** : la structure des rubriques étant différente et majoritairement
  non numérotée, les ID sont des **titres slugifiés** : accents pliés
  (é → e), minuscules, séparateurs non alphanumériques réduits à un tiret —
  p.ex. « Dénomination du médicament » → `denomination-du-medicament`.
- Les doublons éventuels reçoivent un suffixe `-2`, `-3`…

## Cycle de vie d'une requête

```
GET /v1/medicaments/{cis}/{rcp|notice}
        │
        ├─ kill switch (DOCS_ENABLED=false) ────────────────► 501 Not Implemented
        ├─ CIS inconnu (absent du conteneur de données) ───► 404 (aucun appel amont)
        ├─ cache disque : document présent ─────────────────► 200 + ETag + Cache-Control
        │   └─ If-None-Match correspondant ─────────────────► 304 Not Modified
        ├─ tombstone (document absent connu) ───────────────► 404 « Document not available »
        └─ absence (miss) : fetch amont borné dans le temps
            ├─ panneau absent / 404-410 amont ─► tombstone + 404 (jamais retenté)
            ├─ échec amont / parse ────────────► 502 Bad Gateway (retentable)
            └─ succès ─────────────────────────► cache disque puis 200
```

Les erreurs transitoires (5xx, 429, timeout) ne créent **jamais** de
tombstone : la requête suivante retente le téléchargement.

## Cache disque

### Layout

```
{DOCS_CACHE_DIR}/
├── 60016308_rcp.json.gz        # document final (JSON sectionné gzip)
├── 60016308_notice.json.gz
└── 60016309_rcp.missing.json   # tombstone : document absent en amont
```

- Seul le **JSON final traité** est stocké (jamais le HTML brut).
- Écritures **atomiques** : fichier temporaire + `rename` dans le même
  répertoire — un lecteur n'observe jamais un fichier partiel.
- Un **index en mémoire** (≈ 2 Mo max) est reconstruit au démarrage en
  scannant le répertoire : `(cis, docType)` → `{sha256, date source,
  date fetch, taille, tombstone}`.
- **Auto-réparation** : un fichier corrompu ou disparu est ignoré au scan
  (ou retiré de l'index au runtime) et traité comme un miss — le document
  est re-téléchargé au prochain accès.
- Un document réel prime toujours sur une tombstone pour la même clé.

### Tombstones (cache négatif)

L'ANSM rend un panneau par document existant : si le panneau demandé est
absent de la page (ou que la page répond 404/410), le document est
définitivement introuvable en amont. Une tombstone est écrite et la paire
`(cis, docType)` ne sera **plus jamais téléchargée** — les requêtes suivantes
répondent 404 immédiatement, sans trafic réseau.

> **Attention ops** : l'index étant en mémoire, supprimer un fichier
> `.missing.json` à chaud ne suffit pas. Pour forcer un re-téléchargement
> (p.ex. l'ANSM publie plus tard le document), supprimez le marqueur **puis
> redémarrez le service** :
>
> ```bash
> rm /var/lib/medicaments-api/docs/60016309_rcp.missing.json
> sudo systemctl restart medicaments-api
> ```

## En-têtes HTTP et cache

| En-tête        | Valeur                                            |
| -------------- | ------------------------------------------------- |
| `ETag`         | Fort : sha256 du JSON, entre guillemets           |
| `Cache-Control`| `public, max-age=3600, s-maxage=86400`            |
| `Last-Modified`| Horodatage UTC de la mise en cache                |

- **Revalidation client** : une requête `If-None-Match` portant l'ETag reçu
  obtient un `304 Not Modified` à corps vide — gratuit en bande passante.
- **CDN / proxy partagé** (`s-maxage=86400`) : nginx (`proxy_cache`) ou
  Cloudflare peuvent servir la réponse depuis le bord pendant 24 h sans
  toucher l'origine ; les clients navigateurs restent limités à 1 h
  (`max-age=3600`) puis revalident via l'ETag.
- **Note Cloudflare** : l'ETag fort permet à Cloudflare de revalider auprès
  de l'origine (`304`) à l'expiration du TTL de bord. Comme les 404 de
  tombstone n'emportent pas d'en-tête `Cache-Control`, évitez les règles
  « Edge Cache TTL » globales qui mettraient en cache longtemps ces 404 —
  un document publié plus tard par l'ANSM resterait masqué le temps du TTL
  CDN (la tombstone côté origine exige de toute façon une purge manuelle,
  voir ci-dessus).
- Le coût en rate limiting est de **20 tokens** par requête (palier lookup).

## Configuration

| Variable                 | Défaut        | Description                                                        |
| ------------------------ | ------------- | ------------------------------------------------------------------ |
| `DOCS_ENABLED`           | `true`        | Kill switch : `false` → les deux endpoints répondent `501` et aucun téléchargement n'a lieu |
| `DOCS_CACHE_DIR`         | `./files/docs`| Répertoire du cache disque (voir [Déploiement](#déploiement))      |
| `DOCS_FETCH_RATE_PER_SEC`| `2`           | Débit max des appels amont (req/s), min 0.25, max 10               |
| `DOCS_FETCH_TIMEOUT`     | `15s`         | Timeout d'une requête amont individuelle                           |

### Kill switch

`DOCS_ENABLED=false` (ou l'absence d'injection des dépendances document)
désactive la fonctionnalité proprement : `501 Not Implemented` sur les deux
routes, zéro accès disque, zéro appel amont. C'est le premier réflexe en cas
de souci avec l'ANSM.

### Politesse envers l'ANSM

- User-Agent dédié : `medicaments-api/{version} (+https://medicaments-api.giygas.dev)`
- **Rate limiter global amont** (défaut 2 req/s), indépendant du rate limiting
  de l'API publique ;
- Timeout par requête + **1 retry** avec backoff, honorant `Retry-After` ;
- **singleflight** par `(cis, docType)` : N requêtes concurrentes pour un même
  document déclenchent exactement **un** appel amont.

## Codes d'erreur

| Statut | Condition                                                         |
| ------ | ----------------------------------------------------------------- |
| `501`  | `DOCS_ENABLED=false` (kill switch)                                |
| `400`  | CIS mal formé                                                     |
| `404`  | CIS inconnu, ou document absent en amont (tombstone)              |
| `502`  | Échec amont après retry / échec de parse — requête retentable     |
| `500`  | Erreur inattendue du cache disque                                 |

## Attribution (licence Etalab 2.0) — OBLIGATOIRE

Les données de la Base de Données Publique des Médicaments sont réutilisables
sous licence **Etalab Open License 2.0**. Cette licence impose deux
obligations aux réutilisateurs :

1. **Mention de la source** : « ANSM - Base de données publique des
   médicaments » ;
2. **Mention de la date de la dernière mise à jour** des données réutilisées.

C'est précisément pourquoi **chaque réponse** des endpoints documents porte
les champs `source` et `miseAJour` : affichez-les (ou toute mention
équivalente) dans votre application à côté du contenu du document. La licence
interdit par ailleurs toute **altération du sens** des données : l'API
assainit le balisage (scripts, styles, chrome DSFR) mais ne modifie jamais le
texte ni sa structure rubrique par rubrique — conservez cette intégrité dans
vos propres rendus.

Exemple de citation conforme :

```text
Source : ANSM - Base de données publique des médicaments
Date de mise à jour : 07/11/2025 (https://medicaments-api.giygas.dev/v1/medicaments/60016308/rcp)
```

## Déploiement

### Production (binaire + nginx, droplet DigitalOcean)

La production n'utilise **pas** Docker : binaire nu derrière nginx sur une
droplet DigitalOcean, déployé par GitHub Actions. Le déploiement remplace le
binaire — le cache documents doit donc vivre **en dehors du chemin déployé**
pour survivre aux mises en production (et ne jamais être écrasé par lui) :

```bash
# Une seule fois : créer le répertoire et donner les droits au service
sudo mkdir -p /var/lib/medicaments-api/docs
sudo chown medicaments-api:medicaments-api /var/lib/medicaments-api/docs
```

Puis dans le `.env` du service :

```dotenv
DOCS_CACHE_DIR=/var/lib/medicaments-api/docs
```

Dimensionnement : ~15 800 médicaments × 2 documents ≈ 12–30 Ko gzip chacun —
comptez **0,4 à 1 Go** de pire cas si tout le corpus finit caché (en pratique
seul le corpus réellement demandé l'est ; une poignée de très gros RCP —
vaccins, biologiques — pèse davantage que la moyenne).

### systemd (durcissement)

Si l'unité systemd durcit le système de fichiers (`ReadOnlyPaths=`,
`ProtectSystem=strict`…), autorisez explicitement l'écriture dans le cache :

```ini
[Service]
ReadWritePaths=/var/lib/medicaments-api/docs
```

### Docker — hors périmètre

Le câblage d'un volume Docker pour `DOCS_CACHE_DIR` est **explicitement hors
périmètre** et reporté à une tâche optionnelle ultérieure. En Docker, la
fonctionnalité fonctionne avec le répertoire par défaut (`./files/docs`,
éphémère au cycle de vie du conteneur) — les documents seraient re-téléchargés
à chaque recréation du conteneur.

## Observabilité

- Métriques Prometheus : `docs_cache_hits_total`, `docs_cache_misses_total`,
  `ansm_fetch_requests_total{status}`, `ansm_fetch_duration_seconds`,
  `docs_store_documents`, `docs_store_tombstones` (port interne `:9090`).
- `/v1/diagnostics` expose les compteurs du cache documents.
- Journaux : chaque fetch/tombstone/échec est journalisé via le logging
  structuré (`logging.Info/Warn/Error`).
