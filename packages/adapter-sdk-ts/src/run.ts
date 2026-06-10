// CLI scaffold for adapters. A contributor implements only `listYears()` and `fetchYear()`;
// this module provides the `info` / `list-years` / `fetch` commands, raw-snapshot capture,
// hashing, envelope assembly, contract validation, NDJSON stderr logs, and exit codes.

import { randomUUID } from "node:crypto";
import { writeFile } from "node:fs/promises";
import { validate, type FiscalFact, type Entity, type EntityAlias } from "@outlays/contract";
import { ExitCode, SourceUnavailableError } from "./errors.js";
import { computeFactHash, computeResultHash } from "./hash.js";
import { writeSnapshot, type RawSnapshotRef, type SnapshotInput } from "./snapshot.js";

export interface AdapterManifest {
  adapterId: string;
  jurisdiction: string;
  datasets: string[];
  adapterVersion: string;
  contractVersion: string;
  license: string;
  maintainer: string;
}

/** A fact as returned by `fetchYear` — without the hash/DB fields the scaffold fills in. */
export type FactInput = Omit<FiscalFact, "factHash" | "factId" | "runId" | "insertedAt">;

export interface FetchContext {
  readonly year: string;
  readonly rawDir: string;
  /** Capture raw bytes as a content-addressed snapshot; tracked for the envelope. */
  snapshot(input: SnapshotInput): Promise<RawSnapshotRef>;
  /** Structured log line to stderr (NDJSON). */
  log(level: "debug" | "info" | "warn" | "error", msg: string): void;
}

export interface FetchResult {
  facts: FactInput[];
  entities?: Entity[];
  entityAliases?: EntityAlias[];
}

export interface AdapterDefinition {
  manifest: AdapterManifest;
  listYears(): Promise<string[]> | string[];
  fetchYear(ctx: FetchContext): Promise<FetchResult> | FetchResult;
}

const FISCAL_YEAR = /^\d{4}(-\d{2})?$/;

function logLine(level: string, msg: string): void {
  process.stderr.write(JSON.stringify({ level, msg, ts: new Date().toISOString() }) + "\n");
}

function parseFlags(argv: string[]): Record<string, string> {
  const flags: Record<string, string> = {};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a && a.startsWith("--")) {
      const key = a.slice(2);
      const next = argv[i + 1];
      if (next !== undefined && !next.startsWith("--")) {
        flags[key] = next;
        i++;
      } else {
        flags[key] = "true";
      }
    }
  }
  return flags;
}

async function runFetch(def: AdapterDefinition, flags: Record<string, string>): Promise<number> {
  const year = flags["year"];
  const rawDir = flags["raw-dir"];
  const out = flags["out"];

  if (!year || !FISCAL_YEAR.test(year)) {
    logLine("error", `invalid or missing --year (must match ${FISCAL_YEAR})`);
    return ExitCode.Unexpected;
  }
  if (!rawDir) {
    logLine("error", "missing --raw-dir");
    return ExitCode.Unexpected;
  }

  const snapshots: RawSnapshotRef[] = [];
  const ctx: FetchContext = {
    year,
    rawDir,
    async snapshot(input) {
      const ref = await writeSnapshot(rawDir, input);
      snapshots.push(ref);
      return ref;
    },
    log: logLine,
  };

  let result: FetchResult;
  try {
    result = await def.fetchYear(ctx);
  } catch (err) {
    if (err instanceof SourceUnavailableError) {
      logLine("error", `source unavailable: ${err.message}`);
      return ExitCode.SourceUnavailable;
    }
    logLine("error", `fetch failed: ${(err as Error).message}`);
    return ExitCode.Unexpected;
  }

  const facts: FiscalFact[] = result.facts.map((f) => ({
    ...f,
    factHash: computeFactHash(f as unknown as Record<string, unknown>),
  }));

  const envelope = {
    envelopeVersion: "1" as const,
    adapterId: def.manifest.adapterId,
    adapterVersion: def.manifest.adapterVersion,
    runId: randomUUID(),
    fetchedAt: new Date().toISOString(),
    rawSnapshots: snapshots,
    jurisdiction: def.manifest.jurisdiction,
    fiscalYear: year,
    resultHash: computeResultHash(facts as unknown as Array<Record<string, unknown>>),
    signature: null,
    signerKeyId: null,
  };

  const doc = {
    envelope,
    facts,
    ...(result.entities ? { entities: result.entities } : {}),
    ...(result.entityAliases ? { entityAliases: result.entityAliases } : {}),
  };

  const verdict = validate("AdapterOutput", doc);
  if (!verdict.valid) {
    logLine("error", `output failed contract validation: ${verdict.errors.join("; ")}`);
    return ExitCode.ContractInvalid;
  }

  const json = JSON.stringify(doc, null, 2) + "\n";
  if (out && out !== "-") {
    await writeFile(out, json);
  } else {
    process.stdout.write(json);
  }
  logLine("info", `wrote ${facts.length} fact(s), ${snapshots.length} snapshot(s), resultHash=${envelope.resultHash}`);
  return ExitCode.Success;
}

/** Entry point: parse argv, dispatch the adapter command, and exit with the protocol code. */
export async function runAdapter(def: AdapterDefinition, argv: string[] = process.argv.slice(2)): Promise<never> {
  const [cmd, ...rest] = argv;
  let code: number;
  try {
    switch (cmd) {
      case "info":
        process.stdout.write(JSON.stringify(def.manifest, null, 2) + "\n");
        code = ExitCode.Success;
        break;
      case "list-years": {
        const years = [...(await def.listYears())].sort().reverse();
        process.stdout.write(JSON.stringify(years) + "\n");
        code = ExitCode.Success;
        break;
      }
      case "fetch":
        code = await runFetch(def, parseFlags(rest));
        break;
      default:
        logLine("error", `unknown command '${cmd ?? ""}' (expected info | list-years | fetch)`);
        code = ExitCode.Unexpected;
    }
  } catch (err) {
    logLine("error", `unexpected: ${(err as Error).message}`);
    code = ExitCode.Unexpected;
  }
  return process.exit(code);
}
