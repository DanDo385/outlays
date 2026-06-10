# Outlays

> **Mission.** Make any government-spending question answerable in seconds, with a
> cryptographically verifiable citation back to the source row. We do not claim transparency
> alone ends bad spending; we collapse the cost of investigation and comparison. Neutral
> method is the product: no editorializing, ever.

`ARCHITECTURE.md` is the **normative spec**. On any conflict, it wins over code, comments,
or this file.

---

## Hard Rules (binding on every task, no exceptions)

1. **Provenance or nothing** — never emit/store a monetary figure without `raw_sha256` +
   `derivation_query`.
2. **Money is never a float** — JSON decimal string `^-?\d{1,18}(\.\d{1,4})?$` + ISO 4217;
   Postgres `NUMERIC(24,4)`; TS `bigint`/decimal lib; Go `shopspring/decimal`. JS `number`
   is forbidden for amounts.
3. **Hashing** — `rawHash` = SHA-256 over exact upstream response bytes, captured before
   parsing. Derived-document hashes = SHA-256 over RFC 8785 (JCS) canonical JSON.
   `JSON.stringify` for hashing is forbidden.
4. **Append-only, enforced** — `REVOKE UPDATE, DELETE` + `BEFORE UPDATE OR DELETE` reject
   triggers. Corrections are new rows (`supersedes` / `version`). History is evidence.
5. **Never invent a classification** — `source` = government coding; `rule` needs a rule id
   + citation; `model` needs a model version in `basis`; `human` needs a reviewer handle.
   Otherwise unassigned. Unmapped is honest.
6. **Leads are facts, never accusations** — nothing public unless a human set
   `status='published'`. Never assert corruption/fraud/unfairness about a named party.
7. **No named-individual payroll** reachable by any public endpoint. Compensation only as
   aggregates.
8. **Strict input validation** — fiscal year `^\d{4}(-\d{2})?$`; upstream identifiers come
   only from in-source constants; no runtime input interpolated into upstream queries.
9. **Upstream etiquette** — UA `outlays/<version> (+repo URL)`, 1 req/s/host,
   backoff with jitter, honor 429 + ETags. **CI uses recorded fixtures only, never live
   government APIs.**
10. **Secrets via env only** — commit `.env.example`, never keys.

---

## Roadmap

- **Phase 0 — California end to end** on the fact model.
- **Phase 1 — Bulk census sprint:** ingest USAspending bulk archives, IRS 990 e-files,
  Federal Audit Clearinghouse, SAM extracts; entity resolution v1; coverage map.
- **Phase 2 — Leads engine v1**, nonprofit cross-reference, second state, COFOG
  comparability.
- **Phase 3 — Community gap-filling:** council-minutes extraction, records-request tracking,
  guided by the coverage map.

---

## Conventions

- **EVR on every task** — Execute, Verify against acceptance criteria, Report what passed
  and anything you could not verify. Never report success you did not confirm. Do not start
  S(n+1) while S(n) is red.
- **TypeScript strict** everywhere; types in `packages/contract` are **generated**, never
  hand-written.
- **Adapters are pure transforms** — given upstream bytes, they deterministically produce a
  contract-valid facts document. Same input ⇒ same `resultHash`.
- **ARCHITECTURE.md is normative on conflict.** Propose Decision Log entries rather than
  silently changing the spec.
- After each task: tick `BUILD_TASKS.md`, append discovered constraints to `NOTES.md`.

---

## Repo layout

```
CLAUDE.md  ARCHITECTURE.md  BUILD_TASKS.md  NOTES.md  LICENSE(Apache-2.0)  .env.example
packages/{contract, adapter-sdk-ts, adapters/us-ca-procurement, web}/   (pnpm workspace)
py/adapter_sdk/
core/{cmd/{orchestrator,api,conformance,anchor}, internal/{ingest,store,verify,api}, migrations/}
contracts/                              (Foundry: AnchorRegistry.sol + tests)
data/cofog/   docs/sources/             (research lands here via PRs)
deploy/docker-compose.yml               (postgres:16 + minio + bucket init)
.github/workflows/
```
