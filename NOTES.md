# NOTES

Append-only log of discovered constraints and decisions made during the build. Newest at
the bottom.

## S0

- Toolchain present on the build host: Node v24 (spec pins Node 22 LTS — `engines` set to
  `>=22`), pnpm 10.15, Go 1.26, forge 1.5, Docker 28, uv 0.11 (system Python is 3.9; uv
  manages the pinned 3.12). No version conflicts that block S0.
- Repository directory on disk is `outlay/`; the project/brand name is `outlays`, used for
  the module path, npm scope (`@outlays/*`), User-Agent, and docs. The spec's layout is
  applied inside the existing directory.
- LICENSE replaced from MIT → Apache-2.0 per Section 6.
- `packages/web` is scaffolded as a minimal buildable placeholder in S0 and becomes the
  Next.js app in S7, to keep `pnpm -r build` fast and offline-friendly before the UI task.

## S1

- **Schema shape:** the contract is one canonical file,
  `packages/contract/schemas/fiscal.schema.json`, with every type under `$defs`. This avoids
  cross-file `$ref` resolution friction in three different codegen tools and gives the drift
  guard a single artifact.
- **json-schema-to-typescript prunes unreferenced `$defs`.** Fix: a small codegen wrapper
  (`packages/contract/scripts/gen-ts.mjs`) feeds json2ts a synthetic root referencing every
  `$def`, so all named types are emitted.
- **Go generator path:** the maintained `omissis/go-jsonschema` v0.23.1 still declares its
  module path as `github.com/atombender/go-jsonschema`; that is the installable/run path.
  Pinned to `@v0.23.1` in `scripts/codegen.sh` for deterministic output.
- **Go floor raised to 1.25:** the generated Go types use `go-jsonschema/pkg/types`
  (`SerializableDate` for the `date` format), whose module requires Go 1.25, so `go mod tidy`
  set `core/go.mod` to `go 1.25.0`. Still satisfies the spec's "Go 1.23+"; CI Go pinned to
  1.25 accordingly.
- **`allOf` of multiple `if/then` makes go-jsonschema fall back to `interface{}`.** The
  `ClassificationAssignment` rule (rule/model ⇒ `basis` required) is therefore expressed as a
  single top-level `if/then` with `assignedBy` enum `[rule, model]`, which generates a clean
  struct (same pattern `FiscalFact` uses for grain ⇒ rawSha256).
- **Validation runs the JSON Schema directly, not the generated models.** ajv (TS),
  `jsonschema` (Python), `santhosh-tekuri/jsonschema/v6` (Go) all validate against the schema
  so conditional rules (grain ⇒ rawSha256) and enums are enforced uniformly. Generated models
  alone (Pydantic/Go structs) would not catch the conditional. Shared fixtures live in
  `packages/contract/fixtures/` with a `cases.json` manifest run identically by all three.
- **`SchemeId` is a closed enum** in the contract (mirrors the DB FK to
  `classification_scheme`), so an unknown scheme is rejected by pure schema validation. New
  per-source schemes (e.g. CA acquisition type in S3) are added to this enum — adding a scheme
  is a contract change + regen.
- Drift guard verified both ways: green when types match the schema, red when the schema
  changes without regeneration. Codegen confirmed byte-identical across two runs.

## S2

- **`AdapterOutput` added to the contract** as the adapter `--out` document
  (`{ envelope, facts, entities?, entityAliases? }`). Regenerated all three languages.
- **Adapter output shape:** `fetchYear` returns facts *without* `factHash` (and without DB
  fields); the SDK scaffold fills `factHash` and computes `resultHash` (see Decision D15), so
  contributors never hand-roll hashing.
- **`resultHash` parity proven across languages.** The TS SDK (`canonicalize`), Python SDK
  (`rfc8785`), and Go harness (`cyberphone/json-canonicalization`) all produce the *identical*
  `resultHash` (`fd28684f…`) and identical raw `sha256` for the toy adapter — a strong check
  that JCS is implemented consistently. Pinned as a golden value in the Python test.
- **Toy adapters are network-free** and byte-for-byte identical across TS and Python (shared
  `RAW` payload), so conformance needs no recorded fixtures yet; real fixture replay arrives
  with the live CA adapter in S3.
