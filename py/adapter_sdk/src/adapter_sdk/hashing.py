"""Hashing for the adapter pipeline (parity with the TS SDK).

    rawHash    = SHA-256 over the exact upstream bytes (captured before parsing).
    factHash   = SHA-256 over RFC 8785 (JCS) canonical JSON of a fact's content.
    resultHash = SHA-256 over JCS of the facts (sorted by factHash), stripped of volatile
                 fields.

The strip+sort rule MUST stay byte-identical to the TS SDK and the Go conformance harness.
"""

from __future__ import annotations

import hashlib
from typing import Any

import rfc8785

# Volatile fact fields excluded from resultHash (DB-assigned, non-deterministic).
VOLATILE_FACT_FIELDS = ("factId", "runId", "insertedAt")
# Fact fields excluded from factHash (identity = monetary content + provenance).
_FACT_HASH_EXCLUDE = ("factId", "runId", "insertedAt", "factHash", "assignments")


def sha256_hex(data: bytes | str) -> str:
    if isinstance(data, str):
        data = data.encode("utf-8")
    return hashlib.sha256(data).hexdigest()


def jcs(value: Any) -> bytes:
    """RFC 8785 (JCS) canonical JSON bytes."""
    return rfc8785.dumps(value)


def jcs_sha256(value: Any) -> str:
    return sha256_hex(jcs(value))


def _omit(obj: dict[str, Any], keys: tuple[str, ...]) -> dict[str, Any]:
    return {k: v for k, v in obj.items() if k not in keys and v is not None}


def compute_fact_hash(fact: dict[str, Any]) -> str:
    """Deterministic content hash of a single fact (excludes volatile + assignment fields)."""
    return jcs_sha256(_omit(fact, _FACT_HASH_EXCLUDE))


def compute_result_hash(facts: list[dict[str, Any]]) -> str:
    """Drop volatile fields, sort by factHash ascending, then JCS + SHA-256."""
    stripped = [_omit(f, VOLATILE_FACT_FIELDS) for f in facts]
    stripped.sort(key=lambda f: str(f.get("factHash", "")))
    return jcs_sha256(stripped)
