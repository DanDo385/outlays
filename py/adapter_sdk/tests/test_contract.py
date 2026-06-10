"""Runs the shared contract fixtures (packages/contract/fixtures/cases.json) through the
Python validator. Must agree with the TS and Go validators."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from adapter_sdk.contract import SCHEMA_PATH
from adapter_sdk.contract.validate import errors_for, is_valid

FIXTURES_ROOT = SCHEMA_PATH.parent.parent / "fixtures"
CASES = json.loads((FIXTURES_ROOT / "cases.json").read_text(encoding="utf-8"))["cases"]


@pytest.mark.parametrize("case", CASES, ids=[c["file"] for c in CASES])
def test_fixture_validity(case: dict) -> None:
    data = json.loads((FIXTURES_ROOT / case["file"]).read_text(encoding="utf-8"))
    valid = is_valid(case["def"], data)
    assert valid == case["valid"], (
        f"{case['file']} ({case['def']}): expected valid={case['valid']}, got {valid}; "
        f"errors={errors_for(case['def'], data)}"
    )
