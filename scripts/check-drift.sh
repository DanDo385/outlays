#!/usr/bin/env bash
# Drift guard: regenerate all language types and fail if anything changed. Generated types
# must always match the canonical schema (ARCHITECTURE.md Section 4).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GENERATED=(
  "packages/contract/src/generated/types.ts"
  "py/adapter_sdk/src/adapter_sdk/contract/models.py"
  "core/internal/contract/types.go"
)

bash scripts/codegen.sh

if ! git diff --quiet -- "${GENERATED[@]}"; then
  echo "ERROR: generated types are out of date. Run scripts/codegen.sh and commit the result." >&2
  git --no-pager diff -- "${GENERATED[@]}" >&2
  exit 1
fi
echo "[drift] generated types are in sync with the schema."
