# Developing Outlays

Technical guide for contributors. For a plain-language overview, see [README.md](../README.md).

Phase 0 (California end to end on the fact model) is complete: ingest, classify, read API,
web UI, parquet/DuckDB analytical path, private leads scaffold, and on-chain anchoring. See
[BUILD_TASKS.md](../BUILD_TASKS.md) for the full task history and acceptance criteria.

---

## Documentation

| Document | Purpose |
| -------- | ------- |
| [ARCHITECTURE.md](../ARCHITECTURE.md) | **Normative spec**: data model, contract, API, decision log (wins on conflict) |
| [CLAUDE.md](../CLAUDE.md) | Mission, hard rules, roadmap, repo conventions |
| [BUILD_TASKS.md](../BUILD_TASKS.md) | Task sequence and acceptance criteria |
| [NOTES.md](../NOTES.md) | Discovered constraints and implementation notes |
| [docs/openapi.yaml](./openapi.yaml) | Read API OpenAPI description |
| [docs/leads-methodology.md](./leads-methodology.md) | Leads rule library (research deliverable) |
| [docs/sources/](./sources/) | Upstream source research (research branch) |

---

## Prerequisites

| Tool | Version / notes |
| ---- | ---------------- |
| [Docker](https://docs.docker.com/get-docker/) | Compose v2 (Postgres 16 + MinIO) |
| [Node.js](https://nodejs.org/) | 22 or newer |
| [pnpm](https://pnpm.io/) | 10.15.1 (`packageManager` in `package.json`) |
| [Go](https://go.dev/) | 1.25+ (`core/go.mod`) |
| [Foundry](https://book.getfoundry.sh/getting-started/installation) | `forge` / `cast` (contracts + anchor CLI) |
| [uv](https://docs.astral.sh/uv/) | Optional, for Python adapter SDK |
| `make` | Local bootstrap targets |

Clone with submodules (needed for Foundry tests):

```sh
git clone --recurse-submodules git@github.com:DanDo385/outlays.git
git submodule update --init --recursive contracts/lib/forge-std
```

---

## Quick start

From the repo root:

```sh
make up          # .env, Postgres + MinIO, migrations, full build
make seed        # offline CA replay ingest + COFOG classify (sample data)
```

Then start the two dev servers (separate terminals):

```sh
make run-api     # Go read API at http://localhost:8080
pnpm --filter @outlays/web dev   # Next.js UI at http://localhost:3000
```

Open [http://localhost:3000](http://localhost:3000). The web app calls the read API at
`http://localhost:8080` by default (`OUTLAYS_API_URL` overrides this).

Run `make help` for all Makefile targets.

### What `make up` does

1. Copies `.env.example` to `.env` if missing
2. Starts `deploy/docker-compose.yml` (Postgres on host **5433**, MinIO on **9000**)
3. Waits for Postgres and MinIO to be healthy
4. Runs goose migrations (`make migrate`)
5. Creates the app login role from `.env` and grants `app_rw` (`make bootstrap-db`)
6. Builds the pnpm workspace, Go core, and Foundry contracts (`make build`)

`make up` compiles everything and brings up infrastructure. It does **not** start long-running
servers. Use `make run-api` and `pnpm --filter @outlays/web dev` for that.

### Sample data (`make seed`)

Seeds **federal USAspending assistance** for fiscal years **2024** and **2025** using
**recorded fixtures only** (no live government API calls):

- 100 spending facts per year from `packages/adapters/us-fed-usaspending/fixtures/replay`
- Pivot dimensions: awarding agency, award type, recipient (entity)

Re-running `make seed` on unchanged fixtures is safe (deterministic fact hashes).

Legacy California procurement demo (2014-15 + COFOG classify): `make seed-ca`.

---

## Data coverage

**Important:** The UI shows whatever years exist in the database. `make seed` loads federal
FY2024 and FY2025 by default.

### What is available today

| Source | Years | What it gives you |
| ------ | ----- | ----------------- |
| Federal USAspending assistance (`us-fed-usaspending`) | **2024, 2025** (demo fixtures) | Spending (right side); default demo |
| CA purchase orders (`us-ca-procurement`) | **2012-13 through 2014-15 only** in the upstream dataset | Spending; use `make seed-ca` |
| CA budget control total (`us-ca-budget`) | **2014-15 only** pinned so far | Coverage denominator, not ledger rows |
| CA revenue | **Not built yet** | Would fill the left side (tax / revenue) |

California's open purchase-order table on data.ca.gov does **not** include FY 2024-25 or
2025-26. You cannot get those years from the current CA procurement adapter.

### Option A: Full California 2014-15 (live ingest)

If you want **more than the demo slice** for the years the CA source actually has:

```sh
make up
pnpm --filter @outlays/adapter-us-ca-procurement build
make run-orchestrator ARGS='run \
  --adapter "node packages/adapters/us-ca-procurement/dist/cli.js" \
  --years 2012-13,2013-14,2014-15'
make run-orchestrator ARGS='classify \
  --mapping data/cofog/us-ca-procurement.json \
  --jurisdiction us-ca --year 2014-15'
```

This calls **live** data.ca.gov (respect rate limits). Do not use `--replay-dir` or
`OUTLAYS_MAX_PAGES=1` unless you intentionally want the tiny fixture slice.

Revenue (left side) will still be empty until a CA revenue adapter lands
(`docs/sources/us-ca-revenue.md`).

### Option B: Federal FY2024 / FY2025 (default demo)

The `us-fed-usaspending` adapter ingests federal assistance awards via USAspending
`POST /api/v2/search/spending_by_award/`. Offline demo:

```sh
make up
make seed
```

For live ingest (respect 1 req/s; do not use in CI):

```sh
make run-orchestrator ARGS='run \
  --adapter "node packages/adapters/us-fed-usaspending/dist/cli.js" \
  --year 2025'
```

Federal fiscal year 2025 runs October 2024 through September 2025. See
`docs/sources/us-fed-bulk.md` for bulk archive research.

### Option C: California FY 2024-25 / 2025-26 (state budget years)

Requires **new adapters**, not a year picker change:

1. **Revenue (left):** DOF/eBudget summary schedules (PDF extraction), see
   `docs/sources/us-ca-revenue.md`
2. **Spending (right):** a source that actually publishes 2020s transaction or award data (the
   old PO table stops at 2014-15)
3. **Control totals:** extend `us-ca-budget` with pinned figures for 2024-25 and 2025-26

Track progress in [BUILD_TASKS.md](../BUILD_TASKS.md) backlog and [NOTES.md](../NOTES.md).

---

## Local services

| Service | URL / port | Started by |
| ------- | ---------- | ---------- |
| Postgres | `localhost:5433` | `make up` |
| MinIO (S3 API) | `http://localhost:9000` | `make up` |
| MinIO console | `http://localhost:9001` | `make up` |
| Read API | `http://localhost:8080` | `make run-api` |
| Web UI | `http://localhost:3000` | `pnpm --filter @outlays/web dev` |
| Anvil (anchor dev) | `http://localhost:8545` | manual, see Anchor layer below |

Default credentials are in [`.env.example`](../.env.example) (local dev only).

---

## Makefile reference

| Target | Description |
| ------ | ----------- |
| `make up` | Full bootstrap: env, infra, migrate, app role, build |
| `make down` | Stop compose stack |
| `make restart` | `down` then `up` |
| `make seed` | Offline federal FY2024 + FY2025 ingest (USAspending fixtures) |
| `make seed-ca` | Legacy CA 2014-15 ingest + COFOG classify |
| `make run-api` | Start Go read API |
| `make stop-api` | Stop process on PORT (default 8080) |
| `make build` | pnpm + Go + forge build |
| `make migrate` | Run goose migrations only |
| `make test` | Go unit tests + pnpm tests |
| `make integration` | Integration tests (needs compose stack) |
| `make python` | `uv sync` in `py/adapter_sdk` |

---

## Repository layout

```
outlays/
├── packages/          TypeScript workspace (contract, adapters, web UI)
├── py/adapter_sdk/    Python adapter SDK
├── core/              Go (orchestrator, API, store, migrations)
├── contracts/         Foundry (AnchorRegistry)
├── deploy/            docker-compose (Postgres + MinIO)
├── data/cofog/        Classification mapping files
├── docs/              OpenAPI, research, this file
├── scripts/           conformance, bootstrap, anchor verification
└── Makefile
```

---

## CLI tools

### Orchestrator

```sh
# Default demo (federal FY2025, offline)
make run-orchestrator ARGS='run \
  --adapter "node packages/adapters/us-fed-usaspending/dist/cli.js" \
  --year 2025 \
  --replay-dir packages/adapters/us-fed-usaspending/fixtures/replay \
  --max-pages 1'

# Legacy CA demo (offline)
make run-orchestrator ARGS='run \
  --adapter "node packages/adapters/us-ca-procurement/dist/cli.js" \
  --year 2014-15 \
  --replay-dir packages/adapters/us-ca-procurement/fixtures/replay \
  --max-pages 1'

make run-orchestrator ARGS='classify \
  --mapping data/cofog/us-ca-procurement.json \
  --jurisdiction us-ca --year 2014-15'
```

### Read API

```sh
make run-api
curl http://localhost:8080/v1/jurisdictions
curl 'http://localhost:8080/v1/us-fed/2025/view?scheme=us_fed_awarding_agency&flow=spending'
```

### Anchor layer

```sh
anvil --port 8545
bash scripts/e2e_anchor_local.sh <run-uuid>
```

---

## Testing

```sh
make test
make integration
bash scripts/conformance.sh
cd contracts && forge test -vv
bash scripts/check-drift.sh
```

Integration tests reset the public schema. Do not run them against a database you intend to
keep.

---

## Contributing

- [ARCHITECTURE.md](../ARCHITECTURE.md) is normative. Propose Decision Log entries rather
  than silently changing behavior.
- Types in `packages/contract` are generated; run `bash scripts/codegen.sh` and
  `bash scripts/check-drift.sh` after schema changes.
- Do not edit research deliverables under `docs/sources/` in place; note corrections in
  [NOTES.md](../NOTES.md).

---

## License

Apache-2.0. See [LICENSE](../LICENSE).
