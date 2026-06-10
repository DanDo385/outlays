// Deterministic UUIDv5 (RFC 4122 §4.3): SHA-1 over namespace bytes + name bytes. Used to mint
// stable, reproducible provisional entity ids from a normalized name, so the same vendor maps
// to the same id across runs without any central resolution step (NO fuzzy merging — exact
// normalized-name match only).

import { createHash } from "node:crypto";

function parseUuid(uuid: string): Buffer {
  const hex = uuid.replace(/-/g, "");
  if (hex.length !== 32 || /[^0-9a-fA-F]/.test(hex)) {
    throw new Error(`invalid namespace UUID: ${uuid}`);
  }
  return Buffer.from(hex, "hex");
}

function formatUuid(bytes: Buffer): string {
  const h = bytes.subarray(0, 16).toString("hex");
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20, 32)}`;
}

/** RFC 4122 v5 UUID for `name` within `namespace` (a UUID string). */
export function uuidv5(name: string, namespace: string): string {
  const ns = parseUuid(namespace);
  const digest = createHash("sha1").update(Buffer.concat([ns, Buffer.from(name, "utf8")])).digest();
  const bytes = Buffer.from(digest.subarray(0, 16));
  bytes[6] = (bytes[6]! & 0x0f) | 0x50; // version 5
  bytes[8] = (bytes[8]! & 0x3f) | 0x80; // RFC 4122 variant
  return formatUuid(bytes);
}
