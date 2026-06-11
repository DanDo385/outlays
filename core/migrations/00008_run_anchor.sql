-- +goose Up
-- On-chain anchor references (task S12, D31). ingestion_run is append-only and written
-- once, so the tx ref is persisted as a separate append-only row keyed to the run, not an
-- UPDATE. The newest row per run is the current anchor; the chain itself rejects duplicate
-- runIds, so extra rows can only record re-verifications of the same root.

CREATE TABLE run_anchor (
  anchor_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES ingestion_run(run_id),
  merkle_root TEXT NOT NULL,        -- 0x-prefixed 32-byte hex, D31 construction
  fact_count BIGINT NOT NULL,
  chain_id BIGINT NOT NULL,
  contract_address TEXT NOT NULL,
  tx_hash TEXT NOT NULL,
  block_number BIGINT,
  anchored_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX run_anchor_run_idx ON run_anchor (run_id, anchored_at DESC);

CREATE TRIGGER run_anchor_append_only BEFORE UPDATE OR DELETE ON run_anchor
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS run_anchor_append_only ON run_anchor;
DROP TABLE IF EXISTS run_anchor;
