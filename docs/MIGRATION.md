# Guide de Migration vers API v1

## Vue d'ensemble

Les endpoints v1 utilisent des paramètres de requête ou de chemin selon l'opération pour une architecture RESTful cohérente et optimisée.

**⚠️ Date de sunset des endpoints legacy** : **31 juillet 2026**

## Table de Migration

### Endpoints Médicaments

| Endpoint Legacy | Endpoint v1 |
|----------------|--------------|
| `GET /database` | `GET /v1/medicaments/export` |
| `GET /database/{page}` | `GET /v1/medicaments?page={n}` |
| `GET /medicament/{nom}` | `GET /v1/medicaments?search={query}` |
| `GET /medicament/id/{cis}` | `GET /v1/medicaments/{cis}` |
| `GET /medicament/cip/{cip}` | `GET /v1/medicaments?cip={cip}` |

### Endpoints Génériques

| Endpoint Legacy | Endpoint v1 |
|----------------|--------------|
| `GET /generiques/{libelle}` | `GET /v1/generiques?libelle={libelle}` |
| `GET /generiques/group/{id}` | `GET /v1/generiques/{id}` |

## Règles v1

### Un seul paramètre par requête

Seul un paramètre est autorisé à la fois : `page`, `search`, `cip`, `libelle` (CIS et export utilisent des paths séparés).

Les requêtes avec plusieurs paramètres retournent une erreur 400 :
```json
{
  "error": "Bad Request",
  "message": "Only one parameter allowed at a time",
  "code": 400
}
```

### Maximum de 6 mots

Les recherches multi-mots supportent jusqu'à 6 mots (logique ET) pour protection DoS.

Les requêtes avec plus de 6 mots retournent une erreur 400 :
```json
{
  "error": "Bad Request",
  "message": "search query too complex: maximum 6 words allowed",
  "code": 400
}
```

### Paramètres mutuellement exclusifs

Les paramètres de recherche et de pagination sont mutuellement exclusifs. Utiliser soit `search`, soit `page`, soit `cip`, mais pas plusieurs en même temps.

### Caractères ASCII uniquement

Les données source BDPM sont en majuscules sans accents (ex: IBUPROFENE, PARACETAMOL). L'API n'accepte que les caractères ASCII (lettres sans accents).

Les requêtes avec des accents retournent une erreur 400 :
```json
{
  "error": "Bad Request",
  "message": "Accents not supported. Try removing them (e.g., use 'ibuprofene' instead of 'ibuprofène')",
  "code": 400
}
```

### Headers de dépréciation

Les endpoints legacy renvoient les headers suivants pour aider les clients à migrer :

- `Deprecation: true`
- `Sunset: 2026-07-31T23:59:59Z`
- `Link: <https://medicaments-api.giygas.dev/v1/...>; rel="successor-version"`
- `X-Deprecated: Use /v1/... instead`
- `Warning: 299 - "Deprecated endpoint..."`

## Changements Majeurs (Breaking Changes)

### 1. Structure de réponse générique (MAJOR)

La structure de réponse des endpoints génériques a été modifiée de manière significative.

**Before** (`GET /generiques/group/{id}`) :
```json
{
  "cis": 12345678,
  "group": 100,
  "libelle": "Paracétamol",
  "type": "princeps"
}
```

**After** (`GET /v1/generiques/{groupID}`) :
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
      "composition": [
        {
          "elementPharmaceutique": "comprimé",
          "substance": "PARACETAMOL",
          "dosage": "500 mg"
        }
      ]
    }
  ],
  "orphanCIS": [87654321, 98765432]
}
```

**Field mappings :**
- `group` → `groupID` (renamed)
- `cis` → removed (now in medicaments array)
- `type` → removed (now in each medicament in array)
- **NEW** : `medicaments` array with full composition data
- **NEW** : `orphanCIS` array for data quality tracking

**Impact** : Les clients attendant l'ancienne structure vont échouer. La migration vers la nouvelle structure est obligatoire.

### 2. Endpoint santé simplifié

Les métriques système détaillées ont été déplacées de `/health` vers `/v1/diagnostics`.

**`/health`** (endpoint simplifié) :
- Retourne le statut de santé de base
- Idéal pour les health checks rapides

**`/v1/diagnostics`** (endpoint détaillé) :
- Retourne les métriques système détaillées
- Inclut le rapport d'intégrité des données
- Pour monitoring avancé et debugging

### 3. Version Go requise

Minimum Go version : 1.21 → 1.26+ (latest stable)

Cette exigence est nécessaire pour les nouvelles optimisations de performance.

## Exemples de Migration

### JavaScript/TypeScript

**Recherche par nom :**
```javascript
// Legacy
fetch('https://medicaments-api.giygas.dev/medicament/paracetamol')

