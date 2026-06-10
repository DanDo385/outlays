// Adapter CLI exit codes (ARCHITECTURE.md Section 4).
export const ExitCode = {
  Success: 0,
  Unexpected: 1,
  SourceUnavailable: 2,
  ContractInvalid: 3,
} as const;

/**
 * Thrown by an adapter when the upstream source is unavailable or restricted. This is a
 * finding, not a crash: the scaffold exits 2.
 */
export class SourceUnavailableError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "SourceUnavailableError";
  }
}
