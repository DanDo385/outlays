# Source: US:fed — entity identifiers and UEI/EIN linkage
- jurisdiction: US:fed
- base_url: https://open.gsa.gov/api/entity-api; https://sam.gov/content/entity-registration; https://www.irs.gov/charities-non-profits/exempt-organizations-business-master-file-extract-eo-bmf; https://www.irs.gov/about-irs/use-of-content-from-irsgov
- access_method: api | bulk-download | public-web
- working_query_or_file: SAM Entity Management API documentation; IRS EO BMF CSV extracts `eo1.csv`, `eo2.csv`, `eo3.csv`, `eo4.csv`; IRS EO BMF guide Publication 5926; IRS Use of Content page.
- confirmed_at: 2026-06-11T03:06:20Z
- response_status: mixed. SAM documentation verified; approved key-dependent probes against documented SAM Entity Management API paths returned empty 404 responses, so live public entity access and extract generation remain not verified. IRS EO BMF fallback verified live.
- formats_and_sizes: SAM Entity Management API documentation says synchronous JSON is paged at 10 records per page with a 10,000-record synchronous cap, and asynchronous extract downloads are available via `format=json` or `format=csv` with a 1,000,000-record extract cap. Live extract URL generation was not verified because key-based probes returned empty 404 responses. IRS EO BMF is CSV, split by state/region; regional CSV probes returned live headers and sample rows.
- fields_we_map: UEI, legacy DUNS, EIN/TIN, legal name, DBA/name aliases, physical/mailing address, NAICS, PSC, CAGE where available, tax-exempt organization classification fields, source-scoped alias confidence.
- identifiers_present: SAM public tier carries UEI plus entity public profile fields; SAM sensitive tier can include SSN/TIN/EIN but is restricted. IRS EO BMF carries EIN, name, care-of-name, street, city, state, ZIP, exemption/classification/status fields.
- grain: entity | registration | tax-exempt-organization profile | alias evidence
- update_cadence: SAM API cadence not confirmed in this gate; IRS EO BMF guide says updated monthly on the 2nd Monday of the month.
- posting_lag: IRS EO BMF page reported latest update 5/12/2026; direct CSV probes returned `last-modified` headers on 2026-06-08.
- license_or_terms: IRS content created or maintained by federal employees in the course of duties is not subject to copyright and may be freely copied; Treasury/IRS symbols, seals, insignia, badges, and endorsement-implying uses remain restricted. SAM terms include D&B attribution, no D&B Open Data bulk dissemination, no commercial/resale/marketing use of non-open D&B data, anti-harvesting restrictions, and FOUO/Sensitive public-dissemination limits.
- known_gaps: No public official UEI-to-EIN bridge is currently verified. SAM sensitive tier is not a public bridge. Operator-supplied key probes did not verify live SAM public API access or extract download generation; a SAM.gov profile/system-account Public API Key, rather than a generic api.data.gov key, may be required. Name/address crosswalks are feasible fallback evidence, not authoritative joins.
- notes: This gate is the identifier policy gate. The conclusion is intentionally conservative: exact identifiers first, source-scoped aliases second, no destructive entity merges from fuzzy evidence.

## EVR status

### Execute

Completed:

- Extracted official GSA/SAM Entity Management API documentation.
- Extracted official SAM entity-registration page.
- Retrieved operator-supplied key from 1Password item `Data.gov API Key` using field `credential`, without printing the secret.
- Ran key-dependent SAM Entity Management API probes against documented production and alpha paths for v1-v4, using both `X-Api-Key` header and `api_key` query parameter placements where appropriate.
- Ran narrow UEI-scoped `format=json` and `format=csv` extract probes; no broad all-entity extract job was requested.
- Re-ran the bounded public probe suite after the operator stated an api.data.gov key was available. `DATA_GOV_API_KEY` was not visible to the Hermes tool process; the operator approved reusing the existing 1Password item for the retry.
- Extracted official IRS public-disclosure pages for Form 990 context in Gate 2.
- Verified IRS EO BMF official page and guide.
- Verified IRS EO BMF live CSV regional files with HTTP/sample reads.
- Verified IRS site content-use page via search result snippet and URL discovery.

