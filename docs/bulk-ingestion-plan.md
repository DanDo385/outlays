# Bulk ingestion plan

## Addendum gate

A4 — synthesis: rank verified sources by verifiable dollars covered, engineering effort, grain, and entity-identifier quality; recommend Phase 1 ingestion order, maximum five sources; post the operator exit summary.

## EVR status

### Execute

Reviewed Gates 1-7 deliverables plus the addendum licensing/methodology findings:

- `docs/sources/us-fed-bulk.md`
- `docs/sources/nonprofit-stack.md`
- `docs/sources/entity-identifiers.md`
- `docs/feasibility-healthcare-crosscut.md`
- `docs/sources/us-ca-revenue.md`
- `docs/cofog-references.md`
- `docs/sources/us-ny.md`
- `docs/licensing-matrix.md`
- `docs/leads-methodology.md`

Recovered the original brief language from session history because the reconstructed `docs/research-brief.md` only covered Gates 1-3 and did not include the Gate 11 exit-summary bullets.

### Verify

Ranking uses only verified findings from completed source docs:

- live endpoint or file availability;
- verified format/size/shape where captured;
- known identifier fields;
- grain confirmed by docs/probes;
- licensing/terms status from A1;
- feasibility caveats from Gates 4-7.

No new source claims are introduced here without a prior gate note.

### Report

This document is the addendum report.

## Ranking rubric

Scores are qualitative because several sources were verified for shape and access, not complete extracted dollar totals.

- Verifiable dollars covered:
  - Very high: national/federal or official control-total source.
  - High: major public dollars but incomplete scope or filtered population.
  - Medium: state/public authority or domain-specific official totals.
  - Low/none: reference/identifier source, not a money source.
- Engineering effort:
  - Low: stable bulk CSV/JSON with clear schema.
  - Medium: generated bulk/API jobs, pagination, joins, or mild extraction.
  - High: PDFs, async jobs, huge archives, source-specific caveats.
- Grain:
  - Transaction beats award beats filing/audit/entity beats aggregate/reference.
- Identifier quality:
  - High: official stable identifiers such as UEI/EIN/NPI plus names/addresses.
  - Medium: names plus local IDs/source IDs.
  - Low: names only, ambiguous or no stable entity IDs.

## Ranked verified sources

