"""Python adapter SDK for Outlays.

Provides fetch-with-snapshot, JCS (RFC 8785) hashing, contract validation, and a CLI
scaffold so a contributor implements only ``list_years()`` and ``fetch_year()``. Built out
in S2. S0: package marker only.
"""

__version__ = "0.0.0"

# Adapter CLI exit codes (mirrors ARCHITECTURE.md Section 4).
EXIT_SUCCESS = 0
EXIT_UNEXPECTED = 1
EXIT_SOURCE_UNAVAILABLE = 2
EXIT_CONTRACT_INVALID = 3
