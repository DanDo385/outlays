# Source: US:fed — USAspending bulk archives
- jurisdiction: US:fed
- base_url: https://api.usaspending.gov/ and https://files.usaspending.gov/
- access_method: api | bulk-download
- working_query_or_file: FY2024_All_Contracts_Full_20260506.zip; FY2024_All_Assistance_Full_20260506.zip; POST /api/v2/bulk_download/list_agencies/; POST /api/v2/bulk_download/list_monthly_files/; GET /api/v2/bulk_download/list_database_download_files/; POST /api/v2/bulk_download/awards/; POST /api/v2/download/accounts/; POST /api/v2/download/count/
- confirmed_at: 2026-06-10T16:53:44Z
- response_status: live
- formats_and_sizes: ZIP archives containing CSV files; generated account/subaward ZIP downloads; database snapshot ZIPs. Verified sizes listed below.
- fields_we_map: award/transaction identifiers -> fiscal_fact.id/source keys; recipient UEI/parent UEI -> entity identifiers; recipient DUNS/parent DUNS -> legacy entity identifiers; transaction/obligation/outlay amounts -> fiscal_fact.amount candidates; action/fiscal dates -> fiscal_fact.period/date; awarding/funding agency fields -> assignment properties; object class and program activity fields -> classification assignments; treasury account fields -> account/control linkage.
- identifiers_present: UEI, parent UEI, legacy DUNS, parent DUNS. EIN not observed in the inspected FY2024 contract archive header.
- grain: transaction | award | aggregate
- update_cadence: Monthly archive index observed for award data; generated downloads are request/job based; full database snapshots are published as dated ZIPs. Exact official cadence text not yet confirmed.
- posting_lag: Not yet confirmed from official terms/docs in this run.
- license_or_terms: Explicit license/terms not yet confirmed. Attempts to retrieve obvious USAspending terms/about surfaces did not expose usable license text in fetched HTML. Treat as an open Gate 1 gap and verify before redistribution claims.
- known_gaps: Full FY subaward ZIP size not HEAD-verified; full-year all-agency account download row counts not verified; award_financial and account_balances account downloads not yet live-probed; license/terms not confirmed.
- notes: This note records only Gate 1 USAspending bulk-source findings. Do not begin Gate 2 until this file exists and its open gaps are accepted or assigned.

## EVR status

### Execute
Mostly complete for Gate 1 discovery surfaces.

Verified live surfaces from the prior probe run:
- `POST https://api.usaspending.gov/api/v2/bulk_download/list_agencies/`
- `POST https://api.usaspending.gov/api/v2/bulk_download/list_monthly_files/`
- `GET https://api.usaspending.gov/api/v2/bulk_download/list_database_download_files/`
- `POST https://api.usaspending.gov/api/v2/bulk_download/awards/`
- `POST https://api.usaspending.gov/api/v2/download/accounts/`
- `POST https://api.usaspending.gov/api/v2/download/count/`
- Raw S3-style listing for `https://files.usaspending.gov/award_data_archive/?list-type=2&max-keys=5`

A small FY2024 Access Board contracts archive was downloaded, inspected structurally, and deleted after verification.

### Verify
Partially complete.

Verified:
- Endpoint shapes for agency listing, monthly file listing, database snapshot listing, generated award downloads, generated account downloads, and count API.
- HEAD sizes for FY2024 all-agency prime contract and assistance archives.
- HEAD sizes for current database snapshot ZIPs.
- One small archive ZIP structure and CSV header width.
- Key identifier and classification-like fields in the inspected contract archive header.
- Approximate FY2024 row counts from the count API for contracts, assistance, and subawards.

Not yet verified:
- Explicit license/terms suitable for redistribution/commercial-use conclusions.
- Full FY generated subaward download size.
- Full FY all-agency account download row counts and sizes.
- Live probe for `award_financial` account download.
- Live probe for `account_balances` account download.

### Report
This file is the Gate 1 report artifact.

## Verified archive index and file-host behavior

### Award data archive listing

Raw S3-style listing works for the award archive host:

```text
https://files.usaspending.gov/award_data_archive/?list-type=2&max-keys=5
```

Observed response shape:
- XML `ListBucketResult`.

The API monthly-file listing endpoint also returned live all-agency FY2024 archive metadata:

```text
POST https://api.usaspending.gov/api/v2/bulk_download/list_monthly_files/
```

