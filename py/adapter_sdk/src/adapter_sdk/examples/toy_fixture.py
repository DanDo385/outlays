"""Toy fixture adapter (Python) — byte-for-byte parity with the TS toy adapter
(packages/adapters/toy-fixture). Deterministic and network-free; both implementations
produce the same resultHash, a cross-language check on JCS hashing."""

from __future__ import annotations

import json
import sys
from typing import Any

from adapter_sdk.run import CONTRACT_VERSION, FetchContext, FetchResult, run_adapter

# Exact bytes of the synthetic upstream response (ASCII, no trailing newline). Must match the
# TS toy adapter's RAW string exactly.
RAW = (
    '[{"po":"PO-1","dept":"DGS","vendor":"Acme Supplies Inc","amount":"1000.0000","date":"2024-07-01"},'
    '{"po":"PO-2","dept":"CDCR","vendor":"Beacon Logistics LLC","amount":"25000.5000","date":"2024-08-15"},'
    '{"po":"PO-3","dept":"CDCR","vendor":"Acme Supplies Inc","amount":"750.2500","date":"2024-09-30"}]'
)


class ToyFixtureAdapter:
    manifest: dict[str, Any] = {
        "adapterId": "toy-fixture",
        "jurisdiction": "us-xx",
        "datasets": ["toy"],
        "adapterVersion": "0.1.0",
        "contractVersion": CONTRACT_VERSION,
        "license": "Apache-2.0",
        "maintainer": "outlays",
    }

    def list_years(self) -> list[str]:
        return ["2024-25", "2023-24"]

    def fetch_year(self, ctx: FetchContext) -> FetchResult:
        data = RAW.encode("utf-8")
        snap = ctx.snapshot(url=f"toy://toy/{ctx.year}", data=data, http_status=200)
        rows = json.loads(RAW)

        facts: list[dict[str, Any]] = [
            {
                "jurisdiction": "us-xx",
                "fiscalYear": ctx.year,
                "flow": "spending",
                "grain": "award",
                "amount": r["amount"],
                "currency": "USD",
                "occurredOn": r["date"],
                "description": r["vendor"],
                "rawSha256": snap.sha256,
                "derivationQuery": f"toy:row:{r['po']}",
                "assignments": [
                    {
                        "schemeId": "department",
                        "code": r["dept"],
                        "assignedBy": "source",
                        "version": 1,
                        "basis": "toy: department as coded by source",
                    }
                ],
            }
            for r in rows
        ]

        vendor_names = list(dict.fromkeys(r["vendor"] for r in rows))
        entities = [
            {"kind": "vendor", "canonicalName": name, "jurisdiction": "us-xx"} for name in vendor_names
        ]
        entity_aliases = [
            {"nameRaw": name, "matchedBy": "rule", "source": {"normalized": name.lower()}}
            for name in vendor_names
        ]

        ctx.log("info", f"toy-fixture produced {len(facts)} facts for {ctx.year}")
        return FetchResult(facts=facts, entities=entities, entity_aliases=entity_aliases)


def main() -> None:
    sys.exit(run_adapter(ToyFixtureAdapter()))


if __name__ == "__main__":
    main()