// V1
fetch('https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol')
```

**Recherche par CIS :**
```javascript
// Legacy
fetch('https://medicaments-api.giygas.dev/medicament/id/61504672')

// V1
fetch('https://medicaments-api.giygas.dev/v1/medicaments/61504672')
```

**Pagination :**
```javascript
// Legacy
fetch('https://medicaments-api.giygas.dev/database/1')

// V1
fetch('https://medicaments-api.giygas.dev/v1/medicaments?page=1')
```

**Génériques :**
```javascript
// Legacy
fetch('https://medicaments-api.giygas.dev/generiques/paracetamol')

// V1
fetch('https://medicaments-api.giygas.dev/v1/generiques?libelle=paracetamol')
```

**Groupe générique :**
```javascript
// Legacy
fetch('https://medicaments-api.giygas.dev/generiques/group/1234')

// V1
fetch('https://medicaments-api.giygas.dev/v1/generiques/1234')
```

### Python

**Recherche par nom :**
```python
# Legacy
requests.get('https://medicaments-api.giygas.dev/medicament/paracetamol')

# V1
requests.get('https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol')
```

**Recherche par CIS :**
```python
# Legacy
requests.get('https://medicaments-api.giygas.dev/medicament/id/61504672')

# V1
requests.get('https://medicaments-api.giygas.dev/v1/medicaments/61504672')
```

### cURL

**Recherche par nom :**
```bash
# Legacy
curl https://medicaments-api.giygas.dev/medicament/paracetamol

# V1
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol"
```

**Recherche par CIS :**
```bash
# Legacy
curl https://medicaments-api.giygas.dev/medicament/id/61504672

# V1
curl https://medicaments-api.giygas.dev/v1/medicaments/61504672
```

**Pagination :**
```bash
# Legacy
curl https://medicaments-api.giygas.dev/database/1

# V1
curl https://medicaments-api.giygas.dev/v1/medicaments?page=1
```

**Génériques :**
```bash
# Legacy
curl https://medicaments-api.giygas.dev/generiques/paracetamol

# V1
curl "https://medicaments-api.giygas.dev/v1/generiques?libelle=paracetamol"
```

**Groupe générique :**
```bash
# Legacy
curl https://medicaments-api.giygas.dev/generiques/group/1234