Observed all-agency FY2024 archive names included:
- `FY2024_All_Contracts_Full_20260506.zip`
- `FY2024_All_Assistance_Full_20260506.zip`
- `FY(All)_All_Contracts_Delta_20260506.zip`
- `FY(All)_All_Assistance_Delta_20260506.zip`

### Database download listing

Raw file-host listing did not work for the database-download prefix:

```text
https://files.usaspending.gov/database_download/?list-type=2&max-keys=20
```

Observed response:
- 404 HTML site page.

The API database-download index did work:

```text
GET https://api.usaspending.gov/api/v2/bulk_download/list_database_download_files/
```

Finding: discover database snapshots through the API endpoint, not by assuming raw bucket listing works for every prefix.

## Verified formats and sizes

### FY2024 all-agency prime award archives

| File | URL | Verified size bytes | Approx decimal size | Notes |
|---|---:|---:|---:|---|
| `FY2024_All_Contracts_Full_20260506.zip` | `https://files.usaspending.gov/award_data_archive/FY2024_All_Contracts_Full_20260506.zip` | 1,946,043,103 | 1.95 GB | HEAD verified |
| `FY2024_All_Assistance_Full_20260506.zip` | `https://files.usaspending.gov/award_data_archive/FY2024_All_Assistance_Full_20260506.zip` | 1,343,686,418 | 1.34 GB | HEAD verified |
| `FY(All)_All_Contracts_Delta_20260506.zip` | award archive host | 182,017,458 | 182 MB | HEAD verified |
| `FY(All)_All_Assistance_Delta_20260506.zip` | award archive host | 235,506,294 | 236 MB | HEAD verified |

### Database snapshots

| File | Verified size bytes | Approx decimal size | Notes |
|---|---:|---:|---|
| `usaspending-db_20260506.zip` | 172,915,875,021 | 172.9 GB | Listed by database-download API; exceeds 500 MB verification cap; not downloaded |
| `usaspending-db-subset_20260506.zip` | 4,949,209,036 | 4.95 GB | Listed by database-download API; exceeds 500 MB verification cap; not downloaded |

Finding: USAspending publishes full database snapshot ZIPs, but they are too large for Gate 1 sample-download verification under the current cap.

### Small archive structural probe

Probe file:

```text
FY2024_310_Contracts_Full_20260506.zip
```

Observed:
- Agency: Access Board (`310`)
- HEAD size: 2,540 bytes
- ZIP member: `FY2024_310_Contracts_Full_20260508_1.csv`
- Member compressed size: 2,362 bytes
- Member uncompressed size: 8,682 bytes
- CSV rows: header only, no data rows
- CSV column count: 297
- Archive deleted after structural verification

Interpretation:
- This verifies archive packaging and header structure, not populated-row semantics, because the sample contained no data rows.

## Key fields observed in inspected contract archive header

### Award and transaction identifiers

Observed fields included:
- `contract_transaction_unique_key`
- `contract_award_unique_key`
- `award_id_piid`
- `parent_award_id_piid`

Mapping:
- Use transaction/award unique keys as source-stable identifiers for fiscal facts and source records.
- Use PIID and parent PIID for contract award identity and parent-child award lineage.

### Recipient identifiers

Observed fields included:
- `recipient_uei`
- `recipient_parent_uei`
- `recipient_duns`
- `recipient_parent_duns`

Mapping:
- UEI and parent UEI are primary recipient entity identifiers.
- DUNS and parent DUNS are legacy identifiers useful for historical joins and reconciliation.

Not observed in inspected contract header:
- EIN.

Finding:
- Do not assume the prime award archive itself supplies EIN. Nonprofit joins likely require another source or matching layer in later gates.

### Account/classification-like fields

Observed fields included:
- `treasury_accounts_funding_this_award`
- `object_classes_funding_this_award`
- `program_activities_funding_this_award`

Finding:
- The inspected contract archive did not expose simple scalar `object_class_code` or `program_activity_code` fields.
- It exposed bundled/list-style fields for object classes and program activities funding the award.
- For official account-level control and classification joins, use account downloads, especially object-class/program-activity and award-financial submissions.

## Account download service

Endpoint:

```text
POST https://api.usaspending.gov/api/v2/download/accounts/
```

Narrow probe submitted:
- Agency: Access Board
- FY2024 period 2
- `account_level: treasury_account`
- `submission_types: ["object_class_program_activity"]`