| Rank | Source | Verifiable dollars covered | Engineering effort | Grain | Identifier quality | Licensing / terms risk | Verdict |
|---:|---|---|---|---|---|---|---|
| 1 | USAspending FY2024 bulk awards + generated subawards/accounts | Very high: federal awards and account/control surfaces; verified FY2024 counts for contracts, assistance, subawards; account downloads verified by narrow probe | High: multi-GB archives, async account/subaward jobs, dictionary parity still partial | Transaction / award / subaward / account aggregate | High for UEI/DUNS; EIN not observed in inspected contract archive; account linkage requires File C/account downloads | No explicit dataset license stated; public access under DATA Act; D&B-derived fields require caution | First bulk target. Biggest prize and coverage anchor. |
| 2 | IRS Form 990 e-file bulk + EO BMF | High for nonprofit financial universe, not government spending by itself | Medium: bulk XML/indexes plus parsing | Filing / organization | High EIN; names/addresses; no UEI | IRS federal-employee content freely copied; credit requested | Essential nonprofit financial context and EIN spine. |
| 3 | Federal Audit Clearinghouse bulk/API | High for audited federal-award recipients above Single Audit threshold; incomplete for smaller recipients | Medium: CSV/API with multiple related tables | Audit/federal-award rows, findings, auditee | High: auditee identifiers including UEI/EIN-related surfaces verified in Gate 2 docs | Explicit public-domain statement; tribal-reporting exceptions | Best public-domain bridge between federal awards and audited recipient context. |
| 4 | SAM.gov public entity data | No direct dollars; critical identity layer | Medium/high: API key, terms, public/FOUO/sensitive boundaries, D&B restrictions | Entity / registration | Very high for UEI; EIN/TIN is Sensitive/CUI and not public-product material | Significant restrictions for FOUO, Sensitive, D&B-supplied data | Required for UEI-centered entity resolution, but do not use restricted fields. |
| 5 | California DOF/eBudget revenue schedules | Medium/high for official California state revenue/control totals, aggregate grain | High: extraction from official budget schedules/PDFs rather than clean API | Aggregate budget/revenue schedule | Low/medium: state departments/funds/categories, not vendor/entity grain | No explicit license found on checked surfaces | Best first CA revenue adapter despite extraction work. |
| 6 | California data.ca.gov / PIT annual report / related tax stats | Medium for specific California revenue/statistical domains | Medium: CKAN resources; dataset-specific metadata must be captured | Aggregate/statistical tables | Low/medium: category/date/geography more than entity | Portal says most datasets public domain but dataset-specific license required | Secondary CA revenue/statistical sources after DOF backbone. |
| 7 | California CDTFA sales/use tax statistics | Medium for tax-specific revenue context | Medium/high depending table format | Aggregate/statistical | Low: categories/geographies, not entities | No explicit license verified in gate docs | Reference/secondary source, not first adapter. |
| 8 | California SCO ByTheNumbers | Medium for local government finances, not state-level revenue | Medium: Socrata-style portal | Aggregate/local government finance | Medium for local agencies; not state-vendor grain | License not verified | Exclude from first CA state revenue adapter; revisit for local-government module. |
| 9 | New York Data.NY.gov MTA Procurements | Medium for MTA/public-authority procurement, narrow jurisdiction | Low/medium: Socrata API, server-side aggregation verified | Contract/procurement rows with fiscal-year spend fields | Medium/low: vendor name and transaction number; no UEI/EIN verified | Dataset-level license metadata not independently captured in A1 | Good portal mechanics test; not Phase 1 bulk coverage. |
| 10 | CMS NPPES / Provider Data Catalog | No direct government-dollar source in this sprint; strong healthcare entity reference | Medium: public files/API, large provider datasets | Entity/reference | High NPI; not vendor payment identity by itself | FOIA-disclosable; named license not found in checked docs | Reference for healthcare/entity enrichment, not core Phase 1 money ingest. |
| 11 | Acquisition.gov PSC Manual | No dollars; classification reference | Low | Reference/classification | None/entity not applicable | No explicit license found on PSC page | Useful for federal procurement classification mapping. |
| 12 | Census NAICS/API | No dollars for this sprint; classification/statistical reference | Low/medium | Reference/classification/statistical | None/entity not applicable; anti-reidentification obligations | Explicit API terms and required disclaimer | Use as classification reference only. |
| 13 | Eurostat COFOG manual/reference | No direct dollars; classification reference | Low | Reference/classification | Not applicable | Reuse generally permitted with exceptions | Use for COFOG mapping and caveats. |
| 14 | IMF GFSM reference | No direct dollars; reference only | Low | Reference/classification | Not applicable | Terms/license not confirmed | Cite only; do not redistribute. |
| 15 | White House / OMB object-class PDF | No direct transactional dollars; object-class reference/control context | Medium: PDF reference | Reference/aggregate classification | Not applicable | Government-produced materials not copyright protected per White House policy | Use as federal object-class reference. |
| 16 | ProPublica Nonprofit Explorer | No original public dollars beyond derived IRS nonprofit data | Low API effort, but third-party terms | Entity/filing derived API | High EIN | Restrictive ProPublica data terms; no raw redistribution | Convenience/reconciliation only; do not make canonical source. |
| 17 | CalHR / CalPERS healthcare benefits pages/reports | Healthcare context, not verified department-grain spending source | High if extracting; source did not answer target question | Reference/report | Low for entity/payment | License not confirmed | Context only. Gate 4 says healthcare cross-cut not computable at desired grain. |
| 18 | U.S. House Budget Committee budget-functions page | No direct dollars; explanatory classification reference | Low | Reference | Not applicable | No explicit license found | Cite only. |

## Recommended Phase 1 ingestion order

Maximum five sources, per brief.

### 1. USAspending FY2024 bulk awards, subawards, and account downloads

Start here because it is the largest verified prize: federal prime contract transactions, prime assistance transactions, generated subawards, and account downloads together define the core federal spending surface. Gate 1 verified live archive indexes, multi-GB FY2024 contract/assistance archive sizes, count-API row estimates, generated account-download shape, generated subaward shape, and a small archive header with UEI/DUNS and award identifiers. Engineering effort is high, but coverage leverage is unmatched. Include account downloads early, not as a later nicety, because award archives alone do not provide clean account/control/object-class linkage.

### 2. IRS Form 990 e-file bulk plus IRS EO BMF

Ingest this second because it supplies the EIN-centered nonprofit spine. It is not a government-spending source by itself, but it is essential context for grants and nonprofit recipients: organization profiles, filings, financials, and public EIN/name/address fields. This makes the nonprofit join tractable without depending on third-party ProPublica terms. It also lets the entity model represent EIN as first-class public evidence while preserving the Gate 3 finding that SAM.gov did not provide a public UEI→EIN bridge.

### 3. Federal Audit Clearinghouse bulk/API

