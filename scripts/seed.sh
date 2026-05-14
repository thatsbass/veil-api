#!/usr/bin/env bash
# Crée un utilisateur de test et une clé API Veil dans la base locale.
# Usage : ./scripts/seed.sh  ou  make seed

set -euo pipefail

# Charge .env si présent
if [ -f .env ]; then
  export $(grep -v '^#' .env | grep -v '^$' | xargs)
fi

: "${DATABASE_URL:?DATABASE_URL non défini dans .env}"

EMAIL="test@veil.dev"
KEY="vl_live_testkey1234567890abcdefghijklmnopqrstuvw"
HASH=$(echo -n "$KEY" | shasum -a 256 | awk '{print $1}')

echo "→ Insertion user + clé API dans la base..."

psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<SQL
INSERT INTO users (email, plan)
VALUES ('$EMAIL', 'free')
ON CONFLICT (email) DO NOTHING;

INSERT INTO api_keys (user_id, key_hash, name)
SELECT id, '$HASH', 'test-key'
FROM users
WHERE email = '$EMAIL'
ON CONFLICT (key_hash) DO NOTHING;
SQL

echo ""
echo "✓ Clé créée pour $EMAIL"
echo ""
echo "  Clé API  :  $KEY"
echo "  Header   :  Authorization: Bearer $KEY"
echo ""
echo "Dans Swagger → Authorize → coller : Bearer $KEY"
