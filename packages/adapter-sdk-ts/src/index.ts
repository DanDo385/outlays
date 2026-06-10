// @outlays/adapter-sdk — TypeScript adapter SDK.
//
// Provides fetch-with-snapshot, JCS hashing, contract validation, and a CLI scaffold so a
// contributor implements only listYears() and fetchYear() (ARCHITECTURE.md Section 4).

export { ExitCode, SourceUnavailableError } from "./errors.js";
export { sha256Hex, jcs, jcsSha256, computeFactHash, computeResultHash, VOLATILE_FACT_FIELDS } from "./hash.js";
export { writeSnapshot, type SnapshotInput, type RawSnapshotRef } from "./snapshot.js";
export { httpGet, type HttpResponse } from "./http.js";
export { uuidv5 } from "./id.js";
export {
  runAdapter,
  type AdapterDefinition,
  type AdapterManifest,
  type FetchContext,
  type FetchResult,
  type FactInput,
} from "./run.js";

export { ENVELOPE_VERSION, CONTRACT_VERSION } from "@outlays/contract";
