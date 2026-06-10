# Outlays — Architecture (normative spec)

This document is **normative**. On any conflict between code, comments, or other docs and
this file, this file wins. Changes to the spec are proposed as Decision Log entries
(bottom of this file), not silently edited into code.

---

## 1. Hard Rules (binding on every task, no exceptions)

1. **Provenance or nothing.** Never emit or store a monetary figure without full
   provenance: `raw_sha256` + `derivation_query`. No provenance, no row.
2. **Money is never a float.**
   - JSON: decimal string matching `^-?\d{1,18}(\.\d{1,4})?$` plus an ISO 4217 currency.
   - Postgres: `NUMERIC(24,4)`.
   - TypeScript: `bigint` minor units or a decimal library — never JS `number`.
   - Go: `shopspring/decimal`.
   - JS `number` is forbidden for amounts.
3. **Hashing.**
   - `rawHash` = SHA-256 over the **exact bytes** of each upstream HTTP response body,
     captured *before* parsing.
   - Derived-document hashes = SHA-256 over **RFC 8785 (JCS)** canonical JSON.
   - `JSON.stringify` for hashing is forbidden (key-order dependent).
4. **Append-only, enforced.** All data tables are append-only. Enforced by `REVOKE UPDATE,
   DELETE` from the app role **plus** `BEFORE UPDATE OR DELETE` triggers that raise.
   Corrections are new rows (`supersedes` / `version` columns). History is evidence.
5. **Never invent a classification.**
   - `assigned_by='source'` means the government's own coding.
   - `'rule'` requires a rule id with a citation.
   - `'model'` requires a model version recorded in `basis`.
   - `'human'` requires a reviewer handle.
   - Otherwise leave unassigned. Unmapped is honest.
6. **Leads are facts, never accusations.** Leads (anomaly flags) are facts plus
   statistical context. No lead is publicly reachable unless a **human** set
   `status='published'`. The system never asserts corruption, fraud, or unfairness about a
   named party.
7. **No named-individual payroll** rows reachable by any public endpoint. Individual
   compensation is ingested only as aggregates.
8. **Strict input validation.** Fiscal-year params match `^\d{4}(-\d{2})?$`. Upstream query
   identifiers (resource ids, column names) come **only** from constants in source code.
   No runtime input is ever interpolated into upstream queries.
9. **Upstream etiquette.** User-Agent `outlays/<version> (+<repo URL>)`, default
   1 req/s per host, exponential backoff with jitter, honor `429` and ETags. **CI never
   calls live government APIs; CI uses recorded fixtures only.**
10. **Secrets via env only.** Commit `.env.example`; never commit keys.

---

## 2. Pinned Toolchain

- **Node** 22 LTS, **TypeScript** 5.x strict, **pnpm** workspaces.
- **Go** 1.23+, single module under `/core`, **chi** router, **goose** migrations, **slog**
  JSON logging.
- **Python** 3.12 + **Pydantic v2** + **uv** (adapter SDK parity).
- **Solidity** 0.8.2x with **Foundry**.
- **PostgreSQL** 16 (extensions: `ltree`, `pgcrypto`; `pgvector` deferred).
- **Object storage**: S3 API (MinIO local via docker compose).
- **Canonical JSON**: npm `canonicalize`, pypi `rfc8785`, Go
  `cyberphone/json-canonicalization`.
- **Analytical path**: Parquet + DuckDB (task S10).
- **Refused until pain**: Kafka, microservices, GraphQL, document DBs, Protobuf wire format.

---

## 3. Data Model (the core of the system)

Single idea: atomic **FACTS** plus any number of **CLASSIFICATION ASSIGNMENTS**. Category
trees are **computed views** (`GROUP BY` over one scheme), never storage. This is what lets
the same dollars be viewed by department, by function, by object class, or by vendor, and
lets cross-cuts work (e.g. total employee healthcare cost buried across police, education,
and postal budgets; total paid to one vendor across all categories).

The goose migrations own the exact DDL text. The normative *shape*:

