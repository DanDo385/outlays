# Licensing matrix

## Addendum gate

A1 — explicit data license or terms for every source documented in Gates 1-7.

Purpose:

Close the licensing and redistribution gap left open during Gate 1 and make every source's reuse posture visible before ingestion.

## EVR status

### Execute

Reviewed the source documents produced in Gates 1-7 and inventoried their documented public surfaces:

- `docs/sources/us-fed-bulk.md`
- `docs/sources/nonprofit-stack.md`
- `docs/sources/entity-identifiers.md`
- `docs/sources/healthcare-cross-cut.md`
- `docs/feasibility-healthcare-crosscut.md`
- `docs/sources/us-ca-revenue.md`
- `docs/cofog-references.md`
- `docs/sources/us-ny.md`

Checked official source pages or official terms/help pages where available. Dataset-level metadata was not re-fetched through the local shell after a cert-bypass metadata command was blocked; no retry was attempted. Entries below therefore distinguish:

- explicit license or terms verified from official pages;
- no explicit license found on checked official pages;
- restrictions or attribution rules that affect redistribution/commercial use.

### Verify

Minimum fields verified for every matrix row:

- gate coverage;
- source / URL;
- explicit license or terms finding;
- verbatim citation;
- redistribution / commercial-use risk.

### Report

This document is the addendum report.

## Summary verdict

Most U.S. federal government source material is public-facing and often public-domain by copyright posture, but several critical surfaces do not publish a dataset-specific license on the checked page. Do not treat `publicly accessible` as the same thing as `licensed for unrestricted redistribution`.

Highest-risk terms:

1. SAM.gov / GSA entity data has explicit restrictions around FOUO, Sensitive CUI, and D&B-supplied data.
2. Census API has explicit user obligations, including attribution/disclaimer and anti-reidentification restrictions.
3. ProPublica Nonprofit Explorer is a third-party derived service with ProPublica terms and possible dataset-specific terms.
4. IMF eLibrary material is not an ingestion source here and should be treated as reference-only unless reuse terms are separately verified.
5. California and New York portal datasets are generally open-public-data surfaces, but source-specific metadata should be preserved per dataset.

## License / terms matrix

