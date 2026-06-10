# Federated submission verification

## Gate

Gate 10 — federated submission verification.

Question:

Can Outlays accept community/partner adapter outputs without trusting the submitter blindly?

Answer:

Yes, but only if verification means re-derivation and schema/hash checks first, signatures second. A signature proves who submitted an envelope; it does not prove the data is true.

## EVR status

### Execute

Inspected repo implementation for:

- adapter conformance harness,
- `resultHash` recomputation,
- raw snapshot validation,
- signature fields in SDK output,
- provenance queries,
- schema validation,
- append-only persistence.

### Verify

Verified repo evidence:

- `core/internal/conformance/conformance.go` recomputes `resultHash` from emitted facts and checks it matches `envelope.resultHash`.
- Conformance checks declared raw snapshots against files in the raw directory.
- `packages/adapter-sdk-ts/src/hash.ts` defines deterministic hashing:
  - raw hash = SHA-256 over exact upstream bytes,
  - fact hash = SHA-256 over canonical JSON of fact content,
  - result hash = SHA-256 over facts sorted by fact hash with volatile fields stripped.
- TS adapter SDK currently emits:
  - `signature: null`,
  - `signerKeyId: null`.
- `ARCHITECTURE.md` says `IngestionEnvelope` already carries `signature` / `signerKeyId` for Phase-2 federated submissions, but trusted orchestrator runs first.

Not verified:

- A real signature verification implementation.
- Public-key registry.
- Revocation model.
- Replay protection.
- Submitter authentication/authorization.

### Report

This file is the Gate 10 feasibility and design note.

## Verification stack

Federated ingestion must verify four layers, in order:

1. Contract validity.
2. Raw snapshot integrity.
3. Deterministic fact/result hashes.
4. Submitter signature and authorization.

Do not reverse the order. A valid signature over invalid data should still fail.

## Current state

Already present:

```text
AdapterOutput schema validation
raw snapshot hash validation
factHash/resultHash deterministic recomputation
conformance harness
signature and signerKeyId fields in envelope
```

Missing:

```text
non-null signature creation
signature verification
key registry
submitter identity model
replay protection
submission review/acceptance workflow
```

## Threat model

### Threat 1 — bad submitter signs bad data

Mitigation:

- Recompute `resultHash` from facts.
- Verify every fact with transaction/award grain has `rawSha256`.
- Verify every `rawSha256` exists in declared raw snapshots.
- Verify raw snapshot bytes match SHA-256.
- Optionally re-fetch source URLs for spot checks.

Signature alone is insufficient.

### Threat 2 — replayed old envelope

Mitigation:

- Include `source`, `jurisdiction`, `fiscalYear`, `adapterId`, `adapterVersion`, `createdAt`, and `resultHash` in signed payload.
- Store accepted `(signerKeyId, resultHash)` and `(source, fiscalYear, resultHash)`.
- Accept duplicates as idempotent only if byte-identical; otherwise create a superseding/review path.

### Threat 3 — malicious adapter emits plausible but fabricated raw snapshots

Mitigation:

- Store raw bytes.
- Preserve source URL and HTTP metadata when available.
- Require `derivationQuery` to point from facts back into raw snapshot.
- For high-impact submissions, run an independent re-fetch/re-derive job.

### Threat 4 — key compromise

Mitigation:

- Key registry supports statuses:
  - `active`,
  - `revoked`,
  - `expired`,
  - `quarantined`.
- Every accepted run stores signer key fingerprint and verification result.
- Revocation does not delete accepted facts; it flags future trust/review state.

### Threat 5 — schema drift between SDKs

Mitigation:

- Keep drift guard and conformance harness required for community adapters.
- Require adapter output contract version in envelope.
- Reject unknown contract versions unless explicitly allowed.

## Recommended signed payload

Sign canonical JSON over:

```json
{
  "contractVersion": "...",
  "adapterId": "...",
  "adapterVersion": "...",
  "source": "...",
  "jurisdiction": "...",
  "fiscalYear": "...",
  "resultHash": "...",
  "rawSnapshots": [
    {"sha256":"...","bytes":123,"url":"..."}
  ],
  "createdAt": "..."
}
```

Do not sign pretty JSON. Sign RFC 8785/JCS canonical bytes, matching the repo's existing canonical-hash discipline.

Recommended algorithm:

```text
Ed25519 over JCS canonical signed-payload
```

Why Ed25519:

- deterministic,
- simple public-key model,
- fast,
- widely supported.

## Submitter registry

Minimum table:

```text
submitter_key
  key_id
  public_key
  algorithm
  owner_name
  owner_contact
  status
  created_at
  expires_at
  revoked_at
  note
```

Minimum accepted-run fields:

```text
run_id
submitter_key_id
signature
signature_payload_sha256
verification_status
verification_errors
accepted_by
accepted_at
```

## Acceptance workflow

Recommended states:

```text
received -> contract_validated -> hashes_verified -> signature_verified -> accepted | rejected | quarantined
```

Append-only event model:

```text
submission_event
  submission_id
  event_type
  status_after
  actor
  note
  created_at
```

## Public trust label

Public provenance should expose a plain-English trust label:

```text
Source collected by Outlays orchestrator
Community submission, independently re-derived by Outlays
Community submission, signature/hash verified only
Community submission, quarantined/not public
```

Do not present `signature verified` as equivalent to `source verified`.

## Minimum viable Gate 10 implementation

Build a CLI before web workflows:

```text
outlays submissions verify envelope.json --raw-dir ./raw --keyring ./keys.json
```

It should:

1. Validate schema.
2. Verify raw snapshot hashes.
3. Recompute fact hashes/result hash.
4. Verify signed payload if `signature` and `signerKeyId` are non-null.
5. Print a machine-readable verification report.
6. Exit non-zero on failure.

## Gate 10 closeout

Gate 10 status: closed.

Verdict:

Federated submissions are feasible because Outlays already has the hard part: deterministic result hashes and raw snapshot provenance. The missing part is key management and signature verification. Build signature verification as an extra trust layer, not as a replacement for re-derivation.
