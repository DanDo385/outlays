// Hashing for the adapter pipeline.
//   rawHash   = SHA-256 over the exact upstream bytes (captured before parsing).
//   factHash  = SHA-256 over RFC 8785 (JCS) canonical JSON of a fact's content.
//   resultHash = SHA-256 over JCS of the facts (sorted by factHash), stripped of volatile
//                fields. JSON.stringify is forbidden for hashing (Hard Rule 3).
//
// The strip+sort rule MUST stay byte-identical to the Python SDK and the Go conformance
// harness so resultHash is reproducible across languages.

import { createHash } from "node:crypto";
import _canonicalize from "canonicalize";

// canonicalize is CJS; normalize the interop default under NodeNext and re-type as a
// callable (the dual CJS-ESM typings lose the call signature).
const canonicalize = ((_canonicalize as { default?: unknown }).default ?? _canonicalize) as (
  value: unknown,
) => string | undefined;

/** Volatile fact fields excluded from resultHash (DB-assigned, non-deterministic). */
export const VOLATILE_FACT_FIELDS = ["factId", "runId", "insertedAt"] as const;

/** Fact fields excluded from factHash (identity = monetary content + provenance). */
const FACT_HASH_EXCLUDE = ["factId", "runId", "insertedAt", "factHash", "assignments"] as const;

/** SHA-256 hex digest over raw bytes or a UTF-8 string. */
export function sha256Hex(data: Uint8Array | string): string {
  return createHash("sha256").update(data).digest("hex");
}

/** RFC 8785 (JCS) canonical JSON string. Throws on values JCS cannot represent. */
export function jcs(value: unknown): string {
  const s = canonicalize(value);
  if (s === undefined) {
    throw new Error("value is not JCS-serializable");
  }
  return s;
}

/** SHA-256 hex over the JCS canonical form of a value. */
export function jcsSha256(value: unknown): string {
  return sha256Hex(jcs(value));
}

function omit<T extends Record<string, unknown>>(obj: T, keys: readonly string[]): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(obj)) {
    if (!keys.includes(k) && v !== undefined) out[k] = v;
  }
  return out;
}

/** Deterministic content hash of a single fact (excludes volatile + assignment fields). */
export function computeFactHash(fact: Record<string, unknown>): string {
  return jcsSha256(omit(fact, FACT_HASH_EXCLUDE));
}

/**
 * Deterministic hash over a fact set: drop volatile fields, sort by factHash ascending, then
 * JCS + SHA-256. Independent of the order the adapter emitted facts in.
 */
export function computeResultHash(facts: Array<Record<string, unknown>>): string {
  const stripped = facts.map((f) => omit(f, VOLATILE_FACT_FIELDS));
  stripped.sort((a, b) => {
    const fa = String(a["factHash"] ?? "");
    const fb = String(b["factHash"] ?? "");
    return fa < fb ? -1 : fa > fb ? 1 : 0;
  });
  return jcsSha256(stripped);
}
