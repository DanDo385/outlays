# Build Tasks

One task at a time. EVR each (Execute, Verify, Report). Do not start S(n+1) while S(n) is
red. Tick a box only after its ACCEPT criteria are verified.

- [x] **S0 — Scaffold + docs.** Layout, toolchain pins, compose stack, CI (lint + typecheck
  + forge build + schema job placeholders), write the four docs.
  - ACCEPT: `pnpm -r build`, `go build ./...`, `forge build`, `docker compose up -d` all
    green on clean checkout; docs present and self-consistent.

- [x] **S1 — Contract package + codegen + drift guard** (Section 4).
  - ACCEPT: invalid fixtures fail in all three languages (float amount; transaction-grain
    fact missing `raw_sha256`; assignment with unknown scheme); valid fixtures pass; drift
    guard green in CI.

- [x] **S2 — Adapter SDKs (TS + Python) + conformance harness** (Section 4).
  - ACCEPT: a toy fixture adapter passes conformance in both languages; `resultHash` is
    identical across two runs on the same fixtures.

- [x] **S3 — California procurement adapter at award grain.** CKAN datastore on
  data.ca.gov, "Purchase Order Data", resource `bb82edc5-9c78-44e2-8947-68ece26197c5`
  (verify live first; record any move in NOTES.md). One fact per PO; `source` assignments
  for department + acquisition type; supplier names as `entity_alias` on provisional vendor
  entities (`matched_by='rule'`, simple normalization, no fuzzy merging). Record fixtures.
  - ACCEPT: conformance green; one vendor's facts span multiple departments in a single
    query; fixture rerun reproduces identical `fact_hash` values.

- [x] **S4 — Stores + migrations.** Full DDL via goose incl. REVOKE + reject triggers and
  `app_rw` role; object-store writer; Postgres batched COPY writer.
  - ACCEPT: integration test ingests the S3 output on compose; UPDATE/DELETE raises on every
    data table; a correction row chains via `supersedes`.

- [x] **S5 — Orchestrator.** `core/cmd/orchestrator run --adapter <path> --year <Y>`: exec
    per protocol, validate envelope + facts, persist, mark run status. errgroup concurrency;
    rate limit + backoff + project UA.
  - ACCEPT: one command ingests CA 2024-25 into a fresh stack; forced-failure and
    exit-code-2 paths recorded correctly on `ingestion_run`.

- [x] **S6 — Read API** (Section 5; leads stubbed until S11). Provenance endpoint joins
    fact → raw_snapshot → storage key.
  - ACCEPT: curl walk on real data — department view, acquisition-type view, payee view over
    the SAME facts; flows endpoint shows one vendor across departments; provenance resolves
    to a real object-store key. Commit OpenAPI doc.

- [x] **S7 — Web UI** (Next.js, `packages/web`). Two-sided ledger; masthead year switcher;
    balance ribbon; money-in left / money-out right; drill-down; dimension switcher
    (department / acquisition type / vendor); provenance drawer; coverage badge; illustrative
    marker for non-live data; no client-side LLM calls.
  - ACCEPT: pivot the same CA spending across three dimensions and open a real PO-level
    provenance drawer from the UI.

- [x] **S8 — Control totals + coverage.** Ingest CA official enacted-budget total as
    `control_total` with provenance; coverage endpoint + badge wiring.
  - ACCEPT: coverage returns numerator and denominator each with provenance links; the
    honest low number renders in the UI.

- [x] **S9 — Classification ingest.** Loader for reviewed mapping JSON (`data/cofog/*.json`,
    `{sourceCategory: {cofogCode, confidence, basis}}`) into versioned
    `classification_assignment` rows. Never invent mappings; against a fixture if no file.
  - ACCEPT: with a fixture mapping applied, the cofog view endpoint returns a rollup and
    unmapped categories are listed explicitly.

- [x] **S10 — Parquet + DuckDB path.** Export `fiscal_fact` partitions
    (jurisdiction/fiscal_year) as content-addressed Parquet; DuckDB query path behind the
    same API via an internal engine flag; engine correctness check.
  - ACCEPT: a vendor aggregation returns identical totals from Postgres and DuckDB with no
    response-shape change.

- [x] **S11 — Leads scaffold (private).** Rules as versioned SQL with metadata (rule_id,
    citation, required fields); runner writes `draft`; review CLI (list, inspect, set status
    with reviewer handle). Exactly ONE rule end to end on CA (e.g. vendor concentration
    within a department-year), citing its basis.
  - ACCEPT: drafts invisible to `/v1/leads`; one CLI-published lead appears with rule
    citation and fact links.

- [ ] **S12 — Anchor layer.** `contracts/AnchorRegistry.sol`: `anchor(runId, merkleRoot,
    uri)` + event, duplicate-runId rejection, forge tests. `core/cmd/anchor` computes the
    Merkle root over sorted `fact_hash` for a run and submits to local anvil; tx ref
    persisted on the run.
  - ACCEPT: forge tests green; end to end on anvil — ingest, anchor, then an independent
    script recomputes the root from the DB and matches the on-chain event.

## Backlog (blocked on research source docs landing in `docs/sources/`)

USAspending bulk ingest; IRS 990 + Federal Audit Clearinghouse ingest with EIN joins; SAM
entity extracts + entity resolution v1 upgrade; CA revenue adapter; healthcare cross-cut
view; additional lead rules; RAG chatbot over `/v1`; federated submission verification;
zkTLS fetch proofs.
