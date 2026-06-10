# Source: US:CA — revenue source census
- jurisdiction: US:CA
- base_url: https://dof.ca.gov/budget/historical-budget-information/summary-schedules-and-historical-charts; https://ebudget.ca.gov; https://bythenumbers.sco.ca.gov; https://data.ca.gov; https://cdtfa.ca.gov/legal/research-and-statistics/sales-and-use-tax.htm
- access_method: extraction | api | bulk-download
- working_query_or_file: DOF Summary Schedules page and `BS_SCH2.pdf`/`BS_SCH8.pdf`; SCO Socrata dataset `ky7j-fsk5`; FTB `pit-annual-report-2023`; CDTFA sales/use tax pages and data.ca.gov/ArcGIS resources.
- confirmed_at: 2026-06-10
- response_status: live
- formats_and_sizes: DOF/eBudget schedules are PDFs linked from official page; SCO ByTheNumbers is Socrata JSON/CSV; FTB Annual Report is PDF via data.ca.gov; CDTFA exposes HTML/PDF pages plus data.ca.gov/ArcGIS CSV/API resources for rates and some sales/tax datasets.
- fields_we_map: revenue category/source -> classification assignment; fiscal year -> fiscal_fact period; amount -> fiscal_fact.amount where available; jurisdiction/entity -> fiscal_fact payer/payee/context; tax type -> classification assignment.
- identifiers_present: none for entity linkage; jurisdiction names/codes only where source provides them.
- grain: aggregate
- update_cadence: DOF schedules annual with Governor's Budget and selected enactment updates; SCO local datasets irregular/annual; FTB annual report irregular/annual; CDTFA sales/use tax datasets vary by table.
- posting_lag: DOF current budget schedules for 2026-27 live; SCO city revenue sample had FY2024 row and Last-Modified Tue, 13 Jan 2026; FTB 2023 annual report updated June 5, 2026; CDTFA update cadence dataset-specific.
- license_or_terms: data.ca.gov FTB and SCO metadata observed Creative Commons Attribution where applicable; CDTFA ArcGIS/data.ca.gov records observed with `license_title: null` for sales/use-tax rates; DOF/eBudget explicit license not confirmed.
- known_gaps: DOF state revenue schedules appear PDF/extraction-first, not clean API/CSV. SCO ByTheNumbers covers local governments, not statewide state-level revenue. FTB and CDTFA are tax-source supplements, not the full state revenue control source. Explicit licenses missing for some CDTFA/DOF surfaces.
- notes: First California revenue adapter should use DOF Summary Schedules via extraction, with FTB/CDTFA as supporting tax-source detail and SCO reserved for local-government revenue.

## EVR status

### Execute

Verified required source families:

1. State Controller's Office By the Numbers portal.
2. Department of Finance/eBudget summary schedules.
3. Franchise Tax Board annual report data tables / Open Data catalog.
4. CDTFA sales and use tax research/statistics and open-data surfaces.

### Verify

Verified live responses:

- SCO ByTheNumbers homepage returned local-government finance categories.
- SCO Socrata dataset `ky7j-fsk5` returned one live JSON row:
  - `entity_name: Adelanto`,
  - `fiscal_year: 2024`,
  - `total_revenues: 24457824`,
  - `estimated_population: 36131`,
  - `revenues_per_capita: 677`.
- SCO Socrata metadata for `ky7j-fsk5` returned dataset name `City Revenues Per Capita`, category `Cities`, attribution `California State Controller's Office`, description stating per-capita values are total city revenues divided by population.
- DOF Summary Schedules page returned official schedule links including:
  - Schedule 1: General Budget Summary,
  - Schedule 2: Summary of State Tax Collections,
  - Schedule 3: Comparative Yield of State Taxes,
  - Schedule 8: Comparative Statement of Revenues,
  - Schedule 10: Summary of Fund Condition Statements.
- DOF page states schedules are published with the Governor's Budget on January 10 and schedules marked with an asterisk are also updated at budget enactment.
- FTB data.ca.gov package `pit-annual-report-2023` returned live CKAN metadata:
  - organization: California Franchise Tax Board,
  - license: Creative Commons Attribution,
  - public access level: Public,
  - rights: No restrictions on public use,
  - frequency: Irregular,
  - resource: PDF annual report.
- CDTFA sales/use-tax page states CDTFA compiles taxable sales data, revenue allocations to local cities/counties/special districts, and sales tax rates.
- CDTFA/data.ca.gov package search returned live resources for `CDTFA SalesandUseTaxRates Public` and related datasets, including ArcGIS REST and CSV resources.

Not verified:

- A DOF/eBudget CSV or JSON endpoint for Summary Schedule 2 or Schedule 8.
- Full machine-readable FTB annual-report table extraction beyond the package metadata.
- CDTFA taxable-sales dataset schema for every table.
- DOF/eBudget explicit redistribution license.

### Report

This file is the Gate 5 report artifact.

## Source family 1 — SCO By the Numbers

Official portal:

```text
https://bythenumbers.sco.ca.gov
```

Observed portal scope:

```text
Local Government Financial Data
```

