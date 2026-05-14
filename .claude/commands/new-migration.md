# /new-migration

Crée une nouvelle migration SQL avec golang-migrate.

## Usage
```
/new-migration <description>
```

## Exemple
```
/new-migration add_refresh_tokens_table
/new-migration add_index_on_requests_user_id
/new-migration alter_users_add_verified_column
```

## Ce que cette commande fait

1. Génère deux fichiers dans `migrations/` :
   - `XXXXXX_<description>.up.sql`
   - `XXXXXX_<description>.down.sql`
2. Le numéro est auto-incrémenté depuis le dernier fichier existant
3. Ajoute un commentaire de description en haut de chaque fichier

## Règles importantes

```sql
-- up.sql : toujours idempotent
CREATE TABLE IF NOT EXISTS ...
CREATE INDEX IF NOT EXISTS ...
ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...

-- down.sql : toujours l'inverse exact du up
DROP TABLE IF EXISTS ...
DROP INDEX IF EXISTS ...
ALTER TABLE ... DROP COLUMN IF EXISTS ...
```

## Appliquer la migration

```bash
make migrate-up    # Applique toutes les migrations en attente
make migrate-down  # Rollback la dernière migration
```

## Checklist après génération

- [ ] Vérifier que up.sql est idempotent (IF NOT EXISTS)
- [ ] Vérifier que down.sql annule exactement le up.sql
- [ ] Tester up puis down localement
- [ ] Mettre à jour le schéma dans CLAUDE.md si nécessaire
- [ ] Régénérer sqlc si des tables sont modifiées : `make sqlc`