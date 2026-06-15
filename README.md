# Outlays

**See where public money comes from, where it goes, and prove every number back to the source.**

Outlays is a transparency tool for government finance. It pulls official public records,
stores them without rewriting history, and shows them in a simple two-sided view: **money in**
(taxes and revenue) on the left, **money out** (spending) on the right.

We do not editorialize. Every figure links to the exact upstream record it came from.

---

## The analogy: a receipt binder, not a headline

Think of government spending data the way you would think about your own finances.

Most news and dashboards give you **summaries**: "Defense spending rose 4%." That is useful,
but it is hard to audit. You cannot open the summary and see the underlying receipt.

Outlays works more like a **binder of receipts** tied to a **checkbook register**:

```
  YOUR QUESTION                    OUTLAYS ANSWER
  -------------                    --------------
  "How much went out?"      -->    Totals on the ledger screen
  "Who got paid?"           -->    Drill down by recipient
  "Prove this line item"    -->    Click through to source bytes + hash
  "Did you change history?" -->    No. Corrections are new rows, not edits.
```

The register (totals and categories) is computed from the receipts (individual facts). Change
how you sort the receipts (by agency, by award type, by vendor) and the register layout
changes, but the underlying dollars do not.

Another way to say it: **facts are permanent atoms; categories are lenses.** The same dollar
can appear in "Health and Human Services" and "Grant to nonprofit X" without double-counting
at the fact level. You pick which lens you are looking through.

---

## How the application works

Outlays has three layers. You mostly interact with the top one; the bottom two are what make
the numbers trustworthy.

```
                         +---------------------------+
                         |         WEB UI            |
                         |  ledger, drill-down,      |
                         |  provenance drawer        |
                         +-------------+-------------+
                                       |
                         +-------------v-------------+
                         |       READ API (Go)       |
                         |  /v1/.../view, /facts,  |
                         |  /provenance, /coverage   |
                         +-------------+-------------+
                                       |
         +-----------------------------+-----------------------------+
         |                             |                             |
+--------v--------+           +--------v--------+           +--------v--------+
|   PostgreSQL    |           |  Object store   |           |  Adapters (TS/  |
|  facts, codes,  |           |  raw HTTP bytes |           |  Py) fetch gov  |
|  assignments    |           |  (content-      |           |  APIs, emit     |
|  (append-only)  |           |   addressed)    |           |  validated JSON |
+-----------------+           +-----------------+           +-----------------+
         ^                             ^                             ^
         |                             |                             |
         +-----------------------------+-----------------------------+
                                       |
                         +-------------v-------------+
                         |     ORCHESTRATOR (Go)     |
                         |  runs adapters, validates |
                         |  output, writes facts     |
                         +---------------------------+
```

### Step by step (what happens when data loads)

1. **Fetch.** An adapter calls an official government API (or reads a recorded fixture in CI
   and local demo mode). The raw HTTP response bytes are saved first, before any parsing.
2. **Hash.** Each response gets a SHA-256 fingerprint (`raw_sha256`). If the source changes,
   the hash changes. You can detect tampering or silent updates.
3. **Parse.** The adapter turns rows into **facts**: amount, currency, payer/payee, fiscal
   year, jurisdiction, and a `derivation_query` that says exactly which upstream field became
   this number.
4. **Classify (optional).** Facts can carry government codes or mapped categories (for
   example COFOG function codes, department, award type). Classifications are separate rows,
   not baked into the amount.
5. **Store.** Facts land in PostgreSQL under an append-only policy. The database rejects
   updates and deletes on evidence tables. A correction is a **new** fact that supersedes an
   old one; history stays visible.
6. **Serve.** The read API aggregates facts into views (`GROUP BY` a scheme). The web UI
   renders those views and links every total back to underlying facts and raw bytes.

Money never passes through floating-point math. Amounts are exact decimal strings end to end.

---

## The ledger screen

This is the main view. One fiscal year, one jurisdiction, two sides.

```
+------------------------------------------------------------------+
|  United States (federal)  FY 2025          [Coverage badge] [v]  |
+------------------------------------------------------------------+
|  [ By awarding agency ] [ By award type ] [ By recipient ]      |  <- pivot tabs
+------------------------------------------------------------------+
|  Money in          |  Money out          |  Balance              |
|  $0.00             |  $12,345,678.90     |  -$12,345,678.90      |
|  (no revenue yet)  |  (100 facts)        |  (ingested only)      |
+--------------------+---------------------+-----------------------+
|  MONEY IN          |  MONEY OUT                                    |
|  (empty / sketch)  |  Dept of Health .......... $4.1M   [=======] |
|                    |  Dept of Education ....... $2.8M   [=====  ] |
|                    |  Unclassified ............ $0.12M  [=      ] |
|                    |  ... click any row to drill down ...          |
+--------------------+-----------------------------------------------+
```

**Left (money in):** revenue and tax income. Empty until a revenue adapter has been loaded.

**Right (money out):** spending, grouped by whichever tab you selected. Tabs only change the
**right** side's grouping. Totals stay tied to the same underlying facts.

**Balance ribbon:** money in minus money out over **ingested facts only**. It is not an
official government budget surplus or deficit; it is an honest arithmetic summary of what
Outlays has loaded so far.

**Coverage badge:** how much of the official control total your ingested facts represent.
Low coverage early on is expected, not a bug. The badge links to the numerator and denominator
provenance.

---

## Using the app (click path)

