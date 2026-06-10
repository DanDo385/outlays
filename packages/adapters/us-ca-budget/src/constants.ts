// Constants for the California enacted-budget control-total adapter.
//
// Each entry pins one official published total to the exact document that states it. The
// figure is a reviewed in-source constant transcribed from the cited table — never computed
// or scraped at runtime — and the fetched document's bytes are hashed so the transcription
// is verifiable by anyone against the stored raw bytes (Hard Rule 1). All URLs and figures
// are constants; no runtime input is interpolated into a request (Hard Rule 8).

export const JURISDICTION = "us-ca";
export const DATASET = "enacted-budget-summary";

export interface YearSource {
  /** Document URL (canonical host, no redirect). */
  url: string;
  /** Official total, contract decimal string. */
  officialTotal: string;
  /** ISO 4217. */
  currency: string;
  /** What coverage against this denominator means — the badge label. */
  scope: string;
  /** Exact locator of the figure inside the hashed document. */
  locator: string;
}

/**
 * FY 2014-15 — 2014 Budget Act, Full Budget Summary, Figure SUM-02 "2014-15 Total State
 * Expenditures by Agency" (Dollars in Millions): General Fund $107,987M + Special Funds
 * $44,324M + Bond Funds $4,046M = Total $156,357M. This is the FULL state budget, not a
 * procurement-scoped total (none is published machine-readably); coverage against it is
 * therefore labeled "procurement facts vs total budget".
 */
export const YEAR_SOURCES: Record<string, YearSource> = {
  "2014-15": {
    url: "https://ebudget.ca.gov/2014-15/pdf/Enacted/BudgetSummary/FullBudgetSummary.pdf",
    officialTotal: "156357000000.0000",
    currency: "USD",
    scope: "procurement facts vs total budget",
    locator:
      'Figure SUM-02 "2014-15 Total State Expenditures by Agency" (Dollars in Millions), ' +
      "Total row, Totals column: $156,357 million — California State Budget 2014-15 " +
      "(enacted), Full Budget Summary, printed page 11 (PDF page 11 of 69)",
  },
};
