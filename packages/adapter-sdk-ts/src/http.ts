// HTTP GET with three modes (selected by env), so the same adapter code does live fetching,
// fixture recording, and offline replay:
//
//   OUTLAYS_REPLAY_DIR  — read responses from a recorded fixture dir (no network). CI uses
//                         this; a missing URL is a hard error.
//   OUTLAYS_RECORD_DIR  — live fetch, then persist each response into a fixture dir + index.
//   (neither)           — live fetch with project User-Agent, per-host rate limiting, and
//                         exponential backoff with jitter honoring 429/5xx + Retry-After.
//
// Bytes are always captured exactly as received, before parsing (Hard Rule 3).

import { mkdir, readFile, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { sha256Hex } from "./hash.js";

const DEFAULT_UA = "outlays/0.1.0 (+https://github.com/DanDo385/outlay)";

export interface HttpResponse {
  url: string;
  bytes: Uint8Array;
  httpStatus: number;
  headers: Record<string, string>;
}

interface ReplayIndex {
  entries: Record<string, { file: string; status: number }>;
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const lastRequestByHost = new Map<string, number>();

function userAgent(): string {
  return process.env["OUTLAYS_USER_AGENT"] ?? DEFAULT_UA;
}

async function rateLimit(host: string): Promise<void> {
  const rps = Number(process.env["OUTLAYS_RATE_LIMIT_PER_HOST_RPS"] ?? "1") || 1;
  const minGapMs = 1000 / rps;
  const last = lastRequestByHost.get(host) ?? 0;
  const wait = last + minGapMs - Date.now();
  if (wait > 0) await sleep(wait);
  lastRequestByHost.set(host, Date.now());
}

async function loadIndex(dir: string): Promise<ReplayIndex> {
  const path = join(dir, "index.json");
  if (!existsSync(path)) return { entries: {} };
  return JSON.parse(await readFile(path, "utf8")) as ReplayIndex;
}

async function replay(dir: string, url: string): Promise<HttpResponse> {
  const index = await loadIndex(dir);
  const entry = index.entries[url];
  if (!entry) {
    throw new Error(`replay miss: no recorded response for ${url} in ${dir}`);
  }
  const bytes = await readFile(join(dir, entry.file));
  return { url, bytes, httpStatus: entry.status, headers: {} };
}

async function liveFetch(url: string, headers: Record<string, string>): Promise<HttpResponse> {
  const host = new URL(url).host;
  const maxAttempts = 6;
  let lastErr: unknown;
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    await rateLimit(host);
    try {
      const res = await fetch(url, { headers: { "user-agent": userAgent(), ...headers } });
      if (res.status === 429 || res.status >= 500) {
        const retryAfter = Number(res.headers.get("retry-after"));
        const backoff = Number.isFinite(retryAfter) && retryAfter > 0
          ? retryAfter * 1000
          : Math.min(30000, 500 * 2 ** attempt) + Math.random() * 250;
        await sleep(backoff);
        continue;
      }
      const buf = new Uint8Array(await res.arrayBuffer());
      const hdrs: Record<string, string> = {};
      res.headers.forEach((v, k) => {
        hdrs[k] = v;
      });
      return { url, bytes: buf, httpStatus: res.status, headers: hdrs };
    } catch (err) {
      lastErr = err;
      await sleep(Math.min(30000, 500 * 2 ** attempt) + Math.random() * 250);
    }
  }
  throw new Error(`fetch failed after ${maxAttempts} attempts: ${url} (${String(lastErr)})`);
}

async function record(dir: string, url: string, headers: Record<string, string>): Promise<HttpResponse> {
  const res = await liveFetch(url, headers);
  await mkdir(dir, { recursive: true });
  const file = `${sha256Hex(url).slice(0, 16)}.bin`;
  await writeFile(join(dir, file), res.bytes);
  const index = await loadIndex(dir);
  index.entries[url] = { file, status: res.httpStatus };
  await writeFile(join(dir, "index.json"), JSON.stringify(index, null, 2) + "\n");
  return res;
}

/** GET `url`, honoring the active mode (replay / record / live). */
export async function httpGet(url: string, headers: Record<string, string> = {}): Promise<HttpResponse> {
  const replayDir = process.env["OUTLAYS_REPLAY_DIR"];
  if (replayDir) return replay(replayDir, url);
  const recordDir = process.env["OUTLAYS_RECORD_DIR"];
  if (recordDir) return record(recordDir, url, headers);
  return liveFetch(url, headers);
}