Not executed by design:

- No sensitive SAM POST request was attempted.
- No broad all-entity asynchronous extract request was attempted.

### Verify

Verified:

- SAM Entity Management API documentation distinguishes public, FOUO, and sensitive tiers.
- Public SAM data includes public entity fields such as name, UEI, registration details, physical/mailing addresses, business types, PSC, NAICS, and points of contact name/address.
- Sensitive SAM data can include banking information and SSN/TIN/EIN and requires elevated access; sensitive access is not a public official UEI-to-EIN bridge.
- Public SAM API access requires an API key. The GSA documentation says SAM.gov users obtain the Public API Key from the SAM.gov workspace/profile account-details page; sensitive access additionally requires system-account credentials and Basic Auth.
- Key-dependent live probes using the operator-supplied 1Password key returned HTTP 404 with empty bodies for the documented `https://api.sam.gov/entity-information/v[1-4]/entities` paths, for both the Gate 2 UEI `MW4NM5KU2M81` and documentation sample UEIs `ZQGGHJH74DW7~JH9ZARNKWKC7`.
- The same documented paths also returned HTTP 404 with empty bodies against `https://api-alpha.sam.gov` for v2-v4 sample-UEI probes.
- Query-parameter and header key placement both produced the same empty 404 result; no `X-RateLimit-*` headers or api.data.gov-style JSON error codes were observed.
- Narrow `format=json` and `format=csv` extract probes returned empty 404 responses, so live public extract download URL generation was not verified with this key.
- The re-run probe suite produced the same result: 30 public GET probes, 30 HTTP 404 responses, empty bodies, `Content-Length: 0`, no `X-RateLimit-*` headers, no public response schema, and no extract download token or URL.
- Official documentation, not live payload, supports the field conclusion: public data includes UEI, names, registration details, addresses, business types, PSC, NAICS, and points of contact name/address; Sensitive CUI is where SSN/TIN/EIN and banking fields are described.
- IRS EO BMF is official, CSV, cumulative, and updated monthly on the 2nd Monday of the month.
- IRS EO BMF CSV header includes `EIN`, `NAME`, `ICO`, `STREET`, `CITY`, `STATE`, `ZIP`, and many exemption/classification fields.
- IRS EO BMF content-use rights are favorable for federal-employee-created/maintained site content, with Treasury/IRS mark restrictions.

Not verified:

- Any public official source that exposes UEI and EIN together for all SAM entities.
- Any public SAM API response with EIN/TIN at the public tier.
- Any public SAM bulk extract containing EIN/TIN.
- Any live public SAM Entity Management API response using the operator-supplied key.
- Any live SAM public extract download URL generated with the operator-supplied key.
- Any observed SAM `X-RateLimit-*` headers from live probes.
- Practical match rate between SAM public entity profiles and IRS EO BMF by normalized name/address.

### Report

This file is the Gate 3 report artifact.

## Blunt conclusion

No public official UEI-to-EIN bridge is currently verified.

That is the uncomfortable center of the Outlays data model. USAspending and SAM public surfaces are UEI/name/address-first. IRS nonprofit surfaces are EIN/name/address-first. FAC gives a strong UEI↔EIN bridge only for audited entities/report years, not a universal entity crosswalk.

So the policy is:

1. Exact UEI joins where both sources expose UEI.
2. Exact EIN joins where both sources expose EIN.
3. FAC-derived UEI↔EIN pairs only as source-scoped alias evidence.
4. SAM-public-to-IRS-EO-BMF name/address matches only as confidence-scored candidate aliases.
5. No destructive merges from names alone. Names lie. Addresses drift. Humans abbreviate everything. Computers believe them too easily. Bad combo.

## SAM.gov Entity Management API

Official documentation:

```text
https://open.gsa.gov/api/entity-api
```

