# Source: US:fed — healthcare cross-cut feasibility
- jurisdiction: US:fed
- base_url: https://download.cms.gov/nppes/NPI_Files.html; https://npiregistry.cms.hhs.gov/help/help; https://data.cms.gov/provider-data; https://open.gsa.gov/api/assistance-listings-api; https://www.acquisition.gov/sites/default/files/manual/PSC%20Manual%20April%202025.pdf; https://www.census.gov/data/tables/2022/econ/economic-census/naics-sector-62.html
- access_method: bulk-download | api | public-web | source-classification
- working_query_or_file: NPPES monthly full ZIP; CMS Provider Data Catalog metastore records `mj5m-pzi6` and `4pq5-n9py`; SAM Assistance Listings Public API docs; PSC Manual April 2025; Census NAICS Sector 62 page.
- confirmed_at: 2026-06-10
- response_status: live for NPPES, CMS provider metadata, SAM Assistance Listings docs, PSC manual, and NAICS 62 page. Some CMS provider landing pages failed extraction but metastore JSON endpoints returned 200.
- formats_and_sizes: NPPES ZIP full monthly file, weekly increments, deactivation ZIP; CMS Provider Data JSON metadata and downloadable CSV distributions; SAM Assistance Listings JSON API with API key; PSC PDF manual; Census NAICS public tables/FTP files.
- fields_we_map: NPI, provider organization/name, other names, practice locations, endpoints, CCN/CMS Certification Number where dataset-specific, NAICS, PSC, assistance listing IDs, program names, agency codes, source-scoped entity aliases.
- identifiers_present: NPI is a healthcare-provider identifier; CCN identifies CMS-certified provider facilities in applicable datasets; NAICS 62 classifies healthcare/social-assistance industry; PSC Category Q classifies medical services procurement. None of these is a public UEI↔EIN bridge by itself.
- grain: provider | facility | award | assistance-program | procurement-classification | industry-classification | cross-cut analytical view
- update_cadence: NPPES monthly full replacement plus weekly increments; CMS provider datasets expose dataset-specific `nextUpdateDate`; SAM Assistance Listings API cadence not verified; PSC manual dated April 2025; NAICS 2022 sector page static/statistical.
- posting_lag: NPPES May 2026 monthly full ZIP last-modified May 11, 2026; CMS provider metadata examples had next updates in June 2026.
- license_or_terms: NPPES API/help says public API output is provided under the NPPES Data Dissemination Notice; exact redistribution terms not fully extracted in this gate. CMS Provider Data access level returned `public`; exact license terms not fully extracted. PSC/NAICS are official government classification surfaces; reuse terms should still be recorded before production redistribution.
- known_gaps: No verified NPI↔UEI/EIN crosswalk. No verified CCN↔UEI/EIN crosswalk. SAM Assistance Listings requires an API key. Match rates between provider/facility data and federal spending recipients are unknown. NPI issuance does not validate licensure or credentialing.
- notes: Healthcare cross-cut is feasible as an analytical classification view before it is feasible as a perfect entity-resolution view.

## EVR status

### Execute

Verified live surfaces:

- CMS NPPES NPI downloadable files page.
- NPPES Registry help page describing downloadable files and API.
- NPPES May 2026 monthly full ZIP URL.
- NPPES May 2026 deactivation ZIP URL.
- CMS Provider Data Catalog metastore JSON for:
  - `mj5m-pzi6` — Doctors and Clinicians National Downloadable File.
  - `4pq5-n9py` — Provider Information / nursing homes.
- SAM.gov Assistance Listings Public API documentation.
- Acquisition.gov PSC Manual April 2025 PDF.
- Census NAICS Sector 62 page.

### Verify

Verified:

- NPPES current monthly full file exists as `NPPES_Data_Dissemination_May_2026_V2.zip`.
- NPPES full ZIP probe returned HTTP 200, `content-type: application/zip`, `content-length: 1,131,435,518`, and `last-modified: Mon, 11 May 2026 08:54:22 GMT`.
- NPPES page says each zipped file includes three reference files: other names, practice locations, and endpoints.
- NPPES help says the downloadable file is large, intended for technical users, and the NPPES Read API exposes public information associated with an NPI.
- NPPES page warns that NPI issuance does not ensure or validate that the provider is licensed or credentialed.
- CMS Provider Data metastore returned public metadata for the Doctors and Clinicians national downloadable file and nursing-home provider-information dataset.
- `mj5m-pzi6` metadata returned `accessLevel: public`, title `National Downloadable File`, modified `2026-05-04`, released `2026-05-14`, next update `2026-06-11`.
- `4pq5-n9py` metadata returned `accessLevel: public`, title `Provider Information`, modified `2026-05-01`, released `2026-05-27`, next update `2026-06-24`.
- SAM Assistance Listings API docs expose `https://api.sam.gov/assistance-listings/v1/search`, require `api_key`, and can filter assistance listings.
- PSC Manual says Product and Service Codes describe what the federal government purchases; Category Q is Medical.
- Census NAICS Sector 62 page covers Health Care and Social Assistance.

