# Publication practices

## Addendum gate

A3 — how ProPublica, OpenSecrets, USAFacts, and MuckRock caveat, correct, and remove entity-level money data, plus an Outlays draft disclaimer and correction policy.

## EVR status

### Execute

Reviewed official policy, contact, data, API, terms, privacy, and organizational pages for:

- ProPublica / Nonprofit Explorer;
- OpenSecrets;
- USAFacts;
- MuckRock.

### Verify

For each organization, captured:

- public caveat / methodology posture;
- correction channel or correction-adjacent practice;
- removal / takedown / privacy posture;
- implications for Outlays entity-level money data.

Important limitation:

The reviewed pages do not all publish a dedicated entity-level-money-data correction/removal policy. Where a public policy was not found, this document says so directly.

### Report

This document is the addendum report.

## Executive verdict

Outlays should publish entity-level money data like an accountability newsroom, not like a black-box scoring company.

Pattern across mature public-interest data publishers:

1. Cite primary sources and distinguish source facts from analysis.
2. Expose methodology, gaps, and update limits.
3. Provide a correction channel.
4. Do not silently delete public-record facts merely because they are embarrassing.
5. Do remove or restrict genuinely sensitive, unlawful, copyrighted, or mispublished material.
6. Treat entity-level allegations as high-risk publication, especially when algorithmic flags are involved.

## Comparative practices

| Organization | Caveat / methodology practice | Correction practice | Removal / takedown practice | Relevant citation | Outlays implication |
|---|---|---|---|---|---|
| ProPublica | Publishes investigative databases with source notes and API caveats. Nonprofit Explorer documents IRS source inputs, missing 990-N scope, rate limits, and API instability. | Public correction channel exists. Contact page lists `corrections@propublica.org`. | Legal page provides DMCA/copyright agent; privacy policy says users can visit without identifying themselves and gives legal contact. Data terms restrict raw redistribution and disclaim completeness. | Contact page: `Corrections: Email corrections@propublica.org.` Nonprofit Explorer API: `Small organizations filing Form 990-N "e-Postcards" are not included.` Data terms: `We do not guarantee the accuracy or completeness of the data. You acknowledge that the data may contain errors and omissions.` | Outlays needs a visible correction email/form and dataset caveats on every source-backed page. Raw data completeness must never be implied. |
| OpenSecrets | Publishes money-in-politics data with source framing and licensing terms. Privacy policy states most data comes from official records such as FEC, Senate Office of Public Records, IRS, and state agencies. | Users may request to view, amend, or delete account information. For public contribution records, users must correct/refund through the PAC/campaign/source system, not by asking OpenSecrets to delete facts. | Privacy policy says OpenSecrets does not delete contribution data from the site because political contribution data is public record by law and deletion would undermine public transparency. | Privacy policy summary: `Most data is sourced from official government records, including: FEC ... Senate Office of Public Records ... IRS ... State Agencies.` It also states OpenSecrets `do not delete contribution data from their site, as it would undermine their mission of public transparency.` | Outlays should correct its own errors, but source-record disputes should route to the official source agency. Public-record facts should not be removed solely on request. |
| USAFacts | Uses only publicly available government data, emphasizes source transparency, footnotes, revisions, and data gaps. | No dedicated public corrections policy found in reviewed pages. Contact/privacy pages provide communication channels and data-subject rights for personal information. | Privacy policy provides personal-information rights including know/delete/correct/opt out, but that is user-submitted/site-tracking data, not government money data removal. | Our Data page: `USAFacts exclusively uses publicly available government data.` It says USAFacts `explicitly note limitations, gaps, or delays in data` and that `Footnotes and revisions are viewed as signs of excellent, transparent data.` | Outlays should normalize visible caveats, revision notes, and source-health annotations. Silence around limitations is worse than ugly footnotes. |
| MuckRock | FOIA/public-records platform. Terms emphasize user responsibility, public indexing, no defamatory requests, and no sensitive/personal-information abuse. | No single public corrections policy found in reviewed pages. TOS governs user conduct and complaint pathways. | TOS says MuckRock may remove content after complaints of illegality or infringement. It also prohibits harassment, publishing private information, and sensitive/personal information misuse. Historical reporting shows MuckRock may remove records under court order while publicly documenting the dispute. | TOS: `Requests must not contain unfounded allegations or defamatory material.` TOS: `MuckRock reserves the right to remove content upon receiving complaints of illegality or infringement.` May 2016 archive summary: a court TRO forced removal of specific documents and MuckRock characterized the demand as a free-speech attack. | Outlays needs a takedown pathway for illegal, sensitive, or infringing material, but should preserve transparency around removed/withheld records when legally safe. |