| Gate | Source | Checked URL | License / terms finding | Verbatim citation | Redistribution / commercial-use flag | Outlays action |
|---|---|---|---|---|---|---|
| 1 | USAspending.gov API and bulk files | https://api.usaspending.gov/ ; https://www.usaspending.gov/about | No explicit data license found on checked pages. Official pages state public access under the DATA Act, not a named license. | `The USAspending API (Application Programming Interface) allows the public to access comprehensive U.S. government spending data.` Also: `The U.S. Department of the Treasury is building a suite of open-source tools to help federal agencies comply with the DATA Act ... and to deliver the resulting standardized federal spending information back to agencies and to the public.` | No specific redistribution/commercial restriction found for the USAspending award/account bulk files on checked pages. However, do not imply Treasury endorsement. Preserve source URL and retrieval date. | Ingest as public federal spending data. Store provenance. Label license as `no explicit dataset license stated; public access under DATA Act`. |
| 1 | USAspending D&B-derived entity fields, where present | https://www.usaspending.gov/about | Explicit dataset-level license not confirmed. USAspending site notes D&B data disclaimers in source help content. | Checked page summary noted: `The site includes specific disclaimers regarding the use of Dun & Bradstreet (D&B) data and maintains strict adherence to FOIA, Privacy, and Accessibility policies.` | Potential redistribution/commercial-marketing risk for D&B-supplied legal business name/address fields, especially if cross-sourced from SAM.gov. | Keep raw award fields but avoid packaging D&B-enriched entity files as a resale/marketing database. Add D&B provenance flag if fields are known D&B-derived. |
| 2 | Federal Audit Clearinghouse data and CSV/API | https://www.fac.gov/data ; https://www.fac.gov/api/terms | Explicit public-domain statement. | `The data collected by the FAC is free to use and in the public domain.` API terms also state: `All single audit reporting packages, with the possible exception of Indian Tribes and Tribal Organizations, submitted under Uniform Guidance are available to the public per 2 CFR 200.512(b)(1).` | Public domain, but tribal reporting packages may be unavailable/non-public under 2 CFR 200.512(b)(2). | Ingest public FAC CSV/API data. Preserve unavailable/tribal suppression notes. |
| 2 | IRS Form 990 series downloads / XML | https://www.irs.gov/charities-non-profits/form-990-series-downloads ; https://www.irs.gov/about-irs/use-of-content-from-irsgov | IRS site content created by federal employees is not subject to copyright and may be freely copied; credit requested. | `Content on this website that was created or maintained by federal employees in the course of their duties is not subject to copyright and may be freely copied. Credit is requested.` | IRS seals/names/symbols may not be used to imply endorsement. Third-party copyrighted material, if present, requires separate permission. | Ingest IRS public Form 990 XML/indexes. Credit IRS. Do not use IRS marks in product marketing. |
| 2 | ProPublica Nonprofit Explorer API | https://projects.propublica.org/nonprofits/api ; https://www.propublica.org/about/propublica-data-terms-of-use | Third-party service terms apply; free data may generally be used under ProPublica terms, but dataset-specific terms can override. | API page: `Legal: Usage constitutes agreement to ProPublica’s Data Terms of Use.` Terms page: `In general, you may use the free data published by ProPublica under the following terms. However, there may be different terms included for some datasets. It is your responsibility to read carefully any specific terms included with the data you download from our website.` | Copyrighted third-party API/service. Redistribution posture is not equivalent to IRS public-domain source data. | Prefer IRS primary bulk data for ingestion. Use ProPublica API as convenience/reconciliation only with attribution and terms link. |
| 3 | SAM.gov Entity Management API / open.gsa.gov docs | https://open.gsa.gov/api/entity-api ; https://sam.gov/about/terms-of-use | Public API and extract are documented, but live access was not verified in Gate 3 P2/P2.1. Corrected-host probes against `api.sam.gov/entity-information/v3/entities` with v2 fallback returned empty 404 responses with no gateway/rate-limit envelope. SAM terms impose access, sensitivity, and D&B restrictions. Public data is unclassified; FOUO/Sensitive data is restricted. | GSA API doc: `Public Data: Unclassified entity information (Name, UEI, registration details, addresses, business types, PSC, NAICS, and points of contact).` Same page: `Sensitive (CUI) Data: Includes all above plus banking information and tax identifiers (SSN/TIN/EIN).` Docs also state synchronous JSON is paged and extracts are available via `format=csv` or `format=json`. SAM terms summary: FOUO is for internal U.S. Government business and public dissemination is prohibited unless directly associated with a Federal award record. | Restricts FOUO/Sensitive data. Sensitive includes SSN/TIN/EIN. Public API key required. Live public API/extract access is not verified for Outlays after corrected-host P2.1 probes. Do not build a public view from non-public APIs. | Do not ingest FOUO or Sensitive SAM fields. Do not pursue EIN/TIN through SAM for public product. Treat public SAM entity fields as documentation-supported but not Phase 1 ingestion-ready until a key or endpoint variant returns schema/page/extract evidence. |
| 3 | SAM.gov D&B-supplied data | https://sam.gov/about/terms-of-use | Explicit restriction and attribution risk, including no D&B Open Data bulk dissemination and no commercial/resale/marketing use for non-open D&B data. | SAM terms summary: `SAM.gov contains data supplied by D&B, which remains their intellectual property.` D&B Open Data requires written attribution and may not be shared in bulk in amounts sufficient for use as an original data source or substitute for D&B. Other D&B data may not be used for commercial, resale, or marketing purposes. Systematic electronic harvesting or extraction, including bots/spiders, is prohibited. | High redistribution/commercial risk for D&B data; D&B Open Data has attribution and no-bulk-substitute limits; non-open D&B data has commercial/resale/marketing restrictions. | Flag D&B fields. Avoid bulk redistribution, resale, marketing, customer/prospect analysis, or packaging D&B-enriched entity data as a substitute source. Prefer government-generated identifiers and award facts over D&B enrichment. |
| 3 | IRS EO Business Master File extract | https://www.irs.gov/charities-non-profits/exempt-organizations-business-master-file-extract-eo-bmf ; https://www.irs.gov/about-irs/use-of-content-from-irsgov | IRS federal-employee content not subject to copyright; freely copied; credit requested. | `Content on this website that was created or maintained by federal employees in the course of their duties is not subject to copyright and may be freely copied. Credit is requested.` | No commercial restriction found on checked IRS content page; mark IRS attribution and no endorsement. | Ingest EO BMF as public IRS source for nonprofit EIN/name/address bridge with source timestamp. |
| 4 | CMS NPPES downloadable files / NPI registry | https://www.cms.gov/medicare/regulations-guidance/administrative-simplification/data-dissemination ; https://download.cms.gov/nppes/NPI_Files.html | FOIA-disclosable public data. No explicit named license found on checked page. | CMS page summary: `CMS has expanded its data dissemination strategy to provide more frequent and detailed information regarding NPI status.` It also states: `Information in the NPI Registry and downloadable files is FOIA-disclosable.` | CMS advises third-party websites to limit display of deactivated NPI data to NPI and deactivation date only. No intent/quality claims. | Ingest only FOIA-disclosable fields. Preserve deactivated-data display caution. |
| 4 | CMS Data / Provider Data Catalog API | https://data.cms.gov/api-docs ; https://data.cms.gov/provider-data | API docs provide public API access; no explicit named data license found in checked API docs. | `The Centers for Medicare & Medicaid Services (CMS) provides a RESTful API for real-time interaction with public datasets.` And: `All dataset requests follow this base URL structure: data.cms.gov/data-api/v1/dataset/{{dataset_id}}/data.` | No explicit redistribution restriction found in checked API docs. Dataset-specific pages may have additional notices. | Treat as public CMS dataset API, but preserve dataset URL and any dataset-specific notice at ingestion time. |
| 4 | Census NAICS / Census API references | https://www.census.gov/data/developers/about/terms-of-service.html | Explicit Census API terms apply. | `All services, which utilize or access the API, should display the following notice prominently within the application: "This product uses the Census Bureau Data API but is not endorsed or certified by the Census Bureau."` Also: users must not `attempt to identify any individual, household, or business.` | Must display Census disclaimer for API-based product use. Anti-reidentification obligations. API access may be terminated. | Use Census/NAICS as classification reference. Add Census disclaimer wherever Census API content is surfaced. Do not use for reidentification. |
| 4 | Acquisition.gov PSC Manual | https://www.acquisition.gov/psc-manual | No explicit license found on checked PSC page. Official GSA federal website. | PSC page states it is the `official repository for the Product and Service Code (PSC) Manual` and `An official website of the General Services Administration.` | No explicit commercial/redistribution restriction found on checked PSC page. Federal works are generally public domain, but this page did not state a license. | Use PSC as federal classification reference. Cite Acquisition.gov and version/date. |
| 4 | SAM.gov Assistance Listings API | https://open.gsa.gov/api/assistance-listings-api | Same SAM/GSA public API family. No separate license checked beyond SAM terms. | Open GSA API catalog identifies `SAM.gov Assistance Listings Public API` and says users can access/consume assistance listings data in bulk. | Subject to SAM/API terms, keys, rate limits, and public/non-public distinctions. | Treat as public assistance-listings reference if needed. Preserve terms URL and API key requirements. |
| 4 | White House / OMB object-class PDF | https://www.whitehouse.gov/wp-content/uploads/2025/04/BUDGET-2026-OBJCLASS.pdf ; https://www.whitehouse.gov/copyright | Explicit White House copyright policy for government-produced materials. | `Pursuant to federal law, government-produced materials appearing on this site are not copyright protected.` Third-party: `Except where otherwise noted, third-party content on this site is licensed under a Creative Commons Attribution 3.0 License.` | No endorsement. Third-party content requires CC BY 3.0 attribution unless otherwise noted. | Use OMB budget object-class PDF as reference. Cite source and retrieval date. |
| 4 | CalHR health benefits page | https://benefits.calhr.ca.gov/state-employees/general-benefits/health | No explicit data license found in Gate 4 source docs or checked pages. Public state benefits information. | Gate 4 documented this as a public benefits information surface, not a machine-readable expenditure source. No license text was verified. | Reference-only. Not suitable as ingestion source without separate terms/license review. | Do not ingest as spending data. Use only as contextual citation. |
| 4 | CalPERS health program / annual report | https://www.calpers.ca.gov/employers/benefit-programs/health-benefits/calpers-health-program ; https://www.calpers.ca.gov/sites/default/files/documents/2024/11/health-benefits-program-annual-report-2024.pdf | No explicit data license found in checked Gate 4 surfaces. Public report/reference. | Gate 4 verified public CalPERS report availability, not a redistribution license. | Treat PDF/report as reference-only until terms verified. Do not bulk redistribute extracted tables without source-specific review. | Reference for feasibility only, not Phase 1 ingestion. |
| 5 | California Open Data portal / data.ca.gov | https://lab.data.ca.gov/licenses ; https://handbook.data.ca.gov/portal-use | Explicit portal license guidance: most datasets public domain; metadata usually includes license. | `Public domain: Most datasets are released into the public domain. This means the dataset can be used freely without restriction under copyright law.` Handbook: `Every dataset includes descriptive information, such as ... Licensing (typically Public Domain).` | Portal says most/typically, not all. Dataset-specific license must be captured. | At ingestion, persist CKAN `license_id`, `license_title`, and `license_url` for each data.ca.gov dataset. |
| 5 | California DOF historical schedules / eBudget PDFs | https://dof.ca.gov/budget/historical-budget-information/summary-schedules-and-historical-charts ; https://ebudget.ca.gov | No explicit license found on checked Gate 5 surfaces. Official State of California budget publication. | Gate 5 verified DOF/eBudget official publication pages and PDF availability; no license text was confirmed. | No explicit restriction found, but lack of license means do not overstate reuse rights. | Ingest as official state budget publication only after storing source URL, retrieval date, and no-license-stated flag. |
| 5 | California CDTFA sales/use tax statistics | https://cdtfa.ca.gov/legal/research-and-statistics/sales-and-use-tax.htm | No explicit license found on checked Gate 5 surfaces. Official state tax statistics page. | Gate 5 recorded CDTFA as official statistics/reference surface; no license text was confirmed. | No explicit redistribution/commercial restriction found in checked docs; no explicit license confirmed. | Use as reference or secondary source. Preserve no-license-stated flag unless dataset metadata says otherwise. |
| 5 | California SCO ByTheNumbers | https://bythenumbers.sco.ca.gov | No explicit license found in Gate 5 checked sources. Socrata/open-data portal terms not separately verified for this source. | Gate 5 determined SCO ByTheNumbers is local-government revenue, not state revenue. No license text was confirmed. | Not Phase 1 state revenue source; also license remains unconfirmed. | Exclude from Phase 1 state revenue ingestion. If used later, verify dataset metadata and portal terms first. |
| 5 | California ArcGIS sales/use tax rate layer | https://gis.data.ca.gov/api/download/v1/items/01883a79765a4afba132ba54da408d8b/csv?layers=1 ; https://services6.arcgis.com/snwvZ3EmaoXJiugR/arcgis/rest/services/California_Sales_and_Use_Tax_Rates/FeatureServer/1 | No explicit license found in Gate 5 checked source docs. | Gate 5 verified ArcGIS downloadable service availability; no license text was confirmed. | ArcGIS item terms/attribution should be captured before ingestion. | Defer ingestion until ArcGIS item license/terms are retrieved and stored. |
| 6 | Eurostat COFOG manual/reference | https://ec.europa.eu/eurostat/help/copyright-notice ; https://ec.europa.eu/eurostat/web/products-manuals-and-guidelines/-/ks-gq-19-010 | Eurostat permits reuse, including commercial reuse unless exceptions apply. | `How to re-use Eurostat material for commercial purposes: There is no special procedure or requirement for a written licence. Just download the material and use it, unless the material is listed in the exceptions above.` Political context: ESS provides statistics `free of charge as a public good of high quality, irrespective of subsequent commercial or non-commercial use.` | Exceptions exist for third-party/co-publication data, examples include U.S., Japan, China data in tables that may need removal before commercial reuse. | Use Eurostat COFOG as classification reference. Cite Eurostat and watch exceptions. |
| 6 | IMF Government Finance Statistics Manual reference | https://www.elibrary.imf.org/display/book/9781498343763/ch001.xml ; https://www.imf.org/external/pubs/ft/gfs/manual/2014/gfsfinal.pdf | Terms/license not confirmed. IMF direct terms page returned 404 during this run; IMF PDF HEAD previously blocked with 403, chapter page was reachable. | Verified chapter page availability. No reusable license text was confirmed in this addendum run. | Treat as reference-only, not redistributable corpus. Do not ship copied IMF text/tables beyond fair quotation without license review. | Keep citations only. Do not ingest or redistribute IMF content. |
| 6 | U.S. House Budget Committee budget-functions reference | https://democrats-budget.house.gov/budgets/budget-functions | No explicit license found in Gate 6 checked surfaces. U.S. House public website. | Gate 6 verified page availability and used it only as explanatory reference. No license text was confirmed. | Federal legislative website content is public-facing, but no explicit license text captured here. | Cite only as explanatory reference. Not an ingestion source. |
| 7 | New York Data.NY.gov MTA Procurements dataset | https://data.ny.gov/Transportation/MTA-Procurements-Beginning-2018/twsw-2mqa ; https://data.ny.gov/api/views/twsw-2mqa | Dataset page is public and API-accessible. Dataset-level license metadata was not independently retrieved in this addendum after the cert-bypass metadata command was blocked. | Dataset page summary: `This dataset provides comprehensive records of annual procurement contracts reported by the Metropolitan Transportation Authority (MTA) to the Authorities Budget Office via the Public Authorities Reporting Information System (PARIS).` It also states `API Access: Developers can access the data via Socrata-supported endpoints.` | License/terms need dataset metadata capture before production ingestion. No redistribution/commercial restriction confirmed from checked page summary. | Use as mechanics test only until license metadata is stored. Before ingestion, capture Socrata `licenseId`/terms and dataset owner attribution. |