Not verified:

- Exact NPPES CSV header inside the 1.1GB monthly ZIP. The ZIP was not extracted in this gate.
- NPPES Data Dissemination Notice full legal terms.
- CMS Provider Data distribution CSV headers.
- Any source-native NPI↔EIN, NPI↔UEI, CCN↔EIN, or CCN↔UEI bridge.

### Report

This file is the Gate 4 report artifact.

## Blunt conclusion

Healthcare cross-cut is feasible, but not as a magic provider-ID join.

The right first version is a classification view over existing spending facts:

- assistance programs likely health-related by Assistance Listing/program metadata,
- contracts likely health-related by PSC and NAICS,
- entities likely healthcare providers by NPPES/CMS/NAICS/name-address evidence,
- nonprofit healthcare entities linked where FAC/EO BMF/990 evidence supports EIN aliases.

Trying to make NPI or CCN solve UEI↔EIN would be a bad idea. Does not compute. NPI identifies healthcare providers. CCN identifies CMS-certified facilities in CMS contexts. Neither is proven to identify the legal award recipient in USAspending.

## Surface 1: NPPES / NPI

Official page:

```text
https://download.cms.gov/nppes/NPI_Files.html
```

NPPES Registry help:

```text
https://npiregistry.cms.hhs.gov/help/help
```

Verified current monthly file:

```text
https://download.cms.gov/nppes/NPPES_Data_Dissemination_May_2026_V2.zip
```

Observed facts:

- HTTP status: 200.
- `content-type: application/zip`.
- `content-length: 1,131,435,518`.
- `last-modified: Mon, 11 May 2026 08:54:22 GMT`.
- Page-displayed size: 1,079.02 MB.
- Inner ZIP sample began with `npidata_pfile_20050523-20260510.csv`.

NPPES page states each zipped downloadable file includes:

1. Other Name Reference File — additional other names associated with Type 2 NPIs.
2. Practice Location Reference File — all non-primary practice locations associated with Type 1 and Type 2 NPIs.
3. Endpoint Reference File — endpoints associated with Type 1 and Type 2 NPIs.

NPPES page warning:

```text
Issuance of an NPI does not ensure or validate that the Health Care Provider is Licensed or Credentialed.
```

Implication:

- NPI is excellent for provider discovery and healthcare entity enrichment.
- NPI is not a spending-recipient identifier and not an EIN/UEI crosswalk.
- NPI should be stored as a separate identifier namespace and alias evidence.

## Surface 2: CMS Provider Data Catalog

Provider Data Catalog:

```text
https://data.cms.gov/provider-data
```

Metastore examples verified:

```text
https://data.cms.gov/provider-data/api/1/metastore/schemas/dataset/items/mj5m-pzi6
https://data.cms.gov/provider-data/api/1/metastore/schemas/dataset/items/4pq5-n9py
```

Observed `mj5m-pzi6` metadata:

- `accessLevel`: public.
- title: `National Downloadable File`.
- keywords included `Clinicians`, `Quality`, `Location`.
- modified: `2026-05-04`.
- released: `2026-05-14`.
- next update: `2026-06-11`.
- distribution count: 1.

Observed `4pq5-n9py` metadata:

- `accessLevel`: public.
- title: `Provider Information`.
- keywords included `General Information`, `Address`, `Location`, `Ratings`, `Beds`, `Quality`, `Staffing`, `Penalties`.
- modified: `2026-05-01`.
- released: `2026-05-27`.
- next update: `2026-06-24`.
- distribution count: 1.

Implication:

- CMS Provider Data is viable for provider/facility enrichment and quality/context overlays.
- It can help distinguish hospitals, nursing homes, clinicians, dialysis facilities, and other provider classes.
- It does not itself prove that a USAspending recipient, SAM entity, IRS EIN, NPI, and CCN are the same legal entity.

## Surface 3: SAM Assistance Listings

Official API docs:

```text
https://open.gsa.gov/api/assistance-listings-api
```

Production endpoint:

```text
https://api.sam.gov/assistance-listings/v1/search
```

Verified docs facts:

- Requires `api_key`.
- Active/inactive federal assistance listings are available.
- Similar to the former CFDA catalog.
- Rate limits depend on account type:
  - non-federal/no role: 10 requests/day,
  - non-federal with role: 1,000 requests/day,
  - federal user: 1,000 requests/day.
- API supports searching/filtering listing metadata.

Healthcare implication:

- Assistance Listings are a strong program-level classifier for grants and cooperative agreements.
- Health programs can be tagged by assistance listing metadata, agency, program objectives, applicant/beneficiary types, and assistance type.
- This complements USAspending award rows where assistance listing IDs/program numbers appear.
- Key-dependent probe is pending; do not block the feasibility verdict on it.

## Surface 4: PSC Manual for contracts

Official PSC Manual PDF:

