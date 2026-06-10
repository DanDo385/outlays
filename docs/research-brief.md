# Research Brief: Federal Bulk Data Gates

## Status

- Branch: `research/gates`
- Scope: deliverable-first research notes for the Outlays bulk census sprint.
- Current gate state: Gate 1 report landed; Gate 2 and Gate 3 are next and must keep EIN linkage prominent.

## Gate 1 landed: USAspending bulk archives

Deliverable:

- `docs/sources/us-fed-bulk.md`

Blunt finding:

- USAspending FY2024 bulk archives are the largest verified first prize: prime contracts, prime assistance, generated subawards, account downloads, and database snapshots are live surfaces.
- Prime award archives expose UEI and legacy DUNS fields, but EIN was not observed in the inspected contract archive header.
- Account downloads are required for official account/control/classification linkage, especially object class, program activity, award-financial/File C style linkage, and account balances/File A control totals.

Open Gate 1 gaps preserved in the source note:

- explicit license/terms confirmation,
- exact dictionary parity,
- full FY subaward ZIP size/status,
- full FY all-agency account download counts/sizes,
- `award_financial` account download probe,
- `account_balances` account download probe.

## Gate 2 next: nonprofit data stack

Required deliverable:

- `docs/sources/nonprofit-stack.md`

Primary question:

- Can a federal or state grant record be joined to a nonprofit's IRS Form 990 and/or Federal Audit Clearinghouse record by verified identifiers?

EIN linkage must be carried prominently:

- IRS Form 990 data is expected to be EIN-centered.
- Federal Audit Clearinghouse single-audit data is expected to include auditee identifiers, with EIN presence requiring live verification.
- USAspending Gate 1 did not confirm EIN in the inspected prime contract archive header, so Gate 2 must explicitly test whether assistance/subaward records expose EIN or whether UEI-to-EIN linkage requires SAM.gov, recipient profiles, FAC, IRS, or a separate resolution layer.

Minimum Gate 2 output:

- verified bulk access path for IRS Form 990 e-file data,
- verified ProPublica Nonprofit Explorer API status/terms/rate limits,
- verified Federal Audit Clearinghouse bulk data path and fields,
- join-key map covering EIN, UEI, recipient names, addresses, and known gaps,
- one worked example traced by hand from a real grant to one real organization's 990.

## Gate 3 next: entity identifier landscape

Required deliverable:

- `docs/entity-identifiers.md`

Primary question:

- What identifier-first matching policy should Outlays use across UEI, EIN, DUNS, names, addresses, and government-published crosswalks?

EIN linkage must remain central:

- UEI is the apparent first-class identifier in USAspending prime award surfaces.
- EIN is the apparent first-class identifier for nonprofit tax/audit surfaces.
- The key architectural risk is not downloading more files; it is proving or rejecting an official UEI-to-EIN bridge.
- If no official UEI/EIN linkage is public or usable, the research must say so and recommend an append-only alias policy with confidence levels, not destructive merges.

Minimum Gate 3 output:

- SAM.gov entity extract availability and access requirements,
- EIN public availability and constraints,
- legacy DUNS status,
- any official UEI-EIN linkage found or explicitly absent,
- recommended identifier-first matching policy aligned with `ARCHITECTURE.md` append-only alias rules.

## Operating rule for next work

Do not optimize ingestion code before Gate 2 and Gate 3 resolve the identifier story. A beautiful pipeline that cannot join UEI-centered awards to EIN-centered nonprofit records is a fast machine pointed at a fog bank.