- **Conformance harness** (`core/cmd/conformance`, `internal/conformance`) runs an adapter
  command through `info` / `list-years` / `fetch`×2 and checks: manifest fields, year pattern +
  descending order, exit 0, schema validity of the out doc, every `.bin` hashes to its name,
  declared `rawSnapshots` present/correct, recomputed `resultHash` == declared, and
  determinism across two runs. The Go unit test runs the built TS adapter and self-skips when
  it or `node` is absent, keeping `go test ./...` green without the JS toolchain.
- **Go recompute detail:** facts are parsed as `map[string]json.RawMessage`, volatile keys
  deleted, sorted by `factHash`, marshaled (Go sorts map keys), then JCS-canonicalized — the
  canonicalizer normalizes the pretty-printed whitespace, so it matches the SDKs.

## S3

- **Source verified live (2026-06-09):** CKAN resource `bb82edc5-…` is still up, **344,504
  rows**. `datastore_search_sql` works over GET (used by `listYears`); `filters` JSON param
  works for the year filter. Resource id unchanged — no NOTES relocation needed.
- **Coverage is FY 2012-13 … 2014-15 only** (108k / 120k / 116k rows) — older eSCPRS data, no
  recent years. Conformance/fixtures therefore target **2014-15**, not 2024-25.
- **Fiscal-year format mismatch:** source uses `"2014-2015"`; the contract/param regex wants
  `2014-15`. Adapter maps both ways (`toCanonicalFiscalYear` / `toSourceFiscalYear`).
- **Grain decision:** the dataset is **line-item** grain (one row per PO line, with its own
  `Total Price`), so the adapter emits **one award-grain fact per source row**, not an
  aggregated per-PO total. Aggregating would either fabricate or discard line detail; keeping
  the source's own grain is the honest choice (Hard Rule: never fabricate finer grain). Each
  fact's `derivationQuery` pins the row's datastore `_id`; `rawSha256` = the page snapshot.
- **Money:** `Total Price` like `"$1,362.00 "` → `normalizeMoney` (strip `$`,`,`,space, handle
  parens/sign, pad fraction to 4) → contract decimal string; rows with no parseable amount are
  skipped and counted (11/1000 in the sample).
- **Per-source schemes added:** `us_ca_department`, `us_ca_acquisition_type` (closed-enum
  additions per D13), assigned with `assignedBy='source'`.
- **Provisional vendors:** `payeeEntity` = `uuidv5("us-ca:vendor:" + NORMALIZED_NAME)` (exact
  normalized-name match, NO fuzzy merging). Entities + append-only aliases (`matchedBy='rule'`,
  `confidence=0.5`, `source` carries supplierCode + normalized). Deterministic ids make
  `factHash` reproducible and let one vendor be queried across departments.
- **HTTP record/replay** added to the SDK (`OUTLAYS_REPLAY_DIR` / `OUTLAYS_RECORD_DIR`, live
  otherwise; UA + per-host rate limit + backoff). Fixtures committed under the adapter's
  `fixtures/replay/`. **Replay requires `OUTLAYS_MAX_PAGES=1`** since only the first 1000-row
  page is recorded (full year ≈ 116 pages). The Go conformance test and CI set this.
- **Acceptance evidence (replay, 989 facts):** conformance PASS; 58 vendors span ≥2
  departments (e.g. "WESTERN BLUE, AN NWN COMPANY" across 11); two runs produce byte-identical
  `fact_hash` values and `resultHash` `cb9a490d…`.

## S4

- **Port 5433:** the dev host runs Postgres.app on 5432, so the compose Postgres is published
  on **5433** (`5433:5432`) to avoid the collision. `.env.example` and the integration test
  default to 5433. The compose project is `outlays` (renamed in S2); old `fiscal-warehouse-*`
  containers were removed.
- **Migrations** are goose SQL, embedded via `//go:embed` and run programmatically
  (`store.Migrate`, owner role). Four files: schema, append-only triggers, roles, seed.
- **Append-only:** `reject_mutation()` `BEFORE UPDATE OR DELETE` on all 10 tables + `REVOKE
  UPDATE,DELETE,TRUNCATE FROM app_rw` (D20). The seed `Down` uses `session_replication_role =
  replica` to bypass triggers for a controlled rollback.