# V1
curl https://medicaments-api.giygas.dev/v1/generiques/1234
```

## Le Champ OrphanCIS

Le champ `orphanCIS` contient les codes CIS référencés dans un groupe générique mais pour lesquels aucune entrée médicament correspondante n'existe dans la base de données.

### Valeurs possibles

- **Tableau d'entiers** : `[61586325, 60473805]`
- **Null** : `null` (si le groupe ne contient aucun CIS orphelin)

### Utilisation

- Médicaments avec des données complètes (composition, forme pharmaceutique, type) apparaissent dans le tableau `medicaments`
- Les CIS orphelins apparaissent dans le tableau `orphanCIS` sans détails supplémentaires
- Ce champ aide à identifier les incohérences potentielles dans les données BDPM

### Exemple

```json
{
  "groupID": 1368,
  "libelle": "PARACETAMOL 400 mg + CAFEINE 50 mg + CODEINE (PHOSPHATE DE) HEMIHYDRATE 20 mg",
  "medicaments": [
    {
      "cis": 61644230,
      "elementPharmaceutique": "PRONTALGINE, comprimé",
      "formePharmaceutique": "comprimé",
      "type": "Princeps",
      "composition": [...]
    },
    {
      "cis": 63399979,
      "elementPharmaceutique": "PARACETAMOL/CAFEINE/CODEINE ARROW 400 mg/50 mg/20 mg, comprimé",
      "formePharmaceutique": "comprimé",
      "type": "Générique",
      "composition": [...]
    }
  ],
  "orphanCIS": [61586325]
}
```

Dans cet exemple, le CIS `61586325` est référencé dans le groupe générique mais n'a pas d'entrée médicament correspondante dans la base de données.

## Recherche Multi-Mots

L'API supporte désormais la recherche multi-mots avec logique ET.

### Caractéristiques

- **Logique ET** : Tous les mots doivent être présents dans le résultat
- **Maximum 6 mots** : Limite pour protection DoS
- **Mots séparés par + ou espace** : Les deux formats sont supportés

### Exemples

**2 mots - recherche précise :**
```bash
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol+500"
curl "https://medicaments-api.giygas.dev/v1/generiques?libelle=paracetamol+500"
```

**6 mots - recherche très précise :**
```bash
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol+500+mg+comprime+boite+20"
curl "https://medicaments-api.giygas.dev/v1/generiques?libelle=paracetamol+500+mg+comprime+effervescent"
```

**Erreur - trop de mots (7+ mots) :**
```bash
curl "https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol+500+mg+comprime+boite+20+extra"
# Réponse : {"error": "Bad Request", "message": "search query too complex: maximum 6 words allowed", "code": 400}
```

## Checklist de Migration

- [ ] Mettre à jour les appels API vers endpoints v1
- [ ] Adapter les appels pour utiliser un seul paramètre
- [ ] Vérifier que les recherches multi-mots ≤ 6 mots
- [ ] Mettre à jour le parsing de réponse pour le nouveau format générique
- [ ] Utiliser `/v1/diagnostics` pour les métriques système détaillées
- [ ] Utiliser `/health` pour les health checks rapides
- [ ] Surveiller les headers de dépréciation pour identifier les appels legacy
- [ ] Tester l'application avec les nouveaux endpoints
- [ ] Mettre à jour Go version vers 1.26+ si nécessaire
- [ ] Mettre à jour la documentation interne

## Erreurs Courantes

### 400 Bad Request - Multiple parameters

**Erreur :**
```json
{
  "error": "Bad Request",
  "message": "Only one parameter allowed at a time",
  "code": 400
}
```

**Solution :** Utiliser un seul paramètre de requête (page, search, cip, libelle)

### 400 Bad Request - Accents non supportés

**Erreur :**
```json
{
  "error": "Bad Request",
  "message": "Accents not supported. Try removing them (e.g., use 'ibuprofene' instead of 'ibuprofène')",
  "code": 400
}
```

**Solution :** Supprimer les accents (données BDPM sont ASCII-only, ex: IBUPROFENE, PARACETAMOL)

### 400 Bad Request - Trop de mots

**Erreur :**
```json
{
  "error": "Bad Request",
  "message": "search query too complex: maximum 6 words allowed",
  "code": 400
}
```

**Solution :** Réduire le nombre de mots à 6 ou moins

### 404 Not Found

**Erreur :**
```json
{
  "error": "Not Found",
  "message": "No medicaments found",
  "code": 404
}
```

**Solution :** Vérifier que le paramètre de recherche est correct et que le médicament existe dans la base BDPM

### 429 Too Many Requests

**Erreur :**
```json
{
  "error": "Too Many Requests",
  "message": "Rate limit exceeded. Retry after X seconds",
  "code": 429
}
```

**Solution :** Attendre le temps indiqué dans le header `Retry-After` avant de réessayer

## Support

- **Spécification OpenAPI complète** : https://medicaments-api.giygas.dev/docs/openapi.yaml
- **Documentation interactive (Swagger UI)** : https://medicaments-api.giygas.dev/docs
- **GitHub Issues** : https://github.com/giygas/medicaments-api/issues
- **Health check** : https://medicaments-api.giygas.dev/health

## Notes Importantes

- Les endpoints legacy continueront de fonctionner jusqu'au 31 juillet 2026
- Les headers de dépréciation aideront à identifier les appels legacy
- La migration vers v1 est recommandée dès maintenant pour éviter les interruptions
- Consultez la section [Breaking Changes](#changements-majeurs-breaking-changes) pour comprendre les modifications structurelles