### Tables

- `ingestion_run` — one row per adapter execution; holds the `envelope` JSONB and a
  `status` of `running | succeeded | failed`.
- `raw_snapshot` — content-addressed by `sha256` (PK); points at the object-store
  `storage_key`, the source `url`, `http_status`, `bytes`, `fetched_at`, and `run_id`.
- `entity` — `kind` in `government | vendor | nonprofit | individual | unknown`;
  `canonical_name`, optional `uei` / `ein` / `jurisdiction`.
- `entity_alias` — append-only alias rows; `matched_by` in
  `identifier | rule | model | human`, optional `confidence`, `source` JSONB.
  **Resolution policy:** UEI/EIN identifier match is authoritative; name-only matches carry
  confidence; merges are append-only alias additions, never destructive.
- `fiscal_fact` — the atom. `flow` in `revenue | spending`; `grain` in
  `transaction | award | aggregate`; `amount NUMERIC(24,4)` + `currency CHAR(3)`; optional
  `payer_entity` / `payee_entity`; `raw_sha256` → `raw_snapshot`; `derivation_query`;
  `fact_hash`; optional `supersedes` → `fiscal_fact`.
  Indexes on `(jurisdiction, fiscal_year, flow)` and `(payee_entity)`.
  **Aggregate grain is legitimate** (a published category total *is* a fact at coarse
  grain). Never fabricate finer grain from aggregates.
- `classification_scheme` — `scheme_id` PK, `name`, `hierarchical` bool.
- `classification_code` — `(scheme_id, code)` PK, `parent_code`, `name`.
- `classification_assignment` — `fact_id` → `fiscal_fact`; `(scheme_id, code)` FK;
  `assigned_by` in `source | rule | model | human`; `confidence`, `basis`, `version`.
- `control_total` — PK `(jurisdiction, fiscal_year, flow)`; `official_total`, provenance
  (`raw_sha256` + `derivation_query`).
  **Coverage** = `sum(transaction+award grain facts) / official_total`. Public, honest, per
  jurisdiction-year. Low coverage is correct behavior early, not a bug.
- `lead` — `rule_id`, `fact_ids UUID[]`, `score`, `generated_query`, `status` in
  `draft | reviewed | published | dismissed`, `reviewer`, `review_note`.

### Seed data

- Schemes: `cofog` (hierarchical), `object_class`, `department`, `fund`, `program`,
  `recipient_type`, `tag`.
- COFOG codes 01–10: 01 General public services, 02 Defence, 03 Public order and safety,
  04 Economic affairs, 05 Environmental protection, 06 Housing and community amenities,
  07 Health, 08 Recreation culture and religion, 09 Education, 10 Social protection.

### Object storage key scheme

```
raw/{jurisdiction}/{dataset}/{fiscalYear}/{sha256}.bin
raw/{jurisdiction}/{dataset}/{fiscalYear}/{sha256}.meta.json   (url, fetchedAt, httpStatus, selected headers)
```

---

## 4. Contract and Adapter Protocol

`packages/contract` holds JSON Schemas (draft 2020-12) as the **single source of truth**
for:

- `SourceRef` `{ jurisdiction, dataset, resourceId, derivationQuery, pulledAt, rawSha256 }`
- `FiscalFact`, `Entity`, `ClassificationAssignment`, `ControlTotal` (mirror the DDL)
- `FiscalYearView` / `FiscalNodeView` (API response payloads only, **not** storage)
- `IngestionEnvelope` `{ envelopeVersion:"1", adapterId, adapterVersion, runId, fetchedAt,
  rawSnapshots[{sha256,url,bytes,httpStatus}], jurisdiction, fiscalYear, resultHash,
  signature: null, signerKeyId: null }`

The `signature` / `signerKeyId` fields exist now so Phase-2 federated contributor
submissions (signed envelopes, core verifies by re-derivation) need no schema rework.

**Codegen** — types are generated, never hand-written; CI drift guard regenerates and fails
on diff:
- TS via `json-schema-to-typescript`
- Python via `datamodel-code-generator`
- Go via `omissis/go-jsonschema`

