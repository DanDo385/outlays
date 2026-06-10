# Source: US:NY — second-state portal mechanics
- jurisdiction: US:NY
- base_url: https://data.ny.gov; https://data.ny.gov/Transportation/MTA-Procurements-Beginning-2018/twsw-2mqa
- access_method: Socrata API | CSV export | server-side SoQL aggregation
- working_query_or_file: `https://data.ny.gov/resource/twsw-2mqa.json?$limit=1`; `https://data.ny.gov/resource/twsw-2mqa.csv?$limit=5`
- confirmed_at: 2026-06-10
- response_status: live
- formats_and_sizes: JSON sample 864 bytes; metadata JSON 59,688 bytes; CSV five-row sample 2,414 bytes; full CSV export available via Socrata endpoint.
- fields_we_map: `amount_expended_for_fiscal_year` -> fiscal_fact.amount; `fiscal_year_end_date` -> fiscal period; `vendor_name` -> provisional payee alias; `type_of_procurement`/`award_process` -> source classifications; `procurement_description` -> source description; `transaction_number` -> source row/procurement identifier.
- identifiers_present: vendor name; transaction number; address/city/state/postal code. No EIN/UEI verified.
- grain: transaction/contract procurement line as reported by MTA/PARIS.
- update_cadence: metadata `Last-Modified: Wed, 01 Apr 2026 14:09:46 GMT`; exact official schedule not verified.
- posting_lag: not verified.
- license_or_terms: data.ny.gov/Socrata public dataset access observed; explicit dataset license not confirmed in this gate.
- known_gaps: Dataset is MTA/public-authority procurement, not full statewide AP across all New York agencies. Good second-state mechanics test, weak statewide coverage test.
- notes: Chosen because it gives a live Socrata transaction dataset with server-side aggregation, CSV export, metadata, and no key required for sampled access.

## EVR status

### Execute

Verified:

- A live New York Socrata procurement dataset.
- Row sample endpoint.
- Metadata endpoint.
- CSV export endpoint.
- Server-side aggregation by fiscal year.
- Server-side aggregation by vendor.

Blocked:

- A Socrata catalog-search terminal command was blocked by operator/tool approval. It is not retried. This gate proceeds from the concrete dataset candidate already found by web search.

Blocked command class:

```text
Socrata catalog search over api.us.socrata.com/api/catalog/v1 with domains=data.ny.gov and spending/procurement query terms.
```

### Verify

Verified endpoints:

```text
https://data.ny.gov/resource/twsw-2mqa.json?$limit=1
https://data.ny.gov/api/views/twsw-2mqa
https://data.ny.gov/resource/twsw-2mqa.csv?$limit=5
```

Verified aggregation queries:

```text
https://data.ny.gov/resource/twsw-2mqa.json?$select=fiscal_year_end_date,count(*),sum(amount_expended_for_fiscal_year)&$group=fiscal_year_end_date&$order=fiscal_year_end_date&$limit=10
```

Encoded SoQL vendor aggregation:

```text
base: https://data.ny.gov/resource/twsw-2mqa.json
$select=vendor_name,count(*),sum(amount_expended_for_fiscal_year)
$group=vendor_name
$order=sum(amount_expended_for_fiscal_year) DESC
$limit=5
```

Metadata verified:

- id: `twsw-2mqa`.
- name: `MTA Procurements: Beginning 2018`.
- asset type: dataset.
- attribution: Metropolitan Transportation Authority.
- category: Transportation.
- provenance: official.
- description: annual procurement contracts data reported by MTA to the Authorities Budget Office via PARIS.
- row metadata updated: `rowsUpdatedAt` present.
- HTTP last modified: `Wed, 01 Apr 2026 14:09:46 GMT`.

Observed headers:

```text
Content-Type: application/json;charset=utf-8
Content-Type: text/csv; charset=UTF-8
X-SODA2-Fields: [field list]
X-SODA2-Types: [type list]
X-SODA2-Data-Out-Of-Date: false
X-SODA2-Truth-Last-Modified: Wed, 01 Apr 2026 14:09:46 GMT
```

Not verified:

- Exact rate limit for anonymous access.
- Dataset-specific license.
- Full statewide New York agency coverage.

### Report

This file is the Gate 7 report.

## Dataset

Dataset page:

```text
https://data.ny.gov/Transportation/MTA-Procurements-Beginning-2018/twsw-2mqa
```

API resource:

```text
https://data.ny.gov/resource/twsw-2mqa.json
```

CSV resource:

```text
https://data.ny.gov/resource/twsw-2mqa.csv
```

Metadata resource:

```text
https://data.ny.gov/api/views/twsw-2mqa
```

## Sample row fields

Observed first-row keys:

```text
fiscal_year_end_date
vendor_name
transaction_number
procurement_description
status
type_of_procurement
award_process
award_date
begin_date
end_date
contract_amount
amount_expended_for_fiscal_year
amount_expended_to_date
current_or_outstanding_balance
number_of_bids_or_proposals_received
vendor_is_nys_or_fbe
vendor_is_a_mwbe
solicited_mwbe
number_of_mwbe_proposals
exempt_from_article_4c
address_line_1
city
state
postal_code
zip_code_plus_4
country
```

Example first row:

```json
{
  "fiscal_year_end_date": "2018-12-31T00:00:00.000",
  "vendor_name": "CEMBRE, INC.",
  "transaction_number": "6.00E+14",
  "procurement_description": "MOCK EVENT FOR 42-02-1723",
  "status": "Open",
  "type_of_procurement": "Commodities/Supplies",
  "award_process": "Authority Contract - Competitive Bid",
  "contract_amount": "197120.00",
  "amount_expended_for_fiscal_year": "98560.00",
  "amount_expended_to_date": "98560.00",
  "current_or_outstanding_balance": "98560.00",
  "vendor_is_nys_or_fbe": "Foreign",
  "vendor_is_a_mwbe": "N",
  "city": "EDISON",
  "state": "NJ",
  "postal_code": "8837",
  "country": "USA"
}
```

## Server-side aggregation proof

By fiscal year end date:

```json
[
  {"fiscal_year_end_date":"2018-12-31T00:00:00.000","count":"14845","sum_amount_expended_for_fiscal_year":"4754126089.52"},
  {"fiscal_year_end_date":"2019-12-31T00:00:00.000","count":"15033","sum_amount_expended_for_fiscal_year":"5137370142.05"},
  {"fiscal_year_end_date":"2020-12-31T00:00:00.000","count":"13362","sum_amount_expended_for_fiscal_year":"6649302214.34"},
  {"fiscal_year_end_date":"2021-12-31T00:00:00.000","count":"12799","sum_amount_expended_for_fiscal_year":"6413952790.63"},
  {"fiscal_year_end_date":"2022-12-31T00:00:00.000","count":"12156","sum_amount_expended_for_fiscal_year":"6837685984.95"},
  {"fiscal_year_end_date":"2023-12-31T00:00:00.000","count":"12114","sum_amount_expended_for_fiscal_year":"6681146956.30"},
  {"fiscal_year_end_date":"2024-12-31T00:00:00.000","count":"13648","sum_amount_expended_for_fiscal_year":"8148961067.82"},
  {"fiscal_year_end_date":"2025-12-31T00:00:00.000","count":"13546","sum_amount_expended_for_fiscal_year":"9103089239.53"}
]
```

By vendor, top five from server-side SoQL aggregation:

```json
[
  {"vendor_name":"KAWASAKI RAIL CAR INC","count":"1749","sum_amount_expended_for_fiscal_year":"2938261044.86"},
  {"vendor_name":"3RD TRACK CONSTRUCTORS","count":"7","sum_amount_expended_for_fiscal_year":"1587086671.85"},
  {"vendor_name":"SPRAGUE OPERATING RESOURCES, LLC","count":"139","sum_amount_expended_for_fiscal_year":"1008018579.03"},
  {"vendor_name":"JUDLAU CONTRACTING, INC.","count":"61","sum_amount_expended_for_fiscal_year":"989858020.48"},
  {"vendor_name":"NEW FLYER OF AMERICA INC","count":"106","sum_amount_expended_for_fiscal_year":"905385064.89"}
]
```

## Rate-limit and token findings

Observed access:

- No app token required for small sampled reads.
- No app token required for metadata reads.
- No app token required for tested SoQL aggregation.
- No app token required for five-row CSV export.

Not confirmed:

- Anonymous rate limit.
- Full export throttling behavior.
- Whether production ingestion should register a Socrata app token.

Recommendation:

Use unauthenticated access for fixtures and development. Register/use a Socrata app token before production crawling or repeated full exports.

## Adapter mechanics

Recommended package name:

```text
packages/adapters/us-ny-mta-procurement
```

Recommended first extractor:

```text
GET https://data.ny.gov/resource/twsw-2mqa.json
params:
  $limit: page size
  $offset: page offset
  $order: fiscal_year_end_date,transaction_number
```

Recommended fact mapping:

```text
source: data.ny.gov/twsw-2mqa
jurisdiction: US:NY
flow: spending
amount: amount_expended_for_fiscal_year
amount_basis: fiscal-year-expended
period: fiscal_year_end_date year
payee alias: vendor_name
source id: transaction_number + fiscal_year_end_date + row hash
classification assignments:
  - type_of_procurement
  - award_process
  - status
  - vendor_is_nys_or_fbe
  - vendor_is_a_mwbe
```

## Should this be the second state?

Answer: yes for mechanics, no for complete-state coverage.

Good:

- Live Socrata API.
- Transaction/contract-level rows.
- Monetary fields as strings.
- Vendor and procurement metadata.
- Server-side aggregation verified.
- CSV export verified.
- No key required for small probes.

Bad:

- MTA is a major public authority, not all New York state agency spending.
- It does not solve statewide budget/AP coverage.
- Entity identifiers are name/address only, not UEI/EIN.

Recommendation:

Use this as the second-state adapter prototype if the goal is portal mechanics, pagination, SoQL aggregation, and a non-California transaction source. Do not represent it as comprehensive New York state spending.

## Gate 7 closeout

Gate 7 status: closed.

- Candidate: New York Open Data, MTA Procurements Beginning 2018.
- Access: Socrata JSON/CSV and metadata.
- Server-side aggregation: verified.
- Token: not needed for tested calls; production token recommended.
- Caveat: public-authority procurement, not full statewide spending.
