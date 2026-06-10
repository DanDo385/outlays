# Architecture risk register

## Gate

Gate 9 — architecture risk register after source gates.

Purpose:

Capture the failure modes exposed by Gates 1-8 before they harden into product assumptions.

## EVR status

### Execute

Inputs used:

- Gate 1: USAspending bulk archives.
- Gate 2: nonprofit stack.
- Gate 3: entity identifiers and UEI/EIN linkage.
- Gate 4: healthcare cross-cut feasibility.
- Gate 5: California revenue sources.
- Gate 6: COFOG references and CA mapping.
- Gate 7: New York portal mechanics.
- Gate 8: data quality and lead safety.
- Repo architecture hard rules.

### Verify

Verified from prior gate artifacts:

- Bulk-first is the correct path for federal coverage, but huge files and account-linkage complexity remain.
- EIN linkage is real in IRS/FAC/ProPublica, not public-officially bridged to SAM UEI yet.
- SAM public/sensitive tier split blocks official public UEI→EIN proof until API key and access tier are resolved.
- Healthcare embedded employee benefits cannot be computed exactly from verified public sources at department grain.
- California revenue comprehensive source is PDF/extraction-first, not API-first.
- COFOG mapping is often a classification overlay, not a source truth.
- Second-state portal mechanics work on Socrata, but coverage can be authority-specific rather than statewide.
- Lead publication requires human review and neutral copy.

### Report

This file is the Gate 9 risk register.

## Summary

The biggest architectural risk is not technical complexity. It is false precision.

Outlays can win by being brutally honest about grain, coverage, provenance, and unresolved identifiers. It loses trust if it pretends partial public data is a complete ledger.

## Risk register

### R1 — False precision in public totals

Risk:

Source coverage varies by jurisdiction, source, and grain. Users may assume a dashboard total means complete government spending.

Observed evidence:

- USAspending can provide large federal award/account surfaces, but not every downstream dollar.
- California procurement is purchase-order line item grain, not all California spending.
- New York MTA procurement is public-authority procurement, not all New York state spending.

Mitigation:

- Every total displays grain and coverage.
- Every jurisdiction-year has `coverage = loaded facts / official control total` where a control total exists.
- If no control total exists, display `coverage unknown`.
- Public copy says `known public records loaded by Outlays`, not `total spending`, unless coverage proves it.

### R2 — UEI/EIN bridge overclaim

Risk:

Public users want one entity graph across USAspending, IRS, FAC, and SAM. The official public UEI→EIN bridge is not verified.

Observed evidence:

- IRS EO BMF exposes nonprofit EIN/name/address.
- FAC bulk exposes auditee EIN and UEI in audit context.
- SAM public tier exposes UEI/name/address/NAICS/PSC surfaces; EIN/TIN is sensitive-tier.
- No public official SAM UEI→EIN bridge verified.

Mitigation:

- Treat EIN and UEI as separate authoritative identifiers.
- Allow link types:
  - `identifier_ein`,
  - `identifier_uei`,
  - `fac_ein_uei_context`,
  - `name_address_candidate`,
  - `human_verified`.
- Public UI distinguishes confirmed from candidate links.
- Do not merge entities destructively.

### R3 — Name/address fuzzy merge harm

Risk:

Name/address crosswalks can merge unrelated entities, especially common nonprofit names, subsidiaries, DBAs, and vendors with shared addresses.

Observed evidence:

- IRS EO BMF and SAM public data can support name/address candidate matching.
- California and NY procurement often provide vendor name/address but not authoritative IDs.

Mitigation:

- Deterministic provisional IDs by source name are fine.
- Cross-source name/address joins produce candidate aliases, not legal merges.
- Require confidence and basis on every candidate.
- Allow public filtering by `confirmed only` vs `candidate links included`.

### R4 — Healthcare total overreach

Risk:

The motivating cross-cut says healthcare cost can be buried in police, education, postal, and other department budgets. Users may expect exact totals.

Observed evidence:

- Federal object class files can isolate broad personnel benefits and health/medical object classes, but not exact health-insurance premiums inside each department.
- California CalPERS/CalHR verified sources do not publish machine-readable department-level actual employer health premium expenditure in this gate.

Mitigation:

- Publish healthcare in tiers:
  - direct health programs and agencies,
  - medical/health contracts,
  - broad personnel-benefit proxy,
  - unallocated employer health premium gap.
- Label exact vs proxy.
- Never combine exact and proxy figures without a visible method note.

### R5 — PDF-first official sources

