# Journal des modifications

Tous les changements notables de ce projet seront documentés dans ce fichier.

Le format est basé sur [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
et ce projet adhère à [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - Non publié

## [1.1.0] - 2026-02-13

### Ajouté

- **API RESTful v1** avec 9 nouveaux endpoints utilisant le routage par chemin
  - Recherche de médicaments par CIS, CIP ou recherche multi-mots
  - Endpoints de présentation et de groupes génériques
  - `/v1/diagnostics` pour les métriques système et les rapports de qualité des données
- **Métriques Prometheus** sur le port 9090 (compteur de requêtes, histogramme de durée, jauge en cours)
- **Recherche multi-mots** avec logique AND (jusqu'à 6 mots)
- **Cache ETag** avec validation basée sur SHA256
- **Numérotation séquentielle des journaux** pour éviter une croissance non bornée (`_01`, `_02`, etc.)
- **Suivi des présentations orphelines** dans `/v1/diagnostics`
  - Affiche les codes CIP pour les présentations avec CIS inexistant
- **Nouvelle structure de documentation** avec le dossier `docs/` pour une meilleure organisation
  - `docs/ARCHITECTURE.md` - Architecture système, interfaces, middleware, disposition mémoire
  - `docs/DEVELOPMENT.md` - Guide de construction, test, lint et développement
  - `docs/PERFORMANCE.md` - Benchmarks, optimisations et profilage
  - `docs/MIGRATION.md` - Guide de migration v1 avec changements majeurs et exemples
  - `docs/TESTING.md` - Stratégies de test, benchmarks et guide de couverture

### Modifié

- **Réorganisation de la documentation** : README.md simplifié de 1 196 à 328 lignes (-73 %)
  - Meilleure séparation des préoccupations : page d'accueil vs documentation détaillée
  - Navigation améliorée pour les utilisateurs finaux et les contributeurs
  - Référence des endpoints déplacée vers la spécification OpenAPI
  - Détails de l'architecture et des performances déplacés vers des docs dédiés
- **Spécification OpenAPI mise à jour** : Ajout de liens vers la documentation externe pour le guide de migration
- **Noms normalisés pré-calculés** : recherche 5x plus rapide, 170x moins d'allocations
  - Recherche de médicaments : 3 500ns → 750ns (4,7x plus rapide)
  - Recherche de génériques : 3 500ns → 75ns (46,7x plus rapide)
- **Optimisation de la validation des entrées** : 5-10x plus rapide via string.Contains() et pré-compilation de regex
- **Mise en pool des rédacteurs de réponse** réduit les allocations de 2,2 Mo/s à haut débit
- **Journalisation rapide** pour les endpoints /health et /metrics
- **LOG_LEVEL maintenant fonctionnel** avec repli basé sur l'environnement (console uniquement)
- **Coûts des endpoints mis à jour** pour le rate limiting (5-200 tokens par requête)
- **Version Go requise** : 1.21 → 1.26+ (dernière version stable)

### Déprécié

- **Endpoints d'API hérités** (Date de fin de vie : 2026-07-31)
  - `/database` → Utiliser `/v1/medicaments/export`
  - `/database/{page}` → Utiliser `/v1/medicaments?page={n}`
  - `/medicament/{nom}` → Utiliser `/v1/medicaments?search={nom}`
  - `/medicament/id/{cis}` → Utiliser `/v1/medicaments/{cis}`
  - `/medicament/cip/{cip}` → Utiliser `/v1/medicaments?cip={cip}`
  - `/generiques/{libelle}` → Utiliser `/v1/generiques?libelle={libelle}`
  - `/generiques/group/{id}` → Utiliser `/v1/generiques/{id}`

**En-têtes de dépréciation renvoyés** :

```
Deprecation: true
Sunset: 2026-07-31T23:59:59Z
Link: </v1/...>; rel="successor-version"
Warning: 299 - "Deprecated endpoint..."
```

### Corrigé

- **Conditions de course dans le logger rotatif** (fuites de ressources + problèmes de concurrence)
- **/v1/medicaments retourne 404** lorsqu'il n'est pas trouvé (au lieu d'un tableau vide)
- **Validation des génériques** : plage groupID 1-9999 avec messages d'erreur clairs
- **Validation des entrées ASCII uniquement** avec messages de rejet utiles pour les caractères accentués
- **Journalisation de l'arrêt du serveur** corrigée
- **Gestion des cas limites TSV** avec statistiques de saut pour les lignes malformées
- **Bug de décalage de 1 dans la validation** corrigé
- **Encodage de jeu de caractères** : Détection automatique UTF-8/ISO8859-1 dans le téléchargeur
- **Gestion des erreurs de timeout HTTP et scanner** (timeout de téléchargement de 5 minutes pour les fichiers BDPM,
  tampon de scanner de 1 Mo pour un parsing robuste, vérification des erreurs après chaque fichier)
- **Arrêt gracieux des serveurs de métriques/profilage** (annulation de contexte,
  évite les fuites de goroutine, arrêts plus propres avec timeout de 5 secondes)

### Performance

#### Débit HTTP : 13K à 80K+ req/s (améliorations significatives des performances)

##### Optimisation : Noms normalisés pré-calculés

Élimination de la normalisation de chaînes à l'exécution en calculant une fois lors du parsing. Cette optimisation réduit les allocations par requête et améliore considérablement la latence de recherche.

| Metric                        | Avant       | Après        | Amélioration      |
| ----------------------------- | ----------- | ------------ | ----------------- |
| **Débit HTTP**                |             |              |                   |
| └ Médicaments search          | 1,000 req/s | 5,000 req/s  | 5x (+400%)        |
| └ Génériques search           | 5,000 req/s | 20,000 req/s | 4x (+300%)        |
| **Benchmarks algorithmiques** |             |              |                   |
| └ Médicaments - Reqs/sec      | 250         | 1,250        | 5x                |
| └ Médicaments - Latence       | 3,500µs     | 750µs        | 4.7x plus rapide  |
| └ Génériques - Reqs/sec       | 1,500       | 15,000       | 10x               |
| └ Génériques - Latence        | 3,500µs     | 75µs         | 46.7x plus rapide |
| **Allocations par recherche** | 16,000      | 94           | 170x réduction    |

**Compromis mémoire** : 0,75 Mo supplémentaire pour stocker les chaînes normalisées pré-calculées

##### Optimisation : Validation des entrées

- Mise en pool des rédacteurs de réponse réduit les allocations de 2,2 Mo/s à haut débit
- Journalisation rapide saute les opérations coûteuses pour les endpoints health/metrics
- Recherche CIP/CIS à temps constant (O(1)) en utilisant des hash maps
- Pré-compilation de regex au niveau du package
- Détection de motifs dangereux basée sur des chaînes (5-10x plus rapide que regex)
- Validation CIP/CIS directe via strconv.Atoi() sans regex

##### Débit HTTP final

Les optimisations combinées ont entraîné des gains de performance significatifs sur tous les endpoints :

| Endpoint                         | Avant     | Après      | Amélioration |
| -------------------------------- | --------- | ---------- | ------------ |
| `/v1/presentations/{cip}`        | 35K req/s | 77K req/s  | +120%        |
| `/v1/medicaments/{cis}`          | 13K req/s | 78K req/s  | +500%        |
| `/v1/medicaments?cip={code}`     | 35K req/s | 75K req/s  | +114%        |
| `/v1/medicaments?page={n}`       | 20K req/s | 41K req/s  | +105%        |
| `/v1/generiques?libelle={nom}`   | 5K req/s  | 36K req/s  | +620%        |
| `/v1/medicaments?search={query}` | 1K req/s  | 6.1K req/s | +510%        |
| `/health`                        | 30K req/s | 92K req/s  | +207%        |

**Memory** : 55-80MB stable (67.5MB median)

### Sécurité

- **Motif de validation des entrées** : `^[a-zA-Z0-9\s\+\.\-\/']+$` (ASCII uniquement)
  - Rejette les caractères accentués avec un message d'erreur utile
  - Prend en charge alphanumérique + espaces + trait d'union/point/slash/apostrophe/signe plus
  - Bloque explicitement les points consécutifs (`..`) pour prévenir le path traversal
- **Limite de recherche multi-mots** : Maximum 6 mots (prévention DoS)
- **Rate limiting variable** : 5-200 tokens par endpoint (1 000 tokens, recharge 3/sec)
- **Détection de motifs dangereux** : injection SQL, XSS, injection de commande, traversal de chemin (5-10x plus rapide que regex)
- **Validation CIP/CIS directe** via strconv.Atoi() sans regex

### Changements majeurs

**1. Structure de réponse du groupe générique** (MAJEUR)

Avant (`GET /generiques/group/{id}`) :

```json
{
  "cis": 12345678,
  "group": 100,
  "libelle": "Paracétamol",
  "type": "princeps"
}
```

Après (`GET /v1/generiques/{groupID}`) :

```json
{
  "groupID": 100,
  "libelle": "Paracétamol",
  "medicaments": [
    {
      "cis": 12345678,
      "elementPharmaceutique": "PARACETAMOL 500 mg, comprimé",
      "formePharmaceutique": "Comprimé",
      "type": "princeps",
      "composition": [...]
    }
  ],
  "orphanCIS": [87654321, 98765432]
}
```

**Mappage des champs** :

- `group` → `groupID` (renommé)
- `cis` → supprimé (maintenant dans le tableau medicaments)
- `type` → supprimé (maintenant dans chaque médicament du tableau)
- NOUVEAU : tableau `medicaments` avec les données complètes de composition
- NOUVEAU : tableau `orphanCIS` pour le suivi de la qualité des données

**Impact** : Les clients s'attendant à l'ancienne structure cesseront de fonctionner. Doivent migrer vers la nouvelle structure.

**2. Endpoint de santé simplifié**

Les métriques système ont été déplacées de `/health` vers `/v1/diagnostics`

- `/health` : Retourne uniquement le statut de base (endpoint rapide)
- `/v1/diagnostics` : Métriques système détaillées et rapports de qualité des données

**3. Version Go requise**

Version Go minimum : 1.21 → 1.26+ (dernière version stable)

### Guide de migration

**Guide de migration complet disponible dans `docs/MIGRATION.md`** avec les changements majeurs, les exemples et la liste de contrôle

**Référence rapide** :

```javascript
// Hérité
fetch("https://medicaments-api.giygas.dev/medicament/paracetamol");

// V1
fetch("https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol");
```

### Tests et Qualité

- **Couverture globale** : 78,5 %
- **Handlers** : 85,6 %
- **Parser de médicaments** : 84,2 %
- **Nouveaux fichiers de test** : Tests de fumée, validation ETag, endpoints v1, cohérence inter-fichiers
- **Benchmarks CI** : Non bloquants avec tolérance de 25 % de variance

[Non publié]: https://github.com/giygas/medicaments-api/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/giygas/medicaments-api/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/giygas/medicaments-api/releases/tag/v1.0.0