After you start the stack locally (see below), open [http://localhost:3000](http://localhost:3000).

| Step | What you do | What you get |
| ---- | ----------- | ------------ |
| 1 | Land on the ledger | Default demo: federal FY 2025 assistance spending |
| 2 | Use the **year** dropdown | Switch between FY 2024 and FY 2025 (after `make seed`) |
| 3 | Click a **pivot tab** | Regroup spending: agency, award type, or recipient |
| 4 | Click a **category row** | Drill page: individual facts (awards, amounts, payees) |
| 5 | Open **provenance** on a fact | `fact_hash`, `derivation_query`, link to raw snapshot bytes |
| 6 | Click **recipient** drill (payee lens) | Cross-cut: how much that entity received across categories |

Breadcrumb navigation takes you back up: Ledger → category → facts.

If a side shows **$0** with a layout sketch, that means no facts were ingested for that flow
yet. Outlays does not invent placeholder dollars.

---

## Why you can trust a number (technical, but human-readable)

Every stored amount carries **provenance**: where it came from and how it was derived.

| Field | Plain English | Technical role |
| ----- | ------------- | -------------- |
| `raw_sha256` | Fingerprint of the exact file the government returned | SHA-256 over raw HTTP bytes, captured before parsing |
| `derivation_query` | Human-readable recipe: "this JSON field became this fact" | Auditable mapping from source row to fact |
| `fact_hash` | Fingerprint of this fact record | Detect duplicate or altered rows |
| `supersedes` | "This row replaces an earlier one" | Append-only corrections, not silent edits |

Derived documents (adapter output) are hashed with **RFC 8785 canonical JSON**, not
`JSON.stringify`, so hashes are stable across languages.

Public **leads** (anomaly flags) are facts plus statistical context. Nothing is published
unless a human reviewer marks it `published`. The system does not accuse named parties of
fraud or corruption.

---

## What works today

| Piece | Status |
| ----- | ------ |
| Two-sided ledger UI | Yes |
| Pivot spending breakdown (jurisdiction-specific tabs) | Yes |
| Drill-down to facts + provenance | Yes |
| Append-only store + raw byte snapshots | Yes |
| **Default demo data** | **Federal FY 2024 and FY 2025** (100 assistance awards per year, offline fixtures) |
| Federal revenue (left side) | Not loaded yet |
| California tax / revenue (left side) | Not loaded yet |
| California 2014-15 procurement sample | Available via `make seed-ca` |

The default demo uses **recorded USAspending fixtures**, not live government API calls.
That keeps CI and first-time setup fast and polite to upstream servers.

---

## Try it locally

### Prerequisites

| Tool | Why |
| ---- | --- |
| [Docker](https://docs.docker.com/get-docker/) | Postgres + MinIO (object storage) |
| [Node 22+](https://nodejs.org/) + [pnpm](https://pnpm.io/) | Web UI and TypeScript adapters |
| [Go 1.23+](https://go.dev/) | Read API, orchestrator, migrations |

Full developer setup, Makefile reference, and testing: [docs/DEVELOPING.md](./docs/DEVELOPING.md).

### Quick start (four terminals worth, two required)

**1. Bootstrap infrastructure and build**

```sh
make up
make seed
```

`make up` creates `.env` if missing, starts Docker services, runs migrations, and builds the
workspace. `make seed` loads federal FY 2024 and FY 2025 assistance facts from offline
fixtures.

**2. Start the read API** (terminal 1)

```sh
make run-api
```

API base: [http://localhost:8080](http://localhost:8080). Example:

```sh
curl http://localhost:8080/v1/jurisdictions
curl 'http://localhost:8080/v1/us-fed/2025/view?scheme=us_fed_awarding_agency&flow=spending'
```

If port 8080 is busy: `make stop-api`, then try again.

**3. Start the website** (terminal 2)

```sh
pnpm --filter @outlays/web dev
```

**4. Open the app**

[http://localhost:3000](http://localhost:3000)

You should land on **United States (federal) FY 2025**. Use the year dropdown for **2024**.

### Other demo data

```sh
make seed-ca   # legacy California 2014-15 procurement + COFOG classify
```

California purchase-order data on data.ca.gov stops at 2014-15. Recent **state** budget years
need new adapters; see [docs/DEVELOPING.md](./docs/DEVELOPING.md).

---

## Design principles

1. **Provenance or nothing.** No number without a citation to source data.
2. **Append-only history.** Corrections are new rows, not silent edits.
3. **Neutral method.** We show patterns and sources; we do not accuse.
4. **Honest gaps.** Missing data and "Unclassified" buckets stay visible.
5. **Same facts, many views.** Categories are computed; they are not the storage model.

Normative rules and full architecture: [ARCHITECTURE.md](./ARCHITECTURE.md).

---

## Repository map (short)

```
outlays/
├── packages/web/              Next.js ledger UI
├── packages/adapters/         Ingest from government sources (TS)
├── packages/contract/         Generated schemas (do not hand-edit)
├── core/                      Go orchestrator, API, Postgres store
├── contracts/                 On-chain anchor registry (Foundry)
├── deploy/                    docker-compose for local Postgres + MinIO
├── docs/                      OpenAPI, source research, DEVELOPING.md
└── Makefile                   make up | seed | run-api | test
```

---

## For developers

- [docs/DEVELOPING.md](./docs/DEVELOPING.md): install, Makefile, CLI, testing, ingesting other years
- [docs/openapi.yaml](./docs/openapi.yaml): read API specification
- [BUILD_TASKS.md](./BUILD_TASKS.md): backlog and acceptance criteria
- [ARCHITECTURE.md](./ARCHITECTURE.md): normative data model and hard rules

---

## License

Apache-2.0. See [LICENSE](./LICENSE).
