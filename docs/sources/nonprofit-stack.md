# Source: US:fed — nonprofit data stack
- jurisdiction: US:fed
- base_url: https://www.irs.gov/charities-non-profits/form-990-series-downloads; https://projects.propublica.org/nonprofits/api; https://www.fac.gov/data/download; https://api.fac.gov/
- access_method: api | bulk-download
- working_query_or_file: IRS `https://apps.irs.gov/pub/epostcard/990/xml/2024/index_2024.csv`; IRS `https://apps.irs.gov/pub/epostcard/990/xml/2024/2024_TEOS_XML_01A.zip`; ProPublica `GET /nonprofits/api/v2/organizations/742089103.json`; FAC `https://app.fac.gov/dissemination/public-data/gsa/full/general.csv`; FAC `federal_awards.csv`; FAC `additional_eins.csv`; FAC `additional_ueis.csv`; FAC `findings.csv`; USAspending worked-example query by recipient search text `Atascosa Health Center`.
- confirmed_at: 2026-06-10T18:36:47Z
- response_status: live; FAC API endpoints are restricted without an API key but FAC bulk CSV downloads are live.
- formats_and_sizes: IRS annual CSV index and monthly/batch XML ZIPs; ProPublica JSON API; FAC CSV full exports and XLSX/API dictionaries; USAspending JSON API response for worked example.
- fields_we_map: EIN -> entity.identifier/ein; UEI -> entity.identifier/uei; organization/auditee/recipient name and address -> entity alias evidence; Form 990 filing metadata -> nonprofit profile and tax-period evidence; FAC federal award rows -> audited federal program/expenditure evidence; FAC findings rows -> single-audit finding evidence; USAspending assistance awards -> award-grain fiscal facts and UEI/name recipient evidence.
- identifiers_present: IRS Form 990 index has EIN; ProPublica organization endpoint is keyed by EIN; FAC `general.csv` has `auditee_ein` and `auditee_uei`; FAC `additional_eins.csv` and `additional_ueis.csv` support multiple identifiers per report; USAspending worked-example award search returned recipient UEI and recipient name, not EIN in the selected fields.
- grain: transaction | award | aggregate
- update_cadence: IRS page publishes annual index files and ZIP batches organized by year/month; FAC current data covers 2016-present with downloadable full and sliced CSVs; ProPublica data source updates are reflected in API metadata. Exact cadence for ProPublica API refresh not confirmed.
- posting_lag: IRS page last reviewed/updated May 20, 2026; FAC pages returned `last-modified: Mon, 01 Jun 2026 19:11:33 GMT`; ProPublica Atascosa response reported `data_source: current_2026_04_15`. Source-to-public lag not fully characterized.
- license_or_terms: ProPublica Data Terms page says, “In general, you may use the free data published by ProPublica under the following terms. However, there may be different terms included for some datasets. It is your responsibility to read carefully any specific terms included with the data you download from our website.” IRS/FAC explicit redistribution license not yet confirmed in this gate; `.gov` availability is not a redistribution license.
- known_gaps: No official UEI-to-EIN bridge confirmed in this gate; FAC API requires an API key for direct API calls; ProPublica general API rate limit not confirmed beyond documented PDF-link rate limiting; IRS direct XML object URL pattern from `OBJECT_ID` was not confirmed outside ZIP batch access; USAspending assistance-specific EIN field still needs archive/header verification.
- notes: EIN linkage is the central Gate 2 risk. IRS/FAC/ProPublica are EIN-friendly. USAspending evidence remains UEI/name-first unless another verified surface exposes EIN.

## EVR status

### Execute

Verified live surfaces:

- IRS Form 990 series downloads page.
- IRS annual CSV index files for 2024 and 2025.
- IRS monthly/batch XML ZIP HEAD for `2024_TEOS_XML_01A.zip`.
- IRS TEOS schema page.
- ProPublica Nonprofit Explorer API documentation.
- ProPublica organization endpoint for EIN `742089103`.
- ProPublica Data Terms page.
- FAC download landing page.
- FAC current data page.
- FAC current dictionary page and FAC API dictionary page.
- FAC current full CSV exports: `general.csv`, `federal_awards.csv`, `additional_eins.csv`, `additional_ueis.csv`, `findings.csv`.
- FAC API endpoints for `general`, `federal_awards`, and `findings`, which returned 403 without API key.
- USAspending worked-example award search by recipient text for Atascosa Health Center.

### Verify

Verified:

- IRS index CSV headers include `EIN`, `TAX_PERIOD`, `TAXPAYER_NAME`, `RETURN_TYPE`, `OBJECT_ID`, and `XML_BATCH_ID`.
- IRS 2024 index file returned HTTP 200 and was 91,056,866 bytes in the local probe.
- IRS 2024 XML batch ZIP `2024_TEOS_XML_01A.zip` returned HTTP 200, `content-type: application/zip`, `content-length: 104,816,571`.
- ProPublica endpoint is live for a real EIN and returns organization metadata plus filing list.
- FAC full CSV links are live and redirect to short-lived signed S3 URLs.
- FAC `general.csv` header includes both `auditee_uei` and `auditee_ein`.
- FAC `federal_awards.csv` header includes `report_id`, `auditee_uei`, `award_reference`, program fields, and expenditure fields.
- FAC `additional_eins.csv` and `additional_ueis.csv` exist for multi-identifier reports.
- FAC `findings.csv` exists and carries audit finding references by report/award.
- FAC API without API key returns `API_KEY_MISSING`, so API access is not anonymous, while CSV bulk downloads are anonymous/live.

Not yet verified:

- A public, official UEI-to-EIN crosswalk usable at scale.
- A USAspending assistance archive header containing EIN.
- ProPublica general API rate limit beyond the documented note that PDF download links are rate-limited.
- IRS/FAC explicit redistribution/commercial-use license text.

### Report

This file is the Gate 2 report artifact.

## IRS Form 990 e-file bulk data

Official page:

```text
https://www.irs.gov/charities-non-profits/form-990-series-downloads
```

The IRS page states that users may download the most recent 990 series filings in XML formats. It exposes annual CSV index files and XML ZIP batches organized by year/month.

Verified index:

```text
https://apps.irs.gov/pub/epostcard/990/xml/2024/index_2024.csv
```

Observed HTTP/file facts:

- HTTP status: 200.
- Local downloaded bytes: 91,056,866.
- Local raw SHA-256: `c00051a33f65d408ea7f6d5fa008c7f82f9bc6223751f4a6e29d59f5e17f826d`.
- Header:
  - `RETURN_ID`
  - `FILING_TYPE`
  - `EIN`
  - `TAX_PERIOD`
  - `SUB_DATE`
  - `TAXPAYER_NAME`
  - `RETURN_TYPE`
  - `DLN`
  - `OBJECT_ID`
  - `XML_BATCH_ID`

Verified XML ZIP batch:

```text
https://apps.irs.gov/pub/epostcard/990/xml/2024/2024_TEOS_XML_01A.zip
```

Observed HEAD facts:

- HTTP status: 200.
- `content-type: application/zip`.
- `content-length: 104816571`.
- `last-modified: Thu, 06 Mar 2025 14:04:31 GMT`.

Negative/odd finding:

- Direct URL `https://apps.irs.gov/pub/epostcard/990/xml/2024/202410229349201231_public.xml` redirected to IRS 404. Use the index plus ZIP batch path, not an assumed direct-object URL pattern, unless direct object paths are separately verified.

Mapping implication:

- IRS Form 990 is EIN-first. The `EIN` field is the canonical join candidate for tax-exempt organization filings.

## ProPublica Nonprofit Explorer API

Documentation:

```text
https://projects.propublica.org/nonprofits/api
```

Observed API properties from docs and live probe:

- Base URL: `https://projects.propublica.org/nonprofits/api/v2`.
- REST-style API.
- GET only.
- JSON/JSONP responses.
- Organization endpoint returns data for an integer EIN:

```text
GET https://projects.propublica.org/nonprofits/api/v2/organizations/:ein.json
```

Worked endpoint:

```text
https://projects.propublica.org/nonprofits/api/v2/organizations/742089103.json
```

Observed facts:

- HTTP status: 200.
- Local response bytes: 21,366.
- Local raw SHA-256: `efbff3f8f82806d051458170c0febf7bbc3b55cdb8eb17522d62696ba50f91a2`.
- Organization name: `Atascosa Health Center Inc`.
- EIN: `742089103`.
- Address/city/state/ZIP: `310 W OAKLAWN RD`, Pleasanton, TX, `78064-4033`.
- API metadata included `data_source: current_2026_04_15` and `latest_object_id: 202502329349301435`.

Terms/rate limits:

- ProPublica API docs link to ProPublica Data Terms of Use.
- Extracted term text: “In general, you may use the free data published by ProPublica under the following terms. However, there may be different terms included for some datasets. It is your responsibility to read carefully any specific terms included with the data you download from our website.”
- API docs state PDF download links are rate-limited.
- General JSON API rate limit was not found/confirmed in this probe.

Mapping implication:

- ProPublica is useful as an EIN-keyed convenience/index layer, but its data terms must be respected separately from IRS public-source availability.

## Federal Audit Clearinghouse bulk data

Landing page:

```text
https://www.fac.gov/data/download
```

Current data page:

```text
https://www.fac.gov/data/download/current/
```

Current dictionary/API dictionary:

```text
https://www.fac.gov/data/download/current-dictionary/
https://www.fac.gov/api/dictionary
```

FAC says current data covers 2016-present and is available as CSVs. The current data page exposes full CSV exports and year/fiscal-year slices. The page says the current files mirror data available through FAC search.

Verified full CSV links:

```text
https://app.fac.gov/dissemination/public-data/gsa/full/general.csv
https://app.fac.gov/dissemination/public-data/gsa/full/federal_awards.csv
https://app.fac.gov/dissemination/public-data/gsa/full/notes_to_sefa.csv
https://app.fac.gov/dissemination/public-data/gsa/full/findings.csv
https://app.fac.gov/dissemination/public-data/gsa/full/findings_text.csv
https://app.fac.gov/dissemination/public-data/gsa/full/corrective_action_plans.csv
https://app.fac.gov/dissemination/public-data/gsa/full/passthrough.csv
https://app.fac.gov/dissemination/public-data/gsa/full/secondary_auditors.csv
https://app.fac.gov/dissemination/public-data/gsa/full/additional_ueis.csv
https://app.fac.gov/dissemination/public-data/gsa/full/additional_eins.csv
```

Observed behavior:

