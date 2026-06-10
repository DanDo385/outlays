"""Raw-snapshot writer (parity with the TS SDK): content-addressed by SHA-256 over the exact
bytes, with a sidecar meta file. The .bin is the content address; the meta file is
descriptive only and is never hashed."""

from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

from adapter_sdk.hashing import sha256_hex


@dataclass(frozen=True)
class RawSnapshotRef:
    sha256: str
    url: str
    bytes: int
    httpStatus: int


def write_snapshot(
    raw_dir: str | Path,
    *,
    url: str,
    data: bytes,
    http_status: int,
    headers: dict[str, str] | None = None,
) -> RawSnapshotRef:
    raw_dir = Path(raw_dir)
    raw_dir.mkdir(parents=True, exist_ok=True)
    sha = sha256_hex(data)
    (raw_dir / f"{sha}.bin").write_bytes(data)
    meta = {
        "sha256": sha,
        "url": url,
        "fetchedAt": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
        "httpStatus": http_status,
        "headers": headers or {},
    }
    (raw_dir / f"{sha}.meta.json").write_text(json.dumps(meta, indent=2) + "\n", encoding="utf-8")
    return RawSnapshotRef(sha256=sha, url=url, bytes=len(data), httpStatus=http_status)
