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