Official entity-registration page:

```text
https://sam.gov/content/entity-registration
```

Verified documentation facts:

- SAM assigns the Unique Entity ID during entity registration.
- Entities that only need a UEI may request a UEI without completing full registration.
- Public SAM data includes unclassified data such as:
  - legal/business name,
  - UEI,
  - registration details,
  - physical and mailing addresses,
  - business types,
  - PSC codes,
  - NAICS codes,
  - points of contact name and address.
- FOUO/CUI adds additional restricted entity data.
- Sensitive CUI can include banking information and tax identifiers such as SSN/TIN/EIN.
- Sensitive API access requires elevated permissions and POST-style sensitive access flows.
- SAM API supports JSON by default and CSV via the `format` parameter.
- SAM API public access still requires an API key.

### SAM public tier vs sensitive tier

| Tier | Identifier relevance | Public bridge value |
|---|---|---|
| Public | UEI, name, registration, physical/mailing address, business types, PSC, NAICS | Good for UEI-first entity profile and fuzzy/structured alias evidence. |
| FOUO/CUI | Public plus additional restricted fields | Not assumed available for public Outlays ingestion. |
| Sensitive CUI | Can include SSN/TIN/EIN and banking information | Not public. Not a valid public UEI↔EIN bridge. |

The important distinction: SAM apparently has EIN/TIN in restricted contexts, but that does not solve the public data product. A private/elevated lookup cannot be treated as an open reproducible join unless the access terms allow it and the product can publish derived linkage lawfully.

## Resolved SAM pending-key probe finding

The earlier SAM probe was blocked by tool approval. The operator later supplied a 1Password item title, `Data.gov API Key`, and approved key-dependent probes. The key was retrieved from the Dev vault `credential` field without printing it.

The exact endpoint family from the GSA documentation is:

```text
https://api.sam.gov/entity-information/v1/entities
https://api.sam.gov/entity-information/v2/entities
https://api.sam.gov/entity-information/v3/entities
https://api.sam.gov/entity-information/v4/entities
```

The documentation also lists alpha paths under:

```text
https://api-alpha.sam.gov/entity-information/v[1-4]/entities
```

### Key-dependent probe results

Probe timestamp: 2026-06-11T02:53:24Z.

Known UEIs tested:

```text
MW4NM5KU2M81
ZQGGHJH74DW7~JH9ZARNKWKC7
```

Key placements tested:
- `X-Api-Key` header.
- `api_key` query parameter.
- both header and query parameter together for selected variants.

Public-section probes tested:
- default public response by UEI.
- `includeSections=entityRegistration,coreData`.
- `includeSections=All` for the Gate 2 UEI only.

Extract probes tested:
- UEI-scoped `format=json`.
- UEI-scoped `format=csv`.

Observed result:
- All key-dependent entity probes returned HTTP 404 with empty response bodies and `Content-Length: 0`.
- No api.data.gov-style JSON error code was returned.
- No `X-RateLimit-Limit` or `X-RateLimit-Remaining` headers were observed.
- No public response schema was returned, so no public EIN/TIN field can be confirmed or denied from live response payloads.
- No extract download URL or token was returned, so public extract bulk-downloadability is documentation-supported but not live-verified with this key.

Interpretation:
- The pending-key work item is resolved as executed, not as successful API access.
- The GSA documentation says SAM.gov users obtain a Public API Key from the SAM.gov workspace/profile account-details page. The tested item is named `Data.gov API Key`; if this is a generic api.data.gov key rather than a SAM.gov Public API Key, it may not authorize or route SAM Entity Management API requests.
- The current Outlays data model should continue to treat SAM public entity data as desirable future UEI/name/address evidence, not as a verified live ingestion source.

### P2 re-run with approved key source

The operator later stated that an api.data.gov key was available as `DATA_GOV_API_KEY`. That environment variable was not visible to the Hermes tool process. The operator then approved reusing the existing 1Password item `Data.gov API Key` for the P2 retry.

