"""Validate documents against the canonical Outlays JSON Schema (draft 2020-12).

Uses the schema directly (not the generated Pydantic models) so conditional rules such as
"transaction/award grain requires rawSha256" are enforced identically to the TS and Go
validators.
"""

from __future__ import annotations

import json
from functools import lru_cache
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator
from jsonschema import ValidationError as _JsonSchemaValidationError


class ValidationError(Exception):
    """Raised when a document fails contract validation; ``errors`` lists messages."""

    def __init__(self, def_name: str, errors: list[str]) -> None:
        self.def_name = def_name
        self.errors = errors
        super().__init__(f"{def_name} failed contract validation: {'; '.join(errors)}")


def _find_schema() -> Path:
    """Walk up from this file to the repo root and locate the canonical schema."""
    here = Path(__file__).resolve()
    for parent in here.parents:
        candidate = parent / "packages" / "contract" / "schemas" / "fiscal.schema.json"
        if candidate.is_file():
            return candidate
    raise FileNotFoundError("could not locate packages/contract/schemas/fiscal.schema.json")


SCHEMA_PATH = _find_schema()


@lru_cache(maxsize=1)
def load_schema() -> dict[str, Any]:
    return json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))


@lru_cache(maxsize=None)
def _validator_for(def_name: str) -> Draft202012Validator:
    schema = load_schema()
    if def_name not in schema["$defs"]:
        raise KeyError(f"unknown contract type '{def_name}'")
    # Wrap so internal #/$defs/... references resolve within this document.
    wrapper = {"allOf": [{"$ref": f"#/$defs/{def_name}"}], "$defs": schema["$defs"]}
    return Draft202012Validator(wrapper, format_checker=Draft202012Validator.FORMAT_CHECKER)


def is_valid(def_name: str, data: Any) -> bool:
    return _validator_for(def_name).is_valid(data)


def errors_for(def_name: str, data: Any) -> list[str]:
    errs: list[_JsonSchemaValidationError] = sorted(
        _validator_for(def_name).iter_errors(data), key=lambda e: list(e.path)
    )
    return [f"/{'/'.join(str(p) for p in e.path)} {e.message}" for e in errs]


def validate(def_name: str, data: Any) -> None:
    """Raise ``ValidationError`` if ``data`` does not satisfy the named contract type."""
    errs = errors_for(def_name, data)
    if errs:
        raise ValidationError(def_name, errs)