- **Batched COPY writer** uses the temp-staging + `ON CONFLICT DO NOTHING` pattern (D21) so it
  works against append-only tables and is idempotent. `jsonb` columns (`envelope`, alias
  `source`) are passed to `CopyFrom`/`Exec` as JSON strings; numerics/uuids/dates via `pgtype`
  helpers (`pgNumeric`/`pgUUID`/`pgDate`); money never touches float.
- **Idempotency consequence:** content-addressed `fact_id` means re-ingest keeps the original
  `run_id`. The integration test therefore **resets the schema** (`DROP SCHEMA public CASCADE`
  as owner — append-only blocks TRUNCATE/DELETE) before each run to stay hermetic/repeatable.
- **Roles:** migrations create group role `app_rw` (NOLOGIN, SELECT+INSERT only). The login
  role is provisioned out of band (the integration test creates `app_login` + `GRANT app_rw`);
  the app connects as that member and proves it can INSERT but not UPDATE/DELETE.
- **Object store:** `minio-go/v7`; raw bytes + sidecar meta uploaded under
  `raw/{jur}/{dataset}/{fy}/{sha}.bin`. Content-addressed keys make uploads idempotent.
- **Integration test** (gated on infra; skips cleanly otherwise) ingests the real S3 adapter
  output (989 facts / 442 entities / 448 aliases / 1978 assignments / 1 snapshot), asserts
  UPDATE+DELETE raise on every table, the app REVOKE blocks UPDATE, the object key exists, and
  a `supersedes` correction chains. Repeatable across runs.

## S6

- **Unclassified bucket (design decision, requested):** the `view` endpoint groups facts by
  their assignment in the requested scheme via a LEFT JOIN; any fact with **no** assignment for
  that scheme is placed in an explicit `__unclassified__` node (label "Unclassified") and its
  total also surfaced in the view's `unmapped` field. Facts are **never silently dropped**, so
  node totals + unmapped always reconcile to the view total. This matters once COFOG mappings
  land (S9) with partial coverage — unmapped is honest (Hard Rule 5). To avoid double-counting
  when a fact has multiple assignments in one scheme, the join de-duplicates to one assignment
  per fact (`DISTINCT ON (fact_id) … ORDER BY version DESC`).
- **Current facts only:** views/coverage exclude facts that have been superseded
  (`NOT EXISTS (SELECT 1 FROM fiscal_fact s WHERE s.supersedes = f.fact_id)`); correction rows
  themselves are included. Money is returned as exact decimal strings (`sum(amount)::text`),
  never float; the API also sums decimal strings via integer minor-units math (`addDecimals`).
- **Curl-walk evidence (989 facts):** the same CA spending totals to `38695819.9100` under
  `us_ca_department` (57 nodes, unmapped 0), `us_ca_acquisition_type` (5 nodes), and `payee`
  (443 nodes, unmapped 11.1M = no/Unknown vendor); `cofog` returns one `__unclassified__` node
  == total. `entities/{id}/flows` shows Technology Integration Group across 11 departments.
  Provenance resolves to `raw/us-ca/purchase-order-data/2014-15/<sha>.bin`, confirmed present
  in MinIO. `leads?status=published` is empty; `status=draft` → 400.
- **OpenAPI** is a hand-maintained `docs/openapi.yaml` (3.1), not auto-generated from code —
  pragmatic for now; a codegen step could replace it later. The API integration test
  (`//go:build integration`) drives the router via `httptest` over the live stack.
- **`entities/{id}/flows`** is intentionally a department cross-cut (groups the vendor's facts
  by `us_ca_department`) — the simplest endpoint that demonstrates one vendor across many
  departments. A scheme-parameterized flows view can come later.

## S5

- **Acceptance year:** the prompt says "CA 2024-25" but the dataset only covers FY 2012-13 …
  2014-15, so the orchestrator is exercised on **2014-15** (replay). Live ingest of all years
  is the same code path with no `--replay-dir`.
- **Verify by re-derivation:** before persisting, the orchestrator validates the output against
  `AdapterOutput` and recomputes `resultHash` (shared `verify.RecomputeResultHash`), refusing
  to persist on mismatch (D22). The resultHash recompute was extracted from the conformance
  harness into `internal/verify` so both use one implementation.
