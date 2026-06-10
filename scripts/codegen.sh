#!/usr/bin/env bash
# Regenerate all language types from the canonical contract schema.
# Source of truth: packages/contract/schemas/fiscal.schema.json
# Deterministic: pinned generator versions + no timestamps, so the drift guard is stable.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GO_JSONSCHEMA_VERSION="v0.23.1"

echo "[codegen] TypeScript -> packages/contract/src/generated/types.ts"
pnpm --filter @outlays/contract codegen

echo "[codegen] Python -> py/adapter_sdk/src/adapter_sdk/contract/models.py"
( cd py/adapter_sdk && uv run datamodel-codegen \
    --input ../../packages/contract/schemas/fiscal.schema.json \
    --input-file-type jsonschema \
    --output src/adapter_sdk/contract/models.py \
    --output-model-type pydantic_v2.BaseModel \
    --disable-timestamp \
    --use-standard-collections \
    --use-double-quotes \
    --field-constraints \
    --formatters black isort )

echo "[codegen] Go -> core/internal/contract/types.go"
GOFLAGS=-mod=mod go run "github.com/atombender/go-jsonschema@${GO_JSONSCHEMA_VERSION}" \
    -p contract --only-models \
    -o core/internal/contract/types.go \
    packages/contract/schemas/fiscal.schema.json
gofmt -w core/internal/contract/types.go

echo "[codegen] done"
