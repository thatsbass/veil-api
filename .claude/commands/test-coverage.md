# /test-coverage

Lance les tests et affiche le rapport de couverture par module.

## Usage
```
/test-coverage
/test-coverage <module>
```

## Exemples
```
/test-coverage                    # Tous les modules
/test-coverage internal/auth      # Module auth uniquement
/test-coverage internal/billing   # Module billing uniquement
```

## Ce que cette commande fait

```bash
# Tous les modules
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out | tail -n 1

# Module spécifique
go test -race -coverprofile=coverage.out ./<module>/...
go tool cover -func=coverage.out
```

## Objectifs de couverture par module

```
Module                  Minimum    Cible
────────────────────────────────────────
internal/auth           80%        90%
internal/translator     85%        95%
internal/billing        80%        90%
internal/provider       70%        85%
internal/gateway        60%        75%
internal/analytics      60%        70%
```

## Checklist si couverture insuffisante

- [ ] Identifier les fonctions non testées avec go tool cover -func
- [ ] Prioriser les tests sur les chemins critiques (auth, billing)
- [ ] Ajouter des tests de cas d'erreur (not just happy path)
- [ ] Utiliser des mocks pour les dépendances externes