Menu categories observed:

- City Data,
- County Data,
- Special District Data,
- Transit Operator Data,
- Transportation Planning Agency Data,
- Public Retirement System Data,
- Property Tax Data,
- City Street Data,
- County Road Data.

Socrata example:

```text
https://bythenumbers.sco.ca.gov/resource/ky7j-fsk5.json?$limit=1
```

Dataset metadata:

```text
https://bythenumbers.sco.ca.gov/api/views/ky7j-fsk5
```

Finding:

SCO ByTheNumbers is excellent for local-government revenue and spending, but it is not the first state-level revenue adapter. It answers city/county/special-district revenue questions, not California state General Fund revenue control totals.

## Source family 2 — DOF/eBudget Summary Schedules

Official page:

```text
https://dof.ca.gov/budget/historical-budget-information/summary-schedules-and-historical-charts
```

Key linked files observed:

```text
https://ebudget.ca.gov/2026-27/pdf/BudgetSummary/BS_SCH1.pdf
https://ebudget.ca.gov/2026-27/pdf/BudgetSummary/BS_SCH2.pdf
https://ebudget.ca.gov/2026-27/pdf/BudgetSummary/BS_SCH3.pdf
https://ebudget.ca.gov/2026-27/pdf/BudgetSummary/BS_SCH8.pdf
https://ebudget.ca.gov/2026-27/pdf/BudgetSummary/BS_SCH10.pdf
```

Relevant schedules:

- Schedule 1: General Budget Summary.
- Schedule 2: Summary of State Tax Collections.
- Schedule 3: Comparative Yield of State Taxes.
- Schedule 8: Comparative Statement of Revenues.
- Schedule 10: Summary of Fund Condition Statements.

Finding:

DOF/eBudget is the best first source for state-level revenue control totals. It is official, budget-facing, and directly describes state revenue schedules. The bad news: verified surface is PDF/extraction, not clean API. Still the right first adapter because source fit beats implementation comfort.

Recommended adapter shape:

```text
adapter: us-ca-dof-revenue-schedules
method: PDF/table extraction with checks against schedule totals
first files: BS_SCH2.pdf and BS_SCH8.pdf
output grain: aggregate state revenue by tax/source/fund/year
```

## Source family 3 — Franchise Tax Board

Open Data package:

```text
https://data.ca.gov/dataset/pit-annual-report-2023
```

CKAN API:

```text
https://data.ca.gov/api/3/action/package_show?id=pit-annual-report-2023
```

Verified metadata:

- Title: `PIT Annual Report 2023`.
- Organization: California Franchise Tax Board.
- License: Creative Commons Attribution.
- Rights: No restrictions on public use.
- Frequency: Irregular.
- Resource: PDF annual report.

Finding:

FTB is valuable for income-tax detail and statistical background. It should not be the primary state revenue adapter because it covers a subset of tax revenue and ships the current verified package as PDF rather than a universal revenue table.

Use FTB for:

- personal income tax statistics,
- corporate/franchise tax tables where separately documented,
- validation/explanation for DOF tax categories.

## Source family 4 — CDTFA

Official sales/use-tax research page:

```text
https://cdtfa.ca.gov/legal/research-and-statistics/sales-and-use-tax.htm
```

Observed CDTFA/data.ca.gov datasets:

```text
CDTFA SalesandUseTaxRates Public
CDTFA SalesandUseTaxRates
California Sales and Use Tax Rates
```

Example resources observed:

```text
https://services6.arcgis.com/snwvZ3EmaoXJiugR/arcgis/rest/services/California_Sales_and_Use_Tax_Rates/FeatureServer/1
https://gis.data.ca.gov/api/download/v1/items/01883a79765a4afba132ba54da408d8b/csv?layers=1
```

Finding:

CDTFA is strong for sales/use-tax rates, taxable-sales context, and local allocations. It is not the full state revenue control source. It should supplement DOF state revenue schedules, especially for explaining sales/use-tax components and local distributions.

License note:

- data.ca.gov metadata for CDTFA sales/use-tax rates returned `license_title: null` in this gate. Record as no explicit license confirmed for those specific records.

## Recommendation: first California revenue adapter

Build the first California revenue adapter from DOF Summary Schedules, specifically:

1. Schedule 2 — Summary of State Tax Collections.
2. Schedule 8 — Comparative Statement of Revenues.
3. Schedule 1 — General Budget Summary as a control/check.

This is extraction work, not API work.

Reasoning:

- It is the only verified source family in this gate that directly targets statewide state revenue control totals.
- SCO ByTheNumbers is easier technically because Socrata, but it is local-government scope. Using it first would solve the wrong problem.
- FTB and CDTFA are important supporting sources for tax detail, but they do not replace DOF as the comprehensive state revenue surface.

## Gate 5 closeout

Gate 5 status: closed.

- Biggest positive finding: DOF Summary Schedules provide official state revenue control surfaces.
- Biggest technical downside: first CA revenue adapter is PDF/table extraction, not a clean API.
- Biggest scope trap avoided: treating SCO local-government revenue as California state revenue.