Observed generated job status:
- `status: finished`
- `total_size: 2.418`
- `total_columns: 59`
- `total_rows: 46`

HEAD on generated ZIP:
- `Content-Length: 2418`

Finding:
- Account breakdown by object class/program activity is available through generated account downloads.
- This service is separate from pre-generated monthly award archives.

Open account-download probes needed before ingestion lock:
- `submission_types: ["award_financial"]`
- `submission_types: ["account_balances"]`
- Full FY all-agency object-class/program-activity request, likely using period 12.

## Subaward generation service

Endpoint:

```text
POST https://api.usaspending.gov/api/v2/bulk_download/awards/
```

Narrow probe submitted:
- Agency: Access Board
- FY2024 date range
- Subaward request shape using subaward types

Observed generated job status:
- `status: finished`
- `total_columns: 231`
- `total_rows: 0`
- `total_size: 1.889`

HEAD on generated ZIP:
- `Content-Length: 1889`

Finding:
- Subaward generation endpoint works.
- The Access Board FY2024 probe produced zero rows, so it verifies service shape but not populated-row structure.
- Full FY subaward size remains unverified.

## Approximate FY2024 row counts

Endpoint:

```text
POST https://api.usaspending.gov/api/v2/download/count/
```

Date range:

```text
2023-10-01 through 2024-09-30
```

Observed count API results:

| Dataset | Approx row count | Caveat |
|---|---:|---|
| Prime contract transactions | 6,691,990 | API count estimate, not ZIP-extracted row count |
| Prime assistance transactions | 5,194,768 | API count estimate, not ZIP-extracted row count |
| Subawards | 690,073 | API count estimate, not full generated ZIP row count |

Use these as planning counts, not certified archive row counts.

## Recommended minimal FY2024 file set

This is the smallest sensible Gate 1 ingest set for one full federal fiscal year if the product goal is transaction/award-grain facts plus account/control linkage.

### 1. Prime contract transactions

File:

```text
FY2024_All_Contracts_Full_20260506.zip
```

Verified:
- Size: 1,946,043,103 bytes, about 1.95 GB.
- Approx rows from count API: 6,691,990.

Why included:
- Core procurement transaction/award facts.
- Carries recipient UEI and parent UEI.
- Carries award and transaction identifiers.
- Carries bundled treasury account, object class, and program activity funding fields.

### 2. Prime assistance transactions

File:

```text
FY2024_All_Assistance_Full_20260506.zip
```

Verified:
- Size: 1,343,686,418 bytes, about 1.34 GB.
- Approx rows from count API: 5,194,768.

Why included:
- Core grants/loans/assistance award facts.
- Needed for government-to-recipient money graph beyond procurement.

### 3. Subawards

Generate via:

```text
POST https://api.usaspending.gov/api/v2/bulk_download/awards/
```

Recommended request shape:
- Full FY date range: `2023-10-01` through `2024-09-30`
- Include subaward types through the endpoint-supported `sub_award_types` shape.

Verified:
- Service shape works through a generated Access Board probe.
- Approx rows from count API: 690,073.

Not yet verified:
- Full FY generated ZIP size.
- Populated full-FY generated ZIP structure.

Why included:
- Required to move closer to final-recipient/final-performer visibility.
- Bridges prime award visibility toward downstream distribution.

### 4. Account breakdown by program activity/object class

Generate via:

```text
POST https://api.usaspending.gov/api/v2/download/accounts/
```

Recommended `submission_types`:

```json
["object_class_program_activity"]
```

Likely full FY setting:
- Period 12 for FY2024. Confirm exact request semantics before full run.

Verified:
- Narrow Access Board P01-P02 probe returned 46 rows and 59 columns.

Not yet verified:
- Full-year all-agency row count and ZIP size.

Why included:
- Required for official object class and program activity classification at account grain.
- Important for category trees and cross-cut views.

### 5. Account award-financial / File C style linkage

Generate via:

```text
POST https://api.usaspending.gov/api/v2/download/accounts/
```

Recommended `submission_types`:

```json
["award_financial"]
```

Status:
- Not yet live-probed in this run.

Why included:
- Needed for linking awards to accounts/outlays.
- Important bridge between award facts and federal account reporting.

### 6. Account balances / File A control totals

Generate via:

```text
POST https://api.usaspending.gov/api/v2/download/accounts/
```

Recommended `submission_types`:

```json
["account_balances"]
```

