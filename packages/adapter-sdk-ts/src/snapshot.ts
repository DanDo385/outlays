// Raw-snapshot writer: content-addressed by SHA-256 over the exact bytes, with a sidecar
// meta file. Mirrors the object-storage key scheme (ARCHITECTURE.md Section 3); locally it
// writes to a directory, and the orchestrator (S5) uploads to object storage.

import { mkdir, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { sha256Hex } from "./hash.js";

export interface SnapshotInput {
  url: string;
  bytes: Uint8Array;
  httpStatus: number;
  headers?: Record<string, string>;
}

/** A reference to a stored raw snapshot (shape mirrors the contract RawSnapshotRef). */
export interface RawSnapshotRef {
  sha256: string;
  url: string;
  bytes: number;
  httpStatus: number;
}

/**
 * Writes `<sha256>.bin` (exact bytes) and `<sha256>.meta.json` (url, fetchedAt, httpStatus,
 * headers) into `rawDir`, and returns the snapshot reference. The `.bin` is the content
 * address; the meta file is descriptive only and is never hashed.
 */
export async function writeSnapshot(rawDir: string, input: SnapshotInput): Promise<RawSnapshotRef> {
  const sha256 = sha256Hex(input.bytes);
  await mkdir(rawDir, { recursive: true });
  await writeFile(join(rawDir, `${sha256}.bin`), input.bytes);
  const meta = {
    sha256,
    url: input.url,
    fetchedAt: new Date().toISOString(),
    httpStatus: input.httpStatus,
    headers: input.headers ?? {},
  };
  await writeFile(join(rawDir, `${sha256}.meta.json`), JSON.stringify(meta, null, 2) + "\n");
  return { sha256, url: input.url, bytes: input.bytes.byteLength, httpStatus: input.httpStatus };
}