## Practices to copy

### 1. Separate source facts from Outlays analysis

Every entity-level page should have two layers:

- Source record facts: amounts, dates, agencies, recipient/vendor, award IDs, URLs, source file/hash.
- Outlays analysis: flags, comparisons, concentration metrics, anomaly labels, methodology version.

Publication rule:

Never let a lead label look like an official government finding.

Required language:

- `Source record`
- `Outlays analysis`
- `Not an allegation`
- `Requires human review before publication as investigative claim`

### 2. Publish caveats beside the data

Copy the USAFacts/ProPublica pattern: caveats live with the product, not buried in a PDF.

Every source-backed view should show:

- source name and URL;
- retrieval date;
- refresh cadence if known;
- coverage window;
- known omissions;
- amount basis, such as obligation, expenditure, outlay, contract value, award value, or payment;
- whether values are deduplicated;
- methodology version;
- correction contact.

### 3. Correct Outlays errors quickly and visibly

Corrections should produce a visible change log.

Correction categories:

1. Source transcription error: Outlays parsed or copied the source incorrectly.
2. Methodology error: Outlays rule, join, dedupe, or normalization was wrong.
3. Source update: official source changed after publication.
4. Context update: source fact remains correct, but caveat/context was incomplete.
5. Publication-risk removal: data remains internally retained, but public display is removed or redacted.

### 4. Route official-record disputes to the source agency

If the source record says Vendor X received Award Y, and the entity says the government record is wrong, Outlays should:

- add a dispute note if evidence is credible;
- link to correction instructions/source agency when available;
- refresh after the official source changes;
- avoid silently editing official-record facts to match an email claim.

This mirrors the OpenSecrets posture on campaign contribution data.

### 5. Maintain a takedown path, but keep it narrow

Outlays should remove or restrict public display when:

- publication exposes legally protected sensitive data;
- data was obtained or published unlawfully;
- source terms prohibit redistribution;
- data creates credible safety risk not outweighed by public interest;
- content is defamatory because Outlays added a false claim;
- copyright/DMCA complaint is valid;
- a court order requires removal.

Outlays should generally not remove public-record facts merely because:

- an entity dislikes appearing in the database;
- an award, grant, payment, or filing is embarrassing;
- the record is old but still public;
- the source fact is accurate and lawfully public.

## Outlays draft disclaimer

DRAFT PENDING HUMAN LEGAL REVIEW

Outlays publishes public-source government spending and related entity records to improve transparency. Records may come from federal, state, local, nonprofit, audit, procurement, budget, and public-authority data sources. Source systems differ in coverage, update frequency, definitions, identifiers, and quality controls.

Amounts shown may represent obligations, award values, contract values, expenditures, outlays, payments, budget authority, or reported totals depending on the source. These measures are not interchangeable. Outlays labels each amount with its source and basis when known.

Outlays analysis, including anomaly indicators, concentration measures, red-flag screens, rankings, and lead labels, is not an allegation of fraud, waste, abuse, illegality, or intent. These indicators identify records that may warrant further review. They should not be treated as official findings, legal conclusions, or proof of misconduct.

Outlays may contain errors caused by source-data errors, delayed updates, parsing defects, entity-resolution mistakes, duplicate records, missing records, or methodology limitations. Users should inspect the linked source records before relying on any result.

Outlays does not represent or speak for any government agency, data publisher, vendor, grantee, nonprofit, public official, or other entity appearing in the data.

To report an error, missing caveat, source update, privacy concern, or legal issue, contact: [INSERT CORRECTION CONTACT].

## Outlays draft correction policy

DRAFT PENDING HUMAN LEGAL REVIEW

### 1. Scope

This policy applies to public Outlays pages, data exports, API responses, lead displays, source documentation, and methodology documentation.

### 2. How to request a correction

A correction request should include:

