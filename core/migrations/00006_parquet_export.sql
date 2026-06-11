-- +goose Up
-- Registry of content-addressed Parquet exports (task S10, D29). One row per export of a
-- (jurisdiction, fiscal_year) partition: four Parquet artifacts (facts, assignments,
-- classification codes, entities) named by the sha256 of their bytes. Append-only like every
-- data table; the newest row is the partition's current snapshot. app_rw gets SELECT+INSERT
-- via the default privileges in 00003.

CREATE TABLE parquet_export (
  export_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  jurisdiction TEXT NOT NULL,
  fiscal_year TEXT NOT NULL,
  facts_sha256 TEXT NOT NULL,
  facts_key TEXT NOT NULL,
  facts_rows BIGINT NOT NULL,
  assignments_sha256 TEXT NOT NULL,
  assignments_key TEXT NOT NULL,
  assignments_rows BIGINT NOT NULL,
  codes_sha256 TEXT NOT NULL,
  codes_key TEXT NOT NULL,
  codes_rows BIGINT NOT NULL,
  entities_sha256 TEXT NOT NULL,
  entities_key TEXT NOT NULL,
  entities_rows BIGINT NOT NULL,
  exported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX parquet_export_partition_idx ON parquet_export (jurisdiction, fiscal_year, exported_at DESC);

CREATE TRIGGER parquet_export_append_only BEFORE UPDATE OR DELETE ON parquet_export
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS parquet_export_append_only ON parquet_export;
DROP TABLE IF EXISTS parquet_export;