## Explicit restriction flags

### SAM.gov / GSA entity data

Do not ingest or publish non-public SAM.gov data.

Rules:

- Public/unclassified entity data may be available through public API-key access.
- Public SAM API and extract access is documentation-supported but not live-verified for Outlays; Gate 3 P2/P2.1 corrected-host probes returned empty 404 responses with no rate-limit or structured JSON error envelope.
- FOUO/CUI is not public product data.
- Sensitive fields include SSN/TIN/EIN and banking data.
- D&B Open Data has attribution and no-bulk-substitute limits; non-open D&B material has commercial/resale/marketing restrictions; systematic electronic harvesting is prohibited.

Builder implication:

The public UEI/name/address/NAICS/PSC surface may be usable after live access is verified. The desired UEI→EIN bridge cannot be built from SAM Sensitive data for a public/commercial product.

### Census API

If Census API data appears in product, show the required disclaimer:

> This product uses the Census Bureau Data API but is not endorsed or certified by the Census Bureau.

Do not combine Census data to re-identify individuals, households, or businesses.

### ProPublica

Use ProPublica Nonprofit Explorer for discovery or reconciliation, not as the canonical redistributable source. For canonical nonprofit financials, prefer IRS Form 990 bulk data and FAC public-domain data.

### California and New York open-data portals