Ingest FAC third because it is public-domain, audit-grounded, and directly relevant to federal award recipients above the Single Audit threshold. FAC bridges federal awards, auditee identity, audit findings, SEFA context, and corrective-action/finding surfaces. It will not cover every recipient or every grant, but its evidence quality is high and it is a better official bridge for nonprofit/public-recipient review than fuzzy name matching alone. FAC should join to IRS by EIN where available and to USAspending by UEI/name/address only with confidence scoring and append-only aliases.

### 4. SAM.gov public entity extract, once API access is available

Ingest public SAM entity data fourth, or run it in parallel as soon as API credentials exist, because UEI is the federal award identity backbone. Do not use SAM Sensitive or FOUO data for the public product; EIN/TIN belongs in the restricted bucket and is not a public bridge. SAM improves name/address normalization, parent-child entity context, NAICS/PSC fields, and public registration metadata, but D&B-supplied fields and SAM terms require careful provenance and redistribution flags. Treat SAM as an identity/reference layer, not a money source.

### 5. California DOF/eBudget revenue schedules

Ingest California DOF/eBudget fifth because Outlays already has California spending direction and Gate 5 found DOF/eBudget is the best first state revenue adapter. This is aggregate, not transaction grain, and engineering effort is extraction-heavy, but it gives official state revenue/control totals. That matters strategically: Outlays should show that it can trace both sides of public finance, not only vendor/procurement outflows. Dataset-license status remains conservative, so preserve source URLs, retrieval timestamps, and `no_explicit_license_stated` metadata until a source-specific terms review closes the gap.

## Explicit non-Phase-1 calls

### New York MTA Procurements

Keep as a mechanics fixture and Phase 2 candidate. It is useful because Socrata server-side aggregation was verified, but it is narrow public-authority scope and lacks strong universal identifiers. It should not displace federal/IRS/FAC/SAM bulk work.

### California SCO ByTheNumbers

Do not use as the first California state revenue adapter. Gate 5 found it is local-government finance rather than the state-level revenue surface Outlays needed.

### ProPublica Nonprofit Explorer

Use for manual reconciliation and UX inspiration, not canonical ingestion. IRS is the source-of-record path and ProPublica terms restrict raw redistribution/resale.

### Healthcare cross-cut sources

Do not create a healthcare total-cost ingestion track yet. Gate 4 returned an honest negative: federal object-class/account sources can support some aggregate analysis, but the requested cross-cut including employee benefits buried in departments is not publicly computable at the desired department/entity grain for both federal and California.

### COFOG / PSC / Census / Eurostat / IMF references

Keep as classification references. They are not money-ingestion sources and should not consume Phase 1 ingestion capacity.

## Builder-facing architecture implication

Proposed Decision Log entry only; do not silently change the spec:

```text
D26 — Source registry gates ingestion by license, coverage, and identifier evidence.
Every adapter declares source-level metadata before facts persist: license_status,
license_url/terms_url, retrieval timestamp, source grain, amount basis, official control
total availability, identifier fields present, and coverage denominator when known.
The orchestrator refuses public-ingest mode for license_status='unknown' unless the
source is marked reference_only, and it stores source coverage separately from fact rows.
Rationale: Gates 1-11 showed that the hardest failures are not parsing failures; they are
false precision around public reuse rights, dollars covered, and UEI/EIN identity joins.
```

## Exit summary for the operator

1. Biggest verified prize: USAspending bulk remains the first prize. FY2024 prime contracts and assistance archives are live and huge, generated subaward/account jobs work, and account downloads are mandatory for object-class/program-activity/control-total linkage.

2. Biggest blocker found: there is no verified public official UEI→EIN bridge. SAM.gov public surfaces center UEI, while IRS/FAC nonprofit surfaces center EIN. SAM Sensitive data includes TIN/EIN and is not usable for a public/commercial product. Identity resolution must be append-only and confidence-scored, not a destructive merge.

3. Healthcare cross-cut verdict: not computable at the requested grain today. Federal account/object-class data can support aggregate personnel-benefit analysis, but the full total-government healthcare cost including employee health benefits embedded inside police, education, postal, and other departments is not publicly computable end-to-end, and California department-grain health premium expenditure was not verified as machine-readable.

4. Nonprofit join-key verdict: IRS 990/EO BMF gives a strong EIN-centered nonprofit spine; FAC adds public-domain audit/federal-award context and can help bridge some recipients; USAspending award surfaces verified UEI/DUNS but not EIN in the inspected contract header. The join is feasible for subsets, not universal, and every UEI/EIN/name/address association needs source-backed confidence.

5. Architecture response: propose Decision Log D26: source registry gates ingestion by license, coverage, and identifier evidence. The builder should not only persist facts; it should persist whether the source is legally reusable, what dollars are covered, what the amount basis is, which identifiers are authoritative, and what coverage denominator exists. New robot opinion: that metadata is not bureaucracy; it is the product moat.
