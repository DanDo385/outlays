-- +goose Up
-- Core schema (ARCHITECTURE.md Section 3). goose owns the exact DDL text.

CREATE EXTENSION IF NOT EXISTS ltree;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE ingestion_run (
  run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  adapter_id TEXT NOT NULL,
  adapter_version TEXT NOT NULL,
  jurisdiction TEXT NOT NULL,
  fiscal_year TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ,
  status TEXT NOT NULL CHECK (status IN ('running','succeeded','failed')),
  envelope JSONB NOT NULL
);

CREATE TABLE raw_snapshot (
  sha256 TEXT PRIMARY KEY,
  storage_key TEXT NOT NULL,
  url TEXT NOT NULL,
  http_status INT,
  bytes BIGINT NOT NULL,
  fetched_at TIMESTAMPTZ NOT NULL,
  run_id UUID NOT NULL REFERENCES ingestion_run(run_id)
);

CREATE TABLE entity (
  entity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind TEXT NOT NULL CHECK (kind IN ('government','vendor','nonprofit','individual','unknown')),
  canonical_name TEXT NOT NULL,
  uei TEXT,
  ein TEXT,
  jurisdiction TEXT,
  inserted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE entity_alias (
  alias_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_id UUID NOT NULL REFERENCES entity(entity_id),
  name_raw TEXT NOT NULL,
  matched_by TEXT NOT NULL CHECK (matched_by IN ('identifier','rule','model','human')),
  confidence NUMERIC,
  source JSONB NOT NULL,
  inserted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE fiscal_fact (
  fact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES ingestion_run(run_id),
  jurisdiction TEXT NOT NULL,
  fiscal_year TEXT NOT NULL,
  flow TEXT NOT NULL CHECK (flow IN ('revenue','spending')),
  grain TEXT NOT NULL CHECK (grain IN ('transaction','award','aggregate')),
  payer_entity UUID REFERENCES entity(entity_id),
  payee_entity UUID REFERENCES entity(entity_id),
  amount NUMERIC(24,4) NOT NULL,
  currency CHAR(3) NOT NULL DEFAULT 'USD',
  occurred_on DATE,
  description TEXT,
  raw_sha256 TEXT REFERENCES raw_snapshot(sha256),
  derivation_query TEXT NOT NULL,
  fact_hash TEXT NOT NULL,
  supersedes UUID REFERENCES fiscal_fact(fact_id),
  inserted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX fiscal_fact_jur_year_flow_idx ON fiscal_fact (jurisdiction, fiscal_year, flow);
CREATE INDEX fiscal_fact_payee_idx ON fiscal_fact (payee_entity);

CREATE TABLE classification_scheme (
  scheme_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  hierarchical BOOLEAN NOT NULL
);

CREATE TABLE classification_code (
  scheme_id TEXT NOT NULL REFERENCES classification_scheme(scheme_id),
  code TEXT NOT NULL,
  parent_code TEXT,
  name TEXT NOT NULL,
  PRIMARY KEY (scheme_id, code)
);

CREATE TABLE classification_assignment (
  assignment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  fact_id UUID NOT NULL REFERENCES fiscal_fact(fact_id),
  scheme_id TEXT NOT NULL,
  code TEXT NOT NULL,
  assigned_by TEXT NOT NULL CHECK (assigned_by IN ('source','rule','model','human')),
  confidence NUMERIC,
  basis TEXT,
  version INT NOT NULL,
  inserted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (scheme_id, code) REFERENCES classification_code(scheme_id, code)
);

CREATE INDEX classification_assignment_fact_idx ON classification_assignment (fact_id);
CREATE INDEX classification_assignment_scheme_code_idx ON classification_assignment (scheme_id, code);

CREATE TABLE control_total (
  jurisdiction TEXT NOT NULL,
  fiscal_year TEXT NOT NULL,
  flow TEXT NOT NULL,
  official_total NUMERIC(24,4) NOT NULL,
  raw_sha256 TEXT REFERENCES raw_snapshot(sha256),
  derivation_query TEXT NOT NULL,
  inserted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (jurisdiction, fiscal_year, flow)
);

CREATE TABLE lead (
  lead_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id TEXT NOT NULL,
  fact_ids UUID[] NOT NULL,
  score NUMERIC,
  generated_query TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','reviewed','published','dismissed')),
  reviewer TEXT,
  review_note TEXT,
  inserted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS lead;
DROP TABLE IF EXISTS control_total;
DROP TABLE IF EXISTS classification_assignment;
DROP TABLE IF EXISTS classification_code;
DROP TABLE IF EXISTS classification_scheme;
DROP TABLE IF EXISTS fiscal_fact;
DROP TABLE IF EXISTS entity_alias;
DROP TABLE IF EXISTS entity;
DROP TABLE IF EXISTS raw_snapshot;
DROP TABLE IF EXISTS ingestion_run;