- FAC CSV URLs return HTTP 302 to short-lived signed S3 URLs.
- Range/sample reads returned CSV headers and first rows.
- Dictionary XLSX HEAD returned HTTP 200, `content-type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `content-length: 76004`.

### FAC key fields

`general.csv` observed header includes:

- `report_id`
- `auditee_uei`
- `audit_year`
- `auditee_name`
- `auditee_address_line_1`
- `auditee_city`
- `auditee_state`
- `auditee_ein`
- `auditee_zip`
- `fac_accepted_date`
- `fy_start_date`
- `fy_end_date`
- `audit_type`
- `total_amount_expended`
- `entity_type`
- `is_additional_ueis`
- `is_multiple_eins`

`federal_awards.csv` observed header includes:

- `report_id`
- `auditee_uei`
- `audit_year`
- `fac_accepted_date`
- `award_reference`
- `federal_agency_prefix`
- `federal_award_extension`
- `additional_award_identification`
- `federal_program_name`
- `amount_expended`
- `cluster_name`
- `federal_program_total`
- `is_major`
- `is_direct`
- `findings_count`
- `is_passthrough_award`
- `passthrough_amount`

`additional_eins.csv` observed header:

- `report_id`
- `auditee_uei`
- `audit_year`
- `fac_accepted_date`
- `additional_ein`

`additional_ueis.csv` observed header:

- `report_id`
- `auditee_uei`
- `audit_year`
- `fac_accepted_date`
- `additional_uei`

`findings.csv` observed header includes:

- `report_id`
- `auditee_uei`
- `audit_year`
- `award_reference`
- `reference_number`
- finding flags and requirement type fields.

### FAC API status

Unauthenticated probes:

```text
https://api.fac.gov/general?limit=1
https://api.fac.gov/federal_awards?limit=1
https://api.fac.gov/findings?limit=1
```

Observed response:

```json
{"error":{"code":"API_KEY_MISSING","message":"No api_key was supplied. Get one at https://api.fac.gov:443"}}
```

Finding:

- FAC API is not anonymous. Bulk CSV downloads are the safer default ingestion surface until an API key is configured.

FAC result limit guidance:

- FAC docs say there are more than 200,000 audits and more than 2.5M `federal_awards` records dating back to 2016.
- FAC API limits requests to 20,000 results at a time.
- FAC recommends focused queries and pagination rather than broad joins.

## Join-key map

| Source | Primary identifiers verified | Secondary identifiers / alias evidence | Money/award grain | EIN linkage verdict |
|---|---|---|---|---|
| IRS Form 990 XML index | `EIN`, `OBJECT_ID`, `DLN` | `TAXPAYER_NAME`, `TAX_PERIOD`, `RETURN_TYPE`, `XML_BATCH_ID` | Organization tax filing | Strong EIN source. No UEI observed. |
| ProPublica Nonprofit Explorer | EIN endpoint key; organization `ein` | name, address, city, state, ZIP, filing metadata | Organization profile and filing summaries | Strong EIN convenience layer. Terms apply. |
| FAC `general.csv` | `auditee_ein`, `auditee_uei`, `report_id` | auditee name/address, fiscal period, entity type | Single-audit report aggregate | Best verified EIN/UEI bridge in Gate 2, but only for audited entities/report years. |
| FAC `additional_eins.csv` | `report_id`, `auditee_uei`, `additional_ein` | audit year, accepted date | Multi-EIN report metadata | Supports multi-EIN reports; do not collapse blindly. |
| FAC `additional_ueis.csv` | `report_id`, `auditee_uei`, `additional_uei` | audit year, accepted date | Multi-UEI report metadata | Supports multi-UEI reports; do not collapse blindly. |
| FAC `federal_awards.csv` | `report_id`, `auditee_uei`, `award_reference` | federal agency prefix, program name, award identification | Audited federal award/program expenditure | Links audited awards to `general.csv` via `report_id`/UEI, then to EIN via `general.csv`. |
| USAspending assistance search | recipient UEI, recipient name, award ID | agency, award dates, description | Award | Worked example returned UEI/name, not EIN. Requires FAC or another identifier bridge for EIN. |

## Worked example: one real grant to one organization's 990

Entity:

- Organization: Atascosa Health Center Inc / Atascosa Health Center, Inc.
- EIN: `742089103`.
- UEI: `MW4NM5KU2M81`.

### Step 1: FAC bridges EIN and UEI

FAC `general.csv` sample row showed:

- `report_id`: `2023-01-GSAFAC-0000000854`.
- `auditee_uei`: `MW4NM5KU2M81`.
- `auditee_name`: `Atascosa Health Center, Inc.`.
- `auditee_ein`: `742089103`.
- `entity_type`: `non-profit`.

FAC `federal_awards.csv` sample for the same `report_id`/UEI showed:

- `additional_award_identification`: `5 H80CS00405-22-00`.
- `federal_agency_prefix`: `93`.
- `federal_program_name`: consolidated health centers / health center program cluster text.
- `is_direct`: `Y`.

Interpretation:

- FAC supplies the strongest verified Gate 2 bridge: `UEI MW4NM5KU2M81` ↔ `EIN 742089103` for this audited nonprofit and report year.
- This is not a universal UEI/EIN crosswalk. It exists when an entity is in FAC and reports both identifiers.

### Step 2: IRS verifies the EIN has a 990 filing index record

IRS 2024 index scan for EIN `742089103` returned:

- `EIN`: `742089103`.
- `TAX_PERIOD`: `202401`.
- `TAXPAYER_NAME`: `Atascosa Health Centers`.
- `RETURN_TYPE`: `990`.
- `OBJECT_ID`: `202412579349301506`.
- `XML_BATCH_ID`: `2024_TEOS_XML_09A`.

Interpretation:

- The same EIN appears in the official IRS 2024 Form 990 e-file index.
- This is a direct EIN-level trace from FAC auditee to IRS filing index.

### Step 3: ProPublica verifies the same EIN through its API

ProPublica endpoint:

```text
https://projects.propublica.org/nonprofits/api/v2/organizations/742089103.json
```

Observed:

- `ein`: `742089103`.
- `name`: `Atascosa Health Center Inc`.
- `state`: `TX`.
- `latest_object_id`: `202502329349301435`.
- `data_source`: `current_2026_04_15`.

Local raw SHA-256 for the retrieved API response:

```text
efbff3f8f82806d051458170c0febf7bbc3b55cdb8eb17522d62696ba50f91a2
```

### Step 4: USAspending verifies a matching UEI/name award surface

USAspending query:

```text
POST https://api.usaspending.gov/api/v2/search/spending_by_award/
filters.recipient_search_text = ["Atascosa Health Center"]
filters.time_period = 2022-02-01 through 2024-09-30
filters.award_type_codes = ["02", "03", "04", "05"]
```

Local raw SHA-256 for the retrieved USAspending response:

```text
8e64d601207754e6c1ff7ac8f404a725d59c03761c2a6503626c6089b1c497cc
```

Observed first result:

- Award ID: `H8000405`.
- Recipient name: `ATASCOSA HEALTH CENTER, INC.`.
- Recipient UEI: `MW4NM5KU2M81`.
- Awarding/funding agency: Department of Health and Human Services.
- Description: `HEALTH CENTER CLUSTER`.

Interpretation:

- This is a hand trace, not a certified source-native join across all systems.
- The chain is:
  1. USAspending award surface identifies recipient UEI/name.
  2. FAC identifies the same UEI and an EIN for the audited nonprofit.
  3. IRS and ProPublica identify the nonprofit's 990 by EIN.
- The official bridge in this example is FAC, not USAspending itself.

## Gate 2 verdict

Nonprofit stack is viable, but the join is not solved by one magic file.

- IRS and ProPublica are EIN-first.
- USAspending award surfaces are UEI/name-first in the verified worked example.
- FAC current bulk data is the best verified bridge found in Gate 2 because `general.csv` includes both `auditee_uei` and `auditee_ein`, and FAC award/finding tables join by `report_id` and UEI.
- FAC only covers entities that file single audits. It cannot be treated as a universal UEI/EIN crosswalk.
- Outlays should model UEI/EIN/name/address links as append-only `entity_alias` evidence with source and confidence, exactly as `ARCHITECTURE.md` requires.

## Required carry-forward to Gate 3

Gate 3 must answer whether SAM.gov or another official source exposes a public, usable UEI-to-EIN linkage.

If no official UEI/EIN linkage exists:

- Use exact EIN joins for IRS/ProPublica/FAC surfaces.
- Use exact UEI joins for USAspending/SAM/FAC surfaces.
- Promote FAC UEI↔EIN pairs only as source-scoped alias evidence.
- Use name/address matching only as confidence-scored aliases, never destructive merges.
- Publish coverage gaps honestly: nonprofit grant-to-990 traceability is strongest for audited entities with FAC records and weakest for recipients lacking public EIN evidence.
