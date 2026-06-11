#!/usr/bin/env bash
# Local S12 acceptance: anchor a run on anvil and verify independently (D31).
# Uses anvil account #0 (documented in .env.example). Requires compose Postgres + anvil on :8545.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RUN_ID="${1:?usage: $0 <run-id> [registry-address]}"
REGISTRY="${2:-${ANCHOR_REGISTRY_ADDRESS:-}}"

export DATABASE_URL="${DATABASE_URL:-postgres://fiscal_owner:change_me_too@localhost:5433/fiscal?sslmode=disable}"
export ANCHOR_RPC_URL="${ANCHOR_RPC_URL:-http://localhost:8545}"
export ANCHOR_FROM="${ANCHOR_FROM:-0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266}"

if [[ -z "$REGISTRY" ]]; then
  echo "deploying AnchorRegistry to anvil..." >&2
  REGISTRY="$(ETH_FROM=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 \
    forge create "$ROOT/contracts/src/AnchorRegistry.sol:AnchorRegistry" \
    --rpc-url "$ANCHOR_RPC_URL" --unlocked --broadcast 2>&1 | awk '/Deployed to:/ {print $3}')"
fi
export ANCHOR_REGISTRY_ADDRESS="$REGISTRY"

(cd "$ROOT/core" && go build -o /tmp/outlays-anchor ./cmd/anchor)
/tmp/outlays-anchor run --run-id "$RUN_ID"
python3 "$ROOT/scripts/verify_anchor.py" --run-id "$RUN_ID"
