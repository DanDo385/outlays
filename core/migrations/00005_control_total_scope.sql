-- +goose Up
-- control_total gains an explicit currency and a coverage-scope label (S8 / D27). scope
-- names what coverage against this denominator means (e.g. "procurement facts vs total
-- budget") so a scope mismatch between ingested facts and the official total is always
-- stated, never implied away. Both columns are NOT NULL; the table is empty everywhere
-- pre-S8, so no backfill is needed (the transient currency default is dropped immediately).

ALTER TABLE control_total
  ADD COLUMN currency CHAR(3) NOT NULL DEFAULT 'USD',
  ADD COLUMN scope TEXT NOT NULL;
ALTER TABLE control_total ALTER COLUMN currency DROP DEFAULT;

-- +goose Down
ALTER TABLE control_total
  DROP COLUMN scope,
  DROP COLUMN currency;
