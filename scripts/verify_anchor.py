#!/usr/bin/env python3
"""Independent anchor verification (task S12 acceptance).

Recomputes the D31 Merkle root for an ingestion run straight from the database and
compares it against the on-chain Anchored event (and the stored get() view). This script
deliberately shares no code with core/ — it is the third-party reproduction path, written
only from the D31 specification in ARCHITECTURE.md:

    leaves    = the run's fact_hash values (SHA-256 hex) decoded to 32 raw bytes,
                sorted ascending bytewise, duplicates rejected
    leafHash  = SHA-256(0x00 || leaf)
    nodeHash  = SHA-256(0x01 || left || right)
    pairing   = consecutive pairs per level; a trailing odd node is promoted unchanged
    root      = the last remaining node; empty runs are not anchorable

Usage:
    python3 scripts/verify_anchor.py --run-id <uuid> \
        [--database-url postgres://...] [--rpc-url http://localhost:8545] \
        [--address 0x...]

Defaults come from DATABASE_URL / ANCHOR_RPC_URL / ANCHOR_REGISTRY_ADDRESS. Requires
psql (or the outlays compose Postgres container) and Foundry's cast on PATH.

Exit code 0 = MATCH, 1 = MISMATCH or error.
"""

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys

UUID_RE = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
HEX64_RE = re.compile(r"^[0-9a-f]{64}$")


def merkle_root(fact_hashes):
    if not fact_hashes:
        raise SystemExit("no facts for this run; nothing to verify")
    leaves = []
    for h in fact_hashes:
        if not HEX64_RE.match(h):
            raise SystemExit(f"fact hash {h!r} is not 64-char lowercase hex")
        leaves.append(bytes.fromhex(h))
    leaves.sort()
    for a, b in zip(leaves, leaves[1:]):
        if a == b:
            raise SystemExit(f"duplicate fact hash {a.hex()}")
    level = [hashlib.sha256(b"\x00" + leaf).digest() for leaf in leaves]
    while len(level) > 1:
        nxt = [
            hashlib.sha256(b"\x01" + level[i] + level[i + 1]).digest()
            for i in range(0, len(level) - 1, 2)
        ]
        if len(level) % 2 == 1:
            nxt.append(level[-1])  # odd node promoted unchanged
        level = nxt
    return "0x" + level[0].hex()


def db_fact_hashes(database_url, run_id):
    sql = f"SELECT fact_hash FROM fiscal_fact WHERE run_id = '{run_id}' AND fact_hash ~ '^[0-9a-f]{{64}}$'"  # uuid pre-validated
    if shutil.which("psql"):
        cmd = ["psql", database_url, "-At", "-c", sql]
    else:
        cmd = ["docker", "exec", "outlays-postgres-1",
               "psql", "-U", "fiscal_owner", "-d", "fiscal", "-At", "-c", sql]
    out = subprocess.run(cmd, capture_output=True, text=True, check=True).stdout
    return [line.strip() for line in out.splitlines() if line.strip()]


def run_id_bytes32(run_id):
    return "0x" + run_id.replace("-", "") + "0" * 32


def cast(*args):
    return subprocess.run(["cast", *args], capture_output=True, text=True, check=True).stdout


def onchain_event_root(rpc_url, address, run_id):
    topic1 = run_id_bytes32(run_id)
    out = cast("logs", "--json", "--rpc-url", rpc_url, "--address", address,
               "--from-block", "0", "--to-block", "latest",
               "Anchored(bytes32 indexed runId, bytes32 merkleRoot, string uri, address indexed submitter)",
               topic1)
    logs = json.loads(out)
    if not logs:
        raise SystemExit(f"no Anchored event found for run {run_id} at {address}")
    if len(logs) > 1:
        raise SystemExit(f"{len(logs)} Anchored events for one runId — contract invariant broken")
    data = logs[0]["data"]
    if data.startswith("0x"):
        data = data[2:]
    # ABI: data = merkleRoot (32B) || offset(string) || len || bytes — root is word 0.
    return "0x" + data[:64], logs[0]["transactionHash"]


def onchain_get_root(rpc_url, address, run_id):
    out = cast("call", "--rpc-url", rpc_url, address,
               "get(bytes32)(bytes32,string,address,uint64)", run_id_bytes32(run_id))
    return out.strip().splitlines()[0].strip()


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--run-id", required=True)
    ap.add_argument("--database-url", default=os.environ.get("DATABASE_URL", ""))
    ap.add_argument("--rpc-url", default=os.environ.get("ANCHOR_RPC_URL", "http://localhost:8545"))
    ap.add_argument("--address", default=os.environ.get("ANCHOR_REGISTRY_ADDRESS", ""))
    args = ap.parse_args()

    run_id = args.run_id.lower()
    if not UUID_RE.match(run_id):
        raise SystemExit(f"--run-id {args.run_id!r} is not a UUID")
    if not args.address:
        raise SystemExit("--address (or ANCHOR_REGISTRY_ADDRESS) is required")

    hashes = db_fact_hashes(args.database_url, run_id)
    recomputed = merkle_root(hashes)
    event_root, tx = onchain_event_root(args.rpc_url, args.address, run_id)
    get_root = onchain_get_root(args.rpc_url, args.address, run_id)

    print(f"run:               {run_id}")
    print(f"facts in db:       {len(hashes)}")
    print(f"recomputed root:   {recomputed}")
    print(f"on-chain event:    {event_root}  (tx {tx})")
    print(f"on-chain get():    {get_root}")

    if recomputed == event_root == get_root:
        print("MATCH: database recomputation equals the on-chain anchor")
        return 0
    print("MISMATCH: evidence and anchor disagree")
    return 1


if __name__ == "__main__":
    sys.exit(main())