Probe timestamp: 2026-06-11T03:06:20Z.

Bounded public GET probe suite:

- 30 total probes.
- 16 public entity-by-UEI probes across v1-v4, header and query-key auth, for the Gate 2 UEI and GSA documentation sample UEIs.
- 6 minimal paged-API probes across v2-v4, header and query-key auth, using `samRegistered=Yes`, `registrationStatus=A`, `includeSections=entityRegistration,coreData`, and `limit=1`.
- 8 narrow extract-generation probes across v3-v4, header and query-key auth, using the GSA documentation sample UEIs and `format=json` / `format=csv`.
- No sensitive POST request.
- No broad all-entity extract request.

Observed live result:

- 30 / 30 probes returned HTTP 404.
- Response bodies were empty.
- Observed response header evidence was limited to `Content-Length: 0`.
- No `X-RateLimit-Limit`, `X-RateLimit-Remaining`, or `X-RateLimit-Reset` headers were observed.
- No api.data.gov-style JSON error code was returned.
- No public entity schema was returned.
- No extract download token, download URL, CSV, JSON file, or ZIP was returned.

Documentation-supported, not live-verified:

- Paged API access: the GSA documentation says synchronous JSON returns 10 records per page and can return only the first 10,000 records.
- Bulk/extract-style access: the same documentation says the API can serve as an Extract API with `format=csv` or `format=json`, asynchronous download links, and a 1,000,000-record extract cap.
- Public fields: the documentation says Public Data includes name, UEI, registration details, physical and mailing addresses, business types, PSC, NAICS, and points of contact name/address.
- EIN absence from public tier: the documentation places tax identifiers such as SSN/TIN/EIN in Sensitive CUI, not Public Data. No live public payload was returned to independently confirm field absence from a response schema.

P2 verdict:

- Public SAM entity API and extract availability remain documentation-supported but not live-verified with the approved key source.
- The Outlays ingestion plan should not depend on SAM Entity Management as a live Phase 1 source until a SAM.gov Public API Key or system-account entitlement returns a schema, page, extract token, or downloadable artifact.

## IRS EO BMF fallback verification

Official page:

```text
https://www.irs.gov/charities-non-profits/exempt-organizations-business-master-file-extract-eo-bmf
```

Official TEOS bulk context page:

```text
https://www.irs.gov/charities-non-profits/tax-exempt-organization-search
```

Official guide:

```text
https://www.irs.gov/pub/irs-pdf/p5926.pdf
```

IRS describes the Exempt Organizations Business Master File Extract as a cumulative dataset containing the most recent information IRS has for tax-exempt organizations that received a determination of tax-exempt status.

### Publication, format, and files

The IRS page publishes EO BMF files by state/region as CSV.

Observed regional CSV URLs:

```text
https://www.irs.gov/pub/irs-soi/eo1.csv
https://www.irs.gov/pub/irs-soi/eo2.csv
https://www.irs.gov/pub/irs-soi/eo3.csv
https://www.irs.gov/pub/irs-soi/eo4.csv
```

The IRS page also publishes individual state files such as:

```text
https://www.irs.gov/pub/irs-soi/eo_tx.csv
https://www.irs.gov/pub/irs-soi/eo_il.csv
https://www.irs.gov/pub/irs-soi/eo_ca.csv
```

### Verified EO BMF CSV headers

Live regional CSV samples returned the same header:

```text
EIN, NAME, ICO, STREET, CITY, STATE, ZIP, GROUP, SUBSECTION, AFFILIATION, CLASSIFICATION, RULING, DEDUCTIBILITY, FOUNDATION, ACTIVITY, ORGANIZATION, STATUS, TAX_PERIOD, ASSET_CD, INCOME_CD, FILING_REQ_CD, PF_FILING_REQ_CD, ACCT_PD, ASSET_AMT, INCOME_AMT, REVENUE_AMT, NTEE_CD, SORT_NAME
```

Core bridge-relevant fields:

- `EIN`
- `NAME`
- `ICO` — in-care-of name
- `STREET`
- `CITY`
- `STATE`
- `ZIP`
- `NTEE_CD`
- `SUBSECTION`
- `STATUS`

### Verified EO BMF sample rows

`eo1.csv` first sample row:

```text
EIN=000019818
NAME=PALMER SECOND BAPTIST CHURCH
STREET=1050 THORNDIKE ST
CITY=PALMER
STATE=MA
ZIP=01069-1507
```

`eo2.csv` first sample row:

```text
EIN=002120849
NAME=ANCILLA DOMINI SISTERS INC
STREET=LOCAL
CITY=DONALDSON
STATE=IN
ZIP=46513-0000
```

`eo3.csv` first sample row:

```text
EIN=000260049
NAME=CORINTH BAPTIST CHURCH
STREET=PO BOX 92
CITY=HOSFORD
STATE=FL
ZIP=32334-0092
```

`eo4.csv` first sample row:

```text
EIN=010674605
NAME=IGLESIA FUENTE DE AGUA VIVA ORLANDO FL INC
ICO=% RODOLFO O FONT
STREET=PO BOX 3869
CITY=CAROLINA
STATE=PR
ZIP=00984-3869
```

### Verified EO BMF HTTP facts

Live probes returned:

| File | Status | Content type | Last modified | Sample/read bytes observed |
|---|---:|---|---|---:|
| `eo1.csv` | 200 | `text/csv` | Mon, 08 Jun 2026 04:10:48 GMT | 48,931,975 |
| `eo2.csv` | 200 | `text/csv` | Mon, 08 Jun 2026 04:10:51 GMT | 126,787,558 |
| `eo3.csv` | 200 | `text/csv` | Mon, 08 Jun 2026 04:10:54 GMT | 166,149,299 |
| `eo4.csv` | 200 | `text/csv` | Mon, 08 Jun 2026 04:10:56 GMT | 876,431 |

Note: the probe requested a byte range but the server returned full files. The byte counts above are observed response bytes from the local probe, not independently confirmed complete-file sizes by HEAD.

### EO BMF cadence

Publication 5926 says:

```text
The dataset is updated monthly, on the 2nd Monday of the month.
```

The IRS EO BMF page extract showed:

- Latest update: `5/12/2026`.
- Total record count: `1,966,267`.

The live CSV `last-modified` headers showed `Mon, 08 Jun 2026`, which is consistent with a monthly refresh cycle.

### EO BMF license / content-use status

IRS content-use page:

```text
https://www.irs.gov/about-irs/use-of-content-from-irsgov
```

Search result extract states:

```text
Content on this website that was created or maintained by federal employees in the course of their duties is not subject to copyright and may be freely copied.
```

Important restriction:

- Treasury/IRS symbols, emblems, seals, insignia, badges, and endorsement-implying uses are restricted.

Interpretation for Outlays:

- EO BMF is suitable as an official public fallback source for nonprofit EIN/name/address evidence.
- Do not imply IRS endorsement.
- Preserve source URL and retrieval date in provenance.

## Recommended fallback: SAM public entity data ↔ IRS EO BMF crosswalk

Because SAM documentation says public SAM data exposes UEI/name/address/NAICS/PSC but key-dependent probes did not return a live schema, the best nonprofit EIN fallback remains a cautious name/address crosswalk between:

1. SAM public entity profiles:
   - UEI,
   - legal name,
   - DBA names if exposed,
   - physical/mailing address,
   - NAICS/PSC,
   - registration status.

2. IRS EO BMF:
   - EIN,
   - `NAME`,
   - `ICO`,
   - `STREET`,
   - `CITY`,
   - `STATE`,
   - `ZIP`,
   - NTEE and exemption fields.

This is not a bridge. It is a candidate-alias generator.

### Suggested matching tiers