- requester name and contact information;
- affected URL, record ID, award ID, transaction ID, filing ID, or source link;
- specific statement or data field alleged to be wrong;
- explanation of the error;
- supporting evidence, preferably an official source record or correction notice;
- whether the request seeks correction, contextual note, redaction, or removal.

Outlays should acknowledge receipt within 5 business days when contact information is provided.

### 3. Review categories

Outlays will classify requests as one or more of:

- source transcription/parsing error;
- entity-resolution error;
- methodology or calculation error;
- stale source update;
- duplicate or dedupe issue;
- missing caveat/context;
- privacy/safety concern;
- copyright/legal concern;
- official-record dispute.

### 4. Correction outcomes

Possible outcomes:

1. Correct Outlays data or display.
2. Add source/context caveat.
3. Add disputed-record note.
4. Refresh from official source.
5. Hide, redact, or remove public display.
6. Decline request with explanation.
7. Escalate to human legal/editorial review.

### 5. Official-record disputes

If a request disputes an official public record but Outlays copied the source correctly, Outlays may add a dispute note and link to official source-correction procedures. Outlays should not alter source facts unless the official source changes or the discrepancy is independently verified from another authoritative source.

### 6. Lead/anomaly corrections

If a correction affects a published lead or anomaly:

- recompute the lead;
- preserve the methodology version used before and after correction;
- remove the lead if it no longer satisfies publication gates;
- update the change log;
- avoid implying the original lead was misconduct.

### 7. Removal standard

Outlays may remove or restrict public access when continued publication creates legal, privacy, safety, licensing, or defamation risk. Removal decisions should preserve an internal audit trail unless legally prohibited.

Public pages may state:

`This record was removed or restricted after review. Reason category: [privacy/legal/licensing/source error/safety/court order].`

Do not publish details that would worsen the harm.

### 8. Change log

Material corrections should be logged with:

- date;
- affected record/page;
- correction category;
- previous value or description, unless unsafe/legal-restricted;
- corrected value or action;
- source/evidence;
- reviewer.

### 9. Human review gates

Human review is required before:

- publishing any lead that names a person;
- publishing any fraud/waste/abuse-adjacent label;
- denying a credible privacy/safety/legal removal request;
- rejecting a correction request from a named entity with official evidence;
- publishing a disputed-record note involving possible defamation;
- responding to legal threats, subpoenas, or court orders.

## Product copy patterns

### Safe lead language

Use:

- `Records matching review criteria`
- `Potential anomaly requiring review`
- `High concentration relative to comparable agencies`
- `Single-bid procurement indicator`
- `Source record dispute reported`

Avoid:

- `fraud`
- `corruption`
- `bid rigging`
- `kickback`
- `shell company`
- `illegal`
- `intentional`
- `stolen`
- `wasteful`, unless quoting an official finding

### Correction page copy

DRAFT PENDING HUMAN LEGAL REVIEW

Outlays corrects errors in its own data processing, methodology, and presentation. Some records are copied from official public sources. If the official source is wrong, please include evidence and, where possible, request correction from the source agency or filing system. Outlays may add a dispute note while the official record remains unchanged.

### Removal request copy

DRAFT PENDING HUMAN LEGAL REVIEW

Outlays generally does not remove lawfully public source records solely because they are inconvenient or unfavorable. We review removal or redaction requests involving safety, privacy, legal restrictions, source-publisher terms, copyright, court orders, or false statements added by Outlays.

## Implementation checklist

Before public launch:

- [ ] Create correction inbox/form.
- [ ] Add disclaimer to every entity and lead page.
- [ ] Add source and methodology panel to every money-data page.
- [ ] Add correction/change-log table.
- [ ] Add `disputed_record` and `removed_public_display` states to records.
- [ ] Add staff-only audit trail for correction/removal decisions.
- [ ] Add human legal/editorial review workflow for high-risk decisions.
- [ ] Add source-specific caveat snippets from `docs/licensing-matrix.md`.

## Open questions for legal/editorial review

1. What response SLA should Outlays promise, if any?
2. What categories of individual-level records should be excluded by default?
3. Should public pages show removal reason categories, or only an internal audit trail?
4. What level of evidence is required to add a disputed-record note?
5. Who has authority to reject legal/privacy removal requests?
6. How should Outlays handle source data that is public but likely harmful at individual scale?
