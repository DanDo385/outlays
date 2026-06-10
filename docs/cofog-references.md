# COFOG references

## Gate

Gate 6 — COFOG references + California mapping draft.

Deliverables:

- `docs/cofog-references.md`
- `data/cofog/us-ca-procurement.json`

## EVR status

### Execute

Verified or searched:

- Eurostat COFOG manual 2019 product page and PDF.
- IMF GFSM 2014 eLibrary page and official PDF URL behavior.
- U.S. federal budget functions documentation.
- Outlays California procurement adapter fixture for real source categories.

### Verify

Verified:

- Eurostat official product page exists for `Manual on sources and methods for the compilation of COFOG statistics — Classification of the Functions of Government (COFOG) — 2019 edition`.
- Eurostat manual PDF returned HTTP 200, `content-type: application/pdf`, `content-length: 3002820`, `last-modified: Wed, 25 Sep 2019 13:28:04 GMT`.
- IMF GFSM 2014 official eLibrary page exists and describes GFSM 2014 as the IMF fiscal-statistics framework; search result states Chapter 6 covers COFOG.
- Direct HEAD to the IMF `gfsfinal.pdf` URL returned 403 HTML in this environment, so the PDF URL is recorded but not HEAD-verified.
- U.S. federal budget functions documentation exists but is not COFOG. It organizes approximately 20 federal functions, not the 10 COFOG divisions.
- No official U.S. federal-budget-function to COFOG crosswalk was verified in this gate. Absence is a finding.
- Repo fixture contains real California procurement `Department Name` and `Acquisition Type` labels from data.ca.gov `Purchase Order Data` resource `bb82edc5-9c78-44e2-8947-68ece26197c5`.

Not verified:

- A published official California procurement-category to COFOG crosswalk.
- A published official U.S. state/local chart-of-accounts to COFOG crosswalk.
- Full text extraction of the IMF PDF due direct PDF 403; eLibrary page and search metadata were used.

### Report

This file is the Gate 6 reference report.

## Reference 1 — Eurostat COFOG manual

Product page:

```text
https://ec.europa.eu/eurostat/web/products-manuals-and-guidelines/-/ks-gq-19-010
```

PDF:

```text
https://ec.europa.eu/eurostat/documents/3859598/10142242/KS-GQ-19-010-EN-N.pdf
```

Verified metadata:

- Title: `Manual on sources and methods for the compilation of COFOG statistics — Classification of the Functions of Government (COFOG) — 2019 edition`.
- PDF content type: `application/pdf`.
- PDF size: 3,002,820 bytes.
- Last modified: `Wed, 25 Sep 2019 13:28:04 GMT`.

Use in Outlays:

- Primary operational reference for COFOG top-level functions and compilation practice.
- Basis for keeping unmappable items unmapped rather than forcing an invented function.

COFOG top-level codes carried in repo architecture:

```text
01 General public services
02 Defence
03 Public order and safety
04 Economic affairs
05 Environmental protection
06 Housing and community amenities
07 Health
08 Recreation, culture and religion
09 Education
10 Social protection
```

## Reference 2 — IMF GFSM 2014

Official eLibrary page:

```text
https://www.elibrary.imf.org/display/book/9781498343763/ch001.xml
```

Official PDF URL from search result:

```text
https://www.imf.org/external/pubs/ft/gfs/manual/2014/gfsfinal.pdf
```

Verified behavior:

- eLibrary page extracted successfully.
- Direct HEAD to PDF returned HTTP 403 HTML in this environment.

Relevant finding:

- Search/extraction confirms GFSM 2014 is the IMF government finance statistics framework and that Chapter 6 describes COFOG / functions of government.
- Treat IMF GFSM 2014 as the conceptual/statistical-system reference, with Eurostat 2019 as the practical compilation/manual reference for this gate.

## Reference 3 — U.S. federal budget functions are not COFOG

Reference page extracted:

```text
https://democrats-budget.house.gov/budgets/budget-functions
```

Observed federal budget functions include:

```text
050 National Defense
150 International Affairs
250 General Science, Space, and Technology
270 Energy
300 Natural Resources and Environment
350 Agriculture
370 Commerce and Housing Credit
400 Transportation
450 Community and Regional Development
500 Education, Training, and Social Services
550 Health
570 Medicare
600 Income Security
650 Social Security
700 Veterans Benefits and Services
750 Administration of Justice
800 General Government
900 Net Interest
920 Allowances
950 Undistributed Offsetting Receipts
970 Overseas Deployments
```

Finding:

These U.S. budget functions are useful but not COFOG. Some map naturally at high level, but a published official crosswalk was not verified. Outlays should not silently treat federal budget functions as COFOG codes.

Example likely relationships, not official crosswalk:

- `050 National Defense` → COFOG `02 Defence`.
- `550 Health` and `570 Medicare` → COFOG `07 Health`.
- `500 Education, Training, and Social Services` spans COFOG `09 Education` and `10 Social protection`.
- `370 Commerce and Housing Credit` spans COFOG `04 Economic affairs` and `06 Housing and community amenities`.

## Reference 4 — California procurement source categories

Outlays adapter source:

```text
packages/adapters/us-ca-procurement/fixtures/replay/cfd1dbf87597497e.bin
```

Fixture provenance from repo:

```text
data.ca.gov Purchase Order Data
resource bb82edc5-9c78-44e2-8947-68ece26197c5
```

Observed fixture sample:

- records sampled: 1,000.
- total source query count: 115,969.

Observed acquisition types:

```text
NON-IT Goods
IT Goods
NON-IT Services
IT Services
IT Telecommunications
```

Top observed departments in the fixture sample:

```text
Corrections and Rehabilitation, Department of
Consumer Affairs, Department of
Correctional Health Care Services
Transportation, Department of
Water Resources, Department of
Fish and Wildlife, Department of
Forestry and Fire Protection, Department of
State Hospitals, Department of
General Services, Department of
Veterans Affairs, Department of
Franchise Tax Board
Parks & Recreation, Department of
Industrial Relations, Department of
Developmental Services, Department of
Statewide Health Planning & Development, Office of
Water Resources Control Board, State
Pesticide Regulation, Department of
Conservation Corps, California
Military Department
Air Resources Board
Health & Human Services Agency
Highway Patrol, California
Food and Agriculture, Department of
Motor Vehicles, Department of
Employment Development Department
```

## Mapping policy

Use COFOG mapping only when the source category encodes function.

Acquisition type usually encodes input/procurement modality, not purpose. Therefore:

- `IT Goods` does not imply general public services.
- `NON-IT Services` does not imply economic affairs.
- `IT Telecommunications` does not imply communications policy.

Those stay `unmapped` until joined with department/program/commodity context.

Department names can support draft COFOG mapping, but confidence must stay modest unless the department is clearly single-function.

## Gate 6 closeout

Gate 6 status: closed with conservative mapping.

- Eurostat COFOG 2019: verified.
- IMF GFSM 2014: official eLibrary verified; direct PDF HEAD blocked/403.
- Official U.S. federal-budget-function to COFOG crosswalk: not verified.
- California procurement mapping: created as a draft, with acquisition types intentionally unmapped.
