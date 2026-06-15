#!/usr/bin/env bash
# Build both toy adapters (TS + Python) and run the Go conformance harness against each.
# This is the S2 acceptance check: a toy adapter passes conformance in both languages.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== building TS adapters + SDK =="
pnpm install --frozen-lockfile >/dev/null
pnpm --filter @outlays/contract build >/dev/null
pnpm --filter @outlays/adapter-sdk build >/dev/null
pnpm --filter @outlays/adapter-toy-fixture build >/dev/null
pnpm --filter @outlays/adapter-us-ca-procurement build >/dev/null
pnpm --filter @outlays/adapter-us-fed-usaspending build >/dev/null

echo "== syncing Python SDK =="
( cd py/adapter_sdk && uv sync >/dev/null )

TS_CLI="$ROOT/packages/adapters/toy-fixture/dist/cli.js"
PY_CLI="$ROOT/py/adapter_sdk/.venv/bin/toy-fixture"
CA_CLI="$ROOT/packages/adapters/us-ca-procurement/dist/cli.js"
CA_FIX="$ROOT/packages/adapters/us-ca-procurement/fixtures/replay"
FED_CLI="$ROOT/packages/adapters/us-fed-usaspending/dist/cli.js"
FED_FIX="$ROOT/packages/adapters/us-fed-usaspending/fixtures/replay"

echo "== conformance: TS toy adapter =="
( cd core && go run ./cmd/conformance --cmd "node $TS_CLI" --year 2024-25 )

echo "== conformance: Python toy adapter =="
( cd core && go run ./cmd/conformance --cmd "$PY_CLI" --year 2024-25 )

echo "== conformance: California procurement adapter (replay, offline) =="
( cd core && OUTLAYS_REPLAY_DIR="$CA_FIX" OUTLAYS_MAX_PAGES=1 \
    go run ./cmd/conformance --cmd "node $CA_CLI" --year 2014-15 )

echo "== conformance: Federal USAspending adapter (replay, offline) =="
( cd core && OUTLAYS_REPLAY_DIR="$FED_FIX" OUTLAYS_MAX_PAGES=1 \
    go run ./cmd/conformance --cmd "node $FED_CLI" --year 2025 )

echo "== OK: all adapters passed conformance =="
