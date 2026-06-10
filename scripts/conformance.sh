#!/usr/bin/env bash
# Build both toy adapters (TS + Python) and run the Go conformance harness against each.
# This is the S2 acceptance check: a toy adapter passes conformance in both languages.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== building TS toy adapter =="
pnpm install --frozen-lockfile >/dev/null
pnpm --filter @outlays/contract build >/dev/null
pnpm --filter @outlays/adapter-sdk build >/dev/null
pnpm --filter @outlays/adapter-toy-fixture build >/dev/null

echo "== syncing Python SDK =="
( cd py/adapter_sdk && uv sync >/dev/null )

TS_CLI="$ROOT/packages/adapters/toy-fixture/dist/cli.js"
PY_CLI="$ROOT/py/adapter_sdk/.venv/bin/toy-fixture"

echo "== conformance: TS toy adapter =="
( cd core && go run ./cmd/conformance --cmd "node $TS_CLI" --year 2024-25 )

echo "== conformance: Python toy adapter =="
( cd core && go run ./cmd/conformance --cmd "$PY_CLI" --year 2024-25 )

echo "== OK: both adapters passed conformance =="