Portal-level openness is not enough. Store dataset-level license metadata at ingestion time.

Required ingestion fields:

- source URL;
- retrieval timestamp;
- portal dataset ID;
- license ID/title/URL if available;
- attribution/owner;
- no-license-stated flag if missing.

## Gate 1 USAspending license gap closure

Finding:

USAspending bulk files are public federal spending data made publicly accessible under the DATA Act, but the checked USAspending API/about pages did not state a named data license such as CC0, CC BY, ODC-BY, or public domain.

Verbatim support:

- `The USAspending API (Application Programming Interface) allows the public to access comprehensive U.S. government spending data.`
- `The U.S. Department of the Treasury is building a suite of open-source tools to help federal agencies comply with the DATA Act ... and to deliver the resulting standardized federal spending information back to agencies and to the public.`

Outlays label:

`license_status: no_explicit_dataset_license_stated`

`reuse_basis: public federal spending data / DATA Act public access`

`restrictions: no endorsement; preserve source provenance; D&B-derived entity fields require caution`

## Recommended metadata schema additions

Add these fields to source registry / ingestion manifests:

```json
{
  "license_status": "explicit|no_explicit_license_stated|terms_only|restricted|unknown",
  "license_name": "Public Domain|CC BY 4.0|Terms of Use|...",
  "license_url": "https://...",
  "terms_url": "https://...",
  "verbatim_license_quote": "...",
  "commercial_use": "allowed|restricted|not_confirmed",
  "redistribution": "allowed|restricted|not_confirmed",
  "attribution_required": true,
  "endorsement_disclaimer_required": true,
  "sensitive_data_excluded": true,
  "dataset_license_checked_at": "YYYY-MM-DD"
}
```

## Open follow-ups

1. Capture dataset-level license metadata for New York Data.NY.gov MTA Procurements before production ingestion.
2. Capture dataset-level CKAN license fields for every California `data.ca.gov` dataset actually ingested.
3. Verify California DOF/eBudget site terms if extracted state budget PDFs become redistributed datasets rather than citations.
4. Verify IMF reuse terms if any IMF material is copied into product documentation beyond citation/fair quotation.
5. Add an automated ingestion preflight that refuses `license_status: unknown` unless the source is explicitly marked `reference_only`.
