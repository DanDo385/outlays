# zkTLS and fetch-proof feasibility

## Gate

Gate 11 — zkTLS fetch proofs.

Question:

Should Outlays add zkTLS/web-proof infrastructure to prove fetched data came from public or authenticated web sources?

Answer:

Not in the core ingestion path yet.

For public government bulk files and APIs, Outlays gets more trust per engineering hour from:

1. raw byte snapshots,
2. SHA-256 hashes,
3. stored HTTP metadata,
4. deterministic derivation queries,
5. adapter result hashes,
6. optional signatures/federated verification,
7. optional timestamp anchoring.

zkTLS belongs later as a specialized plugin for authenticated/private portals, user-consented data proofs, or adversarial evidence packages where ordinary source hashes are not enough.

## EVR status

### Execute

Verified current ecosystem references:

- TLSNotary official site and browser-extension documentation.
- Reclaim Protocol docs and zkTLS explainer.
- OpenTimestamps client repository/documentation summary.
- Repo architecture for raw snapshots, hashes, resultHash, and anchor roadmap.

### Verify

Verified from TLSNotary docs:

- TLSNotary is open source.
- It lets users prove facts about web data to third parties.
- It is built around browser extension/plugin/verifier/proxy architecture.
- Browser extension uses application-specific JavaScript plugins.
- Sensitive authentication data is intended to stay under user control.
- It works through existing authenticated browser sessions.

Verified from Reclaim docs:

- Reclaim Protocol uses zkTLS to verify user information from websites.
- Reclaim supports public API verification through zkFetch SDK according to docs.
- It uses regex/JSON path style extraction of specific data points.

Verified from Reclaim explainer:

- `zkTLS` is overloaded terminology.
- The critical property is integrity/provenance of HTTPS responses, not always privacy.
- Models include TEE-based, MPC-based, and proxy-based approaches with different trust assumptions.

Verified from OpenTimestamps repository summary:

- OpenTimestamps creates and validates timestamp proofs using Bitcoin blockchain calendars.
- It proves file existence before a time; it does not prove source authenticity by itself.

Not verified:

- A working TLSNotary/Reclaim proof generated against a government data source.
- Production ergonomics for headless bulk ingestion.
- Long-term verifier availability, proof format stability, or integration cost.
- Whether public government APIs would rate-limit, block, or alter behavior under proof tooling.

### Report

This file is the Gate 11 feasibility report.

## What zkTLS proves

Useful claim:

```text
This response was received from this HTTPS origin/session, and selected fields match this proof.
```

What it does not automatically prove:

```text
The government data is complete.
The server's database was correct.
The endpoint is canonical.
The extractor selected the right fields.
The source license permits redistribution.
The entity match is true.
```

Tiny robot siren: provenance is not omniscience. Beep.

## Outlays current proof stack

Current/repo proof primitives:

```text
raw_snapshot.sha256
fiscal_fact.raw_sha256
fiscal_fact.derivation_query
fact_hash
resultHash
conformance recomputation
append-only storage
future anchor layer
```

This already proves:

- exactly which bytes Outlays stored,
- exactly which query/row produced a fact,
- whether facts changed across runs,
- whether SDK implementations agree on result hashes,
- whether persisted facts remain immutable.

For public bulk files, that is the right center of gravity.

## Where zkTLS helps

zkTLS is most useful for sources with at least one of these traits:

1. Authenticated user sessions.
2. No public bulk download.
3. Dynamic pages where the server does not publish stable archives.
4. Evidence packages where an outside verifier needs assurance that a user-visible value came from a site.
5. Community submissions from portals Outlays cannot directly access.
6. Sensitive fields where selective disclosure matters.

Examples:

- a local government procurement portal behind login,
- a grantee portal visible only to an applicant,
- a user-consented proof of a 990 filing dashboard value,
- a one-off web page that may disappear before official archive capture.

## Where zkTLS is overkill

zkTLS is overkill for:

- USAspending official bulk ZIPs,
- IRS EO BMF public CSVs,
- NPPES public monthly ZIPs,
- Socrata public CSV/JSON exports,
- data.ca.gov CKAN resources,
- public PDFs with stored bytes and hashes.

Why:

- The source is already public.
- Outlays can store exact bytes.
- Reproducibility comes from hash + query + source URL + timestamp.
- zkTLS proof generation would add operational fragility without solving completeness or semantic correctness.

## OpenTimestamps vs zkTLS

OpenTimestamps answers:

```text
Did this byte string exist before this time?
```

zkTLS answers:

```text
Did this HTTPS response come from this origin/session, with selected disclosed fields?
```

They are complementary.

Recommended now:

- use source hashes everywhere,
- optionally timestamp high-value raw snapshots/result manifests,
- do not require zkTLS for normal ingestion.

## Suggested proof tiers

### Tier 0 — normal public-source provenance

Required for all facts:

```text
raw bytes
raw_sha256
derivation_query
fact_hash
resultHash
HTTP/source metadata where available
```

### Tier 1 — signed/federated submission

For community adapters:

```text
Tier 0 + signed envelope + key registry + re-derivation verification
```

### Tier 2 — timestamped evidence bundle

For high-impact runs:

```text
Tier 1 or Tier 0 + OpenTimestamps/proof-of-existence over result manifest
```

### Tier 3 — zkTLS/web proof

For authenticated or adversarial web capture:

```text
Tier 0 + zkTLS proof + verifier metadata + disclosed field manifest
```

## Integration recommendation

Do not build zkTLS into the orchestrator core.

Instead, design a sidecar proof interface:

```text
fetch_proof
  proof_id
  source_url
  proof_type: none | http_metadata | timestamp | zktls_tlsnotary | zktls_reclaim | other
  proof_payload_sha256
  proof_storage_key
  verifier
  verifier_version
  disclosed_fields_json
  created_at
```

Facts can optionally reference a `proof_id`, but ingestion should remain valid with ordinary raw snapshot provenance.

## Minimal experiment later

When ready, test one narrowly scoped proof:

```text
source: public Socrata JSON endpoint
tool: Reclaim zkFetch or TLSNotary plugin
claim: top-level JSON field equals expected value
output: proof payload + raw response snapshot + Outlays fact derivation
```

Acceptance:

- proof verifies offline or through documented verifier,
- stored raw bytes hash matches derivation,
- proof does not expose secrets,
- result is reproducible enough for a fixture or documented live dependency.

Do not start with SAM, IRS, or authenticated portals. Start boring. Boring survives.

## Product guidance

Avoid saying:

```text
cryptographically proves government spending
```

Say:

```text
Outlays stores source bytes, hashes, derivation queries, and optional fetch proofs. These prove what Outlays observed and how each number was derived; they do not prove the upstream source was complete or correct.
```

## Gate 11 closeout

Gate 11 status: closed.

Verdict:

Keep zkTLS/fetch proofs out of the core product path for now. Add a proof sidecar interface later, after source ingestion, entity linking, coverage, and lead safety are solid. For current roadmap, source hashes + result hashes + optional timestamping are higher-leverage and lower-risk.
