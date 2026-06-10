"""Integration test: run the Python toy adapter CLI and check the contract output, the
deterministic resultHash across two runs, and the cross-language golden hash (identical to
the TS toy adapter — a JCS-parity anchor)."""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

from adapter_sdk.contract.validate import is_valid

# Golden resultHash shared with the TS toy adapter (packages/adapters/toy-fixture). If the
# toy dataset changes, update both. A mismatch means the SDKs disagree on JCS hashing.
GOLDEN_RESULT_HASH = "fd28684fd1baf412f2b636d3ac8a58086deac692962767974cff31d1d1683f6f"


def _run(tmp: Path, tag: str) -> dict:
    out = tmp / f"{tag}.json"
    raw = tmp / f"{tag}-raw"
    proc = subprocess.run(
        [sys.executable, "-m", "adapter_sdk.examples.toy_fixture",
         "fetch", "--year", "2024-25", "--raw-dir", str(raw), "--out", str(out)],
        capture_output=True,
        text=True,
    )
    assert proc.returncode == 0, f"adapter exited {proc.returncode}: {proc.stderr}"
    return json.loads(out.read_text(encoding="utf-8"))


def test_toy_adapter_output_and_determinism(tmp_path: Path) -> None:
    doc1 = _run(tmp_path, "run1")
    doc2 = _run(tmp_path, "run2")

    assert is_valid("AdapterOutput", doc1), "output must validate against AdapterOutput"
    assert len(doc1["facts"]) == 3
    assert doc1["envelope"]["resultHash"] == doc2["envelope"]["resultHash"], "resultHash must be deterministic"
    assert doc1["envelope"]["resultHash"] == GOLDEN_RESULT_HASH, "cross-language JCS golden hash"
