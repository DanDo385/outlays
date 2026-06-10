# Migrations

goose migrations own the exact DDL text for the schema in ARCHITECTURE.md Section 3.

Created in **S4**: full DDL, `REVOKE UPDATE, DELETE` + `BEFORE UPDATE OR DELETE` reject
triggers, the `app_rw` runtime role, and seed data (schemes + COFOG 01–10). This directory
is intentionally empty until then.