| Tier | Evidence | Action |
|---|---|---|
| Exact identifier | Same UEI or same EIN from two sources | Join confidently within identifier namespace. |
| Official bridge | FAC `auditee_uei` + `auditee_ein` on same report/entity | Store as source-scoped UEI↔EIN alias evidence. |
| High-confidence name/address | Normalized legal name exact or near-exact, ZIP5 exact, street number/name compatible, city/state exact | Create candidate alias with high confidence and provenance; require review/threshold before user-facing “same entity” claim. |
| Medium-confidence name/address | Name similar, city/state/ZIP compatible, address incomplete or PO box mismatch | Candidate only. Never auto-merge. |
| Low-confidence | Name-only match, common names, national affiliates, missing address | Do not merge. Use for search suggestions only. |

### Expected limitations

- Nonprofits may use different legal names across SAM, IRS, FAC, and USAspending.
- SAM physical address may differ from IRS filing/headquarters address.
- Parent/subsidiary/fiscal-sponsor structures can create legitimate name/address divergence.
- Multi-EIN or multi-UEI organizations require one-to-many alias modeling.
- Common names create false positives.
- PO boxes and in-care-of names weaken address matching.
- Churches and some self-declared organizations may be absent from EO BMF, per Publication 5926 exclusions.
- EO BMF state/region is based on filing address/headquarters, not necessarily operating geography.

## Identifier-first matching policy

Outlays should implement entity resolution as append-only evidence, not as destructive truth mutation.

### Canonical identifier namespaces

- `uei`: SAM/USAspending/FAC public entity and award surfaces.
- `ein`: IRS/ProPublica/FAC nonprofit/tax/audit surfaces.
- `duns`: legacy historical identifier only; do not use as post-2022 primary key.
- `name_address`: normalized alias evidence only.

### Matching rules

1. Never store one global `entity_id` as if every source agrees.
2. Store every source assertion separately:
   - source system,
   - retrieval timestamp,
   - raw identifier,
   - normalized identifier,
   - raw name/address,
   - normalized name/address,
   - confidence,
   - matching method,
   - source URL/file hash when available.
3. Let user-facing views say “matched by FAC report” or “candidate match by name/address,” not just “same entity.”
4. Preserve conflicts. Do not overwrite a prior alias because a newer source disagrees.
5. Require stronger evidence before connecting dollars to an EIN-backed nonprofit profile than before showing a search suggestion.

### Proposed schema direction

Minimum tables/concepts:

```text
source_record
entity_identifier
entity_alias_evidence
entity_match_candidate
entity_cluster_snapshot
```

Where:

- `source_record` keeps provenance for every raw file/API row.
- `entity_identifier` stores exact identifiers by namespace.
- `entity_alias_evidence` stores claimed links from official sources such as FAC UEI↔EIN.
- `entity_match_candidate` stores derived name/address candidates with score and method.
- `entity_cluster_snapshot` is a derived view, not a source of truth.

## Gate 3 verdict

Gate 3 can close for now.

- Public SAM tier is documentation-supported for UEI/name/address/NAICS/PSC entity profiles, but live access was not verified with the operator-supplied key.
- EIN/TIN belongs to restricted/sensitive SAM access, not a public bridge.
- No public official universal UEI-to-EIN bridge is currently verified.
- FAC is the best verified official UEI↔EIN evidence source for audited nonprofits, but coverage is partial.
- IRS EO BMF is confirmed as the fallback nonprofit EIN/name/address corpus.
- The recommended bridge strategy is conservative name/address candidate matching between SAM public data and EO BMF, enhanced by FAC where available, with match-rate and false-positive limitations clearly exposed.

## Carry-forward to Gate 4

Gate 4 healthcare feasibility must keep this policy intact:

- Healthcare identifiers may introduce NPI, CCN/CMS Certification Number, CLIA, taxonomy, TIN/EIN, organization names, and addresses.
- Do not assume provider identifiers solve UEI↔EIN.
- Treat healthcare crosswalks as additional alias evidence unless an official source explicitly claims the join.
- For hospitals, FQHCs, universities, state agencies, and nonprofit health centers, expect multi-entity structures and many-to-one/many-to-many relationships.