```text
https://www.acquisition.gov/sites/default/files/manual/PSC%20Manual%20April%202025.pdf
```

Verified extracted facts:

- PSC describes `WHAT` is being purchased in federal contract actions reported in FPDS.
- If a contract includes multiple products/services, the PSC should be selected by predominant item purchased.
- Category Q is Medical.
- Medical-service examples include nursing, pathology, behavioral/mental health, and healthcare environmental cleaning.

Healthcare implication:

- For prime contracts and subawards, PSC is the best first-pass official classification dimension for purchased healthcare goods/services.
- PSC is not enough by itself; some healthcare spending hides in IT, facilities, research, support, and supplies categories.
- Use PSC as a high-precision tag, not as total healthcare-spend truth.

## Surface 5: NAICS Sector 62

Official Census page:

```text
https://www.census.gov/data/tables/2022/econ/economic-census/naics-sector-62.html
```

Verified extracted facts:

- NAICS Sector 62 is Health Care and Social Assistance.
- Census provides 2022 Economic Census tables and FTP files for the sector.
- Tables include summary statistics, patient-care revenue, payer type, telemedicine, grants/transferred contributions, and hospital ownership/control.

Healthcare implication:

- NAICS 62 is a useful entity/vendor industry classifier where SAM/USAspending recipient/vendor NAICS is available.
- NAICS classifies the establishment/entity industry, not the specific purchased object.
- Combining NAICS + PSC gives better precision than either alone.

## Feasibility matrix

| Question | Feasible now? | Best source | Caveat |
|---|---:|---|---|
| Identify explicitly medical federal contracts | Yes | PSC Category Q and related medical product PSCs | PSC is predominant purchase only; undercounts mixed contracts. |
| Identify healthcare/social-assistance vendors/entities | Partly | NAICS 62, NPPES, CMS Provider Data | Entity resolution remains fuzzy without exact identifiers. |
| Identify federal healthcare assistance programs | Yes after key | SAM Assistance Listings + USAspending assistance fields | API key required; program metadata must be mapped carefully. |
| Link healthcare providers to EIN-backed nonprofit profiles | Partly | NPPES/CMS + EO BMF + FAC + ProPublica | Name/address/FAC evidence only; false positives likely. |
| Link NPI directly to USAspending recipient UEI | Not verified | None verified | Requires candidate alias matching unless a source bridge exists. |
| Link CCN directly to USAspending recipient UEI/EIN | Not verified | None verified | Facility/legal-entity boundaries are messy. |
| Compute all-in public healthcare spending | No | Multiple | “All-in” crosses Medicare claims, grants, contracts, payroll benefits, tax expenditures, Medicaid state flows, and local spending. Scope must be constrained. |

## Recommended Gate 4 implementation scope

Build a healthcare cross-cut view in layers:

### Layer 1 — award/program classification

Classify existing spending facts using fields already in award/account surfaces:

- assistance listing/program number and title,
- awarding/funding agency,
- PSC,
- NAICS,
- object class/program activity where available,
- award descriptions with deterministic keyword/rule tags only as secondary evidence.

Output should be a classification assignment, not an entity merge.

### Layer 2 — entity/provider enrichment

Ingest enrichment surfaces:

- NPPES NPI monthly full replacement and weekly increments,
- CMS Provider Data Catalog datasets relevant to hospitals, nursing homes, clinicians, dialysis, home health, hospice, etc.,
- EO BMF and FAC for nonprofit healthcare entities.

Store NPI, CCN, EIN, UEI, names, addresses, and aliases as separate identifiers/evidence rows.

### Layer 3 — cautious candidate crosswalk

Generate candidate healthcare entity links by:

- exact NPI only when both records expose NPI,
- exact CCN only when both records expose CCN,
- FAC UEI↔EIN where present,
- SAM/USAspending name/address ↔ NPPES/CMS/EO BMF name/address with confidence scoring.

Never collapse NPI/CCN/EIN/UEI into one canonical identity without source-backed evidence.

## Gate 4 verdict

Gate 4 can close as feasible with constraints.

Healthcare cross-cut is viable if Outlays treats it as a classification-and-enrichment view over atomic facts:

- High confidence: PSC Category Q medical contracts, Assistance Listing health programs, NAICS 62 entity classifications.
- Medium confidence: NPPES/CMS provider enrichment by name/address.
- Low confidence: name-only matches and broad keyword matches.
- Not solved: direct NPI/CCN↔UEI/EIN bridge.

The first product should say “healthcare-tagged public spending” or “healthcare-likely cross-cut,” not “total healthcare spend.” Precision beats bravado here.

## Carry-forward to Gate 5

Next brief gate should choose the next backlog source family after healthcare:

- CA revenue adapter if the goal is state/local fiscal balance,
- additional lead rules if the goal is product demos,
- RAG chatbot if the goal is user-facing query UX,
- federated submission verification / zkTLS if the goal is provenance hardening.

Do not start ingestion implementation until source-doc gates that unblock the backlog are present in `docs/sources/` and committed.