Status:
- Not yet live-probed in this run.

Why included:
- Needed for official account-level control totals.
- Supports coverage honesty: what share of official totals is traced at transaction/award grain.

## Data dictionary status

Official public source/docs located during the prior probe:
- `fedspendingtransparency/usaspending-api` documentation for bulk download endpoints.
- `list_monthly_files.md` for monthly archive listing response shape.
- `list_agencies.md` for agency-list request/response shape.
- `accounts.md` for account download request shape.
- `awards.md` for bulk award/subaward generation request shape.
- Source path observed for monthly contract generation: `usaspending_api/download/delta_downloads/transaction_contract_monthly.py`

Verified:
- The inspected live contract archive header had 297 columns.
- A clean AST parse of the pinned USAspending generation source found 297 output columns from `ContractMixin.select_cols`.
- Exact ordered column parity is confirmed between the pinned source parser output and the live FY2024 Access Board header-only archive: 297 expected columns, 297 live header columns, 0 missing, 0 extra, 0 order mismatches.
- Key fields listed above were observed in the live header.

Parity evidence:
- Live archive: `https://files.usaspending.gov/award_data_archive/FY2024_310_Contracts_Full_20260506.zip`
- HEAD/GET verified size: 2,540 bytes
- ZIP SHA-256: `666f06abf07c16e7e4205e5c217198ffc4b3b205fe91bba05eed1873ceb580b0`
- ZIP member: `FY2024_310_Contracts_Full_20260508_1.csv`
- Live header count: 297 columns
- Pinned source repository: `fedspendingtransparency/usaspending-api`
- Pinned source ref: `2a0cb61881a2a38e304864383cd5f0e20c3cd30a`
- Pinned source commit date: 2026-04-17T00:15:56Z
- Ref selection rationale: nearest commit touching `usaspending_api/download/delta_downloads/transaction_contract_monthly.py` before the 2026-05-06 archive date.
- Pinned source path: `usaspending_api/download/delta_downloads/transaction_contract_monthly.py`
- Parsed class/property: `ContractMixin.select_cols`
- Parser method: Python AST, using alias string from `.alias("...")` when present and source string from `self.sf.col("...")` otherwise.
- Source text SHA-256 from the pinned raw file: `51c9edbcbb4c2448bc9892f3079c5eccf83881b757764abf87caa23ab7bbcb6a`
- COVID/IIJA alias fields matched exactly, including multiline source-code alias cases:
  - `outlayed_amount_from_COVID-19_supplementals_for_overall_award`
  - `obligated_amount_from_COVID-19_supplementals_for_overall_award`
  - `outlayed_amount_from_IIJA_supplemental_for_overall_award`
  - `obligated_amount_from_IIJA_supplemental_for_overall_award`

Conclusion:
- The earlier regex mismatch was a parser artifact, not a live dictionary mismatch. For FY2024 contract monthly archives, ordered column parity between the pinned USAspending generation code and the inspected live archive header is confirmed.

## Broken, odd, or negative findings

- `agency: 50` for FY2024 monthly files returned empty for both assistance and contracts. Record as a live finding, not an error to hide.
- Raw `database_download` bucket listing returned 404 HTML, while API database-download index returned live JSON.
- Subaward count request using `award_type_codes: ["grant", "procurement"]` returned 400 because those are not valid values for that count endpoint. A retry with default award-type handling and `spending_level: "subawards"` succeeded.
- Terms/license verification is incomplete. Do not state that public data is unrestricted or commercially reusable until Gate 8 or a dedicated license pass verifies terms.

## Builder implications

1. Bulk-first ingestion should start with prime contracts and prime assistance archives.
2. Treat award archive object class/program activity values as bundled/list fields, not clean scalar classification assignments.
3. Use account downloads for official account/control/classification linkage.
4. Do not depend on EIN being present in prime award archives. Later nonprofit/entity gates must solve UEI/EIN or name/address matching explicitly.
5. Database snapshots are available but too large for a minimal first ingest. They may become useful after the narrow FY2024 file path is proven.

## Gate 1 closeout

Gate 1 is report-complete with explicit open verification gaps.

Do not start Gate 2 until the operator accepts this Gate 1 note or assigns the open verification gaps:
- license/terms confirmation,
- full FY subaward ZIP HEAD/status,
- full FY all-agency account download counts/sizes,
- `award_financial` probe,
- `account_balances` probe.
