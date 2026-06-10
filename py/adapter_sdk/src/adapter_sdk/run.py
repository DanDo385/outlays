"""CLI scaffold for Python adapters (parity with the TS SDK). A contributor implements only
``list_years()`` and ``fetch_year(ctx)``; this module provides the info / list-years / fetch
commands, raw-snapshot capture, hashing, envelope assembly, contract validation, NDJSON
stderr logs, and exit codes (ARCHITECTURE.md Section 4)."""

from __future__ import annotations

import json
import re
import sys
import uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Callable, Protocol

from adapter_sdk import (
    EXIT_CONTRACT_INVALID,
    EXIT_SOURCE_UNAVAILABLE,
    EXIT_SUCCESS,
    EXIT_UNEXPECTED,
)
from adapter_sdk.contract.validate import errors_for
from adapter_sdk.hashing import compute_fact_hash, compute_result_hash
from adapter_sdk.snapshot import RawSnapshotRef, write_snapshot

FISCAL_YEAR = re.compile(r"^\d{4}(-\d{2})?$")

ENVELOPE_VERSION = "1"
CONTRACT_VERSION = "0.3.0"


class SourceUnavailableError(Exception):
    """Raised when the upstream source is unavailable or restricted (a finding ⇒ exit 2)."""


def _log(level: str, msg: str) -> None:
    sys.stderr.write(
        json.dumps({"level": level, "msg": msg, "ts": datetime.now(timezone.utc)
                    .strftime("%Y-%m-%dT%H:%M:%S.%fZ")})
        + "\n"
    )


@dataclass
class FetchContext:
    year: str
    raw_dir: str
    snapshots: list[RawSnapshotRef] = field(default_factory=list)

    def snapshot(self, *, url: str, data: bytes, http_status: int,
                 headers: dict[str, str] | None = None) -> RawSnapshotRef:
        ref = write_snapshot(self.raw_dir, url=url, data=data, http_status=http_status, headers=headers)
        self.snapshots.append(ref)
        return ref

    def log(self, level: str, msg: str) -> None:
        _log(level, msg)


@dataclass
class FetchResult:
    facts: list[dict[str, Any]]
    entities: list[dict[str, Any]] | None = None
    entity_aliases: list[dict[str, Any]] | None = None
    # Official published totals (coverage denominators) captured by this run, with provenance.
    control_totals: list[dict[str, Any]] | None = None


class AdapterDefinition(Protocol):
    manifest: dict[str, Any]

    def list_years(self) -> list[str]: ...

    def fetch_year(self, ctx: FetchContext) -> FetchResult: ...


def _parse_flags(argv: list[str]) -> dict[str, str]:
    flags: dict[str, str] = {}
    i = 0
    while i < len(argv):
        a = argv[i]
        if a.startswith("--"):
            key = a[2:]
            nxt = argv[i + 1] if i + 1 < len(argv) else None
            if nxt is not None and not nxt.startswith("--"):
                flags[key] = nxt
                i += 1
            else:
                flags[key] = "true"
        i += 1
    return flags


def _run_fetch(adapter: AdapterDefinition, flags: dict[str, str]) -> int:
    year = flags.get("year")
    raw_dir = flags.get("raw-dir")
    out = flags.get("out")

    if not year or not FISCAL_YEAR.match(year):
        _log("error", f"invalid or missing --year (must match {FISCAL_YEAR.pattern})")
        return EXIT_UNEXPECTED
    if not raw_dir:
        _log("error", "missing --raw-dir")
        return EXIT_UNEXPECTED

    ctx = FetchContext(year=year, raw_dir=raw_dir)
    try:
        result = adapter.fetch_year(ctx)
    except SourceUnavailableError as err:
        _log("error", f"source unavailable: {err}")
        return EXIT_SOURCE_UNAVAILABLE
    except Exception as err:  # noqa: BLE001 - protocol: any other error ⇒ exit 1
        _log("error", f"fetch failed: {err}")
        return EXIT_UNEXPECTED

    facts: list[dict[str, Any]] = []
    for f in result.facts:
        fact = {k: v for k, v in f.items() if v is not None}
        fact["factHash"] = compute_fact_hash(fact)
        facts.append(fact)

    envelope = {
        "envelopeVersion": ENVELOPE_VERSION,
        "adapterId": adapter.manifest["adapterId"],
        "adapterVersion": adapter.manifest["adapterVersion"],
        "runId": str(uuid.uuid4()),
        "fetchedAt": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
        "rawSnapshots": [
            {"sha256": s.sha256, "url": s.url, "bytes": s.bytes, "httpStatus": s.httpStatus}
            for s in ctx.snapshots
        ],
        "jurisdiction": adapter.manifest["jurisdiction"],
        "fiscalYear": year,
        "resultHash": compute_result_hash(facts),
        "signature": None,
        "signerKeyId": None,
    }

    doc: dict[str, Any] = {"envelope": envelope, "facts": facts}
    if result.entities is not None:
        doc["entities"] = result.entities
    if result.entity_aliases is not None:
        doc["entityAliases"] = result.entity_aliases
    if result.control_totals is not None:
        doc["controlTotals"] = result.control_totals

    errs = errors_for("AdapterOutput", doc)
    if errs:
        _log("error", "output failed contract validation: " + "; ".join(errs))
        return EXIT_CONTRACT_INVALID

    text = json.dumps(doc, indent=2) + "\n"
    if out and out != "-":
        Path(out).write_text(text, encoding="utf-8")
    else:
        sys.stdout.write(text)
    _log("info", f"wrote {len(facts)} fact(s), {len(ctx.snapshots)} snapshot(s), "
                 f"resultHash={envelope['resultHash']}")
    return EXIT_SUCCESS


def run_adapter(adapter: AdapterDefinition, argv: list[str] | None = None) -> int:
    argv = list(sys.argv[1:] if argv is None else argv)
    cmd = argv[0] if argv else ""
    rest = argv[1:]
    try:
        if cmd == "info":
            sys.stdout.write(json.dumps(adapter.manifest, indent=2) + "\n")
            return EXIT_SUCCESS
        if cmd == "list-years":
            years = sorted(adapter.list_years(), reverse=True)
            sys.stdout.write(json.dumps(years) + "\n")
            return EXIT_SUCCESS
        if cmd == "fetch":
            return _run_fetch(adapter, _parse_flags(rest))
        _log("error", f"unknown command '{cmd}' (expected info | list-years | fetch)")
        return EXIT_UNEXPECTED
    except Exception as err:  # noqa: BLE001
        _log("error", f"unexpected: {err}")
        return EXIT_UNEXPECTED


def main_for(adapter: AdapterDefinition) -> Callable[[], None]:
    def _main() -> None:
        sys.exit(run_adapter(adapter))

    return _main
