// @outlays/adapter-sdk — TypeScript adapter SDK.
//
// In S2 this provides fetch-with-snapshot, JCS hashing, contract validation, and a CLI
// scaffold so a contributor implements only listYears() and fetchYear(). For S0 it is a
// minimal buildable placeholder.

import { ENVELOPE_VERSION } from "@outlays/contract";

export { ENVELOPE_VERSION };

/** Adapter CLI exit codes (mirrors ARCHITECTURE.md Section 4). */
export const ExitCode = {
  Success: 0,
  Unexpected: 1,
  SourceUnavailable: 2,
  ContractInvalid: 3,
} as const;
