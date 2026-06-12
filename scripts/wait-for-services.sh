#!/usr/bin/env bash
# Wait for compose Postgres and MinIO to accept connections.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
set -a
# shellcheck disable=SC1091
[ -f "$ROOT/.env" ] && . "$ROOT/.env"
set +a

OWNER_USER="${POSTGRES_OWNER_USER:-fiscal_owner}"
DB="${POSTGRES_DB:-fiscal}"
PORT="${POSTGRES_PORT:-5433}"
S3_ENDPOINT="${S3_ENDPOINT:-http://localhost:9000}"

echo "waiting for Postgres on :${PORT}..."
for _ in $(seq 1 60); do
  if docker compose -f "$ROOT/deploy/docker-compose.yml" exec -T postgres \
    pg_isready -U "$OWNER_USER" -d "$DB" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker compose -f "$ROOT/deploy/docker-compose.yml" exec -T postgres \
  pg_isready -U "$OWNER_USER" -d "$DB"

echo "waiting for MinIO at ${S3_ENDPOINT}..."
for _ in $(seq 1 60); do
  if curl -sf "${S3_ENDPOINT}/minio/health/ready" >/dev/null 2>&1; then
    echo "services ready"
    exit 0
  fi
  sleep 1
done

echo "MinIO did not become ready in time" >&2
exit 1
