#!/usr/bin/env bash
# Create the app login role from .env and grant app_rw (migrations only create app_rw).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
set -a
# shellcheck disable=SC1091
[ -f "$ROOT/.env" ] && . "$ROOT/.env"
set +a

APP_USER="${POSTGRES_USER:-fiscal_app}"
APP_PW="${POSTGRES_PASSWORD:-change_me}"
OWNER_USER="${POSTGRES_OWNER_USER:-fiscal_owner}"
DB="${POSTGRES_DB:-fiscal}"

echo "provisioning app role ${APP_USER}..."
docker compose -f "$ROOT/deploy/docker-compose.yml" exec -T postgres \
  psql -v ON_ERROR_STOP=1 -U "$OWNER_USER" -d "$DB" <<SQL
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${APP_USER}') THEN
    CREATE ROLE ${APP_USER} LOGIN PASSWORD '${APP_PW}';
  ELSE
    ALTER ROLE ${APP_USER} WITH LOGIN PASSWORD '${APP_PW}';
  END IF;
END
\$\$;
GRANT app_rw TO ${APP_USER};
SQL