- **Failure recording:** exit 2 → "source unavailable", exit 3 → "contract validation", other →
  "unexpected"; each writes a single `failed` `ingestion_run` row (`store.RecordFailedRun`),
  with the exit code in the envelope JSON. The toy adapter gained an `OUTLAYS_TEST_FAIL=1|2`
  hook to drive these paths deterministically.
- **errgroup backfill** with `--concurrency` (bounded). Caveat: rate limiting is per adapter
  process, so concurrent years relax the global per-host 1 req/s budget — bounded by the
  concurrency flag; live large backfills should keep it small.
- **CLI verified end to end:** `orchestrator run --adapter "node …cli.js" --year 2014-15
  --replay-dir … --max-pages 1` applied migrations and ingested 989 facts (slog JSON logs).
- **Integration tests are tag-gated** (`//go:build integration`) and run serially (`-p 1`)
  because they reset the shared schema; plain `go test ./...` excludes them (D23).

## S7

- **`packages/web` placeholder replaced** by a Next.js 15 App Router app (React 19, TS
  strict, no UI framework deps). All routes are `force-dynamic`, so `pnpm -r build` builds
  the app with no stack running — CI needs no API. Server runs with `OUTLAYS_API_URL`
  (default `http://localhost:8080`).
- **All read-API fetches are server-side** (D25): pages are server components; the
  provenance drawer (the only client-side fetch) goes through a same-origin pass-through
  proxy (`/api/provenance/[id]`), so the Go API needs no CORS headers and the client
  trivially makes no LLM calls.
- **Money in the client:** decimal strings end to end; `lib/decimal.ts` does BigInt
  minor-units math for the balance ribbon and renders via pure string formatting. JS
  `number` appears only for bar-width percentages (a display ratio, not money). Exact 4dp
  values are preserved in `title` attributes wherever display rounds to cents.
- **`/v1/facts` gained an optional `scheme`+`code` node filter** (must come together;
  `scheme=payee` and `__unclassified__` supported, mirroring the view endpoint) — the
  drill-down behind every node click, uniform across all three dimensions. Ordering changed
  from `fact_id` to `amount DESC, fact_id` (still deterministic). `payee`/`code` UUIDs are
  now validated in the handler (400 instead of a 500 from the `::uuid` cast). OpenAPI
  updated.
- **Contract `FiscalYearView`/`FiscalNodeView` do not match the served view payload** (the
  schema has per-node `schemeId` and no `unmapped` requirement; the API emits neither
  per-node `schemeId` nor `hasChildren`). Per D24 `docs/openapi.yaml` is authoritative for
  the read API, so the web types (`lib/types.ts`) mirror it. Proposed correction: either
  align the contract view types with the served shape in a future contract rev, or generate
  them from the OpenAPI doc instead of the fiscal schema.
- **Dimension → scheme mapping is CA-specific** (`lib/dimensions.ts`: department /
  acquisition type / payee → `us_ca_department` / `us_ca_acquisition_type` / `payee`).
  Phase 0 is California end to end; a second jurisdiction turns this into a
  per-jurisdiction lookup.
- **Shared-stack caveat rediscovered:** the store integration test leaves a `supersedes`
  correction row behind (it resets the schema at start, not at end), after which the UI
  total honestly drops by the superseded amount — D24 current-facts-only working as
  designed. Re-running the API integration test restores a pristine 989-fact state.
- **Acceptance evidence:** headless-Chromium walk (12 checks, all green) — identical
  $38,695,819.91 total across department (57 nodes) / acquisition type (5) / payee (443);
  coverage badge renders "coverage unknown — no official total ingested yet" (null
  denominator until S8); money-in side carries the "illustrative — not live data" marker
  with a figure-free sketch; department and payee drill-downs list award-grain PO rows; the
  provenance drawer opens from a row click showing factHash, rawSha256, the
  `raw/us-ca/purchase-order-data/2014-15/<sha>.bin` object key, source URL, and the
  derivation query pinning the row `_id`; payee drill shows the vendor-across-departments
  cross-cut.

## post-S7 (contract alignment)

