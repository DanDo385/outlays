"""Python adapter SDK for Outlays.

Provides fetch-with-snapshot, JCS (RFC 8785) hashing, contract validation, and a CLI
scaffold so a contributor implements only ``list_years()`` and ``fetch_year()``
(ARCHITECTURE.md Section 4).
"""

__version__ = "0.0.0"

# Adapter CLI exit codes (Section 4). Defined before importing submodules that consume them.
EXIT_SUCCESS = 0
EXIT_UNEXPECTED = 1
EXIT_SOURCE_UNAVAILABLE = 2
EXIT_CONTRACT_INVALID = 3

from adapter_sdk.hashing import (  # noqa: E402
    compute_fact_hash,
    compute_result_hash,
    jcs,
    jcs_sha256,
    sha256_hex,
)
from adapter_sdk.run import (  # noqa: E402
    FetchContext,
    FetchResult,
    SourceUnavailableError,
    run_adapter,
)
from adapter_sdk.snapshot import RawSnapshotRef, write_snapshot  # noqa: E402

__all__ = [
    "EXIT_SUCCESS",
    "EXIT_UNEXPECTED",
    "EXIT_SOURCE_UNAVAILABLE",
    "EXIT_CONTRACT_INVALID",
    "sha256_hex",
    "jcs",
    "jcs_sha256",
    "compute_fact_hash",
    "compute_result_hash",
    "write_snapshot",
    "RawSnapshotRef",
    "FetchContext",
    "FetchResult",
    "SourceUnavailableError",
    "run_adapter",
]
