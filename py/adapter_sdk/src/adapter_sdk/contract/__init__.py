"""Outlays contract: generated Pydantic models + JSON Schema validation.

``models`` is generated from packages/contract/schemas/fiscal.schema.json (drift-guarded).
``validate`` runs the canonical schema directly so conditional rules are enforced.
"""

from adapter_sdk.contract.validate import (
    SCHEMA_PATH,
    ValidationError,
    load_schema,
    validate,
)

__all__ = ["SCHEMA_PATH", "ValidationError", "load_schema", "validate"]