- **Contract view types aligned to the served shape, precedence inverted (D26):** the S7
  proposal was approved with the contract as the leader, not OpenAPI. `FiscalNodeView`
  dropped per-node `schemeId` and `hasChildren` (the scheme is carried once on the view);
  `FiscalYearView` dropped the speculative `path` and made `unmapped` required (D24's
  reconciliation guarantee). All three languages regenerated; `CONTRACT_VERSION` 0.1.0 →
  0.2.0 (TS `packages/contract/src/index.ts` + Python `adapter_sdk/run.py` + the package
  version); drift guard green post-commit. `packages/web` now imports its view types from
  `@outlays/contract`.
- The drift guard "fails" locally on an uncommitted intentional schema change by design
  (it diffs regenerated output against git) — run it after committing; determinism was
  additionally checked by hashing two consecutive codegen runs.

## S8

- **Scope decision for the CA denominator (recorded per instruction).** The ingested
  dataset is procurement-only (eSCPRS purchase orders, FY2012-15), so the honest
  denominator would be an official procurement total. None is available machine-readably:
  (a) data.ca.gov's `datastore_search_sql` has a function whitelist — `SUM(CAST(...))` /
  `regexp_replace` / `NULLIF` all return 403 "Not authorized to call function", and `Total
  Price` is text, so the source cannot compute its own parsed-money total (plain `count(*)`
  works: FY2014-15 has 115,969 rows — our replay slice of 989 facts is one page of ~116);
  (b) `package_search` finds no DGS procurement-report or budget-totals dataset. Therefore
  the **full enacted-budget figure** is ingested and the coverage metric is explicitly
  labeled **"procurement facts vs total budget"** (the `scope` column/field, D27) so the
  badge never implies the denominator matches the dataset's scope.
- **The official figure:** 2014 Budget Act, Full Budget Summary
  (`https://ebudget.ca.gov/2014-15/pdf/Enacted/BudgetSummary/FullBudgetSummary.pdf`,
  774,364 bytes, sha256 `daf0be25…`), Figure SUM-02 "2014-15 Total State Expenditures by
  Agency": General $107,987M + Special $44,324M + Bond $4,046M = **$156,357M** →
  `officialTotal 156357000000.0000`. The figure is a reviewed in-source constant in the new
  `us-ca-budget` adapter; the document is fetched and hashed at run time so the
  transcription is verifiable against the stored bytes. (The PDF is deflate-compressed, so
  the adapter cannot cheaply assert the string appears in the raw bytes — verification is
  by a human against the locator, which names figure, row, column, and page.)
- **`us-ca-budget` adapter** emits 0 facts + 1 `controlTotal`; conformance PASS in replay
  (resultHash `4f53cda1…` = the deterministic empty-fact-set hash); fixture = the recorded
  PDF (774KB, committed). Contract 0.2.0 → **0.3.0**: `AdapterOutput.controlTotals?`,
  `ControlTotal` + required `currency`/`scope`. Migration `00005` adds the matching columns.
  Both SDKs (TS + Python) pass `controlTotals` through for parity.
- **`raw_sha256` on `control_total`** is nullable in the DDL but required by the contract;
  the coverage query scans it as non-null (everything arrives via the pipeline). If a
  hand-inserted row ever violated that, coverage would 500 — acceptable until corrections
  for control totals are designed (same open question as the D20 lead workflow).
- **Coverage endpoint** now returns `numeratorBasis` (aggregation description + `factsUrl`
  where every fact carries its own provenance link) and `denominatorBasis` (scope, raw
  hash, object-store key, source URL, locator). Ratio = round(n/d, 6) = **0.000247** →
  badge "coverage 0.0247% — procurement facts vs total budget". OpenAPI documents the
  shape.
- **Found by the NOT NULL change:** `seedWorkflowRows` in the store integration test
  swallowed its INSERT error (`_, _ =`), so the new NOT NULL columns silently emptied the
  control_total trigger test; the seed now includes scope/currency and fails loudly.
- **Acceptance evidence:** API integration test asserts numerator == view total,
  denominator `156357000000.0000`, ratio `0.000247`, both bases present, scope label
  exact, and the denominator's `storageKey` exists in MinIO. Headless-Chromium walk (12
  checks, all green) including the badge rendering the honest low number with the scope
  label in the masthead.

## post-S8 (dependency housekeeping)

