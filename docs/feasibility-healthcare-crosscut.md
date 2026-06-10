# Healthcare cross-cut feasibility

## Gate

Gate 4 — Healthcare cross-cut feasibility.

Question from brief:

> With data available today, can we compute total government healthcare cost INCLUDING employee health benefits buried inside police, education, postal, and other department budgets, for (a) the federal government and (b) California?

Short answer:

- Federal: aggregate/object-class grain only. Not exact total healthcare cost.
- California: aggregate/report grain only for CalPERS health-benefit program totals and department `Staff Benefits`; not computable at department health-premium grain from verified public machine-readable sources.

The honest verdict is negative for the motivating “total healthcare cost including embedded employee benefits everywhere” question. We can compute useful partial views, but not the all-in number without assumptions or nonpublic payroll/benefit allocation data.

## EVR status

### Execute

Probed or extracted:

- Federal object-class documentation:
  - `https://www.whitehouse.gov/wp-content/uploads/2025/04/BUDGET-2026-OBJCLASS.pdf`
  - supporting search result for OMB Circular A-11 Section 83 object classification.
- USAspending account-download surface already verified in Gate 1:
  - `POST https://api.usaspending.gov/api/v2/download/accounts/`
  - `submission_types: ["object_class_program_activity"]`
- CalHR state employee health benefits page:
  - `https://benefits.calhr.ca.gov/state-employees/general-benefits/health`
- CalPERS Health Program page:
  - `https://www.calpers.ca.gov/employers/benefit-programs/health-benefits/calpers-health-program`
- CalPERS Health Benefits Program annual report PDF:
  - `https://www.calpers.ca.gov/sites/default/files/documents/2024/11/health-benefits-program-annual-report-2024.pdf`
- California eBudget Department Index:
  - `https://ebudget.ca.gov/budget/p/2025-26/DepartmentIndex`
- California eBudget sample department PDF:
  - `https://ebudget.ca.gov/2025-26/pdf/GovernorsBudget/4000/4800.pdf`
- California Open Data CKAN search:
  - `https://data.ca.gov/api/3/action/package_search?q=CalPERS%20benefits&rows=5`
  - `https://data.ca.gov/api/3/action/package_search?q=%22health%20benefits%22%20department&rows=5`
  - `https://data.ca.gov/api/3/action/package_search?q=state%20employee%20health%20benefits%20expenditures&rows=10`

### Verify

Verified:

- Federal object class files use object classes for the nature of spending.
- Federal object class analysis exposes government-wide lines such as:
  - `12.1 Civilian personnel benefits`,
  - `12.2 Military personnel benefits`,
  - `13.0 Benefits for former personnel`,
  - `25.6 Medical care`,
  - `42.0 Insurance claims and indemnities`.
- The White House FY2026 object-class analysis includes FY2024 actuals, FY2025 estimates, and FY2026 estimates.
- Gate 1 already verified that USAspending account downloads can generate object-class/program-activity files. A narrow Access Board probe returned 46 rows and 59 columns for `object_class_program_activity`.
- CalHR page states CalPERS administers health insurance coverage for state employees and CalHR administers dental, vision, and voluntary plan benefits.
- CalHR page publishes employer health contributions and CoBen allowances by bargaining unit, not department actual health-premium expenditures.
- CalPERS Health Program page states that in 2024 CalPERS spent over $12.4 billion to purchase health benefits for active and retired members and families on behalf of the State of California, CSU, and nearly 1,200 public agencies and schools.
- CalPERS 2023 annual report extraction reported nearly $11.3 billion spent on health benefits in 2023 and premium expenditures by member type: active, retired, total.
- California eBudget department PDFs are live. Sample `4800 California Health Benefit Exchange` PDF returned HTTP 200, `content-type: application/pdf`, size 225,048 bytes, and includes department budget tables with `Staff Benefits` lines.
- California Open Data CKAN returned a machine-readable dataset titled `CalPERS Supplement - Contracting Agencies' Benefits for Fiscal Years 2002-03 to 2023-24` with CSV/JSON resources, license `Creative Commons Attribution`, and notes saying it compares selected benefits of local public agencies contracted with CalPERS. This is not state department actual health-premium spending.

Not verified:

- Any CalHR or CalPERS machine-readable source with actual health premium expenditures by California state department.
- Any California eBudget machine-readable table that splits each department's `Staff Benefits` into health, dental, vision, retirement, OPEB, etc.
- Any federal public account-level file that splits object class `12.1 Civilian personnel benefits` into employee health insurance versus retirement, Social Security, Medicare taxes, life insurance, and other benefits.
- Any public federal or California source that allocates retired-member health benefits back to the department where service occurred.

### Report

This file is the Gate 4 report artifact.

## Verdict matrix

| Jurisdiction | Can compute total healthcare cost including embedded employee health benefits? | Achievable grain now | Exact files/surfaces |
|---|---:|---|---|
| Federal | No, not exactly | Aggregate/object-class and account/program activity partials | USAspending account downloads, OMB/White House object-class analysis, PSC/award files for purchased medical care |
| California | No, not exactly | Aggregate CalPERS health-program totals; department `Staff Benefits` totals; rates by bargaining unit | CalPERS Health Program page/report, CalHR benefits page, eBudget department PDFs, data.ca.gov CKAN |

## Federal findings

### What is computable now

Federal spending can support partial healthcare views:

1. Medical procurement and purchased care:
   - Contract award data with PSC medical categories.
   - Federal object class `25.6 Medical care` in object-class analysis/account views.

2. Employee/personnel benefits as a broad bucket:
   - Object class `12.1 Civilian personnel benefits`.
   - Object class `12.2 Military personnel benefits`.
   - Object class `13.0 Benefits for former personnel`.

3. Account/program/object-class intersections:
   - Gate 1 verified USAspending account download endpoint for `object_class_program_activity`.
   - That surface can provide account/program activity/object-class detail, subject to generated-job parameters and available columns.

### Why exact total healthcare is not computable

Object class `12.1 Civilian personnel benefits` is not a health-insurance-only field.

It is a bundle. It can include health-related employer costs, but it also includes non-health benefits. Counting all `12.1` as healthcare would overstate healthcare. Excluding it would understate healthcare.

Similar issue for:

- `12.2 Military personnel benefits`,
- `13.0 Benefits for former personnel`,
- retiree health/OPEB-like obligations,
- postal/agency-specific benefit accounting,
- interagency reimbursements.

The public surfaces verified in this gate do not expose a federal account-level field that cleanly splits employee health premiums from the broader personnel-benefit bucket.

### Federal gate verdict

Federal healthcare cross-cut is:

- computable for medical procurement and explicit medical-care object classes,
- aggregate/object-class computable for broad personnel benefits,
- not computable as exact all-in healthcare including embedded employee health benefits.

Recommended label:

```text
Federal healthcare-likely spending, partial coverage, with object-class caveats.
```

Do not label it:

```text
Total federal healthcare cost.
```

That would be bullshit with footnotes. Hermes rejects bullshit with footnotes.

## California findings

### CalHR

Official page:

```text
https://benefits.calhr.ca.gov/state-employees/general-benefits/health
```

Verified facts:

- CalPERS administers health insurance coverage for state employees and retirees.
- CalHR administers dental, vision, and voluntary plan benefits.
- The page publishes employer health contributions and CoBen allowances by bargaining unit.
- Contribution amounts are rates/allowances, not actual department expenditure files.

Useful fields conceptually:

- bargaining unit,
- employer contribution amount by coverage tier,
- CoBen allowance,
- plan/premium choices via CalPERS.

Missing for Outlays all-in computation:

- department actual enrolled headcount by plan/tier/month,
- department actual employer premium expenditure,
- retired-member allocation by former department,
- machine-readable department-year health-benefit total.

### CalPERS

Official page:

```text
https://www.calpers.ca.gov/employers/benefit-programs/health-benefits/calpers-health-program
```

Official annual report:

```text
https://www.calpers.ca.gov/sites/default/files/documents/2024/11/health-benefits-program-annual-report-2024.pdf
```

Verified facts:

- CalPERS says it spent over $12.4 billion in 2024 to purchase health benefits for active and retired members/families on behalf of the State of California, CSU, and nearly 1,200 public agencies and schools.
- The 2023 annual report extraction reported nearly $11.3 billion in health-benefit spending and summarized premium expenditures by active and retired member type.

What this supports:

- Statewide / program-level aggregate health-benefit spending.
- Historic premium trends and active/retired split.

What this does not support:

- California state department actual health-premium expenditure by department.
- Local-vs-state-vs-CSU allocation at a clean department grain from the extracted surfaces.
- All police/education/postal-equivalent embedded employee health costs across all public employers.

### California eBudget

Official department index:

```text
https://ebudget.ca.gov/budget/p/2025-26/DepartmentIndex
```

Sample department PDF:

```text
https://ebudget.ca.gov/2025-26/pdf/GovernorsBudget/4000/4800.pdf
```

Verified facts:

- eBudget department index is live.
- Department PDFs are live and include department budget tables.
- Search/extraction showed department PDFs contain `Staff Benefits` lines.

What this supports:

- Department-level broad staff-benefit totals from PDFs/extraction.

What this does not support:

- Health-only staff-benefit extraction.
- Machine-readable, official department-year health-premium spending.

### California Open Data / SCO CalPERS supplement

CKAN query:

```text
https://data.ca.gov/api/3/action/package_search?q=CalPERS%20benefits&rows=5
```

Verified result:

```text
CalPERS Supplement - Contracting Agencies' Benefits for Fiscal Years 2002-03 to 2023-24
```

Resource examples:

```text
https://bythenumbers.sco.ca.gov/api/views/ew9v-kudu/rows.csv?accessType=DOWNLOAD
https://bythenumbers.sco.ca.gov/api/views/ew9v-kudu/rows.json?accessType=DOWNLOAD
```

Verified metadata:

- organization: California State Controller's Office.
- license: Creative Commons Attribution.
- format: CSV/JSON/RDF/XML resources.
- notes: compares selected benefits of local public agencies contracted with CalPERS.

Finding:

This is useful for local-public-agency benefits context, but it is not the requested state department actual health-premium expenditure file.

### California gate verdict

California healthcare cross-cut is:

- aggregate computable from CalPERS health-program totals,
- broad department staff-benefit extractable from eBudget PDFs,
- rate/allowance computable by bargaining unit from CalHR,
- not computable as exact department-level health-benefit spending from verified machine-readable public sources.

Recommended label:

```text
California public-employee health-benefit aggregate and partial department staff-benefit view.
```

Do not label it:

```text
Total California government healthcare cost by department.
```

## Practical implementation recommendation

Create three separate views instead of one fake total:

### View A — explicit healthcare spending

Use:

- federal PSC medical categories,
- federal object class `25.6 Medical care`,
- Medicare/Medicaid/health program assistance listings where verified,
- California health-program budget lines and Medi-Cal sources when separately documented.

This is high precision but undercounts all-in healthcare burden.

### View B — employee benefits burden

Use:

- federal object classes `12.1`, `12.2`, `13.0`, with explicit note that these are personnel-benefit buckets, not health-only.
- California eBudget `Staff Benefits` lines by department, with explicit note that these are not health-only.

This captures embedded benefits but overstates healthcare if labeled as healthcare.

### View C — health-benefit aggregate controls

Use:

- CalPERS aggregate health-benefit program expenditures.
- any federal government-wide health-benefit aggregate found in later OPM/FEHB-specific research, if a later gate adds it.

This supplies control totals, not transaction/department grain.

## Recommended data model handling

Do not force a single `healthcare_total` metric.

Use separate classifications:

```text
healthcare_explicit = direct medical care / medical procurement / health program spending
employee_benefits_broad = staff/personnel benefit buckets, not health-only
health_benefits_aggregate = source-reported aggregate premiums/claims where available
```

Then publish coverage notes:

```text
This view is not all government healthcare cost. Embedded employee health premiums are only visible at broad benefit or aggregate-program grain in verified public sources.
```

## Carry-forward gaps

If Outlays wants an exact employee-health view later, the missing files are:

1. Federal FEHB or OPM employer contribution outlays by agency/department, machine-readable and public.
2. California CalPERS actual employer health premium expenditure by state department, machine-readable and public.
3. Enrollment/headcount by plan/tier/month and department, if expenditures are not directly published.
4. A method to allocate retiree health benefits to prior employing department, if departmental all-in cost is required.

Until those exist or are obtained, the all-in claim should stay off the product page.

## Gate 4 closeout

Gate 4 status: closed with a constrained/negative verdict.

- Federal: aggregate/object-class only; exact all-in not computable from verified sources.
- California: aggregate/report/rate/staff-benefit only; exact department health-premium spending not verified.
- Product decision: present partial views with explicit coverage notes, not one all-in healthcare total.