Risk:

Some best official sources are PDFs, not APIs. Choosing API convenience can pick the wrong source.

Observed evidence:

- California DOF revenue schedules are the right state revenue control source, but verified surface is PDF/extraction-first.
- CalPERS and budget health-benefit surfaces are PDF/HTML-heavy.

Mitigation:

- Source fit beats developer convenience.
- Build PDF extraction with table checks and source hashes where necessary.
- Use APIs as supplements, not substitutes, when they answer a narrower question.

### R6 — COFOG mapping as invented truth

Risk:

COFOG is valuable for comparability, but many source categories do not encode function.

Observed evidence:

- California `Acquisition Type` labels encode input/procurement type, not function.
- Department labels can be multi-function.
- No official U.S. federal-function to COFOG crosswalk verified.

Mitigation:

- COFOG assignments require basis/confidence/version.
- Unmapped is valid and visible.
- Do not force acquisition-type mapping.
- Prefer program/function codes when available.

### R7 — Lead engine reputational risk

Risk:

Anomaly leads can imply wrongdoing even when phrased casually.

Observed evidence:

- Repo already requires leads to be facts, never accusations.
- Public endpoint must show published leads only.

Mitigation:

- Human publication event required.
- Neutral copy only.
- No fraud/corruption words without official adjudication.
- Show method, coverage, limitations, and next questions.

### R8 — API-key and access-tier dependency

Risk:

Important sources may require keys or privileged tiers, causing invisible coverage holes.

Observed evidence:

- SAM public API requires api.data.gov key.
- SAM sensitive data, including EIN/TIN, requires elevated access.
- Socrata can be sampled anonymously but production may need app tokens.

Mitigation:

- Track key-dependent probes in `pending key` sections.
- Treat blocked/unverified endpoints as findings, not reasons to invent results.
- Add source health checks that report `unauthenticated`, `keyed`, `privileged`, or `blocked`.

### R9 — Append-only workflow mismatch

Risk:

Lead review and ingestion statuses often want updates, but repo storage is append-only.

Observed evidence:

- Architecture says update/delete are blocked.
- Lead status workflow must be append-only or superseding.

Mitigation:

- Model state changes as events.
- Current state is a projection.
- Persist failed runs as single inserted records, not updated lifecycle rows.

### R10 — Rate limits and bulk costs

Risk:

Bulk files are large; APIs may throttle; naive crawls waste time and lose reproducibility.

Observed evidence:

- USAspending bulk files are GB-scale.
- NPPES monthly ZIP is GB-scale.
- Socrata server-side aggregation is useful, but anonymous limits are not fully verified.

Mitigation:

- Prefer official bulk archives when available.
- Hash raw snapshots.
- Keep fixtures small and replayable.
- Use server-side aggregation for discovery, not final reproducibility unless query is stored.
- Register app tokens/keys for production where permitted.

## Severity table

| Risk | Severity | Probability | Near-term mitigation |
|---|---:|---:|---|
| False precision | High | High | Coverage and grain labels everywhere |
| UEI/EIN overclaim | High | High | Separate confirmed/candidate entity links |
| Fuzzy merge harm | High | Medium | Append-only aliases, confidence, no destructive merge |
| Healthcare overreach | High | High | Exact/proxy tiers |
| PDF-first sources | Medium | High | Build extraction when source fit demands it |
| COFOG invention | Medium | High | Unmapped visible, confidence required |
| Lead reputational risk | High | Medium | Human review + neutral copy |
| API-key dependency | Medium | Medium | Pending-key sections and health checks |
| Append-only mismatch | Medium | Medium | Event sourcing for workflow state |
| Rate/bulk cost | Medium | High | Bulk-first, fixtures, cached hashes |

## Architecture recommendations

1. Add `coverage_note` to every public total response.
2. Add `match_type` to entity aliases.
3. Add `lead_event` or equivalent append-only status workflow.
4. Add `source_access_tier` to source docs and source registry.
5. Add `classification_confidence` filtering to public comparison endpoints.
6. Add `proxy_amount` flag for healthcare and benefit estimates.
7. Add explicit `unmapped` buckets in every function view.
8. Add `blocked_probe` and `pending_key` subsections to source-doc template.

## Gate 9 closeout

Gate 9 status: closed.

Operator take: the project is viable if it markets itself as a sourced public-records evidence engine, not a complete universal ledger. The durable moat is provenance, humility about coverage, and safe entity linking. Fancy anomaly detection before those are boringly correct would be a bad idea.