### Adapter CLI protocol

Adapters are standalone executables in any language:

- `adapter info` → manifest JSON `{adapterId, jurisdiction, datasets[], adapterVersion,
  contractVersion, license, maintainer}`
- `adapter list-years` → JSON array of year strings, descending
- `adapter fetch --year <Y> --raw-dir <DIR> [--out <FILE>]`
  - raw bytes to DIR as `<sha256>.bin` + `<sha256>.meta.json`
  - contract-valid facts document to FILE (default stdout)
  - NDJSON logs `{level,msg,ts}` to stderr

**Exit codes:** `0` success, `2` source unavailable or restricted (a finding, not a crash),
`3` output failed contract validation, `1` anything else.

**SDKs** (`packages/adapter-sdk-ts`, `py/adapter_sdk`) provide fetch-with-snapshot, JCS
hashing, validation, and CLI scaffold so a contributor writes only `listYears()` and
`fetchYear()`.

`core/cmd/conformance` runs any adapter binary against recorded fixtures and verifies
protocol, schema validity, rawHash correctness, and resultHash determinism. **Passing
conformance is the merge bar for community adapters.**

---

## 5. API Surface (read-only public, Go + chi)

```
GET /v1/jurisdictions
GET /v1/{jur}/years
GET /v1/{jur}/{year}/view?scheme=<scheme_id>&flow=spending&path=...   one level per call
GET /v1/{jur}/{year}/view?scheme=payee&...                            vendor tree
GET /v1/entities?q=...
GET /v1/entities/{id}/flows?year=...
GET /v1/facts?<filters>                                               paged
GET /v1/fact/{id}/provenance        raw snapshot pointer + hashes + derivation_query
GET /v1/{jur}/{year}/coverage
GET /v1/compare?scheme=cofog&code=07&jurisdictions=...
GET /v1/leads?status=published                                        published only, ever
GET /v1/healthz
```

A generated OpenAPI document is committed. The future RAG chatbot is a **client** of this
API and never receives unsourced numbers; **no LLM calls from the web client.**

---

## Decision Log

Append-only. Each entry: decision, rationale, and (when superseded) a pointer forward.

- **D1 — Fact-dimension model over a single tree.** Store atomic facts + N classification
  assignments; category trees are computed `GROUP BY` views. Enables department / function /
  object-class / vendor pivots and cross-cuts over the same dollars without remodeling.
- **D2 — Decimal-string money on the wire, `NUMERIC(24,4)` at rest.** Floats lose cents and
  are non-deterministic to hash. JS `number` banned for amounts.
- **D3 — JCS (RFC 8785) hashing for derived documents.** Canonical, key-order-independent.
  `JSON.stringify` banned for hashing. Raw upstream bytes hashed before parsing.
- **D4 — Raw-bytes snapshots, content-addressed.** Every figure traces to the exact upstream
  response bytes by `sha256`, stored in object storage with a sidecar meta file.
- **D5 — Append-only, enforced at the database.** `REVOKE` + reject triggers, not just
  convention. Corrections chain via `supersedes`. History is evidence.
- **D6 — Centralized orchestration now, signature-ready envelopes for later.** Phase 0/1
  run a single trusted orchestrator; `IngestionEnvelope` already carries `signature` /
  `signerKeyId` so Phase-2 federated submissions verify by re-derivation with no schema
  rework.
- **D7 — Leads gated on human review.** Nothing reaches `/v1/leads` unless a human set
  `status='published'`. Neutral method: facts + statistical context, never accusations.
- **D8 — Bulk-first roadmap.** After California end-to-end, prioritize bulk archives
  (USAspending, IRS 990, FAC, SAM) over per-API scraping to maximize coverage per unit
  effort.
- **D9 — Parquet + DuckDB analytical path.** Content-addressed Parquet partitions behind the
  same API via an internal engine flag; Postgres remains the system of record.