- **Dependabot alert #1 remediated** (GHSA-qx2v-qp2m-jg93 / CVE-2026-41305, medium,
  CVSS 6.1): postcss < 8.5.10 fails to escape `</style>` when stringifying CSS, enabling
  XSS **when an app parses user-submitted CSS and re-embeds the stringified output in an
  HTML `<style>` tag**. Reachability assessed: postcss 8.4.31 appeared in our tree solely
  as a transitive build-time dependency of Next.js (`packages/web`), processing first-party
  stylesheets at `next build`; nothing in the repo accepts or stringifies user CSS (D25:
  server-rendered UI, no client-side CSS pipeline). Not reachable in our usage — but the
  fix is a one-line pnpm override (`"postcss@<8.5.10": ">=8.5.10"`, resolves 8.5.15), so it
  was remediated rather than deferred. `pnpm -r build` green afterward, including the web
  app whose Next.js pinned 8.4.31 exactly.
- **Branch state discrepancy (recorded):** the task brief said research/gates was merged to
  master, but `origin/master` was still at S8; the Gate 5–10 deliverables (including
  `data/cofog/us-ca-procurement.json`) existed only on `origin/research/gates`. Merged that
  branch into local master before starting (commit `c3f534b`), deliverables unmodified.

## S9

- **Loader shape:** `orchestrator classify` subcommand + `core/internal/classify`. Reads a
  reviewed mapping file (`data/cofog/*.json`) — a research deliverable the loader never
  writes; corrections are proposed here, not edited in. Mapping keys `<prefix>: <category>`
  resolve through a per-jurisdiction prefix table (in-source constants, Hard Rule 8):
  `us-ca` → `department:`→`us_ca_department`, `acquisition_type:`→`us_ca_acquisition_type`.
  Strict validation: cofogCode must be a seeded `01`–`10` or `unmapped`; confidence in
  {low, medium, high}; basis (citation) required; unknown entry fields rejected.
- **`unmapped` is a reviewed non-mapping, not an absence.** Entries with
  `cofogCode='unmapped'` (all five acquisition types + HHS Agency — inputs, not functions,
  per the mapping policy) produce zero rows and are reported as `reviewed_unmapped`;
  source categories absent from the file are reported as `unreviewed`. Both lists carry
  fact counts and amounts and are printed by `--list-unmapped` (and in every apply report):
  38 categories on the replay slice — 6 reviewed-unmapped, 32 unreviewed departments
  beyond the reviewed top-25.
- **Confidence is preserved twice:** the reviewer's ordinal word verbatim inside `basis`,
  and a documented numeric translation (low→0.25, medium→0.5, high→0.75) in the NUMERIC
  `confidence` column so it is comparable with other assigners.
- **Basis is canonical JSON** `{ruleId, citation, sourceCategory, confidence, entrySha256}`
  (Hard Rule 5: rule id + citation). `entrySha256` = JCS+SHA-256 over the exact reviewed
  entry. The whole-file sha is deliberately **not** embedded in basis — unrelated entry
  edits would otherwise re-version every assignment; the file sha is reported per run
  instead, and the entry hash pins the reviewed content precisely.
- **Versioning / idempotency:** first assignment is version 1; if the entry for a fact's
  category changes (code, confidence, or basis), a new row appends at latest+1 and the
  view's `DISTINCT ON ... ORDER BY version DESC` (D24) picks it up. Re-applying an
  unchanged mapping inserts nothing (deterministic `assignment_id`, D21). A fact whose
  latest cofog assignment is `human` is never overridden by the rule loader
  (`skippedHumanOverride`). Facts whose mapped categories disagree on the code get
  nothing — ambiguity stays unmapped and is reported as a conflict (zero on the real file,
  by its own design: acquisition types are all unmapped).
- **Reconciliation is enforced, not assumed:** the loader recomputes the cofog view and an
  independent count/sum over current facts, requires node-sum == view total == fact sum
  and mapped + unclassified == total in exact decimal-string math, and the CLI exits
  non-zero on any mismatch. Replay evidence (989 facts): 861 mapped by 24 department
  entries, 128 unclassified; `30208583.0000 + 8487236.9100 == 38695819.9100` exactly;
  cofog view shows 9 code nodes (no `09` — no mapped department is education-functioned)
  plus `__unclassified__`, identical total to the department/payee pivots.
- **Test-only discovery:** `ruleId` derives from the mapping file name, so a temp copy
  named differently legitimately re-versions everything — the integration test's modified
  mapping must keep the original file name.
