# Data quality and lead-safety policy

## Gate

Gate 8 — data quality and lead-safety rules.

Purpose:

Define the minimum evidence and language gates before Outlays can publish anomaly leads, especially where named vendors, nonprofits, agencies, or officials may be implicated by implication.

## EVR status

### Execute

Inspected repo rules and architecture for:

- provenance,
- decimal money,
- append-only corrections,
- classification provenance,
- entity resolution confidence,
- lead status lifecycle,
- public endpoint constraints,
- public payroll/individual exposure constraints.

### Verify

Verified in repo:

- `CLAUDE.md` says monetary figures require provenance and leads are facts, never accusations.
- `ARCHITECTURE.md` says no monetary figure without `raw_sha256` and `derivation_query`.
- `ARCHITECTURE.md` says classifications may be `source`, `rule`, `model`, or `human`, with required basis/provenance.
- `ARCHITECTURE.md` says no lead is publicly reachable unless a human sets `status='published'`.
- `ARCHITECTURE.md` says the system never asserts corruption, fraud, or unfairness about a named party.
- `ARCHITECTURE.md` says individual payroll rows are not publicly reachable.
- `ARCHITECTURE.md` says `/v1/leads` returns published leads only.
- `ARCHITECTURE.md` notes lead status workflow must be append-only, because update/delete are blocked.

Not verified:

- A dedicated lead-event table implementation.
- A UI workflow for reviewer signoff.
- A production copy deck for public lead pages.

### Report

This file is the Gate 8 policy report.

## Core rule

A lead is not an accusation.

A lead is:

```text
facts + source provenance + reproducible query + neutral statistical context + human publication decision
```

A lead is not:

```text
fraud claim | corruption claim | conflict-of-interest claim | wrongdoing claim | model hunch | vibes
```

If Outlays cannot show the underlying facts and query, the lead does not exist.

## Minimum publish gate

A lead may be published only if all checks pass:

1. Every cited monetary figure has:
   - `raw_sha256`,
   - `derivation_query`,
   - source URL or archive pointer,
   - exact decimal-string amount,
   - currency,
   - fiscal period,
   - grain.
2. Every classification assignment has:
   - source/rule/model/human basis,
   - confidence where not source-coded,
   - version.
3. Every entity match has:
   - authoritative identifier match, or
   - explicit non-authoritative match method and confidence.
4. The lead contains at least one reproducible query.
5. The lead names no individual payroll recipient.
6. The lead uses neutral language.
7. A human reviewer marks it published.
8. The review decision is append-only.

Fail any one gate -> lead remains draft or dismissed.

## Suggested status model

Existing architecture has:

```text
lead.status in draft | reviewed | published | dismissed
```

Because storage is append-only, do not mutate `lead.status` in place.

Recommended implementation:

```text
lead
  id
  rule_id
  fact_ids
  score
  generated_query
  created_at

lead_event
  id
  lead_id
  event_type: generated | reviewed | published | dismissed | superseded
  status_after
  reviewer
  review_note
  created_at
  raw_sha256 optional
  derivation_query optional
```

Public endpoint should compute the latest status from append-only events and return only leads whose latest status is `published`.

## Public language policy

Allowed language:

- `This vendor received X across Y contracts in Z period.`
- `This amount is unusually large relative to the comparison group.`
- `This transaction pattern may warrant review.`
- `The source data does not explain why this occurred.`
- `This is not evidence of wrongdoing by itself.`
- `Method: ...`
- `Limitations: ...`

Forbidden language:

- `fraud`, unless an official adjudication/source says fraud.
- `corrupt`, unless quoting an official adjudication/source.
- `kickback`, unless source-supported by legal record.
- `waste`, unless clearly framed as analyst opinion and not attached to a named party.
- `suspicious`, when attached to a named party; use `flagged by rule` or `warrants review`.
- `shell company`, unless proven by source documents.

## Rule taxonomy

Allowed initial lead rules:

### Concentration

Question:

```text
Does one vendor/nonprofit receive unusually high share inside a department/program/year?
```

Required context:

- denominator total,
- comparison group,
- source coverage percentage,
- top-N distribution,
- prior-year comparison if available.

### Fast growth

Question:

```text
Did a vendor/nonprofit's receipts grow sharply versus its prior baseline?
```

Required context:

- baseline years,
- current year,
- coverage consistency note,
- source/system changes caveat.

### Cross-agency concentration

Question:

```text
Does one entity appear across many agencies or programs?
```

Required context:

- entity-resolution method,
- confidence,
- agencies/program count,
- total and per-agency amounts,
- false-merge risk note.

### Round-number or split-payment pattern

Question:

```text
Are many awards just below a threshold or repeated at round values?
```

Required context:

- statutory/procurement threshold citation,
- full distribution around threshold,
- sample size,
- rule false-positive note.

### Nonprofit funding stack

Question:

```text
Does a nonprofit have public-award receipts, 990 revenue, and audit findings that should be viewed together?
```

Required context:

- EIN match source,
- UEI/name/address match confidence,
- 990 object IDs or BMF row,
- FAC audit year and finding fields,
- award-source coverage note.

## Data quality flags

Every fact may carry non-public or public quality flags. Recommended flags:

```text
missing_amount
missing_period
missing_entity
non_authoritative_entity_match
low_confidence_classification
aggregate_only
coverage_unknown
coverage_low
source_schema_changed
negative_or_refund_amount
duplicate_candidate
superseded
```

Lead rules should either exclude or explicitly disclose relevant flags.

## Coverage rule

Public lead pages must show coverage.

Format:

```text
Coverage: $X transaction/award facts loaded out of $Y official control total = Z%.
```

If no official control total exists:

```text
Coverage: unknown. No official control total verified for this source/grain.
```

No coverage note -> no public lead.

## Entity-resolution rule

Never overstate matches.

Allowed labels:

```text
identifier match: UEI/EIN/other authoritative id matched
name/address candidate: deterministic or fuzzy name/address candidate
name-only candidate: weak candidate, not merged without review
manual match: human-reviewed match with reviewer and note
```

The public UI should distinguish:

- same legal entity,
- likely same entity,
- possible same entity,
- unresolved.

## Payroll and individuals

Public endpoints must not expose named individual payroll rows.

Allowed:

- department-level payroll aggregate,
- job-class aggregate if privacy-safe,
- benefit aggregate,
- total compensation aggregate.

Not allowed:

- named employee row,
- named employee anomaly lead,
- compensation leaderboard by individual.

## Model usage

Models may help draft explanations or candidate rules, but model output is not evidence.

If a model classifies something:

- record model name/version,
- record prompt/template hash or basis,
- record confidence,
- keep source text available,
- require human review before publication.

## Reviewer checklist

Before publishing a lead, reviewer answers:

1. Can I click/source every monetary figure?
2. Does every amount reproduce from stored raw data and query?
3. Does the lead avoid accusing anyone?
4. Does entity matching have a clear confidence label?
5. Are limitations visible to the reader?
6. Is coverage disclosed?
7. Would the named party understand this as a data question rather than an allegation?
8. Is there a safer aggregate framing?

## UI copy template

Title:

```text
Review lead: high concentration in [program/department/year]
```

Body:

```text
Outlays found that [entity/category] accounted for [share] of [scope] in [period], based on [source]. This is a statistical flag, not evidence of wrongdoing.

Why it matters:
[neutral explanation]

Method:
[query and comparison group]

Source limits:
[coverage, grain, missing identifiers, update lag]

Next useful questions:
[records request, audit report, board minutes, contract file]
```

## Gate 8 closeout

Gate 8 status: closed.

- Lead-safety rules are compatible with existing architecture.
- Required next implementation: append-only `lead_event` workflow or equivalent.
- Public launch blocker: no lead can be public without human publication event, provenance, coverage, neutral language, and source limitations.