- **D10 — pgvector deferred.** No semantic search in the core path yet; revisit with the RAG
  client.
- **D11 — One canonical schema file with `$defs`.** The contract is a single
  `fiscal.schema.json`; per-language types are generated from it (TS json-schema-to-typescript,
  Python datamodel-code-generator, Go go-jsonschema) and held in sync by a CI drift guard.
  Single artifact avoids cross-file `$ref` friction across three generators. (S1)
- **D12 — Validate against the schema, not the generated models.** ajv / `jsonschema` /
  `santhosh-tekuri` run the canonical JSON Schema directly so conditional rules (transaction/
  award grain ⇒ `rawSha256`) and enums are enforced identically in all three languages;
  generated models alone would silently miss the conditionals. (S1)
- **D13 — `SchemeId` is a closed enum in the contract.** Mirrors the DB FK to
  `classification_scheme`; an unknown scheme fails pure schema validation. Adding a per-source
  scheme is a deliberate contract change + regen, not a runtime free-for-all. (S1)
- **D16 — California facts at line-item award grain.** The "Purchase Order Data" dataset is
  one row per PO line item; the adapter emits one `award`-grain fact per row (`amount` = that
  row's Total Price), not an aggregated per-PO total. Aggregation would fabricate or discard
  line detail; the source's own grain is kept. `derivationQuery` pins the row `_id`. (S3)
- **D17 — Deterministic provisional vendor ids; no fuzzy merging.** `payeeEntity` =
  UUIDv5 of the normalized supplier name (trim + collapse whitespace + uppercase only). Exact
  normalized-name match maps to one entity; differing raw spellings become append-only aliases
  (`matchedBy='rule'`, `confidence=0.5`). Deterministic ids keep `factHash` reproducible and
  let one vendor be queried across departments without a central resolution step. Real
  resolution (UEI/EIN authority) arrives in Phase 1. (S3)
- **D18 — Per-source classification schemes.** `us_ca_department` and `us_ca_acquisition_type`
  are added to the closed `SchemeId` enum and assigned with `assignedBy='source'` — the
  government's own coding, not an invented taxonomy. Generic `department` stays available for
  later cross-jurisdiction normalization. (S3)
- **D19 — HTTP record/replay for offline fixtures.** The SDK fetch layer runs live, records to
  a fixture dir (`OUTLAYS_RECORD_DIR`), or replays from one (`OUTLAYS_REPLAY_DIR`, used by CI —
  never the network). Fixtures are content-addressed bodies + a URL→file index. A page cap
  (`OUTLAYS_MAX_PAGES`) keeps recorded fixtures small; replay sets the same cap. (S3)
- **D15 — `resultHash` is deterministic over the fact set.** An adapter's `--out` document is
  `AdapterOutput` = `{ envelope, facts, entities?, entityAliases? }`. `envelope.resultHash` =
  SHA-256 over RFC 8785 (JCS) canonical JSON of `facts` after (a) dropping volatile fields
  (`factId`, `runId`, `insertedAt`) and (b) sorting by `factHash` ascending. `factHash` =
  JCS+SHA-256 of a fact's content excluding volatile fields and `assignments`. This makes
  `resultHash` independent of emit order and free of per-run noise, so the same fixtures
  reproduce the same hash — verified byte-identical across the TS SDK, Python SDK, and Go
  conformance harness. The SDKs and the harness MUST keep this rule in lockstep. (S2)
- **D14 — Project renamed `fiscal-warehouse` → `outlays`.** The original name in the build
  prompt was "Fiscal Warehouse" (kebab `fiscal-warehouse`). The project is now **Outlays**:
  npm scope `@outlays/*`, Go module `github.com/djmagro/outlays/core`, User-Agent
  `outlays/<version>`, schema `$id` host `outlays.org`. The domain term *fiscal* (e.g.
  `fiscal_fact`, `fiscalYear`, `fiscal.schema.json`) is unchanged — it denotes government
  finance, not the brand. This entry is intentionally the sole remaining occurrence of the
  old name. (post-S1)
